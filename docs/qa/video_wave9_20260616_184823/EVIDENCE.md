# helix_translate WAVE9 §11.4.153 real-use video evidence — 20260616_184823

NON-COMMITTING parallel stream (wave9). Real input -> real DeepSeek/novita (in-process bridge, LLMSVERIFIER_API_URL unset) -> real output. §11.4.107 + §11.4.154 + §11.4.155.

## Features CONFIRMED (3)

### 1. REST POST /api/v1/translate/batch  (server-w9 :8091 TLS, in-process bridge)
- Real multi-text translate. 3 EN -> SR: "The river flows to the sea." -> "Река тече до мора."; "Books open windows to the world." -> "Knjige otvaraju prozore u svet."; "Time heals all wounds." -> "Vrijeme leči sve rane." 0 errors. provider resolved: llm-novita.
- Video: /Volumes/T7/Downloads/Recordings/helix_translate-rest-batch-version-wave9-20260616_184823.mp4 (frames 790, 7.52s, early!=late)

### 2. REST GET /api/v1/version  (server-w9 :8091)
- Real substantive data: codebase_version 2.3.0, git_commit c2aa7c841c90df2ac1519c147f61bb82a9614963, go_version go1.26.2, build_time, components map. NOT a bare status.
- Video: same clip as #1 (helix_translate-rest-batch-version-wave9-20260616_184823.mp4)

### 3. preparation -> translate two-stage pipeline  (preptrans-w9, in-process bridge)
- Source: The Lighthouse Keeper (EN) -> Spanish. Stage1 real DeepSeek analysis: content_type "short story", genre "literary fiction", 2 characters (Captain Marlowe + speech pattern, Eli), 3 untranslatable terms (Stormcrag Point, Captain Marlowe, Eli), 5 key themes, 1974 tokens. Stage2: analysis informs translation — output EPUB preserved exactly those 3 untranslatable names in the Spanish text ("capitán Marlowe", "Stormcrag Point", "Eli"). Output 3452B EPUB + 13495B analysis.json.
- Real Spanish output sample: "La luz en la tempestad ... El viejo capitán Marlowe había sido farero de Stormcrag Point durante cuarenta años..."
- Video: /Volumes/T7/Downloads/Recordings/helix_translate-prep-translate-two-stage-wave9-20260616_184823.mp4 (frames 790, 10.68s, early!=late)

## NOT confirmable / findings
- pkg/api/server.go Server (/api/translate, /api/batch, /api/languages): NO cmd entry instantiates api.NewServer — unwired alt server, no runnable binary. Not confirmed (did NOT fabricate a runner, §11.4.124/§11.4.6).
- /api/v1/providers: static hardcoded list (openai/anthropic/zhipu/deepseek), NOT real upstream data — per W9 item 5 criterion ("substantive real data, not a bare status") NOT confirmed as substantive.

## Frames (early!=late, not frozen §11.4.107)
- prep_early 52873d28 != prep_late b89b22df
- rest_early 85cbba74 != rest_late f4127073
- OCR dumps: ocr_rest_late.txt, ocr_prep_late.txt
