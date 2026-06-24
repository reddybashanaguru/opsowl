"use client";

import { useState } from "react";
import { DocumentProcessingBadge } from "./DocumentProcessingBadge";

export interface Document {
  id: string;
  case_id: string;
  original_filename: string;
  file_type: string;
  mime_type: string;
  file_size_bytes: number;
  processing_status: string;
  page_count?: number | null;
  chunk_count?: number | null;
  error_message?: string | null;
  uploaded_by: { id: string; full_name: string; email: string };
  download_url?: string;
  created_at: string;
  updated_at: string;
}

interface Props {
  document: Document;
  onDelete?: (id: string) => void;
  onRetry?: (id: string) => void;
  retrying?: boolean;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

const FILE_ICONS: Record<string, string> = {
  pdf:  "📄",
  docx: "📝",
  doc:  "📝",
  xlsx: "📊",
  xls:  "📊",
  txt:  "📃",
  png:  "🖼️",
  jpg:  "🖼️",
  jpeg: "🖼️",
};

export function DocumentCard({ document: doc, onDelete, onRetry, retrying }: Props) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const icon = FILE_ICONS[doc.file_type] ?? "📎";

  return (
    <div className="group bg-slate-800/50 border border-slate-700/50 rounded-lg p-4 hover:border-slate-600 transition-colors">
      <div className="flex items-start gap-3">
        <span className="text-2xl flex-shrink-0 mt-0.5">{icon}</span>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm font-medium text-slate-100 truncate">
              {doc.original_filename}
            </span>
            <DocumentProcessingBadge status={doc.processing_status} />
          </div>

          <div className="flex items-center gap-3 mt-1 text-xs text-slate-500">
            <span>{formatBytes(doc.file_size_bytes)}</span>
            {doc.page_count != null && <span>{doc.page_count} pages</span>}
            {doc.chunk_count != null && <span>{doc.chunk_count} chunks</span>}
            <span>by {doc.uploaded_by.full_name || doc.uploaded_by.email}</span>
            <span>{new Date(doc.created_at).toLocaleDateString()}</span>
          </div>

          {doc.error_message && (
            <p className="mt-1 text-xs text-red-400 truncate" title={doc.error_message}>
              {doc.error_message}
            </p>
          )}
          {(doc.processing_status === "processing" || doc.processing_status === "pending") && !doc.error_message && (
            <p className="mt-1 text-xs text-slate-500 flex items-center gap-1">
              <span className="inline-block w-2.5 h-2.5 border border-slate-400 border-t-cyan-400 rounded-full animate-spin" />
              Extracting text and building AI index… auto-updates when complete.
            </p>
          )}
        </div>

        <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
          {doc.download_url && (
            <a
              href={doc.download_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-cyan-400 hover:text-cyan-300 px-2 py-1 rounded border border-cyan-500/30 hover:border-cyan-400/60 transition-colors"
            >
              Download
            </a>
          )}
          {onRetry && doc.processing_status === "failed" && (
            <button
              onClick={() => onRetry(doc.id)}
              disabled={retrying}
              className="text-xs text-amber-400 hover:text-amber-300 px-2 py-1 rounded border border-amber-500/30 hover:border-amber-400/60 transition-colors disabled:opacity-60"
            >
              {retrying ? "Retrying…" : "Retry"}
            </button>
          )}
          {onDelete && (
            confirmDelete ? (
              <div className="flex items-center gap-1">
                <button
                  onClick={() => { onDelete(doc.id); setConfirmDelete(false); }}
                  className="text-xs text-white bg-red-600 hover:bg-red-500 px-2 py-1 rounded transition-colors"
                >
                  Confirm
                </button>
                <button
                  onClick={() => setConfirmDelete(false)}
                  className="text-xs text-slate-400 hover:text-slate-200 px-2 py-1 rounded border border-slate-600 transition-colors"
                >
                  Cancel
                </button>
              </div>
            ) : (
              <button
                onClick={() => setConfirmDelete(true)}
                className="text-xs text-red-400 hover:text-red-300 px-2 py-1 rounded border border-red-500/30 hover:border-red-400/60 transition-colors"
              >
                Delete
              </button>
            )
          )}
        </div>
      </div>
    </div>
  );
}
