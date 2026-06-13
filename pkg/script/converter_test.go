package script

import (
	"testing"
)

func TestNewConverter(t *testing.T) {
	converter := NewConverter()
	if converter == nil {
		t.Fatal("NewConverter() returned nil")
	}
	if converter.cyrlToLatn == nil {
		t.Error("cyrlToLatn map not initialized")
	}
	if converter.latnToCyrl == nil {
		t.Error("latnToCyrl map not initialized")
	}
}

func TestToLatin(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		cyrillic string
		latin    string
	}{
		// All-caps Cyrillic: digraphs Љ/Њ/Џ each sit before an uppercase letter,
		// so they correctly transliterate all-caps (LJ/NJ/DŽ), not title case.
		{"АБВГДЂЕЖЗИЈКЛЉМНЊОПРСТЋУФХЦЧЏШ", "ABVGDĐEŽZIJKLLJMNNJOPRSTĆUFHCČDŽŠ"},
		{"абвгдђежзијклљмнњопрстћуфхцчџш", "abvgdđežzijklljmnnjoprstćufhcčdžš"},
		{"Пример текста на српском", "Primer teksta na srpskom"},
		{"Љубав је лепа ствар", "Ljubav je lepa stvar"},
		{"Ђорђе је добар човек", "Đorđe je dobar čovek"},
		{"Шта радиш?", "Šta radiš?"},
		{"Чекај мало", "Čekaj malo"},
		{"Цена је 100 динара", "Cena je 100 dinara"},
		{"Hello world", "Hello world"}, // Non-Cyrillic text unchanged
		{"", ""},                       // Empty string
	}

	for _, test := range tests {
		result := converter.ToLatin(test.cyrillic)
		if result != test.latin {
			t.Errorf("ToLatin(%q) = %q, expected %q", test.cyrillic, result, test.latin)
		}
	}
}

func TestToCyrillic(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		latin    string
		cyrillic string
	}{
		{"ABVDGDŽEŽZIJKLLjMNjOPRSTĆUFHCČDžŠ", "АБВДГЏЕЖЗИЈКЛЉМЊОПРСТЋУФХЦЧЏШ"},
		{"abvgdđežzijklljmnnjoprstćufhcčdžš", "абвгдђежзијклљмнњопрстћуфхцчџш"},
		{"Primer teksta na srpskom", "Пример текста на српском"},
		{"Ljubav je lepa stvar", "Љубав је лепа ствар"},
		{"Đorđe je dobar čovek", "Ђорђе је добар човек"},
		{"Šta radiš?", "Шта радиш?"},
		{"Čekaj malo", "Чекај мало"},
		{"Cena je 100 dinara", "Цена је 100 динара"},
		{"Hello world", "Хелло wорлд"}, // English text gets converted (contains Serbian letters)
		{"", ""},                       // Empty string
		{"NjEGOŠ", "ЊЕГОШ"},            // Mixed case
		{"DŽEM", "ЏЕМ"},                // Uppercase digraph
		{"ljUBAV", "љУБАВ"},            // Mixed case digraph
	}

	for _, test := range tests {
		result := converter.ToCyrillic(test.latin)
		if result != test.cyrillic {
			t.Errorf("ToCyrillic(%q) = %q, expected %q", test.latin, result, test.cyrillic)
		}
	}
}

func TestRoundTripConversion(t *testing.T) {
	converter := NewConverter()

	testTexts := []string{
		"Пример текста на српском ћириличном писму",
		"Љубав је лепа ствар која траје вечно",
		"Ђорђе је добар човек и поштен радник",
		"Шта ћемо радити после посла?",
		"Чекаћу те код куће до осам часова",
		"Цена живота је у његовој лепоти",
	}

	for _, original := range testTexts {
		// Cyrillic -> Latin -> Cyrillic
		latin := converter.ToLatin(original)
		backToCyrillic := converter.ToCyrillic(latin)

		if backToCyrillic != original {
			t.Errorf("Round-trip conversion failed:\nOriginal: %q\nLatin: %q\nBack: %q", original, latin, backToCyrillic)
		}
	}
}

func TestDetectScript(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		text     string
		expected ScriptType
	}{
		{"Пример текста на српском", Cyrillic},
		{"Primer teksta na srpskom", Latin},
		{"Hello world", Latin}, // English defaults to Latin
		{"", Latin},            // Empty defaults to Latin
		{"Љубав", Cyrillic},
		{"Ljubav", Latin},
		{"Текст123", Cyrillic}, // Cyrillic with numbers
		{"Text123", Latin},     // Latin with numbers
		{"Пример", Cyrillic},   // Single word
		{"Primer", Latin},      // Single word
	}

	for _, test := range tests {
		result := converter.DetectScript(test.text)
		if result != test.expected {
			t.Errorf("DetectScript(%q) = %v, expected %v", test.text, result, test.expected)
		}
	}
}

func TestConvert(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		input    string
		target   ScriptType
		expected string
	}{
		{"Пример", Latin, "Primer"},
		{"Primer", Cyrillic, "Пример"},
		{"Љубав", Latin, "Ljubav"},
		{"Ljubav", Cyrillic, "Љубав"},
		{"Hello", Latin, "Hello"},    // Already target script
		{"Hello", Cyrillic, "Хелло"}, // Converts English text with Serbian letters
		{"", Latin, ""},              // Empty string
		{"", Cyrillic, ""},           // Empty string
	}

	for _, test := range tests {
		result := converter.Convert(test.input, test.target)
		if result != test.expected {
			t.Errorf("Convert(%q, %v) = %q, expected %q", test.input, test.target, result, test.expected)
		}
	}
}

func TestScriptTypeConstants(t *testing.T) {
	if Cyrillic != "cyrillic" {
		t.Errorf("Cyrillic constant = %q, expected 'cyrillic'", Cyrillic)
	}
	if Latin != "latin" {
		t.Errorf("Latin constant = %q, expected 'latin'", Latin)
	}
}

func TestComplexSentences(t *testing.T) {
	converter := NewConverter()

	cyrillicText := "Љубав је лепа ствар, али није увек лако наћи праву љубав. Ђорђе је срећан човек који има добар посао и лепу породицу."
	expectedLatin := "Ljubav je lepa stvar, ali nije uvek lako naći pravu ljubav. Đorđe je srećan čovek koji ima dobar posao i lepu porodicu."

	result := converter.ToLatin(cyrillicText)
	if result != expectedLatin {
		t.Errorf("Complex sentence conversion failed:\nInput: %q\nExpected: %q\nGot: %q", cyrillicText, expectedLatin, result)
	}

	// Test round-trip
	back := converter.ToCyrillic(result)
	if back != cyrillicText {
		t.Errorf("Round-trip failed for complex sentence")
	}
}

func TestSpecialCharacters(t *testing.T) {
	converter := NewConverter()

	// Test with punctuation and special characters
	cyrillicText := "Шта?! Ћути... Чекај!"
	expectedLatin := "Šta?! Ćuti... Čekaj!"

	result := converter.ToLatin(cyrillicText)
	if result != expectedLatin {
		t.Errorf("Special characters test failed:\nInput: %q\nExpected: %q\nGot: %q", cyrillicText, expectedLatin, result)
	}
}

func TestMultiCharacterSequences(t *testing.T) {
	converter := NewConverter()

	// Test that multi-character sequences are handled correctly
	tests := []struct {
		input    string
		expected string
	}{
		{"Љ", "Lj"},
		{"љ", "lj"},
		{"Њ", "Nj"},
		{"њ", "nj"},
		{"Џ", "Dž"},
		{"џ", "dž"},
		{"Ћ", "Ć"},
		{"ћ", "ć"},
		{"Ч", "Č"},
		{"ч", "č"},
		{"Ш", "Š"},
		{"ш", "š"},
		{"Ђ", "Đ"},
		{"ђ", "đ"},
	}

	for _, test := range tests {
		result := converter.ToLatin(test.input)
		if result != test.expected {
			t.Errorf("ToLatin(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestLatinToCyrillicMultiChar(t *testing.T) {
	converter := NewConverter()

	// Test that Latin multi-character sequences convert back correctly
	tests := []struct {
		input    string
		expected string
	}{
		{"Lj", "Љ"},
		{"lj", "љ"},
		{"Nj", "Њ"},
		{"nj", "њ"},
		{"Dž", "Џ"},
		{"dž", "џ"},
		{"Ć", "Ћ"},
		{"ć", "ћ"},
		{"Č", "Ч"},
		{"č", "ч"},
		{"Š", "Ш"},
		{"š", "ш"},
		{"Đ", "Ђ"},
		{"đ", "ђ"},
	}

	for _, test := range tests {
		result := converter.ToCyrillic(test.input)
		if result != test.expected {
			t.Errorf("ToCyrillic(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
