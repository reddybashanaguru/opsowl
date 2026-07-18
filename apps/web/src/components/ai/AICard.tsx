"use client";

import React, { useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";
import { Citation, CitationCard } from "./CitationCard";
import { ConfidenceBar } from "./ConfidenceBar";
import { AIReviewBadge } from "./AIReviewBadge";

const ACTION_LABELS: Record<string, string> = {
  ask:                "Q&A",
  summarize:          "Summary",
  generate_note:      "Draft Note",
  extract_actions:    "Action Items",
  pattern_detect:     "Pattern Detection",
  generate_narrative: "Narrative Generation",
  process_document:   "Document Processing",
  agent_run:          "Agent Report",
};

function parseResponseText(actionType: string, raw: string): string {
  if (actionType === "generate_narrative") {
    try {
      const parsed = JSON.parse(raw) as Record<string, unknown>;
      if (typeof parsed.narrative === "string" && parsed.narrative.trim()) {
        return parsed.narrative;
      }
    } catch {
      // raw is plain text already
    }
  }
  return raw;
}

export function renderMarkdown(text: string): React.ReactNode {
  const lines = text.split("\n");
  const nodes: React.ReactNode[] = [];
  let listItems: string[] = [];
  let listType: "ul" | "ol" | null = null;

  const flushList = (key: string) => {
    if (!listItems.length) return;
    if (listType === "ol") {
      nodes.push(
        <ol key={key} className="list-decimal list-inside space-y-0.5 my-1 text-slate-300">
          {listItems.map((item, i) => <li key={i}>{inlineMarkdown(item)}</li>)}
        </ol>
      );
    } else {
      nodes.push(
        <ul key={key} className="list-disc list-inside space-y-0.5 my-1 text-slate-300">
          {listItems.map((item, i) => <li key={i}>{inlineMarkdown(item)}</li>)}
        </ul>
      );
    }
    listItems = [];
    listType = null;
  };

  const inlineMarkdown = (s: string): React.ReactNode => {
    const parts = s.split(/(\*\*[^*]+\*\*|\*[^*]+\*)/g);
    return parts.map((p, i) => {
      if (p.startsWith("**") && p.endsWith("**")) {
        return <strong key={i} className="text-slate-100 font-semibold">{p.slice(2, -2)}</strong>;
      }
      if (p.startsWith("*") && p.endsWith("*")) {
        return <em key={i}>{p.slice(1, -1)}</em>;
      }
      return p;
    });
  };

  lines.forEach((line, idx) => {
    const h3 = line.match(/^###\s+(.+)/);
    const h2 = line.match(/^##\s+(.+)/);
    const h1 = line.match(/^#\s+(.+)/);
    const ulItem = line.match(/^[\*\-]\s+(.+)/);
    const olItem = line.match(/^\d+\.\s+(.+)/);

    if (h3) {
      flushList(`l-${idx}`);
      nodes.push(<h3 key={idx} className="text-slate-100 text-sm font-semibold mt-3 mb-1">{inlineMarkdown(h3[1])}</h3>);
    } else if (h2) {
      flushList(`l-${idx}`);
      nodes.push(<h2 key={idx} className="text-slate-100 text-sm font-bold mt-3 mb-1">{inlineMarkdown(h2[1])}</h2>);
    } else if (h1) {
      flushList(`l-${idx}`);
      nodes.push(<h1 key={idx} className="text-slate-100 text-base font-bold mt-2 mb-1">{inlineMarkdown(h1[1])}</h1>);
    } else if (ulItem) {
      if (listType === "ol") flushList(`l-${idx}`);
      listType = "ul";
      listItems.push(ulItem[1]);
    } else if (olItem) {
      if (listType === "ul") flushList(`l-${idx}`);
      listType = "ol";
      listItems.push(olItem[1]);
    } else if (line.trim() === "") {
      flushList(`l-${idx}`);
      nodes.push(<div key={idx} className="h-1" />);
    } else {
      flushList(`l-${idx}`);
      nodes.push(<p key={idx} className="text-slate-300 leading-relaxed">{inlineMarkdown(line)}</p>);
    }
  });
  flushList("end");

  return <>{nodes}</>;
}

export interface AILog {
  id: string;
  case_id: string;
  action_type: string;
  response_text: string;
  model_used: string;
  provider: string;
  tokens_input: number;
  tokens_output: number;
  latency_ms: number;
  confidence_score?: number | null;
  citations: Citation[];
  human_review_status: string;
  insufficient_context: boolean;
  triggered_by: { id: string; full_name: string; email: string };
  reviewed_by?: { id: string; full_name: string } | null;
  reviewed_at?: string | null;
  human_edited_response?: string | null;
  created_at: string;
}

interface AICardProps {
  log: AILog;
  onApprove?: (id: string) => void;
  onReject?: (id: string) => void;
  onEdit?: (id: string, text: string) => void;
  canReview?: boolean;
  defaultExpanded?: boolean;
}

export function AICard({ log, onApprove, onReject, onEdit, canReview, defaultExpanded = false }: AICardProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);
  const [showCitations, setShowCitations] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editText, setEditText] = useState(log.human_edited_response ?? parseResponseText(log.action_type, log.response_text));
  const [rejectReason, setRejectReason] = useState("");
  const [showReject, setShowReject] = useState(false);

  const displayText = log.human_edited_response ?? parseResponseText(log.action_type, log.response_text);
  const latencyS = (log.latency_ms / 1000).toFixed(1);
  const pending = log.human_review_status === "pending";

  // Force-expand when user opens edit/reject flow
  const handleEdit = () => { setExpanded(true); setEditing(true); setEditText(parseResponseText(log.action_type, log.response_text)); };
  const handleReject = () => { setExpanded(true); setShowReject(true); };

  return (
    <div
      className={`bg-[#131f30] border border-l-4 rounded-xl overflow-hidden transition-all ${
        pending && canReview
          ? "border-slate-700 border-l-amber-400"
          : log.human_review_status === "approved"
          ? "border-slate-700/50 border-l-emerald-500"
          : log.human_review_status === "rejected"
          ? "border-slate-700/50 border-l-red-500"
          : "border-slate-700 border-l-violet-500"
      }`}
      data-testid="ai-card"
    >
      {/* ── Collapsed header — always visible ── */}
      <button
        className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-white/[0.02] transition-colors"
        onClick={() => { if (!editing && !showReject) setExpanded((v) => !v); }}
      >
        {/* Left: badges */}
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <span className="bg-violet-500/20 text-violet-300 border border-violet-500/40 text-[10px] font-bold px-2 py-0.5 rounded shrink-0">
            AI
          </span>
          <span className="text-xs text-slate-300 font-medium truncate">
            {ACTION_LABELS[log.action_type] ?? log.action_type}
          </span>
          <span className="text-xs text-slate-600 hidden sm:block shrink-0">·</span>
          <span className="text-xs text-slate-500 truncate hidden sm:block">
            {log.model_used} · {latencyS}s · {log.tokens_input + log.tokens_output} tokens · by {log.triggered_by.full_name}
          </span>
        </div>

        {/* Right: status badge + chevron */}
        <div className="flex items-center gap-2 shrink-0">
          {pending && canReview && (
            <span className="text-[10px] text-amber-400 font-semibold uppercase tracking-wide">Review needed</span>
          )}
          <AIReviewBadge status={log.human_review_status} />
          {expanded
            ? <ChevronUp size={14} className="text-slate-500" />
            : <ChevronDown size={14} className="text-slate-500" />
          }
        </div>
      </button>

      {/* ── Expanded body ── */}
      {expanded && (
        <div className="px-4 pb-4 space-y-4 border-t border-slate-700/40 pt-3">
          {/* Confidence */}
          {log.confidence_score != null && (
            <div>
              <p className="text-xs text-slate-500 mb-1">Confidence</p>
              <ConfidenceBar score={log.confidence_score} />
              {log.confidence_score < 0.5 && (
                <p className="text-xs text-red-400 mt-1">⚠ Low confidence — please verify carefully before approving.</p>
              )}
            </div>
          )}

          {/* Response */}
          {editing ? (
            <textarea
              value={editText}
              onChange={(e) => setEditText(e.target.value)}
              rows={8}
              className="w-full bg-[#080e1a] border border-slate-600 text-slate-200 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-violet-500 resize-none"
              data-testid="edit-textarea"
            />
          ) : (
            <div className="text-sm leading-relaxed" data-testid="response-text">
              {renderMarkdown(displayText)}
            </div>
          )}

          {/* Citations */}
          {log.citations.length > 0 && (
            <div>
              <button
                onClick={() => setShowCitations(!showCitations)}
                className="text-xs text-violet-400 hover:text-violet-300 transition-colors"
              >
                {showCitations ? "▼" : "▶"} {log.citations.length} source{log.citations.length > 1 ? "s" : ""}
              </button>
              {showCitations && (
                <div className="mt-2 space-y-2" data-testid="citations-panel">
                  {log.citations.map((c, i) => (
                    <CitationCard key={`${c.doc_id}-${c.chunk_index}`} citation={c} index={i} />
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Metadata row */}
          <div className="flex flex-wrap gap-3 text-xs text-slate-500 pt-1 border-t border-slate-700/50">
            <span>{log.model_used}</span>
            <span>{latencyS}s</span>
            <span>{log.tokens_input + log.tokens_output} tokens</span>
            <span>by {log.triggered_by.full_name}</span>
            <span>{new Date(log.created_at).toLocaleString()}</span>
            {log.reviewed_by && (
              <span className="text-emerald-400">reviewed by {log.reviewed_by.full_name}</span>
            )}
          </div>

          {/* Review actions */}
          {canReview && pending && !editing && !showReject && (
            <div className="flex gap-2 pt-1">
              <button
                onClick={() => onApprove?.(log.id)}
                className="bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-medium px-3 py-1.5 rounded-lg transition-colors"
                data-testid="approve-btn"
              >
                ✓ Approve
              </button>
              <button
                onClick={handleEdit}
                className="bg-slate-600 hover:bg-slate-500 text-slate-200 text-xs font-medium px-3 py-1.5 rounded-lg transition-colors"
                data-testid="edit-btn"
              >
                ✏ Edit
              </button>
              <button
                onClick={handleReject}
                className="bg-red-900/40 hover:bg-red-800/50 text-red-300 text-xs font-medium px-3 py-1.5 rounded-lg transition-colors"
                data-testid="reject-btn"
              >
                ✗ Reject
              </button>
            </div>
          )}

          {editing && (
            <div className="flex gap-2">
              <button
                onClick={() => { onEdit?.(log.id, editText); setEditing(false); }}
                className="bg-violet-600 hover:bg-violet-500 text-white text-xs font-medium px-3 py-1.5 rounded-lg transition-colors"
              >
                Save Edit
              </button>
              <button
                onClick={() => setEditing(false)}
                className="text-slate-400 hover:text-slate-200 text-xs px-3 py-1.5"
              >
                Cancel
              </button>
            </div>
          )}

          {showReject && (
            <div className="space-y-2">
              <input
                type="text"
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                placeholder="Reason for rejection (required)"
                className="w-full bg-[#080e1a] border border-slate-600 text-white text-xs rounded-lg px-3 py-2 focus:outline-none focus:border-red-500"
                data-testid="reject-reason-input"
              />
              <div className="flex gap-2">
                <button
                  onClick={() => { if (rejectReason.trim()) { onReject?.(log.id); setShowReject(false); } }}
                  disabled={!rejectReason.trim()}
                  className="bg-red-700 hover:bg-red-600 text-white text-xs font-medium px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50"
                >
                  Confirm Reject
                </button>
                <button
                  onClick={() => { setShowReject(false); setRejectReason(""); }}
                  className="text-slate-400 hover:text-slate-200 text-xs px-3 py-1.5"
                >
                  Cancel
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
