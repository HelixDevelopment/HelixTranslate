package verification

import (
	"context"
	"os"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
)

// R-1c intermediate plumbing guard for MultiPassPolisher.
//
// §11.4.115 polarity switch: RED_MODE=1 reproduces the defect (the factory set
// via SetEnsembleFactory is NOT threaded into performPass's BookPolisher
// construction) so the guard FAILs on the broken behaviour; RED_MODE=0 (default)
// is the standing GREEN regression guard asserting the injected factory is
// threaded through SetEnsembleFactory -> performPass ->
// NewBookPolisherWithFactory(ctx, ...) -> factory(ctx).
//
// To capture the RED proof on the threading-removed behaviour, run:
//
//	RED_MODE=1 go test -run TestMultiPassPolisher_EnsembleFactoryThreaded ./pkg/verification/
//
// which skips the SetEnsembleFactory injection, so performPass builds each
// polisher via the built-in (nil-factory) path, the factory is never invoked,
// and the "factory was called" assertion FAILs — proving the test catches an
// intermediate that fails to thread the factory to the leaf seam.

// recordingMPTranslator is a zero-network translator.Translator stub returning a
// parseable verification response so PolishBook completes without any LLM call.
type recordingMPTranslator struct{ name string }

func (s *recordingMPTranslator) Translate(_ context.Context, _ string, _ string) (string, error) {
	return cannedVerification, nil
}

func (s *recordingMPTranslator) TranslateWithProgress(
	_ context.Context, _ string, _ string, _ *events.EventBus, _ string,
) (string, error) {
	return cannedVerification, nil
}

func (s *recordingMPTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}

func (s *recordingMPTranslator) GetName() string { return s.name }

func TestMultiPassPolisher_EnsembleFactoryThreaded(t *testing.T) {
	redMode := os.Getenv("RED_MODE") == "1"

	type call struct {
		invoked bool
		gotCtx  context.Context
	}
	rec := &call{}

	var factory EnsembleTranslatorFactory = func(ctx context.Context) ([]translator.Translator, error) {
		rec.invoked = true
		rec.gotCtx = ctx
		return []translator.Translator{
			&recordingMPTranslator{name: "alpha"},
			&recordingMPTranslator{name: "beta"},
		}, nil
	}

	mpp, err := NewMultiPassPolisher(MultiPassConfig{
		PassCount:    1,
		MinConsensus: 1,
		// Deliberately empty providers/configs: the bridge case supplies the
		// provider set via the threaded factory, not config.
		VerifySpirit:     true,
		VerifyLanguage:   true,
		VerifyContext:    true,
		VerifyVocabulary: true,
	}, nil, "mp-thread-sess")
	if err != nil {
		t.Fatalf("NewMultiPassPolisher returned error: %v", err)
	}

	// The threading under test: inject the factory through the intermediate's
	// public setter, then drive PolishBook -> performPass. RED_MODE=1 simulates
	// the "threading removed" mutation — performPass builds each polisher with a
	// nil factory instead of the injected field — by writing the field to a scratch
	// struct the run never reads, while STILL injecting nothing into the live mpp.
	// The GREEN assertions below (factory MUST be invoked with the pass's ctx) are
	// identical in both modes, so under the mutation they FAIL.
	if redMode {
		(&MultiPassPolisher{}).SetEnsembleFactory(factory) // mutation: field set on a discarded instance, never threaded
	} else {
		mpp.SetEnsembleFactory(factory)
	}

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "sentinel")

	original := &ebook.Book{
		Metadata: ebook.Metadata{Title: "T"},
		Chapters: []ebook.Chapter{{
			Title:    "C1",
			Sections: []ebook.Section{{Content: "some original content for the chapter that is long enough"}},
		}},
	}
	translated := &ebook.Book{
		Metadata: ebook.Metadata{Title: "T"},
		Chapters: []ebook.Chapter{{
			Title:    "C1",
			Sections: []ebook.Section{{Content: "algun contenido original del capitulo suficientemente largo"}},
		}},
	}

	// PolishBook -> performPass (builds the BookPolisher with the threaded
	// factory). In RED_MODE the empty config yields no providers, so PolishBook may
	// return an error; the assertion is on whether the factory was reached.
	_, _ = mpp.PolishBook(ctx, original, translated)

	// Identical assertions in both modes. RED_MODE (threading removed: the live
	// mpp has no factory) leaves rec.invoked false → this FAILs, proving the guard
	// catches an intermediate that does not thread the factory to the leaf seam.
	if !rec.invoked {
		t.Fatal("expected the injected factory to be threaded into performPass and invoked, but it was not")
	}
	if rec.gotCtx == nil || rec.gotCtx.Value(ctxKey{}) != "sentinel" {
		t.Fatalf("performPass must thread the pass's ctx into factory(ctx); got ctx value %v",
			func() any {
				if rec.gotCtx == nil {
					return nil
				}
				return rec.gotCtx.Value(ctxKey{})
			}())
	}
}

// TestMultiPassPolisher_NilFactoryUnchanged proves that with no factory injected
// the intermediate uses the built-in (nil-factory) construction path: performPass
// calls NewBookPolisherWithFactory with a nil factory (equivalent to the original
// NewBookPolisher call). We assert the default field is nil after construction.
func TestMultiPassPolisher_NilFactoryUnchanged(t *testing.T) {
	mpp, err := NewMultiPassPolisher(MultiPassConfig{PassCount: 1}, nil, "mp-nil-sess")
	if err != nil {
		t.Fatalf("NewMultiPassPolisher returned error: %v", err)
	}
	if mpp.ensembleFactory != nil {
		t.Fatal("default construction must leave ensembleFactory nil (behaviour-preserving)")
	}
}
