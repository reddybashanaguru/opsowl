"""Embedding generation via OpenAI text-embedding-3-small."""
from __future__ import annotations

from openai import OpenAI

from app.config import settings

_client: OpenAI | None = None


def _get_client() -> OpenAI:
    global _client
    if _client is None:
        _client = OpenAI(api_key=settings.openai_api_key)
    return _client


def embed_texts(texts: list[str]) -> list[list[float]]:
    """Return a list of 1536-dim embedding vectors, one per input text."""
    if not texts:
        return []
    client = _get_client()
    response = client.embeddings.create(
        model=settings.embedding_model,
        input=texts,
    )
    return [item.embedding for item in response.data]


def embed_single(text: str) -> list[float]:
    return embed_texts([text])[0]
