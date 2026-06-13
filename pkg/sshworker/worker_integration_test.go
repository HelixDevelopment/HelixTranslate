package sshworker

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"digital.vasic.translator/pkg/logger"
)

// sshReachable reports whether a live SSH daemon is reachable using the
// SSH_WORKER_* env vars. Returns the resolved config when reachable.
//
// §11.4.3 per-environment-topology dispatch: these tests need a REAL SSH
// daemon. We never fake one — when the daemon is absent we SKIP-with-reason.
func sshReachable(t *testing.T) (SSHWorkerConfig, bool) {
	t.Helper()
	host := os.Getenv("SSH_WORKER_HOST")
	user := os.Getenv("SSH_WORKER_USER")
	pass := os.Getenv("SSH_WORKER_PASSWORD")
	portStr := os.Getenv("SSH_WORKER_PORT")
	if host == "" || user == "" {
		return SSHWorkerConfig{}, false
	}
	port := 22
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}
	// Probe TCP reachability before claiming the daemon is usable.
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return SSHWorkerConfig{}, false
	}
	_ = conn.Close()
	return SSHWorkerConfig{
		Host:              host,
		Username:          user,
		Password:          pass,
		Port:              port,
		RemoteDir:         os.Getenv("SSH_WORKER_REMOTE_DIR"),
		ConnectionTimeout: 10 * time.Second,
		CommandTimeout:    30 * time.Second,
	}, true
}

// TestIntegration_ConnectAndEcho exercises the real Connect + ExecuteCommand +
// TestConnection round-trip against a live daemon. SKIP-guarded per §11.4.3 —
// a green PASS here means a real SSH echo round-trip actually succeeded.
func TestIntegration_ConnectAndEcho(t *testing.T) {
	cfg, ok := sshReachable(t)
	if !ok {
		t.Skip("SKIP-OK: no live SSH daemon reachable (set SSH_WORKER_HOST/USER[/PASSWORD/PORT]) — §11.4.3 topology absent")
	}
	w := newTestWorker(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := w.Connect(ctx); err != nil {
		t.Fatalf("Connect to live daemon failed: %v", err)
	}
	defer w.Disconnect()

	res, err := w.ExecuteCommand(ctx, "echo connection-test")
	if err != nil {
		t.Fatalf("ExecuteCommand on live daemon failed: %v", err)
	}
	if !res.Success() {
		t.Fatalf("echo command did not succeed: exit=%d stderr=%q", res.ExitCode, res.Stderr)
	}
	if res.Stdout == "" {
		t.Fatal("echo produced empty stdout from live daemon")
	}
}

// TestIntegration_TestConnection drives the high-level TestConnection helper
// end-to-end against a live daemon. SKIP-guarded per §11.4.3.
func TestIntegration_TestConnection(t *testing.T) {
	cfg, ok := sshReachable(t)
	if !ok {
		t.Skip("SKIP-OK: no live SSH daemon reachable — §11.4.3 topology absent")
	}
	w, err := NewSSHWorker(cfg, logger.NewLogger(logger.LoggerConfig{}))
	if err != nil {
		t.Fatalf("NewSSHWorker: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := w.TestConnection(ctx); err != nil {
		t.Fatalf("TestConnection against live daemon failed: %v", err)
	}
}
