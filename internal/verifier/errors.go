package verifier

import "fmt"

// ErrModelNotVerified indicates a model has not passed LLMsVerifier verification.
type ErrModelNotVerified struct {
	ModelID string
}

func (e ErrModelNotVerified) Error() string {
	return fmt.Sprintf("model %s has not passed LLMsVerifier verification", e.ModelID)
}

// ErrVerificationFailed indicates the verification pipeline failed for a model.
type ErrVerificationFailed struct {
	ModelID string
	Reason  string
}

func (e ErrVerificationFailed) Error() string {
	return fmt.Sprintf("verification failed for model %s: %s", e.ModelID, e.Reason)
}

// ErrLLMsVerifierUnreachable indicates the LLMsVerifier service is unavailable.
type ErrLLMsVerifierUnreachable struct {
	URL string
}

func (e ErrLLMsVerifierUnreachable) Error() string {
	return fmt.Sprintf("LLMsVerifier unreachable at %s", e.URL)
}

// ErrProviderNotFound indicates a requested provider is not registered.
type ErrProviderNotFound struct {
	ProviderID string
}

func (e ErrProviderNotFound) Error() string {
	return fmt.Sprintf("provider %s not found in LLMsVerifier registry", e.ProviderID)
}

// ErrScoreBelowThreshold indicates a model's score is below the minimum threshold.
type ErrScoreBelowThreshold struct {
	ModelID   string
	Score     float64
	Threshold float64
}

func (e ErrScoreBelowThreshold) Error() string {
	return fmt.Sprintf("model %s score %.2f is below threshold %.2f", e.ModelID, e.Score, e.Threshold)
}
