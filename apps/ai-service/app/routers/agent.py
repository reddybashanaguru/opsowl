"""Agent endpoint — streams SSE events for an autonomous investigation run."""
from __future__ import annotations

import re

from fastapi import APIRouter, Depends, HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field

from app.agents.investigation_agent import run_agent
from app.service_auth import require_service_token

router = APIRouter(prefix="/agent", dependencies=[Depends(require_service_token)])

_UUID_RE = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", re.I
)

_INJECTION_PATTERNS = [
    re.compile(r"ignore (all )?(previous|prior|above) (instructions?|prompts?)", re.I),
    re.compile(r"you are now", re.I),
    re.compile(r"disregard (all |the |your )?", re.I),
    re.compile(r"forget (all |your )?(previous|prior|the )?instructions?", re.I),
    re.compile(r"jailbreak", re.I),
]


class AgentRunRequest(BaseModel):
    goal: str = Field(..., min_length=3, max_length=1000)


def _check_injection(text: str) -> None:
    for pattern in _INJECTION_PATTERNS:
        if pattern.search(text):
            raise HTTPException(status_code=400, detail="Prompt injection detected.")


@router.post("/{case_id}/run")
async def run_case_agent(case_id: str, req: AgentRunRequest) -> StreamingResponse:
    """
    Autonomous investigation agent for a single case.

    Streams SSE events:
      agent_start, thinking, tool_call, tool_result, answer, done, error

    Client must use fetch() + ReadableStream (not EventSource — POST body needed).
    """
    if not _UUID_RE.match(case_id):
        raise HTTPException(status_code=400, detail="Invalid case_id format.")

    _check_injection(req.goal)

    return StreamingResponse(
        run_agent(case_id, req.goal),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
            "Connection": "keep-alive",
        },
    )
