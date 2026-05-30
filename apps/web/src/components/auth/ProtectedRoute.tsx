"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import { api } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { useIdleTimeout } from "@/lib/hooks/useIdleTimeout";
import { IdleTimeoutWarning } from "@/components/auth/IdleTimeoutWarning";

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, setAuth, clearAuth } = useAuthStore();
  const router = useRouter();
  const pathname = usePathname();
  const [checking, setChecking] = useState(true);

  const handleIdleLogout = () => {
    clearAuth();
    router.replace(`/login?reason=idle&returnUrl=${encodeURIComponent(pathname)}`);
  };

  const { warningVisible, secondsLeft, extendSession } = useIdleTimeout({
    onTimeout: handleIdleLogout,
    enabled: !checking && !!user,
  });

  useEffect(() => {
    if (user) {
      // User already hydrated from sessionStorage — render immediately.
      // The access_token cookie (1h TTL) is still valid; the API client handles
      // 401 → refresh → retry for long-running sessions (see api.ts).
      setChecking(false);
      return;
    }

    // No user in sessionStorage (new tab, Playwright cookie-only restore).
    // Attempt to recover via the httpOnly refresh_token cookie before redirecting.
    api
      .post<{ user?: { id: string; email: string; full_name: string; role: string } }>(
        "/api/v1/auth/refresh",
        {},
      )
      .then((data) => {
        if (data?.user) {
          setAuth(data.user as Parameters<typeof setAuth>[0]);
          setChecking(false);
        } else {
          clearAuth();
          router.replace(`/login?returnUrl=${encodeURIComponent(pathname)}`);
        }
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
  if (!user) return spinner;

  return (
    <>
      {children}
      {warningVisible && (
        <IdleTimeoutWarning
          secondsLeft={secondsLeft}
          onStayLoggedIn={extendSession}
          onLogoutNow={handleIdleLogout}
        />
      )}
    </>
  );
}
