package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"digital.vasic.translator/pkg/grpc/proto"
	"digital.vasic.translator/pkg/logger"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// newTestAPIServerWithBackend builds an APIServer whose gRPC client points at
// addr. With an unreachable addr the connection never reaches connectivity.Ready,
// reproducing the "backend down" condition without a live gRPC server.
func newTestAPIServerWithBackend(t *testing.T, addr string) (*APIServer, func()) {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	s := &APIServer{
		conn:       conn,
		grpcClient: proto.NewTranslationServiceClient(conn),
		logger:     logger.NewLogger(logger.LoggerConfig{Level: logger.ERROR, Format: logger.FORMAT_JSON}),
	}
	return s, func() { _ = conn.Close() }
}

func doHealth(t *testing.T, s *APIServer) (int, map[string]any) {
	t.Helper()
	r := gin.New()
	r.GET("/api/v1/health", s.healthCheck)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	r.ServeHTTP(w, req)
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal health body %q: %v", w.Body.String(), err)
	}
	return w.Code, resp
}

// RED (pre-fix): healthCheck returned HTTP 200 + status="healthy" even though the
// gRPC backend (its sole upstream) was not Ready (state IDLE). That makes k8s/ALB
// health checks route real traffic to an instance that fails every request with
// 500. The fix derives health from conn.GetState(): not-Ready => 503 + unhealthy.
func TestHealthCheck_BackendNotReady_ReportsUnhealthy(t *testing.T) {
	s, cleanup := newTestAPIServerWithBackend(t, "127.0.0.1:1") // nothing listens on port 1
	defer cleanup()

	// Sanity: the backend really is not Ready in this scenario.
	if st := s.conn.GetState(); st == connectivity.Ready {
		t.Skipf("backend unexpectedly Ready (%v); scenario precondition not met", st)
	}

	code, resp := doHealth(t, s)

	if code == http.StatusOK {
		t.Fatalf("HEALTH BUG: backend not Ready but health returned HTTP 200 (body=%v)", resp)
	}
	if code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when backend not Ready, got %d (body=%v)", code, resp)
	}
	data, _ := resp["data"].(map[string]any)
	if got := data["status"]; got == "healthy" {
		t.Fatalf("HEALTH BUG: status reported \"healthy\" while backend not Ready (grpc_connected=%v)", data["grpc_connected"])
	}
	if resp["success"] != false {
		t.Fatalf("expected success=false when backend not Ready, got %v", resp["success"])
	}
}

// GREEN guard: the reported grpc_connected state must reflect the real connection
// state, and the HTTP status/"status" field must be consistent with it (this is the
// invariant the fix establishes — health is derived, never a hardcoded literal).
func TestHealthCheck_StatusConsistentWithGRPCState(t *testing.T) {
	s, cleanup := newTestAPIServerWithBackend(t, "127.0.0.1:1")
	defer cleanup()

	code, resp := doHealth(t, s)
	data, _ := resp["data"].(map[string]any)
	reportedState, _ := data["grpc_connected"].(string)
	reportedStatus, _ := data["status"].(string)

	ready := reportedState == connectivity.Ready.String()
	if ready {
		if code != http.StatusOK || reportedStatus != "healthy" {
			t.Fatalf("Ready backend must be 200/healthy, got code=%d status=%q", code, reportedStatus)
		}
	} else {
		if code != http.StatusServiceUnavailable || reportedStatus == "healthy" {
			t.Fatalf("non-Ready backend (%s) must be 503/non-healthy, got code=%d status=%q", reportedState, code, reportedStatus)
		}
	}
}
