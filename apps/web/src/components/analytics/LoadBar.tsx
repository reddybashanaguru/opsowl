interface LoadBarProps {
  value: number;
  max: number;
  color: string;
}

export function LoadBar({ value, max, color }: LoadBarProps) {
  const pct = max > 0 ? Math.min((value / max) * 100, 100) : 0;
  return (
    <div className="h-1.5 bg-slate-700 rounded-full overflow-hidden">
      <div
        data-testid="loadbar-fill"
        className={`h-full rounded-full transition-all ${color}`}
        style={{ width: `${pct}%` }}
      />
    </div>
  );
}
