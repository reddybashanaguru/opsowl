"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────────────────────

interface Webhook {
  id: string;
  name: string;
  url: string;
  signing_secret?: string;
  event_types: string[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface Delivery {
  id: string;
  webhook_id: string;
  event_type: string;
  case_id: string | null;
  status: "pending" | "success" | "failed" | "exhausted";
  attempt_count: number;
  next_attempt_at: string;
  last_status: number | null;
  last_error: string | null;
  delivered_at: string | null;
  created_at: string;
}

interface DeliveriesPage {
  deliveries: Delivery[];
  has_more: boolean;
  total: number;
}

const STATUS_COLORS: Record<string, string> = {
  pending:   "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
  success:   "bg-green-500/20 text-green-400 border-green-500/30",
  failed:    "bg-red-500/20 text-red-400 border-red-500/30",
  exhausted: "bg-slate-500/20 text-slate-400 border-slate-500/30",
};

// ─── Webhook form (create / edit) ─────────────────────────────────────────────

function WebhookForm({
  initial,
  allEventTypes,
  onClose,
  onSuccess,
}: {
  initial?: Webhook;
  allEventTypes: string[];
  onClose: () => void;
  onSuccess: () => void;
}) {
  const [name, setName]     = useState(initial?.name ?? "");
  const [url, setUrl]       = useState(initial?.url ?? "");
  const [secret, setSecret] = useState(initial?.signing_secret ?? "");
  const [showSecret, setShowSecret] = useState(false);
  const [selectedTypes, setSelectedTypes] = useState<string[]>(initial?.event_types ?? []);
  const [isActive, setIsActive] = useState(initial?.is_active ?? true);
  const [error, setError]   = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: () => {
      const body = {
        name,
        url,
        signing_secret: secret || undefined,
        event_types: selectedTypes,
        is_active: isActive,
      };
      return initial
        ? api.put<Webhook>(`/api/v1/webhooks/${initial.id}`, body)
        : api.post<Webhook>("/api/v1/webhooks", body);
    },
    onSuccess: () => { onSuccess(); onClose(); },
    onError: (e) => setError(e instanceof APIError ? e.message : "Failed to save webhook"),
  });

  const toggleType = (t: string) =>
    setSelectedTypes((prev) =>
      prev.includes(t) ? prev.filter((x) => x !== t) : [...prev, t]
    );

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
      <div className="bg-slate-800 border border-slate-700 rounded-xl p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <h2 className="text-slate-100 font-semibold mb-4">
          {initial ? "Edit Webhook" : "New Webhook"}
        </h2>
        {error && <p className="text-red-400 text-sm mb-3">{error}</p>}

        <div className="space-y-4">
          {/* Name */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">Name</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Core Banking Alert Relay"
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:border-cyan-500"
            />
          </div>

          {/* URL */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">
              Endpoint URL <span className="text-slate-600">(must be https://)</span>
            </label>
            <input
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://your-system.example.com/opsowl-events"
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm font-mono text-cyan-400 focus:outline-none focus:border-cyan-500"
            />
          </div>

          {/* Signing secret */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">
              Signing Secret{" "}
              <span className="text-slate-600">
                {initial ? "(leave blank to keep existing)" : "(leave blank to auto-generate)"}
              </span>
            </label>
            <div className="flex gap-2">
              <input
                type={showSecret ? "text" : "password"}
                value={secret}
                onChange={(e) => setSecret(e.target.value)}
                placeholder={initial ? "••••••••••••••••" : "min 16 characters"}
                className="flex-1 bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm font-mono text-slate-100 focus:outline-none focus:border-cyan-500"
              />
              <button
                type="button"
                onClick={() => setShowSecret((v) => !v)}
                className="px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-300 text-xs rounded transition-colors"
              >
                {showSecret ? "Hide" : "Show"}
              </button>
            </div>
          </div>

          {/* Event types */}
          <div>
            <label className="block text-xs text-slate-400 mb-2">
              Event Types{" "}
              <span className="text-slate-600">(leave all unchecked to subscribe to everything)</span>
            </label>
            <div className="grid grid-cols-2 gap-1.5">
              {allEventTypes.map((t) => (
                <label
                  key={t}
                  className={`flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer text-xs transition-colors ${
                    selectedTypes.includes(t)
                      ? "bg-cyan-500/15 text-cyan-300 border border-cyan-500/30"
                      : "bg-slate-700/50 text-slate-400 border border-slate-600/50 hover:bg-slate-700"
                  }`}
                >
                  <input
                    type="checkbox"
                    className="accent-cyan-500"
                    checked={selectedTypes.includes(t)}
                    onChange={() => toggleType(t)}
                  />
                  <span className="font-mono">{t}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Active toggle */}
          <label className="flex items-center gap-3 cursor-pointer">
            <div
              onClick={() => setIsActive((v) => !v)}
              className={`relative w-10 h-5 rounded-full transition-colors ${isActive ? "bg-cyan-600" : "bg-slate-600"}`}
            >
              <span
                className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${
                  isActive ? "translate-x-5" : ""
                }`}
              />
            </div>
            <span className="text-sm text-slate-300">{isActive ? "Active" : "Inactive"}</span>
          </label>
        </div>

        <div className="flex gap-3 mt-6">
          <button
            onClick={() => mutation.mutate()}
            disabled={!name || !url || mutation.isPending}
            className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm rounded transition-colors"
          >
            {mutation.isPending ? "Saving…" : initial ? "Save Changes" : "Create Webhook"}
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Delivery log panel ───────────────────────────────────────────────────────

function DeliveryLog({ webhook }: { webhook: Webhook }) {
  const qc = useQueryClient();
  const [offset, setOffset] = useState(0);
  const limit = 20;

  const { data, isLoading } = useQuery<DeliveriesPage>({
    queryKey: ["webhook-deliveries", webhook.id, offset],
    queryFn: () =>
      api.get(`/api/v1/webhooks/${webhook.id}/deliveries?limit=${limit}&offset=${offset}`),
    refetchInterval: 15_000,
  });

  const redeliver = useMutation({
    mutationFn: (deliveryId: string) =>
      api.post(`/api/v1/webhooks/${webhook.id}/deliveries/${deliveryId}/redeliver`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["webhook-deliveries", webhook.id] }),
  });

  if (isLoading) {
    return (
      <div className="mt-3 space-y-2">
        {[...Array(3)].map((_, i) => (
          <div key={i} className="h-10 bg-slate-700/40 rounded animate-pulse" />
        ))}
      </div>
    );
  }

  const deliveries = data?.deliveries ?? [];

  if (deliveries.length === 0) {
    return (
      <p className="mt-3 text-sm text-slate-500 italic">No deliveries yet.</p>
    );
  }

  return (
    <div className="mt-3">
      <div className="overflow-x-auto">
        <table className="w-full text-xs">
          <thead>
            <tr className="text-slate-500 border-b border-slate-700">
              <th className="text-left py-1.5 pr-3 font-medium">Event</th>
              <th className="text-left py-1.5 pr-3 font-medium">Status</th>
              <th className="text-left py-1.5 pr-3 font-medium">HTTP</th>
              <th className="text-left py-1.5 pr-3 font-medium">Attempts</th>
              <th className="text-left py-1.5 pr-3 font-medium">Time</th>
              <th className="text-left py-1.5 font-medium">Action</th>
            </tr>
          </thead>
          <tbody>
            {deliveries.map((d) => (
              <tr key={d.id} className="border-b border-slate-800 hover:bg-slate-800/40">
                <td className="py-1.5 pr-3 font-mono text-slate-300">{d.event_type}</td>
                <td className="py-1.5 pr-3">
                  <span className={`px-1.5 py-0.5 rounded border text-xs ${STATUS_COLORS[d.status] ?? ""}`}>
                    {d.status}
                  </span>
                </td>
                <td className="py-1.5 pr-3 text-slate-400">{d.last_status ?? "—"}</td>
                <td className="py-1.5 pr-3 text-slate-400">{d.attempt_count}</td>
                <td className="py-1.5 pr-3 text-slate-500">
                  {new Date(d.created_at).toLocaleString()}
                </td>
                <td className="py-1.5">
                  {(d.status === "failed" || d.status === "exhausted") && (
                    <button
                      onClick={() => redeliver.mutate(d.id)}
                      disabled={redeliver.isPending}
                      className="px-2 py-0.5 bg-slate-700 hover:bg-slate-600 text-slate-300 rounded text-xs transition-colors disabled:opacity-50"
                    >
                      Retry
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <div className="flex items-center gap-3 mt-3">
        <button
          onClick={() => setOffset(Math.max(0, offset - limit))}
          disabled={offset === 0}
          className="px-3 py-1 bg-slate-700 hover:bg-slate-600 disabled:opacity-40 text-slate-300 text-xs rounded transition-colors"
        >
          Previous
        </button>
        <span className="text-xs text-slate-500">
          {offset + 1}–{Math.min(offset + limit, data?.total ?? 0)} of {data?.total ?? 0}
        </span>
        <button
          onClick={() => setOffset(offset + limit)}
          disabled={!data?.has_more}
          className="px-3 py-1 bg-slate-700 hover:bg-slate-600 disabled:opacity-40 text-slate-300 text-xs rounded transition-colors"
        >
          Next
        </button>
      </div>
    </div>
  );
}

// ─── Webhook row ──────────────────────────────────────────────────────────────

function WebhookRow({
  hook,
  allEventTypes,
  onDeleted,
}: {
  hook: Webhook;
  allEventTypes: string[];
  onDeleted: () => void;
}) {
  const qc = useQueryClient();
  const [expanded, setExpanded] = useState(false);
  const [editing, setEditing]   = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const deleteMutation = useMutation({
    mutationFn: () => api.delete(`/api/v1/webhooks/${hook.id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["webhooks"] }); onDeleted(); },
  });

  const toggleMutation = useMutation({
    mutationFn: () =>
      api.put(`/api/v1/webhooks/${hook.id}`, {
        name: hook.name,
        url: hook.url,
        signing_secret: hook.signing_secret ?? "",
        event_types: hook.event_types,
        is_active: !hook.is_active,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["webhooks"] }),
  });

  return (
    <>
      {editing && (
        <WebhookForm
          initial={hook}
          allEventTypes={allEventTypes}
          onClose={() => setEditing(false)}
          onSuccess={() => qc.invalidateQueries({ queryKey: ["webhooks"] })}
        />
      )}

      <div className="bg-slate-800/60 border border-slate-700 rounded-lg p-4">
        {/* Header row */}
        <div className="flex items-start gap-3">
          {/* Active indicator */}
          <button
            onClick={() => toggleMutation.mutate()}
            disabled={toggleMutation.isPending}
            title={hook.is_active ? "Disable" : "Enable"}
            className={`mt-0.5 w-2.5 h-2.5 rounded-full flex-shrink-0 transition-colors ${
              hook.is_active ? "bg-green-400" : "bg-slate-600"
            }`}
          />

          {/* Info */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-slate-100 font-medium text-sm">{hook.name}</span>
              {!hook.is_active && (
                <span className="text-xs text-slate-500 bg-slate-700 px-1.5 py-0.5 rounded">disabled</span>
              )}
            </div>
            <p className="text-slate-400 text-xs font-mono mt-0.5 truncate">{hook.url}</p>
            {hook.event_types.length > 0 ? (
              <div className="flex flex-wrap gap-1 mt-1.5">
                {hook.event_types.map((t) => (
                  <span key={t} className="text-xs font-mono px-1.5 py-0.5 bg-slate-700 text-slate-400 rounded">
                    {t}
                  </span>
                ))}
              </div>
            ) : (
              <p className="text-xs text-slate-500 mt-1">Subscribed to all events</p>
            )}
          </div>

          {/* Actions */}
          <div className="flex items-center gap-2 flex-shrink-0">
            <button
              onClick={() => setExpanded((v) => !v)}
              className="text-xs text-slate-400 hover:text-slate-200 transition-colors"
            >
              {expanded ? "Hide log" : "View log"}
            </button>
            <button
              onClick={() => setEditing(true)}
              className="text-xs text-cyan-400 hover:text-cyan-300 transition-colors"
            >
              Edit
            </button>
            {confirmDelete ? (
              <div className="flex items-center gap-1">
                <button onClick={() => deleteMutation.mutate()} disabled={deleteMutation.isPending} className="text-xs text-white bg-red-600 hover:bg-red-500 px-2 py-0.5 rounded transition-colors disabled:opacity-50">Confirm</button>
                <button onClick={() => setConfirmDelete(false)} className="text-xs text-slate-400 hover:text-slate-200 px-2 py-0.5 rounded border border-slate-700 transition-colors">Cancel</button>
              </div>
            ) : (
              <button onClick={() => setConfirmDelete(true)} className="text-xs text-red-400 hover:text-red-300 transition-colors">Delete</button>
            )}
          </div>
        </div>

        {/* Delivery log */}
        {expanded && <DeliveryLog webhook={hook} />}
      </div>
    </>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function WebhooksPage() {
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);

  const { data: hooksData, isLoading: hooksLoading } = useQuery<{ webhooks: Webhook[] }>({
    queryKey: ["webhooks"],
    queryFn: () => api.get("/api/v1/webhooks/"),
  });

  const { data: typesData } = useQuery<{ event_types: string[] }>({
    queryKey: ["webhook-event-types"],
    queryFn: () => api.get("/api/v1/webhooks/event-types"),
    staleTime: Infinity,
  });

  const hooks = hooksData?.webhooks ?? [];
  const allEventTypes = typesData?.event_types ?? [];

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold text-slate-100">Outbound Webhooks</h1>
          <p className="text-sm text-slate-400 mt-0.5">
            Push case lifecycle events to external systems. Each delivery is HMAC-SHA256 signed.
          </p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium rounded-lg transition-colors"
        >
          + Add Webhook
        </button>
      </div>

      {/* New webhook form modal */}
      {showForm && (
        <WebhookForm
          allEventTypes={allEventTypes}
          onClose={() => setShowForm(false)}
          onSuccess={() => qc.invalidateQueries({ queryKey: ["webhooks"] })}
        />
      )}

      {/* Signing note */}
      <div className="bg-slate-800/40 border border-slate-700 rounded-lg p-4 text-sm text-slate-400">
        <p className="font-medium text-slate-300 mb-1">Verifying deliveries</p>
        <p>
          Each request includes an{" "}
          <code className="text-cyan-400 bg-slate-700 px-1 rounded">X-OpsOwl-Signature</code> header
          containing{" "}
          <code className="text-cyan-400 bg-slate-700 px-1 rounded">sha256=&lt;hmac-hex&gt;</code>.
          Compute{" "}
          <code className="text-cyan-400 bg-slate-700 px-1 rounded">HMAC-SHA256(secret, body)</code>{" "}
          on your end and compare to verify authenticity. Deliveries are retried up to 5 times with
          exponential backoff (30 s → 5 m → 30 m → 2 h).
        </p>
      </div>

      {/* Webhook list */}
      {hooksLoading ? (
        <div className="space-y-3">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="h-20 bg-slate-800/40 rounded-lg animate-pulse" />
          ))}
        </div>
      ) : hooks.length === 0 ? (
        <div className="text-center py-16 text-slate-500">
          <p className="text-4xl mb-3">🔗</p>
          <p className="font-medium">No webhooks configured</p>
          <p className="text-sm mt-1">Add one to start receiving case lifecycle events.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {hooks.map((h) => (
            <WebhookRow
              key={h.id}
              hook={h}
              allEventTypes={allEventTypes}
              onDeleted={() => qc.invalidateQueries({ queryKey: ["webhooks"] })}
            />
          ))}
        </div>
      )}
    </div>
  );
}
