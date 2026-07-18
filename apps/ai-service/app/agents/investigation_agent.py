"""
Investigation Agent — autonomous multi-step reasoning loop for case investigation.

Architecture:
  Anthropic: native tool_use via AsyncAnthropic.messages.create()
  Ollama:    OpenAI-compatible tool_use via /api/chat (qwen2.5:7b, mistral-nemo, etc.)

Agent loop (both providers):
  1. POST initial goal + case_id
  2. If response contains tool calls → execute → append results → loop
  3. If stop_reason == end_turn → emit final answer → done
  4. Hard limit: MAX_ITERATIONS (prevents runaway costs)

SSE event stream:
  agent_start   {"case_id": "...", "goal": "..."}
  thinking      {"text": "..."}
  tool_call     {"id": "...", "name": "...", "input": {...}}
  tool_result   {"id": "...", "name": "...", "preview": "...", "ok": true}
  answer        {"text": "..."}
  done          {"iterations": N, "tokens_in": N, "tokens_out": N, "elapsed_ms": N}
  error         {"message": "..."}
"""
from __future__ import annotations

import json
import logging
import time
from typing import AsyncIterator

import httpx

from app.agents.tools import TOOL_SPECS, execute_tool, to_ollama_tools  # TOOL_SPECS/to_ollama_tools used by Anthropic path
from app.config import settings
from app.services.llm_router import Provider

logger = logging.getLogger(__name__)

MAX_ITERATIONS = 10

_SYSTEM_PROMPT = """You are OpsOwl Investigator, an autonomous AI agent embedded in an enterprise financial crime investigation platform used by compliance teams at banks and NBFCs.

You have read-only access to case data through tools. Use them methodically.

Investigation protocol:
1. Call get_case_context first to understand the case
2. Gather evidence from notes, tasks, findings, transactions, and timeline
3. If the case involves financial flows, call get_case_transactions
4. Search for similar past cases using search_similar_cases
5. Synthesise all gathered data into a structured report

Rules:
- Base every claim strictly on data returned by tools — never fabricate
- Do not call the same tool twice with identical inputs
- If data is missing or inconclusive, state that explicitly
- Be concise in tool inputs; case_id is always the same one you were given

Final answer format (markdown):
## Case Summary
## Key Findings
## Risk Indicators
## Recommended Next Steps
## Confidence Level: HIGH / MEDIUM / LOW
"""


def _sse(event: str, data: dict) -> str:
    return f"event: {event}\ndata: {json.dumps(data)}\n\n"


async def run_agent(case_id: str, goal: str) -> AsyncIterator[str]:
    """Top-level async generator. Yields SSE strings for the full agent run."""
    yield _sse("agent_start", {"case_id": case_id, "goal": goal})

    provider = Provider(settings.default_provider)

    try:
        if provider == Provider.OLLAMA:
            async for chunk in _run_ollama(case_id, goal):
                yield chunk
        else:
            async for chunk in _run_anthropic(case_id, goal):
                yield chunk
    except Exception as exc:
        logger.exception("Agent fatal error for case %s", case_id)
        yield _sse("error", {"message": str(exc)})


# ── Anthropic path ────────────────────────────────────────────────────────────

async def _run_anthropic(case_id: str, goal: str) -> AsyncIterator[str]:
    import anthropic  # lazy import — only needed for Anthropic provider

    client = anthropic.AsyncAnthropic(api_key=settings.anthropic_api_key)
    messages: list[dict] = [
        {"role": "user", "content": f"Goal: {goal}\n\nCase ID: {case_id}"}
    ]
    t0 = time.monotonic()
    total_tok_in = 0
    total_tok_out = 0
    iterations = 0

    while iterations < MAX_ITERATIONS:
        iterations += 1

        response = await client.messages.create(
            model=settings.default_model,
            max_tokens=4096,
            system=_SYSTEM_PROMPT,
            tools=TOOL_SPECS,
            messages=messages,
        )

        total_tok_in += response.usage.input_tokens
        total_tok_out += response.usage.output_tokens

        # Emit reasoning text (visible to analyst while agent thinks)
        for block in response.content:
            if block.type == "text" and block.text.strip():
                yield _sse("thinking", {"text": block.text})

        if response.stop_reason == "end_turn":
            final_text = "\n".join(
                b.text for b in response.content if b.type == "text"
            ).strip()
            yield _sse("answer", {"text": final_text})
            break

        if response.stop_reason != "tool_use":
            logger.warning("Unexpected stop_reason: %s", response.stop_reason)
            break

        tool_use_blocks = [b for b in response.content if b.type == "tool_use"]

        # Append assistant turn to message history (raw dict form)
        assistant_content = []
        for block in response.content:
            if block.type == "text":
                assistant_content.append({"type": "text", "text": block.text})
            elif block.type == "tool_use":
                assistant_content.append({
                    "type": "tool_use",
                    "id": block.id,
                    "name": block.name,
                    "input": block.input,
                })
        messages.append({"role": "assistant", "content": assistant_content})

        # Execute tools and collect results
        tool_results = []
        for block in tool_use_blocks:
            yield _sse("tool_call", {
                "id": block.id,
                "name": block.name,
                "input": block.input,
            })

            try:
                result_text = await execute_tool(block.name, block.input, case_id)
                ok = True
            except Exception as exc:
                result_text = f"Tool execution error: {exc}"
                ok = False
                logger.warning("Tool %s failed: %s", block.name, exc)

            yield _sse("tool_result", {
                "id": block.id,
                "name": block.name,
                "preview": result_text[:300],
                "ok": ok,
            })

            tool_results.append({
                "type": "tool_result",
                "tool_use_id": block.id,
                "content": result_text,
            })

        messages.append({"role": "user", "content": tool_results})

    else:
        # Hit MAX_ITERATIONS — emit what we have
        yield _sse("thinking", {"text": f"[Reached maximum of {MAX_ITERATIONS} iterations]"})

    yield _sse("done", {
        "iterations": iterations,
        "tokens_in": total_tok_in,
        "tokens_out": total_tok_out,
        "elapsed_ms": int((time.monotonic() - t0) * 1000),
    })


# ── Ollama path (on-premise, zero egress) ────────────────────────────────────

# Ordered list of tools to gather before synthesis. Each entry is
# (tool_name, input_overrides). case_id is always injected automatically.
_GATHER_STEPS = [
    ("get_case_context", {}),
    ("get_case_notes", {"limit": 10}),
    ("get_case_tasks", {}),
    ("get_case_findings", {}),
    ("get_case_transactions", {"limit": 30}),
    ("get_case_timeline", {"limit": 15}),
]

_ANALYSIS_SYSTEM = (
    "You are OpsOwl Investigator, an AI financial crime analyst.\n"
    "You are given pre-gathered case data. Analyse it and produce a structured investigation report.\n"
    "Base every claim strictly on the data provided — never fabricate.\n"
    "Be concise and professional. Write in the style of a financial crime analyst."
)


async def _run_ollama(case_id: str, goal: str) -> AsyncIterator[str]:
    """
    Gather-then-analyse approach for on-premise Ollama models.

    Avoids multi-turn tool-calling (unreliable on 3B models). Instead:
      1. Pre-fetch all key case data via direct DB tool calls (same tools, shown in feed)
      2. Send everything to the model in one comprehensive prompt
      3. Emit the analysis as the final answer

    Compatible with llama3.2:3b and any Ollama model — no function-calling required.
    """
    model = settings.ollama_model
    t0 = time.monotonic()
    total_tok_in = 0
    total_tok_out = 0

    # ── Phase 1: gather case data ─────────────────────────────────────────────
    gathered: dict[str, str] = {}
    for step_num, (tool_name, extra_input) in enumerate(_GATHER_STEPS, start=1):
        call_id = f"gather_{step_num}"
        tool_input = {"case_id": case_id, **extra_input}

        yield _sse("tool_call", {"id": call_id, "name": tool_name, "input": tool_input})

        try:
            result = await execute_tool(tool_name, tool_input, case_id)
            gathered[tool_name] = result
            ok = True
        except Exception as exc:
            result = f"[unavailable: {exc}]"
            gathered[tool_name] = result
            ok = False
            logger.warning("Tool %s failed: %s", tool_name, exc)

        yield _sse("tool_result", {
            "id": call_id,
            "name": tool_name,
            "preview": result[:200],
            "ok": ok,
        })

    # ── Phase 2: build analysis prompt ───────────────────────────────────────
    context_block = "\n\n".join(
        f"=== {name.replace('get_case_', '').replace('_', ' ').upper()} ===\n{data}"
        for name, data in gathered.items()
        if data and not data.startswith("[unavailable")
    )

    analysis_prompt = (
        f"Analyst goal: {goal}\n\n"
        f"Case ID: {case_id}\n\n"
        f"--- GATHERED CASE DATA ---\n{context_block}\n\n"
        "--- TASK ---\n"
        "Based on the case data above, produce a structured investigation report with these sections:\n\n"
        "## Case Summary\n"
        "## Key Findings\n"
        "## Risk Indicators\n"
        "## Recommended Next Steps\n"
        "## Confidence Level: HIGH / MEDIUM / LOW\n\n"
        "Write each section with bullet points where appropriate. "
        "Reference specific amounts, dates, counterparties, and entities from the data."
    )

    yield _sse("thinking", {"text": "Analysing gathered data and generating investigation report…"})

    # ── Phase 3: single LLM call for synthesis ───────────────────────────────
    payload = {
        "model": model,
        "messages": [
            {"role": "system", "content": _ANALYSIS_SYSTEM},
            {"role": "user", "content": analysis_prompt},
        ],
        "stream": False,
        "options": {"num_predict": 1200, "temperature": 0.1},
    }

    async with httpx.AsyncClient(base_url=settings.ollama_base_url, timeout=180.0) as http:
        try:
            resp = await http.post("/api/chat", json=payload)
            resp.raise_for_status()
            data = resp.json()
        except Exception as exc:
            yield _sse("error", {"message": f"Ollama error during analysis: {exc}"})
            return

    total_tok_in += data.get("prompt_eval_count", 0)
    total_tok_out += data.get("eval_count", 0)

    answer_text: str = (data.get("message", {}).get("content", "") or "").strip()

    if not answer_text:
        yield _sse("error", {"message": "Model returned an empty response."})
        return

    yield _sse("answer", {"text": answer_text})
    yield _sse("done", {
        "iterations": 1,
        "tokens_in": total_tok_in,
        "tokens_out": total_tok_out,
        "elapsed_ms": int((time.monotonic() - t0) * 1000),
    })
