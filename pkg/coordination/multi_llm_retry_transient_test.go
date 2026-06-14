package coordination

import (
	"context"
	"errors"
	"testing"
)

// TestTranslateWithRetry_RetriesSameInstanceOnTransientError is a REPRODUCE-FIRST
// (§11.4.115 / §11.4.146) guard for a "retry that never retries" defect.
//
// Root cause (FACT): TranslateWithRetry's loop budget is maxRetries*len(instances)
// attempts, but a `triedInstances[instance.ID]` map permanently blocks any instance
// from being attempted more than once (multi_llm.go:324 — `continue` skips an
// already-tried instance). With a SINGLE instance that returns a TRANSIENT error on
// the first call and would succeed on the next, the coordinator therefore makes
// exactly ONE provider call and gives up — despite maxRetries=3 — so a recoverable
// translation is reported as a permanent failure to the end user. The maxRetries
// multiplier in the loop bound is dead code: only len(instances) distinct calls ever
// happen.
//
// RED on the pre-fix code: the single instance is called once, TranslateWithRetry
// returns an error, and the would-be-successful retry never occurs.
//
// GREEN after the fix: the instance is retried up to maxRetries times, the second
// call succeeds, and the real translation is returned.
func TestTranslateWithRetry_RetriesSameInstanceOnTransientError(t *testing.T) {
	// Script: first call transient error, second call succeeds.
	flaky := &faultMockTranslator{script: []faultStep{
		{translation: "", err: errors.New("temporary network error")},
		{translation: "Recovered", err: nil},
	}}
	c := newCoordinator(3, 0, &LLMInstance{ID: "flaky", Translator: flaky, Available: true})

	got, err := c.TranslateWithRetry(context.Background(), "Привет", "")
	if err != nil {
		t.Fatalf("transient error must be retried on the same instance; got error: %v", err)
	}
	if got != "Recovered" {
		t.Fatalf("expected the retry to recover with %q, got %q", "Recovered", got)
	}
	if flaky.callCount() < 2 {
		t.Fatalf("expected the instance to be retried (>=2 calls), got %d call(s) "+
			"(maxRetries=3 but triedInstances blocked the retry)", flaky.callCount())
	}
}

// TestTranslateWithRetry_RetryBudgetRespectsMaxRetries asserts the maxRetries
// multiplier is actually honored: a single always-transiently-failing instance must
// be attempted maxRetries times before exhaustion (not just once). Anti-bluff: this
// pins the user-visible "we really did retry N times" behaviour.
func TestTranslateWithRetry_RetryBudgetRespectsMaxRetries(t *testing.T) {
	const maxRetries = 3
	alwaysTransient := &faultMockTranslator{script: []faultStep{
		{translation: "", err: errors.New("temporary network error")},
	}}
	c := newCoordinator(maxRetries, 0, &LLMInstance{ID: "down", Translator: alwaysTransient, Available: true})

	_, err := c.TranslateWithRetry(context.Background(), "x", "")
	if err == nil {
		t.Fatal("expected exhaustion error after all retries failed")
	}
	if alwaysTransient.callCount() != int64(maxRetries) {
		t.Fatalf("expected exactly %d attempts (maxRetries), got %d", maxRetries, alwaysTransient.callCount())
	}
}
