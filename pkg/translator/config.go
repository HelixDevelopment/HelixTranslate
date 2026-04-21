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
	Options     map[string]interface{}
}
