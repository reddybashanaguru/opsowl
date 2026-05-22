"use client";

import { create } from "zustand";

export type UserRole = "super_admin" | "admin" | "manager" | "analyst" | "viewer";

export interface AuthUser {
  id: string;
  email: string;
  full_name: string;
  role: UserRole;
  last_login_at?: string;
}

interface AuthState {
  user: AuthUser | null;
  accessToken: string | null;
  setAuth: (user: AuthUser, accessToken: string) => void;
  clearAuth: () => void;
  hasRole: (...roles: UserRole[]) => boolean;
}

const ROLE_HIERARCHY: Record<UserRole, number> = {
  super_admin: 5,
  admin: 4,
  manager: 3,
  analyst: 2,
  viewer: 1,
};

// Decode JWT exp claim without verifying the signature (client-side only).
function decodeJwtExp(token: string): number | null {
  try {
    const payload = JSON.parse(atob(token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/")));
    return typeof payload.exp === "number" ? payload.exp : null;
  } catch {
    return null;
  }
}

function loadUser(): AuthUser | null {
  if (typeof window === "undefined") return null;
  try {
    const stored = sessionStorage.getItem("opsowl_user");
    return stored ? (JSON.parse(stored) as AuthUser) : null;
  } catch {
    return null;
  }
}

// Returns the stored access token only if it has more than 30 seconds remaining.
function loadValidToken(): string | null {
  if (typeof window === "undefined") return null;
  try {
    const token = sessionStorage.getItem("opsowl_at");
    if (!token) return null;
    const exp = decodeJwtExp(token);
    if (!exp || exp * 1000 < Date.now() + 30_000) {
      sessionStorage.removeItem("opsowl_at");
      return null;
    }
    return token;
  } catch {
    return null;
  }
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: loadUser(),
  accessToken: loadValidToken(),

  setAuth: (user, accessToken) => {
    if (typeof window !== "undefined") {
      sessionStorage.setItem("opsowl_user", JSON.stringify(user));
      sessionStorage.setItem("opsowl_at", accessToken);
    }
    set({ user, accessToken });
  },

  clearAuth: () => {
    if (typeof window !== "undefined") {
      sessionStorage.removeItem("opsowl_user");
      sessionStorage.removeItem("opsowl_at");
    }
    set({ user: null, accessToken: null });
  },

  hasRole: (...roles) => {
    const user = get().user;
    if (!user) return false;
    const userLevel = ROLE_HIERARCHY[user.role] ?? 0;
    return roles.some((r) => ROLE_HIERARCHY[r] <= userLevel);
  },
}));
