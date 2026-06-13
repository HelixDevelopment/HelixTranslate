package grpc

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
