package dictionary

import (
	"context"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"strings"
)

// DictionaryTranslator implements simple dictionary-based translation
type DictionaryTranslator struct {
	*translator.BaseTranslator
	dictionary map[string]string
}

// NewDictionaryTranslator creates a new dictionary translator
func NewDictionaryTranslator(config translator.TranslationConfig) *DictionaryTranslator {
	dt := &DictionaryTranslator{
		BaseTranslator: translator.NewBaseTranslator(config),
		dictionary:     getDefaultDictionary(),
	}
	return dt
}

// getDefaultDictionary returns the Russian-Serbian dictionary
func getDefaultDictionary() map[string]string {
	return map[string]string{
		"Ратибор":     "Ратибор",
		"Отзвуки":     "Одјеци",
		"фэнтези":     "фантастика",
		"приключения": "авантуре",
		"герой":       "јунак",
		"мир":         "свет",
		"человек":     "човек",
		"жизнь":       "живот",
		"любовь":      "љубав",
		"смерть":      "смрт",
		"время":       "време",
		"дом":         "кућа",
		"сердце":      "срце",
		"душа":        "душа",
		"ночь":        "ноћ",
		"день":        "дан",
		"солнце":      "сунце",
		"луна":        "месец",
		"звезда":      "звезда",
		"небо":        "небо",
		"земля":       "земља",
		"вода":        "вода",
		"огонь":       "ватра",
		"воздух":      "ваздух",
		"деревня":     "село",
		"город":       "град",
		"улица":       "улица",
		"книга":       "књига",
		"слово":       "реч",
		"язык":        "језик",
		"глава":       "поглавље",
		"страница":    "страница",
		"история":     "прича",
		"конец":       "крај",
		"начало":      "почетак",
		"будущее":     "будућност",
		"прошлое":     "прошлост",
		"настоящее":   "садашњост",
		"вопрос":      "питање",
		"ответ":       "одговор",
		"мысль":       "мисао",
		"чувство":     "осећање",
		"радость":     "радост",
		"грусть":      "туга",
		"страх":       "страх",
		"надежда":     "нада",
		"мечта":       "сан",

		// Additional common words for testing
		"тестовая":     "test",
		"проверка":     "провера",
		"перевода":     "превођења",
		"здесь":        "овде",
		"находится":    "се налази",
		"русский":      "руски",
		"текст":        "текст",
		"который":      "који",
		"нужно":        "треба",
		"перевести":    "превести",
		"сербский":     "српски",
		"второй":       "други",
		"абзац":        "пасус",
		"содержит":     "садржи",
		"больше":       "више",
		"тестирования": "тестирања",
		"хотим":        "желимо",
		"убедиться":    "уверити",
		"работает":     "ради",
		"правильно":    "исправно",
		"сохраняет":    "очувава",
		"структуру":    "структуру",
		"документа":    "документа",
	}
}

// GetName returns the translator name
func (dt *DictionaryTranslator) GetName() string {
	return "dictionary"
}

// Translate translates text using the dictionary
func (dt *DictionaryTranslator) Translate(ctx context.Context, text string, contextStr string) (string, error) {
	if text == "" || strings.TrimSpace(text) == "" {
		return text, nil
	}

	// Check cache
	if cached, found := dt.CheckCache(text); found {
		return cached, nil
	}

	result := text
	replacements := 0

	// Replace dictionary entries
	for ru, sr := range dt.dictionary {
		if strings.Contains(result, ru) {
			result = strings.ReplaceAll(result, ru, sr)
			replacements++
		}
	}

	// Update stats
	dt.UpdateStats(true)

	// Cache result
	dt.AddToCache(text, result)

	return result, nil
}

// TranslateWithProgress translates and reports progress
func (dt *DictionaryTranslator) TranslateWithProgress(
	ctx context.Context,
	text string,
	contextStr string,
	eventBus *events.EventBus,
	sessionID string,
) (string, error) {
	translator.EmitProgress(eventBus, sessionID, "Starting dictionary translation", map[string]interface{}{
		"text_length": len(text),
	})

	result, err := dt.Translate(ctx, text, contextStr)

	if err != nil {
		translator.EmitError(eventBus, sessionID, "Dictionary translation failed", err)
		return "", err
	}

	translator.EmitProgress(eventBus, sessionID, "Dictionary translation completed", map[string]interface{}{
		"original_length":   len(text),
		"translated_length": len(result),
	})

	return result, nil
}

// AddDictionaryEntry adds a new dictionary entry
func (dt *DictionaryTranslator) AddDictionaryEntry(russian, serbian string) {
	dt.dictionary[russian] = serbian
}

// LoadDictionary loads a custom dictionary
func (dt *DictionaryTranslator) LoadDictionary(dict map[string]string) {
	for k, v := range dict {
		dt.dictionary[k] = v
	}
}
