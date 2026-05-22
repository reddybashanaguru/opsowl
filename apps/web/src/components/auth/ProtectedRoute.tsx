"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import { api } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, setAuth, clearAuth } = useAuthStore();
  const router = useRouter();
  const pathname = usePathname();
  // Always verify session on mount — the access token is httpOnly so we
  // can't read it in JS. We use the presence of the user in sessionStorage
  // as a hint, but we still need a server round-trip if the token may have
  // expired (e.g., after a page refresh that ran past the 55-min auto-refresh).
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    if (!user) {
      router.replace(`/login?returnUrl=${encodeURIComponent(pathname)}`);
      return;
    }

    // Attempt silent refresh via the httpOnly refresh_token cookie.
    // If successful the server issues a new access_token cookie automatically.
    api
      .post<{ user?: { id: string; email: string; full_name: string; role: string } }>(
        "/api/v1/auth/refresh",
        {},
      )
      .then((data) => {
        if (data?.user) {
          // Update user profile in case it changed server-side.
          setAuth(data.user as Parameters<typeof setAuth>[0]);
        }
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
  if (!user) return spinner;

  return <>{children}</>;
}
