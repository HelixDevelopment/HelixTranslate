package preparation

import "time"

// ContentAnalysis represents the complete analysis of content to be translated
type ContentAnalysis struct {
	// Content classification
	ContentType    string   `json:"content_type"`    // Novel, poem, technical, legal, medical, etc.
	Genre          string   `json:"genre"`           // Main genre
	Subgenres      []string `json:"subgenres"`       // Detailed subgenres

	// Language and style
	Tone           string   `json:"tone"`            // Formal, informal, poetic, technical, etc.
	LanguageStyle  string   `json:"language_style"`  // Literary devices, sentence structure, etc.
	TargetAudience string   `json:"target_audience"` // Expected reader demographics

	// Translation-specific guidance
	UntranslatableTerms []UntranslatableTerm `json:"untranslatable_terms"`
	FootnoteGuidance    []FootnoteGuidance   `json:"footnote_guidance"`
	Characters          []Character          `json:"characters"`
	KeyThemes           []string             `json:"key_themes"`
	CulturalReferences  []CulturalReference  `json:"cultural_references"`

	// Chapter/Section analysis
	ChapterAnalyses []ChapterAnalysis `json:"chapter_analyses"`

	// Meta information
	AnalysisVersion int       `json:"analysis_version"` // Which pass produced this
	AnalyzedAt      time.Time `json:"analyzed_at"`
	AnalyzedBy      string    `json:"analyzed_by"` // Which LLM provider
}

// UntranslatableTerm represents a term that should be kept in original language
type UntranslatableTerm struct {
	Term           string   `json:"term"`
	OriginalScript string   `json:"original_script"` // Cyrillic, Latin, etc.
	Reason         string   `json:"reason"`          // Why it shouldn't be translated
	Context        []string `json:"context"`         // Where it appears
	Transliteration string  `json:"transliteration"` // Optional transliteration
}

// FootnoteGuidance suggests where footnotes/clarifications are needed
type FootnoteGuidance struct {
	Term        string   `json:"term"`
	Explanation string   `json:"explanation"`
	Locations   []string `json:"locations"` // Chapter/section references
	Priority    string   `json:"priority"`  // High, medium, low
}

// Character represents a character in the narrative
type Character struct {
	Name            string            `json:"name"`
	AlternateNames  []string          `json:"alternate_names"`
	Role            string            `json:"role"`            // Protagonist, antagonist, etc.
	SpeechPattern   string            `json:"speech_pattern"`  // Dialect, formality, quirks
	KeyTraits       []string          `json:"key_traits"`
	NameTranslation map[string]string `json:"name_translation"` // How name should be handled per language
}

// CulturalReference represents a culture-specific reference
type CulturalReference struct {
	Reference   string `json:"reference"`
	Origin      string `json:"origin"`      // Culture/country
	Explanation string `json:"explanation"` // What it means
	Handling    string `json:"handling"`    // Keep, translate, add footnote, etc.
}

// ChapterAnalysis contains analysis for a specific chapter/section
type ChapterAnalysis struct {
	ChapterID   string   `json:"chapter_id"`
	ChapterNum  int      `json:"chapter_num"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	KeyPoints   []string `json:"key_points"`
	Caveats     []string `json:"caveats"`     // Translation warnings
	Tone        string   `json:"tone"`        // Chapter-specific tone
	Complexity  string   `json:"complexity"`  // Simple, moderate, complex
	SpecialNotes string  `json:"special_notes"` // Any other important notes
}

// PreparationPass represents a single analysis pass
type PreparationPass struct {
	PassNumber int              `json:"pass_number"`
	Provider   string           `json:"provider"`
	Model      string           `json:"model"`
	Analysis   ContentAnalysis  `json:"analysis"`
	Duration   time.Duration    `json:"duration"`
	TokensUsed int              `json:"tokens_used"`
}

// PreparationResult represents the final preparation output
type PreparationResult struct {
	SourceLanguage string            `json:"source_language"`
	TargetLanguage string            `json:"target_language"`

	// Multiple passes of analysis
	Passes []PreparationPass `json:"passes"`

	// Consolidated final analysis
	FinalAnalysis ContentAnalysis `json:"final_analysis"`

	// Statistics
	TotalDuration  time.Duration `json:"total_duration"`
	TotalTokens    int           `json:"total_tokens"`
	PassCount      int           `json:"pass_count"`

	// Meta
	StartedAt  time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// PreparationConfig configures the preparation phase
type PreparationConfig struct {
	// Multi-pass configuration
	PassCount      int      `json:"pass_count"`       // Number of analysis passes
	Providers      []string `json:"providers"`        // LLM providers to use

	// Analysis focus
	AnalyzeContentType  bool `json:"analyze_content_type"`
	AnalyzeCharacters   bool `json:"analyze_characters"`
	AnalyzeTerminology  bool `json:"analyze_terminology"`
	AnalyzeCulture      bool `json:"analyze_culture"`
	AnalyzeChapters     bool `json:"analyze_chapters"`

	// Depth control
	DetailLevel string `json:"detail_level"` // Basic, standard, comprehensive

	// Languages
	SourceLanguage string `json:"source_language"`
	TargetLanguage string `json:"target_language"`
	
	// API configuration
	APIKey        string `json:"api_key"`        // API key for LLM providers
}
