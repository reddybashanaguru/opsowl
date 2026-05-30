"use client";

import { use, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { CaseSubNav } from "@/components/cases/CaseSubNav";

interface InsightsData {
  id?: string;
  case_id: string;
  risk_score: number | null;
  risk_level: string;
  risk_factors: string[];
  entities: { individuals?: string[]; companies?: string[]; accounts?: string[] } | null;
  anomalies: string[];
  recommended_actions: string[];
  model_used: string | null;
  confidence: number | null;
  analysed_at: string | null;
  ai_available: boolean;
}

const RISK_COLORS: Record<string, string> = {
  high: "text-red-400 border-red-500/40 bg-red-500/10",
  medium: "text-amber-400 border-amber-500/40 bg-amber-500/10",
  low: "text-emerald-400 border-emerald-500/40 bg-emerald-500/10",
  unknown: "text-slate-400 border-slate-600 bg-slate-700/30",
};

function RiskGauge({ score }: { score: number | null }) {
  const pct = score ?? 0;
  const color = pct >= 75 ? "#f87171" : pct >= 40 ? "#fbbf24" : "#34d399";
  const circumference = 2 * Math.PI * 54;
  const dashOffset = circumference - (pct / 100) * circumference;

  return (
    <div className="flex flex-col items-center justify-center py-4">
      <svg width="140" height="140" viewBox="0 0 140 140">
        <circle cx="70" cy="70" r="54" fill="none" stroke="#1e293b" strokeWidth="14" />
        <circle
          cx="70" cy="70" r="54" fill="none"
          stroke={color} strokeWidth="14"
          strokeDasharray={circumference}
          strokeDashoffset={dashOffset}
          strokeLinecap="round"
          transform="rotate(-90 70 70)"
          style={{ transition: "stroke-dashoffset 0.8s ease" }}
        />
        <text x="70" y="66" textAnchor="middle" fill="white" fontSize="28" fontWeight="700">
          {score !== null ? Math.round(score) : "—"}
        </text>
        <text x="70" y="85" textAnchor="middle" fill="#94a3b8" fontSize="11">
          Risk Score
        </text>
      </svg>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5">
      <h3 className="text-slate-300 text-sm font-semibold uppercase tracking-wide mb-4">{title}</h3>
      {children}
    </div>
  );
}

export default function InsightsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const qc = useQueryClient();
  const [analysing, setAnalysing] = useState(false);

  const { data: ins, isLoading } = useQuery<InsightsData>({
    queryKey: ["insights", id],
    queryFn: () => api.get<InsightsData>(`/api/v1/cases/${id}/insights`),
  });

  const analyseMutation = useMutation({
    mutationFn: () => api.post<InsightsData>(`/api/v1/cases/${id}/insights/analyse`, {}),
    onMutate: () => setAnalysing(true),
    onSettled: () => setAnalysing(false),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["insights", id] }),
  });

  const riskColorClass = RISK_COLORS[ins?.risk_level ?? "unknown"] ?? RISK_COLORS.unknown;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-6 h-6 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <CaseSubNav
        caseId={id}
        activeTab="insights"
        action={
          <button
            onClick={() => analyseMutation.mutate()}
            disabled={analysing}
            className="bg-violet-600 hover:bg-violet-500 text-white text-sm font-semibold px-4 py-2 rounded-lg transition-colors disabled:opacity-60 flex items-center gap-2"
          >
            {analysing ? (
              <>
                <div className="w-3.5 h-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                Analysing…
              </>
            ) : "Run AI Analysis"}
          </button>
        }
      />

      {!ins?.ai_available && !ins?.id && (
        <div className="bg-amber-500/10 border border-amber-500/30 text-amber-300 rounded-xl p-4 mb-6 text-sm">
          No analysis available yet. Click <strong>Run AI Analysis</strong> to analyse this case.
          {!ins?.ai_available && " AI service may be unavailable — analysis will use document data when service is online."}
        </div>
      )}

      <div className="grid grid-cols-3 gap-6">
        {/* Left — gauge + level */}
        <div className="space-y-4">
          <Section title="Risk Score">
            <RiskGauge score={ins?.risk_score ?? null} />
            <div className={`border rounded-lg text-center py-2 text-sm font-semibold capitalize ${riskColorClass}`}>
              {ins?.risk_level ?? "Unknown"} Risk
            </div>
            {ins?.confidence !== null && ins?.confidence !== undefined && (
              <p className="text-slate-500 text-xs text-center mt-2">
                Confidence: {Math.round((ins.confidence) * 100)}%
              </p>
            )}
            {ins?.model_used && (
              <p className="text-slate-600 text-xs text-center">Model: {ins.model_used}</p>
            )}
            {ins?.analysed_at && (
              <p className="text-slate-600 text-xs text-center">
                Analysed {new Date(ins.analysed_at).toLocaleString()}
              </p>
            )}
          </Section>
        </div>

        {/* Right — details */}
        <div className="col-span-2 space-y-4">
          <Section title="Risk Factors">
            {!ins?.risk_factors?.length ? (
              <p className="text-slate-600 text-sm italic">No risk factors identified.</p>
            ) : (
              <ul className="space-y-2">
                {ins.risk_factors.map((f, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm text-slate-300">
                    <span className="text-red-400 mt-0.5 flex-shrink-0">▸</span>
                    {f}
                  </li>
                ))}
              </ul>
            )}
          </Section>

          <Section title="Entities Identified">
            {!ins?.entities || Object.keys(ins.entities).length === 0 ? (
              <p className="text-slate-600 text-sm italic">No entities extracted.</p>
            ) : (
              <div className="grid grid-cols-3 gap-4">
                {Object.entries(ins.entities).map(([type, items]) => (
                  <div key={type}>
                    <div className="text-slate-500 text-xs uppercase tracking-wide mb-2">{type}</div>
                    {!items || (items as string[]).length === 0 ? (
                      <p className="text-slate-600 text-xs italic">None</p>
                    ) : (
                      <div className="flex flex-wrap gap-1">
                        {(items as string[]).map((item, i) => (
                          <span key={i} className="bg-slate-700 text-slate-300 text-xs px-2 py-0.5 rounded">
                            {item}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </Section>

          <Section title="Anomalies Detected">
            {!ins?.anomalies?.length ? (
              <p className="text-slate-600 text-sm italic">No anomalies detected.</p>
            ) : (
              <ul className="space-y-2">
                {ins.anomalies.map((a, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm text-slate-300">
                    <span className="text-amber-400 mt-0.5 flex-shrink-0">⚠</span>
                    {a}
                  </li>
                ))}
              </ul>
            )}
          </Section>

          <Section title="Recommended Actions">
            {!ins?.recommended_actions?.length ? (
              <p className="text-slate-600 text-sm italic">No recommendations yet.</p>
            ) : (
              <ul className="space-y-2">
                {ins.recommended_actions.map((a, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm text-slate-300">
                    <span className="text-cyan-400 mt-0.5 flex-shrink-0 font-bold">{i + 1}.</span>
                    {a}
                  </li>
                ))}
              </ul>
            )}
          </Section>
        </div>
      </div>
    </div>
  );
}
