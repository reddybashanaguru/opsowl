"use client";

import { useState, useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { AICard, AILog } from "./AICard";

interface CopilotPanelProps {
  caseId: string;
  open: boolean;
  onClose: () => void;
}

type Tab = "ask" | "summarize" | "note" | "actions";

const WRITER_ROLES = ["analyst", "manager", "admin", "super_admin"];

export function CopilotPanel({ caseId, open, onClose }: CopilotPanelProps) {
  const [tab, setTab] = useState<Tab>("ask");
  const [question, setQuestion] = useState("");
  const [result, setResult] = useState<AILog | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const canReview = WRITER_ROLES.includes(user?.role ?? "");

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

  const runAction = async (endpoint: string, body?: object) => {
    setError(null);
    setResult(null);
    setLoading(true);
    try {
      const res = await api.post<AILog>(`/api/v1/cases/${caseId}/ai/${endpoint}`, body ?? {});
      setResult(res);
      qc.invalidateQueries({ queryKey: ["ai-logs", caseId] });
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "AI service error");
    } finally {
      setLoading(false);
    }
  };

  if (!open) return null;

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/40 z-40" onClick={onClose} />

      {/* Drawer */}
      <div className="fixed right-0 top-0 h-full w-full max-w-2xl bg-[#0f172a] border-l border-slate-700 z-50 flex flex-col overflow-hidden shadow-2xl" data-testid="copilot-panel">
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
        <div className="flex border-b border-slate-700 px-5">
          {(["ask", "summarize", "note", "actions"] as Tab[]).map((t) => {
            const label = { ask: "Ask", summarize: "Summarize", note: "Draft Note", actions: "Extract Tasks" }[t];
            return (
              <button
                key={t}
                onClick={() => { setTab(t); setResult(null); setError(null); }}
                className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
                  tab === t
                    ? "border-violet-500 text-violet-300"
                    : "border-transparent text-slate-400 hover:text-slate-200"
                }`}
                data-testid={`tab-${t}`}
              >
                {label}
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
                  className="flex-1 bg-[#1e293b] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-violet-500"
                  data-testid="ask-input"
                />
                <button
                  onClick={() => question.trim() && runAction("ask", { question })}
                  disabled={loading || !question.trim()}
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
                disabled={loading}
                className="bg-violet-600 hover:bg-violet-500 text-white text-sm font-medium px-5 py-2 rounded-lg transition-colors disabled:opacity-50"
                data-testid="summarize-btn"
              >
                Generate Summary
              </button>
            </div>
          )}

          {tab === "note" && (
            <div className="space-y-3">
              <p className="text-sm text-slate-400">AI drafts an investigation note from case context. You must review and approve before it's saved.</p>
              <button
                onClick={() => runAction("generate-note")}
                disabled={loading}
                className="bg-violet-600 hover:bg-violet-500 text-white text-sm font-medium px-5 py-2 rounded-lg transition-colors disabled:opacity-50"
                data-testid="note-btn"
              >
                Draft Note
              </button>
            </div>
          )}

          {tab === "actions" && (
            <div className="space-y-3">
              <p className="text-sm text-slate-400">Extract suggested investigation tasks from the uploaded documents.</p>
              <button
                onClick={() => runAction("extract-actions", { source_type: "documents" })}
                disabled={loading}
                className="bg-violet-600 hover:bg-violet-500 text-white text-sm font-medium px-5 py-2 rounded-lg transition-colors disabled:opacity-50"
                data-testid="actions-btn"
              >
                Extract Actions
              </button>
            </div>
          )}

          {/* Loading */}
          {loading && (
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

          {/* Current result */}
          {result && (
            <div>
              <p className="text-xs text-slate-500 mb-2 uppercase tracking-wide">AI Response</p>
              <AICard
                log={result}
                canReview={canReview}
                onApprove={(id) => approveMutation.mutate(id)}
                onReject={(id) => {
                  const reason = window.prompt("Rejection reason:");
                  if (reason) rejectMutation.mutate({ logId: id, reason });
                }}
                onEdit={(id, text) => editMutation.mutate({ logId: id, edited_response: text })}
              />
            </div>
          )}

          {/* Recent logs */}
          {recentLogs.length > 0 && !result && (
            <div>
              <p className="text-xs text-slate-500 mb-3 uppercase tracking-wide">Recent Activity</p>
              <div className="space-y-3">
                {recentLogs.map((log) => (
                  <AICard
                    key={log.id}
                    log={log}
                    canReview={canReview}
                    onApprove={(id) => approveMutation.mutate(id)}
                    onReject={(id) => {
                      const reason = window.prompt("Rejection reason:");
                      if (reason) rejectMutation.mutate({ logId: id, reason });
                    }}
                    onEdit={(id, text) => editMutation.mutate({ logId: id, edited_response: text })}
                  />
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </>
  );
}
