package translator

import "time"

// TranslationConfig holds translation configuration (moved to avoid import cycle)
type TranslationConfig struct {
	SourceLang  string
	TargetLang  string
	Provider    string
	Model       string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
	APIKey      string
	BaseURL     string
	Script      string // Script type (cyrillic, latin)
	ChunkSize   int    // Text chunk size for splitting (0 = use default 20000)
	Options     map[string]interface{}
}
