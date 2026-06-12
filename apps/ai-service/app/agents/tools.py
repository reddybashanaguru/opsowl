"""
Investigation Agent — tool definitions and DB-backed implementations.

All tools are READ-ONLY. The agent can observe case data but cannot modify it.
Write operations require explicit human action in the UI.

Tools query the shared DB pool directly — no HTTP round-trip to Go API.
psycopg3 async API: conn.execute(sql, params) → cursor → cursor.fetchall()
"""
from __future__ import annotations

import json
import logging
from typing import Any

from app.db import get_pool

logger = logging.getLogger(__name__)

# ── Anthropic tool_use specs ──────────────────────────────────────────────────

TOOL_SPECS: list[dict] = [
    {
        "name": "get_case_context",
        "description": (
            "Fetch full case details: case number, title, status, priority, description, "
            "assigned analyst, SLA deadline, tags, and dates. "
            "Always call this first to understand what the case is about."
        ),
        "input_schema": {
            "type": "object",
            "properties": {
                "case_id": {"type": "string", "description": "The case UUID"},
            },
            "required": ["case_id"],
        },
    },
    {
        "name": "get_case_notes",
        "description": (
            "Fetch the most recent investigation notes written by analysts. "
            "Notes contain analyst observations, findings, and commentary."
        ),
        "input_schema": {
            "type": "object",
            "properties": {
                "case_id": {"type": "string"},
                "limit": {
                    "type": "integer",
                    "default": 10,
                    "description": "Max notes to return (1–20)",
                },
            },
            "required": ["case_id"],
        },
    },
    {
        "name": "get_case_tasks",
        "description": (
            "Fetch all investigation tasks for the case — what work has been done "
            "and what is still pending. Each task has a status (todo, in_progress, done, cancelled) "
            "and priority."
        ),
        "input_schema": {
            "type": "object",
            "properties": {
                "case_id": {"type": "string"},
            },
            "required": ["case_id"],
        },
    },
    {
        "name": "get_case_findings",
        "description": (
            "Fetch documented findings for this case — specific observations, "
            "evidence items, and their severity and approval status. "
            "Findings are the formal record of suspicious activity identified."
        ),
        "input_schema": {
            "type": "object",
            "properties": {
                "case_id": {"type": "string"},
            },
            "required": ["case_id"],
        },
    },
    {
        "name": "get_case_transactions",
        "description": (
            "Fetch financial transactions linked to this case. Returns amount, currency, "
            "type (debit/credit), channel (NEFT/RTGS/UPI/cash), counterparty names, "
            "and risk signals (flagged, risk_score)."
        ),
        "input_schema": {
            "type": "object",
            "properties": {
                "case_id": {"type": "string"},
                "limit": {
                    "type": "integer",
                    "default": 30,
                    "description": "Max transactions to return (1–100)",
                },
            },
            "required": ["case_id"],
        },
    },
    {
        "name": "get_case_timeline",
        "description": (
            "Fetch the audit trail of all events on this case — status changes, "
            "notes added, tasks completed, documents uploaded, AI queries, etc. "
            "Useful for understanding the investigation history."
        ),
        "input_schema": {
            "type": "object",
            "properties": {
                "case_id": {"type": "string"},
                "limit": {"type": "integer", "default": 20},
            },
            "required": ["case_id"],
        },
    },
    {
        "name": "get_case_documents",
        "description": (
            "Fetch the list of documents uploaded to this case, "
            "with file type, page count, and an extracted text excerpt from each."
        ),
        "input_schema": {
            "type": "object",
            "properties": {
                "case_id": {"type": "string"},
            },
            "required": ["case_id"],
        },
    },
    {
        "name": "search_similar_cases",
        "description": (
            "Full-text search across all cases to find similar investigations. "
            "Useful for identifying patterns, precedents, and benchmarking risk decisions."
        ),
        "input_schema": {
            "type": "object",
            "properties": {
                "query": {
                    "type": "string",
                    "description": "Keywords to search (e.g. 'layering cryptocurrency exchange offshore')",
                },
                "limit": {"type": "integer", "default": 5},
            },
            "required": ["query"],
        },
    },
]


def to_ollama_tools(specs: list[dict]) -> list[dict]:
    """Convert Anthropic tool specs to Ollama/OpenAI function-calling format."""
    return [
        {
            "type": "function",
            "function": {
                "name": s["name"],
                "description": s["description"],
                "parameters": s["input_schema"],
            },
        }
        for s in specs
    ]


# ── Tool implementations ──────────────────────────────────────────────────────

async def execute_tool(name: str, tool_input: dict, case_id: str) -> str:
    """Dispatch a tool call by name and return a text result for the agent."""
    fn = _REGISTRY.get(name)
    if fn is None:
        return f"Unknown tool: {name}"
    try:
        return await fn(tool_input, case_id)
    except Exception as exc:
        logger.warning("Tool %s failed: %s", name, exc, exc_info=True)
        return f"[Tool error — {name}: {exc}]"


async def _get_case_context(inp: dict, case_id: str) -> str:
    cid = inp.get("case_id", case_id)
    pool = await get_pool()
    async with pool.connection() as conn:
        cur = await conn.execute(
            """
            SELECT c.case_number, c.title, c.status, c.priority, c.description,
                   u.full_name AS assigned_to, c.sla_breach_at, c.sla_breached,
                   c.tags, c.created_at, c.updated_at
            FROM cases c
            LEFT JOIN users u ON u.id = c.assigned_to
            WHERE c.id = %s::uuid AND c.deleted_at IS NULL
            """,
            (cid,),
        )
        row = await cur.fetchone()

    if not row:
        return "Case not found."

    sla = row[6].strftime("%Y-%m-%d %H:%M UTC") if row[6] else "None"
    breached = " ⚠️ SLA BREACHED" if row[7] else ""
    tags = ", ".join(row[8]) if row[8] else "None"
    return (
        f"Case: {row[0]} — {row[1]}\n"
        f"Status: {row[2]} | Priority: {row[3]}{breached}\n"
        f"Assigned to: {row[5] or 'Unassigned'} | SLA deadline: {sla}\n"
        f"Tags: {tags}\n"
        f"Created: {row[9].strftime('%Y-%m-%d')} | Updated: {row[10].strftime('%Y-%m-%d')}\n\n"
        f"Description:\n{row[4] or '(no description provided)'}"
    )


async def _get_case_notes(inp: dict, case_id: str) -> str:
    cid = inp.get("case_id", case_id)
    limit = min(int(inp.get("limit", 10)), 20)
    pool = await get_pool()
    async with pool.connection() as conn:
        cur = await conn.execute(
            """
            SELECT n.content_text, u.full_name, n.created_at, n.note_type
            FROM notes n
            LEFT JOIN users u ON u.id = n.author_id
            WHERE n.case_id = %s::uuid AND n.deleted_at IS NULL
            ORDER BY n.created_at DESC
            LIMIT %s
            """,
            (cid, limit),
        )
        rows = await cur.fetchall()

    if not rows:
        return "No notes found for this case."

    parts = []
    for r in rows:
        ts = r[2].strftime("%Y-%m-%d %H:%M")
        parts.append(f"[{ts}] {r[1] or 'Unknown'} ({r[3]}):\n{r[0]}")
    return "\n\n---\n\n".join(parts)


async def _get_case_tasks(inp: dict, case_id: str) -> str:
    cid = inp.get("case_id", case_id)
    pool = await get_pool()
    async with pool.connection() as conn:
        cur = await conn.execute(
            """
            SELECT t.title, t.status, t.priority, u.full_name AS assignee,
                   t.due_date, t.completed_at
            FROM tasks t
            LEFT JOIN users u ON u.id = t.assigned_to
            WHERE t.case_id = %s::uuid AND t.deleted_at IS NULL
            ORDER BY
                CASE t.priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2
                                WHEN 'medium' THEN 3 ELSE 4 END,
                t.created_at ASC
            """,
            (cid,),
        )
        rows = await cur.fetchall()

    if not rows:
        return "No tasks found."

    done = sum(1 for r in rows if r[1] == "done")
    pending = len(rows) - done
    lines = [f"Tasks: {len(rows)} total — {done} done, {pending} pending\n"]
    for r in rows:
        due = r[4].strftime("%Y-%m-%d") if r[4] else "-"
        check = "✓" if r[1] == "done" else ("→" if r[1] == "in_progress" else "○")
        lines.append(
            f"{check} [{r[1].upper()}] {r[0]} (priority: {r[2]}, "
            f"assignee: {r[3] or 'unassigned'}, due: {due})"
        )
    return "\n".join(lines)


async def _get_case_findings(inp: dict, case_id: str) -> str:
    cid = inp.get("case_id", case_id)
    pool = await get_pool()
    async with pool.connection() as conn:
        cur = await conn.execute(
            """
            SELECT f.title, f.description, f.category, f.severity, f.status,
                   f.narrative_weight, u.full_name AS author, f.created_at
            FROM case_findings f
            LEFT JOIN users u ON u.id = f.created_by
            WHERE f.case_id = %s::uuid AND f.deleted_at IS NULL
            ORDER BY
                CASE f.severity WHEN 'critical' THEN 1 WHEN 'high' THEN 2
                               WHEN 'medium' THEN 3 ELSE 4 END,
                f.created_at DESC
            """,
            (cid,),
        )
        rows = await cur.fetchall()

    if not rows:
        return "No findings documented yet for this case."

    parts = []
    for r in rows:
        ts = r[7].strftime("%Y-%m-%d")
        parts.append(
            f"[{r[4].upper()}] {r[0]}\n"
            f"Category: {r[2]} | Severity: {r[3]} | Weight: {r[5]}\n"
            f"By: {r[6] or 'Unknown'} on {ts}\n"
            f"{r[1][:500]}"
        )
    return "\n\n---\n\n".join(parts)


async def _get_case_transactions(inp: dict, case_id: str) -> str:
    cid = inp.get("case_id", case_id)
    limit = min(int(inp.get("limit", 30)), 100)
    pool = await get_pool()
    async with pool.connection() as conn:
        cur = await conn.execute(
            """
            SELECT t.txn_date, t.amount, t.currency, t.txn_type, t.channel,
                   t.from_name_raw, t.to_name_raw, t.description, t.narration,
                   t.is_flagged, t.risk_score, t.flag_reason, ct.relevance_note
            FROM transactions t
            JOIN case_transactions ct ON ct.transaction_id = t.id
            WHERE ct.case_id = %s::uuid
            ORDER BY t.txn_date DESC, t.amount DESC
            LIMIT %s
            """,
            (cid, limit),
        )
        rows = await cur.fetchall()

    if not rows:
        return "No transactions linked to this case."

    total_debit  = sum(float(r[1]) for r in rows if r[3] == "debit")
    total_credit = sum(float(r[1]) for r in rows if r[3] == "credit")
    flagged      = sum(1 for r in rows if r[9])

    summary = (
        f"Transactions: {len(rows)} total | Flagged: {flagged}\n"
        f"Total debits: {total_debit:,.2f} | Total credits: {total_credit:,.2f}\n\n"
    )
    lines = []
    for r in rows:
        flag = " ⚠️" if r[9] else ""
        risk = f" [risk:{r[10]}]" if r[10] > 0 else ""
        desc = r[7] or r[8] or "-"
        lines.append(
            f"{r[0]} | {r[3].upper()} {float(r[1]):,.2f} {r[2]} via {r[4]} "
            f"| {r[5] or '?'} → {r[6] or '?'} | {desc[:80]}{flag}{risk}"
        )
    return summary + "\n".join(lines)


async def _get_case_timeline(inp: dict, case_id: str) -> str:
    cid = inp.get("case_id", case_id)
    limit = min(int(inp.get("limit", 20)), 50)
    pool = await get_pool()
    async with pool.connection() as conn:
        cur = await conn.execute(
            """
            SELECT te.event_type, te.source, te.occurred_at, u.full_name AS actor
            FROM timeline_events te
            LEFT JOIN users u ON u.id = te.actor_id
            WHERE te.case_id = %s::uuid AND te.deleted_at IS NULL
            ORDER BY te.occurred_at DESC
            LIMIT %s
            """,
            (cid, limit),
        )
        rows = await cur.fetchall()

    if not rows:
        return "No timeline events found."

    lines = []
    for r in rows:
        ts = r[2].strftime("%Y-%m-%d %H:%M")
        actor = r[3] or r[1]
        lines.append(f"[{ts}] {r[0].replace('_', ' ')} — {actor}")
    return "\n".join(lines)


async def _get_case_documents(inp: dict, case_id: str) -> str:
    cid = inp.get("case_id", case_id)
    pool = await get_pool()
    async with pool.connection() as conn:
        cur = await conn.execute(
            """
            SELECT d.original_filename, d.file_type, d.page_count,
                   d.file_size_bytes, d.processing_status, d.created_at,
                   LEFT(dc_agg.text_excerpt, 400) AS excerpt
            FROM documents d
            LEFT JOIN LATERAL (
                SELECT string_agg(content, ' ' ORDER BY chunk_index) AS text_excerpt
                FROM document_chunks
                WHERE document_id = d.id
                LIMIT 3
            ) dc_agg ON true
            WHERE d.case_id = %s::uuid AND d.deleted_at IS NULL
            ORDER BY d.created_at DESC
            """,
            (cid,),
        )
        rows = await cur.fetchall()

    if not rows:
        return "No documents uploaded to this case."

    parts = []
    for r in rows:
        size_kb = (r[3] or 0) // 1024
        ts = r[5].strftime("%Y-%m-%d")
        excerpt = (r[6] or "").strip()
        status_note = "" if r[4] == "completed" else f" [{r[4]}]"
        parts.append(
            f"📄 {r[0]} ({r[1]}, {r[2] or '?'} pages, {size_kb}KB, {ts}){status_note}\n"
            f"Excerpt: {excerpt[:350] if excerpt else '(not yet extracted)'}"
        )
    return "\n\n".join(parts)


async def _search_similar_cases(inp: dict, _case_id: str) -> str:
    query = (inp.get("query") or "").strip()
    limit = min(int(inp.get("limit", 5)), 10)
    if not query:
        return "Provide a search query."
    pool = await get_pool()
    async with pool.connection() as conn:
        cur = await conn.execute(
            """
            SELECT c.case_number, c.title, c.status, c.priority, c.created_at
            FROM cases c
            WHERE c.deleted_at IS NULL
              AND to_tsvector('english',
                    COALESCE(c.title, '') || ' ' || COALESCE(c.description, ''))
                  @@ plainto_tsquery('english', %s)
            ORDER BY
                ts_rank(
                    to_tsvector('english',
                        COALESCE(c.title,'') || ' ' || COALESCE(c.description,'')),
                    plainto_tsquery('english', %s)
                ) DESC
            LIMIT %s
            """,
            (query, query, limit),
        )
        rows = await cur.fetchall()

    if not rows:
        return f"No similar cases found for: '{query}'"
    lines = [f"Found {len(rows)} similar case(s) for '{query}':"]
    for r in rows:
        ts = r[4].strftime("%Y-%m")
        lines.append(f"• {r[0]} — {r[1]} [{r[2]}, {r[3]}, {ts}]")
    return "\n".join(lines)


_REGISTRY: dict[str, Any] = {
    "get_case_context":      _get_case_context,
    "get_case_notes":        _get_case_notes,
    "get_case_tasks":        _get_case_tasks,
    "get_case_findings":     _get_case_findings,
    "get_case_transactions": _get_case_transactions,
    "get_case_timeline":     _get_case_timeline,
    "get_case_documents":    _get_case_documents,
    "search_similar_cases":  _search_similar_cases,
}
