-- name: CreateSTRFiling :one
INSERT INTO str_filings (
  case_id, org_id, report_type,
  subject, transactions, filing_entity,
  narrative, suspicion_basis, evidence_summary
) VALUES (
  @case_id, @org_id, @report_type,
  @subject, @transactions, @filing_entity,
  @narrative, @suspicion_basis, @evidence_summary
)
RETURNING *;

-- name: GetSTRFiling :one
SELECT
  f.*,
  submitter.full_name AS submitted_by_name,
  submitter.email     AS submitted_by_email,
  approver.full_name  AS approved_by_name,
  approver.email      AS approved_by_email
FROM str_filings f
LEFT JOIN users submitter ON submitter.id = f.submitted_by
LEFT JOIN users approver  ON approver.id  = f.approved_by
WHERE f.id = $1 AND f.org_id = $2;

-- name: ListSTRFilings :many
SELECT
  f.*,
  submitter.full_name AS submitted_by_name,
  approver.full_name  AS approved_by_name
FROM str_filings f
LEFT JOIN users submitter ON submitter.id = f.submitted_by
LEFT JOIN users approver  ON approver.id  = f.approved_by
WHERE f.case_id = @case_id AND f.org_id = @org_id
ORDER BY f.created_at DESC;

-- name: SubmitSTRFiling :one
UPDATE str_filings
   SET status       = 'submitted',
       submitted_by = @submitted_by,
       submitted_at = NOW(),
       updated_at   = NOW()
 WHERE id = @id AND org_id = @org_id AND status = 'draft'
RETURNING *;

-- name: ApproveSTRFiling :one
UPDATE str_filings
   SET status      = 'approved',
       approved_by = @approved_by,
       approved_at = NOW(),
       updated_at  = NOW()
 WHERE id = @id AND org_id = @org_id AND status = 'submitted'
   AND submitted_by <> @approved_by   -- app-layer maker-checker (mirrors DB constraint)
RETURNING *;

-- name: RejectSTRFiling :one
UPDATE str_filings
   SET status           = 'rejected',
       rejection_reason = @rejection_reason,
       updated_at       = NOW()
 WHERE id = @id AND org_id = @org_id AND status = 'submitted'
RETURNING *;

-- name: MarkSTRFiled :one
UPDATE str_filings
   SET status        = 'filed',
       filed_at      = NOW(),
       fiu_reference = @fiu_reference,
       updated_at    = NOW()
 WHERE id = @id AND org_id = @org_id AND status = 'approved'
RETURNING *;
