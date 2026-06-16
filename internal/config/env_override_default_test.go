package config

import (
	"testing"
)

// TestApplyEnvOverrides_JWTSecretReachesDefaultConfig is the permanent §11.4.135
// regression guard for the create-default-config-skips-env-overrides defect
// discovered by heavy real-service testing against the live nezha stack
// (2026-06-16): cmd/server's loadOrCreateConfig returned a bare DefaultConfig()
// (EnableAuth=true, JWTSecret="") WITHOUT applying environment overrides when no
// config.json existed, so a freshly-deployed server crash-looped with
// "JWT secret is required when authentication is enabled" despite JWT_SECRET
// being correctly present in the container env.
//
// Root cause (FACT from podman logs on nezha helixtranslate-server): the
// create-default branch never ran the env-loader, so the env JWT_SECRET never
// reached c.Security.JWTSecret before Validate().
//
// RED (pre-fix): ApplyEnvOverrides did not exist / was not called → a
// DefaultConfig validated with JWT_SECRET set in env would still fail Validate.
func TestApplyEnvOverrides_JWTSecretReachesDefaultConfig(t *testing.T) {
	t.Setenv("JWT_SECRET", "a-real-64-char-secret-from-env-nezha-not-the-placeholder-value-x")

	cfg := DefaultConfig()
	if !cfg.Security.EnableAuth {
		t.Fatalf("precondition: DefaultConfig must have EnableAuth=true (got false)")
	}
	if cfg.Security.JWTSecret != "" {
		t.Fatalf("precondition: DefaultConfig must have empty JWTSecret (got non-empty)")
	}

	// This is exactly what loadOrCreateConfig's create-default branch must do.
	cfg.ApplyEnvOverrides()

	if cfg.Security.JWTSecret == "" {
		t.Fatalf("ApplyEnvOverrides did not apply JWT_SECRET from env to DefaultConfig")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("DefaultConfig + ApplyEnvOverrides must validate when JWT_SECRET is in env, got: %v", err)
	}
}
