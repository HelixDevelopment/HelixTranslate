package sshworker

import (
	"context"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/logger"
	"golang.org/x/crypto/ssh"
)

// newTestWorker builds an SSHWorker with a quiet logger and no live client.
// All tests using it exercise the disconnected / pure-logic paths only.
func newTestWorker(t *testing.T, cfg SSHWorkerConfig) *SSHWorker {
	t.Helper()
	w, err := NewSSHWorker(cfg, logger.NewLogger(logger.LoggerConfig{}))
	if err != nil {
		t.Fatalf("NewSSHWorker returned error: %v", err)
	}
	if w == nil {
		t.Fatal("NewSSHWorker returned nil worker")
	}
	return w
}

// TestNewSSHWorker_FieldMapping asserts the constructor copies config fields
// onto the worker. Anti-bluff: if the constructor were stubbed to return an
// empty &SSHWorker{}, GetRemoteDir() and the port-based Connect error below
// would no longer reflect the supplied config and these assertions fail.
func TestNewSSHWorker_FieldMapping(t *testing.T) {
	cfg := SSHWorkerConfig{
		Host:      "example.test",
		Username:  "deploy",
		Password:  "secret",
		Port:      2222,
		RemoteDir: "/srv/translator",
	}
	w := newTestWorker(t, cfg)

	if got := w.GetRemoteDir(); got != "/srv/translator" {
		t.Fatalf("GetRemoteDir() = %q, want %q", got, "/srv/translator")
	}
	if w.config.Host != "example.test" {
		t.Fatalf("config.Host = %q, want example.test", w.config.Host)
	}
	if w.config.Username != "deploy" {
		t.Fatalf("config.Username = %q, want deploy", w.config.Username)
	}
}

func TestGetRemoteDir_TableDriven(t *testing.T) {
	cases := []struct {
		name string
		dir  string
	}{
		{"absolute", "/home/worker/app"},
		{"empty", ""},
		{"trailing-slash", "/tmp/x/"},
		{"relative", "work/dir"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newTestWorker(t, SSHWorkerConfig{RemoteDir: tc.dir})
			if got := w.GetRemoteDir(); got != tc.dir {
				t.Fatalf("GetRemoteDir() = %q, want %q", got, tc.dir)
			}
		})
	}
}

// TestConnect_InvalidPort proves the explicit port-range validation in Connect
// fires before any network dial. Anti-bluff: if the range check (port<1||>65535)
// were removed, Connect would attempt a TCP dial to an absurd port and fail with
// a different (network) error, so the substring assertion below would fail.
func TestConnect_InvalidPort(t *testing.T) {
	cases := []struct {
		name string
		port int
	}{
		{"zero", 0},
		{"negative", -5},
		{"too-large", 70000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Password set so auth-method check passes and we reach port validation.
			w := newTestWorker(t, SSHWorkerConfig{Host: "127.0.0.1", Password: "pw", Port: tc.port})
			err := w.Connect(context.Background())
			if err == nil {
				t.Fatalf("Connect with port %d expected error, got nil", tc.port)
			}
			if !strings.Contains(err.Error(), "invalid port number") {
				t.Fatalf("Connect error = %q, want it to mention 'invalid port number'", err.Error())
			}
		})
	}
}

// TestConnect_NoAuthMethod proves Connect rejects when neither a key nor a
// password is available, before validating the port or dialing.
// Anti-bluff: a stub returning nil would let nil through here.
func TestConnect_NoAuthMethod(t *testing.T) {
	t.Setenv("SSH_PRIVATE_KEY_PATH", "") // ensure no key path leaks in from env
	w := newTestWorker(t, SSHWorkerConfig{Host: "127.0.0.1", Port: 22, Password: ""})
	err := w.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect with no auth method expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no authentication method") {
		t.Fatalf("Connect error = %q, want 'no authentication method'", err.Error())
	}
}

// TestNilClientGuards proves every SSH-I/O entry point refuses cleanly when the
// client was never connected (the "SSH client not connected" guard).
// Anti-bluff: removing any guard makes the method nil-dereference panic or
// return a nil error, failing these assertions. These run WITHOUT any daemon.
func TestNilClientGuards(t *testing.T) {
	w := newTestWorker(t, SSHWorkerConfig{Host: "h", Port: 22})
	ctx := context.Background()

	t.Run("ExecuteCommand", func(t *testing.T) {
		res, err := w.ExecuteCommand(ctx, "echo hi")
		if err == nil || res != nil {
			t.Fatalf("ExecuteCommand: want (nil,err), got (%v,%v)", res, err)
		}
		if !strings.Contains(err.Error(), "not connected") {
			t.Fatalf("ExecuteCommand err = %q, want 'not connected'", err.Error())
		}
	})

	t.Run("ExecuteCommandWithOutput", func(t *testing.T) {
		out, err := w.ExecuteCommandWithOutput(ctx, "echo hi")
		if err == nil {
			t.Fatal("ExecuteCommandWithOutput: want error, got nil")
		}
		if out != "" {
			t.Fatalf("ExecuteCommandWithOutput: want empty output, got %q", out)
		}
	})

	t.Run("UploadFile", func(t *testing.T) {
		// Local file must exist would be checked later; the connection guard is first.
		err := w.UploadFile(ctx, "/nonexistent/local", "/remote")
		if err == nil || !strings.Contains(err.Error(), "not connected") {
			t.Fatalf("UploadFile err = %v, want 'not connected'", err)
		}
	})

	t.Run("DownloadFile", func(t *testing.T) {
		err := w.DownloadFile(ctx, "/remote", "/tmp/local")
		if err == nil || !strings.Contains(err.Error(), "not connected") {
			t.Fatalf("DownloadFile err = %v, want 'not connected'", err)
		}
	})

	t.Run("TransferFile_aliasesUpload", func(t *testing.T) {
		err := w.TransferFile(ctx, "/nonexistent/local", "/remote")
		if err == nil || !strings.Contains(err.Error(), "not connected") {
			t.Fatalf("TransferFile err = %v, want 'not connected'", err)
		}
	})

	t.Run("TransferFileFromRemote_aliasesDownload", func(t *testing.T) {
		err := w.TransferFileFromRemote(ctx, "/remote", "/tmp/local")
		if err == nil || !strings.Contains(err.Error(), "not connected") {
			t.Fatalf("TransferFileFromRemote err = %v, want 'not connected'", err)
		}
	})
}

// TestDisconnectAndCloseNilClient proves the nil-client fast path returns nil
// (no panic, no spurious error) on a never-connected worker.
func TestDisconnectAndCloseNilClient(t *testing.T) {
	w := newTestWorker(t, SSHWorkerConfig{Host: "h", Port: 22})
	if err := w.Disconnect(); err != nil {
		t.Fatalf("Disconnect on nil client = %v, want nil", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close on nil client = %v, want nil", err)
	}
}

// TestCommandResult_SuccessAndOutput is a table-driven assertion of the two
// pure helpers. Anti-bluff: Success() must be true ONLY when ExitCode==0 AND
// Error==nil; Output() must concatenate stdout+stderr in that order. A stub
// `return true` for Success would fail the failure rows below.
func TestCommandResult_SuccessAndOutput(t *testing.T) {
	boom := context.DeadlineExceeded // any non-nil error
	cases := []struct {
		name        string
		res         CommandResult
		wantSuccess bool
		wantOutput  string
	}{
		{"clean-success", CommandResult{ExitCode: 0, Stdout: "out", Stderr: ""}, true, "out"},
		{"nonzero-exit", CommandResult{ExitCode: 2, Stdout: "o", Stderr: "e"}, false, "oe"},
		{"zero-exit-but-error", CommandResult{ExitCode: 0, Error: boom, Stdout: "o"}, false, "o"},
		{"both-streams", CommandResult{ExitCode: 0, Stdout: "A", Stderr: "B"}, true, "AB"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.res.Success(); got != tc.wantSuccess {
				t.Fatalf("Success() = %v, want %v", got, tc.wantSuccess)
			}
			if got := tc.res.Output(); got != tc.wantOutput {
				t.Fatalf("Output() = %q, want %q", got, tc.wantOutput)
			}
		})
	}
}

// TestGenerateSSHKey_RealKeyMaterial proves GenerateSSHKey emits a genuine,
// parseable RSA keypair where the public key actually corresponds to the
// private key. Anti-bluff: a stub returning fixed/empty strings would fail
// ssh.ParsePrivateKey, the PEM-header check, OR the derived-pubkey match below.
func TestGenerateSSHKey_RealKeyMaterial(t *testing.T) {
	priv, pub, err := GenerateSSHKey()
	if err != nil {
		t.Fatalf("GenerateSSHKey error: %v", err)
	}
	if !strings.Contains(priv, "RSA PRIVATE KEY") {
		t.Fatalf("private key missing PEM header, got prefix %q", priv[:min(40, len(priv))])
	}
	if !strings.HasPrefix(pub, "ssh-rsa ") {
		t.Fatalf("public key not in authorized_keys form, got prefix %q", pub[:min(20, len(pub))])
	}

	signer, err := ssh.ParsePrivateKey([]byte(priv))
	if err != nil {
		t.Fatalf("generated private key is not parseable: %v", err)
	}

	// Derive the public key from the parsed private key and compare against the
	// returned public string — proves they are a matched pair, not random bytes.
	derived := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
	returned := strings.TrimSpace(pub)
	if derived != returned {
		t.Fatalf("returned public key does not match private key:\n derived = %q\n returned = %q", derived, returned)
	}

	// Two successive calls must produce distinct private keys (real entropy).
	priv2, _, err := GenerateSSHKey()
	if err != nil {
		t.Fatalf("second GenerateSSHKey error: %v", err)
	}
	if priv == priv2 {
		t.Fatal("two GenerateSSHKey calls produced identical private keys (not random)")
	}
}

// TestExecuteCommand_ContextCancelled documents that ExecuteCommand returns the
// connection guard before context handling on a disconnected worker; a fully
// connected variant requires a live daemon (see SKIP-guarded integration test).
func TestExecuteCommand_DisconnectedIgnoresCancelledCtx(t *testing.T) {
	w := newTestWorker(t, SSHWorkerConfig{Host: "h", Port: 22})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := w.ExecuteCommand(ctx, "noop")
	// Connection guard wins; we assert the precise contract, not a guess.
	if err == nil || res != nil || !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("ExecuteCommand(cancelled,disconnected) = (%v,%v), want (nil,'not connected')", res, err)
	}
	_ = time.Now
}
