"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import { RBACGate } from "@/components/auth/RBACGate";

// ─── Types ────────────────────────────────────────────────────────────────────

interface Filters {
  status?: string;
  priority?: string;
  sla_breached?: boolean;
}

interface Schedule {
  id: string;
  name: string;
  frequency: "daily" | "weekly";
  recipients: string[];
  filters: Filters;
  send_hour: number;
  day_of_week?: number;
  is_active: boolean;
  last_sent_at?: string;
  next_run_at: string;
  created_at: string;
}

const DAY_NAMES = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
const HOUR_OPTIONS = Array.from({ length: 24 }, (_, i) => ({
  value: i,
  label: `${String(i).padStart(2, "0")}:00 UTC`,
}));

const CASE_STATUSES = ["open", "in_progress", "pending_review", "escalated", "closed"];
const CASE_PRIORITIES = ["critical", "high", "medium", "low"];

// ─── Schedule form ────────────────────────────────────────────────────────────

function ScheduleForm({
  initial,
  onClose,
  onSuccess,
}: {
  initial?: Schedule;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const [name, setName]           = useState(initial?.name ?? "");
  const [frequency, setFrequency] = useState<"daily" | "weekly">(initial?.frequency ?? "daily");
  const [sendHour, setSendHour]   = useState(initial?.send_hour ?? 7);
  const [dayOfWeek, setDayOfWeek] = useState(initial?.day_of_week ?? 1);
  const [isActive, setIsActive]   = useState(initial?.is_active ?? true);
  const [recipientsRaw, setRecipientsRaw] = useState((initial?.recipients ?? []).join(", "));
  const [filterStatus, setFilterStatus]   = useState(initial?.filters?.status ?? "");
  const [filterPriority, setFilterPriority] = useState(initial?.filters?.priority ?? "");
  const [filterSLA, setFilterSLA]           = useState<boolean | "">(initial?.filters?.sla_breached ?? "");
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: () => {
      const filters: Filters = {};
      if (filterStatus)  filters.status   = filterStatus;
      if (filterPriority) filters.priority = filterPriority;
      if (filterSLA !== "") filters.sla_breached = filterSLA as boolean;

      const body = {
        name,
        frequency,
        recipients: recipientsRaw.split(",").map((e) => e.trim()).filter(Boolean),
        filters,
        send_hour: sendHour,
        day_of_week: frequency === "weekly" ? dayOfWeek : undefined,
        is_active: isActive,
      };
      return initial
        ? api.put<Schedule>(`/api/v1/report-schedules/${initial.id}`, body)
        : api.post<Schedule>("/api/v1/report-schedules", body);
    },
    onSuccess: () => { onSuccess(); onClose(); },
    onError: (e) => setError(e instanceof APIError ? e.message : "Failed to save schedule"),
  });

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
      <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <h2 className="text-slate-100 font-semibold mb-5">
          {initial ? "Edit Report Schedule" : "New Report Schedule"}
        </h2>
        {error && <p className="text-red-400 text-sm mb-3">{error}</p>}

        <div className="space-y-4">
          {/* Name */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">Schedule Name</label>
            <input value={name} onChange={(e) => setName(e.target.value)}
              placeholder="Daily AML Summary"
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:border-cyan-500" />
          </div>

          {/* Frequency + timing */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs text-slate-400 mb-1">Frequency</label>
              <select value={frequency} onChange={(e) => setFrequency(e.target.value as "daily" | "weekly")}
                className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-200 focus:outline-none focus:border-cyan-500">
                <option value="daily">Daily</option>
                <option value="weekly">Weekly</option>
              </select>
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Send at</label>
              <select value={sendHour} onChange={(e) => setSendHour(Number(e.target.value))}
                className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-200 focus:outline-none focus:border-cyan-500">
                {HOUR_OPTIONS.map((h) => (
                  <option key={h.value} value={h.value}>{h.label}</option>
                ))}
              </select>
            </div>
          </div>

          {frequency === "weekly" && (
            <div>
              <label className="block text-xs text-slate-400 mb-1">Day of week</label>
              <div className="flex gap-1 flex-wrap">
                {DAY_NAMES.map((d, i) => (
                  <button key={d} type="button" onClick={() => setDayOfWeek(i)}
                    className={`px-2.5 py-1 text-xs rounded transition-colors ${
                      dayOfWeek === i
                        ? "bg-cyan-600 text-white"
                        : "bg-slate-700 text-slate-400 hover:bg-slate-600"
                    }`}>
                    {d.slice(0, 3)}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Recipients */}
          <div>
            <label className="block text-xs text-slate-400 mb-1">
              Recipients <span className="text-slate-600">(comma-separated emails)</span>
            </label>
            <textarea value={recipientsRaw} onChange={(e) => setRecipientsRaw(e.target.value)} rows={2}
              placeholder="manager@bank.com, compliance@bank.com"
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:border-cyan-500 resize-none" />
          </div>

          {/* Filters */}
          <div>
            <label className="block text-xs text-slate-400 mb-2">
              Filters <span className="text-slate-600">(optional — restrict which cases are counted)</span>
            </label>
            <div className="grid grid-cols-3 gap-2">
              <div>
                <label className="block text-xs text-slate-500 mb-1">Status</label>
                <select value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)}
                  className="w-full bg-slate-900 border border-slate-600 rounded px-2 py-1 text-xs text-slate-200 focus:outline-none focus:border-cyan-500">
                  <option value="">Any</option>
                  {CASE_STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">Priority</label>
                <select value={filterPriority} onChange={(e) => setFilterPriority(e.target.value)}
                  className="w-full bg-slate-900 border border-slate-600 rounded px-2 py-1 text-xs text-slate-200 focus:outline-none focus:border-cyan-500">
                  <option value="">Any</option>
                  {CASE_PRIORITIES.map((p) => <option key={p} value={p}>{p}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs text-slate-500 mb-1">SLA</label>
                <select value={String(filterSLA)} onChange={(e) => {
                  const v = e.target.value;
                  setFilterSLA(v === "" ? "" : v === "true");
                }}
                  className="w-full bg-slate-900 border border-slate-600 rounded px-2 py-1 text-xs text-slate-200 focus:outline-none focus:border-cyan-500">
                  <option value="">Any</option>
                  <option value="true">Breached</option>
                  <option value="false">On track</option>
                </select>
              </div>
            </div>
          </div>

          {/* Active toggle */}
          <label className="flex items-center gap-3 cursor-pointer">
            <div onClick={() => setIsActive((v) => !v)}
              className={`relative w-10 h-5 rounded-full transition-colors ${isActive ? "bg-cyan-600" : "bg-slate-600"}`}>
              <span className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${isActive ? "translate-x-5" : ""}`} />
            </div>
            <span className="text-sm text-slate-300">{isActive ? "Active" : "Inactive"}</span>
          </label>
        </div>

        <div className="flex gap-3 mt-6">
          <button onClick={() => mutation.mutate()} disabled={!name || !recipientsRaw || mutation.isPending}
            className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm rounded transition-colors">
            {mutation.isPending ? "Saving…" : initial ? "Save Changes" : "Create Schedule"}
          </button>
          <button onClick={onClose} className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm transition-colors">
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Schedule card ────────────────────────────────────────────────────────────

function ScheduleCard({ sched, onUpdated }: { sched: Schedule; onUpdated: () => void }) {
  const qc = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const deleteMutation = useMutation({
    mutationFn: () => api.delete(`/api/v1/report-schedules/${sched.id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["report-schedules"] }); onUpdated(); },
  });

  const sendNow = useMutation({
    mutationFn: () => api.post(`/api/v1/report-schedules/${sched.id}/send-now`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["report-schedules"] }),
  });

  const toggleMutation = useMutation({
    mutationFn: () => api.put(`/api/v1/report-schedules/${sched.id}`, {
      name: sched.name, frequency: sched.frequency, recipients: sched.recipients,
      filters: sched.filters, send_hour: sched.send_hour,
      day_of_week: sched.day_of_week, is_active: !sched.is_active,
    }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["report-schedules"] }),
  });

  const nextRunLabel = new Date(sched.next_run_at).toLocaleString();
  const lastSentLabel = sched.last_sent_at ? new Date(sched.last_sent_at).toLocaleString() : "Never";

  const timingLabel = sched.frequency === "weekly"
    ? `Weekly on ${DAY_NAMES[sched.day_of_week ?? 1]} at ${String(sched.send_hour).padStart(2, "0")}:00 UTC`
    : `Daily at ${String(sched.send_hour).padStart(2, "0")}:00 UTC`;

  return (
    <>
      {editing && (
        <ScheduleForm initial={sched} onClose={() => setEditing(false)}
          onSuccess={() => qc.invalidateQueries({ queryKey: ["report-schedules"] })} />
      )}

      <div className="bg-slate-800/60 border border-slate-700 rounded-lg p-4">
        <div className="flex items-start gap-3">
          <button onClick={() => toggleMutation.mutate()} disabled={toggleMutation.isPending}
            title={sched.is_active ? "Disable" : "Enable"}
            className={`mt-1 w-2.5 h-2.5 rounded-full flex-shrink-0 transition-colors ${sched.is_active ? "bg-green-400" : "bg-slate-600"}`} />

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-slate-100 font-medium text-sm">{sched.name}</span>
              <span className={`text-xs px-1.5 py-0.5 rounded ${
                sched.frequency === "daily" ? "bg-blue-500/20 text-blue-400" : "bg-violet-500/20 text-violet-400"
              }`}>{sched.frequency}</span>
              {!sched.is_active && (
                <span className="text-xs text-slate-500 bg-slate-700 px-1.5 py-0.5 rounded">disabled</span>
              )}
            </div>

            <p className="text-slate-400 text-xs mt-1">{timingLabel}</p>

            <div className="flex flex-wrap gap-x-4 gap-y-0.5 mt-1.5">
              <span className="text-xs text-slate-500">
                <span className="text-slate-400">Recipients:</span>{" "}
                {sched.recipients.length > 2
                  ? `${sched.recipients.slice(0, 2).join(", ")} +${sched.recipients.length - 2} more`
                  : sched.recipients.join(", ")}
              </span>
              <span className="text-xs text-slate-500">
                <span className="text-slate-400">Next:</span> {nextRunLabel}
              </span>
              <span className="text-xs text-slate-500">
                <span className="text-slate-400">Last sent:</span> {lastSentLabel}
              </span>
            </div>

            {/* Active filters */}
            {(sched.filters.status || sched.filters.priority || sched.filters.sla_breached !== undefined) && (
              <div className="flex gap-1 mt-2 flex-wrap">
                {sched.filters.status && (
                  <span className="text-xs bg-slate-700 text-slate-400 px-1.5 py-0.5 rounded">
                    status: {sched.filters.status}
                  </span>
                )}
                {sched.filters.priority && (
                  <span className="text-xs bg-slate-700 text-slate-400 px-1.5 py-0.5 rounded">
                    priority: {sched.filters.priority}
                  </span>
                )}
                {sched.filters.sla_breached !== undefined && (
                  <span className="text-xs bg-slate-700 text-slate-400 px-1.5 py-0.5 rounded">
                    sla: {sched.filters.sla_breached ? "breached" : "on track"}
                  </span>
                )}
              </div>
            )}
          </div>

          <div className="flex items-center gap-2 flex-shrink-0">
            <button onClick={() => sendNow.mutate()} disabled={sendNow.isPending}
              title="Send report now"
              className="text-xs text-slate-400 hover:text-slate-200 transition-colors disabled:opacity-50">
              {sendNow.isPending ? "Sending…" : "Send now"}
            </button>
            <button onClick={() => setEditing(true)}
              className="text-xs text-cyan-400 hover:text-cyan-300 transition-colors">Edit</button>
            {confirmDelete ? (
              <>
                <button onClick={() => deleteMutation.mutate()} disabled={deleteMutation.isPending} className="text-xs text-white bg-red-600 hover:bg-red-500 px-2 py-0.5 rounded transition-colors disabled:opacity-50">Confirm</button>
                <button onClick={() => setConfirmDelete(false)} className="text-xs text-slate-400 hover:text-slate-200 px-2 py-0.5 rounded border border-slate-700 transition-colors">Cancel</button>
              </>
            ) : (
              <button onClick={() => setConfirmDelete(true)} className="text-xs text-red-400 hover:text-red-300 transition-colors">Delete</button>
            )}
          </div>
        </div>
      </div>
    </>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function ReportSchedulesPage() {
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);

  const { data, isLoading } = useQuery<{ schedules: Schedule[] }>({
    queryKey: ["report-schedules"],
    queryFn: () => api.get("/api/v1/report-schedules/"),
    refetchInterval: 60_000,
  });

  const schedules = data?.schedules ?? [];

  return (
    <RBACGate allowedRoles={["admin", "super_admin"]} fallback={
      <div className="p-12 flex flex-col items-center justify-center text-center">
        <div className="text-5xl mb-4">🔒</div>
        <p className="text-slate-200 font-semibold">Access Denied</p>
        <p className="text-slate-500 text-sm mt-2">Admin privileges required to manage scheduled reports.</p>
      </div>
    }>
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold text-slate-100">Automated Reports</h1>
          <p className="text-sm text-slate-400 mt-0.5">
            Schedule recurring case summary emails to managers and compliance teams — daily, weekly or monthly.
          </p>
        </div>
        <button onClick={() => setShowForm(true)}
          className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium rounded-lg transition-colors">
          + Add Schedule
        </button>
      </div>

      {showForm && (
        <ScheduleForm onClose={() => setShowForm(false)}
          onSuccess={() => qc.invalidateQueries({ queryKey: ["report-schedules"] })} />
      )}

      {/* Info box */}
      <div className="bg-slate-800/40 border border-slate-700 rounded-lg p-4 text-sm text-slate-400">
        <p className="font-medium text-slate-300 mb-1">What's in the report</p>
        <p>
          Each email includes: new cases created in the period, case counts by status
          (open, in progress, escalated…), active cases by priority, and an SLA breach alert
          if any cases are overdue. Reports are sent as HTML emails with a plain-text fallback.
        </p>
      </div>

      {/* Schedule list */}
      {isLoading ? (
        <div className="space-y-3">
          {[...Array(2)].map((_, i) => (
            <div key={i} className="h-24 bg-slate-800/40 rounded-lg animate-pulse" />
          ))}
        </div>
      ) : schedules.length === 0 ? (
        <div className="text-center py-16 text-slate-500">
          <p className="text-4xl mb-3">📊</p>
          <p className="font-medium">No report schedules configured</p>
          <p className="text-sm mt-1">Create one to start sending automated summaries.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {schedules.map((s) => (
            <ScheduleCard key={s.id} sched={s}
              onUpdated={() => qc.invalidateQueries({ queryKey: ["report-schedules"] })} />
          ))}
        </div>
      )}
    </div>
    </RBACGate>
  );
}
