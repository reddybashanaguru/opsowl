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
  setAuth: (user: AuthUser) => void;
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

function loadUser(): AuthUser | null {
  if (typeof window === "undefined") return null;
  try {
    const stored = sessionStorage.getItem("opsowl_user");
    return stored ? (JSON.parse(stored) as AuthUser) : null;
  } catch {
    return null;
  }
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: loadUser(),

  setAuth: (user) => {
    if (typeof window !== "undefined") {
      sessionStorage.setItem("opsowl_user", JSON.stringify(user));
    }
    set({ user });
  },

  clearAuth: () => {
    if (typeof window !== "undefined") {
      sessionStorage.removeItem("opsowl_user");
    }
    set({ user: null });
  },

  hasRole: (...roles) => {
    const user = get().user;
    if (!user) return false;
    const userLevel = ROLE_HIERARCHY[user.role] ?? 0;
    return roles.some((r) => ROLE_HIERARCHY[r] <= userLevel);
  },
}));
