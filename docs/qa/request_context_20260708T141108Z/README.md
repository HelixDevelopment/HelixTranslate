# Handler Request Context Propagation Fix — Evidence

**Revision:** 1
**Last modified:** 2026-07-08T14:11:08Z

## Fix summary
Commit `98e0450` — `pkg/api/handler.go` + `batch_handlers.go` now propagate `c.Request.Context()` with bounded 5-minute timeout instead of unbounded `context.Background()`. Fixes client-disconnect cancellation gap (a stuck provider would hang the handler indefinitely).

## Evidence: regression guard test

### Test: `TestTranslateText_PropagatesRequestContextCancellation`
```
=== RUN   TestTranslateText_PropagatesRequestContextCancellation
--- PASS: TestTranslateText_PropagatesRequestContextCancellation (0.00s)
```
§11.4.135 regression guard: sends a request whose context is already cancelled (simulating client disconnect), asserts the handler respects the cancellation (returns quickly, does not hang on a stuck provider). A handler using `context.Background()` ignores the cancellation and the safety timer fires (FAIL).

**Run date:** 2026-07-08T14:11:08Z
**Test file:** `pkg/api/request_context_propagation_test.go`
**Commit:** `98e0450`
**Audit finding class:** HIGH (handler request-context misuse — 6 sites at handler.go:277/438/575/924 + batch_handlers.go:145/275 now fixed)
