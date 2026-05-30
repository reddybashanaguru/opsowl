"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "next/navigation";
import { Suspense } from "react";
import { api } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { useConfig } from "@/lib/hooks/useConfig";
import { TaskCard, Task } from "@/components/tasks/TaskCard";

interface ListMeta {
  total_count: number;
  has_more: boolean;
  next_cursor: string | null;
}

type Group = "overdue" | "due_today" | "this_week" | "later" | "done";

const GROUP_CONFIG: Record<Group, { label: string; color: string; bgColor: string; dotColor: string }> = {
  overdue:   { label: "Overdue",       color: "text-red-400",    bgColor: "bg-red-900/10",    dotColor: "bg-red-400" },
  due_today: { label: "Due Today",     color: "text-amber-400",  bgColor: "bg-amber-900/10",  dotColor: "bg-amber-400" },
  this_week: { label: "Due This Week", color: "text-cyan-400",   bgColor: "bg-cyan-900/10",   dotColor: "bg-cyan-400" },
  later:     { label: "Later",         color: "text-slate-400",  bgColor: "bg-transparent",   dotColor: "bg-slate-500" },
  done:      { label: "Done",          color: "text-green-400",  bgColor: "bg-transparent",   dotColor: "bg-green-400" },
};

const GROUP_ORDER: Group[] = ["overdue", "due_today", "this_week", "later", "done"];

function classifyTask(task: Task): Group {
  if (task.status === "done" || task.status === "cancelled") return "done";
  if (task.is_overdue) return "overdue";
  if (!task.due_date) return "later";

  const due = new Date(task.due_date);
  const now = new Date();
  const endOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 23, 59, 59);
  const endOfWeek = new Date(now);
  endOfWeek.setDate(endOfWeek.getDate() + 7);

  if (due <= endOfToday) return "due_today";
  if (due <= endOfWeek) return "this_week";
  return "later";
}

function ProgressRing({ pct, size = 44 }: { pct: number; size?: number }) {
  const r = (size - 6) / 2;
  const circ = 2 * Math.PI * r;
  const dash = (pct / 100) * circ;
  const color = pct === 100 ? "#4ade80" : pct >= 60 ? "#22d3ee" : pct >= 30 ? "#f59e0b" : "#f87171";
  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
      <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="#334155" strokeWidth={4} />
      <circle
        cx={size / 2} cy={size / 2} r={r} fill="none" stroke={color} strokeWidth={4}
        strokeDasharray={`${dash} ${circ}`} strokeLinecap="round"
        style={{ transform: "rotate(-90deg)", transformOrigin: `${size / 2}px ${size / 2}px`, transition: "stroke-dasharray 0.5s ease" }}
      />
      <text x={size / 2} y={size / 2 + 4} textAnchor="middle" fill={color} fontSize="9" fontWeight="bold" fontFamily="monospace">
        {Math.round(pct)}%
      </text>
    </svg>
  );
}

const PRIVILEGED_ROLES = ["manager", "admin", "super_admin"];

function SkeletonCard() {
  return (
    <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4 space-y-2.5">
      <div className="h-4 w-24 bg-slate-700 rounded animate-pulse" />
      <div className="h-4 w-3/4 bg-slate-700/80 rounded animate-pulse" />
      <div className="h-3 w-full bg-slate-800 rounded animate-pulse" />
      <div className="flex gap-2 mt-1">
        <div className="h-5 w-14 bg-slate-800 rounded animate-pulse" />
        <div className="h-5 w-16 bg-slate-800 rounded animate-pulse" />
      </div>
    </div>
  );
}

function TasksPageContent() {
  const router = useRouter();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const cfg = useConfig();
  const searchParams = useSearchParams();

  const isPrivileged = PRIVILEGED_ROLES.includes(user?.role ?? "");

  const scopeParam = searchParams.get("scope");
  const defaultAll = isPrivileged ? scopeParam !== "mine" : false;

  const [viewAll, setViewAll] = useState(defaultAll);
  const [filterAssignee, setFilterAssignee] = useState("");
  const [unassignedOnly, setUnassignedOnly] = useState(false);

  const buildApiUrl = () => {
    const p = new URLSearchParams({ limit: "200" });
    if (viewAll) {
      p.set("scope", "all");
      if (filterAssignee) p.set("assigned_to", filterAssignee);
      if (unassignedOnly) p.set("unassigned", "true");
    }
    return `/api/v1/tasks?${p.toString()}`;
  };

  const { data: usersData } = useQuery<{ data: { id: string; full_name: string }[] }>({
    queryKey: ["users-list"],
    queryFn: () => api.get<{ data: { id: string; full_name: string }[] }>("/api/v1/users?limit=100"),
    staleTime: 5 * 60 * 1000,
    enabled: isPrivileged,
  });
  const usersList = usersData?.data ?? [];

  const { data, isLoading, refetch } = useQuery<{ data: Task[]; meta: ListMeta }>({
    queryKey: ["my-tasks", viewAll, filterAssignee, unassignedOnly],
    queryFn: () => api.getList<Task, ListMeta>(buildApiUrl()),
    enabled: !!user,
  });

  const assignMutation = useMutation({
    mutationFn: (taskId: string) => api.put<Task>(`/api/v1/tasks/${taskId}`, { assigned_to: user?.id }),
    onMutate: async (taskId) => {
      await qc.cancelQueries({ queryKey: ["my-tasks"] });
      const queryKey = ["my-tasks", viewAll, filterAssignee, unassignedOnly];
      const prev = qc.getQueryData(queryKey);
      qc.setQueryData(queryKey, (old: { data: Task[]; meta: ListMeta } | undefined) =>
        old ? {
          ...old,
          data: old.data.map((t) =>
            t.id === taskId
              ? { ...t, assigned_to: { id: user?.id ?? "", full_name: user?.full_name ?? user?.email ?? "", email: user?.email ?? "" } }
              : t
          ),
        } : old
      );
      return { prev, queryKey };
    },
    onError: (_err: unknown, _taskId: string, ctx: { prev: unknown; queryKey: unknown[] } | undefined) => {
      if (ctx?.prev) qc.setQueryData(ctx.queryKey, ctx.prev);
    },
    onSettled: () => qc.invalidateQueries({ queryKey: ["my-tasks"] }),
  });

  // Tasks completed optimistically — stay in their current group with done styling,
  // then move to Done section after 5s (undo window). User can undo within that window.
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

  const [doneTab, setDoneTab] = useState(false);

  const completeMutation = useMutation({
    mutationFn: (taskId: string) => api.post<Task>(`/api/v1/tasks/${taskId}/complete`, {}),
    onSuccess: (_data, taskId) => {
      const task = tasks.find((t) => t.id === taskId);
      setJustCompleted((prev) => new Set([...prev, taskId]));
      setUndoToast({ taskId, title: task?.title ?? "Task" });
      qc.invalidateQueries({ queryKey: ["dashboard-stats"] });

      // After 5s move the task to the Done group — user loses undo window
      const timer = setTimeout(() => dismissUndo(taskId), 5000);
      completionTimers.current.set(taskId, timer);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (taskId: string) => api.delete(`/api/v1/tasks/${taskId}`),
    onSuccess: () => refetch(),
  });

  const unassignMutation = useMutation({
    mutationFn: (taskId: string) => api.put<Task>(`/api/v1/tasks/${taskId}`, { clear_assigned_to: true }),
    onMutate: async (taskId) => {
      await qc.cancelQueries({ queryKey: ["my-tasks"] });
      const queryKey = ["my-tasks", viewAll, filterAssignee, unassignedOnly];
      const prev = qc.getQueryData(queryKey);
      qc.setQueryData(queryKey, (old: { data: Task[]; meta: ListMeta } | undefined) =>
        old ? { ...old, data: old.data.map((t) => t.id === taskId ? { ...t, assigned_to: null } : t) } : old
      );
      return { prev, queryKey };
    },
    onError: (_err: unknown, _taskId: string, ctx: { prev: unknown; queryKey: unknown[] } | undefined) => {
      if (ctx?.prev) qc.setQueryData(ctx.queryKey, ctx.prev);
    },
    onSettled: () => qc.invalidateQueries({ queryKey: ["my-tasks"] }),
  });

  const tasks = data?.data ?? [];
  const done = tasks.filter((t) => t.status === "done" || t.status === "cancelled").length;
  const active = tasks.filter((t) => t.status !== "done" && t.status !== "cancelled").length;
  const overdue = tasks.filter((t) => t.is_overdue && t.status !== "done" && t.status !== "cancelled").length;
  const pct = tasks.length === 0 ? 0 : (done / tasks.length) * 100;

  const groups = GROUP_ORDER.reduce(
    (acc, g) => ({ ...acc, [g]: [] as Task[] }),
    {} as Record<Group, Task[]>,
  );
  tasks.forEach((t) => groups[classifyTask(t)].push(t));

  return (
    <div className="flex flex-col" style={{ height: "100%" }}>
      {/* ── Undo toast ────────────────────────────────────── */}
      {undoToast && (
        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-3 bg-slate-800 border border-slate-600 rounded-xl px-4 py-3 shadow-2xl text-sm animate-in fade-in slide-in-from-bottom-2">
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

      {/* ── Sticky header ─────────────────────────────────── */}
      <div className="sticky top-0 z-10 flex-shrink-0 px-6 pt-6 pb-4 border-b border-slate-700/50 bg-[#0f172a]">
        <div className="max-w-7xl mx-auto flex items-center justify-between gap-4">
          {/* Left: title + toggle + stats */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-3 mb-1">
              <h1 className="text-white text-xl font-bold">
                {viewAll ? `All ${cfg.tasks_label}` : `My ${cfg.tasks_label}`}
              </h1>
              {isPrivileged && (
                <div className="flex items-center bg-slate-800 border border-slate-700 rounded-lg p-0.5 text-xs">
                  <button
                    onClick={() => setViewAll(false)}
                    className={`px-3 py-1 rounded-md transition-colors font-medium ${
                      !viewAll ? "bg-slate-600 text-white" : "text-slate-400 hover:text-slate-200"
                    }`}
                  >
                    Mine
                  </button>
                  <button
                    onClick={() => setViewAll(true)}
                    className={`px-3 py-1 rounded-md transition-colors font-medium ${
                      viewAll ? "bg-cyan-600 text-white" : "text-slate-400 hover:text-slate-200"
                    }`}
                  >
                    All
                  </button>
                </div>
              )}
            </div>
            <p className="text-slate-500 text-sm">
              <span className="text-slate-300 font-medium">{active}</span> active
              {" · "}
              <span className="text-slate-300 font-medium">{done}</span> completed
              {" · "}
              <span className="text-slate-400">{tasks.length} total</span>
              {overdue > 0 && (
                <span className="text-red-400 ml-2 font-medium">· {overdue} overdue</span>
              )}
            </p>
          </div>

          {/* Right: progress ring + bar */}
          <div className="flex items-center gap-4">
            {tasks.length > 0 && (
              <div className="hidden sm:block w-40">
                <div className="flex justify-between text-xs text-slate-500 mb-1">
                  <span>Completion</span>
                  <span className="tabular-nums">{Math.round(pct)}%</span>
                </div>
                <div className="h-1.5 bg-slate-800 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-gradient-to-r from-cyan-500 to-green-400 rounded-full transition-all duration-700"
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>
            )}
            <ProgressRing pct={pct} size={44} />
          </div>
        </div>

        {/* Active / Done tab bar */}
        <div className="max-w-7xl mx-auto flex items-center gap-1 mt-3 border-b border-slate-700/50 pb-0">
          <button
            onClick={() => setDoneTab(false)}
            className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
              !doneTab ? "border-cyan-400 text-cyan-400" : "border-transparent text-slate-500 hover:text-slate-300"
            }`}
          >
            Active
            {active > 0 && (
              <span className={`ml-1.5 text-xs px-1.5 py-0.5 rounded-full ${!doneTab ? "bg-cyan-400/20 text-cyan-400" : "bg-slate-700 text-slate-500"}`}>
                {active}
              </span>
            )}
          </button>
          <button
            onClick={() => setDoneTab(true)}
            className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
              doneTab ? "border-green-400 text-green-400" : "border-transparent text-slate-500 hover:text-slate-300"
            }`}
          >
            Done
            {done > 0 && (
              <span className={`ml-1.5 text-xs px-1.5 py-0.5 rounded-full ${doneTab ? "bg-green-400/20 text-green-400" : "bg-slate-700 text-slate-500"}`}>
                {done}
              </span>
            )}
          </button>
        </div>

        {/* All-view filters: assignee + unassigned */}
        {viewAll && isPrivileged && !doneTab && (
          <div className="max-w-7xl mx-auto flex items-center gap-3 mt-3 flex-wrap">
            <select
              value={filterAssignee}
              onChange={(e) => { setFilterAssignee(e.target.value); setUnassignedOnly(false); }}
              className="bg-slate-800 border border-slate-700 text-slate-300 text-xs rounded-lg px-2.5 py-1.5 focus:outline-none focus:border-cyan-500"
            >
              <option value="">All assignees</option>
              {usersList.map((u) => (
                <option key={u.id} value={u.id}>{u.full_name}</option>
              ))}
            </select>
            <label className="flex items-center gap-1.5 text-xs text-slate-400 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={unassignedOnly}
                onChange={(e) => { setUnassignedOnly(e.target.checked); setFilterAssignee(""); }}
                className="accent-cyan-400"
              />
              Unassigned only
            </label>
            {(filterAssignee || unassignedOnly) && (
              <button
                onClick={() => { setFilterAssignee(""); setUnassignedOnly(false); }}
                className="text-xs text-slate-500 hover:text-slate-300 transition-colors"
              >
                Clear filters ✕
              </button>
            )}
          </div>
        )}
      </div>

      {/* ── Scrollable task body ───────────────────────────── */}
      <div className="flex-1 min-h-0 overflow-y-auto">
        <div className="max-w-7xl mx-auto px-6 py-5">
          {isLoading ? (
            <div className="space-y-6">
              {[...Array(2)].map((_, g) => (
                <div key={g}>
                  <div className="h-6 w-32 bg-slate-700 animate-pulse rounded mb-3" />
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {[...Array(4)].map((_, i) => <SkeletonCard key={i} />)}
                  </div>
                </div>
              ))}
            </div>
          ) : doneTab ? (
            /* ── Done tab ─────────────────────────────────────── */
            groups.done.length === 0 ? (
              <div className="text-center py-24">
                <div className="text-5xl mb-4">✓</div>
                <p className="text-slate-300 text-sm font-medium">Nothing completed yet.</p>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {groups.done.map((task) => (
                  <TaskCard
                    key={task.id}
                    task={task}
                    showCase
                    onClick={() => router.push(`/cases/${task.case_id}/tasks/${task.id}`)}
                    onDelete={() => { if (window.confirm(`Delete "${task.title}"?`)) deleteMutation.mutate(task.id); }}
                  />
                ))}
              </div>
            )
          ) : (
            /* ── Active tab ───────────────────────────────────── */
            <div className="space-y-6">
              {(["overdue", "due_today", "this_week", "later"] as const).map((group) => {
                const displayTasks = groups[group];
                if (displayTasks.length === 0) return null;
                const gcfg = GROUP_CONFIG[group];
                return (
                  <div key={group}>
                    <div className={`flex items-center gap-2 mb-3 py-1.5 px-2 rounded-lg ${gcfg.bgColor}`}>
                      <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${gcfg.dotColor}`} />
                      <h2 className={`text-xs font-bold uppercase tracking-widest ${gcfg.color}`}>
                        {gcfg.label}
                      </h2>
                      <span className={`text-xs font-semibold px-1.5 py-0.5 rounded-full bg-slate-800/80 ${gcfg.color}`}>
                        {displayTasks.length}
                      </span>
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                      {displayTasks.map((task) => {
                        const displayTask = justCompleted.has(task.id) ? { ...task, status: "done" } : task;
                        return (
                          <TaskCard
                            key={task.id}
                            task={displayTask}
                            showCase
                            onClick={() => router.push(`/cases/${task.case_id}/tasks/${task.id}`)}
                            onComplete={justCompleted.has(task.id) ? undefined : () => completeMutation.mutate(task.id)}
                            onDelete={() => { if (window.confirm(`Delete "${task.title}"?`)) deleteMutation.mutate(task.id); }}
                            onAssignToMe={viewAll && !task.assigned_to && !justCompleted.has(task.id) ? () => assignMutation.mutate(task.id) : undefined}
                            onUnassign={task.assigned_to && !justCompleted.has(task.id) ? () => unassignMutation.mutate(task.id) : undefined}
                          />
                        );
                      })}
                    </div>
                  </div>
                );
              })}
              {active === 0 && (
                <div className="text-center py-24">
                  <div className="text-5xl mb-4">✓</div>
                  <p className="text-slate-300 text-sm font-medium">All caught up!</p>
                  <p className="text-slate-500 text-xs mt-1">
                    {viewAll
                      ? `No active ${cfg.tasks_label.toLowerCase()} across all cases.`
                      : `No active ${cfg.tasks_label.toLowerCase()} assigned to you.`}
                  </p>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function MyTasksPage() {
  return (
    <Suspense fallback={
      <div className="flex flex-col h-full overflow-hidden">
        <div className="flex-shrink-0 px-6 pt-6 pb-4 border-b border-slate-700/50">
          <div className="max-w-7xl mx-auto">
            <div className="h-7 w-40 bg-slate-700 animate-pulse rounded mb-2" />
            <div className="h-4 w-56 bg-slate-800 animate-pulse rounded" />
          </div>
        </div>
        <div className="flex-1 overflow-y-auto px-6 py-5">
          <div className="max-w-7xl mx-auto grid grid-cols-1 md:grid-cols-2 gap-3">
            {[...Array(8)].map((_, i) => (
              <div key={i} className="h-28 bg-slate-800 animate-pulse rounded-xl" />
            ))}
          </div>
        </div>
      </div>
    }>
      <TasksPageContent />
    </Suspense>
  );
}
