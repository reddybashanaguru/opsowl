"use client";

import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { useConfig } from "@/lib/hooks/useConfig";
import { useAuth } from "@/lib/hooks/useAuth";
import { api } from "@/lib/api";
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
  Users,
} from "lucide-react";

type Tab = "case" | "notes" | "tasks" | "documents" | "comms" | "insights" | "timeline" | "ai-logs" | "str" | "findings" | "closure" | "transactions" | "narrative" | "ubo";

// Only these two have hard API blocks — closure draft creation fails without them
const HARD_BLOCK_TABS = new Set<Tab>(["findings", "tasks"]);

type DotState = "done" | "pending" | "none";

interface Completion {
  documents: number;
  documents_processing: number;
  tasks: number;
  tasks_open: number;
  notes: number;
  comms: number;
  insights_available: boolean;
  transactions: number;
  findings: number;
  findings_approved: number;
  findings_pending: number;
  str_exists: boolean;
  str_filed: boolean;
  narrative_exists: boolean;
  timeline_events: number;
}

interface Props {
  caseId: string;
  activeTab: Tab;
  action?: React.ReactNode;
  caseNumber?: string;
}

function StatusDot({ state }: { state: DotState }) {
  if (state === "none") return null;
  return (
    <span
      className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${
        state === "done" ? "bg-emerald-400" : "bg-amber-400"
      }`}
      title={state === "pending" ? "Action needed" : "Complete"}
    />
  );
}

function tabDotState(tab: Tab, c: Completion | undefined): DotState {
  if (!c) return "none";
  switch (tab) {
    case "documents":
      if (c.documents === 0) return "none";
      return c.documents_processing > 0 ? "pending" : "done";
    case "tasks":
      if (c.tasks === 0) return "none";
      return c.tasks_open > 0 ? "pending" : "done";
    case "comms":
      return c.comms > 0 ? "done" : "none";
    case "insights":
      return c.insights_available ? "done" : "none";
    case "transactions":
      return c.transactions > 0 ? "done" : "none";
    case "notes":
      return c.notes > 0 ? "done" : "none";
    case "findings":
      if (c.findings === 0) return "none";
      if (c.findings_approved > 0 && c.findings_pending === 0) return "done";
      return "pending"; // has findings but some need review/approval
    case "str":
      if (!c.str_exists) return "none";
      return c.str_filed ? "done" : "pending";
    case "narrative":
      return c.narrative_exists ? "done" : "none";
    case "timeline":
      return c.timeline_events > 0 ? "done" : "none";
    default:
      return "none";
  }
}

export function CaseSubNav({ caseId, activeTab, action, caseNumber }: Props) {
  const router = useRouter();
  const cfg = useConfig();
  const { user } = useAuth();
  const ROLE_HIERARCHY: Record<string, number> = { super_admin: 5, admin: 4, manager: 3, analyst: 2, viewer: 1 };
  const userLevel = ROLE_HIERARCHY[user?.role ?? "viewer"] ?? 0;

  const { data: completion } = useQuery<Completion>({
    queryKey: ["case-completion", caseId],
    queryFn: () => api.get(`/api/v1/cases/${caseId}/completion`),
    staleTime: 30_000,
  });

  const backLabel = "← Cases";
  const backHref  = "/cases";
  const breadcrumbLabel = activeTab === "case" ? null : (caseNumber ?? cfg.case_label);

  // Analyst workflow: Setup → Investigate → Document → Report → Audit → Close
  const ALL_TABS: { tab: Tab; label: string; path: string; Icon: React.ElementType; minLevel?: number }[] = [
    // Phase 1 — Setup
    { tab: "case",         label: "Case",          path: "",              Icon: LayoutGrid },
    { tab: "documents",    label: "Documents",     path: "/documents",    Icon: FileText },
    // Phase 2 — Investigation (follow the money)
    { tab: "transactions", label: "Transactions",  path: "/transactions", Icon: ArrowLeftRight },
    { tab: "insights",     label: "AI Insights",   path: "/insights",     Icon: Sparkles },
    { tab: "tasks",        label: "Tasks",         path: "/tasks",        Icon: CheckSquare },
    { tab: "comms",        label: "Comms",         path: "/comms",        Icon: MessageSquare },
    // Phase 3 — Documentation
    { tab: "notes",        label: "Notes",         path: "/notes",        Icon: StickyNote },
    { tab: "findings",     label: "Findings",      path: "/findings",     Icon: Shield },
    ...(cfg.ubo_enabled ? [{ tab: "ubo" as Tab, label: "UBO", path: "/ubo", Icon: Users, minLevel: 3 }] : []),
    // Phase 4 — Reporting
    { tab: "narrative",    label: "Narrative",     path: "/narrative",    Icon: PenLine },
    { tab: "str",          label: cfg.str_label,   path: "/str",          Icon: ClipboardList, minLevel: 3 },
    // Phase 5 — Audit & Closure
    { tab: "ai-logs",      label: "AI Logs",       path: "/ai-logs",      Icon: ScrollText },
    { tab: "timeline",     label: "Timeline",      path: "/timeline",     Icon: Activity },
    { tab: "closure",      label: "Closure",       path: "/closure",      Icon: Lock,          minLevel: 2 },
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
        {TABS.map(({ tab, label, path, Icon }) => {
          const isActive = activeTab === tab;
          const dot = tabDotState(tab, completion);
          const required = HARD_BLOCK_TABS.has(tab);
          return (
            <button
              key={tab}
              onClick={() => router.push(`/cases/${caseId}${path}`)}
              title={required ? "Required for STR filing or case closure" : undefined}
              className={`flex items-center gap-1.5 text-xs font-medium px-3 py-2 rounded-t transition-colors border-b-2 -mb-px whitespace-nowrap ${
                isActive
                  ? "text-cyan-400 border-cyan-400 bg-cyan-400/5"
                  : "text-slate-500 border-transparent hover:text-slate-300 hover:bg-slate-700/40"
              }`}
            >
              <Icon size={13} />
              {label}
              {required && (
                <span className="text-[9px] font-bold leading-none text-red-400" title="Hard stop — API blocks closure until resolved">*</span>
              )}
              {!isActive && <StatusDot state={dot} />}
            </button>
          );
        })}
      </div>
      {/* Hard-stop warning — only when there is an actual blocker */}
      {completion && (completion.tasks_open > 0 || completion.tasks === 0 || completion.findings_pending > 0 || completion.findings === 0) && (
        <div className="flex items-center gap-2 mt-1.5 px-3 py-2 rounded-lg bg-amber-500/[0.07] border border-amber-500/25">
          <svg className="w-3.5 h-3.5 text-amber-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v4m0 3h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
          </svg>
          <p className="text-xs text-amber-200/80 font-medium">
            <span className="text-amber-400 font-bold">*</span> Open findings or incomplete tasks will block case closure
          </p>
        </div>
      )}
    </div>
  );
}
