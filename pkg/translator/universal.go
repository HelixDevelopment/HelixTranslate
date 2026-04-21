package translator

import (
	"context"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"fmt"
)

// UniversalTranslator handles translation of complete ebooks
type UniversalTranslator struct {
	translator     Translator
	langDetector   *language.Detector
	sourceLanguage language.Language
	targetLanguage language.Language
}

// NewUniversalTranslator creates a new universal translator
func NewUniversalTranslator(
	translator Translator,
	langDetector *language.Detector,
	sourceLang, targetLang language.Language,
) *UniversalTranslator {
	return &UniversalTranslator{
		translator:     translator,
		langDetector:   langDetector,
		sourceLanguage: sourceLang,
		targetLanguage: targetLang,
	}
}

// TranslateBook translates an entire ebook
func (ut *UniversalTranslator) TranslateBook(
	ctx context.Context,
	book *ebook.Book,
	eventBus *events.EventBus,
	sessionID string,
) error {
	if book == nil {
		return fmt.Errorf("book cannot be nil")
	}

	// Detect source language if not specified
	if ut.sourceLanguage.Code == "" && ut.langDetector != nil {
		EmitProgress(eventBus, sessionID, "Detecting source language", nil)

		sample := book.ExtractText()
		if len(sample) > 2000 {
			sample = sample[:2000]
		}

		detectedLang, err := ut.langDetector.Detect(ctx, sample)
		if err == nil {
			ut.sourceLanguage = detectedLang
			EmitProgress(eventBus, sessionID,
				fmt.Sprintf("Detected language: %s", detectedLang.Name),
				map[string]interface{}{
					"language_code": detectedLang.Code,
					"language_name": detectedLang.Name,
				})
		}
	}

	// Update metadata language
	if book.Metadata.Language == "" {
		book.Metadata.Language = ut.targetLanguage.Code
	}

	// Translate metadata
	EmitProgress(eventBus, sessionID, "Translating metadata", nil)
	if err := ut.translateMetadata(ctx, &book.Metadata, eventBus, sessionID); err != nil {
		return fmt.Errorf("failed to translate metadata: %w", err)
	}

	// Translate chapters
	totalChapters := len(book.Chapters)
	for i := range book.Chapters {
		EmitProgress(eventBus, sessionID,
			fmt.Sprintf("Translating chapter %d/%d", i+1, totalChapters),
			map[string]interface{}{
				"chapter":        i + 1,
				"total_chapters": totalChapters,
				"progress":       float64(i+1) / float64(totalChapters) * 100,
			})

		if err := ut.translateChapter(ctx, &book.Chapters[i], eventBus, sessionID); err != nil {
			return fmt.Errorf("failed to translate chapter %d: %w", i+1, err)
		}
	}

	// Update book language
	book.Metadata.Language = ut.targetLanguage.Code

	return nil
}

// translateMetadata translates book metadata
func (ut *UniversalTranslator) translateMetadata(
	ctx context.Context,
	metadata *ebook.Metadata,
	eventBus *events.EventBus,
	sessionID string,
) error {
	// Translate title
	if metadata.Title != "" {
		translated, err := ut.translator.TranslateWithProgress(
			ctx,
			metadata.Title,
			"Book title",
			eventBus,
			sessionID,
		)
		if err != nil {
			return fmt.Errorf("failed to translate title: %w", err)
		}
		metadata.Title = translated
	}

	// Translate description
	if metadata.Description != "" {
		translated, err := ut.translator.TranslateWithProgress(
			ctx,
			metadata.Description,
			"Book description",
			eventBus,
			sessionID,
		)
		if err != nil {
			EmitProgress(eventBus, sessionID, "Warning: Failed to translate description", map[string]interface{}{"error": err.Error()})
		} else {
			metadata.Description = translated
		}
	}

	return nil
}

// translateChapter translates a chapter
func (ut *UniversalTranslator) translateChapter(
	ctx context.Context,
	chapter *ebook.Chapter,
	eventBus *events.EventBus,
	sessionID string,
) error {
	// Translate chapter title
	if chapter.Title != "" {
		translated, err := ut.translator.TranslateWithProgress(
			ctx,
			chapter.Title,
			"Chapter title",
			eventBus,
			sessionID,
		)
		if err != nil {
			return fmt.Errorf("failed to translate chapter title: %w", err)
		}
		chapter.Title = translated
	}

	// Translate sections
	for i := range chapter.Sections {
		if err := ut.translateSection(ctx, &chapter.Sections[i], eventBus, sessionID); err != nil {
			return err
		}
	}

	return nil
}

// translateSection translates a section recursively
func (ut *UniversalTranslator) translateSection(
	ctx context.Context,
	section *ebook.Section,
	eventBus *events.EventBus,
	sessionID string,
) error {
	// Translate section title
	if section.Title != "" {
		translated, err := ut.translator.TranslateWithProgress(
			ctx,
			section.Title,
			"Section title",
			eventBus,
			sessionID,
		)
		if err != nil {
			return fmt.Errorf("failed to translate section title: %w", err)
		}
		section.Title = translated
	}

	// Translate content
	if section.Content != "" {
		translated, err := ut.translator.TranslateWithProgress(
			ctx,
			section.Content,
			"Section content",
			eventBus,
			sessionID,
		)
		if err != nil {
			return fmt.Errorf("failed to translate section content: %w", err)
		}
		section.Content = translated
	}

	// Translate subsections
	for i := range section.Subsections {
		if err := ut.translateSection(ctx, &section.Subsections[i], eventBus, sessionID); err != nil {
			return err
		}
	}

	return nil
}

// GetSourceLanguage returns the source language
func (ut *UniversalTranslator) GetSourceLanguage() language.Language {
	return ut.sourceLanguage
}

// GetTargetLanguage returns the target language
func (ut *UniversalTranslator) GetTargetLanguage() language.Language {
	return ut.targetLanguage
}

// CreatePromptForLanguages creates a translation prompt for any language pair
func CreatePromptForLanguages(text, sourceLang, targetLang, context string) string {
	if context == "" {
		context = "Literary text"
	}

	return fmt.Sprintf(`You are a professional translator specializing in %s to %s translation.
Your task is to translate the following text accurately and naturally.

Guidelines:
1. Preserve the original meaning and tone
2. Use natural, idiomatic %s
3. Maintain cultural context and nuances
4. Keep proper nouns unchanged unless they have standard %s equivalents
5. Preserve formatting and punctuation
6. Ensure grammatical correctness

Context: %s

%s text:
%s

%s translation:`,
		sourceLang, targetLang,
		targetLang,
		targetLang,
		context,
		sourceLang, text,
		targetLang)
}
