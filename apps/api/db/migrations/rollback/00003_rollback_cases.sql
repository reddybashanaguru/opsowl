-- +goose StatementBegin
DROP TABLE IF EXISTS case_activities;
DROP TABLE IF EXISTS case_watchers;
DROP TABLE IF EXISTS case_tags;
DROP TABLE IF EXISTS case_assignments;
DROP TABLE IF EXISTS cases;
DROP SEQUENCE IF EXISTS case_number_seq;
DROP TYPE IF EXISTS case_priority;
DROP TYPE IF EXISTS case_status;
-- +goose StatementEnd
