"use client";

import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { ExportButton } from "@/components/analytics/ExportButton";
import { LoadBar } from "@/components/analytics/LoadBar";
import { RBACGate } from "@/components/auth/RBACGate";
import Link from "next/link";

interface AnalystWorkload {
  id: string;
  full_name: string;
  email: string;
  role: string;
  active_cases: number;
  open_tasks: number;
  overdue_tasks: number;
  avg_case_age_hours: string;
}

interface RiskRow {
  id: string;
  case_number: string;
  title: string;
  status: string;
  priority: string;
  sla_breached: boolean;
  risk_score: string | null;
  risk_level: string | null;
  analysed_at: string | null;
}

const ROLE_BADGE: Record<string, string> = {
  analyst: "bg-blue-500/20 text-blue-300",
  manager: "bg-violet-500/20 text-violet-300",
};

const RISK_COLOR: Record<string, string> = {
  critical: "text-red-400",
  high: "text-amber-400",
  medium: "text-blue-400",
  low: "text-green-400",
};

const PRIORITY_COLOR: Record<string, string> = {
  critical: "text-red-400",
  high: "text-amber-400",
  medium: "text-blue-400",
  low: "text-green-400",
};

function Metric({ label, value, color }: { label: string; value: number | string; color?: string }) {
  return (
    <div className="text-center">
      <div className={`text-xl font-bold ${color ?? "text-white"}`}>{value}</div>
      <div className="text-slate-500 text-xs mt-0.5">{label}</div>
    </div>
  );
}

function avgCaseAge(hours: string): { label: string; stale: boolean } {
  const h = parseFloat(hours);
  if (isNaN(h)) return { label: "—", stale: false };
  if (h < 24) return { label: `${Math.round(h)}h`, stale: false };
  const days = Math.round(h / 24);
  return { label: `${days}d`, stale: days > 14 };
}

export default function WorkloadPage() {
  const [lastRefreshed, setLastRefreshed] = useState<Date | null>(null);
  const { data: analysts, isLoading, error } = useQuery<AnalystWorkload[]>({
    queryKey: ["workload"],
    queryFn: () => api.get<AnalystWorkload[]>("/api/v1/analytics/workload"),
    refetchInterval: 30000,
  });
  useEffect(() => { if (analysts) setLastRefreshed(new Date()); }, [analysts]);

  const { data: riskRows, isLoading: riskLoading } = useQuery<RiskRow[]>({
    queryKey: ["risk-leaderboard"],
    queryFn: () => api.get<RiskRow[]>("/api/v1/analytics/risk?limit=20"),
    refetchInterval: 60000,
  });

  const maxCases = Math.max(...(analysts ?? []).map((a) => a.active_cases), 1);
  const maxTasks = Math.max(...(analysts ?? []).map((a) => a.open_tasks), 1);

  const totalActiveCases = (analysts ?? []).reduce((s, a) => s + a.active_cases, 0);
  const totalOpenTasks = (analysts ?? []).reduce((s, a) => s + a.open_tasks, 0);
  const totalOverdue = (analysts ?? []).reduce((s, a) => s + a.overdue_tasks, 0);

  const today = new Date().toISOString().slice(0, 10);

  return (
    <RBACGate allowedRoles={["manager", "admin", "super_admin"]} fallback={
      <div className="p-12 flex flex-col items-center justify-center text-center">
        <div className="text-5xl mb-4">🔒</div>
        <p className="text-slate-200 font-semibold">Access Denied</p>
        <p className="text-slate-500 text-sm mt-2">Manager or higher privileges required to view team workload.</p>
      </div>
    }>
    <div className="p-6 space-y-8">
      {error && (
        <div className="bg-red-500/10 border border-red-500/30 text-red-300 rounded-xl p-4 text-sm">
          Failed to load workload data. Please try again.
        </div>
      )}
      {/* ── Workload section ── */}
      <div>
        <div className="flex items-start justify-between mb-6">
          <div>
            <h1 className="text-white text-2xl font-bold">Team Workload</h1>
            <p className="text-slate-400 text-sm mt-1">
              Live case and task distribution across investigators
              {lastRefreshed && (
                <span className="text-slate-600 ml-1">
                  · as of {lastRefreshed.toLocaleTimeString("en-IN", { hour: "2-digit", minute: "2-digit" })}
                </span>
              )}
            </p>
          </div>
          <ExportButton
            path="/api/v1/analytics/workload/export"
            filename={`workload-${today}.csv`}
            label="Export Workload"
          />
        </div>

        {/* Summary strip */}
        <div className="grid grid-cols-3 gap-4 mb-6">
          {[
            { label: "Total Active Cases", value: totalActiveCases, color: "text-cyan-400" },
            { label: "Total Open Tasks", value: totalOpenTasks, color: "text-violet-400" },
            { label: "Overdue Tasks", value: totalOverdue, color: totalOverdue > 0 ? "text-red-400" : "text-emerald-400" },
          ].map(({ label, value, color }) => (
            <div key={label} className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5 text-center">
              <div className={`text-3xl font-bold ${color}`}>{value}</div>
              <div className="text-slate-400 text-sm mt-1">{label}</div>
            </div>
          ))}
        </div>

        {isLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5 animate-pulse h-28" />
            ))}
          </div>
        ) : !analysts || analysts.length === 0 ? (
          <div className="text-center py-16 text-slate-600">
            <p className="text-4xl mb-3">👥</p>
            <p className="text-sm">No analysts or managers found.</p>
          </div>
        ) : (
          <div className="space-y-3">
            {analysts.map((a) => {
              const age = avgCaseAge(a.avg_case_age_hours);
              return (
                <Link
                  key={a.id}
                  href={`/cases?assigned_to=${a.id}`}
                  className="block bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5 hover:border-slate-600 transition-colors"
                >
                  <div className="flex items-center gap-4">
                    <div className="w-10 h-10 rounded-full bg-gradient-to-br from-cyan-600 to-violet-600 flex items-center justify-center text-white font-semibold text-sm flex-shrink-0">
                      {a.full_name.charAt(0).toUpperCase()}
                    </div>

                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="text-white font-semibold text-sm">{a.full_name}</span>
                        <span
                          className={`text-xs px-2 py-0.5 rounded font-medium capitalize ${
                            ROLE_BADGE[a.role] ?? "bg-slate-700 text-slate-400"
                          }`}
                        >
                          {a.role}
                        </span>
                        {a.overdue_tasks > 0 && (
                          <span className="text-xs text-red-400 bg-red-500/10 border border-red-500/30 px-2 py-0.5 rounded">
                            {a.overdue_tasks} overdue
                          </span>
                        )}
                        {age.stale && (
                          <span className="text-xs text-amber-400 bg-amber-500/10 border border-amber-500/30 px-2 py-0.5 rounded">
                            avg case &gt;14d
                          </span>
                        )}
                      </div>
                      <p className="text-slate-500 text-xs">{a.email}</p>

                      <div className="mt-2 grid grid-cols-2 gap-3">
                        <div>
                          <div className="flex justify-between text-xs text-slate-500 mb-1">
                            <span>Cases</span>
                            <span>{a.active_cases}</span>
                          </div>
                          <LoadBar value={a.active_cases} max={maxCases} color="bg-cyan-500" />
                        </div>
                        <div>
                          <div className="flex justify-between text-xs text-slate-500 mb-1">
                            <span>Tasks</span>
                            <span>{a.open_tasks}</span>
                          </div>
                          <LoadBar value={a.open_tasks} max={maxTasks} color="bg-violet-500" />
                        </div>
                      </div>
                    </div>

                    <div className="grid grid-cols-3 gap-6 flex-shrink-0">
                      <Metric label="Active Cases" value={a.active_cases} color="text-cyan-400" />
                      <Metric label="Open Tasks" value={a.open_tasks} color="text-violet-400" />
                      <Metric label="Avg Case Age" value={age.label} color={age.stale ? "text-amber-400" : "text-slate-300"} />
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>
        )}

        <p className="text-slate-600 text-xs text-center mt-4">Workload updates every 30 seconds</p>
      </div>

      {/* ── Risk Leaderboard section ── */}
      <div>
        <div className="flex items-start justify-between mb-4">
          <div>
            <h2 className="text-white text-lg font-semibold">AI Risk Leaderboard</h2>
            <p className="text-slate-400 text-sm mt-0.5">Top 20 cases ranked by AI risk score.</p>
          </div>
          <ExportButton
            path="/api/v1/analytics/risk/export?limit=100"
            filename={`risk-leaderboard-${today}.csv`}
            label="Export Risk"
          />
        </div>

        {riskLoading ? (
          <div className="space-y-2">
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] h-12 animate-pulse" />
            ))}
          </div>
        ) : !riskRows || riskRows.length === 0 ? (
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-8 text-center text-slate-500 text-sm">
            No AI-analysed cases yet. Run AI Copilot on a case to generate risk scores.
          </div>
        ) : (
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] overflow-hidden">
            <div className="overflow-y-auto max-h-[420px]">
            <table className="w-full text-sm">
              <thead className="sticky top-0 z-10 bg-[#131f30] shadow-[0_1px_0_0_rgb(51,65,85)]">
                <tr className="text-slate-500 text-xs uppercase tracking-wider font-medium">
                  <th className="text-left px-4 py-3 font-medium">#</th>
                  <th className="text-left px-4 py-3 font-medium">Case</th>
                  <th className="text-left px-4 py-3 font-medium">Status</th>
                  <th className="text-left px-4 py-3 font-medium">Priority</th>
                  <th className="text-right px-4 py-3 font-medium">Risk Score</th>
                  <th className="text-left px-4 py-3 font-medium">Risk Level</th>
                  <th className="text-left px-4 py-3 font-medium">SLA</th>
                </tr>
              </thead>
              <tbody>
                {riskRows.map((row, i) => (
                  <tr
                    key={row.id}
                    onClick={() => window.location.href = `/cases/${row.id}`}
                    className="border-b border-slate-700/50 last:border-0 hover:bg-slate-700/30 transition-colors cursor-pointer"
                  >
                    <td className="px-4 py-3 text-slate-600 text-xs font-mono">{i + 1}</td>
                    <td className="px-4 py-3">
                      <div className="text-white font-medium text-xs">{row.case_number}</div>
                      <div className="text-slate-400 text-xs truncate max-w-[240px]">{row.title}</div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-xs text-slate-400 capitalize">{row.status.replace("_", " ")}</span>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`text-xs font-semibold capitalize ${PRIORITY_COLOR[row.priority] ?? "text-slate-400"}`}>
                        {row.priority}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      {row.risk_score != null ? (
                        <span className="text-white font-mono font-semibold">{row.risk_score}</span>
                      ) : (
                        <span className="text-slate-600">—</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {row.risk_level ? (
                        <span className={`text-xs font-semibold capitalize ${RISK_COLOR[row.risk_level] ?? "text-slate-400"}`}>
                          {row.risk_level}
                        </span>
                      ) : (
                        <span className="text-slate-600 text-xs">—</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {row.sla_breached ? (
                        <span className="text-red-400 text-xs font-semibold">Breached</span>
                      ) : (
                        <span className="text-emerald-400 text-xs">OK</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            </div>
          </div>
        )}

        <p className="text-slate-600 text-xs text-center mt-3">Risk data updates every 60 seconds</p>
      </div>
    </div>
    </RBACGate>
  );
}
