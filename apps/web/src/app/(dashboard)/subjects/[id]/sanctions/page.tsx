"use client";

import { use, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/hooks/useAuth";
import { ShieldAlert, ShieldCheck, RefreshCw, Loader2, CheckCircle2, XCircle, AlertTriangle, ChevronRight } from "lucide-react";

interface SanctionsEntry {
  id: string;
  list_source: string;
  external_id: string;
  entity_type: string;
  primary_name: string;
  aliases: string[];
  nationality: string;
  birth_date: string;
  programs: string[];
  remarks?: string;
}

interface Hit {
  id: string;
  subject_id: string;
  entry_id: string;
  entry?: SanctionsEntry;
  match_type: "exact" | "fuzzy" | "alias";
  match_score: number;
  matched_name: string;
  screened_at: string;
  status: "open" | "false_positive" | "confirmed" | "escalated";
  reviewed_by?: string;
  reviewed_at?: string;
  notes?: string;
}

const SOURCE_LABELS: Record<string, { label: string; color: string }> = {
  eu_consolidated: { label: "EU Consolidated",  color: "bg-blue-500/15 text-blue-300 border-blue-500/30" },
  un_consolidated: { label: "UN Security Council", color: "bg-red-500/15 text-red-300 border-red-500/30" },
  ofac_sdn:        { label: "OFAC SDN",          color: "bg-amber-500/15 text-amber-300 border-amber-500/30" },
};

const STATUS_CONFIG: Record<string, { label: string; color: string; Icon: React.ElementType }> = {
  open:           { label: "Open",           color: "bg-red-500/15 text-red-300 border-red-500/30",       Icon: AlertTriangle },
  false_positive: { label: "False Positive", color: "bg-slate-500/15 text-slate-300 border-slate-500/30", Icon: XCircle },
  confirmed:      { label: "Confirmed",      color: "bg-red-600/20 text-red-200 border-red-600/40",       Icon: ShieldAlert },
  escalated:      { label: "Escalated",      color: "bg-orange-500/15 text-orange-300 border-orange-500/30", Icon: ChevronRight },
};

function MatchScore({ score }: { score: number }) {
  const pct = Math.round(score * 100);
  const color = pct >= 95 ? "text-red-400" : pct >= 85 ? "text-amber-400" : "text-slate-400";
  return <span className={`font-mono text-xs ${color}`}>{pct}%</span>;
}

function AcknowledgeModal({ hit, onClose, onDone }: { hit: Hit; onClose: () => void; onDone: () => void }) {
  const [status, setStatus] = useState<string>("");
  const [notes, setNotes] = useState("");
  const mut = useMutation({
    mutationFn: () => api.patch(`/api/v1/sanctions/hits/${hit.id}`, { status, notes }),
    onSuccess: () => { onDone(); onClose(); },
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-[#0d1a2d] border border-slate-700 rounded-xl w-full max-w-md shadow-2xl p-6">
        <h3 className="text-base font-semibold text-slate-100 mb-1">Review Sanctions Hit</h3>
        <p className="text-slate-500 text-xs mb-4">
          <strong className="text-slate-300">{hit.entry?.primary_name}</strong>
          {" · "}{SOURCE_LABELS[hit.entry?.list_source ?? ""]?.label ?? hit.entry?.list_source}
          {hit.entry?.programs?.length ? " · " + hit.entry.programs.join(", ") : ""}
        </p>
        <div className="space-y-3">
          <div>
            <label className="block text-xs text-slate-400 mb-1">Decision</label>
            <div className="flex gap-2 flex-wrap">
              {[
                { v: "false_positive", l: "False Positive" },
                { v: "confirmed",      l: "Confirm Match" },
                { v: "escalated",      l: "Escalate" },
              ].map(({ v, l }) => (
                <button
                  key={v}
                  onClick={() => setStatus(v)}
                  className={`px-3 py-1.5 text-xs rounded border transition-colors ${
                    status === v
                      ? "bg-cyan-600 border-cyan-500 text-white"
                      : "bg-slate-800 border-slate-600 text-slate-400 hover:text-slate-200"
                  }`}
                >
                  {l}
                </button>
              ))}
            </div>
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Notes</label>
            <textarea
              rows={3}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Explain the decision (required for audit trail)…"
              className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-cyan-500 resize-none"
            />
          </div>
        </div>
        <div className="flex gap-2 mt-4">
          <button
            onClick={() => mut.mutate()}
            disabled={!status || !notes || mut.isPending}
            className="flex-1 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm font-medium rounded"
          >
            {mut.isPending ? "Saving…" : "Save Decision"}
          </button>
          <button onClick={onClose} className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm">
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

export default function SubjectSanctionsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const { user } = useAuth();
  const qc = useQueryClient();
  const [reviewing, setReviewing] = useState<Hit | null>(null);
  const [screenName, setScreenName] = useState("");

  const { data: hits = [], isLoading } = useQuery<Hit[]>({
    queryKey: ["sanctions-hits", id],
    queryFn: () => api.get(`/api/v1/subjects/${id}/sanctions/hits`),
  });

  const screenMut = useMutation({
    mutationFn: (name: string) =>
      api.post(`/api/v1/subjects/${id}/sanctions/screen`, { subject_name: name }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sanctions-hits", id] });
      setScreenName("");
    },
  });

  const canWrite = ["analyst", "manager", "admin", "super_admin"].includes(user?.role ?? "");
  const canReview = ["manager", "admin", "super_admin"].includes(user?.role ?? "");

  const openHits = hits.filter((h) => h.status === "open" || h.status === "escalated");
  const confirmedHits = hits.filter((h) => h.status === "confirmed");
  const closedHits = hits.filter((h) => h.status === "false_positive");

  const statusIcon = confirmedHits.length > 0
    ? <ShieldAlert size={16} className="text-red-400" />
    : openHits.length > 0
    ? <AlertTriangle size={16} className="text-amber-400" />
    : <ShieldCheck size={16} className="text-emerald-400" />;

  const statusLabel = confirmedHits.length > 0
    ? "Confirmed Sanctions Match"
    : openHits.length > 0
    ? "Unresolved Hits — Review Required"
    : "Clear";

  const statusColor = confirmedHits.length > 0
    ? "bg-red-500/10 border-red-500/30 text-red-300"
    : openHits.length > 0
    ? "bg-amber-500/10 border-amber-500/30 text-amber-300"
    : "bg-emerald-500/10 border-emerald-500/30 text-emerald-300";

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="mb-6">
        <a href={`/subjects/${id}`} className="text-slate-400 hover:text-slate-200 text-sm transition-colors block mb-1">
          ← Subject
        </a>
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-semibold text-slate-100">Sanctions & PEP Screening</h1>
          <div className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full border text-xs font-medium ${statusColor}`}>
            {statusIcon}
            {statusLabel}
          </div>
        </div>
        <p className="text-slate-500 text-xs mt-1">
          EU Consolidated List · UN Security Council · OFAC SDN
        </p>
      </div>

      {/* Screen Now panel */}
      {canWrite && (
        <div className="mb-6 p-4 bg-[#0e1929] border border-[#1e3050] rounded-xl">
          <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-3">Run New Screen</h2>
          <div className="flex gap-2">
            <input
              value={screenName}
              onChange={(e) => setScreenName(e.target.value)}
              placeholder="Name to screen (as it appears on official documents)"
              className="flex-1 bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-cyan-500"
            />
            <button
              onClick={() => screenMut.mutate(screenName)}
              disabled={!screenName || screenMut.isPending}
              className="flex items-center gap-1.5 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm font-medium rounded transition-colors"
            >
              {screenMut.isPending ? <Loader2 size={14} className="animate-spin" /> : <RefreshCw size={14} />}
              Screen
            </button>
          </div>
          {screenMut.isSuccess && (
            <p className="text-xs mt-2 text-slate-400">
              Screening complete — {((screenMut.data as Record<string, number>)?.hit_count ?? 0)} hits found.
            </p>
          )}
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center gap-2 text-slate-500 text-sm py-12 justify-center">
          <Loader2 size={16} className="animate-spin" /> Loading hits…
        </div>
      ) : hits.length === 0 ? (
        <div className="text-center py-12 text-slate-500 text-sm">
          <ShieldCheck size={32} className="text-emerald-500 mx-auto mb-3" />
          No screening hits. Run a screen above to check against EU/UN/OFAC lists.
        </div>
      ) : (
        <div className="space-y-4">
          {[
            { title: "Open / Escalated", items: openHits },
            { title: "Confirmed Matches", items: confirmedHits },
            { title: "Closed (False Positive)", items: closedHits },
          ].map(({ title, items }) => items.length > 0 && (
            <div key={title}>
              <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">{title} ({items.length})</h2>
              <div className="bg-[#0e1929] border border-[#1e3050] rounded-xl overflow-hidden">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-slate-700/60">
                      {["List", "Matched Name", "Match Type", "Score", "Programs", "Status", ""].map((h) => (
                        <th key={h} className="px-4 py-2.5 text-left text-[11px] font-medium text-slate-500 uppercase tracking-wider">{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {items.map((hit) => {
                      const src = SOURCE_LABELS[hit.entry?.list_source ?? ""] ?? { label: hit.entry?.list_source ?? "—", color: "" };
                      const st = STATUS_CONFIG[hit.status] ?? STATUS_CONFIG.open;
                      const StIcon = st.Icon;
                      return (
                        <tr key={hit.id} className="border-b border-slate-700/40 hover:bg-slate-800/30">
                          <td className="px-4 py-3">
                            <span className={`text-[11px] px-2 py-0.5 rounded-full border ${src.color}`}>{src.label}</span>
                          </td>
                          <td className="px-4 py-3 text-sm text-slate-200 max-w-[180px] truncate">{hit.matched_name}</td>
                          <td className="px-4 py-3 text-xs text-slate-400 capitalize">{hit.match_type}</td>
                          <td className="px-4 py-3"><MatchScore score={hit.match_score} /></td>
                          <td className="px-4 py-3 text-xs text-slate-400 max-w-[140px] truncate">
                            {hit.entry?.programs?.join(", ") || "—"}
                          </td>
                          <td className="px-4 py-3">
                            <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full border text-[11px] ${st.color}`}>
                              <StIcon size={10} />{st.label}
                            </span>
                          </td>
                          <td className="px-4 py-3">
                            {canReview && hit.status === "open" && (
                              <button
                                onClick={() => setReviewing(hit)}
                                className="text-xs text-cyan-400 hover:text-cyan-300 transition-colors"
                              >
                                Review
                              </button>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          ))}
        </div>
      )}

      {reviewing && (
        <AcknowledgeModal
          hit={reviewing}
          onClose={() => setReviewing(null)}
          onDone={() => qc.invalidateQueries({ queryKey: ["sanctions-hits", id] })}
        />
      )}
    </div>
  );
}
