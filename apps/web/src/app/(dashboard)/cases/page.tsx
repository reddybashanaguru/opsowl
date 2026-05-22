"use client";

import { useState, useMemo, useCallback } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense } from "react";
import { api, APIError, newIdempotencyKey } from "@/lib/api";
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

interface UserOption {
  id: string;
  full_name: string;
  email: string;
  role: string;
}

interface BulkResult {
  updated: number;
  failed: number;
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

const BULK_STATUS_OPTIONS = ["open", "in_progress", "pending_review", "escalated", "closed"];

// --- Bulk modals ---

function BulkStatusModal({
  count,
  onConfirm,
  onClose,
}: {
  count: number;
  onConfirm: (status: string) => void;
  onClose: () => void;
}) {
  const [selected, setSelected] = useState("open");
  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-[#1e293b] border border-slate-700 rounded-2xl w-full max-w-sm p-6 shadow-2xl">
        <h2 className="text-white font-semibold text-lg mb-1">Change Status</h2>
        <p className="text-slate-400 text-sm mb-5">
          Apply to <span className="text-cyan-400 font-medium">{count}</span> selected case{count !== 1 ? "s" : ""}.
        </p>
        <label className="block text-xs text-slate-400 mb-1">New Status</label>
        <select
          value={selected}
          onChange={(e) => setSelected(e.target.value)}
          className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-cyan-500 mb-5"
        >
          {BULK_STATUS_OPTIONS.map((s) => (
            <option key={s} value={s}>{STATUS_LABELS[s] ?? s}</option>
          ))}
        </select>
        <div className="flex gap-3">
          <button
            onClick={() => onConfirm(selected)}
            className="flex-1 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium rounded transition-colors"
          >
            Apply
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm border border-slate-700 rounded transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

function BulkAssignModal({
  count,
  onConfirm,
  onClose,
}: {
  count: number;
  onConfirm: (userId: string) => void;
  onClose: () => void;
}) {
  const [selected, setSelected] = useState("");

  const { data: users } = useQuery({
    queryKey: ["users-list"],
    queryFn: () =>
      api.get<{ users: UserOption[] }>("/api/v1/users?limit=100").then((d) => d.users ?? []),
    staleTime: 60_000,
  });

  const assignable = (users ?? []).filter((u) =>
    ["analyst", "manager", "admin", "super_admin"].includes(u.role)
  );

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-[#1e293b] border border-slate-700 rounded-2xl w-full max-w-sm p-6 shadow-2xl">
        <h2 className="text-white font-semibold text-lg mb-1">Assign Cases</h2>
        <p className="text-slate-400 text-sm mb-5">
          Reassign <span className="text-cyan-400 font-medium">{count}</span> case{count !== 1 ? "s" : ""} to:
        </p>
        <label className="block text-xs text-slate-400 mb-1">Assignee</label>
        <select
          value={selected}
          onChange={(e) => setSelected(e.target.value)}
          className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-cyan-500 mb-5"
        >
          <option value="">— Select user —</option>
          {assignable.map((u) => (
            <option key={u.id} value={u.id}>
              {u.full_name} ({u.role})
            </option>
          ))}
        </select>
        <div className="flex gap-3">
          <button
            onClick={() => onConfirm(selected)}
            disabled={!selected}
            className="flex-1 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm font-medium rounded transition-colors"
          >
            Assign
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm border border-slate-700 rounded transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

function BulkDeleteConfirm({
  count,
  onConfirm,
  onClose,
}: {
  count: number;
  onConfirm: () => void;
  onClose: () => void;
}) {
  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-[#1e293b] border border-red-500/30 rounded-2xl w-full max-w-sm p-6 shadow-2xl">
        <h2 className="text-white font-semibold text-lg mb-1">Delete Cases</h2>
        <p className="text-slate-300 text-sm mb-1">
          Permanently delete <span className="text-red-400 font-semibold">{count}</span> case{count !== 1 ? "s" : ""}?
        </p>
        <p className="text-slate-500 text-xs mb-5">This cannot be undone. All attached documents and tasks will also be removed.</p>
        <div className="flex gap-3">
          <button
            onClick={onConfirm}
            className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-500 text-white text-sm font-medium rounded transition-colors"
          >
            Delete {count} case{count !== 1 ? "s" : ""}
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm border border-slate-700 rounded transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

// --- Floating action bar ---

function BulkActionBar({
  count,
  onStatus,
  onAssign,
  onDelete,
  onClear,
  isLoading,
}: {
  count: number;
  onStatus: () => void;
  onAssign: () => void;
  onDelete: () => void;
  onClear: () => void;
  isLoading: boolean;
}) {
  return (
    <div
      data-testid="bulk-action-bar"
      className="fixed bottom-6 left-1/2 -translate-x-1/2 flex items-center gap-3 bg-[#1e293b] border border-slate-600 rounded-2xl px-5 py-3 shadow-2xl shadow-black/40 z-40 animate-in slide-in-from-bottom-4 duration-200"
    >
      <span className="text-cyan-400 font-semibold text-sm tabular-nums mr-1">
        {count} selected
      </span>
      <div className="w-px h-5 bg-slate-600" />
      <button
        onClick={onStatus}
        disabled={isLoading}
        className="text-slate-200 hover:text-white text-sm px-3 py-1.5 rounded hover:bg-slate-700 transition-colors disabled:opacity-40"
      >
        Change Status
      </button>
      <button
        onClick={onAssign}
        disabled={isLoading}
        className="text-slate-200 hover:text-white text-sm px-3 py-1.5 rounded hover:bg-slate-700 transition-colors disabled:opacity-40"
      >
        Reassign
      </button>
      <button
        onClick={onDelete}
        disabled={isLoading}
        className="text-red-400 hover:text-red-300 text-sm px-3 py-1.5 rounded hover:bg-red-500/10 transition-colors disabled:opacity-40"
      >
        Delete
      </button>
      <div className="w-px h-5 bg-slate-600" />
      <button
        onClick={onClear}
        className="text-slate-500 hover:text-slate-300 text-sm"
      >
        ✕
      </button>
      {isLoading && (
        <span className="w-4 h-4 border border-cyan-400 border-t-transparent rounded-full animate-spin ml-1" />
      )}
    </div>
  );
}

// --- Skeleton row ---

function SkeletonRow({ withCheckbox }: { withCheckbox?: boolean }) {
  const widths = withCheckbox ? [20, 220, 80, 80, 120, 100, 80] : [220, 80, 80, 120, 100, 80];
  return (
    <tr className="border-b border-slate-700/50">
      {widths.map((w, i) => (
        <td key={i} className="px-4 py-3">
          <div className="h-4 bg-slate-700/60 animate-pulse rounded" style={{ width: w }} />
        </td>
      ))}
    </tr>
  );
}

// --- Notification toast ---

function Toast({ message, type }: { message: string; type: "success" | "error" }) {
  return (
    <div className={`fixed top-5 right-5 z-50 px-4 py-3 rounded-xl border text-sm font-medium shadow-xl animate-in slide-in-from-top-3 duration-200 ${
      type === "success"
        ? "bg-emerald-500/15 border-emerald-500/40 text-emerald-300"
        : "bg-red-500/15 border-red-500/40 text-red-300"
    }`}>
      {message}
    </div>
  );
}

// --- Main cases content ---

function CasesContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const cfg = useConfig();
  const qc = useQueryClient();

  const [status, setStatus] = useState(searchParams.get("status") ?? "");
  const [priority, setPriority] = useState(searchParams.get("priority") ?? "");
  const [search, setSearch] = useState("");

  // Bulk selection
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [bulkModal, setBulkModal] = useState<"status" | "assign" | "delete" | null>(null);
  const [toast, setToast] = useState<{ message: string; type: "success" | "error" } | null>(null);

  function showToast(message: string, type: "success" | "error") {
    setToast({ message, type });
    setTimeout(() => setToast(null), 3000);
  }

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
      refetchOnWindowFocus: true,
      staleTime: 30_000,
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

  // Checkbox helpers
  const allVisibleIds = filtered.map((c) => c.id);
  const allSelected = allVisibleIds.length > 0 && allVisibleIds.every((id) => selected.has(id));
  const someSelected = allVisibleIds.some((id) => selected.has(id));

  function toggleAll() {
    if (allSelected) {
      setSelected(new Set());
    } else {
      setSelected(new Set(allVisibleIds));
    }
  }

  function toggleOne(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  // Shared optimistic-update helper: patches infinite-query pages in place
  type InfData = { pages: { data: CaseRow[]; meta: ListMeta }[] };
  function patchCasesCache(patcher: (c: CaseRow) => CaseRow) {
    qc.setQueriesData<InfData>({ queryKey: ["cases"] }, (old) => {
      if (!old) return old;
      return { ...old, pages: old.pages.map((pg) => ({ ...pg, data: pg.data.map(patcher) })) };
    });
  }
  function removeCasesFromCache(ids: Set<string>) {
    qc.setQueriesData<InfData>({ queryKey: ["cases"] }, (old) => {
      if (!old) return old;
      return { ...old, pages: old.pages.map((pg) => ({ ...pg, data: pg.data.filter((c) => !ids.has(c.id)) })) };
    });
  }

  // Bulk mutations
  const bulkStatusMutation = useMutation({
    mutationFn: (newStatus: string) =>
      api.idempotentPost<BulkResult>("/api/v1/cases/bulk/status", { ids: Array.from(selected), status: newStatus }, newIdempotencyKey()),
    onMutate: async (newStatus) => {
      await qc.cancelQueries({ queryKey: ["cases"] });
      const snap = selected;
      patchCasesCache((c) => (snap.has(c.id) ? { ...c, status: newStatus } : c));
      return { snap };
    },
    onSuccess: (res) => {
      showToast(`Updated ${res.updated} case${res.updated !== 1 ? "s" : ""}${res.failed ? ` (${res.failed} failed)` : ""}`, "success");
      setSelected(new Set());
      qc.invalidateQueries({ queryKey: ["cases"] });
    },
    onError: (e, _, ctx) => {
      if (ctx) patchCasesCache((c) => c); // noop — full refetch below restores truth
      qc.invalidateQueries({ queryKey: ["cases"] });
      showToast(e instanceof APIError ? e.message : "Bulk status change failed", "error");
    },
  });

  const bulkAssignMutation = useMutation({
    mutationFn: (userId: string) =>
      api.idempotentPost<BulkResult>("/api/v1/cases/bulk/assign", { ids: Array.from(selected), user_id: userId }, newIdempotencyKey()),
    onSuccess: (res) => {
      showToast(`Assigned ${res.updated} case${res.updated !== 1 ? "s" : ""}`, "success");
      setSelected(new Set());
      qc.invalidateQueries({ queryKey: ["cases"] });
    },
    onError: (e) => showToast(e instanceof APIError ? e.message : "Bulk assign failed", "error"),
  });

  const bulkDeleteMutation = useMutation({
    mutationFn: () =>
      api.idempotentPost<BulkResult>("/api/v1/cases/bulk/delete", { ids: Array.from(selected) }, newIdempotencyKey()),
    onMutate: async () => {
      await qc.cancelQueries({ queryKey: ["cases"] });
      const snap = selected;
      removeCasesFromCache(snap);
      return { snap };
    },
    onSuccess: (res) => {
      showToast(`Deleted ${res.updated} case${res.updated !== 1 ? "s" : ""}`, "success");
      setSelected(new Set());
      qc.invalidateQueries({ queryKey: ["cases"] });
    },
    onError: (e) => {
      qc.invalidateQueries({ queryKey: ["cases"] });
      showToast(e instanceof APIError ? e.message : "Bulk delete failed", "error");
    },
  });

  const bulkPending = bulkStatusMutation.isPending || bulkAssignMutation.isPending || bulkDeleteMutation.isPending;

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {toast && <Toast message={toast.message} type={toast.type} />}

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

        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          className="bg-[#1e293b] border border-slate-600 text-slate-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
        >
          {STATUSES.map((s) => (
            <option key={s} value={s}>{STATUS_LABELS[s] ?? s}</option>
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
              <th className="px-4 py-3 w-8">
                <input
                  type="checkbox"
                  aria-label="Select all"
                  checked={allSelected}
                  ref={(el) => { if (el) el.indeterminate = someSelected && !allSelected; }}
                  onChange={toggleAll}
                  className="accent-cyan-500 cursor-pointer"
                />
              </th>
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
              [...Array(8)].map((_, i) => <SkeletonRow key={i} withCheckbox />)
            ) : filtered.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-4 py-16 text-center">
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
              filtered.map((c) => {
                const isChecked = selected.has(c.id);
                return (
                  <tr
                    key={c.id}
                    className={`border-b border-slate-700/40 hover:bg-slate-700/25 transition-colors ${
                      isChecked ? "bg-cyan-500/5" : ""
                    }`}
                  >
                    <td className="px-4 py-3 w-8" onClick={(e) => e.stopPropagation()}>
                      <input
                        type="checkbox"
                        aria-label={`Select ${c.case_number}`}
                        checked={isChecked}
                        onChange={() => toggleOne(c.id)}
                        className="accent-cyan-500 cursor-pointer"
                      />
                    </td>
                    <td
                      className="px-4 py-3 max-w-xs cursor-pointer"
                      onClick={() => router.push(`/cases/${c.id}`)}
                    >
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
                    <td className="px-4 py-3 cursor-pointer" onClick={() => router.push(`/cases/${c.id}`)}>
                      <StatusBadge status={c.status} />
                    </td>
                    <td className="px-4 py-3 cursor-pointer" onClick={() => router.push(`/cases/${c.id}`)}>
                      <PriorityBadge priority={c.priority} />
                    </td>
                    <td className="px-4 py-3 text-slate-400 text-sm cursor-pointer" onClick={() => router.push(`/cases/${c.id}`)}>
                      {c.assigned_to?.full_name ?? <span className="text-slate-600 italic">Unassigned</span>}
                    </td>
                    <td className="px-4 py-3 cursor-pointer" onClick={() => router.push(`/cases/${c.id}`)}>
                      <SLACountdown breachAt={c.sla_breach_at} breached={c.sla_breached} />
                    </td>
                    <td className="px-4 py-3 text-slate-500 text-xs tabular-nums cursor-pointer" onClick={() => router.push(`/cases/${c.id}`)}>
                      {new Date(c.created_at).toLocaleDateString("en-IN", { day: "2-digit", month: "short", year: "numeric" })}
                    </td>
                  </tr>
                );
              })
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

      {/* Floating bulk action bar */}
      {selected.size > 0 && (
        <BulkActionBar
          count={selected.size}
          onStatus={() => setBulkModal("status")}
          onAssign={() => setBulkModal("assign")}
          onDelete={() => setBulkModal("delete")}
          onClear={() => setSelected(new Set())}
          isLoading={bulkPending}
        />
      )}

      {/* Bulk modals */}
      {bulkModal === "status" && (
        <BulkStatusModal
          count={selected.size}
          onClose={() => setBulkModal(null)}
          onConfirm={(s) => { setBulkModal(null); bulkStatusMutation.mutate(s); }}
        />
      )}
      {bulkModal === "assign" && (
        <BulkAssignModal
          count={selected.size}
          onClose={() => setBulkModal(null)}
          onConfirm={(uid) => { setBulkModal(null); bulkAssignMutation.mutate(uid); }}
        />
      )}
      {bulkModal === "delete" && (
        <BulkDeleteConfirm
          count={selected.size}
          onClose={() => setBulkModal(null)}
          onConfirm={() => { setBulkModal(null); bulkDeleteMutation.mutate(); }}
        />
      )}
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
