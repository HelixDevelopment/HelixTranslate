package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"digital.vasic.translator/pkg/logger"
	"github.com/stretchr/testify/require"
)

// freePort returns an OS-assigned free TCP port.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// TestServerStartStopLifecycle proves that Server.Start actually serves and
// Server.Stop actually shuts the listener down. Before the fix:
//   - Start() blocks forever on ListenAndServe (no way to verify it returned)
//   - Stop() is a no-op placeholder that returns nil while the server keeps
//     accepting connections — a §11.4 PASS-bluff (claims to stop, doesn't).
//
// The test asserts the user-visible contract: after Stop(), the port no longer
// answers HTTP requests.
func TestServerStartStopLifecycle(t *testing.T) {
	port := freePort(t)
	srv := NewServer(ServerConfig{
		Port: port,
		Logger: logger.NewLogger(logger.LoggerConfig{
			Level:  logger.ERROR,
			Format: logger.FORMAT_TEXT,
		}),
	})

	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start(context.Background()) }()

	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	client := &http.Client{Timeout: 2 * time.Second}

	// Wait until the server is up and answering.
	up := false
	for i := 0; i < 50; i++ {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			up = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.True(t, up, "server never came up on port %d (Start did not serve)", port)

	// Stop the server. The contract: Stop returns nil AND the server actually
	// stops accepting connections.
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, srv.Stop(stopCtx), "Stop returned error")

	// After Stop, ListenAndServe must have returned (Start goroutine unblocks).
	select {
	case err := <-startErr:
		require.ErrorIs(t, err, http.ErrServerClosed,
			"Start should return http.ErrServerClosed after graceful Stop, got: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("Start() did not return after Stop() — Stop did not shut the server down (PASS-bluff placeholder)")
	}

	// And the port must no longer answer.
	stillUp := false
	for i := 0; i < 25; i++ {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			stillUp = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.False(t, stillUp, "server still answering after Stop() — Stop did not actually stop the server")
}
