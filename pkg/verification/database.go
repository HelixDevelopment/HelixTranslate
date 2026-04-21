package verification

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// PolishingDatabase manages persistent storage for multi-pass polishing
type PolishingDatabase struct {
	db *sql.DB
}

// PolishingSession represents a complete polishing session
type PolishingSession struct {
	SessionID   string    `json:"session_id"`
	BookID      string    `json:"book_id"`
	BookTitle   string    `json:"book_title"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	ConfigJSON  string    `json:"config_json"`
	TotalPasses int       `json:"total_passes"`
	Status      string    `json:"status"` // running, completed, failed
}

// PassRecord represents a single polishing pass
type PassRecord struct {
	PassID      string    `json:"pass_id"`
	SessionID   string    `json:"session_id"`
	PassNumber  int       `json:"pass_number"`
	Providers   string    `json:"providers"` // JSON array
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Status      string    `json:"status"`
}

// NewPolishingDatabase creates a new polishing database
func NewPolishingDatabase(dbPath string) (*PolishingDatabase, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	pdb := &PolishingDatabase{db: db}

	// Initialize schema
	if err := pdb.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return pdb, nil
}

// initSchema creates database tables
func (pdb *PolishingDatabase) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS polishing_sessions (
		session_id TEXT PRIMARY KEY,
		book_id TEXT NOT NULL,
		book_title TEXT,
		started_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP,
		config_json TEXT,
		total_passes INTEGER DEFAULT 0,
		status TEXT DEFAULT 'running'
	);

	CREATE TABLE IF NOT EXISTS polishing_passes (
		pass_id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		pass_number INTEGER NOT NULL,
		providers TEXT,
		started_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP,
		status TEXT DEFAULT 'running',
		FOREIGN KEY (session_id) REFERENCES polishing_sessions(session_id)
	);

	CREATE TABLE IF NOT EXISTS section_notes (
		note_id TEXT PRIMARY KEY,
		pass_id TEXT NOT NULL,
		section_id TEXT NOT NULL,
		location TEXT,
		provider TEXT,
		note_type TEXT,
		importance TEXT,
		title TEXT,
		content TEXT,
		examples TEXT,
		implications TEXT,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY (pass_id) REFERENCES polishing_passes(pass_id)
	);

	CREATE TABLE IF NOT EXISTS section_results (
		result_id TEXT PRIMARY KEY,
		pass_id TEXT NOT NULL,
		section_id TEXT NOT NULL,
		location TEXT,
		original_text TEXT,
		translated_text TEXT,
		polished_text TEXT,
		spirit_score REAL,
		language_score REAL,
		context_score REAL,
		vocabulary_score REAL,
		overall_score REAL,
		consensus INTEGER,
		confidence REAL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY (pass_id) REFERENCES polishing_passes(pass_id)
	);

	CREATE TABLE IF NOT EXISTS polishing_changes (
		change_id INTEGER PRIMARY KEY AUTOINCREMENT,
		pass_id TEXT NOT NULL,
		section_id TEXT NOT NULL,
		location TEXT,
		change_type TEXT,
		original TEXT,
		polished TEXT,
		reason TEXT,
		agreement INTEGER,
		confidence REAL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY (pass_id) REFERENCES polishing_passes(pass_id)
	);

	CREATE INDEX IF NOT EXISTS idx_notes_pass ON section_notes(pass_id);
	CREATE INDEX IF NOT EXISTS idx_notes_section ON section_notes(section_id);
	CREATE INDEX IF NOT EXISTS idx_results_pass ON section_results(pass_id);
	CREATE INDEX IF NOT EXISTS idx_results_section ON section_results(section_id);
	CREATE INDEX IF NOT EXISTS idx_changes_pass ON polishing_changes(pass_id);
	`

	_, err := pdb.db.Exec(schema)
	return err
}

// CreateSession creates a new polishing session
func (pdb *PolishingDatabase) CreateSession(session *PolishingSession) error {
	query := `INSERT INTO polishing_sessions
	(session_id, book_id, book_title, started_at, config_json, status)
	VALUES (?, ?, ?, ?, ?, ?)`

	_, err := pdb.db.Exec(query,
		session.SessionID,
		session.BookID,
		session.BookTitle,
		session.StartedAt,
		session.ConfigJSON,
		session.Status,
	)

	return err
}

// UpdateSession updates session status
func (pdb *PolishingDatabase) UpdateSession(sessionID string, status string, completedAt time.Time, totalPasses int) error {
	query := `UPDATE polishing_sessions
	SET status = ?, completed_at = ?, total_passes = ?
	WHERE session_id = ?`

	_, err := pdb.db.Exec(query, status, completedAt, totalPasses, sessionID)
	return err
}

// GetSession retrieves a session
func (pdb *PolishingDatabase) GetSession(sessionID string) (*PolishingSession, error) {
	query := `SELECT session_id, book_id, book_title, started_at, completed_at,
	config_json, total_passes, status FROM polishing_sessions WHERE session_id = ?`

	session := &PolishingSession{}
	var completedAt sql.NullTime

	err := pdb.db.QueryRow(query, sessionID).Scan(
		&session.SessionID,
		&session.BookID,
		&session.BookTitle,
		&session.StartedAt,
		&completedAt,
		&session.ConfigJSON,
		&session.TotalPasses,
		&session.Status,
	)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		session.CompletedAt = completedAt.Time
	}

	return session, nil
}

// CreatePass creates a new pass record
func (pdb *PolishingDatabase) CreatePass(pass *PassRecord) error {
	query := `INSERT INTO polishing_passes
	(pass_id, session_id, pass_number, providers, started_at, status)
	VALUES (?, ?, ?, ?, ?, ?)`

	_, err := pdb.db.Exec(query,
		pass.PassID,
		pass.SessionID,
		pass.PassNumber,
		pass.Providers,
		pass.StartedAt,
		pass.Status,
	)

	return err
}

// UpdatePass updates pass status
func (pdb *PolishingDatabase) UpdatePass(passID string, status string, completedAt time.Time) error {
	query := `UPDATE polishing_passes
	SET status = ?, completed_at = ?
	WHERE pass_id = ?`

	_, err := pdb.db.Exec(query, status, completedAt, passID)
	return err
}

// GetPass retrieves a pass record
func (pdb *PolishingDatabase) GetPass(passID string) (*PassRecord, error) {
	query := `SELECT pass_id, session_id, pass_number, providers, started_at, completed_at, status
	FROM polishing_passes WHERE pass_id = ?`

	pass := &PassRecord{}
	var completedAt sql.NullTime

	err := pdb.db.QueryRow(query, passID).Scan(
		&pass.PassID,
		&pass.SessionID,
		&pass.PassNumber,
		&pass.Providers,
		&pass.StartedAt,
		&completedAt,
		&pass.Status,
	)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		pass.CompletedAt = completedAt.Time
	}

	return pass, nil
}

// SaveNote saves a literary note
func (pdb *PolishingDatabase) SaveNote(note *LiteraryNote, passID string) error {
	examplesJSON, _ := json.Marshal(note.Examples)

	query := `INSERT INTO section_notes
	(note_id, pass_id, section_id, location, provider, note_type, importance,
	title, content, examples, implications, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := pdb.db.Exec(query,
		note.ID,
		passID,
		note.SectionID,
		note.Location,
		note.Provider,
		string(note.NoteType),
		string(note.Importance),
		note.Title,
		note.Content,
		string(examplesJSON),
		note.Implications,
		note.CreatedAt,
	)

	return err
}

// GetNotesForSection retrieves all notes for a section
func (pdb *PolishingDatabase) GetNotesForSection(sectionID string) ([]*LiteraryNote, error) {
	query := `SELECT note_id, section_id, location, provider, note_type, importance,
	title, content, examples, implications, created_at
	FROM section_notes WHERE section_id = ? ORDER BY created_at`

	rows, err := pdb.db.Query(query, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := make([]*LiteraryNote, 0)

	for rows.Next() {
		note := &LiteraryNote{}
		var examplesJSON string
		var noteType, importance string

		err := rows.Scan(
			&note.ID,
			&note.SectionID,
			&note.Location,
			&note.Provider,
			&noteType,
			&importance,
			&note.Title,
			&note.Content,
			&examplesJSON,
			&note.Implications,
			&note.CreatedAt,
		)

		if err != nil {
			continue
		}

		note.NoteType = NoteType(noteType)
		note.Importance = ImportanceLevel(importance)
		json.Unmarshal([]byte(examplesJSON), &note.Examples)

		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil
}

// GetNotesForPass retrieves all notes from a specific pass
func (pdb *PolishingDatabase) GetNotesForPass(passID string) ([]*LiteraryNote, error) {
	query := `SELECT note_id, section_id, location, provider, note_type, importance,
	title, content, examples, implications, created_at
	FROM section_notes WHERE pass_id = ? ORDER BY created_at`

	rows, err := pdb.db.Query(query, passID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := make([]*LiteraryNote, 0)

	for rows.Next() {
		note := &LiteraryNote{}
		var examplesJSON string
		var noteType, importance string

		err := rows.Scan(
			&note.ID,
			&note.SectionID,
			&note.Location,
			&note.Provider,
			&noteType,
			&importance,
			&note.Title,
			&note.Content,
			&examplesJSON,
			&note.Implications,
			&note.CreatedAt,
		)

		if err != nil {
			continue
		}

		note.NoteType = NoteType(noteType)
		note.Importance = ImportanceLevel(importance)
		json.Unmarshal([]byte(examplesJSON), &note.Examples)

		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil
}

// SaveResult saves a polishing result
func (pdb *PolishingDatabase) SaveResult(result *PolishingResult, passID string) error {
	query := `INSERT INTO section_results
	(result_id, pass_id, section_id, location, original_text, translated_text, polished_text,
	spirit_score, language_score, context_score, vocabulary_score, overall_score,
	consensus, confidence, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	resultID := fmt.Sprintf("%s_%s", passID, result.SectionID)

	_, err := pdb.db.Exec(query,
		resultID,
		passID,
		result.SectionID,
		result.Location,
		result.OriginalText,
		result.TranslatedText,
		result.PolishedText,
		result.SpiritScore,
		result.LanguageScore,
		result.ContextScore,
		result.VocabularyScore,
		result.OverallScore,
		result.Consensus,
		result.Confidence,
		time.Now(),
	)

	return err
}

// SaveChanges saves all changes from a result
func (pdb *PolishingDatabase) SaveChanges(changes []Change, passID string, sectionID string) error {
	if len(changes) == 0 {
		return nil
	}

	tx, err := pdb.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO polishing_changes
	(pass_id, section_id, location, change_type, original, polished, reason, agreement, confidence, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, change := range changes {
		_, err := stmt.Exec(
			passID,
			sectionID,
			change.Location,
			"improvement",
			change.Original,
			change.Polished,
			change.Reason,
			change.Agreement,
			change.Confidence,
			time.Now(),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetResultsForPass retrieves all results from a pass
func (pdb *PolishingDatabase) GetResultsForPass(passID string) ([]*PolishingResult, error) {
	query := `SELECT section_id, location, original_text, translated_text, polished_text,
	spirit_score, language_score, context_score, vocabulary_score, overall_score,
	consensus, confidence FROM section_results WHERE pass_id = ?`

	rows, err := pdb.db.Query(query, passID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*PolishingResult, 0)

	for rows.Next() {
		result := &PolishingResult{}

		err := rows.Scan(
			&result.SectionID,
			&result.Location,
			&result.OriginalText,
			&result.TranslatedText,
			&result.PolishedText,
			&result.SpiritScore,
			&result.LanguageScore,
			&result.ContextScore,
			&result.VocabularyScore,
			&result.OverallScore,
			&result.Consensus,
			&result.Confidence,
		)

		if err != nil {
			continue
		}

		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// GetSessionStats retrieves statistics for a session
func (pdb *PolishingDatabase) GetSessionStats(sessionID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total passes
	var totalPasses int
	err := pdb.db.QueryRow(
		"SELECT COUNT(*) FROM polishing_passes WHERE session_id = ?", sessionID,
	).Scan(&totalPasses)
	if err == nil {
		stats["total_passes"] = totalPasses
	}

	// Total notes
	var totalNotes int
	err = pdb.db.QueryRow(`
		SELECT COUNT(*) FROM section_notes
		WHERE pass_id IN (SELECT pass_id FROM polishing_passes WHERE session_id = ?)
	`, sessionID).Scan(&totalNotes)
	if err == nil {
		stats["total_notes"] = totalNotes
	}

	// Total changes
	var totalChanges int
	err = pdb.db.QueryRow(`
		SELECT COUNT(*) FROM polishing_changes
		WHERE pass_id IN (SELECT pass_id FROM polishing_passes WHERE session_id = ?)
	`, sessionID).Scan(&totalChanges)
	if err == nil {
		stats["total_changes"] = totalChanges
	}

	// Average scores
	var avgOverallScore float64
	err = pdb.db.QueryRow(`
		SELECT AVG(overall_score) FROM section_results
		WHERE pass_id IN (SELECT pass_id FROM polishing_passes WHERE session_id = ?)
	`, sessionID).Scan(&avgOverallScore)
	if err == nil {
		stats["avg_overall_score"] = avgOverallScore
	}

	return stats, nil
}

// Close closes the database
func (pdb *PolishingDatabase) Close() error {
	return pdb.db.Close()
}

// ExportSession exports all data for a session as JSON
func (pdb *PolishingDatabase) ExportSession(sessionID string) (map[string]interface{}, error) {
	export := make(map[string]interface{})

	// Get session
	session, err := pdb.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	export["session"] = session

	// Get all passes
	rows, err := pdb.db.Query(
		"SELECT pass_id FROM polishing_passes WHERE session_id = ? ORDER BY pass_number",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	passes := make([]map[string]interface{}, 0)

	for rows.Next() {
		var passID string
		rows.Scan(&passID)

		passData := make(map[string]interface{})

		// Get pass details
		pass, _ := pdb.GetPass(passID)
		passData["pass"] = pass

		// Get notes for this pass
		notes, _ := pdb.GetNotesForPass(passID)
		passData["notes"] = notes

		// Get results for this pass
		results, _ := pdb.GetResultsForPass(passID)
		passData["results"] = results

		passes = append(passes, passData)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	export["passes"] = passes
	export["stats"], _ = pdb.GetSessionStats(sessionID)

	return export, nil
}
