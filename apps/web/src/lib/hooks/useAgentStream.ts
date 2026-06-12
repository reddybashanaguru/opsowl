"use client";

import { useState, useRef, useCallback } from "react";

export type AgentEventType =
  | "agent_start"
  | "thinking"
  | "tool_call"
  | "tool_result"
  | "answer"
  | "done"
  | "error";

export interface AgentEvent {
  type: AgentEventType;
  data: Record<string, unknown>;
  timestamp: number;
}

interface AgentStreamState {
  running: boolean;
  events: AgentEvent[];
  answer: string | null;
  error: string | null;
  stats: { iterations: number; tokens_in: number; tokens_out: number; elapsed_ms: number } | null;
}

export function useAgentStream(caseId: string) {
  const [state, setState] = useState<AgentStreamState>({
    running: false,
    events: [],
    answer: null,
    error: null,
    stats: null,
  });
  const abortRef = useRef<AbortController | null>(null);

  const run = useCallback(
    async (goal: string) => {
      if (!goal.trim()) return;

      // Cancel any in-flight run
      abortRef.current?.abort();
      const ctrl = new AbortController();
      abortRef.current = ctrl;

      setState({ running: true, events: [], answer: null, error: null, stats: null });

      try {
        const res = await fetch(`/api/v1/cases/${caseId}/ai/agent`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ goal }),
          signal: ctrl.signal,
        });

        if (!res.ok) {
          const msg = await res.text();
          setState((s) => ({ ...s, running: false, error: `Agent error: ${res.status} ${msg}` }));
          return;
        }

        const reader = res.body?.getReader();
        if (!reader) {
          setState((s) => ({ ...s, running: false, error: "No response stream" }));
          return;
        }

        const decoder = new TextDecoder();
        let buffer = "";

        // eslint-disable-next-line no-constant-condition
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });

          // SSE messages are separated by double newlines
          const parts = buffer.split("\n\n");
          buffer = parts.pop() ?? ""; // last partial chunk stays in buffer

          for (const part of parts) {
            const lines = part.split("\n");
            let eventType = "";
            let dataLine = "";

            for (const line of lines) {
              if (line.startsWith("event: ")) eventType = line.slice(7).trim();
              if (line.startsWith("data: ")) dataLine = line.slice(6).trim();
            }

            if (!eventType || !dataLine) continue;

            let parsed: Record<string, unknown> = {};
            try {
              parsed = JSON.parse(dataLine);
            } catch {
              continue;
            }

            const evt: AgentEvent = {
              type: eventType as AgentEventType,
              data: parsed,
              timestamp: Date.now(),
            };

            setState((s) => {
              const updates: Partial<AgentStreamState> = {
                events: [...s.events, evt],
              };
              if (eventType === "answer") {
                updates.answer = parsed.text as string;
              }
              if (eventType === "done") {
                updates.running = false;
                updates.stats = parsed as AgentStreamState["stats"];
              }
              if (eventType === "error") {
                updates.running = false;
                updates.error = parsed.message as string;
              }
              return { ...s, ...updates };
            });
          }
        }
      } catch (err: unknown) {
        if ((err as Error)?.name === "AbortError") return;
        setState((s) => ({
          ...s,
          running: false,
          error: err instanceof Error ? err.message : "Stream error",
        }));
      } finally {
        setState((s) => (s.running ? { ...s, running: false } : s));
      }
    },
    [caseId]
  );

  const cancel = useCallback(() => {
    abortRef.current?.abort();
    setState((s) => ({ ...s, running: false }));
  }, []);

  const reset = useCallback(() => {
    abortRef.current?.abort();
    setState({ running: false, events: [], answer: null, error: null, stats: null });
  }, []);

  return { ...state, run, cancel, reset };
}

