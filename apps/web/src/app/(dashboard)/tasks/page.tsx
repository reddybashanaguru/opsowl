"use client";

import { useState, useRef, useEffect } from "react";
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

  const apiUrl = viewAll ? "/api/v1/tasks?limit=200&scope=all" : "/api/v1/tasks?limit=200";

  const { data, isLoading, refetch } = useQuery<{ data: Task[]; meta: ListMeta }>({
    queryKey: ["my-tasks", viewAll],
    queryFn: () => api.getList<Task, ListMeta>(apiUrl),
    enabled: !!user,
  });

  // Tasks completed optimistically — stay in their current group with done styling,
  // then move to Done section after a short delay so the UI doesn't jerk.
  const [justCompleted, setJustCompleted] = useState<Set<string>>(new Set());
  const completionTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  useEffect(() => {
    const timers = completionTimers.current;
    return () => { timers.forEach((t) => clearTimeout(t)); };
  }, []);

  const completeMutation = useMutation({
    mutationFn: (taskId: string) => api.post<Task>(`/api/v1/tasks/${taskId}/complete`, {}),
    onSuccess: (_data, taskId) => {
      // Show completed styling in place immediately
      setJustCompleted((prev) => new Set([...prev, taskId]));
      qc.invalidateQueries({ queryKey: ["dashboard-stats"] });

      // After 1.5s move the task to the Done group by refetching
      const timer = setTimeout(() => {
        setJustCompleted((prev) => {
          const next = new Set(prev);
          next.delete(taskId);
          return next;
        });
        completionTimers.current.delete(taskId);
        refetch();
      }, 1500);
      completionTimers.current.set(taskId, timer);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (taskId: string) => api.delete(`/api/v1/tasks/${taskId}`),
    onSuccess: () => refetch(),
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
          ) : tasks.length === 0 ? (
            <div className="text-center py-24">
              <div className="text-5xl mb-4">✓</div>
              <p className="text-slate-300 text-sm font-medium">All caught up!</p>
              <p className="text-slate-500 text-xs mt-1">
                {viewAll
                  ? `No ${cfg.tasks_label.toLowerCase()} found across all cases.`
                  : `No ${cfg.tasks_label.toLowerCase()} assigned to you.`}
              </p>
            </div>
          ) : (
            <div className="space-y-6">
              {GROUP_ORDER.map((group) => {
                const groupTasks = groups[group];
                if (groupTasks.length === 0) return null;
                const gcfg = GROUP_CONFIG[group];
                return (
                  <div key={group}>
                    {/* Group header — full width */}
                    <div className={`flex items-center gap-2 mb-3 py-1.5 px-2 rounded-lg ${gcfg.bgColor}`}>
                      <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${gcfg.dotColor}`} />
                      <h2 className={`text-xs font-bold uppercase tracking-widest ${gcfg.color}`}>
                        {gcfg.label}
                      </h2>
                      <span className={`text-xs font-semibold px-1.5 py-0.5 rounded-full bg-slate-800/80 ${gcfg.color}`}>
                        {groupTasks.length}
                      </span>
                    </div>
                    {/* 2-column card grid */}
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                      {groupTasks.map((task) => {
                        const displayTask = justCompleted.has(task.id)
                          ? { ...task, status: "done" }
                          : task;
                        return (
                          <TaskCard
                            key={task.id}
                            task={displayTask}
                            showCase
                            onClick={() => router.push(`/cases/${task.case_id}/tasks/${task.id}`)}
                            onComplete={justCompleted.has(task.id) ? undefined : () => completeMutation.mutate(task.id)}
                            onDelete={() => deleteMutation.mutate(task.id)}
                          />
                        );
                      })}
                    </div>
                  </div>
                );
              })}
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
