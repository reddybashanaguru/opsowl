"use client";

import { useEffect, useRef, useState } from "react";
import { useConfigStore, DEFAULT_CONFIG, type AppConfig } from "@/lib/stores/config.store";
import { api, APIError } from "@/lib/api";
import { RBACGate } from "@/components/auth/RBACGate";
import { useMutation, useQuery } from "@tanstack/react-query";
import { QRCodeSVG } from "qrcode.react";

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

// Returns regulatory filing field definitions with hints derived from the active config.
// Avoids any reference to other jurisdictions — each hint reflects THIS deployment only.
function getRegulatoryFields(config: AppConfig): { key: keyof AppConfig; label: string; hint: string }[] {
  return [
    { key: "str_label",           label: "Filing type short name",    hint: "Short name for suspicious transaction reports (e.g. STR, SAR, CTR)" },
    { key: "str_reference_label", label: "Regulator reference label", hint: `Reference number label used by ${config.str_regulator_label || "your regulator"}` },
    { key: "str_regulator_label", label: "Regulator / Authority",     hint: "The FIU or competent authority that receives filings" },
    { key: "str_law_reference",   label: "Applicable law",            hint: "Primary AML legislation — leave blank if not applicable" },
    { key: "str_id_label",        label: "Subject ID field label",    hint: "Primary identity document type used in STR filings" },
  ];
}

// Derives a country flag emoji from any ISO 3166-1 alpha-2 code algorithmically —
// no hardcoded country list needed, works for all 250 countries and "EU".
function countryFlag(code: string): string {
  if (!code) return "🌐";
  const upper = code.toUpperCase();
  if (upper === "EU") return "🇪🇺";
  if (upper.length !== 2) return "🌐";
  return String.fromCodePoint(...[...upper].map((c) => 127397 + c.charCodeAt(0)));
}

// Uses the browser's built-in Intl.DisplayNames to resolve any country code to
// its English name — no hardcoded lookup table, covers all ISO 3166-1 countries.
function countryName(code: string): string {
  if (!code) return "—";
  try {
    const names = new Intl.DisplayNames(["en"], { type: "region" });
    return names.of(code.toUpperCase()) ?? code.toUpperCase();
  } catch {
    return code.toUpperCase();
  }
}

function jurisdictionLabel(j: string): string {
  if (j === "eu") return "European Union";
  if (j === "uae") return "UAE";
  return j ? j.charAt(0).toUpperCase() + j.slice(1) : "—";
}

// ── Deployment Info Section (always shown, read-only) ───────────────────────
function DeploymentInfoSection({ config }: { config: AppConfig }) {
  // Flag: use EU flag for EU jurisdiction regardless of member-state country code;
  // otherwise derive from the country code (works for any ISO 3166-1 alpha-2).
  const flag = config.jurisdiction === "eu" ? "🇪🇺" : countryFlag(config.country);
  const jLabel = jurisdictionLabel(config.jurisdiction);
  const cName = countryName(config.country);
  // Derive regulatory display from the config values themselves — no hardcoded map.
  const regulatory = [config.str_law_reference, config.str_regulator_label].filter(Boolean).join(" · ") || "—";

  return (
    <section className="mb-8 p-6 bg-[#0e1929] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04]">
      <div className="flex items-center gap-2 mb-4">
        <span className="text-lg">{flag}</span>
        <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest">
          Deployment Jurisdiction
        </h2>
        <span className="ml-auto text-[10px] px-2 py-0.5 bg-slate-700 text-slate-400 rounded-full">Read-only</span>
      </div>
      <p className="text-slate-500 text-xs mb-4">
        Set via <span className="font-mono text-slate-400">DEPLOYMENT_JURISDICTION</span> and{" "}
        <span className="font-mono text-slate-400">DEPLOYMENT_COUNTRY</span> environment variables.
        Change requires redeployment.
      </p>
      <div className="grid grid-cols-2 gap-x-8 gap-y-2 text-xs max-w-lg">
        <span className="text-slate-500">Jurisdiction</span>
        <span className="text-slate-200 font-medium">{jLabel}</span>
        <span className="text-slate-500">Country</span>
        <span className="text-slate-200 font-medium">{cName}</span>
        <span className="text-slate-500">Regulatory framework</span>
        <span className="text-slate-200">{regulatory}</span>
        <span className="text-slate-500">Record retention</span>
        <span className="text-slate-200">{config.retention_years} years</span>
        {config.ubo_enabled && (
          <>
            <span className="text-slate-500">UBO threshold</span>
            <span className="text-slate-200">{config.ubo_threshold_pct}% beneficial ownership</span>
          </>
        )}
      </div>
      <div className="flex flex-wrap gap-2 mt-4">
        {config.goaml_enabled && (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-blue-500/10 border border-blue-500/30 text-blue-400 text-[11px]">
            <span className="w-1.5 h-1.5 rounded-full bg-blue-400 inline-block" />
            goAML XML reporting
          </span>
        )}
        {config.ubo_enabled && (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-violet-500/10 border border-violet-500/30 text-violet-400 text-[11px]">
            <span className="w-1.5 h-1.5 rounded-full bg-violet-400 inline-block" />
            UBO verification required
          </span>
        )}
        {config.gdpr_enabled && (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-[11px]">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 inline-block" />
            GDPR erasure workflow
          </span>
        )}
      </div>
    </section>
  );
}

// ── goAML Section (shown when goaml_enabled is true) ──────────────────────
function GoAMLSection({ config }: { config: AppConfig }) {
  const regulator = config.str_regulator_label || "your regulator";
  return (
    <section className="mb-8 p-6 bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04]">
      <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest mb-1">
        goAML Reporting
      </h2>
      <p className="text-slate-500 text-xs mb-5">
        goAML XML is the UNODC-standard FIU submission format required by{" "}
        {regulator}. Configure your entity registration details for automated STR export.
      </p>
      <div className="grid grid-cols-2 gap-4 max-w-lg">
        <div>
          <label className="block text-xs text-slate-400 mb-1">Reporting entity ID</label>
          <div className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-1.5 text-sm text-slate-500 cursor-not-allowed">
            Set via <span className="font-mono text-slate-400">GOAML_ENTITY_ID</span>
          </div>
          <p className="text-slate-600 text-[10px] mt-0.5">Assigned by {regulator} after entity registration</p>
        </div>
        <div>
          <label className="block text-xs text-slate-400 mb-1">FIU submission portal</label>
          <div className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-1.5 text-sm text-slate-500 cursor-not-allowed">
            Set via <span className="font-mono text-slate-400">GOAML_PORTAL_URL</span>
          </div>
          <p className="text-slate-600 text-[10px] mt-0.5">
            goAML portal URL provided by {regulator}
          </p>
        </div>
      </div>
    </section>
  );
}

// ── Beneficial Ownership Section (shown when ubo_enabled is true) ────────────
function BeneficialOwnershipSection({ config }: { config: AppConfig }) {
  const legalBasis = config.str_law_reference || "applicable AML legislation";
  return (
    <section className="mb-8 p-6 bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04]">
      <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest mb-1">
        Beneficial Ownership (UBO)
      </h2>
      <p className="text-slate-500 text-xs mb-5">
        Beneficial owner verification is required for subjects with {config.ubo_threshold_pct}% or more
        control or ownership. OpsOwl flags cases requiring UBO verification before closure.
      </p>
      <div className="grid grid-cols-2 gap-x-8 gap-y-2 text-xs max-w-lg">
        <span className="text-slate-500">Control threshold</span>
        <span className="text-slate-200 font-mono">{config.ubo_threshold_pct}%</span>
        <span className="text-slate-500">Legal basis</span>
        <span className="text-slate-200">{legalBasis} (UBO provisions)</span>
        <span className="text-slate-500">Threshold configured via</span>
        <span className="text-slate-400 font-mono">UBO_THRESHOLD_PCT</span>
      </div>
    </section>
  );
}

// ── GDPR Section (shown when gdpr_enabled is true) ─────────────────────────
function GDPRSection({ config }: { config: AppConfig }) {
  const amlLaw = config.str_law_reference || "applicable AML legislation";
  return (
    <section className="mb-8 p-6 bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04]">
      <div className="flex items-start justify-between mb-1">
        <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest">
          GDPR Compliance
        </h2>
        <a
          href="/settings/gdpr"
          className="px-3 py-1 bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-medium rounded transition-colors"
        >
          Manage DSAR Requests →
        </a>
      </div>
      <p className="text-slate-500 text-xs mb-5">
        GDPR right-to-erasure conflicts with AML record retention obligations.
        OpsOwl resolves this via anonymisation — personal data is replaced with pseudonyms
        while preserving the audit trail and transaction record.
      </p>
      <div className="grid grid-cols-2 gap-x-8 gap-y-2 text-xs max-w-lg">
        <span className="text-slate-500">Right-to-erasure handling</span>
        <span className="text-slate-200">Anonymisation (not deletion)</span>
        <span className="text-slate-500">AML retention override</span>
        <span className="text-slate-200">{config.retention_years} years from case close</span>
        <span className="text-slate-500">Legal basis</span>
        <span className="text-slate-200">GDPR Art. 17(3)(b) vs. {amlLaw} (erasure exemption)</span>
        <span className="text-slate-500">DPO name</span>
        <span className="text-slate-400 font-mono">Set via DPO_NAME</span>
        <span className="text-slate-500">DPO contact</span>
        <span className="text-slate-400 font-mono">Set via DPO_EMAIL</span>
      </div>
    </section>
  );
}

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
    <section className="mb-8 p-6 bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04]">
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
          <div className="p-4 bg-slate-900/60 rounded-lg space-y-4">
            <p className="text-slate-300 text-sm font-medium">1. Scan this QR code with your authenticator app</p>
            <div className="flex items-start gap-6">
              <div className="p-3 bg-white rounded-xl flex-shrink-0">
                <QRCodeSVG
                  value={enrollData.otpauth_url}
                  size={160}
                  level="M"
                  includeMargin={false}
                />
              </div>
              <div className="space-y-3 min-w-0">
                <p className="text-slate-400 text-xs leading-relaxed">
                  Open Google Authenticator, Authy, or any TOTP app. Tap the <strong className="text-slate-300">+</strong> button and scan the QR code.
                </p>
                <div>
                  <p className="text-slate-500 text-xs mb-1">Can&apos;t scan? Enter this key manually:</p>
                  <div className="flex items-center gap-2">
                    <code className="font-mono text-xs text-cyan-400 bg-slate-800 rounded px-2 py-1 break-all">
                      {enrollData.secret}
                    </code>
                    <button
                      type="button"
                      onClick={() => navigator.clipboard.writeText(enrollData.secret)}
                      className="text-xs text-slate-500 hover:text-slate-300 transition-colors flex-shrink-0"
                      title="Copy secret"
                    >
                      Copy
                    </button>
                  </div>
                </div>
              </div>
            </div>
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

// ─── AI Configuration Section ────────────────────────────────────────────────

function AIConfigSection() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["ai-mode"],
    queryFn: () => api.get<{ provider: string; mode: string; model?: string }>("/api/v1/system/ai-mode"),
    staleTime: 60_000,
  });

  const isAirGapped = data?.mode === "air-gapped";

  return (
    <section className="mb-8 p-6 bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04]">
      <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest mb-1">
        AI Configuration
      </h2>
      <p className="text-slate-500 text-xs mb-5">
        Shows the active AI inference mode for this deployment. Change via environment variables — see <span className="font-mono text-slate-400">docs/AI_DATA_POLICY.md</span>.
      </p>

      {isLoading && (
        <div className="flex items-center gap-2 text-slate-500 text-sm">
          <div className="w-3 h-3 rounded-full border-2 border-slate-500 border-t-transparent animate-spin" />
          Checking AI service…
        </div>
      )}

      {isError && (
        <div className="flex items-center gap-2 text-amber-400 text-sm">
          <span className="text-base">⚠</span>
          AI service unreachable — cannot determine mode.
        </div>
      )}

      {data && (
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-3">
            {isAirGapped ? (
              <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-green-500/15 border border-green-500/40 text-green-400 text-xs font-semibold tracking-wider">
                <span className="w-1.5 h-1.5 rounded-full bg-green-400 inline-block" />
                Air-Gapped (Local AI)
              </span>
            ) : (
              <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-amber-500/15 border border-amber-500/40 text-amber-400 text-xs font-semibold tracking-wider">
                <span className="w-1.5 h-1.5 rounded-full bg-amber-400 inline-block" />
                Cloud AI — {data.provider.charAt(0).toUpperCase() + data.provider.slice(1)}
              </span>
            )}
          </div>

          <div className="grid grid-cols-2 gap-x-8 gap-y-2 text-xs max-w-sm">
            <span className="text-slate-500">Provider</span>
            <span className="text-slate-300 font-mono">{data.provider}</span>
            {data.model && (
              <>
                <span className="text-slate-500">Model</span>
                <span className="text-slate-300 font-mono">{data.model}</span>
              </>
            )}
            <span className="text-slate-500">Data leaves network</span>
            <span className={isAirGapped ? "text-green-400" : "text-amber-400"}>
              {isAirGapped ? "No" : "Yes — TLS to provider"}
            </span>
            <span className="text-slate-500">RBI recommended</span>
            <span className={isAirGapped ? "text-green-400" : "text-slate-400"}>
              {isAirGapped ? "Yes" : "Use Air-Gapped for regulated clients"}
            </span>
          </div>

          {!isAirGapped && (
            <p className="text-amber-400/80 text-xs leading-relaxed max-w-md border border-amber-500/20 bg-amber-500/5 rounded-lg px-4 py-3">
              Case documents and copilot queries are processed on {data.provider} infrastructure. Ensure a signed Data Processing Agreement (DPA) is in place. For RBI-regulated clients, set <span className="font-mono">DEFAULT_PROVIDER=ollama</span>.
            </p>
          )}
        </div>
      )}
    </section>
  );
}

// ─── Change Password Section ──────────────────────────────────────────────────

function ChangePasswordSection() {
  const [currentPwd, setCurrentPwd] = useState("");
  const [newPwd, setNewPwd] = useState("");
  const [confirmPwd, setConfirmPwd] = useState("");
  const [showCurrent, setShowCurrent] = useState(false);
  const [showNew, setShowNew] = useState(false);

  const mutation = useMutation({
    mutationFn: () =>
      api.put("/api/v1/auth/password", {
        current_password: currentPwd,
        new_password: newPwd,
      }),
    onSuccess: () => {
      setCurrentPwd("");
      setNewPwd("");
      setConfirmPwd("");
    },
  });

  const mismatch = confirmPwd.length > 0 && newPwd !== confirmPwd;
  const tooShort = newPwd.length > 0 && newPwd.length < 8;
  const canSubmit = currentPwd.length > 0 && newPwd.length >= 8 && newPwd === confirmPwd && !mutation.isPending;

  return (
    <section className="mb-8 p-6 bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04]">
      <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest mb-1">
        Change Password
      </h2>
      <p className="text-slate-500 text-xs mb-5">Update your account password. You will remain signed in after the change.</p>

      {mutation.isSuccess && (
        <div className="mb-4 px-4 py-3 rounded-lg text-sm bg-green-500/10 border border-green-500/30 text-green-400">
          Password updated successfully.
        </div>
      )}
      {mutation.isError && (
        <div className="mb-4 px-4 py-3 rounded-lg text-sm bg-red-500/10 border border-red-500/30 text-red-400">
          {mutation.error instanceof APIError ? mutation.error.message : "Failed to change password."}
        </div>
      )}

      <div className="space-y-4 max-w-sm">
        <div>
          <label className="block text-xs text-slate-400 mb-1">Current Password</label>
          <div className="relative">
            <input
              type={showCurrent ? "text" : "password"}
              value={currentPwd}
              onChange={(e) => setCurrentPwd(e.target.value)}
              placeholder="••••••••"
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 pr-14 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none focus:border-cyan-500 transition-colors"
            />
            <button
              type="button"
              onClick={() => setShowCurrent((v) => !v)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 text-xs"
            >
              {showCurrent ? "Hide" : "Show"}
            </button>
          </div>
        </div>

        <div>
          <label className="block text-xs text-slate-400 mb-1">New Password</label>
          <div className="relative">
            <input
              type={showNew ? "text" : "password"}
              value={newPwd}
              onChange={(e) => setNewPwd(e.target.value)}
              placeholder="Min 8 characters"
              className={`w-full bg-slate-900 border rounded px-3 py-2 pr-14 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none transition-colors ${
                tooShort ? "border-red-500/60 focus:border-red-500" : "border-slate-600 focus:border-cyan-500"
              }`}
            />
            <button
              type="button"
              onClick={() => setShowNew((v) => !v)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 text-xs"
            >
              {showNew ? "Hide" : "Show"}
            </button>
          </div>
          {tooShort && <p className="text-red-400 text-[10px] mt-1">Minimum 8 characters required.</p>}
        </div>

        <div>
          <label className="block text-xs text-slate-400 mb-1">Confirm New Password</label>
          <input
            type="password"
            value={confirmPwd}
            onChange={(e) => setConfirmPwd(e.target.value)}
            placeholder="Re-enter new password"
            className={`w-full bg-slate-900 border rounded px-3 py-2 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none transition-colors ${
              mismatch ? "border-red-500/60 focus:border-red-500" : "border-slate-600 focus:border-cyan-500"
            }`}
          />
          {mismatch && <p className="text-red-400 text-[10px] mt-1">Passwords do not match.</p>}
        </div>

        <button
          onClick={() => mutation.mutate()}
          disabled={!canSubmit}
          className="px-5 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-sm font-medium rounded transition-colors"
        >
          {mutation.isPending ? "Updating…" : "Update Password"}
        </button>
      </div>
    </section>
  );
}

export default function SettingsPage() {
  const { config, loaded, setConfig, fetch: refreshConfig } = useConfigStore();

  const [form, setForm] = useState<AppConfig>(config);
  const syncedRef = useRef(loaded);

  // Trigger initial fetch if not yet loaded
  useEffect(() => {
    if (!loaded) refreshConfig();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Sync form once when config first loads (handles the case where
  // the store was empty at mount time)
  useEffect(() => {
    if (loaded && !syncedRef.current) {
      syncedRef.current = true;
      setForm(config);
    }
  }, [loaded, config]);

  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [resetting, setResetting] = useState(false);

  function handleField(key: keyof AppConfig, value: string | number | string[]) {
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
    <div className="p-8">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-slate-100 mb-1">Settings</h1>
        <p className="text-slate-400 text-sm">
          Manage your security settings and workspace configuration.
        </p>
      </div>

      {/* MFA — available to all authenticated users */}
      <MFASection />

      {/* Change Password — available to all authenticated users */}
      <ChangePasswordSection />

      {/* AI Configuration — admin only */}
      <RBACGate allowedRoles={["admin", "super_admin"]} fallback={null}>
        <AIConfigSection />
      </RBACGate>

      {/* Workspace settings — admin only */}
      <RBACGate
        allowedRoles={["admin", "super_admin"]}
        fallback={null}
      >
        {/* ── Deployment Jurisdiction Info — always shown ── */}
        <DeploymentInfoSection config={config} />

        {/* ── goAML — UAE + EU deployments ── */}
        {(config.goaml_enabled) && (
          <GoAMLSection config={config} />
        )}

        {/* ── Beneficial Ownership — EU only ── */}
        {config.ubo_enabled && (
          <BeneficialOwnershipSection config={config} />
        )}

        {/* ── GDPR — EU only ── */}
        {config.gdpr_enabled && (
          <GDPRSection config={config} />
        )}

        {/* ── AI Act Compliance — EU only ── */}
        {config.gdpr_enabled && (
          <section className="mb-8 p-6 bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04]">
            <div className="flex items-start justify-between mb-1">
              <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest">
                AI Act Compliance
              </h2>
              <a
                href="/settings/ai-act"
                className="px-3 py-1 bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-medium rounded transition-colors"
              >
                View AI Act Dashboard →
              </a>
            </div>
            <p className="text-slate-500 text-xs mt-1">
              EU Regulation 2024/1689 · Art. 13 transparency + Art. 14 human oversight for high-risk AI in financial crime detection.
              Conformity checklist, pending review queue, and CSV export for regulator submission.
            </p>
          </section>
        )}

        {/* ── Sanctions — EU + UAE deployments ── */}
        {(config.gdpr_enabled || config.goaml_enabled) && (
          <section className="mb-8 p-6 bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04]">
            <div className="flex items-start justify-between mb-1">
              <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest">
                Sanctions & PEP Screening
              </h2>
              <a
                href="/settings/sanctions"
                className="px-3 py-1 bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-medium rounded transition-colors"
              >
                Manage Sanctions Lists →
              </a>
            </div>
            <p className="text-slate-500 text-xs mt-1">
              EU Consolidated List · UN Security Council · OFAC SDN — fuzzy name matching via pg_trgm.
              View list status, reload databases, and review open/confirmed hits across all subjects.
            </p>
          </section>
        )}

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

        <section className="mb-8">
          <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-widest mb-1">
            Regulatory Filings
          </h2>
          <p className="text-slate-500 text-xs mb-4">
            Customise the regulatory filing terminology for this deployment. Changes apply immediately to all users.
          </p>
          <div className="grid grid-cols-2 gap-4">
            {getRegulatoryFields(config).map(({ key, label, hint }) => (
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

            {/* STR filing deadline */}
            <div>
              <label className="block text-xs text-slate-400 mb-1">Filing deadline (calendar days)</label>
              <input
                type="number"
                min={1}
                max={365}
                value={form.str_deadline_days}
                onChange={(e) => handleField("str_deadline_days", parseInt(e.target.value) || 30)}
                data-testid="field-str_deadline_days"
                className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:border-cyan-500 transition-colors"
              />
              <p className="text-slate-600 text-[10px] mt-0.5">Days from suspicion date to file as required by your regulatory framework</p>
            </div>

            {/* Currency */}
            <div>
              <label className="block text-xs text-slate-400 mb-1">Local currency (ISO 4217)</label>
              <input
                type="text"
                maxLength={3}
                value={form.local_currency}
                onChange={(e) => handleField("local_currency", e.target.value.toUpperCase())}
                placeholder="e.g. INR, AED, EUR, USD"
                data-testid="field-local_currency"
                className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-cyan-500 transition-colors font-mono uppercase"
              />
              <p className="text-slate-600 text-[10px] mt-0.5">Used for amount formatting and CTR threshold displays</p>
            </div>

            {/* CTR threshold */}
            <div>
              <label className="block text-xs text-slate-400 mb-1">CTR threshold (0 = disabled)</label>
              <input
                type="number"
                min={0}
                value={form.ctr_threshold}
                onChange={(e) => handleField("ctr_threshold", parseFloat(e.target.value) || 0)}
                placeholder="0"
                data-testid="field-ctr_threshold"
                className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-cyan-500 transition-colors"
              />
              <p className="text-slate-600 text-[10px] mt-0.5">Transactions at or above this amount show a CTR badge. Set to 0 to disable.</p>
            </div>

            {/* Wire threshold */}
            <div>
              <label className="block text-xs text-slate-400 mb-1">Wire record-keeping threshold (0 = disabled)</label>
              <input
                type="number"
                min={0}
                value={form.wire_threshold}
                onChange={(e) => handleField("wire_threshold", parseFloat(e.target.value) || 0)}
                placeholder="0"
                data-testid="field-wire_threshold"
                className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-cyan-500 transition-colors"
              />
              <p className="text-slate-600 text-[10px] mt-0.5">Wire transfers at or above this amount require record-keeping. Set to 0 to disable.</p>
            </div>

            {/* ID types */}
            <div className="col-span-2">
              <label className="block text-xs text-slate-400 mb-1">
                Accepted ID types <span className="text-slate-600 font-normal">(comma-separated)</span>
              </label>
              <input
                type="text"
                value={(form.id_types ?? []).join(", ")}
                onChange={(e) => handleField("id_types", e.target.value.split(",").map((s) => s.trim().toUpperCase()).filter(Boolean))}
                placeholder="PASSPORT, NATIONAL_ID, DRIVING_LICENSE, OTHER"
                data-testid="field-id_types"
                className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-cyan-500 transition-colors font-mono"
              />
              <p className="text-slate-600 text-[10px] mt-0.5">Drives the ID type dropdown on STR filing forms. Use ALL_CAPS identifiers.</p>
            </div>
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
