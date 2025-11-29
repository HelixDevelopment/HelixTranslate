package distributed

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_Call(t *testing.T) {
	t.Run("Call_ClosedStateSuccess", func(t *testing.T) {
		// Create circuit breaker with test values
		breaker := NewCircuitBreaker(5, 30*time.Second, 3)
		
		// Verify initial state is closed
		if breaker.GetState() != StateClosed {
			t.Errorf("Expected initial state to be closed, got %v", breaker.GetState())
		}
		
		// Test with a successful operation
		called := false
		err := breaker.Call(func() error {
			called = true
			return nil
		})
		
		if err != nil {
			t.Errorf("Expected no error for successful operation, got %v", err)
		}
		if !called {
			t.Error("Expected operation to be called")
		}
		
		// State should still be closed
		if breaker.GetState() != StateClosed {
			t.Errorf("Expected state to remain closed after success, got %v", breaker.GetState())
		}
	})
	
	t.Run("Call_ClosedStateFailure", func(t *testing.T) {
		// Create circuit breaker with low threshold for testing
		breaker := NewCircuitBreaker(2, 50*time.Millisecond, 3)
		
		// First failure
		err := breaker.Call(func() error {
			return errors.New("operation failed")
		})
		if err == nil {
			t.Error("Expected error for failed operation")
		}
		if err.Error() != "operation failed" {
			t.Errorf("Expected 'operation failed' error, got: %v", err)
		}
		
		// State should still be closed (threshold not reached)
		if breaker.GetState() != StateClosed {
			t.Errorf("Expected state to remain closed after single failure, got %v", breaker.GetState())
		}
		
		// Second failure - should open circuit
		err = breaker.Call(func() error {
			return errors.New("operation failed again")
		})
		if err == nil {
			t.Error("Expected error for failed operation")
		}
		
		// State should now be open
		if breaker.GetState() != StateOpen {
			t.Errorf("Expected state to be open after threshold failures, got %v", breaker.GetState())
		}
	})
	
	t.Run("Call_OpenState", func(t *testing.T) {
		// Create circuit breaker with low recovery timeout
		breaker := NewCircuitBreaker(1, 50*time.Millisecond, 3)
		
		// Cause circuit to open
		breaker.Call(func() error {
			return errors.New("operation failed")
		})
		
		// Verify circuit is open
		if breaker.GetState() != StateOpen {
			t.Errorf("Expected circuit to be open, got %v", breaker.GetState())
		}
		
		// Try to call while open - should fail immediately
		called := false
		err := breaker.Call(func() error {
			called = true
			return nil
		})
		
		if err == nil {
			t.Error("Expected error when circuit is open")
		}
		if err.Error() != "circuit breaker is open" {
			t.Errorf("Expected 'circuit breaker is open' error, got: %v", err)
		}
		if called {
			t.Error("Expected operation not to be called when circuit is open")
		}
	})
	
	t.Run("Call_HalfOpenState", func(t *testing.T) {
		// Create circuit breaker with low recovery timeout
		breaker := NewCircuitBreaker(1, 10*time.Millisecond, 2)
		
		// Cause circuit to open
		breaker.Call(func() error {
			return errors.New("operation failed")
		})
		
		// Verify circuit is open
		if breaker.GetState() != StateOpen {
			t.Errorf("Expected circuit to be open, got %v", breaker.GetState())
		}
		
		// Wait for recovery timeout
		time.Sleep(20 * time.Millisecond)
		
		// Try to call - should transition to half-open and execute
		called := false
		err := breaker.Call(func() error {
			called = true
			return nil
		})
		
		if err != nil {
			t.Errorf("Expected no error for successful operation in half-open state, got %v", err)
		}
		if !called {
			t.Error("Expected operation to be called in half-open state")
		}
		
		// Should still be half-open (not enough successes yet)
		if breaker.GetState() != StateHalfOpen {
			t.Errorf("Expected state to be half-open after one success, got %v", breaker.GetState())
		}
		
		// Another success should close circuit
		err = breaker.Call(func() error {
			return nil
		})
		
		if err != nil {
			t.Errorf("Expected no error for second successful operation, got %v", err)
		}
		
		// State should now be closed
		if breaker.GetState() != StateClosed {
			t.Errorf("Expected state to be closed after sufficient successes, got %v", breaker.GetState())
		}
	})
}