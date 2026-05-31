"use client";

import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { api } from "@/lib/api";
import {
  UserSearch, AlertTriangle, Eye, Shield, FileText, FolderKanban,
  TrendingUp, TrendingDown, Minus, Clock
} from "lucide-react";

interface SubjectDetail {
  id: string;
  canonical_name: string;
  entity_type: string;
  id_type: string;
  id_number: string;
  nationality: string;
  date_of_birth: string | null;
  gender: string;
  phones: string[];
  emails: string[];
  risk_score: number;
  risk_level: string;
  risk_factors: string[];
  is_pep: boolean;
  is_sanctioned: boolean;
  is_watchlisted: boolean;
  kyc_tier: string;
  notes: string;
  total_cases: number;
  open_cases: number;
  total_str_filings: number;
  first_seen_at: string;
  last_seen_at: string;
  created_at: string;
  updated_at: string;
}

interface SubjectCase {
  case_id: string;
  case_number: string;
  title: string;
  status: string;
  priority: string;
  linked_at: string;
}

interface SubjectFiling {
  id: string;
  filing_type: string;
  status: string;
  created_at: string;
  case_number: string;
}

interface RiskEntry {
  score: number;
  level: string;
  reason: string;
  changed_by: string;
  recorded_at: string;
}

const RISK_COLORS: Record<string, string> = {
  critical: "text-red-400",
  high:     "text-orange-400",
  medium:   "text-yellow-400",
  low:      "text-green-400",
};

const RISK_BG: Record<string, string> = {
  critical: "bg-red-500/10 border-red-500/30",
  high:     "bg-orange-500/10 border-orange-500/30",
  medium:   "bg-yellow-500/10 border-yellow-500/30",
  low:      "bg-green-500/10 border-green-500/30",
};

function RiskTrend({ entries }: { entries: RiskEntry[] }) {
  if (entries.length < 2) return null;
  const latest = entries[entries.length - 1].score;
  const prev   = entries[entries.length - 2].score;
  const diff   = latest - prev;
  if (diff > 0) return <span className="flex items-center gap-0.5 text-red-400 text-xs"><TrendingUp size={12} />+{diff}</span>;
  if (diff < 0) return <span className="flex items-center gap-0.5 text-green-400 text-xs"><TrendingDown size={12} />{diff}</span>;
  return <span className="flex items-center gap-0.5 text-slate-500 text-xs"><Minus size={12} />stable</span>;
}

export default function SubjectProfilePage() {
  const { id } = useParams<{ id: string }>();

  const { data: subject, isLoading } = useQuery<SubjectDetail>({
    queryKey: ["subject", id],
    queryFn: () => api.get<SubjectDetail>(`/api/v1/subjects/${id}`),
    enabled: !!id,
  });

  const { data: casesData } = useQuery<{ cases: SubjectCase[] }>({
    queryKey: ["subject-cases", id],
    queryFn: () => api.get<{ cases: SubjectCase[] }>(`/api/v1/subjects/${id}/cases`),
    enabled: !!id,
  });

  const { data: filingsData } = useQuery<{ filings: SubjectFiling[] }>({
    queryKey: ["subject-filings", id],
    queryFn: () => api.get<{ filings: SubjectFiling[] }>(`/api/v1/subjects/${id}/filings`),
    enabled: !!id,
  });

  const { data: riskHistoryData } = useQuery<{ history: RiskEntry[] }>({
    queryKey: ["subject-risk-history", id],
    queryFn: () => api.get<{ history: RiskEntry[] }>(`/api/v1/subjects/${id}/risk-history`),
    enabled: !!id,
  });

  if (isLoading) {
    return (
      <div className="p-6 flex items-center justify-center h-64 text-slate-500 text-sm">
        Loading subject profile…
      </div>
    );
  }

  if (!subject) {
    return (
      <div className="p-6 flex items-center justify-center h-64 text-slate-500 text-sm">
        Subject not found.
      </div>
    );
  }

  const cases   = casesData?.cases   ?? [];
  const filings = filingsData?.filings ?? [];
  const history = riskHistoryData?.history ?? [];

  return (
    <div className="p-6 max-w-6xl mx-auto">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-xs text-slate-500 mb-5">
        <Link href="/subjects" className="hover:text-slate-300 transition-colors">Subjects</Link>
        <span>/</span>
        <span className="text-slate-400">{subject.canonical_name}</span>
      </div>

      {/* Hero */}
      <div className={`rounded-xl border p-5 mb-6 ${RISK_BG[subject.risk_level]}`}>
        <div className="flex items-start justify-between flex-wrap gap-4">
          <div>
            <div className="flex items-center gap-3 flex-wrap">
              <h1 className="text-2xl font-bold text-slate-100">{subject.canonical_name}</h1>
              {subject.is_pep && (
                <span className="px-2 py-0.5 bg-violet-500/15 text-violet-400 rounded-full text-xs font-semibold border border-violet-500/30">PEP</span>
              )}
              {subject.is_sanctioned && (
                <span className="flex items-center gap-1 px-2 py-0.5 bg-red-500/15 text-red-400 rounded-full text-xs font-semibold border border-red-500/30">
                  <AlertTriangle size={10} />SANCTIONED
                </span>
              )}
              {subject.is_watchlisted && (
                <span className="flex items-center gap-1 px-2 py-0.5 bg-amber-500/15 text-amber-400 rounded-full text-xs font-semibold border border-amber-500/30">
                  <Eye size={10} />WATCHLISTED
                </span>
              )}
            </div>
            <div className="text-slate-400 text-sm mt-1 capitalize">{subject.entity_type} · {subject.nationality || "Unknown nationality"}</div>
            {subject.id_type && subject.id_number && (
              <div className="text-slate-500 text-sm mt-1">{subject.id_type}: <span className="font-mono text-slate-300">{subject.id_number}</span></div>
            )}
          </div>
          {/* Risk Score */}
          <div className="text-right">
            <div className={`text-4xl font-black ${RISK_COLORS[subject.risk_level]}`}>{subject.risk_score}</div>
            <div className="text-slate-400 text-xs uppercase tracking-wider mt-0.5">Risk Score</div>
            <RiskTrend entries={history} />
          </div>
        </div>

        {/* Risk Factors */}
        {subject.risk_factors.length > 0 && (
          <div className="flex flex-wrap gap-2 mt-4">
            {subject.risk_factors.map((f, i) => (
              <span key={i} className="px-2 py-0.5 bg-slate-800/60 text-slate-300 rounded text-xs border border-slate-700">{f}</span>
            ))}
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        {/* Left: Stats + Details */}
        <div className="space-y-5">
          {/* Stats */}
          <div className="grid grid-cols-3 gap-3">
            {[
              { label: "Open Cases", value: subject.open_cases, color: "text-cyan-400" },
              { label: "Total Cases", value: subject.total_cases, color: "text-slate-300" },
              { label: "STR Filings", value: subject.total_str_filings, color: "text-violet-400" },
            ].map(stat => (
              <div key={stat.label} className="bg-[#1e293b] border border-slate-700 rounded-xl p-3 text-center">
                <div className={`text-xl font-bold ${stat.color}`}>{stat.value}</div>
                <div className="text-slate-500 text-[10px] uppercase tracking-wider mt-0.5">{stat.label}</div>
              </div>
            ))}
          </div>

          {/* Details */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4">
            <h3 className="text-slate-200 text-sm font-semibold mb-3">Details</h3>
            <dl className="space-y-2">
              {[
                { label: "KYC Tier",    value: subject.kyc_tier || "—" },
                { label: "Gender",      value: subject.gender || "—" },
                { label: "Date of Birth", value: subject.date_of_birth || "—" },
                { label: "First Seen",  value: new Date(subject.first_seen_at).toLocaleDateString() },
                { label: "Last Seen",   value: new Date(subject.last_seen_at).toLocaleDateString() },
              ].map(({ label, value }) => (
                <div key={label} className="flex justify-between text-xs">
                  <dt className="text-slate-500">{label}</dt>
                  <dd className="text-slate-300 font-mono">{value}</dd>
                </div>
              ))}
            </dl>
          </div>

          {/* Contact */}
          {(subject.phones.length > 0 || subject.emails.length > 0) && (
            <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4">
              <h3 className="text-slate-200 text-sm font-semibold mb-3">Contact</h3>
              {subject.phones.map((p, i) => (
                <div key={i} className="text-slate-400 text-xs font-mono mb-1">{p}</div>
              ))}
              {subject.emails.map((e, i) => (
                <div key={i} className="text-slate-400 text-xs font-mono mb-1">{e}</div>
              ))}
            </div>
          )}
        </div>

        {/* Right: Cases + Filings + Risk History */}
        <div className="lg:col-span-2 space-y-5">
          {/* Cases */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl overflow-hidden">
            <div className="px-4 py-3 border-b border-slate-700 flex items-center gap-2">
              <FolderKanban size={14} className="text-slate-400" />
              <span className="text-sm font-semibold text-slate-200">Cases ({cases.length})</span>
            </div>
            {cases.length === 0 ? (
              <div className="flex items-center justify-center h-20 text-slate-600 text-xs">No cases linked</div>
            ) : (
              <div className="divide-y divide-slate-700/50">
                {cases.slice(0, 10).map(c => (
                  <div key={c.case_id} className="flex items-center justify-between px-4 py-2.5 hover:bg-slate-700/20 transition-colors">
                    <div>
                      <Link href={`/cases/${c.case_id}`} className="text-sm text-slate-200 hover:text-cyan-400 transition-colors font-medium">
                        {c.case_number}: {c.title}
                      </Link>
                      <div className="text-xs text-slate-500 mt-0.5">{new Date(c.linked_at).toLocaleDateString()}</div>
                    </div>
                    <div className="flex items-center gap-2 ml-4">
                      <span className="text-xs capitalize text-slate-400 px-2 py-0.5 bg-slate-800 rounded">{c.priority}</span>
                      <span className="text-xs capitalize text-slate-400">{c.status.replace("_", " ")}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* STR Filings */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl overflow-hidden">
            <div className="px-4 py-3 border-b border-slate-700 flex items-center gap-2">
              <FileText size={14} className="text-slate-400" />
              <span className="text-sm font-semibold text-slate-200">STR / SAR Filings ({filings.length})</span>
            </div>
            {filings.length === 0 ? (
              <div className="flex items-center justify-center h-20 text-slate-600 text-xs">No filings on record</div>
            ) : (
              <div className="divide-y divide-slate-700/50">
                {filings.slice(0, 8).map((f: SubjectFiling) => (
                  <div key={f.id} className="flex items-center justify-between px-4 py-2.5">
                    <div>
                      <span className="text-sm text-slate-300">{f.filing_type}</span>
                      {f.case_number && <span className="text-xs text-slate-500 ml-2">→ {f.case_number}</span>}
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-xs capitalize text-slate-400">{f.status.replace("_", " ")}</span>
                      <span className="text-xs text-slate-500">{new Date(f.created_at).toLocaleDateString()}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Risk Score History */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl overflow-hidden">
            <div className="px-4 py-3 border-b border-slate-700 flex items-center gap-2">
              <TrendingUp size={14} className="text-slate-400" />
              <span className="text-sm font-semibold text-slate-200">Risk Score History</span>
            </div>
            {history.length === 0 ? (
              <div className="flex items-center justify-center h-20 text-slate-600 text-xs">No risk history recorded</div>
            ) : (
              <div className="divide-y divide-slate-700/50">
                {history.slice(-10).reverse().map((entry, i) => (
                  <div key={i} className="flex items-center justify-between px-4 py-2.5">
                    <div className="flex items-center gap-3">
                      <span className={`text-sm font-bold ${RISK_COLORS[entry.level]}`}>{entry.score}</span>
                      <span className="text-xs text-slate-400 capitalize">{entry.level}</span>
                      {entry.reason && <span className="text-xs text-slate-500">· {entry.reason}</span>}
                    </div>
                    <div className="flex items-center gap-2 text-xs text-slate-500">
                      <Clock size={11} />
                      {new Date(entry.recorded_at).toLocaleDateString()}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Notes */}
      {subject.notes && (
        <div className="mt-5 bg-[#1e293b] border border-slate-700 rounded-xl p-4">
          <h3 className="text-slate-200 text-sm font-semibold mb-2">Notes</h3>
          <p className="text-slate-400 text-sm whitespace-pre-wrap">{subject.notes}</p>
        </div>
      )}
    </div>
  );
}
