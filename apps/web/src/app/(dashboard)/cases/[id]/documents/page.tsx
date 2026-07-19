"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { useConfig } from "@/lib/hooks/useConfig";
import { DocumentCard, Document } from "@/components/documents/DocumentCard";
import { DocumentUploadZone } from "@/components/documents/DocumentUploadZone";
import { PageError } from "@/components/ui/PageError";
import { useCaseSummary } from "@/lib/hooks/useCaseSummary";

interface ListMeta {
  total_count: number;
  has_more: boolean;
  next_cursor: string | null;
}

export default function CaseDocumentsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const queryClient = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const hasRole = useAuthStore((s) => s.hasRole);
  const cfg = useConfig();

  const caseData = useCaseSummary(id);

  const [statusFilter, setStatusFilter] = useState<string>("");
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);

  const isCaseClosed = caseData?.status === "closed";
  const canUpload = hasRole("analyst") && !isCaseClosed;

  // ── List documents ─────────────────────────────────────────────────────────
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, isError, error, refetch } =
    useInfiniteQuery({
      queryKey: ["documents", id, statusFilter],
      queryFn: async ({ pageParam }) => {
        const params = new URLSearchParams({ limit: "20" });
        if (pageParam) params.set("cursor", pageParam as string);
        if (statusFilter) params.set("status", statusFilter);
        return api.getList<Document, ListMeta>(
          `/api/v1/cases/${id}/documents?${params}`,
        );
      },
      initialPageParam: null as string | null,
      getNextPageParam: (last) =>
        last.meta.has_more ? last.meta.next_cursor : undefined,
      retry: (failureCount, err) => {
        if (err instanceof APIError && err.status === 404) return false;
        return failureCount < 2;
      },
      // Poll every 5s while any document is still being extracted — stops automatically once all settle.
      refetchInterval: (query) => {
        const docs = query.state.data?.pages.flatMap((p) => p.data) ?? [];
        const hasInFlight = docs.some(
          (d) => d.processing_status === "processing" || d.processing_status === "pending",
        );
        return hasInFlight ? 5_000 : false;
      },
    });

  const seen = new Set<string>();
  const allDocs = (data?.pages.flatMap((p) => p.data) ?? []).filter((doc) => {
    if (seen.has(doc.id)) return false;
    seen.add(doc.id);
    return true;
  });

  // ── Upload ─────────────────────────────────────────────────────────────────
  const handleUpload = async (file: File) => {
    setUploadError(null);
    setUploading(true);
    try {
      const formData = new FormData();
      formData.append("file", file);
      await api.upload<Document>(`/api/v1/cases/${id}/documents`, formData);
      queryClient.invalidateQueries({ queryKey: ["documents", id] });
    } catch (err) {
      const msg = err instanceof APIError ? err.message : "Upload failed";
      setUploadError(msg);
    } finally {
      setUploading(false);
    }
  };

  // ── Delete ─────────────────────────────────────────────────────────────────
  const deleteMutation = useMutation({
    mutationFn: (docId: string) =>
      api.delete(`/api/v1/cases/${id}/documents/${docId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["documents", id] });
    },
  });

  // ── Retry ──────────────────────────────────────────────────────────────────
  const retryMutation = useMutation({
    mutationFn: (docId: string) =>
      api.post(`/api/v1/cases/${id}/documents/${docId}/retry`, {}),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["documents", id] });
    },
  });

  // ── Render ─────────────────────────────────────────────────────────────────
  return (
    <div className="px-6 pb-6">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-xl font-bold text-white">{cfg.documents_label}</h1>
        <p className="text-slate-500 text-sm mt-0.5">
          Upload files to extract intelligence and feed the AI copilot.
        </p>
      </div>

      {/* Upload zone */}
      {canUpload && (
        <div className="mb-4">
          <DocumentUploadZone onUpload={handleUpload} uploading={uploading} />
          {uploadError && (
            <p className="text-xs text-red-400 mt-1.5">{uploadError}</p>
          )}
        </div>
      )}

      {/* List header row: count + filter */}
      <div className="flex items-center justify-between mb-3">
        <p className="text-xs text-slate-500">
          {isLoading ? "" : `${allDocs.length} document${allDocs.length !== 1 ? "s" : ""}`}
        </p>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="text-xs bg-slate-800 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-300 focus:outline-none focus:border-cyan-500"
        >
          <option value="">All statuses</option>
          <option value="pending">Pending</option>
          <option value="processing">Processing</option>
          <option value="completed">Completed</option>
          <option value="failed">Failed</option>
        </select>
      </div>

      {/* List */}
      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-20 bg-slate-800/40 rounded-lg animate-pulse" />
          ))}
        </div>
      )}

      {isError && (
        <PageError error={error as Error | null} onRetry={() => refetch()} />
      )}

      {!isLoading && allDocs.length === 0 && (
        <div className="text-center py-12 text-slate-500">
          <div className="text-3xl mb-2">📂</div>
          <p className="text-sm font-medium">No documents yet</p>
          <p className="text-xs mt-1">
            {canUpload ? "Upload a file above to get started." : "No files have been uploaded to this case."}
          </p>
        </div>
      )}

      {allDocs.length > 0 && (
        <div className="space-y-2.5">
          {allDocs.map((doc) => (
            <DocumentCard
              key={doc.id}
              document={doc}
              onDelete={canUpload ? (docId) => deleteMutation.mutate(docId) : undefined}
              onRetry={canUpload ? (docId) => retryMutation.mutate(docId) : undefined}
              retrying={retryMutation.isPending && retryMutation.variables === doc.id}
            />
          ))}

          {hasNextPage && (
            <button
              onClick={() => fetchNextPage()}
              disabled={isFetchingNextPage}
              className="w-full py-2 text-sm text-slate-400 hover:text-cyan-400 border border-slate-700/50 rounded-lg hover:border-slate-600 transition-colors disabled:opacity-50"
            >
              {isFetchingNextPage ? "Loading…" : "Load more"}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
