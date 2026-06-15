package grpc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/grpc/proto"
	"digital.vasic.translator/pkg/logger"
)

func newCT() *CoreTranslatorImpl {
	return NewCoreTranslator(logger.NewNoOpLogger()).(*CoreTranslatorImpl)
}

// --- verifyTranslation (pure mapping logic, no backend) ---------------------

func TestVerifyTranslation(t *testing.T) {
	ct := newCT()
	tests := []struct {
		name       string
		text       string
		targetLang string
		script     string
		want       bool
	}{
		{"serbian cyrillic present", "Превод текста на ћирилици", "sr", "cyrillic", true},
		{"serbian cyrillic but only latin text", "Prevod teksta latinicom", "sr", "cyrillic", false},
		{"serbian cyrillic empty", "", "sr", "cyrillic", false},
		{"non-serbian non-empty", "some english text", "en", "latin", true},
		{"non-serbian whitespace only", "   \n\t ", "en", "latin", false},
		{"non-serbian non-empty wins", "hola", "es", "latin", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ct.verifyTranslation(tc.text, tc.targetLang, tc.script)
			if got != tc.want {
				t.Errorf("verifyTranslation(%q,%q,%q) = %v, want %v", tc.text, tc.targetLang, tc.script, got, tc.want)
			}
		})
	}
}

// --- generatePath -----------------------------------------------------------

func TestGeneratePath(t *testing.T) {
	ct := newCT()
	tests := []struct {
		input  string
		suffix string
		want   string
	}{
		{"/a/b/book.epub", "_translated.md", filepath.Join("/a/b", "book_translated.md")},
		{"book.fb2", "_original.md", "book_original.md"},
		{"/x/y/z/novel.txt", "_session_report.md", filepath.Join("/x/y/z", "novel_session_report.md")},
	}
	for _, tc := range tests {
		got := ct.generatePath(tc.input, tc.suffix)
		if got != tc.want {
			t.Errorf("generatePath(%q,%q) = %q, want %q", tc.input, tc.suffix, got, tc.want)
		}
	}
}

// --- getContentType ---------------------------------------------------------

func TestGetContentType(t *testing.T) {
	ct := newCT()
	tests := map[string]string{
		"original_md":   "text/markdown",
		"translated_md": "text/markdown",
		"epub":          "application/epub+zip",
		"report":        "text/markdown",
		"unknown":       "application/octet-stream",
		"":              "application/octet-stream",
	}
	for in, want := range tests {
		if got := ct.getContentType(in); got != want {
			t.Errorf("getContentType(%q) = %q, want %q", in, got, want)
		}
	}
}

// --- step lifecycle ---------------------------------------------------------

func TestStepLifecycle(t *testing.T) {
	ct := newCT()

	step := ct.createStep("parsing", "Parsing input")
	if step.GetName() != "parsing" {
		t.Errorf("name = %q, want parsing", step.GetName())
	}
	if step.GetStatus() != "running" {
		t.Errorf("initial status = %q, want running", step.GetStatus())
	}
	if step.GetStartedAt() == nil {
		t.Error("StartedAt not set on create")
	}

	ct.completeStep(step)
	if step.GetStatus() != "completed" {
		t.Errorf("after complete status = %q, want completed", step.GetStatus())
	}
	if step.GetEndedAt() == nil {
		t.Error("EndedAt not set on complete")
	}

	failed := ct.createStep("translation", "Translating")
	ct.failStep(failed, os.ErrPermission)
	if failed.GetStatus() != "failed" {
		t.Errorf("after fail status = %q, want failed", failed.GetStatus())
	}
	if failed.GetErrorMessage() != os.ErrPermission.Error() {
		t.Errorf("error message = %q, want %q", failed.GetErrorMessage(), os.ErrPermission.Error())
	}
}

// --- updateJobStep (progress math) ------------------------------------------

func TestUpdateJobStep_ProgressMath(t *testing.T) {
	ct := newCT()
	job := &TranslationJob{Steps: make([]*proto.TranslationStep, 0)}

	// Step index 1 -> (1-1)/4*100 = 0
	ct.updateJobStep(job, ct.createStep("s1", ""))
	if job.Progress != 0 {
		t.Errorf("after step1 progress = %v, want 0", job.Progress)
	}
	// Step index 2 -> (2-1)/4*100 = 25
	ct.updateJobStep(job, ct.createStep("s2", ""))
	if job.Progress != 25 {
		t.Errorf("after step2 progress = %v, want 25", job.Progress)
	}
	// Step index 4 -> (4-1)/4*100 = 75
	ct.updateJobStep(job, ct.createStep("s3", ""))
	ct.updateJobStep(job, ct.createStep("s4", ""))
	if job.Progress != 75 {
		t.Errorf("after step4 progress = %v, want 75", job.Progress)
	}
	if job.Step != "s4" {
		t.Errorf("current step = %q, want s4", job.Step)
	}
	if len(job.Steps) != 4 {
		t.Errorf("steps len = %d, want 4", len(job.Steps))
	}
}

// --- addGeneratedFile + content type wiring ---------------------------------

func TestAddGeneratedFile(t *testing.T) {
	ct := newCT()
	job := &TranslationJob{Files: make([]*proto.GeneratedFile, 0)}

	ct.addGeneratedFile(job, "/out/book.epub", "epub", 2048, true, "Valid EPUB format")
	if len(job.Files) != 1 {
		t.Fatalf("files len = %d, want 1", len(job.Files))
	}
	f := job.Files[0]
	if f.GetPath() != "/out/book.epub" {
		t.Errorf("path = %q", f.GetPath())
	}
	if f.GetType() != "epub" {
		t.Errorf("type = %q", f.GetType())
	}
	if f.GetSize() != 2048 {
		t.Errorf("size = %d, want 2048", f.GetSize())
	}
	if f.GetContentType() != "application/epub+zip" {
		t.Errorf("content type = %q, want application/epub+zip", f.GetContentType())
	}
	if !f.GetVerified() {
		t.Error("verified = false, want true")
	}
	if f.GetVerificationMessage() != "Valid EPUB format" {
		t.Errorf("verification message = %q", f.GetVerificationMessage())
	}
	if f.GetCreatedAt() == nil {
		t.Error("CreatedAt not set")
	}
}

// --- createErrorResponse (response shaping) ---------------------------------

func TestCreateErrorResponse(t *testing.T) {
	ct := newCT()
	job := &TranslationJob{ID: "err-sess", Progress: 37.5}
	failed := ct.createStep("translation", "Translating")
	ct.failStep(failed, os.ErrNotExist)

	resp := ct.createErrorResponse(job, failed)
	if resp.GetSessionId() != "err-sess" {
		t.Errorf("session id = %q", resp.GetSessionId())
	}
	if resp.GetStatus() != "failed" {
		t.Errorf("status = %q, want failed", resp.GetStatus())
	}
	if resp.GetProgressPercentage() != 37.5 {
		t.Errorf("progress = %v, want 37.5", resp.GetProgressPercentage())
	}
	if resp.GetCurrentStep() != "translation" {
		t.Errorf("current step = %q, want translation", resp.GetCurrentStep())
	}
	if resp.GetErrorCode() != 500 {
		t.Errorf("error code = %d, want 500", resp.GetErrorCode())
	}
	if resp.GetErrorMessage() != os.ErrNotExist.Error() {
		t.Errorf("error message = %q, want %q", resp.GetErrorMessage(), os.ErrNotExist.Error())
	}
}

// --- buildSSHCommand --------------------------------------------------------

func TestBuildSSHCommand(t *testing.T) {
	ct := newCT()
	cfg := &proto.ProviderConfig{RemoteDir: "/work"}
	cmd := ct.buildSSHCommand(cfg, "/work/input.md", "/work/output.md")
	if !strings.Contains(cmd, "cd /work") {
		t.Errorf("command missing cd into remote dir: %q", cmd)
	}
	if !strings.Contains(cmd, "/work/input.md") || !strings.Contains(cmd, "/work/output.md") {
		t.Errorf("command missing input/output paths: %q", cmd)
	}
	if !strings.Contains(cmd, "llama.cpp") {
		t.Errorf("command missing llama.cpp invocation: %q", cmd)
	}
}

// --- verifyEPUB (real file I/O, no network backend) -------------------------

func TestVerifyEPUB(t *testing.T) {
	ct := newCT()

	t.Run("missing file", func(t *testing.T) {
		if ct.verifyEPUB(filepath.Join(t.TempDir(), "nope.epub")) {
			t.Error("verifyEPUB on missing file = true, want false")
		}
	})

	t.Run("valid epub signature", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "ok.epub")
		// PK signature + mimetype marker required by verifyEPUB
		content := []byte("PK\x03\x04mimetypeapplication/epub+zip")
		if err := os.WriteFile(p, content, 0o644); err != nil {
			t.Fatal(err)
		}
		if !ct.verifyEPUB(p) {
			t.Error("verifyEPUB on valid epub bytes = false, want true")
		}
	})

	t.Run("not an epub", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "plain.txt")
		if err := os.WriteFile(p, []byte("just some plain text not a zip"), 0o644); err != nil {
			t.Fatal(err)
		}
		if ct.verifyEPUB(p) {
			t.Error("verifyEPUB on non-epub = true, want false")
		}
	})
}

// --- getFileSize ------------------------------------------------------------

func TestGetFileSize(t *testing.T) {
	ct := newCT()
	if ct.getFileSize(filepath.Join(t.TempDir(), "absent")) != 0 {
		t.Error("getFileSize on missing file != 0")
	}
	p := filepath.Join(t.TempDir(), "f.bin")
	if err := os.WriteFile(p, []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ct.getFileSize(p); got != 5 {
		t.Errorf("getFileSize = %d, want 5", got)
	}
}

// --- convertToMarkdown default passthrough ----------------------------------

func TestConvertToMarkdown_DefaultPassthrough(t *testing.T) {
	ct := newCT()
	in := "# Already markdown\n\nbody"
	out, err := ct.convertToMarkdown("", in, "txt")
	if err != nil {
		t.Fatalf("convertToMarkdown(txt) error: %v", err)
	}
	if out != in {
		t.Errorf("default passthrough = %q, want %q", out, in)
	}
}

// --- Cancel / GetStatus interface methods on CoreTranslatorImpl -------------

func TestCoreTranslator_CancelAndGetStatus_UnknownSession(t *testing.T) {
	ct := newCT()
	if err := ct.Cancel("unknown"); err == nil {
		t.Error("Cancel(unknown) = nil, want not-found error")
	}
	if _, err := ct.GetStatus("unknown"); err == nil {
		t.Error("GetStatus(unknown) = nil, want not-found error")
	}
}

func TestCoreTranslator_GetStatus_KnownJob(t *testing.T) {
	ct := newCT()
	ct.sessions["job-1"] = &TranslationJob{
		ID:       "job-1",
		Status:   "running",
		Progress: 50,
		Step:     "translation",
	}
	st, err := ct.GetStatus("job-1")
	if err != nil {
		t.Fatalf("GetStatus(job-1): %v", err)
	}
	if st.GetSessionId() != "job-1" || st.GetStatus() != "running" || st.GetProgressPercentage() != 50 {
		t.Errorf("status mismatch: %+v", st)
	}
}

func TestCoreTranslator_Cancel_KnownJob(t *testing.T) {
	ct := newCT()
	called := false
	ct.sessions["job-c"] = &TranslationJob{
		ID:         "job-c",
		Status:     "running",
		CancelFunc: func() { called = true },
	}
	if err := ct.Cancel("job-c"); err != nil {
		t.Fatalf("Cancel(job-c): %v", err)
	}
	if !called {
		t.Error("CancelFunc not invoked")
	}
	if ct.sessions["job-c"].Status != "cancelled" {
		t.Errorf("status = %q, want cancelled", ct.sessions["job-c"].Status)
	}
}

// --- emitProgress nil-bus guard ---------------------------------------------

func TestEmitProgress_NilBusIsNoop(t *testing.T) {
	ct := newCT()
	// must not panic with a nil event bus
	ct.emitProgress(nil, "sess", "evt", "step", 10, "msg")

	bus := events.NewEventBus()
	got := make(chan events.Event, 1)
	bus.SubscribeAll(func(e events.Event) { got <- e })
	ct.emitProgress(bus, "sess", "evt", "step", 10, "msg")
	// EventBus dispatches handlers in goroutines, so wait (don't poll once).
	select {
	case e := <-got:
		if e.SessionID != "sess" {
			t.Errorf("event session id = %q, want sess", e.SessionID)
		}
	case <-time.After(2 * time.Second):
		t.Error("expected event published to bus")
	}
}

// --- ProviderRegistry -------------------------------------------------------

func TestProviderRegistry_DefaultsAndLookup(t *testing.T) {
	reg := NewProviderRegistry()
	all := reg.GetAll()
	if len(all) != 3 {
		t.Fatalf("default providers = %d, want 3", len(all))
	}

	for _, typ := range []string{"openai", "anthropic", "ssh"} {
		p, ok := reg.Get(typ)
		if !ok {
			t.Errorf("Get(%q) not found", typ)
			continue
		}
		if p.GetType() != typ {
			t.Errorf("Get(%q).Type = %q", typ, p.GetType())
		}
	}

	if _, ok := reg.Get("nonexistent"); ok {
		t.Error("Get(nonexistent) = ok, want false")
	}

	// ssh requires ssh config, not api key
	if ssh, _ := reg.Get("ssh"); ssh != nil {
		if !ssh.GetRequiresSshConfig() {
			t.Error("ssh provider should require ssh config")
		}
		if ssh.GetRequiresApiKey() {
			t.Error("ssh provider should not require api key")
		}
	}
	// openai requires api key
	if oai, _ := reg.Get("openai"); oai != nil && !oai.GetRequiresApiKey() {
		t.Error("openai provider should require api key")
	}
}

// --- convertMetadata / timeToProto helpers ----------------------------------

func TestConvertMetadata(t *testing.T) {
	out := convertMetadata(map[string]interface{}{"a": 1, "b": "x", "c": true})
	if out["a"] != "1" || out["b"] != "x" || out["c"] != "true" {
		t.Errorf("convertMetadata = %#v", out)
	}
	if len(convertMetadata(nil)) != 0 {
		t.Error("convertMetadata(nil) should be empty")
	}
}
