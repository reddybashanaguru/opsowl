"""Document processing pipeline: extract → chunk → embed → store."""
from __future__ import annotations

import asyncio
import logging
import uuid as uuid_lib

from fastapi import APIRouter, Depends
from pydantic import BaseModel

from app.config import settings
from app.db import get_pool
from app.service_auth import require_service_token
from app.services import chunker, extractor, storage_client
from app.services.llm_router import llm_router

logger = logging.getLogger(__name__)
router = APIRouter(dependencies=[Depends(require_service_token)])


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
    1. Download file from MinIO (thread pool — blocking I/O)
    2. Extract text      (thread pool — CPU-bound, blocks event loop if called directly)
    3. Chunk text        (thread pool — CPU-bound)
    4. Embed chunks      (async — Ollama HTTP, serialized to avoid overwhelming Ollama on cold load)
    5. Store in document_chunks table
    """
    doc_id = uuid_lib.UUID(req.document_id)
    case_id = uuid_lib.UUID(req.case_id)

    logger.info("process_document START doc_id=%s file=%s", req.document_id, req.original_filename)
    try:
        # 1. Download — run in thread pool so we don't block the event loop
        content = await asyncio.to_thread(storage_client.download, req.stored_path)
        logger.info("process_document DOWNLOADED %d bytes doc_id=%s", len(content), req.document_id)

        # 2. Extract text — CPU-bound (PyMuPDF, Tesseract OCR), must run in thread pool
        result = await asyncio.to_thread(
            extractor.extract, content, req.file_type, req.mime_type
        )
        if result.error:
            logger.warning("extraction error for %s: %s", req.document_id, result.error)
            return ProcessDocumentResponse(success=False, error_message=result.error)

        if not result.text.strip():
            return ProcessDocumentResponse(
                success=False,
                page_count=result.page_count,
                error_message="no extractable text found in document",
            )

        # 3. Chunk — CPU-bound, run in thread pool
        chunks = await asyncio.to_thread(
            chunker.chunk_text,
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

        # 4. Embed — serialize chunk-by-chunk to avoid overwhelming Ollama on cold model load.
        # Each call waits for the previous to finish; Ollama queues are small.
        logger.info("process_document EMBEDDING %d chunks doc_id=%s", len(chunks), req.document_id)
        all_embeddings: list[list[float]] = []
        for i, chunk in enumerate(chunks):
            logger.info("process_document embed chunk %d/%d doc_id=%s", i + 1, len(chunks), req.document_id)
            vec = await llm_router.embed(chunk.content)
            all_embeddings.append(vec)

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

    except BaseException as exc:
        # Catch asyncio.CancelledError (BaseException subclass) too — without this,
        # a cancelled coroutine causes an EOF to the Go caller instead of a clean error response.
        logger.exception("failed to process document %s", req.document_id)
        return ProcessDocumentResponse(success=False, error_message=str(exc) or type(exc).__name__)
