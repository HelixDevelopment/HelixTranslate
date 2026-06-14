package logger

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
)

// RED: an unknown/misconfigured level (e.g. the common typo "warning")
// must NOT silently fail open and log everything including DEBUG.
func TestShouldLog_UnknownConfiguredLevel_DoesNotFailOpen(t *testing.T) {
	l := NewLogger(LoggerConfig{Level: "warning", Format: "text"}).(*StandardLogger)
	// With a misconfigured level, the safe expectation is to behave like the
	// default (INFO): debug must be filtered out.
	if l.shouldLog("debug") {
		t.Errorf("unknown configured level %q caused debug to be logged (fail-open)", l.level)
	}
	if l.shouldLog("info") == false {
		t.Errorf("unknown configured level %q should still log info (default INFO)", l.level)
	}
}

// RED: an unknown message level must NOT be silently dropped. A caller that
// emits at an unrecognized severity should still produce output (fail-safe:
// treat unknown message severity as at least the configured level so it is
// not lost). With the buggy code, messageLevelValue stays -1 and the message
// is filtered out at any configured level >= debug.
func TestShouldLog_UnknownMessageLevel_NotSilentlyDropped(t *testing.T) {
	l := NewLogger(LoggerConfig{Level: "info", Format: "text"}).(*StandardLogger)
	if !l.shouldLog("trace") {
		t.Errorf("unknown message level was silently dropped at configured level %q", l.level)
	}
}

// Behavioral: a misconfigured logger must still emit a real INFO line to its
// output writer (asserts observable behavior, not just shouldLog()).
func TestLog_MisconfiguredLevel_StillEmitsInfo(t *testing.T) {
	var buf bytes.Buffer
	l := &StandardLogger{level: "warning", format: "text", logger: log.New(&buf, "", 0)}
	l.Info("hello", nil)
	if !strings.Contains(buf.String(), "INFO: hello") {
		t.Errorf("misconfigured logger dropped INFO; output=%q", buf.String())
	}
	// And it must NOT emit debug (no fail-open).
	buf.Reset()
	l.Debug("secret-debug", nil)
	if strings.Contains(buf.String(), "secret-debug") {
		t.Errorf("misconfigured logger failed open and emitted DEBUG; output=%q", buf.String())
	}
}

// Concurrency: concurrent log calls on a shared logger must not race. The
// underlying log.Logger is goroutine-safe; this guards against regressions if
// shared mutable state (e.g. a fields map or buffer) is ever added.
func TestLog_ConcurrentCalls_NoRace(t *testing.T) {
	var buf bytes.Buffer
	l := &StandardLogger{level: "debug", format: "json", logger: log.New(&buf, "", 0)}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			l.Info("concurrent", map[string]interface{}{"n": n})
		}(i)
	}
	wg.Wait()
	if c := strings.Count(buf.String(), "\n"); c != 50 {
		t.Errorf("expected 50 log lines, got %d", c)
	}
}

// Behavioral: JSON format must produce parseable JSON with the field merged.
func TestFormatJSON_FieldsMerged_Parseable(t *testing.T) {
	l := &StandardLogger{level: "debug", format: "json"}
	out := l.formatJSON("warn", "m", map[string]interface{}{"a": 1.0}, "ts")
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v (%s)", err, out)
	}
	if parsed["a"] != 1.0 || parsed["level"] != "warn" {
		t.Errorf("unexpected JSON content: %s", out)
	}
}

// Behavioral: NewLogger with OutputFile writes the log line to that file.
func TestNewLogger_WritesToOutputFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/app.log"
	l := NewLogger(LoggerConfig{Level: "info", Format: "text", OutputFile: path})
	l.Info("file-line", map[string]interface{}{"k": "v"})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(data), "INFO: file-line") || !strings.Contains(string(data), "k=v") {
		t.Errorf("log file missing expected content: %q", string(data))
	}
}

// Behavioral: NoOpLogger methods exercise all entry points and produce nothing
// observable (and never panic).
func TestNoOpLogger_AllMethodsNoPanic(t *testing.T) {
	l := NewNoOpLogger()
	l.Debug("d", map[string]interface{}{"x": 1})
	l.Info("i", nil)
	l.Warn("w", nil)
	l.Error("e", nil)
	l.Fatal("f", nil) // must NOT exit the process (no-op)
}

// Behavioral: formatJSON falls back to a valid error envelope when a field is
// not JSON-marshalable (e.g. a channel). Must remain parseable-ish and carry
// the message; asserts the marshal-error branch.
func TestFormatJSON_UnmarshalableField_Fallback(t *testing.T) {
	l := &StandardLogger{level: "debug", format: "json"}
	out := l.formatJSON("error", "boom", map[string]interface{}{"bad": make(chan int)}, "ts")
	if !strings.Contains(out, "failed to marshal log") || !strings.Contains(out, "boom") {
		t.Errorf("expected marshal-error fallback envelope, got: %s", out)
	}
}

// RED (§11.4.115): a user field whose key collides with a reserved JSON log
// key ("level", "message", "timestamp") must NOT clobber the authoritative log
// metadata. On the pre-fix code, fields were merged AFTER the reserved keys, so
// fields["level"]="db-layer" overwrote the real severity ("error") -> the true
// severity was silently DROPPED, corrupting downstream level filtering/alerting.
//
// Correct behaviour: the reserved keys keep the logger-supplied values; the
// colliding user value is preserved (never lost) under a "fields." namespace.
func TestFormatJSON_ReservedKeyCollision_DoesNotClobberSeverity(t *testing.T) {
	l := &StandardLogger{level: "debug", format: "json"}
	out := l.formatJSON("error", "db failure",
		map[string]interface{}{"level": "db-layer", "message": "user-msg", "timestamp": "user-ts"}, "real-ts")
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v (%s)", err, out)
	}
	// Reserved metadata must be the logger-supplied values.
	if parsed["level"] != "error" {
		t.Errorf("real severity clobbered by user field: level=%v want %q (%s)", parsed["level"], "error", out)
	}
	if parsed["message"] != "db failure" {
		t.Errorf("real message clobbered by user field: message=%v want %q (%s)", parsed["message"], "db failure", out)
	}
	if parsed["timestamp"] != "real-ts" {
		t.Errorf("real timestamp clobbered by user field: timestamp=%v want %q (%s)", parsed["timestamp"], "real-ts", out)
	}
	// The colliding user values must be preserved (not silently lost).
	if parsed["fields.level"] != "db-layer" {
		t.Errorf("colliding user field dropped: fields.level=%v want %q (%s)", parsed["fields.level"], "db-layer", out)
	}
	if parsed["fields.message"] != "user-msg" {
		t.Errorf("colliding user field dropped: fields.message=%v want %q (%s)", parsed["fields.message"], "user-msg", out)
	}
	if parsed["fields.timestamp"] != "user-ts" {
		t.Errorf("colliding user field dropped: fields.timestamp=%v want %q (%s)", parsed["fields.timestamp"], "user-ts", out)
	}
}

// Non-colliding fields must still be merged at top level unchanged.
func TestFormatJSON_NonCollidingFields_UnchangedTopLevel(t *testing.T) {
	l := &StandardLogger{level: "debug", format: "json"}
	out := l.formatJSON("warn", "m", map[string]interface{}{"user_id": 42.0, "op": "save"}, "ts")
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v (%s)", err, out)
	}
	if parsed["user_id"] != 42.0 || parsed["op"] != "save" {
		t.Errorf("non-colliding fields must remain top-level: %s", out)
	}
	if parsed["level"] != "warn" || parsed["message"] != "m" {
		t.Errorf("reserved keys altered: %s", out)
	}
}

// Behavioral: formatMessage routes JSON vs text by configured format.
func TestFormatMessage_RoutesByFormat(t *testing.T) {
	j := &StandardLogger{format: "json"}
	if got := j.formatMessage("info", "m", nil); !strings.HasPrefix(got, "{") {
		t.Errorf("json format should yield JSON, got %q", got)
	}
	txt := &StandardLogger{format: "text"}
	if got := txt.formatMessage("info", "m", nil); !strings.HasPrefix(got, "[") {
		t.Errorf("text format should yield bracketed text, got %q", got)
	}
}
