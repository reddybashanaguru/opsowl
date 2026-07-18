"use client";

import { use } from "react";
import { Sparkles } from "lucide-react";
import { CopilotPanel } from "@/components/ai/CopilotPanel";
import { useCopilotStore } from "@/lib/stores/copilot.store";

export default function CaseLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const open = useCopilotStore((s) => s.open);
  const setOpen = useCopilotStore((s) => s.setOpen);

  return (
    <>
      {children}

      {/* Persistent floating AI Copilot button — visible across all case tabs */}
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

      <CopilotPanel
        caseId={id}
        open={open}
        onClose={() => setOpen(false)}
      />
    </>
  );
}
