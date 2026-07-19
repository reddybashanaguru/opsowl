"use client";

import { use, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Users, ExternalLink, Plus, CheckCircle, AlertTriangle, XCircle, Loader2 } from "lucide-react";
import { api } from "@/lib/api";
import { useConfig } from "@/lib/hooks/useConfig";
import { useAuthStore } from "@/lib/stores/auth.store";
import { CaseSubNav } from "@/components/cases/CaseSubNav";

interface UBORecord {
  id: string;
  case_id: string;
  entity_name: string;
  owner_name: string;
  owner_nationality: string;
  owner_id_type: string;
  owner_id_number: string;
  owner_birth_date: string;
  ownership_pct: number;
  control_type: string;
  status: "pending" | "verified" | "discrepant" | "n/a";
  registry_url: string;
  verified_by?: string;
  verified_at?: string;
  discrepancy_note: string;
  created_at: string;
}

import { useCaseSummary } from "@/lib/hooks/useCaseSummary";

const STATUS_CONFIG = {
  pending:    { label: "Pending",    color: "bg-amber-500/15 text-amber-400",   Icon: Loader2 },
  verified:   { label: "Verified",   color: "bg-green-500/15 text-green-400",   Icon: CheckCircle },
  discrepant: { label: "Discrepant", color: "bg-red-500/15 text-red-400",       Icon: AlertTriangle },
  "n/a":      { label: "N/A",        color: "bg-slate-500/15 text-slate-400",   Icon: XCircle },
};

const CONTROL_TYPE_LABELS: Record<string, string> = {
  share:         "Share ownership",
  voting_right:  "Voting rights",
  other_control: "Other control",
};

function AddUBOModal({ caseId, onClose, onCreated }: { caseId: string; onClose: () => void; onCreated: () => void }) {
  const qc = useQueryClient();
  const [form, setForm] = useState({
    entity_name: "", owner_name: "", owner_nationality: "",
    owner_id_type: "PASSPORT", owner_id_number: "", owner_birth_date: "",
    ownership_pct: "", control_type: "share",
  });

  const mutation = useMutation({
    mutationFn: (data: typeof form) =>
      api.post(`/api/v1/cases/${caseId}/ubo`, { ...data, ownership_pct: parseFloat(data.ownership_pct) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["ubo", caseId] });
      onCreated();
      onClose();
    },
  });

  const set = (field: string) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setForm((f) => ({ ...f, [field]: e.target.value }));

  const inputCls = "w-full bg-[#0d1829] border border-[#1e3050] rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-cyan-500";
  const labelCls = "block text-xs text-slate-400 mb-1";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl w-full max-w-2xl">
        <div className="flex items-center justify-between p-5 border-b border-[#1e3050]">
          <h2 className="text-slate-100 font-semibold">Add Beneficial Owner</h2>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300 text-lg leading-none">×</button>
        </div>
        <div className="p-5 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className={labelCls}>Legal Entity Name *</label>
              <input className={inputCls} placeholder="e.g. Aegean Holdings Ltd" value={form.entity_name} onChange={set("entity_name")} />
            </div>
            <div>
              <label className={labelCls}>Beneficial Owner Name *</label>
              <input className={inputCls} placeholder="Full legal name" value={form.owner_name} onChange={set("owner_name")} />
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className={labelCls}>Ownership % *</label>
              <input className={inputCls} type="number" min="0" max="100" step="0.01" placeholder="e.g. 35.00" value={form.ownership_pct} onChange={set("ownership_pct")} />
            </div>
            <div>
              <label className={labelCls}>Control Type</label>
              <select className={inputCls} value={form.control_type} onChange={set("control_type")}>
                <option value="share">Share ownership</option>
                <option value="voting_right">Voting rights</option>
                <option value="other_control">Other control</option>
              </select>
            </div>
            <div>
              <label className={labelCls}>Nationality</label>
              <input className={inputCls} placeholder="ISO code, e.g. GR" value={form.owner_nationality} onChange={set("owner_nationality")} />
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className={labelCls}>ID Type</label>
              <select className={inputCls} value={form.owner_id_type} onChange={set("owner_id_type")}>
                <option value="PASSPORT">Passport</option>
                <option value="NATIONAL_ID">National ID</option>
                <option value="TAX_ID">Tax ID</option>
                <option value="OTHER">Other</option>
              </select>
            </div>
            <div>
              <label className={labelCls}>ID Number</label>
              <input className={inputCls} placeholder="Document number" value={form.owner_id_number} onChange={set("owner_id_number")} />
            </div>
            <div>
              <label className={labelCls}>Date of Birth</label>
              <input className={inputCls} type="date" value={form.owner_birth_date} onChange={set("owner_birth_date")} />
            </div>
          </div>
          {mutation.error && (
            <p className="text-red-400 text-xs">{(mutation.error as Error).message}</p>
          )}
        </div>
        <div className="flex justify-end gap-3 p-5 border-t border-[#1e3050]">
          <button onClick={onClose} className="px-4 py-2 text-sm text-slate-400 hover:text-slate-200 transition-colors">Cancel</button>
          <button
            onClick={() => mutation.mutate(form)}
            disabled={!form.entity_name || !form.owner_name || !form.ownership_pct || mutation.isPending}
            className="px-4 py-2 text-sm bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg transition-colors flex items-center gap-2"
          >
            {mutation.isPending && <Loader2 size={13} className="animate-spin" />}
            Add Record
          </button>
        </div>
      </div>
    </div>
  );
}

function DiscrepancyModal({ record, onClose, onSaved }: { record: UBORecord; onClose: () => void; onSaved: () => void }) {
  const qc = useQueryClient();
  const [note, setNote] = useState(record.discrepancy_note);

  const mutation = useMutation({
    mutationFn: () =>
      api.patch(`/api/v1/cases/${record.case_id}/ubo/${record.id}`, { status: "discrepant", discrepancy_note: note }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["ubo", record.case_id] });
      onSaved();
      onClose();
    },
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl w-full max-w-md">
        <div className="flex items-center justify-between p-5 border-b border-[#1e3050]">
          <h2 className="text-slate-100 font-semibold">Mark Discrepant</h2>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300 text-lg leading-none">×</button>
        </div>
        <div className="p-5">
          <label className="block text-xs text-slate-400 mb-1">Describe the discrepancy *</label>
          <textarea
            rows={3}
            className="w-full bg-[#0d1829] border border-[#1e3050] rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-cyan-500"
            placeholder="e.g. Declared 35% but registry shows 28%"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
        </div>
        <div className="flex justify-end gap-3 p-5 border-t border-[#1e3050]">
          <button onClick={onClose} className="px-4 py-2 text-sm text-slate-400 hover:text-slate-200">Cancel</button>
          <button
            onClick={() => mutation.mutate()}
            disabled={!note.trim() || mutation.isPending}
            className="px-4 py-2 text-sm bg-red-600 hover:bg-red-500 disabled:opacity-50 text-white rounded-lg transition-colors"
          >
            {mutation.isPending ? "Saving…" : "Mark Discrepant"}
          </button>
        </div>
      </div>
    </div>
  );
}

export default function CaseUBOPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: caseId } = use(params);
  const cfg = useConfig();
  const hasRole = useAuthStore((s) => s.hasRole);
  const qc = useQueryClient();

  const [showAdd, setShowAdd] = useState(false);
  const [discrepancyTarget, setDiscrepancyTarget] = useState<UBORecord | null>(null);

  const caseData = useCaseSummary(caseId);

  const { data: records = [], isFetching } = useQuery<UBORecord[]>({
    queryKey: ["ubo", caseId],
    queryFn: () => api.get(`/api/v1/cases/${caseId}/ubo`),
  });

  const verifyMutation = useMutation({
    mutationFn: (id: string) =>
      api.patch(`/api/v1/cases/${caseId}/ubo/${id}`, { status: "verified", discrepancy_note: "" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ubo", caseId] }),
  });

  const naMutation = useMutation({
    mutationFn: (id: string) =>
      api.patch(`/api/v1/cases/${caseId}/ubo/${id}`, { status: "n/a", discrepancy_note: "" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ubo", caseId] }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/cases/${caseId}/ubo/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ubo", caseId] }),
  });

  const canManage = hasRole("manager");
  const pendingCount = records.filter((r) => r.status === "pending" || r.status === "discrepant").length;

  return (
    <div className="px-8 py-6">
      <CaseSubNav caseId={caseId} activeTab="ubo" caseNumber={caseData?.case_number} />

      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-slate-100 font-semibold text-lg flex items-center gap-2">
            <Users size={18} className="text-cyan-400" />
            Beneficial Ownership (UBO)
          </h1>
          <p className="text-slate-400 text-sm mt-0.5">
            EU AMLR 2024 Art. 42 — verify all owners with ≥{cfg.ubo_threshold_pct ?? 25}% control before case closure
          </p>
        </div>
        {canManage && (
          <button
            onClick={() => setShowAdd(true)}
            className="flex items-center gap-2 px-4 py-2 text-sm bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg transition-colors"
          >
            <Plus size={14} />
            Add UBO Record
          </button>
        )}
      </div>

      {/* Closure warning banner */}
      {pendingCount > 0 && (
        <div className="mb-5 flex items-start gap-3 p-4 bg-amber-500/10 border border-amber-500/30 rounded-xl">
          <AlertTriangle size={16} className="text-amber-400 mt-0.5 flex-shrink-0" />
          <div>
            <p className="text-amber-300 text-sm font-medium">Case cannot be closed</p>
            <p className="text-amber-400/80 text-xs mt-0.5">
              {pendingCount} beneficial owner record{pendingCount > 1 ? "s" : ""} require verification before this case can move to review or be closed (EU AMLR 2024 Art. 42).
            </p>
          </div>
        </div>
      )}

      {/* Records */}
      {isFetching && records.length === 0 ? (
        <div className="flex items-center gap-2 text-slate-400 text-sm py-8"><Loader2 size={16} className="animate-spin" /> Loading…</div>
      ) : records.length === 0 ? (
        <div className="text-center py-16 text-slate-500">
          <Users size={32} className="mx-auto mb-3 opacity-30" />
          <p className="text-sm">No UBO records added yet.</p>
          <p className="text-xs mt-1">Add beneficial owners for corporate subjects involved in this case.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {records.map((rec) => {
            const s = STATUS_CONFIG[rec.status] ?? STATUS_CONFIG.pending;
            return (
              <div key={rec.id} className="bg-[#131f30] border border-[#1e3050] rounded-xl p-5">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3 flex-wrap">
                      <span className="text-slate-200 font-medium text-sm">{rec.owner_name}</span>
                      <span className={`px-2 py-0.5 rounded text-xs font-semibold uppercase ${s.color}`}>
                        {s.label}
                      </span>
                      <span className="text-slate-400 text-xs">{rec.ownership_pct.toFixed(1)}% — {CONTROL_TYPE_LABELS[rec.control_type] ?? rec.control_type}</span>
                    </div>
                    <p className="text-slate-400 text-xs mt-1">
                      Entity: <span className="text-slate-300">{rec.entity_name}</span>
                      {rec.owner_nationality && <> · Nationality: <span className="text-slate-300">{rec.owner_nationality.toUpperCase()}</span></>}
                      {rec.owner_id_number && <> · {rec.owner_id_type}: <span className="text-slate-300">{rec.owner_id_number}</span></>}
                    </p>
                    {rec.discrepancy_note && (
                      <p className="text-red-400 text-xs mt-1.5 flex items-start gap-1">
                        <AlertTriangle size={11} className="mt-0.5 flex-shrink-0" />
                        {rec.discrepancy_note}
                      </p>
                    )}
                    {rec.verified_at && (
                      <p className="text-slate-500 text-xs mt-1">
                        Verified {new Date(rec.verified_at).toLocaleDateString()}
                      </p>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-2 flex-shrink-0">
                    {rec.registry_url && (
                      <a
                        href={rec.registry_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center gap-1 px-3 py-1.5 text-xs text-cyan-400 border border-cyan-500/30 rounded-lg hover:bg-cyan-500/10 transition-colors"
                      >
                        <ExternalLink size={11} />
                        Registry
                      </a>
                    )}
                    {canManage && rec.status !== "verified" && (
                      <button
                        onClick={() => verifyMutation.mutate(rec.id)}
                        disabled={verifyMutation.isPending}
                        className="px-3 py-1.5 text-xs text-green-400 border border-green-500/30 rounded-lg hover:bg-green-500/10 transition-colors"
                      >
                        Verify
                      </button>
                    )}
                    {canManage && rec.status !== "discrepant" && (
                      <button
                        onClick={() => setDiscrepancyTarget(rec)}
                        className="px-3 py-1.5 text-xs text-red-400 border border-red-500/30 rounded-lg hover:bg-red-500/10 transition-colors"
                      >
                        Discrepant
                      </button>
                    )}
                    {canManage && rec.status !== "n/a" && (
                      <button
                        onClick={() => naMutation.mutate(rec.id)}
                        className="px-3 py-1.5 text-xs text-slate-400 border border-slate-600/50 rounded-lg hover:bg-slate-700/40 transition-colors"
                      >
                        N/A
                      </button>
                    )}
                    {canManage && (
                      <button
                        onClick={() => { if (confirm("Delete this UBO record?")) deleteMutation.mutate(rec.id); }}
                        className="px-2 py-1.5 text-xs text-slate-500 hover:text-red-400 transition-colors"
                      >
                        ×
                      </button>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {showAdd && <AddUBOModal caseId={caseId} onClose={() => setShowAdd(false)} onCreated={() => {}} />}
      {discrepancyTarget && (
        <DiscrepancyModal
          record={discrepancyTarget}
          onClose={() => setDiscrepancyTarget(null)}
          onSaved={() => setDiscrepancyTarget(null)}
        />
      )}
    </div>
  );
}
