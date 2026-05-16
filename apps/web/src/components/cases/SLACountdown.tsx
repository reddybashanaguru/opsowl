"use client";

import { useEffect, useState } from "react";

interface Props {
  breachAt: string | null;
  breached: boolean;
}

function formatDuration(ms: number): string {
  if (ms <= 0) return "Breached";
  const h = Math.floor(ms / 3600000);
  const m = Math.floor((ms % 3600000) / 60000);
  if (h >= 24) return `${Math.floor(h / 24)}d ${h % 24}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

export function SLACountdown({ breachAt, breached }: Props) {
  const [remaining, setRemaining] = useState<number | null>(null);

  useEffect(() => {
    if (!breachAt) return;
    const target = new Date(breachAt).getTime();
    const tick = () => setRemaining(target - Date.now());
    tick();
    const id = setInterval(tick, 60000);
    return () => clearInterval(id);
  }, [breachAt]);

  if (!breachAt) return <span className="text-slate-500 text-xs">No SLA</span>;

  if (breached || (remaining !== null && remaining <= 0)) {
    return (
      <span className="inline-flex items-center gap-1 text-xs font-medium text-red-400">
        <span className="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse" />
        SLA Breached
      </span>
    );
  }

  if (remaining === null) return null;

  const isWarning = remaining < 3600000 * 2;
  return (
    <span className={`text-xs font-medium ${isWarning ? "text-amber-400" : "text-slate-400"}`}>
      {formatDuration(remaining)}
    </span>
  );
}
