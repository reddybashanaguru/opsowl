"use client";

import { useParams } from "next/navigation";
import { useState, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { api } from "@/lib/api";
import {
  UserSearch, AlertTriangle, Eye, Shield, FileText, FolderKanban,
  TrendingUp, TrendingDown, Minus, Clock, GitFork, Plus, Trash2, X
} from "lucide-react";

// ── Types ──────────────────────────────────────────────────────────────────

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

interface Relationship {
  id: string;
  direction: "outbound" | "inbound";
  peer_subject_id: string;
  peer_name: string;
  peer_entity_type: string;
  peer_risk_level: string;
  peer_risk_score: number;
  peer_is_pep: boolean;
  peer_is_sanctioned: boolean;
  relationship_type: string;
  strength: number;
  source: string;
  notes?: string;
  created_at: string;
}

interface NetworkNode {
  id: string;
  label: string;
  entity_type: string;
  risk_level: string;
  risk_score: number;
  is_pep: boolean;
  is_sanctioned: boolean;
  is_watchlisted: boolean;
  depth: number;
  is_root: boolean;
}

interface NetworkEdge {
  id: string;
  source: string;
  target: string;
  relationship_type: string;
  strength: number;
}

interface NetworkData {
  root_id: string;
  depth: number;
  nodes: NetworkNode[];
  edges: NetworkEdge[];
  high_risk_count: number;
}

// ── Constants ──────────────────────────────────────────────────────────────

const RELATIONSHIP_TYPES = [
  "OWNS_ACCOUNT", "TRANSACTS_WITH", "FAMILY_OF", "EMPLOYED_BY",
  "DIRECTOR_OF", "BENEFICIAL_OWNER_OF", "SAME_ADDRESS", "SAME_PHONE",
  "SAME_IP", "SAME_DEVICE", "GUARANTOR_OF", "INTRODUCED_BY",
];

const REL_LABELS: Record<string, string> = {
  OWNS_ACCOUNT: "Owns Account", TRANSACTS_WITH: "Transacts With",
  FAMILY_OF: "Family Of", EMPLOYED_BY: "Employed By",
  DIRECTOR_OF: "Director Of", BENEFICIAL_OWNER_OF: "Beneficial Owner Of",
  SAME_ADDRESS: "Same Address", SAME_PHONE: "Same Phone",
  SAME_IP: "Same IP", SAME_DEVICE: "Same Device",
  GUARANTOR_OF: "Guarantor Of", INTRODUCED_BY: "Introduced By",
};

const RISK_COLORS: Record<string, string> = {
  critical: "text-red-400", high: "text-orange-400",
  medium: "text-yellow-400", low: "text-green-400",
};
const RISK_BG: Record<string, string> = {
  critical: "bg-red-500/10 border-red-500/30", high: "bg-orange-500/10 border-orange-500/30",
  medium: "bg-yellow-500/10 border-yellow-500/30", low: "bg-green-500/10 border-green-500/30",
};

// Node fill colour in SVG by risk level
const NODE_FILL: Record<string, string> = {
  critical: "#f87171", high: "#fb923c", medium: "#facc15", low: "#4ade80",
};

// ── Risk trend ────────────────────────────────────────────────────────────

function RiskTrend({ entries }: { entries: RiskEntry[] }) {
  if (entries.length < 2) return null;
  const latest = entries[entries.length - 1];
  const prev   = entries[entries.length - 2];
  if (latest.score > prev.score) return <span className="flex items-center gap-0.5 text-red-400 text-xs"><TrendingUp size={12} />rising</span>;
  if (latest.score < prev.score) return <span className="flex items-center gap-0.5 text-green-400 text-xs"><TrendingDown size={12} />falling</span>;
  return <span className="flex items-center gap-0.5 text-slate-500 text-xs"><Minus size={12} />stable</span>;
}

// ── Network SVG graph ──────────────────────────────────────────────────────

function NetworkGraph({ data, rootID }: { data: NetworkData; rootID: string }) {
  const W = 520, H = 380, CX = 260, CY = 190;

  // Place nodes in concentric rings by depth
  const positions: Record<string, { x: number; y: number }> = {};
  const byDepth: Record<number, NetworkNode[]> = {};
  data.nodes.forEach(n => {
    (byDepth[n.depth] = byDepth[n.depth] || []).push(n);
  });

  const radii: Record<number, number> = { 0: 0, 1: 115, 2: 215 };
  Object.entries(byDepth).forEach(([d, nodes]) => {
    const depth = Number(d);
    const r = radii[depth] ?? 215;
    nodes.forEach((n, i) => {
      if (depth === 0) {
        positions[n.id] = { x: CX, y: CY };
      } else {
        const angle = (i / nodes.length) * 2 * Math.PI - Math.PI / 2;
        positions[n.id] = { x: CX + r * Math.cos(angle), y: CY + r * Math.sin(angle) };
      }
    });
  });

  return (
    <svg viewBox={`0 0 ${W} ${H}`} className="w-full h-full" style={{ minHeight: 320 }}>
      {/* Edge glow filter */}
      <defs>
        <filter id="glow">
          <feGaussianBlur stdDeviation="2" result="blur" />
          <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
        </filter>
      </defs>

      {/* Edges */}
      {data.edges.map(e => {
        const s = positions[e.source];
        const t = positions[e.target];
        if (!s || !t) return null;
        const midX = (s.x + t.x) / 2;
        const midY = (s.y + t.y) / 2;
        const opacity = 0.3 + e.strength * 0.5;
        return (
          <g key={e.id}>
            <line x1={s.x} y1={s.y} x2={t.x} y2={t.y}
              stroke="#94a3b8" strokeWidth={1 + e.strength}
              strokeOpacity={opacity} strokeDasharray={e.source !== rootID && e.target !== rootID ? "4 3" : undefined} />
            <text x={midX} y={midY - 5} textAnchor="middle"
              fill="#64748b" fontSize="9" fontFamily="monospace">
              {REL_LABELS[e.relationship_type] ?? e.relationship_type}
            </text>
          </g>
        );
      })}

      {/* Nodes */}
      {data.nodes.map(n => {
        const p = positions[n.id];
        if (!p) return null;
        const r = n.is_root ? 22 : n.depth === 1 ? 16 : 12;
        const fill = n.is_root ? "#22d3ee" : (NODE_FILL[n.risk_level] ?? "#94a3b8");
        const strokeW = n.is_pep || n.is_sanctioned ? 2.5 : 1.5;
        const stroke = n.is_sanctioned ? "#f87171" : n.is_pep ? "#c084fc" : "#1e293b";
        const shortLabel = n.label.length > 12 ? n.label.slice(0, 11) + "…" : n.label;
        return (
          <Link key={n.id} href={`/subjects/${n.id}`}>
            <g className="cursor-pointer group">
              {/* outer glow ring on hover */}
              <circle cx={p.x} cy={p.y} r={r + 6}
                fill="none" stroke={fill} strokeOpacity={0}
                className="group-hover:stroke-opacity-30 transition-all" strokeWidth={8} />
              <circle cx={p.x} cy={p.y} r={r}
                fill={fill} fillOpacity={n.is_root ? 1 : 0.85}
                stroke={stroke} strokeWidth={strokeW} />
              {/* Root pulse ring */}
              {n.is_root && (
                <circle cx={p.x} cy={p.y} r={r + 8}
                  fill="none" stroke="#22d3ee" strokeOpacity={0.2} strokeWidth={2} />
              )}
              {/* PEP/Sanctioned badge */}
              {(n.is_pep || n.is_sanctioned) && (
                <circle cx={p.x + r - 3} cy={p.y - r + 3} r={5}
                  fill={n.is_sanctioned ? "#ef4444" : "#a855f7"} />
              )}
              <text x={p.x} y={p.y + r + 13} textAnchor="middle"
                fill={n.is_root ? "#e2e8f0" : "#94a3b8"}
                fontSize={n.is_root ? "11" : "9"}
                fontWeight={n.is_root ? "700" : "400"}>
                {shortLabel}
              </text>
            </g>
          </Link>
        );
      })}
    </svg>
  );
}

// ── Add Relationship Modal ─────────────────────────────────────────────────

function AddRelationshipModal({
  subjectID, onClose, onAdded,
}: { subjectID: string; onClose: () => void; onAdded: () => void }) {
  const [search, setSearch]   = useState("");
  const [toID, setToID]       = useState("");
  const [relType, setRelType] = useState("TRANSACTS_WITH");
  const [notes, setNotes]     = useState("");
  const [error, setError]     = useState("");

  const { data: searchData } = useQuery<{ data: SubjectDetail[] }>({
    queryKey: ["subject-search", search],
    queryFn: () => api.get<any>(`/api/v1/subjects?search=${encodeURIComponent(search)}&limit=8`),
    enabled: search.length >= 2,
  });

  const mutation = useMutation({
    mutationFn: () => api.post(`/api/v1/subjects/${subjectID}/relationships`, {
      to_subject_id: toID, relationship_type: relType, notes,
    }),
    onSuccess: () => { onAdded(); onClose(); },
    onError: (err: any) => setError(err.message || "Failed to create relationship"),
  });

  const results = (searchData as any)?.data ?? [];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-[#1e293b] border border-slate-700 rounded-xl w-full max-w-md p-6 shadow-2xl">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-slate-100 font-semibold text-base">Link Entity</h2>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300"><X size={16} /></button>
        </div>
        {error && <div className="mb-3 text-red-400 text-sm">{error}</div>}

        <div className="space-y-3">
          {/* Subject search */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">Search Subject *</label>
            <input value={search} onChange={e => { setSearch(e.target.value); setToID(""); }}
              placeholder="Type name or ID number…"
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500" />
            {results.length > 0 && !toID && (
              <div className="mt-1 bg-slate-900 border border-slate-600 rounded-lg overflow-hidden max-h-40 overflow-y-auto">
                {results.map((s: SubjectDetail) => (
                  s.id !== subjectID && (
                    <button key={s.id} onClick={() => { setToID(s.id); setSearch(s.canonical_name); }}
                      className="w-full text-left px-3 py-2 hover:bg-slate-800 transition-colors">
                      <span className="text-sm text-slate-200">{s.canonical_name}</span>
                      <span className="text-xs text-slate-500 ml-2">{s.id_type}: {s.id_number}</span>
                    </button>
                  )
                ))}
              </div>
            )}
            {toID && (
              <div className="mt-1 text-xs text-green-400 px-1">✓ Selected: {search}</div>
            )}
          </div>

          {/* Relationship type */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">Relationship Type *</label>
            <select value={relType} onChange={e => setRelType(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500">
              {RELATIONSHIP_TYPES.map(t => (
                <option key={t} value={t}>{REL_LABELS[t] ?? t}</option>
              ))}
            </select>
          </div>

          {/* Notes */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">Evidence / Notes</label>
            <input value={notes} onChange={e => setNotes(e.target.value)}
              placeholder="Source of evidence, transaction IDs, KYC document…"
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500" />
          </div>
        </div>

        <div className="flex gap-3 mt-5">
          <button onClick={onClose}
            className="flex-1 px-4 py-2 rounded-lg border border-slate-600 text-slate-300 text-sm hover:border-slate-500">
            Cancel
          </button>
          <button onClick={() => mutation.mutate()} disabled={mutation.isPending || !toID}
            className="flex-1 px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium disabled:opacity-50">
            {mutation.isPending ? "Linking…" : "Link Entity"}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Main page ──────────────────────────────────────────────────────────────

export default function SubjectDetailPage() {
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const [addingRel, setAddingRel] = useState(false);

  const { data: subject, isLoading } = useQuery<SubjectDetail>({
    queryKey: ["subject", id],
    queryFn: () => api.get<SubjectDetail>(`/api/v1/subjects/${id}`),
    enabled: !!id,
    select: (d: any) => d,
  });

  const { data: casesData }       = useQuery<{ cases: SubjectCase[] }>({
    queryKey: ["subject-cases", id],
    queryFn: () => api.get<{ cases: SubjectCase[] }>(`/api/v1/subjects/${id}/cases`),
    enabled: !!id,
  });

  const { data: filingsData }     = useQuery<{ filings: SubjectFiling[] }>({
    queryKey: ["subject-filings", id],
    queryFn: () => api.get<{ filings: SubjectFiling[] }>(`/api/v1/subjects/${id}/filings`),
    enabled: !!id,
  });

  const { data: riskHistoryData } = useQuery<{ history: RiskEntry[] }>({
    queryKey: ["subject-risk-history", id],
    queryFn: () => api.get<{ history: RiskEntry[] }>(`/api/v1/subjects/${id}/risk-history`),
    enabled: !!id,
  });

  const { data: relsData } = useQuery<{ relationships: Relationship[]; total: number }>({
    queryKey: ["subject-relationships", id],
    queryFn: () => api.get<any>(`/api/v1/subjects/${id}/relationships`),
    enabled: !!id,
    select: (d: any) => d,
  });

  const { data: networkData } = useQuery<NetworkData>({
    queryKey: ["subject-network", id],
    queryFn: () => api.get<NetworkData>(`/api/v1/subjects/${id}/network?depth=2`),
    enabled: !!id,
    select: (d: any) => d,
  });

  const deleteRelMutation = useMutation({
    mutationFn: (relID: string) => api.delete(`/api/v1/subjects/${id}/relationships/${relID}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["subject-relationships", id] });
      qc.invalidateQueries({ queryKey: ["subject-network", id] });
    },
  });

  if (isLoading) return (
    <div className="p-6 flex items-center justify-center h-64 text-slate-500 text-sm">
      Loading subject profile…
    </div>
  );
  if (!subject) return (
    <div className="p-6 flex items-center justify-center h-64 text-slate-500 text-sm">Subject not found.</div>
  );

  const cases   = casesData?.cases   ?? [];
  const filings = filingsData?.filings ?? [];
  const history = riskHistoryData?.history ?? [];
  const rels    = relsData?.relationships ?? [];

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
          <div className="text-right">
            <div className={`text-4xl font-black ${RISK_COLORS[subject.risk_level]}`}>{subject.risk_score}</div>
            <div className="text-slate-400 text-xs uppercase tracking-wider mt-0.5">Risk Score</div>
            <RiskTrend entries={history} />
          </div>
        </div>
        {subject.risk_factors.length > 0 && (
          <div className="flex flex-wrap gap-2 mt-4">
            {subject.risk_factors.map((f, i) => (
              <span key={i} className="px-2 py-0.5 bg-slate-800/60 text-slate-300 rounded text-xs border border-slate-700">{f}</span>
            ))}
          </div>
        )}
      </div>

      {/* 3-col grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        {/* Left: Stats + Details */}
        <div className="space-y-5">
          <div className="grid grid-cols-3 gap-3">
            {[
              { label: "Open Cases",   value: subject.open_cases,        color: "text-cyan-400" },
              { label: "Total Cases",  value: subject.total_cases,       color: "text-slate-300" },
              { label: "STR Filings",  value: subject.total_str_filings, color: "text-violet-400" },
            ].map(stat => (
              <div key={stat.label} className="bg-[#1e293b] border border-slate-700 rounded-xl p-3 text-center">
                <div className={`text-xl font-bold ${stat.color}`}>{stat.value}</div>
                <div className="text-slate-500 text-[10px] uppercase tracking-wider mt-0.5">{stat.label}</div>
              </div>
            ))}
          </div>
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4">
            <h3 className="text-slate-200 text-sm font-semibold mb-3">Details</h3>
            <dl className="space-y-2">
              {[
                { label: "KYC Tier",      value: subject.kyc_tier || "—" },
                { label: "Gender",        value: subject.gender || "—" },
                { label: "Date of Birth", value: subject.date_of_birth || "—" },
                { label: "First Seen",    value: new Date(subject.first_seen_at).toLocaleDateString() },
                { label: "Last Seen",     value: new Date(subject.last_seen_at).toLocaleDateString() },
              ].map(({ label, value }) => (
                <div key={label} className="flex justify-between text-xs">
                  <dt className="text-slate-500">{label}</dt>
                  <dd className="text-slate-300 font-mono">{value}</dd>
                </div>
              ))}
            </dl>
          </div>
          {(subject.phones.length > 0 || subject.emails.length > 0) && (
            <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4">
              <h3 className="text-slate-200 text-sm font-semibold mb-3">Contact</h3>
              {subject.phones.map((p, i) => <div key={i} className="text-slate-400 text-xs font-mono mb-1">{p}</div>)}
              {subject.emails.map((e, i) => <div key={i} className="text-slate-400 text-xs font-mono mb-1">{e}</div>)}
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
              <div className="flex items-center justify-center h-16 text-slate-600 text-xs">No cases linked</div>
            ) : (
              <div className="divide-y divide-slate-700/50">
                {cases.slice(0, 8).map(c => (
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
              <div className="flex items-center justify-center h-16 text-slate-600 text-xs">No filings on record</div>
            ) : (
              <div className="divide-y divide-slate-700/50">
                {filings.slice(0, 6).map((f: SubjectFiling) => (
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
              <div className="flex items-center justify-center h-16 text-slate-600 text-xs">No risk history recorded</div>
            ) : (
              <div className="divide-y divide-slate-700/50">
                {history.slice(-8).reverse().map((entry, i) => (
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

      {/* ── Connected Entity Network ─────────────────────────────────────── */}
      <div className="mt-6 bg-[#1e293b] border border-slate-700 rounded-xl overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-700 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <GitFork size={14} className="text-cyan-400" />
            <span className="text-sm font-semibold text-slate-200">Connected Entity Network</span>
            {rels.length > 0 && (
              <span className="px-2 py-0.5 bg-cyan-500/15 text-cyan-400 rounded-full text-[10px] font-semibold border border-cyan-500/30">
                {rels.length} connection{rels.length !== 1 ? "s" : ""}
              </span>
            )}
            {networkData && networkData.high_risk_count > 0 && (
              <span className="px-2 py-0.5 bg-red-500/15 text-red-400 rounded-full text-[10px] font-semibold border border-red-500/30">
                {networkData.high_risk_count} high-risk node{networkData.high_risk_count !== 1 ? "s" : ""}
              </span>
            )}
          </div>
          <button onClick={() => setAddingRel(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg text-xs font-medium transition-colors">
            <Plus size={12} />Link Entity
          </button>
        </div>

        {rels.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-32 text-slate-600">
            <GitFork size={28} className="mb-2 text-slate-700" />
            <p className="text-sm">No connections recorded</p>
            <p className="text-xs mt-1">Link related entities to build institutional memory</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-0 divide-x divide-slate-700/50">
            {/* Left: connections table */}
            <div>
              <div className="divide-y divide-slate-700/40">
                {rels.map(rel => (
                  <div key={rel.id} className="flex items-start justify-between px-4 py-3 hover:bg-slate-700/15 transition-colors group">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <Link href={`/subjects/${rel.peer_subject_id}`}
                          className="text-sm text-slate-200 hover:text-cyan-400 font-medium transition-colors">
                          {rel.peer_name}
                        </Link>
                        {rel.peer_is_pep && (
                          <span className="text-[9px] px-1 bg-violet-500/15 text-violet-400 rounded border border-violet-500/20">PEP</span>
                        )}
                        {rel.peer_is_sanctioned && (
                          <span className="text-[9px] px-1 bg-red-500/15 text-red-400 rounded border border-red-500/20">SANC</span>
                        )}
                        <span className={`text-[9px] capitalize px-1 py-0.5 rounded ${
                          rel.peer_risk_level === "critical" ? "bg-red-500/10 text-red-400" :
                          rel.peer_risk_level === "high"     ? "bg-orange-500/10 text-orange-400" :
                          rel.peer_risk_level === "medium"   ? "bg-yellow-500/10 text-yellow-400" :
                          "bg-slate-700 text-slate-500"
                        }`}>{rel.peer_risk_level}</span>
                      </div>
                      <div className="flex items-center gap-2 mt-0.5">
                        <span className="text-[10px] text-slate-500">
                          {rel.direction === "outbound" ? "→" : "←"}&nbsp;
                          <span className="text-slate-400">{REL_LABELS[rel.relationship_type] ?? rel.relationship_type}</span>
                        </span>
                        {rel.strength < 1 && (
                          <span className="text-[9px] text-slate-600">· {Math.round(rel.strength * 100)}% confidence</span>
                        )}
                      </div>
                      {rel.notes && (
                        <div className="text-[10px] text-slate-600 mt-0.5 truncate max-w-xs">{rel.notes}</div>
                      )}
                    </div>
                    <button onClick={() => {
                      if (confirm(`Remove connection to ${rel.peer_name}?`)) {
                        deleteRelMutation.mutate(rel.id);
                      }
                    }} className="opacity-0 group-hover:opacity-100 p-1 text-slate-600 hover:text-red-400 transition-all ml-2 flex-shrink-0">
                      <Trash2 size={12} />
                    </button>
                  </div>
                ))}
              </div>
            </div>

            {/* Right: SVG network graph */}
            <div className="p-3 flex items-center justify-center bg-slate-900/20">
              {networkData && networkData.nodes.length > 1 ? (
                <div className="w-full">
                  <NetworkGraph data={networkData} rootID={id ?? ""} />
                  <div className="flex items-center gap-4 justify-center mt-1 px-3">
                    <div className="flex items-center gap-1.5 text-[10px] text-slate-500">
                      <div className="w-3 h-3 rounded-full bg-cyan-400" />Root
                    </div>
                    <div className="flex items-center gap-1.5 text-[10px] text-slate-500">
                      <div className="w-3 h-3 rounded-full bg-green-400" />Low risk
                    </div>
                    <div className="flex items-center gap-1.5 text-[10px] text-slate-500">
                      <div className="w-3 h-3 rounded-full bg-orange-400" />High risk
                    </div>
                    <div className="flex items-center gap-1.5 text-[10px] text-slate-500">
                      <div className="w-3 h-3 rounded-full bg-red-400" />Critical / Sanctioned
                    </div>
                  </div>
                </div>
              ) : (
                <div className="text-slate-600 text-xs text-center py-8">
                  Graph renders when 2+ connections exist
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Notes */}
      {subject.notes && (
        <div className="mt-5 bg-[#1e293b] border border-slate-700 rounded-xl p-4">
          <h3 className="text-slate-200 text-sm font-semibold mb-2">Notes</h3>
          <p className="text-slate-400 text-sm whitespace-pre-wrap">{subject.notes}</p>
        </div>
      )}

      {addingRel && (
        <AddRelationshipModal
          subjectID={id ?? ""}
          onClose={() => setAddingRel(false)}
          onAdded={() => {
            qc.invalidateQueries({ queryKey: ["subject-relationships", id] });
            qc.invalidateQueries({ queryKey: ["subject-network", id] });
          }}
        />
      )}
    </div>
  );
}
