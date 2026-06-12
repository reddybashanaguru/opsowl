interface TaskStub {
  title: string;
  description: string;
  priority: string;
  sort_order: number;
}

export interface Template {
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

const priorityColor: Record<string, string> = {
  critical: "text-red-400 bg-red-400/10 border-red-400/30",
  high: "text-amber-400 bg-amber-400/10 border-amber-400/30",
  medium: "text-blue-400 bg-blue-400/10 border-blue-400/30",
  low: "text-green-400 bg-green-400/10 border-green-400/30",
};

interface TemplateCardProps {
  template: Template;
  onEdit: (t: Template) => void;
  onDelete: (id: string) => void;
}

export function TemplateCard({ template, onEdit, onDelete }: TemplateCardProps) {
  return (
    <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-4 space-y-3">
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-white font-medium truncate">{template.name}</span>
            <span
              data-testid="active-badge"
              className={`shrink-0 text-[10px] font-bold px-1.5 py-0.5 rounded border uppercase ${
                template.is_active
                  ? "text-emerald-400 bg-emerald-400/10 border-emerald-400/30"
                  : "text-slate-500 bg-slate-500/10 border-slate-500/30"
              }`}
            >
              {template.is_active ? "active" : "inactive"}
            </span>
            <span
              data-testid="priority-badge"
              className={`shrink-0 text-[10px] font-bold px-1.5 py-0.5 rounded border uppercase ${
                priorityColor[template.priority] ?? priorityColor.medium
              }`}
            >
              {template.priority}
            </span>
          </div>
          {template.description_skeleton && (
            <p className="text-slate-400 text-xs line-clamp-2">{template.description_skeleton}</p>
          )}
        </div>
        <div className="flex gap-2 shrink-0">
          <button
            onClick={() => onEdit(template)}
            className="text-slate-400 hover:text-cyan-400 text-xs transition-colors"
          >
            Edit
          </button>
          <button
            onClick={() => onDelete(template.id)}
            className="text-slate-400 hover:text-red-400 text-xs transition-colors"
          >
            Delete
          </button>
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
        <span data-testid="task-count">
          {template.tasks?.length ?? 0} task{template.tasks?.length !== 1 ? "s" : ""} in checklist
        </span>
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
            <div data-testid="overflow-count" className="text-xs text-slate-500 pl-6">
              +{template.tasks.length - 3} more…
            </div>
          )}
        </div>
      )}
    </div>
  );
}
