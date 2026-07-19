"use client";

import { use, useState, useRef, useMemo, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { api, APIError } from "@/lib/api";
import { StatusBadge } from "@/components/cases/StatusBadge";
import { PriorityBadge } from "@/components/cases/PriorityBadge";
import { SLACountdown } from "@/components/cases/SLACountdown";
import { useCaseNav } from "@/lib/contexts/CaseNavContext";
import { useAuthStore } from "@/lib/stores/auth.store";
import { useTimelineSSE, TimelineEvent } from "@/lib/hooks/useTimelineSSE";
import { useCopilotStore } from "@/lib/stores/copilot.store";
import { PageError } from "@/components/ui/PageError";

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

// ── Feature 6: Proactive AI Investigation Partner ────────────────────────────

interface Nudge {
  id: string;
  severity: "critical" | "warning" | "info";
  icon: string;
  message: string;
  actionLabel: string;
  onAction: () => void;
}

interface STRListResponse { filings: Array<{ id: string; status: string }> }
interface AILogListResponse { data: Array<{ id: string }> }

function InvestigationNudgePanel({
  caseId,
  c,
  hasOverdueTasks,
}: {
  caseId: string;
  c: CaseDetail;
  hasOverdueTasks: boolean;
}) {
  const router = useRouter();
  const setCopilotOpen = useCopilotStore((s) => s.setOpen);

  const [dismissed, setDismissed] = useState<Set<string>>(() => {
    try {
      const raw = sessionStorage.getItem(`nudges-dismissed-${caseId}`);
      return raw ? new Set(JSON.parse(raw) as string[]) : new Set();
    } catch { return new Set(); }
  });

  const { data: strData } = useQuery<STRListResponse>({
    queryKey: ["str-list-nudge", caseId],
    queryFn: () => api.get<STRListResponse>(`/api/v1/cases/${caseId}/str`),
    staleTime: 60_000,
  });

  const { data: pendingAI } = useQuery<AILogListResponse>({
    queryKey: ["ai-logs-pending-nudge", caseId],
    queryFn: () => api.getList<{ id: string }>(`/api/v1/cases/${caseId}/ai/logs?status=pending_review&limit=10`),
    staleTime: 30_000,
  });

  const nudges = useMemo((): Nudge[] => {
    const isActive = ["open", "in_progress", "pending_review", "escalated"].includes(c.status);
    if (!isActive) return [];

    const strFilings = strData?.filings ?? [];
    const pendingCount = pendingAI?.data?.length ?? 0;
    const daysSinceUpdate = (Date.now() - new Date(c.updated_at).getTime()) / 86_400_000;

    const list: Nudge[] = [
      c.sla_breached && {
        id: "sla-breached",
        severity: "critical" as const,
        icon: "🚨",
        message: "SLA deadline has passed — this case is overdue.",
        actionLabel: "Escalate →",
        onAction: () => {},
      },
      !c.assigned_to && {
        id: "unassigned",
        severity: "critical" as const,
        icon: "👤",
        message: "No investigator assigned to this case.",
        actionLabel: "Assign",
        onAction: () => {},
      },
      (c.risk_level === "high" || c.risk_level === "critical") && strFilings.length === 0 && {
        id: "high-risk-no-str",
        severity: "warning" as const,
        icon: "📋",
        message: "High-risk case — no STR filing initiated yet.",
        actionLabel: "Initiate STR →",
        onAction: () => router.push(`/cases/${caseId}/str`),
      },
      pendingCount > 0 && {
        id: "ai-pending",
        severity: "warning" as const,
        icon: "🤖",
        message: `${pendingCount} AI output${pendingCount !== 1 ? "s" : ""} awaiting human review.`,
        actionLabel: "Review",
        onAction: () => setCopilotOpen(true),
      },
      hasOverdueTasks && {
        id: "overdue-tasks",
        severity: "warning" as const,
        icon: "⏰",
        message: "One or more investigation tasks are overdue.",
        actionLabel: "View Tasks →",
        onAction: () => router.push(`/cases/${caseId}/tasks`),
      },
      !c.risk_level && (c.documents_count ?? 0) > 0 && {
        id: "no-analysis",
        severity: "info" as const,
        icon: "⚡",
        message: "AI hasn't scored this case — run analysis to surface risk insights.",
        actionLabel: "Run Analysis",
        onAction: () => router.push(`/cases/${caseId}/insights`),
      },
      daysSinceUpdate > 14 && {
        id: "inactive",
        severity: "info" as const,
        icon: "💤",
        message: `No activity for ${Math.floor(daysSinceUpdate)} days — add a note or update the status.`,
        actionLabel: "Add Note",
        onAction: () => document.querySelector<HTMLTextAreaElement>("textarea[placeholder*='Quick']")?.focus(),
      },
      (c.documents_count ?? 0) === 0 && {
        id: "no-docs",
        severity: "info" as const,
        icon: "📎",
        message: "No evidence documents uploaded to support this case.",
        actionLabel: "Upload Docs →",
        onAction: () => router.push(`/cases/${caseId}/documents`),
      },
    ].filter((n): n is Nudge => Boolean(n) && !dismissed.has((n as Nudge).id));

    return list;
  }, [c, strData, pendingAI, hasOverdueTasks, dismissed, caseId, router, setCopilotOpen]);

  const dismiss = (id: string) => {
    setDismissed(prev => {
      const next = new Set([...prev, id]);
      try { sessionStorage.setItem(`nudges-dismissed-${caseId}`, JSON.stringify([...next])); } catch {}
      return next;
    });
  };

  const dismissAll = () => {
    const ids = new Set([...dismissed, ...nudges.map(n => n.id)]);
    try { sessionStorage.setItem(`nudges-dismissed-${caseId}`, JSON.stringify([...ids])); } catch {}
    setDismissed(ids);
  };

  if (nudges.length === 0) return null;

  const hasCritical = nudges.some(n => n.severity === "critical");
  const hasWarning  = !hasCritical && nudges.some(n => n.severity === "warning");
  const borderCls   = hasCritical ? "border-red-500/30"   : hasWarning ? "border-amber-500/30"   : "border-cyan-500/20";
  const bgCls       = hasCritical ? "bg-red-950/20"       : hasWarning ? "bg-amber-950/15"        : "bg-cyan-950/15";
  const badgeCls    = hasCritical ? "bg-red-500/20 text-red-300" : hasWarning ? "bg-amber-500/20 text-amber-300" : "bg-cyan-500/20 text-cyan-300";

  return (
    <div className={`${bgCls} border ${borderCls} rounded-xl p-3 mb-5`} data-testid="investigation-nudge-panel">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-300">
          <span>🤖</span>
          <span>AI Partner</span>
          <span className={`ml-1 px-1.5 py-0.5 rounded-full text-[10px] font-bold ${badgeCls}`}>
            {nudges.length} {nudges.length === 1 ? "item" : "items"}
          </span>
        </div>
        <button
          onClick={dismissAll}
          className="text-slate-600 hover:text-slate-400 text-[10px] transition-colors"
          data-testid="nudge-dismiss-all"
        >
          Dismiss all
        </button>
      </div>

      <div className="space-y-1.5">
        {nudges.map(n => {
          const textCls = n.severity === "critical" ? "text-red-300" : n.severity === "warning" ? "text-amber-300" : "text-slate-300";
          const btnCls  = n.severity === "critical"
            ? "border-red-500/40 text-red-300 hover:bg-red-500/10"
            : n.severity === "warning"
            ? "border-amber-500/40 text-amber-300 hover:bg-amber-500/10"
            : "border-cyan-500/30 text-cyan-400 hover:bg-cyan-500/10";

          return (
            <div key={n.id} className="flex items-center gap-2 group" data-testid={`nudge-${n.id}`}>
              <span className="text-sm flex-shrink-0 leading-none">{n.icon}</span>
              <span className={`text-xs flex-1 leading-snug ${textCls}`}>{n.message}</span>
              <div className="flex items-center gap-1 flex-shrink-0">
                <button
                  onClick={n.onAction}
                  className={`text-[10px] font-medium px-2 py-0.5 rounded border transition-colors ${btnCls}`}
                >
                  {n.actionLabel}
                </button>
                <button
                  onClick={() => dismiss(n.id)}
                  title="Dismiss"
                  className="text-slate-700 hover:text-slate-400 text-xs transition-colors opacity-0 group-hover:opacity-100 leading-none"
                >
                  ✕
                </button>
              </div>
            </div>
          );
        })}
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
  const [exportingPDF, setExportingPDF] = useState(false);
  const { setAction } = useCaseNav();
  const [noteText, setNoteText] = useState("");
  const [noteSuccess, setNoteSuccess] = useState(false);
  const [assigneeOpen, setAssigneeOpen] = useState(false);
  // Inline edit state
  const [editing, setEditing] = useState(false);
  const [editTitle, setEditTitle] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [editPriority, setEditPriority] = useState("");
  const [editDueDate, setEditDueDate] = useState("");
  const assigneeRef = useRef<HTMLDivElement>(null);

  // Live timeline feed
  const [liveEvents, setLiveEvents] = useState<TimelineEvent[]>([]);
  const handleTimelineEvent = (event: TimelineEvent) => {
    setLiveEvents((prev) => {
      if (prev.some((e) => e.id === event.id)) return prev;
      return [event, ...prev].slice(0, 20);
    });
  };
  useTimelineSSE({ caseId: id, onEvent: handleTimelineEvent });

  const { data: c, isLoading, isError, error, refetch } = useQuery<CaseDetail>({
    queryKey: ["case", id],
    queryFn: () => api.get<CaseDetail>(`/api/v1/cases/${id}`),
    retry: (failureCount, err) => {
      // Don't retry on 404 — the case genuinely doesn't exist.
      if (err instanceof APIError && err.status === 404) return false;
      return failureCount < 2;
    },
    refetchInterval: 30_000,
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

  const { data: usersList } = useQuery<UserOption[]>({
    queryKey: ["users-list"],
    queryFn: () => api.get<UserOption[]>("/api/v1/users?limit=100"),
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
      api.post(`/api/v1/cases/${id}/assign`, { assigned_to: userId }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["case", id] });
      qc.invalidateQueries({ queryKey: ["case-activity", id] });
      setAssigneeOpen(false);
    },
  });

  const editMutation = useMutation({
    mutationFn: (payload: Record<string, string | null>) =>
      api.put(`/api/v1/cases/${id}`, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["case", id] });
      qc.invalidateQueries({ queryKey: ["case-activity", id] });
      setEditing(false);
    },
  });

  function openEdit() {
    if (!c) return;
    if (c.status === "closed") return;
    setEditTitle(c.title);
    setEditDescription(c.description ?? "");
    setEditPriority(c.priority);
    setEditDueDate(c.due_date ? c.due_date.slice(0, 10) : "");
    setEditing(true);
  }

  const todayStr = new Date().toISOString().split("T")[0];

  function saveEdit() {
    if (editDueDate && editDueDate < todayStr) return;
    const payload: Record<string, string | null> = {
      title: editTitle.trim() || undefined as unknown as string,
      description: editDescription,
      priority: editPriority,
      due_date: editDueDate ? new Date(editDueDate).toISOString() : null,
    };
    editMutation.mutate(payload);
  }

  const noteMutation = useMutation({
    mutationFn: (content: string) =>
      api.post(`/api/v1/cases/${id}/notes`, {
        content: `<p>${content.replace(/\n/g, "</p><p>")}</p>`,
        content_text: content,
        note_type: "internal",
      }),
    onSuccess: () => {
      setNoteText("");
      setNoteSuccess(true);
      qc.invalidateQueries({ queryKey: ["case-activity", id] });
      setTimeout(() => setNoteSuccess(false), 3000);
    },
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-6 h-6 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  if (isError || !c) {
    return (
      <div className="px-6 pb-6">
        <PageError error={error as Error | null} onRetry={() => refetch()} />
      </div>
    );
  }

  const nextStatuses = VALID_TRANSITIONS[c.status] ?? [];

  // Sync the PDF export button to the shared layout action slot
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    setAction(
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
    );
    return () => setAction(null);
  }, [exportingPDF, c?.case_number, id, setAction]);

  return (
    <div className="px-6 pb-6">
      {/* Header */}
      <div className={`flex items-start justify-between mb-6 pl-4 border-l-4 ${
        c.status === "open"           ? "border-blue-500" :
        c.status === "in_progress"    ? "border-violet-500" :
        c.status === "pending_review" ? "border-amber-500" :
        c.status === "escalated"      ? "border-red-500" :
        c.status === "closed"         ? "border-green-500" :
        "border-slate-600"
      }`}>
        <div className="flex-1 min-w-0 pr-4">
          <div className="flex items-center gap-2 mb-1">
            <div className="font-mono text-sm text-cyan-400">{c.case_number}</div>
            {!editing && c.status !== "closed" && (
              <button
                onClick={openEdit}
                title="Edit case details"
                className="text-slate-600 hover:text-cyan-400 transition-colors p-0.5 rounded"
              >
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
              </button>
            )}
          </div>

          {/* Title — inline edit */}
          {editing ? (
            <input
              value={editTitle}
              onChange={(e) => setEditTitle(e.target.value)}
              autoFocus
              className="w-full bg-[#0d1929] border border-cyan-500 rounded-lg px-3 py-1.5 text-white text-2xl font-bold focus:outline-none mb-2"
            />
          ) : (
            <h1 className="text-white text-2xl font-bold">{c.title}</h1>
          )}

          <div className="flex items-center gap-3 mt-2 flex-wrap">
            <StatusBadge status={c.status} size="lg" />

            {/* Priority — inline edit */}
            {editing ? (
              <select
                value={editPriority}
                onChange={(e) => setEditPriority(e.target.value)}
                className="bg-[#0d1929] border border-cyan-500/60 rounded px-2 py-0.5 text-xs font-medium text-white focus:outline-none"
              >
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
                <option value="critical">Critical</option>
              </select>
            ) : (
              <PriorityBadge priority={c.priority} />
            )}

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

        {/* Status actions — primary flow left, escalate separated right */}
        {!editing && nextStatuses.length > 0 && (() => {
          const primaryStatuses = nextStatuses.filter((s) => s !== "escalated");
          const canEscalate = nextStatuses.includes("escalated");
          return (
            <div className="flex items-center gap-3 flex-wrap">
              {/* Primary workflow progression */}
              <div className="flex gap-2 flex-wrap">
                {primaryStatuses.map((s) => {
                  const isRoutine = ROUTINE_TRANSITIONS.has(s);
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
                        isRoutine
                          ? "bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-300 border border-emerald-500/30"
                          : "bg-slate-700 hover:bg-slate-600 text-slate-200 border border-slate-600/50"
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

              {/* Escalate — emergency action, separated */}
              {canEscalate && (
                <>
                  <span className="text-slate-700 select-none">|</span>
                  <button
                    disabled={statusMutation.isPending}
                    onClick={() => { setTargetStatus("escalated"); setShowStatusModal(true); }}
                    title="Escalate to senior management — requires a reason"
                    className="text-xs font-medium px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50 bg-red-900/20 hover:bg-red-800/30 text-red-400 border border-red-700/40 flex items-center gap-1.5"
                  >
                    <svg className="w-3 h-3 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
                    </svg>
                    Escalate
                  </button>
                </>
              )}
            </div>
          );
        })()}
      </div>

      {/* Feature 6: AI Partner nudge panel */}
      <InvestigationNudgePanel
        caseId={id}
        c={c}
        hasOverdueTasks={(tasksPreview?.data ?? []).some(t => t.is_overdue)}
      />

      <div className="grid grid-cols-3 gap-6">
        {/* Left column — Investigation Feed (65%) */}
        <div className="col-span-2 space-y-5">
          {/* Description — inline editable */}
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5 group/desc">
            <h2 className="text-slate-300 text-sm font-semibold uppercase tracking-wider mb-3">Description</h2>
            {editing ? (
              <textarea
                rows={6}
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
                placeholder="Describe the investigation context…"
                className="w-full bg-[#0d1929] border border-cyan-500/60 rounded-lg px-3 py-2 text-slate-200 text-sm focus:outline-none focus:border-cyan-400 resize-none"
              />
            ) : (
              <div
                onClick={openEdit}
                className="cursor-text group/text relative"
                title="Click to edit description"
              >
                <p className="text-slate-300 text-sm whitespace-pre-wrap">
                  {c.description || <span className="text-slate-600 italic">No description yet. Click to add one.</span>}
                </p>
                <span className="absolute top-0 right-0 opacity-0 group-hover/text:opacity-100 transition-opacity">
                  <svg className="w-3.5 h-3.5 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                  </svg>
                </span>
              </div>
            )}
          </div>

          {/* Quick Note Input */}
          <div className={`bg-[#131f30] border rounded-xl ring-1 ring-white/[0.04] p-4 transition-colors ${noteSuccess ? "border-emerald-500/50" : "border-[#1e3050]"}`}>
            <div className="flex items-center justify-between mb-2">
              <div className="text-slate-500 text-xs uppercase tracking-wider">Quick Note</div>
              <button
                onClick={() => router.push(`/cases/${id}/notes`)}
                className="text-slate-600 hover:text-cyan-400 text-xs transition-colors"
              >
                View all notes →
              </button>
            </div>

            {noteSuccess ? (
              <div className="flex items-center gap-2 py-3 text-emerald-400 text-sm">
                <svg className="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                Note saved — visible in the <button onClick={() => router.push(`/cases/${id}/notes`)} className="underline hover:text-emerald-300">Notes tab</button>
              </div>
            ) : (
              <>
                <textarea
                  value={noteText}
                  onChange={(e) => setNoteText(e.target.value)}
                  placeholder="e.g. Called customer — confirmed transactions were for vendor payments. Awaiting bank statement from Nexgen Traders branch."
                  rows={3}
                  className="w-full bg-slate-900/60 border border-slate-700 rounded-lg px-3 py-2 text-slate-300 text-sm placeholder:text-slate-600 focus:outline-none focus:border-cyan-500 resize-none"
                />
                <div className="flex items-center justify-between mt-2">
                  <span className="text-xs bg-slate-700/60 text-slate-400 border border-slate-600/50 px-2 py-0.5 rounded-full">
                    📋 Saved as internal note · visible in <button onClick={() => router.push(`/cases/${id}/notes`)} className="text-cyan-400 hover:text-cyan-300 underline underline-offset-2">Notes tab</button>
                  </span>
                  <button
                    onClick={() => noteText.trim() && noteMutation.mutate(noteText.trim())}
                    disabled={!noteText.trim() || noteMutation.isPending}
                    className="bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-semibold px-4 py-1.5 rounded-lg transition-colors disabled:opacity-50 flex items-center gap-1.5"
                  >
                    {noteMutation.isPending && (
                      <span className="w-3 h-3 border border-white border-t-transparent rounded-full animate-spin" />
                    )}
                    {noteMutation.isPending ? "Saving…" : "Add Note"}
                  </button>
                </div>
                {noteMutation.isError && (
                  <p className="text-red-400 text-xs mt-1">{(noteMutation.error as Error).message}</p>
                )}
              </>
            )}
          </div>

          {/* Live Timeline Feed */}
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5">
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-slate-300 text-sm font-semibold uppercase tracking-wider">Live Feed</h2>
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
        <div className="space-y-4 pb-16">
          {/* Details card with inline assignee */}
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5 space-y-4">
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wider mb-1">Assigned To</div>
              {user?.role === "analyst" ? (
                <div className="text-slate-200 text-sm">
                  {c.assigned_to?.full_name ?? <span className="text-slate-600 italic">Unassigned</span>}
                </div>
              ) : (
                <div ref={assigneeRef} className="relative">
                  <button
                    onClick={() => setAssigneeOpen((v) => !v)}
                    className="text-slate-200 text-sm hover:text-cyan-400 transition-colors text-left w-full"
                  >
                    {c.assigned_to?.full_name ?? <span className="text-slate-600 italic">Unassigned — click to assign</span>}
                  </button>
                  {assigneeOpen && (
                    <div className="absolute top-7 left-0 bg-[#080e1a] border border-slate-700 rounded-lg shadow-xl z-30 w-56 py-1 max-h-52 overflow-y-auto">
                      {!usersList ? (
                        <div className="flex items-center gap-2 px-3 py-3 text-slate-500 text-sm">
                          <span className="w-3.5 h-3.5 border border-slate-500 border-t-cyan-400 rounded-full animate-spin flex-shrink-0" />
                          Loading users…
                        </div>
                      ) : (
                        usersList
                          .filter((u) => ["analyst", "manager", "admin", "super_admin"].includes(u.role))
                          .map((u) => (
                            <button
                              key={u.id}
                              onClick={() => assignMutation.mutate(u.id)}
                              className={`w-full text-left px-3 py-2 text-sm transition-colors hover:bg-slate-700/60 flex items-center justify-between ${
                                c.assigned_to?.id === u.id ? "text-cyan-400" : "text-slate-300"
                              }`}
                            >
                              <span>{u.full_name}</span>
                              {c.assigned_to?.id === u.id && (
                                <svg className="w-3.5 h-3.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                                </svg>
                              )}
                            </button>
                          ))
                      )}
                    </div>
                  )}
                </div>
              )}
            </div>
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wider mb-1">Created By</div>
              <div className="text-slate-200 text-sm">{c.created_by?.full_name || "—"}</div>
            </div>
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wider mb-1">Created</div>
              <div className="text-slate-200 text-sm">{new Date(c.created_at).toLocaleString()}</div>
            </div>
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wider mb-1">Due Date</div>
              {editing ? (
                <>
                  <input
                    type="date"
                    value={editDueDate}
                    min={todayStr}
                    onChange={(e) => setEditDueDate(e.target.value)}
                    className={`bg-[#0d1929] border rounded px-2 py-1 text-sm focus:outline-none w-full ${
                      editDueDate && editDueDate < todayStr
                        ? "border-red-500 text-red-400 focus:border-red-400"
                        : "border-cyan-500/60 text-slate-200 focus:border-cyan-400"
                    }`}
                  />
                  {editDueDate && editDueDate < todayStr && (
                    <p className="text-red-400 text-[10px] mt-0.5">Must be today or a future date.</p>
                  )}
                </>
              ) : (
                <div
                  onClick={openEdit}
                  className="text-slate-200 text-sm cursor-text hover:text-cyan-300 transition-colors"
                  title="Click to edit due date"
                >
                  {c.due_date ? new Date(c.due_date).toLocaleDateString() : <span className="text-slate-600 italic text-xs">Not set</span>}
                </div>
              )}
            </div>
            {c.sla_breach_at && (
              <div>
                <div className="text-slate-500 text-xs uppercase tracking-wider mb-1">SLA Deadline</div>
                <div className="text-slate-200 text-sm">{new Date(c.sla_breach_at).toLocaleString()}</div>
              </div>
            )}
          </div>

          {/* Open Tasks preview */}
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-4">
            <div className="flex items-center justify-between mb-2">
              <div className="text-slate-500 text-xs uppercase tracking-wider">Open Tasks</div>
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
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-4">
            <div className="flex items-center justify-between mb-2">
              <div className="text-slate-500 text-xs uppercase tracking-wider">Documents</div>
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
            <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-4">
              <div className="text-slate-500 text-xs uppercase tracking-wider mb-2">Tags</div>
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
            <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-4">
              <div className="text-slate-500 text-xs uppercase tracking-wider mb-2">Watchers</div>
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
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-6 w-full max-w-md">
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
              className="w-full bg-[#080e1a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500 resize-none mb-3"
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

      {/* Floating save bar — visible only when editing */}
      {editing && (
        <div className="fixed bottom-0 left-0 right-0 z-50 bg-[#0d1929]/95 backdrop-blur border-t border-cyan-500/30 px-6 py-3 flex items-center justify-between shadow-2xl">
          <div className="flex items-center gap-2 text-slate-400 text-sm">
            <span className="w-2 h-2 rounded-full bg-cyan-400 animate-pulse" />
            Editing case details — changes not saved yet
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setEditing(false)}
              className="px-4 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-300 text-sm rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={saveEdit}
              disabled={editMutation.isPending}
              className="px-5 py-1.5 bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-semibold rounded-lg transition-colors disabled:opacity-50 flex items-center gap-1.5"
            >
              {editMutation.isPending && (
                <span className="w-3 h-3 border border-white border-t-transparent rounded-full animate-spin" />
              )}
              {editMutation.isPending ? "Saving…" : "Save Changes"}
            </button>
          </div>
          {editMutation.isError && (
            <p className="absolute bottom-full left-6 mb-1 text-red-400 text-xs">{(editMutation.error as Error).message}</p>
          )}
        </div>
      )}
    </div>
  );
}
