"use client";

import { useState, useEffect, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { api } from "@/lib/api";

// ── Types ─────────────────────────────────────────────────────────────────────

interface CaseResult {
  id: string;
  case_number: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  sla_breached: boolean;
  created_by_name: string;
  created_at: string;
  rank: number;
  entity_type: "case";
}

interface NoteResult {
  id: string;
  case_id: string;
  case_number: string;
  case_title: string;
  title: string;
  content_text: string;
  note_type: string;
  is_pinned: boolean;
  ai_generated: boolean;
  created_at: string;
  rank: number;
  entity_type: "note";
}

interface TaskResult {
  id: string;
  case_id: string;
  case_number: string;
  case_title: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  is_overdue: boolean;
  created_at: string;
  rank: number;
  entity_type: "task";
}

interface DocumentResult {
  id: string;
  case_id: string;
  case_number: string;
  case_title: string;
  original_filename: string;
  mime_type: string;
  processing_status: string;
  file_size_bytes: number;
  created_at: string;
  rank: number;
  entity_type: "document";
}

interface SearchResponse {
  cases: CaseResult[];
  notes: NoteResult[];
  tasks: TaskResult[];
  documents: DocumentResult[];
  query: string;
  total: number;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

const STATUS_COLOR: Record<string, string> = {
  draft: "text-slate-400 bg-slate-700/50",
  open: "text-cyan-400 bg-cyan-400/10",
  in_progress: "text-blue-400 bg-blue-400/10",
  pending_review: "text-amber-400 bg-amber-400/10",
  closed: "text-emerald-400 bg-emerald-400/10",
  escalated: "text-red-400 bg-red-400/10",
  todo: "text-slate-400 bg-slate-700/50",
  done: "text-emerald-400 bg-emerald-400/10",
  cancelled: "text-red-400/70 bg-red-400/10",
};

const PRIORITY_COLOR: Record<string, string> = {
  critical: "text-red-400",
  high: "text-amber-400",
  medium: "text-cyan-400",
  low: "text-slate-400",
};

function highlight(text: string, query: string): string {
  if (!query || !text) return text;
  const words = query.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return text;
  const pattern = words.map((w) => w.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")).join("|");
  return text.replace(new RegExp(`(${pattern})`, "gi"), "<mark>$1</mark>");
}

function HL({ text, query }: { text: string; query: string }) {
  return (
    <span
      className="[&_mark]:bg-cyan-400/25 [&_mark]:text-cyan-200 [&_mark]:rounded-sm [&_mark]:px-0.5"
      dangerouslySetInnerHTML={{ __html: highlight(text, query) }}
    />
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

// ── Result Cards ──────────────────────────────────────────────────────────────

function CaseCard({ r, query }: { r: CaseResult; query: string }) {
  return (
    <Link
      href={`/cases/${r.id}`}
      className="block bg-slate-800/60 border border-slate-700 rounded-xl px-5 py-4 hover:border-cyan-500/40 hover:bg-slate-800 transition-all"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-[10px] font-mono text-slate-500">{r.case_number}</span>
            <span className={`text-[10px] font-semibold px-1.5 py-0.5 rounded-full ${STATUS_COLOR[r.status] ?? "text-slate-400 bg-slate-700"}`}>
              {r.status.replace("_", " ")}
            </span>
            <span className={`text-[10px] font-medium ${PRIORITY_COLOR[r.priority] ?? "text-slate-400"}`}>
              {r.priority}
            </span>
            {r.sla_breached && (
              <span className="text-[10px] font-bold text-red-400 bg-red-400/10 px-1.5 py-0.5 rounded-full">SLA Breached</span>
            )}
          </div>
          <div className="text-slate-200 font-medium text-sm leading-snug">
            <HL text={r.title} query={query} />
          </div>
          {r.description && (
            <p className="text-slate-500 text-xs mt-1 line-clamp-2">
              <HL text={r.description.slice(0, 200)} query={query} />
            </p>
          )}
        </div>
        <span className="text-xs text-slate-600 shrink-0">📋</span>
      </div>
      <div className="mt-2 text-[10px] text-slate-600">By {r.created_by_name}</div>
    </Link>
  );
}

function NoteCard({ r, query }: { r: NoteResult; query: string }) {
  return (
    <Link
      href={`/cases/${r.case_id}/notes`}
      className="block bg-slate-800/60 border border-slate-700 rounded-xl px-5 py-4 hover:border-cyan-500/40 hover:bg-slate-800 transition-all"
    >
      <div className="flex items-start gap-3">
        <span className="text-base shrink-0 mt-0.5">📝</span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-[10px] font-mono text-slate-500">{r.case_number}</span>
            <span className="text-slate-600 text-[10px]">/</span>
            <span className="text-slate-500 text-[10px] truncate">{r.case_title}</span>
            {r.note_type === "formal" && (
              <span className="text-[10px] font-semibold text-amber-400 bg-amber-400/10 px-1.5 py-0.5 rounded-full">Formal</span>
            )}
            {r.ai_generated && (
              <span className="text-[10px] font-semibold text-violet-400 bg-violet-400/10 px-1.5 py-0.5 rounded-full">AI</span>
            )}
            {r.is_pinned && <span className="text-[10px] text-slate-400">📌</span>}
          </div>
          {r.title && (
            <div className="text-slate-200 font-medium text-sm leading-snug">
              <HL text={r.title} query={query} />
            </div>
          )}
          {r.content_text && (
            <p className="text-slate-500 text-xs mt-1 line-clamp-2">
              <HL text={r.content_text.slice(0, 200)} query={query} />
            </p>
          )}
        </div>
      </div>
    </Link>
  );
}

function TaskCard({ r, query }: { r: TaskResult; query: string }) {
  return (
    <Link
      href={`/cases/${r.case_id}/tasks`}
      className="block bg-slate-800/60 border border-slate-700 rounded-xl px-5 py-4 hover:border-cyan-500/40 hover:bg-slate-800 transition-all"
    >
      <div className="flex items-start gap-3">
        <span className="text-base shrink-0 mt-0.5">✓</span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-[10px] font-mono text-slate-500">{r.case_number}</span>
            <span className="text-slate-600 text-[10px]">/</span>
            <span className="text-slate-500 text-[10px] truncate">{r.case_title}</span>
            <span className={`text-[10px] font-semibold px-1.5 py-0.5 rounded-full ${STATUS_COLOR[r.status] ?? "text-slate-400 bg-slate-700"}`}>
              {r.status.replace("_", " ")}
            </span>
            <span className={`text-[10px] font-medium ${PRIORITY_COLOR[r.priority] ?? "text-slate-400"}`}>
              {r.priority}
            </span>
            {r.is_overdue && (
              <span className="text-[10px] font-bold text-red-400 bg-red-400/10 px-1.5 py-0.5 rounded-full">Overdue</span>
            )}
          </div>
          <div className="text-slate-200 font-medium text-sm leading-snug">
            <HL text={r.title} query={query} />
          </div>
          {r.description && (
            <p className="text-slate-500 text-xs mt-1 line-clamp-1">
              <HL text={r.description.slice(0, 150)} query={query} />
            </p>
          )}
        </div>
      </div>
    </Link>
  );
}

function DocumentCard({ r, query }: { r: DocumentResult; query: string }) {
  const mimeIcon: Record<string, string> = {
    "application/pdf": "📄",
    "image/png": "🖼",
    "image/jpeg": "🖼",
  };
  const icon = mimeIcon[r.mime_type] ?? "📎";
  return (
    <Link
      href={`/cases/${r.case_id}/documents`}
      className="block bg-slate-800/60 border border-slate-700 rounded-xl px-5 py-4 hover:border-cyan-500/40 hover:bg-slate-800 transition-all"
    >
      <div className="flex items-start gap-3">
        <span className="text-base shrink-0 mt-0.5">{icon}</span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-[10px] font-mono text-slate-500">{r.case_number}</span>
            <span className="text-slate-600 text-[10px]">/</span>
            <span className="text-slate-500 text-[10px] truncate">{r.case_title}</span>
            <span className="text-[10px] text-slate-600">{formatBytes(r.file_size_bytes)}</span>
          </div>
          <div className="text-slate-200 font-medium text-sm leading-snug font-mono">
            <HL text={r.original_filename} query={query} />
          </div>
        </div>
      </div>
    </Link>
  );
}

// ── Skeleton ──────────────────────────────────────────────────────────────────

function ResultSkeleton() {
  return (
    <div className="space-y-2">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="bg-slate-800/40 border border-slate-700/50 rounded-xl px-5 py-4 animate-pulse">
          <div className="flex items-center gap-2 mb-2">
            <div className="h-3 w-16 bg-slate-700 rounded" />
            <div className="h-3 w-12 bg-slate-700 rounded" />
          </div>
          <div className="h-4 w-2/3 bg-slate-700 rounded mb-2" />
          <div className="h-3 w-full bg-slate-700/60 rounded" />
        </div>
      ))}
    </div>
  );
}

// ── Section ───────────────────────────────────────────────────────────────────

function Section({ title, count, children }: { title: string; count: number; children: React.ReactNode }) {
  if (count === 0) return null;
  return (
    <div>
      <div className="flex items-center gap-2 mb-3">
        <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-widest">{title}</h2>
        <span className="text-[10px] text-slate-600 bg-slate-700/50 px-1.5 py-0.5 rounded-full">{count}</span>
      </div>
      <div className="space-y-2">{children}</div>
    </div>
  );
}

// ── Filter tab ─────────────────────────────────────────────────────────────────

const FILTERS = [
  { label: "All", value: "" },
  { label: "Cases", value: "cases" },
  { label: "Notes", value: "notes" },
  { label: "Tasks", value: "tasks" },
  { label: "Documents", value: "documents" },
];

// ── Page ──────────────────────────────────────────────────────────────────────

export default function SearchPage() {
  const [input, setInput] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [filter, setFilter] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  // Debounce input → query (350ms)
  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(input.trim()), 350);
    return () => clearTimeout(t);
  }, [input]);

  // Focus on mount
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const { data, isLoading, isFetching } = useQuery<SearchResponse>({
    queryKey: ["search", debouncedQuery, filter],
    queryFn: () =>
      api.get<SearchResponse>(
        `/api/v1/search?q=${encodeURIComponent(debouncedQuery)}&filter=${filter}&limit=10`,
      ),
    enabled: debouncedQuery.length >= 2,
    staleTime: 10_000,
  });

  const showSkeleton = (isLoading || isFetching) && debouncedQuery.length >= 2;
  const hasResults = data && data.total > 0;
  const isEmpty = data && data.total === 0 && debouncedQuery.length >= 2 && !isFetching;

  return (
    <div className="max-w-3xl mx-auto px-6 py-8">
      {/* Search bar */}
      <div className="mb-6">
        <div className="relative">
          <span className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-500 text-lg select-none">⌕</span>
          <input
            ref={inputRef}
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Search cases, notes, tasks, documents…"
            className="w-full bg-slate-800 border border-slate-600 rounded-xl pl-11 pr-4 py-3.5 text-slate-200 placeholder-slate-500 text-sm focus:outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500/30 transition-colors"
          />
          {(isLoading || isFetching) && debouncedQuery.length >= 2 && (
            <span className="absolute right-4 top-1/2 -translate-y-1/2 text-slate-500 text-xs animate-pulse">
              Searching…
            </span>
          )}
        </div>

        {/* Filter tabs */}
        <div className="flex items-center gap-1 mt-3">
          {FILTERS.map((f) => (
            <button
              key={f.value}
              onClick={() => setFilter(f.value)}
              className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors ${
                filter === f.value
                  ? "bg-cyan-500/20 text-cyan-400 border border-cyan-500/30"
                  : "text-slate-500 hover:text-slate-300 hover:bg-slate-700/40"
              }`}
            >
              {f.label}
              {data && f.value === "" && data.total > 0 && (
                <span className="ml-1 text-slate-600">{data.total}</span>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* States */}
      {debouncedQuery.length < 2 && !input && (
        <div className="text-center py-20">
          <div className="text-4xl mb-4 opacity-30">⌕</div>
          <p className="text-slate-500 text-sm">Search across all cases, notes, tasks, and documents</p>
          <p className="text-slate-600 text-xs mt-1">Type at least 2 characters to begin</p>
        </div>
      )}

      {debouncedQuery.length === 1 && (
        <div className="text-center py-12">
          <p className="text-slate-500 text-sm">Type at least 2 characters to search</p>
        </div>
      )}

      {showSkeleton && <ResultSkeleton />}

      {isEmpty && (
        <div className="text-center py-16">
          <div className="text-3xl mb-3 opacity-40">⌕</div>
          <p className="text-slate-400 text-sm font-medium">No results for &ldquo;{debouncedQuery}&rdquo;</p>
          <p className="text-slate-600 text-xs mt-1">Try different keywords or remove filters</p>
        </div>
      )}

      {hasResults && !showSkeleton && (
        <div className="space-y-8">
          <Section title="Cases" count={data.cases.length}>
            {data.cases.map((r) => (
              <CaseCard key={r.id} r={r} query={debouncedQuery} />
            ))}
          </Section>

          <Section title="Notes" count={data.notes.length}>
            {data.notes.map((r) => (
              <NoteCard key={r.id} r={r} query={debouncedQuery} />
            ))}
          </Section>

          <Section title="Tasks" count={data.tasks.length}>
            {data.tasks.map((r) => (
              <TaskCard key={r.id} r={r} query={debouncedQuery} />
            ))}
          </Section>

          <Section title="Documents" count={data.documents.length}>
            {data.documents.map((r) => (
              <DocumentCard key={r.id} r={r} query={debouncedQuery} />
            ))}
          </Section>
        </div>
      )}
    </div>
  );
}
