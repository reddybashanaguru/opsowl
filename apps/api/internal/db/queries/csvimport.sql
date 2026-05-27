-- ── CSV Bulk Import ────────────────────────────────────────────────────────────

-- name: InsertCsvImport :one
INSERT INTO csv_imports (filename, total_rows, created_by)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetCsvImport :one
SELECT * FROM csv_imports WHERE id = $1;

-- name: ListCsvImports :many
SELECT * FROM csv_imports ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateCsvImportStatus :one
UPDATE csv_imports
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: IncrementImportCounts :one
UPDATE csv_imports
SET imported_rows = imported_rows + $2,
    failed_rows   = failed_rows   + $3,
    updated_at    = NOW()
WHERE id = $1
RETURNING *;

-- name: InsertCsvImportRow :one
INSERT INTO csv_import_rows (import_id, row_number, raw_data)
VALUES ($1, $2, $3)
RETURNING *;

-- name: MarkImportRowSuccess :one
UPDATE csv_import_rows
SET status = 'success', case_id = $2
WHERE id = $1
RETURNING *;

-- name: MarkImportRowFailed :one
UPDATE csv_import_rows
SET status = 'failed', error = $2
WHERE id = $1
RETURNING *;

-- name: ListImportRows :many
SELECT * FROM csv_import_rows
WHERE import_id = $1
  AND ($2::text IS NULL OR status = $2)
ORDER BY row_number ASC
LIMIT $3 OFFSET $4;

-- name: CountImportRows :one
SELECT COUNT(*) FROM csv_import_rows
WHERE import_id = $1
  AND ($2::text IS NULL OR status = $2);

-- name: GetPendingImportRows :many
SELECT * FROM csv_import_rows
WHERE import_id = $1 AND status = 'pending'
ORDER BY row_number ASC;
