"""Text chunking with configurable size and overlap."""
from __future__ import annotations

from dataclasses import dataclass


@dataclass
class Chunk:
    index: int
    content: str
    page_number: int | None = None


def chunk_text(text: str, chunk_size: int = 2000, overlap: int = 200) -> list[Chunk]:
    """Split text into overlapping chunks by character count."""
    text = text.strip()
    if not text:
        return []

    chunks: list[Chunk] = []
    start = 0
    idx = 0
    while start < len(text):
        end = start + chunk_size
        piece = text[start:end]
        # Try to break at a sentence or newline boundary to avoid mid-word cuts.
        if end < len(text):
            # Look for last newline or period in the trailing overlap window.
            boundary = _find_boundary(piece, overlap)
            if boundary:
                piece = piece[:boundary]
        chunks.append(Chunk(index=idx, content=piece.strip()))
        idx += 1
        advance = len(piece) - overlap
        if advance <= 0:
            break
        start += advance
    return [c for c in chunks if c.content]


def _find_boundary(text: str, look_back: int) -> int | None:
    """Return the last sentence/paragraph boundary in the trailing look_back chars."""
    search_from = max(0, len(text) - look_back)
    window = text[search_from:]
    for sep in ("\n\n", "\n", ". ", "? ", "! "):
        idx = window.rfind(sep)
        if idx >= 0:
            return search_from + idx + len(sep)
    return None
