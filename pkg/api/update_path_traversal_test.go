package api

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestUploadUpdate_PathTraversal_Rejected proves the X-Update-Version header
// cannot be used to write the uploaded package OUTSIDE the canonical update
// directory (/tmp/translator-updates) via "../" path traversal segments.
//
// FACT root cause (pre-fix): uploadUpdate took the version verbatim from the
// X-Update-Version header (validated only for emptiness) and interpolated it
// into a filename, then filepath.Join(updateDir, "update-<version>.tar.gz").
// filepath.Join CLEANS "../" segments, so a version of
// "../translator-updates-pwned/evil" resolves to a path OUTSIDE updateDir —
// CWE-22 arbitrary file write. This test sends such a version and asserts the
// server REJECTS it (400) and writes NOTHING to the escaped location.
func TestUploadUpdate_PathTraversal_Rejected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	updateDir := "/tmp/translator-updates"
	// The escaped location a "../" version would resolve to.
	escapedDir := "/tmp/translator-updates-pwned"
	escapedFile := filepath.Join(escapedDir, "evil.tar.gz")

	// Clean slate.
	_ = os.RemoveAll(escapedDir)
	defer os.RemoveAll(escapedDir)

	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	handler := &Handler{config: cfg, eventBus: eventBus, wsHub: wsHub}

	router := gin.New()
	router.POST("/api/v1/update/upload", handler.uploadUpdate)

	// Build a multipart body with the required "update_package" file part.
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("update_package", "pkg.tar.gz")
	assert.NoError(t, err)
	_, err = fw.Write([]byte("malicious payload"))
	assert.NoError(t, err)
	assert.NoError(t, mw.Close())

	req, _ := http.NewRequest("POST", "/api/v1/update/upload", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	// Traversal: filepath.Join("/tmp/translator-updates",
	//   "update-../translator-updates-pwned/evil.tar.gz")
	//   == "/tmp/translator-updates-pwned/evil.tar.gz" (escapes updateDir).
	req.Header.Set("X-Update-Version", "../translator-updates-pwned/evil")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// The traversal version MUST be rejected with 400 and MUST NOT have
	// written any file outside the canonical update directory.
	assert.Equal(t, http.StatusBadRequest, w.Code,
		"traversal version must be rejected; got body=%s", w.Body.String())

	_, statErr := os.Stat(escapedFile)
	assert.True(t, os.IsNotExist(statErr),
		"file MUST NOT be written outside updateDir; found %s", escapedFile)

	// Defensive: also ensure the written path (if any) stays under updateDir.
	if statErr == nil {
		abs, _ := filepath.Abs(escapedFile)
		t.Fatalf("path traversal write escaped to %s (updateDir=%s)", abs, updateDir)
	}
}

// TestApplyUpdate_PathTraversal_Rejected proves applyUpdate likewise refuses a
// traversal version rather than resolving it to a file outside updateDir.
func TestApplyUpdate_PathTraversal_Rejected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	handler := &Handler{config: cfg, eventBus: eventBus, wsHub: wsHub}

	router := gin.New()
	router.POST("/api/v1/update/apply", handler.applyUpdate)

	req, _ := http.NewRequest("POST", "/api/v1/update/apply", nil)
	req.Header.Set("X-Update-Version", "../../etc/passwd")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Must be a clean 400 rejection (not a 404 produced after resolving a
	// traversed path, and certainly not applying anything).
	assert.Equal(t, http.StatusBadRequest, w.Code,
		"traversal version must be rejected; got body=%s", w.Body.String())
}

// TestUploadUpdate_NormalVersion_Accepted is the GREEN-side control: a normal
// version still lands INSIDE the canonical update directory, proving the fix
// did not break the happy path.
func TestUploadUpdate_NormalVersion_Accepted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	updateDir := "/tmp/translator-updates"
	version := "9.9.9-traversaltest"
	expectedPath := filepath.Join(updateDir, "update-"+version+".tar.gz")
	defer os.Remove(expectedPath)

	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	handler := &Handler{config: cfg, eventBus: eventBus, wsHub: wsHub}

	router := gin.New()
	router.POST("/api/v1/update/upload", handler.uploadUpdate)

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("update_package", "pkg.tar.gz")
	assert.NoError(t, err)
	_, err = fw.Write([]byte("legit payload"))
	assert.NoError(t, err)
	assert.NoError(t, mw.Close())

	req, _ := http.NewRequest("POST", "/api/v1/update/upload", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Update-Version", version)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "normal version must be accepted; body=%s", w.Body.String())
	_, statErr := os.Stat(expectedPath)
	assert.NoError(t, statErr, "package must be written inside updateDir at %s", expectedPath)
}
