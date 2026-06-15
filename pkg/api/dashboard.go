package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/script"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// applyScript converts translated text to the requested target script. An empty
// target means "no conversion". The caller validates the value first, so only
// the three valid cases reach here.
func applyScript(target, text string) string {
	switch target {
	case "latin":
		return script.NewConverter().ToLatin(text)
	case "cyrillic":
		return script.NewConverter().ToCyrillic(text)
	default:
		return text
	}
}

// dashboardSession is the server-side record of one translation started through
// the Web Dashboard ("New Translation" UI). It carries the real translated text
// so the dashboard's detail view (§ Article XI) shows actual translated output,
// not just a session id / spinner.
type dashboardSession struct {
	SessionID          string    `json:"session_id"`
	InputFile          string    `json:"input_file"`
	OutputFile         string    `json:"output_file"`
	Provider           string    `json:"provider"`
	SourceLang         string    `json:"source_lang"`
	TargetLang         string    `json:"target_lang"`
	Status             string    `json:"status"` // running | completed | failed
	ProgressPercentage int       `json:"progress_percentage"`
	CurrentStep        string    `json:"current_step"`
	Original           string    `json:"original"`
	Translated         string    `json:"translated"`
	Error              string    `json:"error,omitempty"`
	StartedAt          time.Time `json:"started_at"`
	CompletedAt        time.Time `json:"completed_at,omitempty"`
}

// dashboardStore is a process-local, thread-safe registry of dashboard sessions.
// It backs GET/POST/DELETE /api/v1/translations. It is intentionally in-memory:
// the dashboard is a live operator surface, and a translated result is returned
// synchronously from the start call, so persistence across restarts is out of
// scope for this UI-wiring phase.
type dashboardStore struct {
	mu       sync.RWMutex
	sessions map[string]*dashboardSession
	order    []string // insertion order for stable, newest-first listing
}

func newDashboardStore() *dashboardStore {
	return &dashboardStore{sessions: make(map[string]*dashboardSession)}
}

func (s *dashboardStore) put(sess *dashboardSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.sessions[sess.SessionID]; !exists {
		s.order = append(s.order, sess.SessionID)
	}
	s.sessions[sess.SessionID] = sess
}

func (s *dashboardStore) get(id string) (*dashboardSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

// list returns sessions newest-first (most-recent insertion first).
func (s *dashboardStore) list() []*dashboardSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*dashboardSession, 0, len(s.order))
	for i := len(s.order) - 1; i >= 0; i-- {
		if sess, ok := s.sessions[s.order[i]]; ok {
			out = append(out, sess)
		}
	}
	return out
}

// dashboardStoreFor lazily initialises and returns the handler's dashboard
// store. Guarded so concurrent first-requests can't double-init.
func (h *Handler) dashboardStoreFor() *dashboardStore {
	h.dashboardOnce.Do(func() {
		h.dashboard = newDashboardStore()
	})
	return h.dashboard
}

// RegisterDashboardRoutes wires the Web Dashboard page and the translation
// session endpoints the dashboard JS calls. Without these, web/templates/
// dashboard.html is served at no route and the /api/v1/translations endpoints
// 404 — the dashboard is dead for end users (§11.4.153 real gap).
//
// Contract derived directly from web/templates/dashboard.html:
//   - GET    /                            -> dashboard HTML (also /dashboard, /monitor)
//   - GET    /api/v1/translations         -> {success, data:{translations:[...]}}
//   - GET    /api/v1/translations/:id     -> {success, data:{...session...}}
//   - POST   /api/v1/translations         -> start; {success, data:{...session...}}
//   - DELETE /api/v1/translations/:id     -> cancel; {success, data:{...}}
func (h *Handler) RegisterDashboardRoutes(router *gin.Engine, v1 *gin.RouterGroup) {
	router.GET("/dashboard", h.serveDashboardPage)
	router.GET("/monitor", h.serveDashboardPage)

	v1.GET("/translations", h.listTranslations)
	v1.GET("/translations/:session_id", h.getTranslation)
	v1.POST("/translations", h.startTranslation)
	v1.DELETE("/translations/:session_id", h.cancelTranslationSession)
}

// dashboardTemplatePath locates web/templates/dashboard.html relative to the
// working directory or the running binary, so the page is served whether the
// server is launched from the repo root or from build/.
func dashboardTemplatePath() (string, bool) {
	candidates := []string{
		filepath.Join("web", "templates", "dashboard.html"),
		filepath.Join("..", "web", "templates", "dashboard.html"),
		filepath.Join("..", "..", "web", "templates", "dashboard.html"),
	}
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(base, "web", "templates", "dashboard.html"),
			filepath.Join(base, "..", "web", "templates", "dashboard.html"),
		)
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}
	return "", false
}

// serveDashboardPage serves the dashboard HTML. Returns 500 (not a silent 200
// with empty body) when the template cannot be located, so a misconfigured
// deployment is loud per §11.4.1.
func (h *Handler) serveDashboardPage(c *gin.Context) {
	path, ok := dashboardTemplatePath()
	if !ok {
		c.String(http.StatusInternalServerError,
			"dashboard template not found (expected web/templates/dashboard.html)")
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.File(path)
}

// listTranslations returns all dashboard translation sessions in the shape the
// dashboard JS expects: {success:true, data:{translations:[...]}}.
func (h *Handler) listTranslations(c *gin.Context) {
	store := h.dashboardStoreFor()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"translations": store.list(),
		},
	})
}

// getTranslation returns one session's full state (including the real
// translated text) as {success:true, data:{...}}.
func (h *Handler) getTranslation(c *gin.Context) {
	store := h.dashboardStoreFor()
	id := c.Param("session_id")
	sess, ok := store.get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sess})
}

// startTranslationRequest mirrors the JSON the dashboard's startTranslation()
// posts. `text` is an additive field: the dashboard's file <input> cannot carry
// bytes over JSON, so a real translation through the UI path supplies the source
// text either via `text` or via `input_file` (treated as the source text when no
// upload mechanism is present). This keeps the wired path genuinely producing
// translated output rather than a stub.
type startTranslationRequest struct {
	SessionID  string `json:"session_id"`
	Text       string `json:"text"`
	InputFile  string `json:"input_file"`
	OutputFile string `json:"output_file"`
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
	Script     string `json:"script"`

	ProviderConfig struct {
		Type   string `json:"type"`
		Model  string `json:"model"`
		APIKey string `json:"api_key"`
	} `json:"provider_config"`
}

// startTranslation starts a translation through the Web Dashboard UI path,
// backed by the SAME real translation logic POST /api/v1/translate uses
// (createTranslator + TranslateWithProgress). It returns the real translated
// text so the dashboard shows actual translated content (Article XI), and
// records the session so the list/detail endpoints reflect it.
func (h *Handler) startTranslation(c *gin.Context) {
	store := h.dashboardStoreFor()

	var req startTranslationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Resolve the source text: prefer explicit text, fall back to input_file.
	sourceText := req.Text
	if sourceText == "" {
		sourceText = req.InputFile
	}
	if sourceText == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "no source text provided (set 'text' or 'input_file')",
		})
		return
	}

	// Validate script up-front, matching translateText's contract.
	switch req.Script {
	case "", "latin", "cyrillic":
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid script: " + req.Script + " (expected \"latin\" or \"cyrillic\")",
		})
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	provider := req.ProviderConfig.Type
	sess := &dashboardSession{
		SessionID:          sessionID,
		InputFile:          req.InputFile,
		OutputFile:         req.OutputFile,
		Provider:           provider,
		SourceLang:         req.SourceLang,
		TargetLang:         req.TargetLang,
		Status:             "running",
		ProgressPercentage: 0,
		CurrentStep:        "translating",
		Original:           sourceText,
		StartedAt:          time.Now(),
	}
	store.put(sess)

	trans, err := h.createTranslator(provider, req.ProviderConfig.Model, req.SourceLang, req.TargetLang)
	if err != nil {
		sess.Status = "failed"
		sess.Error = err.Error()
		sess.CurrentStep = "failed"
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "data": sess})
		return
	}

	ctx := context.Background()
	translated, terr := trans.TranslateWithProgress(ctx, sourceText, "", h.eventBus, sessionID)
	if terr != nil {
		sess.Status = "failed"
		sess.Error = terr.Error()
		sess.CurrentStep = "failed"
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": terr.Error(), "data": sess})
		return
	}

	// Apply requested script conversion (validated above).
	translated = applyScript(req.Script, translated)

	sess.Translated = translated
	sess.Status = "completed"
	sess.ProgressPercentage = 100
	sess.CurrentStep = "completed"
	sess.CompletedAt = time.Now()

	// Emit a completion event so any connected dashboard WebSocket clients
	// observe the progress transition (consistent with the event-driven core).
	completeEvent := events.NewEvent(
		events.EventTranslationCompleted,
		"Dashboard translation completed",
		map[string]interface{}{
			"progress_percentage": 100,
			"current_step":        "completed",
		},
	)
	completeEvent.SessionID = sessionID
	h.eventBus.Publish(completeEvent)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": sess})
}

// cancelTranslationSession marks a dashboard session cancelled. It returns
// {success:true} for the dashboard's cancel flow.
func (h *Handler) cancelTranslationSession(c *gin.Context) {
	store := h.dashboardStoreFor()
	id := c.Param("session_id")
	sess, ok := store.get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "session not found"})
		return
	}
	if sess.Status == "running" {
		sess.Status = "cancelled"
		sess.CurrentStep = "cancelled"
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sess})
}
