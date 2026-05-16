"use client";

const STATUS_STYLES: Record<string, string> = {
  todo:        "bg-slate-500/20 text-slate-400 border-slate-500/40",
  in_progress: "bg-violet-500/20 text-violet-400 border-violet-500/40",
  done:        "bg-green-500/20 text-green-400 border-green-500/40",
  cancelled:   "bg-slate-600/20 text-slate-500 border-slate-600/40",
};

const STATUS_LABELS: Record<string, string> = {
  todo:        "Todo",
  in_progress: "In Progress",
  done:        "Done",
  cancelled:   "Cancelled",
};

export function TaskStatusBadge({ status }: { status: string }) {
  const style = STATUS_STYLES[status] ?? STATUS_STYLES.todo;
  const label = STATUS_LABELS[status] ?? status;
  return (
    <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium ${style}`}>
      {label}
    </span>
  );
}
