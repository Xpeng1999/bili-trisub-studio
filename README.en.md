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

## Quick Start

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

## Windows One-Click Package

Distribute the complete Windows package instead of the standalone exe. The package contains:

- `bili-trisub-studio.exe`
- `whisperx_Sub/`
- `install.bat`
- `start.bat`
- `README-Windows.txt`

On first use, users run `install.bat` to create the local `.venv` and install subtitle dependencies. After that, they run `start.bat` to open the web UI.

Build the Windows executable with the command below, then package it with the scripts under `packaging/windows/`:

```bash
GOOS=windows GOARCH=amd64 go build -o dist/bili-trisub-studio-windows-amd64.exe .
```

Note: the executable contains the Go application only. The subtitle pipeline still requires Python, FFmpeg, and the Python dependencies listed in `whisperx_Sub/requirements.txt`. Users must provide their own LLM API URL, model name, and API key in the web UI; this project does not ship with a default LLM provider.

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
