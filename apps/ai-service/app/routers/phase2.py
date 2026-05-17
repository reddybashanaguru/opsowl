"""Phase 2 AI endpoints — case risk analysis, comm entity extraction."""
from __future__ import annotations

import logging
from typing import Any

from fastapi import APIRouter
from pydantic import BaseModel

from app.services.llm_router import llm_router

logger = logging.getLogger(__name__)
router = APIRouter()

# ── Request / Response schemas ────────────────────────────────────────────────

class AnalyseCaseRequest(BaseModel):
    case_id: str


class AnalyseCaseResponse(BaseModel):
    risk_score: float
    risk_factors: list[str]
    entities: dict[str, Any]
    anomalies: list[str]
    recommended_actions: list[str]
    model_used: str
    confidence: float


class ExtractCommsRequest(BaseModel):
    body: str
    subject: str = ""
    channel: str = "other"


class ExtractCommsResponse(BaseModel):
    entities: dict[str, Any]
    action_items: list[str]
    sentiment: str
    model_used: str


# ── Case Risk Analysis ────────────────────────────────────────────────────────

@router.post("/analyse/case", response_model=AnalyseCaseResponse)
async def analyse_case(req: AnalyseCaseRequest) -> AnalyseCaseResponse:
    """
    Analyse a case and produce a risk score, risk factors, entities,
    anomalies, and recommended actions. Falls back to rule-based defaults
    if the LLM is unavailable.
    """
    try:
        from app.services import rag

        # Retrieve all document chunks for this case (up to 100)
        chunks = await rag.fetch_all_chunks(req.case_id, limit=100)

        if not chunks:
            return _default_case_response()

        context = rag.build_context_block(chunks[:40])
        prompt = (
            "You are a financial crime risk analyst. Analyse the following case documents "
            "and return a JSON object with exactly these keys:\n"
            '{\n'
            '  "risk_score": <float 0-100>,\n'
            '  "risk_factors": [<list of key risk factor strings>],\n'
            '  "entities": {"individuals": [], "companies": [], "accounts": []},\n'
            '  "anomalies": [<list of anomaly strings>],\n'
            '  "recommended_actions": [<list of recommended action strings>],\n'
            '  "confidence": <float 0-1>\n'
            "}\n"
            "Return ONLY valid JSON, no commentary.\n\n"
            f"Case documents:\n{context}"
        )
        system = (
            "You are a financial crime risk analyst AI. "
            "Always respond with valid JSON only — no markdown, no extra text."
        )
        messages = [{"role": "user", "content": prompt}]
        answer, model, _, _ = await llm_router.complete(messages, system=system, max_tokens=1024)

        import json
        cleaned = answer.strip().strip("```json").strip("```").strip()
        data = json.loads(cleaned)

        return AnalyseCaseResponse(
            risk_score=float(data.get("risk_score", 50.0)),
            risk_factors=data.get("risk_factors", []),
            entities=data.get("entities", {}),
            anomalies=data.get("anomalies", []),
            recommended_actions=data.get("recommended_actions", []),
            model_used=model,
            confidence=float(data.get("confidence", 0.7)),
        )

    except Exception as e:
        logger.warning("case analysis AI failed, returning default: %s", e)
        return _default_case_response()


def _default_case_response() -> AnalyseCaseResponse:
    return AnalyseCaseResponse(
        risk_score=50.0,
        risk_factors=["Insufficient document coverage for automated analysis"],
        entities={"individuals": [], "companies": [], "accounts": []},
        anomalies=[],
        recommended_actions=["Upload more case documents for deeper analysis"],
        model_used="rule-based",
        confidence=0.3,
    )


# ── Communication Entity Extraction ──────────────────────────────────────────

@router.post("/extract/comms-entities", response_model=ExtractCommsResponse)
async def extract_comms_entities(req: ExtractCommsRequest) -> ExtractCommsResponse:
    """
    Extract named entities, action items, and sentiment from a communication log entry.
    """
    try:
        prompt = (
            "Extract structured information from the following communication and return JSON:\n"
            '{\n'
            '  "entities": {"individuals": [], "companies": [], "phone_numbers": [], "emails": []},\n'
            '  "action_items": [<list of follow-up action strings>],\n'
            '  "sentiment": "<positive|neutral|negative|hostile>"\n'
            "}\n"
            "Return ONLY valid JSON.\n\n"
            f"Channel: {req.channel}\n"
            f"Subject: {req.subject}\n"
            f"Body: {req.body}"
        )
        system = (
            "You are an AI that extracts structured data from communications. "
            "Always respond with valid JSON only."
        )
        messages = [{"role": "user", "content": prompt}]
        answer, model, _, _ = await llm_router.complete(messages, system=system, max_tokens=512)

        import json
        cleaned = answer.strip().strip("```json").strip("```").strip()
        data = json.loads(cleaned)

        return ExtractCommsResponse(
            entities=data.get("entities", {}),
            action_items=data.get("action_items", []),
            sentiment=data.get("sentiment", "neutral"),
            model_used=model,
        )

    except Exception as e:
        logger.warning("comms extraction AI failed: %s", e)
        return ExtractCommsResponse(
            entities={},
            action_items=[],
            sentiment="neutral",
            model_used="unavailable",
        )
