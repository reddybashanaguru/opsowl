"use client";

import { use, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Users, ExternalLink, Plus, CheckCircle, AlertTriangle, XCircle, Loader2 } from "lucide-react";
import { api } from "@/lib/api";
import { useConfig } from "@/lib/hooks/useConfig";
import { useAuthStore } from "@/lib/stores/auth.store";

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
  created_by?: string;
  verified_by?: string;
  verified_at?: string;
  discrepancy_note: string;
  created_at: string;
}


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

const ID_TYPES_BY_JURISDICTION: Record<string, { value: string; label: string }[]> = {
  india: [
    { value: "AADHAAR", label: "Aadhaar Card" },
    { value: "PAN", label: "PAN Card" },
    { value: "PASSPORT", label: "Passport" },
    { value: "VOTER_ID", label: "Voter ID" },
    { value: "DRIVING_LICENSE", label: "Driving Licence" },
    { value: "OTHER", label: "Other" },
  ],
  default: [
    { value: "PASSPORT", label: "Passport" },
    { value: "NATIONAL_ID", label: "National ID" },
    { value: "TAX_ID", label: "Tax ID" },
    { value: "OTHER", label: "Other" },
  ],
};

const REG_LABEL: Record<string, string> = {
  india: "Companies Act 2013 Section 90 / RBI KYC MD 2023",
  uae:   "UAE Federal Decree-Law No. 20 of 2018",
  eu:    "EU AMLR 2024 Art. 42",
};

interface FormErrors { entity_name?: string; owner_name?: string; ownership_pct?: string; }

function AddUBOModal({ caseId, jurisdiction, threshold, onClose, onCreated }: {
  caseId: string; jurisdiction: string; threshold: number; onClose: () => void; onCreated: () => void;
}) {
  const qc = useQueryClient();
  const [form, setForm] = useState({
    entity_name: "", owner_name: "", owner_nationality: "",
    owner_id_type: jurisdiction === "india" ? "AADHAAR" : "PASSPORT",
    owner_id_number: "", owner_birth_date: "",
    ownership_pct: "", control_type: "share",
    discrepancy_note: "",
  });
  const [errors, setErrors] = useState<FormErrors>({});
  const [touched, setTouched] = useState<Record<string, boolean>>({});

  const idTypes = ID_TYPES_BY_JURISDICTION[jurisdiction] ?? ID_TYPES_BY_JURISDICTION.default;
  const natPlaceholder = jurisdiction === "india" ? "e.g. IN" : jurisdiction === "eu" ? "e.g. GR" : "e.g. AE";

  const validate = (f: typeof form): FormErrors => {
    const e: FormErrors = {};
    if (!f.entity_name.trim()) e.entity_name = "Legal entity name is required";
    else if (f.entity_name.trim().length < 2) e.entity_name = "Must be at least 2 characters";
    if (!f.owner_name.trim()) e.owner_name = "Beneficial owner name is required";
    else if (f.owner_name.trim().length < 2) e.owner_name = "Must be at least 2 characters";
    const pct = parseFloat(f.ownership_pct);
    if (!f.ownership_pct) e.ownership_pct = "Ownership % is required";
    else if (isNaN(pct) || pct <= 0 || pct > 100) e.ownership_pct = "Must be between 0.01 and 100";
    return e;
  };

  const mutation = useMutation({
    mutationFn: (data: typeof form) =>
      api.post(`/api/v1/cases/${caseId}/ubo`, { ...data, ownership_pct: parseFloat(data.ownership_pct) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["ubo", caseId] });
      onCreated();
      onClose();
    },
  });

  const set = (field: string) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const next = { ...form, [field]: e.target.value };
    setForm(next);
    if (touched[field]) setErrors(validate(next));
  };

  const blur = (field: string) => () => {
    setTouched((t) => ({ ...t, [field]: true }));
    setErrors(validate(form));
  };

  const handleSubmit = () => {
    const allTouched = { entity_name: true, owner_name: true, ownership_pct: true };
    setTouched(allTouched);
    const e = validate(form);
    setErrors(e);
    if (Object.keys(e).length > 0) return;
    mutation.mutate(form);
  };

  const pct = parseFloat(form.ownership_pct);
  const belowThreshold = !isNaN(pct) && pct > 0 && pct < threshold;

  const inputCls = (field?: keyof FormErrors) =>
    `w-full bg-[#0d1829] border rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:outline-none transition-colors ${
      field && errors[field] ? "border-red-500 focus:border-red-400" : "border-[#1e3050] focus:border-cyan-500"
    }`;
  const labelCls = "block text-xs text-slate-400 mb-1";
  const errCls = "text-red-400 text-xs mt-1";

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
              <input
                className={inputCls("entity_name")}
                placeholder="e.g. Decentro Merchant Pvt Ltd"
                value={form.entity_name}
                onChange={set("entity_name")}
                onBlur={blur("entity_name")}
              />
              {errors.entity_name && <p className={errCls}>{errors.entity_name}</p>}
            </div>
            <div>
              <label className={labelCls}>Beneficial Owner Name *</label>
              <input
                className={inputCls("owner_name")}
                placeholder="Full legal name"
                value={form.owner_name}
                onChange={set("owner_name")}
                onBlur={blur("owner_name")}
              />
              {errors.owner_name && <p className={errCls}>{errors.owner_name}</p>}
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className={labelCls}>Ownership % *</label>
              <input
                className={inputCls("ownership_pct")}
                type="number" min="0.01" max="100" step="0.01"
                placeholder="e.g. 35.00"
                value={form.ownership_pct}
                onChange={set("ownership_pct")}
                onBlur={blur("ownership_pct")}
              />
              {errors.ownership_pct && <p className={errCls}>{errors.ownership_pct}</p>}
              {belowThreshold && !errors.ownership_pct && (
                <p className="text-amber-400 text-xs mt-1">Below ≥{threshold}% threshold — document the control basis in notes</p>
              )}
            </div>
            <div>
              <label className={labelCls}>Control Type</label>
              <select className={inputCls()} value={form.control_type} onChange={set("control_type")}>
                <option value="share">Share ownership</option>
                <option value="voting_right">Voting rights</option>
                <option value="other_control">Other control</option>
              </select>
            </div>
            <div>
              <label className={labelCls}>Nationality</label>
              <input className={inputCls()} placeholder={natPlaceholder} value={form.owner_nationality} onChange={set("owner_nationality")} />
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className={labelCls}>ID Type</label>
              <select className={inputCls()} value={form.owner_id_type} onChange={set("owner_id_type")}>
                {idTypes.map((t) => <option key={t.value} value={t.value}>{t.label}</option>)}
              </select>
            </div>
            <div>
              <label className={labelCls}>ID Number</label>
              <input className={inputCls()} placeholder="Document number" value={form.owner_id_number} onChange={set("owner_id_number")} />
            </div>
            <div>
              <label className={labelCls}>Date of Birth</label>
              <input className={inputCls()} type="date" max={new Date().toISOString().split("T")[0]} value={form.owner_birth_date} onChange={set("owner_birth_date")} />
            </div>
          </div>
          <div>
            <label className={labelCls}>
              Source / Verification Notes
              <span className="text-slate-600 ml-1 font-normal">
                {jurisdiction === "india" ? "(e.g. MCA21 annual return MGT-7 dated 15 Apr 2025)" : "(e.g. Companies House filing ref. 123456)"}
              </span>
            </label>
            <textarea
              rows={2}
              className="w-full bg-[#0d1829] border border-[#1e3050] rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-cyan-500"
              placeholder={jurisdiction === "india"
                ? "Where was this ownership verified? e.g. MCA21 Form BEN-2 / ROC filing / share register"
                : "Reference document or registry filing used to confirm this ownership"}
              value={form.discrepancy_note}
              onChange={(e) => setForm((f) => ({ ...f, discrepancy_note: e.target.value }))}
            />
          </div>
          {mutation.error && (
            <p className="text-red-400 text-xs">{(mutation.error as Error).message}</p>
          )}
        </div>
        <div className="flex justify-end gap-3 p-5 border-t border-[#1e3050]">
          <button onClick={onClose} className="px-4 py-2 text-sm text-slate-400 hover:text-slate-200 transition-colors">Cancel</button>
          <button
            onClick={handleSubmit}
            disabled={mutation.isPending}
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
  const hasRole  = useAuthStore((s) => s.hasRole);
  const currentUser = useAuthStore((s) => s.user);
  const qc = useQueryClient();

  const [showAdd, setShowAdd] = useState(false);
  const [discrepancyTarget, setDiscrepancyTarget] = useState<UBORecord | null>(null);

  const { data: records = [], isFetching } = useQuery<UBORecord[]>({
    queryKey: ["ubo", caseId],
    queryFn: () => api.get(`/api/v1/cases/${caseId}/ubo`),
  });

  const verifyMutation = useMutation({
    // discrepancy_note intentionally omitted — backend preserves existing source note on verify
    mutationFn: (id: string) =>
      api.patch(`/api/v1/cases/${caseId}/ubo/${id}`, { status: "verified" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ubo", caseId] }),
  });

  const naMutation = useMutation({
    mutationFn: (id: string) =>
      api.patch(`/api/v1/cases/${caseId}/ubo/${id}`, { status: "n/a" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ubo", caseId] }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/cases/${caseId}/ubo/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ubo", caseId] }),
  });

  const canAdd    = hasRole("analyst");   // analysts research and capture UBO records
  const canVerify = hasRole("manager");   // managers sign off on verification (4-eyes)
  const pendingCount = records.filter((r) => r.status === "pending" || r.status === "discrepant").length;
  const jurisdiction = cfg.jurisdiction ?? "india";
  const threshold = cfg.ubo_threshold_pct || 25;
  const regLabel = REG_LABEL[jurisdiction] ?? REG_LABEL.eu;

  return (
    <div className="px-6 pb-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-slate-100 font-semibold text-lg flex items-center gap-2">
            <Users size={18} className="text-cyan-400" />
            Beneficial Ownership (UBO)
          </h1>
          <p className="text-slate-400 text-sm mt-0.5">
            {regLabel} — verify all owners with ≥{threshold}% control before case closure
          </p>
        </div>
        {canAdd && (
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
              {pendingCount} beneficial owner record{pendingCount > 1 ? "s" : ""} require verification before this case can move to review or be closed ({regLabel}).
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
                    {rec.discrepancy_note && rec.status !== "discrepant" && (
                      <p className="text-slate-400 text-xs mt-1.5 italic">
                        Source: {rec.discrepancy_note}
                      </p>
                    )}
                    {rec.discrepancy_note && rec.status === "discrepant" && (
                      <p className="text-red-400 text-xs mt-1.5 flex items-start gap-1">
                        <AlertTriangle size={11} className="mt-0.5 flex-shrink-0" />
                        {rec.discrepancy_note}
                      </p>
                    )}
                    <p className="text-slate-600 text-xs mt-1">
                      Added {new Date(rec.created_at).toLocaleDateString()}
                      {rec.verified_at && <> · Verified {new Date(rec.verified_at).toLocaleDateString()}</>}
                    </p>
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
                    {canVerify && rec.status !== "verified" && (() => {
                      const isSelf = rec.created_by && currentUser && rec.created_by === currentUser.id;
                      return (
                        <button
                          onClick={() => {
                            if (isSelf && !confirm(
                              "You created this record. Independent verification is recommended for audit integrity.\n\nVerify anyway?"
                            )) return;
                            verifyMutation.mutate(rec.id);
                          }}
                          disabled={verifyMutation.isPending}
                          className={`px-3 py-1.5 text-xs border rounded-lg transition-colors flex items-center gap-1 ${
                            isSelf
                              ? "text-amber-400 border-amber-500/40 hover:bg-amber-500/10"
                              : "text-green-400 border-green-500/30 hover:bg-green-500/10"
                          }`}
                          title={isSelf ? "You added this record — consider asking a colleague to verify" : undefined}
                        >
                          {isSelf && <AlertTriangle size={10} />}
                          Verify
                        </button>
                      );
                    })()}
                    {canVerify && rec.status !== "discrepant" && (
                      <button
                        onClick={() => setDiscrepancyTarget(rec)}
                        className="px-3 py-1.5 text-xs text-red-400 border border-red-500/30 rounded-lg hover:bg-red-500/10 transition-colors"
                      >
                        Discrepant
                      </button>
                    )}
                    {canVerify && rec.status !== "n/a" && (
                      <button
                        onClick={() => {
                          if (confirm("Mark as N/A? Use this only when the subject is an individual, not a legal entity — UBO rules do not apply.")) {
                            naMutation.mutate(rec.id);
                          }
                        }}
                        className="px-3 py-1.5 text-xs text-slate-400 border border-slate-600/50 rounded-lg hover:bg-slate-700/40 transition-colors"
                        title="Mark N/A — subject is an individual, not a legal entity"
                      >
                        N/A
                      </button>
                    )}
                    {canVerify && (
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

      {showAdd && <AddUBOModal caseId={caseId} jurisdiction={jurisdiction} threshold={threshold} onClose={() => setShowAdd(false)} onCreated={() => {}} />}
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
