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

# whisper model: "small"/"medium"/"large-v2" (large-v2 best for Chinese)
whisper_model = "large-v2"

# rePunctuation parameters
WORDS_NUM_LIMITS = 12
