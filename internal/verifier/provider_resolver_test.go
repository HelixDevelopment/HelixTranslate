package verifier

import (
	"os"
	"strings"
	"testing"
)

// fakeEnv returns a getenv-style function backed by an in-memory map so tests
// never read or mutate the real process environment (§11.4.10).
func fakeEnv(kv map[string]string) func(string) string {
	return func(k string) string { return kv[k] }
}

func TestProviderResolver_NamedID(t *testing.T) {
	r := NewProviderResolverWithEnv(fakeEnv(map[string]string{
		"DEEPSEEK_API_KEY": "sk-test-deepseek",
	}))

	got, err := r.Resolve("deepseek")
	if err != nil {
		t.Fatalf("Resolve(deepseek) unexpected error: %v", err)
	}
	if got.FactoryProvider != "openai" {
		t.Errorf("FactoryProvider = %q, want openai (generic OpenAI-compatible)", got.FactoryProvider)
	}
	if got.BaseURL != "https://api.deepseek.com/v1" {
		t.Errorf("BaseURL = %q, want https://api.deepseek.com/v1", got.BaseURL)
	}
	if got.APIKey != "sk-test-deepseek" {
		t.Errorf("APIKey not threaded from env")
	}
	if got.EnvVar != "DEEPSEEK_API_KEY" {
		t.Errorf("EnvVar = %q, want DEEPSEEK_API_KEY", got.EnvVar)
	}
}

// TestProviderResolver_AllNamedProviders asserts every provider in the canonical
// envProviderSpecs table resolves (with its key present) to a materializable
// (factoryProvider, baseURL) pair. This is the §3.3 "EVERY verified provider
// materializes" contract.
func TestProviderResolver_AllNamedProviders(t *testing.T) {
	kv := map[string]string{}
	for _, spec := range envProviderSpecs {
		kv[spec.EnvVar] = "key-for-" + spec.ID
	}
	r := NewProviderResolverWithEnv(fakeEnv(kv))

	for _, spec := range envProviderSpecs {
		got, err := r.Resolve(spec.ID)
		if err != nil {
			t.Errorf("Resolve(%s) error: %v", spec.ID, err)
			continue
		}
		if got.FactoryProvider == "" {
			t.Errorf("Resolve(%s) empty FactoryProvider", spec.ID)
		}
		if got.BaseURL != spec.BaseURL {
			t.Errorf("Resolve(%s) BaseURL = %q, want %q", spec.ID, got.BaseURL, spec.BaseURL)
		}
		if got.APIKey != "key-for-"+spec.ID {
			t.Errorf("Resolve(%s) APIKey not threaded", spec.ID)
		}
	}
}

// TestProviderResolver_NumericID_HTTPPath is the §11.4.115 RED-then-GREEN
// regression for the §3.3 part-1 load-bearing bug: a verified model selected via
// the HTTP server path carries a NUMERIC ProviderID (client.go emits
// fmt.Sprintf("%d", sm.ProviderID)) that the factory switch cannot map. The
// resolver MUST map a numeric id by index into the deterministic
// envProviderSpecs list so the model still materializes.
//
// RED_MODE=1 (default) asserts the bug is FIXED: numeric "1" resolves to
// envProviderSpecs[1]. Set RED_MODE=1 against an unfixed resolver (one that only
// handled named ids) and this test would FAIL — proving it catches the defect.
func TestProviderResolver_NumericID_HTTPPath(t *testing.T) {
	redMode := os.Getenv("RED_MODE") != "0" // default 1 (assert fixed)

	idx := 1
	want := envProviderSpecs[idx]
	kv := map[string]string{want.EnvVar: "numeric-key"}
	r := NewProviderResolverWithEnv(fakeEnv(kv))

	got, err := r.Resolve("1")
	if redMode {
		if err != nil {
			t.Fatalf("numeric ProviderID \"1\" failed to resolve (the §3.3 bug is present): %v", err)
		}
		if got.BaseURL != want.BaseURL {
			t.Errorf("numeric \"1\" resolved to BaseURL %q, want %q (envProviderSpecs[1]=%s)",
				got.BaseURL, want.BaseURL, want.ID)
		}
		if got.FactoryProvider == "" {
			t.Errorf("numeric \"1\" resolved with empty FactoryProvider — not materializable")
		}
	}
}

func TestProviderResolver_NumericID_OutOfRange(t *testing.T) {
	r := NewProviderResolverWithEnv(fakeEnv(map[string]string{}))
	_, err := r.Resolve("9999")
	if err == nil {
		t.Fatal("out-of-range numeric ProviderID should error, got nil")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("error = %q, want it to mention out of range", err.Error())
	}
}

func TestProviderResolver_UnknownID(t *testing.T) {
	r := NewProviderResolverWithEnv(fakeEnv(map[string]string{}))
	_, err := r.Resolve("not-a-real-provider")
	if err == nil {
		t.Fatal("unknown provider id should error, got nil")
	}
	if !strings.Contains(err.Error(), "not a known provider") {
		t.Errorf("error = %q, want it to say not a known provider", err.Error())
	}
}

func TestProviderResolver_EmptyID(t *testing.T) {
	r := NewProviderResolverWithEnv(fakeEnv(map[string]string{}))
	_, err := r.Resolve("   ")
	if err == nil {
		t.Fatal("empty provider id should error, got nil")
	}
}

// TestProviderResolver_MissingKey: a known provider whose env key is unset
// returns the resolved metadata AND an honest error naming the env var — never
// a silent empty-key client (§11.4.6 / §11.4.10).
func TestProviderResolver_MissingKey(t *testing.T) {
	r := NewProviderResolverWithEnv(fakeEnv(map[string]string{})) // no keys
	got, err := r.Resolve("openai")
	if err == nil {
		t.Fatal("missing key should error, got nil")
	}
	if got.APIKey != "" {
		t.Errorf("APIKey should be empty when key unset, got non-empty")
	}
	if !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Errorf("error should name the missing env var, got %q", err.Error())
	}
}
