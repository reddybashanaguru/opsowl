"use client";

import { useRouter } from "next/navigation";
import { APIError } from "@/lib/api";

interface PageErrorProps {
  error?: Error | null;
  title?: string;
  onRetry?: () => void;
}

export function PageError({ error, title, onRetry }: PageErrorProps) {
  const router = useRouter();

  const isNotFound = error instanceof APIError && error.status === 404;
  const heading = title ?? (isNotFound ? "Not Found" : "Something went wrong");
  const message = error instanceof APIError
    ? error.message
    : (error?.message ?? "An unexpected error occurred. Please try again.");

  return (
    <div className="flex flex-col items-center justify-center py-24 text-center px-6">
      <div className="w-14 h-14 rounded-full bg-red-500/10 border border-red-500/30 flex items-center justify-center mb-4">
        {isNotFound ? (
          <svg className="w-6 h-6 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        ) : (
          <svg className="w-6 h-6 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        )}
      </div>
      <h2 className="text-white text-lg font-semibold mb-1">{heading}</h2>
      <p className="text-slate-400 text-sm max-w-sm mb-6">{message}</p>
      <div className="flex gap-3">
        <button
          onClick={() => router.back()}
          className="text-sm text-slate-400 hover:text-slate-200 px-4 py-2 rounded-lg border border-slate-700 hover:border-slate-500 transition-colors"
        >
          ← Go back
        </button>
        {onRetry && (
          <button
            onClick={onRetry}
            className="text-sm text-cyan-400 hover:text-cyan-300 px-4 py-2 rounded-lg border border-cyan-500/40 hover:border-cyan-400 transition-colors"
          >
            Try again
          </button>
        )}
      </div>
    </div>
  );
}
