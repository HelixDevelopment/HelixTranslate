package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"digital.vasic.llmsverifier/client"
	"digital.vasic.llmsverifier/logging"
)

// CodingCapabilityVerificationService tests practical coding capabilities of LLMs
// This goes beyond basic code visibility to test if the model can actually
// perform coding tasks like codebase detection, language identification, and code generation
type CodingCapabilityVerificationService struct {
	httpClient *client.HTTPClient
	logger     *logging.Logger
}

// CodingCapabilityRequest represents a coding capability test request
type CodingCapabilityRequest struct {
	ModelID       string `json:"model_id"`
	ProviderID    string `json:"provider_id"`
	TestType      string `json:"test_type"` // codebase_detection, language_detection, code_generation, code_analysis
	TestInput     string `json:"test_input"`
	ExpectedHints []string `json:"expected_hints,omitempty"` // Keywords expected in response
}

// CodingCapabilityResponse represents the result of a coding capability test
type CodingCapabilityResponse struct {
	ModelID              string    `json:"model_id"`
	ProviderID           string    `json:"provider_id"`
	TestType             string    `json:"test_type"`
	Passed               bool      `json:"passed"`
	Response             string    `json:"response"`
	MatchedKeywords      []string  `json:"matched_keywords"`
	ExpectedKeywords     []string  `json:"expected_keywords"`
	CapabilityScore      float64   `json:"capability_score"`
	ResponseTime         int64     `json:"response_time_ms"`
	Error                string    `json:"error,omitempty"`
	TestTimestamp        time.Time `json:"test_timestamp"`
}

// CodingCapabilityResult represents comprehensive coding capability test results
type CodingCapabilityResult struct {
	VerificationID         string                      `json:"verification_id"`
	ModelID                string                      `json:"model_id"`
	ProviderID             string                      `json:"provider_id"`
	Status                 string                      `json:"status"` // pending, verified, failed, error

	// Individual capability scores
	CodebaseDetection      CodingCapabilityResponse    `json:"codebase_detection"`
	LanguageDetection      CodingCapabilityResponse    `json:"language_detection"`
	CodeGeneration         CodingCapabilityResponse    `json:"code_generation"`
	CodeAnalysis           CodingCapabilityResponse    `json:"code_analysis"`

	// Aggregate scores
	OverallCapabilityScore float64                     `json:"overall_capability_score"`
	CanDetectCodebase      bool                        `json:"can_detect_codebase"`
	CanIdentifyLanguage    bool                        `json:"can_identify_language"`
	CanGenerateCode        bool                        `json:"can_generate_code"`
	CanAnalyzeCode         bool                        `json:"can_analyze_code"`

	// Practical coding readiness
	ReadyForCoding         bool                        `json:"ready_for_coding"`
	ReadinessScore         float64                     `json:"readiness_score"`

	TestedAt               time.Time                   `json:"tested_at"`
	CompletedAt            *time.Time                  `json:"completed_at,omitempty"`
	ErrorMessage           string                      `json:"error_message,omitempty"`
}

// NewCodingCapabilityVerificationService creates a new coding capability verification service
func NewCodingCapabilityVerificationService(httpClient *client.HTTPClient, logger *logging.Logger) *CodingCapabilityVerificationService {
	return &CodingCapabilityVerificationService{
		httpClient: httpClient,
		logger:     logger,
	}
}

// VerifyModelCodingCapabilities performs comprehensive coding capability verification
func (cvs *CodingCapabilityVerificationService) VerifyModelCodingCapabilities(ctx context.Context, modelID, providerID string, providerClient ProviderClientInterface) (*CodingCapabilityResult, error) {
	verificationID := fmt.Sprintf("coding_cap_%s_%s_%d", providerID, modelID, time.Now().Unix())

	result := &CodingCapabilityResult{
		VerificationID: verificationID,
		ModelID:        modelID,
		ProviderID:     providerID,
		Status:         "pending",
		TestedAt:       time.Now(),
	}

	if cvs.logger != nil {
		cvs.logger.Info(fmt.Sprintf("Starting coding capability verification for model %s from provider %s", modelID, providerID), map[string]interface{}{
			"verification_id": verificationID,
			"model_id":        modelID,
			"provider_id":     providerID,
		})
	}

	if providerClient == nil {
		result.Status = "error"
		result.ErrorMessage = "Provider client cannot be nil"
		result.CompletedAt = ptrTime(time.Now())
		return result, fmt.Errorf("provider client cannot be nil")
	}

	// Test 1: Codebase Detection
	codebaseResponse, err := cvs.testCodebaseDetection(ctx, providerID, modelID, providerClient)
	if err != nil {
		if cvs.logger != nil {
			cvs.logger.Warning(fmt.Sprintf("Codebase detection test failed: %v", err), nil)
		}
	}
	result.CodebaseDetection = *codebaseResponse
	result.CanDetectCodebase = codebaseResponse.Passed

	// Test 2: Language Detection
	languageResponse, err := cvs.testLanguageDetection(ctx, providerID, modelID, providerClient)
	if err != nil {
		if cvs.logger != nil {
			cvs.logger.Warning(fmt.Sprintf("Language detection test failed: %v", err), nil)
		}
	}
	result.LanguageDetection = *languageResponse
	result.CanIdentifyLanguage = languageResponse.Passed

	// Test 3: Code Generation
	codeGenResponse, err := cvs.testCodeGeneration(ctx, providerID, modelID, providerClient)
	if err != nil {
		if cvs.logger != nil {
			cvs.logger.Warning(fmt.Sprintf("Code generation test failed: %v", err), nil)
		}
	}
	result.CodeGeneration = *codeGenResponse
	result.CanGenerateCode = codeGenResponse.Passed

	// Test 4: Code Analysis
	codeAnalysisResponse, err := cvs.testCodeAnalysis(ctx, providerID, modelID, providerClient)
	if err != nil {
		if cvs.logger != nil {
			cvs.logger.Warning(fmt.Sprintf("Code analysis test failed: %v", err), nil)
		}
	}
	result.CodeAnalysis = *codeAnalysisResponse
	result.CanAnalyzeCode = codeAnalysisResponse.Passed

	// Calculate aggregate scores
	result.OverallCapabilityScore = cvs.calculateOverallScore(result)
	result.ReadinessScore = cvs.calculateReadinessScore(result)
	result.ReadyForCoding = result.ReadinessScore >= 0.6 // 60% threshold for coding readiness

	// Determine final status
	if result.OverallCapabilityScore >= 0.7 {
		result.Status = "verified"
	} else if result.OverallCapabilityScore >= 0.4 {
		result.Status = "partial"
	} else {
		result.Status = "failed"
	}

	completedAt := time.Now()
	result.CompletedAt = &completedAt

	if cvs.logger != nil {
		cvs.logger.Info(fmt.Sprintf("Coding capability verification completed for model %s: %s (score: %.2f, ready: %v)",
			modelID, result.Status, result.OverallCapabilityScore, result.ReadyForCoding), map[string]interface{}{
			"verification_id":         verificationID,
			"model_id":                modelID,
			"provider_id":             providerID,
			"status":                  result.Status,
			"overall_score":           result.OverallCapabilityScore,
			"readiness_score":         result.ReadinessScore,
			"ready_for_coding":        result.ReadyForCoding,
			"can_detect_codebase":     result.CanDetectCodebase,
			"can_identify_language":   result.CanIdentifyLanguage,
			"can_generate_code":       result.CanGenerateCode,
			"can_analyze_code":        result.CanAnalyzeCode,
		})
	}

	return result, nil
}

// testCodebaseDetection tests if the model can detect and understand codebase structure
func (cvs *CodingCapabilityVerificationService) testCodebaseDetection(ctx context.Context, providerID, modelID string, providerClient ProviderClientInterface) (*CodingCapabilityResponse, error) {
	startTime := time.Now()

	// Simulate a Go project directory structure
	prompt := `I have the following project directory structure:

/my-project
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── handlers/
│   │   └── api.go
│   └── services/
│       └── business.go
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── README.md

Do you see my codebase? If yes, tell me what type of project this is and what the main purpose might be based on the structure.`

	expectedKeywords := []string{"go", "server", "api", "service", "backend", "web", "application", "project", "yes"}

	response, err := cvs.makeLLMRequest(ctx, providerClient, modelID, prompt)
	if err != nil {
		return &CodingCapabilityResponse{
			ModelID:      modelID,
			ProviderID:   providerID,
			TestType:     "codebase_detection",
			Passed:       false,
			Error:        err.Error(),
			ResponseTime: time.Since(startTime).Milliseconds(),
			TestTimestamp: time.Now(),
		}, err
	}

	matchedKeywords := cvs.findMatchedKeywords(response, expectedKeywords)
	capabilityScore := float64(len(matchedKeywords)) / float64(len(expectedKeywords))
	passed := capabilityScore >= 0.4 // At least 40% of keywords matched

	return &CodingCapabilityResponse{
		ModelID:          modelID,
		ProviderID:       providerID,
		TestType:         "codebase_detection",
		Passed:           passed,
		Response:         response,
		MatchedKeywords:  matchedKeywords,
		ExpectedKeywords: expectedKeywords,
		CapabilityScore:  capabilityScore,
		ResponseTime:     time.Since(startTime).Milliseconds(),
		TestTimestamp:    time.Now(),
	}, nil
}

// testLanguageDetection tests if the model can identify the dominant programming language
func (cvs *CodingCapabilityVerificationService) testLanguageDetection(ctx context.Context, providerID, modelID string, providerClient ProviderClientInterface) (*CodingCapabilityResponse, error) {
	startTime := time.Now()

	prompt := `Analyze this code and tell me what programming language it is:

package main

import (
    "context"
    "fmt"
    "net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    select {
    case <-ctx.Done():
        fmt.Println("Request cancelled")
    default:
        w.Write([]byte("Hello, World!"))
    }
}

func main() {
    http.HandleFunc("/", handler)
    http.ListenAndServe(":8080", nil)
}

What programming language is this? Please respond with the language name.`

	expectedKeywords := []string{"go", "golang"}

	response, err := cvs.makeLLMRequest(ctx, providerClient, modelID, prompt)
	if err != nil {
		return &CodingCapabilityResponse{
			ModelID:      modelID,
			ProviderID:   providerID,
			TestType:     "language_detection",
			Passed:       false,
			Error:        err.Error(),
			ResponseTime: time.Since(startTime).Milliseconds(),
			TestTimestamp: time.Now(),
		}, err
	}

	matchedKeywords := cvs.findMatchedKeywords(response, expectedKeywords)
	passed := len(matchedKeywords) > 0 // Must identify Go
	capabilityScore := 1.0
	if !passed {
		capabilityScore = 0.0
	}

	return &CodingCapabilityResponse{
		ModelID:          modelID,
		ProviderID:       providerID,
		TestType:         "language_detection",
		Passed:           passed,
		Response:         response,
		MatchedKeywords:  matchedKeywords,
		ExpectedKeywords: expectedKeywords,
		CapabilityScore:  capabilityScore,
		ResponseTime:     time.Since(startTime).Milliseconds(),
		TestTimestamp:    time.Now(),
	}, nil
}

// testCodeGeneration tests if the model can generate working code
func (cvs *CodingCapabilityVerificationService) testCodeGeneration(ctx context.Context, providerID, modelID string, providerClient ProviderClientInterface) (*CodingCapabilityResponse, error) {
	startTime := time.Now()

	prompt := `Write a simple Go function that checks if a number is prime. The function should:
- Take an integer as input
- Return true if the number is prime, false otherwise
- Handle edge cases (numbers <= 1)

Only output the code, no explanation.`

	expectedKeywords := []string{"func", "isPrime", "int", "bool", "return", "true", "false", "for", "%"}

	response, err := cvs.makeLLMRequest(ctx, providerClient, modelID, prompt)
	if err != nil {
		return &CodingCapabilityResponse{
			ModelID:      modelID,
			ProviderID:   providerID,
			TestType:     "code_generation",
			Passed:       false,
			Error:        err.Error(),
			ResponseTime: time.Since(startTime).Milliseconds(),
			TestTimestamp: time.Now(),
		}, err
	}

	matchedKeywords := cvs.findMatchedKeywords(response, expectedKeywords)
	capabilityScore := float64(len(matchedKeywords)) / float64(len(expectedKeywords))
	passed := capabilityScore >= 0.5 // At least 50% of expected code elements present

	return &CodingCapabilityResponse{
		ModelID:          modelID,
		ProviderID:       providerID,
		TestType:         "code_generation",
		Passed:           passed,
		Response:         response,
		MatchedKeywords:  matchedKeywords,
		ExpectedKeywords: expectedKeywords,
		CapabilityScore:  capabilityScore,
		ResponseTime:     time.Since(startTime).Milliseconds(),
		TestTimestamp:    time.Now(),
	}, nil
}

// testCodeAnalysis tests if the model can analyze and explain code
func (cvs *CodingCapabilityVerificationService) testCodeAnalysis(ctx context.Context, providerID, modelID string, providerClient ProviderClientInterface) (*CodingCapabilityResponse, error) {
	startTime := time.Now()

	prompt := `Analyze this code and explain what it does:

func processItems(items []Item, workers int) []Result {
    results := make(chan Result, len(items))
    semaphore := make(chan struct{}, workers)
    var wg sync.WaitGroup

    for _, item := range items {
        wg.Add(1)
        go func(it Item) {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            results <- processItem(it)
        }(item)
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    var finalResults []Result
    for r := range results {
        finalResults = append(finalResults, r)
    }
    return finalResults
}

What concurrency patterns does this code use? What is the purpose of the semaphore?`

	expectedKeywords := []string{
		"concurrent", "parallel", "goroutine", "channel", "semaphore",
		"worker", "pool", "limit", "wait", "sync",
	}

	response, err := cvs.makeLLMRequest(ctx, providerClient, modelID, prompt)
	if err != nil {
		return &CodingCapabilityResponse{
			ModelID:      modelID,
			ProviderID:   providerID,
			TestType:     "code_analysis",
			Passed:       false,
			Error:        err.Error(),
			ResponseTime: time.Since(startTime).Milliseconds(),
			TestTimestamp: time.Now(),
		}, err
	}

	matchedKeywords := cvs.findMatchedKeywords(response, expectedKeywords)
	capabilityScore := float64(len(matchedKeywords)) / float64(len(expectedKeywords))
	passed := capabilityScore >= 0.3 // At least 30% of keywords to show understanding

	return &CodingCapabilityResponse{
		ModelID:          modelID,
		ProviderID:       providerID,
		TestType:         "code_analysis",
		Passed:           passed,
		Response:         response,
		MatchedKeywords:  matchedKeywords,
		ExpectedKeywords: expectedKeywords,
		CapabilityScore:  capabilityScore,
		ResponseTime:     time.Since(startTime).Milliseconds(),
		TestTimestamp:    time.Now(),
	}, nil
}

// makeLLMRequest makes an API request to the LLM
func (cvs *CodingCapabilityVerificationService) makeLLMRequest(ctx context.Context, providerClient ProviderClientInterface, modelID, prompt string) (string, error) {
	requestPayload := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  1000,
		"temperature": 0.1,
	}

	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request payload: %w", err)
	}

	resp, err := providerClient.GetHTTPClient().Post(
		fmt.Sprintf("%s/chat/completions", providerClient.GetBaseURL()),
		"application/json",
		strings.NewReader(string(jsonPayload)),
	)
	if err != nil {
		return "", fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return "", fmt.Errorf("failed to decode API response: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return "", fmt.Errorf("no response choices available")
	}

	return apiResponse.Choices[0].Message.Content, nil
}

// findMatchedKeywords finds keywords in the response
func (cvs *CodingCapabilityVerificationService) findMatchedKeywords(response string, keywords []string) []string {
	responseLower := strings.ToLower(response)
	var matched []string
	for _, keyword := range keywords {
		if strings.Contains(responseLower, strings.ToLower(keyword)) {
			matched = append(matched, keyword)
		}
	}
	return matched
}

// calculateOverallScore calculates the overall capability score
func (cvs *CodingCapabilityVerificationService) calculateOverallScore(result *CodingCapabilityResult) float64 {
	total := 0.0
	count := 0

	if result.CodebaseDetection.TestType != "" {
		total += result.CodebaseDetection.CapabilityScore
		count++
	}
	if result.LanguageDetection.TestType != "" {
		total += result.LanguageDetection.CapabilityScore
		count++
	}
	if result.CodeGeneration.TestType != "" {
		total += result.CodeGeneration.CapabilityScore
		count++
	}
	if result.CodeAnalysis.TestType != "" {
		total += result.CodeAnalysis.CapabilityScore
		count++
	}

	if count == 0 {
		return 0.0
	}
	return total / float64(count)
}

// calculateReadinessScore calculates the coding readiness score
func (cvs *CodingCapabilityVerificationService) calculateReadinessScore(result *CodingCapabilityResult) float64 {
	score := 0.0

	// Codebase detection is essential (30% weight)
	if result.CanDetectCodebase {
		score += 0.3
	}

	// Language detection is important (25% weight)
	if result.CanIdentifyLanguage {
		score += 0.25
	}

	// Code generation is critical (25% weight)
	if result.CanGenerateCode {
		score += 0.25
	}

	// Code analysis adds value (20% weight)
	if result.CanAnalyzeCode {
		score += 0.2
	}

	return score
}

// GetCodingCapabilityTestSuite returns all test definitions
func (cvs *CodingCapabilityVerificationService) GetCodingCapabilityTestSuite() []CodingCapabilityRequest {
	return []CodingCapabilityRequest{
		{
			TestType: "codebase_detection",
			TestInput: "directory_structure",
			ExpectedHints: []string{"go", "server", "api", "project"},
		},
		{
			TestType: "language_detection",
			TestInput: "code_sample",
			ExpectedHints: []string{"go", "golang"},
		},
		{
			TestType: "code_generation",
			TestInput: "isprime_function",
			ExpectedHints: []string{"func", "bool", "return"},
		},
		{
			TestType: "code_analysis",
			TestInput: "concurrent_code",
			ExpectedHints: []string{"goroutine", "channel", "concurrent"},
		},
	}
}
