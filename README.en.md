# Bili Tri-Sub Studio

> A local Bilibili downloader and trilingual subtitle proofreading studio.

🌐 Language: [中文](README.zh-CN.md) | [English](README.en.md)

## What It Does

Bili Tri-Sub Studio downloads Bilibili videos and generates trilingual subtitles: Chinese, English, and Pinyin. After a task finishes, you can preview the video in the browser, move sentence by sentence, proofread Chinese and English subtitles, regenerate Pinyin from edited Chinese, and save the corrected subtitle files.

## Current Scope

- Supported platform: **Bilibili**
- Removed platform: **Douyin**
- Local UI: `http://127.0.0.1:8080/`
- Workflow: download, transcription, English subtitles, Pinyin subtitles, trilingual alignment, proofreading, and subtitle saving.

Douyin support has been removed because current anti-bot behavior is not reliable with exported cookies alone.

## Recommended Setup

This project is currently distributed as source code only. GitHub Releases and Windows one-click packages are intentionally not provided because the subtitle pipeline depends on Python, FFmpeg, and Whisper model files. The model files are large, and download reliability depends heavily on whether the user's machine can access HuggingFace or a mirror.

Clone the project:

```bash
git clone https://github.com/Xpeng1999/bili-trisub-studio.git
cd bili-trisub-studio
```

Install Go, Python 3.10/3.11, and FFmpeg first. Then create a Python environment and install subtitle dependencies:

```bash
python -m venv .venv

# macOS / Linux
source .venv/bin/activate

# Windows PowerShell
# .\.venv\Scripts\Activate.ps1

python -m pip install --upgrade "pip<26" "setuptools<81" wheel
python -m pip install -r whisperx_Sub/requirements.txt
```

Prepare the Whisper model. The default model is `small`; `faster-whisper` will download it on first use if the machine can access HuggingFace or the configured mirror. For unstable networks, download a faster-whisper model manually and set:

```bash
export LUX_WHISPER_MODEL=/absolute/path/to/faster-whisper-model
```

Windows PowerShell:

```powershell
$env:LUX_WHISPER_MODEL="C:\path\to\faster-whisper-model"
```

Start the local web UI:

```bash
go run . --web --web-addr 127.0.0.1:8080
```

Then open:

```text
http://127.0.0.1:8080/
```

## How To Use

1. Paste a Bilibili video URL.
2. Choose an output directory.
3. Click “Download + Add Subtitles”.
4. Wait for the task to finish.
5. Preview the video and subtitles sentence by sentence.
6. Edit Chinese or English subtitles.
7. After editing Chinese, confirm it to regenerate Pinyin.
8. Save the subtitle files.

Each task creates a subfolder under the selected output directory, keeping the video, subtitles, and trilingual JSON together.

## LLM Settings

The web UI requires your own LLM API URL, model name, and API key for Chinese-to-English subtitle translation. The public repository does not include a default provider or private key.

For local development only, you can create an ignored `local_llm_config.json` file in the repository root:

```json
{
  "baseUrl": "https://your-llm-api.example.com",
  "model": "your-model-name",
  "apiKey": "your-api-key"
}
```

This file is listed in `.gitignore` and must not be committed.

## Windows Notes

Windows users should follow the source-code workflow above instead of downloading a release zip. Install Python 3.11, FFmpeg, and Go, then run the same setup commands in PowerShell. If model download fails, manually download a faster-whisper model and set `LUX_WHISPER_MODEL` to that local model directory.

Build the Windows executable with the command below, then package it with the scripts under `packaging/windows/`:

```bash
GOOS=windows GOARCH=amd64 go build -o dist/bili-trisub-studio-windows-amd64.exe .
```

Note: the executable contains the Go application only. It is not a complete product package. Subtitle generation still requires Python dependencies, FFmpeg, and Whisper model files.

## GitHub References and Licenses

This project builds on or references the following GitHub projects:

| Project | Purpose | License note |
| --- | --- | --- |
| [iawia002/lux](https://github.com/iawia002/lux) | Go video downloading foundation | MIT License |
| [hiddenblue/whisperx_Sub](https://github.com/hiddenblue/whisperx_Sub) | Subtitle pipeline reference and local subtitle tooling | GPL-3.0, kept under `whisperx_Sub/` |
| [m-bain/whisperX](https://github.com/m-bain/whisperX) | Speech recognition and alignment ecosystem | See upstream license |

The root `LICENSE` file is the original MIT license from Lux. The `whisperx_Sub/` directory keeps its upstream GPL-3.0 license. Keep both notices when redistributing this project.

## Product Role

This is a pre-development tool for the next product stage. The goal is a dependable local workflow for video download, subtitle generation, trilingual alignment, and human proofreading. Small, useful, and steady. ✨