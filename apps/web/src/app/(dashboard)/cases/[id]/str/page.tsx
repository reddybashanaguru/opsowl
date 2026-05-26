"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";

interface STRFiling {
  id: string;
  case_id: string;
  report_type: string;
  status: "draft" | "submitted" | "approved" | "rejected" | "filed";
  suspicion_basis: string;
  narrative: string;
  evidence_summary: string;
  submitted_by?: string;
  submitted_by_name?: string;
  approved_by?: string;
  approved_by_name?: string;
  submitted_at?: string;
  approved_at?: string;
  filed_at?: string;
  fiu_reference?: string;
  rejection_reason?: string;
  filing_entity: Record<string, unknown>;
  subject: Record<string, unknown>;
  transactions: unknown[];
  created_at: string;
  updated_at: string;
}

const STATUS_COLORS: Record<string, string> = {
  draft:     "bg-slate-700 text-slate-300",
  submitted: "bg-amber-900/50 text-amber-300 border border-amber-500/30",
  approved:  "bg-emerald-900/50 text-emerald-300 border border-emerald-500/30",
  rejected:  "bg-red-900/50 text-red-300 border border-red-500/30",
  filed:     "bg-cyan-900/50 text-cyan-300 border border-cyan-500/30",
};

const REPORT_TYPE_LABELS: Record<string, string> = {
  STR: "Suspicious Transaction Report",
  SAR: "Suspicious Activity Report",
  CTR: "Cash Transaction Report",
};

function FilingCard({
  filing,
  caseId,
  userId,
  userRole,
  onAction,
}: {
  filing: STRFiling;
  caseId: string;
  userId: string;
  userRole: string;
  onAction: () => void;
}) {
  const qc = useQueryClient();
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [rejectReason, setRejectReason] = useState("");
  const [actionError, setActionError] = useState("");
  const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

  const isManagerOrAbove = ["manager", "admin", "super_admin"].includes(userRole);
  const isAdminOrAbove = ["admin", "super_admin"].includes(userRole);

  const mutOpts = {
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["str-filings", caseId] }); onAction(); setActionError(""); },
    onError: (err: unknown) => setActionError(err instanceof APIError ? err.message : "Action failed"),
  };

  const submitMut = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${caseId}/str/${filing.id}/submit`, {}),
    ...mutOpts,
  });

  const approveMut = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${caseId}/str/${filing.id}/approve`, {}),
    ...mutOpts,
  });

  const rejectMut = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${caseId}/str/${filing.id}/reject`, { reason: rejectReason }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["str-filings", caseId] }); onAction(); setShowRejectModal(false); setRejectReason(""); setActionError(""); },
    onError: (err: unknown) => setActionError(err instanceof APIError ? err.message : "Reject failed"),
  });

  const filedMut = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${caseId}/str/${filing.id}/file`, {}),
    ...mutOpts,
  });

  const handleExport = async () => {
    try {
      const res = await fetch(`${apiBase}/api/v1/cases/${caseId}/str/${filing.id}/export`, {
        credentials: "include",
      });
      if (!res.ok) throw new Error("Export failed");
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `STR-${filing.id.slice(0, 8)}.xml`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Export failed");
    }
  };

  const isSelf = filing.submitted_by === userId;

  return (
    <div className="bg-slate-800/60 border border-slate-700/50 rounded-xl p-5 flex flex-col gap-4">
      {/* Header row */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <span className="font-mono text-xs text-slate-500">{filing.id.slice(0, 8).toUpperCase()}</span>
            <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${STATUS_COLORS[filing.status]}`}>
              {filing.status.toUpperCase()}
            </span>
            <span className="text-xs text-slate-500 bg-slate-700/50 px-2 py-0.5 rounded">
              {REPORT_TYPE_LABELS[filing.report_type] ?? filing.report_type}
            </span>
          </div>
          <p className="text-white text-sm font-medium">{filing.suspicion_basis}</p>
        </div>
        <div className="text-xs text-slate-500 whitespace-nowrap">
          {new Date(filing.created_at).toLocaleDateString()}
        </div>
      </div>

      {/* Narrative */}
      {filing.narrative && (
        <p className="text-slate-300 text-sm leading-relaxed">{filing.narrative}</p>
      )}

      {/* Maker-checker trail */}
      <div className="flex flex-wrap gap-4 text-xs text-slate-500">
        {filing.submitted_by_name && (
          <span>Submitted by <span className="text-slate-300">{filing.submitted_by_name}</span>
            {filing.submitted_at && <> · {new Date(filing.submitted_at).toLocaleString()}</>}
          </span>
        )}
        {filing.approved_by_name && (
          <span>Approved by <span className="text-slate-300">{filing.approved_by_name}</span>
            {filing.approved_at && <> · {new Date(filing.approved_at).toLocaleString()}</>}
          </span>
        )}
        {filing.fiu_reference && (
          <span>FIU Ref: <span className="text-cyan-300 font-mono">{filing.fiu_reference}</span></span>
        )}
        {filing.rejection_reason && (
          <span className="text-red-400">Rejected: {filing.rejection_reason}</span>
        )}
      </div>

      {/* Error */}
      {actionError && (
        <p className="text-red-400 text-xs bg-red-900/20 border border-red-800/30 px-3 py-2 rounded-lg">{actionError}</p>
      )}

      {/* Actions */}
      {isManagerOrAbove && (
        <div className="flex flex-wrap gap-2 pt-1 border-t border-slate-700/50">
          {filing.status === "draft" && (
            <button
              onClick={() => submitMut.mutate()}
              disabled={submitMut.isPending}
              className="text-xs px-3 py-1.5 bg-amber-600/20 hover:bg-amber-600/30 text-amber-300 border border-amber-500/30 rounded-lg transition-colors disabled:opacity-60"
            >
              {submitMut.isPending ? "Submitting…" : "Submit for Approval"}
            </button>
          )}

          {filing.status === "submitted" && !isSelf && (
            <>
              <button
                onClick={() => approveMut.mutate()}
                disabled={approveMut.isPending}
                className="text-xs px-3 py-1.5 bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-300 border border-emerald-500/30 rounded-lg transition-colors disabled:opacity-60"
              >
                {approveMut.isPending ? "Approving…" : "Approve"}
              </button>
              <button
                onClick={() => setShowRejectModal(true)}
                className="text-xs px-3 py-1.5 bg-red-600/20 hover:bg-red-600/30 text-red-300 border border-red-500/30 rounded-lg transition-colors"
              >
                Reject
              </button>
            </>
          )}

          {filing.status === "submitted" && isSelf && (
            <span className="text-xs text-amber-400/70 italic py-1.5">
              PMLA s.12: you submitted this filing — a different officer must approve
            </span>
          )}

          {isAdminOrAbove && filing.status === "approved" && (
            <button
              onClick={() => filedMut.mutate()}
              disabled={filedMut.isPending}
              className="text-xs px-3 py-1.5 bg-cyan-600/20 hover:bg-cyan-600/30 text-cyan-300 border border-cyan-500/30 rounded-lg transition-colors disabled:opacity-60"
            >
              {filedMut.isPending ? "Filing…" : "Mark as Filed with FIU"}
            </button>
          )}

          {isAdminOrAbove && (filing.status === "approved" || filing.status === "filed") && (
            <button
              onClick={handleExport}
              className="text-xs px-3 py-1.5 bg-violet-600/20 hover:bg-violet-600/30 text-violet-300 border border-violet-500/30 rounded-lg transition-colors"
            >
              Export FINnet XML
            </button>
          )}
        </div>
      )}

      {/* Reject modal */}
      {showRejectModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-slate-800 border border-slate-700 rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl">
            <h3 className="text-white font-semibold mb-3">Reject STR Filing</h3>
            <textarea
              rows={3}
              value={rejectReason}
              onChange={(e) => setRejectReason(e.target.value)}
              placeholder="Reason for rejection (optional)…"
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 resize-none focus:outline-none focus:border-cyan-500"
            />
            <div className="flex gap-2 mt-4 justify-end">
              <button
                onClick={() => { setShowRejectModal(false); setRejectReason(""); }}
                className="px-3 py-1.5 text-sm text-slate-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => rejectMut.mutate()}
                disabled={rejectMut.isPending}
                className="px-4 py-1.5 text-sm bg-red-600/20 hover:bg-red-600/40 text-red-300 border border-red-500/30 rounded-lg transition-colors disabled:opacity-60"
              >
                {rejectMut.isPending ? "Rejecting…" : "Confirm Reject"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function STRFilingsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: caseId } = use(params);
  const router = useRouter();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);

  const [showCreate, setShowCreate] = useState(false);
  const [createError, setCreateError] = useState("");

  // Draft form fields
  const [reportType, setReportType] = useState("STR");
  const [suspicionBasis, setSuspicionBasis] = useState("");
  const [narrative, setNarrative] = useState("");
  const [evidenceSummary, setEvidenceSummary] = useState("");
  const [subjectName, setSubjectName] = useState("");
  const [subjectPAN, setSubjectPAN] = useState("");
  const [entityName, setEntityName] = useState("");
  const [reportingOfficer, setReportingOfficer] = useState("");

  const { data: filingData, isLoading } = useQuery<{ filings: STRFiling[] }>({
    queryKey: ["str-filings", caseId],
    queryFn: () => api.get<{ filings: STRFiling[] }>(`/api/v1/cases/${caseId}/str`),
    enabled: !!caseId,
  });

  const filings = filingData?.filings ?? [];

  const createMut = useMutation({
    mutationFn: () => api.post<STRFiling>(`/api/v1/cases/${caseId}/str`, {
      report_type: reportType,
      suspicion_basis: suspicionBasis,
      narrative,
      evidence_summary: evidenceSummary,
      filing_entity: { name: entityName, reporting_officer: reportingOfficer },
      subject: { name: subjectName, pan: subjectPAN },
      transactions: [],
    }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["str-filings", caseId] });
      setShowCreate(false);
      setSuspicionBasis("");
      setNarrative("");
      setEvidenceSummary("");
      setSubjectName("");
      setSubjectPAN("");
      setEntityName("");
      setReportingOfficer("");
      setCreateError("");
    },
    onError: (err) => {
      setCreateError(err instanceof APIError ? err.message : "Failed to create filing");
    },
  });

  const isManagerOrAbove = user && ["manager", "admin", "super_admin"].includes(user.role);

  if (!isManagerOrAbove) {
    return (
      <div className="p-6 max-w-4xl mx-auto">
        <button onClick={() => router.back()} className="text-slate-400 hover:text-slate-200 text-sm mb-6">← Back</button>
        <div className="bg-red-900/20 border border-red-800/30 rounded-xl p-8 text-center">
          <p className="text-red-400 font-medium">Access Restricted</p>
          <p className="text-slate-500 text-sm mt-1">STR/SAR/CTR filings are restricted to compliance officers (manager role and above).</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-4xl mx-auto">
      {/* Nav */}
      <div className="flex items-center gap-3 mb-6">
        <button
          onClick={() => router.push(`/cases/${caseId}`)}
          className="text-slate-400 hover:text-slate-200 text-sm flex items-center gap-1"
        >
          ← Case Detail
        </button>
        <div className="h-4 border-l border-slate-700" />
        <span className="text-slate-400 text-sm">STR / SAR / CTR Filings</span>
      </div>

      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-white text-xl font-bold">Regulatory Filings</h1>
          <p className="text-slate-400 text-sm mt-0.5">
            PMLA 2002 · Dual-authorization maker-checker workflow · FIU-IND FINnet 2.0
          </p>
        </div>
        {isManagerOrAbove && (
          <button
            onClick={() => setShowCreate(true)}
            className="bg-cyan-600/20 hover:bg-cyan-600/30 text-cyan-300 border border-cyan-500/30 text-sm font-medium px-4 py-2 rounded-lg transition-colors"
          >
            + New Filing
          </button>
        )}
      </div>

      {/* PMLA info banner */}
      <div className="bg-amber-900/10 border border-amber-800/20 rounded-xl p-4 mb-6 text-xs text-amber-300/80 leading-relaxed">
        <strong className="text-amber-300">PMLA s.12</strong> — Records must be retained for 10 years.
        A filing must be <em>submitted</em> by one officer and <em>approved</em> by a different officer (maker-checker).
        Analysts are excluded from this workflow (PMLA s.8A tipping-off restriction).
      </div>

      {/* Create form */}
      {showCreate && (
        <div className="bg-slate-800/80 border border-slate-700 rounded-xl p-6 mb-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-white font-semibold">New STR/SAR/CTR Draft</h2>
            <button onClick={() => setShowCreate(false)} className="text-slate-400 hover:text-white text-lg leading-none">×</button>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-slate-400 text-xs mb-1.5 block">Report Type</label>
              <select
                value={reportType}
                onChange={(e) => setReportType(e.target.value)}
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-cyan-500"
              >
                <option value="STR">STR — Suspicious Transaction Report</option>
                <option value="SAR">SAR — Suspicious Activity Report</option>
                <option value="CTR">CTR — Cash Transaction Report</option>
              </select>
            </div>

            <div>
              <label className="text-slate-400 text-xs mb-1.5 block">Suspicion Basis <span className="text-red-400">*</span></label>
              <input
                type="text"
                value={suspicionBasis}
                onChange={(e) => setSuspicionBasis(e.target.value)}
                placeholder="e.g. Unusual cash transactions inconsistent with profile"
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-cyan-500"
              />
            </div>

            <div className="col-span-2">
              <label className="text-slate-400 text-xs mb-1.5 block">Narrative</label>
              <textarea
                rows={3}
                value={narrative}
                onChange={(e) => setNarrative(e.target.value)}
                placeholder="Detailed description of suspicious activity…"
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 resize-none focus:outline-none focus:border-cyan-500"
              />
            </div>

            <div className="col-span-2">
              <label className="text-slate-400 text-xs mb-1.5 block">Evidence Summary</label>
              <textarea
                rows={2}
                value={evidenceSummary}
                onChange={(e) => setEvidenceSummary(e.target.value)}
                placeholder="Supporting evidence (transaction records, documents, etc.)…"
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 resize-none focus:outline-none focus:border-cyan-500"
              />
            </div>

            <div>
              <label className="text-slate-400 text-xs mb-1.5 block">Subject Name</label>
              <input
                type="text"
                value={subjectName}
                onChange={(e) => setSubjectName(e.target.value)}
                placeholder="Individual or entity name"
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-cyan-500"
              />
            </div>

            <div>
              <label className="text-slate-400 text-xs mb-1.5 block">PAN</label>
              <input
                type="text"
                value={subjectPAN}
                onChange={(e) => setSubjectPAN(e.target.value)}
                placeholder="ABCDE1234F"
                maxLength={10}
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 font-mono uppercase focus:outline-none focus:border-cyan-500"
              />
            </div>

            <div>
              <label className="text-slate-400 text-xs mb-1.5 block">Filing Entity</label>
              <input
                type="text"
                value={entityName}
                onChange={(e) => setEntityName(e.target.value)}
                placeholder="Bank / NBFC name"
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-cyan-500"
              />
            </div>

            <div>
              <label className="text-slate-400 text-xs mb-1.5 block">Reporting Officer</label>
              <input
                type="text"
                value={reportingOfficer}
                onChange={(e) => setReportingOfficer(e.target.value)}
                placeholder="Officer name"
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-cyan-500"
              />
            </div>
          </div>

          {createError && (
            <p className="text-red-400 text-xs mt-3 bg-red-900/20 border border-red-800/30 px-3 py-2 rounded-lg">{createError}</p>
          )}

          <div className="flex justify-end gap-2 mt-5">
            <button
              onClick={() => setShowCreate(false)}
              className="px-4 py-2 text-sm text-slate-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={() => createMut.mutate()}
              disabled={!suspicionBasis.trim() || createMut.isPending}
              className="px-4 py-2 text-sm bg-cyan-600/20 hover:bg-cyan-600/30 text-cyan-300 border border-cyan-500/30 rounded-lg transition-colors disabled:opacity-60"
            >
              {createMut.isPending ? "Saving…" : "Save Draft"}
            </button>
          </div>
        </div>
      )}

      {/* Filings list */}
      {isLoading ? (
        <div className="flex items-center justify-center h-48">
          <div className="w-6 h-6 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : filings.length === 0 ? (
        <div className="text-center py-16 text-slate-500">
          <p className="text-4xl mb-3">📋</p>
          <p className="font-medium">No filings yet</p>
          <p className="text-sm mt-1">Create a new draft to start the STR/SAR/CTR workflow.</p>
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          {filings.map((f) => (
            <FilingCard
              key={f.id}
              filing={f}
              caseId={caseId}
              userId={user?.id ?? ""}
              userRole={user?.role ?? ""}
              onAction={() => {}}
            />
          ))}
        </div>
      )}
    </div>
  );
}
