# Wave 10 real-use video evidence (§11.4.153 / §11.4.107 / §11.4.154 / §11.4.155)

Build: build/unified-translator-w10 (HEAD c2aa7c8). Path: real input -> real DeepSeek (deepseek-chat, in-process default, NOT -use-verifier) -> real output. Window-scoped asciinema->agg->ffmpeg @10fps. Prior helix_translate-* recordings NOT deleted.

| # | Feature | Video (Recordings/) | Frames | Real output proof |
|---|---------|---------------------|--------|-------------------|
| W10-1 | ES->EN reverse pair (non-EN source) | helix_translate-es-en-reverse-wave10-20260616_190325.mp4 | 44 | Spanish source -> English "This is a sample text in Spanish to test translation functionality..." (es_en.txt) |
| W10-2 | EN->AR (Arabic RTL, new script) | helix_translate-en-ar-rtl-wave10-20260616_190325.mp4 | 49 | 161 Arabic Unicode chars: محتويات / هذا نماذج نص إنجليزي... (en_ar.txt) |
| W10-3 | -temperature flag effect | helix_translate-temperature-flag-wave10-20260616_190325.mp4 | 59 | T=0.0 "destiné à tester" vs T=1.5 "pour tester" => flag HONORED, outputs differ (t00.txt/t15.txt) |
| W10-4 | -workers 4 multi-chapter EPUB | helix_translate-workers-parallel-epub-wave10-20260616_190325.mp4 | 51 | all 4 chapters EN->DE: Kapitel 1-4 content translated (multi_de2.epub) |
| W10-5 | TXT->DOCX cross-product | helix_translate-txt-to-docx-wave10-20260616_190325.mp4 | 45 | valid "Microsoft Word 2007+" docx, French in word/document.xml (out.docx) |

All confirmed: source != output (§11.4.107), no LLM error/empty/frozen frame, OCR-read of last frame matches real translated content.
