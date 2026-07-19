"use client";

import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { RBACGate } from "@/components/auth/RBACGate";
import { ExportButton } from "@/components/analytics/ExportButton";
import Link from "next/link";
import { AlertTriangle, TrendingUp, TrendingDown, FileCheck, Clock, Users, FolderOpen, Minus } from "lucide-react";

interface MLROSummary {
  cases_opened_mtd: number;
  cases_closed_mtd: number;
  strs_filed_mtd: number;
  strs_pending_review: number;
  sla_breached_open: number;
  avg_case_age_days: number;
  disposition_counts: Record<string, number>;
  investigators: {
    user_id: string;
    full_name: string;
    active_cases: number;
    closed_mtd: number;
    strs_mtd: number;
    breached_open: number;
  }[];
}

const DISPOSITION_LABELS: Record<string, string> = {
  str_filed:                 "STR Filed",
  sar_filed:                 "SAR Filed",
  no_sar_warranted:          "No SAR Required",
  false_positive:            "False Positive",
  no_action:                 "No Action",
  enhanced_monitoring:       "Enhanced Monitoring",
  closed_monitoring:         "Closed — Under Watch",
  account_closed:            "Account Closed",
  escalated_law_enforcement: "Referred to LE",
  referred_law_enforcement:  "Referred to LE",
  referred_internal:         "Referred Internally",
};

const DISPOSITION_COLORS: Record<string, string> = {
  str_filed:                 "bg-amber-500",
  sar_filed:                 "bg-orange-500",
  no_sar_warranted:          "bg-slate-400",
  false_positive:            "bg-slate-600",
  no_action:                 "bg-slate-500",
  enhanced_monitoring:       "bg-blue-500",
  closed_monitoring:         "bg-cyan-600",
  account_closed:            "bg-red-500",
  escalated_law_enforcement: "bg-violet-500",
  referred_law_enforcement:  "bg-violet-500",
  referred_internal:         "bg-indigo-500",
};

function KPICard({
  icon: Icon,
  label,
  value,
  sub,
  color,
  href,
}: {
  icon: React.ElementType;
  label: string;
  value: number | string;
  sub?: string;
  color: string;
  href?: string;
}) {
  const inner = (
    <div className={`bg-slate-800/60 border rounded-xl p-4 flex gap-3 items-start hover:border-slate-600 transition-colors ${color.includes("red") ? "border-red-500/30" : "border-slate-700/50"}`}>
      <div className={`p-2 rounded-lg ${color} flex-shrink-0`}>
        <Icon size={18} className="text-white" />
      </div>
      <div>
        <div className="text-2xl font-bold text-white tabular-nums">{value}</div>
        <div className="text-xs text-slate-400 mt-0.5">{label}</div>
        {sub && <div className="text-xs text-slate-500 mt-0.5">{sub}</div>}
      </div>
    </div>
  );
  return href ? <Link href={href}>{inner}</Link> : inner;
}

function DispositionBar({ counts }: { counts: Record<string, number> }) {
  const entries = Object.entries(counts).sort((a, b) => b[1] - a[1]);
  const total = entries.reduce((s, [, v]) => s + v, 0);
  if (total === 0)
    return <p className="text-slate-500 text-sm">No closed cases yet.</p>;

  return (
    <div className="space-y-3">
      {entries.map(([key, val]) => (
        <div key={key}>
          <div className="flex justify-between text-xs mb-1">
            <span className="text-slate-300">{DISPOSITION_LABELS[key] ?? key}</span>
            <span className="text-slate-400 tabular-nums">{val} ({Math.round((val / total) * 100)}%)</span>
          </div>
          <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full ${DISPOSITION_COLORS[key] ?? "bg-cyan-500"}`}
              style={{ width: `${(val / total) * 100}%`, transition: "width 0.6s ease" }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}

function mtdRange(): string {
  const now = new Date();
  const start = new Date(now.getFullYear(), now.getMonth(), 1);
  const fmt = (d: Date) => d.toLocaleDateString("en-IN", { day: "numeric", month: "short", year: "numeric" });
  return `${fmt(start)} – ${fmt(now)}`;
}

export default function MLRODashboardPage() {
  const [lastRefreshed, setLastRefreshed] = useState<Date | null>(null);
  const { data, isLoading, error } = useQuery<MLROSummary>({
    queryKey: ["mlro-summary"],
    queryFn: () => api.get<MLROSummary>("/api/v1/analytics/mlro-summary"),
    refetchInterval: 30_000,
  });
  useEffect(() => { if (data) setLastRefreshed(new Date()); }, [data]);

  const netBacklog = data ? data.cases_opened_mtd - data.cases_closed_mtd : 0;
  const dateRange = mtdRange();

  return (
    <RBACGate allowedRoles={["manager", "admin", "super_admin"]}>
      <div className="p-6 max-w-6xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-xl font-semibold text-white">Compliance Dashboard</h1>
            <p className="text-slate-400 text-sm mt-0.5">
              Principal Officer view · {dateRange}
              {lastRefreshed && (
                <span className="text-slate-600 ml-2">
                  · as of {lastRefreshed.toLocaleTimeString("en-IN", { hour: "2-digit", minute: "2-digit" })}
                </span>
              )}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Link
              href="/cases?sla_breached=true"
              className="text-xs text-amber-400 hover:text-amber-300 border border-amber-500/30 px-3 py-1.5 rounded-lg transition-colors"
            >
              SLA breached cases →
            </Link>
            <ExportButton
              path="/api/v1/analytics/mlro-summary/export"
              filename={`compliance-report-${new Date().toISOString().slice(0, 7)}.csv`}
              label="Export Report"
            />
          </div>
        </div>

        {isLoading && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="h-24 bg-slate-800/40 rounded-xl animate-pulse" />
            ))}
          </div>
        )}

        {error && (
          <div className="bg-red-500/10 border border-red-500/30 rounded-xl p-4 text-red-400 text-sm mb-8">
            Failed to load compliance summary. You may not have sufficient permissions.
          </div>
        )}

        {data && (
          <>
            {/* KPI grid — 4 columns, 2 rows */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
              <KPICard icon={FolderOpen}  label="Cases Opened (MTD)"  value={data.cases_opened_mtd}    color="bg-cyan-500/20" />
              <KPICard icon={TrendingUp}  label="Cases Closed (MTD)"  value={data.cases_closed_mtd}    color="bg-green-500/20" />
              <KPICard
                icon={netBacklog > 0 ? TrendingUp : netBacklog < 0 ? TrendingDown : Minus}
                label="Net Backlog (MTD)"
                value={netBacklog > 0 ? `+${netBacklog}` : String(netBacklog)}
                sub={netBacklog > 0 ? "Backlog growing" : netBacklog < 0 ? "Backlog shrinking" : "Steady"}
                color={netBacklog > 0 ? "bg-red-500/20" : netBacklog < 0 ? "bg-green-500/20" : "bg-slate-600/40"}
              />
              <KPICard
                icon={AlertTriangle}
                label="SLA Breached (Open)"
                value={data.sla_breached_open}
                color={data.sla_breached_open > 0 ? "bg-red-500/30" : "bg-slate-600/40"}
                href="/cases?sla_breached=true"
              />
              <KPICard icon={FileCheck}   label="STRs Filed (MTD)"    value={data.strs_filed_mtd}      color="bg-violet-500/20" href="/cases?has_str=true" />
              <KPICard icon={Clock}       label="STRs Pending Review"  value={data.strs_pending_review} color="bg-amber-500/20" href="/cases?str_status=draft" />
              <KPICard
                icon={Clock}
                label="Avg Case Age"
                value={`${data.avg_case_age_days}d`}
                sub={data.avg_case_age_days > 7 ? "Exceeds 7-day STR window" : "Within STR window"}
                color={data.avg_case_age_days > 30 ? "bg-red-500/20" : data.avg_case_age_days > 7 ? "bg-amber-500/20" : "bg-green-500/20"}
              />
              <KPICard icon={Users}       label="Active Investigators" value={(data.investigators ?? []).length} color="bg-slate-600/40" />
            </div>

            {/* Two-column: disposition + investigator table */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Disposition breakdown */}
              <div className="bg-slate-800/60 border border-slate-700/50 rounded-xl p-5">
                <h2 className="text-sm font-semibold text-slate-200 mb-4">Case Dispositions (Approved Closures)</h2>
                <DispositionBar counts={data.disposition_counts} />
              </div>

              {/* Investigator table */}
              <div className="bg-slate-800/60 border border-slate-700/50 rounded-xl p-5">
                <h2 className="text-sm font-semibold text-slate-200 mb-4">Investigator Performance (MTD)</h2>
                {(data.investigators ?? []).length === 0 ? (
                  <p className="text-slate-500 text-sm">No investigators found.</p>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-xs">
                      <thead>
                        <tr className="text-slate-500 border-b border-slate-700">
                          <th className="text-left pb-2 font-medium">Name</th>
                          <th className="text-right pb-2 font-medium">Active</th>
                          <th className="text-right pb-2 font-medium">Closed</th>
                          <th className="text-right pb-2 font-medium">STRs</th>
                          <th className="text-right pb-2 font-medium">SLA ⚠</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-700/40">
                        {(data.investigators ?? []).map((inv) => (
                          <tr key={inv.user_id} className="hover:bg-slate-700/20 transition-colors">
                            <td className="py-2">
                              <Link href={`/cases?assigned_to=${inv.user_id}`} className="text-slate-200 font-medium hover:text-cyan-400 transition-colors">
                                {inv.full_name}
                              </Link>
                            </td>
                            <td className="py-2 text-right text-slate-300 tabular-nums">{inv.active_cases}</td>
                            <td className="py-2 text-right text-green-400 tabular-nums">{inv.closed_mtd}</td>
                            <td className="py-2 text-right text-violet-400 tabular-nums">{inv.strs_mtd}</td>
                            <td className={`py-2 text-right tabular-nums font-medium ${inv.breached_open > 0 ? "text-red-400" : "text-slate-500"}`}>
                              {inv.breached_open}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          </>
        )}
      </div>
    </RBACGate>
  );
}
