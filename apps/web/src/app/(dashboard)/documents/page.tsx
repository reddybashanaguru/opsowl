"use client";

import { useState, useCallback } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";

// ── Types ─────────────────────────────────────────────────────────────────────

interface UploaderRef {
  id: string;
  full_name: string;
  email: string;
}

interface Document {
  id: string;
  case_id: string;
  case_number: string;
  case_title: string;
  original_filename: string;
  file_type: string;
  mime_type: string;
  file_size_bytes: number;
  processing_status: "pending" | "processing" | "completed" | "failed";
  page_count?: number;
  chunk_count?: number;
  error_message?: string;
  uploaded_by: UploaderRef;
  created_at: string;
}

interface DocumentMeta {
  total_count: number;
  has_more: boolean;
  next_cursor?: string | null;
}

type DocListPage = { data: Document[]; meta: DocumentMeta };

// ── Helpers ───────────────────────────────────────────────────────────────────

const STATUS_STYLE: Record<string, string> = {
  pending:    "text-slate-400 bg-slate-700/40",
  processing: "text-amber-400 bg-amber-400/10",
  completed:  "text-emerald-400 bg-emerald-400/10",
  failed:     "text-red-400 bg-red-400/10",
};

const STATUS_DOT: Record<string, string> = {
  pending:    "bg-slate-500",
  processing: "bg-amber-400 animate-pulse",
  completed:  "bg-emerald-400",
  failed:     "bg-red-400",
};

const FILE_ICON: Record<string, string> = {
  pdf:  "📄",
  docx: "📝",
  doc:  "📝",
  xlsx: "📊",
  xls:  "📊",
  csv:  "📊",
  txt:  "📃",
  png:  "🖼",
  jpg:  "🖼",
  jpeg: "🖼",
};

function fileIcon(fileType: string) {
  return FILE_ICON[fileType.toLowerCase()] ?? "📎";
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 30) return `${days}d ago`;
  return new Date(dateStr).toLocaleDateString(undefined, { day: "2-digit", month: "short", year: "numeric" });
}

// ── Skeleton ──────────────────────────────────────────────────────────────────

function CardSkeleton() {
  return (
    <div className="bg-slate-800/40 border border-slate-700 rounded-xl p-4 animate-pulse">
      <div className="flex items-start gap-3">
        <div className="w-9 h-9 bg-slate-700 rounded-lg flex-shrink-0" />
        <div className="flex-1 space-y-2">
          <div className="h-3 w-48 bg-slate-700 rounded" />
          <div className="h-2.5 w-32 bg-slate-700 rounded" />
          <div className="h-2.5 w-24 bg-slate-700 rounded" />
        </div>
        <div className="h-5 w-20 bg-slate-700 rounded-full" />
      </div>
    </div>
  );
}

// ── Document Card ─────────────────────────────────────────────────────────────

function DocCard({ doc, onCaseClick }: { doc: Document; onCaseClick: (caseId: string) => void }) {
  return (
    <div className="bg-slate-800/40 border border-slate-700 rounded-xl p-4 hover:border-slate-600 transition-colors">
      <div className="flex items-start gap-3">
        {/* File icon */}
        <div className="w-9 h-9 bg-slate-700/60 rounded-lg flex items-center justify-center flex-shrink-0 text-lg">
          {fileIcon(doc.file_type)}
        </div>

        {/* Main content */}
        <div className="flex-1 min-w-0">
          <p className="text-slate-200 text-sm font-medium truncate">{doc.original_filename}</p>
          <div className="flex items-center gap-2 mt-0.5 flex-wrap">
            <button
              onClick={() => onCaseClick(doc.case_id)}
              className="text-cyan-400/80 text-xs font-mono hover:text-cyan-400 transition-colors"
            >
              {doc.case_number}
            </button>
            <span className="text-slate-600 text-xs truncate max-w-[180px]">{doc.case_title}</span>
          </div>
          <div className="flex items-center gap-3 mt-1.5 text-[11px] text-slate-500">
            <span>{formatBytes(doc.file_size_bytes)}</span>
            {doc.page_count && <span>{doc.page_count} pages</span>}
            {doc.chunk_count && <span>{doc.chunk_count} chunks</span>}
            <span>by {doc.uploaded_by.full_name}</span>
            <span>{timeAgo(doc.created_at)}</span>
          </div>
          {doc.error_message && (
            <p className="text-red-400 text-[11px] mt-1 truncate">{doc.error_message}</p>
          )}
        </div>

        {/* Status badge */}
        <span className={`inline-flex items-center gap-1.5 text-[10px] font-semibold px-2 py-0.5 rounded-full flex-shrink-0 ${STATUS_STYLE[doc.processing_status] ?? STATUS_STYLE.pending}`}>
          <span className={`w-1.5 h-1.5 rounded-full ${STATUS_DOT[doc.processing_status] ?? STATUS_DOT.pending}`} />
          {doc.processing_status}
        </span>
      </div>
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

const STATUSES = ["", "pending", "processing", "completed", "failed"] as const;

export default function DocumentsPage() {
  const router = useRouter();
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [fileTypeFilter, setFileTypeFilter] = useState("");
  const [debouncedFileType, setDebouncedFileType] = useState("");

  const buildUrl = useCallback(
    (cursor?: string) => {
      const params = new URLSearchParams({ limit: "30" });
      if (statusFilter) params.set("status", statusFilter);
      if (cursor) params.set("cursor", cursor);
      return `/api/v1/documents?${params}`;
    },
    [statusFilter],
  );

  const { data, isLoading, isFetchingNextPage, fetchNextPage, hasNextPage } =
    useInfiniteQuery<DocListPage>({
      queryKey: ["documents-global", statusFilter],
      queryFn: ({ pageParam }) =>
        api.getList<Document, DocumentMeta>(buildUrl(pageParam as string | undefined)),
      getNextPageParam: (last) =>
        last.meta.has_more && last.meta.next_cursor ? last.meta.next_cursor : undefined,
      initialPageParam: undefined,
    });

  const seen = new Set<string>();
  const allDocs = (data?.pages.flatMap((p) => p.data) ?? []).filter((doc) => {
    if (seen.has(doc.id)) return false;
    seen.add(doc.id);
    return true;
  });

  const filtered = debouncedFileType
    ? allDocs.filter((d) =>
        d.file_type.toLowerCase().includes(debouncedFileType.toLowerCase()) ||
        d.original_filename.toLowerCase().includes(debouncedFileType.toLowerCase())
      )
    : allDocs;

  function handleFileTypeInput(val: string) {
    setFileTypeFilter(val);
    clearTimeout((window as unknown as Record<string, unknown>)._docDebounce as ReturnType<typeof setTimeout>);
    (window as unknown as Record<string, unknown>)._docDebounce = setTimeout(
      () => setDebouncedFileType(val),
      350,
    );
  }

  const totalCount = data?.pages[0]?.meta.total_count ?? 0;

  return (
    <div className="px-6 py-8 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-semibold text-white">Documents</h1>
          <p className="text-slate-500 text-sm mt-0.5">
            All uploaded files across every case
            {totalCount > 0 && (
              <span className="text-slate-600 ml-1">· {totalCount.toLocaleString()} total</span>
            )}
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3 mb-5 flex-wrap">
        {/* Status filter */}
        <div className="flex items-center gap-1">
          {STATUSES.map((s) => (
            <button
              key={s}
              onClick={() => setStatusFilter(s)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                statusFilter === s
                  ? "bg-cyan-500/20 text-cyan-400 border border-cyan-500/30"
                  : "text-slate-500 hover:text-slate-300 hover:bg-slate-700/40"
              }`}
            >
              {s === "" ? "All" : s}
            </button>
          ))}
        </div>

        {/* File search */}
        <div className="relative max-w-xs">
          <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500 text-xs">⌕</span>
          <input
            type="text"
            value={fileTypeFilter}
            onChange={(e) => handleFileTypeInput(e.target.value)}
            placeholder="Search by filename or type…"
            className="w-full bg-slate-800 border border-slate-600 rounded-lg pl-8 pr-3 py-1.5 text-slate-300 placeholder-slate-600 text-xs focus:outline-none focus:border-cyan-500 transition-colors"
          />
        </div>

        {filtered.length > 0 && (
          <span className="text-xs text-slate-600 ml-auto">
            Showing {filtered.length} document{filtered.length !== 1 ? "s" : ""}
          </span>
        )}
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <CardSkeleton key={i} />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24 text-center">
          <div className="text-4xl mb-3">📁</div>
          <p className="text-slate-400 text-sm">No documents found</p>
          <p className="text-slate-600 text-xs mt-1">
            Upload files from a case page to see them here
          </p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
            {filtered.map((doc) => (
              <DocCard
                key={doc.id}
                doc={doc}
                onCaseClick={(caseId) => router.push(`/cases/${caseId}`)}
              />
            ))}
          </div>

          {/* Load more */}
          {hasNextPage && !debouncedFileType && (
            <div className="flex justify-center mt-6">
              <button
                onClick={() => fetchNextPage()}
                disabled={isFetchingNextPage}
                className="px-4 py-2 bg-slate-700/60 hover:bg-slate-700 border border-slate-600 rounded-lg text-sm text-slate-400 hover:text-white transition-colors disabled:opacity-50"
              >
                {isFetchingNextPage ? "Loading…" : "Load more documents"}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
