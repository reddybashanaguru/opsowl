"use client";

import { useState } from "react";
import { Citation, CitationCard } from "./CitationCard";
import { ConfidenceBar } from "./ConfidenceBar";
import { AIReviewBadge } from "./AIReviewBadge";

const ACTION_LABELS: Record<string, string> = {
  ask:             "Q&A",
  summarize:       "Summary",
  generate_note:   "Draft Note",
  extract_actions: "Action Items",
  pattern_detect:  "Pattern Detection",
};

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
}

export function AICard({ log, onApprove, onReject, onEdit, canReview }: AICardProps) {
  const [showCitations, setShowCitations] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editText, setEditText] = useState(log.human_edited_response ?? log.response_text);
  const [rejectReason, setRejectReason] = useState("");
  const [showReject, setShowReject] = useState(false);

  const displayText = log.human_edited_response ?? log.response_text;
  const latencyS = (log.latency_ms / 1000).toFixed(1);
  const pending = log.human_review_status === "pending";

  return (
    <div
      className="bg-[#131f30] border border-slate-700 border-l-4 border-l-violet-500 rounded-xl p-5 space-y-4"
      data-testid="ai-card"
    >
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-2">
          <span className="bg-violet-500/20 text-violet-300 border border-violet-500/40 text-xs font-bold px-2 py-0.5 rounded">
            AI Generated
          </span>
          <span className="text-xs text-slate-500">{ACTION_LABELS[log.action_type] ?? log.action_type}</span>
        </div>
        <AIReviewBadge status={log.human_review_status} />
      </div>

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
        <div className="text-slate-300 text-sm whitespace-pre-wrap leading-relaxed" data-testid="response-text">
          {displayText}
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

      {/* Metadata */}
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
            onClick={() => { setEditing(true); setEditText(log.response_text); }}
            className="bg-slate-600 hover:bg-slate-500 text-slate-200 text-xs font-medium px-3 py-1.5 rounded-lg transition-colors"
            data-testid="edit-btn"
          >
            ✏ Edit
          </button>
          <button
            onClick={() => setShowReject(true)}
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
  );
}
