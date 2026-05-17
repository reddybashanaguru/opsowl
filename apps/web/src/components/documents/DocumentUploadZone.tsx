"use client";

import { useCallback, useRef, useState } from "react";

const ACCEPTED_TYPES = ".pdf,.docx,.doc,.xlsx,.xls,.txt,.png,.jpg,.jpeg";
const MAX_SIZE_MB = 50;

interface Props {
  onUpload: (file: File) => Promise<void>;
  uploading?: boolean;
}

export function DocumentUploadZone({ onUpload, uploading = false }: Props) {
  const [dragging, setDragging] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleFile = useCallback(
    async (file: File) => {
      setError(null);
      if (file.size > MAX_SIZE_MB * 1024 * 1024) {
        setError(`File exceeds ${MAX_SIZE_MB} MB limit.`);
        return;
      }
      try {
        await onUpload(file);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : "Upload failed";
        setError(msg);
      }
    },
    [onUpload],
  );

  const onDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      setDragging(false);
      const file = e.dataTransfer.files[0];
      if (file) handleFile(file);
    },
    [handleFile],
  );

  const onInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) handleFile(file);
    e.target.value = "";
  };

  return (
    <div className="space-y-2">
      <div
        role="button"
        tabIndex={0}
        aria-label="Upload document"
        onClick={() => !uploading && inputRef.current?.click()}
        onKeyDown={(e) => e.key === "Enter" && !uploading && inputRef.current?.click()}
        onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
        className={`
          border-2 border-dashed rounded-xl p-8 text-center cursor-pointer
          transition-colors select-none
          ${dragging
            ? "border-cyan-400 bg-cyan-500/10"
            : "border-slate-600 bg-slate-800/30 hover:border-slate-500 hover:bg-slate-800/50"
          }
          ${uploading ? "opacity-60 cursor-not-allowed" : ""}
        `}
      >
        <input
          ref={inputRef}
          type="file"
          accept={ACCEPTED_TYPES}
          className="hidden"
          onChange={onInputChange}
          disabled={uploading}
          data-testid="file-input"
        />
        <div className="text-3xl mb-2">{uploading ? "⏳" : "📂"}</div>
        <p className="text-sm text-slate-300 font-medium">
          {uploading ? "Uploading…" : "Drop a file here or click to browse"}
        </p>
        <p className="text-xs text-slate-500 mt-1">
          PDF, DOCX, XLSX, TXT, PNG, JPG — up to {MAX_SIZE_MB} MB
        </p>
      </div>

      {error && (
        <p role="alert" className="text-xs text-red-400 px-1">
          {error}
        </p>
      )}
    </div>
  );
}
