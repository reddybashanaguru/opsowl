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
  cases: "Cases",
  documents: "Documents",
  audit_logs: "Audit Logs",
  notifications: "Notifications",
  sessions: "Sessions",
};

const RESOURCE_DESC: Record<string, string> = {
  cases: "Closed/archived investigation cases",
  documents: "Uploaded evidence and attachments",
  audit_logs: "User action audit trail",
  notifications: "In-app notification records",
  sessions: "Expired user session tokens",
};

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

function PolicyStatusDot({ enabled }: { enabled: boolean }) {
  return (
    <span className={`inline-flex items-center gap-1.5 text-xs font-medium px-2 py-0.5 rounded-full ${
      enabled
        ? "bg-emerald-500/15 text-emerald-400 border border-emerald-500/30"
        : "bg-slate-700/50 text-slate-500 border border-slate-600/50"
    }`}>
      <span className={`w-1.5 h-1.5 rounded-full ${enabled ? "bg-emerald-400" : "bg-slate-600"}`} />
      {enabled ? "Active" : "Inactive"}
    </span>
  );
}

function PolicyRow({ policy, onToggle }: { policy: RetentionPolicy; onToggle: (rt: string, enabled: boolean) => void }) {
  const label = RESOURCE_LABELS[policy.resource_type] ?? policy.resource_type;
  const desc = RESOURCE_DESC[policy.resource_type] ?? "";

  return (
    <tr className="border-b border-slate-700/40 hover:bg-slate-700/15 transition-colors">
      <td className="px-6 py-4">
        <div className="text-slate-200 font-medium text-sm">{label}</div>
        <div className="text-slate-500 text-xs mt-0.5">{desc}</div>
        <code className="text-xs text-slate-600 font-mono">{policy.resource_type}</code>
      </td>
      <td className="px-6 py-4">
        <div className="flex items-center gap-2">
          <span className="text-slate-100 font-semibold tabular-nums text-sm">{policy.retention_days}</span>
          <span className="text-slate-500 text-xs">days</span>
        </div>
        <div className="text-slate-600 text-xs mt-0.5">
          {Math.floor(policy.retention_days / 30)}mo {policy.retention_days % 30 > 0 ? `${policy.retention_days % 30}d` : ""}
        </div>
      </td>
      <td className="px-6 py-4">
        <PolicyStatusDot enabled={policy.purge_enabled} />
      </td>
      <td className="px-6 py-4 text-xs tabular-nums">
        {policy.last_purged_at
          ? <span className="text-slate-400">{timeAgo(policy.last_purged_at)}</span>
          : <span className="text-slate-600 italic">Never run</span>
        }
      </td>
      <td className="px-6 py-4">
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
      </td>
    </tr>
  );
}

function RetentionContent() {
  const qc = useQueryClient();
  const [toggleError, setToggleError] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["retention-policies"],
    queryFn: () => api.get<{ policies: RetentionPolicy[] }>("/api/v1/retention/policies"),
    select: (d) => d.policies ?? [],
  });

  const toggleMutation = useMutation({
    mutationFn: ({ resource_type, purge_enabled }: { resource_type: string; purge_enabled: boolean }) =>
      api.put(`/api/v1/retention/policies/${resource_type}`, { purge_enabled }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["retention-policies"] });
      setToggleError(null);
    },
    onError: (e) => {
      setToggleError(e instanceof APIError ? e.message : "Failed to update policy");
    },
  });

  const policies = data ?? [];
  const activeCount = policies.filter((p) => p.purge_enabled).length;

  return (
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-100 mb-1">Data Retention</h1>
        <p className="text-slate-400 text-sm">
          DPDP Act 2023 compliant data purge policies. Automatically deletes records older than the
          configured retention window for each enabled resource type.
        </p>
      </div>

      {/* Stats bar */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        {[
          { label: "Total Policies", value: isLoading ? "—" : policies.length },
          { label: "Active Purges", value: isLoading ? "—" : activeCount, color: activeCount > 0 ? "text-emerald-400" : "text-slate-500" },
          { label: "Paused", value: isLoading ? "—" : policies.length - activeCount },
        ].map((s) => (
          <div key={s.label} className="bg-[#1e293b] border border-slate-700 rounded-xl px-5 py-4">
            <div className={`text-2xl font-bold ${s.color ?? "text-slate-100"} tabular-nums`}>{s.value}</div>
            <div className="text-slate-500 text-xs mt-0.5">{s.label}</div>
          </div>
        ))}
      </div>

      {toggleError && (
        <div className="mb-4 px-4 py-3 bg-red-500/10 border border-red-500/30 rounded-lg text-red-400 text-sm">
          {toggleError}
        </div>
      )}

      {/* Table */}
      <div className="bg-[#1e293b] border border-slate-700 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-700 text-slate-500 text-xs uppercase tracking-wide">
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
                <tr key={i} aria-label="animate-pulse" className="border-b border-slate-700/40">
                  {[200, 100, 80, 100, 60].map((w, j) => (
                    <td key={j} className="px-6 py-4">
                      <div className="h-4 bg-slate-700/60 animate-pulse rounded" style={{ width: w }} />
                    </td>
                  ))}
                </tr>
              ))
              : policies.length === 0
              ? (
                <tr aria-label="animate-pulse">
                  <td colSpan={5} className="px-6 py-16 text-center">
                    <p className="text-slate-500 text-sm">No retention policies configured.</p>
                    <p className="text-slate-600 text-xs mt-1">
                      Run database migration <code className="font-mono">00021_data_retention</code> to seed default policies.
                    </p>
                  </td>
                </tr>
              )
              : policies.map((p) => (
                <PolicyRow
                  key={p.resource_type}
                  policy={p}
                  onToggle={(rt, enabled) => toggleMutation.mutate({ resource_type: rt, purge_enabled: enabled })}
                />
              ))
            }
          </tbody>
        </table>
      </div>

      <div className="mt-4 p-4 bg-slate-800/50 border border-slate-700/50 rounded-lg">
        <p className="text-slate-400 text-xs">
          <span className="text-slate-300 font-medium">Scheduler: </span>
          Runs once at startup then every 24 hours. Records with{" "}
          <code className="font-mono text-cyan-400/80">created_at &lt; NOW() - retention_days</code>{" "}
          are hard-deleted. Audit logs of the purge itself are retained indefinitely.
        </p>
      </div>
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
