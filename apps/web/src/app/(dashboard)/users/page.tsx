"use client";

import { useState } from "react";
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

interface ListUsersResponse {
  users: User[];
  next_cursor: string | null;
  has_more: boolean;
  total_count: number;
}

const ROLE_COLORS: Record<string, string> = {
  super_admin: "bg-red-500/10 text-red-400 border-red-500/30",
  admin: "bg-amber-500/10 text-amber-400 border-amber-500/30",
  manager: "bg-violet-500/10 text-violet-400 border-violet-500/30",
  analyst: "bg-cyan-500/10 text-cyan-400 border-cyan-500/30",
  viewer: "bg-slate-500/10 text-slate-400 border-slate-600",
};

export default function UsersPage() {
  const { user: currentUser } = useAuth();
  const queryClient = useQueryClient();
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ["users"],
    queryFn: () =>
      api.get<{ data: ListUsersResponse }>("/api/v1/users?limit=50").then(
        (r) => (r as unknown as { data: ListUsersResponse }).data ?? r
      ),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/users/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
      setDeleteConfirm(null);
    },
  });

  const users: User[] = (data as unknown as ListUsersResponse)?.users ?? [];
  const totalCount = (data as unknown as ListUsersResponse)?.total_count ?? 0;

  return (
    <RBACGate
      allowedRoles={["admin", "super_admin"]}
      fallback={
        <div className="p-8 text-slate-400">
          You don&apos;t have permission to view this page.
        </div>
      }
    >
      <div className="p-8">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-white">User Management</h1>
            <p className="text-slate-400 text-sm mt-1">
              {totalCount} {totalCount === 1 ? "user" : "users"} in your organisation
            </p>
          </div>
        </div>

        {isLoading && (
          <div className="space-y-3">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="bg-[#1e293b] border border-slate-700 rounded-xl h-16 animate-pulse" />
            ))}
          </div>
        )}

        {error && (
          <div className="bg-red-500/10 border border-red-500/30 rounded-xl p-4 text-red-400 text-sm">
            {error instanceof APIError ? error.message : "Failed to load users. Try again."}
          </div>
        )}

        {!isLoading && !error && users.length === 0 && (
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-12 text-center">
            <p className="text-slate-400 text-sm">No users found.</p>
          </div>
        )}

        {users.length > 0 && (
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="text-left px-5 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">
                    Name
                  </th>
                  <th className="text-left px-5 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">
                    Email
                  </th>
                  <th className="text-left px-5 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">
                    Role
                  </th>
                  <th className="text-left px-5 py-3 text-xs font-semibold text-slate-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-5 py-3" />
                </tr>
              </thead>
              <tbody>
                {users.map((u) => (
                  <tr key={u.id} className="border-b border-slate-700/50 hover:bg-slate-700/20 transition-colors">
                    <td className="px-5 py-4 text-white font-medium">{u.full_name}</td>
                    <td className="px-5 py-4 text-slate-400">{u.email}</td>
                    <td className="px-5 py-4">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-md text-xs font-semibold border ${
                          ROLE_COLORS[u.role] ?? "bg-slate-700 text-slate-300"
                        }`}
                      >
                        {u.role.replace("_", " ")}
                      </span>
                    </td>
                    <td className="px-5 py-4">
                      <span
                        className={`inline-flex items-center gap-1.5 text-xs font-medium ${
                          u.is_active ? "text-emerald-400" : "text-slate-500"
                        }`}
                      >
                        <span
                          className={`w-1.5 h-1.5 rounded-full ${
                            u.is_active ? "bg-emerald-400" : "bg-slate-500"
                          }`}
                        />
                        {u.is_active ? "Active" : "Inactive"}
                      </span>
                    </td>
                    <td className="px-5 py-4 text-right">
                      {u.id !== currentUser?.id && (
                        <>
                          {deleteConfirm === u.id ? (
                            <div className="flex items-center gap-2 justify-end">
                              <span className="text-xs text-slate-400">Confirm?</span>
                              <button
                                onClick={() => deleteMutation.mutate(u.id)}
                                className="text-xs text-red-400 hover:text-red-300 font-medium"
                              >
                                Yes, deactivate
                              </button>
                              <button
                                onClick={() => setDeleteConfirm(null)}
                                className="text-xs text-slate-500 hover:text-slate-300"
                              >
                                Cancel
                              </button>
                            </div>
                          ) : (
                            <button
                              onClick={() => setDeleteConfirm(u.id)}
                              className="text-xs text-slate-500 hover:text-red-400 transition-colors"
                            >
                              Deactivate
                            </button>
                          )}
                        </>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </RBACGate>
  );
}
