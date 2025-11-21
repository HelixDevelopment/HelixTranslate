package verification

import (
	"context"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
	"fmt"
	"strings"
	"sync"
)

// PolishingConfig configures the multi-LLM polishing process
type PolishingConfig struct {
	// LLM providers to use for verification (e.g., ["openai", "anthropic", "deepseek"])
	Providers []string

	// Minimum number of LLMs that must agree for a change to be accepted
	MinConsensus int

	// Verification dimensions
	VerifySpirit      bool // Verify if translation preserves the spirit of original
	VerifyLanguage    bool // Verify target language quality and naturalness
	VerifyContext     bool // Verify context and deep meanings
	VerifyVocabulary  bool // Verify word choice and vocabulary richness

	// Translation configurations for each provider
	TranslationConfigs map[string]translator.TranslationConfig
}

// PolishingResult contains detailed results of the polishing process
type PolishingResult struct {
	// Section identification
	SectionID      string
	Location       string

	// Content
	OriginalText   string
	TranslatedText string
	PolishedText   string

	// Changes made
	Changes        []Change

	// Consensus details
	Consensus      int     // Number of LLMs that agreed on changes
	Confidence     float64 // Confidence score (0.0-1.0)

	// Issues found
	Issues         []Issue
	Suggestions    []Suggestion

	// Quality scores per dimension
	SpiritScore      float64
	LanguageScore    float64
	ContextScore     float64
	VocabularyScore  float64
	OverallScore     float64
}

// Change represents a modification made during polishing
type Change struct {
	Location    string  // Where in the text
	Original    string  // Original translated text
	Polished    string  // Polished version
	Reason      string  // Why the change was made
	Agreement   int     // How many LLMs agreed
	Confidence  float64 // Confidence in this change
}

// Issue represents a problem found during verification
type Issue struct {
	Type        string  // "spirit", "language", "context", "vocabulary"
	Severity    string  // "critical", "major", "minor"
	Description string
	Location    string
	Suggestion  string
}

// Suggestion represents an improvement suggestion
type Suggestion struct {
	Type        string  // Type of suggestion
	Description string
	Location    string
	Example     string
}

// LLMVerification holds verification from a single LLM
type llmVerification struct {
	Provider       string
	SpiritScore    float64
	LanguageScore  float64
	ContextScore   float64
	VocabularyScore float64
	Suggestions    []string
	PolishedText   string
	Issues         []Issue
}

// BookPolisher performs multi-LLM verification and polishing
type BookPolisher struct {
	config       PolishingConfig
	translators  map[string]*llm.LLMTranslator
	eventBus     *events.EventBus
	sessionID    string
}

// NewBookPolisher creates a new multi-LLM book polisher
func NewBookPolisher(
	config PolishingConfig,
	eventBus *events.EventBus,
	sessionID string,
) (*BookPolisher, error) {
	// Create LLM translators for each provider
	translators := make(map[string]*llm.LLMTranslator)

	for _, provider := range config.Providers {
		translatorConfig, ok := config.TranslationConfigs[provider]
		if !ok {
			return nil, fmt.Errorf("missing translation config for provider: %s", provider)
		}

		translator, err := llm.NewLLMTranslator(translatorConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create translator for %s: %w", provider, err)
		}

		translators[provider] = translator
	}

	return &BookPolisher{
		config:      config,
		translators: translators,
		eventBus:    eventBus,
		sessionID:   sessionID,
	}, nil
}

// PolishBook performs comprehensive multi-LLM verification and polishing
func (bp *BookPolisher) PolishBook(
	ctx context.Context,
	originalBook *ebook.Book,
	translatedBook *ebook.Book,
) (*ebook.Book, *PolishingReport, error) {
	// Create polished book copy
	polishedBook := translatedBook

	// Initialize report
	report := NewPolishingReport(bp.config)

	bp.emitProgress("Starting multi-LLM verification and polishing", map[string]interface{}{
		"providers":      bp.config.Providers,
		"min_consensus":  bp.config.MinConsensus,
		"total_chapters": len(originalBook.Chapters),
	})

	// Polish metadata
	if err := bp.polishMetadata(ctx, &originalBook.Metadata, &polishedBook.Metadata, report); err != nil {
		return nil, report, err
	}

	// Polish chapters
	totalChapters := len(originalBook.Chapters)
	for i := range originalBook.Chapters {
		select {
		case <-ctx.Done():
			return nil, report, ctx.Err()
		default:
		}

		location := fmt.Sprintf("Chapter %d/%d", i+1, totalChapters)
		bp.emitProgress(fmt.Sprintf("Polishing %s", location), map[string]interface{}{
			"chapter":  i + 1,
			"total":    totalChapters,
			"progress": float64(i+1) / float64(totalChapters) * 100,
		})

		if err := bp.polishChapter(
			ctx,
			&originalBook.Chapters[i],
			&polishedBook.Chapters[i],
			i+1,
			report,
		); err != nil {
			return nil, report, err
		}
	}

	// Finalize report
	report.Finalize()

	bp.emitProgress("Polishing completed", map[string]interface{}{
		"total_changes":    report.TotalChanges,
		"overall_score":    report.OverallScore,
		"consensus_rate":   report.ConsensusRate,
		"spirit_score":     report.AverageSpiritScore,
		"language_score":   report.AverageLanguageScore,
		"context_score":    report.AverageContextScore,
		"vocabulary_score": report.AverageVocabularyScore,
	})

	return polishedBook, report, nil
}

// polishMetadata verifies and polishes book metadata
func (bp *BookPolisher) polishMetadata(
	ctx context.Context,
	original *ebook.Metadata,
	translated *ebook.Metadata,
	report *PolishingReport,
) error {
	// Polish title
	if original.Title != "" && translated.Title != "" {
		result, err := bp.polishSection(
			ctx,
			"metadata_title",
			"Book Title",
			original.Title,
			translated.Title,
		)
		if err != nil {
			return err
		}

		translated.Title = result.PolishedText
		report.AddSectionResult(result)
	}

	// Polish description
	if original.Description != "" && translated.Description != "" {
		result, err := bp.polishSection(
			ctx,
			"metadata_description",
			"Book Description",
			original.Description,
			translated.Description,
		)
		if err != nil {
			return err
		}

		translated.Description = result.PolishedText
		report.AddSectionResult(result)
	}

	return nil
}

// polishChapter verifies and polishes a chapter
func (bp *BookPolisher) polishChapter(
	ctx context.Context,
	original *ebook.Chapter,
	translated *ebook.Chapter,
	chapterNum int,
	report *PolishingReport,
) error {
	location := fmt.Sprintf("Chapter %d", chapterNum)

	// Polish chapter title
	if original.Title != "" && translated.Title != "" {
		result, err := bp.polishSection(
			ctx,
			fmt.Sprintf("chapter_%d_title", chapterNum),
			location+" - Title",
			original.Title,
			translated.Title,
		)
		if err != nil {
			return err
		}

		translated.Title = result.PolishedText
		report.AddSectionResult(result)
	}

	// Polish sections
	for i := range original.Sections {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := bp.polishSectionRecursive(
			ctx,
			&original.Sections[i],
			&translated.Sections[i],
			fmt.Sprintf("%s, Section %d", location, i+1),
			report,
		); err != nil {
			return err
		}
	}

	return nil
}

// polishSectionRecursive verifies and polishes a section recursively
func (bp *BookPolisher) polishSectionRecursive(
	ctx context.Context,
	original *ebook.Section,
	translated *ebook.Section,
	location string,
	report *PolishingReport,
) error {
	// Polish section title
	if original.Title != "" && translated.Title != "" {
		result, err := bp.polishSection(
			ctx,
			fmt.Sprintf("%s_title", strings.ReplaceAll(location, " ", "_")),
			location+" - Title",
			original.Title,
			translated.Title,
		)
		if err != nil {
			return err
		}

		translated.Title = result.PolishedText
		report.AddSectionResult(result)
	}

	// Polish section content
	if original.Content != "" && translated.Content != "" {
		result, err := bp.polishSection(
			ctx,
			fmt.Sprintf("%s_content", strings.ReplaceAll(location, " ", "_")),
			location+" - Content",
			original.Content,
			translated.Content,
		)
		if err != nil {
			return err
		}

		translated.Content = result.PolishedText
		report.AddSectionResult(result)
	}

	// Polish subsections
	for i := range original.Subsections {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := bp.polishSectionRecursive(
			ctx,
			&original.Subsections[i],
			&translated.Subsections[i],
			fmt.Sprintf("%s, Subsection %d", location, i+1),
			report,
		); err != nil {
			return err
		}
	}

	return nil
}

// polishSection performs multi-LLM verification and polishing for a single section
func (bp *BookPolisher) polishSection(
	ctx context.Context,
	sectionID string,
	location string,
	originalText string,
	translatedText string,
) (*PolishingResult, error) {
	// Get verifications from all LLMs in parallel
	verifications := make([]llmVerification, len(bp.config.Providers))
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, len(bp.config.Providers))

	for i, provider := range bp.config.Providers {
		wg.Add(1)
		go func(idx int, prov string) {
			defer wg.Done()

			verification, err := bp.verifyWithLLM(
				ctx,
				prov,
				originalText,
				translatedText,
				location,
			)

			mu.Lock()
			if err != nil {
				errors[idx] = err
			} else {
				verifications[idx] = *verification
			}
			mu.Unlock()
		}(i, provider)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			bp.emitWarning(fmt.Sprintf("Verification failed for %s with %s: %v",
				location, bp.config.Providers[i], err))
		}
	}

	// Build consensus from verifications
	result := bp.buildConsensus(
		sectionID,
		location,
		originalText,
		translatedText,
		verifications,
	)

	return result, nil
}

// verifyWithLLM performs verification with a single LLM
func (bp *BookPolisher) verifyWithLLM(
	ctx context.Context,
	provider string,
	originalText string,
	translatedText string,
	location string,
) (*llmVerification, error) {
	translator := bp.translators[provider]

	// Create verification prompt
	prompt := bp.createVerificationPrompt(originalText, translatedText)

	// Get LLM analysis and polishing
	response, err := translator.Translate(ctx, prompt, location)
	if err != nil {
		return nil, fmt.Errorf("LLM verification failed: %w", err)
	}

	// Parse LLM response
	verification := bp.parseVerificationResponse(provider, response, translatedText)

	return verification, nil
}

// createVerificationPrompt creates the multi-dimensional verification prompt
func (bp *BookPolisher) createVerificationPrompt(originalText, translatedText string) string {
	var dimensions []string

	if bp.config.VerifySpirit {
		dimensions = append(dimensions, "**Spirit**: Does the translation preserve the spirit, tone, and emotional resonance of the original?")
	}
	if bp.config.VerifyLanguage {
		dimensions = append(dimensions, "**Language**: Is the target language natural, idiomatic, and grammatically correct?")
	}
	if bp.config.VerifyContext {
		dimensions = append(dimensions, "**Context**: Are all contexts, deep meanings, and nuances properly conveyed?")
	}
	if bp.config.VerifyVocabulary {
		dimensions = append(dimensions, "**Vocabulary**: Is the word choice rich, appropriate, and varied?")
	}

	dimensionsList := strings.Join(dimensions, "\n")

	return fmt.Sprintf(`You are a professional translation quality assessor and polisher. Your task is to verify and improve a literary translation.

**Original Text (Russian):**
%s

**Current Translation (Serbian):**
%s

**Verification Dimensions:**
%s

**CRITICAL REQUIREMENT - Ekavica Dialect:**
All Serbian translations MUST use ONLY Ekavica dialect (екавица), the standard dialect of Serbia.
- Use "е" instead of "ије/је": mleko (not mlijeko), dete (not dijete), pesma (not pjesma)
- Use hteo (not htio), lepo (not lijepo), reka (not rijeka)
- ANY use of Ijekavica forms (ије/је) is a CRITICAL ERROR that must be corrected

**CRITICAL REQUIREMENT - Pure Serbian Vocabulary:**
When translating to Serbian, ONLY use pure Serbian vocabulary. Replace any Croatian, Bosnian, or Montenegrin word choices with standard Serbian equivalents.
- Use standard Serbian words preferred in Serbia, not regional variants from other countries
- Avoid Croatianisms, Bosnianisms, or Montenegrin-specific vocabulary
- This ensures the translation is natural and idiomatic for Serbian readers in Serbia
- Example differences to avoid: Croatian "zrakoplov" → Serbian "avion", Croatian "kazalište" → Serbian "pozorište"

**Your Task:**
1. Evaluate the translation on each dimension listed above
2. Score each dimension from 0.0 to 1.0 (where 1.0 is perfect)
3. **CRITICAL**: Check for Ijekavica dialect usage - this is mandatory verification
4. Identify any issues or improvements needed
5. Provide a polished version if improvements are needed (always use Ekavica)

**Response Format:**
SPIRIT_SCORE: [0.0-1.0]
LANGUAGE_SCORE: [0.0-1.0]
CONTEXT_SCORE: [0.0-1.0]
VOCABULARY_SCORE: [0.0-1.0]

ISSUES:
[List any issues found, one per line with format "TYPE: description"]
[MUST include "DIALECT: Uses Ijekavica instead of Ekavica" if any Ijekavica forms detected]
[MUST include "VOCABULARY: Uses Croatian/Bosnian/Montenegrin words" if any regional vocabulary detected]

POLISHED_TEXT:
[Your improved version in Ekavica dialect, or UNCHANGED if translation is perfect]

EXPLANATION:
[Brief explanation of changes made and why, especially any dialect corrections]`,
		originalText,
		translatedText,
		dimensionsList)
}

// parseVerificationResponse parses LLM verification response
func (bp *BookPolisher) parseVerificationResponse(
	provider string,
	response string,
	originalTranslation string,
) *llmVerification {
	verification := &llmVerification{
		Provider:        provider,
		SpiritScore:     0.9, // Default scores
		LanguageScore:   0.9,
		ContextScore:    0.9,
		VocabularyScore: 0.9,
		Suggestions:     make([]string, 0),
		PolishedText:    originalTranslation,
		Issues:          make([]Issue, 0),
	}

	// Parse scores
	if score := extractScore(response, "SPIRIT_SCORE:"); score >= 0 {
		verification.SpiritScore = score
	}
	if score := extractScore(response, "LANGUAGE_SCORE:"); score >= 0 {
		verification.LanguageScore = score
	}
	if score := extractScore(response, "CONTEXT_SCORE:"); score >= 0 {
		verification.ContextScore = score
	}
	if score := extractScore(response, "VOCABULARY_SCORE:"); score >= 0 {
		verification.VocabularyScore = score
	}

	// Extract polished text
	if polished := extractSection(response, "POLISHED_TEXT:", "EXPLANATION:"); polished != "" {
		polished = strings.TrimSpace(polished)
		if polished != "UNCHANGED" && polished != "" {
			verification.PolishedText = polished
		}
	}

	// Extract issues
	if issuesText := extractSection(response, "ISSUES:", "POLISHED_TEXT:"); issuesText != "" {
		lines := strings.Split(issuesText, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				issueType := strings.ToLower(strings.TrimSpace(parts[0]))
				description := strings.TrimSpace(parts[1])

				verification.Issues = append(verification.Issues, Issue{
					Type:        issueType,
					Severity:    "minor",
					Description: description,
					Location:    "",
				})
			}
		}
	}

	return verification
}

// buildConsensus builds consensus from multiple LLM verifications
func (bp *BookPolisher) buildConsensus(
	sectionID string,
	location string,
	originalText string,
	translatedText string,
	verifications []llmVerification,
) *PolishingResult {
	result := &PolishingResult{
		SectionID:      sectionID,
		Location:       location,
		OriginalText:   originalText,
		TranslatedText: translatedText,
		PolishedText:   translatedText, // Default to original
		Changes:        make([]Change, 0),
		Issues:         make([]Issue, 0),
		Suggestions:    make([]Suggestion, 0),
	}

	// Calculate average scores
	totalSpirit := 0.0
	totalLanguage := 0.0
	totalContext := 0.0
	totalVocabulary := 0.0
	count := float64(len(verifications))

	for _, v := range verifications {
		totalSpirit += v.SpiritScore
		totalLanguage += v.LanguageScore
		totalContext += v.ContextScore
		totalVocabulary += v.VocabularyScore

		// Collect issues
		for _, issue := range v.Issues {
			issue.Location = location
			result.Issues = append(result.Issues, issue)
		}
	}

	if count > 0 {
		result.SpiritScore = totalSpirit / count
		result.LanguageScore = totalLanguage / count
		result.ContextScore = totalContext / count
		result.VocabularyScore = totalVocabulary / count
		result.OverallScore = (result.SpiritScore + result.LanguageScore +
			result.ContextScore + result.VocabularyScore) / 4.0
	}

	// Check consensus for polishing
	polishedVersions := make(map[string]int)
	for _, v := range verifications {
		if v.PolishedText != translatedText {
			polishedVersions[v.PolishedText]++
		}
	}

	// Find most agreed-upon polished version
	maxAgreement := 0
	bestPolished := translatedText

	for polished, agreement := range polishedVersions {
		if agreement > maxAgreement {
			maxAgreement = agreement
			bestPolished = polished
		}
	}

	result.Consensus = maxAgreement
	result.Confidence = float64(maxAgreement) / count

	// Apply polished version if consensus reached
	if maxAgreement >= bp.config.MinConsensus {
		result.PolishedText = bestPolished

		// Record change
		if bestPolished != translatedText {
			result.Changes = append(result.Changes, Change{
				Location:   location,
				Original:   translatedText,
				Polished:   bestPolished,
				Reason:     "Multi-LLM consensus improvement",
				Agreement:  maxAgreement,
				Confidence: result.Confidence,
			})
		}
	}

	return result
}

// Helper functions

func extractScore(text, prefix string) float64 {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			scoreStr := strings.TrimPrefix(strings.TrimSpace(line), prefix)
			scoreStr = strings.TrimSpace(scoreStr)

			var score float64
			if _, err := fmt.Sscanf(scoreStr, "%f", &score); err == nil {
				if score >= 0.0 && score <= 1.0 {
					return score
				}
			}
		}
	}
	return -1.0
}

func extractSection(text, startMarker, endMarker string) string {
	startIdx := strings.Index(text, startMarker)
	if startIdx == -1 {
		return ""
	}

	startIdx += len(startMarker)

	endIdx := strings.Index(text[startIdx:], endMarker)
	if endIdx == -1 {
		return strings.TrimSpace(text[startIdx:])
	}

	return strings.TrimSpace(text[startIdx : startIdx+endIdx])
}

func (bp *BookPolisher) emitProgress(message string, data map[string]interface{}) {
	if bp.eventBus != nil {
		event := events.NewEvent("polishing_progress", message, data)
		event.SessionID = bp.sessionID
		bp.eventBus.Publish(event)
	}
}

func (bp *BookPolisher) emitWarning(message string) {
	if bp.eventBus != nil {
		event := events.NewEvent("polishing_warning", message, nil)
		event.SessionID = bp.sessionID
		bp.eventBus.Publish(event)
	}
}
