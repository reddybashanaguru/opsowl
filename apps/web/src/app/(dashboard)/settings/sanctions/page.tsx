"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/hooks/useAuth";
import { ShieldAlert, RefreshCw, Loader2, AlertTriangle, CheckCircle2, Clock } from "lucide-react";

interface SanctionsList {
  id: string;
  source: string;
  loaded_at: string;
  entry_count: number;
  checksum: string;
}

interface Hit {
  id: string;
  subject_id: string;
  entry_id: string;
  match_type: string;
  match_score: number;
  matched_name: string;
  screened_at: string;
  status: string;
  notes: string;
  entry?: {
    primary_name: string;
    list_source: string;
    programs: string[];
  };
}

const LIST_META: Record<string, { label: string; jurisdiction: string; desc: string; color: string; flagColor: string }> = {
  eu_consolidated: {
    label: "EU Consolidated List",
    jurisdiction: "European Union",
    desc: "EU Financial Sanctions — Council Regulation (EC) No 881/2002",
    color: "border-blue-500/30 bg-blue-500/5",
    flagColor: "text-blue-400",
  },
  un_consolidated: {
    label: "UN Security Council List",
    jurisdiction: "United Nations",
    desc: "UNSCR 1267/1988/2253 consolidated list (ISIL, Al-Qaida, Taliban)",
    color: "border-red-500/30 bg-red-500/5",
    flagColor: "text-red-400",
  },
  ofac_sdn: {
    label: "OFAC Specially Designated Nationals",
    jurisdiction: "United States Treasury",
    desc: "OFAC SDN List — Office of Foreign Assets Control",
    color: "border-amber-500/30 bg-amber-500/5",
    flagColor: "text-amber-400",
  },
  dfat_consolidated: {
    label: "DFAT Autonomous Sanctions",
    jurisdiction: "Australia (DFAT)",
    desc: "Australian autonomous sanctions — Dept. of Foreign Affairs and Trade",
    color: "border-green-500/30 bg-green-500/5",
    flagColor: "text-green-400",
  },
};

const STATUS_CONFIG: Record<string, { label: string; color: string }> = {
  open:           { label: "Open",           color: "bg-red-500/15 text-red-300 border-red-500/30" },
  confirmed:      { label: "Confirmed",      color: "bg-red-600/20 text-red-200 border-red-600/40" },
  escalated:      { label: "Escalated",      color: "bg-orange-500/15 text-orange-300 border-orange-500/30" },
  false_positive: { label: "False Positive", color: "bg-slate-500/15 text-slate-300 border-slate-500/30" },
};

function ListCard({ list, isAdmin }: { list: SanctionsList; isAdmin: boolean }) {
  const meta = LIST_META[list.source] ?? {
    label: list.source, jurisdiction: "", desc: "", color: "border-slate-700", flagColor: "text-slate-400",
  };
  const qc = useQueryClient();
  const reloadMut = useMutation({
    mutationFn: () => api.post(`/api/v1/sanctions/lists/reload`, { source: list.source }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sanctions-lists"] }),
  });

  const loadedAt = new Date(list.loaded_at);
  const ageMs = Date.now() - loadedAt.getTime();
  const ageDays = Math.floor(ageMs / 86400000);
  const isStale = ageDays > 1;

  return (
    <div className={`p-5 rounded-xl border ${meta.color}`}>
      <div className="flex items-start justify-between gap-2">
        <div>
          <div className={`text-xs font-semibold uppercase tracking-wider mb-0.5 ${meta.flagColor}`}>
            {meta.jurisdiction}
          </div>
          <h3 className="text-sm font-semibold text-slate-100">{meta.label}</h3>
          <p className="text-xs text-slate-500 mt-0.5">{meta.desc}</p>
        </div>
        {isAdmin && (
          <button
            onClick={() => reloadMut.mutate()}
            disabled={reloadMut.isPending}
            title="Reload from source"
            className="flex-shrink-0 flex items-center gap-1.5 px-3 py-1.5 bg-slate-700 hover:bg-slate-600 disabled:opacity-50 text-slate-300 text-xs rounded transition-colors"
          >
            {reloadMut.isPending ? <Loader2 size={12} className="animate-spin" /> : <RefreshCw size={12} />}
            Reload
          </button>
        )}
      </div>
      <div className="mt-4 flex items-center gap-6">
        <div>
          <div className="text-2xl font-bold text-slate-100">{list.entry_count.toLocaleString()}</div>
          <div className="text-xs text-slate-500">entries</div>
        </div>
        <div>
          <div className={`flex items-center gap-1 text-xs ${isStale ? "text-amber-400" : "text-emerald-400"}`}>
            {isStale ? <AlertTriangle size={11} /> : <CheckCircle2 size={11} />}
            {isStale ? `${ageDays}d ago — refresh recommended` : "Up to date"}
          </div>
          <div className="text-xs text-slate-500 mt-0.5">
            Last loaded {loadedAt.toLocaleDateString("en-GB", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" })}
          </div>
        </div>
      </div>
      {reloadMut.isSuccess && (
        <p className="text-xs text-emerald-400 mt-2">Reload triggered — list will refresh in the background.</p>
      )}
    </div>
  );
}

export default function SanctionsSettingsPage() {
  const { user } = useAuth();
  const isAdmin = ["admin", "super_admin"].includes(user?.role ?? "");

  const { data: lists = [], isLoading: listsLoading } = useQuery<SanctionsList[]>({
    queryKey: ["sanctions-lists"],
    queryFn: () => api.get("/api/v1/sanctions/lists"),
  });

  const { data: allHits = [], isLoading: hitsLoading } = useQuery<Hit[]>({
    queryKey: ["sanctions-all-hits"],
    queryFn: () => api.get("/api/v1/sanctions/hits"),
  });

  const openHits = allHits.filter((h) => h.status === "open" || h.status === "escalated");
  const confirmedHits = allHits.filter((h) => h.status === "confirmed");
  const recentHits = [...confirmedHits, ...openHits].slice(0, 20);

  const knownSources = ["eu_consolidated", "un_consolidated", "ofac_sdn", "dfat_consolidated"];
  const displayLists = knownSources.map(
    (src) => lists.find((l) => l.source === src) ?? {
      id: src,
      source: src,
      loaded_at: new Date(0).toISOString(),
      entry_count: 0,
      checksum: "",
    }
  );

  return (
    <div className="p-6">
      <div className="mb-6">
        <a href="/settings" className="text-slate-400 hover:text-slate-200 text-sm transition-colors mb-1 block">
          ← Settings
        </a>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold text-slate-100">Sanctions & PEP Lists</h1>
            <p className="text-slate-500 text-xs mt-0.5">
              EU Consolidated · UN Security Council · OFAC SDN · DFAT (Australia) — fuzzy matching via pg_trgm
            </p>
          </div>
          <div className="flex items-center gap-3 text-sm">
            <div className="text-center">
              <div className="text-2xl font-bold text-red-400">{openHits.length}</div>
              <div className="text-xs text-slate-500">Open Hits</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-slate-100">{confirmedHits.length}</div>
              <div className="text-xs text-slate-500">Confirmed</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-slate-400">{allHits.length}</div>
              <div className="text-xs text-slate-500">Total Hits</div>
            </div>
          </div>
        </div>
      </div>

      {/* Sanctions Lists */}
      <div className="mb-8">
        <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-3">Screening Databases</h2>
        {listsLoading ? (
          <div className="flex items-center gap-2 text-slate-500 text-sm py-6">
            <Loader2 size={14} className="animate-spin" /> Loading list status…
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
            {displayLists.map((list) => (
              <ListCard key={list.source} list={list} isAdmin={isAdmin} />
            ))}
          </div>
        )}
      </div>

      {/* Global hits table */}
      <div>
        <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-3">
          Active Hits — All Subjects {recentHits.length > 0 && `(${recentHits.length})`}
        </h2>
        {hitsLoading ? (
          <div className="flex items-center gap-2 text-slate-500 text-sm py-6">
            <Loader2 size={14} className="animate-spin" /> Loading hits…
          </div>
        ) : recentHits.length === 0 ? (
          <div className="text-center py-10 bg-[#0e1929] border border-[#1e3050] rounded-xl">
            <ShieldAlert size={28} className="text-slate-600 mx-auto mb-3" />
            <p className="text-slate-500 text-sm">No open or confirmed sanctions hits across all subjects.</p>
          </div>
        ) : (
          <div className="bg-[#0e1929] border border-[#1e3050] rounded-xl overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700/60">
                  {["Subject", "Matched Against", "List", "Score", "Status", "Screened"].map((h) => (
                    <th key={h} className="px-4 py-2.5 text-left text-[11px] font-medium text-slate-500 uppercase tracking-wider">{h}</th>
                  ))}
                  <th className="px-4 py-2.5" />
                </tr>
              </thead>
              <tbody>
                {recentHits.map((hit) => {
                  const st = STATUS_CONFIG[hit.status] ?? STATUS_CONFIG.open;
                  const listMeta = LIST_META[hit.entry?.list_source ?? ""];
                  const pct = Math.round(hit.match_score * 100);
                  const scoreColor = pct >= 95 ? "text-red-400" : pct >= 85 ? "text-amber-400" : "text-slate-400";
                  return (
                    <tr key={hit.id} className="border-b border-slate-700/30 hover:bg-slate-800/20">
                      <td className="px-4 py-3">
                        <a
                          href={`/subjects/${hit.subject_id}/sanctions`}
                          className="text-cyan-400 hover:text-cyan-300 text-sm transition-colors"
                        >
                          {hit.subject_id.slice(0, 8)}…
                        </a>
                      </td>
                      <td className="px-4 py-3 text-sm text-slate-200 max-w-[200px] truncate">
                        {hit.entry?.primary_name ?? hit.matched_name}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`text-[11px] font-medium ${listMeta?.flagColor ?? "text-slate-400"}`}>
                          {listMeta?.label ?? hit.entry?.list_source ?? "—"}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`font-mono text-xs ${scoreColor}`}>{pct}%</span>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex items-center px-2 py-0.5 rounded-full border text-[11px] ${st.color}`}>
                          {st.label}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-xs text-slate-500">
                        {new Date(hit.screened_at).toLocaleDateString("en-GB", { day: "2-digit", month: "short" })}
                      </td>
                      <td className="px-4 py-3">
                        <a
                          href={`/subjects/${hit.subject_id}/sanctions`}
                          className="text-xs text-slate-400 hover:text-slate-200 transition-colors"
                        >
                          Review →
                        </a>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Info note */}
      <div className="mt-6 p-3 bg-slate-800/40 border border-slate-700/40 rounded-lg">
        <div className="flex items-start gap-2">
          <Clock size={13} className="text-slate-500 flex-shrink-0 mt-0.5" />
          <p className="text-xs text-slate-500">
            Lists are refreshed daily at midnight UTC. Fuzzy match threshold:{" "}
            <span className="text-slate-300 font-mono">SANCTIONS_SIMILARITY_THRESHOLD</span> env var (default 0.75).
            Adjust for false-positive rate vs. coverage trade-off.
          </p>
        </div>
      </div>
    </div>
  );
}
