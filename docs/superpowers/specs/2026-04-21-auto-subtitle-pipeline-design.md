# Auto Subtitle Pipeline Design

**Date:** 2026-04-21  
**Status:** Approved

## Overview

Integrate whisperx_Sub into coding-lux so that every downloaded video automatically receives three subtitle files: Chinese (zh), English (en), and Chinese+Pinyin bilingual. The pipeline runs asynchronously after download so the user's CLI session is not blocked.

---

## Architecture

```
coding-lux (Go)
  └─ downloader.go: Download()
       └─ goroutine → exec.Command("python subtitle_pipeline.py <video_path>")

whisperx_Sub/ (Python, cloned to ~/Desktop/whisperx_Sub)
  ├─ config.py              ← modified: MiniMax API, task="all"
  ├─ subtitle_pipeline.py  ← NEW: orchestration entry point
  └─ pinyin_processor.py   ← NEW: pypinyin post-processor
```

**Data flow:**
1. coding-lux finishes writing `mergedFilePath` (the final video file)
2. Goroutine fires: `python .../subtitle_pipeline.py <mergedFilePath>`
3. whisperx transcribes audio → Chinese .srt
4. MiniMax API translates → English .srt  
5. pinyin_processor reads Chinese .srt → bilingual .srt

**Output (3 files alongside the video):**
- `<name>_zh.srt` — Chinese transcription
- `<name>_en.srt` — English translation
- `<name>_pinyin.srt` — Chinese + pinyin, e.g.:
  ```
  1
  00:00:01,000 --> 00:00:03,000
  你好世界
  nǐ hǎo shì jiè
  ```

---

## Components

### 1. coding-lux: `downloader/downloader.go`

**Change:** In `Download()`, after the final merged file is written and before returning `nil`, launch a goroutine:

```go
go func(videoPath string) {
    cmd := exec.Command("python3",
        "/Users/pengby/Desktop/whisperx_Sub/subtitle_pipeline.py",
        videoPath)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    _ = cmd.Run()
}(mergedFilePath)
```

Insert at two return points:
- Single-part: before `return nil` at line ~685
- Multi-part: before `return nil` at end of function (~line 760)

Add imports: `"os/exec"` (already has `"os"`).

### 2. `subtitle_pipeline.py` (new file in whisperx_Sub/)

- Accept video file path via `sys.argv[1]`
- Derive output stem from video filename
- Call whisperx_Sub transcription → `<stem>_zh.srt`
- Call whisperx_Sub translation via MiniMax → `<stem>_en.srt`
- Call `pinyin_processor.generate_pinyin_srt()` → `<stem>_pinyin.srt`
- Write all three files to the same directory as the input video
- Log to stderr; failures are non-fatal (video is already saved)

### 3. `pinyin_processor.py` (new file in whisperx_Sub/)

- `generate_pinyin_srt(zh_srt_path: str, out_path: str) -> None`
- Parse .srt blocks (index, timestamp, text lines)
- For each text line, call `pypinyin.lazy_pinyin(line, style=Style.TONE)` and join with spaces
- Write new .srt where each block's text is `原文\n拼音`

### 4. `config.py` (modified)

```python
task = "all"
is_using_local_model = False
base_url = "https://api.minimax.chat/v1"
api_key = ""               # loaded from env var MINIMAX_API_KEY at runtime
translation_model_name = "abab6.5s-chat"
```

API key is **never hardcoded**. `subtitle_pipeline.py` loads it from env and passes it, or sets `MINIMAX_API_KEY` before calling.

---

## Environment Setup

- Python 3.10 conda environment: `whisperx_env`
- Dependencies: `whisperx`, `pypinyin`, `openai`, existing `requirements.txt`
- MINIMAX_API_KEY stored in `~/.zshrc` or `.env` (git-ignored)
- ffmpeg required (for audio extraction from video)

---

## Error Handling

- Subtitle failure never fails the download; goroutine errors are logged to stderr only
- If `MINIMAX_API_KEY` is unset, translation step is skipped with a warning; Chinese .srt is still generated
- If whisperx fails (GPU OOM, etc.), all subtitle steps are skipped and error is logged

---

## Out of Scope

- Burning subtitles into video
- MKV soft-subtitle embedding
- Subtitle for audio-only downloads
