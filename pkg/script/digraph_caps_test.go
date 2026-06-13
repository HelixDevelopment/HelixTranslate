package script

import "testing"

// TestToLatin_UppercaseDigraphCaseRendering is the permanent regression guard
// for the digraph capitalization defect: an uppercase Cyrillic digraph (Љ/Њ/Џ)
// must render all-caps (LJ/NJ/DŽ) inside an all-caps run, but title case
// (Lj/Nj/Dž) before a lowercase letter, at a word end, or in isolation.
//
// Mutation proof (§1.1): reverting ToLatin to always emit cyrlToLatn[char]
// (the title-case "Lj") makes the all-caps cases below FAIL ("LjUBAV" != "LJUBAV").
func TestToLatin_UppercaseDigraphCaseRendering(t *testing.T) {
	c := NewConverter()
	cases := []struct {
		name string
		cyr  string
		want string
	}{
		// All-caps runs -> all-caps digraph.
		{"ljubav all caps", "ЉУБАВ", "LJUBAV"},
		{"njegos all caps", "ЊЕГОШ", "NJEGOŠ"},
		{"dzem all caps", "ЏЕМ", "DŽEM"},
		{"odzak caps mid", "ОЏАК", "ODŽAK"},
		{"enja caps", "ЕЊА", "ENJA"},
		// Title case (uppercase digraph then lowercase) -> title case.
		{"ljubav title", "Љубав", "Ljubav"},
		{"njegos title", "Његош", "Njegoš"},
		{"dzak title", "Џак", "Džak"},
		// Lone uppercase digraph (no following letter) -> title case (per existing
		// TestMultiCharacterConversion expectation Љ->Lj).
		{"lone Lj", "Љ", "Lj"},
		{"lone Nj", "Њ", "Nj"},
		{"lone Dz", "Џ", "Dž"},
		// Uppercase digraph followed by a non-letter -> title case.
		{"Lj then space", "Љ ЛЕП", "Lj LEP"},
		{"Nj then digit", "Њ2", "Nj2"},
		// A lone digraph separated from the all-caps word by a space (no adjacent
		// uppercase LETTER) is genuinely ambiguous; render conservative title
		// case rather than guess an all-caps run across the space (§11.4.6).
		{"Lj after space then dot", "ВИДИ Љ.", "VIDI Lj."},
		// Adjacency to an uppercase letter forces all-caps even when the digraph
		// ends the word (prev letter uppercase, next is end-of-text).
		{"digraph closes caps word", "ВИЉ", "VILJ"},
		{"digraph closes caps then dot", "ВИЉ.", "VILJ."},
		// Lowercase digraph is unaffected.
		{"lowercase lj", "љубав", "ljubav"},
		{"lowercase nj", "његош", "njegoš"},
		// Mixed words in a sentence.
		{"sentence mixed", "ЊЕГОШ је написао Горски вијенац", "NJEGOŠ je napisao Gorski vijenac"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.ToLatin(tc.cyr); got != tc.want {
				t.Errorf("ToLatin(%q) = %q, want %q", tc.cyr, got, tc.want)
			}
		})
	}
}

// TestToLatin_DigraphCaps_RoundTripStable confirms that the all-caps Latin form
// round-trips back to the original Cyrillic (uses the LJ/NJ/DŽ reverse map).
func TestToLatin_DigraphCaps_RoundTripStable(t *testing.T) {
	c := NewConverter()
	for _, src := range []string{"ЉУБАВ", "ЊЕГОШ", "ЏЕМ", "ОЏАК", "Љубав", "Његош"} {
		lat := c.ToLatin(src)
		if back := c.ToCyrillic(lat); back != src {
			t.Errorf("round-trip %q -> %q -> %q (mismatch)", src, lat, back)
		}
	}
}
