package main

import "testing"

// TestApplyEnvOverrides_HonorsDocumentedEnvVars proves the environment-variable
// overrides advertised in printHelp() (GRPC_ADDRESS, GRPC_PORT, LOG_LEVEL,
// ENABLE_METRICS, ENABLE_REFLECTION) actually take effect over the flag
// defaults. Before the fix, parseFlags() never read any environment variable,
// so a deployment using the documented `GRPC_PORT=9090 grpc-server` contract
// silently bound to the flag default :50051. This test FAILS on that broken
// behavior and PASSES once the overrides are applied.
func TestApplyEnvOverrides_HonorsDocumentedEnvVars(t *testing.T) {
	// Start from the flag defaults as parseFlags() would produce them.
	config := &ServerConfig{
		Address:          "0.0.0.0",
		Port:             50051,
		MaxConnections:   1000,
		EnableReflection: true,
		EnableMetrics:    true,
		LogLevel:         "info",
	}

	env := map[string]string{
		"GRPC_ADDRESS":      "127.0.0.1",
		"GRPC_PORT":         "9090",
		"LOG_LEVEL":         "debug",
		"ENABLE_METRICS":    "false",
		"ENABLE_REFLECTION": "false",
	}
	getenv := func(k string) string { return env[k] }

	applyEnvOverrides(config, getenv)

	if config.Address != "127.0.0.1" {
		t.Errorf("GRPC_ADDRESS not honored: got Address=%q, want %q", config.Address, "127.0.0.1")
	}
	if config.Port != 9090 {
		t.Errorf("GRPC_PORT not honored: got Port=%d, want %d", config.Port, 9090)
	}
	if config.LogLevel != "debug" {
		t.Errorf("LOG_LEVEL not honored: got LogLevel=%q, want %q", config.LogLevel, "debug")
	}
	if config.EnableMetrics != false {
		t.Errorf("ENABLE_METRICS not honored: got EnableMetrics=%v, want false", config.EnableMetrics)
	}
	if config.EnableReflection != false {
		t.Errorf("ENABLE_REFLECTION not honored: got EnableReflection=%v, want false", config.EnableReflection)
	}
}

// TestApplyEnvOverrides_UnsetLeavesFlagDefaults proves an empty environment
// leaves the flag-provided config untouched (no spurious overwrite).
func TestApplyEnvOverrides_UnsetLeavesFlagDefaults(t *testing.T) {
	config := &ServerConfig{
		Address:          "10.0.0.5",
		Port:             50051,
		EnableReflection: true,
		EnableMetrics:    true,
		LogLevel:         "warn",
	}
	getenv := func(string) string { return "" }

	applyEnvOverrides(config, getenv)

	if config.Address != "10.0.0.5" || config.Port != 50051 ||
		!config.EnableReflection || !config.EnableMetrics || config.LogLevel != "warn" {
		t.Errorf("empty env must not change config: got %+v", config)
	}
}

// TestApplyEnvOverrides_MalformedNumericIgnored proves a non-numeric GRPC_PORT
// or non-bool toggle is ignored rather than zeroing the value.
func TestApplyEnvOverrides_MalformedNumericIgnored(t *testing.T) {
	config := &ServerConfig{Port: 50051, EnableMetrics: true}
	env := map[string]string{"GRPC_PORT": "not-a-number", "ENABLE_METRICS": "maybe"}
	applyEnvOverrides(config, func(k string) string { return env[k] })

	if config.Port != 50051 {
		t.Errorf("malformed GRPC_PORT must be ignored: got Port=%d, want 50051", config.Port)
	}
	if !config.EnableMetrics {
		t.Errorf("malformed ENABLE_METRICS must be ignored: got EnableMetrics=%v, want true", config.EnableMetrics)
	}
}
