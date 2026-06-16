package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/websocket"

	"github.com/gin-gonic/gin"
)

// langRecordingTranslator records the task language pair the bridge factory was
// asked to build, so a test can prove POST /api/v1/translate/fb2 honors the
// requested source_lang/target_lang instead of the historical hardcoded ru→sr.
type langRecordingTranslator struct{}

func (langRecordingTranslator) Translate(_ context.Context, text, _ string) (string, error) {
	return "T:" + text, nil
}
func (langRecordingTranslator) TranslateWithProgress(_ context.Context, text, _ string, _ *events.EventBus, _ string) (string, error) {
	return "T:" + text, nil
}
func (langRecordingTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}
func (langRecordingTranslator) GetName() string { return "lang-recorder" }

// TestTranslateFB2_HonorsRequestedLangs is the §11.4.115 RED-baseline for
// BUG-FB2-HARDCODED-LANG and its §11.4.146 extend-to-all-cases fan-out.
//
// RED on the pre-fix artifact: translateFB2 called createTranslator(provider,
// model, "", "") unconditionally, so the bridge task was ALWAYS ru→sr regardless
// of the requested source_lang/target_lang form fields — the recorded TargetLang
// was "sr" for every case, failing the explicit-language assertions below.
// GREEN after the fix: the requested form fields are parsed and threaded through;
// ru→sr survives ONLY when both are omitted; unknown codes return 400.
func TestTranslateFB2_HonorsRequestedLangs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const fb2 = `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description><title-info><genre>nonfiction</genre><book-title>T</book-title></title-info></description>
  <body><section><p>Hello world</p></section></body>
</FictionBook>`

	type recorded struct {
		mu   sync.Mutex
		src  string
		dst  string
		hits int
	}

	newHandlerWithRecorder := func(rec *recorded) *Handler {
		cfg := &config.Config{
			Translation: config.TranslationConfig{
				DefaultProvider: "openai",
				Providers:       map[string]config.ProviderConfig{"openai": {APIKey: "test-key", Model: "gpt-4"}},
			},
		}
		eb := events.NewEventBus()
		h := &Handler{config: cfg, eventBus: eb, wsHub: websocket.NewHub(eb)}
		h.bridgeTranslatorFactory = func(_ context.Context, task selection.TaskRequirements) (translator.Translator, error) {
			rec.mu.Lock()
			rec.src, rec.dst, rec.hits = task.SourceLang, task.TargetLang, rec.hits+1
			rec.mu.Unlock()
			return langRecordingTranslator{}, nil
		}
		return h
	}

	multipart := func(fields map[string]string) (string, string) {
		const b = "boundary"
		var sb strings.Builder
		fmt.Fprintf(&sb, "--%s\r\nContent-Disposition: form-data; name=\"file\"; filename=\"t.fb2\"\r\nContent-Type: application/xml\r\n\r\n%s\r\n", b, fb2)
		for k, v := range fields {
			fmt.Fprintf(&sb, "--%s\r\nContent-Disposition: form-data; name=\"%s\"\r\n\r\n%s\r\n", b, k, v)
		}
		fmt.Fprintf(&sb, "--%s--\r\n", b)
		return sb.String(), "multipart/form-data; boundary=" + b
	}

	cases := []struct {
		name       string
		fields     map[string]string
		wantSrc    string
		wantDst    string
		wantBadReq bool // expect a 400 (unknown lang) — no translator built
	}{
		{"no langs -> legacy ru->sr default preserved", map[string]string{}, "ru", "sr", false},
		{"explicit en->es honored", map[string]string{"source_lang": "en", "target_lang": "es"}, "en", "es", false},
		{"explicit en->de honored", map[string]string{"source_lang": "en", "target_lang": "de"}, "en", "de", false},
		{"target only fr (source stays default ru)", map[string]string{"target_lang": "fr"}, "ru", "fr", false},
		{"unknown target -> 400, no translator", map[string]string{"source_lang": "en", "target_lang": "klingon"}, "", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := &recorded{}
			h := newHandlerWithRecorder(rec)
			r := gin.New()
			r.POST("/translate/fb2", h.translateFB2)

			body, ct := multipart(c.fields)
			req, _ := http.NewRequest("POST", "/translate/fb2", strings.NewReader(body))
			req.Header.Set("Content-Type", ct)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if c.wantBadReq {
				if w.Code != http.StatusBadRequest {
					t.Fatalf("unknown target_lang must return 400, got %d (body=%s)", w.Code, w.Body.String())
				}
				if rec.hits != 0 {
					t.Fatalf("unknown target_lang must NOT build a translator (recorded %d builds)", rec.hits)
				}
				return
			}
			if rec.hits == 0 {
				t.Fatalf("translator was never built (code=%d body=%s)", w.Code, w.Body.String())
			}
			rec.mu.Lock()
			gotSrc, gotDst := rec.src, rec.dst
			rec.mu.Unlock()
			if gotSrc != c.wantSrc {
				t.Fatalf("BUG-FB2-HARDCODED-LANG: source threaded as %q, want %q", gotSrc, c.wantSrc)
			}
			if gotDst != c.wantDst {
				t.Fatalf("BUG-FB2-HARDCODED-LANG: target threaded as %q, want %q (was hardcoded sr?)", gotDst, c.wantDst)
			}
		})
	}
}
