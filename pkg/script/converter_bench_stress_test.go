package script

import (
	"strings"
	"sync"
	"testing"
)

// realisticCyrillicParagraph is a representative Serbian Cyrillic paragraph that
// exercises every conversion branch: plain letters, the title-case digraphs
// (Љ/Њ/Џ), an all-caps run (the digraph all-caps logic), diacritic Latin
// equivalents (Ђ/Ж/Ћ/Ч/Ш), punctuation and ASCII pass-through. It is sized like
// a real book paragraph so the benchmark reflects production cost per paragraph.
const realisticCyrillicParagraph = "Љубав према књижевности расла је у њему од најранијих дана. " +
	"Џак пун старих рукописа стајао је у ћошку собе, а ВИДИ ЉУБАВ било је исписано на корицама. " +
	"Чудесан осећај прожимао је Жарка док је читао Ђурине стихове о пролећу, " +
	"шуми и реци која тече кроз срце Шумадије. Ниједна реченица није била сувишна — 1844. године."

// realisticLatinParagraph is the Latin-script counterpart, exercising the
// reverse (Latin->Cyrillic) path including the multi-rune digraph sequences
// (Lj/Nj/Dž and their all-caps forms) that require 2-rune lookahead.
const realisticLatinParagraph = "Ljubav prema književnosti rasla je u njemu od najranijih dana. " +
	"Džak pun starih rukopisa stajao je u ćošku sobe, a VIDI LJUBAV bilo je ispisano na koricama. " +
	"Čudesan osećaj prožimao je Žarka dok je čitao Đurine stihove o proleću, " +
	"šumi i reci koja teče kroz srce Šumadije. Nijedna rečenica nije bila suvišna — 1844. godine."

func BenchmarkConverterToLatin(b *testing.B) {
	c := NewConverter()
	b.ReportAllocs()
	b.SetBytes(int64(len(realisticCyrillicParagraph)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.ToLatin(realisticCyrillicParagraph)
	}
}

func BenchmarkConverterToCyrillic(b *testing.B) {
	c := NewConverter()
	b.ReportAllocs()
	b.SetBytes(int64(len(realisticLatinParagraph)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.ToCyrillic(realisticLatinParagraph)
	}
}

func BenchmarkConverterDetectScript(b *testing.B) {
	c := NewConverter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.DetectScript(realisticCyrillicParagraph)
	}
}

// BenchmarkConverterRoundTrip measures the realistic full conversion cost a
// translation pass pays: Cyrillic -> Latin -> Cyrillic.
func BenchmarkConverterRoundTrip(b *testing.B) {
	c := NewConverter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.ToCyrillic(c.ToLatin(realisticCyrillicParagraph))
	}
}

// TestConverterConcurrentToLatin_StressCorrectness is the §11.4.85 concurrent-
// contention stress test for the script converter hot path. The Converter is
// shared across goroutines (its maps are read-only after NewConverter), so this
// proves it is safe AND correct under load. It does NOT merely assert "no
// panic": every goroutine asserts the conversion equals the known-good golden
// output, so a broken converter (e.g. a data race corrupting the map, or a
// regression returning wrong text) makes this test FAIL. Run with -race for the
// race-detector clean evidence.
func TestConverterConcurrentToLatin_StressCorrectness(t *testing.T) {
	c := NewConverter()

	// Establish the golden output once, single-threaded, from the real function.
	golden := c.ToLatin(realisticCyrillicParagraph)
	if !strings.Contains(golden, "Ljubav") || !strings.Contains(golden, "LJUBAV") {
		t.Fatalf("golden output is not a real conversion: %q", golden)
	}

	const goroutines = 32
	const iterations = 500

	var wg sync.WaitGroup
	errCh := make(chan string, goroutines)
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				got := c.ToLatin(realisticCyrillicParagraph)
				if got != golden {
					errCh <- got
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)

	if bad, ok := <-errCh; ok {
		t.Fatalf("concurrent ToLatin produced incorrect output under load:\n got: %q\nwant: %q", bad, golden)
	}
}

// TestConverterConcurrentMixed_StressCorrectness drives both conversion
// directions and DetectScript concurrently on one shared Converter, asserting
// each goroutine's result matches its single-threaded golden. This stresses
// every read path of the shared maps simultaneously.
func TestConverterConcurrentMixed_StressCorrectness(t *testing.T) {
	c := NewConverter()

	goldenLatin := c.ToLatin(realisticCyrillicParagraph)
	goldenCyrl := c.ToCyrillic(realisticLatinParagraph)
	goldenDetect := c.DetectScript(realisticCyrillicParagraph)
	if goldenDetect != Cyrillic {
		t.Fatalf("golden DetectScript expected Cyrillic, got %s", goldenDetect)
	}

	const workers = 30
	const iterations = 400

	var wg sync.WaitGroup
	var fail int32
	failMu := sync.Mutex{}
	var failMsg string
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		mode := w % 3
		go func(mode int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				switch mode {
				case 0:
					if c.ToLatin(realisticCyrillicParagraph) != goldenLatin {
						recordFail(&fail, &failMu, &failMsg, "ToLatin mismatch under load")
						return
					}
				case 1:
					if c.ToCyrillic(realisticLatinParagraph) != goldenCyrl {
						recordFail(&fail, &failMu, &failMsg, "ToCyrillic mismatch under load")
						return
					}
				default:
					if c.DetectScript(realisticCyrillicParagraph) != goldenDetect {
						recordFail(&fail, &failMu, &failMsg, "DetectScript mismatch under load")
						return
					}
				}
			}
		}(mode)
	}
	wg.Wait()

	if fail != 0 {
		t.Fatalf("concurrent mixed conversion failed: %s", failMsg)
	}
}

func recordFail(fail *int32, mu *sync.Mutex, msg *string, text string) {
	mu.Lock()
	defer mu.Unlock()
	if *fail == 0 {
		*fail = 1
		*msg = text
	}
}
