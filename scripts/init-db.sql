-- init-db.sql — PostgreSQL initialization for the HelixTranslate nezha stack.
--
-- Mounted into the postgres container at
-- /docker-entrypoint-initdb.d/ (compose.nezha.yml). The official postgres
-- image runs every *.sql in that directory exactly once, on a brand-new data
-- volume, against ${POSTGRES_DB}.
--
-- The application (pkg/storage/postgres.go initSchema) creates these same
-- tables idempotently on first connect; this script mirrors that schema so a
-- fresh DB volume is ready even before the first app connection, and is
-- intentionally idempotent (CREATE ... IF NOT EXISTS) so it never conflicts
-- with the app's own DDL. Keep this file in sync with pkg/storage/postgres.go.

-- Useful extensions (no-op if already present).
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS translation_sessions (
    id TEXT PRIMARY KEY,
    book_title TEXT NOT NULL,
    input_file TEXT NOT NULL,
    output_file TEXT,
    source_language TEXT NOT NULL,
    target_language TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    status TEXT NOT NULL,
    percent_complete REAL DEFAULT 0,
    current_chapter INTEGER DEFAULT 0,
    total_chapters INTEGER DEFAULT 0,
    items_completed INTEGER DEFAULT 0,
    items_failed INTEGER DEFAULT 0,
    items_total INTEGER DEFAULT 0,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_status ON translation_sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON translation_sessions(created_at DESC);

CREATE TABLE IF NOT EXISTS translation_cache (
    id TEXT PRIMARY KEY,
    source_text TEXT NOT NULL,
    target_text TEXT NOT NULL,
    source_language TEXT NOT NULL,
    target_language TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    lookup_hash TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL,
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cache_lookup ON translation_cache(source_text, source_language, target_language, provider, model);
CREATE INDEX IF NOT EXISTS idx_cache_last_accessed ON translation_cache(last_accessed_at);
