import re

_BULLET_RE = re.compile(r"^\s*(?:[-*+]|\d+[.)])\s+")


def chunk_text(text: str, max_chars: int = 900, overlap_chars: int = 120) -> list[str]:
    normalized_text = text.replace("\r\n", "\n").replace("\r", "\n").strip()
    if not normalized_text:
        return []

    max_chars = max(1, max_chars)
    overlap_chars = max(0, min(overlap_chars, max_chars - 1))

    if len(normalized_text) <= max_chars:
        return [normalized_text]

    blocks = _split_blocks(normalized_text)
    chunks: list[str] = []
    current = ""

    for block in blocks:
        if len(block) > max_chars:
            if current:
                chunks.append(current)
                current = ""
            chunks.extend(_split_long_block(block, max_chars, overlap_chars))
            continue

        candidate = _join_blocks(current, block)
        if len(candidate) <= max_chars:
            current = candidate
            continue

        if current:
            chunks.append(current)

        overlap = _overlap_tail(current, overlap_chars)
        candidate_with_overlap = _join_blocks(overlap, block) if overlap else block
        current = candidate_with_overlap if len(candidate_with_overlap) <= max_chars else block

    if current:
        chunks.append(current)

    return [chunk.strip() for chunk in chunks if chunk.strip()]


def _split_blocks(text: str) -> list[str]:
    blocks: list[str] = []
    paragraph_lines: list[str] = []

    for line in text.split("\n"):
        stripped = line.strip()
        if not stripped:
            _flush_paragraph(blocks, paragraph_lines)
            continue

        if _BULLET_RE.match(stripped):
            _flush_paragraph(blocks, paragraph_lines)
            blocks.append(stripped)
            continue

        paragraph_lines.append(stripped)

    _flush_paragraph(blocks, paragraph_lines)
    return blocks


def _flush_paragraph(blocks: list[str], paragraph_lines: list[str]) -> None:
    if not paragraph_lines:
        return
    blocks.append(" ".join(paragraph_lines).strip())
    paragraph_lines.clear()


def _split_long_block(block: str, max_chars: int, overlap_chars: int) -> list[str]:
    chunks: list[str] = []
    start = 0
    while start < len(block):
        end = min(start + max_chars, len(block))
        if end < len(block):
            boundary = max(block.rfind(" ", start, end), block.rfind("\n", start, end))
            if boundary > start:
                end = boundary
        chunk = block[start:end].strip()
        if chunk:
            chunks.append(chunk)
        if end >= len(block):
            break
        start = max(end - overlap_chars, start + 1)
    return chunks


def _join_blocks(left: str, right: str) -> str:
    if not left:
        return right.strip()
    if not right:
        return left.strip()
    separator = "\n" if _BULLET_RE.match(right.strip()) else "\n\n"
    return f"{left.strip()}{separator}{right.strip()}"


def _overlap_tail(text: str, overlap_chars: int) -> str:
    if overlap_chars <= 0 or not text:
        return ""

    tail = text[-overlap_chars:].strip()
    boundary = min(
        (index for index in [tail.find("\n"), tail.find(" ")] if index >= 0),
        default=-1,
    )
    if boundary > 0:
        tail = tail[boundary:].strip()
    return tail
