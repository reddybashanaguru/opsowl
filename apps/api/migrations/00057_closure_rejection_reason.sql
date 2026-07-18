-- +goose Up
-- Add rejection_reason to case_closures so investigators can see why their closure was rejected
-- without having to dig through the timeline tab.
ALTER TABLE case_closures
    ADD COLUMN IF NOT EXISTS rejection_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS rejected_by      UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS rejected_at      TIMESTAMPTZ;

-- +goose Down
ALTER TABLE case_closures
    DROP COLUMN IF EXISTS rejection_reason,
    DROP COLUMN IF EXISTS rejected_by,
    DROP COLUMN IF EXISTS rejected_at;
