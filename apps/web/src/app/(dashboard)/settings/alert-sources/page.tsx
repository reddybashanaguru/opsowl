"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { Plug, Plus, Trash2, ToggleLeft, ToggleRight, Copy, Check } from "lucide-react";

interface AlertSource {
  id: string;
  name: string;
  source_type: string;
  source_system: string;
  webhook_secret?: string;
  description: string;
  is_active: boolean;
  created_at: string;
}

const SOURCE_SYSTEMS = ["actimize", "sas_aml", "oracle_fccm", "featurespace", "generic"];
const SOURCE_TYPES   = ["webhook", "manual", "csv"];

const SYSTEM_LABELS: Record<string, string> = {
  actimize:     "NICE Actimize",
  sas_aml:      "SAS AML",
  oracle_fccm:  "Oracle FCCM",
  featurespace: "Featurespace ARIC",
  generic:      "Generic / Custom",
};

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <button onClick={copy} className="p-1 rounded text-slate-400 hover:text-slate-200 transition-colors" title="Copy">
      {copied ? <Check size={12} className="text-green-400" /> : <Copy size={12} />}
    </button>
  );
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

function CreateSourceModal({ onClose, onCreated }: { onClose: () => void; onCreated: (src: AlertSource) => void }) {
  const [name, setName]         = useState("");
  const [system, setSystem]     = useState("generic");
  const [type, setType]         = useState("webhook");
  const [desc, setDesc]         = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [error, setError]       = useState("");

  const nameErr = !name.trim() ? "Name is required."
    : name.trim().length < 5   ? `Name must be at least 5 characters (${name.trim().length}/5).`
    : name.length > 100        ? "Name must be under 100 characters."
    : null;
  const descErr = desc.length > 300 ? `Description must be under 300 characters (${desc.length}/300).` : null;
  const hasErrors = !!(nameErr || descErr);

  const mutation = useMutation({
    mutationFn: () => api.post<AlertSource>("/api/v1/alert-sources", { name: name.trim(), source_system: system, source_type: type, description: desc }),
    onSuccess: (data) => { onCreated(data as AlertSource); onClose(); },
    onError: (err: any) => setError(err.message || "Failed to create source"),
  });

  function handleSubmit() {
    setSubmitted(true);
    if (hasErrors) return;
    mutation.mutate();
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-[#1e293b] border border-slate-700 rounded-xl w-full max-w-md p-6 shadow-2xl">
        <h2 className="text-slate-100 font-semibold text-lg mb-4">New Alert Source</h2>
        {error && <div className="mb-3 text-red-400 text-sm bg-red-500/10 border border-red-500/30 rounded-lg px-3 py-2">{error}</div>}
        <div className="space-y-3">
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="text-xs text-slate-400">Name <span className="text-red-400">*</span></label>
              <span className="text-[10px] text-slate-600">{name.trim().length}/100</span>
            </div>
            <input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. Actimize Production TMS" maxLength={101}
              className={`w-full bg-slate-900 border rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500 ${submitted && nameErr ? "border-red-500/60" : "border-slate-600"}`} />
            {submitted && nameErr && <p className="text-red-400 text-[10px] mt-0.5">{nameErr}</p>}
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Source System</label>
            <select value={system} onChange={e => setSystem(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500">
              {SOURCE_SYSTEMS.map(s => <option key={s} value={s}>{SYSTEM_LABELS[s]}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-xs text-slate-400 mb-1">Integration Type</label>
            <select value={type} onChange={e => setType(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500">
              {SOURCE_TYPES.map(t => <option key={t} value={t}>{t.charAt(0).toUpperCase()+t.slice(1)}</option>)}
            </select>
          </div>
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="text-xs text-slate-400">Description <span className="text-slate-600">(optional)</span></label>
              <span className={`text-[10px] ${desc.length > 300 ? "text-red-400" : "text-slate-600"}`}>{desc.length}/300</span>
            </div>
            <input value={desc} onChange={e => setDesc(e.target.value)} placeholder="e.g. Production Actimize instance, connected via webhook" maxLength={301}
              className={`w-full bg-slate-900 border rounded-lg px-3 py-2 text-slate-100 text-sm focus:outline-none focus:border-cyan-500 ${submitted && descErr ? "border-red-500/60" : "border-slate-600"}`} />
            {submitted && descErr && <p className="text-red-400 text-[10px] mt-0.5">{descErr}</p>}
          </div>
          {type === "webhook" && (
            <div className="bg-slate-800/60 border border-slate-700/50 rounded-lg px-3 py-2.5 text-[11px] text-slate-400">
              After creating, you will receive a one-time webhook secret and an ingest URL to configure in your TMS.
            </div>
          )}
        </div>
        <div className="flex gap-3 mt-5">
          <button onClick={onClose} className="flex-1 px-4 py-2 rounded-lg border border-slate-600 text-slate-300 text-sm hover:border-slate-500">Cancel</button>
          <button onClick={handleSubmit} disabled={mutation.isPending}
            className="flex-1 px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium disabled:opacity-50">
            {mutation.isPending ? "Creating…" : "Create Source"}
          </button>
        </div>
      </div>
    </div>
  );
}

export default function AlertSourcesPage() {
  const qc = useQueryClient();
  const [creating, setCreating] = useState(false);
  const [newSource, setNewSource] = useState<AlertSource | null>(null);

  const { data, isFetching } = useQuery<{ sources: AlertSource[] }>({
    queryKey: ["alert-sources"],
    queryFn: () => api.get<{ sources: AlertSource[] }>("/api/v1/alert-sources"),
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, isActive }: { id: string; isActive: boolean }) =>
      api.patch(`/api/v1/alert-sources/${id}`, { is_active: isActive }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-sources"] }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/alert-sources/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-sources"] }),
  });

  const sources = data?.sources ?? [];

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <div className="flex items-center gap-2.5">
            <Plug className="w-5 h-5 text-cyan-400" />
            <h1 className="text-xl font-bold text-slate-100">Alert Sources</h1>
          </div>
          <p className="text-slate-400 text-sm mt-1">Configure TMS integrations that push alerts to OpsOwl</p>
        </div>
        <button onClick={() => setCreating(true)}
          className="flex items-center gap-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg text-sm font-medium transition-colors">
          <Plus size={14} />New Source
        </button>
      </div>

      {/* New source credential card */}
      {newSource?.webhook_secret && (
        <div className="mb-6 bg-amber-500/10 border border-amber-500/30 rounded-xl p-4">
          <div className="text-amber-400 font-semibold text-sm mb-2">⚠ Save your webhook secret — it won't be shown again</div>
          <div className="flex items-center gap-2 bg-slate-900 rounded-lg px-3 py-2">
            <code className="font-mono text-xs text-slate-300 flex-1 break-all">{newSource.webhook_secret}</code>
            <CopyButton value={newSource.webhook_secret} />
          </div>
          <p className="text-slate-400 text-xs mt-2">
            Use this as the HMAC-SHA256 key. Send the signature in the <code className="font-mono text-xs">X-OpsOwl-Signature: sha256=&lt;hex&gt;</code> header.
          </p>
          <div className="mt-3 text-xs text-slate-400">
            Webhook URL: <code className="font-mono text-slate-300 ml-1">/api/v1/alerts/ingest/{newSource.id}</code>
            <CopyButton value={`/api/v1/alerts/ingest/${newSource.id}`} />
          </div>
          <button onClick={() => setNewSource(null)} className="mt-3 text-xs text-slate-500 hover:text-slate-300">Dismiss</button>
        </div>
      )}

      {isFetching && !data ? (
        <div className="flex items-center justify-center h-40 text-slate-500 text-sm">Loading sources…</div>
      ) : sources.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-40 bg-[#1e293b] border border-slate-700 rounded-xl text-slate-500">
          <Plug className="w-8 h-8 mb-3 text-slate-600" />
          <p className="text-sm">No alert sources configured</p>
          <p className="text-xs text-slate-600 mt-1">Add a source to start receiving TMS alerts</p>
        </div>
      ) : (
        <div className="space-y-3">
          {sources.map(src => (
            <div key={src.id} className="bg-[#1e293b] border border-slate-700 rounded-xl p-4 flex items-center justify-between gap-4">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-3">
                  <span className="font-semibold text-slate-200 text-sm">{src.name}</span>
                  <span className={`text-[10px] px-2 py-0.5 rounded-full font-semibold uppercase ${src.is_active ? "bg-green-500/15 text-green-400" : "bg-slate-700 text-slate-400"}`}>
                    {src.is_active ? "Active" : "Inactive"}
                  </span>
                </div>
                <div className="text-slate-500 text-xs mt-1">
                  {SYSTEM_LABELS[src.source_system] ?? src.source_system} · {src.source_type}
                  {src.description && <span> · {src.description}</span>}
                </div>
                <div className="flex items-center gap-1 mt-1.5">
                  <code className="text-slate-600 text-[11px] font-mono">{API_BASE}/api/v1/alerts/ingest/{src.id}</code>
                  <CopyButton value={`${API_BASE}/api/v1/alerts/ingest/${src.id}`} />
                </div>
              </div>
              <div className="flex items-center gap-2 flex-shrink-0">
                <button onClick={() => toggleMutation.mutate({ id: src.id, isActive: !src.is_active })}
                  className="p-1.5 rounded-lg text-slate-400 hover:text-slate-200 transition-colors" title={src.is_active ? "Deactivate" : "Activate"}>
                  {src.is_active ? <ToggleRight size={18} className="text-green-400" /> : <ToggleLeft size={18} />}
                </button>
                <button onClick={() => { if (confirm(`Delete source "${src.name}"?`)) deleteMutation.mutate(src.id); }}
                  className="p-1.5 rounded-lg text-slate-400 hover:text-red-400 transition-colors" title="Delete">
                  <Trash2 size={14} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {creating && (
        <CreateSourceModal
          onClose={() => setCreating(false)}
          onCreated={(src) => {
            setCreating(false);
            if (src.webhook_secret) setNewSource(src);
            qc.invalidateQueries({ queryKey: ["alert-sources"] });
          }}
        />
      )}
    </div>
  );
}
