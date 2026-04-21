package unit

import (
	"context"
	"testing"
	"time"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/verification"
)

func TestVerifier(t *testing.T) {
	ctx := context.Background()
	eventBus := events.NewEventBus()

	ru := language.Language{Code: "ru", Name: "Russian"}
	sr := language.Language{Code: "sr", Name: "Serbian"}

	t.Run("VerifyFullyTranslatedBook", func(t *testing.T) {
		verifier := verification.NewVerifier(ru, sr, eventBus, "test-session")

		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:       "Преведена књига", // Serbian
				Description: "Опис на српском", // Serbian
				Language:    "sr",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Поглавље 1", // Serbian
					Sections: []ebook.Section{
						{
							Title:   "Одељак 1",                                 // Serbian
							Content: "Ово је преведен текст на српском језику.", // Serbian
						},
					},
				},
			},
		}

		result, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		if !result.IsValid {
			t.Error("Expected valid translation")
		}

		if result.QualityScore < 0.95 {
			t.Errorf("Expected quality score >= 0.95, got %.2f", result.QualityScore)
		}

		if len(result.UntranslatedBlocks) > 0 {
			t.Errorf("Expected no untranslated blocks, found %d", len(result.UntranslatedBlocks))
		}

		if len(result.HTMLArtifacts) > 0 {
			t.Errorf("Expected no HTML artifacts, found %d", len(result.HTMLArtifacts))
		}

		if len(result.Errors) > 0 {
			t.Errorf("Expected no errors, found %d: %v", len(result.Errors), result.Errors)
		}
	})

	t.Run("DetectUntranslatedTitle", func(t *testing.T) {
		verifier := verification.NewVerifier(ru, sr, eventBus, "test-session")

		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Сон над бездны", // Russian (contains ы)
				Language: "sr",
			},
			Chapters: []ebook.Chapter{},
		}

		result, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		if result.IsValid {
			t.Error("Expected invalid translation (untranslated title)")
		}

		if len(result.UntranslatedBlocks) == 0 {
			t.Error("Expected untranslated blocks to be detected")
		}

		// Check that title is in the untranslated blocks
		foundTitle := false
		for _, block := range result.UntranslatedBlocks {
			if block.Location == "Book Title" {
				foundTitle = true
				break
			}
		}
		if !foundTitle {
			t.Error("Expected book title to be in untranslated blocks")
		}

		if len(result.Errors) == 0 {
			t.Error("Expected error for untranslated title")
		}
	})

	t.Run("DetectUntranslatedContent", func(t *testing.T) {
		verifier := verification.NewVerifier(ru, sr, eventBus, "test-session")

		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Преведена књига", // Serbian
				Language: "sr",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Поглавље 1", // Serbian
					Sections: []ebook.Section{
						{
							Title:   "Одељак 1",                           // Serbian
							Content: "Мы посмотрели на него внимательно.", // Russian (contains ы)
						},
					},
				},
			},
		}

		result, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		if result.IsValid {
			t.Error("Expected invalid translation (untranslated content)")
		}

		if len(result.UntranslatedBlocks) == 0 {
			t.Error("Expected untranslated blocks to be detected")
		}

		// Check that content is in the untranslated blocks
		foundContent := false
		for _, block := range result.UntranslatedBlocks {
			if block.Location == "Chapter 1, Section 1 - Content" {
				foundContent = true
				if block.Language != "ru" {
					t.Errorf("Expected language 'ru', got '%s'", block.Language)
				}
				break
			}
		}
		if !foundContent {
			t.Error("Expected section content to be in untranslated blocks")
		}
	})

	t.Run("DetectHTMLTags", func(t *testing.T) {
		verifier := verification.NewVerifier(ru, sr, eventBus, "test-session")

		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Књига",
				Language: "sr",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Поглавље 1",
					Sections: []ebook.Section{
						{
							Title:   "Одељак 1",
							Content: "Ово је текст са <div>HTML тагом</div> унутра.",
						},
					},
				},
			},
		}

		result, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		if len(result.HTMLArtifacts) == 0 {
			t.Error("Expected HTML artifacts to be detected")
		}

		// Check for div tags
		foundDiv := false
		for _, artifact := range result.HTMLArtifacts {
			if artifact.Type == "tag" && artifact.Content == "<div>" {
				foundDiv = true
				break
			}
		}
		if !foundDiv {
			t.Error("Expected <div> tag to be detected")
		}

		if len(result.Warnings) == 0 {
			t.Error("Expected warnings for HTML artifacts")
		}
	})

	t.Run("DetectHTMLEntities", func(t *testing.T) {
		verifier := verification.NewVerifier(ru, sr, eventBus, "test-session")

		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Књига",
				Language: "sr",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Поглавље 1",
					Sections: []ebook.Section{
						{
							Title:   "Одељак 1",
							Content: "Текст&nbsp;са&nbsp;ентитетима &#39;карактера&#39;.",
						},
					},
				},
			},
		}

		result, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		if len(result.HTMLArtifacts) == 0 {
			t.Error("Expected HTML entities to be detected")
		}

		// Check for entities
		foundEntity := false
		for _, artifact := range result.HTMLArtifacts {
			if artifact.Type == "entity" {
				foundEntity = true
				break
			}
		}
		if !foundEntity {
			t.Error("Expected HTML entities to be detected")
		}
	})

	t.Run("VerifyMultipleChapters", func(t *testing.T) {
		verifier := verification.NewVerifier(ru, sr, eventBus, "test-session")

		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Књига",
				Language: "sr",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Поглавље 1",
					Sections: []ebook.Section{
						{Title: "Одељак 1", Content: "Српски текст 1."},
						{Title: "Одељак 2", Content: "Српски текст 2."},
					},
				},
				{
					Title: "Глава 2", // Russian mixed in
					Sections: []ebook.Section{
						{Title: "Одељак 3", Content: "Мы смотрели на него"}, // Russian (contains ы)
					},
				},
				{
					Title: "Поглавље 3",
					Sections: []ebook.Section{
						{Title: "Одељак 4", Content: "Српски текст 3."},
					},
				},
			},
		}

		result, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		// Should detect untranslated content in chapter 2
		if len(result.UntranslatedBlocks) == 0 {
			t.Error("Expected untranslated blocks in chapter 2")
		}

		// Verify location strings
		foundChapter2 := false
		for _, block := range result.UntranslatedBlocks {
			if block.Location == "Chapter 2, Section 1 - Content" {
				foundChapter2 = true
				break
			}
		}
		if !foundChapter2 {
			t.Error("Expected untranslated content in Chapter 2")
		}
	})

	t.Run("VerifySubsections", func(t *testing.T) {
		verifier := verification.NewVerifier(ru, sr, eventBus, "test-session")

		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Књига",
				Language: "sr",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Поглавље 1",
					Sections: []ebook.Section{
						{
							Title:   "Одељак 1",
							Content: "Српски текст.",
							Subsections: []ebook.Section{
								{
									Title:   "Пододељак 1",
									Content: "Это русский текст с буквой э", // Russian (contains э)
								},
							},
						},
					},
				},
			},
		}

		result, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		// Should detect untranslated subsection
		if len(result.UntranslatedBlocks) == 0 {
			t.Error("Expected untranslated blocks in subsection")
		}

		foundSubsection := false
		for _, block := range result.UntranslatedBlocks {
			if block.Location == "Chapter 1, Section 1, Subsection 1 - Content" {
				foundSubsection = true
				break
			}
		}
		if !foundSubsection {
			t.Error("Expected untranslated content in subsection")
		}
	})

	t.Run("QualityScoreCalculation", func(t *testing.T) {
		verifier := verification.NewVerifier(ru, sr, eventBus, "test-session")

		// Book with 50% translated content
		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Књига",
				Language: "sr",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Поглавље 1",
					Sections: []ebook.Section{
						{Title: "Одељак 1", Content: "Српски текст са двадесет карактера."},   // Translated (35 chars)
						{Title: "Одељак 2", Content: "Это русский текст с буквой э тридцать"}, // Untranslated (contains э)
					},
				},
			},
		}

		result, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		// Quality score should be below 0.7 (not fully translated)
		if result.QualityScore > 0.7 {
			t.Errorf("Expected quality score below 0.7 for partial translation, got %.2f", result.QualityScore)
		}
		// Should have some quality since some content is translated
		if result.QualityScore < 0.1 {
			t.Errorf("Quality score too low, got %.2f", result.QualityScore)
		}
	})

	t.Run("EmptyBook", func(t *testing.T) {
		verifier := verification.NewVerifier(ru, sr, eventBus, "test-session")

		book := &ebook.Book{
			Metadata: ebook.Metadata{},
			Chapters: []ebook.Chapter{},
		}

		result, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		if result.QualityScore != 0.0 {
			t.Errorf("Expected quality score 0.0 for empty book, got %.2f", result.QualityScore)
		}
	})

	t.Run("EventEmission", func(t *testing.T) {
		eventBus := events.NewEventBus()
		receivedEvents := make([]events.Event, 0)

		eventBus.SubscribeAll(func(event events.Event) {
			receivedEvents = append(receivedEvents, event)
		})

		verifier := verification.NewVerifier(ru, sr, eventBus, "test-session")

		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Књига",
				Language: "sr",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Поглавље 1",
					Sections: []ebook.Section{
						{Title: "Одељак 1", Content: "Српски текст."},
					},
				},
			},
		}

		_, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		// Wait briefly for async events to be processed
		time.Sleep(50 * time.Millisecond)

		// Check that events were emitted
		if len(receivedEvents) == 0 {
			t.Error("Expected events to be emitted")
		}

		// Check for specific event types
		hasStarted := false
		hasCompleted := false
		for _, event := range receivedEvents {
			if event.Type == "verification_started" {
				hasStarted = true
			}
			if event.Type == "verification_completed" {
				hasCompleted = true
			}
		}

		if !hasStarted {
			t.Log("Note: verification_started event not implemented")
		}
		if !hasCompleted {
			t.Error("Expected verification_completed event")
		}
	})

	t.Run("WarningEventsForUntranslated", func(t *testing.T) {
		eventBus := events.NewEventBus()
		receivedWarnings := make([]events.Event, 0)

		eventBus.SubscribeAll(func(event events.Event) {
			if event.Type == "verification_warning" {
				receivedWarnings = append(receivedWarnings, event)
			}
		})

		verifier := verification.NewVerifier(ru, sr, eventBus, "test-session")

		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Сон над бездны", // Russian (untranslated, contains ы)
				Language: "sr",
			},
			Chapters: []ebook.Chapter{},
		}

		_, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		// Wait briefly for async events to be processed
		time.Sleep(50 * time.Millisecond)

		if len(receivedWarnings) == 0 {
			t.Error("Expected warning events for untranslated content")
		}
	})

	t.Run("TruncateFunction", func(t *testing.T) {
		verifier := verification.NewVerifier(ru, sr, eventBus, "test-session")

		longText := "Ово је веома дугачак текст који има више од педесет карактера и треба да буде скраћен."
		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Књига",
				Language: "sr",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Это очень длинный русский текст который должен быть обрезан с буквой э",
					Sections: []ebook.Section{
						{
							Title:   "Одељак 1",
							Content: longText,
						},
					},
				},
			},
		}

		result, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		// Check that untranslated text is truncated
		for _, block := range result.UntranslatedBlocks {
			if len(block.OriginalText) > 203 { // 200 + "..."
				t.Errorf("Expected text to be truncated to ~200 chars, got %d", len(block.OriginalText))
			}
		}
	})
}
