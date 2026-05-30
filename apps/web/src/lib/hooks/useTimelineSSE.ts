"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { getSseTicket } from "@/lib/api";

export type SSEStatus = "connecting" | "connected" | "reconnecting" | "error" | "closed";

export interface TimelineEvent {
  id: string;
  case_id: string;
  event_type: string;
  source: "system" | "user" | "ai";
  actor: { id: string; full_name: string; email: string } | null;
  occurred_at: string;
  created_at: string;
  payload: Record<string, unknown>;
  back_dated: boolean;
}

interface UseTimelineSSEOptions {
  caseId: string;
  enabled?: boolean;
  onEvent?: (event: TimelineEvent) => void;
}

const BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const MAX_RECONNECT_DELAY_MS = 30_000;

export function useTimelineSSE({ caseId, enabled = true, onEvent }: UseTimelineSSEOptions) {
  const [status, setStatus] = useState<SSEStatus>("connecting");
  const esRef = useRef<EventSource | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectDelay = useRef(1000);
  const lastEventId = useRef<string | null>(null);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  const connect = useCallback(async () => {
    if (!enabled) return;

    // Close any existing connection before opening a new one.
    if (esRef.current) {
      esRef.current.close();
      esRef.current = null;
    }

    let token: string;
    try {
      token = await getSseTicket();
    } catch {
      // Session likely expired — stop reconnecting.
      setStatus("error");
      return;
    }

    const url = new URL(`${BASE_URL}/api/v1/cases/${caseId}/timeline/stream`);
    url.searchParams.set("token", token);
    if (lastEventId.current) {
      url.searchParams.set("last_event_id", lastEventId.current);
    }

    const es = new EventSource(url.toString());
    esRef.current = es;
    setStatus("connecting");

    es.addEventListener("connected", () => {
      setStatus("connected");
      reconnectDelay.current = 1000;
    });

    es.addEventListener("timeline_event", (e: MessageEvent) => {
      try {
        const event = JSON.parse(e.data) as TimelineEvent;
        if (e.lastEventId) lastEventId.current = e.lastEventId;
        onEventRef.current?.(event);
      } catch {
        // malformed event — ignore
      }
    });

    es.addEventListener("ping", () => {
      // heartbeat — keep-alive, no action needed
    });

    es.onerror = () => {
      // Always close immediately so the browser never auto-retries with a stale token.
      es.close();
      esRef.current = null;
      setStatus("reconnecting");

      reconnectTimer.current = setTimeout(() => {
        reconnectDelay.current = Math.min(reconnectDelay.current * 2, MAX_RECONNECT_DELAY_MS);
        connect();
      }, reconnectDelay.current);
    };
  }, [caseId, enabled]);

  useEffect(() => {
    if (!enabled) {
      setStatus("closed");
      return;
    }
    connect();
    return () => {
      esRef.current?.close();
      esRef.current = null;
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
      setStatus("closed");
    };
  }, [connect, enabled]);

  return { status };
}
