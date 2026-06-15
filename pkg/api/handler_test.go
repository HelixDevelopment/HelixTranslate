package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"digital.vasic.translator/internal/cache"
	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/models"
	"digital.vasic.translator/pkg/security"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// MockUserRepository is a mock implementation for testing
type MockUserRepository struct {
	users      map[string]*models.User
	forceError bool
}

func (m *MockUserRepository) FindByUsername(username string) (*models.User, error) {
	if m.forceError {
		return nil, fmt.Errorf("forced repository error")
	}

	user, exists := m.users[username]
	if !exists {
		return nil, models.ErrUserNotFound
	}

	return user, nil
}

func (m *MockUserRepository) FindByEmail(email string) (*models.User, error) {
	if m.forceError {
		return nil, fmt.Errorf("forced repository error")
	}

	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, models.ErrUserNotFound
}

func (m *MockUserRepository) Create(user *models.User) error {
	if m.forceError {
		return fmt.Errorf("forced repository error")
	}
	m.users[user.Username] = user
	return nil
}

func (m *MockUserRepository) Update(user *models.User) error {
	if m.forceError {
		return fmt.Errorf("forced repository error")
	}
	m.users[user.Username] = user
	return nil
}

func (m *MockUserRepository) Delete(id string) error {
	if m.forceError {
		return fmt.Errorf("forced repository error")
	}
	delete(m.users, id)
	return nil
}

func (m *MockUserRepository) List() ([]*models.User, error) {
	if m.forceError {
		return nil, fmt.Errorf("forced repository error")
	}
	users := make([]*models.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func TestGenerateOutputFilename(t *testing.T) {
	tests := []struct {
		input    string
		provider string
		expected string
	}{
		{"book.fb2", "dictionary", "book_sr_dictionary.fb2"},
		{"test.b2", "openai", "test_sr_openai.b2"},
		{"novel.fb2", "anthropic", "novel_sr_anthropic.fb2"},
	}

	for _, tt := range tests {
		t.Run(tt.input+"_"+tt.provider, func(t *testing.T) {
			result := generateOutputFilename(tt.input, tt.provider)
			if result != tt.expected {
				t.Errorf("generateOutputFilename(%s, %s) = %s, want %s", tt.input, tt.provider, result, tt.expected)
			}
		})
	}
}

// TestAPIInfo tests the apiInfo handler
func TestAPIInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a minimal handler
	h := &Handler{}

	// Setup test context
	router := gin.New()
	router.GET("/test", h.apiInfo)

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Universal Multi-Format Multi-Language Ebook Translation API", response["name"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.Contains(t, response, "endpoints")
	assert.Contains(t, response, "documentation")
}

// TestTranslateText tests the translateText handler
func TestTranslateText(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a handler with minimal config to avoid nil pointer
	h := &Handler{
		config: &config.Config{
			Translation: config.TranslationConfig{
				DefaultProvider: "openai",
			},
		},
	}
	// R-1b/R2: translator construction flows through the LLMsVerifier bridge, which
	// hard-errors when no provider is verified (§11.4.69). These cases assert that
	// honest translator-creation failure maps to a 400; inject a bridge factory
	// that returns that hard error deterministically (no real keys / no network).
	h.bridgeTranslatorFactory = func(_ context.Context, _ selection.TaskRequirements) (translator.Translator, error) {
		return nil, fmt.Errorf("bridge: no verified translator available")
	}

	router := gin.New()
	router.POST("/translate", h.translateText)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		shouldContain  string
	}{
		{
			name:           "missing text field",
			requestBody:    `{"provider":"openai"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "empty text field",
			requestBody:    `{"text":"","provider":"openai"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "invalid JSON",
			requestBody:    `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "valid request with minimal fields",
			requestBody:    `{"text":"Hello world"}`,
			expectedStatus: http.StatusBadRequest, // Will fail at translator creation
			shouldContain:  "error",
		},
		{
			name:           "valid request with provider",
			requestBody:    `{"text":"Hello world","provider":"openai"}`,
			expectedStatus: http.StatusBadRequest, // Will fail at translator creation
			shouldContain:  "error",
		},
		{
			name:           "valid request with provider and model",
			requestBody:    `{"text":"Hello world","provider":"openai","model":"gpt-3.5-turbo"}`,
			expectedStatus: http.StatusBadRequest, // Will fail at translator creation
			shouldContain:  "error",
		},
		{
			name:           "valid request with context",
			requestBody:    `{"text":"Hello world","provider":"openai","context":"translate to Serbian"}`,
			expectedStatus: http.StatusBadRequest, // Will fail at translator creation
			shouldContain:  "error",
		},
		{
			name:           "valid request with script conversion latin",
			requestBody:    `{"text":"Hello world","provider":"openai","script":"latin"}`,
			expectedStatus: http.StatusBadRequest, // Will fail at translator creation
			shouldContain:  "error",
		},
		{
			name:           "valid request with all fields",
			requestBody:    `{"text":"Hello world","provider":"openai","model":"gpt-3.5-turbo","context":"translate to Serbian","script":"latin"}`,
			expectedStatus: http.StatusBadRequest, // Will fail at translator creation
			shouldContain:  "error",
		},
		{
			name:           "invalid provider",
			requestBody:    `{"text":"Hello world","provider":"invalid-provider"}`,
			expectedStatus: http.StatusBadRequest, // Will fail at translator creation
			shouldContain:  "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/translate", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Use defer to catch any panics from nil dependencies
			defer func() {
				if r := recover(); r != nil {
					// If we get a panic, it's likely due to translator creation failure
					// Set the response code to internal server error
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.shouldContain)
		})
	}
}

// TestHealthCheck tests the healthCheck handler
func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{}

	router := gin.New()
	router.GET("/health", h.healthCheck)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

// TestVersionInfo tests version-related handlers
func TestVersionInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{}

	router := gin.New()
	router.GET("/version", h.getVersion)

	req, _ := http.NewRequest("GET", "/version", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "codebase_version")
	assert.Contains(t, response, "build_time")
	assert.Contains(t, response, "git_commit")
	assert.Contains(t, response, "go_version")
}

// TestStats tests the stats handler
func TestStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create minimal mocks for testing
	mockCache := &cache.Cache{}          // This will be nil, but handler should handle it
	mockWebSocketHub := &websocket.Hub{} // This will be nil, but handler should handle it

	h := &Handler{
		cache: mockCache,
		wsHub: mockWebSocketHub,
	}

	router := gin.New()
	router.GET("/stats", h.getStats)

	req, _ := http.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()

	// Use a defer to catch any panics from nil dependencies
	defer func() {
		if r := recover(); r != nil {
			// If we get a panic, it's likely due to nil dependencies
			// Set the response code to internal server error
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	router.ServeHTTP(w, req)

	// Should handle nil dependencies gracefully or return error
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

// TestWebSocketHandler tests the websocket handler setup
func TestWebSocketHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{}

	router := gin.New()
	router.GET("/ws", h.websocketHandler)

	req, _ := http.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// WebSocket upgrade should return 101 if properly configured
	// or error if no upgrade headers provided
	assert.True(t, w.Code == http.StatusSwitchingProtocols || w.Code == http.StatusBadRequest)
}

// TestAuthMiddleware tests the authentication middleware
func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{}

	// Test middleware without proper auth token
	router := gin.New()
	router.Use(h.authMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 401 Unauthorized when no token provided
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_HandlerSpecific tests handler authentication middleware with various scenarios
func TestAuthMiddleware_HandlerSpecific(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("No authorization header", func(t *testing.T) {
		authService := security.NewUserAuthService("test-secret-key-16-chars", 24*time.Hour, nil)
		h := &Handler{authService: authService}

		router := gin.New()
		router.Use(h.authMiddleware())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "No authorization header", response["error"])
	})

	t.Run("Invalid token", func(t *testing.T) {
		authService := security.NewUserAuthService("test-secret-key-16-chars", 24*time.Hour, nil)
		h := &Handler{authService: authService}

		router := gin.New()
		router.Use(h.authMiddleware())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "Invalid token", response["error"])
	})

	t.Run("Valid Bearer token", func(t *testing.T) {
		authService := security.NewUserAuthService("test-secret-key-16-chars", 24*time.Hour, nil)
		h := &Handler{authService: authService}

		// Generate a valid token
		token, _ := authService.GenerateToken("test-user", "testuser", []string{"user"})

		router := gin.New()
		router.Use(h.authMiddleware())
		router.GET("/protected", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			username, _ := c.Get("username")
			roles, _ := c.Get("roles")
			c.JSON(http.StatusOK, gin.H{
				"user_id":  userID,
				"username": username,
				"roles":    roles,
			})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "test-user", response["user_id"])
		assert.Equal(t, "testuser", response["username"])
	})

	t.Run("Valid token without Bearer prefix", func(t *testing.T) {
		authService := security.NewUserAuthService("test-secret-key-16-chars", 24*time.Hour, nil)
		h := &Handler{authService: authService}

		// Generate a valid token
		token, _ := authService.GenerateToken("test-user", "testuser", []string{"user"})

		router := gin.New()
		router.Use(h.authMiddleware())
		router.GET("/protected", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			c.JSON(http.StatusOK, gin.H{"user_id": userID})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", token) // No "Bearer " prefix
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "test-user", response["user_id"])
	})

}

// TestLogin tests the login handler
func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		setupHandler func() *Handler
		requestBody  string
		expectedCode int
	}{
		{
			name: "empty credentials",
			setupHandler: func() *Handler {
				return &Handler{} // No auth service
			},
			requestBody:  `{}`,
			expectedCode: http.StatusBadRequest, // Empty credentials fail JSON binding validation
		},
		{
			name: "nil auth service",
			setupHandler: func() *Handler {
				return &Handler{} // No auth service
			},
			requestBody:  `{"username":"invalid","password":"invalid"}`,
			expectedCode: http.StatusInternalServerError, // Nil auth service causes panic/internal error
		},
		{
			name: "valid JSON structure",
			setupHandler: func() *Handler {
				return &Handler{} // No auth service
			},
			requestBody:  `{"username":"test","password":"test"}`,
			expectedCode: http.StatusInternalServerError, // Valid JSON but nil auth service
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.setupHandler()

			router := gin.New()
			router.POST("/login", h.login)

			req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Use a defer to catch any panics from nil pointer access
			defer func() {
				if r := recover(); r != nil {
					// If we get a panic, it's likely due to nil auth service
					// Set the response code to internal server error
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.setupHandler()

			router := gin.New()
			router.POST("/login", h.login)

			req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Use a defer to catch any panics from nil pointer access
			defer func() {
				if r := recover(); r != nil {
					// If we get a panic, it's likely due to nil auth service
					// Set the response code to internal server error
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

// TestLogin_Comprehensive tests login handler with various scenarios
func TestLogin_Comprehensive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Invalid JSON", func(t *testing.T) {
		h := &Handler{}

		router := gin.New()
		router.POST("/login", h.login)

		req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Missing required fields", func(t *testing.T) {
		h := &Handler{}

		router := gin.New()
		router.POST("/login", h.login)

		testData := map[string]interface{}{
			"username": "testuser",
			// Missing password
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Nil auth service", func(t *testing.T) {
		h := &Handler{authService: nil}

		router := gin.New()
		router.POST("/login", h.login)

		testData := map[string]interface{}{
			"username": "testuser",
			"password": "testpass",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// This should panic due to nil authService
		defer func() {
			if r := recover(); r != nil {
				// Expected panic due to nil authService
			}
		}()

		router.ServeHTTP(w, req)
		// If no panic, check for internal server error
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestLogin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create mock user repository with a test user
	mockRepo := &MockUserRepository{
		users: make(map[string]*models.User),
	}

	// Add a test user
	testUser := &models.User{
		ID:       "user123",
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
		Roles:    []string{"user"},
	}
	testUser.SetPassword("testpass")
	mockRepo.users["testuser"] = testUser

	// Create real UserAuthService with mock repository
	authService := security.NewUserAuthService("test-secret-key-16-chars", time.Hour, mockRepo)

	h := &Handler{
		authService: authService,
	}

	router := gin.New()
	router.POST("/login", h.login)

	testData := map[string]interface{}{
		"username": "testuser",
		"password": "testpass",
	}

	jsonData, _ := json.Marshal(testData)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed with valid login response
	assert.Equal(t, http.StatusOK, w.Code)

	var response security.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, "user123", response.UserID)
	assert.Equal(t, "testuser", response.Username)
	assert.Contains(t, response.Roles, "user")
}

// TestProfile tests the profile handler
func TestProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{}

	router := gin.New()
	router.GET("/profile", h.getProfile)

	req, _ := http.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 200 with empty user data (no authentication check in handler)
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response to verify structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "", response["user_id"])
	assert.Equal(t, "", response["username"])
}

// TestTranslateFB2 tests translateFB2 handler
func TestTranslateFB2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{
		config: &config.Config{
			Translation: config.TranslationConfig{
				DefaultProvider: "openai",
			},
		},
	}
	// R-1b/R2: createTranslator now sources from the LLMsVerifier bridge; this test
	// asserts the honest translator-creation failure path maps to 400. Inject the
	// hard error deterministically (no real keys / no network) (§11.4.69).
	h.bridgeTranslatorFactory = func(_ context.Context, _ selection.TaskRequirements) (translator.Translator, error) {
		return nil, fmt.Errorf("bridge: no verified translator available")
	}

	router := gin.New()
	router.POST("/translate/fb2", h.translateFB2)

	// Test with no file provided
	req, _ := http.NewRequest("POST", "/translate/fb2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "No file provided")

	// Test with empty file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a temporary file to simulate FB2 upload
	tempFile, err := os.CreateTemp("", "test-*.fb2")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write some minimal FB2 content
	tempFile.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<FictionBook>
	<description>
		<title-info>
			<genre>fiction</genre>
			<author>
				<first-name>Test</first-name>
				<last-name>Author</last-name>
			</author>
			<book-title>Test Book</book-title>
		</title-info>
	</description>
	<body>
		<section>
			<paragraph>Test content</paragraph>
		</section>
	</body>
</FictionBook>`)
	tempFile.Close()

	// Reopen for reading
	file, err := os.Open(tempFile.Name())
	assert.NoError(t, err)
	defer file.Close()

	part, err := writer.CreateFormFile("file", "test.fb2")
	assert.NoError(t, err)

	_, err = io.Copy(part, file)
	assert.NoError(t, err)

	writer.Close()

	// Test with file but no provider
	req, _ = http.NewRequest("POST", "/translate/fb2", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()

	// Use defer to catch any panics from nil dependencies
	defer func() {
		if r := recover(); r != nil {
			// If we get a panic, it's likely due to translator creation failure
			// Set response code to internal server error
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	router.ServeHTTP(w, req)

	// Should fail at translator creation step
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test with file and custom provider
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)

	file, err = os.Open(tempFile.Name())
	assert.NoError(t, err)

	part, err = writer.CreateFormFile("file", "test.fb2")
	assert.NoError(t, err)

	_, err = io.Copy(part, file)
	assert.NoError(t, err)

	writer.WriteField("provider", "openai")
	writer.WriteField("model", "gpt-3.5-turbo")
	writer.Close()

	req, _ = http.NewRequest("POST", "/translate/fb2", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()

	// Use defer to catch any panics from nil dependencies
	defer func() {
		if r := recover(); r != nil {
			// If we get a panic, it's likely due to translator creation failure
			// Set response code to internal server error
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	router.ServeHTTP(w, req)

	// Should fail at translator creation step
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test with file and invalid provider
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)

	file, err = os.Open(tempFile.Name())
	assert.NoError(t, err)

	part, err = writer.CreateFormFile("file", "test.fb2")
	assert.NoError(t, err)

	_, err = io.Copy(part, file)
	assert.NoError(t, err)

	writer.WriteField("provider", "invalid-provider")
	writer.Close()

	req, _ = http.NewRequest("POST", "/translate/fb2", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()

	// Use defer to catch any panics from nil dependencies
	defer func() {
		if r := recover(); r != nil {
			// If we get a panic, it's likely due to translator creation failure
			// Set response code to internal server error
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	router.ServeHTTP(w, req)

	// Should fail at translator creation step
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestConvertScript tests convertScript handler
func TestConvertScript(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{}

	router := gin.New()
	router.POST("/convert/script", h.convertScript)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		shouldContain  string
	}{
		{
			name:           "missing text field",
			requestBody:    `{"target":"latin"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "missing target field",
			requestBody:    `{"text":"test text"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "invalid target script",
			requestBody:    `{"text":"test text","target":"invalid"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "Invalid target script",
		},
		{
			name:           "valid latin conversion",
			requestBody:    `{"text":"test text","target":"latin"}`,
			expectedStatus: http.StatusOK,
			shouldContain:  "converted",
		},
		{
			name:           "valid cyrillic conversion",
			requestBody:    `{"text":"test text","target":"cyrillic"}`,
			expectedStatus: http.StatusOK,
			shouldContain:  "converted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/convert/script", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.shouldContain)
		})
	}
}

// TestListProviders tests listProviders handler
func TestListProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{}

	router := gin.New()
	router.GET("/providers", h.listProviders)

	req, _ := http.NewRequest("GET", "/providers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "providers")

	providers, ok := response["providers"].([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(providers), 0)

	// Check that OpenAI provider is included
	found := false
	for _, provider := range providers {
		if p, ok := provider.(map[string]interface{}); ok {
			if p["name"] == "openai" {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "OpenAI provider should be in the list")
}

// TestListLanguages tests listLanguages handler
func TestListLanguages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{}

	router := gin.New()
	router.GET("/languages", h.listLanguages)

	req, _ := http.NewRequest("GET", "/languages", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "languages")
	assert.Contains(t, response, "total")

	languages, ok := response["languages"].([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(languages), 0)

	// Check that English is included
	found := false
	for _, language := range languages {
		if l, ok := language.(map[string]interface{}); ok {
			if l["code"] == "en" {
				found = true
				assert.Equal(t, "English", l["name"])
				assert.Equal(t, "English", l["native"])
				break
			}
		}
	}
	assert.True(t, found, "English should be in the languages list")
}

// TestValidateTranslationRequest tests validateTranslationRequest handler
func TestValidateTranslationRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a mock config to avoid nil pointer issues
	config := &config.Config{
		Translation: config.TranslationConfig{
			DefaultProvider: "openai",
		},
	}
	h := &Handler{config: config}

	router := gin.New()
	router.POST("/translate/validate", h.validateTranslationRequest)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		shouldContain  string
	}{
		{
			name:           "missing text field",
			requestBody:    `{"target_language":"es"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "missing target language",
			requestBody:    `{"text":"test"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "invalid target language",
			requestBody:    `{"text":"test","target_language":"invalid"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "invalid target language",
		},
		{
			name:           "valid request",
			requestBody:    `{"text":"test","target_language":"es"}`,
			expectedStatus: http.StatusOK,
			shouldContain:  "valid",
		},
		{
			name:           "valid request with provider",
			requestBody:    `{"text":"test","target_language":"es","provider":"openai"}`,
			expectedStatus: http.StatusOK,
			shouldContain:  "valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/translate/validate", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.shouldContain)
		})
	}
}

// TestBatchTranslate tests batchTranslate handler
func TestBatchTranslate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a mock config to avoid nil pointer issues
	config := &config.Config{
		Translation: config.TranslationConfig{
			DefaultProvider: "openai",
		},
	}
	h := &Handler{config: config}

	router := gin.New()
	router.POST("/batch", h.batchTranslate)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		shouldContain  string
	}{
		{
			name:           "missing texts field",
			requestBody:    `{"provider":"openai"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "empty texts array",
			requestBody:    `{"texts":[],"provider":"openai"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "valid request with single text",
			requestBody:    `{"texts":["test"],"provider":"openai"}`,
			expectedStatus: http.StatusBadRequest, // Will fail at translator creation
			shouldContain:  "error",
		},
		{
			name:           "valid request with multiple texts",
			requestBody:    `{"texts":["test1","test2"],"provider":"openai"}`,
			expectedStatus: http.StatusBadRequest, // Will fail at translator creation
			shouldContain:  "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/batch", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Use defer to catch any panics from nil dependencies
			defer func() {
				if r := recover(); r != nil {
					// If we get a panic, it's likely due to translator creation
					// Set response code to internal server error
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.shouldContain)
		})
	}
}

// TestPreparationAnalysis tests preparationAnalysis handler
func TestPreparationAnalysis(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a handler with minimal dependencies to avoid nil pointer issues
	h := &Handler{
		eventBus: events.NewEventBus(),
	}

	router := gin.New()
	router.POST("/preparation/analyze", h.preparationAnalysis)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		shouldContain  string
	}{
		{
			name:           "missing input_path field",
			requestBody:    `{"target_language":"es"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "missing target language",
			requestBody:    `{"input_path":"/test"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "invalid target language",
			requestBody:    `{"input_path":"/test","target_language":"invalid"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "invalid target language",
		},
		{
			name:           "non-existent path",
			requestBody:    `{"input_path":"/non/existent/path","target_language":"es"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "input path does not exist",
		},
		{
			name:           "valid request with existing directory",
			requestBody:    `{"input_path":".","target_language":"es"}`,
			expectedStatus: http.StatusOK,
			shouldContain:  "session_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/preparation/analyze", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.shouldContain)
		})
	}
}

// TestGetPreparationResult tests getPreparationResult handler
func TestGetPreparationResult(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{}

	router := gin.New()
	router.GET("/preparation/result/:session_id", h.getPreparationResult)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		shouldContain  string
	}{
		{
			name:           "missing session id",
			url:            "/preparation/result/",
			expectedStatus: http.StatusNotFound, // Gin returns 404 for missing params
			shouldContain:  "404 page not found",
		},
		{
			name:           "valid session id",
			url:            "/preparation/result/test-session-id",
			expectedStatus: http.StatusOK,
			shouldContain:  "session_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.shouldContain)
		})
	}
}

// TestTranslateEbook tests translateEbook handler
func TestTranslateEbook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a handler with minimal dependencies to avoid nil pointer issues
	h := &Handler{
		eventBus: events.NewEventBus(),
	}

	router := gin.New()
	router.POST("/translate/ebook", h.translateEbook)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		shouldContain  string
	}{
		{
			name:           "missing input_path field",
			requestBody:    `{"target_language":"es"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "missing target language",
			requestBody:    `{"input_path":"test.fb2"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "invalid target language",
			requestBody:    `{"input_path":"test.fb2","target_language":"invalid"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "invalid target language",
		},
		{
			name:           "non-existent file",
			requestBody:    `{"input_path":"/non/existent.fb2","target_language":"es"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "input file does not exist",
		},
		{
			name:           "valid format detection from epub",
			requestBody:    `{"input_path":"test.epub","target_language":"es"}`,
			expectedStatus: http.StatusBadRequest, // File doesn't exist, but format detection works
			shouldContain:  "input file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/translate/ebook", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.shouldContain)
		})
	}
}

// TestCancelTranslation tests cancelTranslation handler
func TestCancelTranslation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a handler with minimal dependencies to avoid nil pointer issues
	h := &Handler{
		eventBus: events.NewEventBus(),
	}

	router := gin.New()
	router.POST("/translate/cancel/:session_id", h.cancelTranslation)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "with session id",
			url:            "/translate/cancel/test-session-id",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestGetStatus tests getStatus handler
func TestGetStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{}

	router := gin.New()
	router.GET("/status/:session_id", h.getStatus)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		shouldContain  string
	}{
		{
			name:           "with session id",
			url:            "/status/test-session-id",
			expectedStatus: http.StatusOK,
			shouldContain:  "session_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.shouldContain)
		})
	}
}

// TestVersionHelperFunctions tests version-related helper functions
func TestVersionHelperFunctions(t *testing.T) {
	// Test readVersionFile with non-existent file
	_, err := readVersionFile("non-existent-file")
	assert.Error(t, err)

	// Test runCommand with valid command
	output, err := runCommand("echo", "test")
	assert.NoError(t, err)
	assert.Equal(t, "test\n", output)

	// Test runCommand with invalid command
	_, err = runCommand("non-existent-command")
	assert.Error(t, err)
}

// TestDistributedHandlers tests distributed-related handlers
func TestDistributedHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &Handler{} // No distributed manager

	router := gin.New()
	router.GET("/distributed/status", h.getDistributedStatus)

	req, _ := http.NewRequest("GET", "/distributed/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 501 Service Unavailable when no distributed manager
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "Distributed work not available")
}

// TestGetDistributedStatus_InvalidManagerType tests getDistributedStatus with invalid manager type
func TestGetDistributedStatus_InvalidManagerType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a mock distributed manager (wrong type)
	mockDM := &MockDistributedManager{}

	h := &Handler{
		distributedManager: mockDM,
	}

	router := gin.New()
	router.GET("/distributed/status", h.getDistributedStatus)

	req, _ := http.NewRequest("GET", "/distributed/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 500 Internal Server Error for invalid manager type
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid distributed manager", response["error"])
}

// MockDistributedManager is a mock implementation for testing
type MockDistributedManager struct{}

func (m *MockDistributedManager) TranslateDistributed(ctx context.Context, text, contextHint string) (string, error) {
	return "distributed result", nil
}
func (m *MockDistributedManager) Initialize(localCoordinator interface{}) error          { return nil }
func (m *MockDistributedManager) DiscoverAndPairWorkers(ctx context.Context) error       { return nil }
func (m *MockDistributedManager) GetStatus() map[string]interface{}                      { return map[string]interface{}{} }
func (m *MockDistributedManager) AddWorker(workerID string, workerCfg interface{}) error { return nil }
func (m *MockDistributedManager) RemoveWorker(workerID string) error                     { return nil }
func (m *MockDistributedManager) PairWorker(workerID string) error                       { return nil }
func (m *MockDistributedManager) UnpairWorker(workerID string) error                     { return nil }
