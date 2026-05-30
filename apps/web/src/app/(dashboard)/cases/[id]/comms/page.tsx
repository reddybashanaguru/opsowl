"use client";

import { use, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { api, APIError } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { CaseSubNav } from "@/components/cases/CaseSubNav";
import { PageError } from "@/components/ui/PageError";

interface CommLog {
  id: string;
  case_id: string;
  channel: string;
  direction: string;
  contact_name: string;
  contact_detail: string | null;
  subject: string | null;
  body: string;
  outcome: string | null;
  occurred_at: string;
  created_at: string;
  logged_by: { id: string; name: string; email: string };
  ai_extraction?: {
    entities?: Record<string, string[]>;
    action_items?: string[];
    sentiment?: string;
    model_used?: string;
  };
}

const CHANNEL_ICONS: Record<string, string> = {
  email: "📧", call: "📞", field_visit: "🚗", letter: "📬",
  whatsapp: "💬", sms: "📱", other: "📋",
};

const CHANNEL_OPTIONS = ["email", "call", "field_visit", "letter", "whatsapp", "sms", "other"];
const DIRECTION_OPTIONS = ["inbound", "outbound"];

const SENTIMENT_COLORS: Record<string, string> = {
  positive: "text-emerald-400 bg-emerald-500/10",
  neutral: "text-slate-400 bg-slate-700/50",
  negative: "text-amber-400 bg-amber-500/10",
  hostile: "text-red-400 bg-red-500/10",
};

function timeAgo(iso: string) {
  const diff = Date.now() - new Date(iso).getTime();
  const m = Math.floor(diff / 60000);
  if (m < 1) return "just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

function CommCard({ log, onDelete, canDelete }: { log: CommLog; onDelete: (id: string) => void; canDelete: boolean }) {
  const [expanded, setExpanded] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  return (
    <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4 hover:border-slate-600 transition-colors">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <span className="text-xl flex-shrink-0">{CHANNEL_ICONS[log.channel] ?? "📋"}</span>
          <div className="min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-white text-sm font-semibold truncate">{log.contact_name}</span>
              {log.contact_detail && (
                <span className="text-slate-500 text-xs">{log.contact_detail}</span>
              )}
              <span className={`text-xs px-1.5 py-0.5 rounded font-medium capitalize ${
                log.direction === "inbound"
                  ? "bg-blue-500/20 text-blue-300"
                  : "bg-purple-500/20 text-purple-300"
              }`}>
                {log.direction}
              </span>
              <span className="text-slate-600 text-xs capitalize">{log.channel.replace("_", " ")}</span>
            </div>
            {log.subject && (
              <p className="text-slate-400 text-xs mt-0.5 truncate">{log.subject}</p>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2 flex-shrink-0">
          {log.ai_extraction?.sentiment && (
            <span className={`text-xs px-2 py-0.5 rounded capitalize ${SENTIMENT_COLORS[log.ai_extraction.sentiment] ?? SENTIMENT_COLORS.neutral}`}>
              {log.ai_extraction.sentiment}
            </span>
          )}
          <span className="text-slate-500 text-xs">{timeAgo(log.occurred_at)}</span>
          {canDelete && (
            confirmDelete ? (
              <div className="flex items-center gap-1">
                <button onClick={() => { onDelete(log.id); setConfirmDelete(false); }} className="text-[10px] text-white bg-red-600 hover:bg-red-500 px-2 py-0.5 rounded transition-colors">Delete</button>
                <button onClick={() => setConfirmDelete(false)} className="text-[10px] text-slate-400 hover:text-slate-200 px-1.5 py-0.5 rounded border border-slate-700 transition-colors">✕</button>
              </div>
            ) : (
              <button
                onClick={() => setConfirmDelete(true)}
                className="text-slate-600 hover:text-red-400 text-xs transition-colors"
                title="Delete"
              >
                ✕
              </button>
            )
          )}
        </div>
      </div>

      {/* Body preview */}
      <div className="mt-3 ml-9">
        <p className={`text-slate-300 text-sm ${expanded ? "" : "line-clamp-2"}`}>{log.body}</p>
        {log.body.length > 120 && (
          <button
            onClick={() => setExpanded(!expanded)}
            className="text-cyan-400 text-xs mt-1 hover:text-cyan-300"
          >
            {expanded ? "Show less" : "Show more"}
          </button>
        )}
        {log.outcome && (
          <p className="text-slate-500 text-xs mt-2 italic">Outcome: {log.outcome}</p>
        )}
      </div>

      {/* AI Extraction */}
      {log.ai_extraction && (log.ai_extraction.action_items?.length || Object.keys(log.ai_extraction.entities ?? {}).length > 0) && (
        <div className="mt-3 ml-9 pt-3 border-t border-slate-700/50">
          <div className="text-slate-500 text-xs uppercase tracking-wide mb-2">AI Extracted</div>
          {log.ai_extraction.action_items && log.ai_extraction.action_items.length > 0 && (
            <div className="mb-2">
              <span className="text-violet-400 text-xs font-medium">Action Items: </span>
              <span className="text-slate-400 text-xs">{log.ai_extraction.action_items.join(" · ")}</span>
            </div>
          )}
          {log.ai_extraction.entities && Object.keys(log.ai_extraction.entities).length > 0 && (
            <div className="flex gap-2 flex-wrap">
              {Object.entries(log.ai_extraction.entities).map(([type, items]) =>
                items.map((item: string, i: number) => (
                  <span key={`${type}-${i}`} className="bg-slate-700 text-slate-300 text-xs px-2 py-0.5 rounded">
                    {item}
                  </span>
                ))
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function CommsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const hasRole = useAuthStore((s) => s.hasRole);

  const canWrite = hasRole("analyst");

  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    channel: "email",
    direction: "inbound",
    contact_name: "",
    contact_detail: "",
    subject: "",
    body: "",
    outcome: "",
    occurred_at: new Date().toISOString().slice(0, 16),
  });
  const [formError, setFormError] = useState("");

  const { data: logs, isLoading, isError, error, refetch: refetchLogs } = useQuery<CommLog[]>({
    queryKey: ["comms", id],
    queryFn: () => api.get<CommLog[]>(`/api/v1/cases/${id}/comms?limit=50`),
    refetchInterval: 10000,
    retry: (failureCount, err) => {
      if (err instanceof APIError && err.status === 404) return false;
      return failureCount < 2;
    },
  });

  const createMutation = useMutation({
    mutationFn: (body: typeof form) =>
      api.post(`/api/v1/cases/${id}/comms`, {
        ...body,
        occurred_at: new Date(body.occurred_at).toISOString(),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["comms", id] });
      setShowForm(false);
      setForm({ channel: "email", direction: "inbound", contact_name: "", contact_detail: "", subject: "", body: "", outcome: "", occurred_at: new Date().toISOString().slice(0, 16) });
      setFormError("");
    },
    onError: (e: Error) => setFormError(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (commId: string) => api.delete(`/api/v1/cases/${id}/comms/${commId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["comms", id] }),
  });

  const handleSubmit = () => {
    if (!form.contact_name.trim() || !form.body.trim()) {
      setFormError("Contact name and body are required.");
      return;
    }
    createMutation.mutate(form);
  };

  return (
    <div className="p-6">
      <CaseSubNav
        caseId={id}
        activeTab="comms"
        action={canWrite ? (
          <button
            onClick={() => setShowForm(!showForm)}
            className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 text-sm font-semibold px-4 py-2 rounded-lg transition-colors"
          >
            + Log Communication
          </button>
        ) : undefined}
      />

      {/* Header */}
      <div className="mb-6">
        <h1 className="text-xl font-bold text-white">Communications</h1>
        <p className="text-slate-400 text-sm mt-1">All contact logs for this case.</p>
      </div>

      {/* Log form */}
      {showForm && (
        <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5 mb-6">
          <h3 className="text-white font-semibold mb-4">New Communication Log</h3>
          <div className="grid grid-cols-2 gap-4 mb-4">
            <div>
              <label className="text-slate-400 text-xs mb-1 block">Channel</label>
              <select
                value={form.channel}
                onChange={(e) => setForm({ ...form, channel: e.target.value })}
                className="w-full bg-[#0f172a] border border-slate-600 text-white text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
              >
                {CHANNEL_OPTIONS.map((c) => (
                  <option key={c} value={c}>{c.replace("_", " ")}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-slate-400 text-xs mb-1 block">Direction</label>
              <select
                value={form.direction}
                onChange={(e) => setForm({ ...form, direction: e.target.value })}
                className="w-full bg-[#0f172a] border border-slate-600 text-white text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
              >
                {DIRECTION_OPTIONS.map((d) => (
                  <option key={d} value={d}>{d}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-slate-400 text-xs mb-1 block">Contact Name *</label>
              <input
                value={form.contact_name}
                onChange={(e) => setForm({ ...form, contact_name: e.target.value })}
                placeholder="John Doe"
                className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
              />
            </div>
            <div>
              <label className="text-slate-400 text-xs mb-1 block">Contact Detail (email/phone)</label>
              <input
                value={form.contact_detail}
                onChange={(e) => setForm({ ...form, contact_detail: e.target.value })}
                placeholder="john@example.com"
                className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
              />
            </div>
            <div>
              <label className="text-slate-400 text-xs mb-1 block">Subject</label>
              <input
                value={form.subject}
                onChange={(e) => setForm({ ...form, subject: e.target.value })}
                placeholder="Regarding account verification"
                className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
              />
            </div>
            <div>
              <label className="text-slate-400 text-xs mb-1 block">Occurred At</label>
              <input
                type="datetime-local"
                value={form.occurred_at}
                onChange={(e) => setForm({ ...form, occurred_at: e.target.value })}
                className="w-full bg-[#0f172a] border border-slate-600 text-white text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
              />
            </div>
          </div>
          <div className="mb-3">
            <label className="text-slate-400 text-xs mb-1 block">Body *</label>
            <textarea
              value={form.body}
              onChange={(e) => setForm({ ...form, body: e.target.value })}
              rows={4}
              placeholder="Describe what was communicated..."
              className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500 resize-none"
            />
          </div>
          <div className="mb-4">
            <label className="text-slate-400 text-xs mb-1 block">Outcome</label>
            <input
              value={form.outcome}
              onChange={(e) => setForm({ ...form, outcome: e.target.value })}
              placeholder="e.g. Customer agreed to submit documents by Friday"
              className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
            />
          </div>
          {formError && <p className="text-red-400 text-xs mb-3">{formError}</p>}
          <div className="flex gap-2">
            <button
              onClick={handleSubmit}
              disabled={createMutation.isPending}
              className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-2 rounded-lg text-sm disabled:opacity-60"
            >
              {createMutation.isPending ? "Saving…" : "Save Log"}
            </button>
            <button
              onClick={() => { setShowForm(false); setFormError(""); }}
              className="text-slate-400 hover:text-slate-200 px-4 py-2 text-sm"
            >
              Cancel
            </button>
          </div>
          <p className="text-violet-400 text-xs mt-3">
            AI will automatically extract entities and action items in the background.
          </p>
        </div>
      )}

      {/* Log list */}
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="bg-[#1e293b] border border-slate-700/50 rounded-xl p-4 animate-pulse">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-8 h-8 rounded-lg bg-slate-700" />
                <div className="flex-1 space-y-1.5">
                  <div className="h-3.5 bg-slate-700 rounded w-1/3" />
                  <div className="h-2.5 bg-slate-700/60 rounded w-1/5" />
                </div>
                <div className="h-2.5 bg-slate-700/40 rounded w-12" />
              </div>
              <div className="space-y-1.5">
                <div className="h-2.5 bg-slate-700/40 rounded w-full" />
                <div className="h-2.5 bg-slate-700/40 rounded w-4/5" />
              </div>
            </div>
          ))}
        </div>
      ) : isError ? (
        <PageError error={error as Error | null} onRetry={() => refetchLogs()} />
      ) : !logs || logs.length === 0 ? (
        <div className="bg-[#1e293b] border border-slate-700/50 rounded-xl p-16 text-center">
          <p className="text-4xl mb-3">📋</p>
          <p className="text-slate-400 text-sm font-medium">No communication logs yet.</p>
          <p className="text-slate-600 text-xs mt-1">Record calls, emails, and field visits with AI-powered extraction.</p>
          {canWrite && (
            <button
              onClick={() => setShowForm(true)}
              className="mt-4 text-cyan-400 text-sm hover:text-cyan-300"
            >
              Log the first communication →
            </button>
          )}
        </div>
      ) : (
        <div className="space-y-3">
          <p className="text-slate-500 text-xs">{logs.length} communication{logs.length !== 1 ? "s" : ""} logged</p>
          {logs.map((log) => (
            <CommCard
              key={log.id}
              log={log}
              canDelete={canWrite && log.logged_by.id === user?.id}
              onDelete={(commId) => deleteMutation.mutate(commId)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
