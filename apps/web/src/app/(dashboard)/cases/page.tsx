"use client";

import { useState, useMemo, useCallback } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense } from "react";
import { api } from "@/lib/api";
import { useConfig } from "@/lib/hooks/useConfig";
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

interface CasesPage {
  data: CaseRow[];
  meta: ListMeta;
}

const STATUSES = ["", "draft", "open", "in_progress", "pending_review", "escalated", "closed"];
const PRIORITIES = ["", "critical", "high", "medium", "low"];

const STATUS_LABELS: Record<string, string> = {
  "": "All Statuses",
  draft: "Draft",
  open: "Open",
  in_progress: "In Progress",
  pending_review: "Pending Review",
  escalated: "Escalated",
  closed: "Closed",
};

// Skeleton row for loading state
function SkeletonRow() {
  return (
    <tr className="border-b border-slate-700/50">
      {[220, 80, 80, 120, 100, 80].map((w, i) => (
        <td key={i} className="px-4 py-3">
          <div className={`h-4 bg-slate-700/60 animate-pulse rounded`} style={{ width: w }} />
        </td>
      ))}
    </tr>
  );
}

function CasesContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const cfg = useConfig();

  const [status, setStatus] = useState(searchParams.get("status") ?? "");
  const [priority, setPriority] = useState(searchParams.get("priority") ?? "");
  const [search, setSearch] = useState("");

  const buildParams = useCallback((cursor?: string) => {
    const p = new URLSearchParams({ limit: "25" });
    if (status) p.set("status", status);
    if (priority) p.set("priority", priority);
    if (cursor) p.set("cursor", cursor);
    return p.toString();
  }, [status, priority]);

  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useInfiniteQuery<CasesPage>({
      queryKey: ["cases", status, priority],
      queryFn: ({ pageParam }) =>
        api.getList<CaseRow, ListMeta>(`/api/v1/cases?${buildParams(pageParam as string | undefined)}`),
      initialPageParam: undefined,
      getNextPageParam: (last) => last.meta.next_cursor ?? undefined,
    });

  const allCases = useMemo(() => {
    const seen = new Set<string>();
    return (data?.pages.flatMap((p) => p.data) ?? []).filter((c) => {
      if (seen.has(c.id)) return false;
      seen.add(c.id);
      return true;
    });
  }, [data]);
  const total = data?.pages[0]?.meta.total_count ?? 0;

  // Client-side search over already-loaded cases
  const filtered = useMemo(() => {
    if (!search.trim()) return allCases;
    const q = search.toLowerCase();
    return allCases.filter(
      (c) =>
        c.title.toLowerCase().includes(q) ||
        c.case_number.toLowerCase().includes(q) ||
        (c.tags ?? []).some((t) => t.toLowerCase().includes(q)) ||
        (c.assigned_to?.full_name ?? "").toLowerCase().includes(q),
    );
  }, [allCases, search]);

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-5">
        <div>
          <h1 className="text-white text-2xl font-bold">{cfg.cases_label}</h1>
          <p className="text-slate-400 text-sm mt-0.5">
            {search ? `${filtered.length} of ${total}` : `${total}`} {total !== 1 ? cfg.cases_label.toLowerCase() : cfg.case_label.toLowerCase()}
            {search && <span className="text-slate-500"> matching &ldquo;{search}&rdquo;</span>}
          </p>
        </div>
        <Link
          href="/cases/new"
          className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-2 rounded-lg text-sm transition-colors"
        >
          + New {cfg.case_label}
        </Link>
      </div>

      {/* Filters + Search */}
      <div className="flex flex-wrap gap-2 mb-5">
        {/* Search */}
        <div className="relative">
          <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500 text-xs">⌕</span>
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={`Search ${cfg.cases_label.toLowerCase()}, tags, assignee…`}
            className="bg-[#1e293b] border border-slate-600 text-slate-300 text-sm rounded-lg pl-7 pr-3 py-2 w-64 focus:outline-none focus:border-cyan-500 placeholder:text-slate-600"
          />
          {search && (
            <button onClick={() => setSearch("")} className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 text-xs">
              ✕
            </button>
          )}
        </div>

        {/* Status */}
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          className="bg-[#1e293b] border border-slate-600 text-slate-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
        >
          {STATUSES.map((s) => (
            <option key={s} value={s}>{STATUS_LABELS[s] ?? s}</option>
          ))}
        </select>

        {/* Priority */}
        <select
          value={priority}
          onChange={(e) => setPriority(e.target.value)}
          className="bg-[#1e293b] border border-slate-600 text-slate-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
        >
          {PRIORITIES.map((p) => (
            <option key={p} value={p}>{p === "" ? "All Priorities" : p.charAt(0).toUpperCase() + p.slice(1)}</option>
          ))}
        </select>

        {/* Clear filters */}
        {(status || priority || search) && (
          <button
            onClick={() => { setStatus(""); setPriority(""); setSearch(""); }}
            className="text-xs text-slate-500 hover:text-slate-300 px-3 py-2 border border-slate-700 rounded-lg hover:border-slate-500 transition-colors"
          >
            Clear all
          </button>
        )}
      </div>

      {/* Table */}
      <div className="bg-[#1e293b] border border-slate-700 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-700 text-slate-500 text-xs uppercase tracking-wide">
              <th className="text-left px-4 py-3 font-medium">Case</th>
              <th className="text-left px-4 py-3 font-medium">Status</th>
              <th className="text-left px-4 py-3 font-medium">Priority</th>
              <th className="text-left px-4 py-3 font-medium">Assigned</th>
              <th className="text-left px-4 py-3 font-medium">SLA</th>
              <th className="text-left px-4 py-3 font-medium">Created</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              [...Array(8)].map((_, i) => <SkeletonRow key={i} />)
            ) : filtered.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-16 text-center">
                  <p className="text-slate-400 text-sm mb-2">
                    {search ? `No cases match "${search}"` : "No cases found"}
                  </p>
                  {!search && (
                    <Link href="/cases/new" className="text-cyan-400 hover:text-cyan-300 text-sm">
                      Create your first case →
                    </Link>
                  )}
                </td>
              </tr>
            ) : (
              filtered.map((c) => (
                <tr
                  key={c.id}
                  onClick={() => router.push(`/cases/${c.id}`)}
                  className="border-b border-slate-700/40 hover:bg-slate-700/25 cursor-pointer transition-colors"
                >
                  <td className="px-4 py-3 max-w-xs">
                    <div className="font-mono text-xs text-cyan-400 mb-0.5">{c.case_number}</div>
                    <div className="text-white font-medium truncate">{c.title}</div>
                    {(c.tags?.length ?? 0) > 0 && (
                      <div className="flex gap-1 mt-1 flex-wrap">
                        {(c.tags ?? []).slice(0, 3).map((t) => (
                          <span key={t} className="text-xs bg-slate-700/80 text-slate-400 px-1.5 py-0.5 rounded">
                            {t}
                          </span>
                        ))}
                        {(c.tags?.length ?? 0) > 3 && (
                          <span className="text-xs text-slate-600">+{(c.tags?.length ?? 0) - 3}</span>
                        )}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={c.status} />
                  </td>
                  <td className="px-4 py-3">
                    <PriorityBadge priority={c.priority} />
                  </td>
                  <td className="px-4 py-3 text-slate-400 text-sm">
                    {c.assigned_to?.full_name ?? <span className="text-slate-600 italic">Unassigned</span>}
                  </td>
                  <td className="px-4 py-3">
                    <SLACountdown breachAt={c.sla_breach_at} breached={c.sla_breached} />
                  </td>
                  <td className="px-4 py-3 text-slate-500 text-xs tabular-nums">
                    {new Date(c.created_at).toLocaleDateString("en-IN", { day: "2-digit", month: "short", year: "numeric" })}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>

        {/* Load more footer */}
        {hasNextPage && !search && (
          <div className="border-t border-slate-700/50 p-3 text-center">
            <button
              onClick={() => fetchNextPage()}
              disabled={isFetchingNextPage}
              className="text-sm text-slate-400 hover:text-cyan-400 transition-colors disabled:opacity-50"
            >
              {isFetchingNextPage ? (
                <span className="flex items-center gap-2 justify-center">
                  <span className="w-3 h-3 border border-cyan-400 border-t-transparent rounded-full animate-spin" />
                  Loading…
                </span>
              ) : (
                `Load more cases (${total - allCases.length} remaining)`
              )}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

export default function CasesPage() {
  return (
    <Suspense fallback={
      <div className="p-6">
        <div className="h-8 w-32 bg-slate-700 animate-pulse rounded mb-4" />
        <div className="space-y-2">
          {[...Array(5)].map((_, i) => <div key={i} className="h-14 bg-slate-800 animate-pulse rounded" />)}
        </div>
      </div>
    }>
      <CasesContent />
    </Suspense>
  );
}
