"use client";

interface ConfidenceBarProps {
  score: number | null | undefined;
  showLabel?: boolean;
}

function getColor(score: number) {
  if (score >= 0.8) return { bar: "bg-emerald-500", label: "text-emerald-400", text: "High" };
  if (score >= 0.5) return { bar: "bg-amber-500", label: "text-amber-400", text: "Medium" };
  return { bar: "bg-red-500", label: "text-red-400", text: "Low" };
}

export function ConfidenceBar({ score, showLabel = true }: ConfidenceBarProps) {
  if (score == null) return null;

  const pct = Math.round(score * 100);
  const color = getColor(score);

  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 h-1.5 bg-slate-700 rounded-full overflow-hidden" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100}>
        <div className={`h-full rounded-full transition-all ${color.bar}`} style={{ width: `${pct}%` }} />
      </div>
      {showLabel && (
        <span className={`text-xs font-medium ${color.label}`}>
          {pct}% · {color.text}
        </span>
      )}
    </div>
  );
}
