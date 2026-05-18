-- Tenant/industry configuration: single-row JSONB store

-- +goose Up

CREATE TABLE system_config (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config      JSONB NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by  UUID REFERENCES users(id)
);

INSERT INTO system_config (config) VALUES ('{
  "app_name": "OpsOwl AI",
  "industry": "financial_services",
  "case_label": "Case",
  "cases_label": "Cases",
  "task_label": "Task",
  "tasks_label": "Tasks",
  "document_label": "Document",
  "documents_label": "Documents",
  "risk_score_label": "Risk Score",
  "risk_level_label": "Risk Level",
  "sla_label": "SLA",
  "comms_label": "Communications",
  "analyst_label": "Analyst",
  "timeline_label": "Timeline",
  "note_label": "Note",
  "notes_label": "Notes"
}');
