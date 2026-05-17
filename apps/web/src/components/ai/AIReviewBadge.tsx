"use client";

const STATUS_CONFIG: Record<string, { label: string; classes: string }> = {
  pending:  { label: "Pending Review", classes: "bg-amber-500/20 text-amber-300 border-amber-500/40" },
  approved: { label: "Verified",       classes: "bg-emerald-500/20 text-emerald-300 border-emerald-500/40" },
  edited:   { label: "Verified (Edited)", classes: "bg-emerald-500/20 text-emerald-300 border-emerald-500/40" },
  rejected: { label: "Rejected",       classes: "bg-red-500/20 text-red-300 border-red-500/40" },
};

interface AIReviewBadgeProps {
  status: string;
}

export function AIReviewBadge({ status }: AIReviewBadgeProps) {
  const cfg = STATUS_CONFIG[status] ?? STATUS_CONFIG.pending;
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold border ${cfg.classes}`} data-testid="review-badge">
      {cfg.label}
    </span>
  );
}
