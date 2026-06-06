package transactions

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ── Internal row types ────────────────────────────────────────────────────────

type accountRow struct {
	ID            uuid.UUID
	OrgID         uuid.UUID
	AccountNumber string
	AccountType   string
	BankName      string
	BankIFSC      string
	HolderName    string
	HolderType    string
	Currency      string
	Country       string
	SubjectID     uuid.NullUUID
	CreatedAt     time.Time
}

type txnRow struct {
	ID                  uuid.UUID
	OrgID               uuid.UUID
	TxnRef              string
	TxnDate             time.Time
	TxnTime             sql.NullString
	Amount              float64
	Currency            string
	TxnType             string
	Channel             string
	Status              string
	Description         string
	Narration           string
	FromAccountID       uuid.NullUUID
	ToAccountID         uuid.NullUUID
	FromAccountRaw      string
	ToAccountRaw        string
	FromNameRaw         string
	ToNameRaw           string
	IsFlagged           bool
	FlagReason          string
	RiskScore           int
	CounterpartyBank    string
	CounterpartyCountry string
	SourceSystem        string
	ExternalID          string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	// joined from case_transactions
	RelevanceNote       string
	// joined account details
	FromAccNum   sql.NullString
	FromAccType  sql.NullString
	FromAccBank  sql.NullString
	FromHolderName sql.NullString
	ToAccNum     sql.NullString
	ToAccType    sql.NullString
	ToAccBank    sql.NullString
	ToHolderName sql.NullString
}

// ── Account CRUD ──────────────────────────────────────────────────────────────

func (r *Repository) CreateAccount(ctx context.Context, orgID uuid.UUID, req CreateAccountRequest) (accountRow, error) {
	var subjectID *uuid.UUID
	if req.SubjectID != nil {
		id, err := uuid.Parse(*req.SubjectID)
		if err == nil {
			subjectID = &id
		}
	}

	currency := req.Currency
	if currency == "" {
		currency = "INR"
	}
	country := req.Country
	if country == "" {
		country = "IN"
	}

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO accounts (org_id, account_number, account_type, bank_name, bank_ifsc,
		                      holder_name, holder_type, currency, country, subject_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (org_id, account_number) DO UPDATE
		  SET bank_name = EXCLUDED.bank_name,
		      holder_name = EXCLUDED.holder_name,
		      updated_at = NOW()
		RETURNING id, org_id, account_number, account_type, bank_name, bank_ifsc,
		          holder_name, holder_type, currency, country, subject_id, created_at`,
		orgID, req.AccountNumber, req.AccountType, req.BankName, req.BankIFSC,
		req.HolderName, req.HolderType, currency, country, subjectID,
	)

	var acc accountRow
	err := row.Scan(
		&acc.ID, &acc.OrgID, &acc.AccountNumber, &acc.AccountType, &acc.BankName,
		&acc.BankIFSC, &acc.HolderName, &acc.HolderType, &acc.Currency, &acc.Country,
		&acc.SubjectID, &acc.CreatedAt,
	)
	return acc, err
}

func (r *Repository) GetAccountByNumber(ctx context.Context, orgID uuid.UUID, number string) (accountRow, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, org_id, account_number, account_type, bank_name, bank_ifsc,
		       holder_name, holder_type, currency, country, subject_id, created_at
		FROM accounts WHERE org_id = $1 AND account_number = $2`, orgID, number)

	var acc accountRow
	err := row.Scan(
		&acc.ID, &acc.OrgID, &acc.AccountNumber, &acc.AccountType, &acc.BankName,
		&acc.BankIFSC, &acc.HolderName, &acc.HolderType, &acc.Currency, &acc.Country,
		&acc.SubjectID, &acc.CreatedAt,
	)
	return acc, err
}

// ── Transaction CRUD ──────────────────────────────────────────────────────────

func (r *Repository) Create(ctx context.Context, orgID, createdBy uuid.UUID, req CreateTransactionRequest) (txnRow, error) {
	txnDate, err := time.Parse("2006-01-02", req.TxnDate)
	if err != nil {
		return txnRow{}, fmt.Errorf("invalid txn_date: %w", err)
	}

	currency := req.Currency
	if currency == "" {
		currency = "INR"
	}
	txnType := req.TxnType
	if txnType == "" {
		txnType = "debit"
	}
	channel := req.Channel
	if channel == "" {
		channel = "unknown"
	}
	status := req.Status
	if status == "" {
		status = "completed"
	}
	source := req.SourceSystem
	if source == "" {
		source = "manual"
	}

	var txnTime *string
	if req.TxnTime != "" {
		txnTime = &req.TxnTime
	}
	var fromAccID, toAccID *uuid.UUID
	if req.FromAccountID != nil {
		id, e := uuid.Parse(*req.FromAccountID)
		if e == nil {
			fromAccID = &id
		}
	}
	if req.ToAccountID != nil {
		id, e := uuid.Parse(*req.ToAccountID)
		if e == nil {
			toAccID = &id
		}
	}

	riskScore := req.RiskScore
	if riskScore < 0 {
		riskScore = 0
	} else if riskScore > 100 {
		riskScore = 100
	}

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO transactions (
		  org_id, txn_ref, txn_date, txn_time, amount, currency, txn_type, channel,
		  status, description, narration, from_account_id, to_account_id,
		  from_account_raw, to_account_raw, from_name_raw, to_name_raw,
		  is_flagged, flag_reason, risk_score, counterparty_bank, counterparty_country,
		  source_system, external_id, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
		RETURNING id, org_id, txn_ref, txn_date, txn_time::TEXT, amount, currency, txn_type,
		          channel, status, description, narration, from_account_id, to_account_id,
		          from_account_raw, to_account_raw, from_name_raw, to_name_raw,
		          is_flagged, flag_reason, risk_score, counterparty_bank, counterparty_country,
		          source_system, external_id, created_at, updated_at`,
		orgID, req.TxnRef, txnDate, txnTime, req.Amount, currency, txnType, channel,
		status, req.Description, req.Narration, fromAccID, toAccID,
		req.FromAccountRaw, req.ToAccountRaw, req.FromNameRaw, req.ToNameRaw,
		req.IsFlagged, req.FlagReason, riskScore, req.CounterpartyBank, req.CounterpartyCountry,
		source, req.ExternalID, createdBy,
	)

	return scanTxnBase(row)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (txnRow, error) {
	row := r.db.QueryRowContext(ctx, txnSelectSQL+` WHERE t.id = $1`, id)
	return scanTxnFull(row)
}

func (r *Repository) GetOrgIDForTxn(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var orgID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT org_id FROM transactions WHERE id = $1`, id).Scan(&orgID)
	return orgID, err
}

func (r *Repository) ListForCase(ctx context.Context, caseID uuid.UUID, limit, offset int) ([]txnRow, int, error) {
	var total int
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM case_transactions WHERE case_id = $1`, caseID).Scan(&total) //nolint:errcheck

	rows, err := r.db.QueryContext(ctx, txnSelectSQL+`
		WHERE ct.case_id = $1
		ORDER BY t.txn_date DESC, t.created_at DESC
		LIMIT $2 OFFSET $3`, caseID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []txnRow
	for rows.Next() {
		row, err := scanTxnFull(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, req UpdateTransactionRequest) error {
	parts := []string{"updated_at = NOW()"}
	args := []any{id}
	i := 2
	if req.IsFlagged != nil {
		parts = append(parts, fmt.Sprintf("is_flagged = $%d", i))
		args = append(args, *req.IsFlagged)
		i++
	}
	if req.FlagReason != nil {
		parts = append(parts, fmt.Sprintf("flag_reason = $%d", i))
		args = append(args, *req.FlagReason)
		i++
	}
	if req.RiskScore != nil {
		parts = append(parts, fmt.Sprintf("risk_score = $%d", i))
		args = append(args, *req.RiskScore)
		i++
	}
	_, err := r.db.ExecContext(ctx,
		fmt.Sprintf("UPDATE transactions SET %s WHERE id = $1", strings.Join(parts, ", ")),
		args...)
	return err
}

// ── Case ↔ Transaction links ──────────────────────────────────────────────────

func (r *Repository) LinkToCase(ctx context.Context, caseID, txnID, addedBy uuid.UUID, note string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO case_transactions (case_id, transaction_id, relevance_note, added_by)
		VALUES ($1, $2, $3, $4)`,
		caseID, txnID, note, addedBy)
	if err != nil && strings.Contains(err.Error(), "unique") {
		return ErrAlreadyLinked
	}
	return err
}

func (r *Repository) UnlinkFromCase(ctx context.Context, caseID, txnID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM case_transactions WHERE case_id = $1 AND transaction_id = $2`, caseID, txnID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotLinked
	}
	return nil
}

func (r *Repository) IsLinkedToCase(ctx context.Context, caseID, txnID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM case_transactions WHERE case_id=$1 AND transaction_id=$2)`,
		caseID, txnID).Scan(&exists)
	return exists, err
}

func (r *Repository) UpdateRelevanceNote(ctx context.Context, caseID, txnID uuid.UUID, note string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE case_transactions SET relevance_note = $3
		WHERE case_id = $1 AND transaction_id = $2`, caseID, txnID, note)
	return err
}

// ── Summary ───────────────────────────────────────────────────────────────────

type summaryRow struct {
	TotalCount     int
	TotalAmount    float64
	FlaggedCount   int
	DebitTotal     float64
	CreditTotal    float64
	UniqueAccounts int
	DateFrom       sql.NullString
	DateTo         sql.NullString
}

func (r *Repository) SummaryForCase(ctx context.Context, caseID uuid.UUID) (summaryRow, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
		  COUNT(*),
		  COALESCE(SUM(t.amount), 0),
		  COUNT(*) FILTER (WHERE t.is_flagged),
		  COALESCE(SUM(t.amount) FILTER (WHERE t.txn_type = 'debit'), 0),
		  COALESCE(SUM(t.amount) FILTER (WHERE t.txn_type = 'credit'), 0),
		  COUNT(DISTINCT COALESCE(t.from_account_id::TEXT, t.from_account_raw)) +
		    COUNT(DISTINCT COALESCE(t.to_account_id::TEXT, t.to_account_raw)),
		  MIN(t.txn_date)::TEXT,
		  MAX(t.txn_date)::TEXT
		FROM case_transactions ct
		JOIN transactions t ON t.id = ct.transaction_id
		WHERE ct.case_id = $1`, caseID)

	var s summaryRow
	err := row.Scan(&s.TotalCount, &s.TotalAmount, &s.FlaggedCount,
		&s.DebitTotal, &s.CreditTotal, &s.UniqueAccounts, &s.DateFrom, &s.DateTo)
	return s, err
}

// ── Flow-of-funds graph ───────────────────────────────────────────────────────

type flowEdgeRaw struct {
	FromKey    string
	FromLabel  string
	FromType   string
	FromIsSubj bool
	ToKey      string
	ToLabel    string
	ToType     string
	ToIsSubj   bool
	TotalAmt   float64
	TxnCount   int
	Channel    string
	HasFlagged bool
}

func (r *Repository) FlowEdgesForCase(ctx context.Context, caseID uuid.UUID) ([]flowEdgeRaw, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
		  COALESCE(t.from_account_id::TEXT, t.from_account_raw, 'unknown')          AS from_key,
		  COALESCE(fa.holder_name, t.from_name_raw, t.from_account_raw, 'Unknown')  AS from_label,
		  COALESCE(fa.account_type, 'unknown')                                        AS from_type,
		  (fa.subject_id IS NOT NULL)                                                 AS from_is_subj,
		  COALESCE(t.to_account_id::TEXT, t.to_account_raw, 'unknown')              AS to_key,
		  COALESCE(ta.holder_name, t.to_name_raw, t.to_account_raw, 'Unknown')      AS to_label,
		  COALESCE(ta.account_type, 'unknown')                                        AS to_type,
		  (ta.subject_id IS NOT NULL)                                                 AS to_is_subj,
		  SUM(t.amount)                                                               AS total_amt,
		  COUNT(*)                                                                     AS txn_count,
		  MODE() WITHIN GROUP (ORDER BY t.channel)                                   AS channel,
		  BOOL_OR(t.is_flagged)                                                       AS has_flagged
		FROM case_transactions ct
		JOIN transactions t ON t.id = ct.transaction_id
		LEFT JOIN accounts fa ON fa.id = t.from_account_id
		LEFT JOIN accounts ta ON ta.id = t.to_account_id
		WHERE ct.case_id = $1
		GROUP BY from_key, from_label, from_type, from_is_subj,
		         to_key, to_label, to_type, to_is_subj`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []flowEdgeRaw
	for rows.Next() {
		var e flowEdgeRaw
		if err := rows.Scan(&e.FromKey, &e.FromLabel, &e.FromType, &e.FromIsSubj,
			&e.ToKey, &e.ToLabel, &e.ToType, &e.ToIsSubj,
			&e.TotalAmt, &e.TxnCount, &e.Channel, &e.HasFlagged); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// ── CSV bulk create ───────────────────────────────────────────────────────────

func (r *Repository) BulkCreate(ctx context.Context, orgID, createdBy uuid.UUID, reqs []CreateTransactionRequest) CSVImportResult {
	result := CSVImportResult{}
	for i, req := range reqs {
		_, err := r.Create(ctx, orgID, createdBy, req)
		if err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %s", i+1, err.Error()))
		} else {
			result.Imported++
		}
	}
	return result
}

// ── SQL fragments ─────────────────────────────────────────────────────────────

const txnSelectSQL = `
SELECT
  t.id, t.org_id, t.txn_ref, t.txn_date, t.txn_time::TEXT,
  t.amount, t.currency, t.txn_type, t.channel, t.status,
  t.description, t.narration,
  t.from_account_id, t.to_account_id,
  t.from_account_raw, t.to_account_raw, t.from_name_raw, t.to_name_raw,
  t.is_flagged, t.flag_reason, t.risk_score,
  t.counterparty_bank, t.counterparty_country, t.source_system, t.external_id,
  t.created_at, t.updated_at,
  COALESCE(ct.relevance_note, '') AS relevance_note,
  fa.account_number, fa.account_type, fa.bank_name, fa.holder_name,
  ta.account_number, ta.account_type, ta.bank_name, ta.holder_name
FROM transactions t
LEFT JOIN case_transactions ct ON ct.transaction_id = t.id
LEFT JOIN accounts fa ON fa.id = t.from_account_id
LEFT JOIN accounts ta ON ta.id = t.to_account_id
`

func scanTxnBase(s interface {
	Scan(...any) error
}) (txnRow, error) {
	var row txnRow
	err := s.Scan(
		&row.ID, &row.OrgID, &row.TxnRef, &row.TxnDate, &row.TxnTime,
		&row.Amount, &row.Currency, &row.TxnType, &row.Channel, &row.Status,
		&row.Description, &row.Narration,
		&row.FromAccountID, &row.ToAccountID,
		&row.FromAccountRaw, &row.ToAccountRaw, &row.FromNameRaw, &row.ToNameRaw,
		&row.IsFlagged, &row.FlagReason, &row.RiskScore,
		&row.CounterpartyBank, &row.CounterpartyCountry, &row.SourceSystem, &row.ExternalID,
		&row.CreatedAt, &row.UpdatedAt,
	)
	return row, err
}

func scanTxnFull(s interface {
	Scan(...any) error
}) (txnRow, error) {
	var row txnRow
	err := s.Scan(
		&row.ID, &row.OrgID, &row.TxnRef, &row.TxnDate, &row.TxnTime,
		&row.Amount, &row.Currency, &row.TxnType, &row.Channel, &row.Status,
		&row.Description, &row.Narration,
		&row.FromAccountID, &row.ToAccountID,
		&row.FromAccountRaw, &row.ToAccountRaw, &row.FromNameRaw, &row.ToNameRaw,
		&row.IsFlagged, &row.FlagReason, &row.RiskScore,
		&row.CounterpartyBank, &row.CounterpartyCountry, &row.SourceSystem, &row.ExternalID,
		&row.CreatedAt, &row.UpdatedAt,
		&row.RelevanceNote,
		&row.FromAccNum, &row.FromAccType, &row.FromAccBank, &row.FromHolderName,
		&row.ToAccNum, &row.ToAccType, &row.ToAccBank, &row.ToHolderName,
	)
	return row, err
}
