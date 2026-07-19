"use client";

import { use, useState, useEffect, useMemo, useCallback, memo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { useCaseNav } from "@/lib/contexts/CaseNavContext";
import { useConfig } from "@/lib/hooks/useConfig";
import { CheckCircle, XCircle, AlertTriangle, ChevronDown, ChevronUp, Brain, Zap, BookOpen, Wand2, Loader2, ShieldCheck } from "lucide-react";
import { formatEmiratesID, emiratesIDError } from "@/lib/utils/emirates-id";

interface STRFiling {
  id: string;
  case_id: string;
  report_type: string;
  status: "draft" | "submitted" | "approved" | "rejected" | "filed";
  suspicion_basis: string;
  narrative: string;
  evidence_summary: string;
  submitted_by?: string;
  submitted_by_name?: string;
  approved_by?: string;
  approved_by_name?: string;
  submitted_at?: string;
  approved_at?: string;
  filed_at?: string;
  fiu_reference?: string;
  rejection_reason?: string;
  filing_entity: Record<string, unknown>;
  subject: Record<string, unknown>;
  transactions: unknown[];
  created_at: string;
  updated_at: string;
}

interface GoAMLValidationError {
  field: string;
  message: string;
}

interface GoAMLReadiness {
  ready: boolean;
  errors: GoAMLValidationError[];
}

// ── Intelligence types ───────────────────────────────────────────────────────

interface Typology {
  id: string;
  code: string;
  name: string;
  category: string;
  description: string;
  risk_keywords: string[];
}

interface NarrativeCheck {
  key: string;
  label: string;
  pass: boolean;
  tip: string;
}

interface NarrativeResult {
  score: number;
  word_count: number;
  checks: NarrativeCheck[];
}

interface TxSignal {
  severity: "high" | "medium" | "low";
  label: string;
  detail: string;
}

interface ScoredTypology extends Typology {
  match_score: number;
}

// ── Auto-populate types ──────────────────────────────────────────────────────

interface CaseSubjectRef {
  id: string;
  canonical_name: string;
  id_type: string;
  id_number: string;
  date_of_birth?: string;
  phones: string[];
  role: string;
}

interface CaseNarrative {
  id: string;
  status: string;
  active_body: string;
  narrative_type: string;
}

interface CaseInsights {
  risk_score: number | null;
  risk_level: string;
  risk_factors: string[];
  anomalies: string[];
}

interface CaseTransaction {
  id: string;
  txn_date: string;
  amount: number;
  currency: string;
  channel: string;
  description: string;
  reference: string;
  from_account_raw: string;
  to_account_raw: string;
  from_account?: { account_number: string };
  to_account?: { account_number: string };
}

interface AutoPopulateResult {
  subject?: CaseSubjectRef;
  narrative?: CaseNarrative;
  insights?: CaseInsights;
  transactions: CaseTransaction[];
}

async function fetchCaseContext(caseId: string): Promise<AutoPopulateResult> {
  const [subjectsRes, narrativesRes, insightsRes, txnsRes] = await Promise.allSettled([
    fetch(`/api/v1/cases/${caseId}/subjects`, { credentials: "include" }),
    fetch(`/api/v1/cases/${caseId}/narratives`, { credentials: "include" }),
    fetch(`/api/v1/cases/${caseId}/insights`, { credentials: "include" }),
    fetch(`/api/v1/cases/${caseId}/transactions?limit=20`, { credentials: "include" }),
  ]);

  let subject: CaseSubjectRef | undefined;
  if (subjectsRes.status === "fulfilled" && subjectsRes.value.ok) {
    const body = await subjectsRes.value.json() as { data: CaseSubjectRef[] };
    subject = (body.data ?? [])[0];
  }

  let narrative: CaseNarrative | undefined;
  if (narrativesRes.status === "fulfilled" && narrativesRes.value.ok) {
    const body = await narrativesRes.value.json() as { data: CaseNarrative[] };
    const narrs = body.data ?? [];
    narrative = narrs.find((n) => n.status === "approved") ?? narrs.find((n) => n.active_body?.trim()) ?? narrs[0];
  }

  let insights: CaseInsights | undefined;
  if (insightsRes.status === "fulfilled" && insightsRes.value.ok) {
    const body = await insightsRes.value.json() as { data: CaseInsights };
    insights = body.data;
  }

  let transactions: CaseTransaction[] = [];
  if (txnsRes.status === "fulfilled" && txnsRes.value.ok) {
    const body = await txnsRes.value.json() as { data: { transactions?: CaseTransaction[]; items?: CaseTransaction[] } };
    transactions = body.data?.transactions ?? body.data?.items ?? [];
  }

  return { subject, narrative, insights, transactions };
}

function channelToTransmode(channel: string): string {
  const c = (channel ?? "").toUpperCase();
  if (c === "CASH") return "C";
  if (c === "WIRE" || c === "SWIFT") return "W";
  if (c === "TRANSFER" || c === "NEFT" || c === "RTGS" || c === "IMPS") return "T";
  if (c === "DEPOSIT") return "D";
  return "X";
}

// ── Intelligence helpers ─────────────────────────────────────────────────────

function scoreNarrative(narrative: string, basis: string): NarrativeResult {
  const combined = `${basis} ${narrative}`.toLowerCase();
  const wordCount = narrative.trim().split(/\s+/).filter(Boolean).length;

  const checks: NarrativeCheck[] = [
    {
      key: "word_count",
      label: "Substantive narrative (≥ 100 words)",
      pass: wordCount >= 100,
      tip: `Currently ${wordCount} words. FIU expects ≥ 100 words of detailed suspicious activity description.`,
    },
    {
      key: "who",
      label: "Subject / entity identified",
      pass: /name|customer|individual|entity|company|firm|person|subject|account holder/.test(combined),
      tip: "Explicitly name the subject or entity whose behavior is suspicious.",
    },
    {
      key: "what",
      label: "Transaction specifics included",
      pass: /amount|transfer|deposit|withdrawal|payment|transaction|₹|rs\.|inr|usd|aed|kes|remittance|wire/.test(combined),
      tip: "Reference specific transaction amounts, types, and account numbers.",
    },
    {
      key: "when",
      label: "Time period / dates referenced",
      pass: /\d{4}|jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec|quarter|month|week|period|between/.test(combined),
      tip: "Specify when the suspicious activity occurred (dates or time window).",
    },
    {
      key: "why",
      label: "Suspicion indicators explained",
      pass: /unusual|suspicious|inconsistent|exceed|pattern|structur|layer|plac|smurfing|hawala|shell|nominee|offshore|high.risk|not consistent|no apparent|unexplained/.test(combined),
      tip: "Explain what specific behaviors or patterns triggered your suspicion.",
    },
  ];

  const passed = checks.filter((c) => c.pass).length;
  return { score: Math.round((passed / checks.length) * 100), word_count: wordCount, checks };
}

function detectTxSignals(transactions: unknown[], ctrThreshold: number): TxSignal[] {
  const txs = (Array.isArray(transactions) ? transactions : []) as Array<{
    amount?: number;
    date?: string;
    transmode_code?: string;
    from_account_number?: string;
    to_account_number?: string;
  }>;

  const signals: TxSignal[] = [];
  if (txs.length === 0) return signals;

  if (ctrThreshold > 0) {
    const structuring = txs.filter(
      (t) => (t.amount ?? 0) >= ctrThreshold * 0.8 && (t.amount ?? 0) < ctrThreshold,
    );
    if (structuring.length >= 2) {
      signals.push({
        severity: "high",
        label: "Structuring / smurfing pattern",
        detail: `${structuring.length} transactions between ${Math.round(ctrThreshold * 0.8).toLocaleString()} and ${ctrThreshold.toLocaleString()} — classic sub-threshold structuring indicative of smurfing.`,
      });
    }
  }

  const roundLarge = txs.filter((t) => (t.amount ?? 0) >= 500_000 && (t.amount ?? 0) % 100_000 === 0);
  if (roundLarge.length >= 2) {
    signals.push({
      severity: "medium",
      label: "Round-number transaction batch",
      detail: `${roundLarge.length} large round-number amounts detected — often associated with layering and placement.`,
    });
  }

  const wireNoCounterparty = txs.filter(
    (t) => (t.transmode_code === "W" || t.transmode_code === "T") && !t.to_account_number,
  );
  if (wireNoCounterparty.length > 0) {
    signals.push({
      severity: "low",
      label: "Missing counterparty on wire/transfer",
      detail: `${wireNoCounterparty.length} transfer transaction(s) lack a destination account number — required for FINnet/goAML XML export.`,
    });
  }

  const dateCounts: Record<string, number> = {};
  txs.forEach((t) => { if (t.date) dateCounts[t.date] = (dateCounts[t.date] ?? 0) + 1; });
  const hotDays = Object.values(dateCounts).filter((c) => c >= 3).length;
  if (hotDays > 0) {
    signals.push({
      severity: "medium",
      label: "High-frequency same-day activity",
      detail: "3 or more transactions on a single date — velocity indicator for ML placement or layering.",
    });
  }

  return signals;
}

function scoreTypologies(typologies: Typology[], text: string): ScoredTypology[] {
  const lower = text.toLowerCase();
  return typologies
    .map((t) => ({
      ...t,
      match_score: t.risk_keywords.filter((kw) => lower.includes(kw.toLowerCase())).length,
    }))
    .filter((t) => t.match_score > 0)
    .sort((a, b) => b.match_score - a.match_score)
    .slice(0, 3);
}

// ── STR Intelligence Panel ───────────────────────────────────────────────────

const SEVERITY_COLORS: Record<string, string> = {
  high:   "text-red-400 bg-red-500/10 border-red-500/30",
  medium: "text-amber-400 bg-amber-500/10 border-amber-500/30",
  low:    "text-slate-400 bg-slate-700/40 border-slate-600/40",
};

const CATEGORY_COLORS: Record<string, string> = {
  PLACEMENT:    "text-red-400 bg-red-500/10 border-red-500/20",
  LAYERING:     "text-orange-400 bg-orange-500/10 border-orange-500/20",
  INTEGRATION:  "text-amber-400 bg-amber-500/10 border-amber-500/20",
  STRUCTURING:  "text-yellow-400 bg-yellow-500/10 border-yellow-500/20",
  TF:           "text-violet-400 bg-violet-500/10 border-violet-500/20",
  PF:           "text-pink-400 bg-pink-500/10 border-pink-500/20",
};

const STATUS_COLORS: Record<string, string> = {
  draft:     "bg-slate-700 text-slate-300",
  submitted: "bg-amber-900/50 text-amber-300 border border-amber-500/30",
  approved:  "bg-emerald-900/50 text-emerald-300 border border-emerald-500/30",
  rejected:  "bg-red-900/50 text-red-300 border border-red-500/30",
  filed:     "bg-cyan-900/50 text-cyan-300 border border-cyan-500/30",
};

const REPORT_TYPE_LABELS: Record<string, string> = {
  STR: "Suspicious Transaction Report",
  SAR: "Suspicious Activity Report",
  CTR: "Cash Transaction Report",
};

const TRANSMODE_OPTIONS = [
  { value: "C", label: "C — Cash" },
  { value: "T", label: "T — Transfer" },
  { value: "W", label: "W — Wire Transfer" },
  { value: "D", label: "D — Deposit" },
  { value: "X", label: "X — Other" },
];

const ID_TYPE_LABELS: Record<string, string> = {
  PASSPORT:         "Passport",
  NATIONAL_ID:      "National ID",
  TAX_ID:           "Tax ID",
  DRIVING_LICENSE:  "Driving License",
  PAN:              "PAN Card",
  AADHAAR_REF:      "Aadhaar Ref",
  VOTER_ID:         "Voter ID",
  EMIRATES_ID:      "Emirates ID",
  UAE_TRADE_LICENSE:"Trade License",
  TRN:              "TRN",
  GCC_NATIONAL_ID:  "GCC National ID",
  RESIDENCY_VISA:   "Residency Visa",
  OTHER:            "Other",
};

// ── Pre-Flight Checklist Panel ───────────────────────────────────────────────
// Unified panel replacing separate GoAML + Intelligence panels.
// Sections: goAML Structural · Narrative Quality · Transaction Risk · FATF Match

function PreFlightChecklistPanel({
  filing, caseId, ctrThreshold,
}: {
  filing: STRFiling;
  caseId: string;
  ctrThreshold: number;
}) {
  const [open, setOpen] = useState(false);
  const [activeSection, setActiveSection] = useState<string | null>(null);

  const { data: goaml, isLoading: goamlLoading, refetch: refetchGoAML } = useQuery<GoAMLReadiness>({
    queryKey: ["goaml-readiness", filing.id],
    queryFn: () => api.get<GoAMLReadiness>(`/api/v1/cases/${caseId}/str/${filing.id}/validate/goaml`),
    enabled: open,
    staleTime: 0,
  });

  const { data: typologies = [] } = useQuery<Typology[]>({
    queryKey: ["typologies-all"],
    queryFn: () => api.get<Typology[]>("/api/v1/typologies?limit=100"),
    staleTime: 5 * 60 * 1000,
    enabled: open,
  });

  const narrative = useMemo(
    () => scoreNarrative(filing.narrative, filing.suspicion_basis),
    [filing.narrative, filing.suspicion_basis],
  );
  const txSignals = useMemo(
    () => detectTxSignals(filing.transactions, ctrThreshold),
    [filing.transactions, ctrThreshold],
  );
  const typologyMatches = useMemo(
    () => scoreTypologies(typologies, `${filing.suspicion_basis} ${filing.narrative}`),
    [typologies, filing.suspicion_basis, filing.narrative],
  );

  const goamlOk  = goaml?.ready === true;
  const narOk    = narrative.score >= 60;
  const txOk     = txSignals.filter((s) => s.severity === "high").length === 0;
  const typoOk   = typologyMatches.length > 0;
  const totalPass = [goamlOk, narOk, txOk, typoOk].filter(Boolean).length;
  const allGreen  = totalPass === 4;

  const toggle = (section: string) => {
    if (!open) { setOpen(true); refetchGoAML(); }
    setActiveSection((v) => (v === section ? null : section));
  };

  const SectionHeader = ({
    id, icon, label, ok, loadingState,
  }: {
    id: string; icon: React.ReactNode; label: string; ok: boolean | null; loadingState?: boolean;
  }) => (
    <button
      onClick={() => toggle(id)}
      className="w-full flex items-center justify-between px-3 py-2 hover:bg-slate-800/40 transition-colors text-xs"
    >
      <span className="flex items-center gap-2">
        {loadingState
          ? <Loader2 size={12} className="text-slate-500 animate-spin" />
          : ok === true
            ? <CheckCircle size={12} className="text-emerald-400" />
            : ok === false
              ? <XCircle size={12} className="text-red-400" />
              : <AlertTriangle size={12} className="text-amber-400" />}
        {icon}
        <span className="text-slate-300 font-medium">{label}</span>
      </span>
      {activeSection === id ? <ChevronUp size={11} className="text-slate-500" /> : <ChevronDown size={11} className="text-slate-500" />}
    </button>
  );

  return (
    <div className="border border-slate-700/50 rounded-lg overflow-hidden" data-testid="str-intelligence-panel">
      {/* Master toggle */}
      <button
        onClick={() => { setOpen((v) => { const next = !v; if (next) refetchGoAML(); return next; }); }}
        className="w-full flex items-center justify-between px-3 py-2.5 bg-slate-800/40 hover:bg-slate-800/70 transition-colors text-xs"
      >
        <span className="flex items-center gap-2">
          <ShieldCheck size={13} className={allGreen ? "text-emerald-400" : "text-amber-400"} />
          <span className="text-slate-300 font-medium">Pre-Flight Checklist</span>
          <span className="text-slate-500">—</span>
          <span className={allGreen ? "text-emerald-400" : "text-amber-400"}>
            {totalPass}/4 checks passed
          </span>
        </span>
        {open ? <ChevronUp size={13} className="text-slate-500" /> : <ChevronDown size={13} className="text-slate-500" />}
      </button>

      {open && (
        <div className="divide-y divide-slate-700/40 bg-slate-900/30">

          {/* ── Section 1: Filing Structural Completeness ── */}
          <div>
            <SectionHeader id="goaml" icon={<span className="text-slate-500">FINnet 2.0</span>} label="Structural Completeness" ok={goaml ? goamlOk : null} loadingState={goamlLoading} />
            {activeSection === "goaml" && (
              <div className="px-4 py-3 space-y-1.5 bg-slate-900/40">
                {goamlLoading && <p className="text-slate-500 text-xs flex items-center gap-1.5"><Loader2 size={11} className="animate-spin" /> Validating fields…</p>}
                {goaml?.ready && (
                  <p className="text-emerald-400 text-xs flex items-center gap-1.5">
                    <CheckCircle size={12} /> All required fields present — ready to export XML
                  </p>
                )}
                {goaml && !goaml.ready && (
                  <div className="space-y-1.5">
                    <p className="text-amber-400 text-xs font-medium">{goaml.errors.length} field{goaml.errors.length !== 1 ? "s" : ""} missing for XML export:</p>
                    {goaml.errors.map((e) => (
                      <div key={e.field} className="flex items-start gap-1.5 text-xs">
                        <XCircle size={11} className="text-red-400 flex-shrink-0 mt-0.5" />
                        <span className="text-slate-300">{e.message}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>

          {/* ── Section 2: Narrative Quality ── */}
          <div>
            <SectionHeader id="narrative" icon={<Brain size={11} className="text-slate-500" />} label={`Narrative Quality — ${narrative.score}%`} ok={narOk} />
            {activeSection === "narrative" && (
              <div className="px-4 py-3 space-y-2 bg-slate-900/40">
                <div className="h-1.5 rounded-full bg-slate-700/50 mb-3">
                  <div
                    className={`h-full rounded-full transition-all ${narrative.score >= 80 ? "bg-emerald-500" : narrative.score >= 60 ? "bg-amber-500" : "bg-red-500"}`}
                    style={{ width: `${narrative.score}%` }}
                  />
                </div>
                {narrative.checks.map((c) => (
                  <div key={c.key} className="flex items-start gap-2">
                    {c.pass
                      ? <CheckCircle size={11} className="text-emerald-400 flex-shrink-0 mt-0.5" />
                      : <XCircle size={11} className="text-red-400 flex-shrink-0 mt-0.5" />}
                    <div>
                      <span className={`text-xs ${c.pass ? "text-slate-400" : "text-slate-300"}`}>{c.label}</span>
                      {!c.pass && <p className="text-[11px] text-slate-500 mt-0.5">{c.tip}</p>}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* ── Section 3: Transaction Risk ── */}
          <div>
            <SectionHeader id="txrisk" icon={<Zap size={11} className="text-slate-500" />} label="Transaction Risk Signals" ok={txOk} />
            {activeSection === "txrisk" && (
              <div className="px-4 py-3 space-y-2 bg-slate-900/40">
                {txSignals.length === 0 ? (
                  <p className="text-xs text-slate-500 flex items-center gap-1.5">
                    <CheckCircle size={11} className="text-emerald-400" /> No anomalous patterns in transaction data
                  </p>
                ) : (
                  txSignals.map((s, i) => (
                    <div key={i} className={`flex items-start gap-2 text-xs px-2.5 py-2 rounded border ${SEVERITY_COLORS[s.severity]}`}>
                      <AlertTriangle size={11} className="flex-shrink-0 mt-0.5" />
                      <div>
                        <span className="font-medium">{s.label}</span>
                        <p className="text-[11px] opacity-80 mt-0.5">{s.detail}</p>
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>

          {/* ── Section 4: FATF Typology Match ── */}
          <div>
            <SectionHeader id="fatf" icon={<BookOpen size={11} className="text-slate-500" />} label="FATF Typology Match" ok={typoOk} />
            {activeSection === "fatf" && (
              <div className="px-4 py-3 space-y-2 bg-slate-900/40">
                {typologies.length === 0 ? (
                  <p className="text-xs text-slate-500 flex items-center gap-1.5"><Loader2 size={11} className="animate-spin" /> Loading typology library…</p>
                ) : typologyMatches.length === 0 ? (
                  <p className="text-xs text-amber-400 flex items-center gap-1.5">
                    <AlertTriangle size={11} /> No FATF typology keywords matched — reference a known ML/TF pattern in your narrative to strengthen the filing.
                  </p>
                ) : (
                  typologyMatches.map((t) => (
                    <div key={t.id} className="flex items-start gap-2 bg-slate-800/40 rounded px-2.5 py-2 border border-slate-700/30">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap mb-0.5">
                          <span className="text-xs text-slate-200 font-medium">{t.name}</span>
                          <span className={`text-[9px] px-1.5 py-0.5 rounded border font-medium ${CATEGORY_COLORS[t.category] ?? "text-slate-400 bg-slate-700/40 border-slate-600/40"}`}>
                            {t.category}
                          </span>
                          <span className="text-[10px] text-cyan-400 font-mono">{t.match_score} kw</span>
                        </div>
                        <p className="text-[11px] text-slate-500 leading-relaxed line-clamp-2">{t.description}</p>
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>

        </div>
      )}
    </div>
  );
}

// ── Transaction row input ────────────────────────────────────────────────────

interface TxRow {
  date: string;
  amount: string;
  transmode_code: string;
  description: string;
  transaction_number: string;
  transaction_location: string;
  from_account_number: string;
  from_institution: string;
  to_account_number: string;
  to_institution: string;
}

function emptyTx(): TxRow {
  return {
    date: "", amount: "", transmode_code: "C", description: "",
    transaction_number: "", transaction_location: "",
    from_account_number: "", from_institution: "",
    to_account_number: "", to_institution: "",
  };
}

function TransactionRows({
  rows, onChange, ctrThreshold, currency,
}: {
  rows: TxRow[];
  onChange: (rows: TxRow[]) => void;
  ctrThreshold: number;
  currency: string;
}) {
  const update = (i: number, field: keyof TxRow, val: string) => {
    const next = rows.map((r, idx) => idx === i ? { ...r, [field]: val } : r);
    onChange(next);
  };
  const add = () => onChange([...rows, emptyTx()]);
  const remove = (i: number) => onChange(rows.filter((_, idx) => idx !== i));

  return (
    <div className="col-span-2 space-y-3">
      <div className="flex items-center justify-between">
        <label className="text-slate-400 text-xs font-medium">
          Transactions <span className="text-red-400">*</span>
          <span className="text-slate-600 ml-1">(at least 1 required for XML export)</span>
        </label>
        <button
          type="button"
          onClick={add}
          className="text-xs text-cyan-400 hover:text-cyan-300 transition-colors"
        >
          + Add transaction
        </button>
      </div>

      {rows.length === 0 && (
        <p className="text-slate-600 text-xs italic">No transactions added yet.</p>
      )}

      {rows.map((tx, i) => (
        <div key={i} className="bg-slate-900/60 border border-slate-700/50 rounded-lg p-3 space-y-2">
          <div className="flex items-center justify-between mb-1">
            <span className="text-xs text-slate-500 font-medium">Transaction #{i + 1}</span>
            {rows.length > 1 && (
              <button
                type="button"
                onClick={() => remove(i)}
                className="text-xs text-red-400/60 hover:text-red-400 transition-colors"
              >
                Remove
              </button>
            )}
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-slate-500 text-[11px] mb-1 block">Date <span className="text-red-400">*</span></label>
              <input
                type="date"
                value={tx.date}
                onChange={(e) => update(i, "date", e.target.value)}
                className="w-full bg-slate-800 border border-slate-600 rounded px-2 py-1.5 text-xs text-white focus:outline-none focus:border-cyan-500"
              />
            </div>
            <div>
              <label className="text-slate-500 text-[11px] mb-1 block">
                Amount ({currency || "local currency"}) <span className="text-red-400">*</span>
                {ctrThreshold > 0 && parseFloat(tx.amount) >= ctrThreshold && (
                  <span className="ml-1.5 text-[10px] px-1.5 py-0.5 bg-red-900/50 text-red-300 border border-red-500/40 rounded-full font-medium animate-pulse">
                    ≥ CTR threshold ({currency} {ctrThreshold.toLocaleString()})
                  </span>
                )}
                {ctrThreshold > 0 && parseFloat(tx.amount) >= ctrThreshold * 0.9 && parseFloat(tx.amount) < ctrThreshold && (
                  <span className="ml-1.5 text-[10px] px-1.5 py-0.5 bg-amber-900/50 text-amber-300 border border-amber-500/40 rounded-full font-medium">
                    Near CTR threshold
                  </span>
                )}
              </label>
              <input
                type="number"
                value={tx.amount}
                onChange={(e) => update(i, "amount", e.target.value)}
                placeholder="e.g. 1500000"
                className="w-full bg-slate-800 border border-slate-600 rounded px-2 py-1.5 text-xs text-white placeholder-slate-600 focus:outline-none focus:border-cyan-500"
              />
            </div>
            <div>
              <label className="text-slate-500 text-[11px] mb-1 block">Mode <span className="text-red-400">*</span></label>
              <select
                value={tx.transmode_code}
                onChange={(e) => update(i, "transmode_code", e.target.value)}
                className="w-full bg-slate-800 border border-slate-600 rounded px-2 py-1.5 text-xs text-white focus:outline-none focus:border-cyan-500"
              >
                {TRANSMODE_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>{o.label}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-slate-500 text-[11px] mb-1 block">Transaction Number</label>
              <input
                type="text"
                value={tx.transaction_number}
                onChange={(e) => update(i, "transaction_number", e.target.value)}
                placeholder="e.g. TXN-A"
                className="w-full bg-slate-800 border border-slate-600 rounded px-2 py-1.5 text-xs text-white placeholder-slate-600 focus:outline-none focus:border-cyan-500"
              />
            </div>
            <div className="col-span-2">
              <label className="text-slate-500 text-[11px] mb-1 block">Description</label>
              <input
                type="text"
                value={tx.description}
                onChange={(e) => update(i, "description", e.target.value)}
                placeholder="e.g. Cash deposit at teller window"
                className="w-full bg-slate-800 border border-slate-600 rounded px-2 py-1.5 text-xs text-white placeholder-slate-600 focus:outline-none focus:border-cyan-500"
              />
            </div>
            <div>
              <label className="text-slate-500 text-[11px] mb-1 block">Branch / Location</label>
              <input
                type="text"
                value={tx.transaction_location}
                onChange={(e) => update(i, "transaction_location", e.target.value)}
                placeholder="e.g. Downtown Branch"
                className="w-full bg-slate-800 border border-slate-600 rounded px-2 py-1.5 text-xs text-white placeholder-slate-600 focus:outline-none focus:border-cyan-500"
              />
            </div>
            <div>
              <label className="text-slate-500 text-[11px] mb-1 block">From Account No.</label>
              <input
                type="text"
                value={tx.from_account_number}
                onChange={(e) => update(i, "from_account_number", e.target.value)}
                placeholder="Account number"
                className="w-full bg-slate-800 border border-slate-600 rounded px-2 py-1.5 text-xs text-white placeholder-slate-600 focus:outline-none focus:border-cyan-500 font-mono"
              />
            </div>
            <div>
              <label className="text-slate-500 text-[11px] mb-1 block">To Account No.</label>
              <input
                type="text"
                value={tx.to_account_number}
                onChange={(e) => update(i, "to_account_number", e.target.value)}
                placeholder="If transfer — destination account"
                className="w-full bg-slate-800 border border-slate-600 rounded px-2 py-1.5 text-xs text-white placeholder-slate-600 focus:outline-none focus:border-cyan-500 font-mono"
              />
            </div>
            <div>
              <label className="text-slate-500 text-[11px] mb-1 block">To Institution</label>
              <input
                type="text"
                value={tx.to_institution}
                onChange={(e) => update(i, "to_institution", e.target.value)}
                placeholder="Destination bank name"
                className="w-full bg-slate-800 border border-slate-600 rounded px-2 py-1.5 text-xs text-white placeholder-slate-600 focus:outline-none focus:border-cyan-500"
              />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// ── Filing card ──────────────────────────────────────────────────────────────

const FilingCard = memo(function FilingCard({
  filing, caseId, userId, userRole, onAction,
}: {
  filing: STRFiling; caseId: string; userId: string; userRole: string; onAction: () => void;
}) {
  const qc = useQueryClient();
  const cfg = useConfig();
  // Draft/submitted need attention — start expanded; approved/filed/rejected start collapsed
  const [expanded, setExpanded] = useState(filing.status === "draft" || filing.status === "submitted");
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [rejectReason, setRejectReason] = useState("");
  const [actionError, setActionError] = useState("");
  const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

  const isManagerOrAbove = ["manager", "admin", "super_admin"].includes(userRole);
  const isAdminOrAbove = ["admin", "super_admin"].includes(userRole);
  const canExport = isAdminOrAbove && (filing.status === "approved" || filing.status === "filed");

  const mutOpts = {
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["str-filings", caseId] }); onAction(); setActionError(""); },
    onError: (err: unknown) => setActionError(err instanceof APIError ? err.message : "Action failed"),
  };

  const submitMut  = useMutation({ mutationFn: () => api.post(`/api/v1/cases/${caseId}/str/${filing.id}/submit`, {}), ...mutOpts });
  const approveMut = useMutation({ mutationFn: () => api.post(`/api/v1/cases/${caseId}/str/${filing.id}/approve`, {}), ...mutOpts });
  const rejectMut  = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${caseId}/str/${filing.id}/reject`, { reason: rejectReason }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["str-filings", caseId] }); onAction(); setShowRejectModal(false); setRejectReason(""); setActionError(""); },
    onError: (err: unknown) => setActionError(err instanceof APIError ? err.message : "Reject failed"),
  });
  const [showFileForm, setShowFileForm] = useState(false);
  const [fiuRef, setFiuRef] = useState("");
  const filedMut = useMutation({
    mutationFn: (ref: string) => api.post(`/api/v1/cases/${caseId}/str/${filing.id}/file`, { fiu_reference: ref }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["str-filings", caseId] }); onAction(); setActionError(""); setShowFileForm(false); setFiuRef(""); },
    onError: (err: unknown) => setActionError(err instanceof APIError ? err.message : "Action failed"),
  });

  const handleExport = async (format: "finnet" | "goaml") => {
    const path = format === "goaml"
      ? `/api/v1/cases/${caseId}/str/${filing.id}/export/goaml`
      : `/api/v1/cases/${caseId}/str/${filing.id}/export`;
    const filename = format === "goaml"
      ? `goAML-STR-${filing.id.slice(0, 8)}.xml`
      : `${cfg.str_label}-${filing.id.slice(0, 8)}.xml`;

    try {
      const res = await fetch(`${apiBase}${path}`, { credentials: "include" });
      if (!res.ok) {
        // goAML export returns 422 with JSON validation errors
        if (res.status === 422) {
          const body = await res.json();
          // response.JSON wraps in {data: {...}} envelope
          const errs: GoAMLValidationError[] = body?.data?.errors ?? body?.errors ?? [];
          const msgs = errs.length > 0
            ? errs.map((e) => e.message).join("; ")
            : (body?.data?.message ?? body?.error?.message ?? "Filing is missing required fields — open the Pre-Flight Checklist to see what needs to be filled in");
          setActionError(`Export validation failed: ${msgs}`);
          return;
        }
        throw new Error("Export failed");
      }
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Export failed");
    }
  };

  const isSelf = filing.submitted_by === userId;

  return (
    <div className="bg-slate-800/60 border border-slate-700/50 rounded-xl overflow-hidden">
      {/* Header — always visible, click to expand/collapse */}
      <button
        className="w-full flex items-start justify-between gap-4 px-5 py-4 text-left hover:bg-white/[0.02] transition-colors"
        onClick={() => setExpanded((v) => !v)}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1 flex-wrap">
            <span className="font-mono text-xs text-slate-500">{filing.id.slice(0, 8).toUpperCase()}</span>
            <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${STATUS_COLORS[filing.status]}`}>
              {filing.status.toUpperCase()}
            </span>
            <span className="text-xs text-slate-500 bg-slate-700/50 px-2 py-0.5 rounded">
              {REPORT_TYPE_LABELS[filing.report_type] ?? filing.report_type}
            </span>
          </div>
          <p className="text-white text-sm font-medium line-clamp-2">{filing.suspicion_basis}</p>
        </div>
        <div className="flex items-center gap-3 shrink-0">
          <span className="text-xs text-slate-500 whitespace-nowrap">
            {new Date(filing.created_at).toLocaleDateString()}
          </span>
          {expanded
            ? <ChevronUp size={14} className="text-slate-500" />
            : <ChevronDown size={14} className="text-slate-500" />
          }
        </div>
      </button>

      {/* Expanded body */}
      {expanded && (
        <div className="px-5 pb-5 flex flex-col gap-4 border-t border-slate-700/50 pt-4">
          {/* Narrative */}
          {filing.narrative && (
            <p className="text-slate-300 text-sm leading-relaxed">{filing.narrative}</p>
          )}

          {/* Maker-checker trail */}
          <div className="flex flex-wrap gap-4 text-xs text-slate-500">
            {filing.submitted_by_name && (
              <span>Submitted by <span className="text-slate-300">{filing.submitted_by_name}</span>
                {filing.submitted_at && <> · {new Date(filing.submitted_at).toLocaleString()}</>}
              </span>
            )}
            {filing.approved_by_name && (
              <span>Approved by <span className="text-slate-300">{filing.approved_by_name}</span>
                {filing.approved_at && <> · {new Date(filing.approved_at).toLocaleString()}</>}
              </span>
            )}
            {filing.fiu_reference && (
              <span>{cfg.str_reference_label}: <span className="text-cyan-300 font-mono">{filing.fiu_reference}</span></span>
            )}
            {filing.rejection_reason && (
              <span className="text-red-400">Rejected: {filing.rejection_reason}</span>
            )}
          </div>

          {/* Unified Pre-Flight Checklist — goAML + narrative + tx risk + FATF */}
          {(filing.status === "draft" || filing.status === "submitted") && (
            <PreFlightChecklistPanel filing={filing} caseId={caseId} ctrThreshold={cfg.ctr_threshold ?? 0} />
          )}

          {/* Error */}
          {actionError && (
            <p className="text-red-400 text-xs bg-red-900/20 border border-red-800/30 px-3 py-2 rounded-lg">{actionError}</p>
          )}

          {/* Actions */}
          {isManagerOrAbove && (
            <div className="flex flex-wrap gap-2 pt-1 border-t border-slate-700/50">
              {filing.status === "draft" && (
                <button
                  onClick={() => submitMut.mutate()}
                  disabled={submitMut.isPending}
                  className="text-xs px-3 py-1.5 bg-amber-600/20 hover:bg-amber-600/30 text-amber-300 border border-amber-500/30 rounded-lg transition-colors disabled:opacity-60"
                >
                  {submitMut.isPending ? "Submitting…" : "Submit for Approval"}
                </button>
              )}

              {filing.status === "submitted" && !isSelf && (
                <>
                  <button
                    onClick={() => approveMut.mutate()}
                    disabled={approveMut.isPending}
                    className="text-xs px-3 py-1.5 bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-300 border border-emerald-500/30 rounded-lg transition-colors disabled:opacity-60"
                  >
                    {approveMut.isPending ? "Approving…" : "Approve"}
                  </button>
                  <button
                    onClick={() => setShowRejectModal(true)}
                    className="text-xs px-3 py-1.5 bg-red-600/20 hover:bg-red-600/30 text-red-300 border border-red-500/30 rounded-lg transition-colors"
                  >
                    Reject
                  </button>
                </>
              )}

              {filing.status === "submitted" && isSelf && (
                <span className="text-xs text-amber-400/70 italic py-1.5">
                  {cfg.str_law_reference ? `${cfg.str_law_reference}: ` : ""}You submitted this filing — a different officer must approve
                </span>
              )}

              {isAdminOrAbove && filing.status === "approved" && !showFileForm && (
                <button
                  onClick={() => setShowFileForm(true)}
                  className="text-xs px-3 py-1.5 bg-cyan-600/20 hover:bg-cyan-600/30 text-cyan-300 border border-cyan-500/30 rounded-lg transition-colors"
                >
                  Mark as Filed with {cfg.str_regulator_label}
                </button>
              )}
              {isAdminOrAbove && filing.status === "approved" && showFileForm && (
                <div className="flex items-center gap-2 flex-1 flex-wrap">
                  <input
                    type="text"
                    value={fiuRef}
                    onChange={(e) => setFiuRef(e.target.value)}
                    placeholder={`${cfg.str_reference_label ?? "FIU Reference"} (e.g. FIU-IND/STR/2024-25/MH/087341)`}
                    className="flex-1 min-w-48 bg-slate-800 border border-slate-600 rounded-lg px-3 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-cyan-500 font-mono"
                  />
                  <button
                    onClick={() => filedMut.mutate(fiuRef)}
                    disabled={filedMut.isPending}
                    className="text-xs px-3 py-1.5 bg-cyan-600/20 hover:bg-cyan-600/30 text-cyan-300 border border-cyan-500/30 rounded-lg transition-colors disabled:opacity-60 whitespace-nowrap"
                  >
                    {filedMut.isPending ? "Filing…" : "Confirm Filing"}
                  </button>
                  <button
                    onClick={() => { setShowFileForm(false); setFiuRef(""); }}
                    className="text-xs text-slate-500 hover:text-slate-300 transition-colors px-1"
                  >
                    Cancel
                  </button>
                </div>
              )}

              {canExport && (
                <div className="flex gap-2 ml-auto">
                  {cfg.jurisdiction === "india" ? (
                    <button
                      onClick={() => handleExport("finnet")}
                      className="text-xs px-3 py-1.5 bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-300 border border-emerald-500/30 rounded-lg transition-colors font-medium"
                      title="Export FINnet 2.0 XML for FIU-IND portal submission"
                    >
                      Export FINnet XML
                    </button>
                  ) : (
                    <button
                      onClick={() => handleExport("goaml")}
                      className="text-xs px-3 py-1.5 bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-300 border border-emerald-500/30 rounded-lg transition-colors font-medium"
                      title="Export goAML 4.0 XML for FIU portal submission"
                    >
                      Export goAML XML
                    </button>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Reject modal */}
      {showRejectModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-6 w-full max-w-md mx-4 shadow-2xl">
            <h3 className="text-white font-semibold mb-3">Reject {cfg.str_label} Filing</h3>
            <textarea
              rows={3}
              value={rejectReason}
              onChange={(e) => setRejectReason(e.target.value)}
              placeholder="Reason for rejection (optional)…"
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 resize-none focus:outline-none focus:border-cyan-500"
            />
            <div className="flex gap-2 mt-4 justify-end">
              <button
                onClick={() => { setShowRejectModal(false); setRejectReason(""); }}
                className="px-3 py-1.5 text-sm text-slate-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => rejectMut.mutate()}
                disabled={rejectMut.isPending}
                className="px-4 py-1.5 text-sm bg-red-600/20 hover:bg-red-600/40 text-red-300 border border-red-500/30 rounded-lg transition-colors disabled:opacity-60"
              >
                {rejectMut.isPending ? "Rejecting…" : "Confirm Reject"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
});

// ── Page ─────────────────────────────────────────────────────────────────────

import { useCaseSummary } from "@/lib/hooks/useCaseSummary";

export default function STRFilingsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: caseId } = use(params);
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const cfg = useConfig();

  const caseData = useCaseSummary(caseId);
  const { setAction } = useCaseNav();
  const noop = useCallback(() => {}, []);
  const [showCreate, setShowCreate] = useState(false);
  const [createError, setCreateError] = useState("");
  const [populating, setPopulating] = useState(false);
  const [populatedFrom, setPopulatedFrom] = useState<{ subjects: number; transactions: number; narrative: boolean; insights: boolean } | null>(null);

  // Basic fields
  const [reportType, setReportType] = useState("STR");
  const [suspicionBasis, setSuspicionBasis] = useState("");
  const [narrative, setNarrative] = useState("");
  const [evidenceSummary, setEvidenceSummary] = useState("");

  // Subject fields
  const [subjectFirstName, setSubjectFirstName] = useState("");
  const [subjectLastName, setSubjectLastName] = useState("");
  const [subjectGender, setSubjectGender] = useState("");
  const [subjectTitle, setSubjectTitle] = useState("");
  const [subjectBirthdate, setSubjectBirthdate] = useState("");
  const [subjectIDType, setSubjectIDType] = useState("NATIONAL_ID");
  const [subjectIDNumber, setSubjectIDNumber] = useState("");
  const [subjectAccountNumber, setSubjectAccountNumber] = useState("");
  const [subjectAccountName, setSubjectAccountName] = useState("");
  const [subjectClientNumber, setSubjectClientNumber] = useState("");

  // Filing entity / reporting officer fields
  const [entityName, setEntityName] = useState("");
  const [rentityID, setRentityID] = useState("");
  const [rentityBranch, setRentityBranch] = useState("");
  const [swiftCode, setSwiftCode] = useState("");
  const [submissionCode, setSubmissionCode] = useState("E");
  const [officerTitle, setOfficerTitle] = useState("Mr");
  const [officerFirstName, setOfficerFirstName] = useState("");
  const [officerLastName, setOfficerLastName] = useState("");
  const [officerGender, setOfficerGender] = useState("M");
  const [officerIDNumber, setOfficerIDNumber] = useState("");
  const [officerIDType, setOfficerIDType] = useState("NATIONAL_ID");
  const [officerPhone, setOfficerPhone] = useState("");
  const [officerEmail, setOfficerEmail] = useState("");
  const [officerOccupation, setOfficerOccupation] = useState("Chief Compliance Officer");
  const [entityAddress, setEntityAddress] = useState("");
  const [entityCity, setEntityCity] = useState("");
  const [entityState, setEntityState] = useState("");
  const [entityZip, setEntityZip] = useState("");
  const [entityCountry, setEntityCountry] = useState("");
  const [fiuName, setFiuName] = useState("");
  const [localCurrency, setLocalCurrency] = useState("");

  // Transactions
  const [txRows, setTxRows] = useState<TxRow[]>([emptyTx()]);

  // Suspicion date — CBUAE Art. 15: 30-day clock starts here, NOT at STR creation
  const [suspicionDate, setSuspicionDate] = useState("");

  // goAML section toggle
  const [showGoAMLFields, setShowGoAMLFields] = useState(false);

  // Seed localCurrency from config once loaded
  useEffect(() => {
    if (cfg.local_currency && !localCurrency) {
      setLocalCurrency(cfg.local_currency);
    }
  }, [cfg.local_currency]); // eslint-disable-line react-hooks/exhaustive-deps

  const { data: filingData, isLoading, isError } = useQuery<{ filings: STRFiling[] }>({
    queryKey: ["str-filings", caseId],
    queryFn: () => api.get<{ filings: STRFiling[] }>(`/api/v1/cases/${caseId}/str`),
    enabled: !!caseId,
    retry: 1,
  });

  const filings = filingData?.filings ?? [];

  const buildPayload = () => ({
    report_type: reportType,
    suspicion_date: suspicionDate || undefined,
    suspicion_basis: suspicionBasis,
    narrative,
    evidence_summary: evidenceSummary,
    filing_entity: {
      name: entityName,
      rentity_id: rentityID,
      rentity_branch: rentityBranch,
      swift_code: swiftCode,
      submission_code: submissionCode,
      officer_title: officerTitle,
      officer_first_name: officerFirstName,
      officer_last_name: officerLastName,
      officer_gender: officerGender,
      officer_id_number: officerIDNumber,
      officer_id_type: officerIDType,
      officer_phone: officerPhone,
      officer_email: officerEmail,
      officer_occupation: officerOccupation,
      entity_address: entityAddress,
      entity_city: entityCity,
      entity_state: entityState,
      entity_zip: entityZip,
      entity_country: entityCountry,
      fiu_name: fiuName,
      local_currency: localCurrency,
    },
    subject: {
      first_name: subjectFirstName,
      last_name: subjectLastName,
      name: `${subjectFirstName} ${subjectLastName}`.trim(),
      gender: subjectGender,
      title: subjectTitle,
      birthdate: subjectBirthdate,
      id_type: subjectIDType,
      id_number: subjectIDNumber,
      pan: subjectIDType === "PAN" ? subjectIDNumber : "",
      account_number: subjectAccountNumber,
      account_name: subjectAccountName,
      client_number: subjectClientNumber,
    },
    transactions: txRows
      .filter((t) => t.date && parseFloat(t.amount) > 0)
      .map((t) => ({
        date: t.date,
        amount: parseFloat(t.amount) || 0,
        currency: localCurrency || "",
        description: t.description,
        transaction_number: t.transaction_number,
        transaction_location: t.transaction_location,
        transmode_code: t.transmode_code,
        from_account_number: t.from_account_number,
        from_institution: t.from_institution || entityName,
        to_account_number: t.to_account_number,
        to_institution: t.to_institution,
      })),
  });

  const resetForm = () => {
    setSuspicionDate(""); setSuspicionBasis(""); setNarrative(""); setEvidenceSummary("");
    setSubjectFirstName(""); setSubjectLastName(""); setSubjectGender(""); setSubjectTitle("");
    setSubjectBirthdate(""); setSubjectIDType("NATIONAL_ID"); setSubjectIDNumber("");
    setSubjectAccountNumber(""); setSubjectAccountName(""); setSubjectClientNumber("");
    setEntityName(""); setRentityID(""); setRentityBranch(""); setSwiftCode(""); setSubmissionCode("E");
    setOfficerTitle("Mr"); setOfficerFirstName(""); setOfficerLastName(""); setOfficerGender("M");
    setOfficerIDNumber(""); setOfficerIDType("NATIONAL_ID"); setOfficerPhone(""); setOfficerEmail("");
    setOfficerOccupation("Chief Compliance Officer");
    setEntityAddress(""); setEntityCity(""); setEntityState(""); setEntityZip("");
    setEntityCountry(""); setFiuName(""); setLocalCurrency("");
    setTxRows([emptyTx()]);
    setShowGoAMLFields(false);
    setCreateError("");
    setPopulatedFrom(null);
  };

  const handleAutoPopulate = async () => {
    setPopulating(true);
    try {
      const ctx = await fetchCaseContext(caseId);

      let subjectCount = 0;
      if (ctx.subject) {
        subjectCount = 1;
        const parts = ctx.subject.canonical_name.trim().split(/\s+/);
        setSubjectFirstName(parts.slice(0, -1).join(" ") || parts[0] || "");
        setSubjectLastName(parts.length > 1 ? parts[parts.length - 1] : "");
        setSubjectIDType(ctx.subject.id_type || "NATIONAL_ID");
        setSubjectIDNumber(ctx.subject.id_number || "");
        if (ctx.subject.date_of_birth) setSubjectBirthdate(ctx.subject.date_of_birth);
        if (ctx.subject.phones?.[0]) setSubjectAccountNumber(ctx.subject.phones[0]);
      }

      if (ctx.narrative?.active_body?.trim()) {
        setNarrative(ctx.narrative.active_body.trim());
      }

      if (ctx.insights) {
        const factors = ctx.insights.risk_factors ?? [];
        const anomalies = ctx.insights.anomalies ?? [];
        const basisParts = [
          factors.slice(0, 3).join("; "),
          anomalies.slice(0, 2).join("; "),
        ].filter(Boolean);
        if (basisParts.length > 0) setSuspicionBasis(basisParts.join(". "));
        if (ctx.insights.risk_level) {
          setEvidenceSummary(`Risk level: ${ctx.insights.risk_level}${ctx.insights.risk_score != null ? ` (score: ${ctx.insights.risk_score})` : ""}. ${factors.join("; ")}`);
        }
      }

      const mappedTxs: typeof txRows = ctx.transactions.slice(0, 10).map((t) => ({
        date: t.txn_date ?? "",
        amount: String(t.amount ?? ""),
        transmode_code: channelToTransmode(t.channel),
        description: t.description ?? "",
        transaction_number: t.reference ?? "",
        transaction_location: "",
        from_account_number: t.from_account?.account_number ?? t.from_account_raw ?? "",
        from_institution: "",
        to_account_number: t.to_account?.account_number ?? t.to_account_raw ?? "",
        to_institution: "",
      }));
      if (mappedTxs.length > 0) setTxRows(mappedTxs);

      setPopulatedFrom({
        subjects: subjectCount,
        transactions: ctx.transactions.length,
        narrative: !!ctx.narrative?.active_body?.trim(),
        insights: !!ctx.insights,
      });
    } catch {
      // Silently fall back — form remains empty, user fills manually
    } finally {
      setPopulating(false);
    }
  };

  const createMut = useMutation({
    mutationFn: () => api.post<STRFiling>(`/api/v1/cases/${caseId}/str`, buildPayload()),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["str-filings", caseId] });
      setShowCreate(false);
      resetForm();
    },
    onError: (err) => {
      setCreateError(err instanceof APIError ? err.message : "Failed to create filing");
    },
  });

  const isManagerOrAbove = user && ["manager", "admin", "super_admin"].includes(user.role);

  useEffect(() => {
    if (!isManagerOrAbove) { setAction(null); return; }
    setAction(
      <button
        onClick={() => setShowCreate(true)}
        className="flex items-center gap-1.5 px-3 py-1.5 bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-medium rounded-lg transition-colors"
      >
        + New Filing
      </button>
    );
    return () => setAction(null);
  }, [isManagerOrAbove, setAction]);

  if (!isManagerOrAbove) {
    return (
      <div className="px-6 pb-6">
        <div className="bg-red-900/20 border border-red-800/30 rounded-xl p-8 text-center">
          <p className="text-red-400 font-medium">Access Restricted</p>
          <p className="text-slate-500 text-sm mt-1">Regulatory filings are restricted to compliance officers (manager role and above).</p>
        </div>
      </div>
    );
  }

  const inputCls = "w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-cyan-500";
  const labelCls = "text-slate-400 text-xs mb-1.5 block";

  return (
    <div className="px-6 pb-6">
      <div className="mb-6">
        <h1 className="text-xl font-bold text-white">Regulatory Filings</h1>
        <p className="text-slate-400 text-sm mt-0.5">
          {[cfg.str_law_reference, "Dual-authorization maker-checker", cfg.jurisdiction === "india" ? "FINnet 2.0 (FIU-IND)" : "goAML 4.0 (UNODC)"].filter(Boolean).join(" · ")}
        </p>
      </div>

      {/* Compliance banner */}
      <div className="bg-amber-900/10 border border-amber-800/20 rounded-xl p-4 mb-6 text-xs text-amber-300/80 leading-relaxed">
        {cfg.str_law_reference && <><strong className="text-amber-300">{cfg.str_law_reference}</strong> — </>}
        Records must be retained for 10 years.
        Maker-checker: submitted by one officer, approved by a different officer.
        Export <strong className="text-amber-300">{cfg.jurisdiction === "india" ? "FINnet XML" : "goAML XML"}</strong> for direct upload to your FIU portal.
        Analysts are excluded from this workflow (tipping-off restriction).
      </div>

      {/* Create form */}
      {showCreate && (
        <div className="bg-slate-800/80 border border-slate-700 rounded-xl p-6 mb-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-white font-semibold">New Regulatory Filing Draft</h2>
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={handleAutoPopulate}
                disabled={populating}
                data-testid="auto-populate-btn"
                className="flex items-center gap-1.5 px-3 py-1.5 bg-violet-600/20 hover:bg-violet-600/30 text-violet-300 border border-violet-500/30 rounded-lg text-xs font-medium transition-colors disabled:opacity-60"
              >
                {populating
                  ? <Loader2 size={12} className="animate-spin" />
                  : <Wand2 size={12} />}
                {populating ? "Populating…" : "Auto-populate from Case"}
              </button>
              <button onClick={() => { setShowCreate(false); resetForm(); }} className="text-slate-400 hover:text-white text-lg leading-none">×</button>
            </div>
          </div>

          {populatedFrom && (
            <div className="flex items-center gap-2 mb-4 px-3 py-2 bg-violet-500/10 border border-violet-500/20 rounded-lg text-xs text-violet-300">
              <Wand2 size={11} />
              <span>
                Auto-populated from case:
                {populatedFrom.subjects > 0 && <> · <strong>{populatedFrom.subjects} subject</strong></>}
                {populatedFrom.transactions > 0 && <> · <strong>{populatedFrom.transactions} transaction{populatedFrom.transactions !== 1 ? "s" : ""}</strong></>}
                {populatedFrom.narrative && <> · <strong>AI narrative</strong></>}
                {populatedFrom.insights && <> · <strong>risk insights</strong></>}
              </span>
              <span className="text-violet-400/60 ml-1">— review and adjust before saving</span>
            </div>
          )}

          <div className="grid grid-cols-2 gap-4">
            {/* ── Basic fields ── */}
            <div>
              <label className={labelCls}>Report Type</label>
              <select value={reportType} onChange={(e) => setReportType(e.target.value)} className={inputCls}>
                <option value="STR">STR — Suspicious Transaction Report</option>
              </select>
            </div>

            <div>
              <label className={labelCls}>Submission Type</label>
              <select value={submissionCode} onChange={(e) => setSubmissionCode(e.target.value)} className={inputCls}>
                <option value="E">E — New (first submission)</option>
                <option value="S">S — Supplementary (additional info)</option>
                <option value="C">C — Correction (amends previous)</option>
              </select>
            </div>

            {/* Suspicion Date — starts the regulatory filing clock */}
            <div>
              <label className={labelCls}>
                Date Suspicion Arose <span className="text-red-400">*</span>
                <span className="text-slate-500 font-normal ml-1">({cfg.str_law_reference || "regulatory"} — {cfg.str_deadline_days || 30}-day clock)</span>
              </label>
              <input
                type="date"
                value={suspicionDate}
                max={new Date().toISOString().split("T")[0]}
                onChange={(e) => setSuspicionDate(e.target.value)}
                className={inputCls}
              />
              {suspicionDate && (() => {
                const d = new Date(suspicionDate);
                d.setDate(d.getDate() + (cfg.str_deadline_days || 30));
                const deadline = d.toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" });
                const today = new Date();
                const daysLeft = Math.ceil((d.getTime() - today.getTime()) / 86400000);
                return (
                  <p className={`text-[11px] mt-1 flex items-center gap-1 ${daysLeft < 0 ? "text-red-400" : daysLeft <= 7 ? "text-amber-400" : "text-slate-500"}`}>
                    {daysLeft < 0
                      ? `⚠ Filing deadline was ${deadline} — overdue by ${Math.abs(daysLeft)} day${Math.abs(daysLeft) !== 1 ? "s" : ""}`
                      : `Filing deadline: ${deadline} (${daysLeft} day${daysLeft !== 1 ? "s" : ""} remaining)`}
                  </p>
                );
              })()}
            </div>
            <div /> {/* grid spacer */}

            <div className="col-span-2">
              <label className={labelCls}>Suspicion Basis <span className="text-red-400">*</span></label>
              <input
                type="text"
                value={suspicionBasis}
                onChange={(e) => setSuspicionBasis(e.target.value)}
                placeholder="e.g. Unusual cash transactions inconsistent with customer profile"
                className={inputCls}
              />
            </div>

            <div className="col-span-2">
              <label className={labelCls}>Narrative / Reason <span className="text-red-400">*</span></label>
              <textarea
                rows={3}
                value={narrative}
                onChange={(e) => setNarrative(e.target.value)}
                placeholder="Detailed description of suspicious activity…"
                className={`${inputCls} resize-none`}
              />
            </div>

            <div className="col-span-2">
              <label className={labelCls}>Evidence Summary</label>
              <textarea
                rows={2}
                value={evidenceSummary}
                onChange={(e) => setEvidenceSummary(e.target.value)}
                placeholder="Supporting evidence (transaction records, documents, KYC anomalies)…"
                className={`${inputCls} resize-none`}
              />
            </div>

            {/* ── Subject ── */}
            <div className="col-span-2">
              <div className="border-t border-slate-700/50 pt-4 mb-3">
                <p className="text-slate-300 text-xs font-semibold uppercase tracking-widest">Subject</p>
              </div>
            </div>

            <div>
              <label className={labelCls}>First Name <span className="text-red-400">*</span></label>
              <input type="text" value={subjectFirstName} onChange={(e) => setSubjectFirstName(e.target.value)} placeholder="First name" className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Last Name</label>
              <input type="text" value={subjectLastName} onChange={(e) => setSubjectLastName(e.target.value)} placeholder="Last name" className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>{cfg.str_id_label || "ID Type"}</label>
              <select value={subjectIDType} onChange={(e) => setSubjectIDType(e.target.value)} className={inputCls}>
                {(cfg.id_types ?? []).map((t) => (
                  <option key={t} value={t}>{ID_TYPE_LABELS[t] ?? t}</option>
                ))}
              </select>
            </div>
            <div>
              <label className={labelCls}>ID Number <span className="text-red-400">*</span></label>
              <input
                type="text"
                value={subjectIDNumber}
                onChange={(e) => {
                  const raw = subjectIDType === "EMIRATES_ID"
                    ? formatEmiratesID(e.target.value)
                    : e.target.value.toUpperCase();
                  setSubjectIDNumber(raw);
                }}
                placeholder={
                  subjectIDType === "PAN" ? "ABCDE1234F"
                  : subjectIDType === "EMIRATES_ID" ? "784-YYYY-XXXXXXX-X"
                  : "Identifier value"
                }
                className={`${inputCls} font-mono uppercase`}
              />
              {subjectIDType === "EMIRATES_ID" && subjectIDNumber && (
                <p className={`text-[11px] mt-0.5 ${emiratesIDError(subjectIDNumber) ? "text-red-400" : "text-emerald-400"}`}>
                  {emiratesIDError(subjectIDNumber) ?? "✓ Valid Emirates ID"}
                </p>
              )}
            </div>
            <div>
              <label className={labelCls}>Title</label>
              <select value={subjectTitle} onChange={(e) => setSubjectTitle(e.target.value)} className={inputCls}>
                <option value="">— select —</option>
                <option value="Mr">Mr</option>
                <option value="Ms">Ms</option>
                <option value="Dr">Dr</option>
              </select>
            </div>
            <div>
              <label className={labelCls}>Gender</label>
              <select value={subjectGender} onChange={(e) => setSubjectGender(e.target.value)} className={inputCls}>
                <option value="">— select —</option>
                <option value="M">M — Male</option>
                <option value="F">F — Female</option>
                <option value="U">U — Undisclosed</option>
              </select>
            </div>
            <div>
              <label className={labelCls}>Date of Birth</label>
              <input type="date" value={subjectBirthdate} max={new Date().toISOString().split("T")[0]} onChange={(e) => setSubjectBirthdate(e.target.value)} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Account Number</label>
              <input type="text" value={subjectAccountNumber} onChange={(e) => setSubjectAccountNumber(e.target.value)} placeholder="Primary account" className={`${inputCls} font-mono`} />
            </div>
            <div>
              <label className={labelCls}>Account Name</label>
              <input type="text" value={subjectAccountName} onChange={(e) => setSubjectAccountName(e.target.value)} placeholder="Account holder name" className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Customer / Client ID</label>
              <input type="text" value={subjectClientNumber} onChange={(e) => setSubjectClientNumber(e.target.value)} placeholder="Internal customer ID" className={inputCls} />
            </div>

            {/* ── Transactions ── */}
            <div className="col-span-2">
              <div className="border-t border-slate-700/50 pt-4 mb-3">
                <p className="text-slate-300 text-xs font-semibold uppercase tracking-widest">Transactions</p>
              </div>
            </div>
            <TransactionRows
              rows={txRows}
              onChange={setTxRows}
              ctrThreshold={cfg.ctr_threshold ?? 0}
              currency={localCurrency || cfg.local_currency || ""}
            />

            {/* ── Reporting officer / goAML fields (collapsible) ── */}
            <div className="col-span-2">
              <div className="border-t border-slate-700/50 pt-4">
                <button
                  type="button"
                  onClick={() => setShowGoAMLFields((v) => !v)}
                  className="flex items-center gap-2 text-xs font-semibold text-slate-300 hover:text-white uppercase tracking-widest transition-colors"
                >
                  {showGoAMLFields ? <ChevronUp size={13} /> : <ChevronDown size={13} />}
                  Reporting Officer &amp; Entity Details
                  <span className="text-slate-500 font-normal normal-case tracking-normal ml-1">(required for goAML export)</span>
                </button>
              </div>
            </div>

            {showGoAMLFields && (
              <>
                <div>
                  <label className={labelCls}>Filing Entity / Bank Name</label>
                  <input type="text" value={entityName} onChange={(e) => setEntityName(e.target.value)} placeholder="Your bank / NBFC name" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>FIU Registration ID <span className="text-red-400">*</span></label>
                  <input
                    type="text"
                    value={rentityID}
                    onChange={(e) => setRentityID(e.target.value)}
                    placeholder="e.g. FIU-IN-BANK-0001, UAEFIU-BANK-001"
                    className={`${inputCls} font-mono`}
                  />
                </div>
                <div>
                  <label className={labelCls}>Branch Identifier</label>
                  <input type="text" value={rentityBranch} onChange={(e) => setRentityBranch(e.target.value)} placeholder="e.g. BRANCH-MUM-001" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>SWIFT Code</label>
                  <input type="text" value={swiftCode} onChange={(e) => setSwiftCode(e.target.value.toUpperCase())} placeholder="e.g. TESTINBB" className={`${inputCls} font-mono uppercase`} />
                </div>
                <div>
                  <label className={labelCls}>Entity Country <span className="text-slate-500 font-normal">(ISO 2-letter)</span></label>
                  <input type="text" maxLength={2} value={entityCountry} onChange={(e) => setEntityCountry(e.target.value.toUpperCase())} placeholder="e.g. IN, AE, KE, US" className={`${inputCls} font-mono uppercase`} />
                </div>
                <div>
                  <label className={labelCls}>FIU Name <span className="text-slate-500 font-normal">(optional)</span></label>
                  <input type="text" value={fiuName} onChange={(e) => setFiuName(e.target.value)} placeholder="e.g. FIU-IND, UAEFIU, FIU Kenya" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>Local Currency <span className="text-slate-500 font-normal">(ISO 4217)</span></label>
                  <input type="text" maxLength={3} value={localCurrency} onChange={(e) => setLocalCurrency(e.target.value.toUpperCase())} placeholder="e.g. INR, AED, KES, USD" className={`${inputCls} font-mono uppercase`} />
                </div>

                <div className="col-span-2">
                  <p className="text-slate-500 text-xs font-medium uppercase tracking-wider mt-2 mb-2">Reporting Officer</p>
                </div>

                <div>
                  <label className={labelCls}>Title</label>
                  <select value={officerTitle} onChange={(e) => setOfficerTitle(e.target.value)} className={inputCls}>
                    <option value="Mr">Mr</option>
                    <option value="Ms">Ms</option>
                    <option value="Dr">Dr</option>
                  </select>
                </div>
                <div>
                  <label className={labelCls}>Gender</label>
                  <select value={officerGender} onChange={(e) => setOfficerGender(e.target.value)} className={inputCls}>
                    <option value="M">M — Male</option>
                    <option value="F">F — Female</option>
                    <option value="U">U — Undisclosed</option>
                  </select>
                </div>
                <div>
                  <label className={labelCls}>First Name <span className="text-red-400">*</span></label>
                  <input type="text" value={officerFirstName} onChange={(e) => setOfficerFirstName(e.target.value)} placeholder="Officer first name" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>Last Name <span className="text-red-400">*</span></label>
                  <input type="text" value={officerLastName} onChange={(e) => setOfficerLastName(e.target.value)} placeholder="Officer last name" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>ID Type</label>
                  <select value={officerIDType} onChange={(e) => setOfficerIDType(e.target.value)} className={inputCls}>
                    {(cfg.id_types ?? []).map((t) => (
                      <option key={t} value={t}>{ID_TYPE_LABELS[t] ?? t}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className={labelCls}>Officer ID Number <span className="text-red-400">*</span></label>
                  <input
                    type="text"
                    value={officerIDNumber}
                    onChange={(e) => setOfficerIDNumber(e.target.value.toUpperCase())}
                    placeholder="National ID / Tax ID"
                    className={`${inputCls} font-mono uppercase`}
                  />
                </div>
                <div>
                  <label className={labelCls}>Phone</label>
                  <input type="text" value={officerPhone} onChange={(e) => setOfficerPhone(e.target.value)} placeholder="+1-555-0100" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>Email</label>
                  <input type="email" value={officerEmail} onChange={(e) => setOfficerEmail(e.target.value)} placeholder="officer@bank.com" className={inputCls} />
                </div>
                <div className="col-span-2">
                  <label className={labelCls}>Designation / Occupation</label>
                  <input type="text" value={officerOccupation} onChange={(e) => setOfficerOccupation(e.target.value)} placeholder="Chief Compliance Officer" className={inputCls} />
                </div>

                <div className="col-span-2">
                  <p className="text-slate-500 text-xs font-medium uppercase tracking-wider mt-2 mb-2">Entity Address</p>
                </div>
                <div className="col-span-2">
                  <label className={labelCls}>Street Address</label>
                  <input type="text" value={entityAddress} onChange={(e) => setEntityAddress(e.target.value)} placeholder="Head office address" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>City</label>
                  <input type="text" value={entityCity} onChange={(e) => setEntityCity(e.target.value)} placeholder="City" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>State / Province</label>
                  <input type="text" value={entityState} onChange={(e) => setEntityState(e.target.value)} placeholder="State / Province" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>Postal Code</label>
                  <input type="text" value={entityZip} onChange={(e) => setEntityZip(e.target.value)} placeholder="Postal code" className={`${inputCls} font-mono`} />
                </div>
              </>
            )}
          </div>

          {createError && (
            <p className="text-red-400 text-xs mt-3 bg-red-900/20 border border-red-800/30 px-3 py-2 rounded-lg">{createError}</p>
          )}

          <div className="flex justify-end gap-2 mt-5">
            <button
              onClick={() => { setShowCreate(false); resetForm(); }}
              className="px-4 py-2 text-sm text-slate-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={() => createMut.mutate()}
              disabled={!suspicionBasis.trim() || !narrative.trim() || createMut.isPending}
              className="px-4 py-2 text-sm bg-cyan-600/20 hover:bg-cyan-600/30 text-cyan-300 border border-cyan-500/30 rounded-lg transition-colors disabled:opacity-60"
            >
              {createMut.isPending ? "Saving…" : "Save Draft"}
            </button>
          </div>
        </div>
      )}

      {/* Filings list */}
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 2 }).map((_, i) => (
            <div key={i} className="h-40 bg-slate-800/40 rounded-xl animate-pulse" />
          ))}
        </div>
      ) : isError ? (
        <div className="text-center py-16 text-slate-500">
          <p className="text-4xl mb-3">⚠️</p>
          <p className="font-medium text-red-400">Could not load filings</p>
          <p className="text-sm mt-1">The {cfg.str_label} service is temporarily unavailable. Try refreshing the page.</p>
        </div>
      ) : filings.length === 0 ? (
        <div className="text-center py-16 text-slate-500">
          <p className="text-4xl mb-3">📋</p>
          <p className="font-medium">No filings yet</p>
          <p className="text-sm mt-1">Create a new draft to start the {cfg.str_label} workflow.</p>
        </div>
      ) : (
        <div className="flex flex-col gap-4 pb-24">
          {filings.map((f) => (
            <FilingCard
              key={f.id}
              filing={f}
              caseId={caseId}
              userId={user?.id ?? ""}
              userRole={user?.role ?? ""}
              onAction={noop}
            />
          ))}
        </div>
      )}
    </div>
  );
}
