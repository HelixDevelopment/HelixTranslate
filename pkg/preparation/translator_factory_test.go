package preparation

import (
	"context"
	"os"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/translator"
)

// R-1c intermediate plumbing guard for PreparationAwareTranslator.
//
// §11.4.115 polarity switch: RED_MODE=1 reproduces the defect (the factory set
// via SetEnsembleFactory is NOT threaded into runPreparation's coordinator
// construction) so the guard FAILs on the broken behaviour; RED_MODE=0 (default)
// is the standing GREEN regression guard asserting the injected factory is
// threaded all the way through SetEnsembleFactory -> runPreparation ->
// NewPreparationCoordinatorWithFactory(ctx, ...) -> buildProviders -> factory(ctx).
//
// To capture the RED proof on the threading-removed behaviour, run:
//
//	RED_MODE=1 go test -run TestPreparationAwareTranslator_EnsembleFactoryThreaded ./pkg/preparation/
//
// which skips the SetEnsembleFactory injection, so runPreparation builds the
// coordinator via the built-in (nil-factory) path, the factory is never invoked,
// and the "factory was called" assertion FAILs — proving the test catches an
// intermediate that fails to thread the factory to the leaf seam.

// recordingFactoryTranslator is a zero-network translator.Translator stub used
// only so the injected factory returns a non-empty provider set (avoiding the
// constructor's honest "no valid LLM providers available" error) — its
// translation output is irrelevant to the threading assertion.
type recordingFactoryTranslator struct{ name string }

func (s *recordingFactoryTranslator) Translate(_ context.Context, _ string, _ string) (string, error) {
	return "{}", nil
}

func (s *recordingFactoryTranslator) TranslateWithProgress(
	ctx context.Context, text string, c string, _ *events.EventBus, _ string,
) (string, error) {
	return s.Translate(ctx, text, c)
}

func (s *recordingFactoryTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}

func (s *recordingFactoryTranslator) GetName() string { return s.name }

func TestPreparationAwareTranslator_EnsembleFactoryThreaded(t *testing.T) {
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
			&recordingFactoryTranslator{name: "stub-a"},
			&recordingFactoryTranslator{name: "stub-b"},
		}, nil
	}

	prepConfig := &PreparationConfig{
		SourceLanguage: "en",
		TargetLanguage: "es",
		PassCount:      1,
		// No analysis flags + no real providers: the stub passes will not produce a
		// usable analysis, but runPreparation swallows per-pass failures and
		// continues — the assertion is on whether the factory was reached, not on
		// analysis content.
	}

	// Base translator is a recording stub: translateWithPreparationContext runs
	// after runPreparation and translates metadata/chapters via this base.
	base := &recordingFactoryTranslator{name: "base"}

	pat := NewPreparationAwareTranslator(
		base,
		nil, // no language detector
		language.Language{Code: "en", Name: "English"},
		language.Language{Code: "es", Name: "Spanish"},
		prepConfig,
	)

	// The threading under test: inject the factory through the intermediate's
	// public setter, then drive runPreparation. RED_MODE=1 simulates the
	// "threading removed" mutation — runPreparation builds its coordinator with a
	// nil factory instead of the injected field — by writing the field to a
	// scratch struct the run never reads, while STILL injecting nothing into the
	// live pat. The GREEN assertions below (factory MUST be invoked with the run's
	// ctx) are identical in both modes, so under the mutation they FAIL.
	if redMode {
		(&PreparationAwareTranslator{}).SetEnsembleFactory(factory) // mutation: field set on a discarded instance, never threaded
	} else {
		pat.SetEnsembleFactory(factory)
	}

	// A ctx carrying a sentinel so we can prove runPreparation passed THIS ctx
	// (the one in scope for the run) down to factory(ctx).
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "sentinel")

	book := &ebook.Book{
		Metadata: ebook.Metadata{Title: "T"},
		Chapters: []ebook.Chapter{{Title: "C1"}},
	}

	// TranslateBook -> runPreparation (builds coordinator with the threaded
	// factory) -> translateWithPreparationContext. We ignore the returned error:
	// preparation failures are non-fatal by design and the stub base translator
	// completes the standard pipeline.
	_ = pat.TranslateBook(ctx, book, nil, "prep-thread-sess")

	// Identical assertions in both modes. RED_MODE (threading removed: the live
	// pat has no factory) leaves rec.invoked false → this FAILs, proving the guard
	// catches an intermediate that does not thread the factory to the leaf seam.
	if !rec.invoked {
		t.Fatal("expected the injected factory to be threaded into runPreparation and invoked, but it was not")
	}
	if rec.gotCtx == nil || rec.gotCtx.Value(ctxKey{}) != "sentinel" {
		t.Fatalf("runPreparation must thread the run's ctx into factory(ctx); got ctx value %v",
			func() any {
				if rec.gotCtx == nil {
					return nil
				}
				return rec.gotCtx.Value(ctxKey{})
			}())
	}
}

// TestPreparationAwareTranslator_NilFactoryUnchanged proves that with no factory
// injected the intermediate uses the built-in (nil-factory) construction path:
// runPreparation calls NewPreparationCoordinatorWithFactory with a nil factory
// (equivalent to the original NewPreparationCoordinator call), so no external
// factory is ever consulted. We assert the default field is nil after construction.
func TestPreparationAwareTranslator_NilFactoryUnchanged(t *testing.T) {
	pat := NewPreparationAwareTranslator(
		&recordingFactoryTranslator{name: "base"},
		nil,
		language.Language{Code: "en", Name: "English"},
		language.Language{Code: "es", Name: "Spanish"},
		&PreparationConfig{SourceLanguage: "en", TargetLanguage: "es", PassCount: 1},
	)

	if pat.ensembleFactory != nil {
		t.Fatal("default construction must leave ensembleFactory nil (behaviour-preserving)")
	}
}
