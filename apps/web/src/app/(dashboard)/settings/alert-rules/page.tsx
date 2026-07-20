"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { Filter, Plus, Trash2, ToggleLeft, ToggleRight, Zap, Inbox, Eye, X } from "lucide-react";
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
  auto_assign_user_id?: string;
  created_at: string;
}

interface UserOption {
  id: string;
  full_name: string;
  role: string;
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
  const [assignTo, setAssignTo]   = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [serverError, setServerError] = useState("");

  const { data: usersData } = useQuery<UserOption[]>({
    queryKey: ["users-picker-rules"],
    queryFn: () => api.get<UserOption[]>("/api/v1/users?limit=100"),
    enabled: mode === "auto",
  });
  const analysts = (usersData ?? []).filter(u => ["analyst","manager","admin","super_admin"].includes(u.role));

  const parsedTags = tagsRaw.split(",").map((t: string) => t.trim()).filter(Boolean);

  const nameErr = !name.trim() ? "Rule name is required."
    : name.trim().length < 5   ? `Must be at least 5 characters (${name.trim().length}/5).`
    : name.length > 100        ? "Must be under 100 characters."
    : null;
  const scoreErr = !Number.isInteger(Number(minScore)) ? "Risk score must be a whole number."
    : minScore < 0 || minScore > 100 ? "Risk score must be between 0 and 100."
    : null;
  const sortErr = !Number.isInteger(Number(sortOrder)) ? "Sort order must be a whole number."
    : sortOrder < 1 || sortOrder > 999 ? "Sort order must be between 1 and 999."
    : null;
  const dedupErr = !Number.isInteger(Number(dedup)) ? "Dedup window must be a whole number."
    : dedup < 0 ? "Dedup window cannot be negative."
    : dedup > 8760 ? "Dedup window cannot exceed 8760 hours (1 year)."
    : null;
  const tagsErr = parsedTags.length > 10 ? "Maximum 10 tags allowed."
    : parsedTags.some(t => t.length > 50) ? "Each tag must be under 50 characters."
    : null;
  const hasErrors = !!(nameErr || scoreErr || sortErr || dedupErr || tagsErr);

  const mutation = useMutation({
    mutationFn: () => api.post<AlertRule>("/api/v1/alert-rules", {
      name: name.trim(), alert_type: alertType, min_risk_score: Number(minScore),
      priority_mapping: priority, mode, dedup_window_hours: Number(dedup),
      sort_order: Number(sortOrder), playbook_case_type: caseType,
      tags: parsedTags,
      ...(mode === "auto" && assignTo ? { auto_assign_user_id: assignTo } : {}),
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
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] w-full max-w-lg p-6 shadow-2xl max-h-[90vh] overflow-y-auto">
        <h2 className="text-slate-100 font-semibold text-lg mb-4">New Triage Rule</h2>
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
              <input type="number" min={0} max={100} step={1} value={minScore} onChange={e => setMinScore(Number(e.target.value))}
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

          {/* Auto-assign analyst (only when mode = auto) */}
          {mode === "auto" && (
            <div>
              <label className="block text-xs text-slate-400 mb-1">
                Auto-assign Analyst <span className="text-slate-600">(optional — leave blank to assign to queue admin)</span>
              </label>
              <select value={assignTo} onChange={e => setAssignTo(e.target.value)}
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500">
                <option value="">— No specific analyst (admin queue) —</option>
                {analysts.map(u => (
                  <option key={u.id} value={u.id}>{u.full_name} ({u.role})</option>
                ))}
              </select>
            </div>
          )}

          {/* Dedup + Sort Order */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs text-slate-400 mb-1">Dedup Window <span className="text-slate-600">(hours)</span></label>
              <input type="number" min={0} max={8760} step={1} value={dedup} onChange={e => setDedup(Number(e.target.value))}
                className={inputCls(dedupErr)} />
              {submitted && dedupErr
                ? <p className="text-red-400 text-[10px] mt-0.5">{dedupErr}</p>
                : <p className="text-slate-600 text-[10px] mt-0.5">Same entity + alert type within this window = skipped (max 8760h)</p>}
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Sort Order <span className="text-slate-600">(1–999)</span></label>
              <input type="number" min={1} max={999} step={1} value={sortOrder} onChange={e => setSortOrder(Number(e.target.value))}
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
            <div className="flex items-center justify-between mb-1">
              <label className="text-xs text-slate-400">Tags <span className="text-slate-600">(comma-separated, applied to auto-created case)</span></label>
              <span className={`text-[10px] ${parsedTags.length > 10 ? "text-red-400" : "text-slate-600"}`}>{parsedTags.length}/10</span>
            </div>
            <input value={tagsRaw} onChange={e => setTagsRaw(e.target.value)} placeholder="e.g. financial-crime, sanctions, urgent"
              className={inputCls(tagsErr)} />
            {submitted && tagsErr && <p className="text-red-400 text-[10px] mt-0.5">{tagsErr}</p>}
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

function RuleDetailModal({ rule, userMap, onClose }: { rule: AlertRule; userMap: Record<string, string>; onClose: () => void }) {
  const createdAt = new Date(rule.created_at).toLocaleString();
  const assignedUser = rule.auto_assign_user_id
    ? (userMap[rule.auto_assign_user_id] ?? "Unknown user")
    : "Admin queue";

  const Row = ({ label, value }: { label: string; value: React.ReactNode }) => (
    <div className="grid grid-cols-[160px_1fr] gap-3 py-2.5 border-b border-slate-800 last:border-0">
      <span className="text-xs font-semibold text-slate-500 uppercase tracking-wide pt-0.5">{label}</span>
      <span className="text-sm text-slate-200">{value}</span>
    </div>
  );

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] w-full max-w-lg shadow-2xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <div>
            <h2 className="text-slate-100 font-semibold text-base">{rule.name}</h2>
            <p className="text-slate-500 text-xs mt-0.5">Alert Triage Rule · sort order {rule.sort_order}</p>
          </div>
          <button onClick={onClose} className="p-1.5 rounded-lg text-slate-400 hover:text-slate-200 transition-colors">
            <X size={16} />
          </button>
        </div>
        <div className="px-6 py-4">
          <Row label="Status" value={
            <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-semibold ${rule.is_active ? "bg-green-500/15 text-green-400 border border-green-500/30" : "bg-slate-700 text-slate-400"}`}>
              <span className={`w-1.5 h-1.5 rounded-full ${rule.is_active ? "bg-green-400" : "bg-slate-500"}`} />
              {rule.is_active ? "Active" : "Inactive"}
            </span>
          } />
          <Row label="Mode" value={
            <span className="inline-flex items-center gap-1.5">
              {rule.mode === "auto" ? (
                <span className="flex items-center gap-1 px-2 py-0.5 bg-amber-500/15 text-amber-400 rounded text-xs font-semibold border border-amber-500/30">
                  <Zap size={10} />AUTO — case created immediately on match
                </span>
              ) : (
                <span className="flex items-center gap-1 px-2 py-0.5 bg-slate-700 text-slate-400 rounded text-xs font-semibold">
                  <Inbox size={10} />QUEUE — analyst reviews in alert queue
                </span>
              )}
            </span>
          } />
          <Row label="Alert Type" value={
            <code className="bg-slate-800 text-cyan-300 rounded px-1.5 py-0.5 text-xs font-mono">
              {rule.alert_type === "*" ? "* (catch-all — matches any type)" : rule.alert_type}
            </code>
          } />
          <Row label="Min Risk Score" value={
            <span>
              <span className="text-slate-200 font-semibold">{rule.min_risk_score}</span>
              <span className="text-slate-500 text-xs ml-1.5">/ 100 — alerts below this score are not matched</span>
            </span>
          } />
          <Row label="Priority Mapping" value={
            <span className={`px-2 py-0.5 rounded text-xs font-semibold uppercase ${
              rule.priority_mapping === "critical" ? "bg-red-500/15 text-red-400" :
              rule.priority_mapping === "high"     ? "bg-orange-500/15 text-orange-400" :
              rule.priority_mapping === "medium"   ? "bg-yellow-500/15 text-yellow-400" :
              "bg-slate-700 text-slate-400"
            }`}>{rule.priority_mapping}</span>
          } />
          <Row label="Dedup Window" value={
            <span>
              <span className="text-slate-200 font-semibold">{rule.dedup_window_hours}h</span>
              <span className="text-slate-500 text-xs ml-1.5">— same entity + alert type within this window is skipped as duplicate</span>
            </span>
          } />
          <Row label="Sort Order" value={
            <span>
              <span className="text-slate-200 font-semibold">{rule.sort_order}</span>
              <span className="text-slate-500 text-xs ml-1.5">— lower numbers evaluated first; first match wins</span>
            </span>
          } />
          {rule.playbook_case_type && (
            <Row label="Case Type" value={
              <span>
                <code className="bg-slate-800 text-cyan-300 rounded px-1.5 py-0.5 text-xs font-mono">{rule.playbook_case_type}</code>
                <span className="text-slate-500 text-xs ml-1.5">— sets case_type on auto-created case, triggers matching playbook</span>
              </span>
            } />
          )}
          {rule.mode === "auto" && (
            <Row label="Auto-assign To" value={
              <span className="text-cyan-400 font-medium">{assignedUser}</span>
            } />
          )}
          <Row label="Tags" value={
            rule.tags.length > 0 ? (
              <div className="flex gap-1.5 flex-wrap">
                {rule.tags.map(t => (
                  <span key={t} className="px-2 py-0.5 bg-slate-800 text-slate-300 rounded text-xs border border-slate-700">{t}</span>
                ))}
              </div>
            ) : <span className="text-slate-600 text-xs">No tags</span>
          } />
          <Row label="Rule ID" value={
            <code className="bg-slate-800 text-slate-400 rounded px-1.5 py-0.5 text-xs font-mono break-all">{rule.id}</code>
          } />
          <Row label="Created" value={<span className="text-slate-400 text-xs">{createdAt}</span>} />
        </div>
        <div className="px-6 pb-5">
          <button onClick={onClose} className="w-full px-4 py-2 rounded-lg border border-slate-600 text-slate-300 text-sm hover:border-slate-500 transition-colors">
            Close
          </button>
        </div>
      </div>
    </div>
  );
}

export default function AlertRulesPage() {
  const qc = useQueryClient();
  const [creating, setCreating] = useState(false);
  const [viewing, setViewing] = useState<AlertRule | null>(null);

  const { data, isFetching } = useQuery<{ rules: AlertRule[] }>({
    queryKey: ["alert-rules"],
    queryFn: () => api.get<{ rules: AlertRule[] }>("/api/v1/alert-rules"),
  });

  const { data: usersData2 } = useQuery<UserOption[]>({
    queryKey: ["users-picker-rules"],
    queryFn: () => api.get<UserOption[]>("/api/v1/users?limit=100"),
    staleTime: 5 * 60_000,
  });
  const userMap = Object.fromEntries((usersData2 ?? []).map(u => [u.id, u.full_name]));

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
            <h1 className="text-xl font-bold text-slate-100">Alert Triage Rules</h1>
          </div>
          <p className="text-slate-400 text-sm mt-1">Define how incoming alerts are scored, prioritised and routed to the right case type. Rules are evaluated in priority order — first match wins.</p>
        </div>
        <button onClick={() => setCreating(true)}
          className="flex items-center gap-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg text-sm font-medium transition-colors">
          <Plus size={14} />New Rule
        </button>
      </div>

      {isFetching && !data ? (
        <div className="flex items-center justify-center h-40 text-slate-500 text-sm">Loading rules…</div>
      ) : rules.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-40 bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] text-slate-500">
          <Filter className="w-8 h-8 mb-3 text-slate-600" />
          <p className="text-sm">No routing rules defined</p>
          <p className="text-xs text-slate-600 mt-1">Create a rule to automate alert triage</p>
        </div>
      ) : (
        <div className="space-y-3 max-h-[calc(100vh-200px)] overflow-y-auto pr-1">
          {rules.map((rule, idx) => (
            <div key={rule.id} className={`bg-[#131f30] border rounded-xl p-4 transition-all ${rule.is_active ? "border-slate-700" : "border-slate-800 opacity-60"}`}>
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
                      {rule.mode === "auto" && (
                        <span>
                          Assigns to:{" "}
                          <span className="text-cyan-400">
                            {rule.auto_assign_user_id && userMap[rule.auto_assign_user_id]
                              ? userMap[rule.auto_assign_user_id]
                              : rule.auto_assign_user_id
                                ? "Unknown user"
                                : "Admin queue"}
                          </span>
                        </span>
                      )}
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
                  <button onClick={() => setViewing(rule)} title="View details"
                    className="p-1.5 rounded-lg text-slate-400 hover:text-cyan-400 transition-colors">
                    <Eye size={14} />
                  </button>
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
      {viewing && (
        <RuleDetailModal
          rule={viewing}
          userMap={userMap}
          onClose={() => setViewing(null)}
        />
      )}
    </div>
  );
}
