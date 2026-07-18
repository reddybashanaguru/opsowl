"use client";

import { Suspense, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { useAuthStore, type AuthUser } from "@/lib/stores/auth.store";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

const credSchema = z.object({
  email: z.string().email("Enter a valid email address"),
  password: z.string().min(1, "Password is required"),
});
const mfaSchema = z.object({
  code: z.string().min(6, "Enter your 6-digit code or 8-character backup code"),
});

type CredValues = z.infer<typeof credSchema>;
type MFAValues = z.infer<typeof mfaSchema>;

interface LoginResponse {
  access_token?: string;
  refresh_token?: string;
  expires_in?: number;
  user?: AuthUser;
  mfa_required?: boolean;
  mfa_token?: string;
}

function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const setAuth = useAuthStore((s) => s.setAuth);

  const [mfaToken, setMfaToken] = useState<string | null>(null);
  const [rateLimitMsg, setRateLimitMsg] = useState<string | null>(null);

  const credForm = useForm<CredValues>({
    resolver: zodResolver(credSchema),
    defaultValues: { email: "", password: "" },
  });
  const mfaForm = useForm<MFAValues>({
    resolver: zodResolver(mfaSchema),
    defaultValues: { code: "" },
  });

  const loginMutation = useMutation({
    mutationFn: (values: CredValues) =>
      api.post<LoginResponse>("/api/v1/auth/login", values),
    onSuccess: (data) => {
      if (data.mfa_required && data.mfa_token) {
        setMfaToken(data.mfa_token);
        setRateLimitMsg(null);
        return;
      }
      if (data.user) {
        setAuth(data.user);
        router.push(decodeURIComponent(searchParams.get("returnUrl") ?? "/dashboard"));
      }
    },
    onError: (err) => {
      setRateLimitMsg(
        err instanceof APIError && err.status === 429
          ? "Too many attempts. Please wait a moment and try again."
          : null
      );
    },
  });

  const mfaMutation = useMutation({
    mutationFn: (values: MFAValues) =>
      api.post<LoginResponse>("/api/v1/auth/mfa/challenge", {
        mfa_token: mfaToken,
        code: values.code,
      }),
    onSuccess: (data) => {
      if (data.user) {
        setAuth(data.user);
        router.push(decodeURIComponent(searchParams.get("returnUrl") ?? "/dashboard"));
      }
    },
  });

  const credError = (() => {
    if (rateLimitMsg) return rateLimitMsg;
    const err = loginMutation.error;
    if (!err) return null;
    if (err instanceof APIError) {
      if (err.status === 401) return "Invalid email or password.";
      if (err.status === 403) return "Your account has been deactivated. Contact your administrator.";
      return err.message;
    }
    return "Something went wrong. Please try again.";
  })();

  const mfaError = (() => {
    const err = mfaMutation.error;
    if (!err) return null;
    if (err instanceof APIError) {
      if (err.status === 401) return "Invalid or expired code. Try again or use a backup code.";
      return err.message;
    }
    return "Verification failed. Please try again.";
  })();

  return (
    <div className="min-h-screen bg-[#080e1a] flex items-center justify-center px-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-2 mb-2">
            <span className="text-2xl font-bold text-cyan-400 tracking-tight">OpsOwl AI</span>
          </div>
          <p className="text-slate-400 text-sm">AI-Native Fincrime Investigations Workspace</p>
        </div>

        <div className="bg-[#131f30] border border-[#1e3050] rounded-2xl ring-1 ring-white/[0.04] p-8 shadow-xl">
          {!mfaToken ? (
            <>
              <h1 className="text-white text-xl font-semibold mb-1">Sign in to your workspace</h1>
              <p className="text-slate-400 text-sm mb-6">Enter your credentials to access the platform.</p>

              <Form {...credForm}>
                <form onSubmit={credForm.handleSubmit((v) => loginMutation.mutate(v))} className="space-y-4">
                  <FormField
                    control={credForm.control}
                    name="email"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel className="text-slate-300 text-sm">Email address</FormLabel>
                        <FormControl>
                          <Input
                            type="email"
                            placeholder="analyst@yourorg.com"
                            autoComplete="email"
                            className="bg-[#080e1a] border-slate-600 text-white placeholder:text-slate-500 focus:border-cyan-500 focus:ring-cyan-500/20"
                            {...field}
                          />
                        </FormControl>
                        <FormMessage className="text-red-400 text-xs" />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={credForm.control}
                    name="password"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel className="text-slate-300 text-sm">Password</FormLabel>
                        <FormControl>
                          <Input
                            type="password"
                            placeholder="••••••••"
                            autoComplete="current-password"
                            className="bg-[#080e1a] border-slate-600 text-white placeholder:text-slate-500 focus:border-cyan-500 focus:ring-cyan-500/20"
                            {...field}
                          />
                        </FormControl>
                        <FormMessage className="text-red-400 text-xs" />
                      </FormItem>
                    )}
                  />
                  {credError && (
                    <div className="bg-red-500/10 border border-red-500/30 rounded-lg px-4 py-3 text-red-400 text-sm">
                      {credError}
                    </div>
                  )}
                  <Button
                    type="submit"
                    disabled={loginMutation.isPending}
                    className="w-full bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold transition-colors disabled:opacity-60"
                  >
                    {loginMutation.isPending ? "Signing in…" : "Sign in"}
                  </Button>
                </form>
              </Form>
            </>
          ) : (
            <>
              <div className="flex items-center gap-2 mb-1">
                <div className="w-8 h-8 rounded-full bg-cyan-500/20 flex items-center justify-center">
                  <svg className="w-4 h-4 text-cyan-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                  </svg>
                </div>
                <h1 className="text-white text-xl font-semibold">Two-factor verification</h1>
              </div>
              <p className="text-slate-400 text-sm mb-6">
                Enter the 6-digit code from your authenticator app, or an 8-character backup code.
              </p>

              <Form {...mfaForm}>
                <form onSubmit={mfaForm.handleSubmit((v) => mfaMutation.mutate(v))} className="space-y-4">
                  <FormField
                    control={mfaForm.control}
                    name="code"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel className="text-slate-300 text-sm">Verification code</FormLabel>
                        <FormControl>
                          <Input
                            type="text"
                            inputMode="numeric"
                            placeholder="000000"
                            autoComplete="one-time-code"
                            autoFocus
                            className="bg-[#080e1a] border-slate-600 text-white placeholder:text-slate-500 focus:border-cyan-500 focus:ring-cyan-500/20 text-center text-xl tracking-widest"
                            {...field}
                          />
                        </FormControl>
                        <FormMessage className="text-red-400 text-xs" />
                      </FormItem>
                    )}
                  />
                  {mfaError && (
                    <div className="bg-red-500/10 border border-red-500/30 rounded-lg px-4 py-3 text-red-400 text-sm">
                      {mfaError}
                    </div>
                  )}
                  <Button
                    type="submit"
                    disabled={mfaMutation.isPending}
                    className="w-full bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold transition-colors disabled:opacity-60"
                  >
                    {mfaMutation.isPending ? "Verifying…" : "Verify"}
                  </Button>
                  <button
                    type="button"
                    onClick={() => { setMfaToken(null); mfaForm.reset(); }}
                    className="w-full text-slate-400 text-sm hover:text-slate-300 transition-colors"
                  >
                    Back to sign in
                  </button>
                </form>
              </Form>
            </>
          )}
        </div>

        <p className="text-center text-slate-600 text-xs mt-6">
          OpsOwl AI · Secure Enterprise Platform · All access is audited
        </p>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={<div className="min-h-screen bg-[#080e1a]" />}>
      <LoginForm />
    </Suspense>
  );
}
