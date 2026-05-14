#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ -f "$HOME/miniconda3/etc/profile.d/conda.sh" ]; then
  source "$HOME/miniconda3/etc/profile.d/conda.sh"
elif [ -f "$HOME/anaconda3/etc/profile.d/conda.sh" ]; then
  source "$HOME/anaconda3/etc/profile.d/conda.sh"
fi

if command -v conda >/dev/null 2>&1; then
  conda activate "${WHISPERX_CONDA_ENV:-whisperx_env}"
fi

export HF_ENDPOINT=https://hf-mirror.com
# API key is read from ~/.zshrc directly by subtitle_pipeline.py as fallback
cd "$SCRIPT_DIR"
mkdir -p temp output
python "$SCRIPT_DIR/subtitle_pipeline.py" "$1" "${2:-}"
