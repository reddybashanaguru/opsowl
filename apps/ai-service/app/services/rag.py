"""RAG pipeline: embed query → pgvector similarity search → build augmented prompt."""
from __future__ import annotations

import logging
import uuid as uuid_lib
from dataclasses import dataclass

from app.config import settings
from app.db import get_pool
from app.services.llm_router import llm_router

logger = logging.getLogger(__name__)

_INSUFFICIENT_CONTEXT_THRESHOLD = 0.35  # min avg similarity to attempt an answer


@dataclass
class RetrievedChunk:
    doc_id: str
    doc_name: str
    chunk_index: int
    page_number: int | None
    content: str
    similarity: float


async def search(case_id: str, query_embedding: list[float], top_k: int = 8) -> list[RetrievedChunk]:
    """Search document_chunks for a case using cosine similarity."""
    pool = await get_pool()
    # Cast embedding list to pgvector format: '[0.1, 0.2, ...]'
    embedding_str = "[" + ",".join(str(v) for v in query_embedding) + "]"

    async with pool.connection() as conn:
        rows = await conn.execute(
            """
            SELECT
                dc.document_id::text   AS doc_id,
                d.original_filename    AS doc_name,
                dc.chunk_index,
                dc.page_number,
                dc.content,
                1 - (dc.embedding <=> %s::vector) AS similarity
            FROM document_chunks dc
            JOIN documents d ON d.id = dc.document_id
            WHERE dc.case_id = %s::uuid
              AND d.deleted_at IS NULL
            ORDER BY dc.embedding <=> %s::vector
            LIMIT %s
            """,
            (embedding_str, case_id, embedding_str, top_k),
        )
        results = await rows.fetchall()

    return [
        RetrievedChunk(
            doc_id=row[0],
            doc_name=row[1],
            chunk_index=row[2],
            page_number=row[3],
            content=row[4],
            similarity=float(row[5]),
        )
        for row in results
    ]


async def fetch_all_chunks(case_id: str, limit: int = 200) -> list[RetrievedChunk]:
    """Fetch all chunks for a case (for summarization — not similarity-based)."""
    pool = await get_pool()

    async with pool.connection() as conn:
        rows = await conn.execute(
            """
            SELECT
                dc.document_id::text   AS doc_id,
                d.original_filename    AS doc_name,
                dc.chunk_index,
                dc.page_number,
                dc.content
            FROM document_chunks dc
            JOIN documents d ON d.id = dc.document_id
            WHERE dc.case_id = %s::uuid
              AND d.deleted_at IS NULL
            ORDER BY d.created_at, dc.chunk_index
            LIMIT %s
            """,
            (case_id, limit),
        )
        results = await rows.fetchall()

    return [
        RetrievedChunk(
            doc_id=row[0],
            doc_name=row[1],
            chunk_index=row[2],
            page_number=row[3],
            content=row[4],
            similarity=1.0,  # not a similarity search
        )
        for row in results
    ]


def build_context_block(chunks: list[RetrievedChunk]) -> str:
    """Format retrieved chunks into a structured context block for the LLM."""
    parts = []
    for i, c in enumerate(chunks, 1):
        page = f" (page {c.page_number})" if c.page_number else ""
        parts.append(f"[Source {i}: {c.doc_name}{page}]\n{c.content}")
    return "\n\n---\n\n".join(parts)


def avg_similarity(chunks: list[RetrievedChunk]) -> float:
    if not chunks:
        return 0.0
    return sum(c.similarity for c in chunks) / len(chunks)


def is_sufficient(chunks: list[RetrievedChunk]) -> bool:
    return bool(chunks) and avg_similarity(chunks) >= _INSUFFICIENT_CONTEXT_THRESHOLD
