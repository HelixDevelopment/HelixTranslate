package verifier

import "testing"

// §11.4.135 standing regression guard for the load-bearing LLMsVerifier
// SSOT-consumption decode (Bug B, 2026-06-13): the verifier server's
// GET /api/models returns an ENVELOPE object {"count":N,"models":[...]} whose
// per-model field names differ from api.Model (model_id / status / score, plus
// an integer "id" DB key). The original client did a bare
// json.Decode(&[]Model) and could not decode the envelope at all, so the System
// consumed ZERO verified models while the server reported them healthy — a
// §11.4 SSOT-layer bluff. This fixture is the REAL captured server payload
// shape (docs/qa/llmsverifier_20260613_182600/api_models_head.json, key-free).
//
// Mutation-proven: reverting decodeModelsResponse to the bare-array decode makes
// this test FAIL (envelope unmarshal yields nil models / type mismatch on "id").
const realServerModelsEnvelope = `{
  "count": 3,
  "models": [
    {"id":20,"model_id":"allam-2-7b","name":"allam-2-7b","provider":"Groq","provider_id":3,"status":"verified","score":94,"capabilities":["text"],"version":"live","deprecated":false},
    {"id":5,"model_id":"canopylabs/orpheus-v1-english","name":"canopylabs/orpheus-v1-english","provider":"Groq","provider_id":3,"status":"verified","score":94,"capabilities":["text"],"version":"live","deprecated":false},
    {"id":2,"model_id":"deepseek-v4-flash","name":"deepseek-v4-flash","provider":"DeepSeek","provider_id":1,"status":"verified","score":95,"capabilities":["text","code"],"version":"live","deprecated":false}
  ]
}`

func TestDecodeModelsResponse_RealServerEnvelope(t *testing.T) {
	models, err := decodeModelsResponse([]byte(realServerModelsEnvelope))
	if err != nil {
		t.Fatalf("decode of real server envelope failed: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("expected 3 models from envelope, got %d (envelope not decoded — Bug B regressed)", len(models))
	}

	// model_id (NOT the integer "id") must become Model.ID — the System selects by it.
	byID := map[string]Model{}
	for _, m := range models {
		byID[m.ID] = m
	}
	got, ok := byID["deepseek-v4-flash"]
	if !ok {
		t.Fatalf("model_id was not mapped to Model.ID; got IDs %v", keysOf(byID))
	}
	if got.VerificationStatus != "verified" {
		t.Errorf("status->VerificationStatus mismatch: got %q want %q", got.VerificationStatus, "verified")
	}
	if got.OverallScore != 95 {
		t.Errorf("score->OverallScore mismatch: got %v want 95", got.OverallScore)
	}
	if !got.Capabilities["code"] || !got.Capabilities["text"] {
		t.Errorf("capabilities[] not mapped to set: %v", got.Capabilities)
	}
	if got.ProviderID != "1" {
		t.Errorf("provider_id->ProviderID mismatch: got %q want %q", got.ProviderID, "1")
	}
}

// TestDecodeModelsResponse_BareArrayFallback proves the forward-compatible path
// (a bare []api.Model array) still decodes, so a future server that drops the
// envelope does not break the System.
func TestDecodeModelsResponse_BareArrayFallback(t *testing.T) {
	bare := `[{"id":"gpt-x","verification_status":"verified","overall_score":88}]`
	models, err := decodeModelsResponse([]byte(bare))
	if err != nil {
		t.Fatalf("bare-array fallback failed: %v", err)
	}
	if len(models) != 1 || models[0].ID != "gpt-x" || models[0].OverallScore != 88 {
		t.Fatalf("bare-array decode wrong: %+v", models)
	}
}

func keysOf(m map[string]Model) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
