package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// D12 CORS security tests. The vulnerability (pre-fix): with the DEFAULT config
// CORSOrigins=["*"], corsMiddleware reflected the ARBITRARY request Origin into
// Access-Control-Allow-Origin AND set Access-Control-Allow-Credentials: true —
// allowing ANY site to make credentialed cross-origin requests (auth bypass; worse
// than a literal "*", which browsers reject with credentials). These tests assert
// the SECURE contract; they are RED on the pre-fix middleware.
func init() { gin.SetMode(gin.TestMode) }

func runCORS(t *testing.T, origins []string, reqOrigin string) http.Header {
	t.Helper()
	r := gin.New()
	r.Use(corsMiddleware(origins))
	r.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if reqOrigin != "" {
		req.Header.Set("Origin", reqOrigin)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Header()
}

// Wildcard config must NOT reflect an arbitrary origin with credentials.
func TestCORS_Wildcard_NoCredentialedOriginReflection(t *testing.T) {
	h := runCORS(t, []string{"*"}, "https://evil.example")
	ao := h.Get("Access-Control-Allow-Origin")
	ac := h.Get("Access-Control-Allow-Credentials")

	if ao == "https://evil.example" && ac == "true" {
		t.Fatalf("CORS VULN: arbitrary origin reflected (%q) WITH credentials (%q) — any site can make credentialed requests", ao, ac)
	}
	// Secure wildcard: literal "*" and NO credentials (the combo browsers forbid).
	if ao != "*" {
		t.Fatalf("wildcard config should emit Allow-Origin: * (got %q)", ao)
	}
	if ac == "true" {
		t.Fatalf("wildcard config must NOT set Allow-Credentials: true (got %q)", ac)
	}
}

// A specific allowlisted origin SHOULD be reflected + may use credentials (safe:
// only the configured origin, not arbitrary).
func TestCORS_Allowlist_ReflectsAndCredentials(t *testing.T) {
	h := runCORS(t, []string{"https://app.example.com"}, "https://app.example.com")
	if got := h.Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("allowlisted origin should be reflected, got %q", got)
	}
	if got := h.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allowlisted origin may use credentials, got %q", got)
	}
}

// A non-allowlisted origin (specific allowlist) must NOT be granted CORS.
func TestCORS_Allowlist_RejectsOtherOrigin(t *testing.T) {
	h := runCORS(t, []string{"https://app.example.com"}, "https://evil.example")
	if got := h.Get("Access-Control-Allow-Origin"); got == "https://evil.example" {
		t.Fatalf("non-allowlisted origin must NOT be reflected, got %q", got)
	}
}
