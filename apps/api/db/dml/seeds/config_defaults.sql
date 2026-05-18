-- Default application configuration seed
-- Run after migrations. Idempotent (INSERT ... ON CONFLICT DO NOTHING).

INSERT INTO app_config (key, value, description)
VALUES
    ('app_name',          '"OpsOwl AI"',          'Product display name — non-negotiable'),
    ('session_timeout',   '3600',                  'JWT session TTL in seconds'),
    ('max_file_size_mb',  '50',                    'Max document upload size in MB'),
    ('ai_model_default',  '"claude-opus-4-5"',     'Default LLM model for copilot features'),
    ('pagination_limit',  '20',                    'Default page size for list endpoints')
ON CONFLICT (key) DO NOTHING;
