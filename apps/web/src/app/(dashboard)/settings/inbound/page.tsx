"use client";

import { Fragment, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────────────────────

interface Source {
  id: string;
  name: string;
  slug: string;
  description: string;
  signing_secret?: string;
  is_active: boolean;
  created_at: string;
}

interface Rule {
  id: string;
  source_id: string;
  event_type_pattern: string;
  case_priority: "critical" | "high" | "medium" | "low";
  tags: string[];
  title_template: string;
  description_template: string;
  auto_assign_user_id: string | null;
  is_active: boolean;
  sort_order: number;
}

interface InboundEvent {
  id: string;
  source_id: string;
  source_name: string;
  idempotency_key: string;
  event_type: string;
  payload: unknown;
  status: "pending" | "processed" | "failed" | "ignored";
  case_id: string | null;
  error_message: string | null;
  attempt_count: number;
  created_at: string;
}

const PRIORITY_COLORS = {
  critical: "bg-red-500/20 text-red-400 border-red-500/30",
  high: "bg-orange-500/20 text-orange-400 border-orange-500/30",
  medium: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
  low: "bg-slate-500/20 text-slate-400 border-slate-500/30",
};

const STATUS_COLORS = {
  pending: "bg-yellow-500/20 text-yellow-400",
  processed: "bg-green-500/20 text-green-400",
  failed: "bg-red-500/20 text-red-400",
  ignored: "bg-slate-500/20 text-slate-400",
};

// ─── Source form ──────────────────────────────────────────────────────────────

function SourceForm({ onClose, onSuccess }: { onClose: () => void; onSuccess: (src: Source) => void }) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: () => api.post<Source>("/api/v1/inbound/sources", { name, slug, description }),
    onSuccess: (src) => { onSuccess(src); onClose(); },
    onError: (e) => setError(e instanceof APIError ? e.message : "Failed to create source"),
  });

  const autoSlug = (v: string) => v.toLowerCase().replace(/[^a-z0-9-]/g, "-").replace(/-+/g, "-");

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-slate-800 border border-slate-700 rounded-xl p-6 w-full max-w-md">
        <h2 className="text-slate-100 font-semibold mb-4">New Inbound Source</h2>
        {error && <p className="text-red-400 text-sm mb-3">{error}</p>}
        <div className="space-y-3">
          <div>
            <label className="block text-xs text-slate-400 mb-1">Name</label>
            <input value={name} onChange={e => { setName(e.target.value); if (!slug) setSlug(autoSlug(e.target.value)); }}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:border-cyan-500"
              placeholder="Finacle Alert Feed" />
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Slug <span className="text-slate-600">(used in headers)</span></label>
            <input value={slug} onChange={e => setSlug(autoSlug(e.target.value))}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm font-mono text-cyan-400 focus:outline-none focus:border-cyan-500"
              placeholder="finacle-alerts" />
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Description <span className="text-slate-600">(optional)</span></label>
            <input value={description} onChange={e => setDescription(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:border-cyan-500"
              placeholder="Fraud alerts from Finacle CBS" />
          </div>
        </div>
        <div className="flex gap-3 mt-5">
          <button onClick={() => mutation.mutate()} disabled={!name || !slug || mutation.isPending}
            className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm rounded transition-colors">
            {mutation.isPending ? "Creating…" : "Create Source"}
          </button>
          <button onClick={onClose} className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm transition-colors">Cancel</button>
        </div>
      </div>
    </div>
  );
}

// ─── Rule form ────────────────────────────────────────────────────────────────

function RuleForm({ sourceID, onClose }: { sourceID: string; onClose: () => void }) {
  const qc = useQueryClient();
  const [pattern, setPattern] = useState("");
  const [priority, setPriority] = useState("medium");
  const [titleTmpl, setTitleTmpl] = useState("{{event_type}} from {{source_name}}");
  const [descTmpl, setDescTmpl] = useState("Source: {{source_name}}\nEvent: {{event_type}}\n\n{{payload}}");
  const [tagsInput, setTagsInput] = useState("");
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: () => api.post<Rule>(`/api/v1/inbound/sources/${sourceID}/rules`, {
      event_type_pattern: pattern,
      case_priority: priority,
      title_template: titleTmpl,
      description_template: descTmpl,
      tags: tagsInput ? tagsInput.split(",").map(t => t.trim()).filter(Boolean) : [],
    }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["inbound-rules", sourceID] }); onClose(); },
    onError: (e) => setError(e instanceof APIError ? e.message : "Failed to create rule"),
  });

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-slate-800 border border-slate-700 rounded-xl p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <h2 className="text-slate-100 font-semibold mb-4">Add Routing Rule</h2>
        {error && <p className="text-red-400 text-sm mb-3">{error}</p>}
        <div className="space-y-3">
          <div>
            <label className="block text-xs text-slate-400 mb-1">Event Type Pattern</label>
            <input value={pattern} onChange={e => setPattern(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm font-mono text-cyan-400 focus:outline-none focus:border-cyan-500"
              placeholder="fraud_alert  or  compliance_*" />
            <p className="text-slate-600 text-[10px] mt-0.5">Exact match or wildcard suffix (e.g. fraud_*)</p>
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Case Priority</label>
            <select value={priority} onChange={e => setPriority(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:border-cyan-500">
              {["critical","high","medium","low"].map(p => <option key={p} value={p}>{p.charAt(0).toUpperCase()+p.slice(1)}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Case Title Template</label>
            <input value={titleTmpl} onChange={e => setTitleTmpl(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm font-mono text-slate-100 focus:outline-none focus:border-cyan-500" />
            <p className="text-slate-600 text-[10px] mt-0.5">Available: {"{{event_type}}"} {"{{source_name}}"} {"{{payload.field}}"}</p>
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Case Description Template</label>
            <textarea value={descTmpl} onChange={e => setDescTmpl(e.target.value)} rows={3}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm font-mono text-slate-100 focus:outline-none focus:border-cyan-500 resize-none" />
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Extra Tags <span className="text-slate-600">(comma-separated)</span></label>
            <input value={tagsInput} onChange={e => setTagsInput(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:border-cyan-500"
              placeholder="fraud, high-risk" />
          </div>
        </div>
        <div className="flex gap-3 mt-5">
          <button onClick={() => mutation.mutate()} disabled={!pattern || mutation.isPending}
            className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm rounded transition-colors">
            {mutation.isPending ? "Adding…" : "Add Rule"}
          </button>
          <button onClick={onClose} className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm transition-colors">Cancel</button>
        </div>
      </div>
    </div>
  );
}

// ─── Source card ──────────────────────────────────────────────────────────────

function SourceCard({ source, onDeleted }: { source: Source; onDeleted: () => void }) {
  const qc = useQueryClient();
  const [expanded, setExpanded] = useState(false);
  const [showSecret, setShowSecret] = useState(false);
  const [newSecret, setNewSecret] = useState<string | null>(null);
  const [showRuleForm, setShowRuleForm] = useState(false);
  const [copied, setCopied] = useState(false);

  const { data: rules = [] } = useQuery<Rule[]>({
    queryKey: ["inbound-rules", source.id],
    queryFn: () => api.get(`/api/v1/inbound/sources/${source.id}/rules`),
    enabled: expanded,
  });

  const regenerate = useMutation({
    mutationFn: () => api.post<{ signing_secret: string }>(`/api/v1/inbound/sources/${source.id}/regenerate-secret`, {}),
    onSuccess: (d) => setNewSecret(d.signing_secret),
  });

  const toggleActive = useMutation({
    mutationFn: () => api.put<Source>(`/api/v1/inbound/sources/${source.id}`, { is_active: !source.is_active }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["inbound-sources"] }),
  });

  const deleteSource = useMutation({
    mutationFn: () => api.delete(`/api/v1/inbound/sources/${source.id}`),
    onSuccess: onDeleted,
  });

  const deleteRule = useMutation({
    mutationFn: (ruleID: string) => api.delete(`/api/v1/inbound/rules/${ruleID}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["inbound-rules", source.id] }),
  });

  const displaySecret = newSecret ?? "••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••";

  const copySecret = (secret: string) => {
    navigator.clipboard.writeText(secret);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="bg-slate-800/50 border border-slate-700 rounded-xl overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between p-4">
        <div className="flex items-center gap-3">
          <button onClick={() => setExpanded(v => !v)} className="text-slate-400 hover:text-slate-200 text-xs">
            {expanded ? "▾" : "▸"}
          </button>
          <div>
            <div className="flex items-center gap-2">
              <span className="text-slate-100 font-medium text-sm">{source.name}</span>
              <span className="font-mono text-xs text-slate-500">{source.slug}</span>
            </div>
            {source.description && <p className="text-slate-500 text-xs mt-0.5">{source.description}</p>}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span className={`px-2 py-0.5 rounded-full text-xs ${source.is_active ? "bg-green-500/20 text-green-400" : "bg-slate-600/40 text-slate-400"}`}>
            {source.is_active ? "Active" : "Disabled"}
          </span>
          <button onClick={() => toggleActive.mutate()}
            className="px-3 py-1 text-xs text-slate-400 hover:text-slate-200 border border-slate-600 rounded transition-colors">
            {source.is_active ? "Disable" : "Enable"}
          </button>
          <button onClick={() => deleteSource.mutate()}
            className="px-3 py-1 text-xs text-red-400 hover:text-red-300 border border-red-500/30 rounded transition-colors">
            Delete
          </button>
        </div>
      </div>

      {/* Expanded panel */}
      {expanded && (
        <div className="border-t border-slate-700 p-4 space-y-4">

          {/* Signing secret section */}
          <div className="p-3 bg-slate-900/60 rounded-lg">
            <div className="flex items-center justify-between mb-2">
              <p className="text-xs text-slate-400 font-medium">Signing Secret <span className="text-slate-600">(HMAC-SHA256)</span></p>
              <div className="flex gap-2">
                <button onClick={() => setShowSecret(v => !v)} className="text-xs text-slate-400 hover:text-slate-200">
                  {showSecret ? "Hide" : "Show"}
                </button>
                {(newSecret || showSecret) && (
                  <button onClick={() => copySecret(newSecret ?? displaySecret)} className="text-xs text-cyan-400 hover:text-cyan-300">
                    {copied ? "Copied!" : "Copy"}
                  </button>
                )}
                <button onClick={() => regenerate.mutate()} disabled={regenerate.isPending}
                  className="text-xs text-orange-400 hover:text-orange-300 disabled:opacity-50">
                  {regenerate.isPending ? "Regenerating…" : "Regenerate"}
                </button>
              </div>
            </div>
            <div className="font-mono text-xs text-slate-300 bg-slate-800 rounded px-3 py-2 break-all">
              {(showSecret || newSecret) ? (newSecret ?? "reveal by clicking Show") : displaySecret}
            </div>
            {newSecret && (
              <p className="text-amber-400 text-xs mt-2">
                ⚠ Copy this secret now — it won&apos;t be shown again. The old secret is immediately invalidated.
              </p>
            )}
            <div className="mt-3 p-3 bg-slate-800 rounded text-xs text-slate-400 space-y-1">
              <p className="font-medium text-slate-300">How to call this endpoint:</p>
              <p className="font-mono text-slate-500">POST /api/v1/inbound/events</p>
              <p className="font-mono text-slate-500">X-OpsOwl-Source: {source.slug}</p>
              <p className="font-mono text-slate-500">X-OpsOwl-Signature: sha256=&lt;HMAC-SHA256(secret, body)&gt;</p>
              <p className="font-mono text-slate-500">X-OpsOwl-Idempotency-Key: &lt;unique-id&gt;</p>
            </div>
          </div>

          {/* Rules section */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <p className="text-xs text-slate-400 font-medium uppercase tracking-widest">Routing Rules</p>
              <button onClick={() => setShowRuleForm(true)}
                className="px-3 py-1 text-xs bg-cyan-600/20 text-cyan-400 border border-cyan-500/30 rounded hover:bg-cyan-600/30 transition-colors">
                + Add Rule
              </button>
            </div>

            {rules.length === 0 ? (
              <div className="text-center py-6 text-slate-500 text-xs border border-slate-700 rounded-lg">
                No rules yet. Add a rule to start creating cases from inbound events.
              </div>
            ) : (
              <div className="space-y-2">
                {rules.map(rule => (
                  <div key={rule.id} className="flex items-start justify-between p-3 bg-slate-900/40 rounded-lg border border-slate-700/50">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-sm text-cyan-400">{rule.event_type_pattern}</span>
                        <span className={`px-2 py-0.5 rounded-full text-xs border ${PRIORITY_COLORS[rule.case_priority]}`}>
                          {rule.case_priority}
                        </span>
                        {!rule.is_active && <span className="text-xs text-slate-500 italic">disabled</span>}
                      </div>
                      <p className="text-xs text-slate-400">{rule.title_template}</p>
                      {rule.tags.length > 0 && (
                        <div className="flex gap-1 flex-wrap">
                          {rule.tags.map(t => (
                            <span key={t} className="px-1.5 py-0.5 bg-slate-700 rounded text-[10px] text-slate-400">{t}</span>
                          ))}
                        </div>
                      )}
                    </div>
                    <button onClick={() => deleteRule.mutate(rule.id)}
                      className="text-xs text-slate-500 hover:text-red-400 transition-colors ml-3 shrink-0">
                      ✕
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {showRuleForm && <RuleForm sourceID={source.id} onClose={() => setShowRuleForm(false)} />}
    </div>
  );
}

// ─── Event log ────────────────────────────────────────────────────────────────

interface EventsPage {
  events: InboundEvent[];
  has_more: boolean;
  next_cursor: string | null;
}

function EventLog({ sources }: { sources: Source[] }) {
  const [sourceFilter, setSourceFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [typeFilter, setTypeFilter] = useState("");
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const [pages, setPages] = useState<InboundEvent[][]>([]);
  const [expandedID, setExpandedID] = useState<string | null>(null);
  const qc = useQueryClient();

  const buildURL = (cur?: string) => {
    const p = new URLSearchParams({ limit: "30" });
    if (sourceFilter) p.set("source_id", sourceFilter);
    if (statusFilter) p.set("status", statusFilter);
    if (cur) p.set("cursor", cur);
    return `/api/v1/inbound/events?${p}`;
  };

  const { data: page, isLoading, isFetching } = useQuery<EventsPage>({
    queryKey: ["inbound-events", sourceFilter, statusFilter, cursor],
    queryFn: () => api.get(buildURL(cursor)),
    refetchInterval: 15_000,
    placeholderData: (prev) => prev,
  });

  const allEvents = pages.flatMap(p => p).concat(page?.events ?? []);
  const visible = typeFilter
    ? allEvents.filter(e => e.event_type.includes(typeFilter))
    : allEvents;

  function loadMore() {
    if (!page?.next_cursor) return;
    setPages(prev => [...prev, page.events]);
    setCursor(page.next_cursor!);
  }

  function refresh() {
    setPages([]);
    setCursor(undefined);
    qc.invalidateQueries({ queryKey: ["inbound-events"] });
  }

  return (
    <div className="space-y-3">
      {/* Toolbar */}
      <div className="flex flex-wrap gap-2 items-center justify-between">
        <div className="flex gap-2 flex-wrap">
          <select
            value={sourceFilter}
            onChange={e => { setSourceFilter(e.target.value); setPages([]); setCursor(undefined); }}
            className="bg-slate-800 border border-slate-700 text-slate-300 text-xs rounded-lg px-3 py-1.5 focus:outline-none focus:border-cyan-500"
          >
            <option value="">All sources</option>
            {sources.map(s => (
              <option key={s.id} value={s.id}>{s.name}</option>
            ))}
          </select>
          <select
            value={statusFilter}
            onChange={e => { setStatusFilter(e.target.value); setPages([]); setCursor(undefined); }}
            className="bg-slate-800 border border-slate-700 text-slate-300 text-xs rounded-lg px-3 py-1.5 focus:outline-none focus:border-cyan-500"
          >
            <option value="">All statuses</option>
            <option value="pending">Pending</option>
            <option value="processed">Processed</option>
            <option value="failed">Failed</option>
            <option value="ignored">Ignored</option>
          </select>
          <input
            type="text"
            placeholder="Filter by event type…"
            value={typeFilter}
            onChange={e => setTypeFilter(e.target.value)}
            className="bg-slate-800 border border-slate-700 text-slate-300 text-xs rounded-lg px-3 py-1.5 w-48 focus:outline-none focus:border-cyan-500 placeholder-slate-600"
          />
        </div>
        <button
          onClick={refresh}
          disabled={isFetching}
          className="text-xs px-3 py-1.5 bg-slate-800 border border-slate-700 text-slate-300 rounded-lg hover:border-cyan-500/50 transition-colors disabled:opacity-50"
        >
          {isFetching ? "Refreshing…" : "↻ Refresh"}
        </button>
      </div>

      {/* Table */}
      {isLoading && pages.length === 0 ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-12 bg-slate-800/50 rounded-lg animate-pulse" />
          ))}
        </div>
      ) : visible.length === 0 ? (
        <div className="text-center py-10 text-slate-500 text-sm border border-slate-700 rounded-xl">
          No events match the current filters.
        </div>
      ) : (
        <div className="border border-slate-700 rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-700 bg-slate-800/30">
                <th className="text-left px-4 py-2 text-xs text-slate-400">Time</th>
                <th className="text-left px-4 py-2 text-xs text-slate-400">Source</th>
                <th className="text-left px-4 py-2 text-xs text-slate-400">Event Type</th>
                <th className="text-left px-4 py-2 text-xs text-slate-400">Status</th>
                <th className="text-left px-4 py-2 text-xs text-slate-400">Attempts</th>
                <th className="text-left px-4 py-2 text-xs text-slate-400">Outcome</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-700/50">
              {visible.map(evt => (
                <Fragment key={evt.id}>
                  <tr
                    onClick={() => setExpandedID(expandedID === evt.id ? null : evt.id)}
                    className={`cursor-pointer transition-colors ${
                      evt.status === "failed" ? "hover:bg-red-950/20" : "hover:bg-slate-800/30"
                    } ${expandedID === evt.id ? "bg-slate-800/40" : ""}`}
                  >
                    <td className="px-4 py-2.5 text-xs text-slate-500 whitespace-nowrap">
                      <div>{new Date(evt.created_at).toLocaleDateString()}</div>
                      <div className="text-slate-600">{new Date(evt.created_at).toLocaleTimeString()}</div>
                    </td>
                    <td className="px-4 py-2.5 text-xs font-mono text-slate-400">{evt.source_name}</td>
                    <td className="px-4 py-2.5 text-xs font-mono text-cyan-400">{evt.event_type}</td>
                    <td className="px-4 py-2.5">
                      <span className={`px-2 py-0.5 rounded-full text-xs ${STATUS_COLORS[evt.status]}`}>
                        {evt.status}
                      </span>
                    </td>
                    <td className="px-4 py-2.5 text-xs text-slate-500 text-center">{evt.attempt_count}</td>
                    <td className="px-4 py-2.5 text-xs">
                      {evt.case_id ? (
                        <a
                          href={`/cases/${evt.case_id}`}
                          onClick={e => e.stopPropagation()}
                          className="text-cyan-400 hover:underline"
                        >
                          View Case →
                        </a>
                      ) : evt.error_message ? (
                        <span className="text-red-400 truncate max-w-[200px] block" title={evt.error_message}>
                          {evt.error_message}
                        </span>
                      ) : (
                        <span className="text-slate-600">—</span>
                      )}
                    </td>
                  </tr>
                  {expandedID === evt.id && (
                    <tr>
                      <td colSpan={6} className="px-4 pb-3 bg-slate-900/60">
                        <div className="text-xs text-slate-400 mb-1 font-medium pt-2">Payload</div>
                        <pre className="text-xs font-mono text-slate-300 bg-slate-900 border border-slate-700 rounded-lg p-3 overflow-x-auto max-h-48">
                          {JSON.stringify(evt.payload, null, 2)}
                        </pre>
                        <div className="text-xs text-slate-600 mt-2">
                          ID: <span className="font-mono">{evt.id}</span>
                          {" · "}
                          Idempotency key: <span className="font-mono">{evt.idempotency_key}</span>
                        </div>
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Load more */}
      {page?.has_more && (
        <div className="flex justify-center pt-1">
          <button
            onClick={loadMore}
            disabled={isFetching}
            className="text-xs px-4 py-2 bg-slate-800 border border-slate-700 text-slate-300 rounded-lg hover:border-cyan-500/50 transition-colors disabled:opacity-50"
          >
            {isFetching ? "Loading…" : "Load more events"}
          </button>
        </div>
      )}

      {/* Live indicator */}
      <div className="flex items-center gap-1.5 text-xs text-slate-600">
        <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" />
        Auto-refreshing every 15 s
      </div>
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────

export default function InboundPage() {
  const qc = useQueryClient();
  const [showSourceForm, setShowSourceForm] = useState(false);
  const [activeTab, setActiveTab] = useState<"sources" | "events">("sources");

  const { data: sources = [], isLoading } = useQuery<Source[]>({
    queryKey: ["inbound-sources"],
    queryFn: () => api.get("/api/v1/inbound/sources"),
  });

  return (
    <div className="p-8 max-w-4xl">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-100 mb-1">Inbound Integrations</h1>
        <p className="text-slate-400 text-sm">
          Register external systems that can push events into OpsOwl and automatically create cases.
          Each source gets a unique signing secret for request authentication.
        </p>
      </div>

      {/* How it works — plain-language explainer */}
      <div className="mb-6 p-4 bg-slate-800/30 border border-slate-700 rounded-xl text-xs text-slate-400 space-y-1">
        <p className="font-medium text-slate-300">How it works</p>
        <p>1. Register a source (e.g. "Finacle Alerts") — OpsOwl generates a secret key.</p>
        <p>2. Add routing rules — define which event types create cases and at what priority.</p>
        <p>3. Configure your source system to POST to <span className="font-mono text-cyan-400">/api/v1/inbound/events</span> with the signed headers.</p>
        <p>4. OpsOwl verifies the signature, applies the first matching rule, and creates the case automatically.</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 border-b border-slate-700">
        {(["sources", "events"] as const).map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm capitalize transition-colors ${
              activeTab === tab
                ? "text-cyan-400 border-b-2 border-cyan-400"
                : "text-slate-400 hover:text-slate-200"
            }`}>
            {tab === "sources" ? `Sources (${sources.length})` : "Event Log"}
          </button>
        ))}
      </div>

      {activeTab === "sources" && (
        <div>
          <div className="flex justify-end mb-4">
            <button onClick={() => setShowSourceForm(true)}
              className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm rounded transition-colors">
              + New Source
            </button>
          </div>

          {isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="h-16 bg-slate-800/50 rounded-xl animate-pulse" />
              ))}
            </div>
          ) : sources.length === 0 ? (
            <div className="text-center py-16 border border-slate-700 rounded-xl">
              <p className="text-slate-400 text-sm mb-1">No inbound sources yet</p>
              <p className="text-slate-600 text-xs">Create your first source to start receiving events from external systems.</p>
            </div>
          ) : (
            <div className="space-y-3">
              {sources.map(src => (
                <SourceCard
                  key={src.id}
                  source={src}
                  onDeleted={() => qc.invalidateQueries({ queryKey: ["inbound-sources"] })}
                />
              ))}
            </div>
          )}
        </div>
      )}

      {activeTab === "events" && <EventLog sources={sources} />}

      {showSourceForm && (
        <SourceForm
          onClose={() => setShowSourceForm(false)}
          onSuccess={() => qc.invalidateQueries({ queryKey: ["inbound-sources"] })}
        />
      )}
    </div>
  );
}
