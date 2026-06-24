"use client";

import { use, useState, useCallback, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { CaseSubNav } from "@/components/cases/CaseSubNav";
import { FlowOfFundsGraph } from "@/components/cases/FlowOfFundsGraph";
import { useConfig } from "@/lib/hooks/useConfig";
import { formatCurrency, getCTRStatus } from "@/lib/utils/currency";
import {
  AlertTriangle,
  ArrowLeftRight,
  ArrowRight,
  ChevronDown,
  ChevronUp,
  Download,
  Flag,
  Plus,
  RotateCcw,
  Search,
  TrendingDown,
  TrendingUp,
  Upload,
  X,
  ZapOff,
} from "lucide-react";

// ── Types ─────────────────────────────────────────────────────────────────────

interface AccountResponse {
  id: string;
  account_number: string;
  account_type: string;
  bank_name: string;
  holder_name: string;
}

interface TransactionResponse {
  id: string;
  txn_ref: string;
  txn_date: string;
  txn_time?: string;
  amount: number;
  currency: string;
  txn_type: string;
  channel: string;
  status: string;
  description: string;
  narration: string;
  from_account?: AccountResponse;
  to_account?: AccountResponse;
  from_account_raw: string;
  to_account_raw: string;
  from_name_raw: string;
  to_name_raw: string;
  is_flagged: boolean;
  flag_reason: string;
  risk_score: number;
  counterparty_bank: string;
  counterparty_country: string;
  source_system: string;
  relevance_note: string;
  created_at: string;
}

interface TransactionSummary {
  total_count: number;
  total_amount: number;
  flagged_count: number;
  debit_total: number;
  credit_total: number;
  unique_accounts: number;
  date_range: { from: string; to: string };
}

interface FlowNode {
  id: string;
  label: string;
  holder_name: string;
  account_type: string;
  is_subject: boolean;
  total_in: number;
  total_out: number;
  txn_count: number;
}

interface FlowEdge {
  id: string;
  source: string;
  target: string;
  amount: number;
  txn_count: number;
  channel: string;
  is_suspicious: boolean;
}

interface FlowGraph {
  nodes: FlowNode[];
  edges: FlowEdge[];
  has_cycles: boolean;
  total_flow: number;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function fmtAmount(amount: number, currency = "INR") {
  return formatCurrency(amount, currency);
}

function fmtDate(iso: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("en-IN", { day: "2-digit", month: "short", year: "numeric" });
}

const CHANNEL_LABELS: Record<string, string> = {
  neft: "NEFT", rtgs: "RTGS", imps: "IMPS", upi: "UPI",
  cash: "Cash", card: "Card", swift: "SWIFT", internal: "Internal", unknown: "—",
};

const TXN_TYPE_STYLES: Record<string, string> = {
  debit:    "text-red-400",
  credit:   "text-emerald-400",
  reversal: "text-amber-400",
  fee:      "text-slate-400",
  interest: "text-cyan-400",
};

// ── Summary Stats ─────────────────────────────────────────────────────────────

function SummaryBar({ summary }: { summary: TransactionSummary }) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 mb-5">
      {[
        { label: "Transactions", value: summary.total_count, fmt: (v: number) => v.toString() },
        { label: "Total Flow", value: summary.total_amount, fmt: (v: number) => fmtAmount(v) },
        { label: "Debits", value: summary.debit_total, fmt: (v: number) => fmtAmount(v), icon: TrendingDown, color: "text-red-400" },
        { label: "Credits", value: summary.credit_total, fmt: (v: number) => fmtAmount(v), icon: TrendingUp, color: "text-emerald-400" },
        { label: "Flagged", value: summary.flagged_count, fmt: (v: number) => v.toString(), color: summary.flagged_count > 0 ? "text-amber-400" : undefined },
        { label: "Accounts", value: summary.unique_accounts, fmt: (v: number) => v.toString() },
      ].map(({ label, value, fmt, color }) => (
        <div key={label} className="bg-slate-800/60 border border-slate-700/50 rounded-lg p-3">
          <p className="text-xs text-slate-500 mb-1">{label}</p>
          <p className={`text-sm font-semibold ${color ?? "text-slate-200"}`}>{fmt(value)}</p>
        </div>
      ))}
    </div>
  );
}

// ── Transaction Row ───────────────────────────────────────────────────────────

function TxnRow({
  txn,
  onFlag,
  onRemove,
  ctrThreshold = 0,
}: {
  txn: TransactionResponse;
  onFlag: (id: string, flagged: boolean) => void;
  onRemove: (id: string) => void;
  ctrThreshold?: number;
}) {
  const [expanded, setExpanded] = useState(false);

  const fromLabel = txn.from_account?.holder_name || txn.from_name_raw || txn.from_account_raw || "—";
  const toLabel = txn.to_account?.holder_name || txn.to_name_raw || txn.to_account_raw || "—";
  const fromBank = txn.from_account?.bank_name || txn.counterparty_bank || "";
  const toBank = txn.to_account?.bank_name || "";

  return (
    <div className={`border rounded-lg transition-colors ${txn.is_flagged ? "border-amber-500/40 bg-amber-900/5" : "border-slate-700/50 bg-slate-800/40"}`}>
      {/* Main row */}
      <div
        className="flex items-center gap-3 px-4 py-3 cursor-pointer select-none"
        onClick={() => setExpanded((v) => !v)}
      >
        {/* Flag indicator */}
        <div className={`w-1.5 h-8 rounded-full shrink-0 ${txn.is_flagged ? "bg-amber-400" : "bg-slate-700"}`} />

        {/* Date + ref */}
        <div className="w-24 shrink-0">
          <p className="text-xs text-slate-200">{fmtDate(txn.txn_date)}</p>
          <p className="text-[10px] text-slate-500 font-mono truncate">{txn.txn_ref}</p>
        </div>

        {/* From → To */}
        <div className="flex-1 min-w-0 flex items-center gap-2">
          <div className="min-w-0">
            <p className="text-xs text-slate-300 truncate">{fromLabel}</p>
            {fromBank && <p className="text-[10px] text-slate-500 truncate">{fromBank}</p>}
          </div>
          <ArrowRight size={12} className="text-slate-600 shrink-0" />
          <div className="min-w-0">
            <p className="text-xs text-slate-300 truncate">{toLabel}</p>
            {toBank && <p className="text-[10px] text-slate-500 truncate">{toBank}</p>}
          </div>
        </div>

        {/* Channel */}
        <span className="text-[10px] px-1.5 py-0.5 bg-slate-700/60 text-slate-400 rounded shrink-0">
          {CHANNEL_LABELS[txn.channel] ?? txn.channel}
        </span>

        {/* Amount */}
        <div className="text-right shrink-0 w-28">
          <p className={`text-sm font-semibold ${TXN_TYPE_STYLES[txn.txn_type] ?? "text-slate-200"}`}>
            {txn.txn_type === "credit" ? "+" : txn.txn_type === "debit" ? "−" : ""}
            {fmtAmount(txn.amount, txn.currency)}
          </p>
          <p className="text-[10px] text-slate-500 capitalize">{txn.txn_type}</p>
          {getCTRStatus(txn.amount, ctrThreshold) === "above" && (
            <span className="inline-block text-[9px] px-1 py-0.5 bg-red-900/50 text-red-300 border border-red-500/40 rounded mt-0.5 animate-pulse">CTR</span>
          )}
          {getCTRStatus(txn.amount, ctrThreshold) === "near" && (
            <span className="inline-block text-[9px] px-1 py-0.5 bg-amber-900/50 text-amber-300 border border-amber-500/40 rounded mt-0.5">Near-CTR</span>
          )}
        </div>

        {/* Risk score */}
        {txn.risk_score > 0 && (
          <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium shrink-0 ${
            txn.risk_score >= 70 ? "bg-red-900/50 text-red-300" :
            txn.risk_score >= 40 ? "bg-amber-900/50 text-amber-300" :
            "bg-slate-700 text-slate-400"
          }`}>
            R{txn.risk_score}
          </span>
        )}

        {/* Expand */}
        {expanded ? <ChevronUp size={12} className="text-slate-500 shrink-0" /> : <ChevronDown size={12} className="text-slate-500 shrink-0" />}
      </div>

      {/* Expanded detail */}
      {expanded && (
        <div className="px-4 pb-4 pt-1 border-t border-slate-700/40 space-y-3">
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 text-xs">
            {txn.narration && (
              <div className="col-span-2 sm:col-span-3">
                <span className="text-slate-500">Narration</span>
                <p className="text-slate-300 mt-0.5 bg-slate-700/40 rounded px-2 py-1 font-mono text-[11px]">{txn.narration}</p>
              </div>
            )}
            {txn.description && (
              <div>
                <span className="text-slate-500">Description</span>
                <p className="text-slate-300 mt-0.5">{txn.description}</p>
              </div>
            )}
            <div>
              <span className="text-slate-500">Status</span>
              <p className="text-slate-300 mt-0.5 capitalize">{txn.status}</p>
            </div>
            <div>
              <span className="text-slate-500">Source</span>
              <p className="text-slate-300 mt-0.5 capitalize">{txn.source_system.replace("_", " ")}</p>
            </div>
            {txn.counterparty_country && (
              <div>
                <span className="text-slate-500">Counterparty Country</span>
                <p className="text-slate-300 mt-0.5">{txn.counterparty_country}</p>
              </div>
            )}
            {txn.relevance_note && (
              <div className="col-span-2 sm:col-span-3">
                <span className="text-slate-500">Relevance Note</span>
                <p className="text-slate-300 mt-0.5 italic">{txn.relevance_note}</p>
              </div>
            )}
            {txn.is_flagged && txn.flag_reason && (
              <div className="col-span-2 sm:col-span-3">
                <span className="text-amber-400">Flag Reason</span>
                <p className="text-amber-300 mt-0.5">{txn.flag_reason}</p>
              </div>
            )}
          </div>

          <div className="flex items-center gap-2 pt-1">
            <button
              onClick={() => onFlag(txn.id, !txn.is_flagged)}
              className={`flex items-center gap-1 text-xs px-2.5 py-1.5 rounded-lg transition-colors ${
                txn.is_flagged
                  ? "bg-amber-900/30 border border-amber-500/40 text-amber-300 hover:bg-amber-900/50"
                  : "bg-slate-700/50 border border-slate-600/50 text-slate-400 hover:text-amber-300 hover:border-amber-500/40"
              }`}
            >
              <Flag size={11} />
              {txn.is_flagged ? "Unflag" : "Flag as Suspicious"}
            </button>
            <button
              onClick={() => onRemove(txn.id)}
              className="flex items-center gap-1 text-xs px-2.5 py-1.5 rounded-lg bg-slate-700/50 border border-slate-600/50 text-slate-400 hover:text-red-400 hover:border-red-500/40 transition-colors"
            >
              <X size={11} />
              Remove from Case
            </button>
          </div>
        </div>
      )}
    </div>
  );
}


// ── CSV Import Modal ──────────────────────────────────────────────────────────

const STANDARD_FIELDS = [
  "txn_ref", "txn_date", "txn_time", "amount", "currency", "txn_type",
  "channel", "description", "narration", "from_account_raw", "to_account_raw",
  "from_name_raw", "to_name_raw", "counterparty_bank", "counterparty_country",
  "is_flagged", "risk_score", "external_id",
];

function CSVImportModal({
  caseId,
  onClose,
  onDone,
}: {
  caseId: string;
  onClose: () => void;
  onDone: () => void;
}) {
  const [step, setStep] = useState<"upload" | "map" | "done">("upload");
  const [headers, setHeaders] = useState<string[]>([]);
  const [rows, setRows] = useState<Record<string, string>[]>([]);
  const [mapping, setMapping] = useState<Record<string, string>>({});
  const [result, setResult] = useState<{ imported: number; skipped: number; errors: string[] } | null>(null);
  const [importing, setImporting] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const handleFile = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      const text = ev.target?.result as string;
      const lines = text.split(/\r?\n/).filter(Boolean);
      if (lines.length < 2) return;
      const hdrs = lines[0].split(",").map((h) => h.trim().replace(/^"|"$/g, ""));
      const parsed = lines.slice(1, 201).map((line) => {
        const vals = line.split(",").map((v) => v.trim().replace(/^"|"$/g, ""));
        return Object.fromEntries(hdrs.map((h, i) => [h, vals[i] ?? ""]));
      });
      setHeaders(hdrs);
      setRows(parsed);
      // Auto-map headers to standard fields by fuzzy match
      const autoMap: Record<string, string> = {};
      hdrs.forEach((h) => {
        const lower = h.toLowerCase().replace(/[\s_-]/g, "_");
        const match = STANDARD_FIELDS.find(
          (f) => f === lower || lower.includes(f) || f.includes(lower)
        );
        if (match) autoMap[h] = match;
      });
      setMapping(autoMap);
      setStep("map");
    };
    reader.readAsText(file);
  }, []);

  const handleImport = async () => {
    setImporting(true);
    try {
      const res = await api.post<{ imported: number; skipped: number; errors: string[] }>(
        `/api/v1/cases/${caseId}/transactions/import`,
        { mapping, rows }
      );
      setResult(res);
      setStep("done");
      onDone();
    } catch (e) {
      setResult({ imported: 0, skipped: rows.length, errors: [(e as APIError).message] });
      setStep("done");
    } finally {
      setImporting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-5 border-b border-slate-700">
          <h2 className="font-semibold text-slate-200 flex items-center gap-2">
            <Upload size={16} className="text-cyan-400" />
            Import Transactions from CSV
          </h2>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300">
            <X size={16} />
          </button>
        </div>

        <div className="p-5">
          {step === "upload" && (
            <div className="space-y-4">
              <p className="text-sm text-slate-400">
                Upload a CSV file (up to 200 rows). The first row must be the header.
                Fields will be auto-detected from column names.
              </p>
              <div
                className="border-2 border-dashed border-slate-600 rounded-lg p-8 text-center cursor-pointer hover:border-cyan-500/50 transition-colors"
                onClick={() => fileRef.current?.click()}
              >
                <Upload size={24} className="mx-auto text-slate-500 mb-2" />
                <p className="text-sm text-slate-400">Click to choose a CSV file</p>
              </div>
              <input ref={fileRef} type="file" accept=".csv" className="hidden" onChange={handleFile} />
              <div>
                <p className="text-xs text-slate-500 mb-2">Recognised fields:</p>
                <div className="flex flex-wrap gap-1">
                  {STANDARD_FIELDS.map((f) => (
                    <span key={f} className="text-[10px] px-1.5 py-0.5 bg-slate-700 text-slate-400 rounded font-mono">{f}</span>
                  ))}
                </div>
              </div>
            </div>
          )}

          {step === "map" && (
            <div className="space-y-4">
              <p className="text-sm text-slate-400">
                Map your CSV columns to transaction fields. {rows.length} data rows detected.
              </p>
              <div className="space-y-2 max-h-64 overflow-y-auto">
                {headers.map((h) => (
                  <div key={h} className="flex items-center gap-3">
                    <span className="text-xs font-mono text-slate-300 w-40 truncate shrink-0">{h}</span>
                    <span className="text-slate-600 text-xs">→</span>
                    <select
                      value={mapping[h] ?? ""}
                      onChange={(e) => setMapping((m) => ({ ...m, [h]: e.target.value }))}
                      className="flex-1 bg-slate-700/60 border border-slate-600 rounded px-2 py-1.5 text-xs text-slate-100 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                    >
                      <option value="">— skip —</option>
                      {STANDARD_FIELDS.map((f) => (
                        <option key={f} value={f}>{f}</option>
                      ))}
                    </select>
                  </div>
                ))}
              </div>

              {/* Preview */}
              {rows[0] && (
                <div>
                  <p className="text-xs text-slate-500 mb-1">First row preview:</p>
                  <div className="bg-slate-700/40 rounded p-2 text-[10px] font-mono text-slate-400 overflow-x-auto whitespace-nowrap">
                    {Object.entries(rows[0]).slice(0, 6).map(([k, v]) => `${k}: ${v}`).join(" | ")}
                  </div>
                </div>
              )}

              <div className="flex gap-3">
                <button onClick={() => setStep("upload")} className="px-3 py-2 text-sm text-slate-400 hover:text-slate-200">← Back</button>
                <button
                  onClick={handleImport}
                  disabled={importing || !Object.values(mapping).includes("txn_ref") || !Object.values(mapping).includes("txn_date")}
                  className="flex-1 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm rounded-lg font-medium transition-colors"
                >
                  {importing ? "Importing…" : `Import ${rows.length} rows`}
                </button>
              </div>
              {!Object.values(mapping).includes("txn_ref") && (
                <p className="text-xs text-amber-400">txn_ref and txn_date mappings are required.</p>
              )}
            </div>
          )}

          {step === "done" && result && (
            <div className="space-y-4 text-center py-4">
              <div className={`w-14 h-14 rounded-full flex items-center justify-center mx-auto ${result.imported > 0 ? "bg-emerald-900/30 border border-emerald-500/30" : "bg-red-900/30 border border-red-500/30"}`}>
                {result.imported > 0 ? (
                  <ArrowLeftRight size={24} className="text-emerald-400" />
                ) : (
                  <ZapOff size={24} className="text-red-400" />
                )}
              </div>
              <div>
                <p className="text-slate-200 font-medium">{result.imported} transactions imported</p>
                {result.skipped > 0 && <p className="text-amber-400 text-sm">{result.skipped} rows skipped</p>}
              </div>
              {result.errors.length > 0 && (
                <div className="text-left bg-slate-700/40 rounded p-3 space-y-1 max-h-32 overflow-y-auto">
                  {result.errors.map((e, i) => (
                    <p key={i} className="text-xs text-red-300 font-mono">{e}</p>
                  ))}
                </div>
              )}
              <button onClick={onClose} className="px-6 py-2 bg-slate-700 hover:bg-slate-600 text-sm text-slate-200 rounded-lg transition-colors">
                Close
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ── Add Transaction Form ──────────────────────────────────────────────────────

const EMPTY_FORM = {
  txn_ref: "", txn_date: "", amount: "", currency: "INR", txn_type: "debit",
  channel: "unknown", narration: "", description: "",
  from_account_raw: "", to_account_raw: "", from_name_raw: "", to_name_raw: "",
  counterparty_bank: "", counterparty_country: "",
  is_flagged: false, flag_reason: "", risk_score: "0",
};

function AddTransactionForm({
  caseId,
  onDone,
  onCancel,
}: {
  caseId: string;
  onDone: () => void;
  onCancel: () => void;
}) {
  const [form, setForm] = useState(EMPTY_FORM);
  const [error, setError] = useState("");

  const mutation = useMutation({
    mutationFn: async () => {
      // Step 1: create transaction
      const txn = await api.post<TransactionResponse>(`/api/v1/cases/${caseId}/transactions`, {
        ...form,
        amount: parseFloat(form.amount) || 0,
        risk_score: parseInt(form.risk_score) || 0,
      });
      // Step 2: link it to the case
      await api.post(`/api/v1/cases/${caseId}/transactions/link`, {
        transaction_id: txn.id,
        relevance_note: "",
      });
    },
    onSuccess: () => { setError(""); onDone(); },
    onError: (e: APIError) => setError(e.message ?? "Failed to add transaction"),
  });

  const set = (field: string, value: string | boolean) =>
    setForm((f) => ({ ...f, [field]: value }));

  return (
    <div className="bg-slate-800/80 border border-cyan-500/30 rounded-xl p-5 space-y-4">
      <h3 className="text-sm font-semibold text-slate-200">Add Transaction</h3>
      <div className="grid grid-cols-2 gap-3">
        {[
          { label: "Reference *", field: "txn_ref", placeholder: "TXN123456" },
          { label: "Date *", field: "txn_date", placeholder: "YYYY-MM-DD", type: "date" },
          { label: "Amount *", field: "amount", placeholder: "0.00", type: "number" },
          { label: "Currency", field: "currency", placeholder: "INR" },
          { label: "From Account", field: "from_account_raw", placeholder: "XXXXXXXX1234" },
          { label: "From Name", field: "from_name_raw", placeholder: "Sender name" },
          { label: "To Account", field: "to_account_raw", placeholder: "XXXXXXXX5678" },
          { label: "To Name", field: "to_name_raw", placeholder: "Recipient name" },
          { label: "Bank", field: "counterparty_bank", placeholder: "SBI / HDFC" },
          { label: "Country", field: "counterparty_country", placeholder: "IN" },
        ].map(({ label, field, placeholder, type }) => (
          <div key={field}>
            <label className="block text-xs text-slate-400 mb-1">{label}</label>
            <input
              type={type ?? "text"}
              value={(form as Record<string, string | boolean>)[field] as string}
              onChange={(e) => set(field, e.target.value)}
              placeholder={placeholder}
              className="w-full bg-slate-700/60 border border-slate-600 rounded px-2.5 py-1.5 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            />
          </div>
        ))}
        <div>
          <label className="block text-xs text-slate-400 mb-1">Type</label>
          <select value={form.txn_type} onChange={(e) => set("txn_type", e.target.value)}
            className="w-full bg-slate-700/60 border border-slate-600 rounded px-2.5 py-1.5 text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-cyan-500">
            {["debit", "credit", "reversal", "fee", "interest"].map((v) => (
              <option key={v} value={v}>{v}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-xs text-slate-400 mb-1">Channel</label>
          <select value={form.channel} onChange={(e) => set("channel", e.target.value)}
            className="w-full bg-slate-700/60 border border-slate-600 rounded px-2.5 py-1.5 text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-cyan-500">
            {["neft", "rtgs", "imps", "upi", "cash", "card", "swift", "internal", "unknown"].map((v) => (
              <option key={v} value={v}>{CHANNEL_LABELS[v] ?? v}</option>
            ))}
          </select>
        </div>
      </div>
      <div>
        <label className="block text-xs text-slate-400 mb-1">Narration</label>
        <input value={form.narration} onChange={(e) => set("narration", e.target.value)}
          placeholder="Raw bank narration text"
          className="w-full bg-slate-700/60 border border-slate-600 rounded px-2.5 py-1.5 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-cyan-500" />
      </div>
      <label className="flex items-center gap-2 cursor-pointer">
        <input type="checkbox" checked={form.is_flagged} onChange={(e) => set("is_flagged", e.target.checked)}
          className="w-4 h-4 rounded accent-amber-500" />
        <span className="text-xs text-slate-300">Flag as suspicious</span>
      </label>
      {form.is_flagged && (
        <input value={form.flag_reason} onChange={(e) => set("flag_reason", e.target.value)}
          placeholder="Reason for flagging"
          className="w-full bg-slate-700/60 border border-slate-600 rounded px-2.5 py-1.5 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-amber-500" />
      )}
      {error && <p className="text-xs text-red-400">{error}</p>}
      <div className="flex gap-2">
        <button onClick={() => mutation.mutate()} disabled={mutation.isPending || !form.txn_ref || !form.txn_date || !form.amount}
          className="flex-1 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm rounded-lg font-medium transition-colors">
          {mutation.isPending ? "Adding…" : "Add Transaction"}
        </button>
        <button onClick={onCancel} className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-sm text-slate-300 rounded-lg transition-colors">
          Cancel
        </button>
      </div>
    </div>
  );
}

// ── Main Page ─────────────────────────────────────────────────────────────────

import { useCaseSummary } from "@/lib/hooks/useCaseSummary";

export default function TransactionsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: caseId } = use(params);
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const cfg = useConfig();

  const caseData = useCaseSummary(caseId);

  const [view, setView] = useState<"list" | "flow">("list");
  const [search, setSearch] = useState("");
  const [filterFlagged, setFilterFlagged] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
  const [showImport, setShowImport] = useState(false);

  const { data: listData, isLoading } = useQuery<TransactionResponse[]>({
    queryKey: ["transactions", caseId],
    queryFn: () => api.get(`/api/v1/cases/${caseId}/transactions?limit=200`),
  });

  const { data: summary } = useQuery<TransactionSummary>({
    queryKey: ["transactions-summary", caseId],
    queryFn: () => api.get(`/api/v1/cases/${caseId}/transactions/summary`),
  });

  const { data: flowGraph } = useQuery<FlowGraph>({
    queryKey: ["transactions-flow", caseId],
    enabled: view === "flow",
    queryFn: () => api.get(`/api/v1/cases/${caseId}/transactions/flow`),
  });

  const flagMutation = useMutation({
    mutationFn: ({ id, flagged }: { id: string; flagged: boolean }) =>
      api.patch(`/api/v1/cases/${caseId}/transactions/${id}`, { is_flagged: flagged }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["transactions", caseId] }),
  });

  const removeMutation = useMutation({
    mutationFn: (id: string) =>
      api.delete(`/api/v1/cases/${caseId}/transactions/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["transactions", caseId] });
      qc.invalidateQueries({ queryKey: ["transactions-summary", caseId] });
      qc.invalidateQueries({ queryKey: ["transactions-flow", caseId] });
    },
  });

  const invalidateAll = () => {
    qc.invalidateQueries({ queryKey: ["transactions", caseId] });
    qc.invalidateQueries({ queryKey: ["transactions-summary", caseId] });
    qc.invalidateQueries({ queryKey: ["transactions-flow", caseId] });
  };

  const transactions = listData ?? [];
  const filtered = transactions.filter((t) => {
    if (filterFlagged && !t.is_flagged) return false;
    if (search) {
      const q = search.toLowerCase();
      return (
        t.txn_ref.toLowerCase().includes(q) ||
        t.from_name_raw.toLowerCase().includes(q) ||
        t.to_name_raw.toLowerCase().includes(q) ||
        t.narration.toLowerCase().includes(q) ||
        t.from_account_raw.toLowerCase().includes(q) ||
        t.to_account_raw.toLowerCase().includes(q)
      );
    }
    return true;
  });

  return (
    <div className="p-6">
      <CaseSubNav
        caseId={caseId}
        activeTab="transactions"
        caseNumber={caseData?.case_number}
        action={
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowImport(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-700/60 hover:bg-slate-700 text-slate-300 text-xs rounded-lg transition-colors border border-slate-600/50"
            >
              <Upload size={13} /> Import CSV
            </button>
            <button
              onClick={() => setShowAdd(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-medium rounded-lg transition-colors"
            >
              <Plus size={13} /> Add Transaction
            </button>
          </div>
        }
      />

      <div className="mb-6">
        <h1 className="text-xl font-bold text-white">Transactions</h1>
        <p className="text-slate-400 text-sm mt-0.5">Financial evidence — link transactions to this case and trace the flow of funds.</p>
      </div>

      {/* Summary bar */}
      {summary && <SummaryBar summary={summary} />}

        {/* Add form */}
        {showAdd && (
          <div className="mb-5">
            <AddTransactionForm
              caseId={caseId}
              onDone={() => { setShowAdd(false); invalidateAll(); }}
              onCancel={() => setShowAdd(false)}
            />
          </div>
        )}

        {/* View toggle + filters */}
        <div className="flex items-center gap-3 mb-4 flex-wrap">
          <div className="flex items-center bg-slate-800 border border-slate-700 rounded-lg p-0.5">
            {(["list", "flow"] as const).map((v) => (
              <button
                key={v}
                onClick={() => setView(v)}
                className={`px-3 py-1.5 text-xs rounded-md transition-colors ${
                  view === v ? "bg-slate-700 text-slate-200" : "text-slate-500 hover:text-slate-300"
                }`}
              >
                {v === "list" ? "Transaction List" : "Flow of Funds"}
              </button>
            ))}
          </div>

          {view === "list" && (
            <>
              <div className="relative flex-1 max-w-xs">
                <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-500" />
                <input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search by ref, name, account, narration…"
                  className="w-full bg-slate-800 border border-slate-700 rounded-lg pl-8 pr-3 py-1.5 text-xs text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                />
              </div>
              <button
                onClick={() => setFilterFlagged((v) => !v)}
                className={`flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg border transition-colors ${
                  filterFlagged
                    ? "bg-amber-900/30 border-amber-500/50 text-amber-300"
                    : "border-slate-700 text-slate-500 hover:text-slate-300"
                }`}
              >
                <Flag size={12} /> Flagged only
              </button>
            </>
          )}

          <span className="text-xs text-slate-500 ml-auto">
            {filtered.length} of {transactions.length}
          </span>
        </div>

        {/* Content */}
        {view === "list" ? (
          <div className="space-y-2">
            {isLoading && (
              <div className="space-y-2">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="h-14 bg-slate-800/40 border border-slate-700/50 rounded-lg animate-pulse" />
                ))}
              </div>
            )}
            {!isLoading && filtered.length === 0 && (
              <div className="text-center py-16 text-slate-500">
                <ArrowLeftRight size={32} className="mx-auto mb-3 opacity-30" />
                <p className="text-sm">No transactions linked to this case yet.</p>
                <p className="text-xs mt-1">Add individual transactions or import from CSV.</p>
              </div>
            )}
            {filtered.map((txn) => (
              <TxnRow
                key={txn.id}
                txn={txn}
                onFlag={(id, flagged) => flagMutation.mutate({ id, flagged })}
                onRemove={(id) => removeMutation.mutate(id)}
                ctrThreshold={cfg.ctr_threshold ?? 0}
              />
            ))}
          </div>
        ) : flowGraph ? (
          <FlowOfFundsGraph graph={flowGraph} currency={cfg.local_currency} />
        ) : (
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl h-32 flex items-center justify-center text-slate-500 text-sm animate-pulse">
            Building flow graph…
          </div>
        )}

      {/* CSV Import Modal */}
      {showImport && (
        <CSVImportModal
          caseId={caseId}
          onClose={() => setShowImport(false)}
          onDone={invalidateAll}
        />
      )}
    </div>
  );
}
