package preparation

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// §11.4.115 RED-baseline-on-the-broken-artifact + polarity switch.
//
// These tests reproduce the real-world LLM-output failure modes of the
// first-`{`..last-`}` extractJSON implementation, then become the standing
// GREEN regression guards once the robust extractor lands.
//
// The contract under test: extractJSON MUST return a span that json.Valid
// considers a valid JSON value for realistic LLM responses (fenced code
// blocks, prose-with-braces preambles, trailing prose, nested objects,
// top-level arrays), and MUST surface "no JSON" honestly rather than emit
// garbage.

// realisticLLMCases enumerates the case-space (§11.4.146 STEP 3 extend).
// `wantValid` = the extracted span must be parseable JSON.
// `wantContains` (when set) = a key/marker that MUST be present in the
// extracted span, proving the intended object was captured (not a wrong
// sub-span grabbed from prose braces).
type extractCase struct {
	name         string
	input        string
	wantValid    bool
	wantContains string
	// wantParsedKey/Val assert a concrete top-level field of the parsed object.
	wantParsedKey string
	wantParsedVal string
}

func robustExtractCases() []extractCase {
	return []extractCase{
		{
			name:          "clean JSON object (no regression)",
			input:         `{"genre": "Fiction"}`,
			wantValid:     true,
			wantParsedKey: "genre",
			wantParsedVal: "Fiction",
		},
		{
			name: "fenced ```json block with prose around it",
			input: "Here is the analysis you requested:\n\n" +
				"```json\n{\"genre\": \"Fiction\", \"tone\": \"Dark\"}\n```\n\n" +
				"Let me know if you need anything else.",
			wantValid:     true,
			wantParsedKey: "genre",
			wantParsedVal: "Fiction",
		},
		{
			name: "bare ``` fenced block (no language tag)",
			input: "Result:\n```\n{\"genre\": \"Poetry\"}\n```\nDone.",
			wantValid:     true,
			wantParsedKey: "genre",
			wantParsedVal: "Poetry",
		},
		{
			name: "prose preamble containing a brace before the real JSON",
			// The classic trap: prose has a `{` in it; first-{..last-} would
			// start the span inside the prose and capture invalid text.
			input:         `Here is the result (format {key:value}): {"genre": "Sci-Fi"}`,
			wantValid:     true,
			wantParsedKey: "genre",
			wantParsedVal: "Sci-Fi",
		},
		{
			name:          "trailing prose after the JSON object",
			input:         `{"genre": "Horror"} -- that concludes the {full} analysis.`,
			wantValid:     true,
			wantParsedKey: "genre",
			wantParsedVal: "Horror",
		},
		{
			name:          "nested object",
			input:         `prefix {"outer": {"inner": "value"}} suffix`,
			wantValid:     true,
			wantContains:  `"inner"`,
		},
		{
			name:          "deeply nested with array",
			input:         `noise {"a": {"b": {"c": [1,2,3]}}} noise`,
			wantValid:     true,
			wantContains:  `"c"`,
		},
		{
			name:          "top-level JSON array",
			input:         `Text before [{"a": 1}, {"b": 2}] text after`,
			wantValid:     true,
			wantContains:  `"b"`,
		},
		{
			name:          "fenced top-level array",
			input:         "```json\n[{\"x\": 1}, {\"y\": 2}]\n```",
			wantValid:     true,
			wantContains:  `"y"`,
		},
		{
			name: "brace inside a JSON string value (must not confuse the scanner)",
			input: `Sure: {"note": "use {placeholder} here", "genre": "Essay"}`,
			wantValid:     true,
			wantParsedKey: "genre",
			wantParsedVal: "Essay",
		},
		{
			name: "escaped quote inside a string value",
			input: `{"title": "She said \"hi\"", "genre": "Drama"}`,
			wantValid:     true,
			wantParsedKey: "genre",
			wantParsedVal: "Drama",
		},
		{
			name:      "no JSON at all -> honest non-JSON",
			input:     `This is just plain prose with no object at all.`,
			wantValid: false,
		},
		{
			name:      "empty input -> honest non-JSON",
			input:     ``,
			wantValid: false,
		},
	}
}

// TestExtractJSON_RobustCases is the standing GREEN regression guard.
// Pre-fix it FAILS (the fenced / prose-brace / array cases), proving the
// defect is genuinely present. Post-fix it PASSES with no regression on the
// clean cases.
func TestExtractJSON_RobustCases(t *testing.T) {
	for _, tc := range robustExtractCases() {
		t.Run(tc.name, func(t *testing.T) {
			got := extractJSON(tc.input)

			if !tc.wantValid {
				assert.False(t, json.Valid([]byte(got)),
					"input has no JSON; extracted span must not be valid JSON, got %q", got)
				return
			}

			require.True(t, json.Valid([]byte(got)),
				"extracted span MUST be valid JSON, got %q", got)

			if tc.wantContains != "" {
				assert.Contains(t, got, tc.wantContains,
					"extracted span must contain the intended marker")
			}

			if tc.wantParsedKey != "" {
				var m map[string]any
				require.NoError(t, json.Unmarshal([]byte(got), &m),
					"extracted object must unmarshal to a map")
				assert.Equal(t, tc.wantParsedVal, m[tc.wantParsedKey],
					"intended top-level field must survive extraction")
			}
		})
	}
}
