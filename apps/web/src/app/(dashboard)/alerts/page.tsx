"use client";

import { useState, useCallback, useRef, useEffect, useMemo, memo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { api, newIdempotencyKey } from "@/lib/api";
import {
  BellRing, CheckCircle, XCircle, RefreshCw,
  Copy, X as CloseIcon,
} from "lucide-react";
import { useKeyboardShortcuts } from "@/lib/hooks/useKeyboardShortcuts";

interface AlertRow {
  id: string;
  external_alert_id: string;
  alert_type: string;
  risk_score: number;
  risk_level: string;
  subject_name: string;
  subject_id_type: string;
  subject_id_number: string;
  account_number: string;
  amount: number;
  currency: string;
  transaction_date: string | null;
  description: string;
  source_system: string;
  status: string;
  linked_case_id: string | null;
  linked_case_number: string;
  created_at: string;
}

interface AlertsPage {
  alerts: AlertRow[];
  total: number;
  limit: number;
  offset: number;
  next_cursor?: string | null;
}

interface BulkResult {
  succeeded: number;
  failed: number;
  errors?: Array<{ alert_id: string; error: string }>;
}

const STATUS_OPTIONS = ["", "pending", "converted", "dismissed", "duplicate"];
const RISK_OPTIONS = ["", "critical", "high", "medium", "low"];

const RISK_COLORS: Record<string, string> = {
  critical: "bg-red-500/15 text-red-400 border-red-500/30",
  high:     "bg-orange-500/15 text-orange-400 border-orange-500/30",
  medium:   "bg-yellow-500/15 text-yellow-400 border-yellow-500/30",
  low:      "bg-slate-500/15 text-slate-400 border-slate-500/30",
};

const STATUS_COLORS: Record<string, string> = {
  pending:   "bg-cyan-500/15 text-cyan-400",
  converted: "bg-green-500/15 text-green-400",
  dismissed: "bg-slate-500/15 text-slate-400",
  duplicate: "bg-purple-500/15 text-purple-400",
  reviewing: "bg-blue-500/15 text-blue-400",
};

function isSelectable(status: string) {
  return status === "pending" || status === "reviewing";
}

function RiskBadge({ level }: { level: string }) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold uppercase border ${RISK_COLORS[level] ?? "bg-slate-700 text-slate-400 border-slate-600"}`}>
      {level}
    </span>
  );
}

function StatusBadge({ status }: { status: string }) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold capitalize ${STATUS_COLORS[status] ?? "bg-slate-700 text-slate-400"}`}>
      {status.replace("_", " ")}
    </span>
  );
}

function formatAmount(amount: number, currency: string) {
  if (!amount) return "—";
  return new Intl.NumberFormat(undefined, {
    style: "currency", currency: currency || "USD", notation: "compact",
  }).format(amount);
}

function ConvertModal({
  alert, onClose, onConverted,
}: { alert: AlertRow; onClose: () => void; onConverted: () => void }) {
  const [title, setTitle]           = useState(`[${alert.alert_type}] ${alert.subject_name}`);
  const [description, setDescription] = useState(alert.description);
  const [priority, setPriority]     = useState(
    alert.risk_level === "critical" ? "critical" : alert.risk_level === "high" ? "high" : "medium",
  );
  const [error, setError] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      api.idempotentPost(`/api/v1/alerts/${alert.id}/convert`, { title, description, priority }, newIdempotencyKey()),
    onSuccess: () => { onConverted(); onClose(); },
    onError: (err: unknown) => setError((err as Error).message || "Failed to convert alert"),
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] w-full max-w-lg p-6 shadow-2xl">
        <h2 className="text-slate-100 font-semibold text-lg mb-4">Convert Alert to Case</h2>
        {error && <div className="mb-3 text-red-400 text-sm">{error}</div>}
        <div className="space-y-3">
          <div>
            <label className="block text-xs text-slate-400 mb-1">Case Title</label>
            <input value={title} onChange={e => setTitle(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500" />
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Description</label>
            <textarea value={description} onChange={e => setDescription(e.target.value)} rows={3}
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500 resize-none" />
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Priority</label>
            <select value={priority} onChange={e => setPriority(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500">
              {["critical", "high", "medium", "low"].map(p => (
                <option key={p} value={p}>{p.charAt(0).toUpperCase() + p.slice(1)}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="flex gap-3 mt-5">
          <button onClick={onClose}
            className="flex-1 px-4 py-2 rounded-lg border border-slate-600 text-slate-300 text-sm hover:border-slate-500 transition-colors">
            Cancel
          </button>
          <button onClick={() => mutation.mutate()} disabled={mutation.isPending || !title.trim()}
            className="flex-1 px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium transition-colors disabled:opacity-50">
            {mutation.isPending ? "Converting…" : "Create Case"}
          </button>
        </div>
      </div>
    </div>
  );
}

function DismissModal({
  alertId, onClose, onDismissed,
}: { alertId: string; onClose: () => void; onDismissed: () => void }) {
  const [reason, setReason] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      api.idempotentPost(`/api/v1/alerts/${alertId}/dismiss`, { reason }, newIdempotencyKey()),
    onSuccess: () => { onDismissed(); onClose(); },
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] w-full max-w-md p-6 shadow-2xl">
        <h2 className="text-slate-100 font-semibold text-lg mb-4">Dismiss Alert</h2>
        <textarea value={reason} onChange={e => setReason(e.target.value)} rows={3}
          placeholder="Reason for dismissal (optional)"
          className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500 resize-none" />
        <div className="flex gap-3 mt-4">
          <button onClick={onClose}
            className="flex-1 px-4 py-2 rounded-lg border border-slate-600 text-slate-300 text-sm hover:border-slate-500 transition-colors">
            Cancel
          </button>
          <button onClick={() => mutation.mutate()} disabled={mutation.isPending}
            className="flex-1 px-4 py-2 rounded-lg bg-red-700 hover:bg-red-600 text-white text-sm font-medium transition-colors disabled:opacity-50">
            {mutation.isPending ? "Dismissing…" : "Dismiss"}
          </button>
        </div>
      </div>
    </div>
  );
}

const LIMIT = 25;

export default function AlertQueuePage() {
  const qc = useQueryClient();
  const [statusFilter,  setStatusFilter]  = useState("pending");
  const [riskFilter,    setRiskFilter]    = useState("");
  const [cursorStack,   setCursorStack]   = useState<(string | null)[]>([null]);
  const [page,          setPage]          = useState(0);
  const [convertAlert,  setConvertAlert]  = useState<AlertRow | null>(null);
  const [dismissId,     setDismissId]     = useState<string | null>(null);

  const [selectedIds,  setSelectedIds]  = useState<Set<string>>(new Set());
  const [bulkReason,   setBulkReason]   = useState("");
  const [bulkFeedback, setBulkFeedback] = useState<string | null>(null);

  const [focusedIndex, setFocusedIndex] = useState(-1);
  const focusedRowRef = useRef<HTMLTableRowElement | null>(null);

  const cursor = cursorStack[page];

  const resetPagination = useCallback(() => {
    setCursorStack([null]);
    setPage(0);
    setSelectedIds(new Set());
  }, []);

  const queryKey = useMemo(
    () => ["alerts", statusFilter, riskFilter, cursor],
    [statusFilter, riskFilter, cursor],
  );

  const { data, isFetching, refetch } = useQuery<AlertsPage>({
    queryKey,
    queryFn: () => {
      const params = new URLSearchParams({ limit: String(LIMIT) });
      if (statusFilter) params.set("status", statusFilter);
      if (riskFilter)   params.set("risk_level", riskFilter);
      if (cursor)       params.set("cursor", cursor);
      return api.get<AlertsPage>(`/api/v1/alerts?${params}`);
    },
    refetchInterval: 30_000,
  });

  const invalidate = useCallback(() => {
    qc.invalidateQueries({ queryKey: ["alerts"] });
    resetPagination();
  }, [qc, resetPagination]);

  const clearSelection = () => {
    setSelectedIds(new Set());
    setBulkReason("");
    setBulkFeedback(null);
  };

  const alerts = data?.alerts ?? [];
  const total  = data?.total ?? 0;

  // Keep focusedIndex in bounds when the list changes
  useEffect(() => {
    if (focusedIndex >= alerts.length) setFocusedIndex(alerts.length - 1);
  }, [alerts.length, focusedIndex]);

  // Scroll focused row into view
  useEffect(() => {
    focusedRowRef.current?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }, [focusedIndex]);

  useKeyboardShortcuts({
    j: () => setFocusedIndex(i => Math.min(i + 1, alerts.length - 1)),
    k: () => setFocusedIndex(i => Math.max(i - 1, 0)),
    x: () => {
      const a = alerts[focusedIndex];
      if (a && isSelectable(a.status)) toggleOne(a.id);
    },
    d: () => {
      const a = alerts[focusedIndex];
      if (a && isSelectable(a.status)) setDismissId(a.id);
    },
    c: () => {
      const a = alerts[focusedIndex];
      if (a && isSelectable(a.status)) setConvertAlert(a);
    },
    Escape: () => {
      if (convertAlert) { setConvertAlert(null); return; }
      if (dismissId)    { setDismissId(null);    return; }
      clearSelection();
      setFocusedIndex(-1);
    },
  }, /* disable when a modal is open */ !convertAlert && !dismissId);

  const selectableAlerts = alerts.filter(a => isSelectable(a.status));
  const allSelectableSelected =
    selectableAlerts.length > 0 && selectableAlerts.every(a => selectedIds.has(a.id));
  const someSelected = selectedIds.size > 0;

  function toggleAll() {
    if (allSelectableSelected) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(selectableAlerts.map(a => a.id)));
    }
  }

  function toggleOne(id: string) {
    setSelectedIds(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  }

  const bulkMutation = useMutation({
    mutationFn: ({ action, ids, reason }: { action: string; ids: string[]; reason: string }) =>
      api.post<BulkResult>("/api/v1/alerts/bulk", { action, alert_ids: ids, reason }),
    onSuccess: (result) => {
      const msg = result.failed > 0
        ? `Done: ${result.succeeded} succeeded, ${result.failed} failed.`
        : `${result.succeeded} alert${result.succeeded !== 1 ? "s" : ""} updated.`;
      setBulkFeedback(msg);
      invalidate();
    },
    onError: (err: unknown) => {
      setBulkFeedback((err as Error).message || "Bulk action failed");
    },
  });

  function runBulk(action: "dismiss" | "mark_duplicate") {
    bulkMutation.mutate({ action, ids: Array.from(selectedIds), reason: bulkReason });
  }

  return (
    <div className="p-6 pb-28">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <div className="flex items-center gap-2.5">
            <BellRing className="w-5 h-5 text-cyan-400" />
            <h1 className="text-xl font-bold text-slate-100">Alert Queue</h1>
            {total > 0 && (
              <span className="px-2 py-0.5 bg-cyan-500/15 text-cyan-400 rounded-full text-xs font-semibold">
                {total}
              </span>
            )}
          </div>
          <p className="text-slate-400 text-sm mt-1">Incoming alerts from connected TMS systems</p>
        </div>
        <button onClick={() => refetch()}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-slate-600 text-slate-300 text-sm hover:border-slate-500 transition-colors">
          <RefreshCw size={13} className={isFetching ? "animate-spin" : ""} />
          Refresh
        </button>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3 mb-5 flex-wrap">
        <select value={statusFilter} onChange={e => { setStatusFilter(e.target.value); resetPagination(); }}
          className="bg-slate-800 border border-slate-700 text-slate-300 text-sm rounded-lg px-3 py-1.5 focus:outline-none focus:border-cyan-500">
          <option value="">All statuses</option>
          {STATUS_OPTIONS.filter(Boolean).map(s => (
            <option key={s} value={s}>{s.charAt(0).toUpperCase() + s.slice(1)}</option>
          ))}
        </select>
        <select value={riskFilter} onChange={e => { setRiskFilter(e.target.value); resetPagination(); }}
          className="bg-slate-800 border border-slate-700 text-slate-300 text-sm rounded-lg px-3 py-1.5 focus:outline-none focus:border-cyan-500">
          <option value="">All risk levels</option>
          {RISK_OPTIONS.filter(Boolean).map(r => (
            <option key={r} value={r}>{r.charAt(0).toUpperCase() + r.slice(1)}</option>
          ))}
        </select>
      </div>

      {/* Table */}
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] overflow-hidden">
        {isFetching && !data ? (
          <div className="flex items-center justify-center h-48 text-slate-500 text-sm">
            Loading alerts…
          </div>
        ) : alerts.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-48 text-slate-500">
            <CheckCircle className="w-10 h-10 mb-3 text-slate-600" />
            <p className="text-sm font-medium">No alerts found</p>
            <p className="text-xs text-slate-600 mt-1">
              {statusFilter === "pending" ? "Your queue is clear" : "Try adjusting the filters"}
            </p>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-700 bg-slate-800/40">
                {/* Select-all checkbox */}
                <th className="px-4 py-3 w-8">
                  {selectableAlerts.length > 0 && (
                    <input
                      type="checkbox"
                      checked={allSelectableSelected}
                      onChange={toggleAll}
                      className="accent-cyan-500 cursor-pointer"
                      title="Select all pending alerts on this page"
                    />
                  )}
                </th>
                <th className="text-left px-4 py-3 text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Type / Source</th>
                <th className="text-left px-4 py-3 text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Subject</th>
                <th className="text-left px-4 py-3 text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Risk</th>
                <th className="text-left px-4 py-3 text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Amount</th>
                <th className="text-left px-4 py-3 text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Status</th>
                <th className="text-left px-4 py-3 text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Received</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody>
              {alerts.map((alert, i) => {
                const selectable = isSelectable(alert.status);
                const checked    = selectedIds.has(alert.id);
                const focused    = focusedIndex === i;
                return (
                  <tr
                    key={alert.id}
                    ref={focused ? (el) => { focusedRowRef.current = el; } : undefined}
                    onClick={() => setFocusedIndex(i)}
                    className={`border-b border-slate-700/50 transition-colors cursor-default ${
                      focused  ? "bg-cyan-900/20 ring-1 ring-inset ring-cyan-500/30" :
                      checked  ? "bg-cyan-900/10" : "hover:bg-slate-700/20"
                    } ${i === alerts.length - 1 ? "border-b-0" : ""}`}
                  >
                    <td className="px-4 py-3 w-8">
                      {selectable && (
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() => toggleOne(alert.id)}
                          className="accent-cyan-500 cursor-pointer"
                        />
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="font-medium text-slate-200 text-xs">
                        {alert.alert_type.replace(/_/g, " ")}
                      </div>
                      <div className="text-slate-500 text-[11px] mt-0.5">{alert.source_system}</div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="text-slate-200 text-xs font-medium">{alert.subject_name || "—"}</div>
                      {alert.subject_id_number && (
                        <div className="text-slate-500 text-[11px] mt-0.5">
                          {alert.subject_id_type}: {alert.subject_id_number}
                        </div>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <RiskBadge level={alert.risk_level} />
                        <span className="text-slate-400 text-xs">{alert.risk_score}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-slate-300 text-xs font-mono">
                      {formatAmount(alert.amount, alert.currency)}
                    </td>
                    <td className="px-4 py-3">
                      {alert.linked_case_number ? (
                        <Link href={`/cases/${alert.linked_case_id}`}
                          className="flex items-center gap-1 text-cyan-400 hover:text-cyan-300 text-xs transition-colors">
                          <StatusBadge status={alert.status} />
                          <span className="ml-1">→ {alert.linked_case_number}</span>
                        </Link>
                      ) : (
                        <StatusBadge status={alert.status} />
                      )}
                    </td>
                    <td className="px-4 py-3 text-slate-500 text-xs">
                      {new Date(alert.created_at).toLocaleDateString(undefined, {
                        day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit",
                      })}
                    </td>
                    <td className="px-4 py-3">
                      {selectable && (
                        <div className="flex items-center gap-2">
                          <button onClick={() => setConvertAlert(alert)}
                            className="p-1.5 rounded-lg bg-cyan-600/20 text-cyan-400 hover:bg-cyan-600/40 transition-colors"
                            title="Convert to Case">
                            <CheckCircle size={14} />
                          </button>
                          <button onClick={() => setDismissId(alert.id)}
                            className="p-1.5 rounded-lg bg-red-600/20 text-red-400 hover:bg-red-600/40 transition-colors"
                            title="Dismiss">
                            <XCircle size={14} />
                          </button>
                        </div>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* Pagination */}
      {total > LIMIT && (
        <div className="flex items-center justify-between mt-4">
          <span className="text-slate-500 text-xs">
            Showing {(page * LIMIT + 1).toLocaleString()}–{(page * LIMIT + alerts.length).toLocaleString()} of {total.toLocaleString()}
          </span>
          <div className="flex gap-2">
            <button disabled={page === 0} onClick={() => setPage(p => p - 1)}
              className="px-3 py-1.5 rounded-lg border border-slate-700 text-slate-300 text-xs disabled:opacity-40 hover:border-slate-500 transition-colors">
              Previous
            </button>
            <button
              disabled={!data?.next_cursor}
              onClick={() => {
                const nc = data?.next_cursor;
                if (!nc) return;
                setCursorStack(prev => { const s = [...prev]; s[page + 1] = nc; return s; });
                setPage(p => p + 1);
              }}
              className="px-3 py-1.5 rounded-lg border border-slate-700 text-slate-300 text-xs disabled:opacity-40 hover:border-slate-500 transition-colors">
              Next
            </button>
          </div>
        </div>
      )}

      {/* ── Floating bulk-action bar ─────────────────────────────────────────── */}
      {someSelected && (
        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-40 flex items-center gap-3
          bg-[#0f1c2e] border border-cyan-500/30 rounded-2xl shadow-2xl px-4 py-3
          ring-1 ring-cyan-500/10 backdrop-blur-sm max-w-2xl w-[calc(100%-3rem)]">
          {/* Count + clear */}
          <div className="flex items-center gap-2 flex-shrink-0">
            <span className="text-cyan-400 font-semibold text-sm">{selectedIds.size}</span>
            <span className="text-slate-400 text-sm">selected</span>
            <button onClick={clearSelection}
              className="p-1 rounded text-slate-500 hover:text-slate-300 transition-colors ml-1"
              title="Clear selection">
              <CloseIcon size={13} />
            </button>
          </div>

          <div className="h-5 w-px bg-slate-700" />

          {/* Optional reason */}
          <input
            value={bulkReason}
            onChange={e => setBulkReason(e.target.value)}
            placeholder="Reason (optional)"
            className="flex-1 bg-slate-800 border border-slate-700 rounded-lg px-3 py-1.5 text-slate-200 text-xs
              focus:outline-none focus:border-cyan-500 placeholder:text-slate-600 min-w-0"
          />

          {/* Actions */}
          <button
            onClick={() => runBulk("dismiss")}
            disabled={bulkMutation.isPending}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-red-700/80 hover:bg-red-600
              text-white text-xs font-medium transition-colors disabled:opacity-50 flex-shrink-0">
            <XCircle size={13} />
            Dismiss
          </button>
          <button
            onClick={() => runBulk("mark_duplicate")}
            disabled={bulkMutation.isPending}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-purple-700/80 hover:bg-purple-600
              text-white text-xs font-medium transition-colors disabled:opacity-50 flex-shrink-0">
            <Copy size={13} />
            Mark Duplicate
          </button>

          {/* Inline feedback */}
          {bulkFeedback && (
            <span className="text-xs text-slate-400 flex-shrink-0">{bulkFeedback}</span>
          )}
          {bulkMutation.isPending && (
            <RefreshCw size={13} className="animate-spin text-cyan-400 flex-shrink-0" />
          )}
        </div>
      )}

      {/* Modals */}
      {convertAlert && (
        <ConvertModal
          alert={convertAlert}
          onClose={() => setConvertAlert(null)}
          onConverted={invalidate}
        />
      )}
      {dismissId && (
        <DismissModal
          alertId={dismissId}
          onClose={() => setDismissId(null)}
          onDismissed={invalidate}
        />
      )}
    </div>
  );
}
