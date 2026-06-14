package main

import "testing"

// TestValidateOutputFormat is the regression guard for the bug where the
// -format CLI flag was never validated against its declared closed set
// {epub, md}. Step 4 of the pipeline runs ONLY for exactly "epub" or exactly
// "md" (main.go conversion branches); any other value (typo "epubb", wrong
// case "EPUB", unsupported "pdf") fell through producing NO output file, yet
// the program still printed "✅ Translation complete!" and listed a "Final
// EPUB" path that was never created — a §11.4.1 false-success / swallowed-error
// bluff: a failed run reported success and pointed the user at a nonexistent file.
//
// §11.4.115 polarity: with the fix wired (validateOutputFormat rejects anything
// outside {epub, md}) this is the GREEN regression-guard. On the pre-fix code
// the helper did not exist, so this test fails by construction (compile error),
// and the silent-no-output behaviour shipped.
func TestValidateOutputFormat(t *testing.T) {
	tests := []struct {
		name   string
		format string
		wantOK bool
	}{
		{"epub is supported", "epub", true},
		{"md is supported", "md", true},
		{"unsupported pdf is rejected", "pdf", false},
		{"typo epubb is rejected", "epubb", false},
		{"wrong-case EPUB is rejected", "EPUB", false},
		{"wrong-case MD is rejected", "MD", false},
		{"markdown long form is rejected", "markdown", false},
		{"empty format is rejected", "", false},
		{"leading space is rejected", " epub", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOutputFormat(tt.format)
			gotOK := err == nil
			if gotOK != tt.wantOK {
				t.Fatalf("validateOutputFormat(%q): ok=%v (err=%v), want ok=%v",
					tt.format, gotOK, err, tt.wantOK)
			}
		})
	}
}
