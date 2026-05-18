"use client";

import { useState } from "react";
import { useConfig } from "@/lib/hooks/useConfig";
import { useConfigStore, DEFAULT_CONFIG, type AppConfig } from "@/lib/stores/config.store";
import { api } from "@/lib/api";
import { RBACGate } from "@/components/auth/RBACGate";

const LABEL_FIELDS: { key: keyof AppConfig; label: string; hint: string }[] = [
  { key: "case_label",      label: "Case (singular)",       hint: "e.g. Case, Claim, Incident, Matter" },
  { key: "cases_label",     label: "Case (plural)",         hint: "e.g. Cases, Claims, Incidents" },
  { key: "task_label",      label: "Task (singular)",       hint: "e.g. Task, Action Item, Playbook Step" },
  { key: "tasks_label",     label: "Task (plural)",         hint: "e.g. Tasks, Action Items" },
  { key: "document_label",  label: "Document (singular)",   hint: "e.g. Document, Evidence, Policy Doc, Filing" },
  { key: "documents_label", label: "Document (plural)",     hint: "e.g. Documents, Evidence, Policy Docs" },
  { key: "analyst_label",   label: "Analyst / Investigator",hint: "e.g. Analyst, Adjuster, Investigator, Associate" },
  { key: "risk_score_label",label: "Risk Score label",      hint: "e.g. Risk Score, Threat Score, Exposure Score" },
  { key: "risk_level_label",label: "Risk Level label",      hint: "e.g. Risk Level, Threat Level, Exposure Level" },
  { key: "sla_label",       label: "SLA label",             hint: "e.g. SLA, TAT, Deadline, Resolution Window" },
  { key: "comms_label",     label: "Communications label",  hint: "e.g. Communications, Correspondence, Contact Log" },
  { key: "timeline_label",  label: "Timeline label",        hint: "e.g. Timeline, Activity Log, Event Log" },
  { key: "note_label",      label: "Note (singular)",       hint: "e.g. Note, Remark, Memo" },
  { key: "notes_label",     label: "Note (plural)",         hint: "e.g. Notes, Remarks, Memos" },
];

export default function SettingsPage() {
  const current = useConfig();
  const { setConfig, fetch: refreshConfig } = useConfigStore();

  const [form, setForm] = useState<AppConfig>(current);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [resetting, setResetting] = useState(false);

  function handleField(key: keyof AppConfig, value: string) {
    setForm((prev) => ({ ...prev, [key]: value }));
    setSaved(false);
  }

  async function handleSave() {
    setSaving(true);
    setError(null);
    try {
      const result = await api.put<AppConfig>("/api/v1/config", form);
      setConfig(result);
      setForm(result);
      setSaved(true);
      await refreshConfig();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to save");
    } finally {
      setSaving(false);
    }
  }

  async function handleReset() {
    setResetting(true);
    try {
      const body = await api.get<AppConfig>("/api/v1/config/defaults");
      setForm(body);
      setSaved(false);
    } catch {
      setForm(DEFAULT_CONFIG);
    } finally {
      setResetting(false);
    }
  }

  return (
    <RBACGate
      allowedRoles={["admin", "super_admin"]}
      fallback={
        <div className="p-8 text-slate-400">
          You don&apos;t have permission to access this page.
        </div>
      }
    >
      <div className="p-8 max-w-3xl">
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-slate-100 mb-1">Workspace Settings</h1>
          <p className="text-slate-400 text-sm">
            Customize labels to match your team&apos;s vocabulary.
            Changes apply immediately across every page for all users.
          </p>
        </div>

        <section className="mb-8">
          <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest mb-4">
            Workspace Terminology
          </h2>

          <div className="grid grid-cols-2 gap-4">
            {LABEL_FIELDS.map(({ key, label, hint }) => (
              <div key={key}>
                <label className="block text-xs text-slate-400 mb-1">{label}</label>
                <input
                  type="text"
                  value={form[key] as string}
                  onChange={(e) => handleField(key, e.target.value)}
                  placeholder={hint}
                  className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-cyan-500 transition-colors"
                />
                <p className="text-slate-600 text-[10px] mt-0.5">{hint}</p>
              </div>
            ))}
          </div>
        </section>

        <div className="flex items-center gap-3 pt-2 border-t border-slate-700/50">
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-5 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm font-medium rounded transition-colors"
          >
            {saving ? "Saving…" : "Save Settings"}
          </button>
          <button
            onClick={handleReset}
            disabled={resetting}
            className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm transition-colors disabled:opacity-50"
          >
            {resetting ? "Loading…" : "Reset to Defaults"}
          </button>
          {saved && (
            <span className="text-green-400 text-sm">
              Saved — all users see updated labels immediately.
            </span>
          )}
          {error && <span className="text-red-400 text-sm">{error}</span>}
        </div>
      </div>
    </RBACGate>
  );
}
