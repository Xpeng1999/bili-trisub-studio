$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Root

function Write-Step($Text) {
  Write-Host ""
  Write-Host "==> $Text" -ForegroundColor Cyan
}

function Find-Python {
  $candidates = @(
    "$Root\.venv\Scripts\python.exe",
    "py",
    "python"
  )
  foreach ($candidate in $candidates) {
    try {
      if ($candidate -like "*.exe" -and (Test-Path $candidate)) {
        return $candidate
      }
      $cmd = Get-Command $candidate -ErrorAction SilentlyContinue
      if ($cmd) {
        return $candidate
      }
    } catch {}
  }
  return ""
}

Write-Step "Checking Python"
$Python = Find-Python
if (-not $Python) {
  $Winget = Get-Command winget -ErrorAction SilentlyContinue
  if (-not $Winget) {
    throw "Python was not found, and winget is unavailable. Please install Python 3.10 or 3.11, then run this installer again."
  }
  Write-Host "Python was not found. Installing Python 3.11 via winget..."
  winget install --id Python.Python.3.11 -e --accept-package-agreements --accept-source-agreements
  $Python = Find-Python
  if (-not $Python) {
    throw "Python installation finished, but python is still not available in PATH. Please reopen this terminal and run the installer again."
  }
}

Write-Step "Checking FFmpeg"
if (-not (Get-Command ffmpeg -ErrorAction SilentlyContinue)) {
  $Winget = Get-Command winget -ErrorAction SilentlyContinue
  if ($Winget) {
    Write-Host "FFmpeg was not found. Installing FFmpeg via winget..."
    winget install --id Gyan.FFmpeg -e --accept-package-agreements --accept-source-agreements
  } else {
    Write-Warning "FFmpeg was not found. Please install FFmpeg manually and add it to PATH."
  }
}

Write-Step "Creating local Python environment"
if (-not (Test-Path "$Root\.venv\Scripts\python.exe")) {
  & $Python -m venv "$Root\.venv"
}
$VenvPython = "$Root\.venv\Scripts\python.exe"

Write-Step "Installing Python dependencies"
& $VenvPython -m pip install --upgrade pip setuptools wheel
& $VenvPython -m pip install -r "$Root\whisperx_Sub\requirements.txt"

Write-Step "Installation complete"
Write-Host "You can now double-click start.bat, then open http://127.0.0.1:8080 in your browser."
Read-Host "Press Enter to exit"
