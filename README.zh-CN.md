# Bili Tri-Sub Studio

> 一个本地运行的 B 站视频下载与三语字幕校对工具。

🌐 语言切换：[中文](README.zh-CN.md) | [English](README.en.md)

## 这个工具做什么

Bili Tri-Sub Studio 可以帮助你下载 B 站视频，并生成中文、英文、拼音三语字幕。下载完成后，你可以直接在网页里预览视频，逐句浏览字幕，并手动校对中英文字幕。

中文字幕确认后，拼音会根据最新的中文内容重新生成；点击保存后，会真实写回工程生成的字幕文件。

## 当前范围

- 支持平台：**B站**
- 已取消平台：**抖音**
- 本地页面：`http://127.0.0.1:8080/`
- 主要流程：下载视频、语音识别、英文字幕、拼音字幕、三语对齐、逐句校对、保存字幕文件

> 说明：抖音下载已取消。我们已经验证过，当前抖音风控下即使导入 Cookie 文件也不稳定，继续保留入口会影响工具可靠性。

## 快速启动

```bash
go run . --web --web-addr 127.0.0.1:8080
```

然后打开：

```text
http://127.0.0.1:8080/
```

## 使用方式

1. 粘贴 B 站视频链接。
2. 选择下载目录。
3. 点击“下载 + 添加字幕”。
4. 等待任务完成。
5. 在预览区逐句浏览视频和三语字幕。
6. 修改中文或英文字幕。
7. 修改中文后点击“确认中文并生成拼音”。
8. 点击“保存字幕文件”。

每个任务都会在你选择的目录下创建一个子文件夹，视频、字幕和三语 JSON 都会放在里面。

## Windows 可执行文件

可以用下面的命令生成 Windows 版 exe：

```bash
GOOS=windows GOARCH=amd64 go build -o dist/bili-trisub-studio-windows-amd64.exe .
```

注意：exe 只包含 Go 程序本体。字幕生成仍然依赖 Python 环境、FFmpeg，以及 `whisperx_Sub/requirements.txt` 中的 Python 依赖。

## 引用的开源项目与 License

本项目基于或参考了以下 GitHub 项目：

| 项目 | 用途 | License 说明 |
| --- | --- | --- |
| [iawia002/lux](https://github.com/iawia002/lux) | Go 视频下载基础能力 | MIT License |
| [hiddenblue/whisperx_Sub](https://github.com/hiddenblue/whisperx_Sub) | 字幕处理流程与本地字幕工具参考 | GPL-3.0，保留在 `whisperx_Sub/` 目录 |
| [m-bain/whisperX](https://github.com/m-bain/whisperX) | 语音识别与对齐生态 | 请以其上游 License 为准 |

根目录的 `LICENSE` 文件来自 Lux 的 MIT License。`whisperx_Sub/` 目录保留其上游 GPL-3.0 License。发布或二次分发时，请同时保留这些 License 信息。

## 项目定位

这是下一个阶段产品的前置开发工具，重点是把视频下载、字幕生成、三语对齐和人工校对串成一个稳定的本地工作流。小而顺手，比大而不稳更重要。✨
