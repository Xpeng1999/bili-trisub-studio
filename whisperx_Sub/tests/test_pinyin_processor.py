import os
import sys
import tempfile
import pytest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))
from pinyin_processor import parse_srt, add_pinyin_line, generate_pinyin_srt


def test_parse_srt_single_block():
    content = "1\n00:00:01,000 --> 00:00:03,000\n你好世界"
    blocks = parse_srt(content)
    assert len(blocks) == 1
    assert blocks[0]['index'] == '1'
    assert blocks[0]['timestamp'] == '00:00:01,000 --> 00:00:03,000'
    assert blocks[0]['text'] == '你好世界'


def test_parse_srt_multiple_blocks():
    content = "1\n00:00:01,000 --> 00:00:03,000\n你好世界\n\n2\n00:00:04,000 --> 00:00:06,000\n学习中文"
    blocks = parse_srt(content)
    assert len(blocks) == 2
    assert blocks[1]['text'] == '学习中文'


def test_add_pinyin_line_format():
    result = add_pinyin_line('你好世界')
    lines = result.split('\n')
    assert len(lines) == 2
    assert lines[0] == '你好世界'


def test_add_pinyin_line_contains_pinyin():
    result = add_pinyin_line('你好')
    lines = result.split('\n')
    assert 'n' in lines[1]
    assert len(lines[1]) > 0


def test_generate_pinyin_srt_creates_file():
    srt_content = "1\n00:00:01,000 --> 00:00:03,000\n你好世界\n\n2\n00:00:04,000 --> 00:00:06,000\n学习中文"
    with tempfile.NamedTemporaryFile(mode='w', suffix='.srt', delete=False, encoding='utf-8') as f:
        f.write(srt_content)
        zh_path = f.name
    out_path = zh_path.replace('.srt', '_pinyin.srt')
    try:
        generate_pinyin_srt(zh_path, out_path)
        assert os.path.exists(out_path)
        with open(out_path, encoding='utf-8') as f:
            result = f.read()
        assert '你好世界' in result
        assert '学习中文' in result
        assert '00:00:01,000 --> 00:00:03,000' in result
        assert '00:00:04,000 --> 00:00:06,000' in result
    finally:
        os.unlink(zh_path)
        if os.path.exists(out_path):
            os.unlink(out_path)
