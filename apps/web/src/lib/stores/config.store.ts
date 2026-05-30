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
}

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
  str_regulator_label: "FIU",
  str_law_reference: "PMLA 2002",
  str_id_label: "PAN",
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
    str_regulator_label: "FIU",
    str_law_reference: "PMLA 2002",
    str_id_label: "PAN",
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
  },
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
        const cfg = (body.data ?? body) as AppConfig;
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
