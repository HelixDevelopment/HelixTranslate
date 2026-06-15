package translator

import (
	"context"
	"errors"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"github.com/stretchr/testify/mock"
)

// TestUniversalTranslator_RespectsCancelledContext is the standing regression
// guard for the context-cancellation defect: TranslateBook used to ignore a
// cancelled context and report a partial / no-op book as a successful
// translation (a §11.4 anti-bluff silent-partial-result defect). A user who
// cancels (timeout / Ctrl-C) MUST get context.Canceled, not a green run.
//
// RED on the pre-fix orchestrator (returned nil); GREEN once TranslateBook
// checks ctx.Err() before start and between chapters.
func TestUniversalTranslator_RespectsCancelledContext(t *testing.T) {
	mockTranslator := &MockTranslator{}
	// translator that ignores ctx and always "succeeds" — exactly the kind of
	// translator that lets the orchestrator-level bug stay invisible.
	mockTranslator.On("TranslateWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("translated", nil).Maybe()

	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}
	ut := NewUniversalTranslator(mockTranslator, nil, sourceLang, targetLang)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled BEFORE the call

	book := &ebook.Book{
		Metadata: ebook.Metadata{Title: "Title"},
		Chapters: []ebook.Chapter{
			{Title: "Ch1", Sections: []ebook.Section{{Title: "S1", Content: "Hello"}}},
			{Title: "Ch2", Sections: []ebook.Section{{Title: "S2", Content: "World"}}},
		},
	}

	err := ut.TranslateBook(ctx, book, events.NewEventBus(), "sess")
	if err == nil {
		t.Fatalf("EXPECTED context.Canceled error from cancelled run; got nil (silent partial-result)")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected error to wrap context.Canceled, got: %v", err)
	}
}

// TestUniversalTranslator_CompletesWithLiveContext confirms the ctx check does
// NOT break the normal (non-cancelled) path — a live context still translates
// the whole book successfully.
func TestUniversalTranslator_CompletesWithLiveContext(t *testing.T) {
	mockTranslator := &MockTranslator{}
	mockTranslator.On("TranslateWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("translated", nil)

	ut := NewUniversalTranslator(mockTranslator, nil,
		language.Language{Code: "en", Name: "English"},
		language.Language{Code: "ru", Name: "Russian"})

	book := &ebook.Book{
		Metadata: ebook.Metadata{Title: "Title"},
		Chapters: []ebook.Chapter{
			{Title: "Ch1", Sections: []ebook.Section{{Title: "S1", Content: "Hello"}}},
		},
	}

	if err := ut.TranslateBook(context.Background(), book, events.NewEventBus(), "sess"); err != nil {
		t.Fatalf("live-context translation must succeed, got: %v", err)
	}
	if book.Chapters[0].Sections[0].Content != "translated" {
		t.Fatalf("content not translated on live path: %q", book.Chapters[0].Sections[0].Content)
	}
}
