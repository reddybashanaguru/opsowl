"use client";

import { useState } from "react";
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
  is_blocked: boolean;
  playbook_step_order: number;
  depends_on_step: number;
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
  onAssignToMe?: () => void;
  onUnassign?: () => void;
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

export function TaskCard({ task, onClick, onComplete, onDelete, onAssignToMe, onUnassign, showCase }: Props) {
  const isDone = task.status === "done";
  const isCancelled = task.status === "cancelled";
  const [confirmDelete, setConfirmDelete] = useState(false);

  return (
    <div
      className={`bg-[#131f30] border rounded-xl p-4 cursor-pointer hover:border-slate-600 transition-colors group ${
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
            {task.is_blocked && !isDone && (
              <span className="inline-flex items-center gap-1 rounded-full border border-orange-500/40 bg-orange-500/10 px-2 py-0.5 text-xs font-medium text-orange-400">
                <svg className="w-3 h-3" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="2">
                  <path d="M8 3v5m0 2.5v.5" strokeLinecap="round" />
                  <circle cx="8" cy="8" r="7" />
                </svg>
                Blocked
              </span>
            )}
            {task.playbook_step_order > 0 && (
              <span className="inline-flex items-center rounded-full border border-violet-500/30 bg-violet-500/10 px-2 py-0.5 text-xs font-medium text-violet-400">
                Step {task.playbook_step_order}
              </span>
            )}
            {task.is_ai_suggested && (
              <span className="inline-flex items-center rounded-full border border-cyan-500/40 bg-cyan-500/10 px-2 py-0.5 text-xs font-medium text-cyan-400">
                AI
              </span>
            )}
          </div>

          {/* Footer */}
          <div className="flex items-center gap-3 mt-2 text-xs">
            {task.assigned_to ? (
              <span className="flex items-center gap-1 text-slate-400">
                <svg className="w-3 h-3 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
                </svg>
                {task.assigned_to.full_name}
                {onUnassign && !isDone && !isCancelled && (
                  <button
                    onClick={(e) => { e.stopPropagation(); onUnassign(); }}
                    className="opacity-0 group-hover:opacity-100 ml-0.5 text-slate-600 hover:text-red-400 transition-all leading-none"
                    title="Unassign"
                  >
                    ×
                  </button>
                )}
              </span>
            ) : (
              <span className="flex items-center gap-1 text-slate-600 italic">
                <svg className="w-3 h-3 flex-shrink-0" fill="none" viewBox="0 0 20 20" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" />
                </svg>
                {onAssignToMe ? (
                  <button
                    onClick={(e) => { e.stopPropagation(); onAssignToMe(); }}
                    className="text-cyan-500 hover:text-cyan-400 not-italic font-medium transition-colors"
                  >
                    Assign to me
                  </button>
                ) : (
                  "Unassigned"
                )}
              </span>
            )}
            {task.due_date && (
              <span className={task.is_overdue && !isDone ? "text-red-400" : "text-slate-600"}>
                Due {new Date(task.due_date).toLocaleDateString()}
              </span>
            )}
            {task.comment_count > 0 && (
              <span className="text-slate-600">💬 {task.comment_count}</span>
            )}
            <span className="ml-auto text-slate-600">{relativeTime(task.updated_at)}</span>
          </div>
        </div>

        {/* Delete button — visible on hover */}
        {onDelete && !isDone && !isCancelled && (
          confirmDelete ? (
            <div className="flex flex-col gap-1 flex-shrink-0" onClick={(e) => e.stopPropagation()}>
              <button
                onClick={() => { onDelete(); setConfirmDelete(false); }}
                className="text-[10px] text-white bg-red-600 hover:bg-red-500 px-2 py-0.5 rounded transition-colors whitespace-nowrap"
              >
                Delete
              </button>
              <button
                onClick={() => setConfirmDelete(false)}
                className="text-[10px] text-slate-400 hover:text-slate-200 px-2 py-0.5 rounded border border-slate-700 transition-colors"
              >
                Cancel
              </button>
            </div>
          ) : (
            <button
              className="flex-shrink-0 opacity-0 group-hover:opacity-100 text-slate-600 hover:text-red-400 transition-all p-1 rounded"
              onClick={(e) => { e.stopPropagation(); setConfirmDelete(true); }}
              title="Delete task"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          )
        )}
      </div>
    </div>
  );
}
