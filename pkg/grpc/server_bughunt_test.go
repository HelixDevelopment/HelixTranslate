package grpc

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"digital.vasic.translator/pkg/grpc/proto"
)

// --- Bug A: nil ProviderConfig crashes the server ---------------------------
//
// proto3 makes provider_config an OPTIONAL message field — a client may send a
// TranslationRequest with no ProviderConfig. StartTranslation dereferenced
// req.ProviderConfig.Type unconditionally in its opening logger call, so a
// single malformed request panicked the handler. grpc-go has no default panic
// recovery here (NewServer registers no recovery interceptor), so the panic
// crashed the serving goroutine — a remote-trigger crash/DoS.
//
// RED (pre-fix): the RPC kills the connection / returns Internal, OR the test
// process panics. GREEN (post-fix): the server rejects the request cleanly with
// InvalidArgument and does NOT panic.
func TestStartTranslation_NilProviderConfig_NoPanicCleanError(t *testing.T) {
	h := newTestHarness(t, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := validRequest("nil-provider-cfg")
	req.ProviderConfig = nil // legal proto3 wire input

	resp, err := h.client.StartTranslation(ctx, req)
	if err == nil {
		t.Fatalf("expected an error for nil ProviderConfig, got resp=%v", resp)
	}
	// Must be a clean validation error, not a server crash (Internal/Unavailable
	// from a panic killing the handler/connection).
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("code = %s, want InvalidArgument (clean validation, not a crash); err=%v", st.Code(), err)
	}

	// Prove the server is STILL ALIVE after the malformed request: a subsequent
	// well-formed call must succeed. If the bad request had crashed the serving
	// goroutine, this would fail.
	good, err := h.client.StartTranslation(ctx, validRequest("nil-provider-cfg-followup"))
	if err != nil {
		t.Fatalf("server did not survive the malformed request: %v", err)
	}
	if good.GetStatus() != "started" {
		t.Fatalf("follow-up status = %q, want started", good.GetStatus())
	}
}

// --- Bug B: duplicate-SessionId overwrite leaks the prior session's context -
//
// StartTranslation stored the new session into s.sessions[req.SessionId]
// unconditionally. Calling it twice with the same SessionId overwrote the first
// session WITHOUT cancelling its CancelFunc, leaking the timeout context (a live
// timer + goroutine held until SessionTimeout, here 24h) and orphaning its
// background runTranslation. A duplicate session id is fully client-controlled.
//
// RED (pre-fix): the second call returns "started", silently clobbering the
// first session; the first session's CancelFunc is never invoked (its context
// is not Done). GREEN (post-fix): the server rejects the duplicate with
// AlreadyExists and leaves the original session intact + running.
func TestStartTranslation_DuplicateSessionId_RejectedNoLeak(t *testing.T) {
	fake := &fakeCoreTranslator{translateHold: make(chan struct{})}
	h := newTestHarness(t, fake, nil)
	defer close(fake.translateHold)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := h.client.StartTranslation(ctx, validRequest("dup-sess")); err != nil {
		t.Fatalf("first StartTranslation: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Capture the original session's context so we can assert it is NOT cancelled
	// by the duplicate call.
	h.server.sessionsMutex.RLock()
	orig := h.server.sessions["dup-sess"]
	h.server.sessionsMutex.RUnlock()
	if orig == nil {
		t.Fatal("original session not registered")
	}
	origCtx := orig.Ctx

	// Second call with the SAME session id.
	resp, err := h.client.StartTranslation(ctx, validRequest("dup-sess"))
	if err == nil {
		t.Fatalf("expected duplicate-session rejection, got resp=%v (the original session was silently overwritten)", resp)
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.AlreadyExists {
		t.Fatalf("code = %s, want AlreadyExists for duplicate session id; err=%v", st.Code(), err)
	}

	// The ORIGINAL session must still be the one in the map and still un-cancelled
	// (not leaked/orphaned by an overwrite).
	h.server.sessionsMutex.RLock()
	cur := h.server.sessions["dup-sess"]
	h.server.sessionsMutex.RUnlock()
	if cur != orig {
		t.Fatal("duplicate call replaced the original session object (overwrite + leak)")
	}
	select {
	case <-origCtx.Done():
		t.Fatal("original session context was cancelled by the duplicate request")
	default:
		// good: original still live
	}
}

// --- Bug C: GetTranslationStatus unknown-session returns a non-gRPC error ----
//
// The handler returned a bare fmt.Errorf("translation session not found: ...")
// for an unknown session id. grpc-go maps a non-status error returned from a
// handler to codes.Unknown — so a client cannot distinguish "this session does
// not exist" (a NotFound the caller can handle by, e.g., re-creating it) from a
// genuine server-side fault. StartTranslation already maps its failure modes to
// precise codes (InvalidArgument / AlreadyExists / ResourceExhausted); this read
// path silently broke that contract. Every other "not found" surface in a gRPC
// service is expected to be codes.NotFound.
//
// RED (pre-fix): status code is Unknown. GREEN (post-fix): codes.NotFound.
func TestGetTranslationStatus_UnknownSession_NotFoundCode(t *testing.T) {
	h := newTestHarness(t, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := h.client.GetTranslationStatus(ctx, &proto.TranslationStatusRequest{SessionId: "ghost-session"})
	if err == nil {
		t.Fatal("expected error for unknown session, got nil")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Fatalf("code = %s, want NotFound (client must distinguish missing session from server fault); err=%v", st.Code(), err)
	}
}

// --- Bug E: emitProgressEvent publishes bus events with an empty SessionId ----
//
// emitProgressEvent(sessionID, ...) fans a TranslationProgressEvent to the
// per-session streams AND re-publishes it onto the main event bus via
//
//	s.eventBus.Publish(events.NewEvent(events.EventType(eventType), message, metadata))
//
// events.NewEvent does NOT set Event.SessionID — it is left "". SubscribeEvents
// then maps every bus event to a proto.SystemEvent{ SessionId: event.SessionID },
// so EVERY translation lifecycle event (started / completed / failed / cancelled)
// delivered to a SubscribeEvents client arrives with an EMPTY session_id. A
// monitoring client therefore cannot tell which translation an event belongs to.
// CoreTranslatorImpl.emitProgress gets this right (it sets event.SessionID before
// publishing); the server's own emitProgressEvent dropped the sessionID it was
// handed.
//
// RED (pre-fix): the SystemEvent received over the real SubscribeEvents stream
// has SessionId == "". GREEN (post-fix): SessionId == the emitting session id.
func TestEmitProgressEvent_BusEventCarriesSessionId(t *testing.T) {
	h := newTestHarness(t, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := h.client.SubscribeEvents(ctx, &proto.EventSubscriptionRequest{
		ClientId:   "sess-id-client",
		EventTypes: []string{"completed"}, // match the emitProgressEvent eventType below
	})
	if err != nil {
		t.Fatalf("SubscribeEvents open: %v", err)
	}

	// Wait for the server's SubscribeAll registration to land before emitting.
	deadline := time.Now().Add(2 * time.Second)
	for h.server.eventBus.HandlerCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Drive the exact server method under test with a concrete session id.
	const wantSession = "evt-session-42"
	h.server.emitProgressEvent(wantSession, "completed", "", 100, "done", nil)

	ev, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv: %v", err)
	}
	if ev.GetEventType() != "completed" {
		t.Fatalf("event type = %q, want \"completed\"", ev.GetEventType())
	}
	if ev.GetSessionId() != wantSession {
		t.Fatalf("session_id = %q, want %q (lifecycle event delivered to subscribers must carry its session id)",
			ev.GetSessionId(), wantSession)
	}
}

// --- Bug D: cleanupOldSessions leaks the per-session timeout context ----------
//
// StartTranslation creates each session's context via context.WithTimeout
// (SessionTimeout). Nothing cancels that context when the translation reaches a
// terminal state — runTranslation just flips Status to "completed"/"failed".
// cleanupOldSessions then deletes the terminal session from the map WITHOUT
// calling session.CancelFunc(), so the WithTimeout timer goroutine and the
// context object stay alive until the (here) timeout fires — on a real server
// SessionTimeout is 24h. Over a long-lived server churning sessions this is an
// unbounded timer/goroutine leak. (This is the classic go-vet "the cancel
// function is not used on all paths" defect.)
//
// RED (pre-fix): after cleanup the session's Ctx is NOT Done (cancel never
// called). GREEN (post-fix): cleanup cancels the context before deleting it, so
// Ctx.Done() is closed and the timer is released.
func TestCleanupOldSessions_CancelsContextNoLeak(t *testing.T) {
	// Short SessionTimeout so the terminal-age gate in cleanupOldSessions fires.
	h := newTestHarness(t, &fakeCoreTranslator{}, &ServerConfig{
		MaxConcurrentTranslations: 10,
		SessionTimeout:            time.Millisecond,
		StreamBufferSize:          10,
	})

	// Register a session directly and drive it to a terminal state, mirroring
	// what runTranslation does, then age it past SessionTimeout.
	sessCtx, cancelFn := context.WithTimeout(context.Background(), 24*time.Hour)
	sess := &TranslationSession{
		ID:         "leak-sess",
		Status:     "completed",
		CreatedAt:  time.Now().Add(-time.Hour),
		UpdatedAt:  time.Now().Add(-time.Hour), // old enough to be cleaned
		CancelFunc: cancelFn,
		Ctx:        sessCtx,
	}
	h.server.sessionsMutex.Lock()
	h.server.sessions["leak-sess"] = sess
	h.server.sessionsMutex.Unlock()

	h.server.cleanupOldSessions()

	// The session must be gone from the map (cleanup ran)...
	h.server.sessionsMutex.RLock()
	_, stillThere := h.server.sessions["leak-sess"]
	h.server.sessionsMutex.RUnlock()
	if stillThere {
		t.Fatal("cleanupOldSessions did not remove the aged terminal session")
	}

	// ...and its context MUST have been cancelled (timer released), not leaked.
	select {
	case <-sessCtx.Done():
		// good: context cancelled, timer freed
	default:
		t.Fatal("cleanupOldSessions deleted the session without cancelling its context (timer/goroutine leak)")
	}
}
