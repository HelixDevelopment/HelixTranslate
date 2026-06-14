package main

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// buildSSHTranslateCommand — the remote llama.cpp command run on the SSH worker.
//
// REGRESSION (de665cb "Auto-commit" stub): the SSH path hardcoded
//   /home/milosvasic/llama.cpp -m /home/milosvasic/models/tiny-llama-working.gguf
//   -p 'Translate from Russian to Serbian Cyrillic: '
// ignoring -llama-binary, -llama-model, -source-lang, -target-lang and -script.
// A user translating English→French via SSH silently got a Russian→Serbian
// Cyrillic prompt with the wrong model — a silent wrong-translation defect.
//
// These assertions FAIL against the hardcoded command (see the mutation proof in
// the fix commit message) and PASS once the command honors the user's flags.
// ---------------------------------------------------------------------------

func TestBuildSSHTranslateCommand_HonorsConfig(t *testing.T) {
	config := &UnifiedConfig{
		RemoteDir:   "/srv/work",
		LlamaBinary: "/opt/llama/main",
		LlamaModel:  "/models/mixtral.gguf",
		SourceLang:  "en",
		TargetLang:  "fr",
		Script:      "latin",
	}
	cmd := buildSSHTranslateCommand(config, "/srv/work/input.md", "/srv/work/output.md")

	// Honors -source-lang / -target-lang in the prompt (NOT the hardcoded
	// Russian→Serbian Cyrillic).
	if !strings.Contains(cmd, "Translate from English to French") {
		t.Fatalf("prompt did not honor source/target lang.\ncmd: %s", cmd)
	}
	if strings.Contains(cmd, "Russian") || strings.Contains(cmd, "Serbian Cyrillic") {
		t.Fatalf("prompt still carries the hardcoded Russian→Serbian Cyrillic text.\ncmd: %s", cmd)
	}
	// Honors -llama-binary and -llama-model (NOT the hardcoded paths).
	if !strings.Contains(cmd, "/opt/llama/main") {
		t.Fatalf("command did not honor -llama-binary.\ncmd: %s", cmd)
	}
	if !strings.Contains(cmd, "/models/mixtral.gguf") {
		t.Fatalf("command did not honor -llama-model.\ncmd: %s", cmd)
	}
	if strings.Contains(cmd, "/home/milosvasic/") {
		t.Fatalf("command still references the hardcoded /home/milosvasic paths.\ncmd: %s", cmd)
	}
	// Still wires the upload/output paths through.
	if !strings.Contains(cmd, "-f /srv/work/input.md") || !strings.Contains(cmd, "> /srv/work/output.md") {
		t.Fatalf("command lost the input/output path wiring.\ncmd: %s", cmd)
	}
	// cd's into the remote dir.
	if !strings.HasPrefix(cmd, "cd /srv/work && ") {
		t.Fatalf("command does not cd into the remote dir.\ncmd: %s", cmd)
	}
}

func TestBuildSSHTranslateCommand_SerbianCyrillicTarget(t *testing.T) {
	config := &UnifiedConfig{
		RemoteDir:  "/tmp/translator",
		SourceLang: "ru",
		TargetLang: "sr",
		Script:     "cyrillic",
	}
	cmd := buildSSHTranslateCommand(config, "/tmp/translator/input.md", "/tmp/translator/output.md")
	if !strings.Contains(cmd, "Translate from Russian to Serbian Cyrillic") {
		t.Fatalf("ru→sr/cyrillic prompt wrong.\ncmd: %s", cmd)
	}
}

func TestBuildSSHTranslateCommand_FallbackPaths(t *testing.T) {
	// Empty binary/model fall back to sane defaults (not an empty -m / no binary).
	config := &UnifiedConfig{
		RemoteDir:  "/tmp/translator",
		SourceLang: "ru",
		TargetLang: "sr",
		Script:     "cyrillic",
	}
	cmd := buildSSHTranslateCommand(config, "/tmp/translator/input.md", "/tmp/translator/output.md")
	if strings.Contains(cmd, "-m  ") || strings.Contains(cmd, "-m -p") {
		t.Fatalf("empty model produced a broken -m flag.\ncmd: %s", cmd)
	}
	if !strings.Contains(cmd, ".gguf") {
		t.Fatalf("fallback model path missing.\ncmd: %s", cmd)
	}
}

// Quote-safety: a single quote in a (hypothetical) prompt value must not break
// out of the shell single-quoting.
func TestShellSingleQuote(t *testing.T) {
	if got := shellSingleQuote("ab"); got != "'ab'" {
		t.Fatalf("shellSingleQuote(ab) = %q", got)
	}
	got := shellSingleQuote("a'b")
	want := `'a'"'"'b'`
	if got != want {
		t.Fatalf("shellSingleQuote(a'b) = %q, want %q", got, want)
	}
}

func TestSSHTranslatePromptUnknownLang(t *testing.T) {
	// Unknown codes fall back to the code itself (no guessing, §11.4.6).
	p := sshTranslatePrompt("xx", "yy", "")
	if !strings.Contains(p, "xx") || !strings.Contains(p, "yy") {
		t.Fatalf("unknown-lang prompt lost codes: %q", p)
	}
}
