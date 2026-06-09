"use client";

import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { api, APIError, newIdempotencyKey } from "@/lib/api";
import { CaseTypeSelect } from "@/components/CaseTypeSelect";

const SLA_LABELS: Record<string, string> = {
  critical: "~4 hours to resolve",
  high:     "~24 hours to resolve",
  medium:   "~72 hours to resolve",
  low:      "~7 days to resolve",
};

const schema = z.object({
  title:       z.string().min(5, "Title must be at least 5 characters"),
  description: z.string().optional(),
  priority:    z.enum(["critical", "high", "medium", "low"]),
  tags:        z.string().optional(),
});
type FormValues = z.infer<typeof schema>;

interface CaseResponse {
  id: string;
  case_number: string;
}

interface TaskStub {
  title: string;
  description: string;
  priority: string;
  sort_order: number;
}

interface Template {
  id: string;
  name: string;
  description_skeleton: string;
  priority: string;
  tags: string[];
  is_active: boolean;
  tasks: TaskStub[];
}

interface TemplatesResponse {
  templates: Template[];
}

interface PlaybookOption {
  id: string;
  name: string;
  case_type: string;
  steps: { id: string }[];
}

export default function NewCasePage() {
  const router = useRouter();
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null);
  const [showPicker, setShowPicker] = useState(false);
  const [selectedCaseType, setSelectedCaseType] = useState("");

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { priority: "medium" },
  });

  const priority = form.watch("priority");

  const { data: templatesData } = useQuery<TemplatesResponse>({
    queryKey: ["templates-active"],
    queryFn: () => api.get<TemplatesResponse>("/api/v1/templates?active=true"),
  });
  const templates = templatesData?.templates ?? [];

  const { data: playbooksData } = useQuery<PlaybookOption[]>({
    queryKey: ["playbooks-list"],
    queryFn: () => api.get<PlaybookOption[]>("/api/v1/playbooks"),
    staleTime: 5 * 60 * 1000,
  });
  const playbooks = (playbooksData ?? []).filter((p) => p.steps?.length > 0);

  const applyTemplate = (tmpl: Template) => {
    setSelectedTemplate(tmpl);
    setShowPicker(false);
    if (tmpl.description_skeleton) {
      form.setValue("description", tmpl.description_skeleton);
    }
    form.setValue("priority", tmpl.priority as FormValues["priority"]);
    if (tmpl.tags?.length) {
      form.setValue("tags", tmpl.tags.join(", "));
    }
  };

  const clearTemplate = () => {
    setSelectedTemplate(null);
    form.reset({ priority: "medium" });
  };

  // Generate a stable idempotency key for this form render.
  // If the network fails and React Query retries, the same key is sent so the
  // server de-duplicates and returns the already-created case instead of creating another.
  const idempotencyKey = useMemo(() => newIdempotencyKey(), []);

  const { mutate, isPending, error } = useMutation({
    mutationFn: (values: FormValues) => {
      const tags = values.tags
        ? values.tags.split(",").map((t) => t.trim()).filter(Boolean)
        : [];
      return api.idempotentPost<CaseResponse>("/api/v1/cases", {
        title: values.title,
        description: values.description ?? "",
        priority: values.priority,
        case_type: selectedCaseType || undefined,
        tags,
      }, idempotencyKey);
    },
    onSuccess: async (data) => {
      if (selectedTemplate) {
        try {
          await api.post(`/api/v1/templates/${selectedTemplate.id}/apply`, { case_id: data.id });
        } catch {
          // Tasks failed to apply; navigate anyway — user can add tasks manually
        }
      }
      router.push(`/cases/${data.id}`);
    },
  });

  const errorMsg = error instanceof APIError ? error.message : error ? "Something went wrong" : null;

  return (
    <div className="p-6">
      <div className="mb-6">
        <button
          onClick={() => router.back()}
          className="text-slate-400 hover:text-slate-200 text-sm mb-4 flex items-center gap-1"
        >
          ← Back
        </button>
        <h1 className="text-white text-2xl font-bold">New Case</h1>
        <p className="text-slate-400 text-sm mt-1">Create a new investigation case</p>
      </div>

      <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-6">
        <form onSubmit={form.handleSubmit((v) => mutate(v))} className="space-y-5">
          {/* Template picker */}
          {templates.length > 0 && (
            <div>
              <label className="block text-slate-300 text-sm font-medium mb-1.5">
                Start from Template{" "}
                <span className="text-slate-500 font-normal text-xs">(optional)</span>
              </label>
              {selectedTemplate ? (
                <div className="flex items-center justify-between bg-cyan-500/10 border border-cyan-500/30 rounded-lg px-3 py-2">
                  <div className="flex items-center gap-2 text-sm">
                    <span className="text-cyan-400">📋</span>
                    <span className="text-cyan-300 font-medium">{selectedTemplate.name}</span>
                    {selectedTemplate.tasks?.length > 0 && (
                      <span className="text-cyan-500 text-xs">
                        · {selectedTemplate.tasks.length} task{selectedTemplate.tasks.length !== 1 ? "s" : ""} will be created
                      </span>
                    )}
                  </div>
                  <button
                    type="button"
                    onClick={clearTemplate}
                    className="text-cyan-500 hover:text-cyan-300 text-xs transition-colors"
                  >
                    Clear
                  </button>
                </div>
              ) : (
                <div className="relative">
                  <button
                    type="button"
                    onClick={() => setShowPicker((v) => !v)}
                    className="w-full text-left bg-[#0f172a] border border-slate-600 text-slate-400 hover:border-slate-400 rounded-lg px-3 py-2 text-sm transition-colors flex items-center justify-between"
                  >
                    <span>Select a template…</span>
                    <span className="text-slate-600">{showPicker ? "▲" : "▼"}</span>
                  </button>
                  {showPicker && (
                    <div className="absolute top-full left-0 right-0 mt-1 bg-[#1e293b] border border-slate-600 rounded-lg shadow-xl z-20 max-h-48 overflow-y-auto">
                      {templates.map((t) => (
                        <button
                          key={t.id}
                          type="button"
                          onClick={() => applyTemplate(t)}
                          className="w-full text-left px-3 py-2.5 hover:bg-slate-700/60 transition-colors border-b border-slate-700/50 last:border-0"
                        >
                          <div className="text-sm text-white font-medium">{t.name}</div>
                          {t.tasks?.length > 0 && (
                            <div className="text-xs text-slate-500 mt-0.5">
                              {t.tasks.length} task checklist · {t.priority} priority
                            </div>
                          )}
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Case Type */}
          <div>
            <label className="block text-slate-300 text-sm font-medium mb-1.5">
              Case Type{" "}
              <span className="text-slate-500 font-normal text-xs">(auto-applies a playbook if one matches)</span>
            </label>
            <CaseTypeSelect value={selectedCaseType} onChange={setSelectedCaseType} />
            {selectedCaseType && playbooks.some(p => p.case_type === selectedCaseType) && (
              <p className="text-cyan-400 text-xs mt-1">
                Playbook tasks will be auto-created when this case is opened.
              </p>
            )}
          </div>

          {/* Title */}
          <div>
            <label className="block text-slate-300 text-sm font-medium mb-1.5">
              Title <span className="text-red-400">*</span>
            </label>
            <input
              {...form.register("title")}
              placeholder="e.g. Suspicious UPI transaction cluster — Acme Corp"
              className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
            />
            {form.formState.errors.title && (
              <p className="text-red-400 text-xs mt-1">{form.formState.errors.title.message}</p>
            )}
          </div>

          {/* Description */}
          <div>
            <label className="block text-slate-300 text-sm font-medium mb-1.5">Description</label>
            <textarea
              {...form.register("description")}
              rows={4}
              placeholder="Describe the case details, observations, and context..."
              className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500 resize-none"
            />
          </div>

          {/* Priority */}
          <div>
            <label className="block text-slate-300 text-sm font-medium mb-1.5">Priority</label>
            <div className="grid grid-cols-4 gap-2">
              {(["critical", "high", "medium", "low"] as const).map((p) => {
                const colors: Record<string, string> = {
                  critical: "border-red-500/60 bg-red-500/10 text-red-400",
                  high:     "border-amber-500/60 bg-amber-500/10 text-amber-400",
                  medium:   "border-blue-500/60 bg-blue-500/10 text-blue-400",
                  low:      "border-green-500/60 bg-green-500/10 text-green-400",
                };
                const isSelected = priority === p;
                return (
                  <button
                    key={p}
                    type="button"
                    onClick={() => form.setValue("priority", p)}
                    className={`rounded-lg border px-3 py-2 text-xs font-semibold uppercase tracking-wide transition-all ${
                      isSelected
                        ? colors[p]
                        : "border-slate-600 bg-transparent text-slate-500 hover:border-slate-400"
                    }`}
                  >
                    {p}
                  </button>
                );
              })}
            </div>
            <p className="text-slate-500 text-xs mt-1.5">{SLA_LABELS[priority]}</p>
          </div>

          {/* Tags */}
          <div>
            <label className="block text-slate-300 text-sm font-medium mb-1.5">
              Tags <span className="text-slate-500 font-normal">(comma-separated)</span>
            </label>
            <input
              {...form.register("tags")}
              placeholder="e.g. upi-fraud, high-value, kyc-pending"
              className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
            />
          </div>

          {errorMsg && (
            <div className="bg-red-500/10 border border-red-500/30 rounded-lg px-4 py-3 text-red-400 text-sm">
              {errorMsg}
            </div>
          )}

          <div className="flex gap-3 pt-2">
            <button
              type="submit"
              disabled={isPending}
              className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-6 py-2 rounded-lg text-sm transition-colors disabled:opacity-60"
            >
              {isPending ? "Creating…" : "Create Case"}
            </button>
            <button
              type="button"
              onClick={() => router.back()}
              className="text-slate-400 hover:text-slate-200 px-4 py-2 text-sm"
            >
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
