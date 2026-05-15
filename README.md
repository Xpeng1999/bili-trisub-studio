# Bili Tri-Sub Studio

> A local-first Bilibili video downloader and trilingual subtitle proofreading studio.

🌐 Language: [中文](README.zh-CN.md) | [English](README.en.md)

## Quick View

Bili Tri-Sub Studio helps you download Bilibili videos, generate Chinese / English / Pinyin subtitles, preview the result in a browser, and manually proofread subtitle lines before saving them back to disk.

The first screen is the actual tool, not a landing page. Paste a Bilibili URL, choose an output folder, start the job, then review the video and subtitle card sentence by sentence.

## Current Scope

- Supported source: **Bilibili**
- Removed source: **Douyin**, because current anti-bot behavior could not be made reliable with exported cookies alone.
- Local Web UI: `http://127.0.0.1:8080/`
- Subtitle workflow: Chinese recognition, English translation, Pinyin generation, trilingual alignment, and manual proofreading.

## GitHub Project References and Licenses

This project builds on and references the following open-source work:

| Project | Use | License note |
| --- | --- | --- |
| [iawia002/lux](https://github.com/iawia002/lux) | Go video downloading foundation | MIT License |
| [hiddenblue/whisperx_Sub](https://github.com/hiddenblue/whisperx_Sub) | Subtitle pipeline reference and local subtitle tooling | GPL-3.0, kept under `whisperx_Sub/` with its own license |
| [m-bain/whisperX](https://github.com/m-bain/whisperX) | Speech recognition/alignment ecosystem used by the subtitle pipeline | See upstream license |

The root `LICENSE` file is the original MIT license from Lux. Components under `whisperx_Sub/` keep their upstream GPL-3.0 license. Please keep these notices when redistributing.

## Recommended Setup

This repository is currently distributed as source code only. GitHub Releases and Windows one-click packages are intentionally not provided because the subtitle pipeline depends on local Python packages, FFmpeg, and Whisper model files. These model files are large and their download reliability depends heavily on the user's network access to HuggingFace or mirrors.

Clone the repository:

```bash
git clone https://github.com/Xpeng1999/bili-trisub-studio.git
cd bili-trisub-studio
```

Install Go, Python 3.10/3.11, and FFmpeg first. Then create a Python environment and install the subtitle dependencies:

```bash
python -m venv .venv

# macOS / Linux
source .venv/bin/activate

# Windows PowerShell
# .\.venv\Scripts\Activate.ps1

python -m pip install --upgrade "pip<26" "setuptools<81" wheel
python -m pip install -r whisperx_Sub/requirements.txt
```

Prepare the Whisper model. The default model is `small`; it will be downloaded by `faster-whisper` on first use if the machine can access HuggingFace or the configured mirror. For unstable networks, download a faster-whisper model manually and set:

```bash
export LUX_WHISPER_MODEL=/absolute/path/to/faster-whisper-model
```

On Windows PowerShell:

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

## LLM Settings

The web UI requires your own LLM API URL, model name, and API key for Chinese-to-English subtitle translation. The public repository does not include any default provider or private key.

For local development only, you can create an ignored file named `local_llm_config.json` in the repository root:

```json
{
  "baseUrl": "https://your-llm-api.example.com",
  "model": "your-model-name",
  "apiKey": "your-api-key"
}
```

This file is listed in `.gitignore` and must not be committed.

## Windows Notes

Windows users should use the source-code workflow above instead of downloading a release zip. Install Python 3.11, FFmpeg, and Go, then run the same commands in PowerShell. If model download fails, manually download a faster-whisper model and set `LUX_WHISPER_MODEL` to that local model directory.

You may still build the Go executable for your own machine:

```bash
GOOS=windows GOARCH=amd64 go build -o dist/bili-trisub-studio-windows-amd64.exe .
```

The executable alone is not a complete product package; subtitle generation still needs Python dependencies and model files.

## Notes

This is a pre-development tool for a larger product workflow. It is designed for local use, fast iteration, subtitle review, and keeping generated assets organized in one output folder per video.