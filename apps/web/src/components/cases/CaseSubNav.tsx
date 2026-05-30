"use client";

import { useRouter } from "next/navigation";
import { useConfig } from "@/lib/hooks/useConfig";
import {
  LayoutGrid,
  StickyNote,
  CheckSquare,
  FileText,
  MessageSquare,
  Sparkles,
  Activity,
  ScrollText,
  ClipboardList,
} from "lucide-react";

type Tab = "case" | "notes" | "tasks" | "documents" | "comms" | "insights" | "timeline" | "ai-logs" | "str";

interface Props {
  caseId: string;
  activeTab: Tab;
  action?: React.ReactNode;
}

export function CaseSubNav({ caseId, activeTab, action }: Props) {
  const router = useRouter();
  const cfg = useConfig();

  const TABS: { tab: Tab; label: string; path: string; Icon: React.ElementType }[] = [
    { tab: "case",      label: "Case",        path: "",          Icon: LayoutGrid },
    { tab: "notes",     label: "Notes",       path: "/notes",    Icon: StickyNote },
    { tab: "tasks",     label: "Tasks",       path: "/tasks",    Icon: CheckSquare },
    { tab: "documents", label: "Documents",   path: "/documents", Icon: FileText },
    { tab: "comms",     label: "Comms",       path: "/comms",    Icon: MessageSquare },
    { tab: "insights",  label: "AI Insights", path: "/insights", Icon: Sparkles },
    { tab: "timeline",  label: "Timeline",    path: "/timeline", Icon: Activity },
    { tab: "ai-logs",   label: "AI Logs",     path: "/ai-logs",  Icon: ScrollText },
    { tab: "str",       label: cfg.str_label, path: "/str",      Icon: ClipboardList },
  ];

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
        {TABS.map(({ tab, label, path, Icon }) => (
          <button
            key={tab}
            onClick={() => router.push(`/cases/${caseId}${path}`)}
            className={`flex items-center gap-1.5 text-xs font-medium px-3 py-2 rounded-t transition-colors border-b-2 -mb-px whitespace-nowrap ${
              activeTab === tab
                ? "text-cyan-400 border-cyan-400 bg-cyan-400/5"
                : "text-slate-500 border-transparent hover:text-slate-300 hover:bg-slate-700/40"
            }`}
          >
            <Icon size={13} />
            {label}
          </button>
        ))}
      </div>
    </div>
  );
}
