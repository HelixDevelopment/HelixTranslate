package verifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineVerify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	pipeline := NewPipeline()
	provider := ProviderConfig{
		ID:      "test-provider",
		APIKey:  "test-key",
		BaseURL: server.URL,
		Models:  []string{"test-model"},
	}

	result := pipeline.Verify(context.Background(), provider, "test-model")

	require.NotNil(t, result)
	assert.Equal(t, "test-model", result.ModelID)
	assert.Equal(t, "test-provider", result.Provider)
	assert.Len(t, result.Steps, 8)
	assert.True(t, result.Passed)
	assert.Greater(t, result.Overall, 0.0)

	// Verify all 8 steps are present
	stepNames := make([]string, len(result.Steps))
	for i, s := range result.Steps {
		stepNames[i] = s.Step
	}
	assert.Contains(t, stepNames, "reachability")
	assert.Contains(t, stepNames, "authentication")
	assert.Contains(t, stepNames, "model_existence")
	assert.Contains(t, stepNames, "response_format")
	assert.Contains(t, stepNames, "latency")
	assert.Contains(t, stepNames, "capabilities")
	assert.Contains(t, stepNames, "rate_limits")
	assert.Contains(t, stepNames, "error_handling")
}

func TestPipelineVerifyUnreachable(t *testing.T) {
	pipeline := NewPipeline()
	provider := ProviderConfig{
		ID:      "unreachable",
		APIKey:  "test-key",
		BaseURL: "http://localhost:1",
		Models:  []string{"test-model"},
	}

	result := pipeline.Verify(context.Background(), provider, "test-model")

	require.NotNil(t, result)
	assert.False(t, result.Passed)
	assert.Less(t, result.Overall, 0.5)

	// Reachability should fail
	reachability := result.Steps[0]
	assert.Equal(t, "reachability", reachability.Step)
	assert.False(t, reachability.Passed)
}

func TestPipelineVerifyNoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	pipeline := NewPipeline()
	provider := ProviderConfig{
		ID:      "no-key",
		BaseURL: server.URL,
		Models:  []string{"test-model"},
	}

	result := pipeline.Verify(context.Background(), provider, "test-model")

	require.NotNil(t, result)
	assert.True(t, result.Passed)

	// Authentication should pass but with reduced score
	auth := result.Steps[1]
	assert.Equal(t, "authentication", auth.Step)
	assert.True(t, auth.Passed)
	assert.Equal(t, 0.5, auth.Score)
}

func TestPipelineVerifyEmptyModelID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	pipeline := NewPipeline()
	provider := ProviderConfig{
		ID:      "test",
		BaseURL: server.URL,
	}

	result := pipeline.Verify(context.Background(), provider, "")

	require.NotNil(t, result)
	// Model existence should fail
	existence := result.Steps[2]
	assert.Equal(t, "model_existence", existence.Step)
	assert.False(t, existence.Passed)
}

func TestCheckReachability(t *testing.T) {
	pipeline := NewPipeline()

	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		result := pipeline.checkReachability(context.Background(), ProviderConfig{BaseURL: server.URL})
		assert.True(t, result.Passed)
		assert.Equal(t, 1.0, result.Score)
	})

	t.Run("unreachable", func(t *testing.T) {
		result := pipeline.checkReachability(context.Background(), ProviderConfig{BaseURL: "http://localhost:1"})
		assert.False(t, result.Passed)
		assert.Equal(t, 0.0, result.Score)
	})
}

func TestCheckAuthentication(t *testing.T) {
	pipeline := NewPipeline()

	t.Run("valid_key", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "Bearer valid-key" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}
		}))
		defer server.Close()

		result := pipeline.checkAuthentication(context.Background(), ProviderConfig{BaseURL: server.URL, APIKey: "valid-key"})
		assert.True(t, result.Passed)
		assert.Equal(t, 1.0, result.Score)
	})

	t.Run("invalid_key", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		result := pipeline.checkAuthentication(context.Background(), ProviderConfig{BaseURL: server.URL, APIKey: "invalid-key"})
		assert.False(t, result.Passed)
		assert.Equal(t, 0.0, result.Score)
	})
}

func TestMeasureLatency(t *testing.T) {
	pipeline := NewPipeline()

	t.Run("fast", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		result := pipeline.measureLatency(context.Background(), ProviderConfig{BaseURL: server.URL}, "model")
		assert.True(t, result.Passed)
		assert.Equal(t, 1.0, result.Score)
		assert.Less(t, result.LatencyMs, int64(1000))
	})
}
