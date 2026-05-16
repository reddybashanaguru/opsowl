"use client";

import { useAuthStore, type UserRole } from "@/lib/stores/auth.store";

interface RBACGateProps {
  allowedRoles: UserRole[];
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

export function RBACGate({ allowedRoles, children, fallback = null }: RBACGateProps) {
  const { hasRole } = useAuthStore();
  if (!hasRole(...allowedRoles)) return <>{fallback}</>;
  return <>{children}</>;
}
