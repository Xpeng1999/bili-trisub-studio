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

## 推荐安装方式

当前项目只推荐通过源码运行，不再提供 GitHub Release 安装包或 Windows 一键包。原因是字幕生成依赖 Python、FFmpeg 和 Whisper 模型文件；模型文件体积较大，并且下载是否成功高度依赖用户电脑能否访问 HuggingFace 或镜像源。

克隆项目：

```bash
git clone https://github.com/Xpeng1999/bili-trisub-studio.git
cd bili-trisub-studio
```

先安装 Go、Python 3.10/3.11 和 FFmpeg。然后创建 Python 环境并安装字幕依赖：

```bash
python -m venv .venv

# macOS / Linux
source .venv/bin/activate

# Windows PowerShell
# .\.venv\Scripts\Activate.ps1

python -m pip install --upgrade "pip<26" "setuptools<81" wheel
python -m pip install -r whisperx_Sub/requirements.txt
```

准备 Whisper 模型。默认模型是 `small`；如果网络可以访问 HuggingFace 或镜像源，`faster-whisper` 会在首次使用时自动下载。若用户网络不稳定，建议手动下载 faster-whisper 模型，并设置：

```bash
export LUX_WHISPER_MODEL=/absolute/path/to/faster-whisper-model
```

Windows PowerShell：

```powershell
$env:LUX_WHISPER_MODEL="C:\path\to\faster-whisper-model"
```

启动本地网页：

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

## 大模型配置

网页中需要填写你自己的大模型 API 地址、模型名称和 API Key，用于把中文字幕翻译成英文字幕。公开仓库不会内置默认大模型服务商，也不会包含任何私有 Key。

仅本地开发时，你可以在仓库根目录创建被忽略的 `local_llm_config.json`：

```json
{
  "baseUrl": "https://your-llm-api.example.com",
  "model": "your-model-name",
  "apiKey": "your-api-key"
}
```

这个文件已经写入 `.gitignore`，不要提交。

## Windows 说明

Windows 用户也建议按源码方式运行，而不是下载 release zip。请安装 Python 3.11、FFmpeg 和 Go，然后在 PowerShell 中执行上面的环境安装命令。如果模型下载失败，请手动下载 faster-whisper 模型，并把 `LUX_WHISPER_MODEL` 指向本地模型目录。

可以用下面的命令生成 Windows 版 exe，再配合 `packaging/windows/` 下的脚本打包：

```bash
GOOS=windows GOARCH=amd64 go build -o dist/bili-trisub-studio-windows-amd64.exe .
```

注意：exe 只包含 Go 程序本体，不是完整产品包。字幕生成仍然依赖 Python 环境、FFmpeg、Python 依赖和 Whisper 模型文件。

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