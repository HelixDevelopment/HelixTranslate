package verifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestAffirmativeResponseIsHardGate is a §11.4.115 RED-baseline-on-the-broken-artifact
// reproduction of the verification-gate correctness bluff: a model whose
// affirmative-response probe FAILED (the "response_format" step, the step the
// persisted Model.AffirmativeResponse field is derived from in run.go) was still
// marked Passed/"verified" because the gate (pipeline.go Verify) only treated
// reachability + authentication as hard gates and folded response_format into the
// averaged Overall score, which other high-scoring steps could carry above the
// 0.5 threshold.
//
// Polarity switch (§11.4.115): the test reproduces the defect on the CURRENT
// (pre-fix) code by default (RED_MODE unset / "1"). Set RED_MODE=0 after the fix
// lands to flip it into the standing GREEN regression guard asserting the defect
// is ABSENT.
//
//   - RED_MODE=1 (default): asserts the buggy behaviour is PRESENT (mv.Passed==true
//     while affirmative_response failed). On the FIXED code this assertion fails —
//     which is the proof the fix took effect.
//   - RED_MODE=0: GREEN guard — asserts a model with affirmative_response failed is
//     NOT Passed (cannot be verified). This MUST hold after the fix.
func TestAffirmativeResponseIsHardGate(t *testing.T) {
	// A provider whose /models, auth (200), and /chat/completions all SUCCEED at the
	// transport layer, but /chat/completions returns an EMPTY assistant message —
	// the affirmative-response probe (response_format step) fails (Passed=false),
	// while every other step passes with high scores. This is the exact shape that
	// let a non-affirmatively-responding model slip through the gate as "verified".
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			// Rate-limit headers present so the rate_limits step scores 1.0,
			// maximising the non-affirmative score so the averaged Overall clears 0.5.
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", "99")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]string{{"id": "empty-responder"}},
			})
		case "/chat/completions":
			var reqBody map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&reqBody)
			model, _ := reqBody["model"].(string)
			// error_handling step probes an invalid model and expects a graceful 4xx.
			if model == "invalid-model-id-00000000" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid model"})
				return
			}
			// 200 OK but EMPTY content => response_format (affirmative_response) FAILS
			// (pipeline.validateResponseFormat: empty content => Passed:false, Score:0.5).
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []map[string]interface{}{
					{"message": map[string]string{"content": ""}},
				},
			})
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
		}
	}))
	defer server.Close()

	pipeline := NewPipeline()
	provider := ProviderConfig{ID: "empty-provider", BaseURL: server.URL, APIKey: "test-key"}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mv := pipeline.Verify(ctx, provider, "empty-responder")

	affirmative := responseFormatPassed(mv)
	if affirmative {
		t.Fatalf("test setup invalid: expected affirmative_response (response_format step) to FAIL, "+
			"but it passed; steps=%+v", mv.Steps)
	}

	redMode := os.Getenv("RED_MODE") != "0" // default RED (reproduce the defect)

	if redMode {
		// RED: prove the bug exists on the current artifact — a model that did NOT
		// affirmatively respond is nonetheless reported as Passed (and would be
		// persisted as "verified" by run.go).
		if !mv.Passed {
			t.Fatalf("RED expectation not met: model with affirmative_response=FAILED was already "+
				"rejected (mv.Passed=false, overall=%.4f) — the defect appears absent. "+
				"If the fix has landed, re-run with RED_MODE=0 to use the GREEN guard.", mv.Overall)
		}
		t.Logf("RED reproduced: model with affirmative_response=FAILED was marked Passed "+
			"(mv.Passed=true, overall=%.4f) — verification gate bluff present.", mv.Overall)
		return
	}

	// GREEN guard: after the fix, affirmative_response=FAILED MUST disqualify the
	// model regardless of how high the other components score.
	if mv.Passed {
		t.Fatalf("GREEN guard FAILED: model with affirmative_response=FAILED was marked Passed "+
			"(overall=%.4f). A zero/failed affirmative response MUST be a hard disqualifier.", mv.Overall)
	}
}
