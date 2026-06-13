package verifier

import (
	"testing"
)

// clearAllEnvProviderKeys unsets every known provider env var for the duration
// of the test so each case starts from a clean, deterministic baseline.
func clearAllEnvProviderKeys(t *testing.T) {
	t.Helper()
	for _, spec := range envProviderSpecs {
		t.Setenv(spec.EnvVar, "")
	}
}

func TestProvidersFromEnv_OnlySetKeysProduceProviders(t *testing.T) {
	tests := []struct {
		name        string
		set         map[string]string
		wantIDs     []string
		wantBaseURL map[string]string
	}{
		{
			name:    "no keys set yields no providers",
			set:     map[string]string{},
			wantIDs: nil,
		},
		{
			name:    "single key set yields single provider",
			set:     map[string]string{"DEEPSEEK_API_KEY": "sk-test-deepseek"},
			wantIDs: []string{"deepseek"},
			wantBaseURL: map[string]string{
				"deepseek": "https://api.deepseek.com/v1",
			},
		},
		{
			name: "multiple keys yield sorted providers with correct base URLs",
			set: map[string]string{
				"GROQ_API_KEY":       "gsk-test",
				"CEREBRAS_API_KEY":   "csk-test",
				"OPENROUTER_API_KEY": "or-test",
				"MISTRAL_API_KEY":    "ms-test",
			},
			// sorted by ID: cerebras, groq, mistral, openrouter
			wantIDs: []string{"cerebras", "groq", "mistral", "openrouter"},
			wantBaseURL: map[string]string{
				"cerebras":   "https://api.cerebras.ai/v1",
				"groq":       "https://api.groq.com/openai/v1",
				"mistral":    "https://api.mistral.ai/v1",
				"openrouter": "https://openrouter.ai/api/v1",
			},
		},
		{
			name: "empty-string key is treated as unset",
			set: map[string]string{
				"DEEPSEEK_API_KEY": "",
				"GROQ_API_KEY":     "gsk-real",
			},
			wantIDs: []string{"groq"},
		},
		{
			name:    "gemini openai-compat subpath base url",
			set:     map[string]string{"GEMINI_API_KEY": "gm-test"},
			wantIDs: []string{"gemini"},
			wantBaseURL: map[string]string{
				"gemini": "https://generativelanguage.googleapis.com/v1beta/openai",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAllEnvProviderKeys(t)
			for k, v := range tt.set {
				t.Setenv(k, v)
			}

			got := ProvidersFromEnv()

			gotIDs := make([]string, 0, len(got))
			for _, c := range got {
				gotIDs = append(gotIDs, c.ID)
			}

			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("provider IDs = %v, want %v", gotIDs, tt.wantIDs)
			}
			for i := range gotIDs {
				if gotIDs[i] != tt.wantIDs[i] {
					t.Fatalf("provider IDs = %v, want %v", gotIDs, tt.wantIDs)
				}
			}

			// Every returned config MUST carry a non-empty APIKey and BaseURL —
			// a provider with a missing key must never be emitted.
			for _, c := range got {
				if c.APIKey == "" {
					t.Errorf("provider %s returned with empty APIKey (must be omitted, not blank)", c.ID)
				}
				if c.BaseURL == "" {
					t.Errorf("provider %s returned with empty BaseURL", c.ID)
				}
				if want, ok := tt.wantBaseURL[c.ID]; ok && c.BaseURL != want {
					t.Errorf("provider %s BaseURL = %q, want %q", c.ID, c.BaseURL, want)
				}
			}
		})
	}
}

func TestProvidersFromEnv_KeyValueCarriedThrough(t *testing.T) {
	clearAllEnvProviderKeys(t)
	t.Setenv("DEEPSEEK_API_KEY", "sk-secret-value-123")

	got := ProvidersFromEnv()
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 provider, got %d", len(got))
	}
	if got[0].APIKey != "sk-secret-value-123" {
		t.Errorf("APIKey not carried through from env var")
	}
}

func TestProvidersFromEnv_IsDeterministicallySorted(t *testing.T) {
	clearAllEnvProviderKeys(t)
	t.Setenv("ZHIPU_API_KEY", "z")
	t.Setenv("CEREBRAS_API_KEY", "c")
	t.Setenv("OPENAI_API_KEY", "o")

	a := EnvProviderIDs()
	b := EnvProviderIDs()
	if len(a) != 3 {
		t.Fatalf("expected 3 IDs, got %v", a)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("EnvProviderIDs not deterministic: %v vs %v", a, b)
		}
	}
	// Sorted ascending.
	if !(a[0] <= a[1] && a[1] <= a[2]) {
		t.Errorf("EnvProviderIDs not sorted: %v", a)
	}
}
