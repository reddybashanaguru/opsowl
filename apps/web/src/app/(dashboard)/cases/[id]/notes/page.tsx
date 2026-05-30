"use client";

import { use, useState, useEffect, useCallback } from "react";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { api, getSseTicket } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { useConfig } from "@/lib/hooks/useConfig";
import { NoteEditor } from "@/components/notes/NoteEditor";
import { NoteCard, Note } from "@/components/notes/NoteCard";
import { CaseSubNav } from "@/components/cases/CaseSubNav";

interface ListMeta {
  next_cursor: string | null;
  has_more: boolean;
  total_count: number;
}

interface NotesPage {
  data: Note[];
  meta: ListMeta;
}

function SkeletonNote() {
  return (
    <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4 space-y-2 animate-pulse">
      <div className="h-4 bg-slate-700 rounded w-3/4" />
      <div className="h-3 bg-slate-700/60 rounded w-1/4" />
      <div className="space-y-1.5 mt-2">
        <div className="h-3 bg-slate-700/40 rounded w-full" />
        <div className="h-3 bg-slate-700/40 rounded w-5/6" />
        <div className="h-3 bg-slate-700/40 rounded w-2/3" />
      </div>
    </div>
  );
}

export default function CaseNotesPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const cfg = useConfig();

  const [showEditor, setShowEditor] = useState(false);
  const [filterType, setFilterType] = useState("");

  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useInfiniteQuery<NotesPage>({
      queryKey: ["case-notes", id, filterType],
      queryFn: ({ pageParam }) => {
        const p = new URLSearchParams({ limit: "20" });
        if (filterType) p.set("note_type", filterType);
        if (pageParam) p.set("cursor", pageParam as string);
        return api.getList<Note, ListMeta>(`/api/v1/cases/${id}/notes?${p}`);
      },
      initialPageParam: undefined,
      getNextPageParam: (last) => last.meta.next_cursor ?? undefined,
    });

  const allNotes = data?.pages.flatMap((p) => p.data) ?? [];
  const total = data?.pages[0]?.meta.total_count ?? 0;

  // SSE for real-time updates — use short-lived ticket so the access token is
  // never exposed in a URL.
  useEffect(() => {
    let es: EventSource | null = null;
    getSseTicket()
      .then((ticket) => {
        const url = `http://localhost:8080/api/v1/cases/${id}/notes/stream?token=${ticket}`;
        es = new EventSource(url);
        es.onmessage = () => {
          qc.invalidateQueries({ queryKey: ["case-notes", id] });
        };
      })
      .catch(() => {/* ignore — notes fall back to manual refresh */});
    return () => es?.close();
  }, [id, qc]);

  const createMutation = useMutation({
    mutationFn: (body: {
      title: string;
      content: string;
      content_text: string;
      note_type: string;
      ai_generated: boolean;
      mentioned_user_ids: string[];
    }) => api.post<Note>(`/api/v1/cases/${id}/notes`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["case-notes", id] });
      setShowEditor(false);
    },
  });

  const handleSave = useCallback(async (data: {
    title: string;
    content: string;
    contentText: string;
    noteType: "internal" | "formal";
    mentionedUserIds: string[];
  }) => {
    await createMutation.mutateAsync({
      title: data.title,
      content: data.content,
      content_text: data.contentText,
      note_type: data.noteType,
      ai_generated: false,
      mentioned_user_ids: data.mentionedUserIds,
    });
  }, [createMutation]);

  const canWrite = ["analyst", "manager", "admin", "super_admin"].includes(user?.role ?? "");

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <CaseSubNav caseId={id} activeTab="notes" />

      {/* Header */}
      <div className="flex items-center justify-between mb-5">
        <div>
          <h1 className="text-white text-xl font-bold">{cfg.notes_label}</h1>
          <p className="text-slate-500 text-sm mt-0.5">{total} {total !== 1 ? cfg.notes_label.toLowerCase() : cfg.note_label.toLowerCase()}</p>
        </div>
        <div className="flex items-center gap-2">
          {/* Filter */}
          <select
            value={filterType}
            onChange={(e) => setFilterType(e.target.value)}
            className="bg-[#1e293b] border border-slate-600 text-slate-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
          >
            <option value="">All Types</option>
            <option value="internal">Internal</option>
            <option value="formal">Formal</option>
          </select>
          {canWrite && !showEditor && (
            <button
              onClick={() => setShowEditor(true)}
              className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-2 rounded-lg text-sm transition-colors"
            >
              + New Note
            </button>
          )}
        </div>
      </div>

      {/* New note editor */}
      {showEditor && (
        <div className="mb-5">
          <NoteEditor
            caseId={id}
            onSave={handleSave}
            onCancel={() => setShowEditor(false)}
            isLoading={createMutation.isPending}
          />
        </div>
      )}

      {/* Notes list */}
      <div className="space-y-4">
        {isLoading ? (
          [...Array(3)].map((_, i) => <SkeletonNote key={i} />)
        ) : allNotes.length === 0 ? (
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-16 text-center">
            <div className="text-4xl mb-3">📝</div>
            <p className="text-slate-400 text-sm font-medium">No notes yet</p>
            <p className="text-slate-600 text-xs mt-1">Document your findings and observations</p>
            {canWrite && (
              <button
                onClick={() => setShowEditor(true)}
                className="mt-4 text-cyan-400 hover:text-cyan-300 text-sm transition-colors"
              >
                Write the first note →
              </button>
            )}
          </div>
        ) : (
          <>
            {allNotes.map((note) => (
              <NoteCard
                key={note.id}
                note={note}
                caseId={id}
                onDeleted={() => qc.invalidateQueries({ queryKey: ["case-notes", id] })}
              />
            ))}

            {hasNextPage && (
              <div className="text-center pt-2">
                <button
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                  className="text-slate-400 hover:text-cyan-400 text-sm transition-colors disabled:opacity-50"
                >
                  {isFetchingNextPage ? (
                    <span className="flex items-center gap-2 justify-center">
                      <span className="w-3 h-3 border border-cyan-400 border-t-transparent rounded-full animate-spin" />
                      Loading…
                    </span>
                  ) : (
                    "Load more notes"
                  )}
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
