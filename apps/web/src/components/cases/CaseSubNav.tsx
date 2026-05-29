"use client";

import { useState, useRef, useEffect } from "react";
import { useRouter } from "next/navigation";

type Tab = "case" | "notes" | "tasks" | "documents" | "comms" | "insights" | "timeline" | "ai-logs" | "str";

interface Props {
  caseId: string;
  activeTab: Tab;
  /** Optional action button shown right-aligned in the top row */
  action?: React.ReactNode;
}

const PRIMARY_TABS: { tab: Tab; label: string; path: string }[] = [
  { tab: "case",      label: "Case",      path: "" },
  { tab: "notes",     label: "Notes",     path: "/notes" },
  { tab: "tasks",     label: "Tasks",     path: "/tasks" },
  { tab: "documents", label: "Documents", path: "/documents" },
  { tab: "comms",     label: "Comms",     path: "/comms" },
];

const MORE_TABS: { tab: Tab; label: string; path: string }[] = [
  { tab: "insights",  label: "AI Insights", path: "/insights" },
  { tab: "timeline",  label: "Timeline",    path: "/timeline" },
  { tab: "ai-logs",   label: "AI Logs",     path: "/ai-logs" },
  { tab: "str",       label: "STR",         path: "/str" },
];

export function CaseSubNav({ caseId, activeTab, action }: Props) {
  const router = useRouter();
  const [moreOpen, setMoreOpen] = useState(false);
  const moreRef = useRef<HTMLDivElement>(null);

  const isMore = MORE_TABS.some((t) => t.tab === activeTab);

  useEffect(() => {
    if (!moreOpen) return;
    const handler = (e: MouseEvent) => {
      if (moreRef.current && !moreRef.current.contains(e.target as Node)) {
        setMoreOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [moreOpen]);

  return (
    <div className="mb-4">
      {/* Row 1: back + action */}
      <div className="flex items-center justify-between mb-3">
        <button
          onClick={() => router.push("/cases")}
          className="text-slate-400 hover:text-slate-200 text-sm flex items-center gap-1 transition-colors"
        >
          ← Cases
        </button>
        {action && <div className="flex items-center gap-2">{action}</div>}
      </div>

      {/* Row 2: tab strip */}
      <div className="flex items-center gap-1 border-b border-slate-700/50 pb-2">
        {PRIMARY_TABS.map(({ tab, label, path }) => (
          <button
            key={tab}
            onClick={() => router.push(`/cases/${caseId}${path}`)}
            className={`text-xs font-medium px-3 py-1 rounded transition-colors ${
              activeTab === tab
                ? "text-cyan-400 bg-cyan-400/10"
                : "text-slate-500 hover:text-slate-300 hover:bg-slate-700/40"
            }`}
          >
            {label}
          </button>
        ))}

        {/* More dropdown */}
        <div className="relative" ref={moreRef}>
          <button
            onClick={() => setMoreOpen((v) => !v)}
            className={`text-xs font-medium px-3 py-1 rounded transition-colors flex items-center gap-1 ${
              isMore
                ? "text-cyan-400 bg-cyan-400/10"
                : "text-slate-500 hover:text-slate-300 hover:bg-slate-700/40"
            }`}
          >
            {isMore ? MORE_TABS.find((t) => t.tab === activeTab)?.label : "More"} ↓
          </button>
          {moreOpen && (
            <div className="absolute top-8 left-0 bg-[#1e293b] border border-slate-700 rounded-lg shadow-xl z-20 min-w-[140px] py-1">
              {MORE_TABS.map(({ tab, label, path }) => (
                <button
                  key={tab}
                  onClick={() => { router.push(`/cases/${caseId}${path}`); setMoreOpen(false); }}
                  className={`w-full text-left px-4 py-2 text-sm transition-colors ${
                    activeTab === tab
                      ? "text-cyan-400 bg-cyan-400/10"
                      : "text-slate-300 hover:text-white hover:bg-slate-700/50"
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
