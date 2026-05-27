-- +goose Up
-- +goose StatementBegin

CREATE TABLE csv_imports (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  filename      TEXT         NOT NULL,
  total_rows    INT          NOT NULL DEFAULT 0,
  imported_rows INT          NOT NULL DEFAULT 0,
  failed_rows   INT          NOT NULL DEFAULT 0,
  status        TEXT         NOT NULL DEFAULT 'pending',
  created_by    UUID         REFERENCES users(id) ON DELETE SET NULL,
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE csv_import_rows (
  id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  import_id  UUID         NOT NULL REFERENCES csv_imports(id) ON DELETE CASCADE,
  row_number INT          NOT NULL,
  raw_data   JSONB        NOT NULL,
  status     TEXT         NOT NULL DEFAULT 'pending',
  case_id    UUID         REFERENCES cases(id) ON DELETE SET NULL,
  error      TEXT,
  created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_csv_import_rows_import ON csv_import_rows(import_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_csv_import_rows_import;
DROP TABLE IF EXISTS csv_import_rows;
DROP TABLE IF EXISTS csv_imports;
-- +goose StatementEnd
