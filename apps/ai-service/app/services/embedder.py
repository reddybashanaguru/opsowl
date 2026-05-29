"""Embedding helpers — thin sync wrappers over llm_router for legacy callers.

The document processing pipeline (documents.py) now calls llm_router.embed_batch()
directly. This module is retained for any remaining sync callers and test mocks.
"""
from __future__ import annotations

import asyncio

from app.services.llm_router import llm_router


def embed_texts(texts: list[str]) -> list[list[float]]:
    """Synchronous batch embed — delegates to llm_router (provider-aware)."""
    if not texts:
        return []
    return asyncio.get_event_loop().run_until_complete(llm_router.embed_batch(texts))


def embed_single(text: str) -> list[float]:
    return embed_texts([text])[0]
