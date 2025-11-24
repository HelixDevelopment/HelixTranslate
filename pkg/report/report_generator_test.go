package report

import (
	"testing"
	"time"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock data structures for testing
type TranslationData struct {
	ID         string            `json:"id"`
	SourceLang string            `json:"source_lang"`
	TargetLang string            `json:"target_lang"`
	InputFile  string            `json:"input_file"`
	OutputFile string            `json:"output_file"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	WordCount  int               `json:"word_count"`
	Status     string            `json:"status"`
	Errors     []string          `json:"errors,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type PerformanceData struct {
	TotalTranslations int           `json:"total_translations"`
	SuccessRate       float64       `json:"success_rate"`
	AverageTime       time.Duration `json:"average_time"`
	MemoryUsage       int64         `json:"memory_usage"`
	CPUUsage          float64       `json:"cpu_usage"`
}

type SystemData struct {
	Version      string    `json:"version"`
	Uptime       time.Duration `json:"uptime"`
	ActiveUsers  int       `json:"active_users"`
	QueueSize    int       `json:"queue_size"`
	MemoryTotal  int64     `json:"memory_total"`
	MemoryUsed   int64     `json:"memory_used"`
	CPUTotal     int       `json:"cpu_total"`
	CPUUsed      int       `json:"cpu_used"`
}

type ReportConfig struct {
	IncludeMetrics bool   `json:"include_metrics"`
	IncludeErrors  bool   `json:"include_errors"`
	Format         string `json:"format"`
	Title          string `json:"title"`
}

func TestReportGenerator_GenerateTranslationReport(t *testing.T) {
	// Test 1: Basic report generation
	t.Run("BasicReportGeneration", func(t *testing.T) {
		generator := NewReportGenerator(ReportConfig{
			IncludeMetrics: true,
			IncludeErrors:  true,
			Format:         "json",
			Title:          "Translation Report",
		})
		
		data := &TranslationData{
			ID:         "test-123",
			SourceLang: "ru",
			TargetLang: "sr",
			InputFile:  "test.fb2",
			OutputFile: "test_sr.fb2",
			StartTime:  time.Now(),
			EndTime:    time.Now().Add(5 * time.Minute),
			WordCount:  1000,
			Status:     "completed",
			Metadata: map[string]interface{}{
				"provider": "openai",
				"model":    "gpt-4",
			},
		}
		
		report, err := generator.GenerateTranslationReport(data)
		require.NoError(t, err)
		assert.NotEmpty(t, report)
		
		// Verify report structure
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(report), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "test-123", parsed["id"])
		assert.Equal(t, "completed", parsed["status"])
		assert.Equal(t, "ru", parsed["source_lang"])
		assert.Equal(t, "sr", parsed["target_lang"])
	})
	
	// Test 2: Report with errors
	t.Run("ReportWithErrors", func(t *testing.T) {
		generator := NewReportGenerator(ReportConfig{
			IncludeMetrics: true,
			IncludeErrors:  true,
			Format:         "json",
		})
		
		data := &TranslationData{
			ID:         "error-test",
			Status:     "failed",
			Errors:     []string{"API timeout", "Invalid format"},
			StartTime:  time.Now(),
			EndTime:    time.Now().Add(1 * time.Minute),
			WordCount:  500,
		}
		
		report, err := generator.GenerateTranslationReport(data)
		require.NoError(t, err)
		
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(report), &parsed)
		require.NoError(t, err)
		
		errors, ok := parsed["errors"].([]interface{})
		require.True(t, ok)
		assert.Len(t, errors, 2)
		assert.Contains(t, errors, "API timeout")
		assert.Contains(t, errors, "Invalid format")
	})
	
	// Test 3: Different formats
	t.Run("DifferentFormats", func(t *testing.T) {
		testFormats := []string{"json", "xml", "csv", "html"}
		
		for _, format := range testFormats {
			t.Run(format+"Format", func(t *testing.T) {
				generator := NewReportGenerator(ReportConfig{
					Format: format,
				})
				
				data := &TranslationData{
					ID:        "format-test",
					Status:    "completed",
					StartTime: time.Now(),
					EndTime:   time.Now().Add(2 * time.Minute),
					WordCount: 750,
				}
				
				report, err := generator.GenerateTranslationReport(data)
				require.NoError(t, err)
				assert.NotEmpty(t, report)
				
				// Verify format-specific structure
				switch format {
				case "json":
					assert.True(t, json.Valid([]byte(report)))
				case "xml":
					assert.Contains(t, report, "<?xml")
					assert.Contains(t, report, "<report>")
				case "csv":
					assert.Contains(t, report, "id,status")
					assert.Contains(t, report, "format-test")
				case "html":
					assert.Contains(t, report, "<html>")
					assert.Contains(t, report, "<body>")
				}
			})
		}
	})
	
	// Test 4: Empty data
	t.Run("EmptyData", func(t *testing.T) {
		generator := NewReportGenerator(ReportConfig{
			Format: "json",
		})
		
		data := &TranslationData{}
		
		report, err := generator.GenerateTranslationReport(data)
		require.NoError(t, err)
		assert.NotEmpty(t, report)
		
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(report), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "", parsed["id"])
		assert.Equal(t, "", parsed["status"])
	})
}

func TestReportGenerator_GeneratePerformanceReport(t *testing.T) {
	generator := NewReportGenerator(ReportConfig{
		IncludeMetrics: true,
		Format:         "json",
	})
	
	performanceData := &PerformanceData{
		TotalTranslations: 100,
		SuccessRate:       0.95,
		AverageTime:       30 * time.Second,
		MemoryUsage:       512 * 1024 * 1024, // 512MB
		CPUUsage:          0.75,
	}
	
	report, err := generator.GeneratePerformanceReport(performanceData)
	require.NoError(t, err)
	
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(report), &parsed)
	require.NoError(t, err)
	assert.Equal(t, float64(100), parsed["total_translations"])
	assert.Equal(t, float64(0.95), parsed["success_rate"])
	assert.Equal(t, "30s", parsed["average_time"])
	assert.Equal(t, float64(536870912), parsed["memory_usage"])
	assert.Equal(t, float64(0.75), parsed["cpu_usage"])
}

func TestReportGenerator_GenerateSystemReport(t *testing.T) {
	generator := NewReportGenerator(ReportConfig{
		IncludeMetrics: true,
		IncludeErrors:  true,
		Format:         "json",
	})
	
	systemData := &SystemData{
		Version:     "2.3.0",
		Uptime:      24 * time.Hour,
		ActiveUsers: 50,
		QueueSize:   10,
		MemoryTotal: 8 * 1024 * 1024 * 1024, // 8GB
		MemoryUsed:  2 * 1024 * 1024 * 1024, // 2GB
		CPUTotal:    8,
		CPUUsed:     2,
	}
	
	report, err := generator.GenerateSystemReport(systemData)
	require.NoError(t, err)
	
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(report), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "2.3.0", parsed["version"])
	assert.Equal(t, float64(50), parsed["active_users"])
	assert.Equal(t, float64(10), parsed["queue_size"])
}

func TestReportGenerator_ErrorHandling(t *testing.T) {
	generator := NewReportGenerator(ReportConfig{
		Format: "json",
	})
	
	// Test with nil data
	t.Run("NilData", func(t *testing.T) {
		_, err := generator.GenerateTranslationReport(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data cannot be nil")
	})
	
	// Test with invalid format
	t.Run("InvalidFormat", func(t *testing.T) {
		generator := NewReportGenerator(ReportConfig{
			Format: "invalid",
		})
		
		data := &TranslationData{ID: "test"}
		_, err := generator.GenerateTranslationReport(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported format")
	})
}

// Mock ReportGenerator implementation for testing
type ReportGenerator struct {
	config ReportConfig
}

func NewReportGenerator(config ReportConfig) *ReportGenerator {
	return &ReportGenerator{
		config: config,
	}
}

func (rg *ReportGenerator) GenerateTranslationReport(data *TranslationData) (string, error) {
	if data == nil {
		return "", fmt.Errorf("data cannot be nil")
	}
	
	switch rg.config.Format {
	case "json":
		return json.MarshalIndent(data, "", "  ")
	case "xml":
		return rg.generateXMLReport(data), nil
	case "csv":
		return rg.generateCSVReport(data), nil
	case "html":
		return rg.generateHTMLReport(data), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", rg.config.Format)
	}
}

func (rg *ReportGenerator) GeneratePerformanceReport(data *PerformanceData) (string, error) {
	switch rg.config.Format {
	case "json":
		return json.MarshalIndent(data, "", "  ")
	default:
		return json.MarshalIndent(data, "", "  ")
	}
}

func (rg *ReportGenerator) GenerateSystemReport(data *SystemData) (string, error) {
	switch rg.config.Format {
	case "json":
		return json.MarshalIndent(data, "", "  ")
	default:
		return json.MarshalIndent(data, "", "  ")
	}
}

func (rg *ReportGenerator) generateXMLReport(data *TranslationData) string {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<report>
	<id>` + data.ID + `</id>
	<status>` + data.Status + `</status>
	<source_lang>` + data.SourceLang + `</source_lang>
	<target_lang>` + data.TargetLang + `</target_lang>
	<word_count>` + strconv.Itoa(data.WordCount) + `</word_count>
</report>`
	return xml
}

func (rg *ReportGenerator) generateCSVReport(data *TranslationData) string {
	return "id,status,source_lang,target_lang,word_count\n" +
		data.ID + "," + data.Status + "," + data.SourceLang + "," + data.TargetLang + "," + strconv.Itoa(data.WordCount)
}

func (rg *ReportGenerator) generateHTMLReport(data *TranslationData) string {
	return `<!DOCTYPE html>
<html>
<head><title>Translation Report</title></head>
<body>
	<h1>Translation Report</h1>
	<p>ID: ` + data.ID + `</p>
	<p>Status: ` + data.Status + `</p>
	<p>Source Language: ` + data.SourceLang + `</p>
	<p>Target Language: ` + data.TargetLang + `</p>
	<p>Word Count: ` + strconv.Itoa(data.WordCount) + `</p>
</body>
</html>`
}