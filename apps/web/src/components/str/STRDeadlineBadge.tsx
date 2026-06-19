"use client";

import { AlertTriangle, Clock } from "lucide-react";

interface Props {
  suspicionDate?: string | null;
  deadlineDays?: number;
  isOverdue?: boolean;
  daysRemaining?: number;
  compact?: boolean;
}

export function STRDeadlineBadge({
  suspicionDate,
  deadlineDays = 30,
  isOverdue,
  daysRemaining,
  compact = false,
}: Props) {
  // Compute locally if daysRemaining/isOverdue not provided
  let days = daysRemaining;
  let overdue = isOverdue;

  if (suspicionDate && days === undefined) {
    const d = new Date(suspicionDate);
    d.setDate(d.getDate() + deadlineDays);
    days = Math.ceil((d.getTime() - Date.now()) / 86400000);
    overdue = days < 0;
  }

  if (days === undefined) return null;

  if (overdue) {
    return (
      <span
        className={`inline-flex items-center gap-1 font-medium rounded-full animate-pulse
          ${compact
            ? "text-[10px] px-1.5 py-0.5 bg-red-900/60 text-red-300 border border-red-500/50"
            : "text-xs px-2.5 py-1 bg-red-900/40 text-red-300 border border-red-500/40"
          }`}
        title={`STR overdue by ${Math.abs(days!)} days`}
      >
        <AlertTriangle size={compact ? 10 : 12} className="shrink-0" />
        {compact ? `${Math.abs(days!)}d OD` : `Overdue by ${Math.abs(days!)} day${Math.abs(days!) !== 1 ? "s" : ""}`}
      </span>
    );
  }

  if (days <= 3) {
    return (
      <span
        className={`inline-flex items-center gap-1 font-medium rounded-full animate-pulse
          ${compact
            ? "text-[10px] px-1.5 py-0.5 bg-amber-900/60 text-amber-300 border border-amber-500/50"
            : "text-xs px-2.5 py-1 bg-amber-900/40 text-amber-300 border border-amber-500/40"
          }`}
        title={`STR due in ${days} day${days !== 1 ? "s" : ""}`}
      >
        <Clock size={compact ? 10 : 12} className="shrink-0" />
        {compact ? `${days}d left` : `Due in ${days} day${days !== 1 ? "s" : ""}`}
      </span>
    );
  }

  if (days <= 7) {
    return (
      <span
        className={`inline-flex items-center gap-1 font-medium rounded-full
          ${compact
            ? "text-[10px] px-1.5 py-0.5 bg-amber-900/40 text-amber-400 border border-amber-500/30"
            : "text-xs px-2.5 py-1 bg-amber-900/20 text-amber-400 border border-amber-500/20"
          }`}
        title={`STR due in ${days} days`}
      >
        <Clock size={compact ? 10 : 12} className="shrink-0" />
        {compact ? `${days}d` : `${days} days to file`}
      </span>
    );
  }

  return (
    <span
      className={`inline-flex items-center gap-1 text-slate-500 rounded-full
        ${compact
          ? "text-[10px] px-1.5 py-0.5"
          : "text-xs px-2.5 py-1"
        }`}
      title={`STR due in ${days} days`}
    >
      <Clock size={compact ? 10 : 12} className="shrink-0" />
      {compact ? `${days}d` : `${days} days to file`}
    </span>
  );
}
