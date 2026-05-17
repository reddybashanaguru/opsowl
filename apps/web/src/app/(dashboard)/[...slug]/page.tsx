"use client";

import Link from "next/link";

export default function NotFoundPage() {
  return (
    <div className="flex flex-col items-center justify-center h-full min-h-[60vh] p-8 text-center">
      <div className="text-6xl font-bold text-slate-700 mb-4">404</div>
      <h1 className="text-xl font-semibold text-white mb-2">Page not found</h1>
      <p className="text-slate-400 text-sm mb-8 max-w-sm">
        This section isn&apos;t available yet or the URL is incorrect.
      </p>
      <Link
        href="/dashboard"
        className="px-4 py-2 bg-cyan-500/10 border border-cyan-500/30 text-cyan-400 rounded-lg text-sm hover:bg-cyan-500/20 transition-colors"
      >
        ← Back to Dashboard
      </Link>
    </div>
  );
}
