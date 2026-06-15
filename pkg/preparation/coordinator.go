package preparation

import (
	"context"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// providerEnvAPIKey returns the API key for a provider from its well-known
// environment variable (e.g. DEEPSEEK_API_KEY). It is the per-provider fallback
// used when no shared API key is configured, so a multi-provider preparation run
// resolves each provider's own credential. An unknown provider returns "" (the
// downstream LLM client then surfaces the precise "<provider> API key is
// required" error rather than this helper guessing a value).
func providerEnvAPIKey(provider string) string {
	envMap := map[string]string{
		"openai":     "OPENAI_API_KEY",
		"anthropic":  "ANTHROPIC_API_KEY",
		"deepseek":   "DEEPSEEK_API_KEY",
		"zhipu":      "ZHIPU_API_KEY",
		"qwen":       "QWEN_API_KEY",
		"gemini":     "GEMINI_API_KEY",
		"groq":       "GROQ_API_KEY",
		"mistral":    "MISTRAL_API_KEY",
		"xai":        "XAI_API_KEY",
		"cohere":     "COHERE_API_KEY",
		"togetherai": "TOGETHER_API_KEY",
	}
	if envVar, ok := envMap[provider]; ok {
		return os.Getenv(envVar)
	}
	return ""
}

// providerDefaultModel returns a sensible default chat model for a provider,
// used when the PreparationConfig does not specify one. It is intentionally
// limited to providers whose LLM client REJECTS an empty model (e.g. DeepSeek
// returns "DeepSeek model is required"); for those, the empty model previously
// left the preparation phase with no valid providers and produced no analysis.
// Providers whose client applies its own default for an empty model (OpenAI,
// Zhipu, etc.) are deliberately left empty so their existing behaviour — and the
// client's model-validation contract — is preserved (§11.4.1: do not break a
// working path). Returned model ids MUST be members of the client's allow-list
// (pkg/translator/llm ValidModels). An unmapped provider returns "".
func providerDefaultModel(provider string) string {
	modelMap := map[string]string{
		"deepseek": "deepseek-chat", // required by NewDeepSeekClient; valid per ValidModels[ProviderDeepSeek]
	}
	return modelMap[provider]
}

// PreparationCoordinator orchestrates multi-pass content analysis
type PreparationCoordinator struct {
	config    PreparationConfig
	providers []translator.Translator
}

// EnsembleTranslatorFactory supplies the preparation providers from an external
// source (e.g. the provider-diverse LLMsVerifier bridge) instead of the
// built-in per-provider NewLLMTranslator construction. nil = use the built-in
// construction (default, behaviour-preserving). This is the injectable seam that
// keeps PreparationCoordinator decoupled from any concrete provider-discovery
// implementation: the factory is a plain function value, so this package never
// imports the bridge or internal/verifier (§11.4.28 decoupling).
type EnsembleTranslatorFactory func(ctx context.Context) ([]translator.Translator, error)

// NewPreparationCoordinator creates a new preparation coordinator using the
// built-in per-provider construction. It is a thin, behaviour-preserving wrapper
// over NewPreparationCoordinatorWithFactory with a nil factory + a background
// context, so the default construction path is provably identical to the
// injectable-factory path's nil branch.
func NewPreparationCoordinator(config PreparationConfig) (*PreparationCoordinator, error) {
	return NewPreparationCoordinatorWithFactory(context.Background(), config, nil)
}

// NewPreparationCoordinatorWithFactory creates a preparation coordinator,
// optionally sourcing its provider-diverse translators from an injected
// EnsembleTranslatorFactory. When factory is nil the EXACT built-in
// per-provider construction runs (default, behaviour-preserving). When factory
// is non-nil the providers come from factory(ctx); an empty result keeps the
// same honest "no valid LLM providers available" error as the built-in path.
func NewPreparationCoordinatorWithFactory(
	ctx context.Context,
	config PreparationConfig,
	factory EnsembleTranslatorFactory,
) (*PreparationCoordinator, error) {
	if config.PassCount < 1 {
		config.PassCount = 2 // Default to 2 passes
	}

	providers, err := buildProviders(ctx, config, factory)
	if err != nil {
		return nil, err
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no valid LLM providers available")
	}

	return &PreparationCoordinator{
		config:    config,
		providers: providers,
	}, nil
}

// buildProviders resolves the coordinator's translators. With a non-nil factory
// the providers are supplied externally; with a nil factory the built-in
// per-provider NewLLMTranslator construction runs unchanged.
func buildProviders(
	ctx context.Context,
	config PreparationConfig,
	factory EnsembleTranslatorFactory,
) ([]translator.Translator, error) {
	if factory != nil {
		return factory(ctx)
	}

	// Initialize LLM providers (built-in default construction).
	var providers []translator.Translator
	for _, providerName := range config.Providers {
		// Resolve the per-provider API key. A shared config.APIKey (when set)
		// applies to all providers; when it is empty, fall back to the
		// provider's own well-known environment variable (e.g. DEEPSEEK_API_KEY,
		// ZHIPU_API_KEY) so a multi-provider list each picks up its own key
		// rather than every client failing "<provider> API key is required".
		apiKey := config.APIKey
		if apiKey == "" {
			apiKey = providerEnvAPIKey(providerName)
		}

		// Create translator config. A default model is supplied per provider
		// when none is configured — some clients (e.g. DeepSeek) reject an empty
		// model with "<provider> model is required", which previously left the
		// preparation phase with no valid providers and produced no analysis.
		tConfig := translator.TranslationConfig{
			SourceLang: config.SourceLanguage,
			TargetLang: config.TargetLanguage,
			Provider:   providerName,
			APIKey:     apiKey,
			Model:      providerDefaultModel(providerName),
		}

		// Create LLM translator
		llmTranslator, err := llm.NewLLMTranslator(tConfig)
		if err != nil {
			log.Printf("Warning: Failed to create %s translator: %v", providerName, err)
			continue
		}

		providers = append(providers, llmTranslator)
	}

	return providers, nil
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
	var chapterAnalyses []ChapterAnalysis
	if pc.config.AnalyzeChapters {
		log.Printf("  Analyzing individual chapters...")
		analyses, err := pc.analyzeChapters(ctx, book)
		if err != nil {
			log.Printf("  Warning: Chapter analysis failed: %v", err)
		} else {
			chapterAnalyses = analyses
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

	// Chapter analyses are deterministic per-chapter artifacts that the
	// consolidation LLM is never asked to reproduce, so a consolidated
	// FinalAnalysis comes back without them. Re-attach the chapter analyses
	// we computed so the downstream translator actually receives per-chapter
	// context instead of silently losing the chapter-analysis work.
	if len(chapterAnalyses) > 0 && len(result.FinalAnalysis.ChapterAnalyses) == 0 {
		result.FinalAnalysis.ChapterAnalyses = chapterAnalyses
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
	var wg sync.WaitGroup

	// Results are written to a per-chapter slot indexed by chapter position so
	// the returned slice is ALWAYS in chapter order regardless of goroutine
	// completion order. Each analysis is additionally stamped with its
	// authoritative ChapterNum below, so downstream GetTranslationContext can
	// attribute by number (robust to the slice being compacted after a failed
	// chapter is dropped). Distinct goroutines write distinct indices, so no
	// mutex is required.
	slots := make([]*ChapterAnalysis, len(book.Chapters))

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
		go func(chapterIdx int, ch ebook.Chapter) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			// Extract chapter content
			chapterContent := pc.extractChapterContent(&ch)

			// Build prompt
			prompt := promptBuilder.BuildChapterAnalysisPrompt(
				chapterIdx+1,
				ch.Title,
				chapterContent,
			)

			// Call LLM
			response, err := provider.Translate(ctx, prompt, "")
			if err != nil {
				log.Printf("    Warning: Chapter %d analysis failed: %v", chapterIdx+1, err)
				return
			}

			// Parse response
			var analysis ChapterAnalysis
			if err := json.Unmarshal([]byte(extractJSON(response)), &analysis); err != nil {
				log.Printf("    Warning: Failed to parse chapter %d analysis: %v", chapterIdx+1, err)
				return
			}

			// Stamp the AUTHORITATIVE chapter number we already know. The LLM is
			// the only source of ChapterNum otherwise, and it frequently omits the
			// field (leaving it 0) or returns a wrong/duplicated value. Either case
			// defeats the by-number attribution in lookupChapterAnalysis: an all-zero
			// slice silently degrades to positional indexing, which mis-attributes
			// every surviving chapter once a failed chapter is compacted out. The
			// coordinator owns the ground truth (chapterIdx+1), so override whatever
			// the LLM returned.
			analysis.ChapterNum = chapterIdx + 1

			slots[chapterIdx] = &analysis

			log.Printf("    ✓ Chapter %d analyzed", chapterIdx+1)
		}(i, chapter)
	}

	wg.Wait()

	// Compact into chapter order, dropping chapters whose analysis failed.
	analyses := make([]ChapterAnalysis, 0, len(slots))
	for _, a := range slots {
		if a != nil {
			analyses = append(analyses, *a)
		}
	}

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

		// Add sections (recursively, including nested subsections).
		for i := range chapter.Sections {
			writeSectionContent(&content, &chapter.Sections[i])
		}
	}

	return content.String()
}

// extractChapterContent extracts text from a single chapter
func (pc *PreparationCoordinator) extractChapterContent(chapter *ebook.Chapter) string {
	var content strings.Builder
	for i := range chapter.Sections {
		writeSectionContent(&content, &chapter.Sections[i])
	}
	return content.String()
}

// writeSectionContent appends a section's title and content, then recurses into
// its subsections. FB2 (and other nested formats) populate Section.Subsections,
// and the translator recurses into them — so the analysis input MUST include
// nested text too, otherwise the LLM analyses an incomplete chapter and the
// resulting terminology / caveats / context silently miss everything in the
// subsections.
//
// Section.Title is included as well: translateSection translates section and
// subsection titles, so a title is real end-user-visible text that can carry
// character names, untranslatable terms, and cultural references. Excluding it
// from the analysis input would let those go undetected — the same data-loss
// class as dropping subsection content.
func writeSectionContent(content *strings.Builder, section *ebook.Section) {
	if section.Title != "" {
		content.WriteString(section.Title)
		content.WriteString("\n\n")
	}
	if section.Content != "" {
		content.WriteString(section.Content)
		content.WriteString("\n\n")
	}
	for i := range section.Subsections {
		writeSectionContent(content, &section.Subsections[i])
	}
}

// extractJSON attempts to extract a valid JSON value (object or array) from an
// LLM response.
//
// Real LLMs wrap JSON in explanatory prose and Markdown code fences, and that
// prose can itself contain braces (e.g. "format {key:value}: {...}"). A naive
// first-`{`..last-`}` slice grabs the wrong span in those cases and yields
// malformed JSON that downstream json.Unmarshal silently mis-parses. This
// implementation, in order:
//
//  1. prefers the contents of a fenced ```json / ``` code block when present;
//  2. otherwise scans for the first balanced, string-and-escape-aware JSON
//     value (object `{...}` or array `[...]`) that json.Valid accepts,
//     tolerating braces in surrounding prose by trying successive candidate
//     start positions.
//
// When no parseable JSON value is found it returns the original response
// unchanged so the caller's json.Unmarshal surfaces an honest parse error
// rather than silently consuming garbage.
func extractJSON(response string) string {
	// 1) Prefer a fenced code block if one is present and parseable.
	if fenced, ok := extractFencedJSON(response); ok {
		return fenced
	}

	// 2) Scan for the first balanced JSON value that json.Valid accepts.
	if scanned, ok := scanBalancedJSON(response); ok {
		return scanned
	}

	// 3) No JSON found — return as-is for an honest downstream parse error.
	return response
}

// extractFencedJSON returns the JSON value inside the first Markdown code fence
// (```json ... ``` or ``` ... ```) whose body, after balanced scanning, is
// valid JSON. ok is false when there is no fence or its contents are not JSON.
func extractFencedJSON(response string) (string, bool) {
	rest := response
	for {
		open := strings.Index(rest, "```")
		if open == -1 {
			return "", false
		}
		// Skip past the opening fence and an optional language tag line.
		afterOpen := rest[open+3:]
		if nl := strings.IndexByte(afterOpen, '\n'); nl != -1 {
			// Treat everything up to the newline as a (possibly empty) info string.
			afterOpen = afterOpen[nl+1:]
		}
		close := strings.Index(afterOpen, "```")
		if close == -1 {
			// Unterminated fence: scan the remainder of the body directly.
			if scanned, ok := scanBalancedJSON(afterOpen); ok {
				return scanned, true
			}
			return "", false
		}
		body := afterOpen[:close]
		if scanned, ok := scanBalancedJSON(body); ok {
			return scanned, true
		}
		// This fence held no JSON; continue past it to any later fence.
		rest = afterOpen[close+3:]
	}
}

// scanBalancedJSON finds the first balanced JSON value (object or array) in s
// that json.Valid accepts. It is string-aware (braces/brackets inside JSON
// string literals are ignored) and escape-aware (`\"` does not end a string).
// Candidate start positions are tried left-to-right so leading prose braces do
// not derail extraction.
func scanBalancedJSON(s string) (string, bool) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '{' && c != '[' {
			continue
		}
		if end, ok := matchBalanced(s, i); ok {
			candidate := s[i : end+1]
			if json.Valid([]byte(candidate)) {
				return candidate, true
			}
		}
	}
	return "", false
}

// matchBalanced returns the index of the closing delimiter that balances the
// opening delimiter at start, honoring nested objects/arrays and ignoring
// delimiters that appear inside JSON string literals. ok is false when the
// value is not balanced before the end of s.
func matchBalanced(s string, start int) (int, bool) {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

// estimateTokens roughly estimates token count
func estimateTokens(text string) int {
	// Rough estimate: ~4 characters per token
	return len(text) / 4
}
