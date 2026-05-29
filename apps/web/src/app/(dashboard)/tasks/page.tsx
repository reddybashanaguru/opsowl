"use client";

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

function ProgressRing({ pct, size = 36 }: { pct: number; size?: number }) {
  const r = (size - 6) / 2;
  const circ = 2 * Math.PI * r;
  const dash = (pct / 100) * circ;
  const color = pct === 100 ? "#4ade80" : pct >= 60 ? "#22d3ee" : pct >= 30 ? "#f59e0b" : "#f87171";
  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
      <circle cx={size/2} cy={size/2} r={r} fill="none" stroke="#1e293b" strokeWidth={4} />
      <circle
        cx={size/2} cy={size/2} r={r} fill="none" stroke={color} strokeWidth={4}
        strokeDasharray={`${dash} ${circ}`} strokeLinecap="round"
        style={{ transform: `rotate(-90deg)`, transformOrigin: `${size/2}px ${size/2}px`, transition: "stroke-dasharray 0.5s ease" }}
      />
      <text x={size/2} y={size/2+4} textAnchor="middle" fill={color} fontSize="9" fontWeight="bold" fontFamily="monospace">
        {Math.round(pct)}%
      </text>
    </svg>
  );
}

function TasksPageContent() {
  const router = useRouter();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const cfg = useConfig();
  const searchParams = useSearchParams();

  const scopeAll = searchParams.get("scope") === "all";
  const apiUrl = scopeAll ? "/api/v1/tasks?limit=200&scope=all" : "/api/v1/tasks?limit=200";

  const { data, isLoading, refetch } = useQuery<{ data: Task[]; meta: ListMeta }>({
    queryKey: ["my-tasks", scopeAll],
    queryFn: () => api.getList<Task, ListMeta>(apiUrl),
    enabled: !!user,
  });

  const completeMutation = useMutation({
    mutationFn: (taskId: string) => api.post<Task>(`/api/v1/tasks/${taskId}/complete`, {}),
    onSuccess: () => { refetch(); qc.invalidateQueries({ queryKey: ["dashboard-stats"] }); },
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

  if (isLoading) {
    return (
      <div className="p-6 max-w-3xl mx-auto">
        <div className="h-7 w-32 bg-slate-700 animate-pulse rounded mb-6" />
        <div className="space-y-2">
          {[...Array(6)].map((_, i) => <div key={i} className="h-16 bg-slate-800 animate-pulse rounded-xl" />)}
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-3xl mx-auto">
      {/* Header with progress summary */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-white text-xl font-bold">{scopeAll ? `All ${cfg.tasks_label}` : `My ${cfg.tasks_label}`}</h1>
          <p className="text-slate-500 text-sm mt-1">
            {active} active · {done} completed · {tasks.length} total
            {overdue > 0 && <span className="text-red-400 ml-2">· {overdue} overdue</span>}
          </p>
        </div>
        <div className="flex flex-col items-center gap-1">
          <ProgressRing pct={pct} size={52} />
          <span className="text-slate-600 text-xs">completion</span>
        </div>
      </div>

      {/* Progress bar */}
      {tasks.length > 0 && (
        <div className="mb-6">
          <div className="h-1.5 bg-slate-800 rounded-full overflow-hidden">
            <div
              className="h-full bg-gradient-to-r from-cyan-500 to-green-400 rounded-full transition-all duration-700"
              style={{ width: `${pct}%` }}
            />
          </div>
        </div>
      )}

      {tasks.length === 0 ? (
        <div className="text-center py-20">
          <div className="text-5xl mb-4">✓</div>
          <p className="text-slate-300 text-sm font-medium">All caught up!</p>
          <p className="text-slate-600 text-xs mt-1">{scopeAll ? `No ${cfg.tasks_label.toLowerCase()} found across all cases.` : `No ${cfg.tasks_label.toLowerCase()} assigned to you.`}</p>
        </div>
      ) : (
        <div className="space-y-7">
          {GROUP_ORDER.map((group) => {
            const groupTasks = groups[group];
            if (groupTasks.length === 0) return null;
            const cfg = GROUP_CONFIG[group];
            return (
              <div key={group}>
                <div className={`flex items-center gap-2 mb-2.5 py-1.5 px-2 rounded-lg ${cfg.bgColor}`}>
                  <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${cfg.dotColor}`} />
                  <h2 className={`text-xs font-bold uppercase tracking-widest ${cfg.color}`}>
                    {cfg.label}
                  </h2>
                  <span className={`text-xs font-semibold px-1.5 py-0.5 rounded-full bg-slate-800 ${cfg.color}`}>
                    {groupTasks.length}
                  </span>
                </div>
                <div className="space-y-2">
                  {groupTasks.map((task) => (
                    <TaskCard
                      key={task.id}
                      task={task}
                      showCase
                      onClick={() => router.push(`/cases/${task.case_id}/tasks/${task.id}`)}
                      onComplete={() => completeMutation.mutate(task.id)}
                      onDelete={() => deleteMutation.mutate(task.id)}
                    />
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default function MyTasksPage() {
  return (
    <Suspense fallback={
      <div className="p-6 max-w-3xl mx-auto">
        <div className="h-7 w-32 bg-slate-700 animate-pulse rounded mb-6" />
        <div className="space-y-2">
          {[...Array(6)].map((_, i) => <div key={i} className="h-16 bg-slate-800 animate-pulse rounded-xl" />)}
        </div>
      </div>
    }>
      <TasksPageContent />
    </Suspense>
  );
}
