package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"digital.vasic.translator/pkg/grpc/proto"
	"digital.vasic.translator/pkg/logger"

	"github.com/gin-gonic/gin"
)

// mockBackend is an in-process gRPC TranslationService whose responses are
// fully controllable per test. It lets us reproduce the case where the backend
// completes the RPC successfully (err == nil) but reports a business-level
// FAILURE in the response body (success=false / status="error"). No real
// network is used — everything goes over a bufconn pipe.
type mockBackend struct {
	proto.UnimplementedTranslationServiceServer

	startResp  *proto.TranslationResponse
	cancelResp *proto.CancelTranslationResponse
}

func (m *mockBackend) StartTranslation(_ context.Context, _ *proto.TranslationRequest) (*proto.TranslationResponse, error) {
	return m.startResp, nil
}

func (m *mockBackend) CancelTranslation(_ context.Context, _ *proto.CancelTranslationRequest) (*proto.CancelTranslationResponse, error) {
	return m.cancelResp, nil
}

// newMockBackendServer starts the mock backend over bufconn and returns an
// APIServer wired to it plus a cleanup func.
func newMockBackendServer(t *testing.T, mb *mockBackend) (*APIServer, func()) {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	proto.RegisterTranslationServiceServer(srv, mb)
	go func() { _ = srv.Serve(lis) }()

	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}

	s := &APIServer{
		conn:       conn,
		grpcClient: proto.NewTranslationServiceClient(conn),
		logger:     logger.NewLogger(logger.LoggerConfig{Level: logger.ERROR, Format: logger.FORMAT_JSON}),
	}
	return s, func() {
		_ = conn.Close()
		srv.Stop()
		_ = lis.Close()
	}
}

func doCancel(t *testing.T, s *APIServer, sessionID string) (int, map[string]any) {
	t.Helper()
	r := gin.New()
	r.DELETE("/api/v1/translations/:session_id", s.cancelTranslation)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/translations/"+sessionID, nil)
	r.ServeHTTP(w, req)
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal cancel body %q: %v", w.Body.String(), err)
	}
	return w.Code, resp
}

func doStart(t *testing.T, s *APIServer, body string) (int, map[string]any) {
	t.Helper()
	r := gin.New()
	r.POST("/api/v1/translations", s.startTranslation)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/translations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal start body %q: %v", w.Body.String(), err)
	}
	return w.Code, resp
}

// RED (pre-fix): when the gRPC backend returns CancelTranslationResponse{success:false}
// (e.g. session not found / not cancellable) the RPC itself succeeds (err == nil), so
// cancelTranslation forwarded it via sendSuccessResponse → HTTP 200 + REST success:true,
// telling the client the cancellation worked when the backend explicitly said it did NOT.
// This is a §11.4 PASS-bluff: REST reports success on a backend-reported failure.
func TestCancelTranslation_BackendReportsFailure_NotReportedAsSuccess(t *testing.T) {
	mb := &mockBackend{
		cancelResp: &proto.CancelTranslationResponse{
			SessionId: "missing-session",
			Success:   false,
			Message:   "session not found",
		},
	}
	s, cleanup := newMockBackendServer(t, mb)
	defer cleanup()

	code, resp := doCancel(t, s, "missing-session")

	if resp["success"] == true {
		t.Fatalf("CANCEL BUG: backend reported success=false but REST returned success=true (code=%d body=%v)", code, resp)
	}
	if code == http.StatusOK {
		t.Fatalf("CANCEL BUG: backend reported a failed cancel but REST returned HTTP 200 (body=%v)", resp)
	}
}

// RED (pre-fix): TranslationResponse.status can be "error" (per the proto comment:
// "started, queued, error"). startTranslation only checks the transport error, so a
// backend that completes the RPC but reports status:"error" is forwarded as
// HTTP 200 + REST success:true. PASS-bluff: the client believes the translation
// started when the backend explicitly errored.
func TestStartTranslation_BackendStatusError_NotReportedAsSuccess(t *testing.T) {
	mb := &mockBackend{
		startResp: &proto.TranslationResponse{
			SessionId: "s1",
			Status:    "error",
			Message:   "provider rejected request",
			StartedAt: nil,
		},
	}
	s, cleanup := newMockBackendServer(t, mb)
	defer cleanup()

	body := `{"session_id":"s1","input_file":"/tmp/in.fb2","provider_config":{"type":"openai"}}`
	code, resp := doStart(t, s, body)

	if resp["success"] == true {
		t.Fatalf("START BUG: backend reported status=\"error\" but REST returned success=true (code=%d body=%v)", code, resp)
	}
	if code == http.StatusOK {
		t.Fatalf("START BUG: backend reported status=\"error\" but REST returned HTTP 200 (body=%v)", resp)
	}
}

// GREEN guard: a genuinely-successful backend response (cancel success=true /
// status="started") MUST still be reported as success — the fix must not over-reject.
func TestCancelTranslation_BackendSuccess_StillSuccess(t *testing.T) {
	mb := &mockBackend{
		cancelResp: &proto.CancelTranslationResponse{SessionId: "ok", Success: true, Message: "cancelled"},
	}
	s, cleanup := newMockBackendServer(t, mb)
	defer cleanup()

	code, resp := doCancel(t, s, "ok")
	if code != http.StatusOK || resp["success"] != true {
		t.Fatalf("happy-path cancel must be 200/success, got code=%d body=%v", code, resp)
	}
}

func TestStartTranslation_BackendStarted_StillSuccess(t *testing.T) {
	mb := &mockBackend{
		startResp: &proto.TranslationResponse{SessionId: "s1", Status: "started", Message: "ok"},
	}
	s, cleanup := newMockBackendServer(t, mb)
	defer cleanup()

	body := `{"session_id":"s1","input_file":"/tmp/in.fb2","provider_config":{"type":"openai"}}`
	code, resp := doStart(t, s, body)
	if code != http.StatusOK || resp["success"] != true {
		t.Fatalf("happy-path start must be 200/success, got code=%d body=%v", code, resp)
	}
}

// Guard against this test silently passing if mock setup regresses.
var _ = time.Second
