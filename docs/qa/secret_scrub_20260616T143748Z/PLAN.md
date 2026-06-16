# Secret-Scrub PLAN — leaked SSH password `WhiteSnake<REDACTED-§11.4.10>`

**Revision:** 1
**Last modified:** 2026-06-16T14:37:48Z
**Authority:** §11.4.10 (credentials-handling) + §11.4.10.A (pre-store credential leak audit) + §11.4.6 (no-guessing — captured `git grep`/`git log` evidence only) + §11.4.30 (no versioned build artifacts)
**Stream:** BACKGROUND read-only investigation. NON-COMMITTING. A separate committer agent is the sole main-checkout committer (§11.4.84). This file is the only write; nothing staged/committed by this stream.
**Repo HEAD at investigation:** `6a1aa8c334cd38ee1dea2a59cb10aa188c0b387f`

---

## 0. Scope + status

- **In scope:** scrub the leaked SSH password literal `WhiteSnake<REDACTED-§11.4.10>` from every TRACKED file (tree-level scrub).
- **Out of scope (operator-confirmed):** history scrub — operator confirmed the credential is already rotated/replaced; history count recorded for §11.4.10.A awareness only.
- **NOT secrets (kept):** hostname `thinker.local` and username `milosvasic` are infrastructure identifiers, NOT credentials/secrets per §11.4.10 — they are NOT scrubbed. Documented for awareness (host+user pairing reduces blast radius if ever re-leaked, but neither is a secret).

---

## 1. TRACKED occurrences of `WhiteSnake<REDACTED-§11.4.10>` (captured `git grep -n`)

### 1a. Text/script/doc occurrences (scrubbable in-place)

| # | path:line | content class | fix classification |
|---|---|---|---|
| 1 | `Documentation/PRODUCTION_COMPLETION_REPORT.md:29` | doc — "Specified credentials" line | `<redacted-per-§11.4.10>` |
| 2 | `Documentation/PRODUCTION_COMPLETION_REPORT.md:64` | doc — example `--password WhiteSnake<REDACTED-§11.4.10> \` | `<redacted-per-§11.4.10>` |
| 3 | `Documentation/PRODUCTION_DOCUMENTATION.md:71` | doc — example `--password WhiteSnake<REDACTED-§11.4.10> \` | `<redacted-per-§11.4.10>` |
| 4 | `Documentation/PRODUCTION_DOCUMENTATION.md:139` | doc — Go struct literal `Password: "WhiteSnake<REDACTED-§11.4.10>"` | `<redacted-per-§11.4.10>` |
| 5 | `SSH_TRANSLATION_PROGRESS.html:186` | doc (HTML export) — `(milosvasic/WhiteSnake<REDACTED-§11.4.10>)` | `<redacted-per-§11.4.10>` (regen from .md per §11.4.65) |
| 6 | `SSH_TRANSLATION_PROGRESS.md:7` | doc — `(milosvasic/WhiteSnake<REDACTED-§11.4.10>)` | `<redacted-per-§11.4.10>` |
| 7 | `demo_ssh_translation.sh:152` | script — echo of example command | `${SSH_WORKER_PASSWORD}` env-ref |
| 8 | `demo_ssh_translation.sh:166` | script — actual `-password="WhiteSnake<REDACTED-§11.4.10>" \` | `${SSH_WORKER_PASSWORD}` env-ref |
| 9 | `docs/final_translation_report.html:263` | doc (HTML export) — `(milosvasic/WhiteSnake<REDACTED-§11.4.10>)` | `<redacted-per-§11.4.10>` (regen from .md) |
| 10 | `docs/final_translation_report.html:356` | doc (HTML export) — full example cmd line | `<redacted-per-§11.4.10>` (regen from .md) |
| 11 | `docs/final_translation_report.md:15` | doc — `(milosvasic/WhiteSnake<REDACTED-§11.4.10>)` | `<redacted-per-§11.4.10>` |
| 12 | `docs/final_translation_report.md:78` | doc — full example cmd line | `<redacted-per-§11.4.10>` |
| 13 | `internal/scripts/final_test.sh:32` | script — `-password WhiteSnake<REDACTED-§11.4.10>` | `${SSH_WORKER_PASSWORD}` env-ref |
| 14 | `internal/scripts/translate_final_clean.sh:32` | script — `-password WhiteSnake<REDACTED-§11.4.10>` | `${SSH_WORKER_PASSWORD}` env-ref |
| 15 | `internal/working/test_ssh.sh:5` | script — expect `send "WhiteSnake<REDACTED-§11.4.10>\r"` | `${SSH_WORKER_PASSWORD}` env-ref (expect: `send "$env(SSH_WORKER_PASSWORD)\r"`) |
| 16 | `internal/working/test_translation_new.sh:17` | script — `-password WhiteSnake<REDACTED-§11.4.10>` | `${SSH_WORKER_PASSWORD}` env-ref |
| 17 | `run_translation_demo.sh:9` | script — `-password="WhiteSnake<REDACTED-§11.4.10>" \` | `${SSH_WORKER_PASSWORD}` env-ref |
| 18 | `scripts/ebook_translation_workflow.sh:13` | script — `REMOTE_PASS="WhiteSnake<REDACTED-§11.4.10>"` | `REMOTE_PASS="${SSH_WORKER_PASSWORD}"` env-ref |
| 19 | `scripts/ebook_translation_workflow.sh:76` | script — help text `(default: WhiteSnake<REDACTED-§11.4.10>)` | `<redacted-per-§11.4.10>` (drop the leaked default from help) |
| 20 | `scripts/ebook_translation_workflow.sh:83` | script — usage example `--password WhiteSnake<REDACTED-§11.4.10>` | `<redacted-per-§11.4.10>` |

### 1b. TRACKED BINARY occurrences (secret BAKED INTO compiled artifacts) — DUAL violation

`git grep` reported `Binary file ... matches` for the embedded literal. Confirmed tracked via `git ls-files`:

| # | path | violation |
|---|---|---|
| B1 | `build/ebook-translator` | §11.4.10 embedded secret + §11.4.30 tracked build artifact |
| B2 | `download-model` | §11.4.10 embedded secret + §11.4.30 tracked build artifact |
| B3 | `ebook-translator` | §11.4.10 embedded secret + §11.4.30 tracked build artifact |
| B4 | `test-enhanced-conversion` | §11.4.10 embedded secret + §11.4.30 tracked build artifact |
| B5 | `test-model` | §11.4.10 embedded secret + §11.4.30 tracked build artifact |
| B6 | `test-simple-translation` | §11.4.10 embedded secret + §11.4.30 tracked build artifact |

**Fix for B1–B6:** `git rm --cached` the binaries (un-track) + add to `.gitignore` (they are §11.4.30 build derivatives, regenerable via `make build` per §11.4.77). Editing the binaries in place is NOT viable — the secret is compiled in; the binaries MUST be un-tracked. Rebuilding from the (scrubbed) source then produces secret-free artifacts that stay gitignored.

---

## 2. History count (§11.4.10.A awareness only — history scrub NOT in scope)

Captured: `git log -S WhiteSnake<REDACTED-§11.4.10> --oneline --all | wc -l` → **17 commits** carry the literal across history. Operator confirmed the credential is rotated/replaced, so per §11.4.10.A the action is documented-only (no `--allow-force-push` history rewrite, consistent with §11.4.113 absolute no-force-push). RECORD for awareness; rotation already done.

---

## 3. OTHER plausibly-leaked literals in the affected files

Scanned all 20 text-file occurrences' parent files for `password|passwd|secret|token|api[_-]?key|-p <literal>` (excluding the known token). **Findings:**

- **No other hardcoded secret VALUE found.** Every other `password`-shaped hit is descriptive prose ("Password-based authentication", "SSH password transmission"), an `expect "password:"` prompt match, a CLI flag name (`--password|-p`), or `"max_tokens": 4096` (a model param, not a secret). None embeds a credential value.
- `WhiteSnake<REDACTED-§11.4.10>` is the SOLE leaked secret literal in the tracked tree.
- The `WhiteSnake` matches in `docs/research/helix_qa/workspace-*/...json` are FALSE POSITIVES — those JSON files contain copies of THIS constitution's §11.4.10.A text mentioning the token NAME; confirmed `git grep -c "WhiteSnake<REDACTED-§11.4.10>"` returns ZERO `8587` in research JSON (only the prose token-name). NOT a leak, NOT scrubbed.

---

## 4. Per-file fix classification summary

- **Scripts (configs/runnable) → `${SSH_WORKER_PASSWORD}` env-ref:** occurrences 7, 8, 13, 14, 15, 16, 17, 18. Matches the canonical env var (`SSH_WORKER_PASSWORD`, per project CLAUDE.md / `config.distributed.*.json` scrub style). Tcl/expect variant (15): `send "$env(SSH_WORKER_PASSWORD)\r"`.
- **Docs / examples / help-text → `<redacted-per-§11.4.10>`:** occurrences 1–6, 9–12, 19, 20. HTML exports (5, 9, 10) MUST be regenerated from the scrubbed `.md` via the §11.4.65 export pipeline, not hand-edited, to keep md/html in sync.
- **Tracked binaries → un-track + gitignore + rebuild:** B1–B6 (`git rm --cached` + `.gitignore` entries; regenerate via `make build`).

---

## 5. Pre-push credential-pattern grep coverage (§11.4.10.A clause 5)

**Captured evidence:**
- `.gitignore` DOES cover the secret-FILE class: `.env`, `.env.*` (with `.example` allow), `secrets.*`, `keys.*`, `config_with_keys.json`, `api_keys.json`, `**/qwen_credentials.json`, `**/oauth_creds.json`. — adequate for files.
- **GAP (clause 5):** NO active git hook is installed — `.git/hooks/` contains only the default `*.sample` files (no `pre-push`, `pre-commit`, `commit-msg`); `git ls-files` shows NO tracked `scripts/git_hooks/*`, NO `install_git_hooks.sh`. There is therefore **NO mechanical credential-pattern grep on commit/push**. A hardcoded literal like `WhiteSnake<REDACTED-§11.4.10>` (an inline VALUE, not a secret FILE) would NOT be caught by `.gitignore` and is NOT caught by any hook.
- **`.gitignore` does NOT cover the tracked-binary class** that carried the embedded secret (`/build/`, root `ebook-translator`, `download-model`, `test-*` are not ignored) — §11.4.30 gap that is the root cause of B1–B6.

**Required remediation (for the committer window):**
1. Add a tracked pre-push (and pre-commit) hook + `install_git_hooks.sh` per §11.4.75 Layers 1+3, whose credential-pattern grep rejects inline secret-shaped literals (at minimum the known token class + generic `-password[ =]"?[^${]` inline-value pattern), so the escaped class (inline value, not a secret file) is caught in the SAME commit per §11.4.10.A clause 5.
2. Add `.gitignore` entries for the build artifacts (`/build/`, `/ebook-translator`, `/download-model`, `/test-enhanced-conversion`, `/test-model`, `/test-simple-translation`) to close the §11.4.30 root cause.

---

## 6. Evidence appendix (commands run, read-only)

- `git rev-parse HEAD` → `6a1aa8c…`
- `git grep -n WhiteSnake<REDACTED-§11.4.10>` → 20 text occurrences + 6 `Binary file … matches` (table §1).
- `git log -S WhiteSnake<REDACTED-§11.4.10> --oneline --all | wc -l` → `17` (§2).
- `git grep -c "WhiteSnake<REDACTED-§11.4.10>" -- 'docs/research/**'` → no `8587` (research JSON clean, §3).
- `git ls-files | grep -E '^build/ebook-translator$|…'` → all 6 binaries tracked (§1b).
- `ls -la .git/hooks/` → only `*.sample` defaults; `git ls-files | grep git_hooks` → empty (§5 gap).
- No secret VALUE printed anywhere beyond the already-known-in-context token.
