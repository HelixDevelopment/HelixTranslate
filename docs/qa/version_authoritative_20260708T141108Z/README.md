# API Server Authoritative Version Fix — Evidence

**Revision:** 1
**Last modified:** 2026-07-08T14:11:08Z

## Fix summary
Commit `62cbb28` — `pkg/api/handler.go` `/health` endpoint now reports `version.AppVersion` instead of hardcoded "1.0.0"/"3.0.0" literals. Commit `98e0450` — handlers propagate `c.Request.Context()` with bounded timeout.

## Evidence: regression guard test

### Test: `TestHealthCheckReportsAuthoritativeVersion`
```
=== RUN   TestHealthCheckReportsAuthoritativeVersion
--- PASS: TestHealthCheckReportsAuthoritativeVersion (0.00s)
```
§11.4.135 deterministic guard: asserts `/health` returns `version.AppVersion` (the authoritative VERSION-bound constant). Reverts to a literal → FAILs (literal != version.AppVersion once VERSION moves).

### Test: `TestAppVersionMatchesVERSIONFile`
```
=== RUN   TestAppVersionMatchesVERSIONFile
--- PASS: TestAppVersionMatchesVERSIONFile (0.00s)
```
Guards the `pkg/version/app.go` const matches the `VERSION` file.

**Run date:** 2026-07-08T14:11:08Z
**Test files:** `pkg/api/handler_test.go`, `pkg/version/app_test.go`
**Commit:** `62cbb28` (version fix), `989f5d1` (guard)
