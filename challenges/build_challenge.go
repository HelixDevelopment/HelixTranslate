package challenges

import (
	"context"
	"os/exec"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// NewBuildChallenge verifies the project compiles successfully.
func NewBuildChallenge() challenge.Challenge {
	return challenge.NewShellChallenge(
		"build-verification",
		"Build Verification",
		"Verifies that go build ./... completes without errors",
		"build",
		nil,
		"scripts/challenges/build_challenge.sh",
		nil,
		".",
	)
}

// NewFormatDetectionChallenge verifies ebook format detection.
type FormatDetectionChallenge struct {
	challenge.BaseChallenge
}

// NewFormatDetectionChallenge creates the format detection challenge.
func NewFormatDetectionChallenge() *FormatDetectionChallenge {
	bc := challenge.NewBaseChallenge(
		"format-detection",
		"Format Detection",
		"Verifies that FB2, EPUB, TXT, HTML, PDF, and DOCX formats are detected correctly",
		"unit",
		nil,
	)
	return &FormatDetectionChallenge{BaseChallenge: bc}
}

// Execute runs the format detection challenge.
func (c *FormatDetectionChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestDetect", "./pkg/format/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "format_detection",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewFB2ParsingChallenge verifies FB2 parsing functionality.
type FB2ParsingChallenge struct {
	challenge.BaseChallenge
}

// NewFB2ParsingChallenge creates the FB2 parsing challenge.
func NewFB2ParsingChallenge() *FB2ParsingChallenge {
	bc := challenge.NewBaseChallenge(
		"fb2-parsing",
		"FB2 Parsing",
		"Verifies FB2 XML parsing, namespace handling, and text extraction",
		"unit",
		nil,
	)
	return &FB2ParsingChallenge{BaseChallenge: bc}
}

// Execute runs the FB2 parsing challenge.
func (c *FB2ParsingChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestFB2", "./pkg/fb2/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "fb2_parsing",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewEPUBParsingChallenge verifies EPUB parsing functionality.
type EPUBParsingChallenge struct {
	challenge.BaseChallenge
}

// NewEPUBParsingChallenge creates the EPUB parsing challenge.
func NewEPUBParsingChallenge() *EPUBParsingChallenge {
	bc := challenge.NewBaseChallenge(
		"epub-parsing",
		"EPUB Parsing",
		"Verifies EPUB container parsing, OPF reading, and content extraction",
		"unit",
		nil,
	)
	return &EPUBParsingChallenge{BaseChallenge: bc}
}

// Execute runs the EPUB parsing challenge.
func (c *EPUBParsingChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestEPUB", "./pkg/ebook/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "epub_parsing",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewMarkdownConversionChallenge verifies markdown conversion workflows.
type MarkdownConversionChallenge struct {
	challenge.BaseChallenge
}

// NewMarkdownConversionChallenge creates the markdown conversion challenge.
func NewMarkdownConversionChallenge() *MarkdownConversionChallenge {
	bc := challenge.NewBaseChallenge(
		"markdown-conversion",
		"Markdown Conversion",
		"Verifies EPUB to Markdown and Markdown to EPUB conversions",
		"integration",
		nil,
	)
	return &MarkdownConversionChallenge{BaseChallenge: bc}
}

// Execute runs the markdown conversion challenge.
func (c *MarkdownConversionChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestMarkdown", "./pkg/markdown/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "markdown_conversion",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewTranslationPipelineChallenge verifies the end-to-end translation pipeline.
type TranslationPipelineChallenge struct {
	challenge.BaseChallenge
}

// NewTranslationPipelineChallenge creates the translation pipeline challenge.
func NewTranslationPipelineChallenge() *TranslationPipelineChallenge {
	bc := challenge.NewBaseChallenge(
		"translation-pipeline",
		"Translation Pipeline",
		"Verifies the complete translation pipeline from parsing through output generation",
		"integration",
		nil,
	)
	return &TranslationPipelineChallenge{BaseChallenge: bc}
}

// Execute runs the translation pipeline challenge.
func (c *TranslationPipelineChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestPipeline", "./pkg/translator/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "translation_pipeline",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewVerificationChallenge verifies translation quality verification.
type VerificationChallenge struct {
	challenge.BaseChallenge
}

// NewVerificationChallenge creates the verification challenge.
func NewVerificationChallenge() *VerificationChallenge {
	bc := challenge.NewBaseChallenge(
		"verification-quality",
		"Verification Quality",
		"Verifies translation quality verification and multi-pass refinement",
		"unit",
		nil,
	)
	return &VerificationChallenge{BaseChallenge: bc}
}

// Execute runs the verification challenge.
func (c *VerificationChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestVerify", "./pkg/verification/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "verification",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewE2ETranslationChallenge verifies end-to-end translation with mock providers.
type E2ETranslationChallenge struct {
	challenge.BaseChallenge
}

// NewE2ETranslationChallenge creates the E2E translation challenge.
func NewE2ETranslationChallenge() *E2ETranslationChallenge {
	bc := challenge.NewBaseChallenge(
		"e2e-translation",
		"E2E Translation",
		"Verifies end-to-end translation workflow with mock LLM providers",
		"e2e",
		nil,
	)
	return &E2ETranslationChallenge{BaseChallenge: bc}
}

// Execute runs the E2E translation challenge.
func (c *E2ETranslationChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-tags", "e2e", "-run", "TestE2E", "./test/e2e/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "e2e_translation",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}
