"use client";

const STATUS_STYLES: Record<string, string> = {
  draft:          "bg-slate-500/20 text-slate-400 border-slate-500/40",
  open:           "bg-blue-500/20 text-blue-400 border-blue-500/40",
  in_progress:    "bg-violet-500/20 text-violet-400 border-violet-500/40",
  pending_review: "bg-amber-500/20 text-amber-400 border-amber-500/40",
  closed:         "bg-green-500/20 text-green-400 border-green-500/40",
  escalated:      "bg-red-500/20 text-red-400 border-red-500/40",
};

const STATUS_LABELS: Record<string, string> = {
  draft:          "Draft",
  open:           "Open",
  in_progress:    "In Progress",
  pending_review: "Pending Review",
  closed:         "Closed",
  escalated:      "Escalated",
};

interface Props {
  status: string;
  size?: "sm" | "md";
}

export function StatusBadge({ status, size = "sm" }: Props) {
  const style = STATUS_STYLES[status] ?? "bg-slate-500/20 text-slate-400 border-slate-500/40";
  const label = STATUS_LABELS[status] ?? status;
  const px = size === "md" ? "px-3 py-1 text-xs" : "px-2 py-0.5 text-xs";
  return (
    <span className={`inline-flex items-center rounded-full border font-medium ${px} ${style}`}>
      {label}
    </span>
  );
}
