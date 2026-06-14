package main

import (
	"errors"
	"net"
	"testing"

	"github.com/gin-gonic/gin"
)

// fakeRunner records the addr it was asked to bind and returns a canned error,
// standing in for *gin.Engine.Run without binding any real socket.
type fakeRunner struct {
	gotAddr string
	err     error
}

func (f *fakeRunner) Run(addr ...string) error {
	if len(addr) > 0 {
		f.gotAddr = addr[0]
	}
	return f.err
}

// TestRunServer_PropagatesStartError proves runServer returns the underlying
// server-start error instead of swallowing it. Before the fix, main() called
// router.Run(...) and discarded the error return entirely, so a failed bind
// produced no signal at all. A helper that dropped the error would return nil
// here and FAIL the test.
func TestRunServer_PropagatesStartError(t *testing.T) {
	wantErr := errors.New("listen tcp :8090: bind: address already in use")
	fr := &fakeRunner{err: wantErr}

	got := runServer(fr, ":8090")

	if got == nil {
		t.Fatal("runServer swallowed the start error: got nil, want non-nil")
	}
	if !errors.Is(got, wantErr) {
		t.Errorf("runServer did not propagate the error: got %v, want %v", got, wantErr)
	}
	if fr.gotAddr != ":8090" {
		t.Errorf("runServer passed wrong addr: got %q, want %q", fr.gotAddr, ":8090")
	}
}

// TestRunServer_NilOnSuccess proves a successful start returns nil (no spurious
// error on the happy path).
func TestRunServer_NilOnSuccess(t *testing.T) {
	if err := runServer(&fakeRunner{err: nil}, ":0"); err != nil {
		t.Errorf("runServer must return nil on success: got %v", err)
	}
}

// TestRunServer_RealBindFailurePropagated proves the contract end-to-end with a
// real *gin.Engine: binding an already-occupied ephemeral port surfaces a real
// error through runServer. Uses :0 to grab a free port (no privileged bind),
// then occupies it and asserts gin's Run reports the conflict.
func TestRunServer_RealBindFailurePropagated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Grab a free ephemeral port and KEEP it occupied for the duration.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not acquire ephemeral port: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String() // host:port now occupied

	router := gin.New()
	got := runServer(router, addr)

	if got == nil {
		t.Fatalf("runServer must return a real bind error for occupied %s, got nil", addr)
	}
}
