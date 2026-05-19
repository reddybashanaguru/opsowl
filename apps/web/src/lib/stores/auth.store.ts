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

export const useAuthStore = create<AuthState>((set, get) => ({
  user: (() => {
    if (typeof window === "undefined") return null;
    try {
      const stored = sessionStorage.getItem("opsowl_user");
      return stored ? (JSON.parse(stored) as AuthUser) : null;
    } catch {
      return null;
    }
  })(),
  accessToken: null,

  setAuth: (user, accessToken) => {
    if (typeof window !== "undefined") {
      sessionStorage.setItem("opsowl_user", JSON.stringify(user));
    }
    set({ user, accessToken });
  },

  clearAuth: () => {
    if (typeof window !== "undefined") {
      sessionStorage.removeItem("opsowl_user");
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
