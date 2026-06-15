package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/language"
)

// TestLanguagesAdvertisedAreActuallyAccepted is a reproduce-first guard for the
// cross-endpoint contract defect: GET /api/v1/languages advertises a language
// list, but POST /api/v1/translate/validate (and every other endpoint that
// calls language.ParseLanguage) rejects most of them with HTTP 400
// "invalid target language". An end user who selects an advertised language
// from /languages and submits it gets a 400 for a language the API itself
// claims to support — a §11.4 user-visible response-correctness defect.
//
// Contract: every code returned by GET /languages MUST be accepted as a valid
// target_language by /translate/validate. The validate endpoint is used here
// because it exercises the identical language.ParseLanguage gate as the real
// translation endpoints WITHOUT requiring any provider API key.
func TestLanguagesAdvertisedAreActuallyAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{
		config: &config.Config{
			Translation: config.TranslationConfig{DefaultProvider: "openai"},
		},
	}

	router := gin.New()
	router.GET("/api/v1/languages", h.listLanguages)
	router.POST("/api/v1/translate/validate", h.validateTranslationRequest)

	// 1) Fetch the advertised language list.
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/languages", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /languages returned %d, want 200", w.Code)
	}
	var langResp struct {
		Languages []struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"languages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &langResp); err != nil {
		t.Fatalf("decode /languages: %v", err)
	}
	if len(langResp.Languages) == 0 {
		t.Fatalf("/languages advertised zero languages")
	}

	// 2) Every advertised language MUST be accepted by /translate/validate as a
	//    target language. (provider openai is always in the valid set, so the
	//    only thing under test is the language gate.)
	var rejected []string
	for _, lang := range langResp.Languages {
		body, _ := json.Marshal(map[string]string{
			"text":            "hello",
			"target_language": lang.Code,
			"provider":        "openai",
		})
		vreq, _ := http.NewRequest(http.MethodPost, "/api/v1/translate/validate", bytes.NewReader(body))
		vreq.Header.Set("Content-Type", "application/json")
		vw := httptest.NewRecorder()
		router.ServeHTTP(vw, vreq)

		if vw.Code != http.StatusOK {
			rejected = append(rejected, fmt.Sprintf("%s(%s)", lang.Code, lang.Name))
		}
	}

	if len(rejected) > 0 {
		t.Fatalf("%d/%d advertised languages were REJECTED by /translate/validate "+
			"(advertised-but-unsupported): %v", len(rejected), len(langResp.Languages), rejected)
	}
}

// TestLanguagesAdvertisedSetMatchesParser is a tighter, fake-free guard that
// pins the /languages handler output directly against the parser's source of
// truth (language.ParseLanguage). It is independent of the validate handler.
func TestLanguagesAdvertisedSetMatchesParser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{}
	router := gin.New()
	router.GET("/languages", h.listLanguages)

	req, _ := http.NewRequest(http.MethodGet, "/languages", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var langResp struct {
		Languages []struct {
			Code string `json:"code"`
		} `json:"languages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &langResp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	var bad []string
	for _, l := range langResp.Languages {
		if _, err := language.ParseLanguage(l.Code); err != nil {
			bad = append(bad, l.Code)
		}
	}
	if len(bad) > 0 {
		t.Fatalf("/languages advertises %d code(s) that language.ParseLanguage rejects: %v", len(bad), bad)
	}
}
