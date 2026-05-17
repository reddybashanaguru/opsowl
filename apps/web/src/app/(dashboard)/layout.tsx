"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { RBACGate } from "@/components/auth/RBACGate";
import { useAuth } from "@/lib/hooks/useAuth";
import { NotificationBell } from "@/components/notifications/NotificationBell";

const nav = [
  { label: "Dashboard", href: "/dashboard", icon: "▦" },
  { label: "Cases", href: "/cases", icon: "⬡" },
  { label: "Tasks", href: "/tasks", icon: "✓" },
  { label: "Documents", href: "/documents", icon: "◻" },
  { label: "Search", href: "/search", icon: "⌕" },
];

const adminNav = [
  { label: "Users", href: "/users", icon: "◎" },
  { label: "Audit Logs", href: "/audit", icon: "◈" },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { user, logout } = useAuth();

  const roleColor: Record<string, string> = {
    super_admin: "text-red-400",
    admin: "text-amber-400",
    manager: "text-violet-400",
    analyst: "text-cyan-400",
    viewer: "text-slate-400",
  };

  return (
    <ProtectedRoute>
      <div className="h-screen bg-[#0f172a] flex overflow-hidden">
        {/* Sidebar — fixed height, independent scroll */}
        <aside className="w-60 min-w-60 bg-[#1e293b] border-r border-slate-700 flex flex-col overflow-y-auto flex-shrink-0">
          {/* Brand */}
          <div className="px-5 py-5 border-b border-slate-700">
            <div className="text-cyan-400 font-bold text-lg tracking-tight">OpsOwl AI</div>
            <div className="text-slate-500 text-[10px] uppercase tracking-widest mt-0.5">
              Operations Intelligence
            </div>
          </div>

          {/* Nav */}
          <nav className="flex-1 py-4 overflow-y-auto">
            <div className="px-4 mb-2 text-[10px] font-semibold text-slate-500 uppercase tracking-widest">
              Workspace
            </div>
            {nav.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className={`flex items-center gap-3 px-4 py-2 text-sm transition-colors border-l-2 ${
                  pathname.startsWith(item.href)
                    ? "text-cyan-400 bg-cyan-400/5 border-cyan-400"
                    : "text-slate-400 border-transparent hover:text-slate-200 hover:bg-slate-700/40"
                }`}
              >
                <span className="text-base">{item.icon}</span>
                {item.label}
              </Link>
            ))}

            <RBACGate allowedRoles={["admin", "super_admin"]}>
              <div className="px-4 mt-4 mb-2 text-[10px] font-semibold text-slate-500 uppercase tracking-widest">
                Administration
              </div>
              {adminNav.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`flex items-center gap-3 px-4 py-2 text-sm transition-colors border-l-2 ${
                    pathname.startsWith(item.href)
                      ? "text-cyan-400 bg-cyan-400/5 border-cyan-400"
                      : "text-slate-400 border-transparent hover:text-slate-200 hover:bg-slate-700/40"
                  }`}
                >
                  <span className="text-base">{item.icon}</span>
                  {item.label}
                </Link>
              ))}
            </RBACGate>
          </nav>

          {/* User + Notification Bell */}
          <div className="px-4 py-4 border-t border-slate-700">
            <div className="flex items-center justify-between mb-1">
              <div className="text-slate-200 text-sm font-medium truncate flex-1 min-w-0">{user?.full_name}</div>
              <NotificationBell />
            </div>
            <div className={`text-xs mt-0.5 ${roleColor[user?.role ?? "viewer"] ?? "text-slate-400"}`}>
              {user?.role?.replace("_", " ")}
            </div>
            <button
              onClick={() => logout()}
              className="mt-3 text-xs text-slate-500 hover:text-slate-300 transition-colors"
            >
              Sign out →
            </button>
          </div>
        </aside>

        {/* Main */}
        <main className="flex-1 overflow-y-auto">{children}</main>
      </div>
    </ProtectedRoute>
  );
}
