# E2E Proof — DeepSeek allowlist un-staled (v4 models accepted)

**Date:** 2026-06-14
**Constitution:** §11.4.150 deep-research-per-issue, §11.4.99 latest-source, §11.4.123 rock-solid proof, §11.4.115 reproduce-first, §11.4.135 regression guard, §11.4.122 no silent removal

## Defect
`ValidModels[ProviderDeepSeek]` listed only legacy {deepseek-chat, deepseek-coder}.
`deepseek.go` HARD-REJECTS any unlisted model, so a user requesting the current
DeepSeek flagship was rejected with "model 'deepseek-v4-pro' is not valid for
DeepSeek" — a real user-impacting staleness defect.

## Authoritative source (not memory — §11.4.99)
Live DeepSeek /models endpoint (verified 2026-06-14) — see live_models.txt:
```
deepseek-v4-flash
deepseek-v4-pro
```
deepseek-v4-flash proven to genuinely translate this session
("Good morning" -> "Добро јутро", finish_reason=stop).

## Fix
Added deepseek-v4-flash + deepseek-v4-pro to the allowlist (ONLY the two the live
endpoint confirmed — deepseek-reasoner was NOT added since /models did not list it,
per §11.4.6 no-guessing). Kept legacy deepseek-chat/coder (still work — §11.4.122).

## Real-system proof (this directory)
`out.txt` — English PDF → real DeepSeek **deepseek-v4-flash** (the now-unblocked
model) → Serbian Cyrillic, 136 Cyrillic chars:
> Храбри витез јахао је преко тихе долине у зору. ...

## Regression guard (§11.4.135)
pkg/translator/llm/deepseek_v4_models_test.go asserts v4-flash + v4-pro accepted
and deepseek-chat backcompat retained. RED proven pre-fix (both v4 rejected:
"not valid for DeepSeek. Valid models: [deepseek-chat deepseek-coder]"); GREEN
post-fix. Mutation = pre-fix allowlist.
