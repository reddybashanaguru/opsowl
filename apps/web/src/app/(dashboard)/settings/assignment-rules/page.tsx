"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { RBACGate } from "@/components/auth/RBACGate";
import { CaseTypeSelect } from "@/components/CaseTypeSelect";

interface AssignmentRule {
  id: string;
  name: string;
  description?: string;
  priority_order: number;
  match_priority?: string;
  match_case_type?: string;
  match_source_id?: string;
  assign_to_user?: string;
  assign_to_role?: string;
  is_active: boolean;
  created_at: string;
}

interface UserOption {
  id: string;
  full_name: string;
  email: string;
  role: string;
}

interface RuleFormData {
  name: string;
  description: string;
  priority_order: number;
  match_priority: string;
  match_case_type: string;
  assign_to_role: string;
  assign_to_user: string;
}

const EMPTY_FORM: RuleFormData = {
  name: "",
  description: "",
  priority_order: 100,
  match_priority: "",
  match_case_type: "",
  assign_to_role: "analyst",
  assign_to_user: "",
};

const PRIORITIES = ["", "critical", "high", "medium", "low"];
const ROLES = ["", "analyst", "manager", "admin"];

const PRIORITY_COLORS: Record<string, string> = {
  critical: "text-red-400 bg-red-400/10",
  high: "text-orange-400 bg-orange-400/10",
  medium: "text-amber-400 bg-amber-400/10",
  low: "text-slate-400 bg-slate-400/10",
};

const NAME_MIN = 5;
const NAME_MAX = 100;
const DESC_MAX = 300;
const PRIORITY_ORDER_MIN = 1;
const PRIORITY_ORDER_MAX = 999;

function validate(form: RuleFormData, targetType: "role" | "user") {
  const name = form.name.trim();
  return {
    name: !name
      ? "Rule name is required."
      : name.length < NAME_MIN
      ? `Must be at least ${NAME_MIN} characters (${name.length}/${NAME_MIN}).`
      : name.length > NAME_MAX
      ? `Must be under ${NAME_MAX} characters.`
      : null,
    priority_order:
      !form.priority_order || form.priority_order < PRIORITY_ORDER_MIN
        ? `Must be at least ${PRIORITY_ORDER_MIN}.`
        : form.priority_order > PRIORITY_ORDER_MAX
        ? `Must be ${PRIORITY_ORDER_MAX} or less.`
        : null,
    description:
      form.description.length > DESC_MAX
        ? `Must be under ${DESC_MAX} characters (${form.description.length}/${DESC_MAX}).`
        : null,
    assign:
      targetType === "role" && !form.assign_to_role
        ? "Select a role to assign cases to."
        : targetType === "user" && !form.assign_to_user
        ? "Select a user to assign cases to."
        : null,
  };
}

function RuleModal({
  rule,
  users,
  onClose,
  onSaved,
}: {
  rule: AssignmentRule | null;
  users: UserOption[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const isEdit = rule !== null;
  const [form, setForm] = useState<RuleFormData>(
    rule
      ? {
          name: rule.name,
          description: rule.description ?? "",
          priority_order: rule.priority_order,
          match_priority: rule.match_priority ?? "",
          match_case_type: rule.match_case_type ?? "",
          assign_to_role: rule.assign_to_role ?? "",
          assign_to_user: rule.assign_to_user ?? "",
        }
      : EMPTY_FORM
  );
  const [targetType, setTargetType] = useState<"role" | "user">(
    rule?.assign_to_user ? "user" : "role"
  );
  const [touched, setTouched] = useState<Set<string>>(new Set());
  const [submitted, setSubmitted] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);

  const errors = validate(form, targetType);
  const hasErrors = Object.values(errors).some(Boolean);

  function touch(k: string) {
    setTouched((prev) => new Set(prev).add(k));
  }

  // Show error for a field if it's been touched OR user already tried to submit
  function show(k: keyof typeof errors) {
    return (touched.has(k) || submitted) && !!errors[k];
  }

  function fieldBorder(k: keyof typeof errors, base = "border-slate-600") {
    return show(k) ? "border-red-500/60" : base;
  }

  const mutation = useMutation({
    mutationFn: () => {
      const body = {
        name: form.name.trim(),
        description: form.description,
        priority_order: form.priority_order,
        match_priority: form.match_priority || undefined,
        match_case_type: form.match_case_type || undefined,
        assign_to_role: targetType === "role" ? form.assign_to_role || undefined : undefined,
        assign_to_user: targetType === "user" ? form.assign_to_user || undefined : undefined,
      };
      return isEdit
        ? api.put(`/api/v1/assignment-rules/${rule!.id}`, body)
        : api.post("/api/v1/assignment-rules", body);
    },
    onSuccess: () => onSaved(),
    onError: (e) => setServerError(e instanceof APIError ? e.message : "Failed to save rule"),
  });

  function field(k: keyof RuleFormData, v: string | number) {
    setForm((prev) => ({ ...prev, [k]: v }));
  }

  function handleSave() {
    setSubmitted(true);
    if (hasErrors) return;
    mutation.mutate();
  }

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="bg-[#1e293b] border border-slate-700 rounded-2xl w-full max-w-lg p-6 shadow-2xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-white font-semibold text-lg">
            {isEdit ? "Edit Rule" : "New Rule"}
          </h2>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300 text-xl">✕</button>
        </div>

        {serverError && (
          <div className="mb-4 px-4 py-3 rounded-lg text-sm bg-red-500/10 border border-red-500/30 text-red-400">
            {serverError}
          </div>
        )}

        <div className="space-y-4">
          {/* Name + Priority Order */}
          <div className="grid grid-cols-3 gap-3">
            <div className="col-span-2">
              <label className="block text-xs text-slate-400 mb-1">
                Rule Name <span className="text-red-400">*</span>
              </label>
              <input
                value={form.name}
                onChange={(e) => field("name", e.target.value)}
                onBlur={() => touch("name")}
                placeholder="e.g. Auto-assign critical AML cases"
                maxLength={NAME_MAX}
                className={`w-full bg-slate-900 border rounded px-3 py-2 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:border-cyan-500 transition-colors ${fieldBorder("name")}`}
              />
              {show("name") && (
                <p className="text-red-400 text-[10px] mt-0.5">{errors.name}</p>
              )}
              {!show("name") && (
                <p className="text-slate-600 text-[10px] mt-0.5">
                  {form.name.trim().length}/{NAME_MAX} · min {NAME_MIN} chars
                </p>
              )}
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">
                Priority Order <span className="text-red-400">*</span>
              </label>
              <input
                type="number"
                value={form.priority_order}
                onChange={(e) => field("priority_order", Number(e.target.value))}
                onBlur={() => touch("priority_order")}
                min={PRIORITY_ORDER_MIN}
                max={PRIORITY_ORDER_MAX}
                className={`w-full bg-slate-900 border rounded px-3 py-2 text-sm text-slate-100 focus:outline-none focus:border-cyan-500 transition-colors ${fieldBorder("priority_order")}`}
              />
              {show("priority_order") ? (
                <p className="text-red-400 text-[10px] mt-0.5">{errors.priority_order}</p>
              ) : (
                <p className="text-slate-600 text-[10px] mt-0.5">1–999, lower fires first</p>
              )}
            </div>
          </div>

          {/* Description */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="block text-xs text-slate-400">Description</label>
              <span className={`text-[10px] ${form.description.length > DESC_MAX ? "text-red-400" : "text-slate-600"}`}>
                {form.description.length}/{DESC_MAX}
              </span>
            </div>
            <input
              value={form.description}
              onChange={(e) => field("description", e.target.value)}
              onBlur={() => touch("description")}
              placeholder="What does this rule do? e.g. Routes critical sanctions cases to the specialist"
              maxLength={DESC_MAX + 1}
              className={`w-full bg-slate-900 border rounded px-3 py-2 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:border-cyan-500 transition-colors ${fieldBorder("description")}`}
            />
            {show("description") && (
              <p className="text-red-400 text-[10px] mt-0.5">{errors.description}</p>
            )}
          </div>

          {/* Match Conditions */}
          <div className="border border-slate-700/60 rounded-lg p-4 space-y-3">
            <p className="text-xs font-semibold text-slate-400 uppercase tracking-widest">Match Conditions</p>
            <p className="text-slate-600 text-[11px] -mt-1">Leave blank to match any value. All filled conditions must match (AND logic).</p>
            {!form.match_priority && !form.match_case_type && (
              <p className="text-amber-400/80 text-[11px] bg-amber-400/5 border border-amber-400/20 rounded px-2 py-1">
                ⚠ Both blank — this rule will match every new case (use as catch-all fallback only).
              </p>
            )}
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-slate-400 mb-1">Priority</label>
                <select
                  value={form.match_priority}
                  onChange={(e) => field("match_priority", e.target.value)}
                  className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-cyan-500"
                >
                  {PRIORITIES.map((p) => (
                    <option key={p} value={p}>
                      {p === "" ? "Any priority" : p.charAt(0).toUpperCase() + p.slice(1)}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs text-slate-400 mb-1">Case Type</label>
                <CaseTypeSelect
                  value={form.match_case_type}
                  onChange={(v) => field("match_case_type", v)}
                  className="w-full bg-slate-900 border border-slate-600 text-slate-300 rounded px-3 py-2 text-sm focus:outline-none focus:border-cyan-500"
                />
              </div>
            </div>
          </div>

          {/* Assign To */}
          <div className="border border-slate-700/60 rounded-lg p-4 space-y-3">
            <div className="flex items-center gap-1.5">
              <p className="text-xs font-semibold text-slate-400 uppercase tracking-widest">Assign To</p>
              <span className="text-red-400 text-xs">*</span>
            </div>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={() => { setTargetType("role"); field("assign_to_user", ""); }}
                className={`px-3 py-1.5 rounded text-xs font-medium transition-colors ${
                  targetType === "role"
                    ? "bg-cyan-500/20 border border-cyan-500/40 text-cyan-400"
                    : "bg-slate-800 border border-slate-700 text-slate-400 hover:border-slate-500"
                }`}
              >
                Role (round-robin)
              </button>
              <button
                type="button"
                onClick={() => { setTargetType("user"); field("assign_to_role", ""); }}
                className={`px-3 py-1.5 rounded text-xs font-medium transition-colors ${
                  targetType === "user"
                    ? "bg-cyan-500/20 border border-cyan-500/40 text-cyan-400"
                    : "bg-slate-800 border border-slate-700 text-slate-400 hover:border-slate-500"
                }`}
              >
                Specific User
              </button>
            </div>

            {targetType === "role" ? (
              <div>
                <label className="block text-xs text-slate-400 mb-1">
                  Role <span className="text-red-400">*</span>
                </label>
                <select
                  value={form.assign_to_role}
                  onChange={(e) => { field("assign_to_role", e.target.value); touch("assign"); }}
                  onBlur={() => touch("assign")}
                  className={`w-full bg-slate-900 border rounded px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-cyan-500 ${fieldBorder("assign")}`}
                >
                  {ROLES.map((r) => (
                    <option key={r} value={r}>
                      {r === "" ? "— select a role —" : r.charAt(0).toUpperCase() + r.slice(1)}
                    </option>
                  ))}
                </select>
                {show("assign") && (
                  <p className="text-red-400 text-[10px] mt-0.5">{errors.assign}</p>
                )}
                <p className="text-slate-600 text-[10px] mt-1">
                  Assigns to the user with fewest open cases in this role.
                </p>
              </div>
            ) : (
              <div>
                <label className="block text-xs text-slate-400 mb-1">
                  User <span className="text-red-400">*</span>
                </label>
                <select
                  value={form.assign_to_user}
                  onChange={(e) => { field("assign_to_user", e.target.value); touch("assign"); }}
                  onBlur={() => touch("assign")}
                  className={`w-full bg-slate-900 border rounded px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-cyan-500 ${fieldBorder("assign")}`}
                >
                  <option value="">— select a user —</option>
                  {users.map((u) => (
                    <option key={u.id} value={u.id}>
                      {u.full_name} ({u.role})
                    </option>
                  ))}
                </select>
                {show("assign") && (
                  <p className="text-red-400 text-[10px] mt-0.5">{errors.assign}</p>
                )}
              </div>
            )}
          </div>
        </div>

        <div className="flex gap-3 mt-6">
          <button
            onClick={handleSave}
            disabled={mutation.isPending}
            className="flex-1 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm font-medium rounded transition-colors"
          >
            {mutation.isPending ? "Saving…" : "Save Rule"}
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm border border-slate-700 rounded transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

function AssignmentRulesContent() {
  const qc = useQueryClient();
  const [modalRule, setModalRule] = useState<AssignmentRule | "new" | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const { data: rules = [], isLoading } = useQuery({
    queryKey: ["assignment-rules"],
    queryFn: () => api.get<{ rules: AssignmentRule[] }>("/api/v1/assignment-rules"),
    select: (d) => (d.rules ?? []).filter((r) => r.is_active),
  });

  const { data: usersData } = useQuery({
    queryKey: ["users-all"],
    queryFn: () => api.get<UserOption[]>("/api/v1/users?limit=100"),
    staleTime: 60_000,
  });
  const users: UserOption[] = usersData ?? [];

  const userMap = Object.fromEntries(users.map((u) => [u.id, u]));

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/assignment-rules/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["assignment-rules"] });
      setDeletingId(null);
    },
  });

  function handleSaved() {
    qc.invalidateQueries({ queryKey: ["assignment-rules"] });
    setModalRule(null);
  }

  return (
    <div className="p-8">
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 mb-1">Assignment Rules</h1>
          <p className="text-slate-400 text-sm">
            Automatically route new cases to users or teams. Rules are evaluated in priority order — first match wins.
          </p>
        </div>
        <button
          onClick={() => setModalRule("new")}
          className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium rounded-lg transition-colors"
        >
          + Add Rule
        </button>
      </div>

      {rules.length === 0 && !isLoading && (
        <div className="p-5 bg-slate-800/40 border border-slate-700/50 rounded-lg text-slate-500 text-sm text-center">
          No rules configured. New cases will remain unassigned until manually routed.
          <button
            onClick={() => setModalRule("new")}
            className="block mx-auto mt-2 text-cyan-400 hover:text-cyan-300"
          >
            Create your first rule →
          </button>
        </div>
      )}

      {isLoading && (
        <div className="space-y-3">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="h-16 bg-slate-800/60 animate-pulse rounded-xl" />
          ))}
        </div>
      )}

      <div className="space-y-3 overflow-y-auto max-h-[calc(100vh-240px)] pr-1">
        {rules.map((rule, idx) => (
          <div
            key={rule.id}
            data-testid="rule-card"
            className={`p-5 bg-[#1e293b] border rounded-xl transition-colors ${
              rule.is_active ? "border-slate-700" : "border-slate-700/40 opacity-60"
            }`}
          >
            <div className="flex items-start justify-between">
              <div className="flex items-start gap-4">
                <span className="w-7 h-7 flex items-center justify-center rounded-full bg-slate-700 text-slate-400 text-xs font-bold tabular-nums shrink-0 mt-0.5">
                  {idx + 1}
                </span>

                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1 flex-wrap">
                    <span className="text-slate-200 font-medium text-sm">{rule.name}</span>
                    {rule.match_priority && (
                      <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium ${PRIORITY_COLORS[rule.match_priority] ?? ""}`}>
                        {rule.match_priority}
                      </span>
                    )}
                    {rule.match_case_type && (
                      <span className="text-[10px] px-2 py-0.5 rounded font-mono bg-cyan-500/10 border border-cyan-500/30 text-cyan-400">
                        {rule.match_case_type}
                      </span>
                    )}
                  </div>
                  {rule.description && (
                    <p className="text-slate-500 text-xs mb-2">{rule.description}</p>
                  )}
                  <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-400">
                    <span>
                      <span className="text-slate-600">Match: </span>
                      {rule.match_priority || rule.match_case_type ? (
                        <span className="text-slate-300">
                          {[rule.match_priority, rule.match_case_type].filter(Boolean).join(" · ")}
                        </span>
                      ) : (
                        <span className="italic text-slate-600">any case</span>
                      )}
                    </span>
                    <span>
                      <span className="text-slate-600">Assign: </span>
                      {rule.assign_to_role ? (
                        <span className="text-cyan-400/80">least-loaded {rule.assign_to_role}</span>
                      ) : rule.assign_to_user ? (
                        <span className="text-cyan-400/80">
                          {userMap[rule.assign_to_user]?.full_name ?? rule.assign_to_user.slice(0, 8) + "…"}
                        </span>
                      ) : (
                        <span className="text-slate-600 italic">not set</span>
                      )}
                    </span>
                  </div>
                </div>
              </div>

              <div className="flex items-center gap-2 ml-4 shrink-0">
                <button
                  onClick={() => setModalRule(rule)}
                  className="text-xs text-slate-500 hover:text-slate-200 px-3 py-1.5 border border-slate-700 rounded hover:border-slate-500 transition-colors"
                >
                  Edit
                </button>
                {deletingId === rule.id ? (
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-red-400">Delete?</span>
                    <button
                      onClick={() => deleteMutation.mutate(rule.id)}
                      className="text-xs px-2 py-1 bg-red-600/20 border border-red-500/30 text-red-400 rounded hover:bg-red-600/30"
                    >
                      Yes
                    </button>
                    <button
                      onClick={() => setDeletingId(null)}
                      className="text-xs text-slate-500 hover:text-slate-300"
                    >
                      No
                    </button>
                  </div>
                ) : (
                  <button
                    onClick={() => setDeletingId(rule.id)}
                    className="text-xs text-slate-600 hover:text-red-400 transition-colors"
                  >
                    Delete
                  </button>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>

      {rules.length > 0 && (
        <p className="text-slate-600 text-xs mt-4">
          Rules are evaluated in order above. Reorder by adjusting the Priority Order field.
        </p>
      )}

      {modalRule !== null && (
        <RuleModal
          rule={modalRule === "new" ? null : modalRule}
          users={users}
          onClose={() => setModalRule(null)}
          onSaved={handleSaved}
        />
      )}
    </div>
  );
}

export default function AssignmentRulesPage() {
  return (
    <RBACGate allowedRoles={["admin", "super_admin"]} fallback={
      <div className="p-8 text-slate-400 text-sm">Admin access required to manage assignment rules.</div>
    }>
      <AssignmentRulesContent />
    </RBACGate>
  );
}
