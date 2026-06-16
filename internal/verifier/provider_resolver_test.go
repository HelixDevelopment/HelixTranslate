package verifier

import (
	"errors"
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

// TestProviderResolver_MissingKey_IsTyped: the missing-key error MUST classify
// as ErrProviderKeyMissing (NOT ErrProviderUnknown), so a caller holding a key
// from another source (config.json) can distinguish "key not in env" from
// "provider unknown" and proceed instead of re-routing through the rejecting
// whitelist (the §11.4.138 review gap).
func TestProviderResolver_MissingKey_IsTyped(t *testing.T) {
	r := NewProviderResolverWithEnv(fakeEnv(map[string]string{})) // no keys
	_, err := r.Resolve("novita")
	if err == nil {
		t.Fatal("missing key should error, got nil")
	}
	if !errors.Is(err, ErrProviderKeyMissing) {
		t.Errorf("missing-key error must be ErrProviderKeyMissing, got %v", err)
	}
	if errors.Is(err, ErrProviderUnknown) {
		t.Errorf("a KNOWN provider with a missing key must NOT classify as ErrProviderUnknown: %v", err)
	}
}

// TestProviderResolver_Unknown_IsTyped: a genuinely unknown / out-of-range id
// MUST classify as ErrProviderUnknown so the caller falls back to the standard
// constructor.
func TestProviderResolver_Unknown_IsTyped(t *testing.T) {
	r := NewProviderResolverWithEnv(fakeEnv(map[string]string{}))
	for _, id := range []string{"not-a-real-provider", "9999", "  "} {
		_, err := r.Resolve(id)
		if err == nil {
			t.Fatalf("Resolve(%q) should error, got nil", id)
		}
		if !errors.Is(err, ErrProviderUnknown) {
			t.Errorf("Resolve(%q) error must be ErrProviderUnknown, got %v", id, err)
		}
		if errors.Is(err, ErrProviderKeyMissing) {
			t.Errorf("Resolve(%q) unknown provider must NOT classify as key-missing: %v", id, err)
		}
	}
}

// TestProviderResolver_ResolveProvider_KeyAgnostic: ResolveProvider materializes
// a KNOWN provider with NO error even when its env key is unset (key-agnostic),
// populating BaseURL/EnvVar/FactoryProvider — the seam buildVerifiedTranslator
// uses to keep a config-keyed-but-env-unset provider OFF the whitelist path.
func TestProviderResolver_ResolveProvider_KeyAgnostic(t *testing.T) {
	r := NewProviderResolverWithEnv(fakeEnv(map[string]string{})) // no keys
	got, err := r.ResolveProvider("novita")
	if err != nil {
		t.Fatalf("ResolveProvider(novita) must NOT error on a known provider with unset key: %v", err)
	}
	if got.BaseURL == "" || got.FactoryProvider == "" || got.EnvVar != "NOVITA_API_KEY" {
		t.Errorf("ResolveProvider(novita) did not fully materialize: %+v", got)
	}
	if got.APIKey != "" {
		t.Errorf("APIKey should be empty when env key unset, got non-empty")
	}

	// And it still errors (ErrProviderUnknown) for a genuinely unknown id.
	if _, err := r.ResolveProvider("not-a-real-provider"); !errors.Is(err, ErrProviderUnknown) {
		t.Errorf("ResolveProvider(unknown) must classify as ErrProviderUnknown, got %v", err)
	}
}
