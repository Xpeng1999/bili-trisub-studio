# Auto Subtitle Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** After every coding-lux video download, automatically generate three `.srt` files (Chinese, English, Chinese+Pinyin) via whisperx_Sub and pypinyin.

**Architecture:** coding-lux fires an async goroutine post-download that calls a shell wrapper; the wrapper activates the `whisperx_env` conda environment and runs `subtitle_pipeline.py`, which chains whisperx transcription → MiniMax translation → pypinyin processing.

**Tech Stack:** Go (`os/exec`), Python 3.10 (whisperx, openai SDK, pypinyin), conda `whisperx_env`, MiniMax API (`https://api.minimax.chat/v1`, model `abab6.5s-chat`), env var `MINIMAX_API_KEY`.

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `~/Desktop/whisperx_Sub/` | Cloned repo (whisperx_Sub) |
| Modify | `~/Desktop/whisperx_Sub/config.py` | MiniMax base_url + model |
| Create | `~/Desktop/whisperx_Sub/pinyin_processor.py` | SRT → bilingual Chinese+pinyin SRT |
| Create | `~/Desktop/whisperx_Sub/tests/test_pinyin_processor.py` | Unit tests for pinyin_processor |
| Create | `~/Desktop/whisperx_Sub/subtitle_pipeline.py` | Orchestration entry point |
| Create | `~/Desktop/whisperx_Sub/run_pipeline.sh` | Shell wrapper that activates conda env |
| Modify | `~/Desktop/coding-lux/downloader/downloader.go` | Launch goroutine after download |

---

## Task 1: Clone repo and create conda environment

**Files:**
- Create: `~/Desktop/whisperx_Sub/` (via git clone)
- Create: conda env `whisperx_env`

- [ ] **Step 1: Clone whisperx_Sub**

```bash
cd ~/Desktop
git clone https://github.com/hiddenblue/whisperx_Sub.git
```

Expected: directory `~/Desktop/whisperx_Sub/` appears with `config.py`, `voice2sub.py`, etc.

- [ ] **Step 2: Create Python 3.10 conda environment**

```bash
conda create -n whisperx_env python=3.10 -y
```

Expected: `Successfully created environment whisperx_env`

- [ ] **Step 3: Install whisperx_Sub dependencies**

```bash
conda run -n whisperx_env pip install -r ~/Desktop/whisperx_Sub/requirements.txt
```

Expected: all packages install without error. (torch download may take several minutes.)

- [ ] **Step 4: Install pypinyin**

```bash
conda run -n whisperx_env pip install pypinyin
```

Expected: `Successfully installed pypinyin-...`

- [ ] **Step 5: Verify Python path**

```bash
ls /Users/pengby/miniconda3/envs/whisperx_env/bin/python3
```

Expected: file exists at that path.

- [ ] **Step 6: Set MINIMAX_API_KEY in shell**

Add to `~/.zshrc` (use your actual key — do NOT commit this file):

```bash
echo 'export MINIMAX_API_KEY="<your-key-here>"' >> ~/.zshrc
source ~/.zshrc
```

Verify:
```bash
echo $MINIMAX_API_KEY
```

Expected: key is printed (not empty).

- [ ] **Step 7: Commit task marker**

```bash
cd ~/Desktop/coding-lux
git add -A
git commit -m "chore: whisperx_Sub cloned and env created"
```

---

## Task 2: Create pinyin_processor.py with TDD

**Files:**
- Create: `~/Desktop/whisperx_Sub/pinyin_processor.py`
- Create: `~/Desktop/whisperx_Sub/tests/__init__.py`
- Create: `~/Desktop/whisperx_Sub/tests/test_pinyin_processor.py`

- [ ] **Step 1: Create tests directory**

```bash
mkdir -p ~/Desktop/whisperx_Sub/tests
touch ~/Desktop/whisperx_Sub/tests/__init__.py
```

- [ ] **Step 2: Write failing tests**

Create `~/Desktop/whisperx_Sub/tests/test_pinyin_processor.py`:

```python
import os
import sys
import tempfile
import pytest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))
from pinyin_processor import parse_srt, add_pinyin_line, generate_pinyin_srt


def test_parse_srt_single_block():
    content = "1\n00:00:01,000 --> 00:00:03,000\n你好世界"
    blocks = parse_srt(content)
    assert len(blocks) == 1
    assert blocks[0]['index'] == '1'
    assert blocks[0]['timestamp'] == '00:00:01,000 --> 00:00:03,000'
    assert blocks[0]['text'] == '你好世界'


def test_parse_srt_multiple_blocks():
    content = "1\n00:00:01,000 --> 00:00:03,000\n你好世界\n\n2\n00:00:04,000 --> 00:00:06,000\n学习中文"
    blocks = parse_srt(content)
    assert len(blocks) == 2
    assert blocks[1]['text'] == '学习中文'


def test_add_pinyin_line_format():
    result = add_pinyin_line('你好世界')
    lines = result.split('\n')
    assert len(lines) == 2
    assert lines[0] == '你好世界'


def test_add_pinyin_line_contains_pinyin():
    result = add_pinyin_line('你好')
    lines = result.split('\n')
    # pypinyin with TONE style: nǐ hǎo
    assert 'n' in lines[1]  # at minimum starts with n for 你
    assert len(lines[1]) > 0


def test_generate_pinyin_srt_creates_file():
    srt_content = "1\n00:00:01,000 --> 00:00:03,000\n你好世界\n\n2\n00:00:04,000 --> 00:00:06,000\n学习中文"
    with tempfile.NamedTemporaryFile(mode='w', suffix='.srt', delete=False,
                                     encoding='utf-8') as f:
        f.write(srt_content)
        zh_path = f.name
    out_path = zh_path.replace('.srt', '_pinyin.srt')
    try:
        generate_pinyin_srt(zh_path, out_path)
        assert os.path.exists(out_path)
        with open(out_path, encoding='utf-8') as f:
            result = f.read()
        assert '你好世界' in result
        assert '学习中文' in result
        # Both blocks preserved
        assert '00:00:01,000 --> 00:00:03,000' in result
        assert '00:00:04,000 --> 00:00:06,000' in result
    finally:
        os.unlink(zh_path)
        if os.path.exists(out_path):
            os.unlink(out_path)
```

- [ ] **Step 3: Run tests — verify they fail**

```bash
cd ~/Desktop/whisperx_Sub
conda run -n whisperx_env python -m pytest tests/test_pinyin_processor.py -v
```

Expected: `ERROR` or `ImportError: cannot import name 'parse_srt'`

- [ ] **Step 4: Implement pinyin_processor.py**

Create `~/Desktop/whisperx_Sub/pinyin_processor.py`:

```python
import re
from pypinyin import lazy_pinyin, Style


def parse_srt(content: str) -> list:
    blocks = []
    for entry in re.split(r'\n\n+', content.strip()):
        lines = entry.strip().split('\n')
        if len(lines) < 3:
            continue
        blocks.append({
            'index': lines[0],
            'timestamp': lines[1],
            'text': '\n'.join(lines[2:]),
        })
    return blocks


def add_pinyin_line(text: str) -> str:
    pinyin = ' '.join(lazy_pinyin(text, style=Style.TONE))
    return f"{text}\n{pinyin}"


def generate_pinyin_srt(zh_srt_path: str, out_path: str) -> None:
    with open(zh_srt_path, encoding='utf-8') as f:
        content = f.read()
    blocks = parse_srt(content)
    output_blocks = []
    for block in blocks:
        new_text = add_pinyin_line(block['text'])
        output_blocks.append(f"{block['index']}\n{block['timestamp']}\n{new_text}")
    with open(out_path, 'w', encoding='utf-8') as f:
        f.write('\n\n'.join(output_blocks))
```

- [ ] **Step 5: Run tests — verify they pass**

```bash
cd ~/Desktop/whisperx_Sub
conda run -n whisperx_env python -m pytest tests/test_pinyin_processor.py -v
```

Expected:
```
test_pinyin_processor.py::test_parse_srt_single_block PASSED
test_pinyin_processor.py::test_parse_srt_multiple_blocks PASSED
test_pinyin_processor.py::test_add_pinyin_line_format PASSED
test_pinyin_processor.py::test_add_pinyin_line_contains_pinyin PASSED
test_pinyin_processor.py::test_generate_pinyin_srt_creates_file PASSED
5 passed
```

- [ ] **Step 6: Commit**

```bash
cd ~/Desktop/whisperx_Sub
git add pinyin_processor.py tests/
git commit -m "feat: add pinyin_processor with SRT bilingual generation"
```

---

## Task 3: Modify config.py for MiniMax

**Files:**
- Modify: `~/Desktop/whisperx_Sub/config.py`

- [ ] **Step 1: Read current config.py**

```bash
cat ~/Desktop/whisperx_Sub/config.py
```

- [ ] **Step 2: Update config.py**

Replace the `base_url`, `api_key`, and `translation_model_name` lines. The full updated file:

```python
import os

# task type: "transcribe" = transcription only, "all" = transcription + translation
task = "all"

# output and temp directory path
output_dir = "./output"
temp_dir = "./temp"
output_format = "all"

# audio or video file path (overridden by subtitle_pipeline.py at runtime)
audio_file = ""

# subtitle translation parameters
is_using_local_model = False
base_url = "https://api.minimax.chat/v1"
api_key = os.environ.get("MINIMAX_API_KEY", "")
translation_model_name = "abab6.5s-chat"
translation_prompt = ""
srt_file_name = ""

# rePunctuation parameters
WORDS_NUM_LIMITS = 12
```

- [ ] **Step 3: Verify key loads from env**

```bash
conda run -n whisperx_env python3 -c "
import os; os.environ['MINIMAX_API_KEY']='test-key'
import config
print('api_key:', config.api_key)
print('base_url:', config.base_url)
print('model:', config.translation_model_name)
"
```

Expected:
```
api_key: test-key
base_url: https://api.minimax.chat/v1
model: abab6.5s-chat
```

- [ ] **Step 4: Commit**

```bash
cd ~/Desktop/whisperx_Sub
git add config.py
git commit -m "config: switch to MiniMax API, load key from env"
```

---

## Task 4: Create subtitle_pipeline.py

**Files:**
- Create: `~/Desktop/whisperx_Sub/subtitle_pipeline.py`

- [ ] **Step 1: Create subtitle_pipeline.py**

```python
#!/usr/bin/env python3
"""
subtitle_pipeline.py — called by coding-lux after video download.

Usage: python subtitle_pipeline.py <video_path>

Writes alongside the video:
  <stem>_zh.srt      Chinese transcription
  <stem>_en.srt      English translation  (skipped if MINIMAX_API_KEY unset)
  <stem>_pinyin.srt  Chinese + pinyin bilingual
"""

import sys
import os
import shutil
import tempfile
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
sys.path.insert(0, str(SCRIPT_DIR))

import config
from voice2sub import sub_transcribe, sub_align
from srt_util import srt_reader, srt_writer, convert_vector_to_Sub
from LLM_api import OPENAI_General_Interface
from pinyin_processor import generate_pinyin_srt


def _log(msg: str) -> None:
    print(f"[subtitle] {msg}", file=sys.stderr, flush=True)


def run(video_path: str) -> None:
    video = Path(video_path).resolve()
    if not video.exists():
        _log(f"ERROR: file not found: {video}")
        return

    stem = video.stem
    out_dir = video.parent
    zh_srt = out_dir / f"{stem}_zh.srt"

    # ── Step 1: Transcribe ──────────────────────────────────────────────────
    _log(f"Transcribing {video.name} ...")
    with tempfile.TemporaryDirectory() as tmpdir:
        try:
            transcribe_result = sub_transcribe(
                str(video),
                language="zh",
                device="cpu",
                compute_type="int8",
                output_dir=tmpdir,
            )
            aligned = sub_align(transcribe_result, str(video), device="cpu")
            convert_vector_to_Sub(
                aligned,
                str(video),
                output_format="srt",
                output_dir=tmpdir,
                align_language="zh",
            )
        except Exception as e:
            _log(f"ERROR during transcription: {e}")
            return

        srt_candidates = list(Path(tmpdir).glob("*.srt"))
        if not srt_candidates:
            _log("ERROR: whisperx produced no SRT file")
            return

        shutil.copy(srt_candidates[0], zh_srt)

    _log(f"Chinese SRT: {zh_srt}")

    # ── Step 2: Translate to English ────────────────────────────────────────
    api_key = os.environ.get("MINIMAX_API_KEY") or config.api_key
    if not api_key:
        _log("WARN: MINIMAX_API_KEY not set — skipping English translation")
    else:
        _log("Translating to English via MiniMax ...")
        try:
            llm = OPENAI_General_Interface(
                api_key=api_key,
                base_url=config.base_url,
                mode="batch",
                model_name=config.translation_model_name,
                translate_prompt=config.translation_prompt,
            )
            entries = srt_reader(str(zh_srt))
            translated = llm.batch_translate(entries)
            en_srt = out_dir / f"{stem}_en.srt"
            srt_writer(translated, str(en_srt))
            _log(f"English SRT: {en_srt}")
        except Exception as e:
            _log(f"ERROR during translation: {e}")

    # ── Step 3: Generate pinyin SRT ─────────────────────────────────────────
    _log("Generating pinyin SRT ...")
    try:
        pinyin_srt = out_dir / f"{stem}_pinyin.srt"
        generate_pinyin_srt(str(zh_srt), str(pinyin_srt))
        _log(f"Pinyin SRT: {pinyin_srt}")
    except Exception as e:
        _log(f"ERROR during pinyin generation: {e}")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: subtitle_pipeline.py <video_path>", file=sys.stderr)
        sys.exit(1)
    run(sys.argv[1])
```

- [ ] **Step 2: Smoke-test the import chain (no video needed)**

```bash
conda run -n whisperx_env python3 -c "
import sys; sys.path.insert(0, '/Users/pengby/Desktop/whisperx_Sub')
from subtitle_pipeline import run
print('imports OK')
"
```

Expected: `imports OK`

- [ ] **Step 3: Commit**

```bash
cd ~/Desktop/whisperx_Sub
git add subtitle_pipeline.py
git commit -m "feat: add subtitle_pipeline orchestration script"
```

---

## Task 5: Create run_pipeline.sh wrapper

**Files:**
- Create: `~/Desktop/whisperx_Sub/run_pipeline.sh`

The shell wrapper activates the conda env so Go does not need to know the Python path.

- [ ] **Step 1: Create run_pipeline.sh**

```bash
cat > ~/Desktop/whisperx_Sub/run_pipeline.sh << 'EOF'
#!/bin/bash
set -euo pipefail
source "/Users/pengby/miniconda3/etc/profile.d/conda.sh"
conda activate whisperx_env
python "/Users/pengby/Desktop/whisperx_Sub/subtitle_pipeline.py" "$1"
EOF
chmod +x ~/Desktop/whisperx_Sub/run_pipeline.sh
```

- [ ] **Step 2: Verify the script is executable**

```bash
ls -la ~/Desktop/whisperx_Sub/run_pipeline.sh
```

Expected: `-rwxr-xr-x ... run_pipeline.sh`

- [ ] **Step 3: Smoke-test the wrapper (just check it activates env)**

```bash
~/Desktop/whisperx_Sub/run_pipeline.sh /nonexistent_file.mp4 2>&1 | head -5
```

Expected output contains: `[subtitle] ERROR: file not found` (not a conda/python error)

- [ ] **Step 4: Commit**

```bash
cd ~/Desktop/whisperx_Sub
git add run_pipeline.sh
git commit -m "feat: add run_pipeline.sh conda wrapper"
```

---

## Task 6: Modify coding-lux downloader.go

**Files:**
- Modify: `~/Desktop/coding-lux/downloader/downloader.go`

Add `"os/exec"` to imports and a helper function `runSubtitlePipeline`, then call it as a goroutine in two places within `Download()`.

- [ ] **Step 1: Add `os/exec` to imports**

In `downloader.go`, the import block currently reads:
```go
import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
    ...
```

Add `"os/exec"` after `"os"`:

```go
	"os"
	"os/exec"
```

- [ ] **Step 2: Add the helper function**

Append this function to the end of `downloader.go` (after the closing brace of `Download()`):

```go
func runSubtitlePipeline(videoPath string) {
	script := filepath.Join(os.Getenv("HOME"), "Desktop", "whisperx_Sub", "run_pipeline.sh")
	cmd := exec.Command("bash", script, videoPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[subtitle] pipeline error: %v\n", err)
	}
}
```

- [ ] **Step 3: Add goroutine in the single-part return path**

Find this block near line 674 (single-part, after EmbedSubtitle check):

```go
		if downloader.option.EmbedSubtitle && len(subtitlePaths) > 0 {
			if !downloader.option.Silent {
				fmt.Println("Embedding subtitles...")
			}
			if err := utils.EmbedSubtitles(mergedFilePath, subtitlePaths, subtitleLangs); err != nil {
				return err
			}
			for _, path := range subtitleFilesToDelete {
				os.Remove(path)
			}
		}
		return nil
```

Replace with:

```go
		if downloader.option.EmbedSubtitle && len(subtitlePaths) > 0 {
			if !downloader.option.Silent {
				fmt.Println("Embedding subtitles...")
			}
			if err := utils.EmbedSubtitles(mergedFilePath, subtitlePaths, subtitleLangs); err != nil {
				return err
			}
			for _, path := range subtitleFilesToDelete {
				os.Remove(path)
			}
		}
		go runSubtitlePipeline(mergedFilePath)
		return nil
```

- [ ] **Step 4: Add goroutine in the multi-part return path**

Find this block near line 748 (multi-part, after EmbedSubtitle check):

```go
	if downloader.option.EmbedSubtitle && len(subtitlePaths) > 0 {
		if !downloader.option.Silent {
			fmt.Println("Embedding subtitles...")
		}
		if err := utils.EmbedSubtitles(mergedFilePath, subtitlePaths, subtitleLangs); err != nil {
			return err
		}
		for _, path := range subtitleFilesToDelete {
			os.Remove(path)
		}
	}

	return nil
```

Replace with:

```go
	if downloader.option.EmbedSubtitle && len(subtitlePaths) > 0 {
		if !downloader.option.Silent {
			fmt.Println("Embedding subtitles...")
		}
		if err := utils.EmbedSubtitles(mergedFilePath, subtitlePaths, subtitleLangs); err != nil {
			return err
		}
		for _, path := range subtitleFilesToDelete {
			os.Remove(path)
		}
	}
	go runSubtitlePipeline(mergedFilePath)
	return nil
```

- [ ] **Step 5: Build coding-lux to verify no compile errors**

```bash
cd ~/Desktop/coding-lux
go build ./...
```

Expected: no output (clean build).

- [ ] **Step 6: Commit**

```bash
cd ~/Desktop/coding-lux
git add downloader/downloader.go
git commit -m "feat: launch subtitle pipeline goroutine after download"
```

---

## Task 7: End-to-end integration test

- [ ] **Step 1: Download a short Bilibili video**

```bash
cd ~/Desktop/coding-lux
go run main.go -o /tmp/subtitle_test "https://www.bilibili.com/video/BV1xx411c7mD"
```

(Replace with any short B站 video URL — ideally under 2 minutes for faster testing.)

- [ ] **Step 2: Wait for subtitle pipeline to finish**

Watch stderr for `[subtitle]` log lines. The pipeline runs in the background — wait until you see:

```
[subtitle] Pinyin SRT: /tmp/subtitle_test/<name>_pinyin.srt
```

- [ ] **Step 3: Verify output files exist**

```bash
ls /tmp/subtitle_test/*.srt
```

Expected:
```
/tmp/subtitle_test/<name>_zh.srt
/tmp/subtitle_test/<name>_en.srt
/tmp/subtitle_test/<name>_pinyin.srt
```

- [ ] **Step 4: Inspect pinyin SRT content**

```bash
head -20 /tmp/subtitle_test/*_pinyin.srt
```

Expected output shows interleaved Chinese + pinyin lines:
```
1
00:00:01,000 --> 00:00:05,000
大家好欢迎收看
dà jiā hǎo huān yíng shōu kàn
```

- [ ] **Step 5: Inspect English SRT content**

```bash
head -20 /tmp/subtitle_test/*_en.srt
```

Expected: English translation of the spoken content.
