"""Document processing pipeline: extract → chunk → embed → store."""
from __future__ import annotations

import logging
import uuid as uuid_lib

from fastapi import APIRouter
from pydantic import BaseModel

from app.config import settings
from app.db import get_pool
from app.services import chunker, embedder, extractor, storage_client

logger = logging.getLogger(__name__)
router = APIRouter()


class ProcessDocumentRequest(BaseModel):
    document_id: str
    case_id: str
    stored_path: str
    original_filename: str
    file_type: str
    mime_type: str


class ProcessDocumentResponse(BaseModel):
    success: bool
    page_count: int | None = None
    chunk_count: int | None = None
    error_message: str | None = None


@router.post("/process-document", response_model=ProcessDocumentResponse)
async def process_document(req: ProcessDocumentRequest) -> ProcessDocumentResponse:
    """
    Full async processing pipeline:
    1. Download file from MinIO
    2. Extract text
    3. Chunk text
    4. Embed chunks
    5. Store in document_chunks table
    """
    doc_id = uuid_lib.UUID(req.document_id)
    case_id = uuid_lib.UUID(req.case_id)

    try:
        # 1. Download
        content = storage_client.download(req.stored_path)

        # 2. Extract text
        result = extractor.extract(content, req.file_type, req.mime_type)
        if result.error:
            logger.warning("extraction error for %s: %s", req.document_id, result.error)
            return ProcessDocumentResponse(success=False, error_message=result.error)

        if not result.text.strip():
            return ProcessDocumentResponse(
                success=False,
                page_count=result.page_count,
                error_message="no extractable text found in document",
            )

        # 3. Chunk
        chunks = chunker.chunk_text(
            result.text,
            chunk_size=settings.max_chunk_size,
            overlap=settings.chunk_overlap,
        )
        if not chunks:
            return ProcessDocumentResponse(
                success=False,
                page_count=result.page_count,
                error_message="chunking produced no content",
            )

        # 4. Embed (batch in groups of 100 to stay within token limits)
        texts = [c.content for c in chunks]
        all_embeddings: list[list[float]] = []
        batch_size = 100
        for i in range(0, len(texts), batch_size):
            batch = texts[i : i + batch_size]
            all_embeddings.extend(embedder.embed_texts(batch))

        # 5. Store chunks
        pool = await get_pool()
        async with pool.connection() as conn:
            await conn.execute("DELETE FROM document_chunks WHERE document_id = %s", (doc_id,))
            for chunk, embedding in zip(chunks, all_embeddings):
                await conn.execute(
                    """
                    INSERT INTO document_chunks
                        (document_id, case_id, chunk_index, content, embedding)
                    VALUES (%s, %s, %s, %s, %s::vector)
                    """,
                    (doc_id, case_id, chunk.index, chunk.content, str(embedding)),
                )

        logger.info(
            "processed document %s: %d chunks, page_count=%s",
            req.document_id, len(chunks), result.page_count,
        )
        return ProcessDocumentResponse(
            success=True,
            page_count=result.page_count,
            chunk_count=len(chunks),
        )

    except Exception as exc:
        logger.exception("failed to process document %s", req.document_id)
        return ProcessDocumentResponse(success=False, error_message=str(exc))
