package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"digital.vasic.llmsverifier/logging"
	opencode_config "digital.vasic.llmsverifier/pkg/opencode/config"
	"digital.vasic.llmsverifier/scoring"
	"digital.vasic.llmsverifier/verification"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// ModelProviderService implements OpenCode's 3-tier model discovery system
// Priority: 1. User config → 2. Provider API → 3. models.dev fallback
type ModelProviderService struct {
	// Priority 1: Configuration files
	configPath     string
	configResolver *opencode_config.EnvResolver

	// Priority 2: Provider API clients
	httpClient      *http.Client
	providerClients map[string]*ProviderClient
	clientsMutex    sync.RWMutex // Protects providerClients map for concurrent access

	// Priority 3: models.dev fallback
	modelsDevClient *verification.EnhancedModelsDevClient
	logger          *logging.Logger

	// Output formatting
	displayFormatter *scoring.ModelDisplayName

	// Cache for provider models (24 hour TTL)
	cache      map[string]*providerCacheEntry
	cacheMutex sync.RWMutex
	cacheTTL   int // hours
}

// ProviderClient handles direct API communication with providers
type ProviderClient struct {
	ProviderID string
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	logger     *logging.Logger
}

// Model represents a unified model structure across all sources
type Model struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	ProviderID      string                 `json:"provider_id"`
	Provider        string                 `json:"provider"`       // Alias for ProviderID for compatibility
	ProviderName    string                 `json:"provider_name"`
	DisplayName     string                 `json:"display_name"`
	Features        map[string]interface{} `json:"features"`
	Metadata        map[string]interface{} `json:"metadata"`       // Additional metadata
	MaxTokens       int                    `json:"max_tokens"`
	ContextWindow   int                    `json:"context_window"` // Alias for MaxTokens for compatibility
	CostPer1MInput  float64                `json:"cost_per_1m_input"`
	CostPer1MOutput float64                `json:"cost_per_1m_output"`
	SupportsBrotli  bool                   `json:"supports_brotli"`
	SupportsHTTP3   bool                   `json:"supports_http3"`
	SupportsToon    bool                   `json:"supports_toon"`
	IsFree          bool                   `json:"is_free"`
	IsOpenSource    bool                   `json:"is_open_source"`
	ResponseTimeMS  float64                `json:"response_time_ms,omitempty"`
	Source          string                 `json:"source"` // "config", "api", or "models.dev"
}

// providerCacheEntry stores cached provider data
type providerCacheEntry struct {
	models     []Model
	timestamp  time.Time
	providerID string
}

// NewModelProviderService creates a new model provider service
func NewModelProviderService(configPath string, logger *logging.Logger) *ModelProviderService {
	// Ensure config path exists
	if _, err := os.Stat(configPath); err != nil {
		if logger != nil {
			logger.Warning(fmt.Sprintf("Config file not found: %s, will use defaults", configPath), nil)
		}
	}

	return &ModelProviderService{
		configPath:       configPath,
		configResolver:   opencode_config.NewEnvResolver(true),
		httpClient:       createHTTPClient(),
		providerClients:  make(map[string]*ProviderClient),
		modelsDevClient:  verification.NewEnhancedModelsDevClient(logger),
		logger:           logger,
		displayFormatter: scoring.NewModelDisplayName(),
		cache:            make(map[string]*providerCacheEntry),
		cacheTTL:         24, // 24 hour cache
	}
}

// createHTTPClient creates a configured HTTP client
func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// logInfo logs info message if logger is not nil
func (mps *ModelProviderService) logInfo(msg string) {
	if mps.logger != nil {
		mps.logger.Info(msg, nil)
	}
}

// logDebug logs debug message if logger is not nil
func (mps *ModelProviderService) logDebug(msg string) {
	if mps.logger != nil {
		mps.logger.Debug(msg, nil)
	}
}

// logWarning logs warning message if logger is not nil
func (mps *ModelProviderService) logWarning(msg string) {
	if mps.logger != nil {
		mps.logger.Warning(msg, nil)
	}
}

// logError logs error message if logger is not nil
func (mps *ModelProviderService) logError(msg string) {
	if mps.logger != nil {
		mps.logger.Error(msg, nil)
	}
}

// RegisterProvider registers a provider API client
func (mps *ModelProviderService) RegisterProvider(providerID, baseURL, apiKey string) {
	client := &ProviderClient{
		ProviderID: providerID,
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		APIKey:     apiKey,
		HTTPClient: mps.httpClient,
		logger:     mps.logger,
	}

	mps.clientsMutex.Lock()
	mps.providerClients[providerID] = client
	mps.clientsMutex.Unlock()
	mps.logInfo(fmt.Sprintf("Registered provider: %s", providerID))
}

// RegisterAllProviders registers all 32 providers from environment variables
func (mps *ModelProviderService) RegisterAllProviders() {
	providerConfigs := mps.getAllProviderConfigs()

	for providerID, config := range providerConfigs {
		mps.RegisterProvider(providerID, config.BaseURL, config.APIKey)
	}

	mps.logInfo(fmt.Sprintf("Registered %d providers", len(providerConfigs)))
}

// providerConfig holds provider configuration
type providerConfig struct {
	BaseURL string
	APIKey  string
}

// getAllProviderConfigs returns configuration for all 32 providers
func (mps *ModelProviderService) getAllProviderConfigs() map[string]providerConfig {
	return map[string]providerConfig{
		"openai":      {BaseURL: "https://api.openai.com/v1", APIKey: os.Getenv("OPENAI_API_KEY")},
		"anthropic":   {BaseURL: "https://api.anthropic.com/v1", APIKey: os.Getenv("ANTHROPIC_API_KEY")},
		"huggingface": {BaseURL: "https://api-inference.huggingface.co", APIKey: os.Getenv("HUGGINGFACE_API_KEY")},
		"groq":        {BaseURL: "https://api.groq.com/openai/v1", APIKey: os.Getenv("GROQ_API_KEY")},
		"gemini":      {BaseURL: "https://generativelanguage.googleapis.com/v1", APIKey: os.Getenv("GEMINI_API_KEY")},
		"deepseek":    {BaseURL: "https://api.deepseek.com/v1", APIKey: os.Getenv("DEEPSEEK_API_KEY")},
		"nvidia":      {BaseURL: "https://integrate.api.nvidia.com/v1", APIKey: os.Getenv("NVIDIA_API_KEY")},
		"openrouter":  {BaseURL: "https://openrouter.ai/api/v1", APIKey: os.Getenv("OPENROUTER_API_KEY")},
		"replicate":   {BaseURL: "https://api.replicate.com/v1", APIKey: os.Getenv("REPLICATE_API_KEY")},
		"fireworks":   {BaseURL: "https://api.fireworks.ai/inference/v1", APIKey: os.Getenv("FIREWORKS_API_KEY")},
		"together":    {BaseURL: "https://api.together.xyz/v1", APIKey: os.Getenv("TOGETHER_API_KEY")},
		"perplexity":  {BaseURL: "https://api.perplexity.ai", APIKey: os.Getenv("PERPLEXITY_API_KEY")},
		"mistral":     {BaseURL: "https://api.mistral.ai/v1", APIKey: os.Getenv("MISTRAL_API_KEY")},
		"codestral":   {BaseURL: "https://codestral.mistral.ai/v1", APIKey: os.Getenv("CODESTRAL_API_KEY")},
		"cloudflare":  {BaseURL: "https://api.cloudflare.com/client/v4/accounts/YOUR_ACCOUNT/ai/v1", APIKey: os.Getenv("CLOUDFLARE_API_KEY")},
		"sambanova":   {BaseURL: "https://api.sambanova.ai/v1", APIKey: os.Getenv("SAMBANOVA_API_KEY")},
		"cerebras":    {BaseURL: "https://api.cerebras.ai/v1", APIKey: os.Getenv("CEREBRAS_API_KEY")},
		"modal":       {BaseURL: "https://api.modal.com/v1", APIKey: os.Getenv("MODAL_API_KEY")},
		"inference":   {BaseURL: "https://api.inference.net/v1", APIKey: os.Getenv("INFERENCE_API_KEY")},
		"siliconflow": {BaseURL: "https://api.siliconflow.cn/v1", APIKey: os.Getenv("SILICONFLOW_API_KEY")},
		"novita":      {BaseURL: "https://api.novita.ai/v3/openai", APIKey: os.Getenv("NOVITA_API_KEY")},
		"upstage":     {BaseURL: "https://api.upstage.ai/v1/solar", APIKey: os.Getenv("UPSTAGE_API_KEY")},
		"nlpcloud":    {BaseURL: "https://api.nlpcloud.io/v1", APIKey: os.Getenv("NLP_API_KEY")},
		"hyperbolic":  {BaseURL: "https://api.hyperbolic.xyz/v1", APIKey: os.Getenv("HYPERBOLIC_API_KEY")},
		"zai":         {BaseURL: "https://open.bigmodel.cn/api/paas/v4", APIKey: os.Getenv("ZAI_API_KEY")},
		"baseten":     {BaseURL: "https://inference.baseten.co/v1", APIKey: os.Getenv("BASETEN_API_KEY")},
		"twelvelabs":  {BaseURL: "https://api.twelvelabs.io/v1", APIKey: os.Getenv("TWELVELABS_API_KEY")},
		"chutes":      {BaseURL: "https://llm.chutes.ai/v1", APIKey: os.Getenv("CHUTES_API_KEY")},
		"kimi":        {BaseURL: "https://api.moonshot.cn/v1", APIKey: os.Getenv("KIMI_API_KEY")},
		"sarvam":      {BaseURL: "https://api.sarvam.ai/v1", APIKey: os.Getenv("SARVAM_API_KEY")},
		"vulavula":    {BaseURL: "https://api.vulavula.com/v1", APIKey: os.Getenv("VULAVULA_API_KEY")},
		"vercel":      {BaseURL: "https://api.vercel.com/v1", APIKey: os.Getenv("VERCEL_API_KEY")},
		"cohere":      {BaseURL: "https://api.cohere.com/v2", APIKey: os.Getenv("COHERE_API_KEY")},
		"ai21":        {BaseURL: "https://api.ai21.com/studio/v1", APIKey: os.Getenv("AI21_API_KEY")},
		"aleph-alpha": {BaseURL: "https://api.aleph-alpha.com", APIKey: os.Getenv("ALEPH_ALPHA_API_KEY")},
		"writer":      {BaseURL: "https://api.writer.com/v1", APIKey: os.Getenv("WRITER_API_KEY")},
		"gooseai":     {BaseURL: "https://api.goose.ai/v1", APIKey: os.Getenv("GOOSEAI_API_KEY")},
		"stability":   {BaseURL: "https://api.stability.ai/v1", APIKey: os.Getenv("STABILITY_API_KEY")},
		"midjourney":  {BaseURL: "https://api.midjourney.com/v1", APIKey: os.Getenv("MIDJOURNEY_API_KEY")},
		"runway":      {BaseURL: "https://api.runwayml.com/v1", APIKey: os.Getenv("RUNWAY_API_KEY")},
		"elevenlabs":  {BaseURL: "https://api.elevenlabs.io/v1", APIKey: os.Getenv("ELEVENLABS_API_KEY")},
		"assemblyai":  {BaseURL: "https://api.assemblyai.com/v1", APIKey: os.Getenv("ASSEMBLYAI_API_KEY")},
		"gladia":      {BaseURL: "https://api.gladia.io/v1", APIKey: os.Getenv("GLADIA_API_KEY")},
		"fal":         {BaseURL: "https://api.fal.ai/v1", APIKey: os.Getenv("FAL_API_KEY")},
		"relevance":   {BaseURL: "https://api.relevance.ai/v1", APIKey: os.Getenv("RELEVANCE_API_KEY")},
	}
}

// GetModels retrieves models using OpenCode's 3-tier priority system
func (mps *ModelProviderService) GetModels(providerID string) ([]Model, error) {
	mps.logInfo(fmt.Sprintf("Getting models for provider: %s", providerID))

	// Check cache first
	if cached := mps.getFromCache(providerID); cached != nil {
		mps.logInfo(fmt.Sprintf("Cache hit for %s: %d models", providerID, len(cached)))
		return cached, nil
	}

	var models []Model
	var source string

	// Tier 1: Try user configuration first (highest priority)
	mps.logDebug("Tier 1: Checking user configuration")
	if configModels := mps.loadFromConfig(providerID); len(configModels) > 0 {
		models = configModels
		source = "config"
		mps.logInfo(fmt.Sprintf("Found %d models in user config", len(models)))
	}

	// Tier 2: Try provider API if no config models
	if len(models) == 0 {
		mps.logDebug("Tier 2: Checking provider API")
		if apiModels, err := mps.fetchFromProviderAPIEnhanced(providerID); err == nil && len(apiModels) > 0 {
			models = apiModels
			source = "api"
			mps.logInfo(fmt.Sprintf("Fetched %d models from provider API", len(models)))
		}
	}

	// Tier 3: Fall back to models.dev API
	if len(models) == 0 {
		mps.logDebug("Tier 3: Falling back to models.dev")
		if devModels, err := mps.fetchFromModelsDev(providerID); err == nil && len(devModels) > 0 {
			models = devModels
			source = "models.dev"
			mps.logInfo(fmt.Sprintf("Fetched %d models from models.dev", len(models)))
		} else if err != nil {
			mps.logError(fmt.Sprintf("Failed to fetch from models.dev: %v", err))
		}
	}

	// If still no models, return empty list (not error)
	if len(models) == 0 {
		mps.logWarning(fmt.Sprintf("No models found for provider %s", providerID))
		return []Model{}, nil
	}

	// Add source metadata and format display names
	for i := range models {
		models[i].Source = source
		models[i].DisplayName = mps.displayFormatter.FormatWithFeatureSuffixes(
			models[i].Name,
			models[i].Features,
		)
	}

	// Deduplicate models
	models = mps.deduplicateModels(models)

	// Sort models by name
	mps.sortModels(models)

	// Cache the result
	mps.saveToCache(providerID, models)

	mps.logInfo(fmt.Sprintf("Returning %d models for %s (source: %s)",
		len(models), providerID, source))

	return models, nil
}

// GetAllModels retrieves models for all registered providers
func (mps *ModelProviderService) GetAllModels() (map[string][]Model, error) {
	allModels := make(map[string][]Model)
	var wg sync.WaitGroup
	var mu sync.Mutex

	mps.logInfo("Fetching models for all providers")

	// Get a snapshot of provider IDs under read lock
	mps.clientsMutex.RLock()
	providerIDs := make([]string, 0, len(mps.providerClients))
	for providerID := range mps.providerClients {
		providerIDs = append(providerIDs, providerID)
	}
	mps.clientsMutex.RUnlock()

	for _, providerID := range providerIDs {
		wg.Add(1)

		go func(pid string) {
			defer wg.Done()

			models, err := mps.GetModels(pid)
			if err != nil {
				mps.logError(fmt.Sprintf("Failed to get models for %s: %v", pid, err))
				return
			}

			mu.Lock()
			allModels[pid] = models
			mu.Unlock()
		}(providerID)
	}

	wg.Wait()

	mps.logInfo(fmt.Sprintf("Retrieved models for %d providers", len(allModels)))

	return allModels, nil
}

// loadFromConfig loads models from user configuration file
func (mps *ModelProviderService) loadFromConfig(providerID string) []Model {
	// Simple JSON-based config loading for now
	// In a full implementation, this would use the opencode config loader
	configData, err := os.ReadFile(mps.configPath)
	if err != nil {
		mps.logDebug(fmt.Sprintf("No config file or error loading: %v", err))
		return nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		mps.logDebug(fmt.Sprintf("Failed to parse config: %v", err))
		return nil
	}

	// Navigate to provider.models
	if providerMap, ok := config["provider"].(map[string]interface{}); ok {
		if providerData, ok := providerMap[providerID].(map[string]interface{}); ok {
			if modelsData, ok := providerData["models"].(map[string]interface{}); ok {
				var models []Model
				for modelID, modelData := range modelsData {
					if modelMap, ok := modelData.(map[string]interface{}); ok {
						model := mps.parseModelFromData(modelID, modelMap, providerID)
						model.Source = "config"
						models = append(models, model)
					}
				}
				return models
			}
		}
	}

	return nil
}

// fetchFromProviderAPI fetches models directly from provider API
func (mps *ModelProviderService) fetchFromProviderAPI(providerID string) ([]Model, error) {
	mps.clientsMutex.RLock()
	client, exists := mps.providerClients[providerID]
	mps.clientsMutex.RUnlock()
	if !exists {
		return nil, fmt.Errorf("provider %s not registered", providerID)
	}

	if client.APIKey == "" {
		return nil, fmt.Errorf("no API key for provider %s", providerID)
	}

	// Call /v1/models endpoint (OpenAI-compatible)
	url := fmt.Sprintf("%s/models", client.BaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResponse struct {
		Data []struct {
			ID      string `json:"id"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	var models []Model
	for _, data := range apiResponse.Data {
		model := Model{
			ID:           data.ID,
			Name:         data.ID, // Use ID as name initially
			ProviderID:   providerID,
			ProviderName: providerID,
			Features:     make(map[string]interface{}),
		}

		// Enhance model info from models.dev
		if matches, err := mps.modelsDevClient.FindModel(context.Background(), data.ID); err == nil && len(matches) > 0 {
			model = mps.enhanceFromModelsDevMatch(model, &matches[0])
		}

		models = append(models, model)
	}

	return models, nil
}

// fetchFromModelsDev fetches models from models.dev API
func (mps *ModelProviderService) fetchFromModelsDev(providerID string) ([]Model, error) {
	// First try to get all providers and find the one we need
	ctx := context.Background()
	allProviders, err := mps.modelsDevClient.FetchAllProviders(ctx)
	if err != nil {
		return nil, err
	}

	// Find our provider
	providerData, exists := allProviders[providerID]
	if !exists {
		return nil, fmt.Errorf("provider %s not found in models.dev", providerID)
	}

	var models []Model
	for modelID, modelData := range providerData.Models {
		model := Model{
			ID:             modelID,
			Name:           modelData.Name,
			ProviderID:     providerID,
			ProviderName:   providerData.Name,
			Features:       mps.extractFeaturesFromDevModel(&modelData),
			MaxTokens:      int(modelData.Limits.Context),
			IsOpenSource:   modelData.OpenWeights,
			SupportsBrotli: false,
		}

		// Check if model is free (cost = 0)
		if modelData.Cost.Input == 0 && modelData.Cost.Output == 0 {
			model.IsFree = true
			model.CostPer1MInput = 0
			model.CostPer1MOutput = 0
		} else {
			model.CostPer1MInput = modelData.Cost.Input
			model.CostPer1MOutput = modelData.Cost.Output
		}

		// Check for HTTP3 and Toon support
		model.SupportsHTTP3 = modelData.ToolCall
		model.SupportsToon = strings.Contains(strings.ToLower(modelData.Name), "toon")

		models = append(models, model)
	}

	return models, nil
}

// enhanceFromModelsDevMatch enhances a model with rich metadata from models.dev
func (mps *ModelProviderService) enhanceFromModelsDevMatch(model Model, match *verification.ModelMatch) Model {
	model.Name = match.ModelData.Name
	model.MaxTokens = int(match.ModelData.Limits.Context)
	model.CostPer1MInput = match.ModelData.Cost.Input
	model.CostPer1MOutput = match.ModelData.Cost.Output
	model.IsFree = match.ModelData.Cost.Input == 0 && match.ModelData.Cost.Output == 0
	model.SupportsHTTP3 = match.ModelData.ToolCall
	model.SupportsToon = match.ModelData.Reasoning
	model.IsOpenSource = match.ModelData.OpenWeights

	return model
}

// extractFeaturesFromDevModel extracts features from models.dev data
func (mps *ModelProviderService) extractFeaturesFromDevModel(modelData *verification.ModelDetails) map[string]interface{} {
	features := make(map[string]interface{})

	features["tool_call"] = modelData.ToolCall
	features["reasoning"] = modelData.Reasoning
	features["attachment"] = modelData.Attachment
	features["structured_output"] = modelData.StructuredOutput
	features["open_weights"] = modelData.OpenWeights
	features["modalities"] = modelData.Modalities
	features["knowledge"] = modelData.Knowledge

	return features
}

// parseModelFromData parses model from generic map data
func (mps *ModelProviderService) parseModelFromData(modelID string, modelData map[string]interface{}, providerID string) Model {
	model := Model{
		ID:           modelID,
		ProviderID:   providerID,
		ProviderName: providerID,
		Features:     make(map[string]interface{}),
	}

	// Extract name
	if name, ok := modelData["name"].(string); ok {
		model.Name = name
	} else {
		model.Name = modelID
	}

	// Extract max tokens
	if maxTokens, ok := modelData["maxTokens"].(float64); ok {
		model.MaxTokens = int(maxTokens)
	}

	// Extract cost
	if cost, ok := modelData["cost"].(map[string]interface{}); ok {
		if input, ok := cost["input"].(float64); ok {
			model.CostPer1MInput = input
		}
		if output, ok := cost["output"].(float64); ok {
			model.CostPer1MOutput = output
		}
		model.IsFree = model.CostPer1MInput == 0 && model.CostPer1MOutput == 0
	}

	// Extract boolean features
	model.SupportsBrotli = getBoolFeature(modelData, "supports_brotli")
	model.SupportsHTTP3 = getBoolFeature(modelData, "supports_http3")
	model.SupportsToon = getBoolFeature(modelData, "supports_toon")
	model.IsOpenSource = getBoolFeature(modelData, "open_weights")

	// Extract response time (if present)
	if responseTime, ok := modelData["response_time_ms"].(float64); ok {
		model.ResponseTimeMS = responseTime
	}

	return model
}

// getBoolFeature safely extracts boolean feature from map
func getBoolFeature(data map[string]interface{}, key string) bool {
	if val, ok := data[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return false
}

// deduplicateModels removes duplicate models based on ID
func (mps *ModelProviderService) deduplicateModels(models []Model) []Model {
	seen := make(map[string]bool)
	unique := make([]Model, 0, len(models))

	for _, model := range models {
		key := fmt.Sprintf("%s:%s", model.ProviderID, model.ID)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, model)
		}
	}

	if len(unique) < len(models) {
		mps.logDebug(fmt.Sprintf("Deduplicated %d models to %d", len(models), len(unique)))
	}

	return unique
}

// sortModels sorts models by name
func (mps *ModelProviderService) sortModels(models []Model) {
	sort.Slice(models, func(i, j int) bool {
		return strings.ToLower(models[i].Name) < strings.ToLower(models[j].Name)
	})
}

// Cache operations

// getFromCache retrieves models from cache if not expired
func (mps *ModelProviderService) getFromCache(providerID string) []Model {
	mps.cacheMutex.RLock()
	defer mps.cacheMutex.RUnlock()

	entry, exists := mps.cache[providerID]
	if !exists {
		return nil
	}

	// Check cache expiration (24 hours)
	cacheAge := time.Since(entry.timestamp)
	cacheDuration := 24 * time.Hour

	if cacheAge > cacheDuration {
		mps.logDebug(fmt.Sprintf("Cache expired for %s (age: %v, TTL: %v)", providerID, cacheAge.Round(time.Minute), cacheDuration))
		delete(mps.cache, providerID)
		return nil
	}

	mps.logDebug(fmt.Sprintf("Cache hit for %s (age: %v)", providerID, cacheAge.Round(time.Minute)))
	return entry.models
}

// saveToCache stores models in cache
func (mps *ModelProviderService) saveToCache(providerID string, models []Model) {
	mps.cacheMutex.Lock()
	defer mps.cacheMutex.Unlock()

	mps.cache[providerID] = &providerCacheEntry{
		models:     models,
		timestamp:  time.Now(),
		providerID: providerID,
	}
}

// ClearCache clears the provider cache
func (mps *ModelProviderService) ClearCache() {
	mps.cacheMutex.Lock()
	defer mps.cacheMutex.Unlock()

	mps.cache = make(map[string]*providerCacheEntry)
	mps.logInfo("Cleared provider cache")
}

// RefreshCache refreshes cache for all providers
func (mps *ModelProviderService) RefreshCache() error {
	mps.logInfo("Refreshing cache for all providers")

	mps.ClearCache()

	_, err := mps.GetAllModels()
	return err
}

// GetAllProviders returns all registered providers
// Note: Returns a copy of the map to prevent concurrent modification issues
func (mps *ModelProviderService) GetAllProviders() map[string]*ProviderClient {
	mps.clientsMutex.RLock()
	defer mps.clientsMutex.RUnlock()

	// Return a copy to prevent concurrent modification
	result := make(map[string]*ProviderClient, len(mps.providerClients))
	for k, v := range mps.providerClients {
		result[k] = v
	}
	return result
}

// Enhanced fetchFromProviderAPI that uses provider-specific adapters
func (mps *ModelProviderService) fetchFromProviderAPIEnhanced(providerID string) ([]Model, error) {
	mps.clientsMutex.RLock()
	client, exists := mps.providerClients[providerID]
	mps.clientsMutex.RUnlock()
	if !exists {
		return nil, fmt.Errorf("provider %s not registered", providerID)
	}

	if client.APIKey == "" {
		return nil, fmt.Errorf("no API key for provider %s", providerID)
	}

	// Create provider-specific adapter based on provider ID
	var models []Model
	var err error

	switch providerID {
	case "openai":
		adapter := NewOpenAIAdapter(client.HTTPClient, client.BaseURL, client.APIKey)
		models, err = mps.fetchModelsFromAdapter(adapter, providerID)
	case "anthropic":
		adapter := NewAnthropicAdapter(client.HTTPClient, client.BaseURL, client.APIKey)
		models, err = mps.fetchModelsFromAdapter(adapter, providerID)
	case "groq":
		adapter := NewGroqAdapter(client.HTTPClient, client.BaseURL, client.APIKey)
		models, err = mps.fetchModelsFromAdapter(adapter, providerID)
	case "mistral":
		adapter := NewMistralAdapter(client.HTTPClient, client.BaseURL, client.APIKey)
		models, err = mps.fetchModelsFromAdapter(adapter, providerID)
	case "deepseek":
		adapter := NewDeepSeekAdapter(client.HTTPClient, client.BaseURL, client.APIKey)
		models, err = mps.fetchModelsFromAdapter(adapter, providerID)
	case "together":
		adapter := NewTogetherAIAdapter(client.HTTPClient, client.BaseURL, client.APIKey)
		models, err = mps.fetchModelsFromAdapter(adapter, providerID)
	case "replicate":
		adapter := NewReplicateAdapter(client.HTTPClient, client.BaseURL, client.APIKey)
		models, err = mps.fetchModelsFromAdapter(adapter, providerID)
	case "cloudflare":
		adapter := NewCloudflareAdapter(client.HTTPClient, client.BaseURL, client.APIKey)
		models, err = mps.fetchModelsFromAdapter(adapter, providerID)
	case "cerebras":
		adapter := NewCerebrasAdapter(client.HTTPClient, client.BaseURL, client.APIKey)
		models, err = mps.fetchModelsFromAdapter(adapter, providerID)
	case "cohere":
		adapter := NewCohereAdapter(client.HTTPClient, client.BaseURL, client.APIKey)
		models, err = mps.fetchModelsFromAdapter(adapter, providerID)
	case "siliconflow":
		adapter := NewSiliconFlowAdapter(client.HTTPClient, client.BaseURL, client.APIKey)
		models, err = mps.fetchModelsFromAdapter(adapter, providerID)
	case "xai":
		adapter := NewxAIAdapter(client.HTTPClient, client.BaseURL, client.APIKey)
		models, err = mps.fetchModelsFromAdapter(adapter, providerID)
	default:
		// Fall back to generic OpenAI-compatible approach
		mps.logDebug(fmt.Sprintf("No specific adapter for %s, using generic OpenAI approach", providerID))
		return mps.fetchFromProviderAPIGeneric(providerID)
	}

	if err != nil {
		return nil, err
	}

	return models, nil
}

// fetchModelsFromAdapter fetches models using a provider-specific adapter
func (mps *ModelProviderService) fetchModelsFromAdapter(adapter interface{}, providerID string) ([]Model, error) {
	ctx := context.Background()

	// Try to call ListModels method on the adapter
	type ListModelsInterface interface {
		ListModels(ctx context.Context) (*OpenAIModelsResponse, error)
	}

	if listModelsAdapter, ok := adapter.(ListModelsInterface); ok {
		resp, err := listModelsAdapter.ListModels(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list models for %s: %w", providerID, err)
		}

		var models []Model
		for _, data := range resp.Data {
			model := Model{
				ID:           data.ID,
				Name:         data.ID, // Use ID as name initially
				ProviderID:   providerID,
				ProviderName: providerID,
				Features:     make(map[string]interface{}),
			}

			// Enhance model info from models.dev
			if matches, err := mps.modelsDevClient.FindModel(context.Background(), data.ID); err == nil && len(matches) > 0 {
				model = mps.enhanceFromModelsDevMatch(model, &matches[0])
			}

			models = append(models, model)
		}

		mps.logInfo(fmt.Sprintf("Fetched %d models from %s API", len(models), providerID))
		return models, nil
	}

	return nil, fmt.Errorf("adapter for %s does not support ListModels", providerID)
}

// fetchFromProviderAPIGeneric uses the original generic approach for providers without specific adapters
func (mps *ModelProviderService) fetchFromProviderAPIGeneric(providerID string) ([]Model, error) {
	mps.clientsMutex.RLock()
	client, exists := mps.providerClients[providerID]
	mps.clientsMutex.RUnlock()
	if !exists {
		return nil, fmt.Errorf("provider %s not registered", providerID)
	}

	if client.APIKey == "" {
		return nil, fmt.Errorf("no API key for provider %s", providerID)
	}

	// Call /v1/models endpoint (OpenAI-compatible)
	url := fmt.Sprintf("%s/models", client.BaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Don't treat this as a fatal error - many providers don't support /v1/models
		mps.logDebug(fmt.Sprintf("Provider %s returned status %d from /models endpoint", providerID, resp.StatusCode))
		return []Model{}, nil
	}

	var apiResponse struct {
		Data []struct {
			ID      string `json:"id"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		mps.logDebug(fmt.Sprintf("Failed to decode response from %s: %v", providerID, err))
		return []Model{}, nil
	}

	var models []Model
	for _, data := range apiResponse.Data {
		model := Model{
			ID:           data.ID,
			Name:         data.ID, // Use ID as name initially
			ProviderID:   providerID,
			ProviderName: providerID,
			Features:     make(map[string]interface{}),
		}

		// Enhance model info from models.dev
		if matches, err := mps.modelsDevClient.FindModel(context.Background(), data.ID); err == nil && len(matches) > 0 {
			model = mps.enhanceFromModelsDevMatch(model, &matches[0])
		}

		models = append(models, model)
	}

	return models, nil
}
