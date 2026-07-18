"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  LayoutDashboard,
  FolderKanban,
  CheckSquare,
  FileText,
  Search,
  BarChart2,
  BarChart,
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
  Library,
  ShieldCheck,
  ChevronRight,
} from "lucide-react";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { RBACGate } from "@/components/auth/RBACGate";
import { useAuth } from "@/lib/hooks/useAuth";
import { useConfig } from "@/lib/hooks/useConfig";
import { NotificationBell } from "@/components/notifications/NotificationBell";
import { useKeyboardShortcuts, useGoShortcuts } from "@/lib/hooks/useKeyboardShortcuts";
import { KeyboardShortcutsHelp } from "@/components/ui/KeyboardShortcutsHelp";

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
          ? "text-cyan-400 bg-cyan-400/8 border-cyan-400"
          : "text-slate-400 border-transparent hover:text-slate-200 hover:bg-[#1a2a40]"
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
  const router = useRouter();
  const pathname = usePathname();
  const [showHelp, setShowHelp] = useState(false);
  const [openGroups, setOpenGroups] = useState<Set<string>>(new Set());

  useKeyboardShortcuts({
    "?": () => setShowHelp(v => !v),
    "/": () => router.push("/search"),
  });

  useGoShortcuts({
    d: () => router.push("/dashboard"),
    c: () => router.push("/cases"),
    a: () => router.push("/alerts"),
    s: () => router.push("/search"),
  });

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
    { label: "MLRO Dashboard",       href: "/analytics",  icon: BarChart },
  ];

  const adminGroups: { heading: string; icon: React.ElementType; items: NavItem[] }[] = [
    {
      heading: "Access & Governance",
      icon: ShieldCheck,
      items: [
        { label: "Users",           href: "/users",               icon: Users },
        { label: "API Keys",        href: "/settings/api-keys",   icon: KeyRound },
        { label: "Audit Logs",      href: "/audit",               icon: ScrollText },
      ],
    },
    {
      heading: "Compliance",
      icon: Timer,
      items: [
        { label: cfg.sla_label + " Policies", href: "/sla",               icon: Timer },
        { label: "Data Retention",            href: "/settings/retention", icon: Archive },
        { label: "AI Governance",             href: "/settings/ai-act",    icon: ShieldCheck },
      ],
    },
    {
      heading: "Alert Pipeline",
      icon: BellRing,
      items: [
        { label: "Alert Sources",       href: "/settings/alert-sources",    icon: Plug },
        { label: "Alert Rules",         href: "/settings/alert-rules",      icon: Filter },
        { label: "Inbound Integrations",href: "/settings/inbound",          icon: Inbox },
        { label: "Auto-Assignment",     href: "/settings/assignment-rules", icon: Shuffle },
      ],
    },
    {
      heading: "Investigation Config",
      icon: Library,
      items: [
        { label: "Templates",       href: "/settings/templates",   icon: LayoutTemplate },
        { label: "Playbooks",       href: "/settings/playbooks",   icon: BookOpen },
        { label: "FATF Typologies", href: "/settings/typologies",  icon: Library },
      ],
    },
    {
      heading: "Reporting & Integrations",
      icon: BarChart3,
      items: [
        { label: "Scheduled Reports", href: "/settings/reports",   icon: BarChart3 },
        { label: "Webhooks",          href: "/settings/webhooks",  icon: Webhook },
      ],
    },
    {
      heading: "System",
      icon: Settings,
      items: [
        { label: "Settings", href: "/settings", icon: Settings, exact: true },
      ],
    },
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
      {showHelp && <KeyboardShortcutsHelp onClose={() => setShowHelp(false)} />}
      <div className="h-screen bg-[#080e1a] flex overflow-hidden">
        {/* Sidebar */}
        <aside className="w-60 min-w-60 bg-[#0b1525] border-r border-[#1e3050] flex flex-col overflow-y-auto flex-shrink-0">
          {/* Brand */}
          <div className="px-5 py-5 border-b border-[#1e3050]">
            <div className="flex items-center gap-2.5">
              <div className="w-7 h-7 bg-cyan-500/15 rounded-lg flex items-center justify-center border border-cyan-500/25 flex-shrink-0 shadow-[0_0_8px_rgba(34,211,238,0.15)]">
                <span className="text-cyan-400 text-sm font-bold leading-none">O</span>
              </div>
              <div>
                <div className="text-cyan-400 font-bold text-base tracking-tight leading-none">OpsOwl AI</div>
                <div className="text-slate-500 text-[10px] uppercase tracking-widest mt-0.5">
                  Fincrime Investigations & Operations
                </div>
              </div>
            </div>
          </div>

          {/* Nav */}
          <nav className="flex-1 py-4 overflow-y-auto">
            <div className="px-4 mb-2 text-[11px] font-bold text-slate-400 uppercase tracking-wider">
              Workspace
            </div>
            {nav.map((item) => <NavLink key={item.href} item={item} />)}

            <RBACGate allowedRoles={["manager", "admin", "super_admin"]}>
              <div className="px-4 mt-5 mb-2 text-[11px] font-bold text-slate-400 uppercase tracking-wider">
                Intelligence
              </div>
              {insightsNav.map((item) => <NavLink key={item.href} item={item} />)}
            </RBACGate>

            <RBACGate allowedRoles={["admin", "super_admin"]}>
              <div className="px-4 mt-5 mb-2 text-[11px] font-bold text-slate-400 uppercase tracking-wider">
                Administration
              </div>
              {adminGroups.map((group, gi) => {
                const hasActive = group.items.some(item =>
                  item.exact ? pathname === item.href : pathname.startsWith(item.href)
                );
                const isOpen = hasActive || openGroups.has(group.heading);
                const GroupIcon = group.icon;
                return (
                  <div key={group.heading} className={gi > 0 ? "border-t border-[#1e3050]/50 mt-0.5" : "mt-0.5"}>
                    <button
                      onClick={() => {
                        if (hasActive) return;
                        setOpenGroups(prev => {
                          const next = new Set(prev);
                          next.has(group.heading) ? next.delete(group.heading) : next.add(group.heading);
                          return next;
                        });
                      }}
                      className={`w-full flex items-center gap-2 px-3 py-2 text-left group/grp transition-colors rounded-sm ${
                        hasActive
                          ? "bg-cyan-500/10"
                          : "hover:bg-slate-700/25"
                      }`}
                    >
                      <GroupIcon
                        size={13}
                        className={`flex-shrink-0 transition-colors ${
                          hasActive ? "text-cyan-400" : "text-slate-600 group-hover/grp:text-slate-400"
                        }`}
                      />
                      <span className={`flex-1 text-[11px] font-semibold uppercase tracking-wide transition-colors ${
                        hasActive ? "text-cyan-400" : "text-slate-500 group-hover/grp:text-slate-300"
                      }`}>
                        {group.heading}
                      </span>
                      <span className={`text-[9px] font-medium px-1.5 py-0.5 rounded-full transition-colors ${
                        hasActive
                          ? "text-cyan-400/60 bg-cyan-500/10"
                          : "text-slate-700 bg-slate-800/60 group-hover/grp:text-slate-500"
                      }`}>
                        {group.items.length}
                      </span>
                      <ChevronRight
                        size={11}
                        className={`flex-shrink-0 transition-transform duration-200 ${isOpen ? "rotate-90" : ""} ${
                          hasActive ? "text-cyan-400/60" : "text-slate-700 group-hover/grp:text-slate-500"
                        }`}
                      />
                    </button>
                    {isOpen && (
                      <div className="ml-3 border-l border-[#1e3050]/70 mb-1">
                        {group.items.map((item) => <NavLink key={item.href} item={item} />)}
                      </div>
                    )}
                  </div>
                );
              })}
            </RBACGate>
          </nav>

          {/* User footer */}
          <div className="px-4 py-4 border-t border-[#1e3050]">
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
