"use client";

import { useAuth } from "@/lib/hooks/useAuth";

export default function DashboardPage() {
  const { user } = useAuth();

  return (
    <div className="p-8">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white">
          Good morning, <span className="text-cyan-400">{user?.full_name?.split(" ")[0]}</span>
        </h1>
        <p className="text-slate-400 mt-1 text-sm">
          Welcome to OpsOwl AI — your operations intelligence workspace.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        {[
          { label: "Open Cases", value: "—", color: "text-cyan-400" },
          { label: "Pending Tasks", value: "—", color: "text-violet-400" },
          { label: "AI Queries Today", value: "—", color: "text-amber-400" },
        ].map((card) => (
          <div
            key={card.label}
            className="bg-[#1e293b] border border-slate-700 rounded-xl p-5"
          >
            <div className="text-slate-400 text-sm mb-2">{card.label}</div>
            <div className={`text-3xl font-bold ${card.color}`}>{card.value}</div>
          </div>
        ))}
      </div>

      <div className="mt-8 bg-[#1e293b] border border-slate-700 rounded-xl p-6">
        <p className="text-slate-400 text-sm">
          Features are being built in phases. Cases, Timeline, AI Copilot and more coming next.
        </p>
      </div>
    </div>
  );
}
