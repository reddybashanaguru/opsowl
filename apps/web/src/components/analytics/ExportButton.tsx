"use client";

import { useState } from "react";
import { api } from "@/lib/api";

interface ExportButtonProps {
  path: string;
  filename: string;
  label: string;
}

export function ExportButton({ path, filename, label }: ExportButtonProps) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleExport = async () => {
    setBusy(true);
    setError(null);
    try {
      await api.download(path, filename);
    } catch {
      setError("Export failed. Please try again.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div>
      <button
        onClick={handleExport}
        disabled={busy}
        aria-label={`Export ${label}`}
        className="flex items-center gap-1.5 bg-slate-700 hover:bg-slate-600 text-slate-300 hover:text-white border border-slate-600 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors disabled:opacity-60"
      >
        {busy ? "Exporting…" : `↓ ${label}`}
      </button>
      {error && (
        <p role="alert" className="text-red-400 text-xs mt-1">
          {error}
        </p>
      )}
    </div>
  );
}
