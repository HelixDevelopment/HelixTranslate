package unit

import (
	"digital.vasic.translator/pkg/script"
	"testing"
)

func TestScriptConverter(t *testing.T) {
	converter := script.NewConverter()

	t.Run("CyrillicToLatin", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"Ратибор", "Ratibor"},
			{"Одјеци", "Odjeci"},
			{"Београд", "Beograd"},
			{"Љубљана", "Ljubljana"},
			{"Њујорк", "Njujork"},
			{"Тестовање", "Testovanje"},
		}

		for _, tt := range tests {
			result := converter.ToLatin(tt.input)
			if result != tt.expected {
				t.Errorf("ToLatin(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("DetectScript", func(t *testing.T) {
		tests := []struct {
			input    string
			expected script.ScriptType
		}{
			{"Ратибор", script.Cyrillic},
			{"Ratibor", script.Latin},
			{"Test", script.Latin},
			{"Београд", script.Cyrillic},
		}

		for _, tt := range tests {
			result := converter.DetectScript(tt.input)
			if result != tt.expected {
				t.Errorf("DetectScript(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("ConvertAutoDetect", func(t *testing.T) {
		// Cyrillic to Latin
		result := converter.Convert("Ратибор", script.Latin)
		if result != "Ratibor" {
			t.Errorf("Convert(Ратибор, Latin) = %s, want Ratibor", result)
		}

		// Already in target script
		result = converter.Convert("Ratibor", script.Latin)
		if result != "Ratibor" {
			t.Errorf("Convert(Ratibor, Latin) = %s, want Ratibor", result)
		}
	})
}
