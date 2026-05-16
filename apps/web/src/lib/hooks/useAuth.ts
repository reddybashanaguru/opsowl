"use client";

import { useCallback, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useAuthStore, type UserRole } from "@/lib/stores/auth.store";

interface RefreshResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export function useAuth() {
  const { user, accessToken, setAuth, clearAuth, hasRole } = useAuthStore();
  const router = useRouter();
  const refreshTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const logout = useCallback(async () => {
    if (accessToken) {
      try {
        await api.post("/api/v1/auth/logout", {});
      } catch {
        // ignore — clear state regardless
      }
    }
    clearAuth();
    router.push("/login");
  }, [accessToken, clearAuth, router]);

  const refresh = useCallback(async (): Promise<boolean> => {
    try {
      const data = await api.post<RefreshResponse>("/api/v1/auth/refresh", {});
      if (user) {
        setAuth(user, data.access_token);
      }
      return true;
    } catch {
      clearAuth();
      router.push("/login");
      return false;
    }
  }, [user, setAuth, clearAuth, router]);

  // Auto-refresh 5 minutes before token expires (55 min into 60 min lifetime)
  useEffect(() => {
    if (!accessToken) return;

    if (refreshTimer.current) {
      clearTimeout(refreshTimer.current);
    }
    refreshTimer.current = setTimeout(() => {
      refresh();
    }, 55 * 60 * 1000);

    return () => {
      if (refreshTimer.current) clearTimeout(refreshTimer.current);
    };
  }, [accessToken, refresh]);

  return {
    user,
    accessToken,
    isAuthenticated: !!user && !!accessToken,
    logout,
    refresh,
    hasRole: (...roles: UserRole[]) => hasRole(...roles),
  };
}
