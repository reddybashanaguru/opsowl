"use client";

import { use, useState, useRef, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  useQuery,
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { useConfig } from "@/lib/hooks/useConfig";
import { TaskCard, Task } from "@/components/tasks/TaskCard";
import { CaseSubNav } from "@/components/cases/CaseSubNav";
import { PageError } from "@/components/ui/PageError";

interface ListMeta {
  total_count: number;
  has_more: boolean;
  next_cursor: string | null;
}

import { useCaseSummary } from "@/lib/hooks/useCaseSummary";

const PRIORITY_OPTIONS = ["critical", "high", "medium", "low"] as const;

export default function CaseTasksPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: caseId } = use(params);
  const router = useRouter();
  const cfg = useConfig();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const hasRole = useAuthStore((s) => s.hasRole);

  const [showCreate, setShowCreate] = useState(false);
  const [taskTab, setTaskTab] = useState<"active" | "done">("active");
  const [filterPriority, setFilterPriority] = useState<string>("");
  const [overdueOnly, setOverdueOnly] = useState(false);

  // Create form state
  const [newTitle, setNewTitle] = useState("");
  const [newTitleTouched, setNewTitleTouched] = useState(false);
  const [newDesc, setNewDesc] = useState("");
  const [newPriority, setNewPriority] = useState("medium");
  const [newDueDate, setNewDueDate] = useState("");
  const [newAssignedTo, setNewAssignedTo] = useState("");
  const [createError, setCreateError] = useState("");

  const todayStr = new Date().toISOString().split("T")[0];

  const caseData = useCaseSummary(caseId);

  const { data: usersList = [] } = useQuery<{ id: string; full_name: string; email: string; role: string; is_active: boolean }[]>({
    queryKey: ["users-list"],
    queryFn: () => api.get<{ id: string; full_name: string; email: string; role: string; is_active: boolean }[]>("/api/v1/users?limit=100"),
    staleTime: 5 * 60 * 1000,
  });
  const users = usersList.filter((u) => u.is_active);

  const buildQueryString = () => {
    const p = new URLSearchParams();
    // Done tab: fetch only done tasks; active tab: fetch all then filter client-side
    if (taskTab === "done") p.set("status", "done");
    if (filterPriority) p.set("priority", filterPriority);
    if (overdueOnly && taskTab === "active") p.set("overdue", "true");
    p.set("limit", "50");
    return p.toString();
  };

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, isError, error, refetch } =
    useInfiniteQuery<{ data: Task[]; meta: ListMeta }>({
      queryKey: ["case-tasks", caseId, taskTab, filterPriority, overdueOnly],
      queryFn: ({ pageParam }) => {
        const qs = buildQueryString();
        const cursor = pageParam ? `&cursor=${pageParam}` : "";
        return api.getList<Task, ListMeta>(`/api/v1/cases/${caseId}/tasks?${qs}${cursor}`);
      },
      initialPageParam: undefined as string | undefined,
      getNextPageParam: (last) => last.meta.next_cursor ?? undefined,
      retry: (failureCount, err) => {
        if (err instanceof APIError && err.status === 404) return false;
        return failureCount < 2;
      },
    });

  const allFetched = data?.pages.flatMap((p) => p.data) ?? [];
  const tasks = taskTab === "active"
    ? allFetched.filter((t) => t.status !== "done" && t.status !== "cancelled")
    : allFetched;
  const totalCount = tasks.length;

  const createMutation = useMutation({
    mutationFn: () =>
      api.post<Task>(`/api/v1/cases/${caseId}/tasks`, {
        title: newTitle,
        description: newDesc,
        priority: newPriority,
        due_date: newDueDate ? new Date(newDueDate).toISOString() : undefined,
        assigned_to: newAssignedTo || undefined,
      }),
    onSuccess: () => {
      setShowCreate(false);
      setNewTitle("");
      setNewTitleTouched(false);
      setNewDesc("");
      setNewPriority("medium");
      setNewDueDate("");
      setNewAssignedTo("");
      setCreateError("");
      refetch();
    },
    onError: (err) => {
      setCreateError(err instanceof APIError ? err.message : "Failed to create task");
    },
  });

  const [justCompleted, setJustCompleted] = useState<Set<string>>(new Set());
  const [undoToast, setUndoToast] = useState<{ taskId: string; title: string } | null>(null);
  const completionTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  useEffect(() => {
    const timers = completionTimers.current;
    return () => { timers.forEach((t) => clearTimeout(t)); };
  }, []);

  const undoMutation = useMutation({
    mutationFn: (taskId: string) => api.put<Task>(`/api/v1/tasks/${taskId}`, { status: "todo" }),
    onSuccess: (_data, taskId) => {
      const timer = completionTimers.current.get(taskId);
      if (timer) { clearTimeout(timer); completionTimers.current.delete(taskId); }
      setJustCompleted((prev) => { const next = new Set(prev); next.delete(taskId); return next; });
      setUndoToast(null);
      refetch();
    },
  });

  const dismissUndo = useCallback((taskId: string) => {
    const timer = completionTimers.current.get(taskId);
    if (timer) { clearTimeout(timer); completionTimers.current.delete(taskId); }
    setJustCompleted((prev) => { const next = new Set(prev); next.delete(taskId); return next; });
    setUndoToast(null);
    refetch();
  }, [refetch]);

  const completeMutation = useMutation({
    mutationFn: (taskId: string) => api.post<Task>(`/api/v1/tasks/${taskId}/complete`, {}),
    onSuccess: (_data, taskId) => {
      const task = tasks.find((t) => t.id === taskId);
      setJustCompleted((prev) => new Set([...prev, taskId]));
      setUndoToast({ taskId, title: task?.title ?? "Task" });
      const timer = setTimeout(() => dismissUndo(taskId), 5000);
      completionTimers.current.set(taskId, timer);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (taskId: string) => api.delete(`/api/v1/tasks/${taskId}`),
    onSuccess: () => refetch(),
  });

  const assignToMeMutation = useMutation({
    mutationFn: (taskId: string) => api.put<Task>(`/api/v1/tasks/${taskId}`, { assigned_to: user?.id }),
    onMutate: async (taskId) => {
      await qc.cancelQueries({ queryKey: ["case-tasks", caseId] });
      const queryKey = ["case-tasks", caseId, taskTab, filterPriority, overdueOnly];
      const prev = qc.getQueryData(queryKey);
      qc.setQueryData(queryKey, (old: { pages: { data: Task[]; meta: ListMeta }[]; pageParams: unknown[] } | undefined) =>
        old ? {
          ...old,
          pages: old.pages.map((page) => ({
            ...page,
            data: page.data.map((t) =>
              t.id === taskId
                ? { ...t, assigned_to: { id: user?.id ?? "", full_name: user?.full_name ?? "", email: user?.email ?? "" } }
                : t
            ),
          })),
        } : old
      );
      return { prev, queryKey };
    },
    onError: (_err: unknown, _taskId: string, ctx: { prev: unknown; queryKey: unknown[] } | undefined) => {
      if (ctx?.prev) qc.setQueryData(ctx.queryKey, ctx.prev);
    },
    onSettled: () => qc.invalidateQueries({ queryKey: ["case-tasks", caseId] }),
  });

  const unassignMutation = useMutation({
    mutationFn: (taskId: string) => api.put<Task>(`/api/v1/tasks/${taskId}`, { clear_assigned_to: true }),
    onMutate: async (taskId) => {
      await qc.cancelQueries({ queryKey: ["case-tasks", caseId] });
      const queryKey = ["case-tasks", caseId, taskTab, filterPriority, overdueOnly];
      const prev = qc.getQueryData(queryKey);
      qc.setQueryData(queryKey, (old: { pages: { data: Task[]; meta: ListMeta }[]; pageParams: unknown[] } | undefined) =>
        old ? {
          ...old,
          pages: old.pages.map((page) => ({
            ...page,
            data: page.data.map((t) => t.id === taskId ? { ...t, assigned_to: null } : t),
          })),
        } : old
      );
      return { prev, queryKey };
    },
    onError: (_err: unknown, _taskId: string, ctx: { prev: unknown; queryKey: unknown[] } | undefined) => {
      if (ctx?.prev) qc.setQueryData(ctx.queryKey, ctx.prev);
    },
    onSettled: () => qc.invalidateQueries({ queryKey: ["case-tasks", caseId] }),
  });

  const canCreate = hasRole("analyst");

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-6 h-6 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="p-6">
        <CaseSubNav caseId={caseId} activeTab="tasks" caseNumber={caseData?.case_number} />
        <PageError error={error as Error | null} onRetry={() => refetch()} />
      </div>
    );
  }

  return (
    <div className="p-6">
      <CaseSubNav caseId={caseId} activeTab="tasks" caseNumber={caseData?.case_number} />

      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-white text-xl font-bold">Tasks</h1>
          {caseData && (
            <p className="text-slate-500 text-sm mt-0.5 font-mono">{caseData.case_number} · {caseData.title}</p>
          )}
        </div>
        {canCreate && (
          <button
            onClick={() => setShowCreate(true)}
            className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-2 rounded-lg text-sm transition-colors flex items-center gap-1.5"
          >
            <span className="text-lg leading-none">+</span> New Task
          </button>
        )}
      </div>

      {/* Active / Done tab bar */}
      <div className="flex items-center gap-1 mb-4 border-b border-slate-700/50">
        <button
          onClick={() => setTaskTab("active")}
          className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
            taskTab === "active" ? "border-cyan-400 text-cyan-400" : "border-transparent text-slate-500 hover:text-slate-300"
          }`}
        >
          Active
        </button>
        <button
          onClick={() => setTaskTab("done")}
          className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
            taskTab === "done" ? "border-green-400 text-green-400" : "border-transparent text-slate-500 hover:text-slate-300"
          }`}
        >
          Done
        </button>
        <div className="ml-auto flex items-center gap-3 pb-1">
          {taskTab === "active" && (
            <>
              <select
                value={filterPriority}
                onChange={(e) => setFilterPriority(e.target.value)}
                className="bg-[#131f30] border border-slate-700 text-slate-300 text-xs rounded-lg px-2.5 py-1.5 focus:outline-none focus:border-cyan-500"
              >
                <option value="">All priorities</option>
                {PRIORITY_OPTIONS.map((p) => (
                  <option key={p} value={p}>{p}</option>
                ))}
              </select>
              <label className="flex items-center gap-1.5 text-xs text-slate-400 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={overdueOnly}
                  onChange={(e) => setOverdueOnly(e.target.checked)}
                  className="accent-cyan-400"
                />
                Overdue only
              </label>
            </>
          )}
          <span className="text-slate-600 text-xs">{totalCount} task{totalCount !== 1 ? "s" : ""}</span>
        </div>
      </div>

      {/* Undo toast */}
      {undoToast && (
        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-3 bg-slate-800 border border-slate-600 rounded-xl px-4 py-3 shadow-2xl text-sm">
          <svg className="w-4 h-4 text-green-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
          </svg>
          <span className="text-slate-300">
            <span className="text-white font-medium truncate max-w-[200px] inline-block align-bottom">{undoToast.title}</span>
            {" "}marked done
          </span>
          <button
            onClick={() => undoMutation.mutate(undoToast.taskId)}
            disabled={undoMutation.isPending}
            className="ml-1 px-3 py-1 rounded-lg bg-cyan-500/20 hover:bg-cyan-500/30 text-cyan-400 font-semibold transition-colors disabled:opacity-50"
          >
            {undoMutation.isPending ? "Undoing…" : "Undo"}
          </button>
          <button
            onClick={() => dismissUndo(undoToast.taskId)}
            className="text-slate-500 hover:text-slate-300 transition-colors"
            aria-label="Dismiss"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      )}

      {/* Task list */}
      {tasks.length === 0 ? (
        <div className="bg-[#131f30] border border-slate-700/50 rounded-xl p-16 text-center">
          {taskTab === "done" ? (
            <>
              <div className="text-4xl mb-3">✓</div>
              <p className="text-slate-500 text-sm font-medium">No completed tasks yet.</p>
            </>
          ) : (
            <>
              <div className="text-4xl mb-3">📋</div>
              <p className="text-slate-400 text-sm font-medium">No active tasks.</p>
              <p className="text-slate-600 text-xs mt-1">Track work items and assign them to team members.</p>
              {canCreate && (
                <button
                  onClick={() => setShowCreate(true)}
                  className="mt-3 text-cyan-400 hover:text-cyan-300 text-sm font-medium"
                >
                  Create the first task →
                </button>
              )}
            </>
          )}
        </div>
      ) : (
        <div className="space-y-3">
          {tasks.map((task) => {
            const displayTask = justCompleted.has(task.id) ? { ...task, status: "done" } : task;
            return (
              <TaskCard
                key={task.id}
                task={displayTask}
                onClick={() => router.push(`/cases/${caseId}/tasks/${task.id}`)}
                onComplete={justCompleted.has(task.id) ? undefined : () => completeMutation.mutate(task.id)}
                onDelete={() => deleteMutation.mutate(task.id)}
                onAssignToMe={!task.assigned_to && !justCompleted.has(task.id) && taskTab === "active" ? () => assignToMeMutation.mutate(task.id) : undefined}
                onUnassign={task.assigned_to && !justCompleted.has(task.id) && taskTab === "active" ? () => unassignMutation.mutate(task.id) : undefined}
              />
            );
          })}

          {hasNextPage && (
            <div className="text-center pt-2">
              <button
                onClick={() => fetchNextPage()}
                disabled={isFetchingNextPage}
                className="text-cyan-400 hover:text-cyan-300 text-sm font-medium disabled:opacity-50"
              >
                {isFetchingNextPage ? "Loading…" : "Load more"}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Create task modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-6 w-full max-w-lg">
            <h3 className="text-white font-semibold mb-4">New Task</h3>

            <div className="space-y-3">
              <div>
                <label className="text-slate-400 text-xs mb-1 block">Title <span className="text-red-400">*</span></label>
                <input
                  value={newTitle}
                  onChange={(e) => setNewTitle(e.target.value)}
                  onBlur={() => setNewTitleTouched(true)}
                  placeholder="Task title…"
                  className={`w-full bg-[#080e1a] border rounded-lg px-3 py-2 text-sm focus:outline-none placeholder:text-slate-500 text-white ${
                    newTitleTouched && !newTitle.trim()
                      ? "border-red-500 focus:border-red-400"
                      : "border-slate-600 focus:border-cyan-500"
                  }`}
                  autoFocus
                />
                {newTitleTouched && !newTitle.trim() && (
                  <p className="text-red-400 text-[10px] mt-0.5">Title is required.</p>
                )}
              </div>

              <div>
                <label className="text-slate-400 text-xs mb-1 block">Description</label>
                <textarea
                  value={newDesc}
                  onChange={(e) => setNewDesc(e.target.value)}
                  rows={3}
                  placeholder="Optional description…"
                  className="w-full bg-[#080e1a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500 resize-none"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-slate-400 text-xs mb-1 block">Priority</label>
                  <select
                    value={newPriority}
                    onChange={(e) => setNewPriority(e.target.value)}
                    className="w-full bg-[#080e1a] border border-slate-600 text-white rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
                  >
                    {PRIORITY_OPTIONS.map((p) => (
                      <option key={p} value={p}>{p}</option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="text-slate-400 text-xs mb-1 block">Due Date</label>
                  <input
                    type="date"
                    value={newDueDate}
                    min={todayStr}
                    onChange={(e) => {
                      setNewDueDate(e.target.value);
                      if (e.target.value && e.target.value < todayStr) {
                        setCreateError("Due date cannot be in the past.");
                      } else {
                        setCreateError("");
                      }
                    }}
                    className={`w-full bg-[#080e1a] border rounded-lg px-3 py-2 text-sm focus:outline-none [color-scheme:dark] ${
                      newDueDate && newDueDate < todayStr
                        ? "border-red-500 text-red-400 focus:border-red-400"
                        : "border-slate-600 text-white focus:border-cyan-500"
                    }`}
                  />
                  {newDueDate && newDueDate < todayStr && (
                    <p className="text-red-400 text-[10px] mt-0.5">Must be today or a future date.</p>
                  )}
                </div>
              </div>

              <div>
                <label className="text-slate-400 text-xs mb-1 block">Assign To</label>
                <select
                  value={newAssignedTo}
                  onChange={(e) => setNewAssignedTo(e.target.value)}
                  className="w-full bg-[#080e1a] border border-slate-600 text-white rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
                >
                  <option value="">— Unassigned —</option>
                  {users.map((u) => (
                    <option key={u.id} value={u.id}>{u.full_name} [{u.role.replace("_", " ")}] · {u.email}</option>
                  ))}
                </select>
              </div>
            </div>

            {createError && (
              <p className="text-red-400 text-xs mt-3">{createError}</p>
            )}

            <div className="flex gap-2 mt-4 items-center">
              {(() => {
                const duePast = !!(newDueDate && newDueDate < todayStr);
                const missingTitle = !newTitle.trim();
                const isDisabled = missingTitle || duePast || createMutation.isPending;
                const tooltip = missingTitle
                  ? "Title is required before creating a task."
                  : duePast
                  ? "Due date cannot be in the past."
                  : "";
                return (
                  <span title={tooltip} className={isDisabled ? "cursor-not-allowed" : ""}>
                    <button
                      onClick={() => { setNewTitleTouched(true); if (!isDisabled) createMutation.mutate(); }}
                      disabled={isDisabled}
                      className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-2 rounded-lg text-sm disabled:opacity-50 disabled:pointer-events-none"
                    >
                      {createMutation.isPending ? "Creating…" : "Create Task"}
                    </button>
                  </span>
                );
              })()}
              <button
                onClick={() => { setShowCreate(false); setCreateError(""); setNewTitleTouched(false); }}
                className="text-slate-400 hover:text-slate-200 px-4 py-2 text-sm"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
