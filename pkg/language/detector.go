package language

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

// Language represents a language with its codes
type Language struct {
	Code   string // ISO 639-1 code (e.g., "en", "ru", "sr")
	Name   string // English name (e.g., "English", "Russian")
	Native string // Native name (e.g., "English", "Русский")
}

// Common languages
var (
	English    = Language{Code: "en", Name: "English", Native: "English"}
	Russian    = Language{Code: "ru", Name: "Russian", Native: "Русский"}
	Serbian    = Language{Code: "sr", Name: "Serbian", Native: "Српски"}
	German     = Language{Code: "de", Name: "German", Native: "Deutsch"}
	French     = Language{Code: "fr", Name: "French", Native: "Français"}
	Spanish    = Language{Code: "es", Name: "Spanish", Native: "Español"}
	Italian    = Language{Code: "it", Name: "Italian", Native: "Italiano"}
	Portuguese = Language{Code: "pt", Name: "Portuguese", Native: "Português"}
	Chinese    = Language{Code: "zh", Name: "Chinese", Native: "中文"}
	Japanese   = Language{Code: "ja", Name: "Japanese", Native: "日本語"}
	Korean     = Language{Code: "ko", Name: "Korean", Native: "한국어"}
	Arabic     = Language{Code: "ar", Name: "Arabic", Native: "العربية"}
	Polish     = Language{Code: "pl", Name: "Polish", Native: "Polski"}
	Ukrainian  = Language{Code: "uk", Name: "Ukrainian", Native: "Українська"}
	Czech      = Language{Code: "cs", Name: "Czech", Native: "Čeština"}
	Slovak     = Language{Code: "sk", Name: "Slovak", Native: "Slovenčina"}
	Croatian   = Language{Code: "hr", Name: "Croatian", Native: "Hrvatski"}
	Bulgarian  = Language{Code: "bg", Name: "Bulgarian", Native: "Български"}
)

// languageMap maps codes and names to Language structs
var languageMap = map[string]Language{
	// Codes
	"en": English, "eng": English,
	"ru": Russian, "rus": Russian,
	"sr": Serbian, "srp": Serbian,
	"de": German, "deu": German, "ger": German,
	"fr": French, "fra": French, "fre": French,
	"es": Spanish, "spa": Spanish,
	"it": Italian, "ita": Italian,
	"pt": Portuguese, "por": Portuguese,
	"zh": Chinese, "zho": Chinese, "chi": Chinese,
	"ja": Japanese, "jpn": Japanese,
	"ko": Korean, "kor": Korean,
	"ar": Arabic, "ara": Arabic,
	"pl": Polish, "pol": Polish,
	"uk": Ukrainian, "ukr": Ukrainian,
	"cs": Czech, "ces": Czech, "cze": Czech,
	"sk": Slovak, "slk": Slovak, "slo": Slovak,
	"hr": Croatian, "hrv": Croatian,
	"bg": Bulgarian, "bul": Bulgarian,

	// Names (lowercase)
	"english":    English,
	"russian":    Russian,
	"serbian":    Serbian,
	"german":     German,
	"french":     French,
	"spanish":    Spanish,
	"italian":    Italian,
	"portuguese": Portuguese,
	"chinese":    Chinese,
	"japanese":   Japanese,
	"korean":     Korean,
	"arabic":     Arabic,
	"polish":     Polish,
	"ukrainian":  Ukrainian,
	"czech":      Czech,
	"slovak":     Slovak,
	"croatian":   Croatian,
	"bulgarian":  Bulgarian,
}

// Detector handles language detection
type Detector struct {
	llmDetector LLMDetector
}

// LLMDetector interface for LLM-based language detection
type LLMDetector interface {
	DetectLanguage(ctx context.Context, text string) (string, error)
}

// NewDetector creates a new language detector
func NewDetector(llmDetector LLMDetector) *Detector {
	return &Detector{
		llmDetector: llmDetector,
	}
}

// Detect detects the language of the given text
func (d *Detector) Detect(ctx context.Context, text string) (Language, error) {
	// Try LLM detection first if available
	if d.llmDetector != nil {
		code, err := d.llmDetector.DetectLanguage(ctx, text)
		if err == nil && code != "" {
			if lang, ok := languageMap[strings.ToLower(code)]; ok {
				return lang, nil
			}
		}
	}

	// Fallback to heuristic detection
	return d.detectHeuristic(text), nil
}

// detectHeuristic uses character-based heuristics to detect language
func (d *Detector) detectHeuristic(text string) Language {
	if text == "" {
		return English // default
	}

	// Sample first 1000 characters
	sample := text
	if len(text) > 1000 {
		sample = text[:1000]
	}

	// Count character types
	var (
		cyrillic int
		latin    int
		cjk      int
		arabic   int
	)

	for _, r := range sample {
		switch {
		case isCyrillic(r):
			cyrillic++
		case isLatin(r):
			latin++
		case isCJK(r):
			cjk++
		case isArabic(r):
			arabic++
		}
	}

	// Determine language by character frequency
	total := cyrillic + latin + cjk + arabic
	if total == 0 {
		return English // default
	}

	// CJK languages
	if float64(cjk)/float64(total) > 0.3 {
		// Could be Chinese, Japanese, or Korean
		// Try to distinguish based on specific characters
		return d.detectCJKLanguage(sample)
	}

	// Arabic
	if float64(arabic)/float64(total) > 0.3 {
		return Arabic
	}

	// Cyrillic scripts
	if float64(cyrillic)/float64(total) > 0.3 {
		// Try to distinguish between Russian, Serbian, Ukrainian, etc.
		return d.detectCyrillicLanguage(sample)
	}

	// Latin scripts - try to distinguish between languages
	return d.detectLatinLanguage(sample)
}

// detectCyrillicLanguage distinguishes between Cyrillic languages
func (d *Detector) detectCyrillicLanguage(text string) Language {
	// Count language-specific characters
	var (
		russianChars  int
		serbianChars  int
		ukrainianChars int
		bulgarianChars int
	)

	for _, r := range text {
		switch r {
		case 'Ё', 'ё', 'Ы', 'ы', 'Э', 'э':
			russianChars++
		case 'Ђ', 'ђ', 'Ћ', 'ћ', 'Љ', 'љ', 'Њ', 'њ', 'Џ', 'џ':
			serbianChars++
		case 'Є', 'є', 'І', 'і', 'Ї', 'ї', 'Ґ', 'ґ':
			ukrainianChars++
		case 'Ъ', 'ъ', 'Щ', 'щ', 'Ь', 'ь':
			bulgarianChars++
		}
	}

	// Return language with most specific characters
	if serbianChars > russianChars && serbianChars > ukrainianChars && serbianChars > bulgarianChars {
		return Serbian
	}
	if ukrainianChars > russianChars && ukrainianChars > serbianChars && ukrainianChars > bulgarianChars {
		return Ukrainian
	}
	if bulgarianChars > russianChars && bulgarianChars > serbianChars && bulgarianChars > ukrainianChars {
		return Bulgarian
	}

	// Default to Russian for Cyrillic
	return Russian
}

// detectLatinLanguage distinguishes between Latin-based languages
func (d *Detector) detectLatinLanguage(text string) Language {
	// Count language-specific characters and words
	var (
		spanishChars   int
		frenchChars    int
		germanChars    int
		italianChars   int
		portugueseChars int
		polishChars    int
		czechChars     int
		slovakChars    int
		croatianChars  int
	)

	// Convert to lowercase for word matching
	lowerText := strings.ToLower(text)

	// Check for language-specific characters
	for _, r := range lowerText {
		switch r {
		// Spanish-specific characters
		case 'ñ', '¿', '¡':
			spanishChars++
		// French-specific characters  
		case 'â', 'æ', 'ç', 'ê', 'ë', 'î', 'ï', 'ô', 'û', 'ÿ':
			frenchChars++
		// German-specific characters
		case 'ä', 'ö', 'ü', 'ß':
			germanChars++
		// Portuguese-specific characters
		case 'ã', 'õ':
			portugueseChars++
		// Polish-specific characters
		case 'ą', 'ć', 'ę', 'ł', 'ń', 'ś', 'ź', 'ż':
			polishChars++
		// Czech-specific characters
		case 'č', 'ě', 'ň', 'ř', 'š', 'ž', 'ť', 'ď':
			czechChars++
		// Slovak-specific characters
		case 'ĺ', 'ľ', 'ŕ', 'ä':
			slovakChars++
		// Croatian-specific characters
		case 'đ':
			croatianChars++
		// Shared accented characters - check by language context
		case 'á', 'é', 'í', 'ó', 'ú':
			// Count for multiple languages but will use word detection
			spanishChars++
			italianChars++
			portugueseChars++
		case 'à', 'è', 'ì', 'ò', 'ù':
			frenchChars++
			italianChars++
		}
	}

	// Check for common words as additional indicators
	spanishWords := strings.Count(lowerText, "hola") + strings.Count(lowerText, "mundo") + strings.Count(lowerText, "gracias")
	frenchWords := strings.Count(lowerText, "bonjour") + strings.Count(lowerText, "monde") + strings.Count(lowerText, "merci")
	germanWords := strings.Count(lowerText, "hallo") + strings.Count(lowerText, "welt") + strings.Count(lowerText, "danke")
	italianWords := strings.Count(lowerText, "ciao") + strings.Count(lowerText, "mondo") + strings.Count(lowerText, "grazie")
	portugueseWords := strings.Count(lowerText, "olá") + strings.Count(lowerText, "mundo") + strings.Count(lowerText, "obrigado")

	// Calculate scores
	spanishScore := spanishChars*10 + spanishWords*20
	frenchScore := frenchChars*10 + frenchWords*20
	germanScore := germanChars*10 + germanWords*20
	italianScore := italianChars*10 + italianWords*20
	portugueseScore := portugueseChars*10 + portugueseWords*20
	polishScore := polishChars * 10
	czechScore := czechChars * 10
	slovakScore := slovakChars * 10
	croatianScore := croatianChars * 10

	// Find language with highest score
	maxScore := spanishScore
	bestLang := Spanish

	if frenchScore > maxScore {
		maxScore = frenchScore
		bestLang = French
	}
	if germanScore > maxScore {
		maxScore = germanScore
		bestLang = German
	}
	if italianScore > maxScore {
		maxScore = italianScore
		bestLang = Italian
	}
	if portugueseScore > maxScore {
		maxScore = portugueseScore
		bestLang = Portuguese
	}
	if polishScore > maxScore {
		maxScore = polishScore
		bestLang = Polish
	}
	if czechScore > maxScore {
		maxScore = czechScore
		bestLang = Czech
	}
	if slovakScore > maxScore {
		maxScore = slovakScore
		bestLang = Slovak
	}
	if croatianScore > maxScore {
		maxScore = croatianScore
		bestLang = Croatian
	}

	return bestLang
}

// detectCJKLanguage distinguishes between CJK languages
func (d *Detector) detectCJKLanguage(text string) Language {
	// Count specific script types
	var (
		hiragana int
		katakana int
		hangul   int
		han      int
	)

	for _, r := range text {
		switch {
		case unicode.Is(unicode.Hiragana, r):
			hiragana++
		case unicode.Is(unicode.Katakana, r):
			katakana++
		case unicode.Is(unicode.Hangul, r):
			hangul++
		case unicode.Is(unicode.Han, r):
			han++
		}
	}

	totalCJK := hiragana + katakana + hangul + han
	if totalCJK == 0 {
		return Chinese // default
	}

	// Korean has Hangul characters
	if float64(hangul)/float64(totalCJK) > 0.3 {
		return Korean
	}

	// Japanese has Hiragana/Katakana mixed with Kanji
	if (float64(hiragana)+float64(katakana))/float64(totalCJK) > 0.2 {
		return Japanese
	}

	// Default to Chinese (mostly Han characters)
	return Chinese
}

// ParseLanguage parses a language string (code or name)
func ParseLanguage(s string) (Language, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if lang, ok := languageMap[s]; ok {
		return lang, nil
	}
	return Language{}, fmt.Errorf("unknown language: %s", s)
}

// GetSupportedLanguages returns list of supported languages
func GetSupportedLanguages() []Language {
	return []Language{
		English, Russian, Serbian, German, French, Spanish,
		Italian, Portuguese, Chinese, Japanese, Korean, Arabic,
		Polish, Ukrainian, Czech, Slovak, Croatian, Bulgarian,
	}
}

// isCyrillic checks if a rune is Cyrillic
func isCyrillic(r rune) bool {
	return unicode.Is(unicode.Cyrillic, r)
}

// isLatin checks if a rune is Latin
func isLatin(r rune) bool {
	return unicode.Is(unicode.Latin, r)
}

// isCJK checks if a rune is CJK (Chinese, Japanese, Korean)
func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}

// isArabic checks if a rune is Arabic
func isArabic(r rune) bool {
	return unicode.Is(unicode.Arabic, r)
}
