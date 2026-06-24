"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/hooks/useAuth";
import { AlertTriangle, Plus, ChevronDown, ChevronRight, Clock, CheckCircle2, XCircle, Pause, Loader2 } from "lucide-react";

interface DSAR {
  id: string;
  subject_name: string;
  subject_email: string;
  request_date: string;
  deadline: string;
  status: "received" | "in_progress" | "completed" | "rejected" | "deferred";
  outcome?: "pseudonymised" | "full_erasure" | "denied";
  legal_basis?: string;
  processed_by?: string;
  processed_at?: string;
  notes?: string;
  created_at: string;
}

interface AnonymisationRecord {
  dsar_id: string;
  subject_id: string;
  fields_masked: string[];
  pseudonym: string;
  legal_basis: string;
  anonymised_at: string;
}

const STATUS_CONFIG: Record<string, { label: string; color: string; Icon: React.ElementType }> = {
  received:    { label: "Received",    color: "bg-blue-500/15 text-blue-300 border-blue-500/30",    Icon: Clock },
  in_progress: { label: "In Progress", color: "bg-amber-500/15 text-amber-300 border-amber-500/30", Icon: Loader2 },
  completed:   { label: "Completed",   color: "bg-emerald-500/15 text-emerald-300 border-emerald-500/30", Icon: CheckCircle2 },
  rejected:    { label: "Rejected",    color: "bg-red-500/15 text-red-300 border-red-500/30",       Icon: XCircle },
  deferred:    { label: "Deferred",    color: "bg-slate-500/15 text-slate-300 border-slate-500/30", Icon: Pause },
};

const OUTCOME_LABELS: Record<string, string> = {
  pseudonymised: "Pseudonymised",
  full_erasure:  "Full Erasure",
  denied:        "Denied",
};

function deadlineBadge(deadlineStr: string) {
  const days = Math.ceil((new Date(deadlineStr).getTime() - Date.now()) / 86_400_000);
  if (days < 0) return <span className="text-[11px] font-medium text-red-400">Overdue</span>;
  if (days <= 7) return <span className="text-[11px] font-medium text-red-400">{days}d left</span>;
  if (days <= 14) return <span className="text-[11px] font-medium text-amber-400">{days}d left</span>;
  return <span className="text-[11px] text-slate-500">{days}d left</span>;
}

function fmtDate(s: string) {
  return new Date(s).toLocaleDateString("en-GB", { day: "2-digit", month: "short", year: "numeric" });
}

// ── Log New Request Modal ────────────────────────────────────────────────────
function LogDSARModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [form, setForm] = useState({ subject_name: "", subject_email: "", notes: "" });
  const [err, setErr] = useState("");
  const mut = useMutation({
    mutationFn: () => api.post<DSAR>("/api/v1/gdpr/dsar", form),
    onSuccess: () => { onCreated(); onClose(); },
    onError: (e: Error) => setErr(e.message),
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-[#0d1a2d] border border-slate-700 rounded-xl w-full max-w-md shadow-2xl p-6">
        <h3 className="text-base font-semibold text-slate-100 mb-1">Log GDPR Erasure Request</h3>
        <p className="text-slate-500 text-xs mb-5">
          A 30-day response deadline is automatically computed from today (GDPR Art. 12(3)).
        </p>
        <div className="space-y-3">
          <div>
            <label className="block text-xs text-slate-400 mb-1">Subject name <span className="text-red-400">*</span></label>
            <input
              value={form.subject_name}
              onChange={(e) => setForm({ ...form, subject_name: e.target.value })}
              placeholder="Full name as provided by the requester"
              className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-cyan-500"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Contact email</label>
            <input
              type="email"
              value={form.subject_email}
              onChange={(e) => setForm({ ...form, subject_email: e.target.value })}
              placeholder="requester@example.com"
              className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-cyan-500"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Notes</label>
            <textarea
              rows={3}
              value={form.notes}
              onChange={(e) => setForm({ ...form, notes: e.target.value })}
              placeholder="Channel received, supporting documentation, etc."
              className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-cyan-500 resize-none"
            />
          </div>
        </div>
        {err && <p className="text-red-400 text-xs mt-2">{err}</p>}
        <div className="flex gap-2 mt-5">
          <button
            onClick={() => mut.mutate()}
            disabled={!form.subject_name || mut.isPending}
            className="flex-1 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm font-medium rounded transition-colors"
          >
            {mut.isPending ? "Logging…" : "Log Request"}
          </button>
          <button onClick={onClose} className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm transition-colors">
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Process Confirmation Modal ───────────────────────────────────────────────
function ProcessConfirmModal({ dsar, onClose, onDone }: { dsar: DSAR; onClose: () => void; onDone: () => void }) {
  const [confirmed, setConfirmed] = useState(false);
  const [err, setErr] = useState("");
  const mut = useMutation({
    mutationFn: () => api.post(`/api/v1/gdpr/dsar/${dsar.id}/process`, {}),
    onSuccess: () => { onDone(); onClose(); },
    onError: (e: Error) => setErr(e.message),
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-[#0d1a2d] border border-red-500/40 rounded-xl w-full max-w-md shadow-2xl p-6">
        <div className="flex items-center gap-2 mb-3">
          <AlertTriangle size={18} className="text-red-400 flex-shrink-0" />
          <h3 className="text-base font-semibold text-slate-100">Irreversible Action</h3>
        </div>
        <p className="text-slate-400 text-sm mb-4">
          This will pseudonymise all personal data for <strong className="text-slate-200">{dsar.subject_name}</strong> across every case in the system.
          Fields affected: name, ID number, date of birth, addresses, phones, emails.
        </p>
        <p className="text-slate-500 text-xs mb-1 font-medium">Legal basis applied:</p>
        <p className="text-slate-400 text-xs mb-5 leading-relaxed">
          If AML retention window is active: GDPR Art. 17(3)(b) — erasure exemption; pseudonymisation applied.<br />
          If retention expired: GDPR Art. 17(1) — full erasure applied.
        </p>
        <label className="flex items-start gap-2 cursor-pointer mb-4">
          <input
            type="checkbox"
            checked={confirmed}
            onChange={(e) => setConfirmed(e.target.checked)}
            className="mt-0.5"
          />
          <span className="text-sm text-slate-300">I confirm this action is authorised and cannot be undone</span>
        </label>
        {err && <p className="text-red-400 text-xs mb-2">{err}</p>}
        <div className="flex gap-2">
          <button
            onClick={() => mut.mutate()}
            disabled={!confirmed || mut.isPending}
            className="flex-1 py-2 bg-red-600 hover:bg-red-500 disabled:opacity-50 text-white text-sm font-medium rounded transition-colors"
          >
            {mut.isPending ? "Processing…" : "Pseudonymise / Erase"}
          </button>
          <button onClick={onClose} className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm transition-colors">
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Single DSAR Row ──────────────────────────────────────────────────────────
function DSARRow({ dsar, onProcess, onStatusChange }: {
  dsar: DSAR;
  onProcess: (d: DSAR) => void;
  onStatusChange: (id: string, status: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const cfg = STATUS_CONFIG[dsar.status] ?? STATUS_CONFIG.received;
  const StatusIcon = cfg.Icon;

  const { data: records } = useQuery<AnonymisationRecord[]>({
    queryKey: ["dsar-records", dsar.id],
    queryFn: () => api.get(`/api/v1/gdpr/dsar/${dsar.id}/records`),
    enabled: expanded && dsar.status === "completed",
  });

  return (
    <>
      <tr className="border-b border-slate-700/40 hover:bg-slate-800/30 transition-colors">
        <td className="px-4 py-3 text-sm text-slate-200 max-w-[160px] truncate">{dsar.subject_name}</td>
        <td className="px-4 py-3 text-xs text-slate-400">{dsar.subject_email || "—"}</td>
        <td className="px-4 py-3 text-xs text-slate-400">{fmtDate(dsar.request_date)}</td>
        <td className="px-4 py-3">
          <div className="flex flex-col">
            <span className="text-xs text-slate-400">{fmtDate(dsar.deadline)}</span>
            {deadlineBadge(dsar.deadline)}
          </div>
        </td>
        <td className="px-4 py-3">
          <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full border text-[11px] font-medium ${cfg.color}`}>
            <StatusIcon size={10} />
            {cfg.label}
          </span>
        </td>
        <td className="px-4 py-3 text-xs text-slate-400">
          {dsar.outcome ? OUTCOME_LABELS[dsar.outcome] ?? dsar.outcome : "—"}
        </td>
        <td className="px-4 py-3">
          <div className="flex items-center gap-2">
            {dsar.status === "received" && (
              <button
                onClick={() => onStatusChange(dsar.id, "in_progress")}
                className="text-xs text-cyan-400 hover:text-cyan-300 transition-colors"
              >
                Start
              </button>
            )}
            {(dsar.status === "received" || dsar.status === "in_progress") && (
              <>
                <button
                  onClick={() => onProcess(dsar)}
                  className="text-xs text-amber-400 hover:text-amber-300 transition-colors"
                >
                  Pseudonymise
                </button>
                <button
                  onClick={() => onStatusChange(dsar.id, "rejected")}
                  className="text-xs text-red-400 hover:text-red-300 transition-colors"
                >
                  Reject
                </button>
                <button
                  onClick={() => onStatusChange(dsar.id, "deferred")}
                  className="text-xs text-slate-400 hover:text-slate-300 transition-colors"
                >
                  Defer
                </button>
              </>
            )}
            {dsar.status === "completed" && (
              <button
                onClick={() => setExpanded(!expanded)}
                className="flex items-center gap-1 text-xs text-slate-400 hover:text-slate-200 transition-colors"
              >
                {expanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
                Audit trail
              </button>
            )}
          </div>
        </td>
      </tr>
      {expanded && dsar.status === "completed" && (
        <tr className="bg-slate-900/40">
          <td colSpan={7} className="px-6 py-3">
            {!records || records.length === 0 ? (
              <p className="text-xs text-slate-500">No anonymisation records found.</p>
            ) : (
              <div className="space-y-2">
                {records.map((rec, i) => (
                  <div key={i} className="text-xs text-slate-400">
                    <span className="text-slate-300 font-mono">{rec.pseudonym}</span>
                    {" · "}
                    Fields: {rec.fields_masked.join(", ")}
                    {" · "}
                    {new Date(rec.anonymised_at).toLocaleString("en-GB")}
                    <p className="text-slate-600 mt-0.5 text-[11px]">{rec.legal_basis}</p>
                  </div>
                ))}
              </div>
            )}
          </td>
        </tr>
      )}
    </>
  );
}

// ── Main Page ────────────────────────────────────────────────────────────────
export default function GDPRPage() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [processing, setProcessing] = useState<DSAR | null>(null);

  const { data: dsars = [], isLoading } = useQuery<DSAR[]>({
    queryKey: ["gdpr-dsar"],
    queryFn: () => api.get("/api/v1/gdpr/dsar"),
  });

  const statusMut = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.patch(`/api/v1/gdpr/dsar/${id}`, { status, legal_basis: "" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["gdpr-dsar"] }),
  });

  const canWrite = ["manager", "admin", "super_admin"].includes(user?.role ?? "");

  const open = dsars.filter((d) => d.status !== "completed" && d.status !== "rejected");
  const closed = dsars.filter((d) => d.status === "completed" || d.status === "rejected");

  const overdueCount = open.filter((d) => new Date(d.deadline) < new Date()).length;
  const nearCount = open.filter((d) => {
    const days = Math.ceil((new Date(d.deadline).getTime() - Date.now()) / 86_400_000);
    return days >= 0 && days <= 7;
  }).length;

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <a href="/settings" className="text-slate-400 hover:text-slate-200 text-sm transition-colors">
              ← Settings
            </a>
          </div>
          <h1 className="text-xl font-semibold text-slate-100">GDPR Erasure Requests (DSAR)</h1>
          <p className="text-slate-500 text-xs mt-0.5">
            Data Subject Access Requests under GDPR Art. 17 — 30-day response window applies
          </p>
        </div>
        {canWrite && (
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-1.5 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium rounded-lg transition-colors"
          >
            <Plus size={14} />
            Log New Request
          </button>
        )}
      </div>

      {/* Alerts */}
      {(overdueCount > 0 || nearCount > 0) && (
        <div className="mb-5 p-3 bg-red-500/10 border border-red-500/30 rounded-lg flex items-start gap-2">
          <AlertTriangle size={16} className="text-red-400 flex-shrink-0 mt-0.5" />
          <div className="text-xs text-red-300">
            {overdueCount > 0 && <span className="font-medium">{overdueCount} overdue. </span>}
            {nearCount > 0 && <span>{nearCount} due within 7 days. </span>}
            GDPR requires response within 30 days (Art. 12(3)). Failure to respond is a reportable breach.
          </div>
        </div>
      )}

      {/* Summary bar */}
      <div className="grid grid-cols-4 gap-3 mb-6">
        {[
          { label: "Total",       value: dsars.length,  color: "text-slate-200" },
          { label: "Open",        value: open.length,   color: "text-cyan-300" },
          { label: "Overdue",     value: overdueCount,  color: overdueCount > 0 ? "text-red-400" : "text-slate-400" },
          { label: "Completed",   value: closed.length, color: "text-emerald-400" },
        ].map(({ label, value, color }) => (
          <div key={label} className="p-4 bg-[#0e1929] border border-[#1e3050] rounded-xl text-center">
            <div className={`text-2xl font-bold ${color}`}>{value}</div>
            <div className="text-slate-500 text-xs mt-0.5">{label}</div>
          </div>
        ))}
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center h-40 text-slate-500 text-sm">
          <Loader2 size={18} className="animate-spin mr-2" /> Loading requests…
        </div>
      ) : dsars.length === 0 ? (
        <div className="text-center py-16 text-slate-500 text-sm">
          No DSAR requests logged yet.
          {canWrite && (
            <button onClick={() => setShowCreate(true)} className="block mt-2 text-cyan-400 hover:text-cyan-300 mx-auto text-xs">
              Log your first request
            </button>
          )}
        </div>
      ) : (
        <>
          {open.length > 0 && (
            <div className="mb-8">
              <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-3">
                Open ({open.length})
              </h2>
              <div className="bg-[#0e1929] border border-[#1e3050] rounded-xl overflow-hidden">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-slate-700/60">
                      {["Subject", "Email", "Received", "Deadline", "Status", "Outcome", "Actions"].map((h) => (
                        <th key={h} className="px-4 py-2.5 text-left text-[11px] font-medium text-slate-500 uppercase tracking-wider">
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {open.map((d) => (
                      <DSARRow
                        key={d.id}
                        dsar={d}
                        onProcess={setProcessing}
                        onStatusChange={(id, status) => statusMut.mutate({ id, status })}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {closed.length > 0 && (
            <div>
              <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-3">
                Closed ({closed.length})
              </h2>
              <div className="bg-[#0e1929] border border-[#1e3050] rounded-xl overflow-hidden">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-slate-700/60">
                      {["Subject", "Email", "Received", "Deadline", "Status", "Outcome", "Actions"].map((h) => (
                        <th key={h} className="px-4 py-2.5 text-left text-[11px] font-medium text-slate-500 uppercase tracking-wider">
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {closed.map((d) => (
                      <DSARRow
                        key={d.id}
                        dsar={d}
                        onProcess={setProcessing}
                        onStatusChange={(id, status) => statusMut.mutate({ id, status })}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}

      {showCreate && (
        <LogDSARModal
          onClose={() => setShowCreate(false)}
          onCreated={() => qc.invalidateQueries({ queryKey: ["gdpr-dsar"] })}
        />
      )}
      {processing && (
        <ProcessConfirmModal
          dsar={processing}
          onClose={() => setProcessing(null)}
          onDone={() => qc.invalidateQueries({ queryKey: ["gdpr-dsar"] })}
        />
      )}
    </div>
  );
}
