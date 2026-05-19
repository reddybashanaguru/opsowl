"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import { api } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";

interface RefreshResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, accessToken, setAuth, clearAuth } = useAuthStore();
  const router = useRouter();
  const pathname = usePathname();
  // Always start as true so server and client initial renders both show the spinner.
  // After mount, the effect immediately clears it if a token is already in session.
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    if (accessToken) {
      setChecking(false);
      return;
    }

    if (!user) {
      router.replace(`/login?returnUrl=${encodeURIComponent(pathname)}`);
      return;
    }

    // user in sessionStorage but no in-memory token — attempt silent refresh via httpOnly cookie
    api
      .post<RefreshResponse>("/api/v1/auth/refresh", {})
      .then((data) => {
        setAuth(user, data.access_token);
        setChecking(false);
      })
      .catch(() => {
        clearAuth();
        router.replace(`/login?returnUrl=${encodeURIComponent(pathname)}`);
      });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const spinner = (
    <div className="min-h-screen bg-[#0f172a] flex items-center justify-center">
      <div className="w-8 h-8 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin" />
    </div>
  );

  if (checking) return spinner;
  if (!user || !accessToken) return spinner;

  return <>{children}</>;
}
