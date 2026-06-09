"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

interface CaseType {
  id: string;
  slug: string;
  label: string;
  description: string;
}

interface Props {
  value: string;
  onChange: (slug: string) => void;
  disabled?: boolean;
  className?: string;
}

export function CaseTypeSelect({ value, onChange, disabled, className }: Props) {
  const qc = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [newSlug, setNewSlug] = useState("");
  const [newLabel, setNewLabel] = useState("");
  const [createError, setCreateError] = useState("");

  const { data: types = [] } = useQuery<CaseType[]>({
    queryKey: ["case-types"],
    queryFn: () => api.get<CaseType[]>("/api/v1/case-types"),
    staleTime: 60_000,
  });

  const createMutation = useMutation({
    mutationFn: (body: { slug: string; label: string }) =>
      api.post<CaseType>("/api/v1/case-types", body),
    onSuccess: (ct) => {
      qc.invalidateQueries({ queryKey: ["case-types"] });
      onChange(ct.slug);
      setShowCreate(false);
      setNewSlug("");
      setNewLabel("");
      setCreateError("");
    },
    onError: (e: Error) => setCreateError(e.message),
  });

  const handleSlugInput = (v: string) =>
    setNewSlug(v.toLowerCase().replace(/[^a-z0-9_]/g, "_"));

  return (
    <div className="space-y-1">
      <select
        value={value}
        onChange={(e) => {
          if (e.target.value === "__create__") {
            setShowCreate(true);
          } else {
            setShowCreate(false);
            onChange(e.target.value);
          }
        }}
        disabled={disabled}
        className={
          className ??
          "w-full bg-[#0f172a] border border-slate-600 text-white rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
        }
      >
        <option value="">— select case type —</option>
        {types.map((ct) => (
          <option key={ct.slug} value={ct.slug}>
            {ct.label} ({ct.slug})
          </option>
        ))}
        <option value="__create__">+ Create new type…</option>
      </select>

      {showCreate && (
        <div className="bg-[#0f172a] border border-cyan-500/30 rounded-lg p-3 space-y-2">
          <p className="text-cyan-400 text-xs font-medium">New Case Type</p>
          <div className="flex gap-2">
            <div className="flex-1">
              <input
                value={newSlug}
                onChange={(e) => handleSlugInput(e.target.value)}
                placeholder="slug (e.g. aml_kyc)"
                className="w-full bg-[#1e293b] border border-slate-600 text-white placeholder:text-slate-500 rounded px-2.5 py-1.5 text-xs focus:outline-none focus:border-cyan-500"
              />
              <p className="text-slate-600 text-xs mt-0.5">lowercase, underscores only</p>
            </div>
            <div className="flex-1">
              <input
                value={newLabel}
                onChange={(e) => setNewLabel(e.target.value)}
                placeholder="label (e.g. AML KYC Review)"
                className="w-full bg-[#1e293b] border border-slate-600 text-white placeholder:text-slate-500 rounded px-2.5 py-1.5 text-xs focus:outline-none focus:border-cyan-500"
              />
            </div>
          </div>
          {createError && <p className="text-red-400 text-xs">{createError}</p>}
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => createMutation.mutate({ slug: newSlug, label: newLabel })}
              disabled={!newSlug || !newLabel || createMutation.isPending}
              className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold text-xs px-3 py-1.5 rounded transition-colors disabled:opacity-50"
            >
              {createMutation.isPending ? "Creating…" : "Create"}
            </button>
            <button
              type="button"
              onClick={() => { setShowCreate(false); setCreateError(""); }}
              className="text-slate-400 hover:text-slate-200 text-xs px-2 py-1.5"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
