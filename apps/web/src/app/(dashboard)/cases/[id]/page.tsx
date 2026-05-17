"use client";

import { use, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { api, APIError } from "@/lib/api";
import { StatusBadge } from "@/components/cases/StatusBadge";
import { PriorityBadge } from "@/components/cases/PriorityBadge";
import { SLACountdown } from "@/components/cases/SLACountdown";
import { useAuthStore } from "@/lib/stores/auth.store";
import { CopilotPanel } from "@/components/ai/CopilotPanel";

interface CaseDetail {
  id: string;
  case_number: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  assigned_to: { id: string; full_name: string; role: string } | null;
  created_by: { id: string; full_name: string };
  sla_breach_at: string | null;
  sla_breached: boolean;
  due_date: string | null;
  tags: string[];
  watchers: { id: string; full_name: string }[] | null;
  created_at: string;
  updated_at: string;
}

interface Activity {
  id: string;
  event_type: string;
  actor: { id: string; full_name: string };
  payload: Record<string, string>;
  created_at: string;
}

const VALID_TRANSITIONS: Record<string, string[]> = {
  draft:          ["open"],
  open:           ["in_progress", "escalated"],
  in_progress:    ["pending_review", "escalated"],
  pending_review: ["closed", "in_progress", "escalated"],
  escalated:      ["in_progress"],
  closed:         [],
};

const STATUS_LABELS: Record<string, string> = {
  open: "Open", in_progress: "In Progress", pending_review: "Pending Review",
  closed: "Closed", escalated: "Escalated", draft: "Draft",
};

function ActivityItem({ a }: { a: Activity }) {
  const label: Record<string, string> = {
    case_created:    "created this case",
    case_updated:    "updated this case",
    status_changed:  `changed status from ${a.payload.from} to ${a.payload.to}`,
    assigned:        "assigned this case",
    tags_updated:    "updated tags",
    case_deleted:    "deleted this case",
  };
  return (
    <div className="flex gap-3 py-3 border-b border-slate-700/50 last:border-0">
      <div className="w-7 h-7 rounded-full bg-slate-700 flex items-center justify-center text-xs text-slate-300 flex-shrink-0 mt-0.5">
        {(a.actor?.full_name ?? "?").charAt(0).toUpperCase()}
      </div>
      <div>
        <span className="text-white text-sm font-medium">{a.actor.full_name}</span>
        <span className="text-slate-400 text-sm"> {label[a.event_type] ?? a.event_type}</span>
        {a.payload.reason && (
          <p className="text-slate-500 text-xs mt-0.5 italic">"{a.payload.reason}"</p>
        )}
        <p className="text-slate-600 text-xs mt-0.5">{new Date(a.created_at).toLocaleString()}</p>
      </div>
    </div>
  );
}

export default function CaseDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);

  const [showStatusModal, setShowStatusModal] = useState(false);
  const [targetStatus, setTargetStatus] = useState("");
  const [reason, setReason] = useState("");
  const [statusError, setStatusError] = useState("");
  const [copilotOpen, setCopilotOpen] = useState(false);

  const { data: c, isLoading } = useQuery<CaseDetail>({
    queryKey: ["case", id],
    queryFn: () => api.get<CaseDetail>(`/api/v1/cases/${id}`),
  });

  const { data: activityData } = useQuery<{ data: Activity[] }>({
    queryKey: ["case-activity", id],
    queryFn: () => api.getList<Activity>(`/api/v1/cases/${id}/activity?limit=20`),
    enabled: !!id,
  });

  const statusMutation = useMutation({
    mutationFn: ({ status, reason }: { status: string; reason: string }) =>
      api.post(`/api/v1/cases/${id}/status`, { status, reason }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["case", id] });
      qc.invalidateQueries({ queryKey: ["case-activity", id] });
      setShowStatusModal(false);
      setReason("");
      setStatusError("");
    },
    onError: (err) => {
      setStatusError(err instanceof APIError ? err.message : "Failed to update status");
    },
  });

  if (isLoading || !c) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-6 h-6 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  const nextStatuses = VALID_TRANSITIONS[c.status] ?? [];

  return (
    <div className="p-6 max-w-6xl mx-auto">
      {/* Nav */}
      <div className="flex items-center gap-4 mb-4">
        <button
          onClick={() => router.push("/cases")}
          className="text-slate-400 hover:text-slate-200 text-sm flex items-center gap-1"
        >
          ← Cases
        </button>
        <div className="h-4 border-l border-slate-700" />
        <button
          onClick={() => router.push(`/cases/${id}/tasks`)}
          className="text-slate-400 hover:text-cyan-400 text-sm font-medium transition-colors"
        >
          Tasks →
        </button>
        <div className="h-4 border-l border-slate-700" />
        <button
          onClick={() => router.push(`/cases/${id}/documents`)}
          className="text-slate-400 hover:text-cyan-400 text-sm font-medium transition-colors"
        >
          Documents →
        </button>
        <div className="h-4 border-l border-slate-700" />
        <button
          onClick={() => router.push(`/cases/${id}/timeline`)}
          className="text-slate-400 hover:text-cyan-400 text-sm font-medium transition-colors"
        >
          Timeline →
        </button>
        <div className="h-4 border-l border-slate-700" />
        <button
          onClick={() => router.push(`/cases/${id}/ai-logs`)}
          className="text-slate-400 hover:text-cyan-400 text-sm font-medium transition-colors"
        >
          AI Logs →
        </button>
        <div className="h-4 border-l border-slate-700" />
        <button
          onClick={() => setCopilotOpen(true)}
          className="bg-violet-600/20 hover:bg-violet-600/30 text-violet-300 hover:text-violet-200 border border-violet-500/30 text-sm font-medium px-3 py-1 rounded-lg transition-colors"
          title="AI Copilot (⌘⇧A)"
        >
          🤖 AI Copilot
        </button>
      </div>

      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <div className="font-mono text-sm text-cyan-400 mb-1">{c.case_number}</div>
          <h1 className="text-white text-2xl font-bold">{c.title}</h1>
          <div className="flex items-center gap-3 mt-2">
            <StatusBadge status={c.status} size="md" />
            <PriorityBadge priority={c.priority} />
            <SLACountdown breachAt={c.sla_breach_at} breached={c.sla_breached} />
          </div>
        </div>

        {/* Status actions */}
        {nextStatuses.length > 0 && (
          <div className="flex gap-2">
            {nextStatuses.map((s) => (
              <button
                key={s}
                onClick={() => { setTargetStatus(s); setShowStatusModal(true); }}
                className="bg-slate-700 hover:bg-slate-600 text-slate-200 text-xs font-medium px-3 py-1.5 rounded-lg transition-colors"
              >
                → {STATUS_LABELS[s] ?? s}
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="grid grid-cols-3 gap-6">
        {/* Main */}
        <div className="col-span-2 space-y-6">
          {/* Description */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5">
            <h2 className="text-slate-300 text-sm font-semibold mb-3 uppercase tracking-wide">Description</h2>
            <p className="text-slate-300 text-sm whitespace-pre-wrap">
              {c.description || <span className="text-slate-600 italic">No description provided.</span>}
            </p>
          </div>

          {/* Activity */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5">
            <h2 className="text-slate-300 text-sm font-semibold mb-4 uppercase tracking-wide">Activity</h2>
            {!activityData || activityData.data.length === 0 ? (
              <p className="text-slate-600 text-sm italic">No activity yet.</p>
            ) : (
              <div>
                {activityData.data.map((a) => <ActivityItem key={a.id} a={a} />)}
              </div>
            )}
          </div>
        </div>

        {/* Sidebar */}
        <div className="space-y-4">
          {/* Details card */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5 space-y-4">
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wide mb-1">Assigned To</div>
              <div className="text-slate-200 text-sm">
                {c.assigned_to?.full_name ?? <span className="text-slate-600">Unassigned</span>}
              </div>
            </div>
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wide mb-1">Created By</div>
              <div className="text-slate-200 text-sm">{c.created_by?.full_name || "—"}</div>
            </div>
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wide mb-1">Created</div>
              <div className="text-slate-200 text-sm">{new Date(c.created_at).toLocaleString()}</div>
            </div>
            {c.due_date && (
              <div>
                <div className="text-slate-500 text-xs uppercase tracking-wide mb-1">Due Date</div>
                <div className="text-slate-200 text-sm">{new Date(c.due_date).toLocaleDateString()}</div>
              </div>
            )}
            {c.sla_breach_at && (
              <div>
                <div className="text-slate-500 text-xs uppercase tracking-wide mb-1">SLA Deadline</div>
                <div className="text-slate-200 text-sm">{new Date(c.sla_breach_at).toLocaleString()}</div>
              </div>
            )}
          </div>

          {/* Tags */}
          {(c.tags?.length ?? 0) > 0 && (
            <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5">
              <div className="text-slate-500 text-xs uppercase tracking-wide mb-3">Tags</div>
              <div className="flex flex-wrap gap-1.5">
                {(c.tags ?? []).map((t) => (
                  <span key={t} className="bg-slate-700 text-slate-300 text-xs px-2 py-1 rounded">
                    {t}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Watchers */}
          {(c.watchers?.length ?? 0) > 0 && (
            <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5">
              <div className="text-slate-500 text-xs uppercase tracking-wide mb-3">Watchers</div>
              <div className="space-y-1">
                {(c.watchers ?? []).map((w) => (
                  <div key={w.id} className="text-slate-300 text-sm">{w.full_name}</div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Status transition modal */}
      {showStatusModal && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-6 w-full max-w-md">
            <h3 className="text-white font-semibold mb-1">
              Move to <span className="text-cyan-400">{STATUS_LABELS[targetStatus]}</span>
            </h3>
            <p className="text-slate-400 text-sm mb-4">
              {["in_progress", "closed", "escalated"].includes(targetStatus)
                ? "Add an optional note for the activity log."
                : "A reason is required for this transition."}
            </p>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={3}
              placeholder="Reason or note..."
              className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500 resize-none mb-3"
            />
            {statusError && (
              <p className="text-red-400 text-xs mb-3">{statusError}</p>
            )}
            <div className="flex gap-2">
              <button
                onClick={() => statusMutation.mutate({ status: targetStatus, reason })}
                disabled={statusMutation.isPending}
                className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-2 rounded-lg text-sm disabled:opacity-60"
              >
                {statusMutation.isPending ? "Updating…" : "Confirm"}
              </button>
              <button
                onClick={() => { setShowStatusModal(false); setStatusError(""); setReason(""); }}
                className="text-slate-400 hover:text-slate-200 px-4 py-2 text-sm"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* AI Copilot drawer */}
      <CopilotPanel
        caseId={id}
        open={copilotOpen}
        onClose={() => setCopilotOpen(false)}
      />
    </div>
  );
}
