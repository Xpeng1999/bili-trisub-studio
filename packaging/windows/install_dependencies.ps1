$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Root

function Write-Step($Text) {
  Write-Host ""
  Write-Host "==> $Text" -ForegroundColor Cyan
}

function Invoke-Checked($FilePath, [string[]]$Arguments) {
  & $FilePath @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "Command failed with exit code ${LASTEXITCODE}: $FilePath $($Arguments -join ' ')"
  }
}

function Get-PythonMinor($PythonExe) {
  try {
    $version = & $PythonExe -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')"
    if ($LASTEXITCODE -eq 0) {
      return ($version | Select-Object -First 1).Trim()
    }
  } catch {}
  return ""
}

function Find-Python311 {
  $localPython = "$Root\.venv\Scripts\python.exe"
  if ((Test-Path $localPython) -and ((Get-PythonMinor $localPython) -eq "3.11")) {
    return $localPython
  }

  if (Get-Command py -ErrorAction SilentlyContinue) {
    try {
      $exe = & py -3.11 -c "import sys; print(sys.executable)" 2>$null
      if ($LASTEXITCODE -eq 0) {
        $path = ($exe | Select-Object -First 1).Trim()
        if ($path -and (Test-Path $path)) {
          return $path
        }
      }
    } catch {}
  }

  foreach ($candidate in @("python3.11", "python")) {
    $cmd = Get-Command $candidate -ErrorAction SilentlyContinue
    if ($cmd) {
      $path = $cmd.Source
      if ((Get-PythonMinor $path) -eq "3.11") {
        return $path
      }
    }
  }
  return ""
}

Write-Step "Checking Python 3.11"
$Python = Find-Python311
if (-not $Python) {
  $Winget = Get-Command winget -ErrorAction SilentlyContinue
  if (-not $Winget) {
    throw "Python 3.11 was not found, and winget is unavailable. Please install Python 3.11, then run this installer again."
  }
  Write-Host "Python 3.11 was not found. Installing Python 3.11 via winget..."
  Invoke-Checked "winget" @("install", "--id", "Python.Python.3.11", "-e", "--accept-package-agreements", "--accept-source-agreements")
  $Python = Find-Python311
  if (-not $Python) {
    throw "Python 3.11 installation finished, but it is still not available. Please reopen this terminal and run the installer again."
  }
}
Write-Host "Using Python: $Python"

Write-Step "Checking FFmpeg"
if (-not (Get-Command ffmpeg -ErrorAction SilentlyContinue)) {
  $Winget = Get-Command winget -ErrorAction SilentlyContinue
  if ($Winget) {
    Write-Host "FFmpeg was not found. Installing FFmpeg via winget..."
    Invoke-Checked "winget" @("install", "--id", "Gyan.FFmpeg", "-e", "--accept-package-agreements", "--accept-source-agreements")
  } else {
    Write-Warning "FFmpeg was not found. Please install FFmpeg manually and add it to PATH."
  }
}

Write-Step "Creating local Python environment"
$VenvPython = "$Root\.venv\Scripts\python.exe"
if ((Test-Path $VenvPython) -and ((Get-PythonMinor $VenvPython) -ne "3.11")) {
  Write-Host "Existing .venv is not Python 3.11. Recreating it..."
  Remove-Item -Recurse -Force "$Root\.venv"
}
if (-not (Test-Path $VenvPython)) {
  Invoke-Checked $Python @("-m", "venv", "$Root\.venv")
}
Write-Host "Using venv Python: $VenvPython"

Write-Step "Installing Python dependencies"
Invoke-Checked $VenvPython @("-m", "pip", "install", "--upgrade", "pip<26", "setuptools<81", "wheel")
Invoke-Checked $VenvPython @("-m", "pip", "install", "--only-binary=:all:", "numpy==1.26.4")
Invoke-Checked $VenvPython @("-m", "pip", "install", "-r", "$Root\whisperx_Sub\requirements.txt")
Invoke-Checked $VenvPython @("-m", "pip", "install", "--upgrade", "setuptools<81")
$env:PYTHONPATH = "$Root\whisperx_Sub"
Invoke-Checked $VenvPython @("-c", "import numpy, pypinyin, ctranslate2, openai, whisperx.transcribe; print('Python dependency check passed')")

Write-Step "Installation complete"
Write-Host "You can now double-click start.bat, then open http://127.0.0.1:8080 in your browser."
Read-Host "Press Enter to exit"