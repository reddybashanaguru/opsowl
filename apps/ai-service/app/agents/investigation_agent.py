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

from app.agents.tools import TOOL_SPECS, execute_tool, to_ollama_tools
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

async def _run_ollama(case_id: str, goal: str) -> AsyncIterator[str]:
    """
    Ollama tool_use path for fully air-gapped on-premise deployments.
    Requires a model with function-calling support (qwen2.5:7b, mistral-nemo, etc.).
    Uses Ollama's OpenAI-compatible /api/chat endpoint with tools parameter.
    """
    model = settings.ollama_model
    ollama_tools = to_ollama_tools()

    messages: list[dict] = [
        {"role": "system", "content": _SYSTEM_PROMPT},
        {"role": "user", "content": f"Goal: {goal}\n\nCase ID: {case_id}"},
    ]
    t0 = time.monotonic()
    total_tok_in = 0
    total_tok_out = 0
    iterations = 0

    async with httpx.AsyncClient(base_url=settings.ollama_base_url, timeout=180.0) as http:
        while iterations < MAX_ITERATIONS:
            iterations += 1

            payload = {
                "model": model,
                "messages": messages,
                "tools": ollama_tools,
                "stream": False,
            }

            try:
                resp = await http.post("/api/chat", json=payload)
                resp.raise_for_status()
                data = resp.json()
            except Exception as exc:
                yield _sse("error", {"message": f"Ollama error: {exc}"})
                return

            total_tok_in += data.get("prompt_eval_count", 0)
            total_tok_out += data.get("eval_count", 0)

            msg = data.get("message", {})
            content_text: str = msg.get("content", "") or ""
            tool_calls: list[dict] = msg.get("tool_calls") or []

            if content_text.strip():
                yield _sse("thinking", {"text": content_text})

            # Append assistant message to history
            messages.append({"role": "assistant", "content": content_text, "tool_calls": tool_calls})

            if not tool_calls:
                # Model is done — content_text is the final answer
                yield _sse("answer", {"text": content_text.strip()})
                break

            # Execute each tool call
            for tc in tool_calls:
                fn = tc.get("function", {})
                name = fn.get("name", "")
                raw_args = fn.get("arguments", "{}")
                try:
                    tool_input = json.loads(raw_args) if isinstance(raw_args, str) else raw_args
                except json.JSONDecodeError:
                    tool_input = {}

                call_id = tc.get("id", f"call_{iterations}_{name}")

                yield _sse("tool_call", {"id": call_id, "name": name, "input": tool_input})

                try:
                    result_text = await execute_tool(name, tool_input, case_id)
                    ok = True
                except Exception as exc:
                    result_text = f"Tool execution error: {exc}"
                    ok = False
                    logger.warning("Tool %s failed: %s", name, exc)

                yield _sse("tool_result", {
                    "id": call_id,
                    "name": name,
                    "preview": result_text[:300],
                    "ok": ok,
                })

                messages.append({
                    "role": "tool",
                    "content": result_text,
                    "name": name,
                })

        else:
            yield _sse("thinking", {"text": f"[Reached maximum of {MAX_ITERATIONS} iterations]"})

    yield _sse("done", {
        "iterations": iterations,
        "tokens_in": total_tok_in,
        "tokens_out": total_tok_out,
        "elapsed_ms": int((time.monotonic() - t0) * 1000),
    })
