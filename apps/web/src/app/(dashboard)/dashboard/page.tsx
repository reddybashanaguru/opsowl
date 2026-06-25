"use client";

import { useQuery } from "@tanstack/react-query";
import { useRef, useEffect, useState } from "react";
import { AlertTriangle, TrendingUp, Bot, ClipboardList, FolderOpen, CheckSquare, Users, Clock, FileWarning } from "lucide-react";
import { STRDeadlineBadge } from "@/components/str/STRDeadlineBadge";
import { useAuth } from "@/lib/hooks/useAuth";
import { useConfig } from "@/lib/hooks/useConfig";
import { api } from "@/lib/api";
import Link from "next/link";

interface DashboardStats {
  open_cases: number;
  pending_tasks: number;
  ai_queries_today: number;
  status_open: number;
  status_in_progress: number;
  status_pending_review: number;
  status_escalated: number;
  status_closed: number;
  prio_critical: number;
  prio_high: number;
  prio_medium: number;
  prio_low: number;
  sla_breached: number;
  pending_ai_reviews: number;
}

// ── Pure-SVG donut chart ──────────────────────────────────────────────────────
interface DonutSlice {
  value: number;
  color: string;
  label: string;
}

function DonutChart({ slices, size = 120 }: { slices: DonutSlice[]; size?: number }) {
  const total = slices.reduce((s, d) => s + d.value, 0);
  if (total === 0) return <div className="w-[120px] h-[120px] rounded-full bg-slate-800 flex items-center justify-center text-slate-600 text-xs">No data</div>;

  const r = 40;
  const cx = size / 2;
  const cy = size / 2;
  const circ = 2 * Math.PI * r;
  const gap = 2; // gap between segments in pixels

  let cumulative = 0;
  const segments = slices.map((s) => {
    const pct = s.value / total;
    const dash = Math.max(0, pct * circ - gap);
    const offset = circ - cumulative * circ;
    cumulative += pct;
    return { ...s, dash, offset };
  });

  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
      {segments.map((seg, i) => (
        <circle
          key={i}
          cx={cx}
          cy={cy}
          r={r}
          fill="none"
          stroke={seg.color}
          strokeWidth={16}
          strokeDasharray={`${seg.dash} ${circ}`}
          strokeDashoffset={seg.offset}
          strokeLinecap="butt"
          style={{ transform: `rotate(-90deg)`, transformOrigin: `${cx}px ${cy}px` }}
        />
      ))}
      {/* Center text */}
      <text x={cx} y={cy - 4} textAnchor="middle" fill="#94a3b8" fontSize="10" fontFamily="monospace">Total</text>
      <text x={cx} y={cy + 11} textAnchor="middle" fill="#f1f5f9" fontSize="16" fontWeight="bold" fontFamily="monospace">{total}</text>
    </svg>
  );
}

// ── Horizontal bar chart ──────────────────────────────────────────────────────
interface BarItem {
  label: string;
  value: number;
  color: string;
  bg: string;
}

function HorizontalBars({ items }: { items: BarItem[] }) {
  const max = Math.max(...items.map((i) => i.value), 1);
  return (
    <div className="space-y-2.5">
      {items.map((item) => (
        <div key={item.label}>
          <div className="flex justify-between items-center mb-1">
            <span className={`text-xs font-medium ${item.color}`}>{item.label}</span>
            <span className="text-xs text-slate-400 tabular-nums">{item.value}</span>
          </div>
          <div className={`h-2 rounded-full ${item.bg} overflow-hidden`}>
            <div
              className={`h-full rounded-full ${item.color.replace("text-", "bg-")}`}
              style={{ width: `${(item.value / max) * 100}%`, transition: "width 0.6s ease" }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}

// ── Stat card ─────────────────────────────────────────────────────────────────
function StatCard({
  label, value, color, href, sub, Icon,
}: {
  label: string; value: number | undefined; color: string; href: string; sub?: string;
  Icon?: React.ElementType;
}) {
  return (
    <Link href={href} className="block bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5 hover:border-slate-600 hover:bg-[#1a2a40] transition-all group">
      <div className="flex items-center justify-between mb-3">
        <div className="text-slate-500 text-xs uppercase tracking-wider font-semibold">{label}</div>
        {Icon && (
          <span className="p-1.5 rounded-lg bg-slate-800/60 group-hover:bg-slate-700/60 transition-colors">
            <Icon size={13} className="text-slate-500 group-hover:text-slate-400 transition-colors" />
          </span>
        )}
      </div>
      <div className={`text-4xl font-bold tabular-nums tracking-tight ${color}`}>
        {value === undefined ? (
          <span className="inline-block w-14 h-10 bg-slate-700 animate-pulse rounded" />
        ) : (
          value.toLocaleString()
        )}
      </div>
      {sub && <div className="text-slate-600 text-[11px] mt-2">{sub}</div>}
    </Link>
  );
}

interface QueueCase {
  id: string;
  case_number: string;
  title: string;
  status: string;
  priority: string;
  sla_breached: boolean;
  sla_breach_at: string | null;
}

interface QueueResponse {
  data: QueueCase[];
  meta: { total_count: number; has_more: boolean };
}

interface RiskCase {
  id: string;
  case_number: string;
  title: string;
  status: string;
  priority: string;
  sla_breached: boolean;
  risk_score: string | null;
  risk_level: string | null;
}

const RISK_COLORS: Record<string, string> = {
  high:    "text-red-400 bg-red-500/10",
  medium:  "text-amber-400 bg-amber-500/10",
  low:     "text-emerald-400 bg-emerald-500/10",
  unknown: "text-slate-500 bg-slate-700/30",
};

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

function slaRemaining(breachAt: string | null): { label: string; urgent: boolean } {
  if (!breachAt) return { label: "—", urgent: false };
  const rem = new Date(breachAt).getTime() - Date.now();
  if (rem <= 0) return { label: "Breached", urgent: true };
  const hrs = Math.floor(rem / 3600000);
  if (hrs < 1) return { label: `${Math.floor(rem / 60000)}m`, urgent: true };
  if (hrs < 4) return { label: `${hrs}h`, urgent: true };
  return { label: `${hrs}h`, urgent: false };
}

const PRIORITY_COLORS: Record<string, string> = {
  critical: "text-red-400 bg-red-500/10 border-red-500/30",
  high:     "text-orange-400 bg-orange-500/10 border-orange-500/30",
  medium:   "text-amber-400 bg-amber-500/10 border-amber-500/30",
  low:      "text-slate-400 bg-slate-700/40 border-slate-600/30",
};

// ── Main page ─────────────────────────────────────────────────────────────────
export default function DashboardPage() {
  const { user } = useAuth();
  const cfg = useConfig();
  const prevBreached = useRef<number | null>(null);
  const [breachPulse, setBreachPulse] = useState(false);

  const { data: stats } = useQuery<DashboardStats>({
    queryKey: ["dashboard-stats"],
    queryFn: () => api.get<DashboardStats>("/api/v1/dashboard/stats"),
    refetchInterval: 15_000,
  });

  const { data: myQueue } = useQuery<QueueResponse>({
    queryKey: ["my-queue", user?.id],
    queryFn: () =>
      api.getList<QueueCase, { total_count: number; has_more: boolean }>(
        `/api/v1/cases?assigned_to=${user?.id}&status=active&limit=5`,
      ).then((r) => ({ data: r.data, meta: r.meta as { total_count: number; has_more: boolean } })),
    enabled: !!user?.id,
    refetchInterval: 30_000,
  });

  const { data: riskCases } = useQuery<RiskCase[]>({
    queryKey: ["risk-leaderboard"],
    queryFn: () => api.get<RiskCase[]>("/api/v1/analytics/risk?limit=5"),
    refetchInterval: 120_000,
  });

  const isComplianceOfficer = ["manager", "admin", "super_admin"].includes(user?.role ?? "");
  const { data: strDeadlines } = useQuery<{
    items: Array<{
      filing_id: string; case_id: string; case_number: string;
      suspicion_date: string; deadline_date: string;
      days_remaining: number; is_overdue: boolean;
      status: string; report_type: string;
    }>;
    count: number;
  }>({
    queryKey: ["str-deadlines", cfg.str_deadline_days],
    queryFn: () => api.get(`/api/v1/str/deadlines?deadline_days=${cfg.str_deadline_days || 30}&window_days=14`),
    enabled: isComplianceOfficer,
    refetchInterval: 120_000,
  });

  useEffect(() => {
    if (stats?.sla_breached === undefined) return;
    if (prevBreached.current !== null && stats.sla_breached > prevBreached.current) {
      setBreachPulse(true);
      const t = setTimeout(() => setBreachPulse(false), 5000);
      return () => clearTimeout(t);
    }
    prevBreached.current = stats.sla_breached;
  }, [stats?.sla_breached]);

  const hour = new Date().getHours();
  const greeting = hour < 12 ? "Good morning" : hour < 17 ? "Good afternoon" : "Good evening";

  const statusSlices: DonutSlice[] = [
    { label: "Open",           value: stats?.status_open          ?? 0, color: "#22d3ee" },
    { label: "In Progress",    value: stats?.status_in_progress   ?? 0, color: "#818cf8" },
    { label: "Pending Review", value: stats?.status_pending_review ?? 0, color: "#f59e0b" },
    { label: "Escalated",      value: stats?.status_escalated     ?? 0, color: "#f87171" },
    { label: "Closed",         value: stats?.status_closed        ?? 0, color: "#4ade80" },
  ];

  const priorityBars: BarItem[] = [
    { label: "Critical", value: stats?.prio_critical ?? 0, color: "text-red-400",    bg: "bg-red-900/20" },
    { label: "High",     value: stats?.prio_high     ?? 0, color: "text-orange-400", bg: "bg-orange-900/20" },
    { label: "Medium",   value: stats?.prio_medium   ?? 0, color: "text-amber-400",  bg: "bg-amber-900/20" },
    { label: "Low",      value: stats?.prio_low      ?? 0, color: "text-slate-400",  bg: "bg-slate-800" },
  ];

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-white">
          {greeting},{" "}
          <span className="text-cyan-400">{user?.full_name?.split(" ")[0]}</span>
        </h1>
        <p className="text-slate-400 mt-1 text-sm">
          AI-Native Investigations & Operations Workspace — {new Date().toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long", year: "numeric" })}
        </p>
      </div>

      {/* Top stat cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <StatCard label="Active Cases"    value={stats?.open_cases}         color="text-cyan-400"   href="/cases?status=active"                          sub="open · in progress · escalated" Icon={FolderOpen} />
        <StatCard label="Pending Tasks"  value={stats?.pending_tasks}      color="text-violet-400" href={user?.role === "analyst" ? "/tasks" : "/tasks?scope=all"} sub="todo + in-progress" Icon={CheckSquare} />
        <StatCard label="My Open Cases"  value={myQueue?.meta?.total_count} color="text-sky-400"   href={`/cases?assigned_to=${user?.id}&status=active`}        sub="assigned to me"    Icon={Users} />
        <StatCard label="SLA Breached"   value={stats?.sla_breached}       color={(stats?.sla_breached ?? 0) > 0 ? "text-red-400" : "text-emerald-400"} href="/cases?sla_breached=true" sub="requires immediate action" Icon={Clock} />
      </div>

      {/* Analytics + Attention row — 3/5 analytics | 2/5 action panel */}
      <div className="grid grid-cols-5 gap-4 mb-4">

        {/* Case Overview — status donut + priority bars (3/5) */}
        <div className="col-span-3 bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5">
          <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-widest mb-5">Case Overview</p>
          <div className="grid grid-cols-2 divide-x divide-slate-700/50 gap-0">
            {/* Status donut */}
            <div className="pr-6">
              <p className="text-[10px] font-semibold text-slate-500 uppercase tracking-widest mb-4">By Status</p>
              {stats ? (
                <div className="flex items-center gap-5">
                  <DonutChart slices={statusSlices} size={110} />
                  <div className="space-y-2.5 flex-1 min-w-0">
                    {statusSlices.map((s) => (
                      <div key={s.label} className="flex items-center gap-2 min-w-0">
                        <span className="w-2 h-2 rounded-full flex-shrink-0" style={{ backgroundColor: s.color }} />
                        <span className="text-slate-400 text-xs truncate flex-1">{s.label}</span>
                        <span className="text-slate-100 text-xs tabular-nums font-semibold w-6 text-right">{s.value}</span>
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="flex items-center gap-5">
                  <div className="w-[110px] h-[110px] bg-slate-800 animate-pulse rounded-full flex-shrink-0" />
                  <div className="space-y-2.5 flex-1">{[...Array(5)].map((_,i)=><div key={i} className="h-4 bg-slate-800 animate-pulse rounded"/>)}</div>
                </div>
              )}
            </div>

            {/* Priority bars */}
            <div className="pl-6">
              <p className="text-[10px] font-semibold text-slate-500 uppercase tracking-widest mb-4">By Priority</p>
              {stats ? (
                <HorizontalBars items={priorityBars} />
              ) : (
                <div className="space-y-3">{[...Array(4)].map((_,i)=><div key={i} className="h-7 bg-slate-800 animate-pulse rounded"/>)}</div>
              )}
            </div>
          </div>
        </div>

        {/* Attention Required (2/5) */}
        <div className={`col-span-2 bg-[#131f30] ring-1 ring-white/[0.04] border rounded-xl p-5 transition-all flex flex-col ${breachPulse ? "border-red-500/60 ring-red-500/20 animate-pulse" : "border-[#1e3050]"}`}>
          <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-widest mb-5">Attention Required</p>
          <div className="flex flex-col gap-0.5 flex-1 justify-between">
            {stats ? (
              <>
                <AlertRow Icon={AlertTriangle}  label={`${cfg.sla_label} Breaches`}          value={stats.sla_breached}          href="/cases?sla_breached=true"        critical={stats.sla_breached > 0} />
                <AlertRow Icon={TrendingUp}     label={`Escalated ${cfg.cases_label}`}       value={stats.status_escalated}      href="/cases?status=escalated"         critical={stats.status_escalated > 0} />
                <AlertRow Icon={Bot}            label="Pending AI Reviews"                   value={stats.pending_ai_reviews}    href="/cases"                          critical={false} />
                <AlertRow Icon={ClipboardList}  label={`Pending Review ${cfg.cases_label}`} value={stats.status_pending_review} href="/cases?status=pending_review"    critical={false} />
              </>
            ) : (
              [...Array(4)].map((_,i) => <div key={i} className="h-12 bg-slate-800 animate-pulse rounded" />)
            )}
          </div>
        </div>
      </div>

      {/* ── Intel + Deadlines + Queue ────────────────────────────────────────── */}
      {(() => {
        const scoredCases = (riskCases ?? []).filter(c => c.risk_score && parseFloat(c.risk_score) > 0);
        const allRiskCases = riskCases ?? [];
        const hasSTR = isComplianceOfficer && strDeadlines && strDeadlines.count > 0;
        const hasRisk = allRiskCases.length > 0;
        return (
          <>
            {(hasRisk || hasSTR) && (
              <div className="grid grid-cols-5 gap-4 mb-4 items-start">
                {/* High-Risk Cases — 2/5 width */}
                {hasRisk && (
                  <div className={`bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5 ${!hasSTR ? "col-span-5" : "col-span-2"}`}>
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-2">
                        <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-widest">High-Risk Cases</p>
                        {scoredCases.length > 0 && (
                          <span className="text-[10px] px-1.5 py-0.5 bg-orange-900/30 text-orange-300 border border-orange-700/40 rounded-full font-medium">{scoredCases.length} scored</span>
                        )}
                      </div>
                      <Link href="/cases" className="text-[11px] text-cyan-500 hover:text-cyan-400 font-medium">View all →</Link>
                    </div>
                    <div className="space-y-1">
                      {allRiskCases.map((c) => {
                        const level = c.risk_level ?? "unknown";
                        const score = c.risk_score ? Math.round(parseFloat(c.risk_score)) : null;
                        const colorClass = score ? (RISK_COLORS[level] ?? RISK_COLORS.unknown) : "bg-slate-800 text-slate-600";
                        return (
                          <Link key={c.id} href={`/cases/${c.id}/insights`} className="flex items-center gap-2.5 p-2 rounded-lg hover:bg-slate-700/30 transition-colors group">
                            <span className={`text-xs font-bold tabular-nums px-2 py-0.5 rounded min-w-[36px] text-center flex-shrink-0 ${colorClass}`}>
                              {score ?? "—"}
                            </span>
                            <span className="font-mono text-cyan-400 text-[11px] flex-shrink-0">{c.case_number}</span>
                            <span className="text-slate-300 text-xs flex-1 truncate group-hover:text-white min-w-0">{c.title}</span>
                            {level !== "unknown" && score && (
                              <span className={`text-[10px] px-1.5 py-0.5 rounded capitalize flex-shrink-0 ${colorClass}`}>{level}</span>
                            )}
                          </Link>
                        );
                      })}
                    </div>
                  </div>
                )}

                {/* STR Filing Deadlines — 3/5 width */}
                {hasSTR && (
                  <div className={`bg-[#131f30] border border-red-900/40 rounded-xl ring-1 ring-red-500/10 p-5 ${!hasRisk ? "col-span-5" : "col-span-3"}`}>
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-2">
                        <FileWarning size={13} className="text-red-400 flex-shrink-0" />
                        <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-widest">{cfg.str_label} Filing Deadlines</p>
                        <span className="text-[10px] px-1.5 py-0.5 bg-red-900/30 text-red-300 border border-red-700/40 rounded-full font-medium">{strDeadlines!.count}</span>
                      </div>
                      <Link href="/cases" className="text-[11px] text-cyan-500 hover:text-cyan-400 font-medium">View all →</Link>
                    </div>
                    <div className="grid grid-cols-2 gap-x-4 gap-y-0.5 max-h-52 overflow-y-auto pr-1">
                      {strDeadlines!.items.map((item) => (
                        <Link
                          key={item.filing_id}
                          href={`/cases/${item.case_id}/str`}
                          className="flex items-center gap-2 p-2 rounded-lg hover:bg-slate-700/30 transition-colors group min-w-0"
                        >
                          <span className="font-mono text-cyan-400 text-[11px] flex-shrink-0">{item.case_number}</span>
                          <span className="text-slate-500 text-[11px] flex-1 truncate min-w-0">
                            {new Date(item.suspicion_date).toLocaleDateString("en-GB", { day: "2-digit", month: "short" })}
                          </span>
                          <STRDeadlineBadge daysRemaining={item.days_remaining} isOverdue={item.is_overdue} compact />
                        </Link>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* My Queue — full width */}
            <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                  <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-widest">My Queue</p>
                  {myQueue && myQueue.data.length > 0 && (
                    <span className="text-[10px] px-1.5 py-0.5 bg-slate-700/60 text-slate-300 border border-slate-600/50 rounded-full font-medium">{myQueue.data.length}</span>
                  )}
                </div>
                <Link href={`/cases?assigned_to=${user?.id}&status=active`} className="text-[11px] text-cyan-500 hover:text-cyan-400 font-medium">View all →</Link>
              </div>
              {!myQueue ? (
                <div className="grid grid-cols-2 gap-3">
                  {[...Array(4)].map((_, i) => <div key={i} className="h-10 bg-slate-800 animate-pulse rounded" />)}
                </div>
              ) : myQueue.data.length === 0 ? (
                <div className="text-slate-500 text-sm py-6 text-center">All clear — no {cfg.cases_label.toLowerCase()} assigned to you.</div>
              ) : (
                <div className="grid grid-cols-2 gap-x-4 gap-y-1">
                  {myQueue.data.map((c) => {
                    const sla = slaRemaining(c.sla_breach_at);
                    const prioClass = PRIORITY_COLORS[c.priority] ?? PRIORITY_COLORS.low;
                    return (
                      <Link
                        key={c.id}
                        href={`/cases/${c.id}`}
                        className="flex items-center gap-2 p-2 rounded-lg hover:bg-slate-700/30 transition-colors group min-w-0"
                      >
                        <span className={`text-[10px] font-semibold px-1.5 py-0.5 rounded border ${prioClass} uppercase flex-shrink-0`}>
                          {c.priority.slice(0, 3)}
                        </span>
                        <span className="font-mono text-cyan-400 text-xs flex-shrink-0">{c.case_number}</span>
                        <span className="text-slate-300 text-xs flex-1 truncate group-hover:text-white min-w-0">{c.title}</span>
                        {sla.urgent && (
                          <span className="text-red-400 text-[10px] font-semibold flex-shrink-0">{sla.label}</span>
                        )}
                      </Link>
                    );
                  })}
                </div>
              )}
            </div>
          </>
        );
      })()}
    </div>
  );
}

function AlertRow({
  Icon, label, value, href, critical,
}: {
  Icon: React.ElementType; label: string; value: number; href: string; critical: boolean;
}) {
  return (
    <Link
      href={href}
      className="flex items-center gap-3 py-2 border-b border-slate-700/50 last:border-0 hover:bg-slate-700/20 -mx-1 px-1 rounded transition-colors group"
    >
      <Icon size={14} className={`flex-shrink-0 ${critical && value > 0 ? "text-red-400" : "text-slate-500"}`} />
      <span className="text-slate-300 text-sm flex-1">{label}</span>
      <span className={`text-sm font-bold tabular-nums px-2 py-0.5 rounded ${critical && value > 0 ? "text-red-400 bg-red-500/10" : "text-slate-400 bg-slate-700/40"}`}>
        {value}
      </span>
    </Link>
  );
}
