"use client";

import { use, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { CaseSubNav } from "@/components/cases/CaseSubNav";
import { PageError } from "@/components/ui/PageError";
import {
  Shield, ShieldAlert, ShieldCheck, Sparkles,
  Plus, ChevronDown, ChevronUp, Paperclip, CheckCircle2,
  Clock, XCircle, AlertTriangle, Eye,
} from "lucide-react";

// ── Types ─────────────────────────────────────────────────────────────────────

interface UserRef {
  id: string;
  full_name: string;
}

interface EvidenceLink {
  id: string;
  evidence_type: string;
  evidence_id: string;
  citation: string;
  relevance_note: string;
  linked_by: string;
  linked_at: string;
}

interface Finding {
  id: string;
  case_id: string;
  title: string;
  description: string;
  category: string;
  severity: string;
  status: string;
  narrative_weight: string;
  confidence_score?: number;
  ai_generated: boolean;
  evidence_count: number;
  created_by: UserRef;
  reviewed_by?: UserRef;
  approved_by?: UserRef;
  reviewed_at?: string;
  approved_at?: string;
  evidence_links?: EvidenceLink[];
  created_at: string;
  updated_at: string;
}

interface FindingsSummary {
  total: number;
  approved: number;
  critical: number;
  high: number;
}

interface CaseData {
  case_number: string;
  title: string;
}

const SEVERITY_ORDER: Record<string, number> = { critical: 0, high: 1, medium: 2, low: 3 };

const severityConfig: Record<string, { color: string; bg: string; icon: React.ReactNode }> = {
  critical: { color: "text-red-400", bg: "bg-red-500/10 border-red-500/30", icon: <ShieldAlert size={14} /> },
  high:     { color: "text-orange-400", bg: "bg-orange-500/10 border-orange-500/30", icon: <Shield size={14} /> },
  medium:   { color: "text-yellow-400", bg: "bg-yellow-500/10 border-yellow-500/30", icon: <Shield size={14} /> },
  low:      { color: "text-slate-400", bg: "bg-slate-700/40 border-slate-600/30", icon: <ShieldCheck size={14} /> },
};

const statusConfig: Record<string, { label: string; color: string; icon: React.ReactNode }> = {
  draft:        { label: "Draft",       color: "text-slate-400 bg-slate-700/40",      icon: <Clock size={11} /> },
  under_review: { label: "In Review",   color: "text-blue-400 bg-blue-500/10",        icon: <Eye size={11} /> },
  approved:     { label: "Approved",    color: "text-emerald-400 bg-emerald-500/10",  icon: <CheckCircle2 size={11} /> },
  rejected:     { label: "Rejected",    color: "text-red-400 bg-red-500/10",          icon: <XCircle size={11} /> },
  contested:    { label: "Contested",   color: "text-purple-400 bg-purple-500/10",    icon: <AlertTriangle size={11} /> },
  archived:     { label: "Archived",    color: "text-slate-500 bg-slate-800/40",      icon: <Clock size={11} /> },
};

const CATEGORY_LABELS: Record<string, string> = {
  structuring: "Structuring", layering: "Layering", integration: "Integration",
  smurfing: "Smurfing", round_tripping: "Round Tripping", pep_exposure: "PEP Exposure",
  sanctions_evasion: "Sanctions Evasion", unusual_cash: "Unusual Cash", rapid_movement: "Rapid Movement",
  identity_fraud: "Identity Fraud", account_takeover: "Account Takeover", other: "Other",
};

// ── Skeleton ──────────────────────────────────────────────────────────────────

function SkeletonFinding() {
  return (
    <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-4 space-y-3 animate-pulse">
      <div className="flex items-center gap-2">
        <div className="h-4 bg-slate-700 rounded w-16" />
        <div className="h-5 bg-slate-700 rounded w-64" />
      </div>
      <div className="h-3 bg-slate-700/60 rounded w-full" />
      <div className="h-3 bg-slate-700/60 rounded w-4/5" />
      <div className="flex gap-2 mt-1">
        <div className="h-5 bg-slate-700/40 rounded w-20" />
        <div className="h-5 bg-slate-700/40 rounded w-16" />
      </div>
    </div>
  );
}

// ── Create Finding Form ───────────────────────────────────────────────────────

function CreateFindingForm({ caseId, onSuccess, onCancel }: {
  caseId: string;
  onSuccess: () => void;
  onCancel: () => void;
}) {
  const [form, setForm] = useState({
    title: "", description: "", category: "other", severity: "medium", narrative_weight: "supporting",
  });
  const qc = useQueryClient();

  const mutation = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${caseId}/findings`, form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["findings", caseId] });
      qc.invalidateQueries({ queryKey: ["findings-summary", caseId] });
      onSuccess();
    },
  });

  return (
    <div className="bg-[#0f1827] border border-cyan-500/30 rounded-xl p-5 space-y-4">
      <h3 className="text-sm font-semibold text-slate-200">New Finding</h3>

      <div className="space-y-1">
        <label className="text-xs text-slate-400">Title *</label>
        <input
          value={form.title}
          onChange={e => setForm(f => ({ ...f, title: e.target.value }))}
          placeholder="e.g. Structuring through multiple cash deposits below ₹10L threshold"
          className="w-full bg-slate-800 border border-slate-600 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-cyan-500"
        />
      </div>

      <div className="space-y-1">
        <label className="text-xs text-slate-400">Description *</label>
        <textarea
          rows={4}
          value={form.description}
          onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
          placeholder="Describe the pattern, timeline, amounts, entities involved, and why this is suspicious..."
          className="w-full bg-slate-800 border border-slate-600 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-cyan-500 resize-none"
        />
      </div>

      <div className="grid grid-cols-3 gap-3">
        <div className="space-y-1">
          <label className="text-xs text-slate-400">Category</label>
          <select
            value={form.category}
            onChange={e => setForm(f => ({ ...f, category: e.target.value }))}
            className="w-full bg-slate-800 border border-slate-600 rounded-lg px-2 py-2 text-sm text-slate-200 focus:outline-none focus:border-cyan-500"
          >
            {Object.entries(CATEGORY_LABELS).map(([v, l]) => (
              <option key={v} value={v}>{l}</option>
            ))}
          </select>
        </div>
        <div className="space-y-1">
          <label className="text-xs text-slate-400">Severity</label>
          <select
            value={form.severity}
            onChange={e => setForm(f => ({ ...f, severity: e.target.value }))}
            className="w-full bg-slate-800 border border-slate-600 rounded-lg px-2 py-2 text-sm text-slate-200 focus:outline-none focus:border-cyan-500"
          >
            {["critical","high","medium","low"].map(s => (
              <option key={s} value={s}>{s.charAt(0).toUpperCase() + s.slice(1)}</option>
            ))}
          </select>
        </div>
        <div className="space-y-1">
          <label className="text-xs text-slate-400">Weight in Narrative</label>
          <select
            value={form.narrative_weight}
            onChange={e => setForm(f => ({ ...f, narrative_weight: e.target.value }))}
            className="w-full bg-slate-800 border border-slate-600 rounded-lg px-2 py-2 text-sm text-slate-200 focus:outline-none focus:border-cyan-500"
          >
            <option value="primary">Primary</option>
            <option value="supporting">Supporting</option>
            <option value="contextual">Contextual</option>
          </select>
        </div>
      </div>

      {mutation.error && (
        <p className="text-xs text-red-400">{String((mutation.error as Error).message)}</p>
      )}

      <div className="flex items-center gap-2 pt-1">
        <button
          onClick={() => mutation.mutate()}
          disabled={!form.title.trim() || !form.description.trim() || mutation.isPending}
          className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition-colors"
        >
          {mutation.isPending ? "Creating..." : "Create Finding"}
        </button>
        <button
          onClick={onCancel}
          className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-300 text-sm rounded-lg transition-colors"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}

// ── Link Evidence Form ────────────────────────────────────────────────────────

function LinkEvidenceForm({ caseId, findingId, onSuccess, onCancel }: {
  caseId: string;
  findingId: string;
  onSuccess: () => void;
  onCancel: () => void;
}) {
  const [form, setForm] = useState({
    evidence_type: "document", evidence_id: "", citation: "", relevance_note: "",
  });
  const qc = useQueryClient();

  const mutation = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${caseId}/findings/${findingId}/evidence`, form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["finding", caseId, findingId] });
      qc.invalidateQueries({ queryKey: ["findings", caseId] });
      onSuccess();
    },
  });

  return (
    <div className="bg-slate-800/60 border border-slate-600 rounded-lg p-4 space-y-3 mt-3">
      <h4 className="text-xs font-semibold text-slate-300">Link Evidence</h4>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <label className="text-xs text-slate-400">Type</label>
          <select
            value={form.evidence_type}
            onChange={e => setForm(f => ({ ...f, evidence_type: e.target.value }))}
            className="w-full bg-slate-700 border border-slate-600 rounded px-2 py-1.5 text-xs text-slate-200 focus:outline-none"
          >
            {["document","note","ai_log","subject","alert"].map(t => (
              <option key={t} value={t}>{t.charAt(0).toUpperCase() + t.slice(1)}</option>
            ))}
          </select>
        </div>
        <div className="space-y-1">
          <label className="text-xs text-slate-400">ID (UUID)</label>
          <input
            value={form.evidence_id}
            onChange={e => setForm(f => ({ ...f, evidence_id: e.target.value }))}
            placeholder="Paste evidence UUID"
            className="w-full bg-slate-700 border border-slate-600 rounded px-2 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none"
          />
        </div>
      </div>
      <div className="space-y-1">
        <label className="text-xs text-slate-400">Citation (e.g. "Page 3, row 7 — wire transfer ₹45L")</label>
        <input
          value={form.citation}
          onChange={e => setForm(f => ({ ...f, citation: e.target.value }))}
          className="w-full bg-slate-700 border border-slate-600 rounded px-2 py-1.5 text-xs text-slate-200 focus:outline-none"
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs text-slate-400">Why is this evidence relevant?</label>
        <input
          value={form.relevance_note}
          onChange={e => setForm(f => ({ ...f, relevance_note: e.target.value }))}
          className="w-full bg-slate-700 border border-slate-600 rounded px-2 py-1.5 text-xs text-slate-200 focus:outline-none"
        />
      </div>
      <div className="flex gap-2">
        <button
          onClick={() => mutation.mutate()}
          disabled={!form.evidence_id.trim() || mutation.isPending}
          className="px-3 py-1.5 bg-cyan-700 hover:bg-cyan-600 disabled:opacity-50 text-white text-xs rounded transition-colors"
        >
          {mutation.isPending ? "Linking..." : "Link"}
        </button>
        <button onClick={onCancel} className="px-3 py-1.5 bg-slate-700 text-slate-300 text-xs rounded">Cancel</button>
      </div>
    </div>
  );
}

// ── Finding Card ──────────────────────────────────────────────────────────────

function FindingCard({ finding, caseId, userRole }: {
  finding: Finding;
  caseId: string;
  userRole: string;
}) {
  const [expanded, setExpanded] = useState(false);
  const [showLinkForm, setShowLinkForm] = useState(false);
  const qc = useQueryClient();

  const sev = severityConfig[finding.severity] ?? severityConfig.low;
  const stat = statusConfig[finding.status] ?? statusConfig.draft;
  const isPrivileged = ["manager","admin","super_admin"].includes(userRole);

  const transition = useMutation({
    mutationFn: (action: string) =>
      api.post(`/api/v1/cases/${caseId}/findings/${finding.id}/${action}`, {}),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["findings", caseId] });
      qc.invalidateQueries({ queryKey: ["findings-summary", caseId] });
    },
  });

  // Fetch full finding with evidence when expanded
  const { data: fullFinding } = useQuery<Finding>({
    queryKey: ["finding", caseId, finding.id],
    queryFn: () => api.get(`/api/v1/cases/${caseId}/findings/${finding.id}`),
    enabled: expanded,
  });

  return (
    <div className={`border rounded-xl overflow-hidden transition-all ${sev.bg}`}>
      {/* Header row */}
      <div
        className="flex items-start gap-3 p-4 cursor-pointer"
        onClick={() => setExpanded(e => !e)}
      >
        <div className={`mt-0.5 ${sev.color}`}>{sev.icon}</div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className={`text-xs font-semibold uppercase ${sev.color}`}>
              {finding.severity}
            </span>
            <span className="text-xs text-slate-500">·</span>
            <span className="text-xs text-slate-400">
              {CATEGORY_LABELS[finding.category] ?? finding.category}
            </span>
            {finding.ai_generated && (
              <span className="flex items-center gap-1 text-xs text-violet-400">
                <Sparkles size={10} /> AI
                {finding.confidence_score != null && (
                  <span className="text-slate-500">({Math.round(finding.confidence_score * 100)}%)</span>
                )}
              </span>
            )}
            <span className={`ml-auto flex items-center gap-1 text-xs px-2 py-0.5 rounded-full ${stat.color}`}>
              {stat.icon}
              {stat.label}
            </span>
          </div>
          <p className="text-sm font-medium text-slate-200 mt-0.5 leading-snug">{finding.title}</p>
          <div className="flex items-center gap-3 mt-1.5">
            <span className="text-xs text-slate-500">by {finding.created_by.full_name}</span>
            {finding.evidence_count > 0 && (
              <span className="flex items-center gap-1 text-xs text-slate-400">
                <Paperclip size={10} /> {finding.evidence_count} evidence
              </span>
            )}
          </div>
        </div>
        <div className="text-slate-500">
          {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
        </div>
      </div>

      {/* Expanded content */}
      {expanded && (
        <div className="border-t border-slate-700/50 px-4 pb-4 pt-3 space-y-4">
          <p className="text-sm text-slate-300 whitespace-pre-wrap leading-relaxed">
            {finding.description}
          </p>

          {/* Evidence list */}
          {(fullFinding?.evidence_links ?? []).length > 0 && (
            <div className="space-y-2">
              <h4 className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Evidence</h4>
              {(fullFinding?.evidence_links ?? []).map(link => (
                <div key={link.id} className="flex gap-2 bg-slate-800/50 rounded-lg p-2.5">
                  <span className="text-xs text-cyan-400 font-medium shrink-0 capitalize">
                    {link.evidence_type}
                  </span>
                  <div className="flex-1 min-w-0">
                    {link.citation && (
                      <p className="text-xs text-slate-300 italic">&ldquo;{link.citation}&rdquo;</p>
                    )}
                    {link.relevance_note && (
                      <p className="text-xs text-slate-400 mt-0.5">{link.relevance_note}</p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Link evidence form */}
          {showLinkForm ? (
            <LinkEvidenceForm
              caseId={caseId}
              findingId={finding.id}
              onSuccess={() => setShowLinkForm(false)}
              onCancel={() => setShowLinkForm(false)}
            />
          ) : (
            finding.status === "draft" || finding.status === "under_review" ? (
              <button
                onClick={(e) => { e.stopPropagation(); setShowLinkForm(true); }}
                className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-cyan-400 transition-colors"
              >
                <Paperclip size={12} /> Attach Evidence
              </button>
            ) : null
          )}

          {/* Action buttons */}
          <div className="flex items-center gap-2 pt-1 flex-wrap">
            {finding.status === "draft" && (
              <button
                onClick={(e) => { e.stopPropagation(); transition.mutate("submit"); }}
                disabled={transition.isPending}
                className="px-3 py-1.5 bg-blue-700/60 hover:bg-blue-600/60 text-blue-300 text-xs rounded-lg transition-colors"
              >
                Submit for Review
              </button>
            )}
            {finding.status === "under_review" && isPrivileged && (
              <>
                <button
                  onClick={(e) => { e.stopPropagation(); transition.mutate("approve"); }}
                  disabled={transition.isPending}
                  className="px-3 py-1.5 bg-emerald-700/60 hover:bg-emerald-600/60 text-emerald-300 text-xs rounded-lg transition-colors"
                >
                  Approve
                </button>
                <button
                  onClick={(e) => { e.stopPropagation(); transition.mutate("reject"); }}
                  disabled={transition.isPending}
                  className="px-3 py-1.5 bg-red-700/60 hover:bg-red-600/60 text-red-300 text-xs rounded-lg transition-colors"
                >
                  Reject
                </button>
              </>
            )}
            {finding.status === "rejected" && finding.created_by.id && (
              <button
                onClick={(e) => { e.stopPropagation(); transition.mutate("contest"); }}
                disabled={transition.isPending}
                className="px-3 py-1.5 bg-purple-700/60 hover:bg-purple-600/60 text-purple-300 text-xs rounded-lg transition-colors"
              >
                Contest Rejection
              </button>
            )}
            {finding.approved_by && (
              <span className="text-xs text-emerald-400 ml-auto">
                Approved by {finding.approved_by.full_name}
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ── Main Page ─────────────────────────────────────────────────────────────────

export default function CaseFindingsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const user = useAuthStore((s) => s.user);
  const [showCreate, setShowCreate] = useState(false);
  const [severityFilter, setSeverityFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");

  const { data: caseData } = useQuery<CaseData>({
    queryKey: ["case", id],
    queryFn: () => api.get(`/api/v1/cases/${id}`),
  });

  const { data: summaryData } = useQuery<FindingsSummary>({
    queryKey: ["findings-summary", id],
    queryFn: () => api.get(`/api/v1/cases/${id}/findings/summary`),
  });

  const { data, isLoading, isError } = useQuery<Finding[]>({
    queryKey: ["findings", id, statusFilter, severityFilter],
    queryFn: () => {
      const params = new URLSearchParams();
      if (statusFilter) params.set("status", statusFilter);
      if (severityFilter) params.set("severity", severityFilter);
      params.set("limit", "50");
      return api.get(`/api/v1/cases/${id}/findings?${params}`);
    },
  });

  const findings = data ?? [];

  if (isError) return (
    <PageError title="Failed to load findings" />
  );

  const summary = summaryData;

  return (
    <div className="p-6">
      <CaseSubNav
        caseId={id}
        activeTab="findings"
        caseNumber={caseData?.case_number}
        action={
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-medium rounded-lg transition-colors"
          >
            <Plus size={13} /> New Finding
          </button>
        }
      />

      <div className="mb-6">
        <h1 className="text-xl font-bold text-white">Findings</h1>
        <p className="text-slate-400 text-sm mt-0.5">Documented investigation findings — approved findings carry evidential weight for STR and case closure.</p>
      </div>

      {/* Summary stats */}
      {summary && (
        <div className="grid grid-cols-4 gap-3 mb-5">
          {[
            { label: "Total", value: summary.total, color: "text-slate-300" },
            { label: "Approved", value: summary.approved, color: "text-emerald-400" },
            { label: "Critical", value: summary.critical, color: "text-red-400" },
            { label: "High", value: summary.high, color: "text-orange-400" },
          ].map(stat => (
            <div key={stat.label} className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-3 text-center">
              <div className={`text-2xl font-bold ${stat.color}`}>{stat.value}</div>
              <div className="text-xs text-slate-500 mt-0.5">{stat.label}</div>
            </div>
          ))}
        </div>
      )}

      {/* Create form */}
      {showCreate && (
        <div className="mb-5">
          <CreateFindingForm
            caseId={id}
            onSuccess={() => setShowCreate(false)}
            onCancel={() => setShowCreate(false)}
          />
        </div>
      )}

      {/* Filters */}
      <div className="flex items-center gap-3 mb-4">
        <select
          value={severityFilter}
          onChange={e => setSeverityFilter(e.target.value)}
          className="bg-slate-800 border border-slate-600 rounded-lg px-2.5 py-1.5 text-xs text-slate-300 focus:outline-none"
        >
          <option value="">All Severities</option>
          {["critical","high","medium","low"].map(s => (
            <option key={s} value={s}>{s.charAt(0).toUpperCase() + s.slice(1)}</option>
          ))}
        </select>
        <select
          value={statusFilter}
          onChange={e => setStatusFilter(e.target.value)}
          className="bg-slate-800 border border-slate-600 rounded-lg px-2.5 py-1.5 text-xs text-slate-300 focus:outline-none"
        >
          <option value="">All Statuses</option>
          {["draft","under_review","approved","rejected","contested"].map(s => (
            <option key={s} value={s}>{statusConfig[s]?.label ?? s}</option>
          ))}
        </select>
        {(severityFilter || statusFilter) && (
          <button
            onClick={() => { setSeverityFilter(""); setStatusFilter(""); }}
            className="text-xs text-slate-400 hover:text-slate-200"
          >
            Clear
          </button>
        )}
        <span className="ml-auto text-xs text-slate-500">{findings.length} finding{findings.length !== 1 ? "s" : ""}</span>
      </div>

      {/* Loading state */}
      {isLoading && (
        <div className="space-y-3">
          {[1,2,3].map(i => <SkeletonFinding key={i} />)}
        </div>
      )}

      {/* Empty state */}
      {!isLoading && findings.length === 0 && (
        <div className="text-center py-16 space-y-3">
          <Shield size={40} className="mx-auto text-slate-600" />
          <p className="text-slate-400 font-medium">No formal findings yet</p>
          <p className="text-slate-500 text-sm max-w-sm mx-auto">
            Create a finding to document what suspicious activity you&apos;ve identified
            and link evidence that supports your conclusion.
          </p>
          <button
            onClick={() => setShowCreate(true)}
            className="mt-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm rounded-lg transition-colors"
          >
            Create First Finding
          </button>
        </div>
      )}

      {/* Findings list — sorted by severity then created_at */}
      {!isLoading && findings.length > 0 && (
        <div className="space-y-3">
          {[...findings]
            .sort((a, b) => (SEVERITY_ORDER[a.severity] ?? 4) - (SEVERITY_ORDER[b.severity] ?? 4))
            .map(finding => (
              <FindingCard
                key={finding.id}
                finding={finding}
                caseId={id}
                userRole={user?.role ?? "viewer"}
              />
            ))}
        </div>
      )}
    </div>
  );
}
