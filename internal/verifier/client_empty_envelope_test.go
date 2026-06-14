package verifier

import "testing"

// TestDecodeModelsResponse_NullModelsEnvelope is a §11.4.115 reproduce-first RED
// guard for the empty/null-envelope decode bug (Wave bug-hunt).
//
// FACT (root cause, client.go:147): the envelope branch was gated on
// `env.Models != nil`. The LLMsVerifier server serialises a zero-length verified
// set as a Go nil slice, which encoding/json marshals as JSON `null`
// ({"models":null,"count":0}). Unmarshalling that into `[]serverModel` leaves
// env.Models == nil, so the envelope branch was SKIPPED and the code fell through
// to the bare-array fallback `json.Unmarshal(body, &[]Model)` — but `body` is a
// JSON OBJECT, not an array, so the fallback returns
// "cannot unmarshal object into Go value of type []verifier.Model" and
// GetVerifiedModels returns a HARD ERROR instead of an empty list. The server
// reported a valid response ("count":0) yet the System could not consume it —
// /api/v1/verified-models breaks whenever zero models are verified.
//
// Contract: a valid envelope with null/empty models MUST decode to an empty,
// non-error result (the count field is authoritative; zero verified models is a
// legitimate, non-error state).
func TestDecodeModelsResponse_NullModelsEnvelope(t *testing.T) {
	// Server emits a nil slice -> JSON null for the models key.
	nullEnvelope := `{"count":0,"models":null}`
	models, err := decodeModelsResponse([]byte(nullEnvelope))
	if err != nil {
		t.Fatalf("null-models envelope must decode without error, got: %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("null-models envelope must yield 0 models, got %d", len(models))
	}

	// Explicit empty array form must also decode to zero models, no error.
	emptyEnvelope := `{"count":0,"models":[]}`
	models, err = decodeModelsResponse([]byte(emptyEnvelope))
	if err != nil {
		t.Fatalf("empty-array envelope must decode without error, got: %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("empty-array envelope must yield 0 models, got %d", len(models))
	}
}
