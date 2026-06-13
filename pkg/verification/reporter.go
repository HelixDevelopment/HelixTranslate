package verification

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// PolishingReport contains comprehensive details of the polishing process
type PolishingReport struct {
	// Configuration
	Config PolishingConfig

	// Timing
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration

	// Section results
	SectionResults []*PolishingResult

	// Aggregate statistics
	TotalSections    int
	TotalChanges     int
	TotalIssues      int
	TotalSuggestions int

	// Consensus statistics
	ConsensusRate     float64 // Percentage of sections where consensus was reached
	AverageConfidence float64

	// Quality scores
	AverageSpiritScore     float64
	AverageLanguageScore   float64
	AverageContextScore    float64
	AverageVocabularyScore float64
	OverallScore           float64

	// Issue breakdown by type
	IssuesByType     map[string]int
	IssuesBySeverity map[string]int

	// Provider statistics
	ProviderAgreements map[string]int // How often each provider agreed with consensus
	ProviderScores     map[string]float64

	// Top issues and changes
	TopIssues          []Issue
	SignificantChanges []Change
}

// NewPolishingReport creates a new polishing report
func NewPolishingReport(config PolishingConfig) *PolishingReport {
	return &PolishingReport{
		Config:             config,
		StartTime:          time.Now(),
		SectionResults:     make([]*PolishingResult, 0),
		IssuesByType:       make(map[string]int),
		IssuesBySeverity:   make(map[string]int),
		ProviderAgreements: make(map[string]int),
		ProviderScores:     make(map[string]float64),
		TopIssues:          make([]Issue, 0),
		SignificantChanges: make([]Change, 0),
	}
}

// AddSectionResult adds a section result to the report
func (pr *PolishingReport) AddSectionResult(result *PolishingResult) {
	pr.SectionResults = append(pr.SectionResults, result)

	// Update statistics
	pr.TotalSections++
	pr.TotalChanges += len(result.Changes)
	pr.TotalIssues += len(result.Issues)
	pr.TotalSuggestions += len(result.Suggestions)

	// Track issues by type and severity
	for _, issue := range result.Issues {
		pr.IssuesByType[issue.Type]++
		pr.IssuesBySeverity[issue.Severity]++

		// Track significant issues
		if issue.Severity == "critical" || issue.Severity == "major" {
			pr.TopIssues = append(pr.TopIssues, issue)
		}
	}

	// Track significant changes (high confidence)
	for _, change := range result.Changes {
		if change.Confidence >= 0.8 {
			pr.SignificantChanges = append(pr.SignificantChanges, change)
		}
	}
}

// Finalize completes the report calculations
func (pr *PolishingReport) Finalize() {
	pr.EndTime = time.Now()
	pr.Duration = pr.EndTime.Sub(pr.StartTime)

	if pr.TotalSections == 0 {
		return
	}

	// Calculate averages
	totalSpirit := 0.0
	totalLanguage := 0.0
	totalContext := 0.0
	totalVocabulary := 0.0
	totalConfidence := 0.0
	consensusCount := 0

	for _, result := range pr.SectionResults {
		totalSpirit += result.SpiritScore
		totalLanguage += result.LanguageScore
		totalContext += result.ContextScore
		totalVocabulary += result.VocabularyScore
		totalConfidence += result.Confidence

		if result.Consensus >= pr.Config.MinConsensus {
			consensusCount++
		}
	}

	count := float64(pr.TotalSections)
	pr.AverageSpiritScore = totalSpirit / count
	pr.AverageLanguageScore = totalLanguage / count
	pr.AverageContextScore = totalContext / count
	pr.AverageVocabularyScore = totalVocabulary / count
	pr.AverageConfidence = totalConfidence / count
	pr.OverallScore = (pr.AverageSpiritScore + pr.AverageLanguageScore +
		pr.AverageContextScore + pr.AverageVocabularyScore) / 4.0
	pr.ConsensusRate = float64(consensusCount) / count * 100.0

	// Sort top issues by severity
	sort.Slice(pr.TopIssues, func(i, j int) bool {
		severityOrder := map[string]int{"critical": 0, "major": 1, "minor": 2}
		return severityOrder[pr.TopIssues[i].Severity] < severityOrder[pr.TopIssues[j].Severity]
	})

	// Limit top issues
	if len(pr.TopIssues) > 50 {
		pr.TopIssues = pr.TopIssues[:50]
	}

	// Sort significant changes by confidence
	sort.Slice(pr.SignificantChanges, func(i, j int) bool {
		return pr.SignificantChanges[i].Confidence > pr.SignificantChanges[j].Confidence
	})

	// Limit significant changes
	if len(pr.SignificantChanges) > 100 {
		pr.SignificantChanges = pr.SignificantChanges[:100]
	}
}

// GenerateMarkdownReport generates a detailed markdown report
//
//nolint:funlen
func (pr *PolishingReport) GenerateMarkdownReport() string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Translation Polishing Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n",
		pr.EndTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n\n",
		pr.Duration.Round(time.Second)))

	// Configuration
	sb.WriteString("## Configuration\n\n")
	sb.WriteString(fmt.Sprintf("- **LLM Providers:** %s\n",
		strings.Join(pr.Config.Providers, ", ")))
	sb.WriteString(fmt.Sprintf("- **Minimum Consensus:** %d/%d providers\n",
		pr.Config.MinConsensus, len(pr.Config.Providers)))
	sb.WriteString("- **Verification Dimensions:**\n")
	if pr.Config.VerifySpirit {
		sb.WriteString("  - ✅ Spirit & Tone\n")
	}
	if pr.Config.VerifyLanguage {
		sb.WriteString("  - ✅ Language Quality\n")
	}
	if pr.Config.VerifyContext {
		sb.WriteString("  - ✅ Context & Meaning\n")
	}
	if pr.Config.VerifyVocabulary {
		sb.WriteString("  - ✅ Vocabulary Richness\n")
	}
	sb.WriteString("\n")

	// Executive Summary
	sb.WriteString("## Executive Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Sections Verified:** %d\n", pr.TotalSections))
	sb.WriteString(fmt.Sprintf("- **Total Changes Made:** %d\n", pr.TotalChanges))
	sb.WriteString(fmt.Sprintf("- **Consensus Rate:** %.1f%%\n", pr.ConsensusRate))
	sb.WriteString(fmt.Sprintf("- **Average Confidence:** %.1f%%\n", pr.AverageConfidence*100))
	sb.WriteString(fmt.Sprintf("- **Overall Quality Score:** %.1f%%\n", pr.OverallScore*100))
	sb.WriteString("\n")

	// Quality Scores
	sb.WriteString("## Quality Scores\n\n")
	sb.WriteString("| Dimension | Score | Grade |\n")
	sb.WriteString("|-----------|-------|-------|\n")
	sb.WriteString(fmt.Sprintf("| **Spirit & Tone** | %.1f%% | %s |\n",
		pr.AverageSpiritScore*100, getGrade(pr.AverageSpiritScore)))
	sb.WriteString(fmt.Sprintf("| **Language Quality** | %.1f%% | %s |\n",
		pr.AverageLanguageScore*100, getGrade(pr.AverageLanguageScore)))
	sb.WriteString(fmt.Sprintf("| **Context & Meaning** | %.1f%% | %s |\n",
		pr.AverageContextScore*100, getGrade(pr.AverageContextScore)))
	sb.WriteString(fmt.Sprintf("| **Vocabulary Richness** | %.1f%% | %s |\n",
		pr.AverageVocabularyScore*100, getGrade(pr.AverageVocabularyScore)))
	sb.WriteString(fmt.Sprintf("| **Overall** | %.1f%% | %s |\n",
		pr.OverallScore*100, getGrade(pr.OverallScore)))
	sb.WriteString("\n")

	// Issues Summary
	if pr.TotalIssues > 0 {
		sb.WriteString("## Issues Summary\n\n")
		sb.WriteString(fmt.Sprintf("**Total Issues Found:** %d\n\n", pr.TotalIssues))

		// By severity
		if len(pr.IssuesBySeverity) > 0 {
			sb.WriteString("### By Severity\n\n")
			for severity, count := range pr.IssuesBySeverity {
				icon := "ℹ️"
				if severity == "critical" {
					icon = "🔴"
				} else if severity == "major" {
					icon = "🟠"
				}
				sb.WriteString(fmt.Sprintf("- %s **%s:** %d\n", icon, severity, count))
			}
			sb.WriteString("\n")
		}

		// By type
		if len(pr.IssuesByType) > 0 {
			sb.WriteString("### By Type\n\n")
			for issueType, count := range pr.IssuesByType {
				sb.WriteString(fmt.Sprintf("- **%s:** %d\n", issueType, count))
			}
			sb.WriteString("\n")
		}
	}

	// Top Issues
	if len(pr.TopIssues) > 0 {
		sb.WriteString("## Top Issues\n\n")
		sb.WriteString("These are the most significant issues found:\n\n")

		displayCount := len(pr.TopIssues)
		if displayCount > 20 {
			displayCount = 20
		}

		for i := 0; i < displayCount; i++ {
			issue := pr.TopIssues[i]
			icon := "ℹ️"
			if issue.Severity == "critical" {
				icon = "🔴"
			} else if issue.Severity == "major" {
				icon = "🟠"
			}

			sb.WriteString(fmt.Sprintf("### %s %s - %s\n\n",
				icon, cases.Title(language.English, cases.Compact).String(issue.Severity), issue.Location))
			sb.WriteString(fmt.Sprintf("**Type:** %s\n\n", issue.Type))
			sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", issue.Description))
			if issue.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("**Suggestion:** %s\n\n", issue.Suggestion))
			}
			sb.WriteString("---\n\n")
		}
	}

	// Significant Changes
	if len(pr.SignificantChanges) > 0 {
		sb.WriteString("## Significant Changes\n\n")
		sb.WriteString("These are the most impactful improvements made:\n\n")

		displayCount := len(pr.SignificantChanges)
		if displayCount > 30 {
			displayCount = 30
		}

		for i := 0; i < displayCount; i++ {
			change := pr.SignificantChanges[i]
			sb.WriteString(fmt.Sprintf("### %s\n\n", change.Location))
			sb.WriteString(fmt.Sprintf("**Confidence:** %.1f%% (%d/%d LLMs agreed)\n\n",
				change.Confidence*100, change.Agreement, len(pr.Config.Providers)))
			sb.WriteString(fmt.Sprintf("**Reason:** %s\n\n", change.Reason))
			sb.WriteString("**Original:**\n```\n")
			sb.WriteString(truncateForDisplay(change.Original, 200))
			sb.WriteString("\n```\n\n")
			sb.WriteString("**Polished:**\n```\n")
			sb.WriteString(truncateForDisplay(change.Polished, 200))
			sb.WriteString("\n```\n\n")
			sb.WriteString("---\n\n")
		}
	}

	// Detailed Section Results
	sb.WriteString("## Detailed Section Results\n\n")
	sb.WriteString("Complete verification results for all sections:\n\n")

	for _, result := range pr.SectionResults {
		sb.WriteString(fmt.Sprintf("### %s\n\n", result.Location))

		// Scores
		sb.WriteString("**Quality Scores:**\n")
		sb.WriteString(fmt.Sprintf("- Spirit: %.1f%%\n", result.SpiritScore*100))
		sb.WriteString(fmt.Sprintf("- Language: %.1f%%\n", result.LanguageScore*100))
		sb.WriteString(fmt.Sprintf("- Context: %.1f%%\n", result.ContextScore*100))
		sb.WriteString(fmt.Sprintf("- Vocabulary: %.1f%%\n", result.VocabularyScore*100))
		sb.WriteString(fmt.Sprintf("- Overall: %.1f%%\n\n", result.OverallScore*100))

		// Consensus
		if result.Consensus >= pr.Config.MinConsensus {
			sb.WriteString(fmt.Sprintf("**Consensus:** ✅ %d/%d providers agreed\n\n",
				result.Consensus, len(pr.Config.Providers)))
		} else {
			sb.WriteString(fmt.Sprintf("**Consensus:** ❌ No consensus (%d/%d required)\n\n",
				pr.Config.MinConsensus, len(pr.Config.Providers)))
		}

		// Changes
		if len(result.Changes) > 0 {
			sb.WriteString("**Changes Made:**\n")
			for _, change := range result.Changes {
				sb.WriteString(fmt.Sprintf("- %s (confidence: %.1f%%)\n",
					change.Reason, change.Confidence*100))
			}
			sb.WriteString("\n")
		}

		// Issues
		if len(result.Issues) > 0 {
			sb.WriteString("**Issues Found:**\n")
			for _, issue := range result.Issues {
				sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n",
					issue.Severity, issue.Type, issue.Description))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("---\n\n")
	}

	// Footer
	sb.WriteString("## Conclusion\n\n")
	if pr.OverallScore >= 0.95 {
		sb.WriteString("✅ **Excellent** - Translation quality is outstanding.\n")
	} else if pr.OverallScore >= 0.85 {
		sb.WriteString("✅ **Good** - Translation quality is very good with minor improvements made.\n")
	} else if pr.OverallScore >= 0.75 {
		sb.WriteString("⚠️ **Acceptable** - Translation quality is acceptable with some improvements made.\n")
	} else {
		sb.WriteString("❌ **Needs Improvement** - Significant issues were found and addressed.\n")
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("A total of **%d changes** were made to improve translation quality.\n",
		pr.TotalChanges))

	return sb.String()
}

// GenerateJSONReport generates a JSON report (structure only, actual JSON marshaling done by caller)
func (pr *PolishingReport) GenerateJSONReport() map[string]interface{} {
	return map[string]interface{}{
		"timestamp": pr.EndTime.Format(time.RFC3339),
		"duration":  pr.Duration.String(),
		"config": map[string]interface{}{
			"providers":         pr.Config.Providers,
			"min_consensus":     pr.Config.MinConsensus,
			"verify_spirit":     pr.Config.VerifySpirit,
			"verify_language":   pr.Config.VerifyLanguage,
			"verify_context":    pr.Config.VerifyContext,
			"verify_vocabulary": pr.Config.VerifyVocabulary,
		},
		"summary": map[string]interface{}{
			"total_sections":     pr.TotalSections,
			"total_changes":      pr.TotalChanges,
			"total_issues":       pr.TotalIssues,
			"total_suggestions":  pr.TotalSuggestions,
			"consensus_rate":     pr.ConsensusRate,
			"average_confidence": pr.AverageConfidence,
		},
		"quality_scores": map[string]interface{}{
			"spirit":     pr.AverageSpiritScore,
			"language":   pr.AverageLanguageScore,
			"context":    pr.AverageContextScore,
			"vocabulary": pr.AverageVocabularyScore,
			"overall":    pr.OverallScore,
		},
		"issues": map[string]interface{}{
			"by_type":     pr.IssuesByType,
			"by_severity": pr.IssuesBySeverity,
			"top_issues":  pr.TopIssues,
		},
		"changes": map[string]interface{}{
			"significant_changes": pr.SignificantChanges,
		},
		"section_results": pr.SectionResults,
	}
}

// GenerateSummary generates a brief text summary
func (pr *PolishingReport) GenerateSummary() string {
	var sb strings.Builder

	sb.WriteString("=== POLISHING SUMMARY ===\n\n")
	sb.WriteString(fmt.Sprintf("Duration: %s\n", pr.Duration.Round(time.Second)))
	sb.WriteString(fmt.Sprintf("Sections Verified: %d\n", pr.TotalSections))
	sb.WriteString(fmt.Sprintf("Changes Made: %d\n", pr.TotalChanges))
	sb.WriteString(fmt.Sprintf("Consensus Rate: %.1f%%\n", pr.ConsensusRate))
	sb.WriteString(fmt.Sprintf("Overall Quality: %.1f%% (%s)\n\n",
		pr.OverallScore*100, getGrade(pr.OverallScore)))

	sb.WriteString("Quality Breakdown:\n")
	sb.WriteString(fmt.Sprintf("  Spirit:     %.1f%%\n", pr.AverageSpiritScore*100))
	sb.WriteString(fmt.Sprintf("  Language:   %.1f%%\n", pr.AverageLanguageScore*100))
	sb.WriteString(fmt.Sprintf("  Context:    %.1f%%\n", pr.AverageContextScore*100))
	sb.WriteString(fmt.Sprintf("  Vocabulary: %.1f%%\n\n", pr.AverageVocabularyScore*100))

	if pr.TotalIssues > 0 {
		sb.WriteString(fmt.Sprintf("Issues Found: %d\n", pr.TotalIssues))
		for severity, count := range pr.IssuesBySeverity {
			sb.WriteString(fmt.Sprintf("  %s: %d\n", cases.Title(language.English, cases.Compact).String(severity), count))
		}
	}

	return sb.String()
}

// Helper functions

func getGrade(score float64) string {
	if score >= 0.95 {
		return "A+"
	} else if score >= 0.90 {
		return "A"
	} else if score >= 0.85 {
		return "A-"
	} else if score >= 0.80 {
		return "B+"
	} else if score >= 0.75 {
		return "B"
	} else if score >= 0.70 {
		return "B-"
	} else if score >= 0.65 {
		return "C+"
	} else if score >= 0.60 {
		return "C"
	} else {
		return "D"
	}
}

func truncateForDisplay(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	// Truncate on a rune boundary, not a byte boundary, so that multi-byte
	// UTF-8 text (e.g. Cyrillic originals/translations) is never split
	// mid-rune into invalid UTF-8 in the generated report.
	cut := maxLen
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut] + "..."
}
