"use client";

import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { TaskCard, Task } from "@/components/tasks/TaskCard";
import { useMutation, useQueryClient } from "@tanstack/react-query";

interface ListMeta {
  total_count: number;
  has_more: boolean;
  next_cursor: string | null;
}

type Group = "overdue" | "due_today" | "this_week" | "later" | "done";

function groupLabel(group: Group): string {
  return {
    overdue:   "Overdue",
    due_today: "Due Today",
    this_week: "Due This Week",
    later:     "Later",
    done:      "Done",
  }[group];
}

function groupAccentClass(group: Group): string {
  return {
    overdue:   "text-red-400",
    due_today: "text-amber-400",
    this_week: "text-cyan-400",
    later:     "text-slate-400",
    done:      "text-green-400",
  }[group];
}

function classifyTask(task: Task): Group {
  if (task.status === "done") return "done";
  if (task.status === "cancelled") return "done";
  if (task.is_overdue) return "overdue";
  if (!task.due_date) return "later";

  const due = new Date(task.due_date);
  const now = new Date();
  const endOfToday = new Date(now);
  endOfToday.setHours(23, 59, 59, 999);
  const endOfWeek = new Date(now);
  endOfWeek.setDate(endOfWeek.getDate() + 7);

  if (due <= endOfToday) return "due_today";
  if (due <= endOfWeek) return "this_week";
  return "later";
}

const GROUP_ORDER: Group[] = ["overdue", "due_today", "this_week", "later", "done"];

export default function MyTasksPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);

  const { data, isLoading, refetch } = useQuery<{ data: Task[]; meta: ListMeta }>({
    queryKey: ["my-tasks"],
    queryFn: () => api.getList<Task, ListMeta>("/api/v1/tasks?limit=100"),
    enabled: !!user,
  });

  const completeMutation = useMutation({
    mutationFn: (taskId: string) => api.post<Task>(`/api/v1/tasks/${taskId}/complete`, {}),
    onSuccess: () => refetch(),
  });

  const deleteMutation = useMutation({
    mutationFn: (taskId: string) => api.delete(`/api/v1/tasks/${taskId}`),
    onSuccess: () => refetch(),
  });

  const tasks = data?.data ?? [];

  // Group tasks by urgency
  const groups = GROUP_ORDER.reduce(
    (acc, g) => ({ ...acc, [g]: [] as Task[] }),
    {} as Record<Group, Task[]>,
  );
  tasks.forEach((t) => groups[classifyTask(t)].push(t));

  const activeTasks = tasks.filter((t) => t.status !== "done" && t.status !== "cancelled");

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-6 h-6 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <div className="p-6 max-w-3xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-white text-xl font-bold">My Tasks</h1>
        <p className="text-slate-500 text-sm mt-1">
          {activeTasks.length} active task{activeTasks.length !== 1 ? "s" : ""} across all cases
        </p>
      </div>

      {tasks.length === 0 ? (
        <div className="text-center py-16">
          <div className="text-4xl mb-4">✓</div>
          <p className="text-slate-400 text-sm font-medium">All caught up!</p>
          <p className="text-slate-600 text-xs mt-1">No tasks assigned to you.</p>
        </div>
      ) : (
        <div className="space-y-8">
          {GROUP_ORDER.map((group) => {
            const groupTasks = groups[group];
            if (groupTasks.length === 0) return null;
            return (
              <div key={group}>
                <div className="flex items-center gap-2 mb-3">
                  <h2 className={`text-xs font-bold uppercase tracking-widest ${groupAccentClass(group)}`}>
                    {groupLabel(group)}
                  </h2>
                  <span className="text-slate-700 text-xs">{groupTasks.length}</span>
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
