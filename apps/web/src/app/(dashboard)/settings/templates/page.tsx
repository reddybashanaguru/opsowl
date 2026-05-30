"use client";

import { useState, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";

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
  created_at: string;
  updated_at: string;
}

interface TemplatesResponse {
  templates: Template[];
}

const PRIORITIES = ["critical", "high", "medium", "low"] as const;
const TASK_PRIORITIES = ["high", "medium", "low"] as const;

const priorityColor: Record<string, string> = {
  critical: "text-red-400 bg-red-400/10 border-red-400/30",
  high: "text-amber-400 bg-amber-400/10 border-amber-400/30",
  medium: "text-blue-400 bg-blue-400/10 border-blue-400/30",
  low: "text-green-400 bg-green-400/10 border-green-400/30",
};

const emptyTask = (): TaskStub => ({ title: "", description: "", priority: "medium", sort_order: 0 });

function TaskStubRow({
  task,
  index,
  onChange,
  onRemove,
}: {
  task: TaskStub;
  index: number;
  onChange: (i: number, field: keyof TaskStub, value: string | number) => void;
  onRemove: (i: number) => void;
}) {
  return (
    <div className="flex items-start gap-2 p-3 bg-[#0f172a] border border-slate-700 rounded-lg">
      <span className="text-slate-600 text-xs pt-2 w-5 text-center shrink-0">{index + 1}</span>
      <div className="flex-1 space-y-2">
        <input
          value={task.title}
          onChange={(e) => onChange(index, "title", e.target.value)}
          placeholder="Task title"
          className="w-full bg-transparent border border-slate-600 text-white placeholder:text-slate-500 rounded px-2 py-1 text-sm focus:outline-none focus:border-cyan-500"
        />
        <input
          value={task.description}
          onChange={(e) => onChange(index, "description", e.target.value)}
          placeholder="Description (optional)"
          className="w-full bg-transparent border border-slate-600 text-white placeholder:text-slate-500 rounded px-2 py-1 text-xs focus:outline-none focus:border-cyan-500"
        />
      </div>
      <select
        value={task.priority}
        onChange={(e) => onChange(index, "priority", e.target.value)}
        className="bg-[#1e293b] border border-slate-600 text-slate-300 rounded px-2 py-1 text-xs focus:outline-none focus:border-cyan-500"
      >
        {TASK_PRIORITIES.map((p) => (
          <option key={p} value={p}>
            {p}
          </option>
        ))}
      </select>
      <button
        type="button"
        onClick={() => onRemove(index)}
        className="text-slate-600 hover:text-red-400 transition-colors text-sm pt-1"
      >
        ✕
      </button>
    </div>
  );
}

function TemplateForm({
  initial,
  onSave,
  onCancel,
  loading,
  error,
}: {
  initial?: Template;
  onSave: (data: {
    name: string;
    description_skeleton: string;
    priority: string;
    tags: string[];
    is_active: boolean;
    tasks: TaskStub[];
  }) => void;
  onCancel: () => void;
  loading: boolean;
  error: string | null;
}) {
  const [name, setName] = useState(initial?.name ?? "");
  const [desc, setDesc] = useState(initial?.description_skeleton ?? "");
  const [priority, setPriority] = useState(initial?.priority ?? "medium");
  const [tags, setTags] = useState(initial?.tags?.join(", ") ?? "");
  const [isActive, setIsActive] = useState(initial?.is_active ?? true);
  const [tasks, setTasks] = useState<TaskStub[]>(
    initial?.tasks?.length ? initial.tasks : [emptyTask()]
  );

  const handleTaskChange = useCallback(
    (i: number, field: keyof TaskStub, value: string | number) => {
      setTasks((prev) => prev.map((t, idx) => (idx === i ? { ...t, [field]: value } : t)));
    },
    []
  );

  const handleRemoveTask = useCallback((i: number) => {
    setTasks((prev) => prev.filter((_, idx) => idx !== i));
  }, []);

  const handleAddTask = () => setTasks((prev) => [...prev, emptyTask()]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave({
      name: name.trim(),
      description_skeleton: desc.trim(),
      priority,
      tags: tags.split(",").map((t) => t.trim()).filter(Boolean),
      is_active: isActive,
      tasks: tasks
        .filter((t) => t.title.trim())
        .map((t, i) => ({ ...t, sort_order: i })),
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {/* Name */}
      <div>
        <label className="block text-slate-300 text-sm font-medium mb-1.5">
          Template Name <span className="text-red-400">*</span>
        </label>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. UPI Fraud Investigation"
          className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
          required
        />
      </div>

      {/* Description skeleton */}
      <div>
        <label className="block text-slate-300 text-sm font-medium mb-1.5">
          Description Template{" "}
          <span className="text-slate-500 font-normal text-xs">(pre-fills the case description)</span>
        </label>
        <textarea
          value={desc}
          onChange={(e) => setDesc(e.target.value)}
          rows={3}
          placeholder="Describe the case context template..."
          className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500 resize-none"
        />
      </div>

      {/* Priority */}
      <div>
        <label className="block text-slate-300 text-sm font-medium mb-1.5">Default Priority</label>
        <div className="flex gap-2">
          {PRIORITIES.map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => setPriority(p)}
              className={`px-3 py-1.5 rounded-lg border text-xs font-semibold uppercase tracking-wide transition-all ${
                priority === p ? priorityColor[p] : "border-slate-600 text-slate-500 hover:border-slate-400"
              }`}
            >
              {p}
            </button>
          ))}
        </div>
      </div>

      {/* Tags */}
      <div>
        <label className="block text-slate-300 text-sm font-medium mb-1.5">
          Tags <span className="text-slate-500 font-normal text-xs">(comma-separated)</span>
        </label>
        <input
          value={tags}
          onChange={(e) => setTags(e.target.value)}
          placeholder="e.g. upi-fraud, aml, kyc"
          className="w-full bg-[#0f172a] border border-slate-600 text-white placeholder:text-slate-500 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
        />
      </div>

      {/* Active toggle */}
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => setIsActive((v) => !v)}
          className={`w-10 h-5 rounded-full transition-colors relative ${
            isActive ? "bg-cyan-500" : "bg-slate-600"
          }`}
        >
          <span
            className={`absolute top-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${
              isActive ? "translate-x-5" : "translate-x-0.5"
            }`}
          />
        </button>
        <span className="text-slate-300 text-sm">
          {isActive ? "Active — visible in case picker" : "Inactive — hidden from case picker"}
        </span>
      </div>

      {/* Task stubs */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-slate-300 text-sm font-medium">
            Task Checklist{" "}
            <span className="text-slate-500 font-normal text-xs">
              (added to the case when this template is applied)
            </span>
          </label>
          <span className="text-slate-500 text-xs">{tasks.filter((t) => t.title.trim()).length} task(s)</span>
        </div>
        <div className="space-y-2">
          {tasks.map((task, i) => (
            <TaskStubRow
              key={i}
              task={task}
              index={i}
              onChange={handleTaskChange}
              onRemove={handleRemoveTask}
            />
          ))}
        </div>
        <button
          type="button"
          onClick={handleAddTask}
          className="mt-2 text-cyan-400 hover:text-cyan-300 text-xs flex items-center gap-1 transition-colors"
        >
          + Add task
        </button>
      </div>

      {error && (
        <div className="bg-red-500/10 border border-red-500/30 rounded-lg px-4 py-3 text-red-400 text-sm">
          {error}
        </div>
      )}

      <div className="flex gap-3 pt-2 border-t border-slate-700">
        <button
          type="submit"
          disabled={loading || !name.trim()}
          className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-5 py-2 rounded-lg text-sm transition-colors disabled:opacity-60"
        >
          {loading ? "Saving…" : initial ? "Update Template" : "Create Template"}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="text-slate-400 hover:text-slate-200 px-4 py-2 text-sm"
        >
          Cancel
        </button>
      </div>
    </form>
  );
}

function TemplateCard({
  template,
  onEdit,
  onDelete,
}: {
  template: Template;
  onEdit: (t: Template) => void;
  onDelete: (id: string) => void;
}) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  return (
    <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-4 space-y-3">
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-white font-medium truncate">{template.name}</span>
            <span
              className={`shrink-0 text-[10px] font-bold px-1.5 py-0.5 rounded border uppercase ${
                template.is_active
                  ? "text-emerald-400 bg-emerald-400/10 border-emerald-400/30"
                  : "text-slate-500 bg-slate-500/10 border-slate-500/30"
              }`}
            >
              {template.is_active ? "active" : "inactive"}
            </span>
            <span
              className={`shrink-0 text-[10px] font-bold px-1.5 py-0.5 rounded border uppercase ${priorityColor[template.priority] ?? priorityColor.medium}`}
            >
              {template.priority}
            </span>
          </div>
          {template.description_skeleton && (
            <p className="text-slate-400 text-xs line-clamp-2">{template.description_skeleton}</p>
          )}
        </div>
        <div className="flex gap-2 shrink-0 items-center">
          <button
            onClick={() => onEdit(template)}
            className="text-slate-400 hover:text-cyan-400 text-xs transition-colors"
          >
            Edit
          </button>
          {confirmDelete ? (
            <>
              <button onClick={() => { onDelete(template.id); setConfirmDelete(false); }} className="text-[10px] text-white bg-red-600 hover:bg-red-500 px-2 py-0.5 rounded transition-colors">Confirm</button>
              <button onClick={() => setConfirmDelete(false)} className="text-[10px] text-slate-400 hover:text-slate-200 px-1.5 py-0.5 rounded border border-slate-700 transition-colors">✕</button>
            </>
          ) : (
            <button
              onClick={() => setConfirmDelete(true)}
              className="text-slate-400 hover:text-red-400 text-xs transition-colors"
            >
              Delete
            </button>
          )}
        </div>
      </div>

      {template.tags?.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {template.tags.map((tag) => (
            <span key={tag} className="text-[10px] text-slate-400 bg-slate-700/60 rounded px-1.5 py-0.5">
              {tag}
            </span>
          ))}
        </div>
      )}

      <div className="flex items-center justify-between text-xs text-slate-500">
        <span>{template.tasks?.length ?? 0} task{template.tasks?.length !== 1 ? "s" : ""} in checklist</span>
        <span>Updated {new Date(template.updated_at).toLocaleDateString()}</span>
      </div>

      {template.tasks?.length > 0 && (
        <div className="border-t border-slate-700/60 pt-2 space-y-1">
          {template.tasks.slice(0, 3).map((t, i) => (
            <div key={i} className="flex items-center gap-2 text-xs text-slate-400">
              <span className="w-4 h-4 rounded-full border border-slate-600 shrink-0" />
              <span className="truncate">{t.title}</span>
            </div>
          ))}
          {template.tasks.length > 3 && (
            <div className="text-xs text-slate-500 pl-6">
              +{template.tasks.length - 3} more…
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function TemplatesPage() {
  const qc = useQueryClient();
  const [modal, setModal] = useState<"create" | Template | null>(null);
  const [mutErr, setMutErr] = useState<string | null>(null);

  const { data, isLoading } = useQuery<TemplatesResponse>({
    queryKey: ["templates"],
    queryFn: () => api.get<TemplatesResponse>("/api/v1/templates"),
  });

  const createMut = useMutation({
    mutationFn: (body: object) => api.post<Template>("/api/v1/templates", body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["templates"] }); setModal(null); setMutErr(null); },
    onError: (e) => setMutErr(e instanceof APIError ? e.message : "Could not create template"),
  });

  const updateMut = useMutation({
    mutationFn: ({ id, body }: { id: string; body: object }) =>
      api.put<Template>(`/api/v1/templates/${id}`, body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["templates"] }); setModal(null); setMutErr(null); },
    onError: (e) => setMutErr(e instanceof APIError ? e.message : "Could not update template"),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/templates/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["templates"] }),
  });

  const handleSave = (data: {
    name: string;
    description_skeleton: string;
    priority: string;
    tags: string[];
    is_active: boolean;
    tasks: TaskStub[];
  }) => {
    setMutErr(null);
    if (modal === "create") {
      createMut.mutate(data);
    } else if (modal && typeof modal === "object") {
      updateMut.mutate({ id: modal.id, body: data });
    }
  };

  const handleDelete = (id: string) => {
    deleteMut.mutate(id);
  };

  const isModalLoading = createMut.isPending || updateMut.isPending;
  const templates = data?.templates ?? [];

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-white text-2xl font-bold">Case Templates</h1>
          <p className="text-slate-400 text-sm mt-1">
            Templates pre-fill description, priority, and tags — and automatically create a task checklist when applied.
          </p>
        </div>
        <button
          onClick={() => { setMutErr(null); setModal("create"); }}
          className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-2 rounded-lg text-sm transition-colors shrink-0"
        >
          + New Template
        </button>
      </div>

      {/* Modal */}
      {modal && (
        <div className="fixed inset-0 bg-black/60 z-50 flex items-center justify-center p-4">
          <div className="bg-[#1e293b] border border-slate-700 rounded-xl w-full max-w-2xl max-h-[90vh] overflow-y-auto p-6">
            <h2 className="text-white font-semibold text-lg mb-5">
              {modal === "create" ? "New Template" : `Edit — ${(modal as Template).name}`}
            </h2>
            <TemplateForm
              initial={modal === "create" ? undefined : (modal as Template)}
              onSave={handleSave}
              onCancel={() => { setModal(null); setMutErr(null); }}
              loading={isModalLoading}
              error={mutErr}
            />
          </div>
        </div>
      )}

      {/* Content */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="bg-[#1e293b] border border-slate-700 rounded-xl p-4 h-40 animate-pulse" />
          ))}
        </div>
      ) : templates.length === 0 ? (
        <div className="bg-[#1e293b] border border-slate-700 rounded-xl p-12 text-center">
          <div className="text-4xl mb-3">📋</div>
          <p className="text-slate-300 font-medium">No templates yet</p>
          <p className="text-slate-500 text-sm mt-1">
            Create your first template to speed up case creation with pre-defined task checklists.
          </p>
          <button
            onClick={() => { setMutErr(null); setModal("create"); }}
            className="mt-4 bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold px-4 py-2 rounded-lg text-sm transition-colors"
          >
            Create Template
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {templates.map((t) => (
            <TemplateCard
              key={t.id}
              template={t}
              onEdit={(tmpl) => { setMutErr(null); setModal(tmpl); }}
              onDelete={handleDelete}
            />
          ))}
        </div>
      )}
    </div>
  );
}
