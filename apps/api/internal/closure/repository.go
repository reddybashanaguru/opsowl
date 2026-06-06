package closure

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type closureRow struct {
	ID                 uuid.UUID
	CaseID             uuid.UUID
	Disposition        string
	Summary            string
	ReasonCode         string
	FiledSTR           bool
	STRID              uuid.NullUUID
	RiskRatingAtClose  sql.NullString
	SARConsidered      bool
	SARRejectionReason sql.NullString
	Status             string
	CreatedBy          uuid.UUID
	CreatedByName      string
	ReviewedBy         uuid.NullUUID
	ReviewedByName     sql.NullString
	ApprovedBy         uuid.NullUUID
	ApprovedByName     sql.NullString
	SubmittedAt        sql.NullTime
	ApprovedAt         sql.NullTime
	VoidedAt           sql.NullTime
	VoidedBy           uuid.NullUUID
	VoidedReason       sql.NullString
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (r *Repository) Create(ctx context.Context, caseID, createdBy uuid.UUID, req CreateClosureRequest) (closureRow, error) {
	disposition := req.Disposition
	if disposition == "" {
		disposition = "no_sar_warranted"
	}

	var strID uuid.NullUUID
	if req.STRFileID != nil {
		if id, err := uuid.Parse(*req.STRFileID); err == nil {
			strID = uuid.NullUUID{UUID: id, Valid: true}
		}
	}

	var id uuid.UUID
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO case_closures
			(case_id, disposition, summary, reason_code, filed_str, str_id,
			 risk_rating_at_close, sar_considered, sar_rejection_reason, created_by, status)
		VALUES ($1,$2::closure_disposition,$3,$4,$5,$6,$7,$8,$9,$10,'draft')
		RETURNING id, created_at, updated_at`,
		caseID, disposition, req.Summary, req.ReasonCode,
		req.FiledSTR, strID,
		nullStr(req.RiskRatingAtClose),
		req.SARConsidered, nullStr(req.SARRejectionReason),
		createdBy,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return closureRow{}, err
	}
	return r.GetByCaseID(ctx, caseID)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (closureRow, error) {
	return r.scan(r.db.QueryRowContext(ctx, selectClosureSQL+" WHERE cl.id = $1", id))
}

func (r *Repository) GetByCaseID(ctx context.Context, caseID uuid.UUID) (closureRow, error) {
	return r.scan(r.db.QueryRowContext(ctx,
		selectClosureSQL+` WHERE cl.case_id = $1 AND cl.status != 'voided' ORDER BY cl.created_at DESC LIMIT 1`, caseID))
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, req UpdateClosureRequest) error {
	var strID interface{}
	if req.STRFileID != nil {
		if uid, err := uuid.Parse(*req.STRFileID); err == nil {
			strID = uid
		}
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE case_closures SET
			disposition         = COALESCE($2::closure_disposition, disposition),
			summary             = COALESCE($3, summary),
			reason_code         = COALESCE($4, reason_code),
			filed_str           = COALESCE($5, filed_str),
			str_id              = COALESCE($6, str_id),
			risk_rating_at_close = COALESCE($7, risk_rating_at_close),
			sar_considered      = COALESCE($8, sar_considered),
			sar_rejection_reason = COALESCE($9, sar_rejection_reason)
		WHERE id = $1 AND status = 'draft'`,
		id,
		req.Disposition, req.Summary, req.ReasonCode, req.FiledSTR, strID,
		req.RiskRatingAtClose, req.SARConsidered, req.SARRejectionReason,
	)
	return err
}

func (r *Repository) UpdateStatus(ctx context.Context, id, actorID uuid.UUID, newStatus string, rejectionReason string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE case_closures SET
			status       = $2::closure_status,
			reviewed_by  = CASE WHEN $2 IN ('submitted','rejected') THEN $3 ELSE reviewed_by END,
			approved_by  = CASE WHEN $2 = 'approved' THEN $3 ELSE approved_by END,
			submitted_at = CASE WHEN $2 = 'submitted' THEN $4 ELSE submitted_at END,
			approved_at  = CASE WHEN $2 = 'approved'  THEN $4 ELSE approved_at  END
		WHERE id = $1`,
		id, newStatus, actorID, now,
	)
	return err
}

// Void marks a closure record as voided (called when case is reopened).
func (r *Repository) Void(ctx context.Context, id, actorID uuid.UUID, reason string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE case_closures SET
			status    = 'voided'::closure_status,
			voided_at = NOW(),
			voided_by = $2,
			voided_reason = $3
		WHERE id = $1`,
		id, actorID, reason,
	)
	return err
}

// CloseCaseRecord stamps closed_by / closed_at / closure_id on the cases table.
func (r *Repository) CloseCaseRecord(ctx context.Context, caseID, closureID, approverID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE cases SET
			status     = 'closed',
			closure_id = $2,
			closed_by  = $3,
			closed_at  = NOW()
		WHERE id = $1`,
		caseID, closureID, approverID,
	)
	return err
}

// ReopenCase resets the case status to 'open' and clears the closure reference.
func (r *Repository) ReopenCase(ctx context.Context, caseID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE cases SET
			status     = 'open',
			closure_id = NULL,
			closed_by  = NULL,
			closed_at  = NULL
		WHERE id = $1`,
		caseID,
	)
	return err
}

// ── Pre-closure checklist helpers ─────────────────────────────────────────────

// CountOpenTasks returns the number of open (non-terminal) tasks on a case.
func (r *Repository) CountOpenTasks(ctx context.Context, caseID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tasks
		WHERE case_id = $1 AND status NOT IN ('completed','cancelled') AND deleted_at IS NULL`, caseID,
	).Scan(&n)
	return n, err
}

// CountApprovedFindings returns how many approved findings exist on a case.
func (r *Repository) CountApprovedFindings(ctx context.Context, caseID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM case_findings
		WHERE case_id = $1 AND status = 'approved' AND deleted_at IS NULL`, caseID,
	).Scan(&n)
	return n, err
}

// GetCaseRiskLevel returns the most recent AI-assessed risk level for a case.
func (r *Repository) GetCaseRiskLevel(ctx context.Context, caseID uuid.UUID) (string, error) {
	var riskLevel sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT risk_level FROM case_ai_insights WHERE case_id = $1 ORDER BY analysed_at DESC LIMIT 1`, caseID,
	).Scan(&riskLevel)
	if err != nil || !riskLevel.Valid {
		return "unknown", nil
	}
	return riskLevel.String, nil
}

// ── Private helpers ───────────────────────────────────────────────────────────

const selectClosureSQL = `
	SELECT
		cl.id, cl.case_id,
		cl.disposition::TEXT, cl.summary, cl.reason_code,
		cl.filed_str, cl.str_id,
		cl.risk_rating_at_close, cl.sar_considered, cl.sar_rejection_reason,
		cl.status::TEXT,
		cl.created_by, COALESCE(cu.full_name,'') AS created_by_name,
		cl.reviewed_by, ru.full_name AS reviewed_by_name,
		cl.approved_by, au.full_name AS approved_by_name,
		cl.submitted_at, cl.approved_at,
		cl.voided_at, cl.voided_by, cl.voided_reason,
		cl.created_at, cl.updated_at
	FROM case_closures cl
	JOIN users cu ON cu.id = cl.created_by
	LEFT JOIN users ru ON ru.id = cl.reviewed_by
	LEFT JOIN users au ON au.id = cl.approved_by`

func (r *Repository) scan(row *sql.Row) (closureRow, error) {
	var c closureRow
	err := row.Scan(
		&c.ID, &c.CaseID,
		&c.Disposition, &c.Summary, &c.ReasonCode,
		&c.FiledSTR, &c.STRID,
		&c.RiskRatingAtClose, &c.SARConsidered, &c.SARRejectionReason,
		&c.Status,
		&c.CreatedBy, &c.CreatedByName,
		&c.ReviewedBy, &c.ReviewedByName,
		&c.ApprovedBy, &c.ApprovedByName,
		&c.SubmittedAt, &c.ApprovedAt,
		&c.VoidedAt, &c.VoidedBy, &c.VoidedReason,
		&c.CreatedAt, &c.UpdatedAt,
	)
	return c, err
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
