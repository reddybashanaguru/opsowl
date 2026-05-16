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

const schema = z.object({
  email: z.string().email("Enter a valid email address"),
  password: z.string().min(1, "Password is required"),
});
type FormValues = z.infer<typeof schema>;

interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: AuthUser;
}

function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const setAuth = useAuthStore((s) => s.setAuth);
  const [rateLimitMsg, setRateLimitMsg] = useState<string | null>(null);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", password: "" },
  });

  const { mutate, isPending, error } = useMutation({
    mutationFn: (values: FormValues) =>
      api.post<LoginResponse>("/api/v1/auth/login", values),
    onSuccess: (data) => {
      setAuth(data.user, data.access_token);
      const returnUrl = searchParams.get("returnUrl") ?? "/dashboard";
      router.push(decodeURIComponent(returnUrl));
    },
    onError: (err) => {
      if (err instanceof APIError && err.status === 429) {
        setRateLimitMsg("Too many attempts. Please wait a moment and try again.");
      } else {
        setRateLimitMsg(null);
      }
    },
  });

  const errorMessage = (() => {
    if (rateLimitMsg) return rateLimitMsg;
    if (!error) return null;
    if (error instanceof APIError) {
      if (error.status === 401) return "Invalid email or password.";
      if (error.status === 403) return "Your account has been deactivated. Contact your administrator.";
      return error.message;
    }
    return "Something went wrong. Please try again.";
  })();

  return (
    <div className="min-h-screen bg-[#0f172a] flex items-center justify-center px-4">
      <div className="w-full max-w-md">
        {/* Logo */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-2 mb-2">
            <span className="text-2xl font-bold text-cyan-400 tracking-tight">OpsOwl AI</span>
          </div>
          <p className="text-slate-400 text-sm">Operations Intelligence Workspace</p>
        </div>

        {/* Card */}
        <div className="bg-[#1e293b] border border-slate-700 rounded-2xl p-8 shadow-xl">
          <h1 className="text-white text-xl font-semibold mb-1">Sign in to your workspace</h1>
          <p className="text-slate-400 text-sm mb-6">Enter your credentials to access the platform.</p>

          <Form {...form}>
            <form onSubmit={form.handleSubmit((v) => mutate(v))} className="space-y-4">
              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className="text-slate-300 text-sm">Email address</FormLabel>
                    <FormControl>
                      <Input
                        type="email"
                        placeholder="analyst@yourorg.com"
                        autoComplete="email"
                        className="bg-[#0f172a] border-slate-600 text-white placeholder:text-slate-500 focus:border-cyan-500 focus:ring-cyan-500/20"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage className="text-red-400 text-xs" />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className="text-slate-300 text-sm">Password</FormLabel>
                    <FormControl>
                      <Input
                        type="password"
                        placeholder="••••••••"
                        autoComplete="current-password"
                        className="bg-[#0f172a] border-slate-600 text-white placeholder:text-slate-500 focus:border-cyan-500 focus:ring-cyan-500/20"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage className="text-red-400 text-xs" />
                  </FormItem>
                )}
              />

              {errorMessage && (
                <div className="bg-red-500/10 border border-red-500/30 rounded-lg px-4 py-3 text-red-400 text-sm">
                  {errorMessage}
                </div>
              )}

              <Button
                type="submit"
                disabled={isPending}
                className="w-full bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold transition-colors disabled:opacity-60"
              >
                {isPending ? "Signing in…" : "Sign in"}
              </Button>
            </form>
          </Form>
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
    <Suspense fallback={<div className="min-h-screen bg-[#0f172a]" />}>
      <LoginForm />
    </Suspense>
  );
}
