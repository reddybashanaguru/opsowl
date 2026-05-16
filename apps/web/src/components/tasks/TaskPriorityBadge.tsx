"use client";

const PRIORITY_STYLES: Record<string, string> = {
  critical: "bg-red-500/20 text-red-400 border-red-500/40",
  high:     "bg-orange-500/20 text-orange-400 border-orange-500/40",
  medium:   "bg-yellow-500/20 text-yellow-400 border-yellow-500/40",
  low:      "bg-slate-500/20 text-slate-400 border-slate-500/40",
};

const PRIORITY_ICONS: Record<string, string> = {
  critical: "▲▲",
  high:     "▲",
  medium:   "●",
  low:      "▽",
};

export function TaskPriorityBadge({ priority }: { priority: string }) {
  const style = PRIORITY_STYLES[priority] ?? PRIORITY_STYLES.medium;
  const icon = PRIORITY_ICONS[priority] ?? "●";
  return (
    <span className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium ${style}`}>
      <span className="text-[10px]">{icon}</span>
      {priority.charAt(0).toUpperCase() + priority.slice(1)}
    </span>
  );
}
