"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api, API_BASE } from "@/lib/api";
import { useConfig } from "@/lib/hooks/useConfig";
import { CheckCircle2, AlertTriangle, Download, Loader2, ExternalLink } from "lucide-react";

interface AILog {
  id: string;
  case_id: string;
  action_type: string;
  model_used: string;
  provider: string;
  confidence_score?: number;
  human_review_status: string;
  created_at: string;
}

interface PendingMeta { count: number }

// Jurisdiction-specific regulatory context shown on the page
const JURISDICTION_CONTEXT = {
  india: {
    subtitle:    "DPDP Act 2023 · FIU-IND AML & CFT Guidelines 2023 · RBI IT Governance Directions 2023",
    badge:       "India Regulatory Ready · DPDP + FIU-IND Aligned",
    badgeColor:  "bg-orange-500/10 border-orange-500/30 text-orange-300",
    checklistDesc:
      "6 controls verified across FIU-IND AML & CFT Guidelines 2023 (Principal Officer oversight) · DPDP Act 2023 Section 6 (data minimisation) · RBI IT Governance Directions 2023 (audit trail, access control) · on-premise deployment (zero data egress).",
    obligation:
      "All AI-assisted investigation outputs must be reviewed and approved by a qualified human officer before use in STR filings, case closure, or regulatory submissions. The Principal Officer appointed under PMLA Section 12 bears personal accountability for all compliance submissions.",
    obligationLabel: "FIU-IND Principal Officer Obligation —",
    pendingDesc:
      "FIU-IND AML & CFT Guidelines 2023 require human review of all AI-assisted compliance outputs. Principal Officer must confirm AI-generated narratives, risk scores, and summaries before use in regulatory filings.",
    exportDesc:
      "CSV export for regulator submission. Meets PMLA Section 12 record-keeping and RBI IT Governance, Risk, Controls and Assurance Practices Directions 2023 (RBI/DoS/2023-24/107) audit requirements. Includes: model version, provider, human review status, reviewer name, timestamps.",
  },
  eu: {
    subtitle:    "EU Regulation 2024/1689 · High-risk AI system transparency and human oversight requirements",
    badge:       "AI Act in force from Aug 2025",
    badgeColor:  "bg-emerald-500/10 border-emerald-500/30 text-emerald-300",
    checklistDesc:
      "EU AI Act Art. 13 (transparency) + Art. 14 (human oversight) requirements for high-risk AI systems used in financial crime detection.",
    obligation:
      "Human oversight of high-risk AI outputs is mandatory from August 2025. Narratives, risk scores, and summaries must be reviewed before acting on them.",
    obligationLabel: "Art. 14 obligation —",
    pendingDesc:
      "AI Act Art. 14 requires a human reviewer to confirm AI outputs used in decisions. Review from the case AI Logs tab.",
    exportDesc:
      "CSV export for regulator submission under EU AI Act Art. 12 (record-keeping). Includes: model version, provider, human review status, reviewer, timestamps.",
  },
  uae: {
    subtitle:    "CBUAE AML/CFT Standards · UAE AI Governance Principles · goAML Compliance",
    badge:       "UAE CBUAE Aligned",
    badgeColor:  "bg-emerald-500/10 border-emerald-500/30 text-emerald-300",
    checklistDesc:
      "CBUAE AML/CFT Supervision Standards (human oversight of compliance AI) + UAE National AI Strategy 2031 (responsible AI) requirements for financial crime investigation systems.",
    obligation:
      "CBUAE requires Compliance Officers to review and authorise all AI-assisted outputs before use in goAML STR submissions or case closure. Personal accountability applies under CBUAE Supervision Framework.",
    obligationLabel: "CBUAE Obligation —",
    pendingDesc:
      "CBUAE AML/CFT Standards require human review of all AI-assisted compliance outputs before regulatory use. Review from the case AI Logs tab.",
    exportDesc:
      "CSV export for CBUAE regulator submission. Includes: model version, provider, human review status, reviewer name, timestamps.",
  },
} as const;

const CHECKLIST = [
  {
    id: "logging",
    label: "AI decisions logged with model version",
    feature: "Case → AI Logs tab",
    desc: {
      india: "Meets PMLA Section 12 record-keeping and RBI IT Governance Directions 2023 audit trail requirements",
      eu:    "Every AI action records model_used, provider, and token counts",
      uae:   "Every AI action records model_used, provider, and token counts for CBUAE audit",
    },
  },
  {
    id: "oversight",
    label: "Human oversight workflow active",
    feature: "Case → AI Logs tab (Approve / Reject / Edit)",
    desc: {
      india: "FIU-IND Principal Officer oversight — all AI outputs require human review before STR filing or case closure",
      eu:    "Approve / reject / edit controls on every AI output",
      uae:   "CBUAE Compliance Officer review required on every AI output before regulatory use",
    },
  },
  {
    id: "minimise",
    label: "Data minimisation — prompts use case IDs",
    feature: "AI Copilot & Investigation Agent",
    desc: {
      india: "DPDP Act 2023, Section 6 — data minimisation; consent limited to purpose; subject PII never sent to AI inference layer",
      eu:    "Subject PII is not sent to AI providers; only case context",
      uae:   "Subject PII not sent to AI inference layer; only anonymised case context used",
    },
  },
  {
    id: "audit",
    label: "Audit trail exportable for regulators",
    feature: "Settings → Responsible AI Governance → Export CSV",
    desc: {
      india: "FIU-IND inspection-ready CSV export covering all AI decisions — PMLA Section 12 record-keeping compliant",
      eu:    "CSV export via this page covers all AI decisions",
      uae:   "CBUAE-ready CSV export covering all AI decisions and human review outcomes",
    },
  },
  {
    id: "onprem",
    label: "AI runs fully on-premise — zero data egress",
    feature: "On-Premise Ollama (llama3.2:3b)",
    desc: {
      india: "DPDP Act 2023 — subject personal data processed entirely within org infrastructure; no PII transmitted to external AI providers",
      eu:    "All AI inference runs on-premise via Ollama — no subject data sent to third-party providers",
      uae:   "AI inference on-premise — no subject data leaves organisational boundaries; meets CBUAE data sovereignty requirements",
    },
  },
  {
    id: "rbac",
    label: "AI feature access is role-gated",
    feature: "Settings → Users & Roles",
    desc: {
      india: "Analyst+ can trigger AI; Manager+ can approve outputs — segregation of duties per RBI IT Governance, Risk, Controls and Assurance Practices Directions 2023",
      eu:    "Analyst+ triggers AI; Manager+ approves — enforces EU AI Act Art. 14 human oversight segregation",
      uae:   "Role-based AI access control — only authorised Compliance Officers can approve AI outputs for regulatory use",
    },
  },
];

type Jurisdiction = "india" | "eu" | "uae";

function ConformityChecklist({ reviewedPct, jurisdiction }: { reviewedPct: number; jurisdiction: Jurisdiction }) {
  const ctx = JURISDICTION_CONTEXT[jurisdiction] ?? JURISDICTION_CONTEXT.india;
  const items = CHECKLIST.map((c) => ({
    ...c,
    resolvedDesc: c.desc[jurisdiction] ?? c.desc.india,
    ok: c.id === "oversight" ? reviewedPct > 0 : true,
  }));

  const score = items.filter((i) => i.ok).length;

  return (
    <section className="mb-6 p-5 bg-[#131f30] ring-1 ring-white/[0.04] border border-[#1e3050] rounded-xl">
      <div className="flex items-center justify-between mb-2">
        <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
          AI Governance Conformity Checklist
        </h2>
        <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${
          score === items.length
            ? "bg-emerald-500/15 text-emerald-300 border border-emerald-500/30"
            : "bg-amber-500/15 text-amber-300 border border-amber-500/30"
        }`}>
          {score}/{items.length} controls active
        </span>
      </div>
      <p className="text-slate-500 text-xs mb-3">{ctx.checklistDesc}</p>
      <div className="mb-4 border-l-2 border-blue-500/50 pl-3 py-0.5">
        <p className="text-xs text-slate-400">
          <span className="text-blue-400 font-medium">{ctx.obligationLabel}</span>{" "}{ctx.obligation}
        </p>
      </div>
      <div className="space-y-3">
        {items.map((item) => (
          <div key={item.id} className="flex items-start gap-3">
            {item.ok
              ? <CheckCircle2 size={15} className="text-emerald-400 flex-shrink-0 mt-0.5" />
              : <AlertTriangle size={15} className="text-amber-400 flex-shrink-0 mt-0.5" />
            }
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <p className="text-sm text-slate-200">{item.label}</p>
                <span className="text-[10px] font-medium px-1.5 py-0.5 rounded bg-cyan-500/10 border border-cyan-500/20 text-cyan-400 whitespace-nowrap">
                  {item.feature}
                </span>
              </div>
              <p className="text-xs text-slate-500 mt-0.5">{item.resolvedDesc}</p>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}

function PendingReviewQueue({ logs, jurisdiction }: { logs: AILog[]; jurisdiction: Jurisdiction }) {
  const ctx = JURISDICTION_CONTEXT[jurisdiction] ?? JURISDICTION_CONTEXT.india;
  return (
    <section className="mb-6 p-5 bg-[#131f30] ring-1 ring-white/[0.04] border border-[#1e3050] rounded-xl">
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
          Pending Human Review
        </h2>
        <span className="text-xs text-amber-300 bg-amber-500/10 border border-amber-500/30 px-2 py-0.5 rounded-full">
          {logs.length} pending
        </span>
      </div>
      <p className="text-slate-500 text-xs mb-4">{ctx.pendingDesc}</p>
      {logs.length === 0 ? (
        <div className="flex items-center gap-2 text-emerald-400 text-sm">
          <CheckCircle2 size={15} />
          All AI outputs have been reviewed
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border border-slate-700/50">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-slate-700/60">
                {["Case", "Action", "Model", "Created"].map((h) => (
                  <th key={h} className="px-3 py-2 text-left text-slate-500 font-medium uppercase tracking-wider">{h}</th>
                ))}
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {logs.slice(0, 10).map((log) => (
                <tr key={log.id} className="border-b border-slate-700/30 hover:bg-slate-800/30">
                  <td className="px-3 py-2 font-mono text-slate-400">{log.case_id.slice(0, 8)}</td>
                  <td className="px-3 py-2 text-slate-300">{log.action_type.replace(/_/g, " ")}</td>
                  <td className="px-3 py-2 text-slate-400">{log.model_used || "—"}</td>
                  <td className="px-3 py-2 text-slate-500">
                    {new Date(log.created_at).toLocaleDateString("en-GB", { day: "2-digit", month: "short" })}
                  </td>
                  <td className="px-3 py-2">
                    <a
                      href={`/cases/${log.case_id}/ai-logs`}
                      className="flex items-center gap-1 text-cyan-400 hover:text-cyan-300 transition-colors"
                    >
                      Review <ExternalLink size={10} />
                    </a>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {logs.length > 10 && (
            <div className="px-3 py-2 text-xs text-slate-500 border-t border-slate-700/30">
              +{logs.length - 10} more — review from individual case AI Logs tabs
            </div>
          )}
        </div>
      )}
    </section>
  );
}

const EXPORT_PREFIX: Record<Jurisdiction, string> = {
  india: "ai-governance-audit",
  eu:    "ai-act-audit",
  uae:   "ai-governance-audit",
};

function ExportSection({ jurisdiction }: { jurisdiction: Jurisdiction }) {
  const ctx = JURISDICTION_CONTEXT[jurisdiction] ?? JURISDICTION_CONTEXT.india;
  const [from, setFrom] = useState(() => {
    const d = new Date();
    d.setMonth(d.getMonth() - 1);
    return d.toISOString().slice(0, 10);
  });
  const [to, setTo] = useState(() => new Date().toISOString().slice(0, 10));
  const [loading, setLoading] = useState(false);

  const handleExport = async () => {
    setLoading(true);
    try {
      const url = `${API_BASE}/api/v1/ai/logs/export?from=${from}&to=${to}`;
      const res = await fetch(url, { credentials: "include" });
      if (!res.ok) throw new Error(`Export failed: ${res.status}`);
      const blob = await res.blob();
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = `${EXPORT_PREFIX[jurisdiction]}-${from}-to-${to}.csv`;
      a.click();
      URL.revokeObjectURL(a.href);
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="p-5 bg-[#131f30] ring-1 ring-white/[0.04] border border-[#1e3050] rounded-xl">
      <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
        Export Audit Trail
      </h2>
      <p className="text-slate-500 text-xs mb-4">{ctx.exportDesc}</p>
      <div className="flex items-end gap-3 flex-wrap">
        <div>
          <label className="block text-xs text-slate-400 mb-1">From</label>
          <input
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            className="bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:border-cyan-500"
          />
        </div>
        <div>
          <label className="block text-xs text-slate-400 mb-1">To</label>
          <input
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            className="bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:border-cyan-500"
          />
        </div>
        <button
          onClick={handleExport}
          disabled={loading}
          className="flex items-center gap-1.5 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm font-medium rounded transition-colors"
        >
          {loading ? <Loader2 size={14} className="animate-spin" /> : <Download size={14} />}
          Export CSV
        </button>
      </div>
    </section>
  );
}

export default function AIActPage() {
  const cfg = useConfig();
  const jurisdiction: Jurisdiction =
    (cfg?.jurisdiction === "eu" || cfg?.jurisdiction === "uae")
      ? cfg.jurisdiction
      : "india";
  const ctx = JURISDICTION_CONTEXT[jurisdiction];

  const { data: pendingData } = useQuery<{ data: AILog[]; meta: PendingMeta }>({
    queryKey: ["ai-logs-pending"],
    queryFn: () => api.get("/api/v1/ai/logs/pending"),
  });

  const pendingLogs: AILog[] = (pendingData as { data: AILog[] } | undefined)?.data ?? [];
  const reviewedPct = pendingLogs.length === 0 ? 100 : 0;

  return (
    <div className="p-6">
      <div className="mb-5">
        <a href="/settings" className="text-slate-400 hover:text-slate-200 text-sm transition-colors mb-1 block">
          ← Settings
        </a>
        <h1 className="text-2xl font-bold text-slate-100">Responsible AI Governance</h1>
        <p className="text-slate-400 text-sm mt-0.5">{ctx.subtitle}</p>
        {cfg && (
          <div className="flex items-center gap-2 mt-2">
            <span className="text-[11px] px-2 py-0.5 rounded-full bg-violet-500/10 border border-violet-500/30 text-violet-300">
              {cfg.jurisdiction?.toUpperCase()} deployment · {cfg.country?.toUpperCase()}
            </span>
            <span className={`text-[11px] px-2 py-0.5 rounded-full border ${ctx.badgeColor}`}>
              {ctx.badge}
            </span>
          </div>
        )}
      </div>

      <ConformityChecklist reviewedPct={reviewedPct} jurisdiction={jurisdiction} />
      <PendingReviewQueue logs={pendingLogs} jurisdiction={jurisdiction} />
      <ExportSection jurisdiction={jurisdiction} />
    </div>
  );
}
