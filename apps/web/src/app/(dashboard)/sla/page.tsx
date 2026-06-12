"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";

interface SLAPolicy {
  id: string;
  priority: string;
  warning_hours: number;
  breach_hours: number;
  warn_pct: number;
  created_at: string;
  updated_at: string;
}

const PRIORITY_COLORS: Record<string, string> = {
  critical: "text-red-400 bg-red-500/10 border-red-500/30",
  high:     "text-orange-400 bg-orange-500/10 border-orange-500/30",
  medium:   "text-amber-400 bg-amber-500/10 border-amber-500/30",
  low:      "text-emerald-400 bg-emerald-500/10 border-emerald-500/30",
};

const PRIORITY_ORDER = ["critical", "high", "medium", "low"];

function hoursToLabel(h: number): string {
  if (h < 24) return `${h}h`;
  const d = Math.floor(h / 24);
  const rem = h % 24;
  return rem > 0 ? `${d}d ${rem}h` : `${d}d`;
}

function SLAProgressBar({ warnPct }: { warnPct: number }) {
  return (
    <div className="relative h-2 bg-slate-700 rounded-full overflow-hidden mt-1">
      <div
        className="absolute left-0 top-0 h-full bg-amber-400 rounded-full transition-all"
        style={{ width: `${warnPct}%` }}
      />
      <div
        className="absolute h-full w-0.5 bg-red-400"
        style={{ left: "100%" }}
      />
    </div>
  );
}

export default function SLAPage() {
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const hasRole = useAuthStore((s) => s.hasRole);
  const canEdit = hasRole("admin");

  const [editingPriority, setEditingPriority] = useState<string | null>(null);
  const [editForm, setEditForm] = useState({ warning_hours: 0, breach_hours: 0, warn_pct: 80 });
  const [saveError, setSaveError] = useState("");

  const { data: policies, isLoading, isError, error, refetch } = useQuery<SLAPolicy[]>({
    queryKey: ["sla-policies"],
    queryFn: () => api.get<SLAPolicy[]>("/api/v1/sla/policies"),
  });

  const upsertMutation = useMutation({
    mutationFn: (body: { priority: string; warning_hours: number; breach_hours: number; warn_pct: number }) =>
      api.put("/api/v1/sla/policies", body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sla-policies"] });
      setEditingPriority(null);
      setSaveError("");
    },
    onError: (e: Error) => setSaveError(e.message),
  });

  const sortedPolicies = [...(policies ?? [])].sort(
    (a, b) => PRIORITY_ORDER.indexOf(a.priority) - PRIORITY_ORDER.indexOf(b.priority)
  );

  const startEdit = (p: SLAPolicy) => {
    setEditingPriority(p.priority);
    setEditForm({ warning_hours: p.warning_hours, breach_hours: p.breach_hours, warn_pct: p.warn_pct });
    setSaveError("");
  };

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-white text-2xl font-bold">SLA Policy Engine</h1>
        <p className="text-slate-400 text-sm mt-1">
          Configure warning and breach thresholds per case priority level.
        </p>
      </div>

      {/* Info banner */}
      <div className="bg-blue-500/10 border border-blue-500/30 text-blue-300 rounded-xl p-4 mb-6 text-sm">
        <strong>How SLA works:</strong> When a case is created, the breach deadline is set based on its priority.
        Cases exceeding the warning threshold appear in amber; breached cases are flagged in red across the dashboard.
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center h-32">
          <div className="w-6 h-6 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : isError ? (
        <div className="bg-red-500/10 border border-red-500/30 rounded-xl p-6 text-center">
          <p className="text-red-400 text-sm font-medium">{error instanceof Error ? error.message : "Failed to load SLA policies"}</p>
          <button onClick={() => refetch()} className="mt-3 text-cyan-400 hover:text-cyan-300 text-sm">Try again →</button>
        </div>
      ) : (
        <div className="space-y-4">
          {sortedPolicies.map((policy) => {
            const colorClass = PRIORITY_COLORS[policy.priority] ?? "text-slate-400 bg-slate-700 border-slate-600";
            const isEditing = editingPriority === policy.priority;

            return (
              <div key={policy.id} className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <span className={`text-xs font-semibold px-2.5 py-1 rounded-lg border capitalize ${colorClass}`}>
                      {policy.priority}
                    </span>
                    <span className="text-slate-300 text-sm">
                      Breach at <strong className="text-white">{hoursToLabel(policy.breach_hours)}</strong>
                      {" · "}Warn at <strong className="text-amber-400">{hoursToLabel(policy.warning_hours)}</strong>
                    </span>
                  </div>
                  {canEdit && !isEditing && (
                    <button
                      onClick={() => startEdit(policy)}
                      className="text-slate-400 hover:text-slate-200 text-xs border border-slate-600 hover:border-slate-500 px-3 py-1.5 rounded-lg transition-colors"
                    >
                      Edit
                    </button>
                  )}
                </div>

                {/* Warning progress bar */}
                <div className="mb-1">
                  <div className="flex justify-between text-xs text-slate-500 mb-1">
                    <span>Warning at {policy.warn_pct}% of breach time</span>
                    <span>{hoursToLabel(Math.round(policy.breach_hours * policy.warn_pct / 100))} → {hoursToLabel(policy.breach_hours)}</span>
                  </div>
                  <SLAProgressBar warnPct={policy.warn_pct} />
                </div>

                {/* Edit form */}
                {isEditing && (
                  <div className="mt-4 pt-4 border-t border-slate-700">
                    <div className="grid grid-cols-3 gap-4 mb-4">
                      <div>
                        <label className="text-slate-400 text-xs mb-1 block">Warning Hours</label>
                        <input
                          type="number"
                          value={editForm.warning_hours}
                          onChange={(e) => setEditForm({ ...editForm, warning_hours: parseInt(e.target.value) || 0 })}
                          min={1}
                          className="w-full bg-[#080e1a] border border-slate-600 text-white text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
                        />
                      </div>
                      <div>
                        <label className="text-slate-400 text-xs mb-1 block">Breach Hours</label>
                        <input
                          type="number"
                          value={editForm.breach_hours}
                          onChange={(e) => setEditForm({ ...editForm, breach_hours: parseInt(e.target.value) || 0 })}
                          min={1}
                          className="w-full bg-[#080e1a] border border-slate-600 text-white text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
                        />
                      </div>
                      <div>
                        <label className="text-slate-400 text-xs mb-1 block">Warning % of Breach</label>
                        <input
                          type="number"
                          value={editForm.warn_pct}
                          onChange={(e) => setEditForm({ ...editForm, warn_pct: parseInt(e.target.value) || 80 })}
                          min={10}
                          max={95}
                          className="w-full bg-[#080e1a] border border-slate-600 text-white text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
                        />
                      </div>
                    </div>
                    {saveError && <p className="text-red-400 text-xs mb-3">{saveError}</p>}
                    <div className="flex gap-2">
                      <button
                        onClick={() => upsertMutation.mutate({ priority: policy.priority, ...editForm })}
                        disabled={upsertMutation.isPending}
                        className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-2 rounded-lg text-sm disabled:opacity-60"
                      >
                        {upsertMutation.isPending ? "Saving…" : "Save"}
                      </button>
                      <button
                        onClick={() => { setEditingPriority(null); setSaveError(""); }}
                        className="text-slate-400 hover:text-slate-200 px-4 py-2 text-sm"
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {!canEdit && (
        <p className="text-slate-500 text-xs mt-4 text-center">
          Only admins can modify SLA policies.
        </p>
      )}
    </div>
  );
}
