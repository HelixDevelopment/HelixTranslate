package main

import "testing"

// TestResolvePort_HonorsEnvVar proves the monitor server honors the
// MONITOR_SERVER_PORT environment variable that is documented in .env.example,
// CLAUDE.md, README.md (incl. the docker-compose `environment:` block) and
// docs/WebSocket_Monitoring_Guide.md as THE knob for configuring the port.
//
// REPRODUCE-FIRST (§11.4.115): before the fix, main() hardcoded `port := 8090`
// and never read os.Getenv("MONITOR_SERVER_PORT"), so a user (or a docker-compose
// deployment overriding the port) got :8090 regardless. The documented feature
// did not work. There was no resolvePort() helper at all, so this test would not
// even compile against the pre-fix tree — that is the RED.
func TestResolvePort_HonorsEnvVar(t *testing.T) {
	t.Setenv("MONITOR_SERVER_PORT", "9123")
	if got := resolvePort(); got != 9123 {
		t.Errorf("resolvePort() ignored MONITOR_SERVER_PORT: got %d, want 9123", got)
	}
}

// TestResolvePort_DefaultWhenUnset proves the documented default (8090) is used
// when the env var is absent — preserving the prior behaviour for users who do
// not set it.
func TestResolvePort_DefaultWhenUnset(t *testing.T) {
	t.Setenv("MONITOR_SERVER_PORT", "")
	if got := resolvePort(); got != 8090 {
		t.Errorf("resolvePort() default wrong: got %d, want 8090", got)
	}
}

// TestResolvePort_DefaultWhenInvalid proves a malformed value falls back to the
// documented default rather than crashing or binding port 0 (which would grab a
// random ephemeral port and silently break the documented :8090 contract).
func TestResolvePort_DefaultWhenInvalid(t *testing.T) {
	t.Setenv("MONITOR_SERVER_PORT", "not-a-number")
	if got := resolvePort(); got != 8090 {
		t.Errorf("resolvePort() invalid-value fallback wrong: got %d, want 8090", got)
	}

	t.Setenv("MONITOR_SERVER_PORT", "70000") // out of valid TCP port range
	if got := resolvePort(); got != 8090 {
		t.Errorf("resolvePort() out-of-range fallback wrong: got %d, want 8090", got)
	}
}
