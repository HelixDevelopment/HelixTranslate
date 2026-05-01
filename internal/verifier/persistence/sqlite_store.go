// Package persistence provides SQLite-backed storage for verified model data.
package persistence

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"digital.vasic.translator/internal/verifier"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore persists verified models to a local SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) the SQLite database at dbPath.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite db: %w", err)
	}
	store := &SQLiteStore{db: db}
	if err := store.init(); err != nil {
		return nil, fmt.Errorf("failed to init sqlite schema: %w", err)
	}
	return store, nil
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS verified_models (
			id TEXT PRIMARY KEY,
			provider_id TEXT NOT NULL,
			name TEXT NOT NULL,
			verification_status TEXT NOT NULL,
			can_see_code INTEGER NOT NULL DEFAULT 0,
			affirmative_response INTEGER NOT NULL DEFAULT 0,
			overall_score REAL NOT NULL DEFAULT 0,
			responsiveness_score REAL NOT NULL DEFAULT 0,
			code_capability_score REAL NOT NULL DEFAULT 0,
			feature_richness_score REAL NOT NULL DEFAULT 0,
			reliability_score REAL NOT NULL DEFAULT 0,
			capabilities TEXT,
			pricing TEXT,
			last_verified_at TEXT
		)
	`)
	return err
}

// SaveModel upserts a verified model into the database.
func (s *SQLiteStore) SaveModel(m verifier.Model) error {
	capsJSON, err := json.Marshal(m.Capabilities)
	if err != nil {
		return fmt.Errorf("failed to marshal capabilities: %w", err)
	}
	pricingJSON, err := json.Marshal(m.Pricing)
	if err != nil {
		return fmt.Errorf("failed to marshal pricing: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO verified_models (
			id, provider_id, name, verification_status,
			can_see_code, affirmative_response,
			overall_score, responsiveness_score, code_capability_score,
			feature_richness_score, reliability_score,
			capabilities, pricing, last_verified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider_id = excluded.provider_id,
			name = excluded.name,
			verification_status = excluded.verification_status,
			can_see_code = excluded.can_see_code,
			affirmative_response = excluded.affirmative_response,
			overall_score = excluded.overall_score,
			responsiveness_score = excluded.responsiveness_score,
			code_capability_score = excluded.code_capability_score,
			feature_richness_score = excluded.feature_richness_score,
			reliability_score = excluded.reliability_score,
			capabilities = excluded.capabilities,
			pricing = excluded.pricing,
			last_verified_at = excluded.last_verified_at
	`, m.ID, m.ProviderID, m.Name, m.VerificationStatus,
		boolToInt(m.CanSeeCode), boolToInt(m.AffirmativeResponse),
		m.OverallScore, m.ResponsivenessScore, m.CodeCapabilityScore,
		m.FeatureRichnessScore, m.ReliabilityScore,
		string(capsJSON), string(pricingJSON), m.LastVerifiedAt.Format(time.RFC3339))

	if err != nil {
		return fmt.Errorf("failed to save model %s: %w", m.ID, err)
	}
	return nil
}

// GetModel retrieves a single model by ID.
func (s *SQLiteStore) GetModel(id string) (verifier.Model, error) {
	row := s.db.QueryRow(`
		SELECT id, provider_id, name, verification_status,
			can_see_code, affirmative_response,
			overall_score, responsiveness_score, code_capability_score,
			feature_richness_score, reliability_score,
			capabilities, pricing, last_verified_at
		FROM verified_models WHERE id = ?
	`, id)
	return s.scanModel(row)
}

// LoadModels returns all persisted models.
func (s *SQLiteStore) LoadModels() ([]verifier.Model, error) {
	rows, err := s.db.Query(`
		SELECT id, provider_id, name, verification_status,
			can_see_code, affirmative_response,
			overall_score, responsiveness_score, code_capability_score,
			feature_richness_score, reliability_score,
			capabilities, pricing, last_verified_at
		FROM verified_models
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query models: %w", err)
	}
	defer rows.Close()

	var models []verifier.Model
	for rows.Next() {
		m, err := s.scanModel(rows)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

// DeleteModel removes a model from the database.
func (s *SQLiteStore) DeleteModel(id string) error {
	_, err := s.db.Exec(`DELETE FROM verified_models WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete model %s: %w", id, err)
	}
	return nil
}

// Count returns the total number of stored models.
func (s *SQLiteStore) Count() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM verified_models`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count models: %w", err)
	}
	return count, nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (s *SQLiteStore) scanModel(row scanner) (verifier.Model, error) {
	var m verifier.Model
	var capsJSON, pricingJSON string
	var lastVerifiedStr string
	var canSeeCode, affirmativeResponse int

	err := row.Scan(
		&m.ID, &m.ProviderID, &m.Name, &m.VerificationStatus,
		&canSeeCode, &affirmativeResponse,
		&m.OverallScore, &m.ResponsivenessScore, &m.CodeCapabilityScore,
		&m.FeatureRichnessScore, &m.ReliabilityScore,
		&capsJSON, &pricingJSON, &lastVerifiedStr,
	)
	if err != nil {
		return verifier.Model{}, fmt.Errorf("failed to scan model: %w", err)
	}

	m.CanSeeCode = canSeeCode != 0
	m.AffirmativeResponse = affirmativeResponse != 0

	if err := json.Unmarshal([]byte(capsJSON), &m.Capabilities); err != nil {
		m.Capabilities = map[string]bool{}
	}
	if err := json.Unmarshal([]byte(pricingJSON), &m.Pricing); err != nil {
		m.Pricing = verifier.PricingInfo{}
	}
	if t, err := time.Parse(time.RFC3339, lastVerifiedStr); err == nil {
		m.LastVerifiedAt = t
	}

	return m, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
