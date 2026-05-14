import re
from pypinyin import lazy_pinyin, Style


def parse_srt(content: str) -> list:
    blocks = []
    for entry in re.split(r'\n\n+', content.strip()):
        lines = entry.strip().split('\n')
        if len(lines) < 3:
            continue
        blocks.append({
            'index': lines[0],
            'timestamp': lines[1],
            'text': '\n'.join(lines[2:]),
        })
    return blocks


def add_pinyin_line(text: str) -> str:
    pinyin = ' '.join(lazy_pinyin(text, style=Style.TONE))
    return f"{text}\n{pinyin}"


def generate_pinyin_srt(zh_srt_path: str, out_path: str) -> None:
    with open(zh_srt_path, encoding='utf-8') as f:
        content = f.read()
    blocks = parse_srt(content)
    output_blocks = []
    for block in blocks:
        new_text = add_pinyin_line(block['text'])
        output_blocks.append(f"{block['index']}\n{block['timestamp']}\n{new_text}")
    with open(out_path, 'w', encoding='utf-8') as f:
        f.write('\n\n'.join(output_blocks))
