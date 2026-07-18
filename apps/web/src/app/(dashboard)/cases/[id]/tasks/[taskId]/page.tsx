"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient, useInfiniteQuery } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { TaskStatusBadge } from "@/components/tasks/TaskStatusBadge";
import { TaskPriorityBadge } from "@/components/tasks/TaskPriorityBadge";
import { Task } from "@/components/tasks/TaskCard";
import { CaseSubNav } from "@/components/cases/CaseSubNav";
import { useCaseSummary } from "@/lib/hooks/useCaseSummary";

interface Comment {
  id: string;
  task_id: string;
  author: { id: string; full_name: string; email: string };
  content: string;
  created_at: string;
  updated_at: string;
}

interface CommentMeta {
  has_more: boolean;
  next_cursor: string | null;
}

const PRIORITY_OPTIONS = ["critical", "high", "medium", "low"] as const;
const STATUS_OPTIONS = ["todo", "in_progress", "done", "cancelled"] as const;

const VALID_TRANSITIONS: Record<string, string[]> = {
  todo:        ["in_progress", "cancelled"],
  in_progress: ["done", "cancelled"],
  done:        ["todo"],
  cancelled:   [],
};

interface UserOption {
  id: string;
  full_name: string;
  email: string;
  role: string;
  is_active: boolean;
}

function relativeTime(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

export default function TaskDetailPage({
  params,
}: {
  params: Promise<{ id: string; taskId: string }>;
}) {
  const { id: caseId, taskId } = use(params);
  const router = useRouter();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const hasRole = useAuthStore((s) => s.hasRole);

  const [isEditing, setIsEditing] = useState(false);
  const [editTitle, setEditTitle] = useState("");
  const [editDesc, setEditDesc] = useState("");
  const [editPriority, setEditPriority] = useState("");
  const [editStatus, setEditStatus] = useState("");
  const [editDueDate, setEditDueDate] = useState("");
  const todayStr = new Date().toISOString().split("T")[0];
  const [editAssignedTo, setEditAssignedTo] = useState("");
  const [editError, setEditError] = useState("");
  const [reassigning, setReassigning] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const [comment, setComment] = useState("");
  const [commentError, setCommentError] = useState("");

  const caseData = useCaseSummary(caseId);

  const { data: task, isLoading } = useQuery<Task>({
    queryKey: ["task", taskId],
    queryFn: () => api.get<Task>(`/api/v1/tasks/${taskId}`),
  });

  const { data: usersData } = useQuery<UserOption[]>({
    queryKey: ["users-list"],
    queryFn: () => api.get<UserOption[]>("/api/v1/users?limit=100"),
    staleTime: 5 * 60 * 1000,
  });
  const users = (usersData ?? []).filter((u) => u.is_active);

  const { data: commentsData, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useInfiniteQuery<{ data: Comment[]; meta: CommentMeta }>({
      queryKey: ["task-comments", taskId],
      queryFn: ({ pageParam }) => {
        const qs = pageParam ? `?cursor=${pageParam}&limit=20` : `?limit=20`;
        return api.getList<Comment, CommentMeta>(`/api/v1/tasks/${taskId}/comments${qs}`);
      },
      initialPageParam: undefined as string | undefined,
      getNextPageParam: (last) => last.meta.next_cursor ?? undefined,
      enabled: !!taskId,
    });

  const comments = commentsData?.pages.flatMap((p) => p.data) ?? [];

  const completeMutation = useMutation({
    mutationFn: () => api.post<Task>(`/api/v1/tasks/${taskId}/complete`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["task", taskId] }),
  });

  const updateMutation = useMutation({
    mutationFn: (body: Record<string, unknown>) =>
      api.put<Task>(`/api/v1/tasks/${taskId}`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["task", taskId] });
      setIsEditing(false);
      setEditError("");
    },
    onError: (err) => {
      setEditError(err instanceof APIError ? err.message : "Update failed");
    },
  });

  const assignMutation = useMutation({
    mutationFn: (userId: string | null) =>
      api.put<Task>(`/api/v1/tasks/${taskId}`, userId ? { assigned_to: userId } : { clear_assigned_to: true }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["task", taskId] });
      setReassigning(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => api.delete(`/api/v1/tasks/${taskId}`),
    onSuccess: () => router.push(`/cases/${caseId}/tasks`),
  });

  const addCommentMutation = useMutation({
    mutationFn: () => api.post<Comment>(`/api/v1/tasks/${taskId}/comments`, { content: comment }),
    onSuccess: () => {
      setComment("");
      setCommentError("");
      qc.invalidateQueries({ queryKey: ["task-comments", taskId] });
      qc.invalidateQueries({ queryKey: ["task", taskId] });
    },
    onError: (err) => {
      setCommentError(err instanceof APIError ? err.message : "Failed to add comment");
    },
  });

  function startEdit() {
    if (!task) return;
    setEditTitle(task.title);
    setEditDesc(task.description ?? "");
    setEditPriority(task.priority);
    setEditStatus(task.status);
    setEditDueDate(task.due_date ? task.due_date.split("T")[0] : "");
    setEditAssignedTo(task.assigned_to?.id ?? "");
    setEditError("");
    setIsEditing(true);
  }

  function submitEdit() {
    const body: Record<string, unknown> = {
      title: editTitle,
      description: editDesc,
      priority: editPriority,
    };
    if (editStatus !== task?.status) body.status = editStatus;
    if (editDueDate) {
      body.due_date = new Date(editDueDate).toISOString();
    } else if (task?.due_date) {
      body.clear_due_date = true;
    }
    const prevAssignee = task?.assigned_to?.id ?? "";
    if (editAssignedTo !== prevAssignee) {
      if (editAssignedTo) body.assigned_to = editAssignedTo;
      else body.clear_assigned_to = true;
    }
    updateMutation.mutate(body);
  }

  const canEdit = hasRole("analyst");
  const isDone = task?.status === "done";
  const isCancelled = task?.status === "cancelled";
  const nextStatuses = task ? VALID_TRANSITIONS[task.status] ?? [] : [];

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-6 h-6 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  if (!task) {
    return (
      <div className="p-6 text-center">
        <p className="text-slate-500">Task not found.</p>
        <button
          onClick={() => router.push(`/cases/${caseId}/tasks`)}
          className="mt-3 text-cyan-400 hover:text-cyan-300 text-sm"
        >
          ← Back to tasks
        </button>
      </div>
    );
  }

  return (
    <div className="p-6">
      <CaseSubNav caseId={caseId} activeTab="tasks" caseNumber={caseData?.case_number} />

      <div className="grid grid-cols-3 gap-6">
        {/* Main */}
        <div className="col-span-2 space-y-6">
          {/* Task header */}
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-6">
            {isEditing ? (
              <div className="space-y-4">
                <input
                  value={editTitle}
                  onChange={(e) => setEditTitle(e.target.value)}
                  className="w-full bg-[#080e1a] border border-slate-600 text-white rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500 font-semibold"
                />
                <textarea
                  value={editDesc}
                  onChange={(e) => setEditDesc(e.target.value)}
                  rows={4}
                  placeholder="Description…"
                  className="w-full bg-[#080e1a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500 resize-none"
                />
                <div className="grid grid-cols-3 gap-3">
                  <div>
                    <label className="text-slate-500 text-xs mb-1 block">Status</label>
                    <select
                      value={editStatus}
                      onChange={(e) => setEditStatus(e.target.value)}
                      className="w-full bg-[#080e1a] border border-slate-600 text-white rounded-lg px-3 py-2 text-xs focus:outline-none focus:border-cyan-500"
                    >
                      {STATUS_OPTIONS.map((s) => {
                        const isValid =
                          s === task.status ||
                          nextStatuses.includes(s);
                        return (
                          <option key={s} value={s} disabled={!isValid}>
                            {s.replace("_", " ")}
                          </option>
                        );
                      })}
                    </select>
                  </div>
                  <div>
                    <label className="text-slate-500 text-xs mb-1 block">Priority</label>
                    <select
                      value={editPriority}
                      onChange={(e) => setEditPriority(e.target.value)}
                      className="w-full bg-[#080e1a] border border-slate-600 text-white rounded-lg px-3 py-2 text-xs focus:outline-none focus:border-cyan-500"
                    >
                      {PRIORITY_OPTIONS.map((p) => (
                        <option key={p} value={p}>{p}</option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="text-slate-500 text-xs mb-1 block">Due Date</label>
                    <input
                      type="date"
                      value={editDueDate}
                      min={todayStr}
                      onChange={(e) => setEditDueDate(e.target.value)}
                      className={`w-full bg-[#080e1a] border rounded-lg px-3 py-2 text-xs focus:outline-none [color-scheme:dark] ${
                        editDueDate && editDueDate < todayStr
                          ? "border-red-500 text-red-400 focus:border-red-400"
                          : "border-slate-600 text-white focus:border-cyan-500"
                      }`}
                    />
                    {editDueDate && editDueDate < todayStr && (
                      <p className="text-red-400 text-[10px] mt-0.5">Must be today or a future date.</p>
                    )}
                  </div>
                </div>
                <div>
                  <label className="text-slate-500 text-xs mb-1 block">Assign To</label>
                  <select
                    value={editAssignedTo}
                    onChange={(e) => setEditAssignedTo(e.target.value)}
                    className="w-full bg-[#080e1a] border border-slate-600 text-white rounded-lg px-3 py-2 text-xs focus:outline-none focus:border-cyan-500"
                  >
                    <option value="">— Unassigned —</option>
                    {users.map((u) => (
                      <option key={u.id} value={u.id}>{u.full_name} [{u.role.replace("_", " ")}] · {u.email}</option>
                    ))}
                  </select>
                </div>
                {editError && <p className="text-red-400 text-xs">{editError}</p>}
                <div className="flex gap-2">
                  <button
                    onClick={submitEdit}
                    disabled={!editTitle.trim() || updateMutation.isPending || !!(editDueDate && editDueDate < todayStr)}
                    className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-1.5 rounded-lg text-sm disabled:opacity-60"
                  >
                    {updateMutation.isPending ? "Saving…" : "Save"}
                  </button>
                  <button
                    onClick={() => setIsEditing(false)}
                    className="text-slate-400 hover:text-slate-200 px-4 py-1.5 text-sm"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            ) : (
              <>
                <div className="flex items-start justify-between gap-3 mb-4">
                  <h1 className={`text-white text-xl font-bold leading-snug ${isDone || isCancelled ? "line-through text-slate-500" : ""}`}>
                    {task.title}
                  </h1>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    {canEdit && !isDone && !isCancelled && (
                      <button
                        onClick={startEdit}
                        className="text-slate-500 hover:text-slate-300 text-xs px-2 py-1 rounded border border-slate-700 hover:border-slate-600 transition-colors"
                      >
                        Edit
                      </button>
                    )}
                    {canEdit && !isDone && !isCancelled && (
                      confirmDelete ? (
                        <div className="flex items-center gap-1">
                          <button onClick={() => deleteMutation.mutate()} className="text-xs text-white bg-red-600 hover:bg-red-500 px-3 py-1 rounded transition-colors">Confirm Delete</button>
                          <button onClick={() => setConfirmDelete(false)} className="text-xs text-slate-400 hover:text-slate-200 px-2 py-1 rounded border border-slate-700 transition-colors">Cancel</button>
                        </div>
                      ) : (
                        <button
                          onClick={() => setConfirmDelete(true)}
                          className="text-slate-500 hover:text-red-400 text-xs px-2 py-1 rounded border border-slate-700 hover:border-red-500/50 transition-colors"
                        >
                          Delete
                        </button>
                      )
                    )}
                  </div>
                </div>
                <div className="flex items-center flex-wrap gap-2 mb-4">
                  <TaskStatusBadge status={task.status} />
                  <TaskPriorityBadge priority={task.priority} />
                  {task.is_overdue && !isDone && (
                    <span className="inline-flex items-center rounded-full border border-red-500/40 bg-red-500/20 px-2 py-0.5 text-xs font-medium text-red-400">
                      Overdue
                    </span>
                  )}
                  {task.is_ai_suggested && (
                    <span className="inline-flex items-center rounded-full border border-violet-500/40 bg-violet-500/10 px-2 py-0.5 text-xs font-medium text-violet-400">
                      AI Suggested
                    </span>
                  )}
                </div>
                {task.description && (
                  <p className="text-slate-400 text-sm whitespace-pre-wrap">{task.description}</p>
                )}
                {!isDone && canEdit && (
                  <div className="flex gap-2 mt-4 pt-4 border-t border-slate-700">
                    {!completeMutation.isPending && !isCancelled && (
                      <button
                        onClick={() => completeMutation.mutate()}
                        className="bg-green-500/20 hover:bg-green-500/30 text-green-400 border border-green-500/40 font-semibold px-4 py-1.5 rounded-lg text-sm transition-colors"
                      >
                        ✓ Mark Complete
                      </button>
                    )}
                    {completeMutation.isPending && (
                      <span className="text-slate-500 text-sm">Completing…</span>
                    )}
                  </div>
                )}
              </>
            )}
          </div>

          {/* Comments */}
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5">
            <h2 className="text-slate-300 text-sm font-semibold mb-4 uppercase tracking-wider">
              Comments ({task.comment_count})
            </h2>

            {/* Comment list */}
            {comments.length === 0 ? (
              <p className="text-slate-600 text-sm italic mb-4">No comments yet.</p>
            ) : (
              <div className="space-y-4 mb-4">
                {comments.map((c) => (
                  <div key={c.id} className="flex gap-3">
                    <div className="w-7 h-7 rounded-full bg-slate-700 flex items-center justify-center text-xs text-slate-300 flex-shrink-0 mt-0.5">
                      {(c.author.full_name || c.author.email || "?").charAt(0).toUpperCase()}
                    </div>
                    <div className="flex-1">
                      <div className="flex items-baseline gap-2">
                        <span className="text-white text-sm font-medium">{c.author.full_name || c.author.email}</span>
                        <span className="text-slate-600 text-xs">{relativeTime(c.created_at)}</span>
                      </div>
                      <p className="text-slate-300 text-sm mt-1 whitespace-pre-wrap">{c.content}</p>
                    </div>
                  </div>
                ))}
                {hasNextPage && (
                  <button
                    onClick={() => fetchNextPage()}
                    disabled={isFetchingNextPage}
                    className="text-cyan-400 hover:text-cyan-300 text-xs font-medium"
                  >
                    {isFetchingNextPage ? "Loading…" : "Load older comments"}
                  </button>
                )}
              </div>
            )}

            {/* Add comment */}
            <div className="border-t border-slate-700 pt-4">
              <textarea
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                rows={3}
                placeholder="Add a comment…"
                className="w-full bg-[#080e1a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500 resize-none"
              />
              {commentError && <p className="text-red-400 text-xs mt-1">{commentError}</p>}
              <div className="flex justify-end mt-2">
                <button
                  onClick={() => addCommentMutation.mutate()}
                  disabled={!comment.trim() || addCommentMutation.isPending}
                  className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-1.5 rounded-lg text-sm disabled:opacity-60"
                >
                  {addCommentMutation.isPending ? "Posting…" : "Post Comment"}
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Sidebar */}
        <div className="space-y-4">
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-5 space-y-4">
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wider mb-1.5">Assigned To</div>
              {reassigning ? (
                <div className="flex items-center gap-2">
                  <select
                    autoFocus
                    defaultValue={task.assigned_to?.id ?? ""}
                    onChange={(e) => assignMutation.mutate(e.target.value || null)}
                    disabled={assignMutation.isPending}
                    onBlur={() => setReassigning(false)}
                    className="flex-1 bg-[#080e1a] border border-cyan-500 text-white rounded-lg px-2 py-1 text-xs focus:outline-none disabled:opacity-60"
                  >
                    <option value="">— Unassigned —</option>
                    {users.map((u) => (
                      <option key={u.id} value={u.id}>{u.full_name} [{u.role.replace("_", " ")}] · {u.email}</option>
                    ))}
                  </select>
                  <button onClick={() => setReassigning(false)} className="text-slate-500 hover:text-slate-300 text-xs">✕</button>
                </div>
              ) : (
                <button
                  onClick={() => canEdit && setReassigning(true)}
                  className={`text-sm text-left w-full rounded px-1 py-0.5 -ml-1 transition-colors ${
                    canEdit ? "hover:bg-slate-700/50 cursor-pointer group" : "cursor-default"
                  }`}
                  title={canEdit ? "Click to reassign" : undefined}
                >
                  {task.assigned_to?.full_name
                    ? <span className="text-slate-200">{task.assigned_to.full_name}</span>
                    : <span className="text-slate-600 italic">Unassigned</span>}
                  {canEdit && (
                    <span className="ml-1.5 text-slate-600 group-hover:text-slate-400 text-xs opacity-0 group-hover:opacity-100 transition-opacity">✎</span>
                  )}
                </button>
              )}
            </div>
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wider mb-1">Created By</div>
              <div className="text-slate-200 text-sm">{task.created_by?.full_name || task.created_by?.email || "—"}</div>
            </div>
            {task.due_date && (
              <div>
                <div className="text-slate-500 text-xs uppercase tracking-wider mb-1">Due Date</div>
                <div className={`text-sm ${task.is_overdue && !isDone ? "text-red-400 font-medium" : "text-slate-200"}`}>
                  {new Date(task.due_date).toLocaleDateString()}
                </div>
              </div>
            )}
            {task.completed_at && (
              <div>
                <div className="text-slate-500 text-xs uppercase tracking-wider mb-1">Completed</div>
                <div className="text-green-400 text-sm">{new Date(task.completed_at).toLocaleString()}</div>
              </div>
            )}
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wider mb-1">Created</div>
              <div className="text-slate-400 text-sm">{new Date(task.created_at).toLocaleString()}</div>
            </div>
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wider mb-1">Last Updated</div>
              <div className="text-slate-400 text-sm">{relativeTime(task.updated_at)}</div>
            </div>
          </div>

          <button
            onClick={() => router.push(`/cases/${caseId}/tasks`)}
            className="w-full flex items-center gap-2 px-3 py-2.5 rounded-lg border border-slate-700 bg-[#131f30] hover:bg-slate-700/30 hover:border-slate-600 text-slate-400 hover:text-slate-200 text-xs font-medium transition-all"
          >
            <svg className="w-3.5 h-3.5 flex-shrink-0" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M10 12L6 8l4-4" />
            </svg>
            All tasks
          </button>
        </div>
      </div>
    </div>
  );
}
