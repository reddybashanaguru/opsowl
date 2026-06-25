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
  if (total === 0) return (
    <div
      className="rounded-full bg-slate-800/60 border border-slate-700/50 flex items-center justify-center text-slate-600 text-[10px] font-medium"
      style={{ width: size, height: size, flexShrink: 0 }}
    >
      No data
    </div>
  );

  const r = 38;
  const cx = size / 2;
  const cy = size / 2;
  const circ = 2 * Math.PI * r;
  const gap = 2.5;

  let cumulative = 0;
  const segments = slices.filter(s => s.value > 0).map((s) => {
    const pct = s.value / total;
    const dash = Math.max(0, pct * circ - gap);
    const offset = -(cumulative * circ);
    cumulative += pct;
    return { ...s, pct, dash, offset };
  });

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      style={{ flexShrink: 0, minWidth: size }}
    >
      {/* Track ring */}
      <circle cx={cx} cy={cy} r={r} fill="none" stroke="#1e293b" strokeWidth={14} />
      {segments.map((seg, i) => (
        <circle
          key={i}
          cx={cx}
          cy={cy}
          r={r}
          fill="none"
          stroke={seg.color}
          strokeWidth={14}
          strokeDasharray={`${seg.dash} ${circ}`}
          strokeDashoffset={seg.offset}
          strokeLinecap="butt"
          style={{ transform: "rotate(-90deg)", transformOrigin: `${cx}px ${cy}px` }}
        />
      ))}
      <text x={cx} y={cy - 5} textAnchor="middle" fill="#64748b" fontSize="9" fontFamily="system-ui, sans-serif" fontWeight="500" letterSpacing="0.05em">TOTAL</text>
      <text x={cx} y={cy + 12} textAnchor="middle" fill="#f1f5f9" fontSize="18" fontWeight="700" fontFamily="system-ui, sans-serif">{total}</text>
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
  label, value, color, href, sub, Icon, accent,
}: {
  label: string; value: number | undefined; color: string; href: string; sub?: string;
  Icon?: React.ElementType; accent?: string;
}) {
  return (
    <Link href={href} className="block bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] overflow-hidden hover:border-slate-600 hover:bg-[#1a2a40] transition-all group relative">
      {accent && (
        <div className={`absolute left-0 top-0 bottom-0 w-[3px] ${accent} rounded-r-sm`} />
      )}
      <div className="p-4 pl-5">
        <div className="flex items-center justify-between mb-2">
          <div className="text-slate-500 text-[10px] uppercase tracking-widest font-semibold">{label}</div>
          {Icon && (
            <span className="p-1 rounded-lg bg-slate-800/50 group-hover:bg-slate-700/50 transition-colors">
              <Icon size={11} className="text-slate-500 group-hover:text-slate-400 transition-colors" />
            </span>
          )}
        </div>
        <div className={`text-3xl font-bold tabular-nums leading-none tracking-tight ${color}`}>
          {value === undefined ? (
            <span className="inline-block w-12 h-8 bg-slate-700/60 animate-pulse rounded-lg" />
          ) : (
            value.toLocaleString()
          )}
        </div>
        {sub && <div className="text-slate-600 text-[10px] mt-2 truncate">{sub}</div>}
      </div>
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

  const scoredCases = (riskCases ?? []).filter(c => c.risk_score && parseFloat(c.risk_score) > 0);
  const hasSTR = isComplianceOfficer && strDeadlines && strDeadlines.count > 0;
  const hasRisk = scoredCases.length > 0;

  return (
    /* h-full fills the <main flex-1> container exactly — no page scroll */
    <div className="h-full overflow-hidden flex flex-col gap-4 p-5">

      {/* ── Header ──────────────────────────────────────────────────────────── */}
      <div className="flex-shrink-0 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-white leading-tight">
            {greeting},{" "}
            <span className="text-cyan-400">{user?.full_name?.split(" ")[0]}</span>
          </h1>
          <p className="text-slate-500 text-xs mt-0.5">
            {new Date().toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long", year: "numeric" })}
          </p>
        </div>
      </div>

      {/* ── Stat cards ──────────────────────────────────────────────────────── */}
      <div className="grid grid-cols-4 gap-3 flex-shrink-0">
        <StatCard label="Active Cases"   value={stats?.open_cases}          color="text-cyan-400"    href="/cases?status=active"                                   sub="open · in progress · escalated" Icon={FolderOpen}  accent="bg-cyan-500" />
        <StatCard label="Pending Tasks"  value={stats?.pending_tasks}       color="text-violet-400"  href={user?.role === "analyst" ? "/tasks" : "/tasks?scope=all"} sub="todo · in progress"             Icon={CheckSquare} accent="bg-violet-500" />
        <StatCard label="My Open Cases"  value={myQueue?.meta?.total_count} color="text-sky-400"     href={`/cases?assigned_to=${user?.id}&status=active`}          sub="assigned to me"                 Icon={Users}       accent="bg-sky-500" />
        <StatCard label="SLA Breached"   value={stats?.sla_breached}        color={(stats?.sla_breached ?? 0) > 0 ? "text-red-400" : "text-emerald-400"} href="/cases?sla_breached=true" sub={(stats?.sla_breached ?? 0) > 0 ? "immediate action needed" : "all cases within SLA"} Icon={Clock} accent={(stats?.sla_breached ?? 0) > 0 ? "bg-red-500" : "bg-emerald-500"} />
      </div>

      {/* ── Analytics + Attention ─── flex-shrink-0, fixed-height cards ─────── */}
      <div className="grid grid-cols-5 gap-3 flex-shrink-0">

        {/* Case Overview — status donut + priority bars (3/5) */}
        <div className="col-span-3 bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] px-5 pt-4 pb-4">
          <div className="flex items-center justify-between mb-3 pb-2.5 border-b border-slate-800">
            <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-widest">Case Overview</p>
            <Link href="/cases" className="text-[11px] text-cyan-500 hover:text-cyan-400 font-medium">View all →</Link>
          </div>
          <div className="grid grid-cols-2 gap-0">
            {/* Status donut */}
            <div className="pr-5 border-r border-slate-800">
              <p className="text-[10px] font-semibold text-slate-500 uppercase tracking-widest mb-3">By Status</p>
              {stats ? (
                <div className="flex items-center gap-3">
                  <DonutChart slices={statusSlices} size={90} />
                  <div className="space-y-1.5 flex-1 min-w-0">
                    {statusSlices.map((s) => (
                      <div key={s.label} className="flex items-center gap-1.5 min-w-0">
                        <span className="w-1.5 h-1.5 rounded-full flex-shrink-0" style={{ backgroundColor: s.color }} />
                        <span className="text-slate-400 text-[11px] truncate flex-1">{s.label}</span>
                        <span className={`text-[11px] tabular-nums font-semibold min-w-[18px] text-right ${s.value > 0 ? "text-slate-100" : "text-slate-600"}`}>{s.value}</span>
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="flex items-center gap-3">
                  <div className="w-[90px] h-[90px] bg-slate-800/60 animate-pulse rounded-full flex-shrink-0" />
                  <div className="space-y-1.5 flex-1">{[...Array(5)].map((_,i)=><div key={i} className="h-3 bg-slate-800 animate-pulse rounded"/>)}</div>
                </div>
              )}
            </div>

            {/* Priority bars */}
            <div className="pl-5">
              <p className="text-[10px] font-semibold text-slate-500 uppercase tracking-widest mb-3">By Priority</p>
              {stats ? (
                <HorizontalBars items={priorityBars} />
              ) : (
                <div className="space-y-2.5">{[...Array(4)].map((_,i)=><div key={i} className="h-6 bg-slate-800 animate-pulse rounded"/>)}</div>
              )}
            </div>
          </div>
        </div>

        {/* Attention Required (2/5) */}
        <div className={`col-span-2 bg-[#131f30] ring-1 ring-white/[0.04] border rounded-xl px-5 pt-4 pb-4 transition-all ${breachPulse ? "border-red-500/40 ring-red-500/15" : "border-[#1e3050]"}`}>
          <div className="flex items-center justify-between mb-3 pb-2.5 border-b border-slate-800">
            <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-widest">Attention Required</p>
            {breachPulse && <span className="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse" />}
          </div>
          {stats ? (
            <div className="grid grid-cols-2 gap-2">
              <AlertCell Icon={AlertTriangle}  label={`${cfg.sla_label} Breaches`}           value={stats.sla_breached}          href="/cases?sla_breached=true"     critical />
              <AlertCell Icon={TrendingUp}     label={`Escalated ${cfg.cases_label}`}        value={stats.status_escalated}      href="/cases?status=escalated"      critical />
              <AlertCell Icon={Bot}            label="AI Reviews"                            value={stats.pending_ai_reviews}    href="/cases"                       critical={false} />
              <AlertCell Icon={ClipboardList}  label="Pending Review"                        value={stats.status_pending_review} href="/cases?status=pending_review" critical={false} />
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-2">
              {[...Array(4)].map((_,i) => <div key={i} className="h-[64px] bg-slate-800 animate-pulse rounded-lg" />)}
            </div>
          )}
        </div>
      </div>

      {/* ── Bottom section: fills ALL remaining viewport height ─────────────── */}
      {/* flex-1 min-h-0 = "take remaining space, don't overflow the flex parent" */}
      {(() => {

        const MyQueueCard = ({ className = "" }: { className?: string }) => (
          <div className={`bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] px-4 pt-4 pb-3 flex flex-col ${className}`}>
            <div className="flex items-center justify-between mb-3 pb-2.5 border-b border-slate-800 flex-shrink-0">
              <div className="flex items-center gap-2">
                <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-widest">My Queue</p>
                {myQueue && myQueue.data.length > 0 && (
                  <span className="text-[10px] px-1.5 py-0.5 bg-slate-700/60 text-slate-300 border border-slate-700/50 rounded-full font-medium tabular-nums">{myQueue.data.length}</span>
                )}
              </div>
              <Link href={`/cases?assigned_to=${user?.id}&status=active`} className="text-[11px] text-cyan-500 hover:text-cyan-400 font-medium">View all →</Link>
            </div>
            {!myQueue ? (
              <div className="space-y-2 flex-1">
                {[...Array(4)].map((_, i) => <div key={i} className="h-9 bg-slate-800 animate-pulse rounded-lg" />)}
              </div>
            ) : myQueue.data.length === 0 ? (
              <div className="flex-1 flex items-center justify-center text-slate-500 text-sm">All clear — no {cfg.cases_label.toLowerCase()} assigned to you.</div>
            ) : (
              <div className="flex-1 overflow-y-auto min-h-0 space-y-0.5">
                {myQueue.data.map((c) => {
                  const sla = slaRemaining(c.sla_breach_at);
                  const prioClass = PRIORITY_COLORS[c.priority] ?? PRIORITY_COLORS.low;
                  return (
                    <Link
                      key={c.id}
                      href={`/cases/${c.id}`}
                      className="flex items-center gap-2.5 p-2 rounded-lg hover:bg-slate-700/30 transition-colors group min-w-0"
                    >
                      <span className={`text-[10px] font-semibold px-1.5 py-0.5 rounded border ${prioClass} uppercase flex-shrink-0`}>
                        {c.priority.slice(0, 3)}
                      </span>
                      <span className="font-mono text-cyan-400 text-[11px] flex-shrink-0">{c.case_number}</span>
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
        );

        const HighRiskCard = () => (
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] px-4 pt-4 pb-3 flex-shrink-0">
            <div className="flex items-center justify-between mb-3 pb-2.5 border-b border-slate-800">
              <div className="flex items-center gap-2">
                <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-widest">High-Risk Cases</p>
                <span className="text-[10px] px-1.5 py-0.5 bg-orange-900/25 text-orange-300 border border-orange-800/40 rounded-full font-medium tabular-nums">{scoredCases.length}</span>
              </div>
              <Link href="/cases" className="text-[11px] text-cyan-500 hover:text-cyan-400 font-medium">View all →</Link>
            </div>
            <div className="space-y-0.5">
              {scoredCases.map((c) => {
                const level = c.risk_level ?? "unknown";
                const score = Math.round(parseFloat(c.risk_score!));
                const colorClass = RISK_COLORS[level] ?? RISK_COLORS.unknown;
                return (
                  <Link key={c.id} href={`/cases/${c.id}/insights`} className="flex items-center gap-2.5 p-2 rounded-lg hover:bg-slate-700/30 transition-colors group">
                    <span className={`text-[11px] font-bold tabular-nums px-1.5 py-0.5 rounded-md min-w-[34px] text-center flex-shrink-0 ${colorClass}`}>{score}</span>
                    <span className="font-mono text-cyan-400 text-[11px] flex-shrink-0">{c.case_number}</span>
                    <span className="text-slate-300 text-xs flex-1 truncate group-hover:text-white min-w-0">{c.title}</span>
                    <span className={`text-[10px] px-1.5 py-0.5 rounded capitalize flex-shrink-0 ${colorClass}`}>{level}</span>
                  </Link>
                );
              })}
            </div>
          </div>
        );

        if (!hasSTR) {
          /* No STR: High-Risk (shrink) + My Queue fills remaining height */
          return (
            <div className="flex-1 min-h-0 flex flex-col gap-3">
              {hasRisk && <HighRiskCard />}
              <MyQueueCard className="flex-1 min-h-0" />
            </div>
          );
        }

        /* STR present: 2/5 left stacked | 3/5 right full-height */
        return (
          <div className="flex-1 min-h-0 grid grid-cols-5 gap-3">
            {/* Left: High-Risk (shrink) above My Queue (fills remaining) */}
            <div className="col-span-2 flex flex-col gap-3 min-h-0">
              {hasRisk && <HighRiskCard />}
              <MyQueueCard className="flex-1 min-h-0" />
            </div>

            {/* Right: STR full height — header fixed, list scrolls */}
            <div className="col-span-3 min-h-0 bg-[#131f30] border border-red-900/30 rounded-xl ring-1 ring-red-500/[0.08] px-4 pt-4 pb-3 flex flex-col">
              <div className="flex items-center justify-between mb-3 pb-2.5 border-b border-red-900/30 flex-shrink-0">
                <div className="flex items-center gap-2">
                  <FileWarning size={12} className="text-red-400 flex-shrink-0" />
                  <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-widest">{cfg.str_label} Filing Deadlines</p>
                  <span className="text-[10px] px-1.5 py-0.5 bg-red-900/25 text-red-300 border border-red-800/40 rounded-full font-medium tabular-nums">{strDeadlines!.count}</span>
                </div>
                <Link href="/cases" className="text-[11px] text-cyan-500 hover:text-cyan-400 font-medium">View all →</Link>
              </div>
              {/* Scrollable list fills remaining card height */}
              <div className="flex-1 overflow-y-auto min-h-0 space-y-0.5">
                {strDeadlines!.items.map((item) => (
                  <Link
                    key={item.filing_id}
                    href={`/cases/${item.case_id}/str`}
                    className="flex items-center gap-3 px-2 py-2 rounded-lg hover:bg-slate-700/30 transition-colors group"
                  >
                    <span className="font-mono text-cyan-400 text-[11px] group-hover:text-cyan-300 transition-colors flex-shrink-0 w-[118px]">{item.case_number}</span>
                    <span className="text-slate-500 text-[11px] flex-shrink-0 w-[48px]">
                      {new Date(item.suspicion_date).toLocaleDateString("en-GB", { day: "2-digit", month: "short" })}
                    </span>
                    <span className="text-slate-600 text-[11px] flex-1 truncate min-w-0 capitalize">{item.report_type ?? cfg.str_label}</span>
                    <STRDeadlineBadge daysRemaining={item.days_remaining} isOverdue={item.is_overdue} compact />
                  </Link>
                ))}
              </div>
            </div>
          </div>
        );
      })()}
    </div>
  );
}

function AlertCell({
  Icon, label, value, href, critical,
}: {
  Icon: React.ElementType; label: string; value: number; href: string; critical: boolean;
}) {
  const isAlert = critical && value > 0;
  const isInfo = !critical && value > 0;
  return (
    <Link
      href={href}
      className={`flex flex-col gap-2 p-3 rounded-xl border transition-all group ${
        isAlert
          ? "bg-red-500/[0.07] border-red-500/20 hover:bg-red-500/10 hover:border-red-500/35"
          : isInfo
          ? "bg-amber-500/[0.06] border-amber-500/15 hover:bg-amber-500/10 hover:border-amber-500/25"
          : "bg-slate-800/40 border-slate-700/30 hover:bg-slate-700/40 hover:border-slate-600/50"
      }`}
    >
      <div className="flex items-center justify-between">
        <Icon
          size={13}
          className={`flex-shrink-0 ${isAlert ? "text-red-400" : isInfo ? "text-amber-400" : "text-emerald-500/60"}`}
        />
        <span className={`text-2xl font-bold tabular-nums leading-none tracking-tight ${
          isAlert ? "text-red-300" : isInfo ? "text-amber-300" : "text-emerald-500/80"
        }`}>
          {value}
        </span>
      </div>
      <p className={`text-[10px] font-medium uppercase tracking-wider leading-tight ${
        isAlert ? "text-red-400/80" : isInfo ? "text-amber-400/70" : "text-slate-500"
      }`}>
        {label}
      </p>
    </Link>
  );
}
