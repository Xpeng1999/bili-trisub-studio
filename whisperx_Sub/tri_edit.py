#!/usr/bin/env python3
import json
import sys

from pypinyin import Style, lazy_pinyin

from tri_align import to_simplified


def pinyin(text: str) -> str:
    return " ".join(lazy_pinyin(to_simplified(text or ""), style=Style.TONE))


def main() -> int:
    if len(sys.argv) < 2 or sys.argv[1] not in {"pinyin", "normalize"}:
        print("Usage: tri_edit.py pinyin|normalize", file=sys.stderr)
        return 2
    payload = json.load(sys.stdin)
    if sys.argv[1] == "normalize":
        segments = payload.get("segments", [])
        for index, segment in enumerate(segments, 1):
            text = to_simplified(segment.get("zh", ""))
            segment["index"] = index
            segment["zh"] = text
            segment["pinyin"] = pinyin(text)
        json.dump({"segments": segments}, sys.stdout, ensure_ascii=False)
        return 0
    text = to_simplified(payload.get("text", ""))
    json.dump({"text": text, "pinyin": pinyin(text)}, sys.stdout, ensure_ascii=False)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
