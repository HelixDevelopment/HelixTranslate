package llm

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// MockLLMClient implements LLMClient interface for testing
type MockLLMClient struct {
	mu             sync.Mutex
	responses      map[string]string
	shouldFail     bool
	sizeError      bool
	callCount      int
	maxCallsToFail int

	// Fault-injection knobs (concurrency-safe via mu) for stress/chaos tests.
	delay        time.Duration // artificial latency per Translate call
	respectCtx   bool          // when true, honor ctx cancellation during delay
	failEveryNth int           // when >0, every Nth call returns an error (transient flapping)
	customErr    error         // when set, returned instead of the generic mock error
}

// NewMockLLMClient creates a new mock LLM client
func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses:      make(map[string]string),
		shouldFail:     false,
		sizeError:      false,
		callCount:      0,
		maxCallsToFail: 0,
	}
}

// SetResponse sets a predefined response for given text
func (m *MockLLMClient) SetResponse(text, response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[text] = response
}

// SetFailure sets the mock client to fail on specific number of calls
func (m *MockLLMClient) SetFailure(shouldFail bool, maxCallsToFail int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
	m.maxCallsToFail = maxCallsToFail
}

// SetSizeError sets whether to return size limit errors
func (m *MockLLMClient) SetSizeError(sizeError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sizeError = sizeError
}

// SetDelay sets an artificial per-call latency. When respectCtx is true the
// delay is interruptible by ctx cancellation (slow-response chaos).
func (m *MockLLMClient) SetDelay(d time.Duration, respectCtx bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = d
	m.respectCtx = respectCtx
}

// SetFailEveryNth makes every Nth call return an error, simulating a flapping
// upstream (transient-error chaos). n<=0 disables.
func (m *MockLLMClient) SetFailEveryNth(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failEveryNth = n
}

// SetCustomError sets a custom error returned by failing calls.
func (m *MockLLMClient) SetCustomError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customErr = err
}

// CallCount returns the number of Translate invocations (concurrency-safe).
func (m *MockLLMClient) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// Translate implements LLMClient interface
func (m *MockLLMClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	m.mu.Lock()
	m.callCount++
	call := m.callCount
	shouldFail := m.shouldFail
	maxCallsToFail := m.maxCallsToFail
	sizeError := m.sizeError
	delay := m.delay
	respectCtx := m.respectCtx
	failEveryNth := m.failEveryNth
	customErr := m.customErr
	response, hasResponse := m.responses[text]
	m.mu.Unlock()

	// Optional artificial latency (slow-response chaos).
	if delay > 0 {
		if respectCtx {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		} else {
			time.Sleep(delay)
		}
	}

	// Honor a cancelled/expired context even without an injected delay.
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if shouldFail && call <= maxCallsToFail {
		if customErr != nil {
			return "", customErr
		}
		if sizeError {
			return "", errors.New("max_tokens limit exceeded")
		}
		return "", errors.New("mock API error")
	}

	if failEveryNth > 0 && call%failEveryNth == 0 {
		if customErr != nil {
			return "", customErr
		}
		return "", errors.New("mock transient API error")
	}

	if hasResponse {
		return response, nil
	}
	return fmt.Sprintf("translated: %s", text), nil
}

// GetProviderName implements LLMClient interface
func (m *MockLLMClient) GetProviderName() string {
	return "mock"
}
