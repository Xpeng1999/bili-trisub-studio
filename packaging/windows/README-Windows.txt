Bili TriSub Studio - Windows package

How to use:

1. Extract the whole zip package to a normal folder, for example:
   C:\Users\YourName\Desktop\BiliTriSubStudio

2. Double-click install.bat.
   It creates a local .venv environment and installs the subtitle dependencies.
   The installer requires Python 3.11. If Python 3.11 or FFmpeg is missing,
   it will try to install them with winget.

3. Double-click start.bat.
   The app opens http://127.0.0.1:8080.

4. In the web page, fill in:
   - Bilibili video link
   - Output folder
   - LLM API URL
   - LLM model name
   - LLM API Key

Notes:

- The executable does not contain Python, PyTorch, FFmpeg, or Whisper model weights.
- The first subtitle run may download model files and can take a long time.
- Keep this folder intact. The exe, whisperx_Sub, and .venv are expected to stay together.
- The LLM settings are passed only to the current task as environment variables.
- If dependency installation fails, install.bat stops with an error instead of reporting success.
- Re-run install.bat after updates; it repairs dependency versions inside the existing .venv.
