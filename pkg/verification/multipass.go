package verification

import (
	"context"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// MultiPassConfig configures multi-pass polishing
type MultiPassConfig struct {
	// Number of polishing passes
	PassCount int

	// Provider sets for each pass (different LLMs per pass)
	PassProviders [][]string // e.g., [["deepseek", "anthropic"], ["openai", "claude"]]

	// Minimum consensus per pass
	MinConsensus int

	// Verification dimensions
	VerifySpirit     bool
	VerifyLanguage   bool
	VerifyContext    bool
	VerifyVocabulary bool

	// Note-taking configuration
	EnableNoteTaking  bool
	MinNoteImportance ImportanceLevel
	CarryNotesForward bool // Carry notes to next pass

	// Database path for persistence
	DatabasePath string

	// Translation configurations for all providers
	TranslationConfigs map[string]translator.TranslationConfig
}

// MultiPassPolisher orchestrates multi-pass polishing
type MultiPassPolisher struct {
	config    MultiPassConfig
	database  *PolishingDatabase
	eventBus  *events.EventBus
	sessionID string

	// ensembleFactory, when non-nil, is the injectable seam (R-1c) that supplies
	// each pass's BookPolisher ensemble translators from an external source (e.g.
	// the provider-diverse LLMsVerifier bridge) instead of the built-in
	// per-provider NewLLMTranslator construction. nil = built-in construction
	// (default, behaviour-preserving). It is set via SetEnsembleFactory by the
	// binary main; the field is a plain function value, so this package stays
	// decoupled from the bridge / internal/verifier packages (§11.4.28).
	ensembleFactory EnsembleTranslatorFactory
}

// SetEnsembleFactory injects the optional provider-diverse ensemble factory used
// to construct each pass's BookPolisher. Passing nil restores the default
// built-in construction (behaviour-preserving). This is the additive plumbing
// (R-1c) that lets the binary main redirect multi-pass polishing to the bridge.
func (mpp *MultiPassPolisher) SetEnsembleFactory(factory EnsembleTranslatorFactory) {
	mpp.ensembleFactory = factory
}

// PassResult contains results from a single pass
type PassResult struct {
	PassNumber  int
	PassID      string
	Providers   []string
	Notes       []*LiteraryNote
	Results     []*PolishingResult
	Report      *PolishingReport
	Duration    time.Duration
	StartedAt   time.Time
	CompletedAt time.Time
}

// MultiPassResult contains results from all passes
type MultiPassResult struct {
	SessionID    string
	BookID       string
	BookTitle    string
	TotalPasses  int
	PassResults  []*PassResult
	FinalBook    *ebook.Book
	FinalReport  *PolishingReport
	AllNotes     *NoteCollection
	TotalChanges int
	StartedAt    time.Time
	CompletedAt  time.Time
	Duration     time.Duration
}

// NewMultiPassPolisher creates a new multi-pass polisher
func NewMultiPassPolisher(
	config MultiPassConfig,
	eventBus *events.EventBus,
	sessionID string,
) (*MultiPassPolisher, error) {
	// Open database
	var database *PolishingDatabase
	if config.DatabasePath != "" {
		db, err := NewPolishingDatabase(config.DatabasePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}
		database = db
	}

	return &MultiPassPolisher{
		config:    config,
		database:  database,
		eventBus:  eventBus,
		sessionID: sessionID,
	}, nil
}

// PolishBook performs multi-pass polishing on a book
func (mpp *MultiPassPolisher) PolishBook(
	ctx context.Context,
	originalBook *ebook.Book,
	translatedBook *ebook.Book,
) (*MultiPassResult, error) {
	startTime := time.Now()

	result := &MultiPassResult{
		SessionID:   mpp.sessionID,
		BookID:      fmt.Sprintf("%s_%d", originalBook.Metadata.Title, time.Now().Unix()),
		BookTitle:   originalBook.Metadata.Title,
		TotalPasses: mpp.config.PassCount,
		PassResults: make([]*PassResult, 0),
		AllNotes:    NewNoteCollection(),
		StartedAt:   startTime,
	}

	// Create session in database
	if mpp.database != nil {
		configJSON, _ := json.Marshal(mpp.config)
		session := &PolishingSession{
			SessionID:  mpp.sessionID,
			BookID:     result.BookID,
			BookTitle:  result.BookTitle,
			StartedAt:  startTime,
			ConfigJSON: string(configJSON),
			Status:     "running",
		}
		if err := mpp.database.CreateSession(session); err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}

	mpp.emitProgress("Starting multi-pass polishing", map[string]interface{}{
		"total_passes": mpp.config.PassCount,
		"book_title":   originalBook.Metadata.Title,
	})

	// Current book state (updated after each pass)
	currentBook := translatedBook

	// Perform each pass
	for passNum := 1; passNum <= mpp.config.PassCount; passNum++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		mpp.emitProgress(fmt.Sprintf("Starting Pass %d/%d", passNum, mpp.config.PassCount), map[string]interface{}{
			"pass_number":  passNum,
			"total_passes": mpp.config.PassCount,
		})

		// Get providers for this pass
		providers := mpp.getProvidersForPass(passNum)

		// Get previous notes
		var previousNotes []*LiteraryNote
		if mpp.config.CarryNotesForward && passNum > 1 {
			previousNotes = result.AllNotes.All
		}

		// Perform pass
		passResult, polishedBook, err := mpp.performPass(
			ctx,
			passNum,
			providers,
			originalBook,
			currentBook,
			previousNotes,
		)

		if err != nil {
			return result, fmt.Errorf("pass %d failed: %w", passNum, err)
		}

		// Update current book for next pass
		currentBook = polishedBook

		// Add pass results
		result.PassResults = append(result.PassResults, passResult)

		// Collect notes
		for _, note := range passResult.Notes {
			result.AllNotes.Add(note)
		}

		// Count changes
		for _, res := range passResult.Results {
			result.TotalChanges += len(res.Changes)
		}

		mpp.emitProgress(fmt.Sprintf("Completed Pass %d/%d", passNum, mpp.config.PassCount), map[string]interface{}{
			"pass_number":  passNum,
			"notes_added":  len(passResult.Notes),
			"changes_made": len(passResult.Results),
		})
	}

	// Final book is the result of last pass
	result.FinalBook = currentBook

	// Generate final report combining all passes
	result.FinalReport = mpp.generateFinalReport(result)

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(startTime)

	// Update session in database
	if mpp.database != nil {
		mpp.database.UpdateSession(mpp.sessionID, "completed", result.CompletedAt, mpp.config.PassCount)
	}

	mpp.emitProgress("Multi-pass polishing completed", map[string]interface{}{
		"total_passes":  mpp.config.PassCount,
		"total_changes": result.TotalChanges,
		"total_notes":   len(result.AllNotes.All),
		"final_score":   result.FinalReport.OverallScore,
		"duration":      result.Duration.String(),
	})

	return result, nil
}

// performPass performs a single polishing pass
func (mpp *MultiPassPolisher) performPass(
	ctx context.Context,
	passNumber int,
	providers []string,
	originalBook *ebook.Book,
	currentBook *ebook.Book,
	previousNotes []*LiteraryNote,
) (*PassResult, *ebook.Book, error) {
	startTime := time.Now()

	passID := fmt.Sprintf("%s_pass_%d", mpp.sessionID, passNumber)

	// Create pass record in database
	if mpp.database != nil {
		providersJSON, _ := json.Marshal(providers)
		passRecord := &PassRecord{
			PassID:     passID,
			SessionID:  mpp.sessionID,
			PassNumber: passNumber,
			Providers:  string(providersJSON),
			StartedAt:  startTime,
			Status:     "running",
		}
		if err := mpp.database.CreatePass(passRecord); err != nil {
			return nil, nil, fmt.Errorf("failed to create pass record: %w", err)
		}
	}

	passResult := &PassResult{
		PassNumber: passNumber,
		PassID:     passID,
		Providers:  providers,
		Notes:      make([]*LiteraryNote, 0),
		Results:    make([]*PolishingResult, 0),
		StartedAt:  startTime,
	}

	// Create polishing config for this pass
	polishingConfig := PolishingConfig{
		Providers:          providers,
		MinConsensus:       mpp.config.MinConsensus,
		VerifySpirit:       mpp.config.VerifySpirit,
		VerifyLanguage:     mpp.config.VerifyLanguage,
		VerifyContext:      mpp.config.VerifyContext,
		VerifyVocabulary:   mpp.config.VerifyVocabulary,
		TranslationConfigs: make(map[string]translator.TranslationConfig),
	}

	// Add configs for providers
	for _, provider := range providers {
		if config, ok := mpp.config.TranslationConfigs[provider]; ok {
			polishingConfig.TranslationConfigs[provider] = config
		}
	}

	// Create polisher. When an ensemble factory has been injected (R-1c), source
	// the pass's ensemble translators from it via NewBookPolisherWithFactory using
	// the ctx already in scope for this pass; otherwise the nil branch runs the
	// EXACT built-in per-provider construction (NewBookPolisher itself delegates to
	// the factory variant with a background ctx + nil factory, so the default path
	// is unchanged).
	polisher, err := NewBookPolisherWithFactory(ctx, polishingConfig, mpp.eventBus, fmt.Sprintf("%s_pass%d", mpp.sessionID, passNumber), mpp.ensembleFactory)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create polisher: %w", err)
	}

	// Polish with note-taking
	polishedBook, report, notes, err := mpp.polishWithNotes(
		ctx,
		polisher,
		passID,
		passNumber,
		originalBook,
		currentBook,
		previousNotes,
	)

	if err != nil {
		return nil, nil, err
	}

	passResult.Notes = notes
	passResult.Report = report
	passResult.Results = report.SectionResults

	passResult.CompletedAt = time.Now()
	passResult.Duration = passResult.CompletedAt.Sub(startTime)

	// Update pass in database
	if mpp.database != nil {
		mpp.database.UpdatePass(passID, "completed", passResult.CompletedAt)
	}

	return passResult, polishedBook, nil
}

// polishWithNotes performs polishing with integrated note-taking
// saveNewResults persists only the PolishingResults appended to report since the
// previous call (identified by the savedResults high-water mark) and returns the
// updated mark. report.SectionResults accumulates across chapters, so saving the
// whole slice each chapter re-saves earlier chapters' results — producing
// duplicate polishing_changes rows (that table has no unique key) and
// PK-conflicting (silently-failing) section_results inserts.
func (mpp *MultiPassPolisher) saveNewResults(report *PolishingReport, savedResults int, passID string) int {
	if report == nil {
		return savedResults
	}
	total := len(report.SectionResults)
	if mpp.database == nil {
		return total
	}
	if savedResults < 0 {
		savedResults = 0
	}
	if savedResults > total {
		savedResults = total
	}
	for _, result := range report.SectionResults[savedResults:] {
		mpp.database.SaveResult(result, passID)
		mpp.database.SaveChanges(result.Changes, passID, result.SectionID)
	}
	return total
}

func (mpp *MultiPassPolisher) polishWithNotes(
	ctx context.Context,
	polisher *BookPolisher,
	passID string,
	passNumber int,
	originalBook *ebook.Book,
	currentBook *ebook.Book,
	previousNotes []*LiteraryNote,
) (*ebook.Book, *PolishingReport, []*LiteraryNote, error) {
	polishedBook := currentBook
	report := NewPolishingReport(polisher.config)
	allNotes := make([]*LiteraryNote, 0)

	// Polish metadata with notes
	if mpp.config.EnableNoteTaking {
		metadataNotes := mpp.generateMetadataNotes(
			ctx,
			passNumber,
			polisher,
			&originalBook.Metadata,
			&currentBook.Metadata,
			previousNotes,
		)
		allNotes = append(allNotes, metadataNotes...)

		// Save notes to database
		if mpp.database != nil {
			for _, note := range metadataNotes {
				mpp.database.SaveNote(note, passID)
			}
		}
	}

	// Polish metadata
	if err := polisher.polishMetadata(ctx, &originalBook.Metadata, &polishedBook.Metadata, report); err != nil {
		return nil, nil, nil, err
	}

	// Polish chapters with notes
	// Bound the loop by the SHORTER of original/current/polished chapter counts.
	// The loop indexes currentBook.Chapters[i] and polishedBook.Chapters[i] using
	// an index ranged over originalBook.Chapters; if the translated/polished book
	// has FEWER chapters than the original (a legitimate outcome — e.g. a chapter
	// translated to empty/merged), indexing the shorter slice panics (index out of
	// range). Capping at the minimum length processes every chapter that exists in
	// all three and safely skips the missing tail (logged) instead of crashing.
	totalChapters := len(originalBook.Chapters)
	maxChapters := totalChapters
	if l := len(currentBook.Chapters); l < maxChapters {
		maxChapters = l
	}
	if l := len(polishedBook.Chapters); l < maxChapters {
		maxChapters = l
	}
	if maxChapters < totalChapters {
		log.Printf("multipass: translated book has %d chapters vs %d in the original; "+
			"polishing the first %d and skipping the missing tail", maxChapters, totalChapters, maxChapters)
	}
	// High-water mark into report.SectionResults (the polisher appends to it as
	// each chapter is polished); we persist only the newly-appended results each
	// iteration via saveNewResults.
	savedResults := 0
	for i := 0; i < maxChapters; i++ {
		select {
		case <-ctx.Done():
			return nil, nil, nil, ctx.Err()
		default:
		}

		location := fmt.Sprintf("Chapter %d/%d", i+1, totalChapters)

		// Generate notes for chapter if enabled
		if mpp.config.EnableNoteTaking {
			chapterNotes := mpp.generateChapterNotes(
				ctx,
				passNumber,
				polisher,
				&originalBook.Chapters[i],
				&currentBook.Chapters[i],
				location,
				previousNotes,
			)
			allNotes = append(allNotes, chapterNotes...)

			// Save notes to database
			if mpp.database != nil {
				for _, note := range chapterNotes {
					mpp.database.SaveNote(note, passID)
				}
			}
		}

		// Polish chapter
		if err := polisher.polishChapter(
			ctx,
			&originalBook.Chapters[i],
			&polishedBook.Chapters[i],
			i+1,
			report,
		); err != nil {
			return nil, nil, nil, err
		}

		// Save only the results newly appended by this chapter's polish.
		savedResults = mpp.saveNewResults(report, savedResults, passID)
	}

	report.Finalize()

	return polishedBook, report, allNotes, nil
}

// generateMetadataNotes generates notes for metadata
func (mpp *MultiPassPolisher) generateMetadataNotes(
	ctx context.Context,
	passNumber int,
	polisher *BookPolisher,
	originalMetadata *ebook.Metadata,
	currentMetadata *ebook.Metadata,
	previousNotes []*LiteraryNote,
) []*LiteraryNote {
	notes := make([]*LiteraryNote, 0)

	// Generate notes from first provider
	if len(polisher.config.Providers) > 0 {
		provider := polisher.config.Providers[0]
		translator := polisher.translators[provider]

		noteTaker := NewNoteTaker(translator, provider)

		// Title notes
		if originalMetadata.Title != "" {
			titleNotes, _ := noteTaker.GenerateNotes(
				ctx,
				passNumber,
				"metadata_title",
				"Book Title",
				originalMetadata.Title,
				currentMetadata.Title,
				filterNotesBySection(previousNotes, "metadata_title"),
			)
			notes = append(notes, titleNotes...)
		}
	}

	return notes
}

// generateChapterNotes generates notes for a chapter
func (mpp *MultiPassPolisher) generateChapterNotes(
	ctx context.Context,
	passNumber int,
	polisher *BookPolisher,
	originalChapter *ebook.Chapter,
	currentChapter *ebook.Chapter,
	location string,
	previousNotes []*LiteraryNote,
) []*LiteraryNote {
	notes := make([]*LiteraryNote, 0)

	// Generate notes from all providers in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, provider := range polisher.config.Providers {
		wg.Add(1)
		go func(prov string) {
			defer wg.Done()

			translator := polisher.translators[prov]
			noteTaker := NewNoteTaker(translator, prov)

			sectionID := fmt.Sprintf("chapter_%s", strings.ReplaceAll(location, " ", "_"))

			// Generate notes for chapter content
			chapterText := extractChapterText(originalChapter)
			currentText := extractChapterText(currentChapter)

			if len(chapterText) > 100 { // Only generate notes for substantial text
				providerNotes, _ := noteTaker.GenerateNotes(
					ctx,
					passNumber,
					sectionID,
					location,
					chapterText,
					currentText,
					filterNotesBySection(previousNotes, sectionID),
				)

				mu.Lock()
				notes = append(notes, providerNotes...)
				mu.Unlock()
			}
		}(provider)
	}

	wg.Wait()

	// Filter by minimum importance
	if mpp.config.MinNoteImportance != "" {
		notes = FilterNotesByImportance(notes, mpp.config.MinNoteImportance)
	}

	return notes
}

// Helper functions

func (mpp *MultiPassPolisher) getProvidersForPass(passNumber int) []string {
	if passNumber <= len(mpp.config.PassProviders) {
		return mpp.config.PassProviders[passNumber-1]
	}

	// Default: use all providers
	providers := make([]string, 0)
	for provider := range mpp.config.TranslationConfigs {
		providers = append(providers, provider)
	}
	return providers
}

func (mpp *MultiPassPolisher) generateFinalReport(result *MultiPassResult) *PolishingReport {
	// Combine reports from all passes
	if len(result.PassResults) == 0 {
		return nil
	}

	// Use last pass report as base
	finalReport := result.PassResults[len(result.PassResults)-1].Report

	// Sum every pass's section count into a LOCAL first. finalReport aliases the
	// last pass's Report pointer, so zeroing finalReport.TotalSections and then
	// summing in-place would (a) zero the last pass's own contribution and (b)
	// re-read finalReport's running total for the last pass — under-counting the
	// combined total. Accumulate separately, assign once.
	totalSections := 0
	for _, passResult := range result.PassResults {
		if passResult.Report != nil {
			totalSections += passResult.Report.TotalSections
		}
	}

	finalReport.TotalSections = totalSections
	finalReport.TotalChanges = result.TotalChanges

	return finalReport
}

func (mpp *MultiPassPolisher) emitProgress(message string, data map[string]interface{}) {
	if mpp.eventBus != nil {
		event := events.NewEvent("multipass_progress", message, data)
		event.SessionID = mpp.sessionID
		mpp.eventBus.Publish(event)
	}
}

func filterNotesBySection(notes []*LiteraryNote, sectionID string) []*LiteraryNote {
	filtered := make([]*LiteraryNote, 0)
	for _, note := range notes {
		if note.SectionID == sectionID {
			filtered = append(filtered, note)
		}
	}
	return filtered
}

func extractChapterText(chapter *ebook.Chapter) string {
	var sb strings.Builder

	if chapter.Title != "" {
		sb.WriteString(chapter.Title)
		sb.WriteString("\n\n")
	}

	for i := range chapter.Sections {
		sb.WriteString(extractSectionText(&chapter.Sections[i]))
	}

	return sb.String()
}

func extractSectionText(section *ebook.Section) string {
	var sb strings.Builder

	if section.Title != "" {
		sb.WriteString(section.Title)
		sb.WriteString("\n")
	}

	if section.Content != "" {
		sb.WriteString(section.Content)
		sb.WriteString("\n")
	}

	for i := range section.Subsections {
		sb.WriteString(extractSectionText(&section.Subsections[i]))
	}

	return sb.String()
}

// Close closes the multi-pass polisher and database
func (mpp *MultiPassPolisher) Close() error {
	if mpp.database != nil {
		return mpp.database.Close()
	}
	return nil
}
