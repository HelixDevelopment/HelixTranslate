package script

import (
	"strings"
	"unicode"
)

// upperDigraphs maps each uppercase Cyrillic digraph rune to its all-caps Latin
// form. Standard Serbian transliteration renders an uppercase digraph in title
// case (Lj/Nj/Dž) before a lowercase letter or in isolation, but all-caps
// (LJ/NJ/DŽ) when it sits inside an all-caps run (followed by an uppercase
// letter), so "ЉУБАВ" -> "LJUBAV" not "LjUBAV".
var upperDigraphs = map[rune]string{
	'Љ': "LJ",
	'Њ': "NJ",
	'Џ': "DŽ",
}

// ScriptType represents the script type
type ScriptType string

const (
	Cyrillic ScriptType = "cyrillic"
	Latin    ScriptType = "latin"
)

// Converter handles Serbian Cyrillic/Latin conversion
type Converter struct {
	cyrlToLatn map[rune]string
	latnToCyrl map[string]rune
}

// NewConverter creates a new script converter
func NewConverter() *Converter {
	cyrlToLatn := map[rune]string{
		'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Ђ': "Đ", 'Е': "E", 'Ж': "Ž", 'З': "Z",
		'И': "I", 'Ј': "J", 'К': "K", 'Л': "L", 'Љ': "Lj", 'М': "M", 'Н': "N", 'Њ': "Nj", 'О': "O",
		'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'Ћ': "Ć", 'У': "U", 'Ф': "F", 'Х': "H", 'Ц': "C",
		'Ч': "Č", 'Џ': "Dž", 'Ш': "Š",
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'ђ': "đ", 'е': "e", 'ж': "ž", 'з': "z",
		'и': "i", 'ј': "j", 'к': "k", 'л': "l", 'љ': "lj", 'м': "m", 'н': "n", 'њ': "nj", 'о': "o",
		'п': "p", 'р': "r", 'с': "s", 'т': "t", 'ћ': "ć", 'у': "u", 'ф': "f", 'х': "h", 'ц': "c",
		'ч': "č", 'џ': "dž", 'ш': "š",
	}

	// Build reverse mapping
	latnToCyrl := make(map[string]rune)
	for cyrl, latn := range cyrlToLatn {
		latnToCyrl[latn] = cyrl
	}
	// Add uppercase digraphs
	latnToCyrl["LJ"] = 'Љ'
	latnToCyrl["NJ"] = 'Њ'
	latnToCyrl["DŽ"] = 'Џ'

	return &Converter{
		cyrlToLatn: cyrlToLatn,
		latnToCyrl: latnToCyrl,
	}
}

// ToLatin converts Cyrillic Serbian to Latin
func (c *Converter) ToLatin(text string) string {
	var result strings.Builder
	result.Grow(len(text))

	runes := []rune(text)
	for i, char := range runes {
		// Uppercase digraphs (Љ/Њ/Џ) render all-caps inside an all-caps run.
		// The digraph is all-caps when the NEXT letter is uppercase (clearly
		// mid-all-caps-word), OR when the next position is NOT a lowercase
		// letter (word end / non-letter) AND the PREVIOUS letter was uppercase
		// (the digraph closes an all-caps run, e.g. "ВИДИ Љ." -> "VIDI LJ.").
		// Otherwise (lowercase next, or isolated) keep title case Lj/Nj/Dž.
		if allCaps, ok := upperDigraphs[char]; ok {
			if digraphIsAllCaps(runes, i) {
				result.WriteString(allCaps)
			} else {
				result.WriteString(c.cyrlToLatn[char])
			}
			continue
		}

		if latin, ok := c.cyrlToLatn[char]; ok {
			result.WriteString(latin)
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
}

// digraphIsAllCaps decides whether the uppercase digraph at index i should
// render all-caps, by inspecting its letter neighbours.
func digraphIsAllCaps(runes []rune, i int) bool {
	nextUpper := isUpperLetterAt(runes, i+1)
	nextLower := isLowerLetterAt(runes, i+1)
	if nextUpper {
		return true
	}
	// Next is not a lowercase letter (word end or punctuation): fall back to the
	// preceding letter to tell an all-caps run from an isolated title-case form.
	if !nextLower && isUpperLetterAt(runes, i-1) {
		return true
	}
	return false
}

func isUpperLetterAt(runes []rune, i int) bool {
	if i < 0 || i >= len(runes) {
		return false
	}
	r := runes[i]
	return unicode.IsLetter(r) && unicode.IsUpper(r)
}

func isLowerLetterAt(runes []rune, i int) bool {
	if i < 0 || i >= len(runes) {
		return false
	}
	r := runes[i]
	return unicode.IsLetter(r) && unicode.IsLower(r)
}

// ToCyrillic converts Latin Serbian to Cyrillic
func (c *Converter) ToCyrillic(text string) string {
	// This is more complex due to multi-character Latin equivalents (Lj, Nj, Dž)
	// We need to check for multi-character sequences first
	result := strings.Builder{}
	result.Grow(len(text))

	i := 0
	runes := []rune(text)
	for i < len(runes) {
		// Try 2-character sequence first
		if i+1 < len(runes) {
			twoChar := string(runes[i : i+2])
			if cyrl, ok := c.latnToCyrl[twoChar]; ok {
				result.WriteRune(cyrl)
				i += 2
				continue
			}
		}

		// Try single character
		oneChar := string(runes[i])
		if cyrl, ok := c.latnToCyrl[oneChar]; ok {
			result.WriteRune(cyrl)
		} else {
			result.WriteRune(runes[i])
		}
		i++
	}

	return result.String()
}

// DetectScript detects the script type of the text
func (c *Converter) DetectScript(text string) ScriptType {
	cyrillicCount := 0
	latinCount := 0

	for _, char := range text {
		if _, ok := c.cyrlToLatn[char]; ok {
			cyrillicCount++
		}
		// Simple heuristic for Latin detection
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			latinCount++
		}
	}

	if cyrillicCount > latinCount {
		return Cyrillic
	}
	return Latin
}

// Convert automatically converts between scripts
func (c *Converter) Convert(text string, targetScript ScriptType) string {
	currentScript := c.DetectScript(text)

	if currentScript == targetScript {
		return text
	}

	if targetScript == Latin {
		return c.ToLatin(text)
	}
	return c.ToCyrillic(text)
}
