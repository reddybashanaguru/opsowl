"use client";

import { Fragment, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { RBACGate } from "@/components/auth/RBACGate";

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
  const [touched, setTouched] = useState({ name: false, slug: false });
  const [error, setError] = useState<string | null>(null);
  const [created, setCreated] = useState<(Source & { signing_secret?: string }) | null>(null);
  const [secretCopied, setSecretCopied] = useState(false);

  const API_BASE = typeof window !== "undefined" ? window.location.origin.replace(":3001", ":8080").replace(":3000", ":8080") : "https://your-opsowl-domain.com";
  const endpoint = `${API_BASE}/api/v1/inbound/events`;

  const SLUG_RE = /^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$/;
  const nameErr = touched.name && name.trim().length < 2 ? "Name must be at least 2 characters" : null;
  const slugErr = touched.slug && slug && !SLUG_RE.test(slug) ? "Use lowercase letters, numbers, and hyphens only (e.g. finacle-alerts)" : null;
  const isValid = name.trim().length >= 2 && slug.length >= 2 && SLUG_RE.test(slug) && !nameErr && !slugErr;

  const mutation = useMutation({
    mutationFn: () => api.post<Source & { signing_secret?: string }>("/api/v1/inbound/sources", { name: name.trim(), slug, description: description.trim() }),
    onSuccess: (src) => { setCreated(src); onSuccess(src); },
    onError: (e) => setError(e instanceof APIError ? e.message : "Failed to create source"),
  });

  const autoSlug = (v: string) => v.toLowerCase().replace(/[^a-z0-9-]/g, "-").replace(/-+/g, "-").replace(/^-+|-+$/g, "");
  const displaySlug = slug || "your-source-slug";

  const copySecret = (secret: string) => {
    navigator.clipboard.writeText(secret);
    setSecretCopied(true);
    setTimeout(() => setSecretCopied(false), 2000);
  };

  if (created) {
    return (
      <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
        <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-6 w-full max-w-lg">
          <div className="flex items-center gap-2 mb-1">
            <span className="w-5 h-5 rounded-full bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-[10px] text-emerald-400">✓</span>
            <h2 className="text-slate-100 font-semibold">Source Created</h2>
          </div>
          <p className="text-xs text-slate-500 mb-4">Save the credentials below — the signing secret is shown only once.</p>

          <div className="space-y-3">
            <div className="bg-slate-900 border border-slate-700 rounded-lg p-3 space-y-2">
              <p className="text-[10px] font-semibold text-slate-500 uppercase tracking-widest">Endpoint URL</p>
              <div className="flex items-center justify-between gap-2">
                <code className="text-xs font-mono text-cyan-400 break-all">{endpoint}</code>
                <button onClick={() => navigator.clipboard.writeText(endpoint)}
                  className="text-[10px] px-2 py-0.5 rounded bg-slate-700 text-slate-300 hover:bg-slate-600 flex-shrink-0">Copy</button>
              </div>
            </div>

            <div className="bg-slate-900 border border-slate-700 rounded-lg p-3 space-y-2">
              <p className="text-[10px] font-semibold text-slate-500 uppercase tracking-widest">Required Headers</p>
              <div className="text-xs font-mono space-y-1 text-slate-400">
                <div><span className="text-slate-600">X-OpsOwl-Source:</span> <span className="text-cyan-400">{created.slug}</span></div>
                <div><span className="text-slate-600">X-OpsOwl-Signature:</span> <span className="text-slate-400">sha256=HMAC_SHA256(secret, body)</span></div>
                <div><span className="text-slate-600">X-OpsOwl-Idempotency-Key:</span> <span className="text-slate-400">&lt;unique-uuid-per-event&gt;</span></div>
              </div>
            </div>

            {created.signing_secret ? (
              <div className="bg-amber-500/10 border border-amber-500/30 rounded-lg p-3 space-y-2">
                <div className="flex items-center justify-between">
                  <p className="text-[10px] font-semibold text-amber-400 uppercase tracking-widest">Signing Secret — copy now, shown only once</p>
                </div>
                <div className="flex items-center gap-2">
                  <code className="flex-1 text-xs font-mono text-amber-300 break-all bg-slate-900/60 px-2 py-1.5 rounded">{created.signing_secret}</code>
                  <button onClick={() => copySecret(created.signing_secret!)}
                    className="text-[10px] px-2 py-0.5 rounded bg-amber-500/20 text-amber-300 hover:bg-amber-500/30 flex-shrink-0">
                    {secretCopied ? "✓ Copied" : "Copy"}
                  </button>
                </div>
                <p className="text-[10px] text-amber-400/70">Store this in your source system&apos;s environment variables. If you lose it, regenerate from the source card.</p>
              </div>
            ) : (
              <div className="bg-slate-800/50 border border-slate-700 rounded-lg p-3">
                <p className="text-xs text-slate-400">Retrieve your signing secret by expanding the source card and clicking <strong className="text-slate-200">Show Secret</strong>.</p>
              </div>
            )}
          </div>

          <div className="flex gap-3 mt-5">
            <button onClick={onClose} className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm rounded transition-colors">
              Done — Go to Setup Guide
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-6 w-full max-w-lg">
        <h2 className="text-slate-100 font-semibold mb-1">New Webhook Source</h2>
        <p className="text-xs text-slate-500 mb-4">Register the external system that will send events to OpsOwl.</p>
        {error && <p className="text-red-400 text-sm mb-3">{error}</p>}
        <div className="space-y-3">
          <div>
            <label className="block text-xs text-slate-400 mb-1">Name <span className="text-red-400">*</span></label>
            <input
              value={name}
              onChange={e => { setName(e.target.value); if (!touched.slug) setSlug(autoSlug(e.target.value)); }}
              onBlur={() => setTouched(t => ({ ...t, name: true }))}
              className={`w-full bg-slate-900 border rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none ${nameErr ? "border-red-500 focus:border-red-400" : "border-slate-600 focus:border-cyan-500"}`}
              placeholder="Finacle Alert Feed"
              maxLength={100}
            />
            {nameErr && <p className="text-red-400 text-xs mt-1">{nameErr}</p>}
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Slug <span className="text-red-400">*</span> <span className="text-slate-600">(sent in X-OpsOwl-Source header)</span></label>
            <input
              value={slug}
              onChange={e => { setSlug(autoSlug(e.target.value)); setTouched(t => ({ ...t, slug: true })); }}
              onBlur={() => setTouched(t => ({ ...t, slug: true }))}
              className={`w-full bg-slate-900 border rounded px-3 py-1.5 text-sm font-mono text-cyan-400 focus:outline-none ${slugErr ? "border-red-500 focus:border-red-400" : "border-slate-600 focus:border-cyan-500"}`}
              placeholder="finacle-alerts"
              maxLength={60}
            />
            {slugErr
              ? <p className="text-red-400 text-xs mt-1">{slugErr}</p>
              : <p className="text-slate-600 text-xs mt-1">Auto-generated from name. Lowercase, numbers, hyphens only.</p>
            }
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Description <span className="text-slate-600">(optional)</span></label>
            <input
              value={description}
              onChange={e => setDescription(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:border-cyan-500"
              placeholder="Fraud alerts from Finacle CBS"
              maxLength={300}
            />
          </div>

          {/* Live endpoint preview — updates as user types slug */}
          <div className={`border rounded-lg p-3 space-y-2 transition-colors ${slug && !slugErr ? "bg-slate-900/60 border-slate-700" : "bg-slate-900/30 border-slate-800"}`}>
            <p className="text-[10px] font-semibold text-slate-500 uppercase tracking-widest">Your system will POST events to this endpoint</p>
            <div className="text-xs font-mono space-y-1.5">
              <div>
                <span className="text-violet-400">POST </span>
                <span className="text-cyan-400">/api/v1/inbound/events</span>
              </div>
              <div className="pl-3 border-l border-slate-700 space-y-1 text-[11px]">
                <div><span className="text-slate-600">X-OpsOwl-Source: </span><span className={slug && !slugErr ? "text-cyan-300" : "text-slate-700"}>{displaySlug}</span></div>
                <div><span className="text-slate-600">X-OpsOwl-Signature: </span><span className="text-slate-600">sha256=HMAC_SHA256(signing_secret, body)</span></div>
                <div><span className="text-slate-600">X-OpsOwl-Idempotency-Key: </span><span className="text-slate-600">&lt;unique-uuid-per-event&gt;</span></div>
              </div>
            </div>
            <p className="text-[10px] text-slate-600">OpsOwl generates a signing secret when this source is created. The full endpoint URL and secret are shown immediately after creation.</p>
          </div>
        </div>
        <div className="flex gap-3 mt-5">
          <button
            onClick={() => { setTouched({ name: true, slug: true }); if (isValid) mutation.mutate(); }}
            disabled={mutation.isPending}
            className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm rounded transition-colors"
          >
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
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
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
    <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] overflow-hidden">
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

// ─── Setup Guide ──────────────────────────────────────────────────────────────

function CopyButton({ value, label = "Copy" }: { value: string; label?: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      onClick={() => { navigator.clipboard.writeText(value); setCopied(true); setTimeout(() => setCopied(false), 2000); }}
      className="text-[10px] font-semibold px-2 py-0.5 rounded bg-slate-700 text-slate-300 hover:bg-slate-600 transition-colors"
    >
      {copied ? "✓ Copied" : label}
    </button>
  );
}

function CodeBlock({ code, lang = "bash" }: { code: string; lang?: string }) {
  return (
    <div className="relative group">
      <div className="absolute top-2 right-2 flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
        <span className="text-[9px] text-slate-600 uppercase tracking-widest">{lang}</span>
        <CopyButton value={code} />
      </div>
      <pre className="text-xs font-mono text-slate-300 bg-slate-900 border border-slate-700 rounded-lg p-4 overflow-x-auto leading-relaxed">
        {code}
      </pre>
    </div>
  );
}

function SetupGuide({ sources }: { sources: Source[] }) {
  const [selectedSource, setSelectedSource] = useState<Source | null>(sources[0] ?? null);
  const API_BASE = typeof window !== "undefined" ? window.location.origin : "https://your-opsowl-domain.com";
  const endpoint = `${API_BASE}/api/v1/inbound/events`;
  const slug = selectedSource?.slug ?? "your-source-slug";

  const curlExample = `curl -X POST "${endpoint}" \\
  -H "Content-Type: application/json" \\
  -H "X-OpsOwl-Source: ${slug}" \\
  -H "X-OpsOwl-Signature: sha256=$(echo -n '${JSON.stringify({ event_type: "fraud_alert", payload: { account_id: "ACC-001", amount: 49500, currency: "INR" } })}' | openssl dgst -sha256 -hmac "YOUR_SIGNING_SECRET" | awk '{print $2}')" \\
  -H "X-OpsOwl-Idempotency-Key: evt-$(date +%s)-001" \\
  -d '{"event_type":"fraud_alert","payload":{"account_id":"ACC-001","amount":49500,"currency":"INR","risk_score":87}}'`;

  const pythonExample = `import hmac, hashlib, json, uuid, requests

SIGNING_SECRET = "your_signing_secret_here"
ENDPOINT = "${endpoint}"
SOURCE_SLUG = "${slug}"

def send_event(event_type: str, payload: dict) -> dict:
    body = json.dumps({"event_type": event_type, "payload": payload})
    signature = hmac.new(
        SIGNING_SECRET.encode(),
        body.encode(),
        hashlib.sha256
    ).hexdigest()

    response = requests.post(
        ENDPOINT,
        data=body,
        headers={
            "Content-Type": "application/json",
            "X-OpsOwl-Source": SOURCE_SLUG,
            "X-OpsOwl-Signature": f"sha256={signature}",
            "X-OpsOwl-Idempotency-Key": str(uuid.uuid4()),
        }
    )
    return response.json()  # {"status": "accepted", "event_id": "..."}

# Example: fraud alert event
send_event("fraud_alert", {
    "account_id": "ACC-001",
    "customer_name": "Rahul Verma",
    "amount": 49500,
    "currency": "INR",
    "risk_score": 87,
    "alert_type": "STRUCTURING"
})`;

  const nodeExample = `const crypto = require("crypto");
const fetch = require("node-fetch"); // or built-in fetch (Node 18+)

const SIGNING_SECRET = "your_signing_secret_here";
const ENDPOINT = "${endpoint}";
const SOURCE_SLUG = "${slug}";

async function sendEvent(eventType, payload) {
  const body = JSON.stringify({ event_type: eventType, payload });
  const signature = crypto
    .createHmac("sha256", SIGNING_SECRET)
    .update(body)
    .digest("hex");

  const res = await fetch(ENDPOINT, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-OpsOwl-Source": SOURCE_SLUG,
      "X-OpsOwl-Signature": \`sha256=\${signature}\`,
      "X-OpsOwl-Idempotency-Key": crypto.randomUUID(),
    },
    body,
  });
  return res.json(); // { status: "accepted", event_id: "..." }
}

// Example
sendEvent("fraud_alert", {
  account_id: "ACC-001",
  customer_name: "Rahul Verma",
  amount: 49500,
  risk_score: 87,
});`;

  const payloadSchema = `{
  "event_type": "string",       // required — must match a routing rule pattern
  "payload": {                  // required — any JSON object; all fields available
    "account_id": "string",     //   in title/description templates as {{payload.account_id}}
    "customer_name": "string",
    "amount": 49500,
    "currency": "INR",
    "risk_score": 87,
    // ... any fields your system sends
  }
}`;

  const steps = [
    {
      num: "01",
      title: "Select the source and get your credentials",
      color: "cyan",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-slate-400">
            Each source has its own unique slug and signing secret. The slug goes in the
            <code className="mx-1 px-1.5 py-0.5 bg-slate-900 rounded text-cyan-400 text-xs font-mono">X-OpsOwl-Source</code>
            header so OpsOwl knows which source is sending the event.
          </p>
          {sources.length > 0 ? (
            <div className="p-3 bg-slate-900/60 rounded-lg border border-slate-700 space-y-2">
              <label className="text-xs text-slate-400">Configuring for source:</label>
              <select
                value={selectedSource?.id ?? ""}
                onChange={e => setSelectedSource(sources.find(s => s.id === e.target.value) ?? null)}
                className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:border-cyan-500"
              >
                {sources.map(s => <option key={s.id} value={s.id}>{s.name} ({s.slug})</option>)}
              </select>
              {selectedSource && (
                <div className="mt-2 space-y-2">
                  <div className="flex items-center justify-between bg-slate-800 rounded px-3 py-2">
                    <span className="text-xs text-slate-400">Source slug</span>
                    <div className="flex items-center gap-2">
                      <code className="text-xs font-mono text-cyan-400">{selectedSource.slug}</code>
                      <CopyButton value={selectedSource.slug} />
                    </div>
                  </div>
                  <div className="flex items-center justify-between bg-slate-800 rounded px-3 py-2">
                    <span className="text-xs text-slate-400">Endpoint URL</span>
                    <div className="flex items-center gap-2">
                      <code className="text-xs font-mono text-slate-300 truncate max-w-[320px]">{endpoint}</code>
                      <CopyButton value={endpoint} />
                    </div>
                  </div>
                  <div className="bg-amber-500/10 border border-amber-500/25 rounded px-3 py-2 text-xs text-amber-400">
                    ⚠ Retrieve your signing secret from the source card in the Sources tab (click ▸ to expand, then Show). It is only shown once at creation — if you lost it, click Regenerate.
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="p-3 bg-slate-900/60 rounded-lg border border-slate-700 text-sm text-slate-500">
              No sources created yet. Go to the Sources tab and create one first.
            </div>
          )}
        </div>
      ),
    },
    {
      num: "02",
      title: "Sign every request with HMAC-SHA256",
      color: "violet",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-slate-400">
            OpsOwl rejects any request with a missing or invalid signature. The signature is computed over the raw request body using your source's signing secret as the HMAC key.
          </p>
          <div className="flex gap-2 text-xs font-mono bg-slate-900 border border-slate-700 rounded-lg px-4 py-3 text-slate-400">
            <span>X-OpsOwl-Signature:</span>
            <span className="text-cyan-400">sha256=HMAC_SHA256(signing_secret, raw_request_body)</span>
          </div>
          <p className="text-xs text-slate-500">
            The body must be the raw JSON bytes exactly as sent — do not pretty-print or re-serialize after signing.
          </p>
          <div className="space-y-2">
            <p className="text-xs font-semibold text-slate-300 uppercase tracking-widest">Python</p>
            <CodeBlock code={pythonExample} lang="python" />
          </div>
          <div className="space-y-2">
            <p className="text-xs font-semibold text-slate-300 uppercase tracking-widest">Node.js</p>
            <CodeBlock code={nodeExample} lang="javascript" />
          </div>
        </div>
      ),
    },
    {
      num: "03",
      title: "Send your first event — test with curl",
      color: "emerald",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-slate-400">
            The event body requires two fields: <code className="px-1 py-0.5 bg-slate-900 rounded text-cyan-400 text-xs font-mono">event_type</code> (must match one of your routing rules) and <code className="px-1 py-0.5 bg-slate-900 rounded text-cyan-400 text-xs font-mono">payload</code> (any JSON — all fields are available in case templates).
          </p>
          <div className="space-y-2">
            <p className="text-xs font-semibold text-slate-300 uppercase tracking-widest">Payload schema</p>
            <CodeBlock code={payloadSchema} lang="json" />
          </div>
          <div className="space-y-2">
            <p className="text-xs font-semibold text-slate-300 uppercase tracking-widest">curl (replace YOUR_SIGNING_SECRET)</p>
            <CodeBlock code={curlExample} lang="bash" />
          </div>
          <div className="p-3 bg-emerald-500/10 border border-emerald-500/25 rounded-lg">
            <p className="text-xs font-semibold text-emerald-400 mb-1">Expected response</p>
            <CodeBlock code={`HTTP/1.1 202 Accepted\n\n{ "status": "accepted", "event_id": "evt_01j..." }`} lang="http" />
            <p className="text-xs text-slate-500 mt-2">
              202 means OpsOwl received and queued the event. Case creation happens asynchronously — check the Event Log tab in a few seconds to see the result.
            </p>
          </div>
          <div className="p-3 bg-red-500/10 border border-red-500/25 rounded-lg">
            <p className="text-xs font-semibold text-red-400 mb-1">Common errors</p>
            <ul className="text-xs text-slate-400 space-y-1">
              <li><code className="font-mono text-red-400 mr-1">401</code>— Missing or invalid HMAC signature. Check you are signing the exact raw body bytes.</li>
              <li><code className="font-mono text-red-400 mr-1">400</code>— Unknown source slug in X-OpsOwl-Source header. Check the slug matches exactly.</li>
              <li><code className="font-mono text-amber-400 mr-1">200 (ignored)</code>— Event received but no routing rule matched the event_type. Add a rule that covers this event type.</li>
            </ul>
          </div>
        </div>
      ),
    },
    {
      num: "04",
      title: "Add routing rules to control case creation",
      color: "amber",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-slate-400">
            Routing rules decide which event types create cases and what those cases look like. Rules are evaluated in sort order — the <strong className="text-slate-200">first matching rule wins</strong>.
          </p>
          <div className="bg-slate-900/60 border border-slate-700 rounded-lg overflow-hidden">
            <table className="w-full text-xs">
              <thead>
                <tr className="border-b border-slate-700 bg-slate-800/40">
                  <th className="text-left px-4 py-2 text-slate-400">Field</th>
                  <th className="text-left px-4 py-2 text-slate-400">What it does</th>
                  <th className="text-left px-4 py-2 text-slate-400">Example</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700/40">
                <tr>
                  <td className="px-4 py-2 font-mono text-cyan-400">event_type_pattern</td>
                  <td className="px-4 py-2 text-slate-400">Exact or wildcard match on the incoming event_type</td>
                  <td className="px-4 py-2 font-mono text-slate-500">fraud_* or kyc_document_forged</td>
                </tr>
                <tr>
                  <td className="px-4 py-2 font-mono text-cyan-400">case_priority</td>
                  <td className="px-4 py-2 text-slate-400">Case priority when created</td>
                  <td className="px-4 py-2 font-mono text-slate-500">critical / high / medium / low</td>
                </tr>
                <tr>
                  <td className="px-4 py-2 font-mono text-cyan-400">title_template</td>
                  <td className="px-4 py-2 text-slate-400">Case title — use payload fields with {`{{payload.field}}`}</td>
                  <td className="px-4 py-2 font-mono text-slate-500">{`{{event_type}}: {{payload.customer_name}}`}</td>
                </tr>
                <tr>
                  <td className="px-4 py-2 font-mono text-cyan-400">description_template</td>
                  <td className="px-4 py-2 text-slate-400">Case description pre-filled from payload fields</td>
                  <td className="px-4 py-2 font-mono text-slate-500">{`Account: {{payload.account_id}}\nAmount: {{payload.amount}}`}</td>
                </tr>
                <tr>
                  <td className="px-4 py-2 font-mono text-cyan-400">tags</td>
                  <td className="px-4 py-2 text-slate-400">Tags applied to the case automatically</td>
                  <td className="px-4 py-2 font-mono text-slate-500">fraud, high-risk, auto-created</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div className="p-3 bg-amber-500/10 border border-amber-500/25 rounded-lg text-xs text-amber-400">
            <p className="font-semibold mb-1">Routing rules vs Alert Source triage rules — what&apos;s the difference?</p>
            <ul className="text-amber-400/80 space-y-1 list-disc list-inside">
              <li><strong className="text-amber-300">Alert Source triage rules</strong> (in Settings → Triage Rules): global, apply across all alert sources, match on alert type + risk score, decide queue vs auto-create.</li>
              <li><strong className="text-amber-300">Inbound Webhook routing rules</strong> (here, per-source): per-source, match on event_type pattern, always create a case directly — they define the case shape (priority, title, tags).</li>
              <li>Use Alert Sources when a human should still decide if it is worth investigating. Use Inbound Webhooks when the source system has already made that call.</li>
            </ul>
          </div>
          <p className="text-xs text-slate-500">
            Add rules by going to the <strong className="text-slate-300">Sources tab</strong>, expanding your source (click ▸), and clicking <strong className="text-slate-300">+ Add Rule</strong>.
          </p>
        </div>
      ),
    },
    {
      num: "05",
      title: "Verify in the Event Log",
      color: "blue",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-slate-400">
            After sending an event, switch to the <strong className="text-slate-200">Event Log tab</strong> to confirm it was received and processed correctly.
          </p>
          <div className="bg-slate-900/60 border border-slate-700 rounded-lg overflow-hidden">
            <table className="w-full text-xs">
              <thead>
                <tr className="border-b border-slate-700 bg-slate-800/40">
                  <th className="text-left px-4 py-2 text-slate-400">Status</th>
                  <th className="text-left px-4 py-2 text-slate-400">Meaning</th>
                  <th className="text-left px-4 py-2 text-slate-400">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700/40">
                <tr>
                  <td className="px-4 py-2"><span className="px-2 py-0.5 rounded-full bg-green-500/20 text-green-400">processed</span></td>
                  <td className="px-4 py-2 text-slate-400">Rule matched, case created successfully</td>
                  <td className="px-4 py-2 text-slate-500">Click "View Case →" in the Outcome column</td>
                </tr>
                <tr>
                  <td className="px-4 py-2"><span className="px-2 py-0.5 rounded-full bg-slate-500/20 text-slate-400">ignored</span></td>
                  <td className="px-4 py-2 text-slate-400">No routing rule matched event_type</td>
                  <td className="px-4 py-2 text-slate-500">Add a rule that matches this event type in the Sources tab</td>
                </tr>
                <tr>
                  <td className="px-4 py-2"><span className="px-2 py-0.5 rounded-full bg-yellow-500/20 text-yellow-400">pending</span></td>
                  <td className="px-4 py-2 text-slate-400">Queued, worker hasn't processed it yet</td>
                  <td className="px-4 py-2 text-slate-500">Wait a few seconds, then refresh</td>
                </tr>
                <tr>
                  <td className="px-4 py-2"><span className="px-2 py-0.5 rounded-full bg-red-500/20 text-red-400">failed</span></td>
                  <td className="px-4 py-2 text-slate-400">Processing error — check the error message</td>
                  <td className="px-4 py-2 text-slate-500">Check the error in the Outcome column, fix the payload or template</td>
                </tr>
              </tbody>
            </table>
          </div>
          <p className="text-xs text-slate-500">
            The idempotency key protects against duplicate cases — sending the same event twice with the same key returns the original event_id without creating a second case.
          </p>
        </div>
      ),
    },
  ];

  const colorMap: Record<string, string> = {
    cyan: "bg-cyan-500/15 border-cyan-500/30 text-cyan-400",
    violet: "bg-violet-500/15 border-violet-500/30 text-violet-400",
    emerald: "bg-emerald-500/15 border-emerald-500/30 text-emerald-400",
    amber: "bg-amber-500/15 border-amber-500/30 text-amber-400",
    blue: "bg-blue-500/15 border-blue-500/30 text-blue-400",
  };

  return (
    <div className="space-y-6">
      <div className="p-4 bg-slate-800/30 border border-slate-700 rounded-xl text-sm text-slate-400">
        This guide shows your backend team exactly how to integrate an external system with OpsOwl Inbound Webhooks — from authentication to sending the first event and verifying the result.
      </div>
      {steps.map(step => (
        <div key={step.num} className="border border-slate-700 rounded-xl overflow-hidden">
          <div className={`flex items-center gap-3 px-5 py-3.5 border-b border-slate-700 bg-slate-800/30`}>
            <span className={`w-7 h-7 rounded-lg border flex items-center justify-center text-xs font-bold flex-shrink-0 ${colorMap[step.color]}`}>
              {step.num}
            </span>
            <h3 className="font-semibold text-slate-100 text-sm">{step.title}</h3>
          </div>
          <div className="p-5">
            {step.content}
          </div>
        </div>
      ))}
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────

export default function InboundPage() {
  const qc = useQueryClient();
  const [showSourceForm, setShowSourceForm] = useState(false);
  const [activeTab, setActiveTab] = useState<"sources" | "events" | "setup">("sources");

  const { data: sources = [], isLoading } = useQuery<Source[]>({
    queryKey: ["inbound-sources"],
    queryFn: () => api.get("/api/v1/inbound/sources"),
  });

  return (
    <RBACGate allowedRoles={["admin", "super_admin"]} fallback={
      <div className="p-12 flex flex-col items-center justify-center text-center">
        <div className="text-5xl mb-4">🔒</div>
        <p className="text-slate-200 font-semibold">Access Denied</p>
        <p className="text-slate-500 text-sm mt-2">Admin privileges required to manage inbound integrations.</p>
      </div>
    }>
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-100 mb-1">Inbound Webhooks</h1>
        <p className="text-slate-400 text-sm">
          Create investigation cases directly from external system events — no alert queue, no manual triage step. Use this when your system has already decided an event needs a case.
        </p>
      </div>

      {/* How it works — plain-language explainer */}
      <div className="mb-6 p-4 bg-slate-800/30 border border-slate-700 rounded-xl text-xs text-slate-400 space-y-2">
        <p className="font-medium text-slate-200 text-sm">How it works</p>
        <div className="grid grid-cols-4 gap-3 py-1">
          <div className="bg-slate-700/40 rounded-lg p-3">
            <p className="text-cyan-400 font-semibold mb-1">① Register Source</p>
            <p>Add your external system (e.g. "Core Banking"). OpsOwl generates an HMAC signing secret unique to this source.</p>
          </div>
          <div className="bg-slate-700/40 rounded-lg p-3">
            <p className="text-cyan-400 font-semibold mb-1">② Define Rules</p>
            <p>Map event type patterns to case priority and a title template. Use wildcards like <span className="font-mono text-cyan-300">fraud_*</span> to catch a family of events.</p>
          </div>
          <div className="bg-slate-700/40 rounded-lg p-3">
            <p className="text-cyan-400 font-semibold mb-1">③ Fire the Event</p>
            <p>Your system POSTs to <span className="font-mono text-cyan-300">/api/v1/inbound/events</span> with the signed headers. OpsOwl returns <span className="font-mono text-cyan-300">202 Accepted</span> instantly.</p>
          </div>
          <div className="bg-slate-700/40 rounded-lg p-3">
            <p className="text-cyan-400 font-semibold mb-1">④ Case Opens</p>
            <p>A background worker matches the rule and creates the case. The same idempotency key a second time? Silently ignored.</p>
          </div>
        </div>
        <div className="pt-2 border-t border-slate-700">
          <p className="text-amber-400 font-medium mb-1">When to use Inbound Webhooks vs Alert Sources</p>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <p className="text-slate-300 font-medium mb-0.5">Use Alert Sources when…</p>
              <p className="text-slate-500">Your fraud engine scores a transaction and an analyst still needs to decide if it warrants investigation. Creates an Alert in the queue → triage → case. Full subject resolution and dedup included.</p>
            </div>
            <div>
              <p className="text-slate-300 font-medium mb-0.5">Use Inbound Webhooks when…</p>
              <p className="text-slate-500">The external system has already made the call — a confirmed account takeover, a forged KYC document, a court-ordered freeze. No triage needed. Case opens immediately, async.</p>
            </div>
          </div>
        </div>
        <div className="pt-2 border-t border-slate-700">
          <p className="text-slate-300 font-medium mb-2">Real-world sources used with Inbound Webhooks</p>
          <div className="grid grid-cols-3 gap-2">
            <div className="bg-slate-800/60 rounded p-2">
              <p className="text-slate-200 font-medium text-[11px]">Fraud Engine (e.g. Decentro)</p>
              <p className="text-slate-500 text-[10px] mt-0.5">Event types: <span className="font-mono text-cyan-400/80">fraud_account_takeover</span>, <span className="font-mono text-cyan-400/80">fraud_*</span></p>
              <p className="text-slate-600 text-[10px] mt-0.5">Use when: ATO or high-confidence fraud confirmed by ML — no human triage needed</p>
            </div>
            <div className="bg-slate-800/60 rounded p-2">
              <p className="text-slate-200 font-medium text-[11px]">KYC Platform (Digio / Karza)</p>
              <p className="text-slate-500 text-[10px] mt-0.5">Event types: <span className="font-mono text-cyan-400/80">kyc_document_forged</span>, <span className="font-mono text-cyan-400/80">kyc_*</span></p>
              <p className="text-slate-600 text-[10px] mt-0.5">Use when: eKYC provider flags a forged Aadhaar, PAN, or address proof during onboarding</p>
            </div>
            <div className="bg-slate-800/60 rounded p-2">
              <p className="text-slate-200 font-medium text-[11px]">Core Banking (Finacle / BankWare)</p>
              <p className="text-slate-500 text-[10px] mt-0.5">Event types: <span className="font-mono text-cyan-400/80">large_wire_transfer</span>, <span className="font-mono text-cyan-400/80">dormant_account_*</span></p>
              <p className="text-slate-600 text-[10px] mt-0.5">Use when: CBS detects transactions exceeding internal thresholds or a dormant account suddenly activates</p>
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 border-b border-slate-700">
        {([
          { key: "sources", label: `Sources (${sources.length})` },
          { key: "events",  label: "Event Log" },
          { key: "setup",   label: "Setup Guide" },
        ] as { key: "sources" | "events" | "setup"; label: string }[]).map(({ key, label }) => (
          <button key={key} onClick={() => setActiveTab(key)}
            className={`px-4 py-2 text-sm transition-colors ${
              activeTab === key
                ? "text-cyan-400 border-b-2 border-cyan-400"
                : "text-slate-400 hover:text-slate-200"
            }`}>
            {label}
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

      {activeTab === "setup" && <SetupGuide sources={sources} />}

      {showSourceForm && (
        <SourceForm
          onClose={() => setShowSourceForm(false)}
          onSuccess={() => qc.invalidateQueries({ queryKey: ["inbound-sources"] })}
        />
      )}
    </div>
    </RBACGate>
  );
}
