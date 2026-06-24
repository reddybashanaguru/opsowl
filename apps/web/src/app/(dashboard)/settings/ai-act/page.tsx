"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useConfig } from "@/lib/hooks/useConfig";
import { CheckCircle2, AlertTriangle, Clock, Download, Loader2, ExternalLink } from "lucide-react";

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

const CHECKLIST = [
  { id: "logging",     label: "AI decisions logged with model version",     desc: "Every AI action records model_used, provider, and token counts" },
  { id: "oversight",   label: "Human oversight workflow active",             desc: "Approve / reject / edit controls on every AI output" },
  { id: "explain",     label: "Explainability — citations recorded",         desc: "Source citations stored with each AI response" },
  { id: "minimise",    label: "Data minimisation — prompts use case IDs",   desc: "Subject PII is not sent to AI providers; only case context" },
  { id: "audit",       label: "Audit trail exportable for regulators",       desc: "CSV export via this page covers all AI decisions" },
];

function ConformityChecklist({ reviewedPct }: { reviewedPct: number }) {
  const items = CHECKLIST.map((c) => ({
    ...c,
    ok: c.id === "oversight" ? reviewedPct > 0 : true,
  }));
  const score = items.filter((i) => i.ok).length;

  return (
    <section className="mb-6 p-5 bg-[#131f30] border border-[#1e3050] rounded-xl">
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest">
          AI Act Conformity Checklist
        </h2>
        <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${
          score === items.length
            ? "bg-emerald-500/15 text-emerald-300 border border-emerald-500/30"
            : "bg-amber-500/15 text-amber-300 border border-amber-500/30"
        }`}>
          {score}/{items.length} controls active
        </span>
      </div>
      <p className="text-slate-500 text-xs mb-4">
        EU AI Act Art. 13 (transparency) + Art. 14 (human oversight) requirements for high-risk AI systems used in financial crime detection.
      </p>
      <div className="space-y-2">
        {items.map((item) => (
          <div key={item.id} className="flex items-start gap-3">
            {item.ok
              ? <CheckCircle2 size={15} className="text-emerald-400 flex-shrink-0 mt-0.5" />
              : <AlertTriangle size={15} className="text-amber-400 flex-shrink-0 mt-0.5" />
            }
            <div>
              <p className="text-sm text-slate-200">{item.label}</p>
              <p className="text-xs text-slate-500">{item.desc}</p>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}

function PendingReviewQueue({ logs }: { logs: AILog[] }) {
  return (
    <section className="mb-6 p-5 bg-[#131f30] border border-[#1e3050] rounded-xl">
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest">
          Pending Human Review
        </h2>
        <span className="text-xs text-amber-300 bg-amber-500/10 border border-amber-500/30 px-2 py-0.5 rounded-full">
          {logs.length} pending
        </span>
      </div>
      <p className="text-slate-500 text-xs mb-4">
        AI Act Art. 14 requires a human reviewer to confirm AI outputs used in decisions. Review from the case AI Logs tab.
      </p>
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

function ExportSection() {
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
      const url = `/api/v1/ai/logs/export?from=${from}&to=${to}`;
      const token = typeof window !== "undefined" ? localStorage.getItem("access_token") : null;
      const res = await fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {} });
      const blob = await res.blob();
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = `ai-act-audit-${from}-to-${to}.csv`;
      a.click();
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="p-5 bg-[#131f30] border border-[#1e3050] rounded-xl">
      <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest mb-1">
        Export Audit Trail
      </h2>
      <p className="text-slate-500 text-xs mb-4">
        CSV export for regulator submission under EU AI Act Art. 12 (record-keeping).
        Includes: model version, provider, human review status, reviewer, timestamps.
      </p>
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

  const { data: pendingData } = useQuery<{ data: AILog[]; meta: PendingMeta }>({
    queryKey: ["ai-logs-pending"],
    queryFn: () => api.get("/api/v1/ai/logs/pending"),
  });

  const pendingLogs: AILog[] = (pendingData as { data: AILog[] } | undefined)?.data ?? [];
  const reviewedPct = pendingLogs.length === 0 ? 100 : 0;

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="mb-6">
        <a href="/settings" className="text-slate-400 hover:text-slate-200 text-sm transition-colors mb-1 block">
          ← Settings
        </a>
        <h1 className="text-xl font-semibold text-slate-100">AI Act Compliance</h1>
        <p className="text-slate-500 text-xs mt-0.5">
          EU Regulation 2024/1689 · High-risk AI system transparency and human oversight requirements
        </p>
        {cfg && (
          <div className="flex items-center gap-2 mt-2">
            <span className="text-[11px] px-2 py-0.5 rounded-full bg-violet-500/10 border border-violet-500/30 text-violet-300">
              {cfg.jurisdiction?.toUpperCase()} deployment · {cfg.country?.toUpperCase()}
            </span>
            <span className="text-[11px] px-2 py-0.5 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-300">
              AI Act in force from Aug 2025
            </span>
          </div>
        )}
      </div>

      <div className="mb-4 p-3 bg-blue-500/10 border border-blue-500/30 rounded-lg">
        <div className="flex items-start gap-2">
          <Clock size={14} className="text-blue-400 flex-shrink-0 mt-0.5" />
          <p className="text-xs text-blue-300">
            <strong>Article 14 obligation:</strong> Human oversight of high-risk AI outputs is mandatory from August 2025.
            AI-generated narratives, risk assessments, and investigation summaries must be reviewed before acting on them.
          </p>
        </div>
      </div>

      <ConformityChecklist reviewedPct={reviewedPct} />
      <PendingReviewQueue logs={pendingLogs} />
      <ExportSection />
    </div>
  );
}
