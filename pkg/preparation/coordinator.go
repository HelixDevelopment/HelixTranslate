package preparation

import (
	"context"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// PreparationCoordinator orchestrates multi-pass content analysis
type PreparationCoordinator struct {
	config    PreparationConfig
	providers []translator.Translator
}

// NewPreparationCoordinator creates a new preparation coordinator
func NewPreparationCoordinator(config PreparationConfig) (*PreparationCoordinator, error) {
	if config.PassCount < 1 {
		config.PassCount = 2 // Default to 2 passes
	}

	// Initialize LLM providers
	var providers []translator.Translator
	for _, providerName := range config.Providers {
		// Create translator config
		tConfig := translator.TranslationConfig{
			SourceLang: config.SourceLanguage,
			TargetLang: config.TargetLanguage,
			Provider:   providerName,
		}

		// Create LLM translator
		llmTranslator, err := llm.NewLLMTranslator(tConfig)
		if err != nil {
			log.Printf("Warning: Failed to create %s translator: %v", providerName, err)
			continue
		}

		providers = append(providers, llmTranslator)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no valid LLM providers available")
	}

	return &PreparationCoordinator{
		config:    config,
		providers: providers,
	}, nil
}

// PrepareBook performs multi-pass analysis on an entire book
func (pc *PreparationCoordinator) PrepareBook(ctx context.Context, book *ebook.Book) (*PreparationResult, error) {
	startTime := time.Now()

	log.Printf("🔍 Starting preparation phase: %d passes with %d providers", pc.config.PassCount, len(pc.providers))
	log.Printf("   Analysis scope: content_type=%t, characters=%t, terminology=%t, culture=%t, chapters=%t",
		pc.config.AnalyzeContentType, pc.config.AnalyzeCharacters, pc.config.AnalyzeTerminology,
		pc.config.AnalyzeCulture, pc.config.AnalyzeChapters)

	result := &PreparationResult{
		SourceLanguage: pc.config.SourceLanguage,
		TargetLanguage: pc.config.TargetLanguage,
		StartedAt:      startTime,
		PassCount:      pc.config.PassCount,
		Passes:         make([]PreparationPass, 0, pc.config.PassCount),
	}

	// Extract full book content for analysis
	bookContent := pc.extractBookContent(book)

	// Perform multiple analysis passes
	var previousAnalysis *ContentAnalysis
	for passNum := 1; passNum <= pc.config.PassCount; passNum++ {
		// Select provider for this pass (rotate through providers)
		providerIndex := (passNum - 1) % len(pc.providers)
		provider := pc.providers[providerIndex]

		log.Printf("  Pass %d/%d: Analyzing with %s...", passNum, pc.config.PassCount,
			provider.GetName())

		// Perform analysis pass
		pass, err := pc.performPass(ctx, passNum, provider, bookContent, previousAnalysis)
		if err != nil {
			log.Printf("  ❌ Pass %d failed: %v", passNum, err)
			continue
		}

		result.Passes = append(result.Passes, *pass)
		result.TotalTokens += pass.TokensUsed
		previousAnalysis = &pass.Analysis

		log.Printf("  ✅ Pass %d complete (%.2fs)",
			passNum, pass.Duration.Seconds())
	}

	// Analyze chapters if requested
	if pc.config.AnalyzeChapters {
		log.Printf("  Analyzing individual chapters...")
		chapterAnalyses, err := pc.analyzeChapters(ctx, book)
		if err != nil {
			log.Printf("  Warning: Chapter analysis failed: %v", err)
		} else {
			// Add chapter analyses to the last pass
			if len(result.Passes) > 0 {
				result.Passes[len(result.Passes)-1].Analysis.ChapterAnalyses = chapterAnalyses
			}
		}
	}

	// Consolidate all analyses into final result
	if len(result.Passes) > 1 {
		log.Printf("  Consolidating %d analyses...", len(result.Passes))
		finalAnalysis, err := pc.consolidateAnalyses(ctx, result.Passes)
		if err != nil {
			log.Printf("  Warning: Consolidation failed: %v", err)
			// Fall back to last pass
			result.FinalAnalysis = result.Passes[len(result.Passes)-1].Analysis
		} else {
			result.FinalAnalysis = *finalAnalysis
		}
	} else if len(result.Passes) == 1 {
		result.FinalAnalysis = result.Passes[0].Analysis
	}

	result.CompletedAt = time.Now()
	result.TotalDuration = result.CompletedAt.Sub(startTime)

	log.Printf("✅ Preparation complete: %d passes in %.2fs", len(result.Passes), result.TotalDuration.Seconds())
	log.Printf("   Final analysis: %s (%s) - %d untranslatable terms, %d footnotes, %d characters, %d cultural refs",
		result.FinalAnalysis.ContentType, result.FinalAnalysis.Genre,
		len(result.FinalAnalysis.UntranslatableTerms), len(result.FinalAnalysis.FootnoteGuidance),
		len(result.FinalAnalysis.Characters), len(result.FinalAnalysis.CulturalReferences))

	return result, nil
}

// performPass executes a single analysis pass
func (pc *PreparationCoordinator) performPass(
	ctx context.Context,
	passNum int,
	provider translator.Translator,
	content string,
	previousAnalysis *ContentAnalysis,
) (*PreparationPass, error) {
	startTime := time.Now()

	// Build prompt
	promptBuilder := NewPreparationPromptBuilder(
		pc.config.SourceLanguage,
		pc.config.TargetLanguage,
		passNum,
	)

	if previousAnalysis != nil {
		promptBuilder.WithPreviousAnalysis(previousAnalysis)
	}

	var prompt string
	if passNum == 1 {
		prompt = promptBuilder.BuildInitialAnalysisPrompt(content)
	} else {
		prompt = promptBuilder.BuildRefinementPrompt(content)
	}

	// Call LLM for analysis
	response, err := provider.Translate(ctx, prompt, "")
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// Parse JSON response
	analysis, err := pc.parseAnalysisResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse analysis: %w", err)
	}

	// Set metadata
	analysis.AnalysisVersion = passNum
	analysis.AnalyzedAt = time.Now()
	analysis.AnalyzedBy = provider.GetName()

	pass := &PreparationPass{
		PassNumber: passNum,
		Provider:   provider.GetName(),
		Model:      "", // Model name not available from Translator interface
		Analysis:   *analysis,
		Duration:   time.Since(startTime),
		TokensUsed: estimateTokens(prompt + response),
	}

	return pass, nil
}

// analyzeChapters performs detailed analysis of each chapter
func (pc *PreparationCoordinator) analyzeChapters(ctx context.Context, book *ebook.Book) ([]ChapterAnalysis, error) {
	var analyses []ChapterAnalysis
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Select a provider for chapter analysis
	provider := pc.providers[0]
	promptBuilder := NewPreparationPromptBuilder(
		pc.config.SourceLanguage,
		pc.config.TargetLanguage,
		1,
	)

	// Analyze chapters in parallel (with concurrency limit)
	semaphore := make(chan struct{}, 3) // Max 3 concurrent analyses

	for i, chapter := range book.Chapters {
		wg.Add(1)
		go func(chapterNum int, ch ebook.Chapter) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			// Extract chapter content
			chapterContent := pc.extractChapterContent(&ch)

			// Build prompt
			prompt := promptBuilder.BuildChapterAnalysisPrompt(
				chapterNum+1,
				ch.Title,
				chapterContent,
			)

			// Call LLM
			response, err := provider.Translate(ctx, prompt, "")
			if err != nil {
				log.Printf("    Warning: Chapter %d analysis failed: %v", chapterNum+1, err)
				return
			}

			// Parse response
			var analysis ChapterAnalysis
			if err := json.Unmarshal([]byte(extractJSON(response)), &analysis); err != nil {
				log.Printf("    Warning: Failed to parse chapter %d analysis: %v", chapterNum+1, err)
				return
			}

			mu.Lock()
			analyses = append(analyses, analysis)
			mu.Unlock()

			log.Printf("    ✓ Chapter %d analyzed", chapterNum+1)
		}(i, chapter)
	}

	wg.Wait()

	return analyses, nil
}

// consolidateAnalyses merges multiple analysis passes into a final result
func (pc *PreparationCoordinator) consolidateAnalyses(
	ctx context.Context,
	passes []PreparationPass,
) (*ContentAnalysis, error) {
	// Extract analyses
	var analyses []ContentAnalysis
	for _, pass := range passes {
		analyses = append(analyses, pass.Analysis)
	}

	// Build consolidation prompt
	promptBuilder := NewPreparationPromptBuilder(
		pc.config.SourceLanguage,
		pc.config.TargetLanguage,
		len(passes)+1,
	)
	prompt := promptBuilder.BuildConsolidationPrompt(analyses)

	// Use first provider for consolidation
	provider := pc.providers[0]
	response, err := provider.Translate(ctx, prompt, "")
	if err != nil {
		return nil, fmt.Errorf("consolidation failed: %w", err)
	}

	// Parse consolidated analysis
	return pc.parseAnalysisResponse(response)
}

// parseAnalysisResponse parses LLM response into ContentAnalysis
func (pc *PreparationCoordinator) parseAnalysisResponse(response string) (*ContentAnalysis, error) {
	// Extract JSON from response (LLM might include extra text)
	jsonStr := extractJSON(response)

	var analysis ContentAnalysis
	if err := json.Unmarshal([]byte(jsonStr), &analysis); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	return &analysis, nil
}

// extractBookContent extracts full text content from a book
func (pc *PreparationCoordinator) extractBookContent(book *ebook.Book) string {
	var content strings.Builder

	// Add metadata
	content.WriteString(fmt.Sprintf("Title: %s\n", book.Metadata.Title))
	if len(book.Metadata.Authors) > 0 {
		content.WriteString(fmt.Sprintf("Authors: %s\n", strings.Join(book.Metadata.Authors, ", ")))
	}
	content.WriteString("\n---\n\n")

	// Add all chapters
	for i, chapter := range book.Chapters {
		content.WriteString(fmt.Sprintf("\n\n## Chapter %d", i+1))
		if chapter.Title != "" {
			content.WriteString(fmt.Sprintf(": %s", chapter.Title))
		}
		content.WriteString("\n\n")

		// Add sections
		for _, section := range chapter.Sections {
			content.WriteString(section.Content)
			content.WriteString("\n\n")
		}
	}

	return content.String()
}

// extractChapterContent extracts text from a single chapter
func (pc *PreparationCoordinator) extractChapterContent(chapter *ebook.Chapter) string {
	var content strings.Builder
	for _, section := range chapter.Sections {
		content.WriteString(section.Content)
		content.WriteString("\n\n")
	}
	return content.String()
}

// extractJSON attempts to extract JSON from LLM response
func extractJSON(response string) string {
	// Try to find JSON block
	if idx := strings.Index(response, "{"); idx != -1 {
		if lastIdx := strings.LastIndex(response, "}"); lastIdx != -1 {
			return response[idx : lastIdx+1]
		}
	}
	return response
}

// estimateTokens roughly estimates token count
func estimateTokens(text string) int {
	// Rough estimate: ~4 characters per token
	return len(text) / 4
}
