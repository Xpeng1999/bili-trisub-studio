#!/usr/bin/env python3
"""
subtitle_pipeline.py — called by coding-lux after video download.

Usage: python subtitle_pipeline.py <video_path> [cc_srt_path]

If cc_srt_path is provided and non-empty, WhisperX transcription is skipped
and the CC subtitle is used directly as the Chinese source.

Writes alongside the video:
  <stem>_zh.srt      Chinese transcription (or CC subtitle copy)
  <stem>_en.srt      English translation  (skipped if DEEPSEEK_API_KEY unset)
  <stem>_pinyin.srt  Chinese + pinyin bilingual
"""

import sys
import os
import json
import shutil
import subprocess
import tempfile
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
sys.path.insert(0, str(SCRIPT_DIR))

import config
from srt_util import srt_reader, srt_writer
from pinyin_processor import generate_pinyin_srt
from tri_align import build_from_zh, finalize as finalize_tri_subtitles

WHISPERX_BIN = Path(sys.executable).parent / "whisperx"


def _local_whisper_model() -> str:
    configured = getattr(config, "whisper_model", "large-v2")
    configured_path = Path(str(configured)).expanduser()
    if configured_path.exists():
        return str(configured_path.resolve())

    model_cache = SCRIPT_DIR / "models--Systran--faster-whisper-large-v2"
    snapshot_root = model_cache / "snapshots"
    if snapshot_root.exists():
        for snapshot in sorted(snapshot_root.iterdir()):
            if (snapshot / "model.bin").exists() and (snapshot / "config.json").exists():
                return str(snapshot.resolve())

    return str(configured)


def _log(msg: str) -> None:
    print(f"[subtitle] {msg}", file=sys.stderr, flush=True)


def _read_key_from_zshrc(key_name: str) -> str:
    import re
    zshrc = Path.home() / ".zshrc"
    if not zshrc.exists():
        return ""
    text = zshrc.read_text(errors="ignore")
    m = re.search(rf'{key_name}=["\']([^"\']+)["\']', text)
    return m.group(1) if m else ""


def _translate_zh_to_en(zh_srt: Path, en_srt: Path, api_key: str) -> None:
    from openai import OpenAI
    client = OpenAI(api_key=api_key, base_url=config.base_url)
    entries = srt_reader(str(zh_srt))
    translated = []
    batch_size = 8
    for i in range(0, len(entries), batch_size):
        batch = entries[i:i + batch_size]
        translations = _translate_batch(client, [row[2] for row in batch])
        for j, row in enumerate(batch):
            translated.append([row[0], row[1], translations[j] if j < len(translations) else ""])
    srt_writer(translated, str(en_srt))


def _translate_batch(client, texts: list[str]) -> list[str]:
    import re

    def valid(lines: list[str]) -> bool:
        if len(lines) != len(texts):
            return False
        return all(not re.search(r"[\u4e00-\u9fff]", line) for line in lines)

    payload = [{"id": i + 1, "text": text} for i, text in enumerate(texts)]
    resp = client.chat.completions.create(
        model=config.translation_model_name,
        messages=[{
            "role": "user",
            "content": (
                "Translate the Chinese subtitle items to concise English subtitles. "
                "Return ONLY a JSON array of strings. The array length must equal the input length. "
                "Each array element must be the translation of the item at the same position. "
                "Do not merge items. Do not include Chinese characters.\n\n"
                + json.dumps(payload, ensure_ascii=False)
            ),
        }],
        stream=False,
    )
    content = resp.choices[0].message.content.strip()
    if content.startswith("```"):
        content = re.sub(r"^```(?:json)?\s*|\s*```$", "", content, flags=re.S)
    try:
        parsed = json.loads(content)
        lines = [str(item).strip() for item in parsed]
        if valid(lines):
            return lines
    except Exception:
        pass

    lines = []
    for text in texts:
        lines.append(_translate_one(client, text))
    return lines


def _translate_one(client, text: str) -> str:
    import re
    resp = client.chat.completions.create(
        model=config.translation_model_name,
        messages=[{
            "role": "user",
            "content": (
                "Translate this Chinese subtitle to one concise English subtitle line. "
                "Return only English, no explanations, no Chinese characters:\n"
                + text
            ),
        }],
        stream=False,
    )
    line = resp.choices[0].message.content.strip().split("\n")[0].strip()
    if re.search(r"[\u4e00-\u9fff]", line):
        return ""
    return line


def run(video_path: str, cc_srt_path: str = "") -> None:
    video = Path(video_path).resolve()
    if not video.exists():
        _log(f"ERROR: file not found: {video}")
        return

    stem = video.stem
    out_dir = video.parent
    zh_srt = out_dir / f"{stem}_zh.srt"

    # ── Step 1: Chinese SRT — use CC subtitle or transcribe via WhisperX ────
    cc = Path(cc_srt_path) if cc_srt_path else None
    if cc and cc.exists() and cc.stat().st_size > 0:
        _log(f"CC subtitle detected, skipping WhisperX transcription")
        shutil.copy(cc, zh_srt)
        _log(f"Chinese SRT (from CC): {zh_srt}")
    else:
        _log(f"No CC subtitle, transcribing {video.name} via WhisperX ...")
        with tempfile.TemporaryDirectory() as tmpdir:
            cmd = [
                str(WHISPERX_BIN), str(video),
                "--language", "zh",
                "--model", _local_whisper_model(),
                "--device", "cpu",
                "--compute_type", "int8",
                "--output_format", "srt",
                "--output_dir", tmpdir,
                "--no_align",
            ]
            env = {**os.environ, "HF_ENDPOINT": "https://hf-mirror.com"}
            result = subprocess.run(cmd, capture_output=True, text=True, env=env)
            if result.returncode != 0:
                _log(f"ERROR during transcription:\n{result.stderr[-500:]}")
                return

            srt_candidates = list(Path(tmpdir).glob("*.srt"))
            if not srt_candidates:
                _log("ERROR: whisperx produced no SRT file")
                return

            shutil.copy(srt_candidates[0], zh_srt)

        _log(f"Chinese SRT: {zh_srt}")

    # ── Step 2: Translate to English ────────────────────────────────────────
    _log("Resegmenting Chinese SRT for tri-language alignment ...")
    try:
        build_from_zh(str(zh_srt))
        _log(f"Aligned Chinese SRT: {zh_srt}")
    except Exception as e:
        _log(f"ERROR during Chinese resegmentation: {e}")

    api_key = os.environ.get("DEEPSEEK_API_KEY") or config.api_key or _read_key_from_zshrc("DEEPSEEK_API_KEY")
    en_srt = out_dir / f"{stem}_en.srt"
    if not api_key:
        _log("WARN: DEEPSEEK_API_KEY not set — skipping English translation")
    else:
        _log("Translating to English via DeepSeek ...")
        try:
            _translate_zh_to_en(zh_srt, en_srt, api_key)
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

    # ── Step 4: Build front-end friendly tri-language JSON ─────────────────
    _log("Building tri-language preview data ...")
    try:
        tri_json = finalize_tri_subtitles(str(video), str(zh_srt), str(en_srt) if en_srt.exists() else "")
        _log(f"Tri-language JSON: {tri_json}")
    except Exception as e:
        _log(f"ERROR during tri-language alignment: {e}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: subtitle_pipeline.py <video_path> [cc_srt_path]", file=sys.stderr)
        sys.exit(1)
    cc = sys.argv[2] if len(sys.argv) >= 3 else ""
    run(sys.argv[1], cc)
