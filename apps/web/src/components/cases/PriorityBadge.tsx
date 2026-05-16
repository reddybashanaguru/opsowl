"use client";

const PRIORITY_STYLES: Record<string, string> = {
  critical: "bg-red-500/15 text-red-400 border-red-500/40",
  high:     "bg-amber-500/15 text-amber-400 border-amber-500/40",
  medium:   "bg-blue-500/15 text-blue-400 border-blue-500/40",
  low:      "bg-green-500/15 text-green-400 border-green-500/40",
};

const PRIORITY_DOTS: Record<string, string> = {
  critical: "bg-red-400",
  high:     "bg-amber-400",
  medium:   "bg-blue-400",
  low:      "bg-green-400",
};

interface Props {
  priority: string;
}

export function PriorityBadge({ priority }: Props) {
  const style = PRIORITY_STYLES[priority] ?? "bg-slate-500/20 text-slate-400 border-slate-500/40";
  const dot = PRIORITY_DOTS[priority] ?? "bg-slate-400";
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium ${style}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${dot}`} />
      {priority.charAt(0).toUpperCase() + priority.slice(1)}
    </span>
  );
}
