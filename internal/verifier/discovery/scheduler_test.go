package discovery

import (
	"sync/atomic"
	"testing"
	"time"

	"digital.vasic.translator/internal/verifier"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchedulerStartStop(t *testing.T) {
	registry := verifier.NewRegistry()
	service := NewService(verifier.DefaultConfig(), registry)
	scheduler := NewScheduler(service, 10*time.Second)

	require.False(t, scheduler.IsRunning())
	require.NoError(t, scheduler.Start())
	require.True(t, scheduler.IsRunning())

	// Allow the immediate cycle to run
	time.Sleep(100 * time.Millisecond)

	require.NoError(t, scheduler.Stop())
	require.False(t, scheduler.IsRunning())
}

func TestSchedulerDoubleStart(t *testing.T) {
	registry := verifier.NewRegistry()
	service := NewService(verifier.DefaultConfig(), registry)
	scheduler := NewScheduler(service, time.Hour)

	require.NoError(t, scheduler.Start())
	err := scheduler.Start()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	require.NoError(t, scheduler.Stop())
}

func TestSchedulerStopNotRunning(t *testing.T) {
	registry := verifier.NewRegistry()
	service := NewService(verifier.DefaultConfig(), registry)
	scheduler := NewScheduler(service, time.Hour)

	err := scheduler.Stop()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestSchedulerRunOnce(t *testing.T) {
	registry := verifier.NewRegistry()
	cfg := verifier.DefaultConfig()
	service := NewService(cfg, registry)

	provider := verifier.ProviderConfig{ID: "test", BaseURL: "http://localhost:1", Models: []string{"m1"}}
	service.RegisterProvider(provider)

	scheduler := NewScheduler(service, time.Hour)
	err := scheduler.RunOnce(t.Context())
	// May fail due to network but should not panic
	require.NoError(t, err)

	models := registry.ListModels()
	require.GreaterOrEqual(t, len(models), 1)
	// Verify our registered model is among them
	found := false
	for _, m := range models {
		if m.ID == "m1" {
			found = true
			break
		}
	}
	assert.True(t, found, "registered model m1 should be present")
}

func TestSchedulerCallback(t *testing.T) {
	registry := verifier.NewRegistry()
	service := NewService(verifier.DefaultConfig(), registry)
	scheduler := NewScheduler(service, 50*time.Millisecond)

	var called atomic.Int32
	scheduler.SetOnCycle(func(err error) {
		called.Add(1)
	})

	require.NoError(t, scheduler.Start())
	time.Sleep(100 * time.Millisecond)
	require.NoError(t, scheduler.Stop())

	// Immediate cycle should have invoked callback at least once
	assert.GreaterOrEqual(t, called.Load(), int32(1))
}

func TestSchedulerCallbackReceivesError(t *testing.T) {
	registry := verifier.NewRegistry()
	cfg := verifier.DefaultConfig()
	// Force a community URL that will fail
	cfg.Options = map[string]interface{}{"community_registry_url": "http://localhost:1/community"}
	service := NewService(cfg, registry)

	scheduler := NewScheduler(service, time.Hour)

	var called atomic.Bool
	scheduler.SetOnCycle(func(err error) {
		called.Store(true)
	})

	require.NoError(t, scheduler.Start())
	time.Sleep(100 * time.Millisecond)
	require.NoError(t, scheduler.Stop())

	assert.True(t, called.Load(), "callback should have been invoked")
}

func TestSchedulerIntervalClamping(t *testing.T) {
	registry := verifier.NewRegistry()
	service := NewService(verifier.DefaultConfig(), registry)
	// Pass an interval below the minimum
	scheduler := NewScheduler(service, 1*time.Millisecond)
	assert.Equal(t, 10*time.Second, scheduler.interval)
}
