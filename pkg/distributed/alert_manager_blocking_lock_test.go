package distributed

import (
	"sync"
	"testing"
	"time"
)

// blockingChannel is an AlertChannel whose SendAlert blocks until released.
// It signals "I am now inside SendAlert" via started, then waits on release.
type blockingChannel struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (b *blockingChannel) Name() string { return "blocking" }

func (b *blockingChannel) SendAlert(_ *DriftAlert) error {
	b.once.Do(func() { close(b.started) })
	<-b.release // simulate a slow / unreachable channel (SMTP dial, webhook POST)
	return nil
}

// TestAlertManager_SendAlertDoesNotHoldLockDuringChannelIO proves AlertManager
// does NOT hold am.mu across the blocking channel.SendAlert() network I/O.
//
// Defect (RED, pre-fix): SendAlert acquires am.mu.Lock() and only releases it
// (deferred) AFTER the channel send loop completes. channel.SendAlert performs
// real blocking I/O (EmailAlertChannel: SMTP dial up to 30s; WebhookAlertChannel:
// HTTP POST up to 30s). While one alert is mid-send against a slow/unreachable
// channel, EVERY other AlertManager method that needs am.mu — GetAlertHistory,
// AcknowledgeAlert, AddChannel, a concurrent SendAlert — is blocked. This is the
// "no blocking operation inside a held lock" violation from the project CLAUDE.md.
//
// Hermetic: the blockingChannel needs no network; it pins SendAlert open with a
// channel. We then assert GetAlertHistory returns PROMPTLY (it must not wait on
// the in-flight send). On pre-fix code GetAlertHistory blocks on am.RLock() until
// release is closed → the test times out → RED.
func TestAlertManager_SendAlertDoesNotHoldLockDuringChannelIO(t *testing.T) {
	am := NewAlertManager(100)

	bc := &blockingChannel{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	am.AddChannel(bc)

	// Fire an alert in the background; its channel send will block.
	sendDone := make(chan struct{})
	go func() {
		_ = am.SendAlert(&DriftAlert{WorkerID: "w1", Severity: "high"})
		close(sendDone)
	}()

	// Wait until we are genuinely inside channel.SendAlert (lock held in pre-fix code).
	select {
	case <-bc.started:
	case <-time.After(2 * time.Second):
		close(bc.release)
		t.Fatal("channel.SendAlert never started")
	}

	// Now, while the send is in flight, an independent reader must NOT be blocked.
	historyReturned := make(chan int, 1)
	go func() {
		h := am.GetAlertHistory(0)
		historyReturned <- len(h)
	}()

	select {
	case <-historyReturned:
		// GOOD (GREEN): reader completed while a channel send was in flight.
	case <-time.After(1 * time.Second):
		// BAD (RED): GetAlertHistory is blocked on am.mu held across channel I/O.
		close(bc.release) // unblock so the goroutine can exit
		<-sendDone
		t.Fatal("GetAlertHistory blocked while a channel send held am.mu across blocking I/O")
	}

	// Cleanup: release the in-flight send.
	close(bc.release)
	<-sendDone
}
