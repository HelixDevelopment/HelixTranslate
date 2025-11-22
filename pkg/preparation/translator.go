package preparation

import (
	"context"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/translator"
	"fmt"
	"log"
	"strings"
)

// PreparationAwareTranslator wraps a translator with preparation phase capabilities
type PreparationAwareTranslator struct {
	baseTranslator    translator.Translator
	langDetector      *language.Detector
	sourceLanguage    language.Language
	targetLanguage    language.Language
	preparationConfig *PreparationConfig
	preparationResult *PreparationResult
	enablePreparation bool
}

// NewPreparationAwareTranslator creates a translator with optional preparation phase
func NewPreparationAwareTranslator(
	baseTranslator translator.Translator,
	langDetector *language.Detector,
	sourceLang, targetLang language.Language,
	preparationConfig *PreparationConfig,
) *PreparationAwareTranslator {
	return &PreparationAwareTranslator{
		baseTranslator:    baseTranslator,
		langDetector:      langDetector,
		sourceLanguage:    sourceLang,
		targetLanguage:    targetLang,
		preparationConfig: preparationConfig,
		enablePreparation: preparationConfig != nil,
	}
}

// TranslateBook translates a book with optional preparation phase
func (pat *PreparationAwareTranslator) TranslateBook(
	ctx context.Context,
	book *ebook.Book,
	eventBus *events.EventBus,
	sessionID string,
) error {
	// Run preparation phase if enabled
	if pat.enablePreparation {
		if err := pat.runPreparation(ctx, book, eventBus, sessionID); err != nil {
			log.Printf("Warning: Preparation phase failed: %v", err)
			translator.EmitProgress(eventBus, sessionID,
				"Preparation phase failed, continuing with standard translation",
				map[string]interface{}{"error": err.Error()})
		}
	}

	// Run the standard translation pipeline
	return pat.translateWithPreparationContext(ctx, book, eventBus, sessionID)
}

// runPreparation executes the multi-pass preparation phase
func (pat *PreparationAwareTranslator) runPreparation(
	ctx context.Context,
	book *ebook.Book,
	eventBus *events.EventBus,
	sessionID string,
) error {
	translator.EmitProgress(eventBus, sessionID,
		fmt.Sprintf("🔍 Starting preparation phase (%d passes)", pat.preparationConfig.PassCount),
		map[string]interface{}{
			"phase":      "preparation",
			"pass_count": pat.preparationConfig.PassCount,
			"providers":  pat.preparationConfig.Providers,
		})

	// Create preparation coordinator
	coordinator, err := NewPreparationCoordinator(*pat.preparationConfig)
	if err != nil {
		return fmt.Errorf("failed to create preparation coordinator: %w", err)
	}

	// Run multi-pass analysis
	result, err := coordinator.PrepareBook(ctx, book)
	if err != nil {
		return fmt.Errorf("preparation failed: %w", err)
	}

	pat.preparationResult = result

	// Emit preparation summary
	translator.EmitProgress(eventBus, sessionID,
		"✅ Preparation phase complete",
		map[string]interface{}{
			"phase":             "preparation",
			"content_type":      result.FinalAnalysis.ContentType,
			"genre":             result.FinalAnalysis.Genre,
			"untranslatable":    len(result.FinalAnalysis.UntranslatableTerms),
			"footnotes":         len(result.FinalAnalysis.FootnoteGuidance),
			"characters":        len(result.FinalAnalysis.Characters),
			"cultural_refs":     len(result.FinalAnalysis.CulturalReferences),
			"chapters_analyzed": len(result.FinalAnalysis.ChapterAnalyses),
			"duration":          result.TotalDuration.Seconds(),
		})

	log.Printf("\n%s\n", FormatPreparationSummary(result))

	return nil
}

// translateWithPreparationContext translates the book using preparation context
func (pat *PreparationAwareTranslator) translateWithPreparationContext(
	ctx context.Context,
	book *ebook.Book,
	eventBus *events.EventBus,
	sessionID string,
) error {
	// Detect source language if not specified
	sourceLang := pat.sourceLanguage
	targetLang := pat.targetLanguage

	if sourceLang.Code == "" && pat.langDetector != nil {
		translator.EmitProgress(eventBus, sessionID, "Detecting source language", nil)

		sample := book.ExtractText()
		if len(sample) > 2000 {
			sample = sample[:2000]
		}

		detectedLang, err := pat.langDetector.Detect(ctx, sample)
		if err == nil {
			pat.sourceLanguage = detectedLang
			sourceLang = detectedLang
			translator.EmitProgress(eventBus, sessionID,
				fmt.Sprintf("Detected language: %s", detectedLang.Name),
				map[string]interface{}{
					"language_code": detectedLang.Code,
					"language_name": detectedLang.Name,
				})
		}
	}

	// Update metadata language
	if book.Metadata.Language == "" {
		book.Metadata.Language = targetLang.Code
	}

	// Translate metadata
	translator.EmitProgress(eventBus, sessionID, "Translating metadata", nil)
	if err := pat.translateMetadata(ctx, &book.Metadata, eventBus, sessionID); err != nil {
		return fmt.Errorf("failed to translate metadata: %w", err)
	}

	// Translate chapters with preparation context
	totalChapters := len(book.Chapters)
	for i := range book.Chapters {
		translator.EmitProgress(eventBus, sessionID,
			fmt.Sprintf("Translating chapter %d/%d", i+1, totalChapters),
			map[string]interface{}{
				"chapter":        i + 1,
				"total_chapters": totalChapters,
				"progress":       float64(i+1) / float64(totalChapters) * 100,
			})

		// Get chapter-specific preparation context
		chapterContext := pat.getChapterContext(i + 1)

		if err := pat.translateChapter(ctx, &book.Chapters[i], chapterContext, eventBus, sessionID); err != nil {
			return fmt.Errorf("failed to translate chapter %d: %w", i+1, err)
		}
	}

	// Update book language
	book.Metadata.Language = targetLang.Code

	return nil
}

// getChapterContext generates translation context for a specific chapter
func (pat *PreparationAwareTranslator) getChapterContext(chapterNum int) string {
	if pat.preparationResult == nil {
		return "Literary text"
	}

	return GetTranslationContext(&pat.preparationResult.FinalAnalysis, chapterNum)
}

// translateMetadata translates book metadata with preparation context
func (pat *PreparationAwareTranslator) translateMetadata(
	ctx context.Context,
	metadata *ebook.Metadata,
	eventBus *events.EventBus,
	sessionID string,
) error {
	// Translate title (if not in untranslatable terms)
	if metadata.Title != "" && !pat.isUntranslatable(metadata.Title) {
		translated, err := pat.baseTranslator.TranslateWithProgress(
			ctx,
			metadata.Title,
			"Book title - translate to Serbian while preserving literary style",
			eventBus,
			sessionID,
		)
		if err != nil {
			return fmt.Errorf("failed to translate title: %w", err)
		}
		metadata.Title = translated
	}

	// Translate description/annotation
	if metadata.Description != "" {
		context := "Book description/annotation - translate to Serbian, maintain literary tone"
		if pat.preparationResult != nil {
			// Add preparation context about tone and style
			analysis := pat.preparationResult.FinalAnalysis
			context += fmt.Sprintf("\n\nBook style: %s, %s", analysis.ContentType, analysis.Tone)
		}

		translated, err := pat.baseTranslator.TranslateWithProgress(
			ctx,
			metadata.Description,
			context,
			eventBus,
			sessionID,
		)
		if err != nil {
			translator.EmitProgress(eventBus, sessionID, "Warning: Failed to translate description", map[string]interface{}{"error": err.Error()})
		} else {
			metadata.Description = translated
		}
	}

	// Note: Authors, Publisher, ISBN, Date, and Cover are intentionally kept in original form
	// as they are proper nouns, identifiers, or binary data that shouldn't be translated

	return nil
}

// isUntranslatable checks if a term should not be translated based on preparation analysis
func (pat *PreparationAwareTranslator) isUntranslatable(term string) bool {
	if pat.preparationResult == nil {
		return false
	}

	// Check untranslatable terms
	for _, ut := range pat.preparationResult.FinalAnalysis.UntranslatableTerms {
		if strings.Contains(strings.ToLower(term), strings.ToLower(ut.Term)) {
			return true
		}
	}

	return false
}

// translateChapter translates a chapter with preparation context
func (pat *PreparationAwareTranslator) translateChapter(
	ctx context.Context,
	chapter *ebook.Chapter,
	prepContext string,
	eventBus *events.EventBus,
	sessionID string,
) error {
	// Translate chapter title
	if chapter.Title != "" {
		titleContext := prepContext
		if titleContext == "" {
			titleContext = "Chapter title"
		} else {
			titleContext = "Chapter title\n\n" + titleContext
		}

		translated, err := pat.baseTranslator.TranslateWithProgress(
			ctx,
			chapter.Title,
			titleContext,
			eventBus,
			sessionID,
		)
		if err != nil {
			return fmt.Errorf("failed to translate chapter title: %w", err)
		}
		chapter.Title = translated
	}

	// Translate sections with context
	for i := range chapter.Sections {
		if err := pat.translateSection(ctx, &chapter.Sections[i], prepContext, eventBus, sessionID); err != nil {
			return err
		}
	}

	return nil
}

// translateSection translates a section recursively with preparation context
func (pat *PreparationAwareTranslator) translateSection(
	ctx context.Context,
	section *ebook.Section,
	prepContext string,
	eventBus *events.EventBus,
	sessionID string,
) error {
	// Translate section title
	if section.Title != "" {
		titleContext := prepContext
		if titleContext == "" {
			titleContext = "Section title"
		} else {
			titleContext = "Section title\n\n" + titleContext
		}

		translated, err := pat.baseTranslator.TranslateWithProgress(
			ctx,
			section.Title,
			titleContext,
			eventBus,
			sessionID,
		)
		if err != nil {
			return fmt.Errorf("failed to translate section title: %w", err)
		}
		section.Title = translated
	}

	// Translate content with full preparation context
	if section.Content != "" {
		contentContext := prepContext
		if contentContext == "" {
			contentContext = "Section content"
		} else {
			contentContext = "Section content\n\n" + contentContext
		}

		translated, err := pat.baseTranslator.TranslateWithProgress(
			ctx,
			section.Content,
			contentContext,
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
		if err := pat.translateSection(ctx, &section.Subsections[i], prepContext, eventBus, sessionID); err != nil {
			return err
		}
	}

	return nil
}

// GetPreparationResult returns the preparation result
func (pat *PreparationAwareTranslator) GetPreparationResult() *PreparationResult {
	return pat.preparationResult
}

// SavePreparationAnalysis saves the preparation analysis to a file
func (pat *PreparationAwareTranslator) SavePreparationAnalysis(outputPath string) error {
	if pat.preparationResult == nil {
		return fmt.Errorf("no preparation result available")
	}

	return SavePreparationResult(pat.preparationResult, outputPath)
}
