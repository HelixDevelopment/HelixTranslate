package llmverifier

import "time"

// VerificationResult stores the results of verifying a single LLM
type VerificationResult struct {
	ModelInfo              ModelInfo                  `json:"model_info"`
	Availability           AvailabilityResult         `json:"availability"`
	ResponseTime           ResponseTimeResult         `json:"response_time"`
	FeatureDetection       FeatureDetectionResult     `json:"feature_detection"`
	CodeCapabilities       CodeCapabilityResult       `json:"code_capabilities"`
	GenerativeCapabilities GenerativeCapabilityResult `json:"generative_capabilities,omitempty"`
	PerformanceScores      PerformanceScore           `json:"performance_scores"`
	Timestamp              time.Time                  `json:"timestamp"`
	Error                  string                     `json:"error,omitempty"`
	ScoreDetails           ScoreDetails               `json:"score_details"`
}

// ModelInfo contains basic information about the model
type ModelInfo struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	OwnedBy           string         `json:"owned_by"`
	Root              string         `json:"root,omitempty"`
	Parent            string         `json:"parent,omitempty"`
	Permissions       []Permission   `json:"permissions,omitempty"`
	ScalingPolicy     *ScalingPolicy `json:"scaling_policy,omitempty"`
	Capabilities      Capabilities   `json:"capabilities,omitempty"`
	ContextWindow     ContextWindow  `json:"context_window,omitempty"`
	MaxOutputTokens   int            `json:"max_output_tokens,omitempty"`
	InputPrices       InputPrices    `json:"input_prices,omitempty"`
	OutputPrices      OutputPrices   `json:"output_prices,omitempty"`
	HasTrainingData   bool           `json:"has_training_data,omitempty"`
	Description       string         `json:"description,omitempty"`
	Architecture      Architecture   `json:"architecture,omitempty"`
	Tokenizer         string         `json:"tokenizer,omitempty"`
	Organization      string         `json:"organization,omitempty"`
	ReleaseDate       string         `json:"release_date,omitempty"`
	LanguageSupport   []string       `json:"language_support,omitempty"`
	UseCase           string         `json:"use_case,omitempty"`
	Version           string         `json:"version,omitempty"`
	MaxInputTokens    int            `json:"max_input_tokens,omitempty"`
	SupportsVision    bool           `json:"supports_vision,omitempty"`
	SupportsAudio     bool           `json:"supports_audio,omitempty"`
	SupportsVideo     bool           `json:"supports_video,omitempty"`
	SupportsReasoning bool           `json:"supports_reasoning,omitempty"`
	SupportsHTTP3     bool           `json:"supports_http3,omitempty"`
	SupportsToon      bool           `json:"supports_toon,omitempty"`
	SupportsBrotli    bool           `json:"supports_brotli,omitempty"`
	OpenSource        bool           `json:"open_source,omitempty"`
	Deprecated        bool           `json:"deprecated,omitempty"`
	Tags              []string       `json:"tags,omitempty"`
	Endpoint          string         `json:"endpoint"`
}

type Permission struct {
	ID                 string `json:"id"`
	Object             string `json:"object"`
	Created            int64  `json:"created"`
	AllowCreate_engine bool   `json:"allow_create_engine"`
	AllowSampling      bool   `json:"allow_sampling"`
	AllowLogprobs      bool   `json:"allow_logprobs"`
	AllowSearchIndices bool   `json:"allow_search_indices"`
	AllowView          bool   `json:"allow_view"`
	AllowFineTuning    bool   `json:"allow_fine_tuning"`
	Organization       string `json:"organization"`
	Group              string `json:"group,omitempty"`
	IsBlocking         bool   `json:"is_blocking"`
	Type               string `json:"type"`
}

type ScalingPolicy struct {
	MaxBatchSize        int `json:"max_batch_size"`
	MaxRequestPerWindow int `json:"max_requests_per_window"`
	TimeWindowSeconds   int `json:"time_window_seconds"`
}

type Capabilities struct {
	Completion      bool `json:"completion"`
	Chat            bool `json:"chat"`
	Embedding       bool `json:"embedding"`
	FineTuning      bool `json:"fine_tuning"`
	ImageGeneration bool `json:"image_generation"`
	CodeGeneration  bool `json:"code_generation"`
	ToolUse         bool `json:"tool_use"`
	Multimodal      bool `json:"multimodal"`
	FunctionCalling bool `json:"function_calling"`
	Voice           bool `json:"voice"`
	Rerank          bool `json:"rerank"`
}

type ContextWindow struct {
	MaxInputTokens  int `json:"max_input_tokens"`
	MaxOutputTokens int `json:"max_output_tokens"`
	TotalMaxTokens  int `json:"total_max_tokens"`
}

type InputPrices struct {
	Prompt string `json:"prompt"`
	Cached string `json:"cached,omitempty"`
}

type OutputPrices struct {
	Completion string `json:"completion"`
	Storage    string `json:"storage,omitempty"`
	Requests   string `json:"requests,omitempty"`
}

type Architecture struct {
	ReasoningEfficiency float64 `json:"reasoning_efficiency"`
	CoTTokenEfficiency  float64 `json:"cot_token_efficiency"`
	NumParameters       float64 `json:"num_parameters,omitempty"`
	ArchitectureType    string  `json:"architecture_type,omitempty"`
}

// AvailabilityResult captures whether the model is available and responsive
type AvailabilityResult struct {
	Exists      bool          `json:"exists"`
	Responsive  bool          `json:"responsive"`
	Overloaded  bool          `json:"overloaded"`
	Latency     time.Duration `json:"latency"`
	LastChecked time.Time     `json:"last_checked"`
	Error       string        `json:"error,omitempty"`
}

// ResponseTimeResult captures response time measurements
type ResponseTimeResult struct {
	AverageLatency   time.Duration `json:"average_latency"`
	P95Latency       time.Duration `json:"p95_latency"`
	MinLatency       time.Duration `json:"min_latency"`
	MaxLatency       time.Duration `json:"max_latency"`
	Throughput       float64       `json:"throughput"` // Requests per second
	MeasurementCount int           `json:"measurement_count"`
}

// FeatureDetectionResult captures the features supported by the model
type FeatureDetectionResult struct {
	ToolUse          bool                 `json:"tool_use"`
	Functions        []FunctionDefinition `json:"functions"`
	CodeGeneration   bool                 `json:"code_generation"`
	CodeCompletion   bool                 `json:"code_completion"`
	CodeReview       bool                 `json:"code_review"`
	CodeExplanation  bool                 `json:"code_explanation"`
	Embeddings       bool                 `json:"embeddings"`
	Reranking        bool                 `json:"reranking"`
	ImageGeneration  bool                 `json:"image_generation"`
	AudioGeneration  bool                 `json:"audio_generation"`
	VideoGeneration  bool                 `json:"video_generation"`
	MCPs             bool                 `json:"mcps"`
	LSPs             bool                 `json:"lsps"`
	ACPs             bool                 `json:"acps"`
	Multimodal       bool                 `json:"multimodal"`
	Streaming        bool                 `json:"streaming"`
	JSONMode         bool                 `json:"json_mode"`
	StructuredOutput bool                 `json:"structured_output"`
	Reasoning        bool                 `json:"reasoning"`
	FunctionCalling  bool                 `json:"function_calling"`
	ParallelToolUse  bool                 `json:"parallel_tool_use"`
	MaxParallelCalls int                  `json:"max_parallel_calls"`
	Modalities       []string             `json:"modalities"`
	BatchProcessing  bool                 `json:"batch_processing"`
	SupportsBrotli   bool                 `json:"supports_brotli"`
	SupportsHTTP3    bool                 `json:"supports_http3"`
	SupportsToon     bool                 `json:"supports_toon"`
}

type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// GenerativeCapabilityResult captures the creative/generative abilities of the model
type GenerativeCapabilityResult struct {
	CreativeWriting      bool    `json:"creative_writing"`
	Storytelling         bool    `json:"storytelling"`
	ContentGeneration    bool    `json:"content_generation"`
	ArtisticCreativity   bool    `json:"artistic_creativity"`
	ProblemSolving       bool    `json:"problem_solving"`
	MultimodalGenerative bool    `json:"multimodal_generative"`
	OriginalityScore     float64 `json:"originality_score"`
	CreativityScore      float64 `json:"creativity_score"`
}

// CodeCapabilityResult captures the coding abilities of the model
type CodeCapabilityResult struct {
	LanguageSupport    []string           `json:"language_support"`
	CodeGeneration     bool               `json:"code_generation"`
	CodeCompletion     bool               `json:"code_completion"`
	CodeDebugging      bool               `json:"code_debugging"`
	CodeOptimization   bool               `json:"code_optimization"`
	CodeReview         bool               `json:"code_review"`
	CodeExplanation    bool               `json:"code_explanation"`
	TestGeneration     bool               `json:"test_generation"`
	Documentation      bool               `json:"documentation"`
	Refactoring        bool               `json:"refactoring"`
	ErrorResolution    bool               `json:"error_resolution"`
	Architecture       bool               `json:"architecture"`
	SecurityAssessment bool               `json:"security_assessment"`
	PatternRecognition bool               `json:"pattern_recognition"`
	DebuggingAccuracy  float64            `json:"debugging_accuracy"`
	ComplexityHandling ComplexityMetrics  `json:"complexity_handling"`
	PromptResponse     PromptResponseTest `json:"prompt_response"`
}

type ComplexityMetrics struct {
	MaxHandledDepth   int     `json:"max_handled_depth"`
	MaxTokens         int     `json:"max_tokens"`
	CodeQuality       float64 `json:"code_quality"`
	LogicCorrectness  float64 `json:"logic_correctness"`
	RuntimeEfficiency float64 `json:"runtime_efficiency"`
}

type PromptResponseTest struct {
	PythonSuccessRate     float64       `json:"python_success_rate"`
	JavascriptSuccessRate float64       `json:"javascript_success_rate"`
	GoSuccessRate         float64       `json:"go_success_rate"`
	JavaSuccessRate       float64       `json:"java_success_rate"`
	CppSuccessRate        float64       `json:"cpp_success_rate"`
	TypescriptSuccessRate float64       `json:"typescript_success_rate"`
	OverallSuccessRate    float64       `json:"overall_success_rate"`
	AvgResponseTime       time.Duration `json:"avg_response_time"`
}

// PerformanceScore contains the calculated scores for the model
type PerformanceScore struct {
	OverallScore     float64 `json:"overall_score"`
	CodeCapability   float64 `json:"code_capability"`
	Responsiveness   float64 `json:"responsiveness"`
	Reliability      float64 `json:"reliability"`
	FeatureRichness  float64 `json:"feature_richness"`
	ValueProposition float64 `json:"value_proposition"`
}

// ScoreDetails provides detailed breakdown of how scores were calculated
type ScoreDetails struct {
	CodeCapabilityBreakdown CodeCapabilityBreakdown `json:"code_capability_breakdown"`
	ResponseTimeBreakdown   ResponseTimeBreakdown   `json:"response_time_breakdown"`
	FeatureSupportBreakdown FeatureSupportBreakdown `json:"feature_support_breakdown"`
	ReliabilityBreakdown    ReliabilityBreakdown    `json:"reliability_breakdown"`
}

type CodeCapabilityBreakdown struct {
	GenerationScore    float64 `json:"generation_score"`
	CompletionScore    float64 `json:"completion_score"`
	DebuggingScore     float64 `json:"debugging_score"`
	ReviewScore        float64 `json:"review_score"`
	TestGenScore       float64 `json:"test_gen_score"`
	DocumentScore      float64 `json:"document_score"`
	ArchitectureScore  float64 `json:"architecture_score"`
	OptimizationScore  float64 `json:"optimization_score"`
	ComplexityHandling float64 `json:"complexity_handling"`
	WeightedAverage    float64 `json:"weighted_average"`
}

type ResponseTimeBreakdown struct {
	LatencyScore     float64 `json:"latency_score"`
	ThroughputScore  float64 `json:"throughput_score"`
	ConsistencyScore float64 `json:"consistency_score"`
	WeightedAverage  float64 `json:"weighted_average"`
}

type FeatureSupportBreakdown struct {
	CoreFeaturesScore         float64 `json:"core_features_score"`
	AdvancedFeaturesScore     float64 `json:"advanced_features_score"`
	ExperimentalFeaturesScore float64 `json:"experimental_features_score"`
	WeightedAverage           float64 `json:"weighted_average"`
}

type ReliabilityBreakdown struct {
	AvailabilityScore float64 `json:"availability_score"`
	ConsistencyScore  float64 `json:"consistency_score"`
	ErrorRateScore    float64 `json:"error_rate_score"`
	StabilityScore    float64 `json:"stability_score"`
	WeightedAverage   float64 `json:"weighted_average"`
}

// Summary represents the summary of all verification results
type Summary struct {
	TotalModels       int              `json:"total_models"`
	AvailableModels   int              `json:"available_models"`
	FailedModels      int              `json:"failed_models"`
	StartTime         time.Time        `json:"start_time"`
	EndTime           time.Time        `json:"end_time"`
	Duration          time.Duration    `json:"duration"`
	AverageScore      float64          `json:"average_score"`
	BrotliSupportRate float64          `json:"brotli_support_rate"`
	TopPerformers     []TopPerformer   `json:"top_performers"`
	CategoryRankings  CategoryRankings `json:"category_rankings"`
}

type TopPerformer struct {
	ModelName string  `json:"model_name"`
	Score     float64 `json:"score"`
	Rank      int     `json:"rank"`
}

type CategoryRankings struct {
	ByCodeCapability  []TopPerformer `json:"by_code_capability"`
	ByResponsiveness  []TopPerformer `json:"by_responsiveness"`
	ByReliability     []TopPerformer `json:"by_reliability"`
	ByFeatureRichness []TopPerformer `json:"by_feature_richness"`
	ByValue           []TopPerformer `json:"by_value"`
}
