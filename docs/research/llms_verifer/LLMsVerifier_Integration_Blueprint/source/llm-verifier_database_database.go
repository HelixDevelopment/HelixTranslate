package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents the main database interface
type Database struct {
	conn *sql.DB
}

// configureSQLCipher configures SQL Cipher encryption settings
func configureSQLCipher(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA cipher_page_size = 4096",
		"PRAGMA kdf_iter = 64000",
		"PRAGMA cipher_hmac_algorithm = HMAC_SHA256",
		"PRAGMA cipher_kdf_algorithm = PBKDF2_HMAC_SHA256",
		"PRAGMA cipher_use_hmac = 1",
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = 10000",
		"PRAGMA temp_store = memory",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute pragma '%s': %w", pragma, err)
		}
	}

	return nil
}

// MigrateToEncrypted migrates an existing unencrypted database to encrypted
func MigrateToEncrypted(sourcePath, destPath, encryptionKey string) error {
	// Open source database
	sourceDB, err := sql.Open("sqlite3", sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer sourceDB.Close()

	// Configure source database
	if err := configureMigrationDB(sourceDB); err != nil {
		return fmt.Errorf("failed to configure source database: %w", err)
	}

	// Create destination database with encryption
	destConnStr := fmt.Sprintf("%s?_pragma_key=%s&_pragma_cipher_page_size=4096&_pragma_kdf_iter=64000",
		destPath, encryptionKey)

	destDB, err := sql.Open("sqlite3", destConnStr)
	if err != nil {
		return fmt.Errorf("failed to create destination database: %w", err)
	}
	defer destDB.Close()

	// Configure destination database
	if err := configureSQLCipher(destDB); err != nil {
		return fmt.Errorf("failed to configure destination database: %w", err)
	}

	// Get all table names from source
	tables, err := getTableNames(sourceDB)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	// Migrate each table
	for _, table := range tables {
		if err := migrateTable(sourceDB, destDB, table); err != nil {
			return fmt.Errorf("failed to migrate table %s: %w", table, err)
		}
	}

	// Migrate indexes and triggers
	if err := migrateIndexesAndTriggers(sourceDB, destDB); err != nil {
		return fmt.Errorf("failed to migrate indexes and triggers: %w", err)
	}

	return nil
}

// configureMigrationDB configures database for migration
func configureMigrationDB(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA foreign_keys = OFF",
		"PRAGMA journal_mode = DELETE",
		"PRAGMA synchronous = FULL",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute pragma '%s': %w", pragma, err)
		}
	}

	return nil
}

// getTableNames returns all table names in the database
func getTableNames(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}

	return tables, rows.Err()
}

// migrateTable migrates a single table from source to destination
func migrateTable(sourceDB, destDB *sql.DB, tableName string) error {
	// Get table schema
	var sql string
	err := sourceDB.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?",
		tableName).Scan(&sql)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	// Create table in destination
	if _, err := destDB.Exec(sql); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Copy data
	return copyTableData(sourceDB, destDB, tableName)
}

// copyTableData copies all data from one table to another
func copyTableData(sourceDB, destDB *sql.DB, tableName string) error {
	// Get column names
	rows, err := sourceDB.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName)) // #nosec G201
	if err != nil {
		return err
	}

	var columns []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			rows.Close()
			return err
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("error iterating table info rows: %w", err)
	}
	rows.Close()

	if len(columns) == 0 {
		return nil
	}

	// Build SELECT and INSERT queries with validation
	quotedTable, err := QuoteTableName(tableName)
	if err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}

	quotedColumns, err := QuoteColumnNames(columns)
	if err != nil {
		return fmt.Errorf("invalid column names: %w", err)
	}

	selectQuery := fmt.Sprintf("SELECT %s FROM %s", strings.Join(quotedColumns, ", "), quotedTable) // #nosec G201 - quotedTable and quotedColumns are validated via QuoteTableName/QuoteColumnNames whitelist
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",                                   // #nosec G201 - quotedTable and quotedColumns are validated via QuoteTableName/QuoteColumnNames whitelist
		quotedTable,
		strings.Join(quotedColumns, ", "),
		strings.Repeat("?,", len(columns))+"?")

	// Copy data
	sourceRows, err := sourceDB.Query(selectQuery)
	if err != nil {
		return err
	}
	defer sourceRows.Close()

	for sourceRows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := sourceRows.Scan(valuePtrs...); err != nil {
			return err
		}

		if _, err := destDB.Exec(insertQuery, values...); err != nil {
			return err
		}
	}

	return sourceRows.Err()
}

// migrateIndexesAndTriggers migrates indexes and triggers
func migrateIndexesAndTriggers(sourceDB, destDB *sql.DB) error {
	// Get all indexes
	rows, err := sourceDB.Query(`
		SELECT name, sql FROM sqlite_master
		WHERE type IN ('index', 'trigger') AND name NOT LIKE 'sqlite_%'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, sql string
		if err := rows.Scan(&name, &sql); err != nil {
			return err
		}

		if sql != "" { // Some indexes might not have SQL
			if _, err := destDB.Exec(sql); err != nil {
				return fmt.Errorf("failed to create %s: %w", name, err)
			}
		}
	}

	return rows.Err()
}

// TxFunc represents a function that can be executed within a transaction
type TxFunc func(*sql.Tx) error

// ConnectionPoolConfig holds advanced connection pool configuration
type ConnectionPoolConfig struct {
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
}

// DefaultConnectionPoolConfig returns sensible defaults for connection pooling
func DefaultConnectionPoolConfig() ConnectionPoolConfig {
	return ConnectionPoolConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	}
}

// New creates a new database connection
func New(dbPath string) (*Database, error) {
	return NewWithConfig(dbPath, DefaultConnectionPoolConfig(), "")
}

// NewEncrypted creates a new encrypted database connection
func NewEncrypted(dbPath, encryptionKey string) (*Database, error) {
	return NewWithConfig(dbPath, DefaultConnectionPoolConfig(), encryptionKey)
}

// NewWithConfig creates a new database connection with custom pool configuration
func NewWithConfig(dbPath string, config ConnectionPoolConfig, encryptionKey ...string) (*Database, error) {
	// Build connection string
	connStr := dbPath
	if len(encryptionKey) > 0 && encryptionKey[0] != "" {
		// SQL Cipher connection string with encryption key
		connStr = fmt.Sprintf("%s?_pragma_key=%s&_pragma_cipher_page_size=4096&_pragma_kdf_iter=64000",
			dbPath, encryptionKey[0])
	}

	// Open database
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure SQL Cipher if encryption is enabled
	if len(encryptionKey) > 0 && encryptionKey[0] != "" {
		if err := configureSQLCipher(db); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to configure SQL Cipher: %w", err)
		}
	}

	// Configure database connection with advanced pooling
	if err := configureConnectionPool(db, config); err != nil {
		return nil, fmt.Errorf("failed to configure connection pool: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{conn: db}

	// Initialize schema
	if err := database.initializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Initialize and run migrations
	migrationManager := NewMigrationManager(database)
	migrationManager.SetupDefaultMigrations()

	if err := migrationManager.InitializeMigrationTable(); err != nil {
		return nil, fmt.Errorf("failed to initialize migration table: %w", err)
	}

	if err := migrationManager.MigrateUp(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return database, nil
}

// configureConnectionPool sets up advanced connection pooling parameters
func configureConnectionPool(db *sql.DB, config ConnectionPoolConfig) error {
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Enable WAL mode for better concurrency (SQLite specific)
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign key constraints
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Set synchronous mode to NORMAL for better performance (safer than OFF)
	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		return fmt.Errorf("failed to set synchronous mode: %w", err)
	}

	// Optimize cache size for better performance
	if _, err := db.Exec("PRAGMA cache_size=-64000"); err != nil { // 64MB cache
		return fmt.Errorf("failed to set cache size: %w", err)
	}

	// Enable memory-mapped I/O for better performance
	if _, err := db.Exec("PRAGMA mmap_size=268435456"); err != nil { // 256MB
		return fmt.Errorf("failed to set mmap size: %w", err)
	}

	// Set busy timeout to handle concurrent access
	if _, err := db.Exec("PRAGMA busy_timeout=30000"); err != nil { // 30 seconds
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Set cache size to 64MB
	if _, err := db.Exec("PRAGMA cache_size=-65536"); err != nil {
		return fmt.Errorf("failed to set cache size: %w", err)
	}

	// Enable memory-mapped I/O for better performance
	if _, err := db.Exec("PRAGMA mmap_size=268435456"); err != nil { // 256MB
		return fmt.Errorf("failed to set mmap size: %w", err)
	}

	return nil
}

// initializeSchema creates the database tables and indexes
func (d *Database) initializeSchema() error {
	schema := `
	-- Enable foreign keys
	PRAGMA foreign_keys = ON;

	-- Users table (system users for authentication and authorization)
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		full_name TEXT,
		role TEXT NOT NULL DEFAULT 'user',
		is_active BOOLEAN DEFAULT 1,
		last_login TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		preferences TEXT
	);

	-- API keys table (for programmatic access)
	CREATE TABLE IF NOT EXISTS api_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		key_hash TEXT NOT NULL UNIQUE,
		scopes TEXT NOT NULL,
		expires_at TIMESTAMP,
		last_used TIMESTAMP,
		is_active BOOLEAN DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- Providers table
	CREATE TABLE IF NOT EXISTS providers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		endpoint TEXT NOT NULL,
		api_key_encrypted TEXT,
		description TEXT,
		website TEXT,
		support_email TEXT,
		documentation_url TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_checked TIMESTAMP,
		is_active BOOLEAN DEFAULT 1,
		reliability_score REAL DEFAULT 0.0,
		average_response_time_ms INTEGER DEFAULT 0
	);

	-- Models table
	CREATE TABLE IF NOT EXISTS models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider_id INTEGER NOT NULL,
		model_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		version TEXT,
		architecture TEXT,
		parameter_count INTEGER,
		context_window_tokens INTEGER,
		max_output_tokens INTEGER,
		training_data_cutoff DATE,
		release_date DATE,
		is_multimodal BOOLEAN DEFAULT 0,
		supports_vision BOOLEAN DEFAULT 0,
		supports_audio BOOLEAN DEFAULT 0,
		supports_video BOOLEAN DEFAULT 0,
		supports_reasoning BOOLEAN DEFAULT 0,
		open_source BOOLEAN DEFAULT 0,
		deprecated BOOLEAN DEFAULT 0,
		tags TEXT,
		language_support TEXT,
		use_case TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_verified TIMESTAMP,
		verification_status TEXT DEFAULT 'pending',
		overall_score REAL DEFAULT 0.0,
		code_capability_score REAL DEFAULT 0.0,
		responsiveness_score REAL DEFAULT 0.0,
		reliability_score REAL DEFAULT 0.0,
		feature_richness_score REAL DEFAULT 0.0,
		value_proposition_score REAL DEFAULT 0.0,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
	);

	-- Pricing table
	CREATE TABLE IF NOT EXISTS pricing (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		input_token_cost REAL DEFAULT 0.0,
		output_token_cost REAL DEFAULT 0.0,
		cached_input_token_cost REAL DEFAULT 0.0,
		storage_cost REAL DEFAULT 0.0,
		request_cost REAL DEFAULT 0.0,
		currency TEXT DEFAULT 'USD',
		pricing_model TEXT DEFAULT 'per_token',
		effective_from DATE,
		effective_to DATE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
	);

	-- Limits table
	CREATE TABLE IF NOT EXISTS limits (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		limit_type TEXT NOT NULL,
		limit_value INTEGER NOT NULL,
		current_usage INTEGER DEFAULT 0,
		reset_period TEXT,
		reset_time TIMESTAMP,
		is_hard_limit BOOLEAN DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
	);

	-- Verification results table
	CREATE TABLE IF NOT EXISTS verification_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		verification_type TEXT NOT NULL,
		started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		status TEXT DEFAULT 'running',
		error_message TEXT,
		model_exists BOOLEAN,
		responsive BOOLEAN,
		overloaded BOOLEAN,
		latency_ms INTEGER,
		supports_tool_use BOOLEAN DEFAULT 0,
		supports_function_calling BOOLEAN DEFAULT 0,
		supports_code_generation BOOLEAN DEFAULT 0,
		supports_code_completion BOOLEAN DEFAULT 0,
		supports_code_review BOOLEAN DEFAULT 0,
		supports_code_explanation BOOLEAN DEFAULT 0,
		supports_embeddings BOOLEAN DEFAULT 0,
		supports_reranking BOOLEAN DEFAULT 0,
		supports_image_generation BOOLEAN DEFAULT 0,
		supports_audio_generation BOOLEAN DEFAULT 0,
		supports_video_generation BOOLEAN DEFAULT 0,
		supports_mcps BOOLEAN DEFAULT 0,
					supports_lsps BOOLEAN DEFAULT 0,
			supports_acps BOOLEAN DEFAULT 0,
			supports_multimodal BOOLEAN DEFAULT 0,
		supports_streaming BOOLEAN DEFAULT 0,
		supports_json_mode BOOLEAN DEFAULT 0,
		supports_structured_output BOOLEAN DEFAULT 0,
		supports_reasoning BOOLEAN DEFAULT 0,
		supports_parallel_tool_use BOOLEAN DEFAULT 0,
		max_parallel_calls INTEGER DEFAULT 0,
	supports_batch_processing BOOLEAN DEFAULT 0,
	supports_brotli BOOLEAN DEFAULT 0,
	code_language_support TEXT,
	code_debugging BOOLEAN DEFAULT 0,
		code_optimization BOOLEAN DEFAULT 0,
		test_generation BOOLEAN DEFAULT 0,
		documentation_generation BOOLEAN DEFAULT 0,
		refactoring BOOLEAN DEFAULT 0,
		error_resolution BOOLEAN DEFAULT 0,
		architecture_design BOOLEAN DEFAULT 0,
		security_assessment BOOLEAN DEFAULT 0,
		pattern_recognition BOOLEAN DEFAULT 0,
		debugging_accuracy REAL DEFAULT 0.0,
		max_handled_depth INTEGER DEFAULT 0,
		code_quality_score REAL DEFAULT 0.0,
		logic_correctness_score REAL DEFAULT 0.0,
		runtime_efficiency_score REAL DEFAULT 0.0,
		overall_score REAL DEFAULT 0.0,
		code_capability_score REAL DEFAULT 0.0,
		responsiveness_score REAL DEFAULT 0.0,
		reliability_score REAL DEFAULT 0.0,
		feature_richness_score REAL DEFAULT 0.0,
		value_proposition_score REAL DEFAULT 0.0,
		score_details TEXT,
		avg_latency_ms INTEGER DEFAULT 0,
		p95_latency_ms INTEGER DEFAULT 0,
		min_latency_ms INTEGER DEFAULT 0,
		max_latency_ms INTEGER DEFAULT 0,
		throughput_rps REAL DEFAULT 0.0,
		raw_request TEXT,
		raw_response TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
	);

	-- Issues table
	CREATE TABLE IF NOT EXISTS issues (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		issue_type TEXT NOT NULL,
		severity TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		symptoms TEXT,
		workarounds TEXT,
		affected_features TEXT,
		first_detected TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_occurred TIMESTAMP,
		resolved_at TIMESTAMP,
		resolution_notes TEXT,
		verification_result_id INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
		FOREIGN KEY (verification_result_id) REFERENCES verification_results(id) ON DELETE SET NULL
	);

	-- Events table
	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		severity TEXT DEFAULT 'info',
		title TEXT NOT NULL,
		message TEXT NOT NULL,
		details TEXT,
		model_id INTEGER,
		provider_id INTEGER,
		verification_result_id INTEGER,
		issue_id INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
		FOREIGN KEY (verification_result_id) REFERENCES verification_results(id) ON DELETE CASCADE,
		FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
	);

	-- Schedules table
	CREATE TABLE IF NOT EXISTS schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		schedule_type TEXT NOT NULL,
		cron_expression TEXT,
		interval_seconds INTEGER,
		target_type TEXT NOT NULL,
		target_id INTEGER,
		is_active BOOLEAN DEFAULT 1,
		last_run TIMESTAMP,
		next_run TIMESTAMP,
		run_count INTEGER DEFAULT 0,
		max_runs INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT,
		FOREIGN KEY (target_id) REFERENCES models(id) ON DELETE CASCADE
	);

	-- Schedule runs table
	CREATE TABLE IF NOT EXISTS schedule_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		schedule_id INTEGER NOT NULL,
		started_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP,
		status TEXT NOT NULL,
		results_count INTEGER DEFAULT 0,
		errors_count INTEGER DEFAULT 0,
		error_message TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE
	);

	-- Configuration exports table
	CREATE TABLE IF NOT EXISTS config_exports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		export_type TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		config_data TEXT NOT NULL,
		target_models TEXT,
		target_providers TEXT,
		filters TEXT,
		is_verified BOOLEAN DEFAULT 0,
		verification_notes TEXT,
		created_by TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		download_count INTEGER DEFAULT 0
	);

	-- Logs table
	CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		level TEXT NOT NULL,
		logger TEXT NOT NULL,
		message TEXT NOT NULL,
		details TEXT,
		request_id TEXT,
		user_id INTEGER,
		model_id INTEGER,
		provider_id INTEGER,
		verification_result_id INTEGER,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
		FOREIGN KEY (verification_result_id) REFERENCES verification_results(id) ON DELETE CASCADE
	);

	-- Verification scores table
	CREATE TABLE IF NOT EXISTS verification_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		verification_result_id INTEGER,
		score INTEGER NOT NULL,
		score_type TEXT NOT NULL,
		scoring_method TEXT NOT NULL,
		category TEXT NOT NULL,
		code_correctness_score INTEGER,
		code_quality_score INTEGER,
		code_speed_score INTEGER,
		error_handling_score INTEGER,
		context_understanding_score INTEGER,
		evidence TEXT,
		benchmark_version TEXT NOT NULL,
		scored_by TEXT NOT NULL,
		confidence_level INTEGER NOT NULL,
		scored_at TIMESTAMP NOT NULL,
		expires_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
		FOREIGN KEY (verification_result_id) REFERENCES verification_results(id) ON DELETE SET NULL
	);

	-- Basic indexes
	CREATE INDEX IF NOT EXISTS idx_providers_name ON providers(name);
	CREATE INDEX IF NOT EXISTS idx_providers_endpoint ON providers(endpoint);
	CREATE INDEX IF NOT EXISTS idx_providers_active ON providers(is_active);
	CREATE INDEX IF NOT EXISTS idx_models_provider ON models(provider_id);
	CREATE INDEX IF NOT EXISTS idx_models_model_id ON models(model_id);
	CREATE INDEX IF NOT EXISTS idx_models_verification_status ON models(verification_status);
	CREATE INDEX IF NOT EXISTS idx_models_overall_score ON models(overall_score);
	CREATE INDEX IF NOT EXISTS idx_pricing_model ON pricing(model_id);
	CREATE INDEX IF NOT EXISTS idx_limits_model ON limits(model_id);
	CREATE INDEX IF NOT EXISTS idx_verification_results_model ON verification_results(model_id);
	CREATE INDEX IF NOT EXISTS idx_verification_results_status ON verification_results(status);
	CREATE INDEX IF NOT EXISTS idx_verification_results_created_at ON verification_results(created_at);
	CREATE INDEX IF NOT EXISTS idx_issues_model ON issues(model_id);
	CREATE INDEX IF NOT EXISTS idx_issues_severity ON issues(severity);
	CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
	CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
	CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);
	CREATE INDEX IF NOT EXISTS idx_verification_scores_model ON verification_scores(model_id);
	CREATE INDEX IF NOT EXISTS idx_verification_scores_score_type ON verification_scores(score_type);
	CREATE INDEX IF NOT EXISTS idx_verification_scores_scored_at ON verification_scores(scored_at);
	`

	_, err := d.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.conn.Close()
}

// Ping tests the database connection
func (d *Database) Ping() error {
	return d.conn.Ping()
}

// Transaction executes a function within a database transaction
func (d *Database) Transaction(fn func(*sql.Tx) error) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Provider represents a model provider
type Provider struct {
	ID                    int64      `json:"id"`
	Name                  string     `json:"name"`
	Endpoint              string     `json:"endpoint"`
	APIKeyEncrypted       string     `json:"api_key_encrypted"`
	Description           string     `json:"description"`
	Website               string     `json:"website"`
	SupportEmail          string     `json:"support_email"`
	DocumentationURL      string     `json:"documentation_url"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	LastChecked           *time.Time `json:"last_checked"`
	IsActive              bool       `json:"is_active"`
	ReliabilityScore      float64    `json:"reliability_score"`
	AverageResponseTimeMs int        `json:"average_response_time_ms"`
}

// User represents a system user
type User struct {
	ID           int64          `json:"id"`
	Username     string         `json:"username"`
	Email        string         `json:"email"`
	PasswordHash string         `json:"-"` // Never expose password hash
	FullName     string         `json:"full_name"`
	Role         string         `json:"role"`
	IsActive     bool           `json:"is_active"`
	LastLogin    *time.Time     `json:"last_login"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Preferences  map[string]any `json:"preferences"`
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Name      string     `json:"name"`
	KeyHash   string     `json:"-"` // Never expose key hash
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
	LastUsed  *time.Time `json:"last_used"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
}

// Model represents an LLM model
type Model struct {
	ID                    int64      `json:"id"`
	ProviderID            int64      `json:"provider_id"`
	ModelID               string     `json:"model_id"`
	Name                  string     `json:"name"`
	Description           string     `json:"description"`
	Version               string     `json:"version"`
	Architecture          string     `json:"architecture"`
	ParameterCount        *int64     `json:"parameter_count"`
	ContextWindowTokens   *int       `json:"context_window_tokens"`
	MaxOutputTokens       *int       `json:"max_output_tokens"`
	TrainingDataCutoff    *time.Time `json:"training_data_cutoff"`
	ReleaseDate           *time.Time `json:"release_date"`
	IsMultimodal          bool       `json:"is_multimodal"`
	SupportsVision        bool       `json:"supports_vision"`
	SupportsAudio         bool       `json:"supports_audio"`
	SupportsVideo         bool       `json:"supports_video"`
	SupportsReasoning     bool       `json:"supports_reasoning"`
	OpenSource            bool       `json:"open_source"`
	Deprecated            bool       `json:"deprecated"`
	Tags                  []string   `json:"tags"`
	LanguageSupport       []string   `json:"language_support"`
	UseCase               string     `json:"use_case"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	LastVerified          *time.Time `json:"last_verified"`
	VerificationStatus    string     `json:"verification_status"`
	OverallScore          float64    `json:"overall_score"`
	CodeCapabilityScore   float64    `json:"code_capability_score"`
	ResponsivenessScore   float64    `json:"responsiveness_score"`
	ReliabilityScore      float64    `json:"reliability_score"`
	FeatureRichnessScore  float64    `json:"feature_richness_score"`
	ValuePropositionScore float64    `json:"value_proposition_score"`
}

// VerificationResult represents a verification run result
type VerificationResult struct {
	ID                       int64      `json:"id"`
	ModelID                  int64      `json:"model_id"`
	VerificationType         string     `json:"verification_type"`
	StartedAt                time.Time  `json:"started_at"`
	CompletedAt              *time.Time `json:"completed_at"`
	Status                   string     `json:"status"`
	ErrorMessage             *string    `json:"error_message"`
	ModelExists              *bool      `json:"model_exists"`
	Responsive               *bool      `json:"responsive"`
	Overloaded               *bool      `json:"overloaded"`
	LatencyMs                *int       `json:"latency_ms"`
	SupportsToolUse          bool       `json:"supports_tool_use"`
	SupportsFunctionCalling  bool       `json:"supports_function_calling"`
	SupportsCodeGeneration   bool       `json:"supports_code_generation"`
	SupportsCodeCompletion   bool       `json:"supports_code_completion"`
	SupportsCodeReview       bool       `json:"supports_code_review"`
	SupportsCodeExplanation  bool       `json:"supports_code_explanation"`
	SupportsEmbeddings       bool       `json:"supports_embeddings"`
	SupportsReranking        bool       `json:"supports_reranking"`
	SupportsImageGeneration  bool       `json:"supports_image_generation"`
	SupportsAudioGeneration  bool       `json:"supports_audio_generation"`
	SupportsVideoGeneration  bool       `json:"supports_video_generation"`
	SupportsMCPs             bool       `json:"supports_mcps"`
	SupportsLSPs             bool       `json:"supports_lsps"`
	SupportsACPs             bool       `json:"supports_acps"`
	SupportsMultimodal       bool       `json:"supports_multimodal"`
	SupportsStreaming        bool       `json:"supports_streaming"`
	SupportsJSONMode         bool       `json:"supports_json_mode"`
	SupportsStructuredOutput bool       `json:"supports_structured_output"`
	SupportsReasoning        bool       `json:"supports_reasoning"`
	SupportsParallelToolUse  bool       `json:"supports_parallel_tool_use"`
	MaxParallelCalls         int        `json:"max_parallel_calls"`
	SupportsBatchProcessing  bool       `json:"supports_batch_processing"`
	SupportsBrotli           bool       `json:"supports_brotli"`
	CodeLanguageSupport      []string   `json:"code_language_support"`
	CodeDebugging            bool       `json:"code_debugging"`
	CodeOptimization         bool       `json:"code_optimization"`
	TestGeneration           bool       `json:"test_generation"`
	DocumentationGeneration  bool       `json:"documentation_generation"`
	Refactoring              bool       `json:"refactoring"`
	ErrorResolution          bool       `json:"error_resolution"`
	ArchitectureDesign       bool       `json:"architecture_design"`
	SecurityAssessment       bool       `json:"security_assessment"`
	PatternRecognition       bool       `json:"pattern_recognition"`
	DebuggingAccuracy        float64    `json:"debugging_accuracy"`
	MaxHandledDepth          int        `json:"max_handled_depth"`
	CodeQualityScore         float64    `json:"code_quality_score"`
	LogicCorrectnessScore    float64    `json:"logic_correctness_score"`
	RuntimeEfficiencyScore   float64    `json:"runtime_efficiency_score"`
	OverallScore             float64    `json:"overall_score"`
	CodeCapabilityScore      float64    `json:"code_capability_score"`
	ResponsivenessScore      float64    `json:"responsiveness_score"`
	ReliabilityScore         float64    `json:"reliability_score"`
	FeatureRichnessScore     float64    `json:"feature_richness_score"`
	ValuePropositionScore    float64    `json:"value_proposition_score"`
	ScoreDetails             string     `json:"score_details"`
	AvgLatencyMs             int        `json:"avg_latency_ms"`
	P95LatencyMs             int        `json:"p95_latency_ms"`
	MinLatencyMs             int        `json:"min_latency_ms"`
	MaxLatencyMs             int        `json:"max_latency_ms"`
	ThroughputRPS            float64    `json:"throughput_rps"`
	RawRequest               *string    `json:"raw_request"`
	RawResponse              *string    `json:"raw_response"`
	CreatedAt                time.Time  `json:"created_at"`
}

// Event represents a system event
type Event struct {
	ID                   int64     `json:"id"`
	EventType            string    `json:"event_type"`
	Severity             string    `json:"severity"`
	Title                string    `json:"title"`
	Message              string    `json:"message"`
	Details              *string   `json:"details"`
	ModelID              *int64    `json:"model_id"`
	ProviderID           *int64    `json:"provider_id"`
	VerificationResultID *int64    `json:"verification_result_id"`
	IssueID              *int64    `json:"issue_id"`
	CreatedAt            time.Time `json:"created_at"`
}

// Schedule represents a scheduled verification task
type Schedule struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Description     *string    `json:"description"`
	ScheduleType    string     `json:"schedule_type"` // cron, interval, manual
	CronExpression  *string    `json:"cron_expression"`
	IntervalSeconds *int       `json:"interval_seconds"`
	TargetType      string     `json:"target_type"` // all_models, provider, specific_model
	TargetID        *int64     `json:"target_id"`   // provider_id or model_id depending on target_type
	IsActive        bool       `json:"is_active"`
	LastRun         *time.Time `json:"last_run"`
	NextRun         *time.Time `json:"next_run"`
	RunCount        int        `json:"run_count"`
	MaxRuns         *int       `json:"max_runs"` // NULL for unlimited
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CreatedBy       *string    `json:"created_by"` // For future multi-user support
}

// ScheduleRun represents a schedule execution run
type ScheduleRun struct {
	ID           int64      `json:"id"`
	ScheduleID   int64      `json:"schedule_id"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	Status       string     `json:"status"` // running, completed, failed, cancelled
	ResultsCount int        `json:"results_count"`
	ErrorsCount  int        `json:"errors_count"`
	ErrorMessage *string    `json:"error_message"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ConfigExport represents a configuration export
type ConfigExport struct {
	ID                int64     `json:"id"`
	ExportType        string    `json:"export_type"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	ConfigData        string    `json:"config_data"`
	TargetModels      *string   `json:"target_models"`
	TargetProviders   *string   `json:"target_providers"`
	Filters           *string   `json:"filters"`
	IsVerified        bool      `json:"is_verified"`
	VerificationNotes *string   `json:"verification_notes"`
	CreatedBy         *string   `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	DownloadCount     int       `json:"download_count"`
}

// LogEntry represents a log entry
type LogEntry struct {
	ID                   int64     `json:"id"`
	Timestamp            time.Time `json:"timestamp"`
	Level                string    `json:"level"`
	Logger               string    `json:"logger"`
	Message              string    `json:"message"`
	Details              *string   `json:"details"`
	RequestID            *string   `json:"request_id"`
	UserID               *int64    `json:"user_id"` // For future multi-user support
	ModelID              *int64    `json:"model_id"`
	ProviderID           *int64    `json:"provider_id"`
	VerificationResultID *int64    `json:"verification_result_id"`
}

// Helper function to scan nullable time
func scanNullableTime(nullTime sql.NullTime) *time.Time {
	if nullTime.Valid {
		return &nullTime.Time
	}
	return nil
}

// Helper function to scan nullable string
func scanNullableString(nullString sql.NullString) *string {
	if nullString.Valid {
		return &nullString.String
	}
	return nil
}

// Helper function to scan nullable int64
func scanNullableInt64(nullInt64 sql.NullInt64) *int64 {
	if nullInt64.Valid {
		return &nullInt64.Int64
	}
	return nil
}

// Helper function to scan nullable bool from string
func scanNullableBoolFromString(nullString sql.NullString) *bool {
	switch nullString.String {
	case "":
		return nil
	case "true", "1":
		val := true
		return &val
	case "false", "0":
		val := false
		return &val
	}

	return nil
}

// Helper function to scan nullable time from string
func scanNullableTimeFromString(nullString sql.NullString) *time.Time {
	if !nullString.Valid || nullString.String == "" {
		return nil
	}

	// Try to parse as RFC3339 timestamp
	if t, err := time.Parse(time.RFC3339, nullString.String); err == nil {
		return &t
	}

	// Try to parse as Unix timestamp
	if timestamp, err := strconv.ParseInt(nullString.String, 10, 64); err == nil {
		t := time.Unix(timestamp, 0)
		return &t
	}

	return nil
}

// Helper function to scan nullable int from int64
func scanNullableIntFromInt64(nullInt64 sql.NullInt64) *int {
	if nullInt64.Valid {
		val := int(nullInt64.Int64)
		return &val
	}
	return nil
}

// Helper function to scan JSON string
func scanJSONString(nullString sql.NullString) []string {
	if !nullString.Valid || nullString.String == "" {
		return []string{}
	}

	var result []string
	if err := json.Unmarshal([]byte(nullString.String), &result); err != nil {
		return []string{}
	}
	return result
}

// Helper functions to convert from Go pointers to SQL nullable types

// toNullTime converts a *time.Time to sql.NullTime
func toNullTime(t *time.Time) sql.NullTime {
	if t != nil {
		return sql.NullTime{Time: *t, Valid: true}
	}
	return sql.NullTime{Valid: false}
}

// toNullString converts a *string to sql.NullString
func toNullString(s *string) sql.NullString {
	if s != nil {
		return sql.NullString{String: *s, Valid: true}
	}
	return sql.NullString{Valid: false}
}

// toNullInt converts a *int to sql.NullInt32
func toNullInt(i *int) sql.NullInt32 {
	if i != nil {
		return sql.NullInt32{Int32: int32(*i), Valid: true}
	}
	return sql.NullInt32{Valid: false}
}

// toNullBool converts a *bool to sql.NullBool
func toNullBool(b *bool) sql.NullBool {
	if b != nil {
		return sql.NullBool{Bool: *b, Valid: true}
	}
	return sql.NullBool{Valid: false}
}

// Pricing represents model pricing information
type Pricing struct {
	ID                   int64      `json:"id"`
	ModelID              int64      `json:"model_id"`
	InputTokenCost       float64    `json:"input_token_cost"`
	OutputTokenCost      float64    `json:"output_token_cost"`
	CachedInputTokenCost float64    `json:"cached_input_token_cost"`
	StorageCost          float64    `json:"storage_cost"`
	RequestCost          float64    `json:"request_cost"`
	Currency             string     `json:"currency"`
	PricingModel         string     `json:"pricing_model"`
	EffectiveFrom        *time.Time `json:"effective_from"`
	EffectiveTo          *time.Time `json:"effective_to"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// Limit represents rate limits and quotas
type Limit struct {
	ID           int64      `json:"id"`
	ModelID      int64      `json:"model_id"`
	LimitType    string     `json:"limit_type"`
	LimitValue   int        `json:"limit_value"`
	CurrentUsage int        `json:"current_usage"`
	ResetPeriod  string     `json:"reset_period"`
	ResetTime    *time.Time `json:"reset_time"`
	IsHardLimit  bool       `json:"is_hard_limit"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Issue represents documented problems with models
type Issue struct {
	ID                   int64      `json:"id"`
	ModelID              int64      `json:"model_id"`
	IssueType            string     `json:"issue_type"`
	Severity             string     `json:"severity"`
	Title                string     `json:"title"`
	Description          string     `json:"description"`
	Symptoms             *string    `json:"symptoms"`
	Workarounds          *string    `json:"workarounds"`
	AffectedFeatures     []string   `json:"affected_features"`
	FirstDetected        time.Time  `json:"first_detected"`
	LastOccurred         *time.Time `json:"last_occurred"`
	ResolvedAt           *time.Time `json:"resolved_at"`
	ResolutionNotes      *string    `json:"resolution_notes"`
	VerificationResultID *int64     `json:"verification_result_id"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// Notification represents a sent notification and its status
type Notification struct {
	ID         int64          `json:"id"`
	Type       string         `json:"type"`
	Channel    string         `json:"channel"`
	Priority   string         `json:"priority"`
	Title      string         `json:"title"`
	Message    string         `json:"message"`
	Data       map[string]any `json:"data"`
	Recipient  string         `json:"recipient"`
	Sent       bool           `json:"sent"`
	Error      string         `json:"error"`
	RetryCount int            `json:"retry_count"`
	CreatedAt  time.Time      `json:"created_at"`
	SentAt     *time.Time     `json:"sent_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// ==================== Transaction Helper Methods ====================

// createVerificationResultTx creates a verification result within a transaction
func (d *Database) createVerificationResultTx(tx *sql.Tx, verificationResult *VerificationResult) error {
	langSupportJSON, err := json.Marshal(verificationResult.CodeLanguageSupport)
	if err != nil {
		return fmt.Errorf("failed to marshal code language support: %w", err)
	}

	query := `
		INSERT INTO verification_results (
			model_id, verification_type, started_at, completed_at, status, error_message,
			model_exists, responsive, overloaded, latency_ms, supports_tool_use,
			supports_function_calling, supports_code_generation, supports_code_completion,
			supports_code_review, supports_code_explanation, supports_embeddings,
			supports_reranking, supports_image_generation, supports_audio_generation,
			supports_video_generation, supports_mcps, supports_lsps, supports_acps, supports_multimodal,
			supports_streaming, supports_json_mode, supports_structured_output,
			supports_reasoning, supports_parallel_tool_use, max_parallel_calls,
		supports_batch_processing, supports_brotli, code_language_support, code_debugging,
		code_optimization, test_generation, documentation_generation, refactoring,
		error_resolution, architecture_design, security_assessment,
		pattern_recognition, debugging_accuracy, max_handled_depth,
		code_quality_score, logic_correctness_score, runtime_efficiency_score,
		overall_score, code_capability_score, responsiveness_score,
		reliability_score, feature_richness_score, value_proposition_score,
		score_details, avg_latency_ms, p95_latency_ms, min_latency_ms,
		max_latency_ms, throughput_rps, raw_request, raw_response
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := tx.Exec(query,
		verificationResult.ModelID,
		verificationResult.VerificationType,
		verificationResult.StartedAt,
		verificationResult.CompletedAt,
		verificationResult.Status,
		verificationResult.ErrorMessage,
		verificationResult.ModelExists,
		verificationResult.Responsive,
		verificationResult.Overloaded,
		verificationResult.LatencyMs,
		verificationResult.SupportsToolUse,
		verificationResult.SupportsFunctionCalling,
		verificationResult.SupportsCodeGeneration,
		verificationResult.SupportsCodeCompletion,
		verificationResult.SupportsCodeReview,
		verificationResult.SupportsCodeExplanation,
		verificationResult.SupportsEmbeddings,
		verificationResult.SupportsReranking,
		verificationResult.SupportsImageGeneration,
		verificationResult.SupportsAudioGeneration,
		verificationResult.SupportsVideoGeneration,
		verificationResult.SupportsMCPs,
		verificationResult.SupportsLSPs,
		verificationResult.SupportsACPs,
		verificationResult.SupportsMultimodal,
		verificationResult.SupportsStreaming,
		verificationResult.SupportsJSONMode,
		verificationResult.SupportsStructuredOutput,
		verificationResult.SupportsReasoning,
		verificationResult.SupportsParallelToolUse,
		verificationResult.MaxParallelCalls,
		verificationResult.SupportsBatchProcessing,
		verificationResult.SupportsBrotli,
		string(langSupportJSON),
		verificationResult.CodeDebugging,
		verificationResult.CodeOptimization,
		verificationResult.TestGeneration,
		verificationResult.DocumentationGeneration,
		verificationResult.Refactoring,
		verificationResult.ErrorResolution,
		verificationResult.ArchitectureDesign,
		verificationResult.SecurityAssessment,
		verificationResult.PatternRecognition,
		verificationResult.DebuggingAccuracy,
		verificationResult.MaxHandledDepth,
		verificationResult.CodeQualityScore,
		verificationResult.LogicCorrectnessScore,
		verificationResult.RuntimeEfficiencyScore,
		verificationResult.OverallScore,
		verificationResult.CodeCapabilityScore,
		verificationResult.ResponsivenessScore,
		verificationResult.ReliabilityScore,
		verificationResult.FeatureRichnessScore,
		verificationResult.ValuePropositionScore,
		verificationResult.ScoreDetails,
		verificationResult.AvgLatencyMs,
		verificationResult.P95LatencyMs,
		verificationResult.MinLatencyMs,
		verificationResult.MaxLatencyMs,
		verificationResult.ThroughputRPS,
		verificationResult.RawRequest,
		verificationResult.RawResponse,
	)

	if err != nil {
		return fmt.Errorf("failed to create verification result: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	verificationResult.ID = id
	return nil
}

// createModelTx creates a model within a transaction
func (d *Database) createModelTx(tx *sql.Tx, model *Model) error {
	tagsJSON, err := json.Marshal(model.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	langSupportJSON, err := json.Marshal(model.LanguageSupport)
	if err != nil {
		return fmt.Errorf("failed to marshal language support: %w", err)
	}

	query := `
		INSERT INTO models (
			provider_id, model_id, name, description, version, architecture,
			parameter_count, context_window_tokens, max_output_tokens,
			training_data_cutoff, release_date, is_multimodal, supports_vision,
			supports_audio, supports_video, supports_reasoning, open_source,
			deprecated, tags, language_support, use_case, verification_status,
			overall_score, code_capability_score, responsiveness_score,
			reliability_score, feature_richness_score, value_proposition_score
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := tx.Exec(query,
		model.ProviderID,
		model.ModelID,
		model.Name,
		model.Description,
		model.Version,
		model.Architecture,
		model.ParameterCount,
		model.ContextWindowTokens,
		model.MaxOutputTokens,
		model.TrainingDataCutoff,
		model.ReleaseDate,
		model.IsMultimodal,
		model.SupportsVision,
		model.SupportsAudio,
		model.SupportsVideo,
		model.SupportsReasoning,
		model.OpenSource,
		model.Deprecated,
		string(tagsJSON),
		string(langSupportJSON),
		model.UseCase,
		model.VerificationStatus,
		model.OverallScore,
		model.CodeCapabilityScore,
		model.ResponsivenessScore,
		model.ReliabilityScore,
		model.FeatureRichnessScore,
		model.ValuePropositionScore,
	)

	if err != nil {
		return fmt.Errorf("failed to create model: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	model.ID = id
	return nil
}

// updateModelTx updates a model within a transaction
func (d *Database) updateModelTx(tx *sql.Tx, model *Model) error {
	tagsJSON, err := json.Marshal(model.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	langSupportJSON, err := json.Marshal(model.LanguageSupport)
	if err != nil {
		return fmt.Errorf("failed to marshal language support: %w", err)
	}

	query := `
		UPDATE models SET
			provider_id = ?, model_id = ?, name = ?, description = ?, version = ?,
			architecture = ?, parameter_count = ?, context_window_tokens = ?,
			max_output_tokens = ?, training_data_cutoff = ?, release_date = ?,
			is_multimodal = ?, supports_vision = ?, supports_audio = ?,
			supports_video = ?, supports_reasoning = ?, open_source = ?,
			deprecated = ?, tags = ?, language_support = ?, use_case = ?,
			last_verified = ?, verification_status = ?, overall_score = ?,
			code_capability_score = ?, responsiveness_score = ?, reliability_score = ?,
			feature_richness_score = ?, value_proposition_score = ?
		WHERE id = ?
	`

	_, err = tx.Exec(query,
		model.ProviderID,
		model.ModelID,
		model.Name,
		model.Description,
		model.Version,
		model.Architecture,
		model.ParameterCount,
		model.ContextWindowTokens,
		model.MaxOutputTokens,
		model.TrainingDataCutoff,
		model.ReleaseDate,
		model.IsMultimodal,
		model.SupportsVision,
		model.SupportsAudio,
		model.SupportsVideo,
		model.SupportsReasoning,
		model.OpenSource,
		model.Deprecated,
		string(tagsJSON),
		string(langSupportJSON),
		model.UseCase,
		model.LastVerified,
		model.VerificationStatus,
		model.OverallScore,
		model.CodeCapabilityScore,
		model.ResponsivenessScore,
		model.ReliabilityScore,
		model.FeatureRichnessScore,
		model.ValuePropositionScore,
		model.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	return nil
}

// createPricingTx creates pricing within a transaction
func (d *Database) createPricingTx(tx *sql.Tx, pricing *Pricing) error {
	query := `
		INSERT INTO pricing (
			model_id, input_token_cost, output_token_cost, cached_input_token_cost,
			storage_cost, request_cost, currency, pricing_model, effective_from,
			effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := tx.Exec(query,
		pricing.ModelID,
		pricing.InputTokenCost,
		pricing.OutputTokenCost,
		pricing.CachedInputTokenCost,
		pricing.StorageCost,
		pricing.RequestCost,
		pricing.Currency,
		pricing.PricingModel,
		pricing.EffectiveFrom,
		pricing.EffectiveTo,
	)

	if err != nil {
		return fmt.Errorf("failed to create pricing: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	pricing.ID = id
	return nil
}

// createLimitTx creates a limit within a transaction
func (d *Database) createLimitTx(tx *sql.Tx, limit *Limit) error {
	query := `
		INSERT INTO limits (
			model_id, limit_type, limit_value, current_usage, reset_period,
			reset_time, is_hard_limit
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := tx.Exec(query,
		limit.ModelID,
		limit.LimitType,
		limit.LimitValue,
		limit.CurrentUsage,
		limit.ResetPeriod,
		limit.ResetTime,
		limit.IsHardLimit,
	)

	if err != nil {
		return fmt.Errorf("failed to create limit: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	limit.ID = id
	return nil
}

// deletePricingForModelTx deletes all pricing for a model within a transaction
func (d *Database) deletePricingForModelTx(tx *sql.Tx, modelID int64) error {
	query := `DELETE FROM pricing WHERE model_id = ?`

	_, err := tx.Exec(query, modelID)
	if err != nil {
		return fmt.Errorf("failed to delete pricing for model: %w", err)
	}

	return nil
}

// deleteLimitsForModelTx deletes all limits for a model within a transaction
func (d *Database) deleteLimitsForModelTx(tx *sql.Tx, modelID int64) error {
	query := `DELETE FROM limits WHERE model_id = ?`

	_, err := tx.Exec(query, modelID)
	if err != nil {
		return fmt.Errorf("failed to delete limits for model: %w", err)
	}

	return nil
}

// ==================== Transaction Support ====================

// WithTransaction executes a function within a database transaction
func (d *Database) WithTransaction(fn TxFunc) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed and rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreateVerificationWithResults creates a verification result along with associated pricing and limits
func (d *Database) CreateVerificationWithResults(modelID int64, verificationResult *VerificationResult, pricing []*Pricing, limits []*Limit) error {
	return d.WithTransaction(func(tx *sql.Tx) error {
		// Create verification result
		if err := d.createVerificationResultTx(tx, verificationResult); err != nil {
			return fmt.Errorf("failed to create verification result: %w", err)
		}

		// Create pricing records
		for _, price := range pricing {
			if err := d.createPricingTx(tx, price); err != nil {
				return fmt.Errorf("failed to create pricing: %w", err)
			}
		}

		// Create limit records
		for _, limit := range limits {
			if err := d.createLimitTx(tx, limit); err != nil {
				return fmt.Errorf("failed to create limit: %w", err)
			}
		}

		return nil
	})
}

// CreateModelWithDependencies creates a model along with its pricing and limits
func (d *Database) CreateModelWithDependencies(model *Model, pricing []*Pricing, limits []*Limit) error {
	return d.WithTransaction(func(tx *sql.Tx) error {
		// Create model
		if err := d.createModelTx(tx, model); err != nil {
			return fmt.Errorf("failed to create model: %w", err)
		}

		// Create pricing records
		for _, price := range pricing {
			price.ModelID = model.ID
			if err := d.createPricingTx(tx, price); err != nil {
				return fmt.Errorf("failed to create pricing: %w", err)
			}
		}

		// Create limit records
		for _, limit := range limits {
			limit.ModelID = model.ID
			if err := d.createLimitTx(tx, limit); err != nil {
				return fmt.Errorf("failed to create limit: %w", err)
			}
		}

		return nil
	})
}

// UpdateModelWithDependencies updates a model and its associated pricing/limits
func (d *Database) UpdateModelWithDependencies(model *Model, pricing []*Pricing, limits []*Limit) error {
	return d.WithTransaction(func(tx *sql.Tx) error {
		// Update model
		if err := d.updateModelTx(tx, model); err != nil {
			return fmt.Errorf("failed to update model: %w", err)
		}

		// Delete existing pricing and limits for this model
		if err := d.deletePricingForModelTx(tx, model.ID); err != nil {
			return fmt.Errorf("failed to delete existing pricing: %w", err)
		}
		if err := d.deleteLimitsForModelTx(tx, model.ID); err != nil {
			return fmt.Errorf("failed to delete existing limits: %w", err)
		}

		// Create new pricing records
		for _, price := range pricing {
			price.ModelID = model.ID
			if err := d.createPricingTx(tx, price); err != nil {
				return fmt.Errorf("failed to create pricing: %w", err)
			}
		}

		// Create new limit records
		for _, limit := range limits {
			limit.ModelID = model.ID
			if err := d.createLimitTx(tx, limit); err != nil {
				return fmt.Errorf("failed to create limit: %w", err)
			}
		}

		return nil
	})
}
