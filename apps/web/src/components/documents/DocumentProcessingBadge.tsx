"use client";

const STATUS_CONFIG: Record<string, { label: string; className: string }> = {
  pending:    { label: "Pending",    className: "bg-slate-500/20 text-slate-400 border-slate-500/40" },
  processing: { label: "Processing", className: "bg-violet-500/20 text-violet-400 border-violet-500/40 animate-pulse" },
  completed:  { label: "Processed",  className: "bg-emerald-500/20 text-emerald-400 border-emerald-500/40" },
  failed:     { label: "Failed",     className: "bg-red-500/20 text-red-400 border-red-500/40" },
};

interface Props {
  status: string;
}

export function DocumentProcessingBadge({ status }: Props) {
  const cfg = STATUS_CONFIG[status] ?? STATUS_CONFIG.pending;
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium border ${cfg.className}`}>
      {status === "processing" && (
        <span className="w-1.5 h-1.5 rounded-full bg-violet-400 animate-ping" />
      )}
      {cfg.label}
    </span>
  );
}
