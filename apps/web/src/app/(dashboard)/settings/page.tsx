"use client";

import { useEffect, useState } from "react";
import { useConfig } from "@/lib/hooks/useConfig";
import { useConfigStore, DEFAULT_CONFIG, type AppConfig } from "@/lib/stores/config.store";
import { api, APIError } from "@/lib/api";
import { RBACGate } from "@/components/auth/RBACGate";
import { useMutation, useQuery } from "@tanstack/react-query";

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

interface MFAEnrollResponse {
  secret: string;
  otpauth_url: string;
  backup_codes: string[];
}

type MFAStep = "idle" | "enrolling" | "verifying" | "disabling";

function MFASection() {
  const [step, setStep] = useState<MFAStep>("idle");
  const [enrollData, setEnrollData] = useState<MFAEnrollResponse | null>(null);
  const [verifyCode, setVerifyCode] = useState("");
  const [disablePassword, setDisablePassword] = useState("");
  const [feedback, setFeedback] = useState<{ ok: boolean; msg: string } | null>(null);
  const [backupCodesVisible, setBackupCodesVisible] = useState(false);

  // Check if MFA is active by attempting to get active secret (we infer from a dedicated endpoint).
  // We use a simple approach: call enroll; if 409 returned, it's already active.
  const [mfaActive, setMfaActive] = useState<boolean | null>(null);

  const { data: mfaStatusData } = useQuery({
    queryKey: ["mfa-status"],
    queryFn: () => api.get<{ active: boolean }>("/api/v1/auth/mfa/status"),
  });

  useEffect(() => {
    if (mfaStatusData !== undefined) {
      setMfaActive(mfaStatusData.active);
    }
  }, [mfaStatusData]);

  const enrollMutation = useMutation({
    mutationFn: () => api.post<MFAEnrollResponse>("/api/v1/auth/mfa/enroll", {}),
    onSuccess: (data) => {
      setEnrollData(data);
      setStep("verifying");
      setFeedback(null);
    },
    onError: (e) => {
      if (e instanceof APIError && e.status === 409) {
        setFeedback({ ok: false, msg: "MFA is already active. Disable it first." });
      } else {
        setFeedback({ ok: false, msg: "Failed to start enrollment." });
      }
    },
  });

  const verifyMutation = useMutation({
    mutationFn: (code: string) =>
      api.post("/api/v1/auth/mfa/verify-enroll", { code }),
    onSuccess: () => {
      setMfaActive(true);
      setStep("idle");
      setEnrollData(null);
      setVerifyCode("");
      setFeedback({ ok: true, msg: "MFA enabled. Keep your backup codes safe." });
    },
    onError: (e) => {
      setFeedback({
        ok: false,
        msg: e instanceof APIError && e.status === 400
          ? "Invalid code — check your authenticator app and try again."
          : "Verification failed.",
      });
    },
  });

  const disableMutation = useMutation({
    mutationFn: (password: string) =>
      api.post("/api/v1/auth/mfa/disable", { password }),
    onSuccess: () => {
      setMfaActive(false);
      setStep("idle");
      setDisablePassword("");
      setFeedback({ ok: true, msg: "MFA disabled." });
    },
    onError: (e) => {
      setFeedback({
        ok: false,
        msg: e instanceof APIError && e.status === 400
          ? "Incorrect password."
          : "Failed to disable MFA.",
      });
    },
  });

  return (
    <section className="mb-8 p-6 bg-slate-800/50 border border-slate-700 rounded-xl">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest">
            Two-Factor Authentication
          </h2>
          <p className="text-slate-400 text-xs mt-0.5">
            Protect your account with a TOTP authenticator app (Google Authenticator, Authy, etc.)
          </p>
        </div>
        <div className={`px-2.5 py-1 rounded-full text-xs font-medium ${
          mfaActive === null ? "bg-slate-700 text-slate-400" :
          mfaActive ? "bg-green-500/20 text-green-400" : "bg-slate-700 text-slate-400"
        }`}>
          {mfaActive === null ? "Checking…" : mfaActive ? "Active" : "Disabled"}
        </div>
      </div>

      {feedback && (
        <div className={`mb-4 px-4 py-3 rounded-lg text-sm ${
          feedback.ok
            ? "bg-green-500/10 border border-green-500/30 text-green-400"
            : "bg-red-500/10 border border-red-500/30 text-red-400"
        }`}>
          {feedback.msg}
        </div>
      )}

      {step === "idle" && (
        <div className="flex gap-3">
          {!mfaActive ? (
            <button
              onClick={() => { setFeedback(null); enrollMutation.mutate(); }}
              disabled={enrollMutation.isPending || mfaActive === null}
              className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm rounded transition-colors"
            >
              {enrollMutation.isPending ? "Starting…" : "Enable MFA"}
            </button>
          ) : (
            <button
              onClick={() => { setStep("disabling"); setFeedback(null); }}
              className="px-4 py-2 bg-red-600/20 hover:bg-red-600/30 border border-red-500/30 text-red-400 text-sm rounded transition-colors"
            >
              Disable MFA
            </button>
          )}
        </div>
      )}

      {step === "verifying" && enrollData && (
        <div className="space-y-4">
          <div className="p-4 bg-slate-900/60 rounded-lg space-y-3">
            <p className="text-slate-300 text-sm font-medium">1. Scan QR code or enter secret manually</p>
            <div className="flex items-center gap-3">
              {/* QR code rendered via a data URI or external QR lib; show URL as fallback */}
              <div className="text-xs text-slate-400 font-mono break-all bg-slate-800 rounded px-3 py-2">
                {enrollData.otpauth_url}
              </div>
            </div>
            <p className="text-slate-500 text-xs">
              Manual secret: <span className="font-mono text-cyan-400">{enrollData.secret}</span>
            </p>
          </div>

          <div className="p-4 bg-slate-900/60 rounded-lg space-y-2">
            <div className="flex items-center justify-between">
              <p className="text-slate-300 text-sm font-medium">2. Save your backup codes</p>
              <button
                type="button"
                onClick={() => setBackupCodesVisible((v) => !v)}
                className="text-xs text-cyan-400 hover:text-cyan-300"
              >
                {backupCodesVisible ? "Hide" : "Show"} codes
              </button>
            </div>
            {backupCodesVisible && (
              <div className="grid grid-cols-4 gap-2 mt-2">
                {enrollData.backup_codes.map((c) => (
                  <span key={c} className="font-mono text-xs text-slate-300 bg-slate-800 rounded px-2 py-1 text-center">
                    {c}
                  </span>
                ))}
              </div>
            )}
            <p className="text-slate-500 text-xs">Each code can only be used once. Store them in a safe place.</p>
          </div>

          <div className="space-y-2">
            <p className="text-slate-300 text-sm font-medium">3. Enter the 6-digit code to confirm</p>
            <div className="flex gap-3">
              <input
                type="text"
                inputMode="numeric"
                maxLength={6}
                value={verifyCode}
                onChange={(e) => setVerifyCode(e.target.value.replace(/\D/g, ""))}
                placeholder="000000"
                className="w-36 bg-slate-900 border border-slate-600 rounded px-3 py-2 text-center font-mono text-white text-lg tracking-widest focus:outline-none focus:border-cyan-500"
              />
              <button
                onClick={() => verifyMutation.mutate(verifyCode)}
                disabled={verifyCode.length < 6 || verifyMutation.isPending}
                className="px-5 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm rounded transition-colors"
              >
                {verifyMutation.isPending ? "Verifying…" : "Activate MFA"}
              </button>
              <button
                onClick={() => { setStep("idle"); setEnrollData(null); setVerifyCode(""); setFeedback(null); }}
                className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {step === "disabling" && (
        <div className="space-y-3">
          <p className="text-slate-400 text-sm">
            Enter your current password to confirm disabling MFA.
          </p>
          <div className="flex gap-3">
            <input
              type="password"
              value={disablePassword}
              onChange={(e) => setDisablePassword(e.target.value)}
              placeholder="Current password"
              className="bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-cyan-500 w-60"
            />
            <button
              onClick={() => disableMutation.mutate(disablePassword)}
              disabled={!disablePassword || disableMutation.isPending}
              className="px-4 py-2 bg-red-600 hover:bg-red-500 disabled:opacity-50 text-white text-sm rounded transition-colors"
            >
              {disableMutation.isPending ? "Disabling…" : "Confirm Disable"}
            </button>
            <button
              onClick={() => { setStep("idle"); setDisablePassword(""); setFeedback(null); }}
              className="px-4 py-2 text-slate-400 hover:text-slate-200 text-sm transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </section>
  );
}

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
    <div className="p-8 max-w-3xl">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-slate-100 mb-1">Settings</h1>
        <p className="text-slate-400 text-sm">
          Manage your security settings and workspace configuration.
        </p>
      </div>

      {/* MFA — available to all authenticated users */}
      <MFASection />

      {/* Workspace settings — admin only */}
      <RBACGate
        allowedRoles={["admin", "super_admin"]}
        fallback={null}
      >
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
      </RBACGate>
    </div>
  );
}
