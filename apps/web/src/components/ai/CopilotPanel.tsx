"use client";

import { useState, useEffect, useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { useAgentStream, AgentEvent } from "@/lib/hooks/useAgentStream";
import { AICard, AILog, renderMarkdown } from "./AICard";

interface CopilotPanelProps {
  caseId: string;
  open: boolean;
  onClose: () => void;
}

type Tab = "ask" | "summarize" | "note" | "actions" | "agent";

const TAB_ENDPOINT: Partial<Record<Tab, string>> = {
  ask: "ask",
  summarize: "summarize",
  note: "generate-note",
  actions: "extract-actions",
};

// Maps each tab to the action_type stored in the DB log
const TAB_ACTION_TYPE: Partial<Record<Tab, string>> = {
  ask: "ask",
  summarize: "summarize",
  note: "generate_note",
  actions: "extract_actions",
  agent: "agent_run",
};

export function CopilotPanel({ caseId, open, onClose }: CopilotPanelProps) {
  const [tab, setTab] = useState<Tab>("ask");
  const [question, setQuestion] = useState("");
  const [agentGoal, setAgentGoal] = useState("");
  const [result, setResult] = useState<AILog | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loadingEndpoint, setLoadingEndpoint] = useState<string | null>(null);
  const [rejectingLogId, setRejectingLogId] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState("");
  const feedEndRef = useRef<HTMLDivElement>(null);
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const hasRole = useAuthStore((s) => s.hasRole);
  const agent = useAgentStream(caseId);

  // Cmd+Shift+A shortcut to toggle
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key === "a") {
        e.preventDefault();
        onClose();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [onClose]);

  // Auto-scroll agent feed to bottom as new events arrive
  useEffect(() => {
    if (tab === "agent") {
      feedEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [agent.events.length, tab]);

  // Recent logs for this case
  const { data: logsData } = useQuery<{ data: AILog[] }>({
    queryKey: ["ai-logs", caseId],
    queryFn: () => api.get<{ data: AILog[] }>(`/api/v1/cases/${caseId}/ai/logs?limit=5`),
    enabled: open,
  });
  const recentLogs = logsData?.data ?? [];

  // Review mutations
  const approveMutation = useMutation({
    mutationFn: (logId: string) => api.post(`/api/v1/ai/logs/${logId}/approve`, {}),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["ai-logs", caseId] }); setResult(null); },
  });
  const rejectMutation = useMutation({
    mutationFn: ({ logId, reason }: { logId: string; reason: string }) =>
      api.post(`/api/v1/ai/logs/${logId}/reject`, { reason }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["ai-logs", caseId] }); setResult(null); },
  });
  const editMutation = useMutation({
    mutationFn: ({ logId, edited_response }: { logId: string; edited_response: string }) =>
      api.put(`/api/v1/ai/logs/${logId}/edit`, { edited_response }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["ai-logs", caseId] }); setResult(null); },
  });

  // Auto-save agent result to ai_logs when the agent finishes so it persists
  // in the AI Activity Log and survives copilot panel close/reopen.
  const saveAgentMutation = useMutation({
    mutationFn: (payload: { answer: string; goal: string; tokens_in: number; tokens_out: number; elapsed_ms: number; model_used: string; provider: string }) =>
      api.post(`/api/v1/cases/${caseId}/ai/agent-result`, payload),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["ai-logs", caseId] }); },
  });
  const savedAgentAnswerRef = useRef<string | null>(null);
  useEffect(() => {
    if (agent.answer && agent.stats && agent.answer !== savedAgentAnswerRef.current) {
      savedAgentAnswerRef.current = agent.answer;
      saveAgentMutation.mutate({
        answer: agent.answer,
        goal: agentGoal,
        tokens_in: agent.stats.tokens_in,
        tokens_out: agent.stats.tokens_out,
        elapsed_ms: agent.stats.elapsed_ms,
        model_used: "llama3.2:3b",
        provider: "ollama",
      });
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agent.answer, agent.stats]);

  const runAction = async (endpoint: string, body?: object) => {
    setError(null);
    setResult(null);
    setLoadingEndpoint(endpoint);
    try {
      const res = await api.post<AILog>(`/api/v1/cases/${caseId}/ai/${endpoint}`, body ?? {});
      setResult(res);
      qc.invalidateQueries({ queryKey: ["ai-logs", caseId] });
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "AI service error");
    } finally {
      setLoadingEndpoint(null);
    }
  };

  if (!open) return null;

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/40 z-40" onClick={onClose} />

      {/* Drawer */}
      <div className="fixed right-0 top-0 h-full w-full max-w-2xl bg-[#080e1a] border-l border-slate-700 z-50 flex flex-col overflow-hidden shadow-2xl" data-testid="copilot-panel">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-700">
          <div className="flex items-center gap-2">
            <span className="text-violet-400 font-bold text-lg">🤖</span>
            <h2 className="text-white font-semibold">AI Copilot</h2>
            <span className="text-xs text-slate-500 ml-1">⌘⇧A</span>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-200 text-xl font-light">✕</button>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-slate-700 px-5 overflow-x-auto">
          {(["ask", "summarize", "note", "actions", "agent"] as Tab[]).map((t) => {
            const label: Record<Tab, string> = {
              ask: "Ask",
              summarize: "Summarize",
              note: "Draft Note",
              actions: "Extract Tasks",
              agent: "⚡ Agent",
            };
            return (
              <button
                key={t}
                onClick={() => { setTab(t); setResult(null); setError(null); /* result falls back to recentLogs per tab */ }}
                className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors whitespace-nowrap ${
                  tab === t
                    ? t === "agent"
                      ? "border-cyan-500 text-cyan-300"
                      : "border-violet-500 text-violet-300"
                    : "border-transparent text-slate-400 hover:text-slate-200"
                }`}
                data-testid={`tab-${t}`}
              >
                {label[t]}
              </button>
            );
          })}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-5 space-y-4">
          {/* Ask */}
          {tab === "ask" && (
            <div className="space-y-3">
              <div className="flex gap-2">
                <input
                  type="text"
                  value={question}
                  onChange={(e) => setQuestion(e.target.value)}
                  onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey && question.trim()) runAction("ask", { question }); }}
                  placeholder="Ask a question about the documents…"
                  maxLength={500}
                  className="flex-1 bg-[#131f30] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-violet-500"
                  data-testid="ask-input"
                />
                <button
                  onClick={() => question.trim() && runAction("ask", { question })}
                  disabled={loadingEndpoint === "ask" || !question.trim()}
                  className="bg-violet-600 hover:bg-violet-500 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors disabled:opacity-50"
                  data-testid="ask-submit"
                >
                  Ask
                </button>
              </div>
              <p className="text-xs text-slate-500">Based on uploaded documents only. Uses RAG + pgvector similarity search.</p>
            </div>
          )}

          {tab === "summarize" && (
            <div className="space-y-3">
              <p className="text-sm text-slate-400">Generate a structured summary of all documents uploaded to this case.</p>
              <button
                onClick={() => runAction("summarize")}
                disabled={loadingEndpoint === "summarize"}
                className="bg-violet-600 hover:bg-violet-500 text-white text-sm font-medium px-5 py-2 rounded-lg transition-colors disabled:opacity-50"
                data-testid="summarize-btn"
              >
                {loadingEndpoint === "summarize" ? "Generating…" : "Generate Summary"}
              </button>
            </div>
          )}

          {tab === "note" && (
            <div className="space-y-3">
              <p className="text-sm text-slate-400">AI drafts an investigation note from case context. You must review and approve before it's saved.</p>
              <button
                onClick={() => runAction("generate-note")}
                disabled={loadingEndpoint === "generate-note"}
                className="bg-violet-600 hover:bg-violet-500 text-white text-sm font-medium px-5 py-2 rounded-lg transition-colors disabled:opacity-50"
                data-testid="note-btn"
              >
                {loadingEndpoint === "generate-note" ? "Drafting…" : "Draft Note"}
              </button>
            </div>
          )}

          {tab === "actions" && (
            <div className="space-y-3">
              <p className="text-sm text-slate-400">Extract suggested investigation tasks from the uploaded documents.</p>
              <button
                onClick={() => runAction("extract-actions", { source_type: "documents" })}
                disabled={loadingEndpoint === "extract-actions"}
                className="bg-violet-600 hover:bg-violet-500 text-white text-sm font-medium px-5 py-2 rounded-lg transition-colors disabled:opacity-50"
                data-testid="actions-btn"
              >
                {loadingEndpoint === "extract-actions" ? "Extracting…" : "Extract Actions"}
              </button>
            </div>
          )}

          {/* ── Agent tab ── */}
          {tab === "agent" && (
            <div className="space-y-4">
              <div>
                <p className="text-sm text-slate-300 font-medium mb-1">Autonomous Investigation</p>
                <p className="text-xs text-slate-500">
                  The agent autonomously reads case data (notes, tasks, findings, transactions, timeline) and
                  produces a structured investigation report. All actions are read-only.
                </p>
              </div>

              {/* Goal input */}
              {!agent.running && !agent.answer && (
                <div className="space-y-2">
                  <textarea
                    value={agentGoal}
                    onChange={(e) => setAgentGoal(e.target.value)}
                    placeholder="What should the agent investigate? e.g. 'Analyse this case for suspicious transaction patterns and summarise the risk'"
                    rows={3}
                    maxLength={1000}
                    className="w-full bg-[#131f30] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500 resize-none"
                    data-testid="agent-goal-input"
                  />
                  <button
                    onClick={() => agentGoal.trim() && agent.run(agentGoal)}
                    disabled={!agentGoal.trim()}
                    className="bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-semibold px-5 py-2 rounded-lg transition-colors disabled:opacity-40"
                    data-testid="agent-run-btn"
                  >
                    ⚡ Run Agent
                  </button>
                </div>
              )}

              {/* Running header */}
              {agent.running && (
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 text-cyan-400 text-sm">
                    <div className="w-3 h-3 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
                    Agent is investigating…
                  </div>
                  <button
                    onClick={agent.cancel}
                    className="text-xs text-slate-500 hover:text-red-400 transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              )}

              {/* Event feed */}
              {agent.events.length > 0 && (
                <div className="space-y-1.5" data-testid="agent-feed">
                  {agent.events.map((evt, i) => (
                    <AgentEventRow key={i} event={evt} />
                  ))}
                  <div ref={feedEndRef} />
                </div>
              )}

              {/* Final answer */}
              {agent.answer && (
                <div className="rounded-xl border border-cyan-500/20 bg-cyan-500/5 p-4">
                  <p className="text-xs text-cyan-400 font-semibold uppercase tracking-wider mb-2">Investigation Report</p>
                  <div className="text-sm leading-relaxed">{renderMarkdown(agent.answer)}</div>
                  {agent.stats && (
                    <p className="text-xs text-slate-600 mt-3">
                      {agent.stats.iterations} iterations · {agent.stats.tokens_in + agent.stats.tokens_out} tokens ·{" "}
                      {(agent.stats.elapsed_ms / 1000).toFixed(1)}s
                    </p>
                  )}
                  <button
                    onClick={agent.reset}
                    className="mt-3 text-xs text-slate-500 hover:text-slate-300 transition-colors"
                  >
                    Run again
                  </button>
                </div>
              )}

              {/* Agent error */}
              {agent.error && (
                <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-400">
                  {agent.error}
                  <button onClick={agent.reset} className="ml-3 underline text-xs">Reset</button>
                </div>
              )}
            </div>
          )}

          {/* Loading — only shown when THIS tab's request is in-flight */}
          {loadingEndpoint !== null && loadingEndpoint === TAB_ENDPOINT[tab] && (
            <div className="flex items-center gap-3 text-slate-400 text-sm" data-testid="loading-indicator">
              <div className="w-4 h-4 border-2 border-violet-400 border-t-transparent rounded-full animate-spin" />
              AI is thinking…
            </div>
          )}

          {/* Error */}
          {error && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-400" role="alert">
              {error}
            </div>
          )}

          {/* Inline reject form */}
          {rejectingLogId && (
            <div className="bg-[#131f30] border border-red-500/30 rounded-xl p-4">
              <p className="text-sm text-slate-300 mb-2 font-medium">Rejection reason</p>
              <textarea
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                rows={2}
                placeholder="Optional — describe why this response is being rejected"
                className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-red-400 resize-none"
              />
              <div className="flex gap-2 mt-2">
                <button
                  onClick={() => {
                    rejectMutation.mutate({ logId: rejectingLogId, reason: rejectReason });
                    setRejectingLogId(null);
                    setRejectReason("");
                  }}
                  className="text-xs bg-red-600 hover:bg-red-500 text-white px-3 py-1.5 rounded-lg transition-colors"
                >
                  Confirm Rejection
                </button>
                <button
                  onClick={() => { setRejectingLogId(null); setRejectReason(""); }}
                  className="text-xs text-slate-400 hover:text-slate-200 px-3 py-1.5 rounded-lg border border-slate-600 transition-colors"
                >
                  Cancel
                </button>
              </div>
            </div>
          )}

          {/* Current result — falls back to most recent DB log for this tab so switching
              tabs and returning does not lose the output */}
          {(() => {
            const tabActionType = TAB_ACTION_TYPE[tab];
            const displayResult = result ?? (
              tabActionType
                ? recentLogs.find((l) => l.action_type === tabActionType) ?? null
                : null
            );
            const canReviewLog = (log: AILog) =>
              hasRole("analyst") && log.triggered_by?.id !== user?.id;

            if (displayResult) {
              return (
                <div>
                  <p className="text-xs text-slate-500 mb-2 uppercase tracking-wider">AI Response</p>
                  <AICard
                    log={displayResult}
                    canReview={canReviewLog(displayResult)}
                    defaultExpanded
                    onApprove={(id) => approveMutation.mutate(id)}
                    onReject={(id) => { setRejectingLogId(id); setRejectReason(""); }}
                    onEdit={(id, text) => editMutation.mutate({ logId: id, edited_response: text })}
                  />
                </div>
              );
            }

            // No result for this tab yet — show recent activity from other actions
            const otherLogs = recentLogs.filter((l) => l.action_type !== tabActionType);
            if (otherLogs.length > 0 && !tabActionType) {
              return (
                <div>
                  <p className="text-xs text-slate-500 mb-3 uppercase tracking-wider">Recent Activity</p>
                  <div className="space-y-3">
                    {otherLogs.map((log) => (
                      <AICard
                        key={log.id}
                        log={log}
                        canReview={canReviewLog(log)}
                        defaultExpanded
                        onApprove={(id) => approveMutation.mutate(id)}
                        onReject={(id) => { setRejectingLogId(id); setRejectReason(""); }}
                        onEdit={(id, text) => editMutation.mutate({ logId: id, edited_response: text })}
                      />
                    ))}
                  </div>
                </div>
              );
            }

            return null;
          })()}
        </div>
      </div>
    </>
  );
}

// ── Agent event feed row ──────────────────────────────────────────────────────

const TOOL_LABEL: Record<string, string> = {
  get_case_context: "Case Context",
  get_case_notes: "Notes",
  get_case_tasks: "Tasks",
  get_case_findings: "Findings",
  get_case_transactions: "Transactions",
  get_case_timeline: "Timeline",
  get_case_documents: "Documents",
  search_similar_cases: "Similar Cases",
};

function AgentEventRow({ event }: { event: AgentEvent }) {
  const { type, data } = event;

  if (type === "thinking") {
    const text = (data.text as string) ?? "";
    if (!text.trim()) return null;
    return (
      <div className="text-xs text-slate-400 leading-relaxed pl-2 border-l border-slate-700 italic">
        {text.length > 200 ? text.slice(0, 200) + "…" : text}
      </div>
    );
  }

  if (type === "tool_call") {
    const name = (data.name as string) ?? "";
    return (
      <div className="flex items-center gap-2 text-xs text-slate-400 py-0.5">
        <span className="text-cyan-500">→</span>
        <span className="text-slate-300 font-medium">{TOOL_LABEL[name] ?? name}</span>
        <span className="text-slate-600">fetching…</span>
      </div>
    );
  }

  if (type === "tool_result") {
    const name = (data.name as string) ?? "";
    const ok = data.ok as boolean;
    const preview = (data.preview as string) ?? "";
    return (
      <div className="text-xs py-0.5">
        <span className={ok ? "text-emerald-500" : "text-red-500"}>
          {ok ? "✓" : "✗"}
        </span>
        <span className="text-slate-400 ml-1.5">{TOOL_LABEL[name] ?? name}:</span>
        <span className="text-slate-500 ml-1">{preview.slice(0, 100)}{preview.length > 100 ? "…" : ""}</span>
      </div>
    );
  }

  if (type === "error") {
    return (
      <div className="text-xs text-red-400 pl-2 border-l border-red-500/30">
        Error: {(data.message as string) ?? "unknown"}
      </div>
    );
  }

  return null;
}
