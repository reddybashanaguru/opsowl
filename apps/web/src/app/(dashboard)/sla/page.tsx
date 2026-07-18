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

const PRIORITY_COLORS: Record<string, { badge: string; accent: string; ring: string }> = {
  critical: { badge: "text-red-400 bg-red-500/10 border-red-500/30", accent: "border-red-500/40", ring: "ring-red-500/10" },
  high:     { badge: "text-orange-400 bg-orange-500/10 border-orange-500/30", accent: "border-orange-500/30", ring: "ring-orange-500/10" },
  medium:   { badge: "text-amber-400 bg-amber-500/10 border-amber-500/30", accent: "border-amber-500/30", ring: "ring-amber-500/10" },
  low:      { badge: "text-emerald-400 bg-emerald-500/10 border-emerald-500/30", accent: "border-emerald-500/20", ring: "ring-emerald-500/10" },
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
    <div className="relative h-6 bg-slate-800 rounded-lg overflow-hidden">
      {/* On-track zone */}
      <div
        className="absolute left-0 top-0 h-full bg-cyan-500/20"
        style={{ width: `${warnPct}%` }}
      >
        {warnPct > 18 && (
          <span className="absolute inset-0 flex items-center justify-center text-[9px] text-cyan-400/70 font-semibold tracking-widest uppercase">
            On Track
          </span>
        )}
      </div>
      {/* Warning zone */}
      <div
        className="absolute top-0 h-full bg-amber-400/20"
        style={{ left: `${warnPct}%`, right: "4px" }}
      >
        {(100 - warnPct) > 12 && (
          <span className="absolute inset-0 flex items-center justify-center text-[9px] text-amber-400/70 font-semibold tracking-widest uppercase">
            Warning
          </span>
        )}
      </div>
      {/* Zone divider */}
      <div
        className="absolute top-0 h-full w-px bg-slate-600/80"
        style={{ left: `${warnPct}%` }}
      />
      {/* Breach strip at right edge */}
      <div className="absolute right-0 top-0 h-full w-1 bg-red-500" />
    </div>
  );
}

export default function SLAPage() {
  const qc = useQueryClient();
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
    (a, b) => PRIORITY_ORDER.indexOf(a.priority) - PRIORITY_ORDER.indexOf(b.priority),
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
        Cases enter the warning zone at the amber threshold and are flagged as breached in red across the dashboard.
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
            const colors = PRIORITY_COLORS[policy.priority] ?? PRIORITY_COLORS.low;
            const isEditing = editingPriority === policy.priority;
            const warnTime = Math.round(policy.breach_hours * policy.warn_pct / 100);
            const dangerPct = 100 - policy.warn_pct;

            return (
              <div
                key={policy.id}
                className={`bg-[#131f30] border ${colors.accent} rounded-xl ring-1 ${colors.ring} p-5`}
              >
                {/* Card header */}
                <div className="flex items-start justify-between mb-5">
                  <div className="flex items-center gap-3">
                    <span className={`text-xs font-semibold px-2.5 py-1 rounded-lg border capitalize ${colors.badge}`}>
                      {policy.priority}
                    </span>
                    <div>
                      <div className="text-white text-sm font-medium">
                        Total SLA window:{" "}
                        <span className="text-white font-bold">{hoursToLabel(policy.breach_hours)}</span>
                      </div>
                      <div className="text-slate-500 text-xs mt-0.5">
                        Warning triggers at {policy.warn_pct}% · {dangerPct}% danger window before breach
                      </div>
                    </div>
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

                {/* Three zone chips */}
                <div className="grid grid-cols-3 gap-3 mb-4">
                  <div className="bg-cyan-500/8 border border-cyan-500/20 rounded-lg px-3 py-2.5">
                    <div className="text-cyan-400 text-[10px] font-semibold uppercase tracking-wider mb-1.5">
                      ✓ On Track
                    </div>
                    <div className="text-white text-sm font-mono font-medium">
                      0 → {hoursToLabel(warnTime)}
                    </div>
                    <div className="text-slate-500 text-[10px] mt-0.5">
                      {policy.warn_pct}% of SLA window
                    </div>
                  </div>
                  <div className="bg-amber-500/8 border border-amber-500/20 rounded-lg px-3 py-2.5">
                    <div className="text-amber-400 text-[10px] font-semibold uppercase tracking-wider mb-1.5">
                      ⚠ Warning
                    </div>
                    <div className="text-white text-sm font-mono font-medium">
                      {hoursToLabel(warnTime)} → {hoursToLabel(policy.breach_hours)}
                    </div>
                    <div className="text-slate-500 text-[10px] mt-0.5">
                      {dangerPct}% of SLA window
                    </div>
                  </div>
                  <div className="bg-red-500/8 border border-red-500/20 rounded-lg px-3 py-2.5">
                    <div className="text-red-400 text-[10px] font-semibold uppercase tracking-wider mb-1.5">
                      🔴 Breach
                    </div>
                    <div className="text-white text-sm font-mono font-medium">
                      After {hoursToLabel(policy.breach_hours)}
                    </div>
                    <div className="text-slate-500 text-[10px] mt-0.5">
                      Regulatory deadline
                    </div>
                  </div>
                </div>

                {/* Visual zone bar */}
                <SLAProgressBar warnPct={policy.warn_pct} />

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
                        {editForm.warning_hours > 0 && (
                          <p className="text-amber-400/70 text-[10px] mt-1 font-mono">
                            = {hoursToLabel(editForm.warning_hours)}
                          </p>
                        )}
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
                        {editForm.breach_hours > 0 && (
                          <p className="text-red-400/70 text-[10px] mt-1 font-mono">
                            = {hoursToLabel(editForm.breach_hours)}
                          </p>
                        )}
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
                        {editForm.breach_hours > 0 && editForm.warn_pct > 0 && (
                          <p className="text-slate-500 text-[10px] mt-1 font-mono">
                            warn at {hoursToLabel(Math.round(editForm.breach_hours * editForm.warn_pct / 100))}
                          </p>
                        )}
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
