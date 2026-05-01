package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Scheduler runs periodic model discovery in the background.
type Scheduler struct {
	service   *Service
	interval  time.Duration
	mu        sync.Mutex
	running   bool
	stopCh    chan struct{}
	wg        sync.WaitGroup
	onCycle   func(error) // optional callback after each cycle
}

// NewScheduler creates a background discovery scheduler.
func NewScheduler(service *Service, interval time.Duration) *Scheduler {
	const minInterval = 10 * time.Second
	if interval < minInterval {
		interval = minInterval
	}
	return &Scheduler{
		service:  service,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// SetOnCycle configures a callback invoked after each discovery cycle.
func (s *Scheduler) SetOnCycle(fn func(error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onCycle = fn
}

// Start begins the background discovery loop. It is safe to call multiple times
// but will return an error if already running.
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	s.running = true
	s.stopCh = make(chan struct{})
	s.wg.Add(1)

	go s.loop()
	return nil
}

// Stop signals the scheduler to shut down and waits for the goroutine to exit.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler not running")
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	s.wg.Wait()
	return nil
}

// IsRunning reports whether the scheduler is currently active.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// RunOnce executes a single discovery cycle synchronously.
func (s *Scheduler) RunOnce(ctx context.Context) error {
	return s.service.Discover(ctx)
}

func (s *Scheduler) loop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	s.runCycle()

	for {
		select {
		case <-ticker.C:
			s.runCycle()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Scheduler) runCycle() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := s.service.Discover(ctx)

	s.mu.Lock()
	cb := s.onCycle
	s.mu.Unlock()

	if cb != nil {
		cb(err)
	}
}
