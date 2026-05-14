@echo off
setlocal
cd /d "%~dp0"

if not exist "%~dp0bili-trisub-studio.exe" (
  echo Cannot find bili-trisub-studio.exe in this folder.
  pause
  exit /b 1
)

if not exist "%~dp0whisperx_Sub\subtitle_pipeline.py" (
  echo Cannot find whisperx_Sub. Please extract the whole zip package first.
  pause
  exit /b 1
)

if not exist "%~dp0.venv\Scripts\python.exe" (
  echo Python environment is not installed yet.
  echo Please run install.bat first.
  pause
  exit /b 1
)

set LUX_WHISPERX_SUB_DIR=%~dp0whisperx_Sub
start "" http://127.0.0.1:8080
"%~dp0bili-trisub-studio.exe" --web
pause
