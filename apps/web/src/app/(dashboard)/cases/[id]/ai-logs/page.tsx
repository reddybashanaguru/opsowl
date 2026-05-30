"use client";

import { use, useState } from "react";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { AICard, AILog } from "@/components/ai/AICard";
import { CaseSubNav } from "@/components/cases/CaseSubNav";
import { PageError } from "@/components/ui/PageError";

interface ListMeta {
  total_count: number;
  has_more: boolean;
  next_cursor: string | null;
}

export default function AILogsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const hasRole = useAuthStore((s) => s.hasRole);
  const canReview = hasRole("analyst");

  const [statusFilter, setStatusFilter] = useState("");
  const [rejectingLogId, setRejectingLogId] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState("");

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, error, refetch } =
    useInfiniteQuery({
      queryKey: ["ai-logs", id, statusFilter],
      queryFn: async ({ pageParam }) => {
        const params = new URLSearchParams({ limit: "20" });
        if (pageParam) params.set("cursor", pageParam as string);
        if (statusFilter) params.set("status", statusFilter);
        const res = await api.getList<AILog, ListMeta>(`/api/v1/cases/${id}/ai/logs?${params}`);
        return res;
      },
      initialPageParam: null as string | null,
      getNextPageParam: (last) => last.meta.has_more ? last.meta.next_cursor : undefined,
    });

  const allLogs: AILog[] = data?.pages.flatMap((p) => p.data) ?? [];

  const approveMutation = useMutation({
    mutationFn: (logId: string) => api.post(`/api/v1/ai/logs/${logId}/approve`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ai-logs", id] }),
  });

  const rejectMutation = useMutation({
    mutationFn: ({ logId, reason }: { logId: string; reason: string }) =>
      api.post(`/api/v1/ai/logs/${logId}/reject`, { reason }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ai-logs", id] }),
  });

  const editMutation = useMutation({
    mutationFn: ({ logId, edited_response }: { logId: string; edited_response: string }) =>
      api.put(`/api/v1/ai/logs/${logId}/edit`, { edited_response }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ai-logs", id] }),
  });

  return (
    <div className="p-6">
      <CaseSubNav caseId={id} activeTab="ai-logs" />

      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-white">AI Activity Log</h1>
          <p className="text-slate-400 text-sm mt-1">Every AI interaction for this case. Immutable audit trail.</p>
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="text-xs bg-slate-800 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-300 focus:outline-none focus:border-violet-500"
        >
          <option value="">All statuses</option>
          <option value="pending">Pending Review</option>
          <option value="approved">Approved</option>
          <option value="edited">Edited</option>
          <option value="rejected">Rejected</option>
        </select>
      </div>

      {/* Reject reason inline popover */}
      {rejectingLogId && (
        <div className="mb-4 bg-[#1e293b] border border-red-500/30 rounded-xl p-4">
          <p className="text-sm text-slate-300 mb-2 font-medium">Rejection reason</p>
          <textarea
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            rows={2}
            placeholder="Optional — describe why this response is being rejected"
            className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-red-400 resize-none"
          />
          <div className="flex gap-2 mt-2">
            <button
              onClick={() => {
                rejectMutation.mutate({ logId: rejectingLogId, reason: rejectReason });
                setRejectingLogId(null);
                setRejectReason("");
              }}
              className="text-xs bg-red-600 hover:bg-red-500 text-white px-3 py-1.5 rounded-lg transition-colors"
            >
              Confirm Rejection
            </button>
            <button
              onClick={() => { setRejectingLogId(null); setRejectReason(""); }}
              className="text-xs text-slate-400 hover:text-slate-200 px-3 py-1.5 rounded-lg border border-slate-600 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Loading skeleton */}
      {isLoading && (
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-32 bg-slate-800/40 rounded-xl animate-pulse" />
          ))}
        </div>
      )}

      {/* Error */}
      {error && <PageError error={error as Error | null} onRetry={() => refetch()} />}

      {/* Empty state */}
      {!isLoading && allLogs.length === 0 && (
        <div className="text-center py-16 text-slate-500">
          <div className="text-4xl mb-3">🤖</div>
          <p className="text-sm font-medium">No AI activity yet</p>
          <p className="text-xs mt-1">Open the AI Copilot panel to start asking questions.</p>
        </div>
      )}

      {/* Logs list */}
      {allLogs.length > 0 && (
        <div className="space-y-4">
          {allLogs.map((log) => (
            <AICard
              key={log.id}
              log={log}
              canReview={canReview}
              onApprove={(logId) => approveMutation.mutate(logId)}
              onReject={(logId) => { setRejectingLogId(logId); setRejectReason(""); }}
              onEdit={(logId, text) => editMutation.mutate({ logId, edited_response: text })}
            />
          ))}

          {hasNextPage && (
            <button
              onClick={() => fetchNextPage()}
              disabled={isFetchingNextPage}
              className="w-full py-2 text-sm text-slate-400 hover:text-violet-400 border border-slate-700/50 rounded-lg hover:border-slate-600 transition-colors disabled:opacity-50"
            >
              {isFetchingNextPage ? "Loading…" : "Load more"}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
