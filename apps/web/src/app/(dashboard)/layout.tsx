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
  Scale,
  Zap,
  Sliders,
  Send,
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

type AdminGroup = {
  heading: string;
  icon: React.ElementType;
  items: NavItem[];
};

function NavLink({ item, compact }: { item: NavItem; compact?: boolean }) {
  const pathname = usePathname();
  const isActive = item.exact ? pathname === item.href : pathname.startsWith(item.href);
  const Icon = item.icon;
  return (
    <Link
      href={item.href}
      className={`flex items-center gap-3 px-4 py-2 text-sm transition-colors border-l-2 ${
        isActive
          ? "text-cyan-400 bg-cyan-400/8 border-cyan-400"
          : "text-slate-400 border-transparent hover:text-slate-200 hover:bg-white/[0.035]"
      }`}
    >
      <Icon size={15} className="flex-shrink-0" />
      <span className="truncate">{item.label}</span>
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
    { label: "Dashboard",        href: "/dashboard", icon: LayoutDashboard },
    { label: cfg.cases_label,    href: "/cases",     icon: FolderKanban },
    { label: cfg.tasks_label,    href: "/tasks",     icon: CheckSquare },
    { label: cfg.documents_label,href: "/documents", icon: FileText },
    { label: "Alert Queue",      href: "/alerts",    icon: BellRing },
    { label: "Subjects",         href: "/subjects",  icon: UserSearch },
    { label: "Search",           href: "/search",    icon: Search },
  ];

  const insightsNav: NavItem[] = [
    { label: "Workload",       href: "/workload",  icon: BarChart2 },
    { label: "Compliance",     href: "/analytics", icon: BarChart },
  ];

  const adminGroups: AdminGroup[] = [
    {
      heading: "Users & Access",
      icon: ShieldCheck,
      items: [
        { label: "Team Members", href: "/users",             icon: Users },
        { label: "API Keys",     href: "/settings/api-keys", icon: KeyRound },
      ],
    },
    {
      heading: "Compliance & Audit",
      icon: Scale,
      items: [
        { label: "Response SLAs",  href: "/sla",               icon: Timer },
        { label: "Data Retention", href: "/settings/retention", icon: Archive },
        { label: "AI Oversight",   href: "/settings/ai-act",    icon: ShieldCheck },
        { label: "Audit Trail",    href: "/audit",              icon: ScrollText },
      ],
    },
    {
      heading: "Alert Pipeline",
      icon: Zap,
      items: [
        { label: "Alert Sources",    href: "/settings/alert-sources", icon: Plug },
        { label: "Inbound Webhooks", href: "/settings/inbound",       icon: Inbox },
        { label: "Triage Rules",     href: "/settings/alert-rules",   icon: Filter },
      ],
    },
    {
      heading: "Investigation Workflows",
      icon: Sliders,
      items: [
        { label: "Case Routing",    href: "/settings/assignment-rules", icon: Shuffle },
        { label: "Case Templates",  href: "/settings/templates",        icon: LayoutTemplate },
        { label: "Playbooks",       href: "/settings/playbooks",        icon: BookOpen },
        { label: "Risk Typologies", href: "/settings/typologies",       icon: Library },
      ],
    },
    {
      heading: "Reports & Notifications",
      icon: Send,
      items: [
        { label: "Automated Reports",  href: "/settings/reports",  icon: BarChart3 },
        { label: "Outbound Webhooks",  href: "/settings/webhooks", icon: Webhook },
      ],
    },
  ];

  const roleColor: Record<string, string> = {
    super_admin: "text-red-400 bg-red-500/10",
    admin:       "text-amber-400 bg-amber-500/10",
    manager:     "text-violet-400 bg-violet-500/10",
    analyst:     "text-cyan-400 bg-cyan-500/10",
    viewer:      "text-slate-400 bg-slate-500/10",
  };

  const roleLabel: Record<string, string> = {
    super_admin: "Super Admin",
    admin:       "Admin",
    manager:     "Manager",
    analyst:     "Analyst",
    viewer:      "Viewer",
  };

  const initials = user?.full_name
    ?.split(" ")
    .slice(0, 2)
    .map(n => n[0])
    .join("")
    .toUpperCase() ?? "?";

  return (
    <ProtectedRoute>
      {showHelp && <KeyboardShortcutsHelp onClose={() => setShowHelp(false)} />}
      <div className="h-screen bg-[#080e1a] flex overflow-hidden">
        {/* Sidebar */}
        <aside className="w-64 min-w-64 bg-[#0b1525] border-r border-[#1e3050] flex flex-col overflow-y-auto flex-shrink-0">
          {/* Brand */}
          <div className="px-5 py-5 border-b border-[#1e3050]">
            <div className="flex items-center gap-2.5">
              <div className="w-7 h-7 bg-cyan-500/15 rounded-lg flex items-center justify-center border border-cyan-500/25 flex-shrink-0 shadow-[0_0_8px_rgba(34,211,238,0.15)]">
                <span className="text-cyan-400 text-sm font-bold leading-none">O</span>
              </div>
              <div>
                <div className="text-cyan-400 font-bold text-base tracking-tight leading-none">OpsOwl AI</div>
                <div className="text-slate-500 text-[10px] uppercase tracking-widest mt-0.5">
                  Fincrime Investigations
                </div>
              </div>
            </div>
          </div>

          {/* Nav */}
          <nav className="flex-1 py-4 overflow-y-auto">
            {/* Workspace */}
            <div className="px-4 mb-2 flex items-center gap-2">
              <span className="text-[10px] font-bold text-slate-500 uppercase tracking-widest">Workspace</span>
              <div className="flex-1 h-px bg-[#1e3050]/60" />
            </div>
            {nav.map((item) => <NavLink key={item.href} item={item} />)}

            {/* Analytics */}
            <RBACGate allowedRoles={["manager", "admin", "super_admin"]}>
              <div className="px-4 mt-5 mb-2 flex items-center gap-2">
                <span className="text-[10px] font-bold text-slate-500 uppercase tracking-widest">Analytics</span>
                <div className="flex-1 h-px bg-[#1e3050]/60" />
              </div>
              {insightsNav.map((item) => <NavLink key={item.href} item={item} />)}
            </RBACGate>

            {/* Administration */}
            <RBACGate allowedRoles={["admin", "super_admin"]}>
              <div className="px-4 mt-5 mb-2 flex items-center gap-2">
                <span className="text-[10px] font-bold text-slate-500 uppercase tracking-widest">Administration</span>
                <div className="flex-1 h-px bg-[#1e3050]/60" />
              </div>

              <div className="space-y-0.5">
                {adminGroups.map((group) => {
                  const hasActive = group.items.some(item =>
                    item.exact ? pathname === item.href : pathname.startsWith(item.href)
                  );
                  const isOpen = hasActive || openGroups.has(group.heading);
                  const GroupIcon = group.icon;

                  return (
                    <div key={group.heading}>
                      <button
                        onClick={() => {
                          if (hasActive) return;
                          setOpenGroups(prev => {
                            const next = new Set(prev);
                            next.has(group.heading) ? next.delete(group.heading) : next.add(group.heading);
                            return next;
                          });
                        }}
                        className={`w-full flex items-center gap-3 px-4 py-2 text-left group/grp transition-colors ${
                          hasActive ? "bg-white/[0.04]" : "hover:bg-white/[0.035]"
                        }`}
                      >
                        <GroupIcon
                          size={15}
                          className={`flex-shrink-0 transition-colors ${
                            hasActive ? "text-cyan-400" : "text-slate-400 group-hover/grp:text-slate-200"
                          }`}
                        />

                        <span className={`flex-1 truncate text-sm transition-colors ${
                          hasActive ? "text-cyan-400" : "text-slate-400 group-hover/grp:text-slate-200"
                        }`}>
                          {group.heading}
                        </span>

                        <ChevronRight
                          size={12}
                          className={`flex-shrink-0 transition-transform duration-200 ${isOpen ? "rotate-90" : ""} ${
                            hasActive ? "text-slate-400" : "text-slate-600 group-hover/grp:text-slate-400"
                          }`}
                        />
                      </button>

                      {isOpen && (
                        <div className="ml-3 border-l border-[#1e3050]/70 mb-0.5 mt-0.5">
                          {group.items.map((item) => <NavLink key={item.href} item={item} />)}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>

              {/* Settings — standalone at bottom of admin */}
              <div className="mt-2 pt-2 border-t border-[#1e3050]/50">
                <NavLink item={{ label: "Settings", href: "/settings", icon: Settings, exact: true }} />
              </div>
            </RBACGate>
          </nav>

          {/* User footer */}
          <div className="px-4 py-4 border-t border-[#1e3050]">
            <div className="flex items-center gap-3 mb-2">
              {/* Avatar */}
              <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-slate-700 to-slate-800 border border-slate-600/60 flex items-center justify-center flex-shrink-0">
                <span className="text-xs font-semibold text-slate-300">{initials}</span>
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-slate-200 text-sm font-medium truncate leading-tight">{user?.full_name}</div>
                <span className={`inline-block mt-0.5 text-[10px] font-medium px-1.5 py-0.5 rounded ${
                  roleColor[user?.role ?? "viewer"] ?? "text-slate-400 bg-slate-500/10"
                }`}>
                  {roleLabel[user?.role ?? "viewer"] ?? user?.role}
                </span>
              </div>
              <NotificationBell />
            </div>
            <button
              onClick={() => logout()}
              className="flex items-center gap-1.5 text-xs text-slate-600 hover:text-slate-400 transition-colors"
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
