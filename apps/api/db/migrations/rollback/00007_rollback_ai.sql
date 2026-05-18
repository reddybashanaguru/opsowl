
DROP INDEX IF EXISTS idx_ai_logs_action;
DROP INDEX IF EXISTS idx_ai_logs_user;
DROP INDEX IF EXISTS idx_ai_logs_review;
DROP INDEX IF EXISTS idx_ai_logs_case;
DROP TABLE IF EXISTS ai_logs;

DROP TRIGGER IF EXISTS ai_prompts_updated_at ON ai_prompts;
DROP TABLE IF EXISTS ai_prompts;
