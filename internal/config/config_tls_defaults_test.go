package config

import (
	"os"
	"path/filepath"
	"testing"
)

// §11.4.115 RED-baseline polarity switch for the TLS-default backfill bug.
// RED_MODE=1 asserts the pre-fix behaviour (empty TLS paths after LoadConfig —
// the cause of the "open : no such file" startup crash). RED_MODE=0 (default)
// is the GREEN regression guard asserting the defaults are backfilled so the
// server can start out-of-the-box.
func tlsRedMode() bool { return os.Getenv("RED_MODE") == "1" }

// TestLoadConfig_BackfillsTLSDefaults proves a config.json that omits
// server.tls_cert_file / server.tls_key_file (exactly the committed config.json)
// is loaded with the standard certs/ paths applied, instead of leaving them
// empty (which made tls.LoadX509KeyPair("", "") fail at startup).
func TestLoadConfig_BackfillsTLSDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// A config WITHOUT any tls_cert_file / tls_key_file keys, like the shipped one.
	content := `{
  "server": {"host": "0.0.0.0", "port": 8080, "enable_http3": false},
  "security": {"enable_auth": false, "jwt_secret": "x"},
  "translation": {"default_provider": "openai", "providers": {}}
}`
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if tlsRedMode() {
		if cfg.Server.TLSCertFile != "" || cfg.Server.TLSKeyFile != "" {
			t.Fatalf("RED_MODE expects empty TLS paths (pre-fix defect), got cert=%q key=%q",
				cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
		}
		return
	}

	if cfg.Server.TLSCertFile != "certs/server.crt" {
		t.Errorf("TLSCertFile not backfilled: got %q want %q", cfg.Server.TLSCertFile, "certs/server.crt")
	}
	if cfg.Server.TLSKeyFile != "certs/server.key" {
		t.Errorf("TLSKeyFile not backfilled: got %q want %q", cfg.Server.TLSKeyFile, "certs/server.key")
	}
}

// TestLoadConfig_ExplicitTLSPathsWin proves an explicit value in the file is
// never overwritten by the default backfill.
func TestLoadConfig_ExplicitTLSPathsWin(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	content := `{
  "server": {"tls_cert_file": "/etc/ssl/my.crt", "tls_key_file": "/etc/ssl/my.key"},
  "translation": {"providers": {}}
}`
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Server.TLSCertFile != "/etc/ssl/my.crt" {
		t.Errorf("explicit cert overwritten: got %q", cfg.Server.TLSCertFile)
	}
	if cfg.Server.TLSKeyFile != "/etc/ssl/my.key" {
		t.Errorf("explicit key overwritten: got %q", cfg.Server.TLSKeyFile)
	}
}
