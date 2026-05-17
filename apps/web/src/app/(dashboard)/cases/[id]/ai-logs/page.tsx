"use client";

import { use, useState } from "react";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { api, APIError } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { AICard, AILog } from "@/components/ai/AICard";

const WRITER_ROLES = ["analyst", "manager", "admin", "super_admin"];

interface ListMeta {
  total_count: number;
  has_more: boolean;
  next_cursor: string | null;
}

export default function AILogsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const canReview = WRITER_ROLES.includes(user?.role ?? "");

  const [statusFilter, setStatusFilter] = useState("");

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, error } =
    useInfiniteQuery({
      queryKey: ["ai-logs", id, statusFilter],
      queryFn: async ({ pageParam }) => {
        const params = new URLSearchParams({ limit: "20" });
        if (pageParam) params.set("cursor", pageParam as string);
        if (statusFilter) params.set("status", statusFilter);
        const res = await api.get<{ data: AILog[]; meta: ListMeta }>(`/api/v1/cases/${id}/ai/logs?${params}`);
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
    <div className="min-h-screen bg-[#0D1117] text-slate-100 p-6">
      <div className="max-w-4xl mx-auto space-y-6">
        {/* Navigation */}
        <div className="flex items-center gap-3 text-sm text-slate-400">
          <button onClick={() => router.push("/cases")} className="hover:text-cyan-400 transition-colors">← Cases</button>
          <div className="h-4 border-l border-slate-700" />
          <button onClick={() => router.push(`/cases/${id}`)} className="hover:text-cyan-400 transition-colors">Case</button>
          <div className="h-4 border-l border-slate-700" />
          <span className="text-slate-300">AI Logs</span>
          <div className="h-4 border-l border-slate-700" />
          <button onClick={() => router.push(`/cases/${id}/documents`)} className="hover:text-cyan-400 transition-colors">Documents →</button>
        </div>

        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-slate-100">AI Activity Log</h1>
            <p className="text-slate-400 text-sm mt-1">Every AI interaction for this case. Immutable audit trail.</p>
          </div>
        </div>

        {/* Filter */}
        <div className="flex items-center gap-3">
          <label className="text-xs text-slate-400">Filter by review status:</label>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="text-xs bg-slate-800 border border-slate-700 rounded px-2 py-1 text-slate-300 focus:outline-none focus:border-violet-500"
          >
            <option value="">All</option>
            <option value="pending">Pending Review</option>
            <option value="approved">Approved</option>
            <option value="edited">Edited</option>
            <option value="rejected">Rejected</option>
          </select>
        </div>

        {/* Loading skeleton */}
        {isLoading && (
          <div className="space-y-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-32 bg-slate-800/40 rounded-xl animate-pulse" />
            ))}
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-400">
            {error instanceof APIError ? error.message : "Failed to load AI logs"}
          </div>
        )}

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
                onReject={(logId) => {
                  const reason = window.prompt("Rejection reason:");
                  if (reason) rejectMutation.mutate({ logId, reason });
                }}
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
    </div>
  );
}
