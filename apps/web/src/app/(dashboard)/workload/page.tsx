"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

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

const ROLE_BADGE: Record<string, string> = {
  analyst: "bg-blue-500/20 text-blue-300",
  manager: "bg-violet-500/20 text-violet-300",
};

function Metric({ label, value, color }: { label: string; value: number | string; color?: string }) {
  return (
    <div className="text-center">
      <div className={`text-xl font-bold ${color ?? "text-white"}`}>{value}</div>
      <div className="text-slate-500 text-xs mt-0.5">{label}</div>
    </div>
  );
}

function LoadBar({ value, max, color }: { value: number; max: number; color: string }) {
  const pct = max > 0 ? Math.min((value / max) * 100, 100) : 0;
  return (
    <div className="h-1.5 bg-slate-700 rounded-full overflow-hidden">
      <div
        className={`h-full rounded-full transition-all ${color}`}
        style={{ width: `${pct}%` }}
      />
    </div>
  );
}

function avgCaseAge(hours: string): string {
  const h = parseFloat(hours);
  if (isNaN(h)) return "—";
  if (h < 24) return `${Math.round(h)}h`;
  return `${Math.round(h / 24)}d`;
}

export default function WorkloadPage() {
  const { data: analysts, isLoading, error } = useQuery<AnalystWorkload[]>({
    queryKey: ["workload"],
    queryFn: () => api.get<AnalystWorkload[]>("/api/v1/analytics/workload"),
    refetchInterval: 30000,
  });

  const maxCases = Math.max(...(analysts ?? []).map((a) => a.active_cases), 1);
  const maxTasks = Math.max(...(analysts ?? []).map((a) => a.open_tasks), 1);

  const totalActiveCases = (analysts ?? []).reduce((s, a) => s + a.active_cases, 0);
  const totalOpenTasks = (analysts ?? []).reduce((s, a) => s + a.open_tasks, 0);
  const totalOverdue = (analysts ?? []).reduce((s, a) => s + a.overdue_tasks, 0);

  if (error) {
    return (
      <div className="p-6 max-w-5xl mx-auto">
        <div className="bg-red-500/10 border border-red-500/30 text-red-300 rounded-xl p-4 text-sm">
          Access denied or failed to load workload data. Manager role required.
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="mb-6">
        <h1 className="text-white text-2xl font-bold">Team Workload</h1>
        <p className="text-slate-400 text-sm mt-1">
          Real-time case and task distribution across analysts and managers.
        </p>
      </div>

      {/* Summary strip */}
      <div className="grid grid-cols-3 gap-4 mb-8">
        {[
          { label: "Total Active Cases", value: totalActiveCases, color: "text-cyan-400" },
          { label: "Total Open Tasks", value: totalOpenTasks, color: "text-violet-400" },
          { label: "Overdue Tasks", value: totalOverdue, color: totalOverdue > 0 ? "text-red-400" : "text-emerald-400" },
        ].map(({ label, value, color }) => (
          <div key={label} className="bg-[#1e293b] border border-slate-700 rounded-xl p-5 text-center">
            <div className={`text-3xl font-bold ${color}`}>{value}</div>
            <div className="text-slate-400 text-sm mt-1">{label}</div>
          </div>
        ))}
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="bg-[#1e293b] border border-slate-700 rounded-xl p-5 animate-pulse h-28" />
          ))}
        </div>
      ) : !analysts || analysts.length === 0 ? (
        <div className="text-center py-16 text-slate-600">
          <p className="text-4xl mb-3">👥</p>
          <p className="text-sm">No analysts or managers found.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {analysts.map((a) => (
            <div key={a.id} className="bg-[#1e293b] border border-slate-700 rounded-xl p-5 hover:border-slate-600 transition-colors">
              <div className="flex items-center gap-4">
                {/* Avatar */}
                <div className="w-10 h-10 rounded-full bg-gradient-to-br from-cyan-600 to-violet-600 flex items-center justify-center text-white font-semibold text-sm flex-shrink-0">
                  {a.full_name.charAt(0).toUpperCase()}
                </div>

                {/* Name + role */}
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-white font-semibold text-sm">{a.full_name}</span>
                    <span className={`text-xs px-2 py-0.5 rounded font-medium capitalize ${ROLE_BADGE[a.role] ?? "bg-slate-700 text-slate-400"}`}>
                      {a.role}
                    </span>
                    {a.overdue_tasks > 0 && (
                      <span className="text-xs text-red-400 bg-red-500/10 border border-red-500/30 px-2 py-0.5 rounded">
                        {a.overdue_tasks} overdue
                      </span>
                    )}
                  </div>
                  <p className="text-slate-500 text-xs">{a.email}</p>

                  {/* Load bars */}
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

                {/* Metrics */}
                <div className="grid grid-cols-3 gap-6 flex-shrink-0">
                  <Metric label="Active Cases" value={a.active_cases} color="text-cyan-400" />
                  <Metric label="Open Tasks" value={a.open_tasks} color="text-violet-400" />
                  <Metric label="Avg Case Age" value={avgCaseAge(a.avg_case_age_hours)} color="text-slate-300" />
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      <p className="text-slate-600 text-xs text-center mt-6">Updates every 30 seconds</p>
    </div>
  );
}
