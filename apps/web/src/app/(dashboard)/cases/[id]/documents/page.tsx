"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { DocumentCard, Document } from "@/components/documents/DocumentCard";
import { DocumentUploadZone } from "@/components/documents/DocumentUploadZone";

interface ListMeta {
  total_count: number;
  has_more: boolean;
  next_cursor: string | null;
}

const WRITER_ROLES = ["analyst", "manager", "admin", "super_admin"];

export default function CaseDocumentsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const queryClient = useQueryClient();
  const user = useAuthStore((s) => s.user);

  const [statusFilter, setStatusFilter] = useState<string>("");
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);

  const canUpload = WRITER_ROLES.includes(user?.role ?? "");

  // ── List documents ─────────────────────────────────────────────────────────
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, error } =
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
    });

  const allDocs = data?.pages.flatMap((p) => p.data) ?? [];

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
    <div className="min-h-screen bg-[#0D1117] text-slate-100 p-6">
      <div className="max-w-4xl mx-auto space-y-6">
        {/* Navigation */}
        <div className="flex items-center gap-3 text-sm text-slate-400">
          <button
            onClick={() => router.push("/cases")}
            className="hover:text-cyan-400 transition-colors"
          >
            ← Cases
          </button>
          <div className="h-4 border-l border-slate-700" />
          <button
            onClick={() => router.push(`/cases/${id}`)}
            className="hover:text-cyan-400 transition-colors"
          >
            Case
          </button>
          <div className="h-4 border-l border-slate-700" />
          <span className="text-slate-300">Documents</span>
          <div className="h-4 border-l border-slate-700" />
          <button
            onClick={() => router.push(`/cases/${id}/tasks`)}
            className="hover:text-cyan-400 transition-colors"
          >
            Tasks →
          </button>
          <div className="h-4 border-l border-slate-700" />
          <button
            onClick={() => router.push(`/cases/${id}/timeline`)}
            className="hover:text-cyan-400 transition-colors"
          >
            Timeline →
          </button>
        </div>

        {/* Header */}
        <div>
          <h1 className="text-2xl font-bold text-slate-100">Documents</h1>
          <p className="text-slate-400 text-sm mt-1">
            Upload files to extract intelligence and feed the AI copilot.
          </p>
        </div>

        {/* Upload zone */}
        {canUpload && (
          <div className="space-y-2">
            <DocumentUploadZone onUpload={handleUpload} uploading={uploading} />
            {uploadError && (
              <p className="text-xs text-red-400">{uploadError}</p>
            )}
          </div>
        )}

        {/* Filter */}
        <div className="flex items-center gap-3">
          <label className="text-xs text-slate-400">Filter by status:</label>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="text-xs bg-slate-800 border border-slate-700 rounded px-2 py-1 text-slate-300 focus:outline-none focus:border-cyan-500"
          >
            <option value="">All</option>
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

        {error && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-400">
            {error instanceof APIError ? error.message : "Failed to load documents"}
          </div>
        )}

        {!isLoading && allDocs.length === 0 && (
          <div className="text-center py-16 text-slate-500">
            <div className="text-4xl mb-3">📂</div>
            <p className="text-sm font-medium">No documents yet</p>
            <p className="text-xs mt-1">
              {canUpload ? "Upload a file above to get started." : "No files have been uploaded to this case."}
            </p>
          </div>
        )}

        {allDocs.length > 0 && (
          <div className="space-y-3">
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
    </div>
  );
}
