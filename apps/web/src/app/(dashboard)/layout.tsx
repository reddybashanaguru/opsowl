"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  FolderKanban,
  CheckSquare,
  FileText,
  Search,
  BarChart2,
  Users,
  ScrollText,
  Timer,
  Inbox,
  Webhook,
  BarChart3,
  LayoutTemplate,
  KeyRound,
  Shuffle,
  Archive,
  Settings,
  BookOpen,
  BellRing,
  UserSearch,
  Plug,
  Filter,
} from "lucide-react";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { RBACGate } from "@/components/auth/RBACGate";
import { useAuth } from "@/lib/hooks/useAuth";
import { useConfig } from "@/lib/hooks/useConfig";
import { NotificationBell } from "@/components/notifications/NotificationBell";

type NavItem = {
  label: string;
  href: string;
  icon: React.ElementType;
  exact?: boolean;
};

function NavLink({ item }: { item: NavItem }) {
  const pathname = usePathname();
  const isActive = item.exact ? pathname === item.href : pathname.startsWith(item.href);
  const Icon = item.icon;
  return (
    <Link
      href={item.href}
      className={`flex items-center gap-3 px-4 py-2 text-sm transition-colors border-l-2 ${
        isActive
          ? "text-cyan-400 bg-cyan-400/5 border-cyan-400"
          : "text-slate-400 border-transparent hover:text-slate-200 hover:bg-slate-700/40"
      }`}
    >
      <Icon size={16} className="flex-shrink-0" />
      {item.label}
    </Link>
  );
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user, logout } = useAuth();
  const cfg = useConfig();

  const nav: NavItem[] = [
    { label: "Dashboard",           href: "/dashboard",  icon: LayoutDashboard },
    { label: cfg.cases_label,        href: "/cases",      icon: FolderKanban },
    { label: cfg.tasks_label,        href: "/tasks",      icon: CheckSquare },
    { label: cfg.documents_label,    href: "/documents",  icon: FileText },
    { label: "Alert Queue",          href: "/alerts",     icon: BellRing },
    { label: "Subjects",             href: "/subjects",   icon: UserSearch },
    { label: "Search",               href: "/search",     icon: Search },
  ];

  const insightsNav: NavItem[] = [
    { label: "Workload",             href: "/workload",   icon: BarChart2 },
  ];

  const adminNav: NavItem[] = [
    { label: "Users",                href: "/users",                      icon: Users },
    { label: "Audit Logs",           href: "/audit",                      icon: ScrollText },
    { label: cfg.sla_label + " Policies", href: "/sla",                  icon: Timer },
    { label: "Inbound Events",       href: "/settings/inbound",           icon: Inbox },
    { label: "Webhooks",             href: "/settings/webhooks",          icon: Webhook },
    { label: "Scheduled Reports",    href: "/settings/reports",           icon: BarChart3 },
    { label: "Templates",            href: "/settings/templates",         icon: LayoutTemplate },
    { label: "Playbooks",            href: "/settings/playbooks",         icon: BookOpen },
    { label: "API Keys",             href: "/settings/api-keys",          icon: KeyRound },
    { label: "Alert Sources",         href: "/settings/alert-sources",     icon: Plug },
    { label: "Alert Rules",          href: "/settings/alert-rules",       icon: Filter },
    { label: "Auto-Assignment",      href: "/settings/assignment-rules",  icon: Shuffle },
    { label: "Data Retention",       href: "/settings/retention",         icon: Archive },
    { label: "Settings",             href: "/settings",                   icon: Settings, exact: true },
  ];

  const roleColor: Record<string, string> = {
    super_admin: "text-red-400",
    admin:       "text-amber-400",
    manager:     "text-violet-400",
    analyst:     "text-cyan-400",
    viewer:      "text-slate-400",
  };

  const roleLabel: Record<string, string> = {
    super_admin: "Super Admin",
    admin:       "Admin",
    manager:     "Manager",
    analyst:     "Analyst",
    viewer:      "Viewer",
  };

  return (
    <ProtectedRoute>
      <div className="h-screen bg-[#0f172a] flex overflow-hidden">
        {/* Sidebar */}
        <aside className="w-60 min-w-60 bg-[#1e293b] border-r border-slate-700 flex flex-col overflow-y-auto flex-shrink-0">
          {/* Brand */}
          <div className="px-5 py-5 border-b border-slate-700">
            <div className="flex items-center gap-2.5">
              <div className="w-7 h-7 bg-cyan-500/20 rounded-lg flex items-center justify-center border border-cyan-500/30 flex-shrink-0">
                <span className="text-cyan-400 text-sm font-bold leading-none">O</span>
              </div>
              <div>
                <div className="text-cyan-400 font-bold text-base tracking-tight leading-none">OpsOwl AI</div>
                <div className="text-slate-500 text-[10px] uppercase tracking-widest mt-0.5">
                  Operations Intelligence
                </div>
              </div>
            </div>
          </div>

          {/* Nav */}
          <nav className="flex-1 py-4 overflow-y-auto">
            <div className="px-4 mb-2 text-[10px] font-semibold text-slate-500 uppercase tracking-widest">
              Workspace
            </div>
            {nav.map((item) => <NavLink key={item.href} item={item} />)}

            <RBACGate allowedRoles={["manager", "admin", "super_admin"]}>
              <div className="px-4 mt-5 mb-2 text-[10px] font-semibold text-slate-500 uppercase tracking-widest">
                Intelligence
              </div>
              {insightsNav.map((item) => <NavLink key={item.href} item={item} />)}
            </RBACGate>

            <RBACGate allowedRoles={["admin", "super_admin"]}>
              <div className="px-4 mt-5 mb-2 text-[10px] font-semibold text-slate-500 uppercase tracking-widest">
                Administration
              </div>
              {adminNav.map((item) => <NavLink key={item.href} item={item} />)}
            </RBACGate>
          </nav>

          {/* User footer */}
          <div className="px-4 py-4 border-t border-slate-700">
            <div className="flex items-center justify-between mb-1">
              <div className="text-slate-200 text-sm font-medium truncate flex-1 min-w-0">{user?.full_name}</div>
              <NotificationBell />
            </div>
            <div className={`text-xs mt-0.5 ${roleColor[user?.role ?? "viewer"] ?? "text-slate-400"}`}>
              {roleLabel[user?.role ?? "viewer"] ?? user?.role}
            </div>
            <button
              onClick={() => logout()}
              className="mt-3 flex items-center gap-1.5 text-xs text-slate-500 hover:text-slate-300 transition-colors"
            >
              <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
              </svg>
              Sign out
            </button>
          </div>
        </aside>

        {/* Main content */}
        <main className="flex-1 overflow-y-auto">{children}</main>
      </div>
    </ProtectedRoute>
  );
}
