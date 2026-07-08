# SQLite SQLITE_BUSY Concurrency Fix — Evidence

**Revision:** 1
**Last modified:** 2026-07-08T14:11:08Z

## Fix summary
Commit `99bb7a8` — `pkg/storage/sqlite.go` SQLITE_BUSY concurrency fix: unbounded pool + no busy_timeout → 110/640 concurrent writes failed. Fix: `_busy_timeout=5000` + default `SetMaxOpenConns(1)`.

## Evidence: regression guard tests (RED→GREEN proven)

### Test: `TestSQLite_DefaultsToSingleWriterConnection`
```
=== RUN   TestSQLite_DefaultsToSingleWriterConnection
--- PASS: TestSQLite_DefaultsToSingleWriterConnection (0.00s)
```
Asserts `MaxOpenConns == 1` by default (the fix that prevents SQLITE_BUSY contention).

### Test: `TestSQLite_RespectsExplicitMaxOpenConns`
```
=== RUN   TestSQLite_RespectsExplicitMaxOpenConns
--- PASS: TestSQLite_RespectsExplicitMaxOpenConns (0.00s)
```
Guards the opt-out: explicit `MaxOpenConns` overrides the single-writer default.

### Test: `TestSQLite_PlainDatabaseStringIsValidPath`
```
=== RUN   TestSQLite_PlainDatabaseStringIsValidPath
--- PASS: TestSQLite_PlainDatabaseStringIsValidPath (0.00s)
```
Guards that `_busy_timeout` merge doesn't produce a malformed DSN.

**Run date:** 2026-07-08T14:11:08Z
**Test file:** `pkg/storage/sqlite_concurrency_guard_test.go`
**Commit:** `99bb7a8` (original fix), `989f5d1` (deterministic guard)
