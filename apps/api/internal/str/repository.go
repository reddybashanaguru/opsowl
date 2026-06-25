package str

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

var (
	ErrNotFound      = errors.New("str filing not found")
	ErrWrongStatus   = errors.New("filing is not in the required status for this action")
	ErrMakerChecker  = errors.New("approver must be different from the person who submitted the filing")
	ErrSelfApproval  = ErrMakerChecker
)

type filing struct {
	ID              uuid.UUID
	CaseID          uuid.UUID
	OrgID           uuid.UUID
	ReportType      string
	Status          string
	SubmittedBy     uuid.NullUUID
	ApprovedBy      uuid.NullUUID
	SubmittedAt     sql.NullTime
	ApprovedAt      sql.NullTime
	FiledAt         sql.NullTime
	FIUReference    sql.NullString
	RejectionReason sql.NullString
	SuspicionDate   sql.NullTime
	SuspicionBasis  string
	Narrative       string
	EvidenceSummary string
	FilingEntity    json.RawMessage
	Subject         json.RawMessage
	Transactions       json.RawMessage
	EUFATFIndicators   pq.StringArray
	SubmissionCountry  string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	SubmittedByName    sql.NullString
	SubmittedByEmail   sql.NullString
	ApprovedByName     sql.NullString
	ApprovedByEmail    sql.NullString
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, caseID, orgID uuid.UUID, req CreateDraftRequest) (*filing, error) {
	if req.ReportType == "" {
		req.ReportType = "STR"
	}
	if req.FilingEntity == nil {
		req.FilingEntity = json.RawMessage("{}")
	}
	if req.Subject == nil {
		req.Subject = json.RawMessage("{}")
	}
	if req.Transactions == nil {
		req.Transactions = json.RawMessage("[]")
	}

	var f filing
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO str_filings (
			case_id, org_id, report_type, suspicion_date, suspicion_basis, narrative,
			evidence_summary, filing_entity, subject, transactions
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, case_id, org_id, report_type, status,
		          submitted_by, approved_by, submitted_at, approved_at,
		          filed_at, fiu_reference, rejection_reason,
		          suspicion_date, suspicion_basis, narrative, evidence_summary,
		          filing_entity, subject, transactions, created_at, updated_at`,
		caseID, orgID, req.ReportType, req.SuspicionDate, req.SuspicionBasis, req.Narrative,
		req.EvidenceSummary, req.FilingEntity, req.Subject, req.Transactions,
	).Scan(
		&f.ID, &f.CaseID, &f.OrgID, &f.ReportType, &f.Status,
		&f.SubmittedBy, &f.ApprovedBy, &f.SubmittedAt, &f.ApprovedAt,
		&f.FiledAt, &f.FIUReference, &f.RejectionReason,
		&f.SuspicionDate, &f.SuspicionBasis, &f.Narrative, &f.EvidenceSummary,
		&f.FilingEntity, &f.Subject, &f.Transactions, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repository) GetByID(ctx context.Context, id, orgID uuid.UUID) (*filing, error) {
	var f filing
	err := r.db.QueryRowContext(ctx, `
		SELECT
			f.id, f.case_id, f.org_id, f.report_type, f.status,
			f.submitted_by, f.approved_by, f.submitted_at, f.approved_at,
			f.filed_at, f.fiu_reference, f.rejection_reason,
			f.suspicion_date, f.suspicion_basis, f.narrative, f.evidence_summary,
			f.filing_entity, f.subject, f.transactions,
			f.eu_fatf_indicators, f.submission_country,
			f.created_at, f.updated_at,
			s.full_name, s.email,
			a.full_name, a.email
		FROM str_filings f
		LEFT JOIN users s ON s.id = f.submitted_by
		LEFT JOIN users a ON a.id = f.approved_by
		WHERE f.id = $1 AND f.org_id = $2`,
		id, orgID,
	).Scan(
		&f.ID, &f.CaseID, &f.OrgID, &f.ReportType, &f.Status,
		&f.SubmittedBy, &f.ApprovedBy, &f.SubmittedAt, &f.ApprovedAt,
		&f.FiledAt, &f.FIUReference, &f.RejectionReason,
		&f.SuspicionDate, &f.SuspicionBasis, &f.Narrative, &f.EvidenceSummary,
		&f.FilingEntity, &f.Subject, &f.Transactions,
		&f.EUFATFIndicators, &f.SubmissionCountry,
		&f.CreatedAt, &f.UpdatedAt,
		&f.SubmittedByName, &f.SubmittedByEmail,
		&f.ApprovedByName, &f.ApprovedByEmail,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &f, err
}

// SaveEUIndicators saves manually selected EU FATF typology codes on a filing.
// Called when a compliance officer confirms/edits the auto-detected indicators.
func (r *Repository) SaveEUIndicators(ctx context.Context, id, orgID uuid.UUID, indicators []string, country string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE str_filings
		   SET eu_fatf_indicators = $3, submission_country = $4, updated_at = NOW()
		 WHERE id = $1 AND org_id = $2`,
		id, orgID, pq.Array(indicators), country,
	)
	return err
}

func (r *Repository) ListByCaseID(ctx context.Context, caseID, orgID uuid.UUID) ([]filing, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			f.id, f.case_id, f.org_id, f.report_type, f.status,
			f.submitted_by, f.approved_by, f.submitted_at, f.approved_at,
			f.filed_at, f.fiu_reference, f.rejection_reason,
			f.suspicion_date, f.suspicion_basis, f.narrative, f.evidence_summary,
			f.filing_entity, f.subject, f.transactions,
			f.eu_fatf_indicators, f.submission_country,
			f.created_at, f.updated_at,
			s.full_name, s.email,
			a.full_name, a.email
		FROM str_filings f
		LEFT JOIN users s ON s.id = f.submitted_by
		LEFT JOIN users a ON a.id = f.approved_by
		WHERE f.case_id = $1 AND f.org_id = $2
		ORDER BY f.created_at DESC`,
		caseID, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []filing
	for rows.Next() {
		var f filing
		if err := rows.Scan(
			&f.ID, &f.CaseID, &f.OrgID, &f.ReportType, &f.Status,
			&f.SubmittedBy, &f.ApprovedBy, &f.SubmittedAt, &f.ApprovedAt,
			&f.FiledAt, &f.FIUReference, &f.RejectionReason,
			&f.SuspicionDate, &f.SuspicionBasis, &f.Narrative, &f.EvidenceSummary,
			&f.FilingEntity, &f.Subject, &f.Transactions,
			&f.EUFATFIndicators, &f.SubmissionCountry,
			&f.CreatedAt, &f.UpdatedAt,
			&f.SubmittedByName, &f.SubmittedByEmail,
			&f.ApprovedByName, &f.ApprovedByEmail,
		); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repository) Submit(ctx context.Context, id, orgID, submitterID uuid.UUID) (*filing, error) {
	var f filing
	err := r.db.QueryRowContext(ctx, `
		UPDATE str_filings
		   SET status = 'submitted', submitted_by = $3, submitted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND org_id = $2 AND status = 'draft'
		RETURNING id, case_id, org_id, report_type, status,
		          submitted_by, approved_by, submitted_at, approved_at,
		          filed_at, fiu_reference, rejection_reason,
		          suspicion_date, suspicion_basis, narrative, evidence_summary,
		          filing_entity, subject, transactions, created_at, updated_at`,
		id, orgID, submitterID,
	).Scan(
		&f.ID, &f.CaseID, &f.OrgID, &f.ReportType, &f.Status,
		&f.SubmittedBy, &f.ApprovedBy, &f.SubmittedAt, &f.ApprovedAt,
		&f.FiledAt, &f.FIUReference, &f.RejectionReason,
		&f.SuspicionDate, &f.SuspicionBasis, &f.Narrative, &f.EvidenceSummary,
		&f.FilingEntity, &f.Subject, &f.Transactions, &f.CreatedAt, &f.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWrongStatus
	}
	return &f, err
}

func (r *Repository) Approve(ctx context.Context, id, orgID, approverID uuid.UUID) (*filing, error) {
	var f filing
	// The WHERE clause enforces maker-checker at the DB layer too:
	// submitted_by <> $3 means the UPDATE silently returns no rows if
	// the approver IS the submitter (mirrors the CHECK constraint).
	err := r.db.QueryRowContext(ctx, `
		UPDATE str_filings
		   SET status = 'approved', approved_by = $3, approved_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND org_id = $2 AND status = 'submitted'
		   AND submitted_by <> $3
		RETURNING id, case_id, org_id, report_type, status,
		          submitted_by, approved_by, submitted_at, approved_at,
		          filed_at, fiu_reference, rejection_reason,
		          suspicion_date, suspicion_basis, narrative, evidence_summary,
		          filing_entity, subject, transactions, created_at, updated_at`,
		id, orgID, approverID,
	).Scan(
		&f.ID, &f.CaseID, &f.OrgID, &f.ReportType, &f.Status,
		&f.SubmittedBy, &f.ApprovedBy, &f.SubmittedAt, &f.ApprovedAt,
		&f.FiledAt, &f.FIUReference, &f.RejectionReason,
		&f.SuspicionDate, &f.SuspicionBasis, &f.Narrative, &f.EvidenceSummary,
		&f.FilingEntity, &f.Subject, &f.Transactions, &f.CreatedAt, &f.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		// Could be wrong status OR self-approval — fetch to distinguish
		existing, fetchErr := r.GetByID(ctx, id, orgID)
		if fetchErr != nil {
			return nil, ErrNotFound
		}
		if existing.SubmittedBy.Valid && existing.SubmittedBy.UUID == approverID {
			return nil, ErrMakerChecker
		}
		return nil, ErrWrongStatus
	}
	return &f, err
}

func (r *Repository) Reject(ctx context.Context, id, orgID uuid.UUID, reason string) (*filing, error) {
	var f filing
	err := r.db.QueryRowContext(ctx, `
		UPDATE str_filings
		   SET status = 'rejected', rejection_reason = $3, updated_at = NOW()
		 WHERE id = $1 AND org_id = $2 AND status = 'submitted'
		RETURNING id, case_id, org_id, report_type, status,
		          submitted_by, approved_by, submitted_at, approved_at,
		          filed_at, fiu_reference, rejection_reason,
		          suspicion_date, suspicion_basis, narrative, evidence_summary,
		          filing_entity, subject, transactions, created_at, updated_at`,
		id, orgID, reason,
	).Scan(
		&f.ID, &f.CaseID, &f.OrgID, &f.ReportType, &f.Status,
		&f.SubmittedBy, &f.ApprovedBy, &f.SubmittedAt, &f.ApprovedAt,
		&f.FiledAt, &f.FIUReference, &f.RejectionReason,
		&f.SuspicionDate, &f.SuspicionBasis, &f.Narrative, &f.EvidenceSummary,
		&f.FilingEntity, &f.Subject, &f.Transactions, &f.CreatedAt, &f.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWrongStatus
	}
	return &f, err
}

func (r *Repository) MarkFiled(ctx context.Context, id, orgID uuid.UUID, fiuRef string) (*filing, error) {
	var f filing
	err := r.db.QueryRowContext(ctx, `
		UPDATE str_filings
		   SET status = 'filed', filed_at = NOW(), fiu_reference = $3, updated_at = NOW()
		 WHERE id = $1 AND org_id = $2 AND status = 'approved'
		RETURNING id, case_id, org_id, report_type, status,
		          submitted_by, approved_by, submitted_at, approved_at,
		          filed_at, fiu_reference, rejection_reason,
		          suspicion_date, suspicion_basis, narrative, evidence_summary,
		          filing_entity, subject, transactions, created_at, updated_at`,
		id, orgID, fiuRef,
	).Scan(
		&f.ID, &f.CaseID, &f.OrgID, &f.ReportType, &f.Status,
		&f.SubmittedBy, &f.ApprovedBy, &f.SubmittedAt, &f.ApprovedAt,
		&f.FiledAt, &f.FIUReference, &f.RejectionReason,
		&f.SuspicionDate, &f.SuspicionBasis, &f.Narrative, &f.EvidenceSummary,
		&f.FilingEntity, &f.Subject, &f.Transactions, &f.CreatedAt, &f.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWrongStatus
	}
	return &f, err
}

// DeadlineItem is a lightweight row returned by the deadlines endpoint.
type DeadlineItem struct {
	FilingID      string  `json:"filing_id"`
	CaseID        string  `json:"case_id"`
	CaseNumber    string  `json:"case_number"`
	SuspicionDate string  `json:"suspicion_date"`
	DeadlineDate  string  `json:"deadline_date"`
	DaysRemaining int     `json:"days_remaining"`
	IsOverdue     bool    `json:"is_overdue"`
	Status        string  `json:"status"`
	ReportType    string  `json:"report_type"`
}

// Deadlines returns STR filings with approaching or overdue deadlines.
// Only considers filings where suspicion_date is set and status is not 'filed'.
// windowDays: how many days ahead to look (default 14).
// deadlineDays: regulatory filing deadline from config (default 30).
func (r *Repository) Deadlines(ctx context.Context, orgID uuid.UUID, deadlineDays, windowDays int) ([]DeadlineItem, error) {
	if deadlineDays <= 0 {
		deadlineDays = 30
	}
	if windowDays <= 0 {
		windowDays = 14
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			f.id, f.case_id, c.case_number,
			f.suspicion_date,
			(f.suspicion_date + ($1 * INTERVAL '1 day'))::date AS deadline_date,
			EXTRACT(DAY FROM (f.suspicion_date + ($1 * INTERVAL '1 day')) - NOW())::int AS days_remaining,
			f.status, f.report_type
		FROM str_filings f
		JOIN cases c ON c.id = f.case_id AND c.deleted_at IS NULL
		WHERE f.org_id = $2
		  AND f.suspicion_date IS NOT NULL
		  AND f.status NOT IN ('filed')
		  AND (f.suspicion_date + ($1 * INTERVAL '1 day')) <= (NOW() + ($3 * INTERVAL '1 day'))
		ORDER BY deadline_date ASC
		LIMIT 100`,
		deadlineDays, orgID, windowDays,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DeadlineItem
	for rows.Next() {
		var it DeadlineItem
		var suspDate, dlDate sql.NullString
		var daysLeft sql.NullInt64
		if err := rows.Scan(
			&it.FilingID, &it.CaseID, &it.CaseNumber,
			&suspDate, &dlDate, &daysLeft,
			&it.Status, &it.ReportType,
		); err != nil {
			return nil, err
		}
		it.SuspicionDate = suspDate.String
		it.DeadlineDate = dlDate.String
		it.DaysRemaining = int(daysLeft.Int64)
		it.IsOverdue = !daysLeft.Valid || daysLeft.Int64 < 0
		out = append(out, it)
	}
	return out, rows.Err()
}

// toResponse maps a DB filing row to the API response shape.
// deadlineDays is sourced from system_config (str_deadline_days); defaults to 30 if zero.
func toResponse(f *filing, deadlineDays ...int) *FilingResponse {
	dlDays := 30
	if len(deadlineDays) > 0 && deadlineDays[0] > 0 {
		dlDays = deadlineDays[0]
	}

	r := &FilingResponse{
		ID:              f.ID,
		CaseID:          f.CaseID,
		ReportType:      f.ReportType,
		Status:          f.Status,
		SuspicionBasis:  f.SuspicionBasis,
		Narrative:       f.Narrative,
		EvidenceSummary: f.EvidenceSummary,
		FilingEntity:    f.FilingEntity,
		Subject:         f.Subject,
		Transactions:    f.Transactions,
		CreatedAt:       f.CreatedAt,
		UpdatedAt:       f.UpdatedAt,
	}

	if f.SuspicionDate.Valid {
		t := f.SuspicionDate.Time
		r.SuspicionDate = &t
		deadline := t.AddDate(0, 0, dlDays)
		r.DeadlineDays = int(time.Until(deadline).Hours() / 24)
		r.IsOverdue = time.Now().After(deadline)
	}
	if f.SubmittedBy.Valid {
		id := f.SubmittedBy.UUID
		r.SubmittedBy = &id
	}
	if f.ApprovedBy.Valid {
		id := f.ApprovedBy.UUID
		r.ApprovedBy = &id
	}
	if f.SubmittedAt.Valid {
		t := f.SubmittedAt.Time
		r.SubmittedAt = &t
	}
	if f.ApprovedAt.Valid {
		t := f.ApprovedAt.Time
		r.ApprovedAt = &t
	}
	if f.FiledAt.Valid {
		t := f.FiledAt.Time
		r.FiledAt = &t
	}
	if f.FIUReference.Valid {
		r.FIUReference = f.FIUReference.String
	}
	if f.RejectionReason.Valid {
		r.RejectionReason = f.RejectionReason.String
	}
	if f.SubmittedByName.Valid {
		r.SubmittedByName = f.SubmittedByName.String
	}
	if f.ApprovedByName.Valid {
		r.ApprovedByName = f.ApprovedByName.String
	}
	if len(f.EUFATFIndicators) > 0 {
		r.EUFATFIndicators = []string(f.EUFATFIndicators)
	}
	r.SubmissionCountry = f.SubmissionCountry
	return r
}
