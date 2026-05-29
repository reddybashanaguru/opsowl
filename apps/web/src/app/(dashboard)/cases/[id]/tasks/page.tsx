"use client";

import { use, useState } from "react";
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

interface ListMeta {
  total_count: number;
  has_more: boolean;
  next_cursor: string | null;
}

interface CaseSummary {
  id: string;
  case_number: string;
  title: string;
}

const PRIORITY_OPTIONS = ["critical", "high", "medium", "low"] as const;
const STATUS_OPTIONS = ["todo", "in_progress", "done", "cancelled"] as const;

export default function CaseTasksPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: caseId } = use(params);
  const router = useRouter();
  const cfg = useConfig();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);

  const [showCreate, setShowCreate] = useState(false);
  const [filterStatus, setFilterStatus] = useState<string>("");
  const [filterPriority, setFilterPriority] = useState<string>("");
  const [overdueOnly, setOverdueOnly] = useState(false);

  // Create form state
  const [newTitle, setNewTitle] = useState("");
  const [newDesc, setNewDesc] = useState("");
  const [newPriority, setNewPriority] = useState("medium");
  const [newDueDate, setNewDueDate] = useState("");
  const [createError, setCreateError] = useState("");

  const { data: caseData } = useQuery<CaseSummary>({
    queryKey: ["case-summary", caseId],
    queryFn: () => api.get<CaseSummary>(`/api/v1/cases/${caseId}`),
  });

  const buildQueryString = () => {
    const p = new URLSearchParams();
    if (filterStatus) p.set("status", filterStatus);
    if (filterPriority) p.set("priority", filterPriority);
    if (overdueOnly) p.set("overdue", "true");
    p.set("limit", "20");
    return p.toString();
  };

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, refetch } =
    useInfiniteQuery<{ data: Task[]; meta: ListMeta }>({
      queryKey: ["case-tasks", caseId, filterStatus, filterPriority, overdueOnly],
      queryFn: ({ pageParam }) => {
        const qs = buildQueryString();
        const cursor = pageParam ? `&cursor=${pageParam}` : "";
        return api.getList<Task, ListMeta>(`/api/v1/cases/${caseId}/tasks?${qs}${cursor}`);
      },
      initialPageParam: undefined as string | undefined,
      getNextPageParam: (last) => last.meta.next_cursor ?? undefined,
    });

  const tasks = data?.pages.flatMap((p) => p.data) ?? [];
  const totalCount = data?.pages[0]?.meta.total_count ?? 0;

  const createMutation = useMutation({
    mutationFn: () =>
      api.post<Task>(`/api/v1/cases/${caseId}/tasks`, {
        title: newTitle,
        description: newDesc,
        priority: newPriority,
        due_date: newDueDate ? new Date(newDueDate).toISOString() : undefined,
      }),
    onSuccess: () => {
      setShowCreate(false);
      setNewTitle("");
      setNewDesc("");
      setNewPriority("medium");
      setNewDueDate("");
      setCreateError("");
      refetch();
    },
    onError: (err) => {
      setCreateError(err instanceof APIError ? err.message : "Failed to create task");
    },
  });

  const completeMutation = useMutation({
    mutationFn: (taskId: string) => api.post<Task>(`/api/v1/tasks/${taskId}/complete`, {}),
    onSuccess: () => refetch(),
  });

  const deleteMutation = useMutation({
    mutationFn: (taskId: string) => api.delete(`/api/v1/tasks/${taskId}`),
    onSuccess: () => refetch(),
  });

  const canCreate = user?.role && ["analyst", "manager", "admin", "super_admin"].includes(user.role);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-6 h-6 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <CaseSubNav caseId={caseId} activeTab="tasks" />

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

      {/* Filters */}
      <div className="flex items-center gap-3 mb-4 flex-wrap">
        <select
          value={filterStatus}
          onChange={(e) => setFilterStatus(e.target.value)}
          className="bg-[#1e293b] border border-slate-700 text-slate-300 text-xs rounded-lg px-2.5 py-1.5 focus:outline-none focus:border-cyan-500"
        >
          <option value="">All statuses</option>
          {STATUS_OPTIONS.map((s) => (
            <option key={s} value={s}>{s.replace("_", " ")}</option>
          ))}
        </select>

        <select
          value={filterPriority}
          onChange={(e) => setFilterPriority(e.target.value)}
          className="bg-[#1e293b] border border-slate-700 text-slate-300 text-xs rounded-lg px-2.5 py-1.5 focus:outline-none focus:border-cyan-500"
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

        <span className="text-slate-600 text-xs ml-auto">{totalCount} task{totalCount !== 1 ? "s" : ""}</span>
      </div>

      {/* Task list */}
      {tasks.length === 0 ? (
        <div className="text-center py-16">
          <p className="text-slate-600 text-sm">No tasks yet.</p>
          {canCreate && (
            <button
              onClick={() => setShowCreate(true)}
              className="mt-3 text-cyan-400 hover:text-cyan-300 text-sm font-medium"
            >
              Create the first task →
            </button>
          )}
        </div>
      ) : (
        <div className="space-y-3">
          {tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              onClick={() => router.push(`/cases/${caseId}/tasks/${task.id}`)}
              onComplete={() => completeMutation.mutate(task.id)}
              onDelete={() => deleteMutation.mutate(task.id)}
            />
          ))}

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
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-6 w-full max-w-lg">
            <h3 className="text-white font-semibold mb-4">New Task</h3>

            <div className="space-y-3">
              <div>
                <label className="text-slate-400 text-xs mb-1 block">Title *</label>
                <input
                  value={newTitle}
                  onChange={(e) => setNewTitle(e.target.value)}
                  placeholder="Task title…"
                  className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
                  autoFocus
                />
              </div>

              <div>
                <label className="text-slate-400 text-xs mb-1 block">Description</label>
                <textarea
                  value={newDesc}
                  onChange={(e) => setNewDesc(e.target.value)}
                  rows={3}
                  placeholder="Optional description…"
                  className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500 resize-none"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-slate-400 text-xs mb-1 block">Priority</label>
                  <select
                    value={newPriority}
                    onChange={(e) => setNewPriority(e.target.value)}
                    className="w-full bg-[#0f172a] border border-slate-600 text-white rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
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
                    onChange={(e) => setNewDueDate(e.target.value)}
                    className="w-full bg-[#0f172a] border border-slate-600 text-white rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500 [color-scheme:dark]"
                  />
                </div>
              </div>
            </div>

            {createError && (
              <p className="text-red-400 text-xs mt-3">{createError}</p>
            )}

            <div className="flex gap-2 mt-4">
              <button
                onClick={() => createMutation.mutate()}
                disabled={!newTitle.trim() || createMutation.isPending}
                className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-2 rounded-lg text-sm disabled:opacity-60"
              >
                {createMutation.isPending ? "Creating…" : "Create Task"}
              </button>
              <button
                onClick={() => { setShowCreate(false); setCreateError(""); }}
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
