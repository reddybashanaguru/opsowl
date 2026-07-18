"use client";

import { use, useState, useRef, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { CaseSubNav } from "@/components/cases/CaseSubNav";
import {
  AlertCircle,
  CheckCircle,
  ChevronDown,
  ChevronRight,
  ChevronUp,
  Clock,
  Copy,
  Download,
  PenLine,
  Plus,
  RefreshCw,
  ThumbsDown,
  ThumbsUp,
  Trash2,
  XCircle,
  Zap,
} from "lucide-react";

// ── Types ─────────────────────────────────────────────────────────────────────

interface UserRef {
  id: string;
  full_name: string;
}

interface NarrativeResponse {
  id: string;
  case_id: string;
  narrative_type: string;
  status: "draft" | "approved" | "rejected";
  body: string;
  body_edited?: string;
  active_body: string;
  generated_by?: UserRef;
  reviewed_by?: UserRef;
  reviewed_at?: string;
  rejection_reason?: string;
  model_used: string;
  prompt_version: string;
  findings_included: number;
  txns_included: number;
  ai_log_id?: string;
  created_at: string;
  updated_at: string;
}

// ── Constants ─────────────────────────────────────────────────────────────────

const NARRATIVE_TYPES = [
  {
    value: "investigation_summary",
    label: "Investigation Summary",
    desc: "Comprehensive narrative of what happened, evidence found, and why it's suspicious. 5–8 paragraphs.",
  },
  {
    value: "str_suspicion_basis",
    label: "STR Suspicion Basis",
    desc: "Structured text for the Suspicion Basis field of an FIU-IND STR filing. Regulatory language.",
  },
  {
    value: "sar_narrative",
    label: "SAR Narrative",
    desc: "Suspicious Activity Report narrative for FinCEN-style filing. Includes activity, subject, amounts.",
  },
  {
    value: "closure_summary",
    label: "Closure Summary",
    desc: "Concise text for the Case Closure form — what was investigated, found, and why the case is closed.",
  },
];

const STATUS_STYLES: Record<string, string> = {
  draft:    "bg-slate-700 text-slate-300",
  approved: "bg-emerald-900/50 text-emerald-300 border border-emerald-500/30",
  rejected: "bg-red-900/50 text-red-300 border border-red-500/30",
};

const TYPE_LABELS: Record<string, string> = Object.fromEntries(
  NARRATIVE_TYPES.map((t) => [t.value, t.label])
);

// ── Helpers ───────────────────────────────────────────────────────────────────

function fmtDate(iso: string) {
  return new Date(iso).toLocaleString("en-IN", { dateStyle: "medium", timeStyle: "short" });
}

function wordCount(text: string) {
  return text.trim().split(/\s+/).filter(Boolean).length;
}

// ── Generate Modal ────────────────────────────────────────────────────────────

function GenerateModal({
  caseId,
  onClose,
  onGenerated,
}: {
  caseId: string;
  onClose: () => void;
  onGenerated: () => void;
}) {
  const [selectedType, setSelectedType] = useState("investigation_summary");
  const [notes, setNotes] = useState("");
  const [error, setError] = useState("");
  const submittingRef = useRef(false);

  const mutation = useMutation({
    mutationFn: () =>
      api.post(`/api/v1/cases/${caseId}/narratives/generate`, {
        narrative_type: selectedType,
        investigator_notes: notes,
      }),
    onSuccess: () => {
      submittingRef.current = false;
      setError("");
      onGenerated();
      onClose();
    },
    onError: (e: APIError) => {
      submittingRef.current = false;
      setError(e.message ?? "Generation failed. Check that the AI service is running.");
    },
  });

  const handleGenerate = () => {
    if (submittingRef.current || mutation.isPending) return;
    submittingRef.current = true;
    mutation.mutate();
  };

  const typeInfo = NARRATIVE_TYPES.find((t) => t.value === selectedType);

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] w-full max-w-lg">
        <div className="flex items-center justify-between p-5 border-b border-slate-700">
          <h2 className="font-semibold text-slate-200 flex items-center gap-2">
            <Zap size={16} className="text-cyan-400" />
            Generate AI Narrative
          </h2>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300 text-xl leading-none">×</button>
        </div>

        <div className="p-5 space-y-4">
          <div>
            <label className="block text-xs font-medium text-slate-300 mb-2">Narrative Type</label>
            <div className="space-y-2">
              {NARRATIVE_TYPES.map((t) => (
                <label
                  key={t.value}
                  className={`flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${
                    selectedType === t.value
                      ? "border-cyan-500/60 bg-cyan-900/10"
                      : "border-slate-700/50 hover:border-slate-600"
                  }`}
                >
                  <input
                    type="radio"
                    name="narrative_type"
                    value={t.value}
                    checked={selectedType === t.value}
                    onChange={() => setSelectedType(t.value)}
                    className="mt-0.5 accent-cyan-500"
                  />
                  <div>
                    <p className="text-sm font-medium text-slate-200">{t.label}</p>
                    <p className="text-xs text-slate-500 mt-0.5">{t.desc}</p>
                  </div>
                </label>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">
              Investigator Notes <span className="text-slate-500">(optional — context for the AI)</span>
            </label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
              placeholder="e.g. Focus on the round-tripping pattern in October. Subject is PEP-linked."
              className="w-full bg-slate-700/60 border border-slate-600 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 resize-none"
            />
          </div>

          <div className="bg-slate-700/30 rounded-lg p-3 text-xs text-slate-400 space-y-1">
            <p>The AI will use:</p>
            <ul className="list-disc list-inside space-y-0.5 ml-1">
              <li>All <strong>approved findings</strong> for this case</li>
              <li>All <strong>linked transactions</strong> (flagged first)</li>
              <li><strong>Subject profile</strong> — identity, KYC tier, risk level</li>
              <li><strong>AI Insights</strong> — risk factors &amp; anomalies from document analysis</li>
            </ul>
            <p className="text-slate-500 mt-1">
              Generation is logged to AI Logs for regulatory auditability.
              A manager must approve before the narrative is finalised (4-eye rule).
            </p>
          </div>

          {error && (
            <div className="flex items-center gap-2 p-3 bg-red-900/20 border border-red-500/30 rounded-lg text-red-300 text-sm">
              <AlertCircle size={14} />
              {error}
            </div>
          )}

          <div className="flex gap-3">
            <button
              onClick={handleGenerate}
              disabled={mutation.isPending}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-60 text-white text-sm rounded-lg font-medium transition-colors"
            >
              {mutation.isPending ? (
                <>
                  <RefreshCw size={14} className="animate-spin" />
                  Generating… (may take 30–60s)
                </>
              ) : (
                <>
                  <Zap size={14} />
                  Generate Narrative
                </>
              )}
            </button>
            <button onClick={onClose} className="px-4 py-2.5 bg-slate-700 hover:bg-slate-600 text-sm text-slate-300 rounded-lg transition-colors">
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Narrative Card ────────────────────────────────────────────────────────────

function NarrativeCard({
  narrative,
  caseId,
  canReview,
  userId,
  onMutated,
}: {
  narrative: NarrativeResponse;
  caseId: string;
  canReview: boolean;
  userId: string;
  onMutated: () => void;
}) {
  const qc = useQueryClient();
  const [expanded, setExpanded] = useState(narrative.status === "approved");
  const [isEditing, setIsEditing] = useState(false);
  const [editBody, setEditBody] = useState(narrative.active_body);
  const [showRejectForm, setShowRejectForm] = useState(false);
  const [rejectReason, setRejectReason] = useState("");
  const [copied, setCopied] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const PREVIEW_CHARS = 160;

  useEffect(() => {
    if (isEditing && textareaRef.current) {
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height = textareaRef.current.scrollHeight + "px";
    }
  }, [isEditing, editBody]);

  const editMutation = useMutation({
    mutationFn: () => api.patch(`/api/v1/cases/${caseId}/narratives/${narrative.id}`, { body: editBody }),
    onSuccess: () => { setIsEditing(false); onMutated(); },
  });

  const approveMutation = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${caseId}/narratives/${narrative.id}/approve`, {}),
    onSuccess: onMutated,
  });

  const rejectMutation = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${caseId}/narratives/${narrative.id}/reject`, { reason: rejectReason }),
    onSuccess: () => { setShowRejectForm(false); setRejectReason(""); onMutated(); },
  });

  const deleteMutation = useMutation({
    mutationFn: () => api.delete(`/api/v1/cases/${caseId}/narratives/${narrative.id}`),
    onSuccess: onMutated,
  });

  const copyToClipboard = () => {
    navigator.clipboard.writeText(narrative.active_body);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const downloadTxt = () => {
    const blob = new Blob([narrative.active_body], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `narrative-${narrative.narrative_type}-${narrative.id.slice(0, 8)}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const isApproved = narrative.status === "approved";
  const isCreator = narrative.generated_by?.id === userId;
  const wc = wordCount(narrative.active_body);

  const preview = narrative.active_body.slice(0, PREVIEW_CHARS);
  const hasMore = narrative.active_body.length > PREVIEW_CHARS;

  return (
    <div className={`border rounded-xl overflow-hidden transition-colors ${
      isApproved ? "border-emerald-500/30 bg-emerald-900/5" : "border-slate-700/50 bg-slate-800/40"
    }`}>
      {/* Header — always visible, click to expand/collapse */}
      {/* div instead of button because it contains nested action buttons (Copy, Download) */}
      <div
        role="button"
        tabIndex={0}
        className="w-full flex items-start gap-3 px-5 py-4 text-left hover:bg-white/[0.02] transition-colors cursor-pointer"
        onClick={() => { if (!isEditing && !showRejectForm) setExpanded((v) => !v); }}
        onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); if (!isEditing && !showRejectForm) setExpanded((v) => !v); } }}
      >
        <PenLine size={16} className="text-cyan-400 mt-0.5 shrink-0" />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <h3 className="text-sm font-semibold text-slate-200">
              {TYPE_LABELS[narrative.narrative_type] ?? narrative.narrative_type}
            </h3>
            <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium ${STATUS_STYLES[narrative.status]}`}>
              {narrative.status}
            </span>
            {narrative.body_edited && (
              <span className="text-[10px] px-1.5 py-0.5 bg-amber-900/30 text-amber-400 rounded border border-amber-500/30">
                edited
              </span>
            )}
          </div>
          <div className="flex items-center gap-3 mt-1 text-[11px] text-slate-500 flex-wrap">
            <span>Generated {fmtDate(narrative.created_at)}</span>
            {narrative.generated_by && <span>by {narrative.generated_by.full_name}</span>}
            <span>·</span>
            <span className="font-mono">{narrative.model_used}</span>
            <span>·</span>
            <span>{wc} words</span>
            {narrative.findings_included > 0 && <span>· {narrative.findings_included} findings</span>}
            {narrative.txns_included > 0 && <span>· {narrative.txns_included} txns</span>}
          </div>

          {/* Preview — only shown when collapsed */}
          {!expanded && (
            <p className="text-slate-500 text-xs mt-1.5 leading-relaxed line-clamp-2">
              {preview}{hasMore ? "…" : ""}
            </p>
          )}
        </div>

        {/* Action buttons + chevron */}
        <div className="flex items-center gap-1.5 shrink-0" onClick={(e) => e.stopPropagation()}>
          <button onClick={copyToClipboard} title="Copy to clipboard"
            className="p-1.5 text-slate-500 hover:text-slate-300 rounded transition-colors">
            <Copy size={13} />
          </button>
          <button onClick={downloadTxt} title="Download as .txt"
            className="p-1.5 text-slate-500 hover:text-slate-300 rounded transition-colors">
            <Download size={13} />
          </button>
          {copied && <span className="text-xs text-emerald-400">Copied!</span>}
        </div>
        <div className="shrink-0 mt-0.5">
          {expanded ? <ChevronUp size={14} className="text-slate-500" /> : <ChevronDown size={14} className="text-slate-500" />}
        </div>
      </div>

      {/* Expanded body */}
      {expanded && (
        <>
          {/* Rejection reason */}
          {narrative.status === "rejected" && narrative.rejection_reason && (
            <div className="mx-5 mb-3 p-2 bg-red-900/20 border border-red-500/20 rounded-lg text-xs text-red-300">
              <span className="font-medium">Rejection reason: </span>{narrative.rejection_reason}
            </div>
          )}

          {/* Body */}
          <div className="px-5 pb-4 border-t border-slate-700/40 pt-3">
            {isEditing ? (
              <div className="space-y-2">
                <textarea
                  ref={textareaRef}
                  value={editBody}
                  onChange={(e) => setEditBody(e.target.value)}
                  className="w-full bg-slate-700/60 border border-cyan-500/50 rounded-lg px-3 py-2.5 text-sm text-slate-100 leading-relaxed focus:outline-none focus:ring-1 focus:ring-cyan-500 resize-none min-h-48"
                  style={{ overflow: "hidden" }}
                />
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => editMutation.mutate()}
                    disabled={editMutation.isPending || !editBody.trim()}
                    className="px-3 py-1.5 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-xs rounded-lg font-medium transition-colors"
                  >
                    {editMutation.isPending ? "Saving…" : "Save Edit"}
                  </button>
                  <button onClick={() => { setIsEditing(false); setEditBody(narrative.active_body); }}
                    className="px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-xs text-slate-300 rounded-lg transition-colors">
                    Cancel
                  </button>
                  <span className="text-xs text-slate-500 ml-auto">{wordCount(editBody)} words</span>
                </div>
              </div>
            ) : (
              <div className="prose prose-sm prose-invert max-w-none">
                <p className="text-slate-300 text-sm leading-relaxed whitespace-pre-wrap font-serif">
                  {narrative.active_body}
                </p>
              </div>
            )}
          </div>

          {/* Review info for approved */}
          {isApproved && narrative.reviewed_by && (
            <div className="px-5 pb-3">
              <p className="text-xs text-emerald-400">
                <CheckCircle size={11} className="inline mr-1" />
                Approved by {narrative.reviewed_by.full_name}
                {narrative.reviewed_at && <> on {fmtDate(narrative.reviewed_at)}</>}
              </p>
            </div>
          )}

          {/* Actions */}
          {!isApproved && (
            <div className="px-5 pb-4 flex items-center gap-2 flex-wrap border-t border-slate-700/40 pt-3">
              {!isEditing && (
                <button
                  onClick={() => setIsEditing(true)}
                  className="flex items-center gap-1 text-xs px-2.5 py-1.5 bg-slate-700/50 border border-slate-600/50 text-slate-400 hover:text-slate-200 rounded-lg transition-colors"
                >
                  <PenLine size={11} /> Edit
                </button>
              )}

              {narrative.status === "draft" && !isEditing && canReview && !isCreator && (
                <>
                  <button
                    onClick={() => approveMutation.mutate()}
                    disabled={approveMutation.isPending}
                    className="flex items-center gap-1 text-xs px-2.5 py-1.5 bg-emerald-900/30 border border-emerald-500/40 text-emerald-300 hover:bg-emerald-900/50 rounded-lg transition-colors"
                  >
                    <ThumbsUp size={11} />
                    {approveMutation.isPending ? "Approving…" : "Approve"}
                  </button>
                  <button
                    onClick={() => setShowRejectForm((v) => !v)}
                    className="flex items-center gap-1 text-xs px-2.5 py-1.5 bg-red-900/20 border border-red-500/30 text-red-400 hover:bg-red-900/40 rounded-lg transition-colors"
                  >
                    <ThumbsDown size={11} /> Reject
                  </button>
                </>
              )}
              {narrative.status === "draft" && !isEditing && isCreator && (
                <span className="flex items-center gap-1.5 text-xs text-amber-400 px-2.5 py-1.5 bg-amber-900/20 border border-amber-500/30 rounded-lg">
                  <Clock size={11} /> Awaiting a different approver (4-eye rule)
                </span>
              )}

              {narrative.status !== "approved" && (
                <button
                  onClick={() => { if (confirm("Delete this narrative draft?")) deleteMutation.mutate(); }}
                  disabled={deleteMutation.isPending}
                  className="flex items-center gap-1 text-xs px-2.5 py-1.5 text-slate-600 hover:text-red-400 rounded-lg transition-colors ml-auto"
                >
                  <Trash2 size={11} />
                  {deleteMutation.isPending ? "Deleting…" : "Delete"}
                </button>
              )}
            </div>
          )}

          {/* Reject form */}
          {showRejectForm && (
            <div className="px-5 pb-4 space-y-2 border-t border-slate-700/40 pt-3">
              <label className="text-xs text-slate-300 font-medium">Rejection reason</label>
              <textarea
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                rows={2}
                placeholder="What needs to be corrected or regenerated?"
                className="w-full bg-slate-700/60 border border-slate-600 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-red-500 resize-none"
              />
              <button
                onClick={() => rejectMutation.mutate()}
                disabled={rejectMutation.isPending || !rejectReason.trim()}
                className="px-3 py-1.5 bg-red-700 hover:bg-red-600 disabled:opacity-50 text-white text-xs rounded-lg font-medium transition-colors"
              >
                {rejectMutation.isPending ? "Rejecting…" : "Confirm Rejection"}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ── Main page ─────────────────────────────────────────────────────────────────

import { useCaseSummary } from "@/lib/hooks/useCaseSummary";

export default function NarrativePage({ params }: { params: Promise<{ id: string }> }) {
  const { id: caseId } = use(params);
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const roleLevel = { super_admin: 5, admin: 4, manager: 3, analyst: 2, viewer: 1 }[user?.role ?? "viewer"] ?? 0;
  const canReview = roleLevel >= 3;

  const caseData = useCaseSummary(caseId);

  const [showGenerate, setShowGenerate] = useState(false);
  const [filterType, setFilterType] = useState("");

  const { data, isLoading } = useQuery<NarrativeResponse[]>({
    queryKey: ["narratives", caseId],
    queryFn: () => api.get(`/api/v1/cases/${caseId}/narratives`),
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ["narratives", caseId] });

  const narratives = data ?? [];
  const filtered = filterType ? narratives.filter((n) => n.narrative_type === filterType) : narratives;
  const approvedCount = narratives.filter((n) => n.status === "approved").length;

  return (
    <div className="p-6">
      <CaseSubNav
        caseId={caseId}
        activeTab="narrative"
        caseNumber={caseData?.case_number}
        action={
          <button
            onClick={() => setShowGenerate(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-medium rounded-lg transition-colors"
          >
            <Plus size={13} /> Generate Narrative
          </button>
        }
      />

      <div className="mb-6">
        <h1 className="text-xl font-bold text-white">Investigation Narratives</h1>
        <p className="text-slate-400 text-sm mt-0.5">AI-drafted narratives grounded in approved findings and linked transactions. Edit, approve, and export for STR/SAR filings or case closure.</p>
      </div>

        {/* Stats */}
        {narratives.length > 0 && (
          <div className="flex items-center gap-4 mb-5 text-xs text-slate-500">
            <span>{narratives.length} narrative{narratives.length !== 1 ? "s" : ""}</span>
            {approvedCount > 0 && (
              <span className="text-emerald-400 flex items-center gap-1">
                <CheckCircle size={11} /> {approvedCount} approved
              </span>
            )}
            <div className="flex items-center gap-1.5 ml-auto">
              <span>Filter:</span>
              <select
                value={filterType}
                onChange={(e) => setFilterType(e.target.value)}
                className="bg-slate-800 border border-slate-700 rounded px-2 py-1 text-xs text-slate-300 focus:outline-none"
              >
                <option value="">All types</option>
                {NARRATIVE_TYPES.map((t) => (
                  <option key={t.value} value={t.value}>{t.label}</option>
                ))}
              </select>
            </div>
          </div>
        )}

        {/* Content */}
        {isLoading && (
          <div className="space-y-3">
            {[1, 2].map((i) => (
              <div key={i} className="h-32 bg-slate-800/40 border border-slate-700/50 rounded-xl animate-pulse" />
            ))}
          </div>
        )}

        {!isLoading && filtered.length === 0 && (
          <div className="text-center py-20">
            <div className="w-16 h-16 rounded-full bg-cyan-900/20 border border-cyan-500/20 flex items-center justify-center mx-auto mb-4">
              <PenLine size={24} className="text-cyan-400/60" />
            </div>
            <h3 className="text-slate-300 font-medium mb-2">No narratives yet</h3>
            <p className="text-slate-500 text-sm max-w-sm mx-auto mb-6">
              Generate an AI narrative grounded in this case's approved findings and transactions.
              Use it for STR filings, SAR reports, or the case closure form.
            </p>
            <button
              onClick={() => setShowGenerate(true)}
              className="flex items-center gap-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm rounded-lg font-medium mx-auto transition-colors"
            >
              <Zap size={14} /> Generate your first narrative
            </button>

            {/* How it works */}
            <div className="mt-10 text-left max-w-md mx-auto space-y-3">
              <p className="text-xs text-slate-500 font-medium uppercase tracking-wider">How it works</p>
              {[
                { icon: ChevronRight, text: "AI reads all approved findings and linked transactions for this case" },
                { icon: ChevronRight, text: "Generates a structured narrative in formal compliance language" },
                { icon: ChevronRight, text: "You can edit the text before approving" },
                { icon: ChevronRight, text: "Approved narratives are preserved with AI log cross-references for audit" },
              ].map(({ icon: Icon, text }, i) => (
                <div key={i} className="flex items-start gap-2 text-xs text-slate-500">
                  <Icon size={12} className="text-cyan-600 mt-0.5 shrink-0" />
                  {text}
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="space-y-4 pb-24">
          {filtered.map((n) => (
            <NarrativeCard
              key={n.id}
              narrative={n}
              caseId={caseId}
              canReview={canReview}
              userId={user?.id ?? ""}
              onMutated={invalidate}
            />
          ))}
        </div>

      {showGenerate && (
        <GenerateModal
          caseId={caseId}
          onClose={() => setShowGenerate(false)}
          onGenerated={invalidate}
        />
      )}
    </div>
  );
}
