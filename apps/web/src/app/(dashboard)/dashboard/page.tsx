"use client";

import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/lib/hooks/useAuth";
import { api } from "@/lib/api";
import Link from "next/link";

interface DashboardStats {
  open_cases: number;
  pending_tasks: number;
  ai_queries_today: number;
}

function StatCard({
  label,
  value,
  color,
  href,
}: {
  label: string;
  value: number | undefined;
  color: string;
  href: string;
}) {
  return (
    <Link href={href} className="block bg-[#1e293b] border border-slate-700 rounded-xl p-5 hover:border-slate-500 transition-colors">
      <div className="text-slate-400 text-sm mb-2">{label}</div>
      <div className={`text-3xl font-bold ${color}`}>
        {value === undefined ? (
          <span className="inline-block w-12 h-8 bg-slate-700 animate-pulse rounded" />
        ) : (
          value.toLocaleString()
        )}
      </div>
    </Link>
  );
}

export default function DashboardPage() {
  const { user } = useAuth();

  const { data: stats } = useQuery<DashboardStats>({
    queryKey: ["dashboard-stats"],
    queryFn: () => api.get<DashboardStats>("/api/v1/dashboard/stats"),
    refetchInterval: 60_000,
  });

  const hour = new Date().getHours();
  const greeting = hour < 12 ? "Good morning" : hour < 17 ? "Good afternoon" : "Good evening";

  return (
    <div className="p-8">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white">
          {greeting},{" "}
          <span className="text-cyan-400">{user?.full_name?.split(" ")[0]}</span>
        </h1>
        <p className="text-slate-400 mt-1 text-sm">
          Here&apos;s what&apos;s happening across your investigation workspace today.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        <StatCard
          label="Active Cases"
          value={stats?.open_cases}
          color="text-cyan-400"
          href="/cases"
        />
        <StatCard
          label="Pending Tasks"
          value={stats?.pending_tasks}
          color="text-violet-400"
          href="/tasks"
        />
        <StatCard
          label="AI Queries Today"
          value={stats?.ai_queries_today}
          color="text-amber-400"
          href="/cases"
        />
      </div>

      <div className="mt-8 grid grid-cols-1 md:grid-cols-2 gap-5">
        <Link
          href="/cases"
          className="bg-[#1e293b] border border-slate-700 rounded-xl p-6 hover:border-slate-500 transition-colors group"
        >
          <div className="flex items-center gap-3 mb-2">
            <span className="text-2xl">⬡</span>
            <h2 className="text-white font-semibold text-lg">Investigation Cases</h2>
          </div>
          <p className="text-slate-400 text-sm">
            View, create and manage fraud investigation cases. Track SLA breaches, escalations and resolutions.
          </p>
          <div className="mt-3 text-cyan-400 text-sm group-hover:underline">View all cases →</div>
        </Link>

        <Link
          href="/tasks"
          className="bg-[#1e293b] border border-slate-700 rounded-xl p-6 hover:border-slate-500 transition-colors group"
        >
          <div className="flex items-center gap-3 mb-2">
            <span className="text-2xl">✓</span>
            <h2 className="text-white font-semibold text-lg">Task Queue</h2>
          </div>
          <p className="text-slate-400 text-sm">
            Review and complete assigned investigation tasks. Monitor overdue items and priority work.
          </p>
          <div className="mt-3 text-violet-400 text-sm group-hover:underline">View my tasks →</div>
        </Link>
      </div>
    </div>
  );
}
