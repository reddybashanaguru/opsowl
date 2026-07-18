"use client";

import { use, useState, useEffect, useMemo, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { CaseSubNav } from "@/components/cases/CaseSubNav";
import { PageError } from "@/components/ui/PageError";

// ── Types ──────────────────────────────────────────────────────────────────────

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

interface Typology {
  id: string;
  code: string;
  name: string;
  category: string;
  description: string;
  red_flags: string[];
  risk_keywords: string[];
  fatf_reference: string;
}

interface CaseTypology extends Typology {
  tagged_by?: string;
  tagged_at: string;
  notes: string;
  is_ai_suggested: boolean;
}

import { useCaseSummary } from "@/lib/hooks/useCaseSummary";

// ── Constants ──────────────────────────────────────────────────────────────────

const RISK_COLORS: Record<string, string> = {
  high: "text-red-400 border-red-500/40 bg-red-500/10",
  medium: "text-amber-400 border-amber-500/40 bg-amber-500/10",
  low: "text-emerald-400 border-emerald-500/40 bg-emerald-500/10",
  unknown: "text-slate-400 border-slate-600 bg-slate-700/30",
};

const CATEGORY_COLORS: Record<string, string> = {
  PLACEMENT:   "bg-red-500/15 text-red-300 border-red-500/30",
  LAYERING:    "bg-amber-500/15 text-amber-300 border-amber-500/30",
  INTEGRATION: "bg-emerald-500/15 text-emerald-300 border-emerald-500/30",
  STRUCTURING: "bg-orange-500/15 text-orange-300 border-orange-500/30",
  TF:          "bg-purple-500/15 text-purple-300 border-purple-500/30",
  PF:          "bg-pink-500/15 text-pink-300 border-pink-500/30",
};

const ALL_CATEGORIES = ["PLACEMENT", "LAYERING", "INTEGRATION", "STRUCTURING", "TF", "PF"];

// ── Helpers ────────────────────────────────────────────────────────────────────

function categoryBadge(cat: string) {
  return (
    <span className={`text-[10px] font-semibold px-1.5 py-0.5 rounded border uppercase tracking-wider ${CATEGORY_COLORS[cat] ?? "bg-slate-700 text-slate-400 border-slate-600"}`}>
      {cat}
    </span>
  );
}

function scoreTypology(typology: Typology, riskText: string): number {
  if (!riskText) return 0;
  const lower = riskText.toLowerCase();
  const matched = typology.risk_keywords.filter(kw => lower.includes(kw.toLowerCase()));
  return matched.length;
}

// ── Sub-components ─────────────────────────────────────────────────────────────

function RiskGauge({ score }: { score: number | null }) {
  const pct = score ?? 0;
  const color = pct >= 75 ? "#f87171" : pct >= 40 ? "#fbbf24" : "#34d399";
  const circumference = 2 * Math.PI * 54;
  const dashOffset = circumference - (pct / 100) * circumference;

  return (
    <div className="flex flex-col items-center justify-center py-4">
      <svg width="140" height="140" viewBox="0 0 140 140">
        <circle cx="70" cy="70" r="54" fill="none" stroke="#131f30" strokeWidth="14" />
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

function Section({ title, children, action, className }: { title: string; children: React.ReactNode; action?: React.ReactNode; className?: string }) {
  return (
    <div className={`bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5 ${className ?? ""}`}>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-slate-300 text-sm font-semibold uppercase tracking-wider">{title}</h3>
        {action}
      </div>
      {children}
    </div>
  );
}

// ── Tag Typology Modal ─────────────────────────────────────────────────────────

function TagTypologyModal({
  caseId,
  taggedIds,
  onClose,
}: {
  caseId: string;
  taggedIds: Set<string>;
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => { inputRef.current?.focus(); }, []);

  const { data: library = [] } = useQuery<Typology[]>({
    queryKey: ["typology-library"],
    queryFn: () => api.get<Typology[]>("/api/v1/typologies"),
    staleTime: 5 * 60 * 1000,
  });

  const [taggingId, setTaggingId] = useState<string | null>(null);
  const [localTagged, setLocalTagged] = useState<Set<string>>(new Set());
  const tagMutation = useMutation({
    mutationFn: (typologyId: string) =>
      api.post(`/api/v1/cases/${caseId}/typologies`, { typology_id: typologyId, notes: "", is_ai_suggested: false }),
    onMutate: (typologyId) => {
      setTaggingId(typologyId);
      setLocalTagged(prev => new Set([...prev, typologyId]));
    },
    onSettled: () => {
      setTaggingId(null);
      qc.invalidateQueries({ queryKey: ["case-typologies", caseId] });
    },
    onError: (_, typologyId) => {
      setLocalTagged(prev => { const next = new Set(prev); next.delete(typologyId); return next; });
    },
  });

  const filtered = useMemo(() => {
    const q = search.toLowerCase();
    return library.filter(t => {
      const matchCat = !category || t.category === category;
      const matchQ = !q || t.name.toLowerCase().includes(q) || t.description.toLowerCase().includes(q) || t.code.toLowerCase().includes(q);
      return matchCat && matchQ;
    });
  }, [library, search, category]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div
        className="bg-[#0d1824] border border-[#1e3050] rounded-2xl w-full max-w-2xl max-h-[80vh] flex flex-col shadow-2xl"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#1e3050]">
          <h2 className="text-white font-semibold text-base">Tag FATF Typology</h2>
          <button onClick={onClose} className="text-slate-500 hover:text-white transition-colors text-xl leading-none">×</button>
        </div>

        {/* Filters */}
        <div className="px-6 py-3 border-b border-[#1e3050] flex gap-3">
          <input
            ref={inputRef}
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search typologies…"
            className="flex-1 bg-[#131f30] border border-[#1e3050] rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-cyan-500/50"
          />
          <select
            value={category}
            onChange={e => setCategory(e.target.value)}
            className="bg-[#131f30] border border-[#1e3050] rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-cyan-500/50"
          >
            <option value="">All categories</option>
            {ALL_CATEGORIES.map(c => <option key={c} value={c}>{c}</option>)}
          </select>
        </div>

        {/* List */}
        <div className="overflow-y-auto flex-1 px-4 py-3 space-y-2">
          {filtered.length === 0 && (
            <p className="text-slate-500 text-sm text-center py-8">No typologies match your search.</p>
          )}
          {filtered.map(t => {
            const isTagged = taggedIds.has(t.id) || localTagged.has(t.id);
            const isPending = taggingId === t.id;
            return (
              <div key={t.id} className="flex items-center gap-3 bg-[#131f30] border border-[#1e3050] rounded-xl px-4 py-3 hover:border-cyan-500/20 transition-colors">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    {categoryBadge(t.category)}
                    <span className="text-white text-sm font-medium truncate">{t.name}</span>
                  </div>
                  <p className="text-slate-500 text-xs line-clamp-1">{t.description}</p>
                </div>
                <button
                  disabled={isTagged || isPending}
                  onClick={() => tagMutation.mutate(t.id)}
                  className={`flex-shrink-0 text-xs font-semibold px-3 py-1.5 rounded-lg transition-colors flex items-center gap-1.5 ${
                    isTagged
                      ? "bg-emerald-500/15 text-emerald-400 cursor-default"
                      : "bg-cyan-600 hover:bg-cyan-500 text-white disabled:opacity-60"
                  }`}
                >
                  {isPending && <div className="w-3 h-3 border border-white/30 border-t-white rounded-full animate-spin" />}
                  {isTagged ? "Tagged ✓" : "Tag"}
                </button>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// ── Matched Typologies Section ─────────────────────────────────────────────────

function MatchedTypologies({
  caseId,
  riskFactors,
}: {
  caseId: string;
  riskFactors: string[];
}) {
  const qc = useQueryClient();
  const [showModal, setShowModal] = useState(false);

  const { data: caseTypologies = [], isLoading } = useQuery<CaseTypology[]>({
    queryKey: ["case-typologies", caseId],
    queryFn: () => api.get<CaseTypology[]>(`/api/v1/cases/${caseId}/typologies`),
  });

  const { data: library = [] } = useQuery<Typology[]>({
    queryKey: ["typology-library"],
    queryFn: () => api.get<Typology[]>("/api/v1/typologies"),
    staleTime: 5 * 60 * 1000,
  });

  const untagMutation = useMutation({
    mutationFn: (typologyId: string) =>
      api.delete(`/api/v1/cases/${caseId}/typologies/${typologyId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["case-typologies", caseId] }),
  });

  const [taggingId, setTaggingId] = useState<string | null>(null);
  const tagMutation = useMutation({
    mutationFn: ({ typologyId, isAI }: { typologyId: string; isAI: boolean }) =>
      api.post(`/api/v1/cases/${caseId}/typologies`, {
        typology_id: typologyId,
        notes: isAI ? "Auto-suggested based on risk factor keyword match" : "",
        is_ai_suggested: isAI,
      }),
    onMutate: ({ typologyId }) => setTaggingId(typologyId),
    onSettled: () => {
      setTaggingId(null);
      qc.invalidateQueries({ queryKey: ["case-typologies", caseId] });
    },
  });

  const taggedIds = useMemo(() => new Set(caseTypologies.map(t => t.id)), [caseTypologies]);

  // Client-side AI suggestions: score typologies against concatenated risk factors
  const suggestions = useMemo(() => {
    if (!riskFactors.length || !library.length) return [];
    const riskText = riskFactors.join(" ");
    return library
      .filter(t => !taggedIds.has(t.id))
      .map(t => ({ ...t, score: scoreTypology(t, riskText) }))
      .filter(t => t.score > 0)
      .sort((a, b) => b.score - a.score)
      .slice(0, 5);
  }, [library, taggedIds, riskFactors]);

  return (
    <>
      <Section
        title="Matched FATF Typologies"
        action={
          <button
            onClick={() => setShowModal(true)}
            className="bg-[#0d1824] hover:bg-[#1e3050] border border-[#1e3050] text-slate-300 hover:text-white text-xs font-semibold px-3 py-1.5 rounded-lg transition-colors flex items-center gap-1.5"
          >
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Tag Typology
          </button>
        }
      >
        {isLoading ? (
          <div className="flex items-center gap-2 text-slate-500 text-sm py-2">
            <div className="w-3.5 h-3.5 border-2 border-slate-600 border-t-slate-400 rounded-full animate-spin" />
            Loading…
          </div>
        ) : caseTypologies.length === 0 ? (
          <p className="text-slate-600 text-sm italic">No typologies tagged to this case yet.</p>
        ) : (
          <div className="space-y-2">
            {caseTypologies.map(ct => (
              <div key={ct.id} className="flex items-center gap-3 bg-[#0d1824] border border-[#1e3050] rounded-xl px-4 py-3">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap mb-1">
                    {categoryBadge(ct.category)}
                    <span className="text-white text-sm font-medium">{ct.name}</span>
                    {ct.is_ai_suggested && (
                      <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded border bg-violet-500/15 text-violet-300 border-violet-500/30 uppercase tracking-wider">
                        AI Suggested
                      </span>
                    )}
                  </div>
                  <p className="text-slate-500 text-xs line-clamp-1">{ct.description}</p>
                  {ct.notes && (
                    <p className="text-slate-600 text-xs mt-1 italic">"{ct.notes}"</p>
                  )}
                </div>
                <button
                  onClick={() => untagMutation.mutate(ct.id)}
                  disabled={untagMutation.isPending}
                  className="flex-shrink-0 text-xs text-slate-500 hover:text-red-400 transition-colors p-1 rounded hover:bg-red-500/10 disabled:opacity-50"
                  title="Remove typology"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            ))}
          </div>
        )}

        {/* AI Suggestions */}
        {suggestions.length > 0 && (
          <div className="mt-4 pt-4 border-t border-[#1e3050]">
            <p className="text-slate-500 text-xs uppercase tracking-wider font-semibold mb-3 flex items-center gap-2">
              <svg className="w-3 h-3 text-violet-400" fill="currentColor" viewBox="0 0 20 20">
                <path d="M11 3a1 1 0 10-2 0v1a1 1 0 102 0V3zM15.657 5.757a1 1 0 00-1.414-1.414l-.707.707a1 1 0 001.414 1.414l.707-.707zM18 10a1 1 0 01-1 1h-1a1 1 0 110-2h1a1 1 0 011 1zM5.05 6.464A1 1 0 106.464 5.05l-.707-.707a1 1 0 00-1.414 1.414l.707.707zM5 10a1 1 0 01-1 1H3a1 1 0 110-2h1a1 1 0 011 1zM8 16v-1h4v1a2 2 0 11-4 0zM12 14c.015-.298.03-.6.03-.9A8 8 0 103.97 13.1c0 .3.015.602.03.9H12z" />
              </svg>
              AI-Suggested Matches
            </p>
            <div className="space-y-2">
              {suggestions.map(s => (
                <div key={s.id} className="flex items-center gap-3 bg-violet-500/5 border border-violet-500/20 rounded-xl px-4 py-3">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap mb-1">
                      {categoryBadge(s.category)}
                      <span className="text-slate-200 text-sm font-medium">{s.name}</span>
                      <span className="text-[10px] text-violet-400 font-medium">{s.score} keyword{s.score !== 1 ? "s" : ""} matched</span>
                    </div>
                    <p className="text-slate-500 text-xs line-clamp-1">{s.description}</p>
                  </div>
                  <button
                    onClick={() => tagMutation.mutate({ typologyId: s.id, isAI: true })}
                    disabled={taggingId === s.id}
                    className="flex-shrink-0 text-xs font-semibold px-3 py-1.5 rounded-lg bg-violet-600 hover:bg-violet-500 text-white transition-colors disabled:opacity-60 flex items-center gap-1.5"
                  >
                    {taggingId === s.id && (
                      <div className="w-3 h-3 border border-white/30 border-t-white rounded-full animate-spin" />
                    )}
                    Tag
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}
      </Section>

      {showModal && (
        <TagTypologyModal
          caseId={caseId}
          taggedIds={taggedIds}
          onClose={() => setShowModal(false)}
        />
      )}
    </>
  );
}

// ── Page ───────────────────────────────────────────────────────────────────────

const MAX_POLL_MS = 3 * 60 * 1000; // 3 minutes — covers cold model start-up

export default function InsightsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const qc = useQueryClient();
  const [polling, setPolling] = useState(false);
  const [pollTimedOut, setPollTimedOut] = useState(false);
  const pollStartRef = useRef<number | null>(null);
  const prevAnalysedAtRef = useRef<string | null>(null);

  const caseData = useCaseSummary(id);

  const { data: ins, isLoading, isError, error, refetch } = useQuery<InsightsData>({
    queryKey: ["insights", id],
    queryFn: () => api.get<InsightsData>(`/api/v1/cases/${id}/insights`),
    refetchInterval: polling ? 3_000 : false,
    retry: (failureCount, err) => {
      if (err instanceof APIError && err.status === 404) return false;
      return failureCount < 2;
    },
  });

  useEffect(() => {
    if (!polling) return;
    // Stop only when a NEW analysis arrives (analysed_at changed from what it was before polling)
    if (ins?.ai_available && ins?.analysed_at && ins.analysed_at !== prevAnalysedAtRef.current) {
      setPolling(false);
      pollStartRef.current = null;
      return;
    }
    // Stop polling if the AI service hasn't responded in MAX_POLL_MS
    if (pollStartRef.current && Date.now() - pollStartRef.current > MAX_POLL_MS) {
      setPolling(false);
      setPollTimedOut(true);
      pollStartRef.current = null;
    }
  }, [polling, ins?.ai_available]);

  const analyseMutation = useMutation({
    mutationFn: () => api.post<InsightsData>(`/api/v1/cases/${id}/insights/analyse`, {}),
    onSuccess: () => {
      prevAnalysedAtRef.current = ins?.analysed_at ?? null;
      pollStartRef.current = Date.now();
      setPollTimedOut(false);
      setPolling(true);
      qc.invalidateQueries({ queryKey: ["insights", id] });
    },
  });

  const analysing = analyseMutation.isPending || polling;
  const riskColorClass = RISK_COLORS[ins?.risk_level ?? "unknown"] ?? RISK_COLORS.unknown;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-6 h-6 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="p-6">
        <CaseSubNav caseId={id} activeTab="insights" caseNumber={caseData?.case_number} />
        <PageError error={error as Error | null} onRetry={() => refetch()} />
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Fixed header — tabs + banners */}
      <div className="flex-shrink-0 px-6 pt-6 pb-3">
        <CaseSubNav
          caseId={id}
          activeTab="insights"
          caseNumber={caseData?.case_number}
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
              ) : ins?.ai_available ? "Re-analyse" : "Run AI Analysis"}
            </button>
          }
        />

        {polling && (
          <div className="bg-violet-500/10 border border-violet-500/30 text-violet-300 rounded-xl p-4 mt-4 text-sm flex items-center gap-3">
            <div className="w-4 h-4 border-2 border-violet-400/40 border-t-violet-400 rounded-full animate-spin flex-shrink-0" />
            AI analysis in progress — results will appear automatically when complete…
          </div>
        )}
        {pollTimedOut && (
          <div className="bg-red-500/10 border border-red-500/30 text-red-300 rounded-xl p-4 mt-4 text-sm flex items-center justify-between">
            <span>Analysis is taking longer than expected. The AI model may be loading — please try again in a moment.</span>
            <button
              onClick={() => { setPollTimedOut(false); analyseMutation.mutate(); }}
              className="ml-4 flex-shrink-0 text-xs font-semibold px-3 py-1.5 rounded-lg bg-red-600/30 hover:bg-red-600/50 text-red-200 transition-colors"
            >
              Retry
            </button>
          </div>
        )}
        {!polling && !pollTimedOut && !ins?.ai_available && !ins?.id && (
          <div className="bg-amber-500/10 border border-amber-500/30 text-amber-300 rounded-xl p-4 mt-4 text-sm">
            No analysis available yet. Click <strong>Run AI Analysis</strong> to analyse this case.
          </div>
        )}
      </div>

      {/* Scrollable content */}
      <div className="flex-1 min-h-0 overflow-y-auto px-6 pb-6 space-y-6">

      {/* Row 1 — Risk Score + Risk Factors equal height */}
      <div className="grid grid-cols-3 gap-6 items-stretch">
        <Section title="Risk Score" className="h-full">
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

        <Section title="Risk Factors" className="col-span-2 h-full">
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
      </div>

      {/* Row 2 — Anomalies + Recommended Actions side by side */}
      <div className="grid grid-cols-2 gap-6 items-stretch">
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

      {/* Row 3 — FATF Typologies full width */}
      <div>
        <MatchedTypologies caseId={id} riskFactors={ins?.risk_factors ?? []} />
      </div>

      {/* Row 4 — Entities full width */}
      <Section title="Entities Identified">
        {!ins?.entities || Object.keys(ins.entities).length === 0 ? (
          <p className="text-slate-600 text-sm italic">No entities extracted.</p>
        ) : (
          <div className="grid grid-cols-3 gap-4">
            {Object.entries(ins.entities).map(([type, items]) => (
              <div key={type}>
                <div className="text-slate-500 text-xs uppercase tracking-wider mb-2">{type}</div>
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

      </div>{/* end scrollable content */}
    </div>
  );
}
