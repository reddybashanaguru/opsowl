"use client";

import { useCallback, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useAuthStore, type UserRole } from "@/lib/stores/auth.store";

export function useAuth() {
  const { user, setAuth, clearAuth, hasRole } = useAuthStore();
  const router = useRouter();
  const refreshTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const logout = useCallback(async () => {
    try {
      await api.post("/api/v1/auth/logout", {});
    } catch {
      // ignore — clear state regardless
    }
    clearAuth();
    router.push("/login");
  }, [clearAuth, router]);

  const refresh = useCallback(async (): Promise<boolean> => {
    try {
      // The server sets a new access_token httpOnly cookie in the response.
      // We only need the user info from the body.
      const data = await api.post<{ user?: { id: string; email: string; full_name: string; role: UserRole } }>(
        "/api/v1/auth/refresh",
        {},
      );
      if (data?.user && user) {
        // Refresh user state if server returned updated profile; otherwise keep current.
        setAuth(data.user as Parameters<typeof setAuth>[0]);
      }
      return true;
    } catch {
      clearAuth();
      router.push("/login");
      return false;
    }
  }, [user, setAuth, clearAuth, router]);

  // Auto-refresh 5 minutes before token expires (55 min into 60 min lifetime).
  // Without the token in JS we use a fixed 55-min interval; the server sets the
  // new access_token cookie automatically.
  useEffect(() => {
    if (!user) return;

    if (refreshTimer.current) {
      clearTimeout(refreshTimer.current);
    }
    refreshTimer.current = setTimeout(() => {
      refresh();
    }, 55 * 60 * 1000);

    return () => {
      if (refreshTimer.current) clearTimeout(refreshTimer.current);
    };
  }, [user, refresh]);

  return {
    user,
    isAuthenticated: !!user,
    logout,
    refresh,
    hasRole: (...roles: UserRole[]) => hasRole(...roles),
  };
}
