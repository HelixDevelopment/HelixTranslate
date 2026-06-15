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
		fmt.Println(translated) //nolint:forbidigo
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
		fmt.Println(translated) //nolint:forbidigo
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
		// Honor context cancellation: stop processing further files once the
		// caller has cancelled (or the deadline has elapsed).
		if err := ctx.Err(); err != nil {
			return results, err
		}

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

			// Honor context cancellation: a worker that starts after the caller
			// cancelled must not parse/write its file. Record the cancellation in
			// its result slot (keeping every slot populated) instead of processing.
			if cerr := ctx.Err(); cerr != nil {
				results[idx] = ProcessingResult{
					InputPath:  filePath,
					OutputPath: "",
					Success:    false,
					Error:      cerr,
				}
				return
			}

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

	// Surface cancellation to the caller after all workers have drained.
	if err := ctx.Err(); err != nil {
		return results, err
	}

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

	// Translate the book using the provided translator. Previously this method
	// parsed the input and wrote it straight to the output WITHOUT translating —
	// so a directory/file batch (incl. the /translate/directory API endpoint,
	// which supplies a real translator) reported Success:true while shipping an
	// untranslated copy of the input (a §11.4 / CONST-035 PASS-bluff). Translate
	// every chapter title + every section (recursively) so the output genuinely
	// carries the target-language text the caller asked for.
	if bp.options.Translator != nil {
		if err := bp.translateBook(ctx, book); err != nil {
			return nil, fmt.Errorf("failed to translate file: %w", err)
		}
	}

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

// translateBook translates every chapter title and every section (recursively,
// including subsections) of the parsed book in place using the configured
// translator. Context cancellation is honored between units so a cancelled batch
// stops promptly instead of completing a long translation.
func (bp *BatchProcessor) translateBook(ctx context.Context, book *ebook.Book) error {
	for i := range book.Chapters {
		if err := ctx.Err(); err != nil {
			return err
		}

		if translated, err := bp.translateText(ctx, book.Chapters[i].Title); err != nil {
			return err
		} else {
			book.Chapters[i].Title = translated
		}

		for j := range book.Chapters[i].Sections {
			if err := bp.translateSection(ctx, &book.Chapters[i].Sections[j]); err != nil {
				return err
			}
		}
	}
	return nil
}

// translateSection translates a section's title and content, then recurses into
// its subsections, mutating the section in place.
func (bp *BatchProcessor) translateSection(ctx context.Context, section *ebook.Section) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if translated, err := bp.translateText(ctx, section.Title); err != nil {
		return err
	} else {
		section.Title = translated
	}

	if translated, err := bp.translateText(ctx, section.Content); err != nil {
		return err
	} else {
		section.Content = translated
	}

	for k := range section.Subsections {
		if err := bp.translateSection(ctx, &section.Subsections[k]); err != nil {
			return err
		}
	}
	return nil
}

// translateText translates a single string, skipping empty/whitespace-only input
// (no point spending an LLM call on it) and returning the original on a
// whitespace-only string.
func (bp *BatchProcessor) translateText(ctx context.Context, text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return text, nil
	}
	return bp.options.Translator.Translate(ctx, text, "")
}

// autoOutputName builds a deterministic, collision-free output file name for the
// auto-naming (directory / write-next-to-input) branches.
//
// The source extension is embedded into the name so two distinct inputs that
// share a stem but differ in extension (e.g. "book.fb2" and "book.epub", both
// supported formats discovered by findSupportedFiles) never map onto the same
// output file. Stripping the source extension — as the previous "%s_%s.%s" with
// base=TrimSuffix(path,ext) did — silently collapsed such inputs onto one path,
// so the second translation overwrote the first (sequential) or both goroutines
// wrote the same file (parallel): in both cases translated output was lost.
//
// stem is the input path/relpath with its extension already removed; srcExt is
// the input's extension WITH the leading dot ("" for an extensionless input).
func autoOutputName(stem, srcExt, lang, outputFormat string) string {
	if outputFormat == "" {
		outputFormat = "epub"
	}
	// Use the dotless source extension as a discriminator. When the source has
	// no extension there is nothing to disambiguate, so fall back to the
	// stem-only form.
	if srcExt == "" {
		return fmt.Sprintf("%s_%s.%s", stem, lang, outputFormat)
	}
	return fmt.Sprintf("%s_%s_%s.%s", stem, strings.TrimPrefix(srcExt, "."), lang, outputFormat)
}

// computeOutputPath computes the output path preserving directory structure
func (bp *BatchProcessor) computeOutputPath(inputPath string) (string, error) {
	if bp.options.OutputPath == "" {
		// Generate output path in same directory
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		lang := bp.options.TargetLanguage.Code
		return autoOutputName(base, ext, lang, bp.options.OutputFormat), nil
	}

	// Check if output is a directory
	outputInfo, err := os.Stat(bp.options.OutputPath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	isOutputDir := err == nil && outputInfo.IsDir()

	// A non-existent OutputPath with no file extension is an as-yet-uncreated
	// destination DIRECTORY (e.g. "-output /some/new/dir"), not a single output
	// file. Treating it as a file path would map every input in a directory batch
	// onto the same output, silently overwriting all but the last (data loss).
	if !isOutputDir && os.IsNotExist(err) && filepath.Ext(bp.options.OutputPath) == "" {
		isOutputDir = true
	}

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

	// Change extension and add language suffix. Embed the source extension so
	// same-stem/different-extension inputs do not collide onto one output file.
	ext := filepath.Ext(relPath)
	base := strings.TrimSuffix(relPath, ext)
	lang := bp.options.TargetLanguage.Code

	outputFile := autoOutputName(base, ext, lang, bp.options.OutputFormat)
	outputPath := filepath.Join(bp.options.OutputPath, outputFile)

	// Create output directory if needed
	outputDir := filepath.Dir(outputPath)
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	return outputPath, nil
}
