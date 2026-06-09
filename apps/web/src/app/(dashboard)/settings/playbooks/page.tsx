"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { RBACGate } from "@/components/auth/RBACGate";
import { CaseTypeSelect } from "@/components/CaseTypeSelect";

interface StepResponse {
  id: string;
  playbook_id: string;
  title: string;
  description: string;
  priority: string;
  step_order: number;
  due_hours: number | null;
  depends_on_step: number;
  created_at: string;
}

interface PlaybookResponse {
  id: string;
  name: string;
  description: string;
  case_type: string;
  is_active: boolean;
  steps: StepResponse[];
  created_at: string;
  updated_at: string;
}

type PlaybooksList = PlaybookResponse[];

const PRIORITIES = ["critical", "high", "medium", "low"] as const;

const priorityColor: Record<string, string> = {
  critical: "text-red-400 bg-red-400/10 border-red-400/30",
  high: "text-amber-400 bg-amber-400/10 border-amber-400/30",
  medium: "text-blue-400 bg-blue-400/10 border-blue-400/30",
  low: "text-green-400 bg-green-400/10 border-green-400/30",
};

interface StepInput {
  title: string;
  description: string;
  priority: string;
  step_order: number;
  due_hours: string;
  depends_on_step: number;
}

const emptyStep = (order: number): StepInput => ({
  title: "",
  description: "",
  priority: "medium",
  step_order: order,
  due_hours: "",
  depends_on_step: 0,
});

function StepRow({
  step,
  index,
  totalSteps,
  onChange,
  onRemove,
}: {
  step: StepInput;
  index: number;
  totalSteps: number;
  onChange: (i: number, field: keyof StepInput, value: string | number) => void;
  onRemove: (i: number) => void;
}) {
  return (
    <div className="flex items-start gap-2 p-3 bg-[#0f172a] border border-slate-700 rounded-lg">
      <span className="text-cyan-400 text-xs pt-2.5 w-6 text-center shrink-0 font-mono">
        {index + 1}
      </span>
      <div className="flex-1 space-y-2">
        <input
          value={step.title}
          onChange={(e) => onChange(index, "title", e.target.value)}
          placeholder="Step title (e.g. Freeze accounts)"
          className="w-full bg-[#1e293b] border border-slate-600 text-white placeholder:text-slate-600 rounded px-2.5 py-1.5 text-sm focus:outline-none focus:border-cyan-500"
        />
        <div className="flex gap-2">
          <select
            value={step.priority}
            onChange={(e) => onChange(index, "priority", e.target.value)}
            className="bg-[#1e293b] border border-slate-600 text-slate-300 rounded px-2 py-1 text-xs focus:outline-none focus:border-cyan-500"
          >
            {PRIORITIES.map((p) => (
              <option key={p} value={p}>{p}</option>
            ))}
          </select>
          <input
            type="number"
            value={step.due_hours}
            onChange={(e) => onChange(index, "due_hours", e.target.value)}
            placeholder="Due hours"
            min={1}
            className="w-28 bg-[#1e293b] border border-slate-600 text-slate-300 placeholder:text-slate-600 rounded px-2 py-1 text-xs focus:outline-none focus:border-cyan-500"
          />
          <select
            value={step.depends_on_step}
            onChange={(e) => onChange(index, "depends_on_step", Number(e.target.value))}
            className="bg-[#1e293b] border border-slate-600 text-slate-300 rounded px-2 py-1 text-xs focus:outline-none focus:border-cyan-500"
          >
            <option value={0}>No dependency</option>
            {Array.from({ length: totalSteps }, (_, i) => i + 1)
              .filter((n) => n !== index + 1)
              .map((n) => (
                <option key={n} value={n}>
                  After step {n}
                </option>
              ))}
          </select>
        </div>
        <input
          value={step.description}
          onChange={(e) => onChange(index, "description", e.target.value)}
          placeholder="Description (optional)"
          className="w-full bg-[#1e293b] border border-slate-600 text-white placeholder:text-slate-600 rounded px-2.5 py-1.5 text-xs focus:outline-none focus:border-cyan-500"
        />
      </div>
      <button
        type="button"
        onClick={() => onRemove(index)}
        className="mt-2 text-slate-600 hover:text-red-400 text-xs transition-colors"
      >
        ✕
      </button>
    </div>
  );
}

function PlaybookForm({
  onSuccess,
  onCancel,
}: {
  onSuccess: () => void;
  onCancel: () => void;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [caseType, setCaseType] = useState("");
  const [steps, setSteps] = useState<StepInput[]>([emptyStep(1)]);
  const [error, setError] = useState("");

  const mutation = useMutation({
    mutationFn: () => {
      const payload = {
        name,
        description,
        case_type: caseType,
        steps: steps
          .filter((s) => s.title.trim())
          .map((s, i) => ({
            title: s.title,
            description: s.description,
            priority: s.priority,
            step_order: i + 1,
            due_hours: s.due_hours ? parseInt(s.due_hours) : undefined,
            depends_on_step: s.depends_on_step,
          })),
      };
      return api.post<PlaybookResponse>("/api/v1/playbooks", payload);
    },
    onSuccess: () => onSuccess(),
    onError: (err) =>
      setError(err instanceof APIError ? err.message : "Failed to create playbook"),
  });

  const updateStep = (i: number, field: keyof StepInput, value: string | number) => {
    setSteps((prev) => prev.map((s, idx) => (idx === i ? { ...s, [field]: value } : s)));
  };

  const addStep = () =>
    setSteps((prev) => [...prev, emptyStep(prev.length + 1)]);

  const removeStep = (i: number) =>
    setSteps((prev) => prev.filter((_, idx) => idx !== i).map((s, idx) => ({ ...s, step_order: idx + 1 })));

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-slate-300 text-xs font-medium mb-1">Playbook Name *</label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. AML Investigation"
            className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-600 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
          />
        </div>
        <div>
          <label className="block text-slate-300 text-xs font-medium mb-1">Case Type *</label>
          <CaseTypeSelect value={caseType} onChange={setCaseType} />
        </div>
      </div>

      <div>
        <label className="block text-slate-300 text-xs font-medium mb-1">Description</label>
        <input
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Brief description of when to use this playbook"
          className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-600 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
        />
      </div>

      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-slate-300 text-xs font-medium">Steps</label>
          <button
            type="button"
            onClick={addStep}
            className="text-cyan-400 hover:text-cyan-300 text-xs transition-colors"
          >
            + Add step
          </button>
        </div>
        <div className="space-y-2 max-h-80 overflow-y-auto pr-1">
          {steps.map((step, i) => (
            <StepRow
              key={i}
              step={step}
              index={i}
              totalSteps={steps.length}
              onChange={updateStep}
              onRemove={removeStep}
            />
          ))}
        </div>
        <p className="text-slate-600 text-xs mt-2">
          Steps with "After step N" dependency will be blocked until step N is completed.
        </p>
      </div>

      {error && (
        <div className="bg-red-500/10 border border-red-500/30 rounded-lg px-3 py-2 text-red-400 text-sm">
          {error}
        </div>
      )}

      <div className="flex gap-3">
        <button
          type="button"
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending || !name.trim() || !caseType.trim()}
          className="bg-cyan-500 hover:bg-cyan-400 text-white font-semibold px-5 py-2 rounded-lg text-sm transition-colors disabled:opacity-50"
        >
          {mutation.isPending ? "Creating…" : "Create Playbook"}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="text-slate-400 hover:text-slate-200 px-4 py-2 text-sm"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}

function PlaybookCard({ playbook, onDelete }: { playbook: PlaybookResponse; onDelete: () => void }) {
  const [expanded, setExpanded] = useState(false);

  const toggleMutation = useMutation({
    mutationFn: () =>
      api.patch<PlaybookResponse>(`/api/v1/playbooks/${playbook.id}`, {
        is_active: !playbook.is_active,
      }),
    onSuccess: () => window.location.reload(),
  });

  return (
    <div className={`bg-[#1e293b] border rounded-xl overflow-hidden transition-colors ${
      playbook.is_active ? "border-slate-700" : "border-slate-700/40 opacity-60"
    }`}>
      <div
        className="p-4 cursor-pointer flex items-start justify-between gap-3"
        onClick={() => setExpanded((v) => !v)}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-white font-semibold text-sm">{playbook.name}</span>
            <span className="text-xs font-mono bg-cyan-500/10 border border-cyan-500/30 text-cyan-400 rounded px-1.5 py-0.5">
              {playbook.case_type}
            </span>
            {!playbook.is_active && (
              <span className="text-xs bg-slate-700 text-slate-400 rounded px-1.5 py-0.5">inactive</span>
            )}
          </div>
          {playbook.description && (
            <p className="text-slate-500 text-xs mt-1">{playbook.description}</p>
          )}
          <p className="text-slate-600 text-xs mt-1">
            {playbook.steps.length} step{playbook.steps.length !== 1 ? "s" : ""}
          </p>
        </div>
        <span className="text-slate-600 text-xs mt-1">{expanded ? "▲" : "▼"}</span>
      </div>

      {expanded && (
        <div className="px-4 pb-4 border-t border-slate-700/60 pt-3 space-y-2">
          {playbook.steps.length === 0 ? (
            <p className="text-slate-600 text-xs">No steps defined.</p>
          ) : (
            playbook.steps.map((step) => (
              <div
                key={step.id}
                className="flex items-start gap-3 p-2.5 bg-[#0f172a] rounded-lg border border-slate-700/60"
              >
                <span className="text-cyan-400 text-xs font-mono w-5 text-center shrink-0 mt-0.5">
                  {step.step_order}
                </span>
                <div className="flex-1 min-w-0">
                  <p className="text-white text-sm font-medium">{step.title}</p>
                  {step.description && (
                    <p className="text-slate-500 text-xs mt-0.5">{step.description}</p>
                  )}
                  <div className="flex items-center gap-2 mt-1.5 flex-wrap">
                    <span className={`text-xs border rounded px-1.5 py-0.5 ${priorityColor[step.priority]}`}>
                      {step.priority}
                    </span>
                    {step.due_hours && (
                      <span className="text-slate-500 text-xs">
                        Due: +{step.due_hours}h
                      </span>
                    )}
                    {step.depends_on_step > 0 && (
                      <span className="text-orange-400 text-xs">
                        After step {step.depends_on_step}
                      </span>
                    )}
                  </div>
                </div>
              </div>
            ))
          )}

          <div className="flex gap-2 pt-1">
            <button
              onClick={() => toggleMutation.mutate()}
              disabled={toggleMutation.isPending}
              className={`text-xs px-3 py-1.5 rounded-lg border transition-colors ${
                playbook.is_active
                  ? "border-slate-600 text-slate-400 hover:border-red-500/40 hover:text-red-400"
                  : "border-green-500/40 text-green-400 hover:border-green-400"
              }`}
            >
              {toggleMutation.isPending ? "…" : playbook.is_active ? "Deactivate" : "Activate"}
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                if (confirm(`Delete playbook "${playbook.name}"?`)) onDelete();
              }}
              className="text-xs px-3 py-1.5 rounded-lg border border-slate-700 text-slate-500 hover:border-red-500/40 hover:text-red-400 transition-colors"
            >
              Delete
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default function PlaybooksPage() {
  const qc = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [statusFilter, setStatusFilter] = useState<"active" | "inactive" | "all">("active");

  const { data, isLoading } = useQuery<PlaybooksList>({
    queryKey: ["playbooks"],
    queryFn: () => api.get<PlaybooksList>("/api/v1/playbooks"),
  });
  const allPlaybooks = data ?? [];
  const playbooks = allPlaybooks.filter((pb) => {
    if (statusFilter === "active") return pb.is_active;
    if (statusFilter === "inactive") return !pb.is_active;
    return true;
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/playbooks/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["playbooks"] }),
  });

  return (
    <RBACGate allowedRoles={["admin", "super_admin"]}>
      <div className="p-6 max-w-4xl">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-white text-2xl font-bold">Investigation Playbooks</h1>
            <p className="text-slate-400 text-sm mt-1">
              Ordered task sequences auto-applied when a case with a matching type is created.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as typeof statusFilter)}
              className="bg-[#1e293b] border border-slate-600 text-slate-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-cyan-500"
            >
              <option value="active">Active ({allPlaybooks.filter(p => p.is_active).length})</option>
              <option value="inactive">Inactive ({allPlaybooks.filter(p => !p.is_active).length})</option>
              <option value="all">All ({allPlaybooks.length})</option>
            </select>
            {!showCreate && (
              <button
                onClick={() => setShowCreate(true)}
                className="bg-cyan-500 hover:bg-cyan-400 text-white font-semibold px-4 py-2 rounded-lg text-sm transition-colors"
              >
                New Playbook
              </button>
            )}
          </div>
        </div>

        {showCreate && (
          <div className="bg-[#1e293b] border border-cyan-500/30 rounded-xl p-5 mb-6">
            <h2 className="text-white font-semibold text-sm mb-4">New Playbook</h2>
            <PlaybookForm
              onSuccess={() => {
                setShowCreate(false);
                qc.invalidateQueries({ queryKey: ["playbooks"] });
              }}
              onCancel={() => setShowCreate(false)}
            />
          </div>
        )}

        {isLoading ? (
          <div className="space-y-3">
            {[1, 2].map((i) => (
              <div key={i} className="h-20 bg-[#1e293b] rounded-xl animate-pulse" />
            ))}
          </div>
        ) : playbooks.length === 0 ? (
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-10 text-center">
            <div className="text-4xl mb-3">📖</div>
            <p className="text-white font-semibold mb-1">
              {allPlaybooks.length === 0 ? "No playbooks yet" : `No ${statusFilter} playbooks`}
            </p>
            <p className="text-slate-500 text-sm">
              {allPlaybooks.length === 0
                ? "Create your first playbook to auto-apply ordered task sequences to new cases."
                : `Switch the filter to view ${statusFilter === "active" ? "inactive" : "active"} playbooks.`}
            </p>
          </div>
        ) : (
          <div className="space-y-3 overflow-y-auto max-h-[calc(100vh-280px)] min-h-[160px] pr-1">
            {playbooks.map((pb) => (
              <PlaybookCard
                key={pb.id}
                playbook={pb}
                onDelete={() => deleteMutation.mutate(pb.id)}
              />
            ))}
          </div>
        )}
      </div>
    </RBACGate>
  );
}
