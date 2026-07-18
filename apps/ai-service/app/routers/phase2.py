"""Phase 2 AI endpoints — case risk analysis, comm entity extraction, report summary."""
from __future__ import annotations

import logging
from typing import Any

from fastapi import APIRouter, Depends
from pydantic import BaseModel

from app.service_auth import require_service_token
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


class ReportSummaryRequest(BaseModel):
    case_number: str
    title: str
    status: str
    priority: str
    description: str = ""
    task_count: int = 0
    document_count: int = 0


class ReportSummaryResponse(BaseModel):
    summary: str
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


def _template_narrative(req: "GenerateNarrativeRequest") -> "GenerateNarrativeResponse":
    """Deterministic fallback when the LLM is unavailable."""
    high = [f for f in req.findings if f.severity in ("high", "critical")]
    total_amt = sum(t.amount for t in req.transactions)
    flagged_count = sum(1 for t in req.transactions if t.is_flagged)

    if req.narrative_type == "str_suspicion_basis":
        body = (
            f"This Suspicious Transaction Report relates to case {req.case_number} concerning "
            f"{', '.join(req.subject_names) or 'the subject(s) of investigation'}. "
            f"The investigation identified {len(req.findings)} finding(s), of which "
            f"{len(high)} were classified as high or critical severity. "
        )
        if req.transactions:
            body += (
                f"A total of {len(req.transactions)} transaction(s) were reviewed with a combined "
                f"value of {req.transactions[0].currency} {total_amt:,.2f}. "
                f"{flagged_count} transaction(s) were flagged as suspicious. "
            )
        body += (
            f"Based on the findings and evidence gathered during this investigation, "
            f"the activity is assessed as meeting the threshold for suspicious transaction reporting "
            f"under the applicable AML/CFT framework."
        )
    elif req.narrative_type == "closure_summary":
        body = (
            f"Case {req.case_number} ({req.case_title}) was investigated following the generation of "
            f"an alert. The investigation reviewed {len(req.transactions)} financial transaction(s) "
            f"and identified {len(req.findings)} finding(s). "
            f"Risk level was assessed as {req.risk_level}. "
            f"Based on the evidence reviewed and analysis conducted, a closure determination "
            f"has been made as documented in this record."
        )
    else:
        body = (
            f"Investigation {req.case_number} was initiated to examine {req.case_title}. "
            f"The case was assessed at {req.risk_level} risk level. "
        )
        if req.findings:
            body += f"{len(req.findings)} finding(s) were documented, "
            if high:
                body += f"including {len(high)} high/critical severity finding(s). "
        if req.transactions:
            body += (
                f"Financial analysis covered {len(req.transactions)} transaction(s) totalling "
                f"{req.transactions[0].currency if req.transactions else 'INR'} {total_amt:,.2f}. "
            )
        body += (
            "All evidence and findings have been documented in the case record. "
            "This narrative was generated from structured case data without LLM assistance."
        )

    return GenerateNarrativeResponse(
        narrative=body,
        model_used="template",
        findings_used=len(req.findings),
        txns_used=len(req.transactions),
    )


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


# ── Report Executive Summary ──────────────────────────────────────────────────

@router.post("/report/summary", response_model=ReportSummaryResponse)
async def report_summary(req: ReportSummaryRequest) -> ReportSummaryResponse:
    """
    Generate a 3-4 sentence executive summary for a case report.
    Falls back to a structured template if LLM is unavailable.
    """
    try:
        prompt = (
            f"Write a concise executive summary (3-4 sentences, professional tone) for a "
            f"financial investigation case report. Use only these facts:\n\n"
            f"Case: {req.case_number} — {req.title}\n"
            f"Status: {req.status}\n"
            f"Priority: {req.priority}\n"
            f"Description: {req.description or 'Not provided'}\n"
            f"Tasks: {req.task_count} tasks\n"
            f"Documents: {req.document_count} documents collected\n\n"
            "Write in third person. Start with 'Case [case_number]...' or 'This investigation...' "
            "Do not use headers or bullet points. Plain paragraphs only."
        )
        system = (
            "You are a financial crime compliance officer writing formal investigation reports. "
            "Be concise, factual, and professional. No speculation."
        )
        messages = [{"role": "user", "content": prompt}]
        answer, model, _, _ = await llm_router.complete(messages, system=system, max_tokens=300)

        summary = answer.strip()
        if len(summary) < 20:
            raise ValueError("summary too short")

        return ReportSummaryResponse(summary=summary, model_used=model)

    except Exception as e:
        logger.warning("report summary AI failed: %s", e)
        summary = (
            f"Case {req.case_number} ({req.title}) is an investigation of {req.priority} priority "
            f"currently in {req.status.replace('_', ' ')} status. "
            f"The investigation has collected {req.document_count} document(s) and generated "
            f"{req.task_count} task(s). "
            f"This report provides a comprehensive record of all investigation activities, "
            f"evidence, communications, and findings to date."
        )
        return ReportSummaryResponse(summary=summary, model_used="template")


# ── AI Investigation Narrative ────────────────────────────────────────────────

class NarrativeFinding(BaseModel):
    title: str
    category: str
    severity: str
    description: str
    narrative_weight: str = "supporting"


class NarrativeTransaction(BaseModel):
    txn_ref: str
    txn_date: str
    amount: float
    currency: str = "INR"
    txn_type: str
    channel: str
    from_name: str = ""
    to_name: str = ""
    narration: str = ""
    is_flagged: bool = False
    flag_reason: str = ""


class SubjectProfile(BaseModel):
    name: str
    id_type: str = ""
    id_number: str = ""
    kyc_tier: str = ""
    risk_level: str = ""
    is_pep: bool = False
    notes: str = ""


class AIInsights(BaseModel):
    risk_score: float = 0.0
    risk_factors: list[str] = []
    anomalies: list[str] = []
    recommended_actions: list[str] = []


class GenerateNarrativeRequest(BaseModel):
    case_id: str
    case_number: str
    case_title: str
    case_description: str = ""
    risk_level: str = "unknown"
    narrative_type: str = "investigation_summary"
    # investigation_summary | str_suspicion_basis | sar_narrative | closure_summary
    findings: list[NarrativeFinding] = []
    transactions: list[NarrativeTransaction] = []
    subject_names: list[str] = []
    subject_profiles: list[SubjectProfile] = []
    ai_insights: AIInsights | None = None
    investigator_notes: str = ""


class GenerateNarrativeResponse(BaseModel):
    narrative: str
    model_used: str
    findings_used: int
    txns_used: int
    prompt_version: str = "v1"


@router.post("/narrative/generate", response_model=GenerateNarrativeResponse,
             dependencies=[Depends(require_service_token)])
async def generate_narrative(req: GenerateNarrativeRequest) -> GenerateNarrativeResponse:
    """
    Generate a structured investigation narrative for regulatory filing or case closure.
    The narrative is grounded in the provided findings, transactions, and case context.
    Falls back to a structured template when the LLM is unavailable.
    """
    type_instructions = {
        "investigation_summary": (
            "Write a comprehensive investigation narrative (5-8 paragraphs). "
            "Cover: (1) what triggered the investigation, (2) the subject's profile and behaviour pattern, "
            "(3) the key findings and evidence, (4) transaction analysis, (5) why the activity is suspicious, "
            "(6) the risk assessment. Write in formal third-person compliance language. No bullet points."
        ),
        "str_suspicion_basis": (
            "Write the 'Suspicion Basis' section for an FIU-IND Suspicious Transaction Report (STR). "
            "This must be factual, specific, and clearly explain WHY the transactions are suspicious. "
            "Reference specific amounts, dates, and patterns. 2-4 paragraphs. "
            "Use formal regulatory language appropriate for submission to FIU-IND."
        ),
        "sar_narrative": (
            "Write the Suspicious Activity Report (SAR) narrative. "
            "Include: (1) description of suspicious activity, (2) known subject information, "
            "(3) financial amounts and date range, (4) how this deviates from normal behaviour, "
            "(5) any law enforcement or internal referral context. "
            "Use FinCEN-style SAR narrative format. 3-5 paragraphs."
        ),
        "closure_summary": (
            "Write a concise case closure summary (2-3 paragraphs) suitable for the case closure form. "
            "Cover: what was investigated, what was found, the disposition rationale, and risk mitigation. "
            "This is the formal record of why the case was closed with this outcome."
        ),
    }

    instruction = type_instructions.get(req.narrative_type, type_instructions["investigation_summary"])

    # Build the context block
    findings_block = ""
    if req.findings:
        lines = [f"FINDINGS ({len(req.findings)} total):"]
        for f in req.findings[:20]:
            lines.append(
                f"  [{f.severity.upper()}] {f.title} (category: {f.category}, weight: {f.narrative_weight})\n"
                f"  {f.description[:300]}"
            )
        findings_block = "\n".join(lines)

    txns_block = ""
    if req.transactions:
        flagged = [t for t in req.transactions if t.is_flagged]
        all_txns = req.transactions[:30]
        lines = [f"TRANSACTIONS ({len(req.transactions)} total, {len(flagged)} flagged):"]
        for t in all_txns:
            flag_str = f" ⚑ FLAGGED: {t.flag_reason}" if t.is_flagged else ""
            lines.append(
                f"  {t.txn_date} | {t.txn_type.upper()} | {t.currency} {t.amount:,.2f} | "
                f"{t.channel.upper()} | {t.from_name or 'Unknown'} → {t.to_name or 'Unknown'}"
                f"{flag_str}"
            )
            if t.narration:
                lines.append(f"    Narration: {t.narration[:120]}")
        txns_block = "\n".join(lines)

    subjects_block = ""
    if req.subject_profiles:
        lines = ["SUBJECTS OF INVESTIGATION:"]
        for sp in req.subject_profiles:
            line = f"  - {sp.name}"
            if sp.id_type and sp.id_number:
                line += f" | {sp.id_type}: {sp.id_number}"
            if sp.kyc_tier:
                line += f" | KYC Tier: {sp.kyc_tier}"
            if sp.risk_level:
                line += f" | Risk: {sp.risk_level.upper()}"
            if sp.is_pep:
                line += " | PEP: YES"
            if sp.notes:
                line += f"\n    Profile notes: {sp.notes[:300]}"
            lines.append(line)
        subjects_block = "\n".join(lines)
    elif req.subject_names:
        subjects_block = f"SUBJECTS OF INVESTIGATION: {', '.join(req.subject_names)}"

    insights_block = ""
    if req.ai_insights and (req.ai_insights.risk_factors or req.ai_insights.anomalies):
        lines = [f"AI RISK ANALYSIS (risk score: {req.ai_insights.risk_score:.0f}/100):"]
        if req.ai_insights.risk_factors:
            lines.append("  Risk Factors Identified:")
            for rf in req.ai_insights.risk_factors:
                lines.append(f"    • {rf}")
        if req.ai_insights.anomalies:
            lines.append("  Transaction Anomalies Detected:")
            for a in req.ai_insights.anomalies:
                lines.append(f"    • {a}")
        insights_block = "\n".join(lines)

    context_parts = [
        f"CASE: {req.case_number} — {req.case_title}",
        f"RISK LEVEL: {req.risk_level.upper()}",
    ]
    if req.case_description:
        context_parts.append(f"DESCRIPTION: {req.case_description[:500]}")
    if subjects_block:
        context_parts.append(subjects_block)
    if insights_block:
        context_parts.append(insights_block)
    if findings_block:
        context_parts.append(findings_block)
    if txns_block:
        context_parts.append(txns_block)
    if req.investigator_notes:
        context_parts.append(f"INVESTIGATOR NOTES: {req.investigator_notes[:400]}")

    case_context = "\n\n".join(context_parts)

    prompt = (
        f"{instruction}\n\n"
        f"Use ONLY the following case data — do not invent facts. "
        f"CRITICAL: All subject occupation, employment, and income details must come EXCLUSIVELY "
        f"from the SUBJECTS OF INVESTIGATION section below. "
        f"CRITICAL: Use the aggregate total from the AI RISK ANALYSIS section — "
        f"do not independently sum transactions.\n\n"
        f"{case_context}\n\n"
        "Write the narrative now. Do not include headers, titles, or section labels. "
        "Plain paragraphs only."
    )
    system = (
        "You are a senior financial crime compliance officer writing formal investigation documentation "
        "for regulatory review. Your writing is factual, precise, and defensible under regulatory scrutiny. "
        "Never speculate beyond the provided evidence. Never invent amounts, dates, names, or occupations. "
        "Use only facts explicitly stated in the case data provided."
    )

    try:
        messages = [{"role": "user", "content": prompt}]
        answer, model, _, _ = await llm_router.complete(messages, system=system, max_tokens=2048)

        narrative = answer.strip()
        if len(narrative) < 100:
            raise ValueError("narrative too short")

        return GenerateNarrativeResponse(
            narrative=narrative,
            model_used=model,
            findings_used=len(req.findings),
            txns_used=len(req.transactions),
        )

    except Exception as e:
        logger.warning("narrative generation AI failed, using template: %s", e)
        return _template_narrative(req)
