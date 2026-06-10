"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { Filter, Plus, Trash2, ToggleLeft, ToggleRight, Zap, Inbox } from "lucide-react";
import { CaseTypeSelect } from "@/components/CaseTypeSelect";

interface AlertRule {
  id: string;
  name: string;
  alert_type: string;
  min_risk_score: number;
  priority_mapping: string;
  tags: string[];
  playbook_case_type: string;
  dedup_window_hours: number;
  mode: string;
  sort_order: number;
  is_active: boolean;
  created_at: string;
}

const KNOWN_ALERT_TYPES = [
  "*",
  "UNUSUAL_CASH_ACTIVITY", "RAPID_MOVEMENT", "VELOCITY_BREACH", "STRUCTURING",
  "LAYERING", "TRADE_BASED_ML", "HIGH_RISK_GEOGRAPHY", "DORMANT_REACTIVATION",
  "NEW_ACCOUNT_SPIKE", "SANCTIONS_HIT", "PEP_MATCH", "ADVERSE_MEDIA",
  "CTR_THRESHOLD", "UNUSUAL_PATTERN", "BENEFICIAL_OWNERSHIP",
];

const PRIORITIES = ["critical", "high", "medium", "low"];

function CreateRuleModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [name, setName]           = useState("");
  const [alertType, setAlertType] = useState("*");
  const [minScore, setMinScore]   = useState(0);
  const [priority, setPriority]   = useState("medium");
  const [mode, setMode]           = useState("queue");
  const [dedup, setDedup]         = useState(72);
  const [sortOrder, setSortOrder] = useState(100);
  const [caseType, setCaseType]   = useState("");
  const [tagsRaw, setTagsRaw]     = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [serverError, setServerError] = useState("");

  const nameErr = !name.trim() ? "Rule name is required."
    : name.trim().length < 5   ? `Must be at least 5 characters (${name.trim().length}/5).`
    : name.length > 100        ? "Must be under 100 characters."
    : null;
  const scoreErr = minScore < 0 || minScore > 100 ? "Risk score must be between 0 and 100." : null;
  const sortErr  = sortOrder < 1 || sortOrder > 999 ? "Sort order must be between 1 and 999." : null;
  const dedupErr = dedup < 0 ? "Dedup window cannot be negative." : null;
  const hasErrors = !!(nameErr || scoreErr || sortErr || dedupErr);

  const mutation = useMutation({
    mutationFn: () => api.post<AlertRule>("/api/v1/alert-rules", {
      name: name.trim(), alert_type: alertType, min_risk_score: Number(minScore),
      priority_mapping: priority, mode, dedup_window_hours: Number(dedup),
      sort_order: Number(sortOrder), playbook_case_type: caseType,
      tags: tagsRaw.split(",").map((t: string) => t.trim()).filter(Boolean),
    }),
    onSuccess: () => { onCreated(); onClose(); },
    onError: (err: any) => setServerError(err.message || "Failed to create rule"),
  });

  function handleSubmit() {
    setSubmitted(true);
    if (hasErrors) return;
    mutation.mutate();
  }

  const inputCls = (err: string | null) =>
    `w-full bg-slate-900 border rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500 ${submitted && err ? "border-red-500/60" : "border-slate-600"}`;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-[#1e293b] border border-slate-700 rounded-xl w-full max-w-lg p-6 shadow-2xl max-h-[90vh] overflow-y-auto">
        <h2 className="text-slate-100 font-semibold text-lg mb-4">New Routing Rule</h2>
        {serverError && <div className="mb-3 text-red-400 text-sm bg-red-500/10 border border-red-500/30 rounded-lg px-3 py-2">{serverError}</div>}
        <div className="space-y-3">

          {/* Name */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="text-xs text-slate-400">Rule Name <span className="text-red-400">*</span></label>
              <span className="text-[10px] text-slate-600">{name.trim().length}/100</span>
            </div>
            <input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. Sanctions hit → auto-escalate" maxLength={101}
              className={inputCls(nameErr)} />
            {submitted && nameErr && <p className="text-red-400 text-[10px] mt-0.5">{nameErr}</p>}
          </div>

          {/* Alert Type + Min Risk Score */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs text-slate-400 mb-1">Alert Type</label>
              <select value={alertType} onChange={e => setAlertType(e.target.value)}
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500">
                {KNOWN_ALERT_TYPES.map(t => <option key={t} value={t}>{t === "*" ? "* (catch-all)" : t}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Min Risk Score <span className="text-slate-600">(0–100)</span></label>
              <input type="number" min={0} max={100} value={minScore} onChange={e => setMinScore(Number(e.target.value))}
                className={inputCls(scoreErr)} />
              {submitted && scoreErr && <p className="text-red-400 text-[10px] mt-0.5">{scoreErr}</p>}
            </div>
          </div>

          {/* Priority + Mode */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs text-slate-400 mb-1">Priority Mapping</label>
              <select value={priority} onChange={e => setPriority(e.target.value)}
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500">
                {PRIORITIES.map(p => <option key={p} value={p}>{p.charAt(0).toUpperCase()+p.slice(1)}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Mode</label>
              <select value={mode} onChange={e => setMode(e.target.value)}
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500">
                <option value="queue">Queue — analyst reviews manually</option>
                <option value="auto">Auto — convert to case immediately</option>
              </select>
              {mode === "auto" && (
                <p className="text-amber-400/80 text-[10px] mt-0.5">Case + playbook created automatically on every matching alert.</p>
              )}
            </div>
          </div>

          {/* Dedup + Sort Order */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs text-slate-400 mb-1">Dedup Window <span className="text-slate-600">(hours)</span></label>
              <input type="number" min={0} value={dedup} onChange={e => setDedup(Number(e.target.value))}
                className={inputCls(dedupErr)} />
              {submitted && dedupErr
                ? <p className="text-red-400 text-[10px] mt-0.5">{dedupErr}</p>
                : <p className="text-slate-600 text-[10px] mt-0.5">Same entity + alert type within this window = skipped</p>}
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Sort Order <span className="text-slate-600">(1–999)</span></label>
              <input type="number" min={1} max={999} value={sortOrder} onChange={e => setSortOrder(Number(e.target.value))}
                className={inputCls(sortErr)} />
              {submitted && sortErr
                ? <p className="text-red-400 text-[10px] mt-0.5">{sortErr}</p>
                : <p className="text-slate-600 text-[10px] mt-0.5">Lower number evaluated first</p>}
            </div>
          </div>

          {/* Playbook Case Type */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">
              Case Type <span className="text-slate-600">(optional — sets case_type on auto-created case, triggers matching playbook)</span>
            </label>
            <CaseTypeSelect
              value={caseType}
              onChange={setCaseType}
              className="w-full bg-slate-900 border border-slate-600 text-slate-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
            />
          </div>

          {/* Tags */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">Tags <span className="text-slate-600">(comma-separated, applied to auto-created case)</span></label>
            <input value={tagsRaw} onChange={e => setTagsRaw(e.target.value)} placeholder="e.g. financial-crime, sanctions, urgent"
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500" />
          </div>
        </div>
        <div className="flex gap-3 mt-5">
          <button onClick={onClose} className="flex-1 px-4 py-2 rounded-lg border border-slate-600 text-slate-300 text-sm hover:border-slate-500">Cancel</button>
          <button onClick={handleSubmit} disabled={mutation.isPending}
            className="flex-1 px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium disabled:opacity-50">
            {mutation.isPending ? "Creating…" : "Create Rule"}
          </button>
        </div>
      </div>
    </div>
  );
}

export default function AlertRulesPage() {
  const qc = useQueryClient();
  const [creating, setCreating] = useState(false);

  const { data, isFetching } = useQuery<{ rules: AlertRule[] }>({
    queryKey: ["alert-rules"],
    queryFn: () => api.get<{ rules: AlertRule[] }>("/api/v1/alert-rules"),
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, isActive }: { id: string; isActive: boolean }) =>
      api.patch(`/api/v1/alert-rules/${id}`, { is_active: isActive }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules"] }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/alert-rules/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules"] }),
  });

  const rules = data?.rules ?? [];

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <div className="flex items-center gap-2.5">
            <Filter className="w-5 h-5 text-cyan-400" />
            <h1 className="text-xl font-bold text-slate-100">Alert Routing Rules</h1>
          </div>
          <p className="text-slate-400 text-sm mt-1">Define how incoming alerts are triaged and routed. Rules are evaluated in sort_order ascending — first match wins.</p>
        </div>
        <button onClick={() => setCreating(true)}
          className="flex items-center gap-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg text-sm font-medium transition-colors">
          <Plus size={14} />New Rule
        </button>
      </div>

      {isFetching && !data ? (
        <div className="flex items-center justify-center h-40 text-slate-500 text-sm">Loading rules…</div>
      ) : rules.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-40 bg-[#1e293b] border border-slate-700 rounded-xl text-slate-500">
          <Filter className="w-8 h-8 mb-3 text-slate-600" />
          <p className="text-sm">No routing rules defined</p>
          <p className="text-xs text-slate-600 mt-1">Create a rule to automate alert triage</p>
        </div>
      ) : (
        <div className="space-y-3 max-h-[calc(100vh-200px)] overflow-y-auto pr-1">
          {rules.map((rule, idx) => (
            <div key={rule.id} className={`bg-[#1e293b] border rounded-xl p-4 transition-all ${rule.is_active ? "border-slate-700" : "border-slate-800 opacity-60"}`}>
              <div className="flex items-start justify-between gap-4">
                <div className="flex items-start gap-3 flex-1 min-w-0">
                  <div className="text-slate-600 text-xs font-mono mt-0.5 w-5 text-right flex-shrink-0">
                    {rule.sort_order}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-semibold text-slate-200 text-sm">{rule.name}</span>
                      {rule.mode === "auto" ? (
                        <span className="flex items-center gap-1 px-1.5 py-0.5 bg-amber-500/15 text-amber-400 rounded text-[10px] font-semibold border border-amber-500/30">
                          <Zap size={9} />AUTO
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 px-1.5 py-0.5 bg-slate-700 text-slate-400 rounded text-[10px] font-semibold">
                          <Inbox size={9} />QUEUE
                        </span>
                      )}
                      <span className={`text-[10px] px-1.5 py-0.5 rounded font-semibold uppercase ${
                        rule.priority_mapping === "critical" ? "bg-red-500/15 text-red-400" :
                        rule.priority_mapping === "high"     ? "bg-orange-500/15 text-orange-400" :
                        rule.priority_mapping === "medium"   ? "bg-yellow-500/15 text-yellow-400" :
                        "bg-slate-700 text-slate-400"
                      }`}>{rule.priority_mapping}</span>
                    </div>
                    <div className="text-slate-500 text-xs mt-1.5 flex items-center gap-3 flex-wrap">
                      <span>
                        Type: <span className="text-slate-300">{rule.alert_type === "*" ? "* (catch-all)" : rule.alert_type}</span>
                      </span>
                      <span>Min score: <span className="text-slate-300">{rule.min_risk_score}</span></span>
                      <span>Dedup: <span className="text-slate-300">{rule.dedup_window_hours}h</span></span>
                      {rule.playbook_case_type && <span>Playbook: <span className="text-slate-300">{rule.playbook_case_type}</span></span>}
                    </div>
                    {rule.tags.length > 0 && (
                      <div className="flex gap-1.5 mt-2 flex-wrap">
                        {rule.tags.map(t => (
                          <span key={t} className="px-1.5 py-0.5 bg-slate-800 text-slate-400 rounded text-[10px] border border-slate-700">{t}</span>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  <button onClick={() => toggleMutation.mutate({ id: rule.id, isActive: !rule.is_active })}
                    className="p-1.5 rounded-lg text-slate-400 hover:text-slate-200 transition-colors">
                    {rule.is_active ? <ToggleRight size={18} className="text-green-400" /> : <ToggleLeft size={18} />}
                  </button>
                  <button onClick={() => { if (confirm(`Delete rule "${rule.name}"?`)) deleteMutation.mutate(rule.id); }}
                    className="p-1.5 rounded-lg text-slate-400 hover:text-red-400 transition-colors">
                    <Trash2 size={14} />
                  </button>
                </div>
              </div>
            </div>
          ))}

          <div className="mt-4 bg-slate-800/40 border border-slate-700/50 rounded-xl p-4">
            <p className="text-slate-500 text-xs">
              <span className="text-slate-400 font-semibold">Default fallback:</span> Alerts that don&apos;t match any rule go to the queue with priority derived from their risk level.
            </p>
          </div>
        </div>
      )}

      {creating && (
        <CreateRuleModal
          onClose={() => setCreating(false)}
          onCreated={() => {
            setCreating(false);
            qc.invalidateQueries({ queryKey: ["alert-rules"] });
          }}
        />
      )}
    </div>
  );
}
