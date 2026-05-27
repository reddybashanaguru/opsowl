"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { api, APIError } from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────────────────────

interface ColumnMapping {
  csv_header: string;
  case_field: string;
}

interface ParsePreview {
  headers: string[];
  sample_rows: Record<string, string>[];
  total_rows: number;
  detected_map: ColumnMapping[];
}

interface ImportJob {
  id: string;
  filename: string;
  total_rows: number;
  imported_rows: number;
  failed_rows: number;
  status: "pending" | "processing" | "done" | "partial" | "failed";
  created_at: string;
  updated_at: string;
}

interface ImportRowResult {
  id: string;
  row_number: number;
  raw_data: Record<string, string>;
  status: "pending" | "success" | "failed";
  case_id?: string;
  error?: string;
}

interface ImportRowsPage {
  rows: ImportRowResult[];
  has_more: boolean;
  total: number;
}

const CASE_FIELDS = [
  { value: "title",              label: "Title",              required: true },
  { value: "description",        label: "Description",        required: false },
  { value: "priority",           label: "Priority",           required: false },
  { value: "tags",               label: "Tags (comma-sep)",   required: false },
  { value: "due_date",           label: "Due Date",           required: false },
  { value: "assigned_to_email",  label: "Assign To (email)",  required: false },
];

const STATUS_COLORS: Record<string, string> = {
  pending:    "bg-yellow-500/20 text-yellow-400",
  processing: "bg-blue-500/20 text-blue-400",
  done:       "bg-green-500/20 text-green-400",
  partial:    "bg-orange-500/20 text-orange-400",
  failed:     "bg-red-500/20 text-red-400",
  success:    "bg-green-500/20 text-green-400",
};

// ─── Step 1 — File upload ─────────────────────────────────────────────────────

function UploadStep({ onPreview }: { onPreview: (file: File, preview: ParsePreview) => void }) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);
  const [loading, setLoading]   = useState(false);
  const [error, setError]       = useState<string | null>(null);

  const handleFile = useCallback(async (file: File) => {
    if (!file.name.endsWith(".csv")) {
      setError("Only .csv files are supported.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const form = new FormData();
      form.append("file", file);
      const preview = await api.upload<ParsePreview>("/api/v1/imports/preview", form);
      onPreview(file, preview);
    } catch (e) {
      setError(e instanceof APIError ? e.message : "Failed to parse file.");
    } finally {
      setLoading(false);
    }
  }, [onPreview]);

  const onDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
    const file = e.dataTransfer.files[0];
    if (file) handleFile(file);
  }, [handleFile]);

  return (
    <div className="max-w-lg mx-auto">
      <div
        onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
        onClick={() => inputRef.current?.click()}
        className={`border-2 border-dashed rounded-xl p-12 text-center cursor-pointer transition-colors ${
          dragging ? "border-cyan-400 bg-cyan-500/10" : "border-slate-600 hover:border-slate-400 bg-slate-800/40"
        }`}
      >
        <input
          ref={inputRef}
          type="file"
          accept=".csv"
          className="hidden"
          onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f); }}
        />
        <p className="text-4xl mb-3">{loading ? "⏳" : "📂"}</p>
        <p className="text-slate-200 font-medium">
          {loading ? "Parsing…" : "Drop a CSV file here, or click to browse"}
        </p>
        <p className="text-slate-500 text-sm mt-1">Max 5 MB · Max 5,000 rows</p>
      </div>
      {error && <p className="mt-3 text-sm text-red-400">{error}</p>}

      {/* Format hint */}
      <div className="mt-6 bg-slate-800/40 border border-slate-700 rounded-lg p-4 text-sm text-slate-400">
        <p className="font-medium text-slate-300 mb-2">Expected format</p>
        <p className="font-mono text-xs bg-slate-900 rounded px-3 py-2 text-slate-300">
          title,priority,description,tags,due_date<br />
          "Suspicious transfer — Acct 112233","high","₹45L transfer…","aml,upi","2026-06-15"
        </p>
        <p className="mt-2">
          Only <strong className="text-slate-300">title</strong> is required. Priority values:{" "}
          <code className="text-cyan-400">critical · high · medium · low</code>.
          Tags are comma-separated inside the cell.
        </p>
      </div>
    </div>
  );
}

// ─── Step 2 — Column mapping ──────────────────────────────────────────────────

function MappingStep({
  file,
  preview,
  onStart,
  onBack,
}: {
  file: File;
  preview: ParsePreview;
  onStart: (job: ImportJob) => void;
  onBack: () => void;
}) {
  const [mapping, setMapping] = useState<Record<string, string>>(() => {
    const m: Record<string, string> = {};
    for (const d of preview.detected_map) {
      m[d.csv_header] = d.case_field;
    }
    return m;
  });
  const [loading, setLoading] = useState(false);
  const [error, setError]     = useState<string | null>(null);

  const setField = (header: string, field: string) =>
    setMapping((prev) => ({ ...prev, [header]: field }));

  const hasTitleMapped = Object.values(mapping).includes("title");

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);
    try {
      const columnMapping: ColumnMapping[] = Object.entries(mapping)
        .filter(([, v]) => v !== "")
        .map(([csv_header, case_field]) => ({ csv_header, case_field }));

      const form = new FormData();
      form.append("file", file);
      form.append("mapping", JSON.stringify(columnMapping));
      const job = await api.upload<ImportJob>("/api/v1/imports", form);
      onStart(job);
    } catch (e) {
      setError(e instanceof APIError ? e.message : "Failed to start import.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Summary */}
      <div className="flex items-center gap-4 text-sm text-slate-400">
        <span className="text-slate-200 font-medium">{file.name}</span>
        <span>{preview.total_rows.toLocaleString()} rows detected</span>
      </div>

      {/* Mapping table */}
      <div>
        <h2 className="text-slate-200 font-medium mb-3">Map CSV columns to case fields</h2>
        <div className="space-y-2">
          {preview.headers.map((header) => (
            <div key={header} className="flex items-center gap-3 bg-slate-800/60 border border-slate-700 rounded-lg px-4 py-2.5">
              <span className="flex-1 font-mono text-sm text-slate-300 truncate">{header}</span>
              <span className="text-slate-600">→</span>
              <select
                value={mapping[header] ?? ""}
                onChange={(e) => setField(header, e.target.value)}
                className="w-48 bg-slate-900 border border-slate-600 rounded px-2 py-1 text-sm text-slate-200 focus:outline-none focus:border-cyan-500"
              >
                <option value="">— skip —</option>
                {CASE_FIELDS.map((f) => (
                  <option key={f.value} value={f.value}>
                    {f.label}{f.required ? " *" : ""}
                  </option>
                ))}
              </select>
            </div>
          ))}
        </div>
        {!hasTitleMapped && (
          <p className="mt-2 text-sm text-amber-400">Map at least one column to <strong>Title</strong> to proceed.</p>
        )}
      </div>

      {/* Sample rows */}
      {preview.sample_rows.length > 0 && (
        <div>
          <h2 className="text-slate-200 font-medium mb-2 text-sm">Preview (first {preview.sample_rows.length} rows)</h2>
          <div className="overflow-x-auto rounded-lg border border-slate-700">
            <table className="w-full text-xs">
              <thead className="bg-slate-800">
                <tr>
                  {preview.headers.map((h) => (
                    <th key={h} className="px-3 py-2 text-left text-slate-400 font-medium whitespace-nowrap">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {preview.sample_rows.map((row, i) => (
                  <tr key={i} className="border-t border-slate-800 hover:bg-slate-800/40">
                    {preview.headers.map((h) => (
                      <td key={h} className="px-3 py-2 text-slate-300 max-w-xs truncate">{row[h] ?? ""}</td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {error && <p className="text-sm text-red-400">{error}</p>}

      <div className="flex gap-3">
        <button
          onClick={handleSubmit}
          disabled={loading || !hasTitleMapped}
          className="px-5 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition-colors"
        >
          {loading ? "Starting…" : `Import ${preview.total_rows.toLocaleString()} rows`}
        </button>
        <button onClick={onBack} className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm transition-colors">
          Back
        </button>
      </div>
    </div>
  );
}

// ─── Step 3 — Progress ────────────────────────────────────────────────────────

function ProgressStep({ initialJob }: { initialJob: ImportJob }) {
  const router = useRouter();
  const [job, setJob] = useState(initialJob);
  const [failedRows, setFailedRows] = useState<ImportRowResult[]>([]);
  const [polling, setPolling] = useState(true);

  // Poll job status every 2 seconds while processing
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const startPolling = useCallback(() => {
    if (pollRef.current) return;
    pollRef.current = setInterval(async () => {
      try {
        const updated = await api.get<ImportJob>(`/api/v1/imports/${initialJob.id}`);
        setJob(updated);
        if (updated.status !== "pending" && updated.status !== "processing") {
          clearInterval(pollRef.current!);
          pollRef.current = null;
          setPolling(false);
          // Load failed rows
          const page = await api.get<ImportRowsPage>(`/api/v1/imports/${initialJob.id}/rows?status=failed&limit=50`);
          setFailedRows(page.rows ?? []);
        }
      } catch {
        // ignore transient errors
      }
    }, 2000);
  }, [initialJob.id]);

  // Start polling on mount
  useEffect(() => { startPolling(); }, [startPolling]);

  const pct = job.total_rows > 0 ? Math.round(((job.imported_rows + job.failed_rows) / job.total_rows) * 100) : 0;
  const done = job.status !== "pending" && job.status !== "processing";

  return (
    <div className="max-w-xl space-y-6">
      {/* Status badge */}
      <div className="flex items-center gap-3">
        <span className={`px-3 py-1 rounded-full text-sm font-medium ${STATUS_COLORS[job.status] ?? ""}`}>
          {job.status}
        </span>
        <span className="text-slate-400 text-sm">{job.filename}</span>
      </div>

      {/* Progress bar */}
      <div>
        <div className="flex justify-between text-xs text-slate-400 mb-1">
          <span>{job.imported_rows + job.failed_rows} / {job.total_rows} rows processed</span>
          <span>{pct}%</span>
        </div>
        <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all duration-500 ${
              job.status === "failed" ? "bg-red-500" : job.failed_rows > 0 ? "bg-orange-500" : "bg-cyan-500"
            }`}
            style={{ width: `${pct}%` }}
          />
        </div>
      </div>

      {/* Counts */}
      <div className="grid grid-cols-3 gap-3">
        {[
          { label: "Total",    value: job.total_rows,    color: "text-slate-300" },
          { label: "Imported", value: job.imported_rows, color: "text-green-400" },
          { label: "Failed",   value: job.failed_rows,   color: "text-red-400"   },
        ].map((stat) => (
          <div key={stat.label} className="bg-slate-800/60 border border-slate-700 rounded-lg p-3 text-center">
            <p className={`text-2xl font-bold ${stat.color}`}>{stat.value.toLocaleString()}</p>
            <p className="text-xs text-slate-500 mt-0.5">{stat.label}</p>
          </div>
        ))}
      </div>

      {!done && (
        <p className="text-sm text-slate-400 animate-pulse">Processing rows…</p>
      )}

      {/* Failed rows */}
      {done && failedRows.length > 0 && (
        <div>
          <h3 className="text-slate-200 font-medium text-sm mb-2">
            Failed rows ({failedRows.length}{job.failed_rows > failedRows.length ? "+" : ""})
          </h3>
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {failedRows.map((row) => (
              <div key={row.id} className="bg-red-900/20 border border-red-700/40 rounded-lg px-3 py-2 text-xs">
                <span className="text-slate-400 mr-2">Row {row.row_number}</span>
                <span className="text-red-300">{row.error}</span>
                {Object.entries(row.raw_data ?? {}).slice(0, 3).map(([k, v]) => (
                  <span key={k} className="ml-2 text-slate-500">{k}: <span className="text-slate-400">{v}</span></span>
                ))}
              </div>
            ))}
          </div>
        </div>
      )}

      {done && (
        <div className="flex gap-3">
          <button
            onClick={() => router.push("/cases")}
            className="px-5 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium rounded-lg transition-colors"
          >
            View Cases
          </button>
          <button
            onClick={() => router.push("/cases/import")}
            className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-200 text-sm rounded-lg transition-colors"
          >
            Import Another
          </button>
        </div>
      )}
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

type Step = "upload" | "mapping" | "progress";

export default function CaseImportPage() {
  const [step, setStep]       = useState<Step>("upload");
  const [file, setFile]       = useState<File | null>(null);
  const [preview, setPreview] = useState<ParsePreview | null>(null);
  const [job, setJob]         = useState<ImportJob | null>(null);

  const STEP_LABELS = ["Upload", "Map Columns", "Progress"];
  const STEP_INDEX  = { upload: 0, mapping: 1, progress: 2 };

  return (
    <div className="space-y-8 max-w-3xl">
      {/* Header */}
      <div>
        <h1 className="text-lg font-semibold text-slate-100">Bulk Case Import</h1>
        <p className="text-sm text-slate-400 mt-0.5">
          Upload a CSV of flagged entities to create investigation cases in bulk.
        </p>
      </div>

      {/* Stepper */}
      <div className="flex items-center gap-0">
        {STEP_LABELS.map((label, i) => {
          const active  = STEP_INDEX[step] === i;
          const done    = STEP_INDEX[step] > i;
          return (
            <div key={label} className="flex items-center">
              <div className="flex items-center gap-2">
                <div className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold transition-colors ${
                  active ? "bg-cyan-600 text-white" : done ? "bg-green-600 text-white" : "bg-slate-700 text-slate-400"
                }`}>
                  {done ? "✓" : i + 1}
                </div>
                <span className={`text-sm ${active ? "text-slate-200" : "text-slate-500"}`}>{label}</span>
              </div>
              {i < STEP_LABELS.length - 1 && (
                <div className={`w-12 h-px mx-3 ${done ? "bg-green-600" : "bg-slate-700"}`} />
              )}
            </div>
          );
        })}
      </div>

      {/* Step content */}
      {step === "upload" && (
        <UploadStep
          onPreview={(f, p) => {
            setFile(f);
            setPreview(p);
            setStep("mapping");
          }}
        />
      )}

      {step === "mapping" && file && preview && (
        <MappingStep
          file={file}
          preview={preview}
          onStart={(j) => { setJob(j); setStep("progress"); }}
          onBack={() => setStep("upload")}
        />
      )}

      {step === "progress" && job && (
        <ProgressStep initialJob={job} />
      )}
    </div>
  );
}
