"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense } from "react";
import { api } from "@/lib/api";
import { StatusBadge } from "@/components/cases/StatusBadge";
import { PriorityBadge } from "@/components/cases/PriorityBadge";
import { SLACountdown } from "@/components/cases/SLACountdown";

interface CaseRow {
  id: string;
  case_number: string;
  title: string;
  status: string;
  priority: string;
  assigned_to: { id: string; full_name: string } | null;
  sla_breach_at: string | null;
  sla_breached: boolean;
  tags: string[];
  created_at: string;
}

interface ListMeta {
  next_cursor: string | null;
  has_more: boolean;
  total_count: number;
}

const STATUSES = ["", "draft", "open", "in_progress", "pending_review", "closed", "escalated"];
const PRIORITIES = ["", "critical", "high", "medium", "low"];

function CasesContent() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [status, setStatus] = useState(searchParams.get("status") ?? "");
  const [priority, setPriority] = useState(searchParams.get("priority") ?? "");

  const params = new URLSearchParams();
  if (status) params.set("status", status);
  if (priority) params.set("priority", priority);
  params.set("limit", "20");

  const { data, isLoading, error } = useQuery<{ data: CaseRow[]; meta: ListMeta }>({
    queryKey: ["cases", status, priority],
    queryFn: () => api.getList<CaseRow, ListMeta>(`/api/v1/cases?${params.toString()}`),
  });

  const cases = data?.data ?? [];
  const total = data?.meta?.total_count ?? 0;

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-white text-2xl font-bold">Cases</h1>
          <p className="text-slate-400 text-sm mt-0.5">
            {total > 0 ? `${total} cases` : "No cases yet"}
          </p>
        </div>
        <Link
          href="/cases/new"
          className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-2 rounded-lg text-sm transition-colors"
        >
          + New Case
        </Link>
      </div>

      {/* Filters */}
      <div className="flex gap-3 mb-6">
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          className="bg-[#1e293b] border border-slate-600 text-slate-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
        >
          {STATUSES.map((s) => (
            <option key={s} value={s}>{s === "" ? "All Statuses" : s.replace("_", " ")}</option>
          ))}
        </select>
        <select
          value={priority}
          onChange={(e) => setPriority(e.target.value)}
          className="bg-[#1e293b] border border-slate-600 text-slate-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
        >
          {PRIORITIES.map((p) => (
            <option key={p} value={p}>{p === "" ? "All Priorities" : p.charAt(0).toUpperCase() + p.slice(1)}</option>
          ))}
        </select>
      </div>

      {/* Table */}
      <div className="bg-[#1e293b] border border-slate-700 rounded-xl overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center h-40">
            <div className="w-6 h-6 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
          </div>
        ) : error ? (
          <div className="text-center py-12 text-red-400 text-sm">Failed to load cases</div>
        ) : cases.length === 0 ? (
          <div className="text-center py-16">
            <p className="text-slate-400 text-sm mb-4">No cases found</p>
            <Link href="/cases/new" className="text-cyan-400 hover:text-cyan-300 text-sm">
              Create your first case →
            </Link>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-700 text-slate-400 text-xs uppercase tracking-wide">
                <th className="text-left px-4 py-3 font-medium">Case</th>
                <th className="text-left px-4 py-3 font-medium">Status</th>
                <th className="text-left px-4 py-3 font-medium">Priority</th>
                <th className="text-left px-4 py-3 font-medium">Assigned</th>
                <th className="text-left px-4 py-3 font-medium">SLA</th>
                <th className="text-left px-4 py-3 font-medium">Created</th>
              </tr>
            </thead>
            <tbody>
              {cases.map((c) => (
                <tr
                  key={c.id}
                  onClick={() => router.push(`/cases/${c.id}`)}
                  className="border-b border-slate-700/50 hover:bg-slate-700/30 cursor-pointer transition-colors"
                >
                  <td className="px-4 py-3">
                    <div className="font-mono text-xs text-cyan-400 mb-0.5">{c.case_number}</div>
                    <div className="text-white font-medium truncate max-w-xs">{c.title}</div>
                    {(c.tags?.length ?? 0) > 0 && (
                      <div className="flex gap-1 mt-1 flex-wrap">
                        {(c.tags ?? []).slice(0, 3).map((t) => (
                          <span key={t} className="text-xs bg-slate-700 text-slate-400 px-1.5 py-0.5 rounded">
                            {t}
                          </span>
                        ))}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={c.status} />
                  </td>
                  <td className="px-4 py-3">
                    <PriorityBadge priority={c.priority} />
                  </td>
                  <td className="px-4 py-3 text-slate-400">
                    {c.assigned_to?.full_name ?? <span className="text-slate-600">Unassigned</span>}
                  </td>
                  <td className="px-4 py-3">
                    <SLACountdown breachAt={c.sla_breach_at} breached={c.sla_breached} />
                  </td>
                  <td className="px-4 py-3 text-slate-500 text-xs">
                    {new Date(c.created_at).toLocaleDateString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}

export default function CasesPage() {
  return (
    <Suspense fallback={<div className="p-6 text-slate-400">Loading...</div>}>
      <CasesContent />
    </Suspense>
  );
}
