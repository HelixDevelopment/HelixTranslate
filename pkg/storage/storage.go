package storage

import (
	"context"
	"time"
)

// TranslationSession represents a translation session in storage
type TranslationSession struct {
	ID              string     `json:"id"`
	BookTitle       string     `json:"book_title"`
	InputFile       string     `json:"input_file"`
	OutputFile      string     `json:"output_file"`
	SourceLanguage  string     `json:"source_language"`
	TargetLanguage  string     `json:"target_language"`
	Provider        string     `json:"provider"`
	Model           string     `json:"model"`
	Status          string     `json:"status"`
	PercentComplete float64    `json:"percent_complete"`
	CurrentChapter  int        `json:"current_chapter"`
	TotalChapters   int        `json:"total_chapters"`
	ItemsCompleted  int        `json:"items_completed"`
	ItemsFailed     int        `json:"items_failed"`
	ItemsTotal      int        `json:"items_total"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TranslationCache represents a cached translation
type TranslationCache struct {
	ID             string    `json:"id"`
	SourceText     string    `json:"source_text"`
	TargetText     string    `json:"target_text"`
	SourceLanguage string    `json:"source_language"`
	TargetLanguage string    `json:"target_language"`
	Provider       string    `json:"provider"`
	Model          string    `json:"model"`
	CreatedAt      time.Time `json:"created_at"`
	AccessCount    int       `json:"access_count"`
	LastAccessedAt time.Time `json:"last_accessed_at"`
}

// Storage interface defines the methods for persistence
type Storage interface {
	// Session management
	CreateSession(ctx context.Context, session *TranslationSession) error
	GetSession(ctx context.Context, sessionID string) (*TranslationSession, error)
	UpdateSession(ctx context.Context, session *TranslationSession) error
	ListSessions(ctx context.Context, limit, offset int) ([]*TranslationSession, error)
	DeleteSession(ctx context.Context, sessionID string) error

	// Translation cache
	GetCachedTranslation(ctx context.Context, sourceText, sourceLanguage, targetLanguage, provider, model string) (*TranslationCache, error)
	CacheTranslation(ctx context.Context, cache *TranslationCache) error
	CleanupOldCache(ctx context.Context, olderThan time.Duration) error

	// Statistics
	GetStatistics(ctx context.Context) (*Statistics, error)

	// Health check
	Ping(ctx context.Context) error

	// Close the storage connection
	Close() error
}

// Statistics represents translation statistics
type Statistics struct {
	TotalSessions      int64   `json:"total_sessions"`
	CompletedSessions  int64   `json:"completed_sessions"`
	FailedSessions     int64   `json:"failed_sessions"`
	InProgressSessions int64   `json:"in_progress_sessions"`
	TotalTranslations  int64   `json:"total_translations"`
	CacheHitRate       float64 `json:"cache_hit_rate"`
	AverageDuration    float64 `json:"average_duration_seconds"`
}

// Config represents storage configuration
type Config struct {
	Type     string `json:"type"` // "sqlite", "postgres", "redis"
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode"`

	// SQLCipher encryption key
	EncryptionKey string `json:"encryption_key,omitempty"`

	// Connection pool settings
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
}
