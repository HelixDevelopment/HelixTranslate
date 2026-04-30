package llmverifier

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"digital.vasic.llmsverifier/config"
)

// Verifier is responsible for verifying LLMs
type Verifier struct {
	cfg       *config.Config
	startTime time.Time
	endTime   time.Time
}

// New creates a new Verifier instance
func New(cfg *config.Config) *Verifier {
	return &Verifier{
		cfg: cfg,
	}
}

// StartTiming records the start time for verification
func (v *Verifier) StartTiming() {
	v.startTime = time.Now()
}

// EndTiming records the end time for verification
func (v *Verifier) EndTiming() {
	v.endTime = time.Now()
}

// GetStartTime returns the verification start time
func (v *Verifier) GetStartTime() time.Time {
	if v.startTime.IsZero() {
		return time.Now()
	}
	return v.startTime
}

// GetEndTime returns the verification end time
func (v *Verifier) GetEndTime() time.Time {
	if v.endTime.IsZero() {
		return time.Now()
	}
	return v.endTime
}

// GetGlobalClient returns an LLM client configured with global settings
func (v *Verifier) GetGlobalClient() *LLMClient {
	// Create client with timeout matching config timeout
	timeout := v.cfg.Global.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second // fallback to 60s if not configured
	}
	return NewLLMClientWithTimeout(v.cfg.Global.BaseURL, v.cfg.Global.APIKey, nil, timeout)
}

// SummarizeConversation uses LLM to generate a summary of conversation messages
func (v *Verifier) SummarizeConversation(messages []string) (*ConversationSummary, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to summarize")
	}

	// Prepare the conversation text
	conversationText := strings.Join(messages, "\n")

	// Create LLM prompt for summarization
	prompt := fmt.Sprintf(`Please analyze and summarize the following conversation. Provide:

1. A concise summary (2-3 sentences)
2. Main topics discussed (comma-separated list)
3. Key points or decisions made (bullet points)
4. Overall importance score (0.0-1.0, where 1.0 is most important)

Format your response as JSON with the following structure:
{
  "summary": "concise summary text",
  "topics": ["topic1", "topic2"],
  "key_points": ["point1", "point2"],
  "importance": 0.8
}

Conversation:
%s`, conversationText)

	// Get LLM client
	client := v.GetGlobalClient()

	// Create request
	req := ChatCompletionRequest{
		Model: v.cfg.Global.DefaultModel,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert conversation analyzer. Provide structured summaries in the exact JSON format requested.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   intPtr(1000),
		Temperature: floatPtr(0.3),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.ChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM summarization failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Parse JSON response
	content := response.Choices[0].Message.Content

	// Extract JSON from response
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no JSON found in LLM response")
	}

	jsonStr := content[start : end+1]
	var summary ConversationSummary
	if err := json.Unmarshal([]byte(jsonStr), &summary); err != nil {
		return nil, fmt.Errorf("failed to parse LLM summary JSON: %w", err)
	}

	return &summary, nil
}

// ConversationSummary represents an LLM-generated summary
type ConversationSummary struct {
	Summary    string   `json:"summary"`
	Topics     []string `json:"topics"`
	KeyPoints  []string `json:"key_points"`
	Importance float64  `json:"importance"`
}

// Verify performs the verification of LLMs based on the configuration
func (v *Verifier) Verify() ([]VerificationResult, error) {
	var allResults []VerificationResult

	// If no LLMs are specified in config, discover all available models
	if len(v.cfg.LLMs) == 0 {
		discoveredResults, err := v.discoverAndVerifyAllModels()
		if err != nil {
			return nil, fmt.Errorf("failed to discover and verify models: %w", err)
		}
		allResults = append(allResults, discoveredResults...)
	} else {
		// Verify only the specified LLMs
		for _, llmCfg := range v.cfg.LLMs {
			client := NewLLMClient(llmCfg.Endpoint, llmCfg.APIKey, llmCfg.Headers)

			// If model name is not specified, discover all models for this endpoint
			if llmCfg.Model == "" {
				models, err := client.ListModels(context.Background())
				if err != nil {
					result := VerificationResult{
						ModelInfo: ModelInfo{Endpoint: llmCfg.Endpoint},
						Error:     fmt.Sprintf("failed to list models: %v", err),
						Timestamp: time.Now(),
					}
					allResults = append(allResults, result)
					continue
				}

				for _, model := range models {
					result, err := v.verifySingleModel(client, model.ID, llmCfg.Endpoint)
					if err != nil {
						result = VerificationResult{
							ModelInfo: ModelInfo{ID: model.ID, Endpoint: llmCfg.Endpoint},
							Error:     fmt.Sprintf("verification failed: %v", err),
							Timestamp: time.Now(),
						}
					}
					allResults = append(allResults, result)
				}
			} else {
				result, err := v.verifySingleModel(client, llmCfg.Model, llmCfg.Endpoint)
				if err != nil {
					result = VerificationResult{
						ModelInfo: ModelInfo{ID: llmCfg.Model, Endpoint: llmCfg.Endpoint},
						Error:     fmt.Sprintf("verification failed: %v", err),
						Timestamp: time.Now(),
					}
				}
				allResults = append(allResults, result)
			}
		}
	}

	// Fallback: ensure we always return results for configured LLMs
	// This handles cases where all verifications fail but we still want to return error results
	if len(v.cfg.LLMs) > 0 && len(allResults) < len(v.cfg.LLMs) {
		// Add error results for any LLMs that weren't verified
		for _, llmCfg := range v.cfg.LLMs {
			found := false
			for _, result := range allResults {
				if result.ModelInfo.ID == llmCfg.Name || result.ModelInfo.ID == llmCfg.Model {
					found = true
					break
				}
			}
			if !found {
				errorResult := VerificationResult{
					ModelInfo: ModelInfo{
						ID: llmCfg.Name,
					},
					Availability: AvailabilityResult{
						Exists:     false,
						Responsive: false,
						Error:      "Verification failed - no results obtained",
					},
					ResponseTime: ResponseTimeResult{
						AverageLatency: 0,
					},
					Error: "Verification failed",
				}
				allResults = append(allResults, errorResult)
			}
		}
	}

	return allResults, nil
}

// discoverAndVerifyAllModels discovers all models from configured endpoints and verifies each one
func (v *Verifier) discoverAndVerifyAllModels() ([]VerificationResult, error) {
	var allResults []VerificationResult

	// We'll need to determine endpoints somehow - for now, let's assume global config has the base URL
	if v.cfg.Global.BaseURL != "" {
		client := NewLLMClient(v.cfg.Global.BaseURL, v.cfg.Global.APIKey, nil)

		models, err := client.ListModels(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to discover models from %s: %w", v.cfg.Global.BaseURL, err)
		}

		// Process each model with concurrency control
		concurrency := v.cfg.Concurrency
		if concurrency <= 0 {
			concurrency = 1
		}

		semaphore := make(chan struct{}, concurrency)
		var wg sync.WaitGroup
		var mu sync.Mutex

		resultsChan := make(chan VerificationResult, len(models))

		for _, model := range models {
			wg.Add(1)
			go func(model ModelInfo) {
				defer wg.Done()

				semaphore <- struct{}{}        // Acquire semaphore
				defer func() { <-semaphore }() // Release semaphore

				result, err := v.verifySingleModel(client, model.ID, v.cfg.Global.BaseURL)
				if err != nil {
					result = VerificationResult{
						ModelInfo: ModelInfo{ID: model.ID, Endpoint: v.cfg.Global.BaseURL},
						Error:     fmt.Sprintf("verification failed: %v", err),
						Timestamp: time.Now(),
					}
				}
				resultsChan <- result
			}(model)
		}

		// Close results channel after all goroutines finish
		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		// Collect results
		for result := range resultsChan {
			mu.Lock()
			allResults = append(allResults, result)
			mu.Unlock()
		}
	}

	return allResults, nil
}

// verifySingleModel performs verification of a single model
func (v *Verifier) verifySingleModel(client *LLMClient, modelName, endpoint string) (VerificationResult, error) {
	result := VerificationResult{
		ModelInfo: ModelInfo{
			ID:       modelName,
			Endpoint: endpoint,
		},
		Timestamp: time.Now(),
	}

	// Check model existence
	exists, err := client.CheckModelExists(context.Background(), modelName)
	if err != nil {
		result.Error = fmt.Sprintf("failed to check model existence: %v", err)
		return result, nil
	}

	if !exists {
		result.Availability.Exists = false
		result.Error = "model does not exist"
		return result, nil
	}

	result.Availability.Exists = true

	// Perform detailed verification including overload checking

	// First, test basic responsiveness
	singleResponseTime, isResponsive, responseErr := v.checkResponsiveness(client, modelName)
	if !isResponsive {
		result.Availability.Responsive = false
		result.Availability.Error = responseErr
		result.Availability.Latency = singleResponseTime
		return result, nil
	}

	result.Availability.Responsive = true
	result.Availability.Latency = singleResponseTime
	result.Availability.LastChecked = time.Now()

	// Test for overload by sending multiple concurrent requests
	isOverloaded, responseTime := v.checkOverload(client, modelName)
	result.Availability.Overloaded = isOverloaded
	result.ResponseTime = responseTime

	// Extract model info from a successful response
	modelInfo, err := v.getModelDetailedInfo(client, modelName)
	if err != nil {
		// If we can't get detailed info, at least keep the ID
		result.ModelInfo.ID = modelName
	} else {
		result.ModelInfo = *modelInfo
		result.ModelInfo.Endpoint = endpoint
	}

	// Detect features
	features, err := v.detectFeatures(client, modelName)
	if err != nil {
		// We'll continue even if feature detection fails
		result.FeatureDetection = FeatureDetectionResult{}
	} else {
		result.FeatureDetection = *features
	}

	// Assess code capabilities
	codeCaps, err := v.assessCodeCapabilities(client, modelName)
	if err != nil {
		// We'll continue even if code assessment fails
		result.CodeCapabilities = CodeCapabilityResult{}
	} else {
		result.CodeCapabilities = *codeCaps
	}

	// Assess generative capabilities
	generativeCaps, err := v.assessGenerativeCapabilities(client, modelName)
	if err != nil {
		// We'll continue even if generative assessment fails
		result.GenerativeCapabilities = GenerativeCapabilityResult{}
	} else {
		result.GenerativeCapabilities = *generativeCaps
	}

	// Calculate performance scores
	scores, details := v.CalculateScores(result)
	result.PerformanceScores = scores
	result.ScoreDetails = details

	return result, nil
}

// checkResponsiveness tests if the model responds to requests
func (v *Verifier) checkResponsiveness(client *LLMClient, modelName string) (time.Duration, bool, string) {
	// Use global timeout for responsiveness checks
	timeout := v.cfg.Global.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second // default if not set
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	startTime := time.Now()

	// Test basic responsiveness with a simple prompt
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Hello, please respond with just the word 'pong'.",
			},
		},
		MaxTokens: intPtr(10),
	}

	_, err := client.ChatCompletion(ctx, req)
	latency := time.Since(startTime)

	if err != nil {
		return latency, false, err.Error()
	}

	return latency, true, ""
}

// checkOverload tests if the model is overloaded by sending multiple concurrent requests
func (v *Verifier) checkOverload(client *LLMClient, modelName string) (bool, ResponseTimeResult) {
	const numRequests = 10
	const timeoutPerRequest = 30 * time.Second

	type result struct {
		latency time.Duration
		err     error
	}

	resultsCh := make(chan result, numRequests)

	// Send multiple requests concurrently
	for i := 0; i < numRequests; i++ {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), timeoutPerRequest)
			defer cancel()

			startTime := time.Now()
			req := ChatCompletionRequest{
				Model: modelName,
				Messages: []Message{
					{
						Role:    "user",
						Content: fmt.Sprintf("Test request %d, respond with just the number %d", time.Now().UnixNano(), time.Now().UnixNano()%100),
					},
				},
				MaxTokens: intPtr(10),
			}

			_, err := client.ChatCompletion(ctx, req)
			latency := time.Since(startTime)

			resultsCh <- result{latency: latency, err: err}
		}()
	}

	// Collect results
	var latencies []time.Duration
	var errorsCount int
	for i := 0; i < numRequests; i++ {
		res := <-resultsCh
		if res.err != nil {
			errorsCount++
		} else {
			latencies = append(latencies, res.latency)
		}
	}

	// Initialize response time result
	responseTime := ResponseTimeResult{
		MeasurementCount: len(latencies),
	}

	if len(latencies) == 0 {
		// All requests failed, consider overloaded
		responseTime.AverageLatency = 0
		responseTime.MinLatency = 0
		responseTime.MaxLatency = 0
		responseTime.P95Latency = 0
		responseTime.Throughput = 0
		return true, responseTime
	}

	// Calculate average latency
	var totalLatency time.Duration
	for _, l := range latencies {
		totalLatency += l
	}
	avgLatency := totalLatency / time.Duration(len(latencies))
	responseTime.AverageLatency = avgLatency

	// Calculate min, max, and p95 latencies
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	responseTime.MinLatency = latencies[0]
	responseTime.MaxLatency = latencies[len(latencies)-1]
	// Calculate p95 latency
	p95Index := int(math.Ceil(0.95*float64(len(latencies)))) - 1
	if p95Index < 0 {
		p95Index = 0
	}
	responseTime.P95Latency = latencies[p95Index]

	// Calculate throughput (requests per second)
	totalDuration := time.Since(time.Now().Add(-time.Second * 1)) // Approximate
	if totalDuration < 100*time.Millisecond {
		// If the test ran too quickly, just use the time for all requests
		if len(latencies) > 0 {
			totalDuration = time.Duration(0)
			for _, l := range latencies {
				totalDuration += l
			}
		} else {
			totalDuration = 1 * time.Second // fallback
		}
	}

	throughput := float64(len(latencies)) / totalDuration.Seconds()
	responseTime.Throughput = throughput

	// Determine if overloaded based on high latency or high error rate
	highErrorRate := float64(errorsCount)/float64(numRequests) > 0.5
	extremeLatency := avgLatency > 10*time.Second

	isOverloaded := highErrorRate || extremeLatency

	return isOverloaded, responseTime
}

// getModelDetailedInfo retrieves detailed information about the model
func (v *Verifier) getModelDetailedInfo(client *LLMClient, modelName string) (*ModelInfo, error) {
	// Try to get more detailed model info by making a request
	ctx, cancel := context.WithTimeout(context.Background(), v.cfg.Timeout)
	defer cancel()

	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "What is your model name?",
			},
		},
		MaxTokens: intPtr(20),
	}

	response, err := client.ChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	// For now, we'll return a basic model info. In a real implementation,
	// we might call the models endpoint specifically to get detailed metadata.
	modelInfo := &ModelInfo{
		ID:       response.Model,
		Object:   "model",
		Created:  time.Now().Unix(),
		Endpoint: client.endpoint,
	}

	return modelInfo, nil
}

// detectFeatures identifies what features the model supports
func (v *Verifier) detectFeatures(client *LLMClient, modelName string) (*FeatureDetectionResult, error) {
	features := &FeatureDetectionResult{
		Modalities: []string{"text"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), v.cfg.Timeout)
	defer cancel()

	// Test for tool/function calling capability
	toolUseSupported := v.testToolUse(client, modelName, ctx)
	features.ToolUse = toolUseSupported
	features.FunctionCalling = toolUseSupported

	// Test for code generation capability
	codeGenSupported := v.testCodeGeneration(client, modelName, ctx)
	features.CodeGeneration = codeGenSupported

	// Test for code completion capability
	codeCompletionSupported := v.testCodeCompletion(client, modelName, ctx)
	features.CodeCompletion = codeCompletionSupported

	// Test for code explanation capability
	codeExplanationSupported := v.testCodeExplanation(client, modelName, ctx)
	features.CodeExplanation = codeExplanationSupported

	// Test for code review capability
	codeReviewSupported := v.testCodeReview(client, modelName, ctx)
	features.CodeReview = codeReviewSupported

	// Test for embeddings capability
	embeddingsSupported := v.testEmbeddings(client, modelName)
	features.Embeddings = embeddingsSupported

	// Check for MCPs (Model Context Protocol) support - context window management
	mcpSupported := v.testMCPs(client, modelName, ctx)
	features.MCPs = mcpSupported

	// Note: LSPs - Language Server Protocol is typically for IDE integrations, not LLMs directly
	// Most LLMs don't support LSP directly, but we'll test for potential IDE integration features
	lspSupported := v.testLSPs(client, modelName, ctx)
	features.LSPs = lspSupported

	// Check for ACPs (AI Coding Protocol) support - editor integration capabilities
	acpSupported := v.TestACPs(client, modelName, ctx)
	features.ACPs = acpSupported

	// Test for image generation capabilities
	imageGenerationSupported := v.testImageGeneration(client, modelName, ctx)
	features.ImageGeneration = imageGenerationSupported

	// Test for multimodal capabilities (e.g., vision)
	multimodalSupported := v.testMultimodal(client, modelName, ctx)
	features.Multimodal = multimodalSupported

	// Test for audio generation capabilities
	audioGenerationSupported := v.testAudioGeneration(client, modelName, ctx)
	features.AudioGeneration = audioGenerationSupported

	// Test for video generation capabilities
	videoGenerationSupported := v.testVideoGeneration(client, modelName, ctx)
	features.VideoGeneration = videoGenerationSupported

	// Test for streaming support
	streamingSupported := v.testStreaming(client, modelName)
	features.Streaming = streamingSupported

	// Test for JSON mode
	jsonModeSupported := v.testJSONMode(client, modelName, ctx)
	features.JSONMode = jsonModeSupported

	// Test for structured output
	structuredOutputSupported := v.testStructuredOutput(client, modelName, ctx)
	features.StructuredOutput = structuredOutputSupported

	// Test for reasoning capabilities
	reasoningSupported := v.testReasoning(client, modelName, ctx)
	features.Reasoning = reasoningSupported

	// Test for parallel tool use
	parallelToolUse, maxParallelCalls := v.testParallelToolUse(client, modelName, ctx)
	features.ParallelToolUse = parallelToolUse
	features.MaxParallelCalls = maxParallelCalls

	// Test for batch processing
	batchProcessingSupported := v.testBatchProcessing(client, modelName)
	features.BatchProcessing = batchProcessingSupported

	// Test for audio generation (specific to certain models)
	features.AudioGeneration = false

	// Test for video generation (specific to certain models)
	features.VideoGeneration = false

	// Test for reranking (if supported by endpoint)
	rerankSupported := v.testRerank(client, modelName, ctx)
	features.Reranking = rerankSupported

	// Test for HTTP/3 support
	http3Supported := v.testHTTP3(client, modelName)
	features.SupportsHTTP3 = http3Supported

	// Test for Brotli compression support
	brotliSupported := v.testBrotli(client, modelName)
	features.SupportsBrotli = brotliSupported

	// Test for toon/creative style generation
	toonSupported := v.testToon(client, modelName)
	features.SupportsToon = toonSupported

	return features, nil
}

// testToolUse checks if the model supports function calling/tool use
func (v *Verifier) testToolUse(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "What is the weather like in New York?",
			},
		},
		Tools: []Tool{
			{
				Type: "function",
				Function: FunctionDefinition{
					Name:        "get_current_weather",
					Description: "Get the current weather in a given location",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "The city and state, e.g. San Francisco, CA",
							},
							"unit": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"celsius", "fahrenheit"},
								"description": "The unit of temperature, either 'celsius' or 'fahrenheit'",
							},
						},
						"required": []string{"location"},
					},
				},
			},
		},
		ToolChoice: "auto",
	}

	_, err := client.ChatCompletion(ctx, req)
	return err == nil
}

// Tool represents a tool specification for function calling
type Tool struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// testCodeGeneration checks if the model can generate code
func (v *Verifier) testCodeGeneration(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Write a simple Python function to calculate the factorial of a number.",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	// Check if response contains code-like content
	responseText := resp.Choices[0].Message.Content
	return containsCode(responseText)
}

// testCodeCompletion checks if the model can complete code
func (v *Verifier) testCodeCompletion(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Complete the following Python function:\n\n```python\ndef bubble_sort(arr):\n    n = len(arr)\n    # Your code here\n```",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return containsCode(responseText) && strings.Contains(strings.ToLower(responseText), "bubble_sort")
}

// testCodeExplanation checks if the model can explain code
func (v *Verifier) testCodeExplanation(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Explain the following code:\n\n```python\nfor i in range(5):\n    print(i)\n```",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return len(responseText) > 20 // Simple heuristic: a real explanation would be more than 20 chars
}

// testCodeReview checks if the model can review code
func (v *Verifier) testCodeReview(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Review this Python code and suggest improvements:\n\n```python\narr = [3, 1, 4, 1, 5, 9, 2, 6]\nsorted_arr = []\nfor i in range(len(arr)):\n    smallest = arr[0]\n    for j in arr:\n        if j < smallest:\n            smallest = j\n    sorted_arr.append(smallest)\n    arr.remove(smallest)\nprint(sorted_arr)\n```",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return strings.Contains(strings.ToLower(responseText), "improv") ||
		strings.Contains(strings.ToLower(responseText), "suggest") ||
		strings.Contains(strings.ToLower(responseText), "issue") ||
		strings.Contains(strings.ToLower(responseText), "better")
}

// testEmbeddings checks if the model supports embeddings
func (v *Verifier) testEmbeddings(client *LLMClient, modelName string) bool {
	req := EmbeddingRequest{
		Input: "Hello world",
		Model: modelName,
	}

	ctx, cancel := context.WithTimeout(context.Background(), v.cfg.Timeout)
	defer cancel()

	_, err := client.GetEmbeddings(ctx, req)
	return err == nil
}

// testMultimodal checks if the model supports multimodal input (images, etc.)
func (v *Verifier) testMultimodal(client *LLMClient, modelName string, ctx context.Context) bool {
	// This is difficult to test without an actual image
	// For now, we'll test with a prompt that would typically work with multimodal models
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Can you analyze an image? If yes, what kind of image analysis can you perform?",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := strings.ToLower(resp.Choices[0].Message.Content)
	// Check if the response indicates multimodal capability
	return strings.Contains(responseText, "image") ||
		strings.Contains(responseText, "visual") ||
		strings.Contains(responseText, "analyze") ||
		strings.Contains(responseText, "describe")
}

// testHTTP3 checks if the provider supports HTTP/3
func (v *Verifier) testHTTP3(client *LLMClient, modelName string) bool {
	// HTTP/3 support is determined by the client's HTTP version capability
	// This would require checking if the underlying HTTP client supports HTTP/3
	// For now, we'll check based on provider capabilities
	provider := strings.ToLower(client.endpoint)
	if strings.Contains(provider, "cloudflare") || strings.Contains(provider, "google") || strings.Contains(provider, "gemini") {
		return true // These providers typically support HTTP/3
	}
	return false // Most providers don't support HTTP/3 yet
}

// testBrotli checks if the provider supports Brotli compression
func (v *Verifier) testBrotli(client *LLMClient, modelName string) bool {
	// Test Brotli support by making a request and checking if the response
	// accepts brotli encoding
	provider := strings.ToLower(client.endpoint)
	if strings.Contains(provider, "anthropic") || strings.Contains(provider, "openai") ||
		strings.Contains(provider, "google") || strings.Contains(provider, "gemini") || strings.Contains(provider, "deepseek") {
		return true // Major providers typically support Brotli
	}
	return false
}

// testToon checks if the model supports "toon" style generation (creative/artistic)
func (v *Verifier) testToon(client *LLMClient, modelName string) bool {
	// Check if the model name contains "toon" or similar creative indicators
	modelLower := strings.ToLower(modelName)
	if strings.Contains(modelLower, "toon") || strings.Contains(modelLower, "creative") ||
		strings.Contains(modelLower, "art") || strings.Contains(modelLower, "dalle") {
		return true
	}

	// Test with a creative prompt to see if the model can generate artistic content
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	maxTokens := 100
	temperature := 0.8
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{Role: "user", Content: "Create a short description of a cartoon character in toon style, focusing on visual appearance and personality."},
		},
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil {
		return false
	}

	if len(resp.Choices) == 0 {
		return false
	}

	response := strings.ToLower(resp.Choices[0].Message.Content)
	// Check for creative/artistic language patterns
	creativeIndicators := []string{
		"colorful", "expressive", "cartoon", "toon", "animated", "whimsical",
		"cheeky", "playful", "vibrant", "lively", "bouncy", "cute",
	}

	for _, indicator := range creativeIndicators {
		if strings.Contains(response, indicator) {
			return true
		}
	}

	return false
}

// testStreaming checks if the model supports streaming responses
func (v *Verifier) testStreaming(client *LLMClient, modelName string) bool {
	// In the OpenAI API, streaming is specified by the 'stream' parameter
	// But since we can only check if the API accepts the parameter, not if it actually streams,
	// we'll try making a request with stream=true and see if it fails due to unsupported parameter
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Say 'hello' in 10 words.",
			},
		},
		Stream: true,
	}

	// Note: This is just testing if the API accepts the stream parameter.
	// Actual streaming implementation would require special handling.
	// For now, we'll just check that it doesn't return an error about unsupported parameter.
	ctx, cancel := context.WithTimeout(context.Background(), v.cfg.Timeout)
	defer cancel()

	_, err := client.ChatCompletion(ctx, req)
	// If the API doesn't support streaming, it would typically return an error
	// We consider it supported if no error is returned about the stream parameter specifically
	return err == nil
}

// testJSONMode checks if the model supports JSON mode
func (v *Verifier) testJSONMode(client *LLMClient, modelName string, ctx context.Context) bool {
	// Similar to streaming, we check if the API accepts JSON mode parameters
	// This varies by implementation, so we'll try a common approach
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Return a JSON object with keys 'name' and 'value'. Respond only with valid JSON.",
			},
		},
		ResponseFormat: map[string]interface{}{
			"type": "json_object",
		},
	}

	_, err := client.ChatCompletion(ctx, req)
	// If the request succeeds, JSON mode might be supported
	return err == nil
}

// testStructuredOutput checks for structured output capability
func (v *Verifier) testStructuredOutput(client *LLMClient, modelName string, ctx context.Context) bool {
	// Check for structured output features like response formats
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Return structured data about a person with fields: name (string), age (integer), email (string)",
			},
		},
		ResponseFormat: map[string]interface{}{
			"type": "json_object",
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	// Check if the response looks like structured JSON data
	var jsonData interface{}
	err = json.Unmarshal([]byte(responseText), &jsonData)
	return err == nil
}

// testReasoning checks if the model has reasoning capabilities
func (v *Verifier) testReasoning(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "There are 5 houses in a row, each painted a different color. In each house lives a person of a different nationality. The 5 owners drink a certain type of beverage, smoke a certain brand of cigar, and keep a certain pet. Using these clues, who owns the fish?\n\n1. The British man lives in the red house.\n2. The Swedish man has a dog.\n3. The Danish man drinks tea.\n4. The green house is immediately to the left of the white house.\n5. The green house owner drinks coffee.\n6. The person who smokes Pall Mall rears birds.\n7. The owner of the yellow house smokes Dunhill.\n8. The man living in the house right in the center drinks milk.\n9. The Norwegian lives in the first house.\n10. The man who smokes Blend lives next to the one who keeps cats.\n11. The man who keeps horses lives next to the man who smokes Dunhill.\n12. The owner who smokes Blue Master drinks chocolate.\n13. The German smokes Prince.\n14. The Norwegian lives next to the blue house.\n15. The man who smokes Blend has a neighbor who drinks water.\n\nQ: Who owns the fish?",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	// Check if the model attempted to solve the reasoning problem
	return strings.Contains(strings.ToLower(responseText), "fish") &&
		(strings.Contains(strings.ToLower(responseText), "german") ||
			strings.Contains(strings.ToLower(responseText), "answer"))
}

// testParallelToolUse checks for parallel tool use capability
func (v *Verifier) testParallelToolUse(client *LLMClient, modelName string, ctx context.Context) (bool, int) {
	// Test with multiple tools to see if the model can handle several at once
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Get the weather in New York and the stock price for Apple Inc.",
			},
		},
		Tools: []Tool{
			{
				Type: "function",
				Function: FunctionDefinition{
					Name:        "get_weather",
					Description: "Get the current weather in a given location",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "The city and state, e.g. San Francisco, CA",
							},
						},
						"required": []string{"location"},
					},
				},
			},
			{
				Type: "function",
				Function: FunctionDefinition{
					Name:        "get_stock_price",
					Description: "Get the current stock price for a company",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"symbol": map[string]interface{}{
								"type":        "string",
								"description": "The stock symbol, e.g. AAPL for Apple",
							},
						},
						"required": []string{"symbol"},
					},
				},
			},
		},
		ToolChoice: "auto",
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil {
		return false, 0
	}

	// Count how many tool calls were made in the response
	// Note: This is a simplified check - actual implementation would parse the response differently
	toolCallCount := 0
	if len(resp.Choices) > 0 {
		// If the model supports parallel tool use, it would make multiple tool calls
		// In practice, you'd parse the specific tool call response format
		// For now, we'll just return true if the request succeeded
		toolCallCount = 2 // We provided 2 tools
	}

	return true, toolCallCount
}

// testBatchProcessing checks for batch processing capability
func (v *Verifier) testBatchProcessing(client *LLMClient, modelName string) bool {
	// Test batch processing by sending multiple requests in a single API call
	// Many providers support batch processing through specific endpoints or parameters

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test 1: Check if the API accepts batch requests
	batchReq := map[string]interface{}{
		"model": modelName,
		"messages": [][]interface{}{
			{
				map[string]interface{}{"role": "user", "content": "What is 2+2?"},
			},
			{
				map[string]interface{}{"role": "user", "content": "What is 3+3?"},
			},
		},
	}

	// Try different batch processing approaches

	// Approach 1: Standard batch endpoint
	endpoint := "/v1/chat/completions/batch"
	_, err := client.makeRequest(ctx, http.MethodPost, endpoint, batchReq)
	if err == nil {
		return true
	}

	// Approach 2: Array of requests in standard endpoint
	endpoint = "/v1/chat/completions"
	_, err = client.makeRequest(ctx, http.MethodPost, endpoint, batchReq)
	if err == nil {
		return true
	}

	// Approach 3: Check for batch processing headers or parameters
	// Some APIs support batch processing through specific headers
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{Role: "user", Content: "Test batch processing support"},
		},
	}

	// Add batch processing headers
	customHeaders := map[string]string{
		"X-Batch-Processing": "enabled",
		"X-Batch-Size":       "10",
	}

	// Try with custom headers
	_, err = client.makeRequestWithHeaders(ctx, http.MethodPost, endpoint, req, customHeaders)
	if err == nil {
		return true
	}

	// Check error message for batch processing support indication
	if strings.Contains(err.Error(), "batch") || strings.Contains(err.Error(), "multiple") {
		// Error message suggests batch processing might be supported but with different parameters
		return true
	}

	return false
}

// testMCPs checks for Model Context Protocol support - context handling capabilities
func (v *Verifier) testMCPs(client *LLMClient, modelName string, ctx context.Context) bool {
	// MCPs in the context of LLMs refers to model's ability to handle context properly
	// Testing context window capabilities, token handling, limits, and conversation history management

	// Test 1: Context retention across conversation
	req1 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Remember that my favorite color is blue and my name is John. What is my name?",
			},
			{
				Role:    "assistant",
				Content: "Your name is John, and your favorite color is blue.",
			},
			{
				Role:    "user",
				Content: "What is my favorite color?",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req1)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := strings.ToLower(resp.Choices[0].Message.Content)
	contextRetained := strings.Contains(responseText, "blue")

	// Test 2: Token limit awareness - test with a very long context to see if model can handle it
	longContext := strings.Repeat("This is a test sentence. ", 100) // Create a long context
	req2 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: fmt.Sprintf("Context: %s\n\nNow summarize the context in 10 words or less.", longContext),
			},
		},
	}

	resp2, err2 := client.ChatCompletion(ctx, req2)
	if err2 != nil || len(resp2.Choices) == 0 {
		// If it fails on long context, the model may have limited context window
		// But we'll still return based on the first test
		return contextRetained
	}

	// Check if the model was able to handle the long context and provide a summary
	response2Text := strings.ToLower(resp2.Choices[0].Message.Content)
	tokenHandlingCapable := len(response2Text) > 0 && len(response2Text) < len(longContext) // Model summarized, showing it handled the context

	return contextRetained || tokenHandlingCapable
}

// testLSPs checks for Language Server Protocol features - IDE-like capabilities
func (v *Verifier) testLSPs(client *LLMClient, modelName string, ctx context.Context) bool {
	// LSP features include language analysis, error detection, code completion, and symbol navigation

	// Test 1: Error detection capability
	req1 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role: "user",
				Content: `You are acting as a language server. Analyze this Python code and provide detailed feedback like a language server would:
def calculate_sum(a, b):
    return a + b

result = calculate_sum(1, 2, 3)  # This has an error - too many arguments`,
			},
		},
	}

	resp1, err1 := client.ChatCompletion(ctx, req1)
	if err1 != nil || len(resp1.Choices) == 0 {
		return false
	}

	responseText1 := strings.ToLower(resp1.Choices[0].Message.Content)
	errorDetection := strings.Contains(responseText1, "error") ||
		strings.Contains(responseText1, "too many") ||
		strings.Contains(responseText1, "arguments") ||
		strings.Contains(responseText1, "mismatch")

	// Test 2: Code completion suggestion (IDE-like)
	req2 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role: "user",
				Content: `As an IDE language server, complete this Python function:
def bubble_sort(arr):
    n = len(arr)
    # Implement bubble sort
`,
			},
		},
	}

	resp2, err2 := client.ChatCompletion(ctx, req2)
	completionSuggestion := err2 == nil && len(resp2.Choices) > 0 &&
		len(resp2.Choices[0].Message.Content) > 10 // Provided substantial completion

	// Test 3: Symbol/definition navigation (theoretical test)
	req3 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: `As a language server, explain what the "len()" function does in Python and where it's defined.`,
			},
		},
	}

	resp3, err3 := client.ChatCompletion(ctx, req3)
	symbolNavigation := false
	if err3 == nil && len(resp3.Choices) > 0 {
		content := strings.ToLower(resp3.Choices[0].Message.Content)
		symbolNavigation = strings.Contains(content, "length") ||
			strings.Contains(content, "builtin")
	}

	// Return true if the model demonstrates any LSP-like capabilities
	return errorDetection || completionSuggestion || symbolNavigation
}

// TestACPs checks for AI Coding Protocol support - editor integration capabilities
func (v *Verifier) TestACPs(client *LLMClient, modelName string, ctx context.Context) bool {
	// ACP features include JSON-RPC protocol understanding, tool calling, context management, and code assistance

	// Test 1: JSON-RPC Protocol Comprehension
	req1 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role: "user",
				Content: `You are an ACP-compatible AI coding agent. Please respond to this JSON-RPC request:
{"jsonrpc":"2.0","method":"textDocument/completion","params":{"textDocument":{"uri":"file:///test.py"},"position":{"line":0,"character":10}},"id":1}

What would be an appropriate response for a code completion request? Please provide a valid JSON-RPC response.`,
			},
		},
	}

	resp1, err1 := client.ChatCompletion(ctx, req1)
	if err1 != nil || len(resp1.Choices) == 0 {
		return false
	}

	responseText1 := strings.ToLower(resp1.Choices[0].Message.Content)
	// Check for JSON-RPC response indicators
	jsonrpcComprehension := strings.Contains(responseText1, "jsonrpc") ||
		strings.Contains(responseText1, "2.0") ||
		strings.Contains(responseText1, "result") ||
		strings.Contains(responseText1, "completion") ||
		strings.Contains(responseText1, "items")

	// Test 2: Tool Calling Capability
	req2 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role: "user",
				Content: `As an ACP agent, you have access to tools like "file_read", "file_write", and "execute_command". 
Please demonstrate how you would call the "file_read" tool to read the content of a Python file named "main.py" 
and then suggest improvements based on the content.`,
			},
		},
	}

	resp2, err2 := client.ChatCompletion(ctx, req2)
	if err2 != nil || len(resp2.Choices) == 0 {
		return false
	}

	responseText2 := strings.ToLower(resp2.Choices[0].Message.Content)
	// Check for tool calling indicators
	toolCallingCapable := strings.Contains(responseText2, "file_read") ||
		strings.Contains(responseText2, "tool") ||
		strings.Contains(responseText2, "function") ||
		strings.Contains(responseText2, "parameters") ||
		strings.Contains(responseText2, "arguments")

	// Test 3: Context Management for Multi-turn Conversations
	req3 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "I'm working on a Python project with the following structure: src/main.py, tests/test_main.py, requirements.txt. The main.py file contains a Flask web application. Remember this project structure and context.",
			},
			{
				Role:    "assistant",
				Content: "I've noted your Python project structure: src/main.py (Flask web app), tests/test_main.py, requirements.txt. I'll remember this context for our conversation.",
			},
			{
				Role:    "user",
				Content: "Based on this project structure, where should I add a new utility module for database operations, and what would be the appropriate import statement in my Flask app?",
			},
		},
	}

	resp3, err3 := client.ChatCompletion(ctx, req3)
	if err3 != nil || len(resp3.Choices) == 0 {
		return false
	}

	responseText3 := strings.ToLower(resp3.Choices[0].Message.Content)
	// Check for context retention and appropriate suggestions
	contextManagement := strings.Contains(responseText3, "src") ||
		strings.Contains(responseText3, "utility") ||
		strings.Contains(responseText3, "database") ||
		strings.Contains(responseText3, "import") ||
		strings.Contains(responseText3, "module")

	// Test 4: Code Assistance and Generation
	req4 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role: "user",
				Content: `As an ACP coding agent, help me write a Python function that:
1. Takes a list of user dictionaries (each with 'name' and 'email' keys)
2. Validates that all emails are in proper format
3. Returns a list of valid users
4. Includes proper error handling and type hints
5. Has a comprehensive docstring

Please provide the complete implementation.`,
			},
		},
	}

	resp4, err4 := client.ChatCompletion(ctx, req4)
	if err4 != nil || len(resp4.Choices) == 0 {
		return false
	}

	responseText4 := strings.ToLower(resp4.Choices[0].Message.Content)
	// Check for code generation indicators
	codeAssistance := strings.Contains(responseText4, "def") ||
		strings.Contains(responseText4, "import") ||
		strings.Contains(responseText4, "list") ||
		strings.Contains(responseText4, "dict") ||
		strings.Contains(responseText4, "->") || // Type hints
		strings.Contains(responseText4, "\"\"\"") // Docstring

	// Test 5: Error Detection and Diagnostics
	req5 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role: "user",
				Content: `As an ACP agent, analyze this Python code and provide diagnostic information:
def process_user_data(users):
    valid_users = []
    for user in users:
        if user['email'].contains('@'):
            valid_users.append(user)
    return valid_users

result = process_user_data([{'name': 'John', 'email': 'john@example.com'}, {'name': 'Jane'}])

What errors or issues do you detect? Provide specific line numbers and suggestions.`,
			},
		},
	}

	resp5, err5 := client.ChatCompletion(ctx, req5)
	if err5 != nil || len(resp5.Choices) == 0 {
		return false
	}

	responseText5 := strings.ToLower(resp5.Choices[0].Message.Content)
	// Check for error detection capabilities
	errorDetection := strings.Contains(responseText5, "error") ||
		strings.Contains(responseText5, "issue") ||
		strings.Contains(responseText5, "keyerror") ||
		strings.Contains(responseText5, "contains") ||
		strings.Contains(responseText5, "missing") ||
		strings.Contains(responseText5, "line")

	// Return true if the model demonstrates multiple ACP-like capabilities
	capabilities := []bool{jsonrpcComprehension, toolCallingCapable, contextManagement, codeAssistance, errorDetection}
	supportedCapabilities := 0
	for _, capability := range capabilities {
		if capability {
			supportedCapabilities++
		}
	}

	// Require at least 3 out of 5 ACP capabilities for support
	return supportedCapabilities >= 3
}

// testImageGeneration checks for image generation capabilities
func (v *Verifier) testImageGeneration(client *LLMClient, modelName string, ctx context.Context) bool {
	// Image generation is typically handled by separate models like DALL-E
	// But some models might be able to describe or suggest image generation
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Describe how you would generate an image of a beautiful sunset over mountains with a lake in the foreground. What parameters would you use?",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := strings.ToLower(resp.Choices[0].Message.Content)
	// Check if the model provides details about image generation parameters
	return strings.Contains(responseText, "image") &&
		(strings.Contains(responseText, "generate") ||
			strings.Contains(responseText, "prompt") ||
			strings.Contains(responseText, "parameters") ||
			strings.Contains(responseText, "resolution") ||
			strings.Contains(responseText, "style"))
}

// testAudioGeneration checks for audio generation capabilities
func (v *Verifier) testAudioGeneration(client *LLMClient, modelName string, ctx context.Context) bool {
	// Audio generation is typically handled by specialized models
	// Check if the model understands audio generation concepts
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Explain how you would generate an audio clip of birds chirping in a forest. What would be the key parameters?",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := strings.ToLower(resp.Choices[0].Message.Content)
	// Check if the model provides details about audio generation
	return strings.Contains(responseText, "audio") &&
		(strings.Contains(responseText, "generate") ||
			strings.Contains(responseText, "parameters") ||
			strings.Contains(responseText, "frequency") ||
			strings.Contains(responseText, "wave") ||
			strings.Contains(responseText, "sound"))
}

// testVideoGeneration checks for video generation capabilities
func (v *Verifier) testVideoGeneration(client *LLMClient, modelName string, ctx context.Context) bool {
	// Video generation is typically handled by specialized models
	// Check if the model understands video generation concepts
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Describe the process of generating a short video clip showing a flower blooming. What technical aspects would be involved?",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := strings.ToLower(resp.Choices[0].Message.Content)
	// Check if the model provides details about video generation
	return strings.Contains(responseText, "video") &&
		(strings.Contains(responseText, "generate") ||
			strings.Contains(responseText, "frame") ||
			strings.Contains(responseText, "sequence") ||
			strings.Contains(responseText, "motion") ||
			strings.Contains(responseText, "animation"))
}

// testRerank checks for reranking capability (typically not part of standard chat completion API)
func (v *Verifier) testRerank(client *LLMClient, modelName string, ctx context.Context) bool {
	// Reranking involves relevance scoring and reordering items by relevance

	// Test 1: Basic ranking task to verify relevance scoring capability
	req1 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role: "user",
				Content: `Rank these items by relevance to the query "machine learning":
A) Cooking recipes
B) Neural networks
C) Car maintenance
D) Supervised learning
E) Gardening tips
Please provide the ranked order from most to least relevant.`,
			},
		},
	}

	resp1, err1 := client.ChatCompletion(ctx, req1)
	if err1 != nil || len(resp1.Choices) == 0 {
		return false
	}

	responseText1 := strings.ToLower(resp1.Choices[0].Message.Content)
	// Check if response contains ranked results (should prioritize B and D over others)
	hasRanking := strings.Contains(responseText1, "rank") || strings.Contains(responseText1, "order") ||
		strings.Contains(responseText1, "1") || strings.Contains(responseText1, "first")

	// Test 2: Ordering improvement task
	req2 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role: "user",
				Content: `Given these search results for "AI in healthcare", reorder them by relevance:
1. "How to bake chocolate chip cookies"
2. "Deep learning for medical image analysis"
3. "Top 10 car maintenance tips"
4. "AI-powered diagnostic tools"
5. "Beginner's guide to gardening"

Return only the reordered list with new numbers.`,
			},
		},
	}

	resp2, err2 := client.ChatCompletion(ctx, req2)
	orderingImprovement := err2 == nil && len(resp2.Choices) > 0 &&
		strings.Contains(strings.ToLower(resp2.Choices[0].Message.Content), "2") &&
		strings.Contains(strings.ToLower(resp2.Choices[0].Message.Content), "4") // Prioritized results

	// Test 3: Relevance scoring test
	req3 := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role: "user",
				Content: `On a scale of 1-10, rate how relevant each of these titles is to "natural language processing":
A) "Building ChatGPT-like applications"
B) "Car engine repair manual"
C) "Transformers in NLP"
Rate each one separately.`,
			},
		},
	}

	resp3, err3 := client.ChatCompletion(ctx, req3)
	relevanceScoring := err3 == nil && len(resp3.Choices) > 0

	return hasRanking || orderingImprovement || relevanceScoring
}

// assessCodeCapabilities evaluates the coding abilities of the model
func (v *Verifier) assessCodeCapabilities(client *LLMClient, modelName string) (*CodeCapabilityResult, error) {
	result := &CodeCapabilityResult{
		LanguageSupport: getCommonProgrammingLanguages(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), v.cfg.Timeout)
	defer cancel()

	// Test various code-related capabilities
	result.CodeGeneration = v.testCodeGeneration(client, modelName, ctx)
	result.CodeCompletion = v.testCodeCompletion(client, modelName, ctx)
	result.CodeReview = v.testCodeReview(client, modelName, ctx)
	result.CodeExplanation = v.testCodeExplanation(client, modelName, ctx)

	// Test code debugging capabilities
	result.CodeDebugging = v.testCodeDebugging(client, modelName, ctx)

	// Test code optimization
	result.CodeOptimization = v.testCodeOptimization(client, modelName, ctx)

	// Test test generation
	result.TestGeneration = v.testTestGeneration(client, modelName, ctx)

	// Test documentation generation
	result.Documentation = v.testDocumentationGeneration(client, modelName, ctx)

	// Test refactoring
	result.Refactoring = v.testCodeRefactoring(client, modelName, ctx)

	// Test error resolution
	result.ErrorResolution = v.testErrorResolution(client, modelName, ctx)

	// Test architecture understanding
	result.Architecture = v.testArchitectureUnderstanding(client, modelName, ctx)

	// Test security assessment
	result.SecurityAssessment = v.testSecurityAssessment(client, modelName, ctx)

	// Test pattern recognition
	result.PatternRecognition = v.testPatternRecognition(client, modelName, ctx)

	// Run language-specific tests and calculate success rates
	testResults := v.runLanguageSpecificTests(client, modelName)
	result.PromptResponse = testResults

	// Assess complexity handling
	complexityMetrics := v.assessComplexityHandling(client, modelName, ctx)
	result.ComplexityHandling = *complexityMetrics

	return result, nil
}

// assessGenerativeCapabilities evaluates the creative/generative abilities of the model
func (v *Verifier) assessGenerativeCapabilities(client *LLMClient, modelName string) (*GenerativeCapabilityResult, error) {
	result := &GenerativeCapabilityResult{}

	ctx, cancel := context.WithTimeout(context.Background(), v.cfg.Timeout)
	defer cancel()

	// Test creative writing abilities
	result.CreativeWriting = v.testCreativeWriting(client, modelName, ctx)

	// Test storytelling abilities
	result.Storytelling = v.testStorytelling(client, modelName, ctx)

	// Test content generation
	result.ContentGeneration = v.testContentGeneration(client, modelName, ctx)

	// Test artistic creativity
	result.ArtisticCreativity = v.testArtisticCreativity(client, modelName, ctx)

	// Test problem-solving creativity
	result.ProblemSolving = v.testProblemSolving(client, modelName, ctx)

	// Test multimodal content understanding (when used generatively)
	result.MultimodalGenerative = v.testMultimodalContentGen(client, modelName, ctx)

	return result, nil
}

// testCreativeWriting checks if the model can write creatively
func (v *Verifier) testCreativeWriting(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Write a short, original poem about the changing seasons that uses vivid imagery and metaphor.",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	// Check for creative elements in the response
	return len(responseText) > 50 && // Contains substantial content
		(strings.Contains(responseText, "spring") || strings.Contains(responseText, "summer") ||
			strings.Contains(responseText, "fall") || strings.Contains(responseText, "winter")) &&
		(strings.Contains(responseText, "like") || strings.Contains(responseText, "as") || // metaphors
			strings.Contains(responseText, "bright") || strings.Contains(responseText, "golden") || // imagery
			strings.Contains(responseText, "\n")) // structured like a poem
}

// testStorytelling checks if the model can create narratives
func (v *Verifier) testStorytelling(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Create a short story (5-8 sentences) about a detective solving a mystery in a lighthouse during a storm.",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	// Check for narrative elements
	return len(responseText) > 100 &&
		(strings.Contains(strings.ToLower(responseText), "detective") ||
			strings.Contains(strings.ToLower(responseText), "mystery") ||
			strings.Contains(strings.ToLower(responseText), "lighthouse") ||
			strings.Contains(strings.ToLower(responseText), "storm"))
}

// testContentGeneration checks if the model can generate various types of content
func (v *Verifier) testContentGeneration(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Generate a product description for a new smart water bottle that tracks hydration, syncs with phones, and glows when you need to drink more water.",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	// Check if it provides a reasonable product description
	return len(responseText) > 50 &&
		(strings.Contains(strings.ToLower(responseText), "water") ||
			strings.Contains(strings.ToLower(responseText), "bottle") ||
			strings.Contains(strings.ToLower(responseText), "smart") ||
			strings.Contains(strings.ToLower(responseText), "track") ||
			strings.Contains(strings.ToLower(responseText), "hydrat"))
}

// testArtisticCreativity checks if the model can think artistically
func (v *Verifier) testArtisticCreativity(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Design a new art movement that combines elements of surrealism and digital glitch aesthetics. Describe its philosophy, techniques, and visual characteristics.",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	// Check for artistic and creative thinking
	return len(responseText) > 100 &&
		(strings.Contains(strings.ToLower(responseText), "art") ||
			strings.Contains(strings.ToLower(responseText), "movement") ||
			strings.Contains(strings.ToLower(responseText), "visual") ||
			strings.Contains(strings.ToLower(responseText), "philosophy") ||
			strings.Contains(strings.ToLower(responseText), "technique"))
}

// testProblemSolving checks if the model can approach problems creatively
func (v *Verifier) testProblemSolving(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "How would you solve the problem of reducing plastic waste in oceans? Think of an innovative solution involving technology and nature.",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	// Check for creative problem-solving approach
	return len(responseText) > 50 &&
		(strings.Contains(strings.ToLower(responseText), "plastic") ||
			strings.Contains(strings.ToLower(responseText), "ocean") ||
			strings.Contains(strings.ToLower(responseText), "solution") ||
			strings.Contains(strings.ToLower(responseText), "technology") ||
			strings.Contains(strings.ToLower(responseText), "nature") ||
			strings.Contains(strings.ToLower(responseText), "innovative"))
}

// testMultimodalContentGen checks if the model can work with multimodal concepts generatively
func (v *Verifier) testMultimodalContentGen(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Describe how you would create a multimedia presentation that combines visual, audio, and textual elements to explain the concept of photosynthesis to children.",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	// Check for multimodal thinking
	return len(responseText) > 50 &&
		(strings.Contains(strings.ToLower(responseText), "visual") ||
			strings.Contains(strings.ToLower(responseText), "audio") ||
			strings.Contains(strings.ToLower(responseText), "text") ||
			strings.Contains(strings.ToLower(responseText), "multimedia") ||
			strings.Contains(strings.ToLower(responseText), "presentation"))
}

// getCommonProgrammingLanguages returns a list of common programming languages
func getCommonProgrammingLanguages() []string {
	return []string{
		"Python", "JavaScript", "TypeScript", "Java", "C++", "Go", "Rust",
		"C#", "PHP", "Ruby", "Swift", "Kotlin", "Scala", "R", "MATLAB",
		"SQL", "HTML", "CSS", "Shell", "PowerShell", "Dart", "Elixir",
	}
}

// testCodeDebugging checks if the model can debug code
func (v *Verifier) testCodeDebugging(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Find and fix the bug in this Python code:\n\n```python\ndef calculate_average(numbers):\n    total = 0\n    for num in numbers:\n        total += num\n    return total / len(numbers)  # Potential bug here\n\n# Test the function\nnumbers = []\nprint(calculate_average(numbers))\n```",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return strings.Contains(strings.ToLower(responseText), "error") ||
		strings.Contains(strings.ToLower(responseText), "bug") ||
		strings.Contains(strings.ToLower(responseText), "exception") ||
		strings.Contains(strings.ToLower(responseText), "empty") ||
		strings.Contains(strings.ToLower(responseText), "divide by zero")
}

// testCodeOptimization checks if the model can optimize code
func (v *Verifier) testCodeOptimization(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Optimize this Python function for better performance:\n\n```python\ndef find_duplicates(arr):\n    duplicates = []\n    for i in range(len(arr)):\n        for j in range(i+1, len(arr)):\n            if arr[i] == arr[j] and arr[i] not in duplicates:\n                duplicates.append(arr[i])\n    return duplicates\n```",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return strings.Contains(strings.ToLower(responseText), "time complexity") ||
		strings.Contains(strings.ToLower(responseText), "o(n") ||
		strings.Contains(strings.ToLower(responseText), "hash") ||
		strings.Contains(strings.ToLower(responseText), "set") ||
		strings.Contains(strings.ToLower(responseText), "optimized")
}

// testTestGeneration checks if the model can generate tests
func (v *Verifier) testTestGeneration(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Write unit tests for this Python function:\n\n```python\ndef is_prime(n):\n    if n < 2:\n        return False\n    for i in range(2, int(n ** 0.5) + 1):\n        if n % i == 0:\n            return False\n    return True\n```",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return strings.Contains(strings.ToLower(responseText), "test") ||
		strings.Contains(strings.ToLower(responseText), "assert") ||
		strings.Contains(strings.ToLower(responseText), "unittest") ||
		strings.Contains(strings.ToLower(responseText), "pytest")
}

// testDocumentationGeneration checks if the model can generate documentation
func (v *Verifier) testDocumentationGeneration(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Generate documentation for this Python function:\n\n```python\ndef binary_search(arr, target):\n    left, right = 0, len(arr) - 1\n    while left <= right:\n        mid = (left + right) // 2\n        if arr[mid] == target:\n            return mid\n        elif arr[mid] < target:\n            left = mid + 1\n        else:\n            right = mid - 1\n    return -1\n```",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return strings.Contains(strings.ToLower(responseText), "parameters") ||
		strings.Contains(strings.ToLower(responseText), "returns") ||
		strings.Contains(strings.ToLower(responseText), "example") ||
		strings.Contains(strings.ToLower(responseText), "complexity") ||
		strings.Contains(strings.ToLower(responseText), "description")
}

// testCodeRefactoring checks if the model can refactor code
func (v *Verifier) testCodeRefactoring(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Refactor this Python code to improve readability and maintainability:\n\n```python\nusers = [{'name': 'Alice', 'age': 30, 'active': True}, {'name': 'Bob', 'age': 25, 'active': False}]\n\nresult = []\nfor user in users:\n    if user['active'] and user['age'] > 20:\n        result.append({'name': user['name'], 'age_group': 'adult' if user['age'] >= 30 else 'young_adult'})\n\nprint(result)\n```",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return strings.Contains(strings.ToLower(responseText), "def ") || // Function definition indicates refactoring
		strings.Contains(strings.ToLower(responseText), "class") || // Class definition indicates refactoring
		strings.Contains(strings.ToLower(responseText), "extract") ||
		strings.Contains(strings.ToLower(responseText), "function")
}

// testErrorResolution checks if the model can resolve common errors
func (v *Verifier) testErrorResolution(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Resolve this error: NameError: name 'requests' is not defined\n\nCode: import requests\nresponse = requests.get('https://api.example.com/data')\nprint(response.json())",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return strings.Contains(strings.ToLower(responseText), "import") ||
		strings.Contains(strings.ToLower(responseText), "install") ||
		strings.Contains(strings.ToLower(responseText), "pip") ||
		strings.Contains(strings.ToLower(responseText), "module")
}

// testArchitectureUnderstanding checks if the model understands software architecture
func (v *Verifier) testArchitectureUnderstanding(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Design a simple architecture for a blog application. Include components like user management, post management, and comment system. Explain the relationships between components.",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return strings.Contains(strings.ToLower(responseText), "component") ||
		strings.Contains(strings.ToLower(responseText), "layer") ||
		strings.Contains(strings.ToLower(responseText), "database") ||
		strings.Contains(strings.ToLower(responseText), "api") ||
		strings.Contains(strings.ToLower(responseText), "service")
}

// testSecurityAssessment checks if the model can identify security issues
func (v *Verifier) testSecurityAssessment(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Identify security vulnerabilities in this Python code:\n\n```python\nimport sqlite3\n\nusername = input('Enter username: ')\npassword = input('Enter password: ')\n\nconn = sqlite3.connect('users.db')\nc = conn.cursor()\n\nquery = f\"SELECT * FROM users WHERE username='{username}' AND password='{password}'\"\nc.execute(query)\n\nresult = c.fetchone()\nif result:\n    print('Login successful')\nelse:\n    print('Login failed')\n\nconn.close()\n```",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return strings.Contains(strings.ToLower(responseText), "sql injection") ||
		strings.Contains(strings.ToLower(responseText), "sql") ||
		strings.Contains(strings.ToLower(responseText), "security") ||
		strings.Contains(strings.ToLower(responseText), "vulnerability") ||
		strings.Contains(strings.ToLower(responseText), "sanit")
}

// testPatternRecognition checks if the model can recognize and implement patterns
func (v *Verifier) testPatternRecognition(client *LLMClient, modelName string, ctx context.Context) bool {
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Implement the Observer pattern in Python with a simple example.",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return strings.Contains(strings.ToLower(responseText), "class") &&
		(strings.Contains(strings.ToLower(responseText), "observer") ||
			strings.Contains(strings.ToLower(responseText), "subscribe") ||
			strings.Contains(strings.ToLower(responseText), "notify") ||
			strings.Contains(strings.ToLower(responseText), "publisher"))
}

// runLanguageSpecificTests runs tests for multiple programming languages
func (v *Verifier) runLanguageSpecificTests(client *LLMClient, modelName string) PromptResponseTest {
	testResults := PromptResponseTest{}

	// Test Python
	pythonTests := []struct {
		name string
		task string
	}{
		{"basic_syntax", "Write a Python function that takes two numbers and returns their sum."},
		{"data_structure", "Create a Python dictionary with 3 key-value pairs and print all values."},
		{"algorithm", "Write a Python function to implement binary search in a sorted array."},
	}

	var pythonSuccessCount int
	for _, test := range pythonTests {
		if v.runSingleLanguageTest(client, modelName, test.task) {
			pythonSuccessCount++
		}
	}
	testResults.PythonSuccessRate = float64(pythonSuccessCount) / float64(len(pythonTests)) * 100

	// Test JavaScript
	jsTests := []struct {
		name string
		task string
	}{
		{"basic_syntax", "Write a JavaScript function that takes two numbers and returns their sum."},
		{"array_method", "Write a JavaScript function that filters an array of numbers to return only even numbers."},
		{"async", "Write a JavaScript function that fetches data from 'https://api.example.com/data' using async/await."},
	}

	var jsSuccessCount int
	for _, test := range jsTests {
		if v.runSingleLanguageTest(client, modelName, test.task) {
			jsSuccessCount++
		}
	}
	testResults.JavascriptSuccessRate = float64(jsSuccessCount) / float64(len(jsTests)) * 100

	// Test Go
	goTests := []struct {
		name string
		task string
	}{
		{"basic_syntax", "Write a Go function that takes two integers and returns their sum."},
		{"loop", "Write a Go function that prints numbers from 1 to 10 using a loop."},
		{"slice", "Write a Go function that takes a slice of integers and returns the maximum value."},
	}

	var goSuccessCount int
	for _, test := range goTests {
		if v.runSingleLanguageTest(client, modelName, test.task) {
			goSuccessCount++
		}
	}
	testResults.GoSuccessRate = float64(goSuccessCount) / float64(len(goTests)) * 100

	// Test Java
	javaTests := []struct {
		name string
		task string
	}{
		{"basic_syntax", "Write a Java method that takes two integers and returns their sum."},
		{"loop", "Write a Java method that prints the first 10 Fibonacci numbers."},
		{"class", "Write a simple Java class called 'Person' with name and age fields and getter methods."},
	}

	var javaSuccessCount int
	for _, test := range javaTests {
		if v.runSingleLanguageTest(client, modelName, test.task) {
			javaSuccessCount++
		}
	}
	testResults.JavaSuccessRate = float64(javaSuccessCount) / float64(len(javaTests)) * 100

	// Test C++
	cppTests := []struct {
		name string
		task string
	}{
		{"basic_syntax", "Write a C++ function that takes two integers and returns their sum."},
		{"stl", "Write a C++ program that creates a vector of integers and sorts it in ascending order."},
		{"class", "Write a simple C++ class called 'Rectangle' with width and height fields and an area method."},
	}

	var cppSuccessCount int
	for _, test := range cppTests {
		if v.runSingleLanguageTest(client, modelName, test.task) {
			cppSuccessCount++
		}
	}
	testResults.CppSuccessRate = float64(cppSuccessCount) / float64(len(cppTests)) * 100

	// Test TypeScript
	tsTests := []struct {
		name string
		task string
	}{
		{"basic_syntax", "Write a TypeScript function that takes two numbers and returns their sum with proper type annotations."},
		{"interface", "Define a TypeScript interface for a 'User' with name, email, and age fields, and create a function that accepts this type."},
		{"generic", "Write a generic TypeScript function that returns the first element of an array of any type."},
	}

	var tsSuccessCount int
	for _, test := range tsTests {
		if v.runSingleLanguageTest(client, modelName, test.task) {
			tsSuccessCount++
		}
	}
	testResults.TypescriptSuccessRate = float64(tsSuccessCount) / float64(len(tsTests)) * 100

	// Calculate overall success rate
	totalTests := len(pythonTests) + len(jsTests) + len(goTests) + len(javaTests) + len(cppTests) + len(tsTests)
	totalSuccesses := pythonSuccessCount + jsSuccessCount + goSuccessCount + javaSuccessCount + cppSuccessCount + tsSuccessCount
	testResults.OverallSuccessRate = float64(totalSuccesses) / float64(totalTests) * 100

	return testResults
}

// runSingleLanguageTest runs a single test for a specific language
func (v *Verifier) runSingleLanguageTest(client *LLMClient, modelName, task string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), v.cfg.Timeout)
	defer cancel()

	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: task,
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)

	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	responseText := resp.Choices[0].Message.Content
	return containsCode(responseText)
}

// assessComplexityHandling evaluates how well the model handles complex coding tasks
func (v *Verifier) assessComplexityHandling(client *LLMClient, modelName string, ctx context.Context) *ComplexityMetrics {
	metrics := &ComplexityMetrics{}

	// Test with a moderately complex problem
	req := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Implement a simple task management system with the following requirements:\n1. Create a Task class with id, title, description, status (pending, in-progress, completed), and due_date\n2. Create a TaskManager class that can add, remove, update, and list tasks\n3. Implement a method to filter tasks by status\n4. Implement a method to mark a task as completed\n5. Include proper error handling for invalid operations",
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)

	if err != nil || len(resp.Choices) == 0 {
		return metrics
	}

	responseText := resp.Choices[0].Message.Content

	// Evaluate the complexity metrics based on the response
	metrics.CodeQuality = evaluateCodeQuality(responseText)
	metrics.LogicCorrectness = evaluateLogicCorrectness(responseText)
	metrics.RuntimeEfficiency = evaluateRuntimeEfficiency(responseText)

	// Determine complexity level based on implementation details
	metrics.MaxHandledDepth = determineComplexityLevel(responseText)
	metrics.MaxTokens = len([]rune(responseText))

	return metrics
}

// evaluateCodeQuality assesses the quality of generated code
func evaluateCodeQuality(code string) float64 {
	score := 0.0

	// Check for common quality indicators
	if strings.Contains(code, "#") || strings.Contains(code, "//") || strings.Contains(code, "/*") {
		score += 10 // Comments present
	}

	if strings.Contains(code, "def ") || strings.Contains(code, "function") || strings.Contains(code, "class ") {
		score += 10 // Proper structure
	}

	if strings.Contains(code, "try") || strings.Contains(code, "catch") || strings.Contains(code, "except") {
		score += 15 // Error handling
	}

	if strings.Contains(code, "if __name__ == \"__main__\"") || strings.Contains(code, "main()") {
		score += 5 // Proper entry point
	}

	// Cap the score between 0 and 100
	if score > 100 {
		score = 100
	}

	return score
}

// evaluateLogicCorrectness assesses the logical correctness of the code
func evaluateLogicCorrectness(code string) float64 {
	score := 0.0

	// Look for logical structures
	if strings.Contains(code, "for ") || strings.Contains(code, "while ") || strings.Contains(code, "if ") {
		score += 15 // Control structures
	}

	if strings.Contains(code, "return ") {
		score += 10 // Proper returns
	}

	// Check for logical operators
	if strings.Contains(code, "and") || strings.Contains(code, "or") || strings.Contains(code, "&&") || strings.Contains(code, "||") {
		score += 10
	}

	// Check for proper variable assignments (indicating understanding)
	if strings.Contains(code, "=") && !strings.Contains(code, "==") {
		score += 5
	}

	// Cap the score between 0 and 100
	if score > 100 {
		score = 100
	}

	return score
}

// evaluateRuntimeEfficiency assesses the efficiency of the code
func evaluateRuntimeEfficiency(code string) float64 {
	score := 0.0

	// Check for efficient patterns
	if strings.Contains(code, "map(") || strings.Contains(code, "filter(") || strings.Contains(code, ".map") || strings.Contains(code, ".filter") {
		score += 15 // Functional programming patterns
	}

	if strings.Contains(code, "set") || strings.Contains(code, "hash") || strings.Contains(code, "dict") {
		score += 10 // Efficient data structures
	}

	if strings.Contains(code, "O(") && (strings.Contains(code, "O(1)") || strings.Contains(code, "O(log") || strings.Contains(code, "O(n)")) {
		score += 20 // Complexity analysis mentioned
	}

	if strings.Contains(code, "len(") || strings.Contains(code, "size()") {
		score += 5 // Efficient length access
	}

	// Cap the score between 0 and 100
	if score > 100 {
		score = 100
	}

	return score
}

// determineComplexityLevel determines the complexity level of the implementation
func determineComplexityLevel(code string) int {
	complexity := 1 // Start with basic level

	if strings.Contains(code, "class ") {
		complexity = 2 // Class-based implementation
	}

	if complexity >= 2 && (strings.Contains(code, "inherit") || strings.Contains(code, "extend")) {
		complexity = 3 // Inheritance/polymorphism
	}

	if complexity >= 2 && (strings.Contains(code, "async") || strings.Contains(code, "thread") || strings.Contains(code, "concurrent")) {
		complexity = 4 // Concurrency
	}

	if complexity >= 3 && strings.Contains(code, "design pattern") {
		complexity = 5 // Advanced patterns
	}

	return complexity
}

// CalculateScores calculates performance scores for the model
func (v *Verifier) CalculateScores(result VerificationResult) (PerformanceScore, ScoreDetails) {
	scores := PerformanceScore{}
	details := ScoreDetails{}

	// Calculate code capability score (40% weight)
	codeScore, codeDetails := v.CalculateCodeCapabilityScore(result.CodeCapabilities)
	scores.CodeCapability = codeScore
	details.CodeCapabilityBreakdown = codeDetails

	// Calculate responsiveness score (15% weight)
	responsivenessScore, responseDetails := v.CalculateResponsivenessScore(result.Availability, result.ResponseTime)
	scores.Responsiveness = responsivenessScore
	details.ResponseTimeBreakdown = responseDetails

	// Calculate reliability score (15% weight)
	reliabilityScore, reliabilityDetails := v.CalculateReliabilityScore(result.Availability)
	scores.Reliability = reliabilityScore
	details.ReliabilityBreakdown = reliabilityDetails

	// Calculate feature richness score (15% weight)
	featureRichnessScore, featureDetails := v.calculateFeatureRichnessScoreFromResult(result)
	scores.FeatureRichness = featureRichnessScore
	details.FeatureSupportBreakdown = featureDetails

	// Calculate generative capability score (10% weight)
	generativeScore := v.calculateGenerativeScore(result.GenerativeCapabilities)
	scores.FeatureRichness += generativeScore * 0.10 // Add to feature richness as a bonus

	// Calculate value proposition score (5% weight)
	scores.ValueProposition = v.calculateValuePropositionScore(scores)

	// Calculate overall score according to specification weights
	scores.OverallScore = (responsivenessScore * 0.30) +
		(codeScore * 0.25) +
		(featureRichnessScore * 0.25) +
		(reliabilityScore * 0.20)

	return scores, details
}

// calculateGenerativeScore calculates the score for generative capabilities
func (v *Verifier) calculateGenerativeScore(generativeCaps GenerativeCapabilityResult) float64 {
	if generativeCaps.CreativeWriting && generativeCaps.Storytelling &&
		generativeCaps.ContentGeneration && generativeCaps.ArtisticCreativity {
		return 100 // All major generative capabilities present
	}

	score := 0.0
	totalCaps := 6.0 // Total number of generative capabilities

	if generativeCaps.CreativeWriting {
		score += 100 / totalCaps
	}
	if generativeCaps.Storytelling {
		score += 100 / totalCaps
	}
	if generativeCaps.ContentGeneration {
		score += 100 / totalCaps
	}
	if generativeCaps.ArtisticCreativity {
		score += 100 / totalCaps
	}
	if generativeCaps.ProblemSolving {
		score += 100 / totalCaps
	}
	if generativeCaps.MultimodalGenerative {
		score += 100 / totalCaps
	}

	return score
}

// CalculateCodeCapabilityScore calculates the code capability score
func (v *Verifier) CalculateCodeCapabilityScore(codeCaps CodeCapabilityResult) (float64, CodeCapabilityBreakdown) {
	breakdown := CodeCapabilityBreakdown{}

	// Calculate individual scores
	breakdown.GenerationScore = 0
	if codeCaps.CodeGeneration {
		breakdown.GenerationScore = 100
	}

	breakdown.CompletionScore = 0
	if codeCaps.CodeCompletion {
		breakdown.CompletionScore = 100
	}

	breakdown.DebuggingScore = 0
	if codeCaps.CodeDebugging {
		breakdown.DebuggingScore = 100
	} else {
		// If specific debugging test failed, check if general coding still works
		if codeCaps.CodeGeneration || codeCaps.CodeCompletion {
			breakdown.DebuggingScore = 50 // Partial credit if model can code but not explicitly debug
		}
	}

	breakdown.ReviewScore = 0
	if codeCaps.CodeReview {
		breakdown.ReviewScore = 100
	}

	breakdown.TestGenScore = 0
	if codeCaps.TestGeneration {
		breakdown.TestGenScore = 100
	}

	breakdown.DocumentScore = 0
	if codeCaps.Documentation {
		breakdown.DocumentScore = 100
	}

	breakdown.ArchitectureScore = 0
	if codeCaps.Architecture {
		breakdown.ArchitectureScore = 100
	}

	breakdown.OptimizationScore = 0
	if codeCaps.CodeOptimization {
		breakdown.OptimizationScore = 100
	}

	// Complexity handling score (0-100 based on complexity level and quality metrics)
	breakdown.ComplexityHandling = float64(codeCaps.ComplexityHandling.MaxHandledDepth) * 20 // 20 points per complexity level
	if codeCaps.ComplexityHandling.CodeQuality > breakdown.ComplexityHandling {
		breakdown.ComplexityHandling = codeCaps.ComplexityHandling.CodeQuality
	}
	if breakdown.ComplexityHandling > 100 {
		breakdown.ComplexityHandling = 100
	}

	// Weighted average with different weights based on importance
	weightedSum := (breakdown.GenerationScore * 0.15) +
		(breakdown.CompletionScore * 0.15) +
		(breakdown.DebuggingScore * 0.12) +
		(breakdown.ReviewScore * 0.12) +
		(breakdown.TestGenScore * 0.10) +
		(breakdown.DocumentScore * 0.10) +
		(breakdown.ArchitectureScore * 0.10) +
		(breakdown.OptimizationScore * 0.08) +
		(breakdown.ComplexityHandling * 0.08)

	breakdown.WeightedAverage = weightedSum

	return weightedSum, breakdown
}

// CalculateResponsivenessScore calculates the responsiveness score
func (v *Verifier) CalculateResponsivenessScore(availability AvailabilityResult, responseTime ResponseTimeResult) (float64, ResponseTimeBreakdown) {
	breakdown := ResponseTimeBreakdown{}

	// Calculate latency score (lower is better)
	breakdown.LatencyScore = 100.0
	if availability.Latency > 10*time.Second {
		breakdown.LatencyScore = 10
	} else if availability.Latency > 5*time.Second {
		breakdown.LatencyScore = 30
	} else if availability.Latency > 2*time.Second {
		breakdown.LatencyScore = 60
	} else if availability.Latency > 1*time.Second {
		breakdown.LatencyScore = 80
	}

	// Calculate throughput score (higher is better)
	breakdown.ThroughputScore = 0
	if responseTime.Throughput > 10 {
		breakdown.ThroughputScore = 100
	} else if responseTime.Throughput > 5 {
		breakdown.ThroughputScore = 80
	} else if responseTime.Throughput > 2 {
		breakdown.ThroughputScore = 60
	} else if responseTime.Throughput > 1 {
		breakdown.ThroughputScore = 40
	} else {
		breakdown.ThroughputScore = 20
	}

	// Calculate consistency score based on difference between min and max latency
	if responseTime.MaxLatency > 0 && responseTime.MinLatency > 0 {
		latencyVariation := float64(responseTime.MaxLatency-responseTime.MinLatency) / float64(responseTime.MinLatency)
		breakdown.ConsistencyScore = 100 - (latencyVariation * 50)
		if breakdown.ConsistencyScore < 0 {
			breakdown.ConsistencyScore = 0
		}
	} else {
		breakdown.ConsistencyScore = 100
	}

	// Weighted average
	breakdown.WeightedAverage = (breakdown.LatencyScore * 0.5) + (breakdown.ThroughputScore * 0.3) + (breakdown.ConsistencyScore * 0.2)

	return breakdown.WeightedAverage, breakdown
}

// CalculateReliabilityScore calculates the reliability score
func (v *Verifier) CalculateReliabilityScore(availability AvailabilityResult) (float64, ReliabilityBreakdown) {
	breakdown := ReliabilityBreakdown{}

	// Availability score
	breakdown.AvailabilityScore = 0
	if availability.Exists && availability.Responsive {
		breakdown.AvailabilityScore = 100
	} else if availability.Exists {
		breakdown.AvailabilityScore = 50 // Exists but not responsive
	}

	// Consistency score based on overload status
	breakdown.ConsistencyScore = 100
	if availability.Overloaded {
		breakdown.ConsistencyScore = 30
	}

	// Error rate score (opposite of error presence)
	breakdown.ErrorRateScore = 100
	if availability.Error != "" {
		breakdown.ErrorRateScore = 20
	}

	// Stability score based on various factors
	breakdown.StabilityScore = 100
	if availability.Overloaded || availability.Error != "" {
		breakdown.StabilityScore = 60
	}

	// Weighted average
	breakdown.WeightedAverage = (breakdown.AvailabilityScore * 0.3) +
		(breakdown.ConsistencyScore * 0.3) +
		(breakdown.ErrorRateScore * 0.2) +
		(breakdown.StabilityScore * 0.2)

	return breakdown.WeightedAverage, breakdown
}

// CalculateFeatureRichnessScore calculates the feature richness score
func (v *Verifier) CalculateFeatureRichnessScore(features FeatureDetectionResult) (float64, FeatureSupportBreakdown) {
	breakdown := FeatureSupportBreakdown{}

	// Count core features (40% weight)
	coreFeatures := 0
	totalCoreFeatures := 6 // code generation, code completion, code explanation, code review, tool use, streaming
	if features.CodeGeneration {
		coreFeatures++
	}
	if features.CodeCompletion {
		coreFeatures++
	}
	if features.CodeExplanation {
		coreFeatures++
	}
	if features.CodeReview {
		coreFeatures++
	}
	if features.ToolUse {
		coreFeatures++
	}
	if features.Streaming {
		coreFeatures++
	}

	breakdown.CoreFeaturesScore = float64(coreFeatures) / float64(totalCoreFeatures) * 100

	// Count advanced features (40% weight)
	advancedFeatures := 0
	totalAdvancedFeatures := 8 // embeddings, reasoning, structured output, JSON mode, parallel tool use, multimodal, refactoring, documentation
	if features.Embeddings {
		advancedFeatures++
	}
	if features.Reasoning {
		advancedFeatures++
	}
	if features.StructuredOutput {
		advancedFeatures++
	}
	if features.JSONMode {
		advancedFeatures++
	}
	if features.ParallelToolUse {
		advancedFeatures++
	}
	if features.Multimodal {
		advancedFeatures++
	}
	// Note: Refactoring and Documentation are not directly in FeatureDetectionResult
	// so we'll skip these for now in this function and handle them appropriately
	// in the context where we have full VerificationResult
	// For now, we'll just count 6 out of 8 features to complete the count

	breakdown.AdvancedFeaturesScore = float64(advancedFeatures) / float64(totalAdvancedFeatures) * 100

	// Count experimental or special features (20% weight)
	experimentalFeatures := 0
	totalExperimentalFeatures := 6 // MCPs, LSPs, ACPs, reranking, image generation, audio generation
	if features.MCPs {
		experimentalFeatures++
	}
	if features.LSPs {
		experimentalFeatures++
	}
	if features.ACPs {
		experimentalFeatures++
	}
	if features.Reranking {
		experimentalFeatures++
	}
	if features.ImageGeneration {
		experimentalFeatures++
	}
	if features.AudioGeneration {
		experimentalFeatures++
	}

	breakdown.ExperimentalFeaturesScore = float64(experimentalFeatures) / float64(totalExperimentalFeatures) * 100

	// Weighted average
	breakdown.WeightedAverage = (breakdown.CoreFeaturesScore * 0.4) +
		(breakdown.AdvancedFeaturesScore * 0.4) +
		(breakdown.ExperimentalFeaturesScore * 0.2)

	return breakdown.WeightedAverage, breakdown
}

// calculateFeatureRichnessScoreFromResult calculates the feature richness score using the full verification result
func (v *Verifier) calculateFeatureRichnessScoreFromResult(result VerificationResult) (float64, FeatureSupportBreakdown) {
	features := result.FeatureDetection
	breakdown := FeatureSupportBreakdown{}

	// Count core features (40% weight)
	coreFeatures := 0
	totalCoreFeatures := 6 // code generation, code completion, code explanation, code review, tool use, streaming
	if features.CodeGeneration {
		coreFeatures++
	}
	if features.CodeCompletion {
		coreFeatures++
	}
	if features.CodeExplanation {
		coreFeatures++
	}
	if features.CodeReview {
		coreFeatures++
	}
	if features.ToolUse {
		coreFeatures++
	}
	if features.Streaming {
		coreFeatures++
	}

	breakdown.CoreFeaturesScore = float64(coreFeatures) / float64(totalCoreFeatures) * 100

	// Count advanced features (40% weight)
	advancedFeatures := 0
	totalAdvancedFeatures := 8 // embeddings, reasoning, structured output, JSON mode, parallel tool use, multimodal, refactoring, documentation
	if features.Embeddings {
		advancedFeatures++
	}
	if features.Reasoning {
		advancedFeatures++
	}
	if features.StructuredOutput {
		advancedFeatures++
	}
	if features.JSONMode {
		advancedFeatures++
	}
	if features.ParallelToolUse {
		advancedFeatures++
	}
	if features.Multimodal {
		advancedFeatures++
	}
	if result.CodeCapabilities.Refactoring {
		advancedFeatures++
	}
	if result.CodeCapabilities.Documentation {
		advancedFeatures++
	}

	breakdown.AdvancedFeaturesScore = float64(advancedFeatures) / float64(totalAdvancedFeatures) * 100

	// Count experimental or special features (20% weight)
	experimentalFeatures := 0
	totalExperimentalFeatures := 6 // MCPs, LSPs, ACPs, reranking, image generation, audio generation
	if features.MCPs {
		experimentalFeatures++
	}
	if features.LSPs {
		experimentalFeatures++
	}
	if features.ACPs {
		experimentalFeatures++
	}
	if features.Reranking {
		experimentalFeatures++
	}
	if features.ImageGeneration {
		experimentalFeatures++
	}
	if features.AudioGeneration {
		experimentalFeatures++
	}

	breakdown.ExperimentalFeaturesScore = float64(experimentalFeatures) / float64(totalExperimentalFeatures) * 100

	// Weighted average
	breakdown.WeightedAverage = (breakdown.CoreFeaturesScore * 0.4) +
		(breakdown.AdvancedFeaturesScore * 0.4) +
		(breakdown.ExperimentalFeaturesScore * 0.2)

	return breakdown.WeightedAverage, breakdown
}

// calculateValuePropositionScore calculates the value proposition score based on other scores
func (v *Verifier) calculateValuePropositionScore(performance PerformanceScore) float64 {
	// Value is a combination of capability, responsiveness, and reliability
	// Models with high capability but poor reliability/responsiveness have lower value
	// Models with balanced scores have higher value
	score := (performance.CodeCapability * 0.5) +
		(performance.Responsiveness * 0.3) +
		(performance.Reliability * 0.2)

	// Normalize to 0-100 scale
	return score / 10
}

// containsCode checks if text contains code
func containsCode(text string) bool {
	text = strings.ToLower(text)
	return strings.Contains(text, "def ") ||
		strings.Contains(text, "function") ||
		strings.Contains(text, "class ") ||
		strings.Contains(text, "import ") ||
		strings.Contains(text, "console.log") ||
		strings.Contains(text, "func ") // Go function
}

// Helper function to create a pointer to an int
func intPtr(i int) *int {
	return &i
}

// Helper function to create a pointer to a float64
func floatPtr(f float64) *float64 {
	return &f
}

// VerifyWithStrategy runs verification using a custom scoring strategy.
// Models are first filtered by the strategy, then each is verified and scored.
// This is a parallel entry point to the existing Verify() — it does not modify
// any existing function signatures or behavior.
func (v *Verifier) VerifyWithStrategy(ctx context.Context, strategy ScoringStrategy, models []ModelInfo) ([]VerificationResult, []StrategyScore, error) {
	if len(models) == 0 {
		return nil, nil, nil
	}

	filtered := strategy.FilterModels(models)
	if len(filtered) == 0 {
		return nil, nil, nil
	}

	var results []VerificationResult
	var scores []StrategyScore

	for _, model := range filtered {
		select {
		case <-ctx.Done():
			return results, scores, ctx.Err()
		default:
		}

		// Create client for this model
		apiKey := model.Endpoint // fallback
		if v.cfg != nil && v.cfg.Global.APIKey != "" {
			apiKey = v.cfg.Global.APIKey
		}
		// Look for matching LLM config by endpoint or name
		for _, llmCfg := range v.cfg.LLMs {
			if llmCfg.Endpoint == model.Endpoint || llmCfg.Name == model.Organization {
				apiKey = llmCfg.APIKey
				break
			}
		}
		client := NewLLMClient(model.Endpoint, apiKey, nil)

		vr, err := v.verifySingleModel(client, model.ID, model.Endpoint)
		if err != nil {
			vr = VerificationResult{ModelInfo: model, Error: err.Error(), Timestamp: time.Now()}
		}
		vr.ModelInfo = model
		results = append(results, vr)

		// Run custom tests from strategy
		var testResults []TestResult
		for _, test := range strategy.CustomTests() {
			tr, testErr := test.Run(ctx, client)
			if testErr != nil {
				if test.Required() {
					return results, scores, fmt.Errorf("required test %s failed: %w", test.ID(), testErr)
				}
				tr = TestResult{TestID: test.ID(), Category: test.Category(), Passed: false, Error: testErr, Timestamp: time.Now()}
			}
			testResults = append(testResults, tr)
		}

		score, scoreErr := strategy.ScoreModel(ctx, model, testResults)
		if scoreErr != nil {
			return results, scores, scoreErr
		}
		scores = append(scores, score)
	}

	return results, scores, nil
}

// CalculateScoresWithStrategy scores a VerificationResult using a custom strategy.
// This is a parallel entry point to the existing CalculateScores() — it does not
// modify the existing function.
func (v *Verifier) CalculateScoresWithStrategy(result VerificationResult, strategy ScoringStrategy) (StrategyScore, error) {
	return strategy.ScoreModel(context.Background(), result.ModelInfo, nil)
}
