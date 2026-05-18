"use client";

import { useState, useCallback } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { RBACGate } from "@/components/auth/RBACGate";

// ── Types ─────────────────────────────────────────────────────────────────────

interface AuditEntry {
  id: string;
  event_code: string;
  actor_id?: string;
  actor_name?: string;
  actor_email?: string;
  ip_address?: string;
  resource_type?: string;
  resource_id?: string;
  metadata: Record<string, unknown>;
  severity: "info" | "warn" | "error";
  created_at: string;
}

interface AuditPage {
  logs: AuditEntry[];
  has_more: boolean;
  next_cursor?: string | null;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

const SEVERITY_STYLE: Record<string, string> = {
  info: "text-slate-400 bg-slate-700/40",
  warn: "text-amber-400 bg-amber-400/10",
  error: "text-red-400 bg-red-400/10",
};

const SEVERITY_DOT: Record<string, string> = {
  info: "bg-slate-500",
  warn: "bg-amber-400",
  error: "bg-red-400",
};

function timeFormat(dateStr: string): string {
  return new Date(dateStr).toLocaleString("en-IN", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}

// ── Skeleton ──────────────────────────────────────────────────────────────────

function TableSkeleton() {
  return (
    <>
      {Array.from({ length: 10 }).map((_, i) => (
        <tr key={i} className="border-b border-slate-700/40 animate-pulse">
          <td className="px-4 py-3"><div className="h-3 w-32 bg-slate-700 rounded" /></td>
          <td className="px-4 py-3"><div className="h-3 w-36 bg-slate-700 rounded" /></td>
          <td className="px-4 py-3"><div className="h-3 w-28 bg-slate-700 rounded" /></td>
          <td className="px-4 py-3"><div className="h-3 w-16 bg-slate-700 rounded" /></td>
          <td className="px-4 py-3"><div className="h-3 w-24 bg-slate-700 rounded" /></td>
          <td className="px-4 py-3"><div className="h-3 w-32 bg-slate-700 rounded" /></td>
        </tr>
      ))}
    </>
  );
}

// ── Metadata popover ──────────────────────────────────────────────────────────

function MetaCell({ meta }: { meta: Record<string, unknown> }) {
  const [open, setOpen] = useState(false);
  const entries = Object.entries(meta).filter(([, v]) => v !== null && v !== undefined);
  if (entries.length === 0) return <span className="text-slate-700 text-xs">—</span>;
  return (
    <div className="relative inline-block">
      <button
        onClick={() => setOpen(!open)}
        className="text-xs text-slate-500 hover:text-cyan-400 transition-colors underline decoration-dotted"
      >
        {entries.length} field{entries.length !== 1 ? "s" : ""}
      </button>
      {open && (
        <div className="absolute z-50 right-0 bottom-6 w-72 bg-slate-900 border border-slate-600 rounded-lg p-3 shadow-2xl text-left">
          <pre className="text-[10px] text-slate-300 font-mono overflow-auto max-h-40 whitespace-pre-wrap">
            {JSON.stringify(meta, null, 2)}
          </pre>
          <button
            onClick={() => setOpen(false)}
            className="mt-2 text-[10px] text-slate-600 hover:text-slate-400"
          >
            close
          </button>
        </div>
      )}
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function AuditPage() {
  const [severity, setSeverity] = useState("");
  const [eventFilter, setEventFilter] = useState("");
  const [debouncedEvent, setDebouncedEvent] = useState("");

  const buildUrl = useCallback(
    (cursor?: string) => {
      const params = new URLSearchParams({ limit: "50" });
      if (severity) params.set("severity", severity);
      if (debouncedEvent) params.set("event_code", debouncedEvent);
      if (cursor) params.set("cursor", cursor);
      return `/api/v1/audit?${params}`;
    },
    [severity, debouncedEvent],
  );

  const { data, isLoading, isFetchingNextPage, fetchNextPage, hasNextPage } = useInfiniteQuery<AuditPage>({
    queryKey: ["audit", severity, debouncedEvent],
    queryFn: ({ pageParam }) =>
      api.get<AuditPage>(buildUrl(pageParam as string | undefined)),
    getNextPageParam: (last) => (last.has_more && last.next_cursor) ? last.next_cursor : undefined,
    initialPageParam: undefined,
  });

  const allLogs = data?.pages.flatMap((p) => p.logs) ?? [];
  const total = allLogs.length;

  function handleEventInputChange(val: string) {
    setEventFilter(val);
    clearTimeout((window as unknown as Record<string, unknown>)._auditDebounce as ReturnType<typeof setTimeout>);
    (window as unknown as Record<string, unknown>)._auditDebounce = setTimeout(() => setDebouncedEvent(val), 400);
  }

  async function handleExportCSV() {
    const { useAuthStore } = await import("@/lib/stores/auth.store");
    const token = useAuthStore.getState().accessToken;
    const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
    const res = await fetch(`${apiBase}/api/v1/audit/export`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    if (!res.ok) return;
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `audit-logs-${new Date().toISOString().slice(0, 10)}.csv`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  }

  return (
    <RBACGate allowedRoles={["admin", "super_admin"]} fallback={
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <div className="text-3xl mb-3">🔒</div>
          <p className="text-slate-400">Admin access required</p>
        </div>
      </div>
    }>
      <div className="px-6 py-8 max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-xl font-semibold text-white">Audit Logs</h1>
            <p className="text-slate-500 text-sm mt-0.5">All system mutations across every module</p>
          </div>
          <button
            onClick={handleExportCSV}
            className="flex items-center gap-2 px-3 py-2 bg-slate-700/60 hover:bg-slate-700 border border-slate-600 rounded-lg text-sm text-slate-300 hover:text-white transition-colors"
          >
            <span>↓</span> Export CSV
          </button>
        </div>

        {/* Filters */}
        <div className="flex items-center gap-3 mb-4">
          <div className="flex items-center gap-1">
            {(["", "info", "warn", "error"] as const).map((s) => (
              <button
                key={s}
                onClick={() => setSeverity(s)}
                className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                  severity === s
                    ? "bg-cyan-500/20 text-cyan-400 border border-cyan-500/30"
                    : "text-slate-500 hover:text-slate-300 hover:bg-slate-700/40"
                }`}
              >
                {s === "" ? "All" : s}
              </button>
            ))}
          </div>

          <div className="flex-1 max-w-xs relative">
            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500 text-xs">⌕</span>
            <input
              type="text"
              value={eventFilter}
              onChange={(e) => handleEventInputChange(e.target.value)}
              placeholder="Filter by event code…"
              className="w-full bg-slate-800 border border-slate-600 rounded-lg pl-8 pr-3 py-1.5 text-slate-300 placeholder-slate-600 text-xs focus:outline-none focus:border-cyan-500 transition-colors"
            />
          </div>

          {total > 0 && (
            <span className="text-xs text-slate-600 ml-auto">
              Showing {total} event{total !== 1 ? "s" : ""}
            </span>
          )}
        </div>

        {/* Table */}
        <div className="bg-slate-800/40 border border-slate-700 rounded-xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-700 bg-slate-800/60">
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">Timestamp</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">Event Code</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">Actor</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">Severity</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">Resource</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">Metadata</th>
                </tr>
              </thead>
              <tbody>
                {isLoading ? (
                  <TableSkeleton />
                ) : allLogs.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-4 py-12 text-center text-slate-600 text-sm">
                      No audit events found
                    </td>
                  </tr>
                ) : (
                  allLogs.map((log) => (
                    <tr
                      key={log.id}
                      className="border-b border-slate-700/40 hover:bg-slate-700/20 transition-colors"
                    >
                      <td className="px-4 py-3 font-mono text-[11px] text-slate-500 whitespace-nowrap">
                        {timeFormat(log.created_at)}
                      </td>
                      <td className="px-4 py-3">
                        <span className="font-mono text-[11px] text-cyan-400/80 break-all">
                          {log.event_code}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        {log.actor_email ? (
                          <div>
                            <div className="text-slate-300 text-xs">{log.actor_name ?? log.actor_email}</div>
                            <div className="text-slate-600 text-[10px] font-mono">{log.actor_email}</div>
                          </div>
                        ) : (
                          <span className="text-slate-700 text-xs">system</span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex items-center gap-1.5 text-[10px] font-semibold px-2 py-0.5 rounded-full ${SEVERITY_STYLE[log.severity] ?? SEVERITY_STYLE.info}`}>
                          <span className={`w-1.5 h-1.5 rounded-full ${SEVERITY_DOT[log.severity] ?? SEVERITY_DOT.info}`} />
                          {log.severity}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        {log.resource_type ? (
                          <div className="text-slate-500 text-xs">
                            <span className="text-slate-400">{log.resource_type}</span>
                            {log.resource_id && (
                              <span className="text-slate-700 font-mono ml-1 text-[10px]">
                                {log.resource_id.slice(0, 8)}…
                              </span>
                            )}
                          </div>
                        ) : (
                          <span className="text-slate-700 text-xs">—</span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <MetaCell meta={log.metadata} />
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Load more */}
          {hasNextPage && (
            <div className="px-4 py-3 border-t border-slate-700/40 flex justify-center">
              <button
                onClick={() => fetchNextPage()}
                disabled={isFetchingNextPage}
                className="text-xs text-slate-500 hover:text-cyan-400 transition-colors disabled:opacity-50"
              >
                {isFetchingNextPage ? "Loading…" : "Load more events"}
              </button>
            </div>
          )}
        </div>

        {/* IP address note */}
        <p className="text-[10px] text-slate-700 mt-3">
          IP addresses shown are from the API server&apos;s perspective and may reflect internal Docker network addresses in development.
        </p>
      </div>
    </RBACGate>
  );
}
