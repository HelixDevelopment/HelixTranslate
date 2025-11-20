package unit

import (
	"context"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/dictionary"
	"strings"
	"testing"
)

func TestDictionaryTranslator(t *testing.T) {
	config := translator.TranslationConfig{
		SourceLang: "ru",
		TargetLang: "sr",
		Provider:   "dictionary",
	}

	trans := dictionary.NewDictionaryTranslator(config)
	ctx := context.Background()

	t.Run("TranslateSimpleWord", func(t *testing.T) {
		result, err := trans.Translate(ctx, "герой", "")
		if err != nil {
			t.Fatalf("Translation failed: %v", err)
		}

		if result != "јунак" {
			t.Errorf("Expected 'јунак', got '%s'", result)
		}
	})

	t.Run("TranslateSentence", func(t *testing.T) {
		result, err := trans.Translate(ctx, "Это герой мира.", "")
		if err != nil {
			t.Fatalf("Translation failed: %v", err)
		}

		if !strings.Contains(result, "јунак") {
			t.Errorf("Expected translation to contain 'јунак', got '%s'", result)
		}

		if !strings.Contains(result, "свет") {
			t.Errorf("Expected translation to contain 'свет', got '%s'", result)
		}
	})

	t.Run("TranslateEmptyString", func(t *testing.T) {
		result, err := trans.Translate(ctx, "", "")
		if err != nil {
			t.Fatalf("Translation failed: %v", err)
		}

		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})

	t.Run("CacheWorking", func(t *testing.T) {
		text := "герой мира"

		// First translation
		result1, err := trans.Translate(ctx, text, "")
		if err != nil {
			t.Fatalf("First translation failed: %v", err)
		}

		// Second translation (should be cached)
		result2, err := trans.Translate(ctx, text, "")
		if err != nil {
			t.Fatalf("Second translation failed: %v", err)
		}

		if result1 != result2 {
			t.Errorf("Cached result differs: %s vs %s", result1, result2)
		}

		stats := trans.GetStats()
		if stats.Cached < 1 {
			t.Error("Expected at least one cached translation")
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := trans.GetStats()

		if stats.Total < 1 {
			t.Error("Expected at least one translation in stats")
		}
	})

	t.Run("GetName", func(t *testing.T) {
		name := trans.GetName()
		if name != "dictionary" {
			t.Errorf("Expected name 'dictionary', got '%s'", name)
		}
	})

	t.Run("AddDictionaryEntry", func(t *testing.T) {
		trans.AddDictionaryEntry("тест", "тест")
		result, err := trans.Translate(ctx, "тест", "")
		if err != nil {
			t.Fatalf("Translation failed: %v", err)
		}

		if result != "тест" {
			t.Errorf("Expected 'тест', got '%s'", result)
		}
	})
}
