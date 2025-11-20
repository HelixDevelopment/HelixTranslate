package script

import "strings"

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

	return &Converter{
		cyrlToLatn: cyrlToLatn,
		latnToCyrl: latnToCyrl,
	}
}

// ToLatin converts Cyrillic Serbian to Latin
func (c *Converter) ToLatin(text string) string {
	var result strings.Builder
	result.Grow(len(text))

	for _, char := range text {
		if latin, ok := c.cyrlToLatn[char]; ok {
			result.WriteString(latin)
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
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
