"use client";

import { use, useState, useCallback, useRef, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useAuthStore } from "@/lib/stores/auth.store";
import { useConfig } from "@/lib/hooks/useConfig";
import { useTimelineSSE, type TimelineEvent } from "@/lib/hooks/useTimelineSSE";
import { TimelineEventCard } from "@/components/cases/TimelineEventCard";
import { api } from "@/lib/api";

interface TimelineMeta {
  total_count: number;
  has_more: boolean;
  next_cursor: string | null;
}

interface TimelinePage {
  data: TimelineEvent[];
  meta: TimelineMeta;
}

const SOURCE_OPTIONS = ["system", "user", "ai"] as const;
type Source = (typeof SOURCE_OPTIONS)[number];

const EVENT_TYPE_OPTIONS = [
  "case_created", "case_status_changed", "case_assigned", "case_tagged",
  "note_added", "user_action", "sla_warning", "sla_breached",
] as const;

function StatusDot({ status }: { status: string }) {
  const colors: Record<string, string> = {
    connected:    "bg-green-400",
    connecting:   "bg-amber-400 animate-pulse",
    reconnecting: "bg-amber-400 animate-pulse",
    error:        "bg-red-400",
    closed:       "bg-slate-500",
  };
  return (
    <span className="flex items-center gap-1.5 text-xs text-slate-500">
      <span className={`w-2 h-2 rounded-full ${colors[status] ?? "bg-slate-500"}`} />
      {status}
    </span>
  );
}

export default function TimelinePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const cfg = useConfig();

  const [selectedSources, setSelectedSources] = useState<Set<Source>>(new Set());
  const [selectedTypes, setSelectedTypes] = useState<Set<string>>(new Set());
  const [showFilters, setShowFilters] = useState(false);
  const [liveEvents, setLiveEvents] = useState<TimelineEvent[]>([]);
  const [newCount, setNewCount] = useState(0);
  const feedRef = useRef<HTMLDivElement>(null);
  const isAtTop = useRef(true);

  const buildQueryString = useCallback((cursor?: string) => {
    const p = new URLSearchParams({ limit: "50" });
    if (cursor) p.set("cursor", cursor);
    selectedSources.forEach((s) => p.append("source", s));
    selectedTypes.forEach((t) => p.append("event_type", t));
    return p.toString();
  }, [selectedSources, selectedTypes]);

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
    refetch,
  } = useInfiniteQuery<TimelinePage>({
    queryKey: ["timeline", id, [...selectedSources].sort(), [...selectedTypes].sort()],
    queryFn: ({ pageParam }) =>
      api.getList<TimelineEvent, TimelineMeta>(
        `/api/v1/cases/${id}/timeline?${buildQueryString(pageParam as string | undefined)}`,
      ) as Promise<TimelinePage>,
    initialPageParam: undefined,
    getNextPageParam: (last) => last.meta.next_cursor ?? undefined,
  });

  const handleSSEEvent = useCallback((event: TimelineEvent) => {
    if (isAtTop.current) {
      setLiveEvents((prev) => [event, ...prev]);
    } else {
      setNewCount((n) => n + 1);
    }
  }, []);

  const { status: sseStatus } = useTimelineSSE({
    caseId: id,
    enabled: !!user,
    onEvent: handleSSEEvent,
  });

  useEffect(() => {
    const el = feedRef.current;
    if (!el) return;
    const handler = () => {
      isAtTop.current = el.scrollTop < 100;
    };
    el.addEventListener("scroll", handler);
    return () => el.removeEventListener("scroll", handler);
  }, []);

  const scrollToTopAndFlush = () => {
    feedRef.current?.scrollTo({ top: 0, behavior: "smooth" });
    setNewCount(0);
    isAtTop.current = true;
  };

  const resetFilters = () => {
    setSelectedSources(new Set());
    setSelectedTypes(new Set());
  };

  const toggleSource = (s: Source) =>
    setSelectedSources((prev) => {
      const next = new Set(prev);
      next.has(s) ? next.delete(s) : next.add(s);
      return next;
    });

  const toggleType = (t: string) =>
    setSelectedTypes((prev) => {
      const next = new Set(prev);
      next.has(t) ? next.delete(t) : next.add(t);
      return next;
    });

  const allEvents = data?.pages.flatMap((p) => p.data) ?? [];
  const totalCount = data?.pages[0]?.meta.total_count ?? 0;
  const activeFilters = selectedSources.size + selectedTypes.size;

  return (
    <div className="flex flex-col h-[calc(100vh-56px)]">
      {/* Header */}
      <div className="border-b border-slate-700 bg-[#0f172a] px-6 py-4 flex items-center justify-between flex-shrink-0">
        <div className="flex items-center gap-4">
          <button
            onClick={() => router.push(`/cases/${id}`)}
            className="text-slate-400 hover:text-slate-200 text-sm flex items-center gap-1"
          >
            ← {cfg.case_label}
          </button>
          <div>
            <h1 className="text-white font-semibold text-lg">Timeline</h1>
            <p className="text-slate-500 text-xs">{totalCount} events total</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <StatusDot status={sseStatus} />
          <button
            onClick={() => setShowFilters((v) => !v)}
            className={`flex items-center gap-1.5 text-sm px-3 py-1.5 rounded-lg border transition-colors ${
              activeFilters > 0
                ? "border-cyan-500 text-cyan-400 bg-cyan-900/20"
                : "border-slate-600 text-slate-400 hover:border-slate-500"
            }`}
          >
            Filters
            {activeFilters > 0 && (
              <span className="bg-cyan-500 text-slate-900 text-xs font-bold w-4 h-4 rounded-full flex items-center justify-center">
                {activeFilters}
              </span>
            )}
          </button>
          <button
            onClick={() => { setLiveEvents([]); refetch(); }}
            className="text-slate-400 hover:text-slate-200 text-sm px-3 py-1.5 rounded-lg border border-slate-600 hover:border-slate-500 transition-colors"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Filters panel */}
      {showFilters && (
        <div className="border-b border-slate-700 bg-[#1e293b] px-6 py-4 flex-shrink-0">
          <div className="flex gap-8 flex-wrap">
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wide mb-2">Source</div>
              <div className="flex gap-2">
                {SOURCE_OPTIONS.map((s) => (
                  <button
                    key={s}
                    onClick={() => toggleSource(s)}
                    className={`text-xs px-3 py-1 rounded-full border transition-colors ${
                      selectedSources.has(s)
                        ? "border-cyan-500 text-cyan-400 bg-cyan-900/30"
                        : "border-slate-600 text-slate-400 hover:border-slate-500"
                    }`}
                  >
                    {s}
                  </button>
                ))}
              </div>
            </div>
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wide mb-2">Event Type</div>
              <div className="flex gap-2 flex-wrap">
                {EVENT_TYPE_OPTIONS.map((t) => (
                  <button
                    key={t}
                    onClick={() => toggleType(t)}
                    className={`text-xs px-3 py-1 rounded-full border transition-colors ${
                      selectedTypes.has(t)
                        ? "border-violet-500 text-violet-400 bg-violet-900/30"
                        : "border-slate-600 text-slate-400 hover:border-slate-500"
                    }`}
                  >
                    {t.replace(/_/g, " ")}
                  </button>
                ))}
              </div>
            </div>
          </div>
          {activeFilters > 0 && (
            <button
              onClick={resetFilters}
              className="mt-3 text-xs text-slate-500 hover:text-slate-300 underline"
            >
              Reset all filters
            </button>
          )}
        </div>
      )}

      {/* New events banner */}
      {newCount > 0 && (
        <div className="flex-shrink-0 flex justify-center py-2 bg-[#0f172a]">
          <button
            onClick={scrollToTopAndFlush}
            className="bg-cyan-500 text-slate-900 text-xs font-semibold px-4 py-1.5 rounded-full hover:bg-cyan-400 transition-colors"
          >
            ↑ {newCount} new event{newCount !== 1 ? "s" : ""} — scroll to top
          </button>
        </div>
      )}

      {/* Feed */}
      <div ref={feedRef} className="flex-1 overflow-y-auto px-6 py-6">
        {isLoading ? (
          <div className="flex justify-center py-12">
            <div className="w-6 h-6 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
          </div>
        ) : (
          <>
            {/* Live events prepended at top */}
            {liveEvents.map((e) => (
              <TimelineEventCard key={e.id} event={e} isNew />
            ))}

            {/* Historical events from query */}
            {allEvents.length === 0 && liveEvents.length === 0 ? (
              <div className="text-center py-16">
                <div className="text-4xl mb-3">📋</div>
                <p className="text-slate-400 text-sm">No timeline events yet.</p>
                <p className="text-slate-600 text-xs mt-1">
                  Events will appear here as activity occurs on this case.
                </p>
              </div>
            ) : (
              allEvents.map((e) => (
                <TimelineEventCard key={e.id} event={e} />
              ))
            )}

            {/* Load more */}
            {hasNextPage && (
              <div className="flex justify-center mt-4">
                <button
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                  className="text-slate-400 hover:text-slate-200 text-sm px-4 py-2 border border-slate-700 rounded-lg hover:border-slate-500 transition-colors disabled:opacity-50"
                >
                  {isFetchingNextPage ? "Loading…" : "Load older events"}
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
