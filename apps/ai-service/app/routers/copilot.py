"""AI Copilot endpoints — ask, summarize, generate-note, extract-actions."""
from __future__ import annotations

import logging
import re
import time
from typing import Any

from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel, Field

from app.config import settings
from app.service_auth import require_service_token
from app.services import pii_masker, rag
from app.services.llm_router import llm_router

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/copilot", dependencies=[Depends(require_service_token)])

# ── Prompt injection patterns ─────────────────────────────────────────────────

_INJECTION_PATTERNS = [
    re.compile(r"ignore (all )?(previous|prior|above) (instructions?|prompts?)", re.I),
    re.compile(r"you are now", re.I),
    re.compile(r"disregard (all |the |your )?(previous|prior)?", re.I),
    re.compile(r"forget (all |your )?(previous|prior|the )?instructions?", re.I),
    re.compile(r"pretend (you are|to be|you're)", re.I),
    re.compile(r"new (persona|role|identity|instructions?)", re.I),
    re.compile(r"act as (an? |the )?", re.I),
    re.compile(r"jailbreak", re.I),
    re.compile(r"DAN\b", re.I),
]


def _check_injection(text: str) -> None:
    for pattern in _INJECTION_PATTERNS:
        if pattern.search(text):
            raise HTTPException(status_code=400, detail="Prompt injection detected. Request rejected.")


def _scan_chunks_for_injection(chunks: list) -> None:
    """Reject the request if any retrieved document chunk contains injection patterns."""
    for chunk in chunks:
        _check_injection(chunk.content)


# ── Request / Response schemas ────────────────────────────────────────────────

class AskRequest(BaseModel):
    case_id: str
    question: str = Field(..., min_length=1, max_length=500)
    top_k: int = Field(default=8, ge=1, le=20)


class SummarizeRequest(BaseModel):
    case_id: str


class GenerateNoteRequest(BaseModel):
    case_id: str


class ExtractActionsRequest(BaseModel):
    case_id: str
    source_type: str = "documents"


class CitationOut(BaseModel):
    doc_id: str
    doc_name: str
    chunk_index: int
    page_number: int | None = None
    excerpt: str
    similarity: float


class CopilotResponse(BaseModel):
    answer: str
    confidence_score: float | None
    citations: list[CitationOut]
    model_used: str
    provider: str
    tokens_input: int
    tokens_output: int
    latency_ms: int
    insufficient_context: bool
    prompt_text: str


# ── Shared helpers ────────────────────────────────────────────────────────────

_SYSTEM_BASE = (
    "You are OpsOwl, an AI assistant embedded in an enterprise investigation workspace. "
    "You help financial crime analysts investigate cases. "
    "Always base your answers strictly on the provided context. "
    "If you cannot answer from the provided context, respond with exactly: INSUFFICIENT_CONTEXT\n"
    "Be concise and factual. Do not speculate or add information not present in the context."
)


def _build_response(
    *,
    answer: str,
    chunks: list[rag.RetrievedChunk],
    model_used: str,
    provider: str,
    tokens_input: int,
    tokens_output: int,
    latency_ms: int,
    prompt_text: str,
) -> CopilotResponse:
    insufficient = answer.strip() == "INSUFFICIENT_CONTEXT"
    if insufficient:
        answer = "I don't have enough context in the uploaded documents to answer this question. Please upload more relevant documents to this case."

    confidence: float | None = None
    citations: list[CitationOut] = []
    if chunks:
        confidence = round(rag.avg_similarity(chunks), 2)
        citations = [
            CitationOut(
                doc_id=c.doc_id,
                doc_name=c.doc_name,
                chunk_index=c.chunk_index,
                page_number=c.page_number,
                excerpt=c.content[:300],
                similarity=round(c.similarity, 3),
            )
            for c in chunks
        ]

    return CopilotResponse(
        answer=answer,
        confidence_score=confidence,
        citations=citations,
        model_used=model_used,
        provider=provider,
        tokens_input=tokens_input,
        tokens_output=tokens_output,
        latency_ms=latency_ms,
        insufficient_context=insufficient,
        prompt_text=prompt_text,
    )


# ── Endpoints ─────────────────────────────────────────────────────────────────

@router.post("/ask", response_model=CopilotResponse)
async def ask(req: AskRequest) -> CopilotResponse:
    """RAG-based Q&A: embed question → pgvector search → LLM answer with citations."""
    _check_injection(req.question)

    masked = pii_masker.mask(req.question)
    t0 = time.monotonic()

    try:
        query_embedding = await llm_router.embed(masked.masked_text)
        chunks = await rag.search(req.case_id, query_embedding, top_k=req.top_k)
    except Exception as exc:
        logger.warning("Embedding/RAG unavailable for ask endpoint: %s", exc)
        return _build_response(
            answer="INSUFFICIENT_CONTEXT",
            chunks=[],
            model_used="none",
            provider="none",
            tokens_input=0,
            tokens_output=0,
            latency_ms=int((time.monotonic() - t0) * 1000),
            prompt_text=masked.masked_text,
        )

    if not chunks:
        return _build_response(
            answer="INSUFFICIENT_CONTEXT",
            chunks=[],
            model_used="none",
            provider="none",
            tokens_input=0,
            tokens_output=0,
            latency_ms=int((time.monotonic() - t0) * 1000),
            prompt_text=masked.masked_text,
        )

    _scan_chunks_for_injection(chunks)
    context = rag.build_context_block(chunks)
    prompt_text = f"Question: {masked.masked_text}\n\nContext from documents:\n{context}"

    system = _SYSTEM_BASE + "\n\nYour citations must reference the [Source N] labels provided in the context."
    messages = [{"role": "user", "content": prompt_text}]

    answer, model, tok_in, tok_out = await llm_router.complete(messages, system=system)
    latency_ms = int((time.monotonic() - t0) * 1000)

    return _build_response(
        answer=answer,
        chunks=chunks,
        model_used=model,
        provider=settings.default_provider,
        tokens_input=tok_in,
        tokens_output=tok_out,
        latency_ms=latency_ms,
        prompt_text=prompt_text,
    )


@router.post("/summarize", response_model=CopilotResponse)
async def summarize(req: SummarizeRequest) -> CopilotResponse:
    """Summarize all documents in a case."""
    t0 = time.monotonic()
    chunks = await rag.fetch_all_chunks(req.case_id, limit=150)

    if not chunks:
        return _build_response(
            answer="INSUFFICIENT_CONTEXT",
            chunks=[],
            model_used="none",
            provider="none",
            tokens_input=0,
            tokens_output=0,
            latency_ms=int((time.monotonic() - t0) * 1000),
            prompt_text="[no documents available]",
        )

    _scan_chunks_for_injection(chunks)
    context = rag.build_context_block(chunks)
    prompt_text = (
        "Please provide a structured case summary covering:\n"
        "1. Key entities and parties involved\n"
        "2. Main timeline of events\n"
        "3. Primary risk factors or red flags\n"
        "4. Outstanding questions or gaps in the documentation\n\n"
        f"Documents:\n{context}"
    )

    system = (
        _SYSTEM_BASE
        + "\n\nProduce a structured summary. Use headers for each section. "
        "Reference source documents by their [Source N] label."
    )
    messages = [{"role": "user", "content": prompt_text}]

    answer, model, tok_in, tok_out = await llm_router.complete(messages, system=system, max_tokens=2048)
    latency_ms = int((time.monotonic() - t0) * 1000)

    return _build_response(
        answer=answer,
        chunks=chunks,
        model_used=model,
        provider=settings.default_provider,
        tokens_input=tok_in,
        tokens_output=tok_out,
        latency_ms=latency_ms,
        prompt_text=prompt_text,
    )


@router.post("/generate-note", response_model=CopilotResponse)
async def generate_note(req: GenerateNoteRequest) -> CopilotResponse:
    """Draft an investigation note from case documents."""
    t0 = time.monotonic()

    # Use embeddings of a "general investigation note" query to get relevant chunks.
    seed_query = "key findings risk factors timeline entities"
    try:
        seed_embedding = await llm_router.embed(seed_query)
        chunks = await rag.search(req.case_id, seed_embedding, top_k=10)
    except Exception as exc:
        logger.warning("Embedding/RAG unavailable for generate-note endpoint: %s", exc)
        chunks = []

    if not chunks:
        return _build_response(
            answer="INSUFFICIENT_CONTEXT",
            chunks=[],
            model_used="none",
            provider="none",
            tokens_input=0,
            tokens_output=0,
            latency_ms=int((time.monotonic() - t0) * 1000),
            prompt_text="[no documents available]",
        )

    _scan_chunks_for_injection(chunks)
    context = rag.build_context_block(chunks)
    prompt_text = (
        "Draft a professional investigation note based on the following case documents. "
        "The note should be written as if by a financial crime analyst and include:\n"
        "- Case overview\n"
        "- Key findings\n"
        "- Risk assessment\n"
        "- Recommended next steps\n\n"
        f"Context:\n{context}"
    )

    system = (
        _SYSTEM_BASE
        + "\n\nWrite in a professional, factual tone. "
        "Mark this as a DRAFT at the top. "
        "Include a 'Sources' section at the end listing documents referenced."
    )
    messages = [{"role": "user", "content": prompt_text}]

    answer, model, tok_in, tok_out = await llm_router.complete(messages, system=system, max_tokens=1500)
    latency_ms = int((time.monotonic() - t0) * 1000)

    return _build_response(
        answer=answer,
        chunks=chunks,
        model_used=model,
        provider=settings.default_provider,
        tokens_input=tok_in,
        tokens_output=tok_out,
        latency_ms=latency_ms,
        prompt_text=prompt_text,
    )


@router.post("/extract-actions", response_model=CopilotResponse)
async def extract_actions(req: ExtractActionsRequest) -> CopilotResponse:
    """Extract suggested action items from case documents."""
    t0 = time.monotonic()

    seed_query = "action items tasks follow-up required investigation steps"
    try:
        seed_embedding = await llm_router.embed(seed_query)
        chunks = await rag.search(req.case_id, seed_embedding, top_k=12)
    except Exception as exc:
        logger.warning("Embedding/RAG unavailable for extract-actions endpoint: %s", exc)
        chunks = []

    if not chunks:
        return _build_response(
            answer="INSUFFICIENT_CONTEXT",
            chunks=[],
            model_used="none",
            provider="none",
            tokens_input=0,
            tokens_output=0,
            latency_ms=int((time.monotonic() - t0) * 1000),
            prompt_text="[no documents available]",
        )

    _scan_chunks_for_injection(chunks)
    context = rag.build_context_block(chunks)
    prompt_text = (
        "Analyze the following case documents and extract specific, actionable investigation tasks. "
        "For each task provide:\n"
        "- title: short action title (max 80 chars)\n"
        "- description: what needs to be done and why\n"
        "- priority: critical | high | medium | low\n"
        "- source_citation: which document/section this is based on\n\n"
        "Return as a numbered list. Each task on its own line.\n\n"
        f"Documents:\n{context}"
    )

    system = (
        _SYSTEM_BASE
        + "\n\nOnly extract tasks that have clear evidence in the documents. "
        "Do not suggest generic tasks. "
        "Maximum 10 tasks. Focus on the most critical ones."
    )
    messages = [{"role": "user", "content": prompt_text}]

    answer, model, tok_in, tok_out = await llm_router.complete(messages, system=system, max_tokens=1000)
    latency_ms = int((time.monotonic() - t0) * 1000)

    return _build_response(
        answer=answer,
        chunks=chunks,
        model_used=model,
        provider=settings.default_provider,
        tokens_input=tok_in,
        tokens_output=tok_out,
        latency_ms=latency_ms,
        prompt_text=prompt_text,
    )
