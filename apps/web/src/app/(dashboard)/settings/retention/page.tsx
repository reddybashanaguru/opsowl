"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { RBACGate } from "@/components/auth/RBACGate";

interface RetentionPolicy {
  resource_type: string;
  retention_days: number;
  purge_enabled: boolean;
  last_purged_at: string | null;
}

const RESOURCE_LABELS: Record<string, string> = {
  cases:         "Cases",
  documents:     "Documents",
  audit_logs:    "Audit Logs",
  notifications: "Notifications",
  sessions:      "Sessions",
  ai_logs:       "AI Logs",
  str_filings:   "STR Filings",
};

const RESOURCE_DESC: Record<string, string> = {
  cases:         "Closed/archived investigation cases",
  documents:     "Uploaded evidence and attachments",
  audit_logs:    "User action audit trail",
  notifications: "In-app notification records",
  sessions:      "Expired user session tokens",
  ai_logs:       "AI decision and audit records",
  str_filings:   "Suspicious Transaction Report filings",
};

// Resource types that are PMLA-protected (cannot go below 3650 days)
const PMLA_PROTECTED = new Set(["audit_logs", "cases", "documents", "ai_logs", "str_filings"]);

function timeAgo(iso: string) {
  const diff = Date.now() - new Date(iso).getTime();
  const m = Math.floor(diff / 60_000);
  if (m < 1) return "just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 30) return `${d}d ago`;
  return new Date(iso).toLocaleDateString(undefined, { day: "2-digit", month: "short", year: "numeric" });
}

function daysLabel(days: number) {
  const years = Math.floor(days / 365);
  const rem = days % 365;
  const months = Math.floor(rem / 30);
  if (years > 0 && months > 0) return `${years}y ${months}mo`;
  if (years > 0) return `${years} year${years > 1 ? "s" : ""}`;
  if (months > 0) return `${months} month${months > 1 ? "s" : ""}`;
  return `${days} days`;
}

function PolicyRow({
  policy,
  onEdit,
  onToggle,
}: {
  policy: RetentionPolicy;
  onEdit: (p: RetentionPolicy) => void;
  onToggle: (rt: string, enabled: boolean) => void;
}) {
  const label = RESOURCE_LABELS[policy.resource_type] ?? policy.resource_type;
  const desc = RESOURCE_DESC[policy.resource_type] ?? "";
  const isProtected = PMLA_PROTECTED.has(policy.resource_type);

  return (
    <tr className="border-b border-slate-700/40 hover:bg-slate-700/10 transition-colors">
      <td className="px-6 py-4">
        <div className="flex items-center gap-2">
          <span className="text-slate-200 font-medium text-sm">{label}</span>
          {isProtected && (
            <span className="text-[10px] font-medium px-1.5 py-0.5 rounded bg-amber-500/10 border border-amber-500/30 text-amber-400">
              PMLA
            </span>
          )}
        </div>
        {desc && <div className="text-slate-500 text-xs mt-0.5">{desc}</div>}
        <code className="text-xs text-slate-600 font-mono">{policy.resource_type}</code>
      </td>
      <td className="px-6 py-4">
        <div className="text-slate-100 font-semibold tabular-nums text-sm">{policy.retention_days} days</div>
        <div className="text-slate-500 text-xs mt-0.5">{daysLabel(policy.retention_days)}</div>
      </td>
      <td className="px-6 py-4">
        <span className={`inline-flex items-center gap-1.5 text-xs font-medium px-2 py-0.5 rounded-full border ${
          policy.purge_enabled
            ? "bg-emerald-500/15 text-emerald-400 border-emerald-500/30"
            : "bg-slate-700/50 text-slate-500 border-slate-600/50"
        }`}>
          <span className={`w-1.5 h-1.5 rounded-full ${policy.purge_enabled ? "bg-emerald-400" : "bg-slate-600"}`} />
          {policy.purge_enabled ? "Active" : "Paused"}
        </span>
      </td>
      <td className="px-6 py-4 text-xs tabular-nums">
        {policy.last_purged_at
          ? <span className="text-slate-400">{timeAgo(policy.last_purged_at)}</span>
          : <span className="text-slate-600 italic">Never run</span>}
      </td>
      <td className="px-6 py-4">
        <div className="flex items-center gap-2">
          <button
            onClick={() => onEdit(policy)}
            className="text-xs px-3 py-1.5 rounded border border-slate-600 text-slate-300 hover:border-slate-500 hover:text-white transition-colors"
          >
            Edit
          </button>
          <button
            onClick={() => onToggle(policy.resource_type, !policy.purge_enabled)}
            className={`text-xs px-3 py-1.5 rounded border transition-colors ${
              policy.purge_enabled
                ? "text-amber-400 border-amber-500/30 hover:bg-amber-500/10"
                : "text-emerald-400 border-emerald-500/30 hover:bg-emerald-500/10"
            }`}
          >
            {policy.purge_enabled ? "Pause" : "Enable"}
          </button>
        </div>
      </td>
    </tr>
  );
}

// ── Edit modal ────────────────────────────────────────────────────────────────

function EditModal({
  policy,
  onClose,
  onSave,
  saving,
  error,
}: {
  policy: RetentionPolicy;
  onClose: () => void;
  onSave: (days: number, enabled: boolean) => void;
  saving: boolean;
  error: string | null;
}) {
  const isProtected = PMLA_PROTECTED.has(policy.resource_type);
  const [days, setDays] = useState(String(policy.retention_days));
  const [enabled, setEnabled] = useState(policy.purge_enabled);
  const label = RESOURCE_LABELS[policy.resource_type] ?? policy.resource_type;
  const daysNum = parseInt(days, 10);
  const floorViolation = isProtected && !isNaN(daysNum) && daysNum < 3650;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm px-4">
      <div className="bg-[#0d1f35] border border-slate-700 rounded-2xl w-full max-w-md shadow-2xl">
        <div className="flex items-center justify-between p-5 border-b border-slate-700/60">
          <div>
            <h2 className="text-white font-semibold text-base">Edit Retention Policy</h2>
            <p className="text-slate-400 text-xs mt-0.5">{label} · <code className="font-mono">{policy.resource_type}</code></p>
          </div>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300 p-1 transition-colors">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="p-5 space-y-5">
          {isProtected && (
            <div className="px-3 py-2.5 bg-amber-500/10 border border-amber-500/30 rounded-lg text-xs text-amber-400">
              <span className="font-semibold">PMLA Section 12 protected.</span> Minimum 3650 days (10 years). Cannot be set lower.
            </div>
          )}

          <div>
            <label className="text-slate-400 text-xs font-medium block mb-1.5">
              Retention Window (days)
            </label>
            <input
              type="number"
              value={days}
              min={isProtected ? 3650 : 1}
              onChange={(e) => setDays(e.target.value)}
              className={`w-full bg-[#080e1a] border rounded-lg px-3 py-2.5 text-sm text-white focus:outline-none ${
                floorViolation ? "border-red-500 focus:border-red-400" : "border-slate-600 focus:border-cyan-500"
              }`}
            />
            {!isNaN(daysNum) && daysNum > 0 && (
              <p className="text-slate-500 text-xs mt-1">≈ {daysLabel(daysNum)}</p>
            )}
            {floorViolation && (
              <p className="text-red-400 text-xs mt-1">Must be ≥ 3650 days for PMLA-protected types</p>
            )}
          </div>

          <div>
            <label className="text-slate-400 text-xs font-medium block mb-2">Auto-Purge</label>
            <div className="flex gap-3">
              {[
                { val: false, label: "Paused", desc: "Records are kept, no deletion runs" },
                { val: true,  label: "Active",  desc: "Records older than the window are deleted daily" },
              ].map((opt) => (
                <button
                  key={String(opt.val)}
                  onClick={() => setEnabled(opt.val)}
                  className={`flex-1 text-left px-3 py-2.5 rounded-lg border text-xs transition-colors ${
                    enabled === opt.val
                      ? opt.val
                        ? "border-emerald-500/50 bg-emerald-500/10 text-emerald-400"
                        : "border-slate-500 bg-slate-700/40 text-slate-300"
                      : "border-slate-700 text-slate-500 hover:border-slate-600 hover:text-slate-400"
                  }`}
                >
                  <div className="font-semibold">{opt.label}</div>
                  <div className="text-[10px] mt-0.5 opacity-80">{opt.desc}</div>
                </button>
              ))}
            </div>
          </div>

          {error && (
            <p className="text-red-400 text-xs px-3 py-2 bg-red-500/10 border border-red-500/20 rounded-lg">{error}</p>
          )}
        </div>

        <div className="flex gap-3 p-5 pt-0">
          <button
            onClick={onClose}
            className="flex-1 px-4 py-2 rounded-lg border border-slate-600 text-slate-300 text-sm hover:border-slate-500 transition-colors"
          >
            Cancel
          </button>
          <button
            disabled={saving || isNaN(daysNum) || daysNum <= 0 || floorViolation}
            onClick={() => onSave(daysNum, enabled)}
            className="flex-1 px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium disabled:opacity-50 disabled:pointer-events-none transition-colors"
          >
            {saving ? "Saving…" : "Save Changes"}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Add policy modal ──────────────────────────────────────────────────────────

function AddModal({
  onClose,
  onSave,
  saving,
  error,
}: {
  onClose: () => void;
  onSave: (rt: string, days: number, enabled: boolean) => void;
  saving: boolean;
  error: string | null;
}) {
  const [rt, setRt] = useState("");
  const [days, setDays] = useState("365");
  const [enabled, setEnabled] = useState(false);
  const daysNum = parseInt(days, 10);

  const handleSlug = (v: string) => setRt(v.toLowerCase().replace(/[^a-z0-9_]/g, "_"));

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm px-4">
      <div className="bg-[#0d1f35] border border-slate-700 rounded-2xl w-full max-w-md shadow-2xl">
        <div className="flex items-center justify-between p-5 border-b border-slate-700/60">
          <div>
            <h2 className="text-white font-semibold text-base">Add Retention Policy</h2>
            <p className="text-slate-400 text-xs mt-0.5">Define a custom data type and its retention window</p>
          </div>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300 p-1 transition-colors">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="p-5 space-y-4">
          <div>
            <label className="text-slate-400 text-xs font-medium block mb-1.5">Resource Type <span className="text-red-400">*</span></label>
            <input
              value={rt}
              onChange={(e) => handleSlug(e.target.value)}
              placeholder="e.g. alert_logs, comms_logs"
              className="w-full bg-[#080e1a] border border-slate-600 text-white placeholder:text-slate-600 rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:border-cyan-500"
            />
            <p className="text-slate-600 text-xs mt-1">Lowercase, underscores only. Must match a table name or data category.</p>
          </div>

          <div>
            <label className="text-slate-400 text-xs font-medium block mb-1.5">Retention Window (days) <span className="text-red-400">*</span></label>
            <input
              type="number"
              value={days}
              min={1}
              onChange={(e) => setDays(e.target.value)}
              className="w-full bg-[#080e1a] border border-slate-600 text-white rounded-lg px-3 py-2.5 text-sm focus:outline-none focus:border-cyan-500"
            />
            {!isNaN(daysNum) && daysNum > 0 && (
              <p className="text-slate-500 text-xs mt-1">≈ {daysLabel(daysNum)}</p>
            )}
          </div>

          <div>
            <label className="text-slate-400 text-xs font-medium block mb-2">Auto-Purge</label>
            <div className="flex gap-3">
              {([false, true] as const).map((val) => (
                <button
                  key={String(val)}
                  onClick={() => setEnabled(val)}
                  className={`flex-1 px-3 py-2 rounded-lg border text-xs font-medium transition-colors ${
                    enabled === val
                      ? val
                        ? "border-emerald-500/50 bg-emerald-500/10 text-emerald-400"
                        : "border-slate-500 bg-slate-700/40 text-slate-300"
                      : "border-slate-700 text-slate-500 hover:border-slate-600"
                  }`}
                >
                  {val ? "Active (delete after window)" : "Paused (keep indefinitely)"}
                </button>
              ))}
            </div>
          </div>

          {error && (
            <p className="text-red-400 text-xs px-3 py-2 bg-red-500/10 border border-red-500/20 rounded-lg">{error}</p>
          )}
        </div>

        <div className="flex gap-3 p-5 pt-0">
          <button onClick={onClose} className="flex-1 px-4 py-2 rounded-lg border border-slate-600 text-slate-300 text-sm hover:border-slate-500 transition-colors">
            Cancel
          </button>
          <button
            disabled={saving || !rt || isNaN(daysNum) || daysNum <= 0}
            onClick={() => onSave(rt, daysNum, enabled)}
            className="flex-1 px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium disabled:opacity-50 disabled:pointer-events-none transition-colors"
          >
            {saving ? "Adding…" : "Add Policy"}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Main component ────────────────────────────────────────────────────────────

function RetentionContent() {
  const qc = useQueryClient();
  const [editPolicy, setEditPolicy] = useState<RetentionPolicy | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const [mutationError, setMutationError] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["retention-policies"],
    queryFn: () => api.get<{ policies: RetentionPolicy[] }>("/api/v1/retention/policies"),
    select: (d) => d.policies ?? [],
  });

  const updateMutation = useMutation({
    mutationFn: ({ resource_type, retention_days, purge_enabled }: { resource_type: string; retention_days?: number; purge_enabled?: boolean }) =>
      api.put<{ policies: RetentionPolicy[] }>(`/api/v1/retention/policies/${resource_type}`, { retention_days, purge_enabled }),
    onSuccess: (result) => {
      qc.setQueryData(["retention-policies"], result);
      setEditPolicy(null);
      setMutationError(null);
    },
    onError: (e) => setMutationError(e instanceof APIError ? e.message : "Failed to update policy"),
  });

  const addMutation = useMutation({
    mutationFn: (body: { resource_type: string; retention_days: number; purge_enabled: boolean }) =>
      api.post<{ policies: RetentionPolicy[] }>("/api/v1/retention/policies", body),
    onSuccess: (result) => {
      qc.setQueryData(["retention-policies"], result);
      setShowAdd(false);
      setMutationError(null);
    },
    onError: (e) => setMutationError(e instanceof APIError ? e.message : "Failed to add policy"),
  });

  const policies = data ?? [];
  const activeCount = policies.filter((p) => p.purge_enabled).length;

  return (
    <div className="p-8">
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 mb-1">Data Retention</h1>
          <p className="text-slate-400 text-sm">
            DPDP Act 2023 compliant data purge policies. Configure how long each data type is kept before auto-deletion.
          </p>
        </div>
        <button
          onClick={() => { setShowAdd(true); setMutationError(null); }}
          className="flex-shrink-0 flex items-center gap-1.5 bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <span className="text-base leading-none">+</span> Add Policy
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        {[
          { label: "Total Policies", value: isLoading ? "—" : policies.length },
          { label: "Active Purges", value: isLoading ? "—" : activeCount, color: activeCount > 0 ? "text-emerald-400" : "text-slate-500" },
          { label: "Paused", value: isLoading ? "—" : policies.length - activeCount },
        ].map((s) => (
          <div key={s.label} className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] px-5 py-4">
            <div className={`text-2xl font-bold ${s.color ?? "text-slate-100"} tabular-nums`}>{s.value}</div>
            <div className="text-slate-500 text-xs mt-0.5">{s.label}</div>
          </div>
        ))}
      </div>

      {/* Table */}
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-700 text-slate-500 text-xs uppercase tracking-wider">
              <th className="text-left px-6 py-3 font-medium">Resource</th>
              <th className="text-left px-6 py-3 font-medium">Retention Window</th>
              <th className="text-left px-6 py-3 font-medium">Status</th>
              <th className="text-left px-6 py-3 font-medium">Last Purged</th>
              <th className="px-6 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading
              ? [...Array(5)].map((_, i) => (
                <tr key={i} className="border-b border-slate-700/40">
                  {[200, 100, 80, 100, 80].map((w, j) => (
                    <td key={j} className="px-6 py-4">
                      <div className="h-4 bg-slate-700/60 animate-pulse rounded" style={{ width: w }} />
                    </td>
                  ))}
                </tr>
              ))
              : policies.length === 0
              ? (
                <tr>
                  <td colSpan={5} className="px-6 py-16 text-center">
                    <p className="text-slate-500 text-sm">No retention policies configured.</p>
                    <p className="text-slate-600 text-xs mt-1">Click "Add Policy" to create the first one.</p>
                  </td>
                </tr>
              )
              : policies.map((p) => (
                <PolicyRow
                  key={p.resource_type}
                  policy={p}
                  onEdit={(pol) => { setEditPolicy(pol); setMutationError(null); }}
                  onToggle={(rt, enabled) => updateMutation.mutate({ resource_type: rt, purge_enabled: enabled })}
                />
              ))}
          </tbody>
        </table>
      </div>

      <div className="mt-4 p-4 bg-slate-800/50 border border-slate-700/50 rounded-lg">
        <p className="text-slate-400 text-xs">
          <span className="text-slate-300 font-medium">Scheduler: </span>
          Runs once at startup then every 24 hours. Records with{" "}
          <code className="font-mono text-cyan-400/80">created_at &lt; NOW() - retention_days</code>{" "}
          are hard-deleted. Audit logs of the purge itself are retained indefinitely.{" "}
          <span className="text-amber-400/80 font-medium">PMLA-tagged</span>{" "}
          types have a minimum 3650-day floor enforced at the database level.
        </p>
      </div>

      {editPolicy && (
        <EditModal
          policy={editPolicy}
          onClose={() => { setEditPolicy(null); setMutationError(null); }}
          onSave={(days, enabled) => updateMutation.mutate({ resource_type: editPolicy.resource_type, retention_days: days, purge_enabled: enabled })}
          saving={updateMutation.isPending}
          error={mutationError}
        />
      )}

      {showAdd && (
        <AddModal
          onClose={() => { setShowAdd(false); setMutationError(null); }}
          onSave={(rt, days, enabled) => addMutation.mutate({ resource_type: rt, retention_days: days, purge_enabled: enabled })}
          saving={addMutation.isPending}
          error={mutationError}
        />
      )}
    </div>
  );
}

export default function RetentionPage() {
  return (
    <RBACGate allowedRoles={["admin", "super_admin"]} fallback={
      <div className="p-8 text-slate-400 text-sm">Admin access required to view retention policies.</div>
    }>
      <RetentionContent />
    </RBACGate>
  );
}
