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

function fmtAmount(amount: number, currency = "EUR") {
  return formatCurrency(amount, currency);
}

function fmtDate(iso: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("en-IN", { day: "2-digit", month: "short", year: "numeric" });
}

const CHANNEL_LABELS: Record<string, string> = {
  sepa:     "SEPA Credit Transfer",
  target2:  "TARGET2 (RTGS)",
  swift:    "SWIFT / Cross-border Wire",
  chaps:    "CHAPS",
  neft:     "NEFT",
  rtgs:     "RTGS",
  imps:     "IMPS",
  upi:      "UPI",
  cash:     "Cash",
  card:     "Card",
  internal: "Internal Transfer",
  unknown:  "—",
};

const TXN_TYPE_STYLES: Record<string, string> = {
  debit:    "text-red-400",
  credit:   "text-emerald-400",
  reversal: "text-amber-400",
  fee:      "text-slate-400",
  interest: "text-cyan-400",
};

// ── Summary Stats ─────────────────────────────────────────────────────────────

function SummaryBar({ summary, currency }: { summary: TransactionSummary; currency: string }) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 mb-5">
      {[
        { label: "Transactions", value: summary.total_count, fmt: (v: number) => v.toString() },
        { label: "Total Flow", value: summary.total_amount, fmt: (v: number) => fmtAmount(v, currency) },
        { label: "Debits", value: summary.debit_total, fmt: (v: number) => fmtAmount(v, currency), color: "text-red-400" },
        { label: "Credits", value: summary.credit_total, fmt: (v: number) => fmtAmount(v, currency), color: "text-emerald-400" },
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
              Remove from Evidence
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
  "is_flagged", "flag_reason", "risk_score", "external_id",
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
              {(result.errors?.length ?? 0) > 0 && (
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

// IBAN: 15–34 alphanumeric chars, starts with 2-letter country code
const IBAN_RE = /^[A-Z]{2}[0-9]{2}[A-Z0-9]{11,30}$/;
// BIC/SWIFT: 8 or 11 alphanumeric chars
const BIC_RE = /^[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?$/;

function ibanError(val: string): string {
  if (!val) return "";
  const clean = val.replace(/\s/g, "").toUpperCase();
  if (!IBAN_RE.test(clean)) return "Invalid IBAN format (e.g. GR16 0110 1250 0000 0001 2300 695)";
  return "";
}
function bicError(val: string): string {
  if (!val) return "";
  const clean = val.replace(/\s/g, "").toUpperCase();
  if (!BIC_RE.test(clean)) return "Invalid BIC/SWIFT (e.g. ETHNGRAA or BARCGB22)";
  return "";
}

const EU_CHANNEL_ORDER = ["sepa", "target2", "swift", "chaps", "cash", "card", "internal", "unknown"];
const IN_CHANNEL_ORDER = ["neft", "rtgs", "imps", "upi", "cash", "card", "swift", "internal", "unknown"];

function AddTransactionForm({
  caseId,
  defaultCurrency,
  defaultCountry,
  jurisdiction,
  onDone,
  onCancel,
}: {
  caseId: string;
  defaultCurrency: string;
  defaultCountry: string;
  jurisdiction: string;
  onDone: () => void;
  onCancel: () => void;
}) {
  const isEU = jurisdiction === "eu" || jurisdiction === "europe";
  const emptyForm = {
    txn_ref: "", txn_date: "", amount: "", currency: defaultCurrency || "EUR", txn_type: "debit",
    channel: isEU ? "sepa" : "unknown", narration: "", description: "",
    from_account_raw: "", to_account_raw: "", from_name_raw: "", to_name_raw: "",
    counterparty_bank: "", counterparty_country: (defaultCountry || "").toUpperCase(),
    bic_swift: "",
    is_flagged: false, flag_reason: "", risk_score: "0",
  };

  const [form, setForm] = useState(emptyForm);
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const submittingRef = useRef(false);

  const set = (field: string, value: string | boolean) => {
    setForm((f) => ({ ...f, [field]: value }));
    setFieldErrors((e) => ({ ...e, [field]: "" }));
  };

  const validate = (): boolean => {
    const errs: Record<string, string> = {};
    if (!form.txn_ref.trim()) errs.txn_ref = "Reference is required";
    if (!form.txn_date) errs.txn_date = "Date is required";
    if (!form.amount || isNaN(parseFloat(form.amount))) errs.amount = "Valid amount required";
    if (parseFloat(form.amount) <= 0) errs.amount = "Amount must be positive";
    if (isEU) {
      const fromErr = form.from_account_raw ? ibanError(form.from_account_raw) : "";
      if (fromErr) errs.from_account_raw = fromErr;
      const toErr = form.to_account_raw ? ibanError(form.to_account_raw) : "";
      if (toErr) errs.to_account_raw = toErr;
      const bErr = form.bic_swift ? bicError(form.bic_swift) : "";
      if (bErr) errs.bic_swift = bErr;
    }
    setFieldErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const mutation = useMutation({
    mutationFn: async () => {
      const { bic_swift, ...rest } = form;
      await api.post<TransactionResponse>(`/api/v1/cases/${caseId}/transactions`, {
        ...rest,
        counterparty_bank: form.counterparty_bank || (bic_swift ? `BIC: ${bic_swift}` : ""),
        amount: parseFloat(form.amount) || 0,
        risk_score: parseInt(form.risk_score) || 0,
      });
    },
    onSuccess: () => { setError(""); submittingRef.current = false; onDone(); },
    onError: (e: APIError) => { submittingRef.current = false; setError(e.message ?? "Failed to add transaction"); },
  });

  const handleSubmit = () => {
    if (submittingRef.current || mutation.isPending) return;
    if (!validate()) return;
    submittingRef.current = true;
    mutation.mutate();
  };

  const inputCls = (field: string) =>
    `w-full bg-slate-700/60 border ${fieldErrors[field] ? "border-red-500" : "border-slate-600"} rounded px-2.5 py-1.5 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-cyan-500`;

  const channelOrder = isEU ? EU_CHANNEL_ORDER : IN_CHANNEL_ORDER;
  const countryPlaceholder = defaultCountry ? defaultCountry.toUpperCase() : "e.g. GR";
  const bankPlaceholder = isEU
    ? "e.g. National Bank of Greece / Alpha Bank"
    : "e.g. SBI / HDFC";
  const refPlaceholder = isEU ? "e.g. SEPA2026062400001" : "TXN123456";
  const ibanPlaceholder = isEU ? "GR16 0110 1250 0000 0001 2300 695" : "Account number";

  return (
    <div className="bg-slate-800/80 border border-cyan-500/30 rounded-xl p-5 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-slate-200">Add Transaction</h3>
        {isEU && (
          <span className="text-[10px] bg-blue-900/40 text-blue-300 border border-blue-500/30 px-2 py-0.5 rounded-full font-medium">
            EU / SEPA · IBAN validated
          </span>
        )}
      </div>

      <div className="grid grid-cols-2 gap-3">
        {/* Reference */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">Reference <span className="text-red-400">*</span></label>
          <input type="text" value={form.txn_ref} onChange={(e) => set("txn_ref", e.target.value)}
            placeholder={refPlaceholder} className={inputCls("txn_ref")} />
          {fieldErrors.txn_ref && <p className="text-[10px] text-red-400 mt-0.5">{fieldErrors.txn_ref}</p>}
        </div>

        {/* Date */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">Date <span className="text-red-400">*</span></label>
          <input type="date" value={form.txn_date} onChange={(e) => set("txn_date", e.target.value)}
            max={new Date().toISOString().split("T")[0]}
            className={inputCls("txn_date")} />
          {fieldErrors.txn_date && <p className="text-[10px] text-red-400 mt-0.5">{fieldErrors.txn_date}</p>}
        </div>

        {/* Amount */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">Amount <span className="text-red-400">*</span></label>
          <div className="relative">
            <span className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-500 text-sm select-none">
              {defaultCurrency || "EUR"}
            </span>
            <input type="number" min="0.01" step="0.01" value={form.amount}
              onChange={(e) => set("amount", e.target.value)}
              placeholder="0.00"
              className={`${inputCls("amount")} pl-12`} />
          </div>
          {fieldErrors.amount && <p className="text-[10px] text-red-400 mt-0.5">{fieldErrors.amount}</p>}
        </div>

        {/* Currency — read-only display from config */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">Currency</label>
          <input type="text" value={form.currency} readOnly
            className="w-full bg-slate-900/60 border border-slate-700 rounded px-2.5 py-1.5 text-sm text-slate-400 cursor-not-allowed" />
        </div>

        {/* From Account (IBAN for EU) */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">
            From Account {isEU && <span className="text-slate-600">(IBAN)</span>}
          </label>
          <input type="text" value={form.from_account_raw}
            onChange={(e) => set("from_account_raw", e.target.value.replace(/\s/g, "").toUpperCase())}
            placeholder={ibanPlaceholder} className={inputCls("from_account_raw")} />
          {fieldErrors.from_account_raw && <p className="text-[10px] text-red-400 mt-0.5">{fieldErrors.from_account_raw}</p>}
        </div>

        {/* From Name */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">From Name / Payer</label>
          <input type="text" value={form.from_name_raw} onChange={(e) => set("from_name_raw", e.target.value)}
            placeholder="e.g. Aegean Holdings SA" className={inputCls("from_name_raw")} />
        </div>

        {/* To Account (IBAN for EU) */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">
            To Account {isEU && <span className="text-slate-600">(IBAN)</span>}
          </label>
          <input type="text" value={form.to_account_raw}
            onChange={(e) => set("to_account_raw", e.target.value.replace(/\s/g, "").toUpperCase())}
            placeholder={ibanPlaceholder} className={inputCls("to_account_raw")} />
          {fieldErrors.to_account_raw && <p className="text-[10px] text-red-400 mt-0.5">{fieldErrors.to_account_raw}</p>}
        </div>

        {/* To Name */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">To Name / Beneficiary</label>
          <input type="text" value={form.to_name_raw} onChange={(e) => set("to_name_raw", e.target.value)}
            placeholder="e.g. Hellenic Commerce Ltd" className={inputCls("to_name_raw")} />
        </div>

        {/* Bank */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">
            {"Counterparty Bank"}
          </label>
          <input type="text" value={form.counterparty_bank} onChange={(e) => set("counterparty_bank", e.target.value)}
            placeholder={bankPlaceholder} className={inputCls("counterparty_bank")} />
        </div>

        {/* BIC/SWIFT — EU only */}
        {isEU ? (
          <div>
            <label className="block text-xs text-slate-400 mb-1">BIC / SWIFT</label>
            <input type="text" value={form.bic_swift}
              onChange={(e) => set("bic_swift", e.target.value.replace(/\s/g, "").toUpperCase())}
              placeholder="e.g. ETHNGRAA" className={inputCls("bic_swift")} maxLength={11} />
            {fieldErrors.bic_swift && <p className="text-[10px] text-red-400 mt-0.5">{fieldErrors.bic_swift}</p>}
          </div>
        ) : (
          <div>
            <label className="block text-xs text-slate-400 mb-1">Country</label>
            <input type="text" value={form.counterparty_country} onChange={(e) => set("counterparty_country", e.target.value.toUpperCase())}
              placeholder={countryPlaceholder} maxLength={2} className={inputCls("counterparty_country")} />
          </div>
        )}

        {/* Country — EU shows separately after BIC row */}
        {isEU && (
          <div>
            <label className="block text-xs text-slate-400 mb-1">Counterparty Country</label>
            <input type="text" value={form.counterparty_country} onChange={(e) => set("counterparty_country", e.target.value.toUpperCase())}
              placeholder={countryPlaceholder} maxLength={2} className={inputCls("counterparty_country")} />
          </div>
        )}

        {/* Transaction Type */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">Type</label>
          <select value={form.txn_type} onChange={(e) => set("txn_type", e.target.value)}
            className="w-full bg-slate-700/60 border border-slate-600 rounded px-2.5 py-1.5 text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-cyan-500">
            {["debit", "credit", "reversal", "fee", "interest"].map((v) => (
              <option key={v} value={v}>{v.charAt(0).toUpperCase() + v.slice(1)}</option>
            ))}
          </select>
        </div>

        {/* Channel */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">Channel / Rail</label>
          <select value={form.channel} onChange={(e) => set("channel", e.target.value)}
            className="w-full bg-slate-700/60 border border-slate-600 rounded px-2.5 py-1.5 text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-cyan-500">
            {channelOrder.map((v) => (
              <option key={v} value={v}>{CHANNEL_LABELS[v] ?? v}</option>
            ))}
          </select>
        </div>
      </div>

      <div>
        <label className="block text-xs text-slate-400 mb-1">Narration / Description</label>
        <input value={form.narration} onChange={(e) => set("narration", e.target.value)}
          placeholder={isEU ? "e.g. SEPA CT /ROC/INV-2026-0042 Aegean Holdings" : "Raw bank narration text"}
          className="w-full bg-slate-700/60 border border-slate-600 rounded px-2.5 py-1.5 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-cyan-500" />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="flex items-center gap-2 col-span-2 sm:col-span-1">
          <input type="checkbox" checked={form.is_flagged} onChange={(e) => set("is_flagged", e.target.checked)}
            className="w-4 h-4 rounded accent-amber-500 cursor-pointer" />
          <span className="text-xs text-slate-300">Flag as suspicious</span>
        </div>
        <div>
          <label className="block text-xs text-slate-400 mb-1">
            Txn Risk Score <span className="text-slate-600">(0–100)</span>
            <span className="relative ml-1 group inline-block align-middle cursor-help">
              <span className="text-slate-600 hover:text-slate-400 text-[10px] border border-slate-600 rounded-full px-1">?</span>
              <span className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-2 w-56 bg-slate-900 border border-slate-700 rounded-lg p-2.5 text-[10px] text-slate-300 opacity-0 group-hover:opacity-100 transition-opacity z-50 shadow-xl leading-relaxed">
                <span className="block font-semibold text-slate-200 mb-1">Scoring Rubric</span>
                <span className="block text-emerald-400">0–39 · Low</span>
                <span className="block text-amber-400">40–69 · Medium</span>
                <span className="block text-orange-400">70–84 · High</span>
                <span className="block text-red-400">85–100 · Critical</span>
                <span className="block mt-1.5 text-slate-500 font-semibold uppercase tracking-wide text-[9px]">Add points for:</span>
                <span className="block">+20 · Amount &gt;10× monthly income</span>
                <span className="block">+15 · Round amount pattern</span>
                <span className="block">+15 · Small Finance Bank counterparty</span>
                <span className="block">+15 · Cash / SWIFT channel</span>
                <span className="block">+25 · High-risk jurisdiction</span>
                <span className="block mt-1 text-slate-500 italic">TMS-imported scores override manual entry.</span>
              </span>
            </span>
          </label>
          <input type="number" min="0" max="100" value={form.risk_score}
            onChange={(e) => set("risk_score", e.target.value)}
            placeholder="0"
            className={inputCls("risk_score")} />
        </div>
      </div>
      {form.is_flagged && (
        <input value={form.flag_reason} onChange={(e) => set("flag_reason", e.target.value)}
          placeholder="Reason for flagging"
          className="w-full bg-slate-700/60 border border-amber-500/50 rounded px-2.5 py-1.5 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-amber-500" />
      )}
      {error && <p className="text-xs text-red-400">{error}</p>}
      <div className="flex gap-2">
        <button onClick={handleSubmit} disabled={mutation.isPending}
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

  const isCaseClosed = caseData?.status === "closed";
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
    <div className="flex flex-col h-full overflow-hidden">

      {/* ── Fixed header: subnav + title + summary + filters ── */}
      <div className="flex-shrink-0 px-6 pt-6 pb-3">
        <CaseSubNav
          caseId={caseId}
          activeTab="transactions"
          caseNumber={caseData?.case_number}
          action={
            !isCaseClosed ? (
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
            ) : undefined
          }
        />

        <div className="mb-5">
          <h1 className="text-xl font-bold text-white">Transactions</h1>
          <p className="text-slate-400 text-sm mt-0.5">Financial evidence — link transactions to this case and trace the flow of funds.</p>
        </div>

        {summary && <SummaryBar summary={summary} currency={cfg.local_currency || "EUR"} />}

        {showAdd && (
          <div className="mb-4">
            <AddTransactionForm
              caseId={caseId}
              defaultCurrency={cfg.local_currency || "EUR"}
              defaultCountry={cfg.country || "GR"}
              jurisdiction={cfg.jurisdiction || "eu"}
              onDone={() => { setShowAdd(false); invalidateAll(); }}
              onCancel={() => setShowAdd(false)}
            />
          </div>
        )}

        {/* View toggle + filters */}
        <div className="flex items-center gap-3 flex-wrap">
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
      </div>

      {/* ── Scrollable content area ── */}
      {view === "list" ? (
        <div className="flex-1 min-h-0 overflow-y-auto px-6 pb-6 space-y-2">
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
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto px-6 pb-6">
          {flowGraph ? (
            <FlowOfFundsGraph graph={flowGraph} currency={cfg.local_currency} />
          ) : (
            <div className="bg-[#131f30] border border-[#1e3050] rounded-xl h-32 flex items-center justify-center text-slate-500 text-sm animate-pulse">
              Building flow graph…
            </div>
          )}
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
