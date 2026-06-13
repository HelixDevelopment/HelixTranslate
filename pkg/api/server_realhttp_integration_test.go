//go:build integration

// Package api integration tests for the REST/HTTP API server (W15, api leg).
//
// These exercise the REAL pkg/api.Handler over REAL HTTP (net/http/httptest
// driving the real gin router with the real security.UserAuthService and a real
// in-memory user repository), and — for the persistence leg — a REAL ephemeral
// PostgreSQL booted on demand via the containers submodule's brokertest helper
// (digital.vasic.containers/pkg/brokertest). No mocks, no fakes for the
// user-visible surface (§11.4.27, §11.4.76 on-demand-infra invariant). The
// Postgres container is memory-limited (§12.6), bound to 127.0.0.1, and torn
// down on every exit path (§11.4.14).
//
// Run:  go test -tags=integration -run TestAPIRealHTTP ./pkg/api/
// Requires (Postgres leg only): a working container runtime (podman/docker).
// If absent/unreachable the Postgres leg SKIPs with reason (§11.4.3) rather
// than failing; the HTTP-surface legs need no container and always run.
//
// ANTI-BLUFF NOTES (§11.4):
//   - Every PASS asserts a concrete user-visible outcome: a real status code, a
//     real JSON field, a real JWT that validates, or a row that actually
//     round-trips through real PostgreSQL — never just "no error".
//   - The LLM *translation* leg requires a real provider API key (none is
//     available in CI). It is NOT faked: see TestAPIRealHTTP_TranslateText_SkipNoLLM,
//     which t.Skips with the §11.4.3 reason. A fabricated translation assertion
//     would be a §11.4 PASS-bluff and is forbidden.
//   - Architecture fact (NOT a bluff): pkg/api.Handler is wired to internal/cache
//     (in-memory) + models.NewInMemoryUserRepository — it does NOT import
//     pkg/storage, and GET /api/v1/status/:id returns a stubbed "completed".
//     There is therefore no api-HTTP → Postgres session seam to exercise.
//     The Postgres leg below boots a REAL Postgres and round-trips a session
//     through the REAL pkg/storage backend (the persistence layer the system
//     relies on) directly, asserting real INSERT/SELECT/UPDATE — proving the
//     storage round-trip the api ecosystem depends on actually persists.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"digital.vasic.containers/pkg/brokertest"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/cache"
	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/models"
	"digital.vasic.translator/pkg/security"
	"digital.vasic.translator/pkg/storage"
	"digital.vasic.translator/pkg/websocket"
)

const (
	itJWTSecret = "w15-integration-jwt-secret-32-bytes-min!"
	itUsername  = "alice"
	itPassword  = "correct horse battery staple"
)

// newRealAPIServer builds a REAL gin router with the REAL pkg/api.Handler,
// auth enabled, backed by a REAL security.UserAuthService over a REAL in-memory
// user repository seeded with one active user. Returned together with the live
// auth service so tests can mint genuinely-signed tokens.
func newRealAPIServer(t *testing.T) (*gin.Engine, *security.UserAuthService) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := models.NewInMemoryUserRepository()
	auth := security.NewUserAuthService(itJWTSecret, time.Hour, repo)

	// Seed a real user through the real repo (bcrypt-hashes the password).
	_, err := auth.CreateUser(security.CreateUserRequest{
		Username: itUsername,
		Email:    "alice@example.com",
		Password: itPassword,
		Roles:    []string{"user"},
	})
	require.NoError(t, err, "seed real user via real UserAuthService")

	cfg := &config.Config{}
	cfg.Security.EnableAuth = true
	cfg.Security.JWTSecret = itJWTSecret

	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	translationCache := cache.NewCache(time.Minute, true)

	h := NewHandler(cfg, eventBus, translationCache, auth, wsHub, nil)
	router := gin.New()
	h.RegisterRoutes(router)
	return router, auth
}

// TestAPIRealHTTP_HealthEndpoint asserts GET /health returns a real, well-formed
// status payload over a real HTTP roundtrip — not a loading spinner, an actual
// status field a client could act on.
func TestAPIRealHTTP_HealthEndpoint(t *testing.T) {
	router, _ := newRealAPIServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "health endpoint returns 200")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), "health body is valid JSON")
	assert.Equal(t, "healthy", body["status"], "real status field present and healthy")
	assert.NotEmpty(t, body["version"], "version present in health payload")
	assert.NotEmpty(t, body["time"], "server time present in health payload")
}

// TestAPIRealHTTP_JWTProtectedRoute drives the REAL authMiddleware on
// /api/v1/profile over real HTTP: no header -> 401, garbage token -> 401, and a
// genuinely-signed token from the real UserAuthService -> 200 with the real
// claims echoed back. This proves the JWT gate both rejects and accepts for the
// right reasons against the real signing key.
func TestAPIRealHTTP_JWTProtectedRoute(t *testing.T) {
	router, auth := newRealAPIServer(t)

	// (1) No Authorization header -> 401.
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/profile", nil)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code, "missing token rejected")
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "No authorization header", body["error"])
	}

	// (2) Invalid/garbage token -> 401 (real signature validation fails).
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/profile", nil)
		req.Header.Set("Authorization", "Bearer not.a.real.jwt")
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code, "invalid token rejected")
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "Invalid token", body["error"])
	}

	// (3) A real signed token -> 200, with the real claims echoed back.
	{
		token, err := auth.GenerateToken("uid-1", itUsername, []string{"user"})
		require.NoError(t, err, "mint a real JWT with the real signing key")
		require.NotEmpty(t, token)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/profile", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code, "valid token accepted")
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "uid-1", body["user_id"], "claims from the validated token are echoed")
		assert.Equal(t, itUsername, body["username"])
		roles, _ := body["roles"].([]any)
		require.Len(t, roles, 1, "roles claim round-trips")
		assert.Equal(t, "user", roles[0])
	}
}

// TestAPIRealHTTP_LoginTokenRoundTrip exercises the full login flow over real
// HTTP: POST /api/v1/auth/login with the seeded user's real credentials returns
// a real JWT, and that JWT then unlocks the protected /api/v1/profile route.
// Wrong credentials are rejected 401. This is the real end-to-end auth journey.
func TestAPIRealHTTP_LoginTokenRoundTrip(t *testing.T) {
	router, _ := newRealAPIServer(t)

	// Wrong password -> 401 (real bcrypt comparison fails).
	{
		bad, _ := json.Marshal(security.LoginRequest{Username: itUsername, Password: "wrong"})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(bad))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code, "wrong password rejected by real auth")
	}

	// Correct credentials -> 200 + real token.
	good, _ := json.Marshal(security.LoginRequest{Username: itUsername, Password: itPassword})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(good))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "valid login returns 200")
	var login security.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &login))
	require.NotEmpty(t, login.Token, "login returns a non-empty JWT")
	assert.Equal(t, itUsername, login.Username, "login echoes the authenticated username")

	// The returned token must actually unlock the protected route.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req2.Header.Set("Authorization", "Bearer "+login.Token)
	router.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, "the login-issued token unlocks /profile")
	var prof map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &prof))
	assert.Equal(t, itUsername, prof["username"], "profile reflects the logged-in user")
}

// verifierModelsPayload is a canned LLMsVerifier /api/models response mixing
// verified/unverified, can-see-code true/false, and above/below-threshold scores
// so the filtering branches are exercised against the REAL VerifierHandler.
const verifierModelsPayload = `[
  {"id":"m-good","provider_id":"openai","name":"Good","verification_status":"verified","can_see_code":true,"affirmative_response":true,"overall_score":0.9,"capabilities":{"code":true},"pricing":{"currency":"USD"},"last_verified_at":"2026-01-01T00:00:00Z"},
  {"id":"m-low","provider_id":"zhipu","name":"Low","verification_status":"verified","can_see_code":true,"affirmative_response":true,"overall_score":0.2}
]`

// TestAPIRealHTTP_VerifiedModels stands up a REAL httptest LLMsVerifier upstream,
// wires the REAL api.VerifierHandler at /api/v1, and asserts GET
// /api/v1/verified-models returns a real, well-formed, correctly-filtered
// payload over a real HTTP roundtrip (the m-low model is below the 0.5 threshold
// and must be filtered out; m-good must be present with its real fields).
func TestAPIRealHTTP_VerifiedModels(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
		case "/api/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(verifierModelsPayload))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(upstream.Close)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	vh := NewVerifierHandler(&verifier.Config{
		APIURL:            upstream.URL,
		APIKey:            "test",
		CacheTTL:          time.Hour,
		MinScoreThreshold: 0.5,
		ScoringWeights: verifier.ScoreWeights{
			ResponseSpeed: 0.2, CostEffectiveness: 0.3, ModelEfficiency: 0.25,
			Capability: 0.2, Recency: 0.05,
		},
	})
	vh.RegisterVerifierRoutes(router.Group("/api/v1"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verified-models", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "verified-models returns 200")
	var resp struct {
		Models []map[string]any `json:"models"`
		Count  int              `json:"count"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "payload is valid JSON")
	require.Equal(t, 1, resp.Count, "only the above-threshold verified model survives filtering")
	require.Len(t, resp.Models, 1)
	m := resp.Models[0]
	assert.Equal(t, "m-good", m["id"], "the real model id is returned")
	assert.Equal(t, "openai", m["provider_id"])
	assert.Equal(t, "verified", m["verification_status"])
	assert.InDelta(t, 0.9, m["overall_score"], 0.0001, "the real score round-trips")
}

// TestAPIRealHTTP_StoragePostgresSessionRoundTrip boots a REAL PostgreSQL via
// brokertest and round-trips a TranslationSession through the REAL pkg/storage
// backend the system relies on: create -> read back -> update status -> read the
// persisted change. Assertions are on the actually-persisted row (not just a
// 200), satisfying §11.4 "session/job create -> status round-trip persisted in
// REAL Postgres".
//
// NOTE (architecture fact, not a bluff): pkg/api.Handler does not import
// pkg/storage and its GET /api/v1/status/:id is a stubbed "completed"; there is
// no api-HTTP -> Postgres session seam to drive. This test therefore asserts the
// real persistence layer directly. If/when the api gains a storage seam, the
// HTTP path can layer on top of this proven round-trip.
func TestAPIRealHTTP_StoragePostgresSessionRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dsn, stop, err := brokertest.StartPostgres(ctx, brokertest.WithMemoryLimit("256m"))
	if err != nil {
		t.Skipf("SKIP-OK: container runtime unavailable for real Postgres — %v (§11.4.3 topology absent)", err)
	}
	defer stop() // §11.4.14 cleanup on every exit path

	u, err := url.Parse(dsn)
	require.NoError(t, err, "parse brokertest DSN")
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)
	pass, _ := u.User.Password()
	ssl := u.Query().Get("sslmode")
	if ssl == "" {
		ssl = "disable"
	}
	st, err := storage.NewPostgreSQLStorage(&storage.Config{
		Type:     "postgres",
		Host:     u.Hostname(),
		Port:     port,
		Database: u.Path[1:],
		Username: u.User.Username(),
		Password: pass,
		SSLMode:  ssl,
	})
	require.NoError(t, err, "NewPostgreSQLStorage against the booted Postgres")
	defer func() { _ = st.Close() }()
	require.NoError(t, st.Ping(ctx), "Ping the real Postgres")

	now := time.Now().UTC().Truncate(time.Second)
	sess := &storage.TranslationSession{
		ID:             "w15-api-job-1",
		BookTitle:      "Гавран и бокал",
		InputFile:      "in.fb2",
		OutputFile:     "out.epub",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		Status:         "queued",
		TotalChapters:  2,
		ItemsTotal:     5,
		StartTime:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	// CREATE (job create)
	require.NoError(t, st.CreateSession(ctx, sess), "CreateSession persists the job")

	// STATUS round-trip #1: the persisted row reads back with the created state.
	got, err := st.GetSession(ctx, "w15-api-job-1")
	require.NoError(t, err, "GetSession reads the persisted job")
	require.NotNil(t, got)
	assert.Equal(t, "queued", got.Status, "created status persisted")
	assert.Equal(t, "Гавран и бокал", got.BookTitle, "title round-trips incl. Cyrillic")

	// UPDATE status -> STATUS round-trip #2: the change is actually persisted.
	got.Status = "completed"
	got.PercentComplete = 100
	got.ItemsCompleted = 5
	require.NoError(t, st.UpdateSession(ctx, got), "UpdateSession persists the status change")

	after, err := st.GetSession(ctx, "w15-api-job-1")
	require.NoError(t, err)
	assert.Equal(t, "completed", after.Status, "status update is persisted in real Postgres")
	assert.InDelta(t, 100.0, after.PercentComplete, 0.001, "progress update persisted")
}

// TestAPIRealHTTP_TranslateText_SkipNoLLM documents the LLM leg honestly: the
// POST /api/v1/translate endpoint requires a real provider API key to produce a
// real translation. None is available, and fabricating a translated string would
// be a §11.4 PASS-bluff. We assert only the genuinely-reachable input-validation
// outcome (a malformed body is rejected 400 — a real, non-LLM user-visible
// outcome) and SKIP the actual translation assertion with reason.
func TestAPIRealHTTP_TranslateText_SkipNoLLM(t *testing.T) {
	router, _ := newRealAPIServer(t)

	// Real, non-LLM outcome we CAN assert: malformed request body -> 400.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/translate", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, "malformed translate body rejected 400 (real validation)")

	t.Skip("SKIP §11.4.3: actual translation needs a real LLM provider key; DB/auth/protocol surface covered separately")
}
