"use client";

import { use } from "react";
import { usePathname } from "next/navigation";
import { Sparkles } from "lucide-react";
import dynamic from "next/dynamic";
const CopilotPanel = dynamic(
  () => import("@/components/ai/CopilotPanel").then(m => ({ default: m.CopilotPanel })),
  { ssr: false, loading: () => <div className="w-full h-full bg-[#0c1929] animate-pulse rounded-xl" /> },
);
import { useCopilotStore } from "@/lib/stores/copilot.store";
import { CaseNavProvider, useCaseNav } from "@/lib/contexts/CaseNavContext";
import { CaseSubNav } from "@/components/cases/CaseSubNav";
import { useCaseSummary } from "@/lib/hooks/useCaseSummary";

type Tab =
  | "case" | "notes" | "tasks" | "documents" | "comms"
  | "insights" | "timeline" | "ai-logs" | "str" | "findings"
  | "closure" | "transactions" | "narrative" | "ubo";

const SEGMENT_TO_TAB: Record<string, Tab> = {
  "":            "case",
  "documents":   "documents",
  "transactions":"transactions",
  "insights":    "insights",
  "tasks":       "tasks",
  "comms":       "comms",
  "notes":       "notes",
  "findings":    "findings",
  "ubo":         "ubo",
  "narrative":   "narrative",
  "str":         "str",
  "ai-logs":     "ai-logs",
  "timeline":    "timeline",
  "closure":     "closure",
};

function tabFromPathname(pathname: string, caseId: string): Tab {
  const base = `/cases/${caseId}`;
  const suffix = pathname.startsWith(base) ? pathname.slice(base.length) : "";
  const segment = suffix.replace(/^\//, "").split("/")[0];
  return SEGMENT_TO_TAB[segment] ?? "case";
}

function CaseLayoutInner({
  id,
  children,
}: {
  id: string;
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const { action } = useCaseNav();
  const open = useCopilotStore((s) => s.open);
  const setOpen = useCopilotStore((s) => s.setOpen);
  const caseData = useCaseSummary(id);
  const activeTab = tabFromPathname(pathname, id);

  return (
    <>
      {/*
        CaseSubNav is flex-shrink-0 (never scrolls away).
        Children get the remaining viewport height in their own scroll container
        so sticky headers inside child pages, and h-full pages (timeline,
        findings, transactions), all work correctly.
      */}
      <div className="h-full flex flex-col">
        <div className="px-6 pt-6 flex-shrink-0">
          <CaseSubNav
            caseId={id}
            activeTab={activeTab}
            caseNumber={caseData?.case_number}
            action={action}
          />
        </div>
        <div className="flex-1 min-h-0 overflow-y-auto">
          {children}
        </div>
      </div>

      {!open && (
        <button
          onClick={() => setOpen(true)}
          className="fixed bottom-6 right-8 z-40 h-10 px-4 rounded-full bg-[#0d1f35] border border-cyan-500/50 text-cyan-300 text-sm font-semibold shadow-lg shadow-cyan-900/50 ring-1 ring-cyan-500/10 flex items-center gap-2 transition-all hover:bg-[#132840] hover:border-cyan-400/70 hover:text-cyan-200 hover:shadow-cyan-800/40 hover:scale-105 active:scale-95"
          title="AI Copilot (⌘⇧A)"
        >
          <Sparkles size={14} className="text-cyan-400 shrink-0" />
          AI Copilot
        </button>
      )}

      <CopilotPanel caseId={id} open={open} onClose={() => setOpen(false)} />
    </>
  );
}

export default function CaseLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);

  return (
    <CaseNavProvider>
      <CaseLayoutInner id={id}>{children}</CaseLayoutInner>
    </CaseNavProvider>
  );
}
