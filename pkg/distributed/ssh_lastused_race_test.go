package distributed

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestSSHConnection_LastUsedNoRace is a REPRODUCE-FIRST (§11.4.115) guard for the
// inconsistent-lock data race on SSHConnection.LastUsed.
//
// Root cause (FACT): LastUsed is mutated by ExecuteCommand under conn.mu, but
// read/written by SSHPool.GetConnection and SSHPool.cleanup under p.mu (the pool
// lock). Two different locks guarding the same field is not synchronization, so
// concurrent ExecuteCommand + GetConnection/cleanup race on conn.LastUsed.
//
// ExecuteCommand sets LastUsed BEFORE its nil-client check, so this is reachable
// with Client == nil (no real SSH server needed).
//
// RED on pre-fix code: `go test -race` reports a DATA RACE on
// distributed.SSHConnection.LastUsed. GREEN after LastUsed is guarded by conn.mu
// everywhere it is touched.
func TestSSHConnection_LastUsedNoRace(t *testing.T) {
	pool := NewSSHPool(WithCleanupTiming(time.Millisecond, time.Millisecond))
	defer pool.Close()

	cfg := &WorkerConfig{ID: "w", Enabled: true, SSH: SSHConfig{Host: "127.0.0.1", Port: 1}}
	pool.AddWorker(cfg)

	conn := &SSHConnection{Config: cfg, Client: nil, LastUsed: time.Now(), CreatedAt: time.Now()}
	pool.mu.Lock()
	pool.connections["w"] = conn
	pool.mu.Unlock()

	ctx := context.Background()
	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writer: ExecuteCommand mutates conn.LastUsed (under conn.mu). Returns
	// quickly because Client == nil.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_, _ = conn.ExecuteCommand(ctx, "noop")
			}
		}
	}()

	// Reader/writer via the pool: GetConnection touches conn.LastUsed under p.mu.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_, _ = pool.GetConnection("w")
			}
		}
	}()

	time.Sleep(80 * time.Millisecond) // let cleanup ticker also touch LastUsed
	close(stop)
	wg.Wait()
}
