"use client";

import { TaskStatusBadge } from "./TaskStatusBadge";
import { TaskPriorityBadge } from "./TaskPriorityBadge";

export interface Task {
  id: string;
  case_id: string;
  case_number?: string;
  case_title?: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  is_overdue: boolean;
  is_ai_suggested: boolean;
  due_date: string | null;
  completed_at: string | null;
  sort_order: number;
  comment_count: number;
  created_by: { id: string; full_name: string; email: string };
  assigned_to: { id: string; full_name: string; email: string } | null;
  created_at: string;
  updated_at: string;
}

interface Props {
  task: Task;
  onClick?: () => void;
  onComplete?: () => void;
  onDelete?: () => void;
  showCase?: boolean;
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

export function TaskCard({ task, onClick, onComplete, onDelete, showCase }: Props) {
  const isDone = task.status === "done";
  const isCancelled = task.status === "cancelled";

  return (
    <div
      className={`bg-[#1e293b] border rounded-xl p-4 cursor-pointer hover:border-slate-600 transition-colors group ${
        task.is_overdue && !isDone ? "border-red-500/40" : "border-slate-700"
      }`}
      onClick={onClick}
    >
      <div className="flex items-start gap-3">
        {/* Complete checkbox */}
        <button
          className={`flex-shrink-0 mt-0.5 w-5 h-5 rounded border-2 flex items-center justify-center transition-colors ${
            isDone
              ? "bg-green-500 border-green-500 text-white"
              : "border-slate-600 hover:border-cyan-400"
          }`}
          onClick={(e) => {
            e.stopPropagation();
            if (!isDone && !isCancelled) onComplete?.();
          }}
          disabled={isDone || isCancelled}
          title={isDone ? "Completed" : "Mark complete"}
        >
          {isDone && (
            <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 12 12">
              <path d="M10 3L5 8.5 2 5.5" stroke="currentColor" strokeWidth="2" fill="none" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          )}
        </button>

        <div className="flex-1 min-w-0">
          {/* Case context (for My Tasks view) */}
          {showCase && task.case_number && (
            <div className="text-xs text-cyan-400 font-mono mb-1">{task.case_number}</div>
          )}

          {/* Title */}
          <p className={`text-sm font-medium leading-snug ${isDone || isCancelled ? "line-through text-slate-500" : "text-white"}`}>
            {task.title}
          </p>

          {/* Description */}
          {task.description && (
            <p className="text-slate-500 text-xs mt-1 line-clamp-2">{task.description}</p>
          )}

          {/* Badges row */}
          <div className="flex items-center flex-wrap gap-1.5 mt-2">
            <TaskStatusBadge status={task.status} />
            <TaskPriorityBadge priority={task.priority} />

            {task.is_overdue && !isDone && (
              <span className="inline-flex items-center rounded-full border border-red-500/40 bg-red-500/20 px-2 py-0.5 text-xs font-medium text-red-400">
                Overdue
              </span>
            )}
            {task.is_ai_suggested && (
              <span className="inline-flex items-center rounded-full border border-cyan-500/40 bg-cyan-500/10 px-2 py-0.5 text-xs font-medium text-cyan-400">
                AI
              </span>
            )}
          </div>

          {/* Footer */}
          <div className="flex items-center gap-3 mt-2 text-slate-600 text-xs">
            {task.assigned_to ? (
              <span>→ {task.assigned_to.full_name}</span>
            ) : (
              <span className="italic">Unassigned</span>
            )}
            {task.due_date && (
              <span className={task.is_overdue && !isDone ? "text-red-400" : ""}>
                Due {new Date(task.due_date).toLocaleDateString()}
              </span>
            )}
            {task.comment_count > 0 && (
              <span>💬 {task.comment_count}</span>
            )}
            <span className="ml-auto">{relativeTime(task.updated_at)}</span>
          </div>
        </div>

        {/* Delete button — visible on hover */}
        {onDelete && !isDone && !isCancelled && (
          <button
            className="flex-shrink-0 opacity-0 group-hover:opacity-100 text-slate-600 hover:text-red-400 transition-all p-1 rounded"
            onClick={(e) => { e.stopPropagation(); onDelete(); }}
            title="Delete task"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        )}
      </div>
    </div>
  );
}
