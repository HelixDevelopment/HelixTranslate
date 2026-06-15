package grpc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/grpc/proto"
)

// TestTranslate_NilOptions_NoPanic_CompletesPipeline is a REPRODUCE-FIRST
// (§11.4.115) test for a nil-pointer-dereference crash in the gRPC core
// translation pipeline.
//
// proto3 makes every message field optional on the wire. TranslationRequest.Options
// is a *TranslationOptions, so a client may legitimately send a request with
// Options unset (the StartTranslation validator only requires ProviderConfig +
// session_id, NOT Options). executeTranslationPipeline reads
// `req.Options.EnableMonitoring` unconditionally at the report-generation step,
// AFTER parse / markdown / translation / EPUB have all done their real work and
// the output EPUB has already been written to disk.
//
// With Options == nil that dereference panics. Because the panic happens inside
// the translation goroutine (s.runTranslation -> ct.Translate -> pipeline) with
// no recover, the goroutine dies: the session is left stuck "running" forever,
// the EPUB is silently produced but never reported as completed, and a single
// well-formed request crashes a serving goroutine. That is the exact
// "work done but reported broken / never reported" anti-bluff failure mode.
//
// Drives the REAL pipeline (real format detection, real markdown passthrough,
// real EPUB generation) using the offline "mock" LLM provider — no network, no
// real LLM call. RED on the pre-fix code (panics); GREEN after the nil-guard.
func TestTranslate_NilOptions_NoPanic_CompletesPipeline(t *testing.T) {
	dir := t.TempDir()
	inputTxt := filepath.Join(dir, "book.txt")
	if err := os.WriteFile(inputTxt, []byte("Hello world. This is a sentence to translate."), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	outputEpub := filepath.Join(dir, "book_out.epub")

	ct := newCT()
	// R-1b/R2: the API translation arm sources from the LLMsVerifier bridge; inject
	// the deterministic in-memory factory (mock "translated: <text>" transform) so
	// the pipeline runs offline without real provider keys (§11.4.27).
	installMockBridge(ct)
	req := &proto.TranslationRequest{
		SessionId:  "nil-options-session",
		InputFile:  inputTxt,
		OutputFile: outputEpub,
		SourceLang: "en",
		TargetLang: "sr",
		ProviderConfig: &proto.ProviderConfig{
			Type: "mock", // offline, deterministic — no network/LLM
		},
		Options: nil, // <-- the load-bearing condition: Options unset on the wire
	}

	bus := events.NewEventBus()

	// A panic here is the defect (current code). t.Fatalf via recover surfaces it
	// as a clean test failure rather than crashing the test binary.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Translate panicked on nil Options (the bug): %v", r)
		}
	}()

	resp, err := ct.Translate(context.Background(), req, bus)
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Translate returned nil response")
	}
	if resp.Status != "completed" {
		t.Fatalf("expected status completed, got %q (msg=%q)", resp.Status, resp.Message)
	}

	// Anti-bluff: the EPUB must actually exist on disk (real work happened).
	if _, statErr := os.Stat(outputEpub); statErr != nil {
		t.Fatalf("output EPUB not produced: %v", statErr)
	}

	// And the translated markdown must contain the mock provider's real output,
	// proving the translation step actually ran (not faked).
	translatedMD := filepath.Join(dir, "book_translated.md")
	data, rdErr := os.ReadFile(translatedMD)
	if rdErr != nil {
		t.Fatalf("translated markdown not produced: %v", rdErr)
	}
	if !strings.Contains(string(data), "translated:") {
		t.Fatalf("translated markdown lacks real mock output: %q", string(data))
	}
}
