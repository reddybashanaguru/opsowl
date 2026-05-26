"use client";

interface Props {
  secondsLeft: number;
  onStayLoggedIn: () => void;
  onLogoutNow: () => void;
}

export function IdleTimeoutWarning({ secondsLeft, onStayLoggedIn, onLogoutNow }: Props) {
  const mins = Math.floor(secondsLeft / 60);
  const secs = secondsLeft % 60;
  const countdown = mins > 0
    ? `${mins}:${String(secs).padStart(2, "0")}`
    : `${secs}s`;

  return (
    <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-[9999]">
      <div className="bg-[#1e293b] border border-amber-500/40 rounded-xl p-6 w-full max-w-sm shadow-2xl">
        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 rounded-full bg-amber-400/10 flex items-center justify-center flex-shrink-0">
            <svg className="w-5 h-5 text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
            </svg>
          </div>
          <div>
            <h3 className="text-white font-semibold">Session Expiring Soon</h3>
            <p className="text-amber-400 text-sm font-mono">{countdown} remaining</p>
          </div>
        </div>

        <p className="text-slate-400 text-sm mb-5">
          Your session will be ended due to inactivity. Click &ldquo;Stay Logged In&rdquo; to continue.
        </p>

        <div className="flex gap-2">
          <button
            onClick={onStayLoggedIn}
            className="flex-1 bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold py-2 rounded-lg text-sm transition-colors"
          >
            Stay Logged In
          </button>
          <button
            onClick={onLogoutNow}
            className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm transition-colors"
          >
            Log Out
          </button>
        </div>
      </div>
    </div>
  );
}
