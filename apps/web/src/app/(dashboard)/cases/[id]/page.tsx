"use client";

import { use, useState, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { api, APIError } from "@/lib/api";
import { StatusBadge } from "@/components/cases/StatusBadge";
import { PriorityBadge } from "@/components/cases/PriorityBadge";
import { SLACountdown } from "@/components/cases/SLACountdown";
import { CaseSubNav } from "@/components/cases/CaseSubNav";
import { useAuthStore } from "@/lib/stores/auth.store";
import { CopilotPanel } from "@/components/ai/CopilotPanel";
import { useTimelineSSE, TimelineEvent } from "@/lib/hooks/useTimelineSSE";

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
  risk_score?: string | null;
  risk_level?: string | null;
  open_tasks_count?: number;
  documents_count?: number;
}

interface PreviewTask {
  id: string;
  title: string;
  status: string;
  priority: string;
  is_overdue: boolean;
}

interface PreviewDocument {
  id: string;
  original_filename: string;
  processing_status: string;
  file_size_bytes: number;
}

interface UserOption {
  id: string;
  full_name: string;
  role: string;
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

// Routine transitions fire immediately without a modal (low-consequence, reversible).
const ROUTINE_TRANSITIONS = new Set(["in_progress", "pending_review"]);

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

const RISK_COLORS: Record<string, string> = {
  high:    "text-red-400 bg-red-500/10 border-red-500/30",
  medium:  "text-amber-400 bg-amber-500/10 border-amber-500/30",
  low:     "text-emerald-400 bg-emerald-500/10 border-emerald-500/30",
};

function timelineLabel(event: TimelineEvent): string {
  const actor = event.actor?.full_name ?? "System";
  const p = event.payload as Record<string, string>;
  switch (event.event_type) {
    case "case_status_changed": return `${actor} moved to ${p.to ?? ""}`;
    case "note_created":        return `${actor} added a note`;
    case "task_created":        return `${actor} created task "${p.title ?? ""}"`;
    case "task_completed":      return `${actor} completed a task`;
    case "document_uploaded":   return `${actor} uploaded a document`;
    case "case_assigned":       return `${actor} reassigned the case`;
    case "case_tagged":         return `${actor} updated tags`;
    default:                    return `${actor}: ${event.event_type.replace(/_/g, " ")}`;
  }
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
  const [exportingPDF, setExportingPDF] = useState(false);
  const [noteText, setNoteText] = useState("");
  const [assigneeOpen, setAssigneeOpen] = useState(false);
  const [isAnalysing, setIsAnalysing] = useState(false);
  const analysingTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const assigneeRef = useRef<HTMLDivElement>(null);

  // Live timeline feed
  const [liveEvents, setLiveEvents] = useState<TimelineEvent[]>([]);
  const handleTimelineEvent = (event: TimelineEvent) => {
    setLiveEvents((prev) => [event, ...prev].slice(0, 20));
  };
  useTimelineSSE({ caseId: id, onEvent: handleTimelineEvent });

  const { data: c, isLoading } = useQuery<CaseDetail>({
    queryKey: ["case", id],
    queryFn: () => api.get<CaseDetail>(`/api/v1/cases/${id}`),
    // Poll at 3s while AI analysis is in-flight to pick up risk score quickly.
    refetchInterval: isAnalysing ? 3_000 : 30_000,
  });

  const { data: activityData } = useQuery<{ data: Activity[] }>({
    queryKey: ["case-activity", id],
    queryFn: () => api.getList<Activity>(`/api/v1/cases/${id}/activity?limit=20`),
    enabled: !!id,
  });

  const { data: tasksPreview } = useQuery<{ data: PreviewTask[] }>({
    queryKey: ["case-tasks-preview", id],
    queryFn: () => api.getList<PreviewTask>(`/api/v1/cases/${id}/tasks?limit=3`),
    enabled: !!id,
  });

  const { data: docsPreview } = useQuery<{ data: PreviewDocument[] }>({
    queryKey: ["case-docs-preview", id],
    queryFn: () => api.getList<PreviewDocument>(`/api/v1/cases/${id}/documents?limit=2`),
    enabled: !!id,
  });

  const { data: usersData } = useQuery<{ users: UserOption[] }>({
    queryKey: ["users-list"],
    queryFn: () => api.get<{ users: UserOption[] }>("/api/v1/users?limit=100"),
    staleTime: 60_000,
    enabled: assigneeOpen,
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

  const assignMutation = useMutation({
    mutationFn: (userId: string) =>
      api.patch(`/api/v1/cases/${id}`, { assigned_to: userId }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["case", id] });
      setAssigneeOpen(false);
    },
  });

  const noteMutation = useMutation({
    mutationFn: (content: string) =>
      api.post(`/api/v1/cases/${id}/notes`, {
        content: `<p>${content.replace(/\n/g, "</p><p>")}</p>`,
        content_text: content,
        note_type: "internal",
      }),
    onSuccess: () => {
      setNoteText("");
      qc.invalidateQueries({ queryKey: ["case-activity", id] });
    },
  });

  const analyseMutation = useMutation({
    mutationFn: () => api.post(`/api/v1/cases/${id}/insights/analyse`, {}),
    onSuccess: () => {
      // Analysis is async (202). Poll the case query at 3s intervals for up to
      // 30 seconds to pick up the updated risk badge once the worker finishes.
      setIsAnalysing(true);
      if (analysingTimer.current) clearTimeout(analysingTimer.current);
      analysingTimer.current = setTimeout(() => setIsAnalysing(false), 30_000);
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
      <CaseSubNav
        caseId={id}
        activeTab="case"
        action={
          <>
            <button
              onClick={() => analyseMutation.mutate()}
              disabled={analyseMutation.isPending || isAnalysing}
              className="bg-amber-600/15 hover:bg-amber-600/25 text-amber-300 border border-amber-500/30 text-sm font-medium px-3 py-1 rounded-lg transition-colors disabled:opacity-50 flex items-center gap-1.5"
              title={isAnalysing ? "AI analysis in progress…" : "Re-run AI Analysis"}
            >
              {(analyseMutation.isPending || isAnalysing) ? (
                <span className="w-3 h-3 border border-amber-300 border-t-transparent rounded-full animate-spin" />
              ) : "⚡"}
              {isAnalysing ? "Analysing…" : "Analyse"}
            </button>
            <button
              onClick={() => setCopilotOpen(true)}
              className="bg-violet-600/20 hover:bg-violet-600/30 text-violet-300 hover:text-violet-200 border border-violet-500/30 text-sm font-medium px-3 py-1 rounded-lg transition-colors"
              title="AI Copilot (⌘⇧A)"
            >
              🤖 AI Copilot
            </button>
            <button
              onClick={async () => {
                setExportingPDF(true);
                try {
                  const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
                  const res = await fetch(`${apiBase}/api/v1/cases/${id}/report`, {
                    credentials: "include",
                  });
                  if (!res.ok) throw new Error("Failed to generate report");
                  const blob = await res.blob();
                  const url = URL.createObjectURL(blob);
                  const a = document.createElement("a");
                  a.href = url;
                  a.download = `${c?.case_number ?? id}-report.pdf`;
                  document.body.appendChild(a);
                  a.click();
                  a.remove();
                  URL.revokeObjectURL(url);
                } finally {
                  setExportingPDF(false);
                }
              }}
              disabled={exportingPDF}
              className="bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-300 hover:text-emerald-200 border border-emerald-500/30 text-sm font-medium px-3 py-1 rounded-lg transition-colors disabled:opacity-60 flex items-center gap-1.5"
              title="Export PDF Report"
            >
              {exportingPDF ? (
                <>
                  <span className="w-3 h-3 border-2 border-emerald-400/30 border-t-emerald-400 rounded-full animate-spin" />
                  Generating…
                </>
              ) : "📄 Export PDF"}
            </button>
          </>
        }
      />

      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <div className="font-mono text-sm text-cyan-400 mb-1">{c.case_number}</div>
          <h1 className="text-white text-2xl font-bold">{c.title}</h1>
          <div className="flex items-center gap-3 mt-2 flex-wrap">
            <StatusBadge status={c.status} size="md" />
            <PriorityBadge priority={c.priority} />
            <SLACountdown breachAt={c.sla_breach_at} breached={c.sla_breached} />
            {c.risk_level && (
              <span className={`text-xs font-medium px-2 py-0.5 rounded border ${RISK_COLORS[c.risk_level] ?? "text-slate-400 bg-slate-700/40 border-slate-600/30"}`}>
                Risk: {c.risk_level}
                {c.risk_score ? ` (${parseFloat(c.risk_score).toFixed(0)})` : ""}
              </span>
            )}
            {(c.open_tasks_count !== undefined || c.documents_count !== undefined) && (
              <span className="text-slate-500 text-xs">
                {c.open_tasks_count ?? 0} tasks · {c.documents_count ?? 0} docs
              </span>
            )}
          </div>
        </div>

        {/* Status actions */}
        {nextStatuses.length > 0 && (
          <div className="flex gap-2 flex-wrap">
            {nextStatuses.map((s) => {
              const isRoutine = ROUTINE_TRANSITIONS.has(s);
              const isEscalate = s === "escalated";
              return (
                <button
                  key={s}
                  disabled={statusMutation.isPending}
                  onClick={() => {
                    if (isRoutine) {
                      statusMutation.mutate({ status: s, reason: "" });
                    } else {
                      setTargetStatus(s);
                      setShowStatusModal(true);
                    }
                  }}
                  className={`text-xs font-medium px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50 ${
                    isEscalate
                      ? "bg-red-600/20 hover:bg-red-600/30 text-red-300 border border-red-500/30"
                      : isRoutine
                      ? "bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-300 border border-emerald-500/30"
                      : "bg-slate-700 hover:bg-slate-600 text-slate-200"
                  }`}
                >
                  {statusMutation.isPending && isRoutine ? (
                    <span className="flex items-center gap-1">
                      <span className="w-2.5 h-2.5 border border-current border-t-transparent rounded-full animate-spin" />
                      {STATUS_LABELS[s] ?? s}
                    </span>
                  ) : (
                    `→ ${STATUS_LABELS[s] ?? s}`
                  )}
                </button>
              );
            })}
          </div>
        )}
      </div>

      <div className="grid grid-cols-3 gap-6">
        {/* Left column — Investigation Feed (65%) */}
        <div className="col-span-2 space-y-5">
          {/* Description */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5">
            <h2 className="text-slate-300 text-sm font-semibold mb-3 uppercase tracking-wide">Description</h2>
            <p className="text-slate-300 text-sm whitespace-pre-wrap">
              {c.description || <span className="text-slate-600 italic">No description provided.</span>}
            </p>
          </div>

          {/* Quick Note Input */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4">
            <div className="text-slate-500 text-xs uppercase tracking-wide mb-2">Quick Note</div>
            <textarea
              value={noteText}
              onChange={(e) => setNoteText(e.target.value)}
              placeholder="Add an internal note…"
              rows={3}
              className="w-full bg-slate-900/60 border border-slate-700 rounded-lg px-3 py-2 text-slate-300 text-sm placeholder:text-slate-600 focus:outline-none focus:border-cyan-500 resize-none"
            />
            <div className="flex justify-end mt-2">
              <button
                onClick={() => noteText.trim() && noteMutation.mutate(noteText.trim())}
                disabled={!noteText.trim() || noteMutation.isPending}
                className="bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-semibold px-4 py-1.5 rounded-lg transition-colors disabled:opacity-50"
              >
                {noteMutation.isPending ? "Saving…" : "Add Note"}
              </button>
            </div>
          </div>

          {/* Live Timeline Feed */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5">
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-slate-300 text-sm font-semibold uppercase tracking-wide">Live Feed</h2>
              <button
                onClick={() => router.push(`/cases/${id}/timeline`)}
                className="text-cyan-400 hover:text-cyan-300 text-xs"
              >
                View full timeline →
              </button>
            </div>
            {liveEvents.length === 0 && (!activityData || activityData.data.length === 0) ? (
              <p className="text-slate-600 text-sm italic">No activity yet.</p>
            ) : (
              <div>
                {liveEvents.map((e) => (
                  <div key={e.id} className="flex gap-2 py-2 border-b border-slate-700/50 last:border-0">
                    <span className="w-1.5 h-1.5 rounded-full bg-cyan-400 mt-2 flex-shrink-0" />
                    <div>
                      <p className="text-slate-300 text-xs">{timelineLabel(e)}</p>
                      <p className="text-slate-600 text-[10px]">{new Date(e.occurred_at).toLocaleString()}</p>
                    </div>
                  </div>
                ))}
                {activityData?.data.map((a) => <ActivityItem key={a.id} a={a} />)}
              </div>
            )}
          </div>
        </div>

        {/* Right column — Context Panel (35%) */}
        <div className="space-y-4">
          {/* Details card with inline assignee */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-5 space-y-4">
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wide mb-1">Assigned To</div>
              <div ref={assigneeRef} className="relative">
                <button
                  onClick={() => setAssigneeOpen((v) => !v)}
                  className="text-slate-200 text-sm hover:text-cyan-400 transition-colors text-left w-full"
                >
                  {c.assigned_to?.full_name ?? <span className="text-slate-600 italic">Unassigned — click to assign</span>}
                </button>
                {assigneeOpen && (
                  <div className="absolute top-7 left-0 bg-[#0f172a] border border-slate-700 rounded-lg shadow-xl z-30 w-56 py-1 max-h-52 overflow-y-auto">
                    <select
                      size={5}
                      className="w-full bg-transparent text-slate-300 text-sm px-3 py-1 focus:outline-none"
                      onChange={(e) => e.target.value && assignMutation.mutate(e.target.value)}
                      defaultValue=""
                    >
                      <option value="" disabled>Select assignee</option>
                      {(usersData?.users ?? [])
                        .filter((u) => ["analyst", "manager", "admin", "super_admin"].includes(u.role))
                        .map((u) => (
                          <option key={u.id} value={u.id}>{u.full_name}</option>
                        ))}
                    </select>
                  </div>
                )}
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

          {/* Open Tasks preview */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4">
            <div className="flex items-center justify-between mb-2">
              <div className="text-slate-500 text-xs uppercase tracking-wide">Open Tasks</div>
              <button onClick={() => router.push(`/cases/${id}/tasks`)} className="text-cyan-400 text-xs hover:text-cyan-300">View all →</button>
            </div>
            {!tasksPreview || tasksPreview.data.length === 0 ? (
              <p className="text-slate-600 text-xs italic">No open tasks.</p>
            ) : (
              <div className="space-y-1.5">
                {tasksPreview.data.map((t) => (
                  <button
                    key={t.id}
                    onClick={() => router.push(`/cases/${id}/tasks`)}
                    className="w-full text-left flex items-center gap-2 p-1.5 rounded hover:bg-slate-700/30 transition-colors"
                  >
                    <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${t.is_overdue ? "bg-red-400" : "bg-slate-500"}`} />
                    <span className="text-slate-300 text-xs truncate flex-1">{t.title}</span>
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Documents preview */}
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4">
            <div className="flex items-center justify-between mb-2">
              <div className="text-slate-500 text-xs uppercase tracking-wide">Documents</div>
              <button onClick={() => router.push(`/cases/${id}/documents`)} className="text-cyan-400 text-xs hover:text-cyan-300">View all →</button>
            </div>
            {!docsPreview || docsPreview.data.length === 0 ? (
              <p className="text-slate-600 text-xs italic">No documents.</p>
            ) : (
              <div className="space-y-1.5">
                {docsPreview.data.map((d) => (
                  <button
                    key={d.id}
                    onClick={() => router.push(`/cases/${id}/documents`)}
                    className="w-full text-left flex items-center gap-2 p-1.5 rounded hover:bg-slate-700/30 transition-colors"
                  >
                    <span className="text-slate-500 text-xs">◻</span>
                    <span className="text-slate-300 text-xs truncate flex-1">{d.original_filename}</span>
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Tags */}
          {(c.tags?.length ?? 0) > 0 && (
            <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4">
              <div className="text-slate-500 text-xs uppercase tracking-wide mb-2">Tags</div>
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
            <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4">
              <div className="text-slate-500 text-xs uppercase tracking-wide mb-2">Watchers</div>
              <div className="space-y-1">
                {(c.watchers ?? []).map((w) => (
                  <div key={w.id} className="text-slate-300 text-xs">{w.full_name}</div>
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
