"use client";

import { useState } from "react";
import type { TimelineEvent } from "@/lib/hooks/useTimelineSSE";

// ─── event type config ────────────────────────────────────────────────────────

interface EventConfig {
  icon: string;
  label: string;
  accent: string; // tailwind text color
  bg: string;     // tailwind bg color
}

const EVENT_CONFIG: Record<string, EventConfig> = {
  case_created:            { icon: "✦",  label: "Case Created",           accent: "text-cyan-400",   bg: "bg-cyan-900/30" },
  case_status_changed:     { icon: "⟳",  label: "Status Changed",         accent: "text-blue-400",   bg: "bg-blue-900/30" },
  case_assigned:           { icon: "→",  label: "Assigned",               accent: "text-violet-400", bg: "bg-violet-900/30" },
  case_tagged:             { icon: "#",  label: "Tags Updated",           accent: "text-amber-400",  bg: "bg-amber-900/30" },
  document_uploaded:       { icon: "↑",  label: "Document Uploaded",      accent: "text-green-400",  bg: "bg-green-900/30" },
  document_processed:      { icon: "⚙", label: "Document Processed",     accent: "text-green-300",  bg: "bg-green-900/20" },
  document_ai_analyzed:    { icon: "✦",  label: "AI Analyzed",            accent: "text-violet-300", bg: "bg-violet-900/20" },
  note_added:              { icon: "✎",  label: "Note Added",             accent: "text-slate-300",  bg: "bg-slate-700/40" },
  note_edited:             { icon: "✎",  label: "Note Edited",            accent: "text-slate-400",  bg: "bg-slate-700/30" },
  comment_added:           { icon: "✉",  label: "Comment Added",          accent: "text-slate-300",  bg: "bg-slate-700/40" },
  task_created:            { icon: "□",  label: "Task Created",           accent: "text-amber-300",  bg: "bg-amber-900/20" },
  task_completed:          { icon: "✔",  label: "Task Completed",         accent: "text-green-400",  bg: "bg-green-900/30" },
  task_overdue:            { icon: "!",  label: "Task Overdue",           accent: "text-red-400",    bg: "bg-red-900/30" },
  ai_query:                { icon: "?",  label: "AI Query",               accent: "text-violet-400", bg: "bg-violet-900/30" },
  ai_summary_generated:    { icon: "✦",  label: "AI Summary",             accent: "text-violet-300", bg: "bg-violet-900/20" },
  ai_action_suggested:     { icon: "→",  label: "AI Suggestion",         accent: "text-violet-400", bg: "bg-violet-900/30" },
  user_login:              { icon: "⏵",  label: "User Login",             accent: "text-slate-400",  bg: "bg-slate-700/30" },
  user_action:             { icon: "⏵",  label: "User Action",            accent: "text-slate-300",  bg: "bg-slate-700/40" },
  sla_warning:             { icon: "⚠",  label: "SLA Warning",            accent: "text-amber-400",  bg: "bg-amber-900/30" },
  sla_breached:            { icon: "⚠",  label: "SLA Breached",           accent: "text-red-400",    bg: "bg-red-900/30" },
  notification_sent:       { icon: "✉",  label: "Notification Sent",      accent: "text-slate-400",  bg: "bg-slate-700/30" },
};

const SOURCE_COLORS: Record<string, string> = {
  system: "text-slate-400",
  user:   "text-blue-400",
  ai:     "text-violet-400",
};

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const s = Math.floor(diff / 1000);
  if (s < 60)  return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60)  return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24)  return `${h}h ago`;
  const d = Math.floor(h / 24);
  return `${d}d ago`;
}

function PayloadView({ payload }: { payload: Record<string, unknown> }) {
  if (Object.keys(payload).length === 0) return null;
  return (
    <pre className="mt-3 text-xs font-mono text-slate-400 bg-[#0d1b2e] border border-slate-700/50 rounded-lg p-3 overflow-x-auto whitespace-pre-wrap break-words">
      {JSON.stringify(payload, null, 2)}
    </pre>
  );
}

interface Props {
  event: TimelineEvent;
  isNew?: boolean;
}

export function TimelineEventCard({ event, isNew }: Props) {
  const [expanded, setExpanded] = useState(false);
  const cfg = EVENT_CONFIG[event.event_type] ?? { icon: "·", label: event.event_type, accent: "text-slate-400", bg: "bg-slate-700/30" };
  const actorName = event.actor?.full_name ?? "System";

  return (
    <div
      className={`relative flex gap-3 group cursor-pointer select-none ${isNew ? "animate-pulse-once" : ""}`}
      onClick={() => setExpanded((v) => !v)}
    >
      {/* Timeline line */}
      <div className="flex flex-col items-center">
        <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold flex-shrink-0 ${cfg.bg} ${cfg.accent}`}>
          {cfg.icon}
        </div>
        <div className="w-px flex-1 bg-slate-700/50 mt-1" />
      </div>

      {/* Content */}
      <div className="flex-1 pb-5">
        <div className="flex items-start justify-between gap-2">
          <div className="flex items-center gap-2 flex-wrap">
            <span className={`text-sm font-semibold ${cfg.accent}`}>{cfg.label}</span>
            {event.back_dated && (
              <span className="text-xs bg-amber-900/40 text-amber-400 border border-amber-800/50 px-1.5 py-0.5 rounded">
                backdated
              </span>
            )}
            {isNew && (
              <span className="text-xs bg-cyan-900/40 text-cyan-400 border border-cyan-800/50 px-1.5 py-0.5 rounded">
                new
              </span>
            )}
            <span className={`text-xs ${SOURCE_COLORS[event.source] ?? "text-slate-500"}`}>
              via {event.source}
            </span>
          </div>
          <span
            className="text-xs text-slate-500 flex-shrink-0 group-hover:text-slate-400 transition-colors"
            title={new Date(event.occurred_at).toLocaleString()}
          >
            {relativeTime(event.occurred_at)}
          </span>
        </div>

        <div className="text-slate-400 text-xs mt-0.5">
          {event.actor ? (
            <span className="text-slate-300">{actorName}</span>
          ) : (
            <span className="text-slate-500">System</span>
          )}
          {event.actor?.email && (
            <span className="text-slate-600"> · {event.actor.email}</span>
          )}
        </div>

        {expanded && <PayloadView payload={event.payload} />}
      </div>
    </div>
  );
}
