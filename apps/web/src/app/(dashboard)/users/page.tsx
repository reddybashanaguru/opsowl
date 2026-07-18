"use client";

import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { RBACGate } from "@/components/auth/RBACGate";
import { useAuth } from "@/lib/hooks/useAuth";

interface User {
  id: string;
  email: string;
  full_name: string;
  role: string;
  is_active: boolean;
  created_at: string;
}

const ROLE_COLORS: Record<string, { badge: string; dot: string }> = {
  super_admin: { badge: "bg-red-500/10 text-red-400 border-red-500/30",    dot: "bg-red-400" },
  admin:       { badge: "bg-amber-500/10 text-amber-400 border-amber-500/30", dot: "bg-amber-400" },
  manager:     { badge: "bg-violet-500/10 text-violet-400 border-violet-500/30", dot: "bg-violet-400" },
  analyst:     { badge: "bg-cyan-500/10 text-cyan-400 border-cyan-500/30",   dot: "bg-cyan-400" },
  viewer:      { badge: "bg-slate-500/10 text-slate-400 border-slate-600",   dot: "bg-slate-500" },
};

const ROLES = ["", "super_admin", "admin", "manager", "analyst", "viewer"];
const ASSIGNABLE_ROLES = ["super_admin", "admin", "manager", "analyst", "viewer"] as const;
const ROLE_LABELS: Record<string, string> = {
  "": "All Roles", super_admin: "Super Admin", admin: "Admin",
  manager: "Manager", analyst: "Analyst", viewer: "Viewer",
};

function Avatar({ name }: { name: string }) {
  const initials = name.split(" ").map((w) => w[0]).slice(0, 2).join("").toUpperCase();
  const colors = ["bg-cyan-700", "bg-violet-700", "bg-amber-700", "bg-rose-700", "bg-emerald-700"];
  const color = colors[name.charCodeAt(0) % colors.length];
  return (
    <span className={`w-8 h-8 rounded-full ${color} flex items-center justify-center text-xs font-bold text-white flex-shrink-0`}>
      {initials}
    </span>
  );
}

// ─── Invite User Modal ────────────────────────────────────────────────────────

interface InviteModalProps {
  onClose: () => void;
  onSuccess: () => void;
}

function InviteModal({ onClose, onSuccess }: InviteModalProps) {
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<string>("analyst");
  const [password, setPassword] = useState("");
  const [showPwd, setShowPwd] = useState(false);
  const [error, setError] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      api.post("/api/v1/users", { email, full_name: fullName, role, password }),
    onSuccess: () => {
      onSuccess();
      onClose();
    },
    onError: (err) => {
      setError(err instanceof APIError ? err.message : "Failed to create user");
    },
  });

  const canSubmit = fullName.trim() && email.trim() && password.length >= 8;

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-slate-100 font-semibold text-lg">Invite User</h2>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300 text-lg leading-none">✕</button>
        </div>

        {error && (
          <div className="bg-red-500/10 border border-red-500/30 text-red-400 text-sm rounded-lg px-3 py-2 mb-4">
            {error}
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="block text-xs text-slate-400 mb-1">Full Name *</label>
            <input
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              placeholder="Priya Sharma"
              autoFocus
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:border-cyan-500 transition-colors"
            />
          </div>

          <div>
            <label className="block text-xs text-slate-400 mb-1">Email Address *</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="analyst@yourorg.com"
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:border-cyan-500 transition-colors"
            />
          </div>

          <div>
            <label className="block text-xs text-slate-400 mb-1">Role *</label>
            <select
              value={role}
              onChange={(e) => setRole(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-sm text-slate-100 focus:outline-none focus:border-cyan-500 transition-colors"
            >
              {ASSIGNABLE_ROLES.map((r) => (
                <option key={r} value={r}>{ROLE_LABELS[r]}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs text-slate-400 mb-1">Temporary Password * (min 8 chars)</label>
            <div className="relative">
              <input
                type={showPwd ? "text" : "password"}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 pr-10 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:border-cyan-500 transition-colors"
              />
              <button
                type="button"
                onClick={() => setShowPwd((v) => !v)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 text-xs"
              >
                {showPwd ? "Hide" : "Show"}
              </button>
            </div>
            <p className="text-slate-600 text-[10px] mt-1">The user should change this on first login.</p>
          </div>
        </div>

        <div className="flex gap-3 mt-6 pt-4 border-t border-slate-700/60">
          <button
            onClick={() => mutation.mutate()}
            disabled={!canSubmit || mutation.isPending}
            className="px-5 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition-colors"
          >
            {mutation.isPending ? "Creating…" : "Create User"}
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Inline Role Selector ─────────────────────────────────────────────────────

interface RoleSelectorProps {
  userId: string;
  currentRole: string;
  onSuccess: () => void;
  isSuperAdmin: boolean;
}

function RoleSelector({ userId, currentRole, onSuccess, isSuperAdmin }: RoleSelectorProps) {
  const [editing, setEditing] = useState(false);
  const [selected, setSelected] = useState(currentRole);
  const [error, setError] = useState("");

  const mutation = useMutation({
    mutationFn: (role: string) =>
      api.put(`/api/v1/users/${userId}`, { role }),
    onSuccess: () => {
      setEditing(false);
      setError("");
      onSuccess();
    },
    onError: (err) => {
      setError(err instanceof APIError ? err.message : "Failed to update role");
    },
  });

  const cfg = ROLE_COLORS[currentRole] ?? { badge: "bg-slate-700 text-slate-300 border-slate-600", dot: "bg-slate-500" };

  if (!editing) {
    return (
      <div className="flex items-center gap-2 group/role">
        <span className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-md text-xs font-semibold border ${cfg.badge}`}>
          <span className={`w-1.5 h-1.5 rounded-full ${cfg.dot}`} />
          {ROLE_LABELS[currentRole] ?? currentRole}
        </span>
        <button
          onClick={() => { setSelected(currentRole); setEditing(true); }}
          className="opacity-0 group-hover/role:opacity-100 text-slate-600 hover:text-slate-300 text-[10px] transition-all"
          title="Change role"
        >
          ✎
        </button>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2">
      <select
        value={selected}
        onChange={(e) => setSelected(e.target.value)}
        autoFocus
        className="bg-slate-900 border border-cyan-500 text-slate-100 text-xs rounded px-2 py-1 focus:outline-none"
      >
        {ASSIGNABLE_ROLES.filter((r) => isSuperAdmin || r !== "super_admin").map((r) => (
          <option key={r} value={r}>{ROLE_LABELS[r]}</option>
        ))}
      </select>
      <button
        onClick={() => mutation.mutate(selected)}
        disabled={selected === currentRole || mutation.isPending}
        className="text-xs text-cyan-400 hover:text-cyan-300 font-medium disabled:opacity-40 transition-colors"
      >
        {mutation.isPending ? "…" : "Save"}
      </button>
      <button
        onClick={() => { setEditing(false); setError(""); }}
        className="text-xs text-slate-500 hover:text-slate-300 transition-colors"
      >
        ✕
      </button>
      {error && <span className="text-red-400 text-[10px]">{error}</span>}
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function UsersPage() {
  const { user: currentUser } = useAuth();
  const queryClient = useQueryClient();
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [roleFilter, setRoleFilter] = useState("");
  const [search, setSearch] = useState("");
  const [showInvite, setShowInvite] = useState(false);

  const { data, isLoading, error } = useQuery({
    queryKey: ["users"],
    queryFn: () => api.getList<User>("/api/v1/users?limit=200"),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/users/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
      setDeleteConfirm(null);
    },
  });

  const allUsers: User[] = data?.data ?? [];

  const filtered = useMemo(() => {
    return allUsers.filter((u) => {
      if (roleFilter && u.role !== roleFilter) return false;
      if (search) {
        const q = search.toLowerCase();
        return u.full_name.toLowerCase().includes(q) || u.email.toLowerCase().includes(q);
      }
      return true;
    });
  }, [allUsers, roleFilter, search]);

  // Role distribution for the summary bar
  const roleCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    allUsers.forEach((u) => { counts[u.role] = (counts[u.role] ?? 0) + 1; });
    return counts;
  }, [allUsers]);

  const canManage = ["admin", "super_admin"].includes(currentUser?.role ?? "");

  return (
    <RBACGate
      allowedRoles={["admin", "super_admin"]}
      fallback={
        <div className="p-8 text-slate-400">You don&apos;t have permission to view this page.</div>
      }
    >
      <div className="p-6">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-white">User Management</h1>
            <p className="text-slate-400 text-sm mt-1">
              {filtered.length} of {allUsers.length} user{allUsers.length !== 1 ? "s" : ""}
              {(roleFilter || search) && " (filtered)"}
            </p>
          </div>
          {canManage && (
            <button
              onClick={() => setShowInvite(true)}
              className="flex items-center gap-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
            >
              <span className="text-base leading-none">+</span> Invite User
            </button>
          )}
        </div>

        {/* Role distribution strip */}
        {allUsers.length > 0 && !isLoading && (
          <div className="flex gap-2 flex-wrap mb-5">
            {Object.entries(roleCounts).map(([role, count]) => {
              const cfg = ROLE_COLORS[role];
              return (
                <button
                  key={role}
                  onClick={() => setRoleFilter(roleFilter === role ? "" : role)}
                  className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium border transition-colors ${
                    roleFilter === role ? cfg.badge + " ring-1 ring-current" : "bg-slate-800 text-slate-400 border-slate-700 hover:border-slate-500"
                  }`}
                >
                  <span className={`w-1.5 h-1.5 rounded-full ${cfg?.dot}`} />
                  {ROLE_LABELS[role] ?? role} · {count}
                </button>
              );
            })}
          </div>
        )}

        {/* Search */}
        <div className="flex gap-2 mb-5">
          <div className="relative">
            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500 text-xs">⌕</span>
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search name or email…"
              className="bg-[#131f30] border border-slate-600 text-slate-300 text-sm rounded-lg pl-7 pr-3 py-2 w-60 focus:outline-none focus:border-cyan-500 placeholder:text-slate-600"
            />
            {search && (
              <button onClick={() => setSearch("")} className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 text-xs">✕</button>
            )}
          </div>
        </div>

        {isLoading && (
          <div className="space-y-2">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] h-16 animate-pulse" />
            ))}
          </div>
        )}

        {error && (
          <div className="bg-red-500/10 border border-red-500/30 rounded-xl p-4 text-red-400 text-sm">
            {error instanceof APIError ? error.message : "Failed to load users. Try again."}
          </div>
        )}

        {!isLoading && !error && filtered.length === 0 && (
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-12 text-center">
            <p className="text-slate-400 text-sm">No users match your filters.</p>
          </div>
        )}

        {filtered.length > 0 && (
          <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="text-left px-5 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">User</th>
                  <th className="text-left px-5 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">Role</th>
                  <th className="text-left px-5 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">Status</th>
                  <th className="text-left px-5 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">Joined</th>
                  <th className="px-5 py-3" />
                </tr>
              </thead>
              <tbody>
                {filtered.map((u) => {
                  return (
                    <tr key={u.id} className="border-b border-slate-700/40 hover:bg-slate-700/15 transition-colors">
                      <td className="px-5 py-3.5">
                        <div className="flex items-center gap-3">
                          <Avatar name={u.full_name} />
                          <div>
                            <div className="text-white font-medium text-sm">{u.full_name}</div>
                            <div className="text-slate-500 text-xs">{u.email}</div>
                          </div>
                        </div>
                      </td>
                      <td className="px-5 py-3.5">
                        {u.id !== currentUser?.id && canManage && (currentUser?.role === "super_admin" || u.role !== "super_admin") ? (
                          <RoleSelector
                            userId={u.id}
                            currentRole={u.role}
                            isSuperAdmin={currentUser?.role === "super_admin"}
                            onSuccess={() => queryClient.invalidateQueries({ queryKey: ["users"] })}
                          />
                        ) : (
                          <span className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-md text-xs font-semibold border ${(ROLE_COLORS[u.role] ?? { badge: "bg-slate-700 text-slate-300 border-slate-600", dot: "bg-slate-500" }).badge}`}>
                            <span className={`w-1.5 h-1.5 rounded-full ${(ROLE_COLORS[u.role] ?? { badge: "", dot: "bg-slate-500" }).dot}`} />
                            {ROLE_LABELS[u.role] ?? u.role}
                          </span>
                        )}
                      </td>
                      <td className="px-5 py-3.5">
                        <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${u.is_active ? "text-emerald-400" : "text-slate-500"}`}>
                          <span className={`w-1.5 h-1.5 rounded-full ${u.is_active ? "bg-emerald-400" : "bg-slate-500"}`} />
                          {u.is_active ? "Active" : "Inactive"}
                        </span>
                      </td>
                      <td className="px-5 py-3.5 text-slate-500 text-xs tabular-nums">
                        {new Date(u.created_at).toLocaleDateString(undefined, { day: "2-digit", month: "short", year: "numeric" })}
                      </td>
                      <td className="px-5 py-3.5 text-right">
                        {u.id !== currentUser?.id && (currentUser?.role === "super_admin" || u.role !== "super_admin") && (
                          deleteConfirm === u.id ? (
                            <div className="flex items-center gap-2 justify-end">
                              <span className="text-xs text-slate-400">Deactivate?</span>
                              <button onClick={() => deleteMutation.mutate(u.id)} className="text-xs text-red-400 hover:text-red-300 font-medium">Yes</button>
                              <button onClick={() => setDeleteConfirm(null)} className="text-xs text-slate-500 hover:text-slate-300">No</button>
                            </div>
                          ) : (
                            <button onClick={() => setDeleteConfirm(u.id)} className="text-xs text-slate-600 hover:text-red-400 transition-colors">
                              Deactivate
                            </button>
                          )
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {showInvite && (
        <InviteModal
          onClose={() => setShowInvite(false)}
          onSuccess={() => queryClient.invalidateQueries({ queryKey: ["users"] })}
        />
      )}
    </RBACGate>
  );
}
