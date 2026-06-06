"use client";

import { useRouter } from "next/navigation";
import { useConfig } from "@/lib/hooks/useConfig";
import { useAuth } from "@/lib/hooks/useAuth";
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
  Shield,
  Lock,
  ArrowLeftRight,
  PenLine,
} from "lucide-react";

type Tab = "case" | "notes" | "tasks" | "documents" | "comms" | "insights" | "timeline" | "ai-logs" | "str" | "findings" | "closure" | "transactions" | "narrative";

interface Props {
  caseId: string;
  activeTab: Tab;
  action?: React.ReactNode;
  caseNumber?: string;
}

// Subpages where back goes to case detail (not the cases list)
const BACK_TO_CASE = new Set<Tab>(["notes", "timeline", "comms", "insights", "ai-logs", "str"]);

export function CaseSubNav({ caseId, activeTab, action, caseNumber }: Props) {
  const router = useRouter();
  const cfg = useConfig();
  const { user } = useAuth();
  const ROLE_HIERARCHY: Record<string, number> = { super_admin: 5, admin: 4, manager: 3, analyst: 2, viewer: 1 };
  const userLevel = ROLE_HIERARCHY[user?.role ?? "viewer"] ?? 0;

  const backLabel = BACK_TO_CASE.has(activeTab) ? "← Case" : "← Cases";
  const backHref  = BACK_TO_CASE.has(activeTab) ? `/cases/${caseId}` : "/cases";

  // Breadcrumb button shown between back link and tabs on subpages
  const breadcrumbLabel =
    activeTab === "tasks"     ? (caseNumber ?? null) :
    activeTab === "documents" ? cfg.case_label :
    null;

  const ALL_TABS: { tab: Tab; label: string; path: string; Icon: React.ElementType; minLevel?: number }[] = [
    { tab: "case",      label: "Case",              path: "",          Icon: LayoutGrid },
    { tab: "notes",     label: "Notes",           path: "/notes",    Icon: StickyNote },
    { tab: "tasks",     label: "Tasks",           path: "/tasks",    Icon: CheckSquare },
    { tab: "documents", label: "Documents",       path: "/documents", Icon: FileText },
    { tab: "comms",     label: "Comms",           path: "/comms",    Icon: MessageSquare },
    { tab: "insights",  label: "AI Insights",     path: "/insights", Icon: Sparkles },
    { tab: "timeline",  label: "Timeline",        path: "/timeline", Icon: Activity },
    { tab: "ai-logs",   label: "AI Logs",         path: "/ai-logs",  Icon: ScrollText },
    { tab: "str",       label: cfg.str_label,     path: "/str",      Icon: ClipboardList, minLevel: 3 },
    { tab: "findings",      label: "Findings",      path: "/findings",      Icon: Shield },
    { tab: "transactions",  label: "Transactions",  path: "/transactions",  Icon: ArrowLeftRight },
    { tab: "narrative",     label: "Narrative",     path: "/narrative",     Icon: PenLine },
    { tab: "closure",       label: "Closure",       path: "/closure",       Icon: Lock, minLevel: 2 },
  ];

  const TABS = ALL_TABS.filter((t) => userLevel >= (t.minLevel ?? 0));

  return (
    <div className="mb-4">
      {/* Row 1: back breadcrumb + action */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <button
            onClick={() => router.push(backHref)}
            className="text-slate-400 hover:text-slate-200 text-sm transition-colors"
          >
            {backLabel}
          </button>
          {breadcrumbLabel && (
            <>
              <span className="text-slate-600 text-xs">/</span>
              <button
                onClick={() => router.push(`/cases/${caseId}`)}
                className="text-slate-400 hover:text-slate-200 text-sm transition-colors"
              >
                {breadcrumbLabel}
              </button>
            </>
          )}
        </div>
        {action && <div className="flex items-center gap-2">{action}</div>}
      </div>

      {/* Row 2: tab strip */}
      <div className="flex items-center gap-0.5 border-b border-slate-700/50 pb-0 overflow-x-auto scrollbar-none">
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
