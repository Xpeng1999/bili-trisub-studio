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

## Start

```bash
go run . --web --web-addr 127.0.0.1:8080
```

Then open:

```text
http://127.0.0.1:8080/
```

## Windows Build

A Windows executable can be built with:

```bash
GOOS=windows GOARCH=amd64 go build -o dist/bili-trisub-studio-windows-amd64.exe .
```

For the subtitle pipeline, Windows users still need Python, FFmpeg, and the Python dependencies described in `whisperx_Sub/requirements.txt`.

## Notes

This is a pre-development tool for a larger product workflow. It is designed for local use, fast iteration, subtitle review, and keeping generated assets organized in one output folder per video.
