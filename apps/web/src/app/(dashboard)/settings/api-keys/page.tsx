"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { RBACGate } from "@/components/auth/RBACGate";

interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  scopes: string[];
  last_used_at: string | null;
  expires_at: string | null;
  created_at: string;
}

interface CreatedKey extends APIKey {
  key: string;
}

const SCOPE_OPTIONS = [
  { value: "cases:read", label: "Cases — Read" },
  { value: "cases:write", label: "Cases — Write" },
  { value: "documents:read", label: "Documents — Read" },
  { value: "audit:read", label: "Audit Logs — Read" },
];

function timeAgo(iso: string) {
  const diff = Date.now() - new Date(iso).getTime();
  const m = Math.floor(diff / 60_000);
  if (m < 1) return "just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

function KeyRevealBanner({ createdKey, onClose }: { createdKey: CreatedKey; onClose: () => void }) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(createdKey.key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="mb-6 p-5 bg-amber-500/10 border border-amber-500/40 rounded-xl">
      <div className="flex items-start justify-between mb-3">
        <div>
          <p className="text-amber-400 font-semibold text-sm">API key created — copy it now</p>
          <p className="text-amber-400/70 text-xs mt-0.5">
            This key will not be shown again. Store it securely.
          </p>
        </div>
        <button onClick={onClose} className="text-amber-400/50 hover:text-amber-400 text-lg leading-none">
          ✕
        </button>
      </div>
      <div className="flex items-center gap-3">
        <code className="flex-1 bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm font-mono text-cyan-300 overflow-x-auto whitespace-nowrap">
          {createdKey.key}
        </code>
        <button
          onClick={handleCopy}
          className="px-4 py-2 bg-amber-500/20 hover:bg-amber-500/30 border border-amber-500/40 text-amber-400 text-sm rounded transition-colors whitespace-nowrap"
        >
          {copied ? "Copied ✓" : "Copy key"}
        </button>
      </div>
      <p className="text-amber-400/60 text-xs mt-2">
        Use as: <code className="text-amber-300/80">X-API-Key: {createdKey.key.slice(0, 20)}…</code>
      </p>
    </div>
  );
}

function CreateKeyModal({ onClose, onCreated }: { onClose: () => void; onCreated: (key: CreatedKey) => void }) {
  const [name, setName] = useState("");
  const [scopes, setScopes] = useState<string[]>(["cases:read"]);
  const [expiresInDays, setExpiresInDays] = useState(0);
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: () =>
      api.post<CreatedKey>("/api/v1/api-keys", {
        name,
        scopes,
        expires_in_days: expiresInDays,
      }),
    onSuccess: (data) => {
      onCreated(data);
    },
    onError: (e) => {
      setError(e instanceof APIError ? e.message : "Failed to create API key");
    },
  });

  function toggleScope(s: string) {
    setScopes((prev) =>
      prev.includes(s) ? prev.filter((x) => x !== s) : [...prev, s]
    );
  }

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-[#1e293b] border border-slate-700 rounded-2xl w-full max-w-md p-6 shadow-2xl">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-white font-semibold text-lg">New API Key</h2>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300 text-xl">✕</button>
        </div>

        {error && (
          <div className="mb-4 px-4 py-3 rounded-lg text-sm bg-red-500/10 border border-red-500/30 text-red-400">
            {error}
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="block text-xs text-slate-400 mb-1">Key Name *</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. CI Pipeline, Finacle Integration"
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:border-cyan-500 transition-colors"
            />
          </div>

          <div>
            <label className="block text-xs text-slate-400 mb-2">Scopes</label>
            <div className="space-y-2">
              {SCOPE_OPTIONS.map((opt) => (
                <label key={opt.value} className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={scopes.includes(opt.value)}
                    onChange={() => toggleScope(opt.value)}
                    className="accent-cyan-500"
                  />
                  <span className="text-sm text-slate-300">{opt.label}</span>
                  <code className="text-xs text-slate-500 font-mono">{opt.value}</code>
                </label>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-xs text-slate-400 mb-1">Expiry</label>
            <select
              value={expiresInDays}
              onChange={(e) => setExpiresInDays(Number(e.target.value))}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-cyan-500"
            >
              <option value={0}>Never expires</option>
              <option value={30}>30 days</option>
              <option value={90}>90 days</option>
              <option value={180}>180 days</option>
              <option value={365}>1 year</option>
            </select>
          </div>
        </div>

        <div className="flex gap-3 mt-6">
          <button
            onClick={() => mutation.mutate()}
            disabled={!name.trim() || scopes.length === 0 || mutation.isPending}
            className="flex-1 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm font-medium rounded transition-colors"
          >
            {mutation.isPending ? "Creating…" : "Create Key"}
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm border border-slate-700 rounded transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

function APIKeyRow({ keyData, onRevoke }: { keyData: APIKey; onRevoke: (id: string) => void }) {
  const [confirming, setConfirming] = useState(false);

  return (
    <tr className="border-b border-slate-700/40 hover:bg-slate-700/20 transition-colors">
      <td className="px-5 py-4">
        <div className="text-slate-200 font-medium text-sm">{keyData.name}</div>
        <code className="text-xs text-cyan-400/80 font-mono">{keyData.key_prefix}…</code>
      </td>
      <td className="px-5 py-4">
        <div className="flex flex-wrap gap-1">
          {keyData.scopes.map((s) => (
            <span key={s} className="text-xs bg-slate-700/70 text-slate-300 px-2 py-0.5 rounded font-mono">
              {s}
            </span>
          ))}
        </div>
      </td>
      <td className="px-5 py-4 text-slate-400 text-xs tabular-nums">
        {keyData.last_used_at ? timeAgo(keyData.last_used_at) : <span className="text-slate-600 italic">Never</span>}
      </td>
      <td className="px-5 py-4 text-slate-400 text-xs tabular-nums">
        {keyData.expires_at
          ? new Date(keyData.expires_at).toLocaleDateString("en-IN", { day: "2-digit", month: "short", year: "numeric" })
          : <span className="text-slate-600">No expiry</span>
        }
      </td>
      <td className="px-5 py-4 text-slate-500 text-xs tabular-nums">
        {new Date(keyData.created_at).toLocaleDateString("en-IN", { day: "2-digit", month: "short", year: "numeric" })}
      </td>
      <td className="px-5 py-4">
        {confirming ? (
          <div className="flex items-center gap-2">
            <span className="text-xs text-red-400">Revoke?</span>
            <button
              onClick={() => { onRevoke(keyData.id); setConfirming(false); }}
              className="text-xs px-2 py-1 bg-red-600/20 border border-red-500/30 text-red-400 rounded hover:bg-red-600/30 transition-colors"
            >
              Yes
            </button>
            <button
              onClick={() => setConfirming(false)}
              className="text-xs px-2 py-1 text-slate-500 hover:text-slate-300 transition-colors"
            >
              No
            </button>
          </div>
        ) : (
          <button
            onClick={() => setConfirming(true)}
            className="text-xs text-slate-500 hover:text-red-400 transition-colors"
          >
            Revoke
          </button>
        )}
      </td>
    </tr>
  );
}

function APIKeysContent() {
  const qc = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [createdKey, setCreatedKey] = useState<CreatedKey | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["api-keys"],
    queryFn: () => api.get<{ api_keys: APIKey[] }>("/api/v1/api-keys"),
    select: (d) => d.api_keys ?? [],
  });

  const revokeMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/api-keys/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["api-keys"] }),
  });

  function handleCreated(key: CreatedKey) {
    setShowCreate(false);
    setCreatedKey(key);
    qc.invalidateQueries({ queryKey: ["api-keys"] });
  }

  const keys = data ?? [];

  return (
    <div className="p-8 max-w-5xl">
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 mb-1">API Keys</h1>
          <p className="text-slate-400 text-sm">
            Programmatic access to OpsOwl. Keys authenticate via the <code className="text-cyan-400/80 font-mono text-xs">X-API-Key</code> header.
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium rounded-lg transition-colors"
        >
          + Generate Key
        </button>
      </div>

      {/* Reveal banner shown once after creation */}
      {createdKey && (
        <KeyRevealBanner createdKey={createdKey} onClose={() => setCreatedKey(null)} />
      )}

      {/* Table */}
      <div className="bg-[#1e293b] border border-slate-700 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-700 text-slate-500 text-xs uppercase tracking-wide">
              <th className="text-left px-5 py-3 font-medium">Name / Prefix</th>
              <th className="text-left px-5 py-3 font-medium">Scopes</th>
              <th className="text-left px-5 py-3 font-medium">Last Used</th>
              <th className="text-left px-5 py-3 font-medium">Expires</th>
              <th className="text-left px-5 py-3 font-medium">Created</th>
              <th className="px-5 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              [...Array(3)].map((_, i) => (
                <tr key={i} className="border-b border-slate-700/40">
                  {[200, 180, 80, 80, 80, 60].map((w, j) => (
                    <td key={j} className="px-5 py-4">
                      <div className="h-4 bg-slate-700/60 animate-pulse rounded" style={{ width: w }} />
                    </td>
                  ))}
                </tr>
              ))
            ) : keys.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-5 py-16 text-center">
                  <p className="text-slate-400 text-sm mb-2">No API keys yet</p>
                  <button
                    onClick={() => setShowCreate(true)}
                    className="text-cyan-400 hover:text-cyan-300 text-sm"
                  >
                    Generate your first key →
                  </button>
                </td>
              </tr>
            ) : (
              keys.map((k) => (
                <APIKeyRow
                  key={k.id}
                  keyData={k}
                  onRevoke={(id) => revokeMutation.mutate(id)}
                />
              ))
            )}
          </tbody>
        </table>
      </div>

      <div className="mt-4 p-4 bg-slate-800/50 border border-slate-700/50 rounded-lg">
        <p className="text-slate-400 text-xs">
          <span className="text-slate-300 font-medium">Usage: </span>
          <code className="font-mono text-cyan-400/80">curl -H &quot;X-API-Key: your_key&quot; http://localhost:8080/api/v1/cases</code>
          {" — "}Keys bypass JWT auth and authenticate as the key owner with the configured scopes.
        </p>
      </div>

      {showCreate && (
        <CreateKeyModal onClose={() => setShowCreate(false)} onCreated={handleCreated} />
      )}
    </div>
  );
}

export default function APIKeysPage() {
  return (
    <RBACGate allowedRoles={["admin", "super_admin"]} fallback={
      <div className="p-8 text-slate-400 text-sm">Admin access required to manage API keys.</div>
    }>
      <APIKeysContent />
    </RBACGate>
  );
}
