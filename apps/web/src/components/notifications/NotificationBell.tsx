"use client";

import { useState, useEffect, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";

interface Notification {
  id: string;
  type: string;
  priority: string;
  title: string;
  message: string;
  link_url?: string;
  is_read: boolean;
  case_id?: string;
  task_id?: string;
  created_at: string;
}

const TYPE_ICON: Record<string, string> = {
  case_assigned: "📋",
  case_status_changed: "🔄",
  case_sla_warning: "⚠️",
  case_sla_breached: "🚨",
  task_assigned: "✓",
  task_due_soon: "⏰",
  task_overdue: "🔴",
  mention: "@",
  ai_review_required: "🤖",
  document_processed: "📄",
  comment_added: "💬",
};

const PRIORITY_COLOR: Record<string, string> = {
  critical: "text-red-400",
  high: "text-amber-400",
  medium: "text-cyan-400",
  low: "text-slate-400",
};

function NotifItem({ n, onClick }: { n: Notification; onClick: () => void }) {
  const timeAgo = (dateStr: string) => {
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return "just now";
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ago`;
    return `${Math.floor(hrs / 24)}d ago`;
  };

  return (
    <button
      onClick={onClick}
      className={`w-full text-left px-4 py-3 hover:bg-slate-700/40 transition-colors border-b border-slate-700/50 last:border-0 ${
        !n.is_read ? "bg-slate-700/20" : ""
      }`}
    >
      <div className="flex items-start gap-2.5">
        <span className="text-base flex-shrink-0 mt-0.5">{TYPE_ICON[n.type] ?? "🔔"}</span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5">
            <span className={`text-xs font-semibold ${PRIORITY_COLOR[n.priority] ?? "text-slate-400"}`}>
              {n.title}
            </span>
            {!n.is_read && (
              <span className="w-1.5 h-1.5 bg-cyan-400 rounded-full flex-shrink-0" />
            )}
          </div>
          <p className="text-slate-400 text-xs mt-0.5 leading-snug truncate">{n.message}</p>
          <p className="text-slate-600 text-[10px] mt-1">{timeAgo(n.created_at)}</p>
        </div>
      </div>
    </button>
  );
}

export function NotificationBell() {
  const router = useRouter();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const accessToken = useAuthStore((s) => s.accessToken);
  const [open, setOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const { data: countData } = useQuery({
    queryKey: ["notif-unread-count"],
    queryFn: () => api.get<{ unread_count: number }>("/api/v1/notifications/unread-count")
      .then((r) => r?.unread_count ?? 0),
    refetchInterval: 30000,
    enabled: !!user,
  });

  const { data: notifData, isLoading } = useQuery({
    queryKey: ["notifications"],
    queryFn: () => api.get<Notification[]>("/api/v1/notifications?limit=15").then((r) => r ?? []),
    enabled: open && !!user,
    refetchOnWindowFocus: true,
  });

  const markReadMutation = useMutation({
    mutationFn: (id: string) => api.put(`/api/v1/notifications/${id}/read`, {}),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["notifications"] });
      qc.invalidateQueries({ queryKey: ["notif-unread-count"] });
    },
  });

  const markAllMutation = useMutation({
    mutationFn: () => api.put("/api/v1/notifications/read-all", {}),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["notifications"] });
      qc.invalidateQueries({ queryKey: ["notif-unread-count"] });
    },
  });

  // SSE for real-time notifications
  useEffect(() => {
    if (!accessToken || !user) return;
    const url = `http://localhost:8080/api/v1/notifications/stream?token=${accessToken}`;
    const es = new EventSource(url);
    es.onmessage = () => {
      qc.invalidateQueries({ queryKey: ["notifications"] });
      qc.invalidateQueries({ queryKey: ["notif-unread-count"] });
    };
    return () => es.close();
  }, [accessToken, user, qc]);

  // Close on outside click
  useEffect(() => {
    function handler(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const unread = typeof countData === "number" ? countData : 0;
  const notifications = notifData ?? [];

  const handleNotifClick = (n: Notification) => {
    if (!n.is_read) markReadMutation.mutate(n.id);
    if (n.link_url) router.push(n.link_url);
    setOpen(false);
  };

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setOpen(!open)}
        className="relative w-8 h-8 flex items-center justify-center text-slate-400 hover:text-slate-200 transition-colors rounded-lg hover:bg-slate-700/40"
        title="Notifications"
      >
        <span className="text-base">🔔</span>
        {unread > 0 && (
          <span className="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 bg-cyan-500 text-slate-900 text-[9px] font-bold rounded-full flex items-center justify-center px-0.5">
            {unread > 99 ? "99+" : unread}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute bottom-full left-0 mb-2 w-80 bg-[#1e293b] border border-slate-700 rounded-xl shadow-2xl overflow-hidden z-50">
          {/* Header */}
          <div className="px-4 py-3 border-b border-slate-700 flex items-center justify-between">
            <span className="text-white text-sm font-semibold">Notifications</span>
            {unread > 0 && (
              <button
                onClick={() => markAllMutation.mutate()}
                className="text-xs text-slate-500 hover:text-cyan-400 transition-colors"
              >
                Mark all read
              </button>
            )}
          </div>

          {/* List */}
          <div className="max-h-80 overflow-y-auto">
            {isLoading ? (
              <div className="px-4 py-6 text-center text-slate-600 text-sm">Loading…</div>
            ) : notifications.length === 0 ? (
              <div className="px-4 py-8 text-center">
                <div className="text-2xl mb-2">🔔</div>
                <p className="text-slate-500 text-sm">You&apos;re all caught up!</p>
              </div>
            ) : (
              notifications.map((n) => (
                <NotifItem key={n.id} n={n} onClick={() => handleNotifClick(n)} />
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}
