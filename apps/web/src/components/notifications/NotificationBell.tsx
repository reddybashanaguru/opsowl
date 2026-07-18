"use client";

import { useState, useEffect, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { Bell, CheckCheck, AlertTriangle, AlertCircle, Clock, FileText, Bot, MessageSquare, RefreshCw, CheckSquare } from "lucide-react";

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

const TYPE_ICON: Record<string, React.ElementType> = {
  case_assigned:        FileText,
  case_status_changed:  RefreshCw,
  case_sla_warning:     Clock,
  case_sla_breached:    AlertTriangle,
  task_assigned:        CheckSquare,
  task_due_soon:        Clock,
  task_overdue:         AlertCircle,
  mention:              MessageSquare,
  ai_review_required:   Bot,
  document_processed:   FileText,
  comment_added:        MessageSquare,
};

const PRIORITY_RING: Record<string, string> = {
  critical: "text-red-400 bg-red-500/10",
  high:     "text-amber-400 bg-amber-500/10",
  medium:   "text-cyan-400 bg-cyan-500/10",
  low:      "text-slate-400 bg-slate-500/10",
};

function timeAgo(dateStr: string) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

function NotifItem({ n, onClick }: { n: Notification; onClick: () => void }) {
  const Icon = TYPE_ICON[n.type] ?? Bell;
  const iconClass = PRIORITY_RING[n.priority] ?? "text-slate-400 bg-slate-500/10";

  return (
    <button
      onClick={onClick}
      className={`w-full text-left px-4 py-3 hover:bg-[#1a2a40] transition-colors border-b border-slate-700/40 last:border-0 ${
        !n.is_read ? "bg-cyan-500/[0.04]" : ""
      }`}
    >
      <div className="flex items-start gap-3">
        <div className={`w-7 h-7 rounded-lg flex items-center justify-center flex-shrink-0 mt-0.5 ${iconClass}`}>
          <Icon size={13} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2">
            <span className="text-xs font-semibold text-slate-200 truncate">{n.title}</span>
            <span className="text-[10px] text-slate-600 flex-shrink-0">{timeAgo(n.created_at)}</span>
          </div>
          <p className="text-slate-500 text-xs mt-0.5 leading-relaxed line-clamp-2">{n.message}</p>
        </div>
        {!n.is_read && (
          <span className="w-1.5 h-1.5 bg-cyan-400 rounded-full flex-shrink-0 mt-1.5" />
        )}
      </div>
    </button>
  );
}

export function NotificationBell() {
  const router = useRouter();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const [open, setOpen] = useState(false);
  const [dropPos, setDropPos] = useState({ bottom: 0, left: 0 });
  const bellRef = useRef<HTMLButtonElement>(null);
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
    queryFn: () => api.get<Notification[]>("/api/v1/notifications?limit=20").then((r) => r ?? []),
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

  useEffect(() => {
    function handler(e: MouseEvent) {
      if (
        dropdownRef.current && !dropdownRef.current.contains(e.target as Node) &&
        bellRef.current && !bellRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const handleBellClick = () => {
    if (!open && bellRef.current) {
      const rect = bellRef.current.getBoundingClientRect();
      // Use CSS `bottom` so the panel's bottom edge sits just above the bell
      // regardless of how many notifications are shown (panel grows upward)
      const bottom = window.innerHeight - rect.top + 6;
      const left = Math.min(rect.right + 10, window.innerWidth - 348);
      setDropPos({ bottom, left });
    }
    setOpen((v) => !v);
  };

  const unread = typeof countData === "number" ? countData : 0;
  const notifications = notifData ?? [];

  const handleNotifClick = (n: Notification) => {
    if (!n.is_read) markReadMutation.mutate(n.id);
    if (n.link_url) router.push(n.link_url);
    setOpen(false);
  };

  return (
    <>
      <button
        ref={bellRef}
        onClick={handleBellClick}
        className="relative w-8 h-8 flex items-center justify-center text-slate-400 hover:text-slate-200 transition-colors rounded-lg hover:bg-slate-700/40"
        title="Notifications"
      >
        <Bell size={15} />
        {unread > 0 && (
          <span className="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 bg-cyan-500 text-slate-900 text-[9px] font-bold rounded-full flex items-center justify-center px-0.5 leading-none">
            {unread > 99 ? "99+" : unread}
          </span>
        )}
      </button>

      {open && (
        <div
          ref={dropdownRef}
          style={{ bottom: dropPos.bottom, left: dropPos.left }}
          className="fixed w-[340px] bg-[#0d1b2e] border border-[#1e3050] rounded-xl shadow-[0_8px_32px_rgba(0,0,0,0.5)] overflow-hidden z-[9999] flex flex-col"
        >
          {/* Header */}
          <div className="flex items-center justify-between px-4 py-3 border-b border-slate-700/60 bg-[#0b1525]">
            <div className="flex items-center gap-2">
              <Bell size={14} className="text-cyan-400" />
              <span className="text-sm font-semibold text-white">Notifications</span>
              {unread > 0 && (
                <span className="px-1.5 py-0.5 bg-cyan-500/20 text-cyan-400 text-[10px] font-bold rounded-full border border-cyan-500/30">
                  {unread} new
                </span>
              )}
            </div>
            {unread > 0 && (
              <button
                onClick={() => markAllMutation.mutate()}
                disabled={markAllMutation.isPending}
                className="flex items-center gap-1 text-[11px] text-slate-500 hover:text-cyan-400 transition-colors disabled:opacity-50"
              >
                <CheckCheck size={11} />
                Mark all read
              </button>
            )}
          </div>

          {/* List */}
          <div className="overflow-y-auto" style={{ maxHeight: 380 }}>
            {isLoading ? (
              <div className="space-y-1 p-3">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="h-14 bg-slate-700/30 rounded-lg animate-pulse" />
                ))}
              </div>
            ) : notifications.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-10 px-4 gap-3">
                <div className="w-10 h-10 rounded-full bg-slate-700/40 flex items-center justify-center">
                  <Bell size={18} className="text-slate-600" />
                </div>
                <div className="text-center">
                  <p className="text-slate-300 text-sm font-medium">All caught up</p>
                  <p className="text-slate-600 text-xs mt-0.5">No new notifications</p>
                </div>
              </div>
            ) : (
              notifications.map((n) => (
                <NotifItem key={n.id} n={n} onClick={() => handleNotifClick(n)} />
              ))
            )}
          </div>

          {/* Footer */}
          {notifications.length > 0 && (
            <div className="px-4 py-2.5 border-t border-slate-700/60 bg-[#0b1525]">
              <p className="text-[10px] text-slate-600 text-center">
                Showing last {notifications.length} notification{notifications.length !== 1 ? "s" : ""}
              </p>
            </div>
          )}
        </div>
      )}
    </>
  );
}
