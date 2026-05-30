"use client";

import { useState, useRef, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useConfig } from "@/lib/hooks/useConfig";
import {
  LayoutGrid,
  StickyNote,
  CheckSquare,
  FileText,
  MessageSquare,
  ChevronDown,
} from "lucide-react";

type Tab = "case" | "notes" | "tasks" | "documents" | "comms" | "insights" | "timeline" | "ai-logs" | "str";

interface Props {
  caseId: string;
  activeTab: Tab;
  /** Optional action button shown right-aligned in the top row */
  action?: React.ReactNode;
}

const PRIMARY_TABS: { tab: Tab; label: string; path: string; Icon: React.ElementType }[] = [
  { tab: "case",      label: "Case",      path: "",          Icon: LayoutGrid },
  { tab: "notes",     label: "Notes",     path: "/notes",    Icon: StickyNote },
  { tab: "tasks",     label: "Tasks",     path: "/tasks",    Icon: CheckSquare },
  { tab: "documents", label: "Documents", path: "/documents", Icon: FileText },
  { tab: "comms",     label: "Comms",     path: "/comms",    Icon: MessageSquare },
];

const MORE_TABS_BASE: { tab: Tab; label: string; path: string }[] = [
  { tab: "insights",  label: "AI Insights", path: "/insights" },
  { tab: "timeline",  label: "Timeline",    path: "/timeline" },
  { tab: "ai-logs",   label: "AI Logs",     path: "/ai-logs" },
  { tab: "str",       label: "",            path: "/str" },
];

export function CaseSubNav({ caseId, activeTab, action }: Props) {
  const router = useRouter();
  const cfg = useConfig();
  const [moreOpen, setMoreOpen] = useState(false);
  const moreRef = useRef<HTMLDivElement>(null);

  const MORE_TABS = MORE_TABS_BASE.map((t) =>
    t.tab === "str" ? { ...t, label: cfg.str_label } : t
  );

  const isMore = MORE_TABS.some((t) => t.tab === activeTab);
  const activeMoreLabel = MORE_TABS.find((t) => t.tab === activeTab)?.label;

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
          className="text-slate-400 hover:text-slate-200 text-sm flex items-center gap-1.5 transition-colors"
        >
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
          </svg>
          Cases
        </button>
        {action && <div className="flex items-center gap-2">{action}</div>}
      </div>

      {/* Row 2: tab strip */}
      <div className="flex items-center gap-0.5 border-b border-slate-700/50 pb-0">
        {PRIMARY_TABS.map(({ tab, label, path, Icon }) => (
          <button
            key={tab}
            onClick={() => router.push(`/cases/${caseId}${path}`)}
            className={`flex items-center gap-1.5 text-xs font-medium px-3 py-2 rounded-t transition-colors border-b-2 -mb-px ${
              activeTab === tab
                ? "text-cyan-400 border-cyan-400 bg-cyan-400/5"
                : "text-slate-500 border-transparent hover:text-slate-300 hover:bg-slate-700/40"
            }`}
          >
            <Icon size={13} />
            {label}
          </button>
        ))}

        {/* More dropdown */}
        <div className="relative" ref={moreRef}>
          <button
            onClick={() => setMoreOpen((v) => !v)}
            className={`flex items-center gap-1 text-xs font-medium px-3 py-2 rounded-t transition-colors border-b-2 -mb-px ${
              isMore
                ? "text-cyan-400 border-cyan-400 bg-cyan-400/5"
                : "text-slate-500 border-transparent hover:text-slate-300 hover:bg-slate-700/40"
            }`}
          >
            {isMore ? activeMoreLabel : "More"}
            <ChevronDown size={11} className={`transition-transform ${moreOpen ? "rotate-180" : ""}`} />
          </button>
          {moreOpen && (
            <div className="absolute top-full left-0 mt-1 bg-[#1e293b] border border-slate-700 rounded-lg shadow-xl z-20 min-w-[144px] py-1">
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
