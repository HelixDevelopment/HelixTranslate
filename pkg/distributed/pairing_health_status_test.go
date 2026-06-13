package distributed

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

func hostPortFromURL(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	return hostPortFromAddr(t, strings.TrimPrefix(rawURL, "http://"))
}

func hostPortFromAddr(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("SplitHostPort(%q): %v", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("Atoi(%q): %v", portStr, err)
	}
	return host, port
}

// TestCheckServiceHealth_PreservesPairedStatus is a REPRODUCE-FIRST (§11.4.115)
// regression guard for the paired->online status-clobber bug.
//
// Root cause (FACT): checkServiceHealth unconditionally set service.Status =
// "online" on any successful /health response, demoting a *paired* worker to
// "online". GetPairedServices() filters by Status == "paired", so after the
// first 30s health-check tick every paired worker silently vanished from the
// coordinator's view (lost work distribution).
//
// RED on the pre-fix code: a reachable paired worker becomes "online" and
// disappears from GetPairedServices(). GREEN after the fix: it stays "paired".
func TestCheckServiceHealth_PreservesPairedStatus(t *testing.T) {
	// A reachable health endpoint (200 OK) — the "service is online" path.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer srv.Close()

	host, port := hostPortFromURL(t, srv.URL)

	sshPool := NewSSHPool()
	defer sshPool.Close()
	pm := NewPairingManager(sshPool, events.NewEventBus())
	defer pm.Close()

	pm.httpClient = &http.Client{Timeout: 5 * time.Second}

	paired := &RemoteService{
		WorkerID: "w1",
		Name:     "Worker 1",
		Host:     host,
		Port:     port,
		Protocol: "http",
		Status:   "paired",
		LastSeen: time.Now(),
	}
	pm.services["w1"] = paired

	// Sanity: it is paired before the health check.
	if _, ok := pm.GetPairedServices()["w1"]; !ok {
		t.Fatal("precondition: w1 should be in paired services before health check")
	}

	// Drive a single health check directly (no goroutine, no real SSH/daemon).
	pm.checkServiceHealth("w1", paired)

	if paired.Status != "paired" {
		t.Errorf("reachable paired worker was demoted: Status = %q, want %q", paired.Status, "paired")
	}
	if _, ok := pm.GetPairedServices()["w1"]; !ok {
		t.Errorf("paired worker vanished from GetPairedServices() after a successful health check")
	}
}

// TestCheckServiceHealth_OnlineStaysOnline verifies a non-paired worker that is
// reachable is reported online (we must not over-correct the fix).
func TestCheckServiceHealth_OnlineStaysOnline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port := hostPortFromURL(t, srv.URL)

	sshPool := NewSSHPool()
	defer sshPool.Close()
	pm := NewPairingManager(sshPool, events.NewEventBus())
	defer pm.Close()
	pm.httpClient = &http.Client{Timeout: 5 * time.Second}

	svc := &RemoteService{WorkerID: "w2", Host: host, Port: port, Protocol: "http", Status: "offline"}
	pm.services["w2"] = svc

	pm.checkServiceHealth("w2", svc)

	if svc.Status != "online" {
		t.Errorf("reachable offline worker should recover to online, got %q", svc.Status)
	}
}

// TestCheckServiceHealth_UnreachableMarksOffline verifies the failure path:
// an unreachable worker (paired or online) is marked offline.
func TestCheckServiceHealth_UnreachableMarksOffline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	addr := srv.Listener.Addr().String()
	srv.Close() // close immediately so the port is unreachable
	host, port := hostPortFromAddr(t, addr)

	sshPool := NewSSHPool()
	defer sshPool.Close()
	pm := NewPairingManager(sshPool, events.NewEventBus())
	defer pm.Close()
	pm.httpClient = &http.Client{Timeout: 500 * time.Millisecond}

	svc := &RemoteService{WorkerID: "w3", Host: host, Port: port, Protocol: "http", Status: "paired"}
	pm.services["w3"] = svc

	pm.checkServiceHealth("w3", svc)

	if svc.Status != "offline" {
		t.Errorf("unreachable worker should be marked offline, got %q", svc.Status)
	}
}
