package preparation

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PreparationPromptBuilder builds prompts for content analysis
type PreparationPromptBuilder struct {
	sourceLang  string
	targetLang  string
	passNumber  int
	previousAnalysis *ContentAnalysis
}

// NewPreparationPromptBuilder creates a new prompt builder
func NewPreparationPromptBuilder(sourceLang, targetLang string, passNumber int) *PreparationPromptBuilder {
	return &PreparationPromptBuilder{
		sourceLang: sourceLang,
		targetLang: targetLang,
		passNumber: passNumber,
	}
}

// WithPreviousAnalysis adds previous pass analysis for refinement
func (b *PreparationPromptBuilder) WithPreviousAnalysis(analysis *ContentAnalysis) *PreparationPromptBuilder {
	b.previousAnalysis = analysis
	return b
}

// BuildInitialAnalysisPrompt creates the prompt for the first analysis pass
func (b *PreparationPromptBuilder) BuildInitialAnalysisPrompt(content string) string {
	return fmt.Sprintf(`You are a professional translator and literary analyst preparing for high-quality translation from %s to %s.

Your task is to perform a COMPREHENSIVE CONTENT ANALYSIS before translation begins. This analysis will guide the translation process to ensure accuracy, cultural sensitivity, and stylistic appropriateness.

## CONTENT TO ANALYZE:
%s

## ANALYSIS REQUIREMENTS:

### 1. CONTENT CLASSIFICATION
- **Content Type**: Determine if this is a novel, short story, poem, technical documentation, legal text, medical literature, scientific paper, business document, etc.
- **Genre**: Identify the primary genre (e.g., detective fiction, romance, science fiction, horror, literary fiction, etc.)
- **Subgenres**: List specific subgenres (e.g., noir detective, psychological thriller, hard science fiction, etc.)

### 2. LANGUAGE AND STYLE
- **Tone**: Describe the overall tone (formal, informal, poetic, technical, conversational, archaic, etc.)
- **Language Style**: Identify literary devices, sentence structure patterns, vocabulary level, narrative voice
- **Target Audience**: Who is this written for? (age group, education level, professional field, etc.)

### 3. UNTRANSLATABLE TERMS
Identify terms that should be KEPT IN ORIGINAL LANGUAGE:
- Proper nouns (names, places)
- Culture-specific terms without direct equivalents
- Technical jargon that is internationally recognized
- Terms where translation would lose critical meaning
- For EACH term provide: original form, transliteration (if needed), reason, and contexts where it appears

### 4. FOOTNOTE GUIDANCE
Identify concepts that will need clarification for %s readers:
- Cultural references unfamiliar to target audience
- Historical context
- Wordplay or puns that don't translate directly
- Idiomatic expressions
- For EACH, provide: term/concept, explanation needed, priority (high/medium/low)

### 5. CHARACTERS (if narrative content)
For each significant character:
- Name and alternate names
- Role (protagonist, antagonist, supporting, etc.)
- Speech patterns (dialect, formality, unique quirks)
- Key character traits
- How their name should be handled in translation

### 6. CULTURAL REFERENCES
Identify all culture-specific references:
- References to literature, art, music, film
- Historical events
- Social customs and traditions
- Food, clothing, architecture specific to source culture
- For EACH: explain what it is, why it matters, how it should be handled (keep original, translate, add explanation)

### 7. KEY THEMES
List the main themes and motifs that must be preserved in translation

## OUTPUT FORMAT:
Provide your analysis in JSON format matching this structure:
{
  "content_type": "...",
  "genre": "...",
  "subgenres": ["..."],
  "tone": "...",
  "language_style": "...",
  "target_audience": "...",
  "untranslatable_terms": [
    {
      "term": "...",
      "original_script": "...",
      "reason": "...",
      "context": ["..."],
      "transliteration": "..."
    }
  ],
  "footnote_guidance": [
    {
      "term": "...",
      "explanation": "...",
      "locations": ["..."],
      "priority": "high|medium|low"
    }
  ],
  "characters": [
    {
      "name": "...",
      "alternate_names": ["..."],
      "role": "...",
      "speech_pattern": "...",
      "key_traits": ["..."],
      "name_translation": {"sr": "..."}
    }
  ],
  "key_themes": ["..."],
  "cultural_references": [
    {
      "reference": "...",
      "origin": "...",
      "explanation": "...",
      "handling": "..."
    }
  ]
}

Provide ONLY the JSON output, no additional text.`, b.sourceLang, b.targetLang, truncateContent(content, 15000), b.targetLang)
}

// BuildRefinementPrompt creates a prompt to refine previous analysis
func (b *PreparationPromptBuilder) BuildRefinementPrompt(content string) string {
	if b.previousAnalysis == nil {
		return b.BuildInitialAnalysisPrompt(content)
	}

	prevJSON, _ := json.MarshalIndent(b.previousAnalysis, "", "  ")

	return fmt.Sprintf(`You are a professional translator and literary analyst conducting Pass #%d of content analysis.

## PREVIOUS ANALYSIS (Pass #%d):
%s

## CONTENT TO ANALYZE:
%s

## YOUR TASK:
Review and IMPROVE previous analysis. Focus on:

1. **Validation**: Verify all identifications are accurate
2. **Completeness**: Find what was missed
   - Additional untranslatable terms
   - More cultural references
   - Subtle nuances in tone or style
   - Character details that weren't captured
3. **Refinement**: Improve explanations and guidance
   - Make footnote explanations clearer
   - Add more context where needed
   - Clarify ambiguous points
4. **Prioritization**: Adjust priorities based on importance
5. **Consolidation**: Merge duplicate entries, organize better

## SPECIFIC IMPROVEMENTS TO MAKE:
- Check if content_type and genre classifications are precise
- Ensure ALL significant untranslatable terms are captured
- Verify cultural references are explained adequately for %s readers
- Confirm character speech patterns are accurately described
- Validate that key themes are comprehensive

## OUTPUT FORMAT:
Provide your IMPROVED analysis in the same JSON format:
{
  "content_type": "...",
  "genre": "...",
  "subgenres": ["..."],
  ...
}

This should be your ENHANCED version, not just a copy of the previous analysis.
Provide ONLY the JSON output, no additional text.`, b.passNumber, b.passNumber-1, string(prevJSON), truncateContent(content, 15000), b.targetLang)
}

// BuildChapterAnalysisPrompt creates a prompt for analyzing a specific chapter
func (b *PreparationPromptBuilder) BuildChapterAnalysisPrompt(chapterNum int, chapterTitle, chapterContent string) string {
	return fmt.Sprintf(`You are analyzing Chapter %d for translation preparation from %s to %s.

## CHAPTER INFORMATION:
**Number**: %d
**Title**: %s
**Content**:
%s

## ANALYSIS REQUIREMENTS:

### 1. SUMMARY
Provide a concise summary (2-3 sentences) of what happens in this chapter.

### 2. KEY POINTS
List the most important points/events/information in this chapter (4-6 bullet points).

### 3. TRANSLATION CAVEATS
Identify specific challenges for translating THIS chapter:
- Complex terminology
- Cultural references specific to this chapter
- Tone shifts
- Character introductions or developments
- Timeline or setting changes
- Any other translation challenges

### 4. TONE ANALYSIS
Describe the specific tone of this chapter (may differ from overall work):
- Tense, relaxed, humorous, somber, etc.
- Narrative pace (fast, slow, varied)
- Emotional register

### 5. COMPLEXITY ASSESSMENT
Rate the translation complexity: Simple / Moderate / Complex
Explain why.

### 6. SPECIAL NOTES
Any other observations relevant to translation.

## OUTPUT FORMAT:
Provide your analysis in JSON format:
{
  "chapter_id": "chapter_%d",
  "chapter_num": %d,
  "title": "%s",
  "summary": "...",
  "key_points": ["...", "..."],
  "caveats": ["...", "..."],
  "tone": "...",
  "complexity": "simple|moderate|complex",
  "special_notes": "..."
}

Provide ONLY the JSON output, no additional text.`, chapterNum, b.sourceLang, b.targetLang, chapterNum, chapterTitle,
		truncateContent(chapterContent, 10000), chapterNum, chapterNum, chapterTitle)
}

// BuildConsolidationPrompt creates a prompt to consolidate multiple analyses
func (b *PreparationPromptBuilder) BuildConsolidationPrompt(analyses []ContentAnalysis) string {
	var analysesJSON strings.Builder
	for _, analysis := range analyses {
		jsonData, _ := json.MarshalIndent(analysis, "    ", "  ")
		analysesJSON.WriteString(fmt.Sprintf("\n### Analysis from Pass #%d (Provider: %s):\n",
			analysis.AnalysisVersion, analysis.AnalyzedBy))
		analysesJSON.WriteString(string(jsonData))
		analysesJSON.WriteString("\n")
	}

	return fmt.Sprintf(`You are creating the FINAL CONSOLIDATED ANALYSIS from multiple preparation passes.

## ANALYSES TO CONSOLIDATE:
%s

## YOUR TASK:
Create the DEFINITIVE, HIGHEST-QUALITY analysis by:

1. **Merging**: Combine insights from all passes
2. **Validating**: Include only accurate, verified information
3. **Deduplicating**: Remove redundant entries
4. **Prioritizing**: Keep the most important items
5. **Clarifying**: Use the clearest explanations
6. **Organizing**: Present information logically

## CONSOLIDATION GUIDELINES:
- If multiple passes agree on something, it's likely correct
- If passes disagree, use your judgment to pick the most accurate
- Include items that appear in ANY pass if they're valid
- For untranslatable terms: merge similar entries, keep all valid ones
- For footnotes: consolidate similar concepts, prioritize the most important
- For characters: merge information, keep the most comprehensive descriptions
- For cultural references: combine explanations for completeness

## OUTPUT FORMAT:
Provide the FINAL consolidated analysis in JSON format (same structure as individual analyses).
This will be the definitive guide for translation.

Provide ONLY the JSON output, no additional text.`, analysesJSON.String())
}

// truncateContent truncates content to maxChars while trying to preserve sentence boundaries
func truncateContent(content string, maxChars int) string {
	if len(content) <= maxChars {
		return content
	}

	// Try to truncate at sentence boundary
	truncated := content[:maxChars]
	lastPeriod := strings.LastIndex(truncated, ".")
	lastNewline := strings.LastIndex(truncated, "\n\n")

	cutPoint := maxChars
	if lastPeriod > maxChars-200 && lastPeriod > 0 {
		cutPoint = lastPeriod + 1
	} else if lastNewline > maxChars-200 && lastNewline > 0 {
		cutPoint = lastNewline
	}

	truncatedContent := content[:cutPoint]
	if truncatedContent == "" {
		// If nothing is left, just return the truncation indicator
		return "[... content truncated for analysis ...]"
	}
	
	// Only add newlines if content doesn't already end with them
	if strings.HasSuffix(truncatedContent, "\n\n") {
		return truncatedContent + "[... content truncated for analysis ...]"
	}
	return truncatedContent + "\n\n[... content truncated for analysis ...]"
}
