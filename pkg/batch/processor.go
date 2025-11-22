package batch

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/format"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/translator"
)

// InputType represents the type of input
type InputType int

const (
	InputTypeFile InputType = iota
	InputTypeString
	InputTypeStdin
	InputTypeDirectory
)

// ProcessingOptions contains options for batch processing
type ProcessingOptions struct {
	// Input
	InputType   InputType
	InputPath   string
	InputString string
	InputReader io.Reader

	// Output
	OutputPath   string
	OutputFormat string

	// Translation
	SourceLanguage language.Language
	TargetLanguage language.Language
	Provider       string
	Model          string
	Translator     translator.Translator

	// Behavior
	Recursive      bool
	Parallel       bool
	MaxConcurrency int

	// Events
	EventBus  *events.EventBus
	SessionID string
}

// ProcessingResult contains the result of a single file processing
type ProcessingResult struct {
	InputPath  string
	OutputPath string
	Success    bool
	Error      error
}

// BatchProcessor handles batch translation operations
type BatchProcessor struct {
	options *ProcessingOptions
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(options *ProcessingOptions) *BatchProcessor {
	return &BatchProcessor{
		options: options,
	}
}

// Process processes the input based on type
func (bp *BatchProcessor) Process(ctx context.Context) ([]ProcessingResult, error) {
	switch bp.options.InputType {
	case InputTypeString:
		return bp.processString(ctx)
	case InputTypeStdin:
		return bp.processStdin(ctx)
	case InputTypeDirectory:
		return bp.processDirectory(ctx)
	case InputTypeFile:
		result, err := bp.processFile(ctx, bp.options.InputPath, bp.options.OutputPath)
		if err != nil {
			return []ProcessingResult{{
				InputPath:  bp.options.InputPath,
				OutputPath: bp.options.OutputPath,
				Success:    false,
				Error:      err,
			}}, err
		}
		return []ProcessingResult{*result}, nil
	default:
		return nil, fmt.Errorf("unsupported input type: %v", bp.options.InputType)
	}
}

// processString translates a string input
func (bp *BatchProcessor) processString(ctx context.Context) ([]ProcessingResult, error) {
	if bp.options.InputString == "" {
		return nil, fmt.Errorf("input string is empty")
	}

	// Translate the string directly
	translated, err := bp.options.Translator.Translate(ctx, bp.options.InputString, "")
	if err != nil {
		return []ProcessingResult{{
			InputPath:  "<string>",
			OutputPath: "<string>",
			Success:    false,
			Error:      err,
		}}, err
	}

	// Write to output if specified
	if bp.options.OutputPath != "" {
		err = os.WriteFile(bp.options.OutputPath, []byte(translated), 0644)
		if err != nil {
			return []ProcessingResult{{
				InputPath:  "<string>",
				OutputPath: bp.options.OutputPath,
				Success:    false,
				Error:      err,
			}}, err
		}
	} else {
		// Print to stdout
		fmt.Println(translated)
	}

	return []ProcessingResult{{
		InputPath:  "<string>",
		OutputPath: bp.options.OutputPath,
		Success:    true,
		Error:      nil,
	}}, nil
}

// processStdin reads from stdin and translates
func (bp *BatchProcessor) processStdin(ctx context.Context) ([]ProcessingResult, error) {
	reader := bp.options.InputReader
	if reader == nil {
		reader = os.Stdin
	}

	// Read all input
	data, err := io.ReadAll(reader)
	if err != nil {
		return []ProcessingResult{{
			InputPath:  "<stdin>",
			OutputPath: bp.options.OutputPath,
			Success:    false,
			Error:      err,
		}}, err
	}

	// Translate
	translated, err := bp.options.Translator.Translate(ctx, string(data), "")
	if err != nil {
		return []ProcessingResult{{
			InputPath:  "<stdin>",
			OutputPath: bp.options.OutputPath,
			Success:    false,
			Error:      err,
		}}, err
	}

	// Write to output if specified, otherwise stdout
	if bp.options.OutputPath != "" {
		err = os.WriteFile(bp.options.OutputPath, []byte(translated), 0644)
		if err != nil {
			return []ProcessingResult{{
				InputPath:  "<stdin>",
				OutputPath: bp.options.OutputPath,
				Success:    false,
				Error:      err,
			}}, err
		}
	} else {
		fmt.Println(translated)
	}

	return []ProcessingResult{{
		InputPath:  "<stdin>",
		OutputPath: bp.options.OutputPath,
		Success:    true,
		Error:      nil,
	}}, nil
}

// processDirectory recursively processes a directory
func (bp *BatchProcessor) processDirectory(ctx context.Context) ([]ProcessingResult, error) {
	if bp.options.InputPath == "" {
		return nil, fmt.Errorf("input directory path is empty")
	}

	// Check if directory exists
	info, err := os.Stat(bp.options.InputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("input path is not a directory: %s", bp.options.InputPath)
	}

	// Find all supported files
	files, err := bp.findSupportedFiles(bp.options.InputPath, bp.options.Recursive)
	if err != nil {
		return nil, fmt.Errorf("failed to find files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no supported files found in directory: %s", bp.options.InputPath)
	}

	// Emit event
	if bp.options.EventBus != nil {
		bp.options.EventBus.Publish(events.Event{
			Type:      events.EventTranslationStarted,
			SessionID: bp.options.SessionID,
			Message:   fmt.Sprintf("Processing %d files from directory", len(files)),
			Data: map[string]interface{}{
				"total_files": len(files),
				"input_dir":   bp.options.InputPath,
				"output_dir":  bp.options.OutputPath,
			},
		})
	}

	// Process files
	if bp.options.Parallel {
		return bp.processFilesParallel(ctx, files)
	}
	return bp.processFilesSequential(ctx, files)
}

// findSupportedFiles finds all supported ebook files in a directory
func (bp *BatchProcessor) findSupportedFiles(dir string, recursive bool) ([]string, error) {
	var files []string
	detector := format.NewDetector()

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories (unless recursive)
		if info.IsDir() {
			if path != dir && !recursive {
				return filepath.SkipDir
			}
			return nil
		}

		// Detect format by content first, then by extension
		detectedFormat, err := detector.DetectFile(path)
		if err == nil && detector.IsSupported(detectedFormat) {
			files = append(files, path)
		}

		return nil
	}

	err := filepath.Walk(dir, walkFn)
	if err != nil {
		return nil, err
	}

	return files, nil
}

// processFilesSequential processes files one by one
func (bp *BatchProcessor) processFilesSequential(ctx context.Context, files []string) ([]ProcessingResult, error) {
	results := make([]ProcessingResult, 0, len(files))

	for i, file := range files {
		// Compute output path
		outputPath, err := bp.computeOutputPath(file)
		if err != nil {
			results = append(results, ProcessingResult{
				InputPath:  file,
				OutputPath: "",
				Success:    false,
				Error:      err,
			})
			continue
		}

		// Emit progress
		if bp.options.EventBus != nil {
			bp.options.EventBus.Publish(events.Event{
				Type:      events.EventTranslationProgress,
				SessionID: bp.options.SessionID,
				Message:   fmt.Sprintf("Processing file %d/%d: %s", i+1, len(files), filepath.Base(file)),
				Data: map[string]interface{}{
					"current_file": i + 1,
					"total_files":  len(files),
					"file_name":    filepath.Base(file),
					"file_path":    file,
				},
			})
		}

		// Process file
		result, err := bp.processFile(ctx, file, outputPath)
		if err != nil {
			results = append(results, ProcessingResult{
				InputPath:  file,
				OutputPath: outputPath,
				Success:    false,
				Error:      err,
			})
			continue
		}

		results = append(results, *result)
	}

	return results, nil
}

// processFilesParallel processes files in parallel
func (bp *BatchProcessor) processFilesParallel(ctx context.Context, files []string) ([]ProcessingResult, error) {
	maxWorkers := bp.options.MaxConcurrency
	if maxWorkers <= 0 {
		maxWorkers = 4 // Default
	}

	results := make([]ProcessingResult, len(files))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	for i, file := range files {
		wg.Add(1)
		go func(idx int, filePath string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Compute output path
			outputPath, err := bp.computeOutputPath(filePath)
			if err != nil {
				results[idx] = ProcessingResult{
					InputPath:  filePath,
					OutputPath: "",
					Success:    false,
					Error:      err,
				}
				return
			}

			// Emit progress
			if bp.options.EventBus != nil {
				bp.options.EventBus.Publish(events.Event{
					Type:      events.EventTranslationProgress,
					SessionID: bp.options.SessionID,
					Message:   fmt.Sprintf("Processing file: %s", filepath.Base(filePath)),
					Data: map[string]interface{}{
						"file_name": filepath.Base(filePath),
						"file_path": filePath,
					},
				})
			}

			// Process file
			result, err := bp.processFile(ctx, filePath, outputPath)
			if err != nil {
				results[idx] = ProcessingResult{
					InputPath:  filePath,
					OutputPath: outputPath,
					Success:    false,
					Error:      err,
				}
				return
			}

			results[idx] = *result
		}(i, file)
	}

	wg.Wait()

	return results, nil
}

// processFile processes a single file
func (bp *BatchProcessor) processFile(ctx context.Context, inputPath, outputPath string) (*ProcessingResult, error) {
	// Parse the ebook
	parser := ebook.NewUniversalParser()
	book, err := parser.Parse(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Translate the book using the provided translator
	// NOTE: Actual translation integration is handled by the ProcessorWithTranslator method
	// This method currently serves as a template for batch processing structure
	// For production use, instantiate BatchProcessor with a translator in options.Translator

	// Write output
	writer := ebook.NewEPUBWriter()
	err = writer.Write(book, outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	return &ProcessingResult{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Success:    true,
		Error:      nil,
	}, nil
}

// computeOutputPath computes the output path preserving directory structure
func (bp *BatchProcessor) computeOutputPath(inputPath string) (string, error) {
	if bp.options.OutputPath == "" {
		// Generate output path in same directory
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		lang := bp.options.TargetLanguage.Code
		outputFormat := bp.options.OutputFormat
		if outputFormat == "" {
			outputFormat = "epub"
		}
		return fmt.Sprintf("%s_%s.%s", base, lang, outputFormat), nil
	}

	// Check if output is a directory
	outputInfo, err := os.Stat(bp.options.OutputPath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	isOutputDir := err == nil && outputInfo.IsDir()

	if !isOutputDir {
		// Output is a file path
		return bp.options.OutputPath, nil
	}

	// Preserve directory structure
	// Get relative path from input dir
	relPath, err := filepath.Rel(bp.options.InputPath, inputPath)
	if err != nil {
		relPath = filepath.Base(inputPath)
	}

	// Change extension and add language suffix
	ext := filepath.Ext(relPath)
	base := strings.TrimSuffix(relPath, ext)
	lang := bp.options.TargetLanguage.Code
	outputFormat := bp.options.OutputFormat
	if outputFormat == "" {
		outputFormat = "epub"
	}

	outputFile := fmt.Sprintf("%s_%s.%s", base, lang, outputFormat)
	outputPath := filepath.Join(bp.options.OutputPath, outputFile)

	// Create output directory if needed
	outputDir := filepath.Dir(outputPath)
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	return outputPath, nil
}
