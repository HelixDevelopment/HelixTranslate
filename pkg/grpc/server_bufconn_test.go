package grpc

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/grpc/proto"
	"digital.vasic.translator/pkg/logger"
)

// fakeCoreTranslator is an in-package test double standing in for the real
// translation backend (ebook/format/llm) which is exercised by the W15
// integration tier. Per §11.4.27 this fake lives ONLY in test sources and is
// used to drive the gRPC SERVER layer (request validation, status, error
// codes, response shaping, streaming) over a REAL in-process gRPC connection
// (bufconn) — the wire protocol, codec, and dispatch are genuine, not mocked.
type fakeCoreTranslator struct {
	mu sync.Mutex

	translateCalls int
	cancelCalls    int
	getStatusCalls int

	// behaviour knobs
	translateResp *proto.TranslationStatusResponse
	translateErr  error
	translateHold chan struct{} // if non-nil, Translate blocks until closed

	cancelErr error

	statusResp *proto.TranslationStatusResponse
	statusErr  error
}

func (f *fakeCoreTranslator) Translate(ctx context.Context, req *proto.TranslationRequest, _ *events.EventBus) (*proto.TranslationStatusResponse, error) {
	f.mu.Lock()
	f.translateCalls++
	hold := f.translateHold
	resp := f.translateResp
	err := f.translateErr
	f.mu.Unlock()

	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if resp == nil && err == nil {
		resp = &proto.TranslationStatusResponse{
			SessionId: req.GetSessionId(),
			Status:    "completed",
			Files:     []*proto.GeneratedFile{{Path: "out.epub", Type: "epub", Verified: true}},
			Steps:     []*proto.TranslationStep{{Name: "translation", Status: "completed"}},
		}
	}
	return resp, err
}

func (f *fakeCoreTranslator) Cancel(_ string) error {
	f.mu.Lock()
	f.cancelCalls++
	err := f.cancelErr
	f.mu.Unlock()
	return err
}

func (f *fakeCoreTranslator) GetStatus(_ string) (*proto.TranslationStatusResponse, error) {
	f.mu.Lock()
	f.getStatusCalls++
	resp := f.statusResp
	err := f.statusErr
	f.mu.Unlock()
	if resp == nil && err == nil {
		// emulate "no live status from core" so the server falls back to its
		// own session-derived response (mirrors CoreTranslatorImpl returning a
		// not-found error for an unknown session).
		return nil, fmt.Errorf("translation session not found")
	}
	return resp, err
}

func (f *fakeCoreTranslator) snapshot() (translate, cancel, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.translateCalls, f.cancelCalls, f.getStatusCalls
}

// testHarness wires a real grpc.Server (from NewServer) onto a bufconn listener
// and dials a real client over it. The returned client speaks the genuine gRPC
// protocol in-process.
type testHarness struct {
	client proto.TranslationServiceClient
	server *Server
	fake   *fakeCoreTranslator
	conn   *grpc.ClientConn
	lis    *bufconn.Listener
}

func newTestHarness(t *testing.T, fake *fakeCoreTranslator, cfg *ServerConfig) *testHarness {
	t.Helper()
	if fake == nil {
		fake = &fakeCoreTranslator{}
	}
	lis := bufconn.Listen(1024 * 1024)
	srv := NewServer(events.NewEventBus(), logger.NewNoOpLogger(), fake, cfg)

	proto.RegisterTranslationServiceServer(srv.GetGRPCServer(), srv)

	go func() {
		// Serve returns when the listener is closed; ignore the resulting error.
		_ = srv.GetGRPCServer().Serve(lis)
	}()

	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}

	h := &testHarness{
		client: proto.NewTranslationServiceClient(conn),
		server: srv,
		fake:   fake,
		conn:   conn,
		lis:    lis,
	}
	t.Cleanup(func() {
		_ = conn.Close()
		srv.GetGRPCServer().Stop()
		_ = lis.Close()
	})
	return h
}

func validRequest(sessionID string) *proto.TranslationRequest {
	return &proto.TranslationRequest{
		SessionId:      sessionID,
		InputFile:      "/tmp/in.epub",
		OutputFile:     "/tmp/out.epub",
		SourceLang:     "ru",
		TargetLang:     "sr",
		Script:         "cyrillic",
		ProviderConfig: &proto.ProviderConfig{Type: "openai", Model: "gpt-4"},
		Options:        &proto.TranslationOptions{},
	}
}

// --- StartTranslation -------------------------------------------------------

func TestStartTranslation_RealRPC_ReturnsStarted(t *testing.T) {
	fake := &fakeCoreTranslator{translateHold: make(chan struct{})} // hold so session stays active
	h := newTestHarness(t, fake, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.client.StartTranslation(ctx, validRequest("sess-start-1"))
	if err != nil {
		t.Fatalf("StartTranslation RPC error: %v", err)
	}
	if resp.GetStatus() != "started" {
		t.Errorf("status = %q, want \"started\"", resp.GetStatus())
	}
	if resp.GetSessionId() != "sess-start-1" {
		t.Errorf("session id = %q, want sess-start-1", resp.GetSessionId())
	}
	if resp.GetStartedAt() == nil {
		t.Error("StartedAt timestamp not populated in response")
	}
	if resp.GetEstimatedDurationSeconds() != 300 {
		t.Errorf("estimated duration = %d, want 300", resp.GetEstimatedDurationSeconds())
	}
	// Real backend dispatch happened: runTranslation -> fake.Translate.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if tr, _, _ := fake.snapshot(); tr >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if tr, _, _ := fake.snapshot(); tr < 1 {
		t.Error("expected runTranslation to invoke the core translator backend")
	}
	close(fake.translateHold)
}

func TestStartTranslation_MaxConcurrentReached_ReturnsError(t *testing.T) {
	fake := &fakeCoreTranslator{translateHold: make(chan struct{})}
	h := newTestHarness(t, fake, &ServerConfig{
		MaxConcurrentTranslations: 1,
		SessionTimeout:            time.Hour,
		StreamBufferSize:          10,
	})
	defer close(fake.translateHold)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First fills the single slot (its Translate blocks on the hold channel).
	if _, err := h.client.StartTranslation(ctx, validRequest("sess-cap-1")); err != nil {
		t.Fatalf("first StartTranslation: %v", err)
	}
	// give the session map time to register
	time.Sleep(50 * time.Millisecond)

	// Second must be rejected at the concurrency gate.
	resp, err := h.client.StartTranslation(ctx, validRequest("sess-cap-2"))
	if err == nil {
		t.Fatal("expected error when max concurrent translations reached, got nil")
	}
	// server returns a populated response alongside the error
	if resp != nil && resp.GetStatus() != "error" {
		t.Errorf("rejection status = %q, want \"error\"", resp.GetStatus())
	}
}

// --- GetTranslationStatus ---------------------------------------------------

func TestGetTranslationStatus_UnknownSession_ReturnsError(t *testing.T) {
	h := newTestHarness(t, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := h.client.GetTranslationStatus(ctx, &proto.TranslationStatusRequest{SessionId: "does-not-exist"})
	if err == nil {
		t.Fatal("expected error for unknown session, got nil")
	}
}

func TestGetTranslationStatus_KnownSession_ReturnsSessionState(t *testing.T) {
	fake := &fakeCoreTranslator{translateHold: make(chan struct{})}
	h := newTestHarness(t, fake, nil)
	defer close(fake.translateHold)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := h.client.StartTranslation(ctx, validRequest("sess-status-1")); err != nil {
		t.Fatalf("StartTranslation: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	st, err := h.client.GetTranslationStatus(ctx, &proto.TranslationStatusRequest{SessionId: "sess-status-1"})
	if err != nil {
		t.Fatalf("GetTranslationStatus RPC error: %v", err)
	}
	if st.GetSessionId() != "sess-status-1" {
		t.Errorf("session id = %q, want sess-status-1", st.GetSessionId())
	}
	if st.GetStartedAt() == nil {
		t.Error("StartedAt not populated")
	}
}

func TestGetTranslationStatus_CoreStatusOverridesSessionFields(t *testing.T) {
	fake := &fakeCoreTranslator{
		translateHold: make(chan struct{}),
		statusResp: &proto.TranslationStatusResponse{
			Status:             "running",
			ProgressPercentage: 42.5,
			CurrentStep:        "translation",
		},
	}
	h := newTestHarness(t, fake, nil)
	defer close(fake.translateHold)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := h.client.StartTranslation(ctx, validRequest("sess-status-override")); err != nil {
		t.Fatalf("StartTranslation: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	st, err := h.client.GetTranslationStatus(ctx, &proto.TranslationStatusRequest{SessionId: "sess-status-override"})
	if err != nil {
		t.Fatalf("GetTranslationStatus: %v", err)
	}
	// server merges the live core status over the session-derived fields
	if st.GetProgressPercentage() != 42.5 {
		t.Errorf("progress = %v, want 42.5 (core override)", st.GetProgressPercentage())
	}
	if st.GetCurrentStep() != "translation" {
		t.Errorf("current step = %q, want \"translation\"", st.GetCurrentStep())
	}
}

// --- ListTranslations -------------------------------------------------------

func TestListTranslations_EmptyAndPopulated(t *testing.T) {
	fake := &fakeCoreTranslator{translateHold: make(chan struct{})}
	h := newTestHarness(t, fake, nil)
	defer close(fake.translateHold)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// empty
	lst, err := h.client.ListTranslations(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("ListTranslations (empty): %v", err)
	}
	if lst.GetTotalCount() != 0 {
		t.Errorf("empty total count = %d, want 0", lst.GetTotalCount())
	}

	// populate with two active sessions
	for _, id := range []string{"list-1", "list-2"} {
		if _, err := h.client.StartTranslation(ctx, validRequest(id)); err != nil {
			t.Fatalf("StartTranslation %s: %v", id, err)
		}
	}
	time.Sleep(80 * time.Millisecond)

	lst, err = h.client.ListTranslations(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("ListTranslations (populated): %v", err)
	}
	if lst.GetTotalCount() != 2 {
		t.Errorf("total count = %d, want 2", lst.GetTotalCount())
	}
	if len(lst.GetTranslations()) != 2 {
		t.Errorf("translations slice len = %d, want 2", len(lst.GetTranslations()))
	}
}

// --- CancelTranslation ------------------------------------------------------

func TestCancelTranslation_UnknownSession_NotFound(t *testing.T) {
	h := newTestHarness(t, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.client.CancelTranslation(ctx, &proto.CancelTranslationRequest{SessionId: "nope", Reason: "test"})
	if err != nil {
		t.Fatalf("CancelTranslation RPC error: %v", err)
	}
	if resp.GetSuccess() {
		t.Error("Success = true for unknown session, want false")
	}
	if resp.GetMessage() == "" {
		t.Error("expected a not-found message")
	}
}

func TestCancelTranslation_KnownSession_Succeeds(t *testing.T) {
	fake := &fakeCoreTranslator{translateHold: make(chan struct{})}
	h := newTestHarness(t, fake, nil)
	defer close(fake.translateHold)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := h.client.StartTranslation(ctx, validRequest("cancel-1")); err != nil {
		t.Fatalf("StartTranslation: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	resp, err := h.client.CancelTranslation(ctx, &proto.CancelTranslationRequest{SessionId: "cancel-1", Reason: "user requested"})
	if err != nil {
		t.Fatalf("CancelTranslation RPC error: %v", err)
	}
	if !resp.GetSuccess() {
		t.Errorf("Success = false, want true; message=%q", resp.GetMessage())
	}
	// the server delegates cancellation to the core translator backend
	if _, cancelCalls, _ := fake.snapshot(); cancelCalls < 1 {
		t.Error("expected core translator Cancel to be invoked")
	}
}

// --- GetProviders -----------------------------------------------------------

func TestGetProviders_RealRPC_ReturnsDefaults(t *testing.T) {
	h := newTestHarness(t, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.client.GetProviders(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("GetProviders RPC error: %v", err)
	}
	if len(resp.GetProviders()) != 3 {
		t.Fatalf("provider count = %d, want 3 (openai, anthropic, ssh)", len(resp.GetProviders()))
	}
	byType := map[string]*proto.ProviderInfo{}
	for _, p := range resp.GetProviders() {
		byType[p.GetType()] = p
	}
	for _, want := range []string{"openai", "anthropic", "ssh"} {
		if _, ok := byType[want]; !ok {
			t.Errorf("missing provider type %q in response", want)
		}
	}
	// response shaping: capabilities + status survive the wire round-trip
	if oai := byType["openai"]; oai != nil {
		if oai.GetStatus() == nil || !oai.GetStatus().GetAvailable() {
			t.Error("openai provider status not available over the wire")
		}
		if oai.GetCapabilities()["quality"] != "high" {
			t.Errorf("openai quality capability = %q, want \"high\"", oai.GetCapabilities()["quality"])
		}
	}
}

// --- StreamTranslationProgress (server streaming setup) ---------------------

func TestStreamTranslationProgress_SendsInitialStatusEvent(t *testing.T) {
	fake := &fakeCoreTranslator{translateHold: make(chan struct{})}
	h := newTestHarness(t, fake, nil)
	defer close(fake.translateHold)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := h.client.StartTranslation(ctx, validRequest("stream-1")); err != nil {
		t.Fatalf("StartTranslation: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	stream, err := h.client.StreamTranslationProgress(ctx, &proto.TranslationStreamRequest{
		SessionId: "stream-1",
		ClientId:  "client-A",
	})
	if err != nil {
		t.Fatalf("StreamTranslationProgress open: %v", err)
	}

	// The server immediately sends an initial status_update event for a known session.
	ev, err := stream.Recv()
	if err != nil {
		t.Fatalf("stream.Recv: %v", err)
	}
	if ev.GetSessionId() != "stream-1" {
		t.Errorf("event session id = %q, want stream-1", ev.GetSessionId())
	}
	if ev.GetEventType() != "status_update" {
		t.Errorf("event type = %q, want status_update", ev.GetEventType())
	}
	if ev.GetTimestamp() == nil {
		t.Error("event timestamp not populated")
	}
}

func TestStreamTranslationProgress_ReceivesEmittedProgressEvent(t *testing.T) {
	fake := &fakeCoreTranslator{translateHold: make(chan struct{})}
	h := newTestHarness(t, fake, nil)
	defer close(fake.translateHold)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := h.client.StartTranslation(ctx, validRequest("stream-2")); err != nil {
		t.Fatalf("StartTranslation: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	stream, err := h.client.StreamTranslationProgress(ctx, &proto.TranslationStreamRequest{
		SessionId: "stream-2",
		ClientId:  "client-B",
	})
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	// drain the initial status_update
	if _, err := stream.Recv(); err != nil {
		t.Fatalf("initial recv: %v", err)
	}

	// Give the stream goroutine a moment to register in s.streams, then emit.
	time.Sleep(50 * time.Millisecond)
	h.server.emitProgressEvent("stream-2", "progress", "translation", 55, "halfway", map[string]interface{}{"k": "v"})

	ev, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv emitted event: %v", err)
	}
	if ev.GetEventType() != "progress" {
		t.Errorf("event type = %q, want progress", ev.GetEventType())
	}
	if ev.GetProgressPercentage() != 55 {
		t.Errorf("progress = %v, want 55", ev.GetProgressPercentage())
	}
	if ev.GetMessage() != "halfway" {
		t.Errorf("message = %q, want halfway", ev.GetMessage())
	}
	if ev.GetMetadata()["k"] != "v" {
		t.Errorf("metadata[k] = %q, want v", ev.GetMetadata()["k"])
	}
}

// --- SubscribeEvents (server streaming, event-bus fan-out) ------------------

func TestSubscribeEvents_ReceivesBusEventFilteredByType(t *testing.T) {
	h := newTestHarness(t, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := h.client.SubscribeEvents(ctx, &proto.EventSubscriptionRequest{
		ClientId:   "evt-client",
		EventTypes: []string{string(events.EventTranslationProgress)},
	})
	if err != nil {
		t.Fatalf("SubscribeEvents open: %v", err)
	}

	// allow the server's SubscribeAll registration to land
	time.Sleep(80 * time.Millisecond)

	// publish a NON-matching event (must be filtered out) then a matching one
	h.server.eventBus.Publish(events.NewEvent(events.EventTranslationStarted, "ignored", nil))
	h.server.eventBus.Publish(events.NewEvent(events.EventTranslationProgress, "matched", map[string]interface{}{"pct": 10}))

	ev, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv: %v", err)
	}
	if ev.GetEventType() != string(events.EventTranslationProgress) {
		t.Errorf("event type = %q, want %q (filter should drop the started event)",
			ev.GetEventType(), events.EventTranslationProgress)
	}
	if ev.GetClientId() != "evt-client" {
		t.Errorf("client id = %q, want evt-client", ev.GetClientId())
	}
	if ev.GetData()["pct"] != "10" {
		t.Errorf("data[pct] = %q, want \"10\" (int formatted via %%v)", ev.GetData()["pct"])
	}
}

// TestSubscribeEvents_NoHandlerLeakAfterStreamEnds proves the gRPC runtime
// signature of the Unsubscribe fix on the real wire path: when a SubscribeEvents
// stream ends (client cancels), the server removes its event-bus handler instead
// of leaking it. Before the fix SubscribeEvents called SubscribeAll with no way
// to deregister, so every stream permanently appended a dead handler to the bus
// — invoked (and a send-on-closed panic recovered) on every future Publish,
// growing unbounded on a long-running server.
func TestSubscribeEvents_NoHandlerLeakAfterStreamEnds(t *testing.T) {
	h := newTestHarness(t, nil, nil)

	if got := h.server.eventBus.HandlerCount(); got != 0 {
		t.Fatalf("baseline HandlerCount = %d, want 0", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	if _, err := h.client.SubscribeEvents(ctx, &proto.EventSubscriptionRequest{
		ClientId: "leak-client",
	}); err != nil {
		cancel()
		t.Fatalf("SubscribeEvents open: %v", err)
	}

	// wait for the server's SubscribeAll registration to land
	deadline := time.Now().Add(2 * time.Second)
	for h.server.eventBus.HandlerCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := h.server.eventBus.HandlerCount(); got != 1 {
		cancel()
		t.Fatalf("after stream open: HandlerCount = %d, want 1 (handler should be registered)", got)
	}

	// end the stream: the server goroutine returns via ctx.Done() and runs its
	// deferred Unsubscribe.
	cancel()

	// poll until the handler is removed (bounded — cancellation propagates over
	// bufconn asynchronously).
	deadline = time.Now().Add(3 * time.Second)
	for h.server.eventBus.HandlerCount() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := h.server.eventBus.HandlerCount(); got != 0 {
		t.Fatalf("after stream cancel: HandlerCount = %d, want 0 (handler leaked)", got)
	}
}

// closeStreamSanity ensures a stream over bufconn terminates cleanly when the
// client context is cancelled (exercises the ctx.Done() path).
func TestStream_ClientCancel_TerminatesCleanly(t *testing.T) {
	fake := &fakeCoreTranslator{translateHold: make(chan struct{})}
	h := newTestHarness(t, fake, nil)
	defer close(fake.translateHold)

	startCtx, startCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer startCancel()
	if _, err := h.client.StartTranslation(startCtx, validRequest("stream-cancel")); err != nil {
		t.Fatalf("StartTranslation: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	streamCtx, streamCancel := context.WithCancel(context.Background())
	stream, err := h.client.StreamTranslationProgress(streamCtx, &proto.TranslationStreamRequest{
		SessionId: "stream-cancel",
		ClientId:  "c",
	})
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	if _, err := stream.Recv(); err != nil {
		t.Fatalf("initial recv: %v", err)
	}
	streamCancel()
	// subsequent Recv must return an error (context cancelled / EOF), not hang.
	_, err = stream.Recv()
	if err == nil {
		t.Error("expected error after client cancel, got nil")
	}
	_ = io.EOF // documents the acceptable terminal condition
}
