"use client";

import { use, useState, useCallback, useRef, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useInfiniteQuery } from "@tanstack/react-query";
import { useAuthStore } from "@/lib/stores/auth.store";
import { useConfig } from "@/lib/hooks/useConfig";
import { useTimelineSSE, type TimelineEvent } from "@/lib/hooks/useTimelineSSE";
import { TimelineEventCard } from "@/components/cases/TimelineEventCard";
import { CaseSubNav } from "@/components/cases/CaseSubNav";
import { api } from "@/lib/api";
import { PageError } from "@/components/ui/PageError";

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
  "note_added", "task_created", "task_completed", "task_status_changed",
  "document_uploaded", "document_processed",
  "ai_query", "ai_summary_generated", "ai_insights_generated",
  "str_draft_created", "str_submitted", "str_approved", "str_rejected", "str_filed",
  "comm_logged",
  "sla_warning", "sla_breached",
  "user_action",
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

import { useCaseSummary } from "@/lib/hooks/useCaseSummary";

export default function TimelinePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const cfg = useConfig();

  const caseData = useCaseSummary(id);

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
    isError,
    error,
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
    <div className="flex flex-col h-full overflow-hidden">
      <div className="px-6 pt-4 flex-shrink-0">
        <CaseSubNav caseId={id} activeTab="timeline" caseNumber={caseData?.case_number} />
      </div>
      <div className="flex flex-col flex-1 overflow-hidden">
      {/* Header */}
      <div className="border-b border-slate-700 bg-[#080e1a] px-6 py-4 flex items-center justify-between flex-shrink-0">
        <div className="flex items-center gap-4">
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
              showFilters || activeFilters > 0
                ? "border-cyan-500 text-cyan-400 bg-cyan-900/20"
                : "border-slate-600 text-slate-400 hover:border-slate-500"
            }`}
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2a1 1 0 01-.293.707L13 13.414V19a1 1 0 01-.553.894l-4 2A1 1 0 017 21v-7.586L3.293 6.707A1 1 0 013 6V4z" />
            </svg>
            Filters
            {activeFilters > 0 ? (
              <span className="bg-cyan-500 text-slate-900 text-xs font-bold w-4 h-4 rounded-full flex items-center justify-center">
                {activeFilters}
              </span>
            ) : (
              <svg className={`w-3 h-3 transition-transform ${showFilters ? "rotate-180" : ""}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
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
        <div className="border-b border-slate-700 bg-[#0d1929] flex-shrink-0">
          {/* Panel header */}
          <div className="flex items-center justify-between px-6 pt-3 pb-2 border-b border-slate-700/60">
            <div className="flex items-center gap-3">
              <span className="text-slate-300 text-sm font-medium">Filter Timeline</span>
              {activeFilters > 0 && (
                <span className="text-xs bg-cyan-500/20 text-cyan-400 border border-cyan-500/30 px-2 py-0.5 rounded-full">
                  {activeFilters} active
                </span>
              )}
            </div>
            <div className="flex items-center gap-3">
              {activeFilters > 0 && (
                <button
                  onClick={resetFilters}
                  className="text-xs text-red-400 hover:text-red-300 flex items-center gap-1 transition-colors"
                >
                  <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                  Clear all
                </button>
              )}
              <button
                onClick={() => setShowFilters(false)}
                className="text-slate-500 hover:text-slate-300 transition-colors p-1 rounded hover:bg-slate-700/50"
                title="Close filters"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>

          {/* Filter options */}
          <div className="px-6 py-4 space-y-4">
            {/* Source */}
            <div>
              <div className="text-slate-500 text-xs uppercase tracking-wide mb-2">Source</div>
              <div className="flex gap-2">
                {SOURCE_OPTIONS.map((s) => {
                  const active = selectedSources.has(s);
                  return (
                    <button
                      key={s}
                      onClick={() => toggleSource(s)}
                      className={`flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-full border font-medium transition-all ${
                        active
                          ? "border-cyan-500 text-cyan-300 bg-cyan-900/40 shadow-[0_0_8px_rgba(6,182,212,0.2)]"
                          : "border-slate-600 text-slate-500 hover:border-slate-400 hover:text-slate-300"
                      }`}
                    >
                      {active && (
                        <svg className="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                        </svg>
                      )}
                      {s}
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Event Type */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <div className="text-slate-500 text-xs uppercase tracking-wide">Event Type</div>
                {selectedTypes.size > 0 && (
                  <button
                    onClick={() => setSelectedTypes(new Set())}
                    className="text-xs text-slate-500 hover:text-slate-400 transition-colors"
                  >
                    clear event types
                  </button>
                )}
              </div>
              <div className="flex gap-2 flex-wrap">
                {EVENT_TYPE_OPTIONS.map((t) => {
                  const active = selectedTypes.has(t);
                  return (
                    <button
                      key={t}
                      onClick={() => toggleType(t)}
                      className={`flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-full border font-medium transition-all ${
                        active
                          ? "border-violet-500 text-violet-300 bg-violet-900/40 shadow-[0_0_8px_rgba(139,92,246,0.2)]"
                          : "border-slate-700 text-slate-500 hover:border-slate-500 hover:text-slate-300"
                      }`}
                    >
                      {active && (
                        <svg className="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                        </svg>
                      )}
                      {t.replace(/_/g, " ")}
                    </button>
                  );
                })}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* New events banner */}
      {newCount > 0 && (
        <div className="flex-shrink-0 flex justify-center py-2 bg-[#080e1a]">
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
        ) : isError ? (
          <PageError error={error as Error | null} onRetry={() => refetch()} />
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
    </div>
  );
}
