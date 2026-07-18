"use client";

import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

interface Typology {
  id: string;
  code: string;
  name: string;
  category: string;
  description: string;
  red_flags: string[];
  risk_keywords: string[];
  fatf_reference: string;
  created_at: string;
}

const CATEGORY_META: Record<string, { label: string; color: string; bg: string; activeBg: string }> = {
  PLACEMENT:   { label: "Placement",              color: "text-red-300",     bg: "bg-red-500/15 border-red-500/30",         activeBg: "bg-red-500/25 border-red-400/50 text-red-200" },
  LAYERING:    { label: "Layering",               color: "text-amber-300",   bg: "bg-amber-500/15 border-amber-500/30",     activeBg: "bg-amber-500/25 border-amber-400/50 text-amber-200" },
  INTEGRATION: { label: "Integration",            color: "text-emerald-300", bg: "bg-emerald-500/15 border-emerald-500/30", activeBg: "bg-emerald-500/25 border-emerald-400/50 text-emerald-200" },
  STRUCTURING: { label: "Structuring",            color: "text-orange-300",  bg: "bg-orange-500/15 border-orange-500/30",   activeBg: "bg-orange-500/25 border-orange-400/50 text-orange-200" },
  TF:          { label: "Terrorist Financing",    color: "text-purple-300",  bg: "bg-purple-500/15 border-purple-500/30",   activeBg: "bg-purple-500/25 border-purple-400/50 text-purple-200" },
  PF:          { label: "Proliferation Financing",color: "text-pink-300",    bg: "bg-pink-500/15 border-pink-500/30",       activeBg: "bg-pink-500/25 border-pink-400/50 text-pink-200" },
};

const ALL_CATEGORIES = Object.keys(CATEGORY_META);

function CategoryBadge({ cat }: { cat: string }) {
  const meta = CATEGORY_META[cat] ?? { label: cat, color: "text-slate-400", bg: "bg-slate-700/60 border-slate-600", activeBg: "" };
  return (
    <span className={`text-[10px] font-bold px-2 py-0.5 rounded border uppercase tracking-wider ${meta.bg} ${meta.color}`}>
      {meta.label}
    </span>
  );
}

function TypologyCard({ t }: { t: Typology }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className={`bg-[#0f1c2e] border rounded-xl overflow-hidden transition-colors ${
      expanded ? "border-[#2a4060]" : "border-[#1e3050] hover:border-[#2a4060]"
    }`}>
      <button
        className="w-full text-left px-6 py-4 flex items-start gap-4"
        onClick={() => setExpanded(v => !v)}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2.5 flex-wrap mb-1.5">
            <CategoryBadge cat={t.category} />
            <span className="text-white font-semibold text-sm">{t.name}</span>
            <span className="text-slate-600 text-[11px] font-mono">{t.code}</span>
          </div>
          <p className={`text-slate-400 text-xs leading-relaxed ${expanded ? "" : "line-clamp-2"}`}>{t.description}</p>
        </div>
        <svg
          className={`w-4 h-4 text-slate-500 flex-shrink-0 mt-0.5 transition-transform duration-200 ${expanded ? "rotate-180" : ""}`}
          fill="none" stroke="currentColor" viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {expanded && (
        <div className="px-6 pb-5 pt-3 border-t border-[#1e3050] grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <h4 className="text-slate-500 text-[10px] font-bold uppercase tracking-widest mb-2.5 flex items-center gap-1.5">
              <span className="text-red-400">▸</span> Red Flags
            </h4>
            <ul className="space-y-1.5">
              {t.red_flags.map((f, i) => (
                <li key={i} className="flex items-start gap-2 text-xs text-slate-300">
                  <span className="text-red-400/60 mt-0.5 flex-shrink-0">•</span>
                  {f}
                </li>
              ))}
            </ul>
          </div>

          <div className="space-y-4">
            <div>
              <h4 className="text-slate-500 text-[10px] font-bold uppercase tracking-widest mb-2">Risk Keywords</h4>
              <div className="flex flex-wrap gap-1.5">
                {t.risk_keywords.map((kw, i) => (
                  <span key={i} className="bg-[#0b1525] border border-[#1e3050] text-slate-400 text-[11px] px-2 py-0.5 rounded font-mono">
                    {kw}
                  </span>
                ))}
              </div>
            </div>
            <div>
              <h4 className="text-slate-500 text-[10px] font-bold uppercase tracking-widest mb-1.5">FATF Reference</h4>
              <p className="text-slate-400 text-xs italic">{t.fatf_reference}</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function TypologiesPage() {
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("");

  const { data: typologies = [], isLoading } = useQuery<Typology[]>({
    queryKey: ["typology-library"],
    queryFn: () => api.get<Typology[]>("/api/v1/typologies"),
    staleTime: 5 * 60 * 1000,
  });

  const filtered = useMemo(() => {
    const q = search.toLowerCase();
    return typologies.filter(t => {
      const matchCat = !category || t.category === category;
      const matchQ = !q || t.name.toLowerCase().includes(q) || t.description.toLowerCase().includes(q) || t.code.toLowerCase().includes(q);
      return matchCat && matchQ;
    });
  }, [typologies, search, category]);

  const counts = useMemo(() => {
    const m: Record<string, number> = {};
    typologies.forEach(t => { m[t.category] = (m[t.category] ?? 0) + 1; });
    return m;
  }, [typologies]);

  return (
    <>
      {/* Sticky header — sticks within the main scroll container */}
      <div className="sticky top-0 z-10 bg-[#080e1a] border-b border-[#1e3050]/70 px-8 pt-7 pb-4">
        <div className="mb-4">
          <h1 className="text-2xl font-bold text-white mb-1">Risk Typology Library</h1>
          <p className="text-slate-400 text-sm">
            {typologies.length} curated money laundering and fraud typologies aligned with FATF guidance.
            Used by the AI engine and investigators to classify case risk patterns.
          </p>
        </div>

        {/* Search + dropdown */}
        <div className="flex gap-3 mb-3">
          <div className="relative flex-1 max-w-lg">
            <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search by name, description or code…"
              className="w-full bg-[#0f1c2e] border border-[#1e3050] rounded-lg pl-10 pr-4 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-cyan-500/50 transition-colors"
            />
          </div>
          <select
            value={category}
            onChange={e => setCategory(e.target.value)}
            className="bg-[#0f1c2e] border border-[#1e3050] rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-cyan-500/50 min-w-[200px]"
          >
            <option value="">All categories ({typologies.length})</option>
            {ALL_CATEGORIES.map(c => (
              <option key={c} value={c}>
                {CATEGORY_META[c]?.label ?? c} ({counts[c] ?? 0})
              </option>
            ))}
          </select>
        </div>

        {/* Category pills */}
        <div className="flex flex-wrap gap-2">
          <button
            onClick={() => setCategory("")}
            className={`text-xs font-semibold px-3 py-1 rounded-full border transition-colors ${
              !category ? "bg-cyan-600 text-white border-cyan-600" : "text-slate-400 border-[#1e3050] hover:border-slate-500 hover:text-slate-300"
            }`}
          >
            All
          </button>
          {ALL_CATEGORIES.filter(c => (counts[c] ?? 0) > 0).map(c => {
            const meta = CATEGORY_META[c];
            const active = category === c;
            return (
              <button
                key={c}
                onClick={() => setCategory(active ? "" : c)}
                className={`text-xs font-semibold px-3 py-1 rounded-full border transition-colors ${
                  active ? meta.activeBg : `${meta.color} border-current opacity-50 hover:opacity-100`
                }`}
              >
                {meta.label} <span className="opacity-70">({counts[c] ?? 0})</span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Scrollable list */}
      <div className="px-8 py-6">
        {isLoading ? (
          <div className="space-y-3">
            {[...Array(8)].map((_, i) => (
              <div key={i} className="bg-[#0f1c2e] border border-[#1e3050] rounded-xl h-[72px] animate-pulse" />
            ))}
          </div>
        ) : filtered.length === 0 ? (
          <div className="bg-[#0f1c2e] border border-[#1e3050] rounded-xl p-12 text-center">
            <p className="text-slate-500 text-sm">No typologies match your search.</p>
            <button onClick={() => { setSearch(""); setCategory(""); }} className="mt-3 text-cyan-400 text-xs hover:underline">
              Clear filters
            </button>
          </div>
        ) : (
          <>
            <p className="text-slate-500 text-xs mb-4">
              Showing {filtered.length} of {typologies.length} typologies
              {category ? ` · ${CATEGORY_META[category]?.label ?? category}` : ""}
            </p>
            <div className="space-y-2.5">
              {filtered.map(t => <TypologyCard key={t.id} t={t} />)}
            </div>
          </>
        )}
      </div>
    </>
  );
}
