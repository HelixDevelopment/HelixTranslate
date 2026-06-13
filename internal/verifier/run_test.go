package verifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// memSink is an in-memory ModelSink for unit testing RunVerification without a
// real database (unit-test scope, mocks permitted per §11.4.27).
type memSink struct {
	mu     sync.Mutex
	models map[string]Model
}

func newMemSink() *memSink { return &memSink{models: map[string]Model{}} }

func (s *memSink) SaveModel(m Model) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.models[m.ID] = m
	return nil
}

func (s *memSink) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.models)
}

// fakeOpenAIServer stands in for an OpenAI-compatible provider so the unit test
// exercises the REAL RunVerification + Pipeline HTTP code paths deterministically.
func fakeOpenAIServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{{"id": "model-a"}, {"id": "model-b"}},
		})
	})
	mux.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		body, _ := json.Marshal(map[string]any{})
		var req struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(readAll(r), &req)
		if strings.HasPrefix(req.Model, "invalid-model") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(body)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "Hello!"}},
			},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func readAll(r *http.Request) []byte {
	if r.Body == nil {
		return nil
	}
	buf := make([]byte, 0, 512)
	tmp := make([]byte, 256)
	for {
		n, err := r.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return buf
}

func TestRunVerification_PersistsVerifiedModels(t *testing.T) {
	srv := fakeOpenAIServer(t)

	reg := NewRegistry()
	pipe := NewPipeline()
	sink := newMemSink()

	providers := []ProviderConfig{
		{ID: "fakeai", APIKey: "test-key", BaseURL: srv.URL},
	}

	res, err := RunVerification(context.Background(), reg, pipe, sink, providers, RunOptions{})
	if err != nil {
		t.Fatalf("RunVerification error: %v", err)
	}

	if len(res.Providers) != 1 {
		t.Fatalf("expected 1 provider result, got %d", len(res.Providers))
	}
	pv := res.Providers[0]
	if !pv.ReachabilityPass {
		t.Errorf("expected reachability pass against fake server")
	}
	if !pv.AuthPass {
		t.Errorf("expected auth pass with valid key, status=%d err=%q", pv.AuthStatusCode, pv.AuthError)
	}
	if len(pv.CandidateModels) != 2 {
		t.Fatalf("expected 2 candidate models discovered from /models, got %v", pv.CandidateModels)
	}
	// Both models return valid chat completions → both verified + persisted.
	if pv.VerifiedCount != 2 {
		t.Errorf("expected 2 verified models, got %d", pv.VerifiedCount)
	}
	if res.TotalVerified != 2 {
		t.Errorf("expected TotalVerified=2, got %d", res.TotalVerified)
	}
	if sink.count() != 2 {
		t.Errorf("expected 2 persisted models, got %d", sink.count())
	}
}

func TestRunVerification_BadKeyAuthFailsNoPersist(t *testing.T) {
	srv := fakeOpenAIServer(t)

	reg := NewRegistry()
	pipe := NewPipeline()
	sink := newMemSink()

	// Wrong key → /models returns 401 → auth fails → no candidates, no persist.
	providers := []ProviderConfig{
		{ID: "fakeai", APIKey: "wrong-key", BaseURL: srv.URL},
	}

	res, err := RunVerification(context.Background(), reg, pipe, sink, providers, RunOptions{})
	if err != nil {
		t.Fatalf("RunVerification error: %v", err)
	}
	pv := res.Providers[0]
	if pv.AuthPass {
		t.Errorf("expected auth FAIL with wrong key, got pass (status=%d)", pv.AuthStatusCode)
	}
	if pv.AuthStatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", pv.AuthStatusCode)
	}
	if len(pv.CandidateModels) != 0 {
		t.Errorf("expected 0 candidate models on auth failure (no fabrication), got %v", pv.CandidateModels)
	}
	if sink.count() != 0 {
		t.Errorf("expected 0 persisted models on auth failure, got %d", sink.count())
	}
}

func TestRunVerification_EmptyBaseURLSkipped(t *testing.T) {
	reg := NewRegistry()
	pipe := NewPipeline()
	sink := newMemSink()
	providers := []ProviderConfig{{ID: "nourl", APIKey: "k", BaseURL: ""}}

	res, err := RunVerification(context.Background(), reg, pipe, sink, providers, RunOptions{})
	if err != nil {
		t.Fatalf("RunVerification error: %v", err)
	}
	if res.Providers[0].ReachabilityPass {
		t.Errorf("empty base URL must not be reachable")
	}
	if sink.count() != 0 {
		t.Errorf("expected nothing persisted for empty base URL")
	}
}

func TestRunVerification_MaxModelsCap(t *testing.T) {
	srv := fakeOpenAIServer(t)
	reg := NewRegistry()
	pipe := NewPipeline()
	sink := newMemSink()
	providers := []ProviderConfig{{ID: "fakeai", APIKey: "test-key", BaseURL: srv.URL}}

	res, err := RunVerification(context.Background(), reg, pipe, sink, providers,
		RunOptions{MaxModelsPerProvider: 1})
	if err != nil {
		t.Fatalf("RunVerification error: %v", err)
	}
	if len(res.Providers[0].CandidateModels) != 1 {
		t.Errorf("expected cap to 1 candidate, got %v", res.Providers[0].CandidateModels)
	}
}
