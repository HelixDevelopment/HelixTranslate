# Version-Const + SQLite Deterministic Guard — Evidence

**Revision:** 1
**Last modified:** 2026-07-08T14:11:08Z

## Fix summary
Commit `989f5d1` — `pkg/version/app.go` AppVersion 2.3.0→2.3.1 (was RED — VERSION bumped at 236bac8 without the const). + deterministic SQLite single-writer regression guard.

## Evidence: all version + SQLite guards

### Test: `TestAppVersionMatchesVERSIONFile`
```
=== RUN   TestAppVersionMatchesVERSIONFile
--- PASS: TestAppVersionMatchesVERSIONFile (0.00s)
```
Guards the in-code const matches the VERSION file (the single-source-of-truth for the version number).

### Test: `TestSQLite_DefaultsToSingleWriterConnection`
```
=== RUN   TestSQLite_DefaultsToSingleWriterConnection
--- PASS: TestSQLite_DefaultsToSingleWriterConnection (0.00s)
```
Deterministic §11.4.135 regression guard for the SQLITE_BUSY fix.

**Run date:** 2026-07-08T14:11:08Z
**Test files:** `pkg/version/app_test.go`, `pkg/storage/sqlite_concurrency_guard_test.go`
**Commit:** `989f5d1`
