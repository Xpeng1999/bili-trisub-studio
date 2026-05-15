import os

# task type: "transcribe" = transcription only, "all" = transcription + translation
task = "all"

# output and temp directory path
output_dir = "./output"
temp_dir = "./temp"
output_format = "all"

# audio or video file path (overridden by subtitle_pipeline.py at runtime)
audio_file = ""

# subtitle translation parameters
is_using_local_model = False
base_url = os.environ.get("LUX_LLM_BASE_URL", "")
api_key = os.environ.get("LUX_LLM_API_KEY", "")
translation_model_name = os.environ.get("LUX_LLM_MODEL", "")
translation_prompt = ""
srt_file_name = ""

# whisper model: "small"/"medium"/"large-v2" or a local model directory.
# The Windows installer preloads "small" because large-v2 is too large for a
# normal GitHub release package. Advanced users can override with LUX_WHISPER_MODEL.
whisper_model = os.environ.get("LUX_WHISPER_MODEL", "small")

# rePunctuation parameters
WORDS_NUM_LIMITS = 12