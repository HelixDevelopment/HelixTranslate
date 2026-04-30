package llmverifier

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// GenerateMarkdownReport generates a human-readable markdown report
func (v *Verifier) GenerateMarkdownReport(results []VerificationResult, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	reportPath := filepath.Join(outputDir, "llm_verification_report.md")
	file, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	// Write report header
	fmt.Fprintf(file, "# LLM Verification Report\n\n")
	fmt.Fprintf(file, "Generated on: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// Generate summary
	summary := v.generateSummary(results)
	v.writeSummary(file, summary)

	// Write individual model reports
	for _, result := range results {
		if result.Error != "" {
			v.writeFailedModelReport(file, result)
		} else {
			v.writeModelReport(file, result)
		}
	}

	// Write category rankings
	v.writeCategoryRankings(file, results)

	return nil
}

// GenerateJSONReport generates a JSON report for programmatic use
func (v *Verifier) GenerateJSONReport(results []VerificationResult, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Add summary to results
	summary := v.generateSummary(results)
	jsonReport := map[string]interface{}{
		"summary": summary,
		"results": results,
		"metadata": map[string]interface{}{
			"generated_at": time.Now().Format(time.RFC3339),
			"total_models": len(results),
		},
	}

	jsonPath := filepath.Join(outputDir, "llm_verification_report.json")
	file, err := os.Create(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to create JSON report file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonReport)
}

// generateSummary creates a summary of the verification results
func (v *Verifier) generateSummary(results []VerificationResult) Summary {
	startTime := v.GetStartTime()
	endTime := v.GetEndTime()

	summary := Summary{
		TotalModels:     len(results),
		StartTime:       startTime,
		EndTime:         endTime,
		Duration:        endTime.Sub(startTime),
		AvailableModels: 0,
		FailedModels:    0,
	}

	var totalScore float64
	var allPerformers []TopPerformer
	var brotliSupported int

	for i, result := range results {
		if result.Error != "" {
			summary.FailedModels++
		} else {
			summary.AvailableModels++
			totalScore += result.PerformanceScores.OverallScore

			// Count Brotli support
			if result.FeatureDetection.SupportsBrotli {
				brotliSupported++
			}

			// Add to performers list for rankings
			allPerformers = append(allPerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.OverallScore,
				Rank:      i + 1, // Will be re-ranked later
			})
		}
	}

	if summary.AvailableModels > 0 {
		summary.AverageScore = totalScore / float64(summary.AvailableModels)
		summary.BrotliSupportRate = float64(brotliSupported) / float64(summary.AvailableModels) * 100
	} else {
		summary.AverageScore = 0
		summary.BrotliSupportRate = 0
	}

	// Generate category rankings
	summary.CategoryRankings = v.generateCategoryRankings(results)

	return summary
}

// generateCategoryRankings generates rankings by different categories
func (v *Verifier) generateCategoryRankings(results []VerificationResult) CategoryRankings {
	rankings := CategoryRankings{}

	// Overall score rankings
	overallPerformers := make([]TopPerformer, 0)
	for _, result := range results {
		if result.Error == "" {
			overallPerformers = append(overallPerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.OverallScore,
			})
		}
	}
	sort.Slice(overallPerformers, func(i, j int) bool {
		return overallPerformers[i].Score > overallPerformers[j].Score
	})
	// Assign ranks
	for i := range overallPerformers {
		overallPerformers[i].Rank = i + 1
	}
	rankings.ByCodeCapability = overallPerformers

	// Code capability rankings
	codePerformers := make([]TopPerformer, 0)
	for _, result := range results {
		if result.Error == "" {
			codePerformers = append(codePerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.CodeCapability,
			})
		}
	}
	sort.Slice(codePerformers, func(i, j int) bool {
		return codePerformers[i].Score > codePerformers[j].Score
	})
	for i := range codePerformers {
		codePerformers[i].Rank = i + 1
	}
	rankings.ByCodeCapability = codePerformers

	// Responsiveness rankings
	responsivenessPerformers := make([]TopPerformer, 0)
	for _, result := range results {
		if result.Error == "" {
			responsivenessPerformers = append(responsivenessPerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.Responsiveness,
			})
		}
	}
	sort.Slice(responsivenessPerformers, func(i, j int) bool {
		return responsivenessPerformers[i].Score > responsivenessPerformers[j].Score
	})
	for i := range responsivenessPerformers {
		responsivenessPerformers[i].Rank = i + 1
	}
	rankings.ByResponsiveness = responsivenessPerformers

	// Reliability rankings
	reliabilityPerformers := make([]TopPerformer, 0)
	for _, result := range results {
		if result.Error == "" {
			reliabilityPerformers = append(reliabilityPerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.Reliability,
			})
		}
	}
	sort.Slice(reliabilityPerformers, func(i, j int) bool {
		return reliabilityPerformers[i].Score > reliabilityPerformers[j].Score
	})
	for i := range reliabilityPerformers {
		reliabilityPerformers[i].Rank = i + 1
	}
	rankings.ByReliability = reliabilityPerformers

	// Feature richness rankings
	featurePerformers := make([]TopPerformer, 0)
	for _, result := range results {
		if result.Error == "" {
			featurePerformers = append(featurePerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.FeatureRichness,
			})
		}
	}
	sort.Slice(featurePerformers, func(i, j int) bool {
		return featurePerformers[i].Score > featurePerformers[j].Score
	})
	for i := range featurePerformers {
		featurePerformers[i].Rank = i + 1
	}
	rankings.ByFeatureRichness = featurePerformers

	// Value rankings
	valuePerformers := make([]TopPerformer, 0)
	for _, result := range results {
		if result.Error == "" {
			valuePerformers = append(valuePerformers, TopPerformer{
				ModelName: result.ModelInfo.ID,
				Score:     result.PerformanceScores.ValueProposition,
			})
		}
	}
	sort.Slice(valuePerformers, func(i, j int) bool {
		return valuePerformers[i].Score > valuePerformers[j].Score
	})
	for i := range valuePerformers {
		valuePerformers[i].Rank = i + 1
	}
	rankings.ByValue = valuePerformers

	return rankings
}

// writeSummary writes the summary section of the report
func (v *Verifier) writeSummary(file *os.File, summary Summary) {
	fmt.Fprintf(file, "## Summary\n\n")
	fmt.Fprintf(file, "- Total Models: %d\n", summary.TotalModels)
	fmt.Fprintf(file, "- Available Models: %d\n", summary.AvailableModels)
	fmt.Fprintf(file, "- Failed Models: %d\n", summary.FailedModels)
	fmt.Fprintf(file, "- Average Overall Score: %.2f\n", summary.AverageScore)
	fmt.Fprintf(file, "- Brotli Support Rate: %.2f%%\n", summary.BrotliSupportRate)
	fmt.Fprintf(file, "\n")

	// Show top performers by overall score
	fmt.Fprintf(file, "### Top Performers by Overall Score\n\n")
	for i, performer := range summary.CategoryRankings.ByCodeCapability {
		if i >= 5 { // Show top 5
			break
		}
		fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelName, performer.Score)
	}
	fmt.Fprintf(file, "\n")
}

// writeModelReport writes the report for a single successfully verified model
func (v *Verifier) writeModelReport(file *os.File, result VerificationResult) {
	fmt.Fprintf(file, "## Model: %s\n\n", result.ModelInfo.ID)

	// Basic information
	fmt.Fprintf(file, "### Basic Information\n")
	fmt.Fprintf(file, "- **Endpoint**: %s\n", result.ModelInfo.Endpoint)
	fmt.Fprintf(file, "- **Verified at**: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "\n")

	// Scores
	fmt.Fprintf(file, "### Performance Scores\n")
	fmt.Fprintf(file, "- **Overall Score**: %.2f\n", result.PerformanceScores.OverallScore)
	fmt.Fprintf(file, "- **Code Capability**: %.2f\n", result.PerformanceScores.CodeCapability)
	fmt.Fprintf(file, "- **Responsiveness**: %.2f\n", result.PerformanceScores.Responsiveness)
	fmt.Fprintf(file, "- **Reliability**: %.2f\n", result.PerformanceScores.Reliability)
	fmt.Fprintf(file, "- **Feature Richness**: %.2f\n", result.PerformanceScores.FeatureRichness)
	fmt.Fprintf(file, "- **Value Proposition**: %.2f\n", result.PerformanceScores.ValueProposition)
	fmt.Fprintf(file, "\n")

	// Availability
	fmt.Fprintf(file, "### Availability\n")
	fmt.Fprintf(file, "- **Exists**: %t\n", result.Availability.Exists)
	fmt.Fprintf(file, "- **Responsive**: %t\n", result.Availability.Responsive)
	fmt.Fprintf(file, "- **Overloaded**: %t\n", result.Availability.Overloaded)
	fmt.Fprintf(file, "- **Response Time**: %s\n", result.Availability.Latency.String())
	fmt.Fprintf(file, "\n")

	// Response time metrics
	if result.ResponseTime.AverageLatency > 0 {
		fmt.Fprintf(file, "### Response Time Metrics\n")
		fmt.Fprintf(file, "- **Average Latency**: %s\n", result.ResponseTime.AverageLatency.String())
		// fmt.Fprintf(file, "- **P95 Latency**: %s\n", result.ResponseTime.P95Latency.String())  // We're not calculating P95 currently
		fmt.Fprintf(file, "- **Throughput**: %.2f requests/sec\n", result.ResponseTime.Throughput)
		fmt.Fprintf(file, "\n")
	}

	// Features
	fmt.Fprintf(file, "### Supported Features\n")
	fmt.Fprintf(file, "- **Tool Use**: %t\n", result.FeatureDetection.ToolUse)
	fmt.Fprintf(file, "- **Function Calling**: %t\n", result.FeatureDetection.FunctionCalling)
	fmt.Fprintf(file, "- **Code Generation**: %t\n", result.FeatureDetection.CodeGeneration)
	fmt.Fprintf(file, "- **Code Completion**: %t\n", result.FeatureDetection.CodeCompletion)
	fmt.Fprintf(file, "- **Code Explanation**: %t\n", result.FeatureDetection.CodeExplanation)
	fmt.Fprintf(file, "- **Code Review**: %t\n", result.FeatureDetection.CodeReview)
	fmt.Fprintf(file, "- **Embeddings**: %t\n", result.FeatureDetection.Embeddings)
	fmt.Fprintf(file, "- **Reranking**: %t\n", result.FeatureDetection.Reranking)
	fmt.Fprintf(file, "- **Image Generation**: %t\n", result.FeatureDetection.ImageGeneration)
	fmt.Fprintf(file, "- **Audio Generation**: %t\n", result.FeatureDetection.AudioGeneration)
	fmt.Fprintf(file, "- **Video Generation**: %t\n", result.FeatureDetection.VideoGeneration)
	fmt.Fprintf(file, "- **MCPs**: %t\n", result.FeatureDetection.MCPs)
	fmt.Fprintf(file, "- **LSPs**: %t\n", result.FeatureDetection.LSPs)
	fmt.Fprintf(file, "- **ACPs**: %t\n", result.FeatureDetection.ACPs)
	fmt.Fprintf(file, "- **Multimodal**: %t\n", result.FeatureDetection.Multimodal)
	fmt.Fprintf(file, "- **Streaming**: %t\n", result.FeatureDetection.Streaming)
	fmt.Fprintf(file, "- **JSON Mode**: %t\n", result.FeatureDetection.JSONMode)
	fmt.Fprintf(file, "- **Structured Output**: %t\n", result.FeatureDetection.StructuredOutput)
	fmt.Fprintf(file, "- **Reasoning**: %t\n", result.FeatureDetection.Reasoning)
	fmt.Fprintf(file, "- **Parallel Tool Use**: %t (Max %d calls)\n", result.FeatureDetection.ParallelToolUse, result.FeatureDetection.MaxParallelCalls)
	fmt.Fprintf(file, "- **Brotli Compression**: %t\n", result.FeatureDetection.SupportsBrotli)
	fmt.Fprintf(file, "\n")

	// Code capabilities
	fmt.Fprintf(file, "### Code Capabilities\n")
	fmt.Fprintf(file, "- **Language Support**: %s\n", strings.Join(result.CodeCapabilities.LanguageSupport, ", "))
	fmt.Fprintf(file, "- **Code Generation**: %t\n", result.CodeCapabilities.CodeGeneration)
	fmt.Fprintf(file, "- **Code Completion**: %t\n", result.CodeCapabilities.CodeCompletion)
	fmt.Fprintf(file, "- **Code Debugging**: %t\n", result.CodeCapabilities.CodeDebugging)
	fmt.Fprintf(file, "- **Code Optimization**: %t\n", result.CodeCapabilities.CodeOptimization)
	fmt.Fprintf(file, "- **Code Review**: %t\n", result.CodeCapabilities.CodeReview)
	fmt.Fprintf(file, "- **Test Generation**: %t\n", result.CodeCapabilities.TestGeneration)
	fmt.Fprintf(file, "- **Documentation**: %t\n", result.CodeCapabilities.Documentation)
	fmt.Fprintf(file, "- **Refactoring**: %t\n", result.CodeCapabilities.Refactoring)
	fmt.Fprintf(file, "- **Error Resolution**: %t\n", result.CodeCapabilities.ErrorResolution)
	fmt.Fprintf(file, "- **Architecture Understanding**: %t\n", result.CodeCapabilities.Architecture)
	fmt.Fprintf(file, "- **Security Assessment**: %t\n", result.CodeCapabilities.SecurityAssessment)
	fmt.Fprintf(file, "- **Pattern Recognition**: %t\n", result.CodeCapabilities.PatternRecognition)

	// Complexity handling
	fmt.Fprintf(file, "- **Complexity Level**: %d/5\n", result.CodeCapabilities.ComplexityHandling.MaxHandledDepth)
	fmt.Fprintf(file, "- **Code Quality Score**: %.2f\n", result.CodeCapabilities.ComplexityHandling.CodeQuality)
	fmt.Fprintf(file, "- **Logic Correctness Score**: %.2f\n", result.CodeCapabilities.ComplexityHandling.LogicCorrectness)
	fmt.Fprintf(file, "- **Runtime Efficiency Score**: %.2f\n", result.CodeCapabilities.ComplexityHandling.RuntimeEfficiency)
	fmt.Fprintf(file, "\n")

	// Language-specific scores
	fmt.Fprintf(file, "### Language-Specific Performance\n")
	fmt.Fprintf(file, "- **Python Success Rate**: %.2f%%\n", result.CodeCapabilities.PromptResponse.PythonSuccessRate)
	fmt.Fprintf(file, "- **JavaScript Success Rate**: %.2f%%\n", result.CodeCapabilities.PromptResponse.JavascriptSuccessRate)
	fmt.Fprintf(file, "- **Go Success Rate**: %.2f%%\n", result.CodeCapabilities.PromptResponse.GoSuccessRate)
	fmt.Fprintf(file, "- **Java Success Rate**: %.2f%%\n", result.CodeCapabilities.PromptResponse.JavaSuccessRate)
	fmt.Fprintf(file, "- **C++ Success Rate**: %.2f%%\n", result.CodeCapabilities.PromptResponse.CppSuccessRate)
	fmt.Fprintf(file, "- **TypeScript Success Rate**: %.2f%%\n", result.CodeCapabilities.PromptResponse.TypescriptSuccessRate)
	fmt.Fprintf(file, "- **Overall Success Rate**: %.2f%%\n", result.CodeCapabilities.PromptResponse.OverallSuccessRate)
	fmt.Fprintf(file, "\n")
}

// writeFailedModelReport writes the report for a model that failed verification
func (v *Verifier) writeFailedModelReport(file *os.File, result VerificationResult) {
	fmt.Fprintf(file, "## Model: %s (FAILED)\n\n", result.ModelInfo.ID)
	fmt.Fprintf(file, "**Error**: %s\n\n", result.Error)
	fmt.Fprintf(file, "- **Endpoint**: %s\n", result.ModelInfo.Endpoint)
	fmt.Fprintf(file, "- **Attempted at**: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "\n")
}

// writeCategoryRankings writes the category-wise rankings
func (v *Verifier) writeCategoryRankings(file *os.File, results []VerificationResult) {
	fmt.Fprintf(file, "## Category Rankings\n\n")

	// Create sorted lists for each category
	overallSorted := v.SortResultsByScore(results, func(r VerificationResult) float64 { return r.PerformanceScores.OverallScore })
	codeSorted := v.SortResultsByScore(results, func(r VerificationResult) float64 { return r.PerformanceScores.CodeCapability })
	responsivenessSorted := v.SortResultsByScore(results, func(r VerificationResult) float64 { return r.PerformanceScores.Responsiveness })
	reliabilitySorted := v.SortResultsByScore(results, func(r VerificationResult) float64 { return r.PerformanceScores.Reliability })
	featureSorted := v.SortResultsByScore(results, func(r VerificationResult) float64 { return r.PerformanceScores.FeatureRichness })
	valueSorted := v.SortResultsByScore(results, func(r VerificationResult) float64 { return r.PerformanceScores.ValueProposition })

	// Overall rankings
	fmt.Fprintf(file, "### Overall Performance\n")
	for i, performer := range overallSorted {
		if i >= 10 { // Show top 10
			break
		}
		if performer.Error == "" {
			fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelInfo.ID, performer.PerformanceScores.OverallScore)
		}
	}
	fmt.Fprintf(file, "\n")

	// Code Capability Rankings
	fmt.Fprintf(file, "### By Code Capability\n")
	for i, performer := range codeSorted {
		if i >= 10 { // Show top 10
			break
		}
		if performer.Error == "" {
			fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelInfo.ID, performer.PerformanceScores.CodeCapability)
		}
	}
	fmt.Fprintf(file, "\n")

	// Responsiveness Rankings
	fmt.Fprintf(file, "### By Responsiveness\n")
	for i, performer := range responsivenessSorted {
		if i >= 10 { // Show top 10
			break
		}
		if performer.Error == "" {
			fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelInfo.ID, performer.PerformanceScores.Responsiveness)
		}
	}
	fmt.Fprintf(file, "\n")

	// Reliability Rankings
	fmt.Fprintf(file, "### By Reliability\n")
	for i, performer := range reliabilitySorted {
		if i >= 10 { // Show top 10
			break
		}
		if performer.Error == "" {
			fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelInfo.ID, performer.PerformanceScores.Reliability)
		}
	}
	fmt.Fprintf(file, "\n")

	// Feature Richness Rankings
	fmt.Fprintf(file, "### By Feature Richness\n")
	for i, performer := range featureSorted {
		if i >= 10 { // Show top 10
			break
		}
		if performer.Error == "" {
			fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelInfo.ID, performer.PerformanceScores.FeatureRichness)
		}
	}
	fmt.Fprintf(file, "\n")

	// Value Proposition Rankings
	fmt.Fprintf(file, "### By Value Proposition\n")
	for i, performer := range valueSorted {
		if i >= 10 { // Show top 10
			break
		}
		if performer.Error == "" {
			fmt.Fprintf(file, "%d. **%s**: %.2f\n", i+1, performer.ModelInfo.ID, performer.PerformanceScores.ValueProposition)
		}
	}
	fmt.Fprintf(file, "\n")
}

// SortResultsByScore sorts verification results by a given score function
func (v *Verifier) SortResultsByScore(results []VerificationResult, scoreFunc func(VerificationResult) float64) []VerificationResult {
	// Create a copy to sort
	sorted := make([]VerificationResult, len(results))
	copy(sorted, results)

	// Sort by the provided score function in descending order
	sort.Slice(sorted, func(i, j int) bool {
		return scoreFunc(sorted[i]) > scoreFunc(sorted[j])
	})

	return sorted
}
