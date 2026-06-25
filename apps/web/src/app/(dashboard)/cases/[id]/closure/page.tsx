"use client";

import { use, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { CaseSubNav } from "@/components/cases/CaseSubNav";
import {
  CheckCircle,
  XCircle,
  AlertCircle,
  Clock,
  ChevronDown,
  ChevronUp,
  Lock,
  Unlock,
  Send,
  ThumbsUp,
  ThumbsDown,
  RotateCcw,
  Info,
} from "lucide-react";

// ── Types ─────────────────────────────────────────────────────────────────────

interface ChecklistItem {
  name: string;
  passed: boolean;
  blocking: boolean;
  detail: string;
}

interface ChecklistResult {
  can_proceed: boolean;
  checks: ChecklistItem[];
}

interface UserRef {
  id: string;
  full_name: string;
}

interface ClosureResponse {
  id: string;
  case_id: string;
  disposition: string;
  summary: string;
  reason_code: string;
  filed_str: boolean;
  str_file_id?: string;
  status: "not_started" | "draft" | "submitted" | "approved" | "rejected" | "voided";
  sar_considered: boolean;
  sar_rejection_reason?: string;
  risk_rating_at_close?: string;
  created_by: UserRef;
  reviewed_by?: UserRef;
  approved_by?: UserRef;
  submitted_at?: string;
  approved_at?: string;
  voided_at?: string;
  voided_reason?: string;
  created_at: string;
  updated_at: string;
}

// ── Constants ─────────────────────────────────────────────────────────────────

const DISPOSITION_OPTIONS = [
  { value: "sar_filed",                label: "SAR Filed",                     desc: "Suspicious Activity Report submitted to FIU-IND" },
  { value: "no_sar_warranted",         label: "No SAR Warranted",              desc: "Reviewed — activity is not suspicious" },
  { value: "false_positive",           label: "False Positive",                desc: "Alert was a false positive, no suspicious activity" },
  { value: "closed_monitoring",        label: "Closed — Continued Monitoring", desc: "Case closed but subject remains under monitoring" },
  { value: "escalated_law_enforcement",label: "Escalated to Law Enforcement",  desc: "Referred to police / ED / CBI" },
  { value: "referred_internal",        label: "Referred Internally",           desc: "Escalated to a different internal team" },
];

const REASON_CODE_OPTIONS = [
  { value: "insufficient_evidence", label: "Insufficient Evidence" },
  { value: "customer_clarification", label: "Customer Provided Clarification" },
  { value: "known_business_purpose", label: "Known Business Purpose" },
  { value: "periodic_review_cleared", label: "Cleared on Periodic Review" },
  { value: "sar_filed_fiu", label: "SAR Filed with FIU-IND" },
  { value: "law_enforcement_referral", label: "Law Enforcement Referral" },
  { value: "other", label: "Other" },
];

const STATUS_STYLES: Record<string, string> = {
  draft:     "bg-slate-700 text-slate-300",
  submitted: "bg-amber-900/50 text-amber-300 border border-amber-500/30",
  approved:  "bg-emerald-900/50 text-emerald-300 border border-emerald-500/30",
  rejected:  "bg-red-900/50 text-red-300 border border-red-500/30",
  voided:    "bg-slate-800 text-slate-500 line-through",
};

const HIGH_RISK_DISPOSITIONS = new Set(["no_sar_warranted", "false_positive", "closed_monitoring"]);

// ── Helpers ───────────────────────────────────────────────────────────────────

function fmt(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("en-IN", { dateStyle: "medium", timeStyle: "short" });
}

// ── Sub-components ────────────────────────────────────────────────────────────

function ChecklistPanel({ caseId }: { caseId: string }) {
  const { data, isLoading } = useQuery<ChecklistResult>({
    queryKey: ["closure-checklist", caseId],
    queryFn: () => api.get(`/api/v1/cases/${caseId}/closure/checklist`),
  });

  if (isLoading) return (
    <div className="animate-pulse space-y-2">
      {[1, 2, 3].map((i) => (
        <div key={i} className="h-12 bg-slate-700/50 rounded-lg" />
      ))}
    </div>
  );

  if (!data) return null;

  return (
    <div className="space-y-2">
      {data.checks.map((check) => (
        <div
          key={check.name}
          className={`flex items-start gap-3 p-3 rounded-lg border ${
            check.passed
              ? "bg-emerald-900/20 border-emerald-500/20"
              : check.blocking
              ? "bg-red-900/20 border-red-500/30"
              : "bg-slate-700/30 border-slate-600/30"
          }`}
        >
          {check.passed ? (
            <CheckCircle size={16} className="text-emerald-400 mt-0.5 shrink-0" />
          ) : check.blocking ? (
            <XCircle size={16} className="text-red-400 mt-0.5 shrink-0" />
          ) : (
            <Info size={16} className="text-slate-400 mt-0.5 shrink-0" />
          )}
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-xs font-medium text-slate-200">
                {check.name.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase())}
              </span>
              {check.blocking && !check.passed && (
                <span className="text-[10px] px-1.5 py-0.5 bg-red-900/60 text-red-300 rounded font-medium">BLOCKING</span>
              )}
            </div>
            <p className="text-xs text-slate-400 mt-0.5">{check.detail}</p>
          </div>
        </div>
      ))}
    </div>
  );
}

function ClosureStatusBanner({ closure }: { closure: ClosureResponse }) {
  const banners: Record<string, React.ReactNode> = {
    submitted: (
      <div className="flex items-center gap-2 p-3 bg-amber-900/20 border border-amber-500/30 rounded-lg text-amber-300 text-sm">
        <Clock size={15} />
        Awaiting manager review — submitted {fmt(closure.submitted_at)}
      </div>
    ),
    approved: (
      <div className="flex items-center gap-2 p-3 bg-emerald-900/20 border border-emerald-500/30 rounded-lg text-emerald-300 text-sm">
        <CheckCircle size={15} />
        Closure approved by {closure.approved_by?.full_name ?? "—"} on {fmt(closure.approved_at)}. Case is closed.
      </div>
    ),
    rejected: (
      <div className="flex items-center gap-2 p-3 bg-red-900/20 border border-red-500/30 rounded-lg text-red-300 text-sm">
        <XCircle size={15} />
        Rejected by {closure.reviewed_by?.full_name ?? "—"} — returned to draft for revision.
      </div>
    ),
    voided: (
      <div className="flex items-center gap-2 p-3 bg-slate-700/50 border border-slate-600/30 rounded-lg text-slate-400 text-sm">
        <Unlock size={15} />
        Closure voided on {fmt(closure.voided_at)} — case was reopened. Reason: {closure.voided_reason ?? "—"}
      </div>
    ),
  };
  return banners[closure.status] ?? null;
}

// ── Main page ─────────────────────────────────────────────────────────────────

import { useCaseSummary } from "@/lib/hooks/useCaseSummary";

export default function ClosurePage({ params }: { params: Promise<{ id: string }> }) {
  const { id: caseId } = use(params);
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const roleLevel = { super_admin: 5, admin: 4, manager: 3, analyst: 2, viewer: 1 }[user?.role ?? "viewer"] ?? 0;
  const canReview = roleLevel >= 3;

  const caseData = useCaseSummary(caseId);
  const canReopen = roleLevel >= 4;

  // Form state
  const [disposition, setDisposition] = useState("");
  const [summary, setSummary] = useState("");
  const [reasonCode, setReasonCode] = useState("");
  const [filedSTR, setFiledSTR] = useState(false);
  const [sarConsidered, setSarConsidered] = useState(false);
  const [sarRejectionReason, setSarRejectionReason] = useState("");
  const [formError, setFormError] = useState("");

  // Rejection / reopen state
  const [rejectReason, setRejectReason] = useState("");
  const [reopenReason, setReopenReason] = useState("");
  const [showRejectForm, setShowRejectForm] = useState(false);
  const [showReopenForm, setShowReopenForm] = useState(false);

  // Checklist expansion
  const [checklistExpanded, setChecklistExpanded] = useState(true);

  const { data: checklist } = useQuery<ChecklistResult>({
    queryKey: ["closure-checklist", caseId],
    queryFn: () => api.get(`/api/v1/cases/${caseId}/closure/checklist`),
  });

  const { data: rawClosure, isLoading } = useQuery({
    queryKey: ["closure", caseId],
    queryFn: () => api.get<ClosureResponse>(`/api/v1/cases/${caseId}/closure`),
  });
  const closure: ClosureResponse | null =
    rawClosure?.status === "not_started" ? null : (rawClosure ?? null);

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ["closure", caseId] });
    qc.invalidateQueries({ queryKey: ["closure-checklist", caseId] });
    qc.invalidateQueries({ queryKey: ["case", caseId] });
  };

  const createMutation = useMutation({
    mutationFn: () =>
      api.post(`/api/v1/cases/${caseId}/closure`, {
        disposition,
        summary,
        reason_code: reasonCode,
        filed_str: filedSTR,
        sar_considered: sarConsidered,
        sar_rejection_reason: sarRejectionReason,
      }),
    onSuccess: () => { setFormError(""); invalidate(); },
    onError: (e: APIError) => setFormError(e.message ?? "Failed to create closure"),
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      api.patch(`/api/v1/cases/${caseId}/closure/${closure?.id}`, {
        disposition: disposition || undefined,
        summary: summary || undefined,
        reason_code: reasonCode || undefined,
        filed_str: filedSTR,
        sar_considered: sarConsidered,
        sar_rejection_reason: sarRejectionReason || undefined,
      }),
    onSuccess: () => { setFormError(""); invalidate(); },
    onError: (e: APIError) => setFormError(e.message ?? "Failed to update closure"),
  });

  const submitMutation = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${caseId}/closure/${closure?.id}/submit`, {}),
    onSuccess: invalidate,
    onError: (e: APIError) => setFormError(e.message ?? "Failed to submit closure"),
  });

  const approveMutation = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${caseId}/closure/${closure?.id}/approve`, {}),
    onSuccess: invalidate,
  });

  const rejectMutation = useMutation({
    mutationFn: () =>
      api.post(`/api/v1/cases/${caseId}/closure/${closure?.id}/reject`, { reason: rejectReason }),
    onSuccess: () => { setShowRejectForm(false); setRejectReason(""); invalidate(); },
  });

  const reopenMutation = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${caseId}/reopen`, { reason: reopenReason }),
    onSuccess: () => { setShowReopenForm(false); setReopenReason(""); invalidate(); },
  });

  // Derive whether SAR fields are required
  const isHighRisk = checklist?.checks.find((c) => c.name === "risk_level_snapshot")?.detail?.includes("high") ||
    checklist?.checks.find((c) => c.name === "risk_level_snapshot")?.detail?.includes("critical");
  const showSARFields = isHighRisk && HIGH_RISK_DISPOSITIONS.has(disposition);

  const effectiveDisposition = disposition || closure?.disposition || "";
  const effectiveSummary = summary || closure?.summary || "";

  // Populate form from existing closure when first loaded
  const initFromClosure = (cl: ClosureResponse) => {
    if (!disposition && cl.disposition) setDisposition(cl.disposition);
    if (!summary && cl.summary) setSummary(cl.summary);
    if (!reasonCode && cl.reason_code) setReasonCode(cl.reason_code);
    if (cl.filed_str) setFiledSTR(cl.filed_str);
    if (cl.sar_considered) setSarConsidered(cl.sar_considered);
    if (!sarRejectionReason && cl.sar_rejection_reason) setSarRejectionReason(cl.sar_rejection_reason);
  };

  if (!isLoading && closure && !disposition) {
    initFromClosure(closure);
  }

  const isDraft = !closure || closure.status === "draft" || closure.status === "rejected";
  const canEdit = isDraft;
  const canSubmit = isDraft && closure && effectiveSummary.length >= 100 && effectiveDisposition;

  return (
    <div className="p-6">
      <CaseSubNav
        caseId={caseId}
        activeTab="closure"
        caseNumber={caseData?.case_number}
        action={closure ? (
          <span className={`text-xs px-2.5 py-1 rounded-full font-medium ${STATUS_STYLES[closure.status]}`}>
            {closure.status.charAt(0).toUpperCase() + closure.status.slice(1)}
          </span>
        ) : undefined}
      />

      <div className="mb-6">
        <h1 className="text-xl font-bold text-white">Case Closure</h1>
        <p className="text-slate-400 text-sm mt-0.5">Complete the pre-closure checklist, fill the closure form, and submit for manager approval.</p>
      </div>

        {/* Status banner */}
        {closure && <div className="mb-6"><ClosureStatusBanner closure={closure} /></div>}

        <div className="space-y-6">
          {/* ── Step 1: Pre-closure Checklist ── */}
          <section className="bg-slate-800/60 border border-slate-700/50 rounded-xl overflow-hidden">
            <button
              onClick={() => setChecklistExpanded((v) => !v)}
              className="w-full flex items-center justify-between px-5 py-4 text-left hover:bg-slate-700/30 transition-colors"
            >
              <div className="flex items-center gap-2">
                <span className="text-sm font-semibold text-slate-200">Step 1 — Pre-Closure Checklist</span>
                {checklist && (
                  checklist.can_proceed
                    ? <CheckCircle size={14} className="text-emerald-400" />
                    : <XCircle size={14} className="text-red-400" />
                )}
              </div>
              {checklistExpanded ? <ChevronUp size={14} className="text-slate-500" /> : <ChevronDown size={14} className="text-slate-500" />}
            </button>
            {checklistExpanded && (
              <div className="px-5 pb-5">
                <ChecklistPanel caseId={caseId} />
              </div>
            )}
          </section>

          {/* ── Step 2: Closure Form ── */}
          <section className="bg-slate-800/60 border border-slate-700/50 rounded-xl p-5 space-y-5">
            <h2 className="text-sm font-semibold text-slate-200">Step 2 — Closure Details</h2>

            {/* Disposition */}
            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1.5">
                Disposition <span className="text-red-400">*</span>
              </label>
              <select
                value={disposition}
                onChange={(e) => setDisposition(e.target.value)}
                disabled={!canEdit}
                className="w-full bg-slate-700/60 border border-slate-600 rounded-lg px-3 py-2 text-sm text-slate-100 disabled:opacity-60 focus:outline-none focus:ring-1 focus:ring-cyan-500"
              >
                <option value="">Select disposition…</option>
                {DISPOSITION_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>{o.label}</option>
                ))}
              </select>
              {disposition && (
                <p className="text-xs text-slate-500 mt-1">
                  {DISPOSITION_OPTIONS.find((o) => o.value === disposition)?.desc}
                </p>
              )}
            </div>

            {/* SAR consideration (high-risk cases, non-SAR disposition) */}
            {showSARFields && (
              <div className="p-3 bg-amber-900/20 border border-amber-500/30 rounded-lg space-y-3">
                <div className="flex items-start gap-2">
                  <AlertCircle size={14} className="text-amber-400 mt-0.5 shrink-0" />
                  <p className="text-xs text-amber-300">
                    This is a <strong>high-risk case</strong>. You must confirm SAR was considered before closing without filing.
                  </p>
                </div>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={sarConsidered}
                    onChange={(e) => setSarConsidered(e.target.checked)}
                    disabled={!canEdit}
                    className="w-4 h-4 rounded accent-amber-500"
                  />
                  <span className="text-xs text-slate-300">I confirm SAR was considered and is not warranted</span>
                </label>
                {sarConsidered && (
                  <div>
                    <label className="block text-xs font-medium text-slate-300 mb-1">
                      Reason SAR was not filed <span className="text-red-400">*</span>
                    </label>
                    <textarea
                      value={sarRejectionReason}
                      onChange={(e) => setSarRejectionReason(e.target.value)}
                      disabled={!canEdit}
                      rows={2}
                      placeholder="Explain why SAR is not warranted for this high-risk case…"
                      className="w-full bg-slate-700/60 border border-slate-600 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 disabled:opacity-60 focus:outline-none focus:ring-1 focus:ring-cyan-500 resize-none"
                    />
                  </div>
                )}
              </div>
            )}

            {/* Summary */}
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="text-xs font-medium text-slate-300">
                  Investigation Summary <span className="text-red-400">*</span>
                </label>
                <span className={`text-xs ${summary.length < 100 ? "text-amber-400" : "text-emerald-400"}`}>
                  {summary.length} / 100 min
                </span>
              </div>
              <textarea
                value={summary}
                onChange={(e) => setSummary(e.target.value)}
                disabled={!canEdit}
                rows={8}
                placeholder="Summarise the investigation: what triggered the alert, what was found, evidence reviewed, why this disposition was chosen, and any risk mitigants applied. Minimum 100 characters required."
                className="w-full bg-slate-700/60 border border-slate-600 rounded-lg px-3 py-2.5 text-sm text-slate-100 placeholder:text-slate-500 disabled:opacity-60 focus:outline-none focus:ring-1 focus:ring-cyan-500 resize-none"
              />
              {summary.length > 0 && summary.length < 100 && (
                <p className="text-xs text-amber-400 mt-1">
                  Summary must be at least 100 characters ({100 - summary.length} more needed)
                </p>
              )}
            </div>

            {/* Reason code */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">Reason Code</label>
                <select
                  value={reasonCode}
                  onChange={(e) => setReasonCode(e.target.value)}
                  disabled={!canEdit}
                  className="w-full bg-slate-700/60 border border-slate-600 rounded-lg px-3 py-2 text-sm text-slate-100 disabled:opacity-60 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                >
                  <option value="">Select reason code…</option>
                  {REASON_CODE_OPTIONS.map((o) => (
                    <option key={o.value} value={o.value}>{o.label}</option>
                  ))}
                </select>
              </div>
              <div className="flex items-end">
                <label className="flex items-center gap-2 cursor-pointer pb-2">
                  <input
                    type="checkbox"
                    checked={filedSTR}
                    onChange={(e) => setFiledSTR(e.target.checked)}
                    disabled={!canEdit}
                    className="w-4 h-4 rounded accent-cyan-500"
                  />
                  <span className="text-xs text-slate-300">STR/SAR filed with FIU-IND</span>
                </label>
              </div>
            </div>

            {/* Error */}
            {formError && (
              <div className="flex items-center gap-2 p-3 bg-red-900/20 border border-red-500/30 rounded-lg text-red-300 text-sm">
                <AlertCircle size={14} />
                {formError}
              </div>
            )}

            {/* Actions */}
            {canEdit && (
              <div className="flex items-center gap-3 pt-1">
                {!closure ? (
                  <button
                    onClick={() => createMutation.mutate()}
                    disabled={createMutation.isPending || !disposition || summary.length < 100}
                    className="flex items-center gap-1.5 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm rounded-lg font-medium transition-colors"
                  >
                    {createMutation.isPending ? "Saving…" : "Save Draft"}
                  </button>
                ) : (
                  <button
                    onClick={() => updateMutation.mutate()}
                    disabled={updateMutation.isPending || !disposition || summary.length < 100}
                    className="flex items-center gap-1.5 px-4 py-2 bg-slate-600 hover:bg-slate-500 disabled:opacity-50 text-white text-sm rounded-lg font-medium transition-colors"
                  >
                    {updateMutation.isPending ? "Saving…" : "Save Changes"}
                  </button>
                )}
              </div>
            )}
          </section>

          {/* ── Step 3: Submit for Review ── */}
          {closure && closure.status === "draft" && (
            <section className="bg-slate-800/60 border border-slate-700/50 rounded-xl p-5">
              <h2 className="text-sm font-semibold text-slate-200 mb-3">Step 3 — Submit for Manager Review</h2>
              <p className="text-xs text-slate-400 mb-4">
                Once submitted, this closure request will be sent to a manager for approval. You will not be able to edit it until it is reviewed.
              </p>
              <button
                onClick={() => submitMutation.mutate()}
                disabled={submitMutation.isPending || !canSubmit || !checklist?.can_proceed}
                className="flex items-center gap-1.5 px-4 py-2 bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white text-sm rounded-lg font-medium transition-colors"
              >
                <Send size={14} />
                {submitMutation.isPending ? "Submitting…" : "Submit for Review"}
              </button>
              {!checklist?.can_proceed && (
                <p className="text-xs text-red-400 mt-2">
                  Resolve all blocking checklist items before submitting.
                </p>
              )}
            </section>
          )}

          {/* ── Manager Review Panel ── */}
          {canReview && closure && closure.status === "submitted" && (
            <section className="bg-slate-800/60 border border-amber-500/30 rounded-xl p-5 space-y-4">
              <h2 className="text-sm font-semibold text-amber-300 flex items-center gap-2">
                <AlertCircle size={14} />
                Manager Review
              </h2>

              {/* Closure read-only summary */}
              <div className="space-y-3 text-sm">
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <span className="text-xs text-slate-500 uppercase tracking-wider">Disposition</span>
                    <p className="text-slate-200 mt-0.5">
                      {DISPOSITION_OPTIONS.find((o) => o.value === closure.disposition)?.label ?? closure.disposition}
                    </p>
                  </div>
                  <div>
                    <span className="text-xs text-slate-500 uppercase tracking-wider">Risk at Close</span>
                    <p className="text-slate-200 mt-0.5 capitalize">{closure.risk_rating_at_close ?? "—"}</p>
                  </div>
                  <div>
                    <span className="text-xs text-slate-500 uppercase tracking-wider">STR Filed</span>
                    <p className={`mt-0.5 ${closure.filed_str ? "text-emerald-400" : "text-slate-400"}`}>
                      {closure.filed_str ? "Yes" : "No"}
                    </p>
                  </div>
                  <div>
                    <span className="text-xs text-slate-500 uppercase tracking-wider">Submitted By</span>
                    <p className="text-slate-200 mt-0.5">{closure.created_by.full_name}</p>
                  </div>
                </div>
                <div>
                  <span className="text-xs text-slate-500 uppercase tracking-wider">Summary</span>
                  <p className="text-slate-300 mt-0.5 text-sm leading-relaxed whitespace-pre-wrap bg-slate-700/40 rounded-lg p-3">
                    {closure.summary}
                  </p>
                </div>
                {closure.sar_considered && closure.sar_rejection_reason && (
                  <div>
                    <span className="text-xs text-slate-500 uppercase tracking-wider">SAR Rejection Reason</span>
                    <p className="text-amber-300 mt-0.5 text-sm">{closure.sar_rejection_reason}</p>
                  </div>
                )}
              </div>

              {/* Approve */}
              <div className="flex items-center gap-3 pt-1">
                <button
                  onClick={() => approveMutation.mutate()}
                  disabled={approveMutation.isPending}
                  className="flex items-center gap-1.5 px-4 py-2 bg-emerald-700 hover:bg-emerald-600 disabled:opacity-50 text-white text-sm rounded-lg font-medium transition-colors"
                >
                  <ThumbsUp size={14} />
                  {approveMutation.isPending ? "Approving…" : "Approve & Close Case"}
                </button>
                <button
                  onClick={() => setShowRejectForm((v) => !v)}
                  className="flex items-center gap-1.5 px-4 py-2 bg-red-900/50 hover:bg-red-800/60 border border-red-500/40 text-red-300 text-sm rounded-lg font-medium transition-colors"
                >
                  <ThumbsDown size={14} />
                  Reject
                </button>
              </div>

              {showRejectForm && (
                <div className="space-y-2">
                  <label className="text-xs text-slate-300 font-medium">Rejection reason <span className="text-slate-500">(shown to analyst)</span></label>
                  <textarea
                    value={rejectReason}
                    onChange={(e) => setRejectReason(e.target.value)}
                    rows={3}
                    placeholder="Explain what needs to be corrected before re-submission…"
                    className="w-full bg-slate-700/60 border border-slate-600 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-red-500 resize-none"
                  />
                  <button
                    onClick={() => rejectMutation.mutate()}
                    disabled={rejectMutation.isPending || !rejectReason.trim()}
                    className="flex items-center gap-1.5 px-3 py-1.5 bg-red-700 hover:bg-red-600 disabled:opacity-50 text-white text-xs rounded-lg font-medium transition-colors"
                  >
                    {rejectMutation.isPending ? "Rejecting…" : "Confirm Rejection"}
                  </button>
                </div>
              )}
            </section>
          )}

          {/* ── Approved closure detail ── */}
          {closure && closure.status === "approved" && (
            <section className="bg-slate-800/60 border border-emerald-500/20 rounded-xl p-5 space-y-4">
              <h2 className="text-sm font-semibold text-emerald-300 flex items-center gap-2">
                <Lock size={14} />
                Closed — Authoritative Record
              </h2>
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <span className="text-xs text-slate-500 uppercase tracking-wider">Disposition</span>
                  <p className="text-slate-200 mt-0.5">
                    {DISPOSITION_OPTIONS.find((o) => o.value === closure.disposition)?.label ?? closure.disposition}
                  </p>
                </div>
                <div>
                  <span className="text-xs text-slate-500 uppercase tracking-wider">Risk at Close</span>
                  <p className="text-slate-200 mt-0.5 capitalize">{closure.risk_rating_at_close ?? "—"}</p>
                </div>
                <div>
                  <span className="text-xs text-slate-500 uppercase tracking-wider">Prepared By</span>
                  <p className="text-slate-200 mt-0.5">{closure.created_by.full_name}</p>
                </div>
                <div>
                  <span className="text-xs text-slate-500 uppercase tracking-wider">Approved By</span>
                  <p className="text-slate-200 mt-0.5">{closure.approved_by?.full_name ?? "—"}</p>
                </div>
                <div>
                  <span className="text-xs text-slate-500 uppercase tracking-wider">Submitted</span>
                  <p className="text-slate-200 mt-0.5">{fmt(closure.submitted_at)}</p>
                </div>
                <div>
                  <span className="text-xs text-slate-500 uppercase tracking-wider">Approved</span>
                  <p className="text-slate-200 mt-0.5">{fmt(closure.approved_at)}</p>
                </div>
              </div>
              <div>
                <span className="text-xs text-slate-500 uppercase tracking-wider">Summary</span>
                <p className="text-slate-300 mt-0.5 text-sm leading-relaxed whitespace-pre-wrap bg-slate-700/40 rounded-lg p-3">
                  {closure.summary}
                </p>
              </div>
              {closure.sar_considered && closure.sar_rejection_reason && (
                <div>
                  <span className="text-xs text-slate-500 uppercase tracking-wider">SAR Rejection Reason</span>
                  <p className="text-amber-300 mt-1 text-sm">{closure.sar_rejection_reason}</p>
                </div>
              )}

              {/* Reopen */}
              {canReopen && (
                <div className="pt-1">
                  <button
                    onClick={() => setShowReopenForm((v) => !v)}
                    className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200 transition-colors"
                  >
                    <RotateCcw size={12} />
                    Reopen case (voids this closure)
                  </button>
                  {showReopenForm && (
                    <div className="mt-3 space-y-2">
                      <label className="text-xs text-slate-300 font-medium">
                        Reason for reopening <span className="text-red-400">*</span>
                      </label>
                      <textarea
                        value={reopenReason}
                        onChange={(e) => setReopenReason(e.target.value)}
                        rows={2}
                        placeholder="Why is this case being reopened? This will be preserved in the audit trail."
                        className="w-full bg-slate-700/60 border border-slate-600 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-amber-500 resize-none"
                      />
                      <button
                        onClick={() => reopenMutation.mutate()}
                        disabled={reopenMutation.isPending || !reopenReason.trim()}
                        className="flex items-center gap-1.5 px-3 py-1.5 bg-amber-700 hover:bg-amber-600 disabled:opacity-50 text-white text-xs rounded-lg font-medium transition-colors"
                      >
                        <Unlock size={12} />
                        {reopenMutation.isPending ? "Reopening…" : "Confirm Reopen"}
                      </button>
                    </div>
                  )}
                </div>
              )}
            </section>
          )}
        </div>
    </div>
  );
}
