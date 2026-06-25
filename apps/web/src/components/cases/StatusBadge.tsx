"use client";

const STATUS_STYLES: Record<string, { badge: string; dot: string; pulse: boolean }> = {
  draft:          { badge: "bg-slate-500/20 text-slate-400 border-slate-500/40",   dot: "bg-slate-400",   pulse: false },
  open:           { badge: "bg-blue-500/20 text-blue-300 border-blue-500/50",      dot: "bg-blue-400",    pulse: true  },
  in_progress:    { badge: "bg-violet-500/20 text-violet-300 border-violet-500/50", dot: "bg-violet-400", pulse: true  },
  pending_review: { badge: "bg-amber-500/20 text-amber-300 border-amber-500/50",   dot: "bg-amber-400",   pulse: true  },
  closed:         { badge: "bg-green-500/20 text-green-300 border-green-500/40",   dot: "bg-green-400",   pulse: false },
  escalated:      { badge: "bg-red-500/20 text-red-300 border-red-500/50",         dot: "bg-red-400",     pulse: true  },
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
  size?: "sm" | "md" | "lg";
}

export function StatusBadge({ status, size = "sm" }: Props) {
  const config = STATUS_STYLES[status] ?? { badge: "bg-slate-500/20 text-slate-400 border-slate-500/40", dot: "bg-slate-400", pulse: false };
  const label = STATUS_LABELS[status] ?? status;

  if (size === "lg") {
    return (
      <span className={`inline-flex items-center gap-2 rounded-lg border px-3 py-1.5 text-sm font-semibold tracking-wider ${config.badge}`}>
        <span className="relative flex items-center justify-center w-2.5 h-2.5">
          {config.pulse && (
            <span className={`absolute inline-flex w-full h-full rounded-full opacity-60 animate-ping ${config.dot}`} />
          )}
          <span className={`relative w-2 h-2 rounded-full ${config.dot}`} />
        </span>
        {label}
      </span>
    );
  }

  const px = size === "md" ? "px-3 py-1 text-xs" : "px-2 py-0.5 text-xs";
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full border font-medium ${px} ${config.badge}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${config.dot}`} />
      {label}
    </span>
  );
}
