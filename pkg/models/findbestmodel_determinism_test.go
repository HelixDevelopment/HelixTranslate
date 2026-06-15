package models

import "testing"

// TestFindBestModel_Deterministic is a reproduce-first guard for the
// non-deterministic "best model" selection bug: FindBestModel iterated the
// registry map (random order) and broke score-ties by "first iterated wins",
// so identical (RAM, langs, GPU) inputs returned DIFFERENT models across runs.
// A translation "best model" selector MUST be deterministic (§11.4.50).
func TestFindBestModel_Deterministic(t *testing.T) {
	cases := []struct {
		name   string
		maxRAM uint64
		langs  []string
		hasGPU bool
	}{
		{"tie_at_top_64GB_en_ru_sr", 64 * 1024 * 1024 * 1024, []string{"en", "ru", "sr"}, false},
		{"8GB_en_ru", 8 * 1024 * 1024 * 1024, []string{"en", "ru"}, false},
		{"16GB_de_es_fr", 16 * 1024 * 1024 * 1024, []string{"de", "es", "fr"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			first, err := r.FindBestModel(tc.maxRAM, tc.langs, tc.hasGPU)
			if err != nil {
				t.Fatalf("FindBestModel error: %v", err)
			}
			// Identical inputs must yield the identical model every time.
			for i := 0; i < 100; i++ {
				got, err := r.FindBestModel(tc.maxRAM, tc.langs, tc.hasGPU)
				if err != nil {
					t.Fatalf("iter %d: FindBestModel error: %v", i, err)
				}
				if got.ID != first.ID {
					t.Fatalf("non-deterministic selection: iter %d returned %q, first returned %q",
						i, got.ID, first.ID)
				}
			}
		})
	}
}
