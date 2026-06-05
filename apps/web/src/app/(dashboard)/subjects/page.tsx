"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { api } from "@/lib/api";
import { UserSearch, Shield, AlertTriangle, Eye } from "lucide-react";

interface SubjectRow {
  id: string;
  canonical_name: string;
  entity_type: string;
  id_type: string;
  id_number: string;
  nationality: string;
  risk_score: number;
  risk_level: string;
  is_pep: boolean;
  is_sanctioned: boolean;
  is_watchlisted: boolean;
  total_cases: number;
  open_cases: number;
  total_str_filings: number;
  created_at: string;
}

interface SubjectsMeta {
  total: number;
  has_more: boolean;
}

const RISK_COLORS: Record<string, string> = {
  critical: "text-red-400",
  high:     "text-orange-400",
  medium:   "text-yellow-400",
  low:      "text-slate-400",
};

const ENTITY_TYPE_LABELS: Record<string, string> = {
  individual: "Individual",
  company:    "Company",
  account:    "Account",
  unknown:    "Unknown",
};

export default function SubjectsPage() {
  const [search, setSearch] = useState("");
  const [riskFilter, setRiskFilter] = useState("");
  const [entityFilter, setEntityFilter] = useState("");
  const [offset, setOffset] = useState(0);
  const LIMIT = 25;

  const { data, isFetching } = useQuery<{ data: SubjectRow[]; meta: SubjectsMeta }>({
    queryKey: ["subjects", search, riskFilter, entityFilter, offset],
    queryFn: () => {
      const params = new URLSearchParams({ limit: String(LIMIT), offset: String(offset) });
      if (search.trim())  params.set("q", search.trim());
      if (riskFilter)     params.set("risk_level", riskFilter);
      if (entityFilter)   params.set("entity_type", entityFilter);
      return api.getList<SubjectRow, SubjectsMeta>(`/api/v1/subjects?${params}`);
    },
    refetchInterval: 60_000,
  });

  const subjects = data?.data ?? [];
  const total    = data?.meta?.total ?? 0;

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center gap-2.5">
          <UserSearch className="w-5 h-5 text-cyan-400" />
          <h1 className="text-xl font-bold text-slate-100">Subject Directory</h1>
          {total > 0 && <span className="px-2 py-0.5 bg-slate-700 text-slate-400 rounded-full text-xs font-semibold">{total.toLocaleString()}</span>}
        </div>
        <p className="text-slate-400 text-sm mt-1">Deduplicated entity records across all alerts and cases</p>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3 mb-5 flex-wrap">
        <div className="relative flex-1 min-w-48 max-w-sm">
          <input value={search} onChange={e => { setSearch(e.target.value); setOffset(0); }}
            placeholder="Search by name, ID…"
            className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-1.5 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-cyan-500" />
        </div>
        <select value={riskFilter} onChange={e => { setRiskFilter(e.target.value); setOffset(0); }}
          className="bg-slate-800 border border-slate-700 text-slate-300 text-sm rounded-lg px-3 py-1.5 focus:outline-none focus:border-cyan-500">
          <option value="">All risk levels</option>
          {["critical","high","medium","low"].map(r => (
            <option key={r} value={r}>{r.charAt(0).toUpperCase()+r.slice(1)}</option>
          ))}
        </select>
        <select value={entityFilter} onChange={e => { setEntityFilter(e.target.value); setOffset(0); }}
          className="bg-slate-800 border border-slate-700 text-slate-300 text-sm rounded-lg px-3 py-1.5 focus:outline-none focus:border-cyan-500">
          <option value="">All entity types</option>
          {["individual","company","account"].map(t => (
            <option key={t} value={t}>{ENTITY_TYPE_LABELS[t] ?? t}</option>
          ))}
        </select>
      </div>

      {/* Table */}
      <div className="bg-[#1e293b] border border-slate-700 rounded-xl overflow-hidden">
        {isFetching && !data ? (
          <div className="flex items-center justify-center h-48 text-slate-500 text-sm">Loading subjects…</div>
        ) : subjects.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-48 text-slate-500">
            <UserSearch className="w-10 h-10 mb-3 text-slate-600" />
            <p className="text-sm font-medium">No subjects found</p>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-700 bg-slate-800/40">
                <th className="text-left px-4 py-3 text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Name</th>
                <th className="text-left px-4 py-3 text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Identity</th>
                <th className="text-left px-4 py-3 text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Risk</th>
                <th className="text-left px-4 py-3 text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Flags</th>
                <th className="text-left px-4 py-3 text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Cases</th>
                <th className="text-left px-4 py-3 text-[10px] font-semibold text-slate-400 uppercase tracking-wider">STRs</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody>
              {subjects.map((s, i) => (
                <tr key={s.id} className={`border-b border-slate-700/50 hover:bg-slate-700/20 transition-colors ${i === subjects.length - 1 ? "border-b-0" : ""}`}>
                  <td className="px-4 py-3">
                    <Link href={`/subjects/${s.id}`} className="font-medium text-slate-200 hover:text-cyan-400 transition-colors text-sm">
                      {s.canonical_name}
                    </Link>
                    <div className="text-slate-500 text-[11px] mt-0.5">{ENTITY_TYPE_LABELS[s.entity_type] ?? s.entity_type}</div>
                  </td>
                  <td className="px-4 py-3 text-slate-400 text-xs">
                    {s.id_type && s.id_number ? (
                      <span>{s.id_type}: <span className="font-mono text-slate-300">{s.id_number}</span></span>
                    ) : <span className="text-slate-600">—</span>}
                  </td>
                  <td className="px-4 py-3">
                    <div className={`font-semibold text-sm ${RISK_COLORS[s.risk_level] ?? "text-slate-400"}`}>
                      {s.risk_score}
                      <span className="text-xs font-normal ml-1 opacity-70">{s.risk_level}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1.5">
                      {s.is_pep && (
                        <span className="px-1.5 py-0.5 bg-violet-500/15 text-violet-400 rounded text-[10px] font-semibold border border-violet-500/30">PEP</span>
                      )}
                      {s.is_sanctioned && (
                        <span className="px-1.5 py-0.5 bg-red-500/15 text-red-400 rounded text-[10px] font-semibold border border-red-500/30 flex items-center gap-0.5">
                          <AlertTriangle size={9} />SANC
                        </span>
                      )}
                      {s.is_watchlisted && (
                        <span className="px-1.5 py-0.5 bg-amber-500/15 text-amber-400 rounded text-[10px] font-semibold border border-amber-500/30 flex items-center gap-0.5">
                          <Eye size={9} />WATCH
                        </span>
                      )}
                      {!s.is_pep && !s.is_sanctioned && !s.is_watchlisted && (
                        <span className="text-slate-600 text-[11px]">—</span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-slate-300 text-xs">
                    {s.total_cases > 0 ? (
                      <span>{s.open_cases > 0 && <span className="text-cyan-400 font-semibold">{s.open_cases} open</span>}
                        {s.open_cases > 0 && s.total_cases > s.open_cases ? " / " : ""}
                        {s.total_cases > s.open_cases ? <span className="text-slate-400">{s.total_cases} total</span> : null}
                        {s.open_cases === 0 && <span className="text-slate-500">{s.total_cases} total</span>}
                      </span>
                    ) : <span className="text-slate-600">—</span>}
                  </td>
                  <td className="px-4 py-3 text-slate-300 text-xs">
                    {s.total_str_filings > 0 ? s.total_str_filings : <span className="text-slate-600">—</span>}
                  </td>
                  <td className="px-4 py-3">
                    <Link href={`/subjects/${s.id}`} className="p-1.5 rounded-lg text-slate-400 hover:text-cyan-400 hover:bg-slate-700/50 transition-colors inline-flex">
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                      </svg>
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Pagination */}
      {total > LIMIT && (
        <div className="flex items-center justify-between mt-4">
          <span className="text-slate-500 text-xs">Showing {offset + 1}–{Math.min(offset + LIMIT, total)} of {total.toLocaleString()}</span>
          <div className="flex gap-2">
            <button disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - LIMIT))}
              className="px-3 py-1.5 rounded-lg border border-slate-700 text-slate-300 text-xs disabled:opacity-40 hover:border-slate-500 transition-colors">
              Previous
            </button>
            <button disabled={offset + LIMIT >= total} onClick={() => setOffset(offset + LIMIT)}
              className="px-3 py-1.5 rounded-lg border border-slate-700 text-slate-300 text-xs disabled:opacity-40 hover:border-slate-500 transition-colors">
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
