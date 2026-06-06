package narrative

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

// ── Internal row type ─────────────────────────────────────────────────────────

type narrativeRow struct {
	ID               uuid.UUID
	CaseID           uuid.UUID
	NarrativeType    string
	Status           string
	Body             string
	BodyEdited       sql.NullString
	GeneratedBy      uuid.NullUUID
	GeneratedByName  sql.NullString
	AILogID          uuid.NullUUID
	ReviewedBy       uuid.NullUUID
	ReviewedByName   sql.NullString
	ReviewedAt       sql.NullTime
	RejectionReason  string
	ModelUsed        string
	PromptVersion    string
	FindingsIncluded int
	TxnsIncluded     int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

const selectNarrativeSQL = `
SELECT
  n.id, n.case_id, n.narrative_type, n.status, n.body, n.body_edited,
  n.generated_by, u1.full_name AS generated_by_name,
  n.ai_log_id,
  n.reviewed_by, u2.full_name AS reviewed_by_name, n.reviewed_at,
  n.rejection_reason, n.model_used, n.prompt_version,
  n.findings_included, n.txns_included,
  n.created_at, n.updated_at
FROM case_narratives n
LEFT JOIN users u1 ON u1.id = n.generated_by
LEFT JOIN users u2 ON u2.id = n.reviewed_by
`

func scanNarrative(row interface{ Scan(...any) error }) (narrativeRow, error) {
	var r narrativeRow
	err := row.Scan(
		&r.ID, &r.CaseID, &r.NarrativeType, &r.Status, &r.Body, &r.BodyEdited,
		&r.GeneratedBy, &r.GeneratedByName,
		&r.AILogID,
		&r.ReviewedBy, &r.ReviewedByName, &r.ReviewedAt,
		&r.RejectionReason, &r.ModelUsed, &r.PromptVersion,
		&r.FindingsIncluded, &r.TxnsIncluded,
		&r.CreatedAt, &r.UpdatedAt,
	)
	return r, err
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

func (r *Repository) Create(ctx context.Context, caseID, generatedBy uuid.UUID, aiLogID *uuid.UUID, body, narrativeType, modelUsed, promptVersion string, findingsUsed, txnsUsed int) (narrativeRow, error) {
	var aiLogNullUUID uuid.NullUUID
	if aiLogID != nil {
		aiLogNullUUID = uuid.NullUUID{UUID: *aiLogID, Valid: true}
	}

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO case_narratives
		  (case_id, narrative_type, status, body, generated_by, ai_log_id,
		   model_used, prompt_version, findings_included, txns_included)
		VALUES ($1,$2,'draft',$3,$4,$5,$6,$7,$8,$9)
		RETURNING id`,
		caseID, narrativeType, body, generatedBy, aiLogNullUUID,
		modelUsed, promptVersion, findingsUsed, txnsUsed,
	)
	var id uuid.UUID
	if err := row.Scan(&id); err != nil {
		return narrativeRow{}, err
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (narrativeRow, error) {
	row := r.db.QueryRowContext(ctx, selectNarrativeSQL+` WHERE n.id = $1`, id)
	return scanNarrative(row)
}

func (r *Repository) GetLatestForCase(ctx context.Context, caseID uuid.UUID, narrativeType string) (narrativeRow, error) {
	q := selectNarrativeSQL + ` WHERE n.case_id = $1`
	args := []any{caseID}
	if narrativeType != "" {
		q += ` AND n.narrative_type = $2`
		args = append(args, narrativeType)
	}
	q += ` ORDER BY n.created_at DESC LIMIT 1`
	row := r.db.QueryRowContext(ctx, q, args...)
	return scanNarrative(row)
}

func (r *Repository) ListForCase(ctx context.Context, caseID uuid.UUID) ([]narrativeRow, error) {
	rows, err := r.db.QueryContext(ctx,
		selectNarrativeSQL+` WHERE n.case_id = $1 ORDER BY n.created_at DESC`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []narrativeRow
	for rows.Next() {
		row, err := scanNarrative(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *Repository) UpdateBody(ctx context.Context, id uuid.UUID, editedBody string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE case_narratives
		SET body_edited = $2, status = 'draft', updated_at = NOW()
		WHERE id = $1`, id, editedBody)
	return err
}

func (r *Repository) UpdateStatus(ctx context.Context, id, reviewedBy uuid.UUID, status, reason string) error {
	var reviewedAt *time.Time
	if status == "approved" || status == "rejected" {
		now := time.Now()
		reviewedAt = &now
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE case_narratives
		SET status = $2, reviewed_by = $3, reviewed_at = $4,
		    rejection_reason = $5, updated_at = NOW()
		WHERE id = $1`, id, status, reviewedBy, reviewedAt, reason)
	return err
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM case_narratives WHERE id = $1`, id)
	return err
}

// ── Context gathering for AI ──────────────────────────────────────────────────

type caseContext struct {
	CaseNumber  string
	Title       string
	Description string
	RiskLevel   string
}

func (r *Repository) GetCaseContext(ctx context.Context, caseID uuid.UUID) (caseContext, error) {
	var cc caseContext
	err := r.db.QueryRowContext(ctx, `
		SELECT c.case_number, c.title, COALESCE(c.description, ''),
		       COALESCE(ci.risk_level, 'unknown')
		FROM cases c
		LEFT JOIN case_insights ci ON ci.case_id = c.id
		WHERE c.id = $1`, caseID).Scan(&cc.CaseNumber, &cc.Title, &cc.Description, &cc.RiskLevel)
	return cc, err
}

type findingContext struct {
	Title          string
	Category       string
	Severity       string
	Description    string
	NarrativeWeight string
}

func (r *Repository) GetApprovedFindings(ctx context.Context, caseID uuid.UUID) ([]findingContext, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT title, category::TEXT, severity::TEXT, description, narrative_weight::TEXT
		FROM case_findings
		WHERE case_id = $1 AND status = 'approved' AND deleted_at IS NULL
		ORDER BY
		  CASE severity::TEXT
		    WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4
		  END,
		  CASE narrative_weight::TEXT
		    WHEN 'primary' THEN 1 WHEN 'corroborating' THEN 2 ELSE 3
		  END
		LIMIT 30`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []findingContext
	for rows.Next() {
		var f findingContext
		if err := rows.Scan(&f.Title, &f.Category, &f.Severity, &f.Description, &f.NarrativeWeight); err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, rows.Err()
}

type txnContext struct {
	TxnRef    string
	TxnDate   time.Time
	Amount    float64
	Currency  string
	TxnType   string
	Channel   string
	FromName  string
	ToName    string
	Narration string
	IsFlagged bool
	FlagReason string
}

func (r *Repository) GetCaseTransactions(ctx context.Context, caseID uuid.UUID) ([]txnContext, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.txn_ref, t.txn_date, t.amount, t.currency, t.txn_type, t.channel,
		       COALESCE(fa.holder_name, t.from_name_raw, '') AS from_name,
		       COALESCE(ta.holder_name, t.to_name_raw, '') AS to_name,
		       t.narration, t.is_flagged, t.flag_reason
		FROM case_transactions ct
		JOIN transactions t ON t.id = ct.transaction_id
		LEFT JOIN accounts fa ON fa.id = t.from_account_id
		LEFT JOIN accounts ta ON ta.id = t.to_account_id
		WHERE ct.case_id = $1
		ORDER BY t.txn_date DESC, t.is_flagged DESC
		LIMIT 50`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []txnContext
	for rows.Next() {
		var t txnContext
		if err := rows.Scan(&t.TxnRef, &t.TxnDate, &t.Amount, &t.Currency, &t.TxnType, &t.Channel,
			&t.FromName, &t.ToName, &t.Narration, &t.IsFlagged, &t.FlagReason); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *Repository) GetSubjectNames(ctx context.Context, caseID uuid.UUID) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.canonical_name
		FROM case_subjects cs
		JOIN subjects s ON s.id = cs.subject_id
		WHERE cs.case_id = $1 AND cs.deleted_at IS NULL
		LIMIT 10`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}
