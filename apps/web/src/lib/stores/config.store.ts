"use client";

import { create } from "zustand";

export interface AppConfig {
  app_name: string;
  industry: string;
  case_label: string;
  cases_label: string;
  task_label: string;
  tasks_label: string;
  document_label: string;
  documents_label: string;
  risk_score_label: string;
  risk_level_label: string;
  sla_label: string;
  comms_label: string;
  analyst_label: string;
  timeline_label: string;
  note_label: string;
  notes_label: string;
  str_label: string;
  str_reference_label: string;
  str_regulator_label: string;
  str_law_reference: string;
  str_id_label: string;
  str_deadline_days: number;
  // Currency & reporting thresholds
  local_currency: string;
  currency_symbol: string;
  ctr_threshold: number;
  wire_threshold: number;
  // Jurisdiction-specific identity document types
  id_types: string[];
}

const INDIA_IDS = [
  "PASSPORT", "NATIONAL_ID", "TAX_ID", "DRIVING_LICENSE",
  "PAN", "AADHAAR_REF", "VOTER_ID", "OTHER",
];

const UAE_IDS = [
  "PASSPORT", "EMIRATES_ID", "UAE_TRADE_LICENSE",
  "TRN", "GCC_NATIONAL_ID", "RESIDENCY_VISA",
  "DRIVING_LICENSE", "OTHER",
];

export const DEFAULT_CONFIG: AppConfig = {
  app_name: "OpsOwl AI",
  industry: "financial_services",
  case_label: "Case",
  cases_label: "Cases",
  task_label: "Task",
  tasks_label: "Tasks",
  document_label: "Document",
  documents_label: "Documents",
  risk_score_label: "Risk Score",
  risk_level_label: "Risk Level",
  sla_label: "SLA",
  comms_label: "Communications",
  analyst_label: "Analyst",
  timeline_label: "Timeline",
  note_label: "Note",
  notes_label: "Notes",
  str_label: "STR",
  str_reference_label: "FIU Reference",
  str_regulator_label: "FIU-IND",
  str_law_reference: "PMLA 2002",
  str_id_label: "PAN",
  str_deadline_days: 30,
  local_currency: "INR",
  currency_symbol: "₹",
  ctr_threshold: 1000000,
  wire_threshold: 0,
  id_types: INDIA_IDS,
};

export const INDUSTRY_PRESETS: Record<string, Partial<AppConfig>> = {
  financial_services: {
    industry: "financial_services",
    case_label: "Case",
    cases_label: "Cases",
    task_label: "Task",
    tasks_label: "Tasks",
    document_label: "Document",
    documents_label: "Documents",
    risk_score_label: "Risk Score",
    risk_level_label: "Risk Level",
    sla_label: "SLA",
    comms_label: "Communications",
    analyst_label: "Analyst",
    timeline_label: "Timeline",
    note_label: "Note",
    notes_label: "Notes",
    str_label: "STR",
    str_reference_label: "FIU Reference",
    str_regulator_label: "FIU-IND",
    str_law_reference: "PMLA 2002",
    str_id_label: "PAN",
    str_deadline_days: 30,
    local_currency: "INR",
    currency_symbol: "₹",
    ctr_threshold: 1000000,
    wire_threshold: 0,
    id_types: INDIA_IDS,
  },
  insurance: {
    industry: "insurance",
    case_label: "Claim",
    cases_label: "Claims",
    task_label: "Action",
    tasks_label: "Actions",
    document_label: "Policy Doc",
    documents_label: "Policy Docs",
    risk_score_label: "Exposure Score",
    risk_level_label: "Exposure Level",
    sla_label: "SLA",
    comms_label: "Communications",
    analyst_label: "Adjuster",
    timeline_label: "Timeline",
    note_label: "Note",
    notes_label: "Notes",
    str_label: "SAR",
    str_reference_label: "Regulator Reference",
    str_regulator_label: "Regulator",
    str_law_reference: "",
    str_id_label: "ID Number",
    str_deadline_days: 30,
    local_currency: "INR",
    currency_symbol: "₹",
    ctr_threshold: 0,
    wire_threshold: 0,
    id_types: INDIA_IDS,
  },
  security_ops: {
    industry: "security_ops",
    case_label: "Incident",
    cases_label: "Incidents",
    task_label: "Playbook Step",
    tasks_label: "Playbook Steps",
    document_label: "Evidence",
    documents_label: "Evidence Items",
    risk_score_label: "Threat Score",
    risk_level_label: "Threat Level",
    sla_label: "SLA",
    comms_label: "Communications",
    analyst_label: "Analyst",
    timeline_label: "Timeline",
    note_label: "Note",
    notes_label: "Notes",
    str_label: "Report",
    str_reference_label: "Reference No.",
    str_regulator_label: "Authority",
    str_law_reference: "",
    str_id_label: "ID Number",
    str_deadline_days: 30,
    local_currency: "INR",
    currency_symbol: "₹",
    ctr_threshold: 0,
    wire_threshold: 0,
    id_types: ["PASSPORT", "NATIONAL_ID", "DRIVING_LICENSE", "OTHER"],
  },
  legal: {
    industry: "legal",
    case_label: "Matter",
    cases_label: "Matters",
    task_label: "Task",
    tasks_label: "Tasks",
    document_label: "Filing",
    documents_label: "Filings",
    risk_score_label: "Risk Score",
    risk_level_label: "Risk Level",
    sla_label: "SLA",
    comms_label: "Communications",
    analyst_label: "Associate",
    timeline_label: "Timeline",
    note_label: "Note",
    notes_label: "Notes",
    str_label: "Report",
    str_reference_label: "Reference No.",
    str_regulator_label: "Authority",
    str_law_reference: "",
    str_id_label: "ID Number",
    str_deadline_days: 30,
    local_currency: "INR",
    currency_symbol: "₹",
    ctr_threshold: 0,
    wire_threshold: 0,
    id_types: ["PASSPORT", "NATIONAL_ID", "TAX_ID", "DRIVING_LICENSE", "OTHER"],
  },
  hr: {
    industry: "hr",
    case_label: "Investigation",
    cases_label: "Investigations",
    task_label: "Action",
    tasks_label: "Actions",
    document_label: "Document",
    documents_label: "Documents",
    risk_score_label: "Severity Score",
    risk_level_label: "Severity Level",
    sla_label: "SLA",
    comms_label: "Communications",
    analyst_label: "Investigator",
    timeline_label: "Timeline",
    note_label: "Note",
    notes_label: "Notes",
    str_label: "Report",
    str_reference_label: "Reference No.",
    str_regulator_label: "Authority",
    str_law_reference: "",
    str_id_label: "ID Number",
    str_deadline_days: 30,
    local_currency: "INR",
    currency_symbol: "₹",
    ctr_threshold: 0,
    wire_threshold: 0,
    id_types: ["PASSPORT", "NATIONAL_ID", "DRIVING_LICENSE", "OTHER"],
  },
};

// UAE 2026 — Federal Decree-Law No. 20 of 2018 on AML/CFT
export const UAE_PRESET: Partial<AppConfig> = {
  industry: "financial_services",
  case_label: "Case",
  cases_label: "Cases",
  task_label: "Task",
  tasks_label: "Tasks",
  document_label: "Document",
  documents_label: "Documents",
  risk_score_label: "Risk Score",
  risk_level_label: "Risk Level",
  sla_label: "SLA",
  comms_label: "Communications",
  analyst_label: "Analyst",
  timeline_label: "Timeline",
  note_label: "Note",
  notes_label: "Notes",
  str_label: "STR",
  str_reference_label: "UAE FIU Reference",
  str_regulator_label: "UAE FIU",
  str_law_reference: "Federal Decree-Law No. 20 of 2018",
  str_id_label: "Emirates ID",
  str_deadline_days: 30,
  local_currency: "AED",
  currency_symbol: "AED",
  ctr_threshold: 55000,    // CBUAE cash transaction report threshold
  wire_threshold: 3675,    // USD 1,000 at 3.6725 peg — wire record-keeping
  id_types: UAE_IDS,
};

interface ConfigState {
  config: AppConfig;
  loaded: boolean;
  fetching: boolean;
  fetch: () => Promise<void>;
  setConfig: (cfg: AppConfig) => void;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const useConfigStore = create<ConfigState>((set, get) => ({
  config: DEFAULT_CONFIG,
  loaded: false,
  fetching: false,

  fetch: async () => {
    if (get().fetching || get().loaded) return;
    set({ fetching: true });
    try {
      const res = await fetch(`${API_BASE}/api/v1/config`);
      if (res.ok) {
        const body = await res.json();
        const raw = (body.data ?? body) as Partial<AppConfig>;
        // Merge with defaults so new fields always have values even on old DB rows
        const cfg: AppConfig = { ...DEFAULT_CONFIG, ...raw };
        // Ensure id_types is always a non-empty array
        if (!Array.isArray(cfg.id_types) || cfg.id_types.length === 0) {
          cfg.id_types = DEFAULT_CONFIG.id_types;
        }
        set({ config: cfg, loaded: true });
      } else {
        set({ loaded: true });
      }
    } catch {
      set({ loaded: true });
    } finally {
      set({ fetching: false });
    }
  },

  setConfig: (cfg) => set({ config: cfg }),
}));
