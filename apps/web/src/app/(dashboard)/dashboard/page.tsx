"use client";

import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/lib/hooks/useAuth";
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
  label, value, color, href, sub,
}: {
  label: string; value: number | undefined; color: string; href: string; sub?: string;
}) {
  return (
    <Link href={href} className="block bg-[#1e293b] border border-slate-700 rounded-xl p-5 hover:border-slate-500 transition-colors group">
      <div className="text-slate-400 text-sm mb-1">{label}</div>
      <div className={`text-3xl font-bold tabular-nums ${color}`}>
        {value === undefined ? (
          <span className="inline-block w-14 h-8 bg-slate-700 animate-pulse rounded" />
        ) : (
          value.toLocaleString()
        )}
      </div>
      {sub && <div className="text-slate-500 text-xs mt-1">{sub}</div>}
    </Link>
  );
}

interface RiskCase {
  id: string;
  case_number: string;
  title: string;
  status: string;
  priority: string;
  sla_breached: boolean;
  risk_score: { String: string; Valid: boolean } | null;
  risk_level: { String: string; Valid: boolean } | null;
}

const RISK_COLORS: Record<string, string> = {
  high:    "text-red-400 bg-red-500/10",
  medium:  "text-amber-400 bg-amber-500/10",
  low:     "text-emerald-400 bg-emerald-500/10",
  unknown: "text-slate-500 bg-slate-700/30",
};

// ── Main page ─────────────────────────────────────────────────────────────────
export default function DashboardPage() {
  const { user } = useAuth();

  const { data: stats } = useQuery<DashboardStats>({
    queryKey: ["dashboard-stats"],
    queryFn: () => api.get<DashboardStats>("/api/v1/dashboard/stats"),
    refetchInterval: 60_000,
  });

  const { data: riskCases } = useQuery<RiskCase[]>({
    queryKey: ["risk-leaderboard"],
    queryFn: () => api.get<RiskCase[]>("/api/v1/analytics/risk?limit=5"),
    refetchInterval: 120_000,
  });

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
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-white">
          {greeting},{" "}
          <span className="text-cyan-400">{user?.full_name?.split(" ")[0]}</span>
        </h1>
        <p className="text-slate-400 mt-1 text-sm">
          Operations intelligence workspace — {new Date().toLocaleDateString("en-IN", { weekday: "long", day: "numeric", month: "long", year: "numeric" })}
        </p>
      </div>

      {/* Top stat cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <StatCard label="Active Cases"    value={stats?.open_cases}       color="text-cyan-400"   href="/cases"   sub="open + in-progress" />
        <StatCard label="Pending Tasks"   value={stats?.pending_tasks}     color="text-violet-400" href="/tasks"   sub="todo + in-progress" />
        <StatCard label="AI Queries Today" value={stats?.ai_queries_today} color="text-amber-400"  href="/cases"   sub="across all cases" />
        <StatCard label="SLA Breached"    value={stats?.sla_breached}      color={stats?.sla_breached ? "text-red-400" : "text-emerald-400"} href="/cases?priority=critical" sub="active cases" />
      </div>

      {/* Charts row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        {/* Donut — case status */}
        <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5">
          <div className="text-slate-400 text-xs uppercase tracking-widest font-semibold mb-4">Case Status Breakdown</div>
          <div className="flex items-center gap-5">
            <DonutChart slices={statusSlices} size={120} />
            <div className="space-y-1.5 flex-1 min-w-0">
              {statusSlices.map((s) => (
                <div key={s.label} className="flex items-center gap-2 min-w-0">
                  <span className="w-2 h-2 rounded-full flex-shrink-0" style={{ backgroundColor: s.color }} />
                  <span className="text-slate-400 text-xs truncate flex-1">{s.label}</span>
                  <span className="text-slate-200 text-xs tabular-nums font-medium">{s.value}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Horizontal bars — priority */}
        <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5">
          <div className="text-slate-400 text-xs uppercase tracking-widest font-semibold mb-4">Active Cases by Priority</div>
          {stats ? (
            <HorizontalBars items={priorityBars} />
          ) : (
            <div className="space-y-3">
              {[...Array(4)].map((_, i) => <div key={i} className="h-6 bg-slate-800 animate-pulse rounded" />)}
            </div>
          )}
        </div>

        {/* Alert panel */}
        <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5">
          <div className="text-slate-400 text-xs uppercase tracking-widest font-semibold mb-4">Attention Required</div>
          <div className="space-y-3">
            {stats ? (
              <>
                <AlertRow
                  icon="🔴"
                  label="SLA Breaches"
                  value={stats.sla_breached}
                  href="/cases"
                  critical={stats.sla_breached > 0}
                />
                <AlertRow
                  icon="⚠️"
                  label="Escalated Cases"
                  value={stats.status_escalated}
                  href="/cases?status=escalated"
                  critical={stats.status_escalated > 0}
                />
                <AlertRow
                  icon="🤖"
                  label="Pending AI Reviews"
                  value={stats.pending_ai_reviews}
                  href="/cases"
                  critical={false}
                />
                <AlertRow
                  icon="📋"
                  label="Pending Review Cases"
                  value={stats.status_pending_review}
                  href="/cases?status=pending_review"
                  critical={false}
                />
              </>
            ) : (
              [...Array(4)].map((_, i) => <div key={i} className="h-8 bg-slate-800 animate-pulse rounded" />)
            )}
          </div>
        </div>
      </div>

      {/* AI Risk Leaderboard */}
      {riskCases && riskCases.length > 0 && (
        <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5 mb-6">
          <div className="flex items-center justify-between mb-4">
            <div className="text-slate-400 text-xs uppercase tracking-widest font-semibold">Top Cases by AI Risk Score</div>
            <Link href="/cases" className="text-cyan-400 text-xs hover:text-cyan-300">View all →</Link>
          </div>
          <div className="space-y-2">
            {riskCases.map((c) => {
              const level = c.risk_level?.Valid ? c.risk_level.String : "unknown";
              const score = c.risk_score?.Valid ? parseFloat(c.risk_score.String) : null;
              const colorClass = RISK_COLORS[level] ?? RISK_COLORS.unknown;
              return (
                <Link key={c.id} href={`/cases/${c.id}/insights`} className="flex items-center gap-3 p-2 rounded-lg hover:bg-slate-700/30 transition-colors group">
                  <span className={`text-xs font-bold px-2 py-0.5 rounded min-w-[42px] text-center ${colorClass}`}>
                    {score !== null ? Math.round(score) : "—"}
                  </span>
                  <span className="font-mono text-cyan-400 text-xs w-24 flex-shrink-0">{c.case_number}</span>
                  <span className="text-slate-300 text-sm flex-1 truncate">{c.title}</span>
                  <span className={`text-xs px-1.5 py-0.5 rounded capitalize ${colorClass}`}>{level}</span>
                </Link>
              );
            })}
          </div>
        </div>
      )}

      {/* Quick-action cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Link
          href="/cases"
          className="bg-[#1e293b] border border-slate-700 rounded-xl p-5 hover:border-cyan-500/40 transition-colors group"
        >
          <div className="flex items-center gap-3 mb-1.5">
            <span className="text-xl">⬡</span>
            <h2 className="text-white font-semibold">Investigation Cases</h2>
            <span className="ml-auto text-cyan-400 text-sm opacity-0 group-hover:opacity-100 transition-opacity">Open →</span>
          </div>
          <p className="text-slate-500 text-xs">
            View, create and manage fraud investigation cases. Track SLA breaches, escalations and resolutions.
          </p>
        </Link>

        <Link
          href="/tasks"
          className="bg-[#1e293b] border border-slate-700 rounded-xl p-5 hover:border-violet-500/40 transition-colors group"
        >
          <div className="flex items-center gap-3 mb-1.5">
            <span className="text-xl">✓</span>
            <h2 className="text-white font-semibold">Task Queue</h2>
            <span className="ml-auto text-violet-400 text-sm opacity-0 group-hover:opacity-100 transition-opacity">Open →</span>
          </div>
          <p className="text-slate-500 text-xs">
            Review and complete assigned investigation tasks. Monitor overdue items and priority work.
          </p>
        </Link>

        <Link
          href="/workload"
          className="bg-[#1e293b] border border-slate-700 rounded-xl p-5 hover:border-amber-500/40 transition-colors group"
        >
          <div className="flex items-center gap-3 mb-1.5">
            <span className="text-xl">◑</span>
            <h2 className="text-white font-semibold">Team Workload</h2>
            <span className="ml-auto text-amber-400 text-sm opacity-0 group-hover:opacity-100 transition-opacity">Open →</span>
          </div>
          <p className="text-slate-500 text-xs">
            Monitor analyst capacity — active cases, open tasks, overdue items and average case age across the team.
          </p>
        </Link>

        <Link
          href="/sla"
          className="bg-[#1e293b] border border-slate-700 rounded-xl p-5 hover:border-emerald-500/40 transition-colors group"
        >
          <div className="flex items-center gap-3 mb-1.5">
            <span className="text-xl">◷</span>
            <h2 className="text-white font-semibold">SLA Policies</h2>
            <span className="ml-auto text-emerald-400 text-sm opacity-0 group-hover:opacity-100 transition-opacity">Open →</span>
          </div>
          <p className="text-slate-500 text-xs">
            Configure breach and warning thresholds per case priority. Keep SLA compliance visible across operations.
          </p>
        </Link>
      </div>
    </div>
  );
}

function AlertRow({
  icon, label, value, href, critical,
}: {
  icon: string; label: string; value: number; href: string; critical: boolean;
}) {
  return (
    <Link
      href={href}
      className="flex items-center gap-3 py-2 border-b border-slate-700/50 last:border-0 hover:bg-slate-700/20 -mx-1 px-1 rounded transition-colors"
    >
      <span className="text-sm">{icon}</span>
      <span className="text-slate-300 text-sm flex-1">{label}</span>
      <span className={`text-sm font-bold tabular-nums ${critical && value > 0 ? "text-red-400" : "text-slate-300"}`}>
        {value}
      </span>
    </Link>
  );
}
