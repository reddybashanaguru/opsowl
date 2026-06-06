package transactions

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound       = errors.New("transaction not found")
	ErrForbidden      = errors.New("forbidden")
	ErrBadRequest     = errors.New("bad request")
	ErrAlreadyLinked  = errors.New("transaction already linked to this case")
	ErrNotLinked      = errors.New("transaction is not linked to this case")
	ErrDuplicateRef   = errors.New("a transaction with this reference already exists")
)

// ── Account types ─────────────────────────────────────────────────────────────

type CreateAccountRequest struct {
	AccountNumber string  `json:"account_number"`
	AccountType   string  `json:"account_type"`
	BankName      string  `json:"bank_name"`
	BankIFSC      string  `json:"bank_ifsc"`
	HolderName    string  `json:"holder_name"`
	HolderType    string  `json:"holder_type"`
	Currency      string  `json:"currency"`
	Country       string  `json:"country"`
	SubjectID     *string `json:"subject_id,omitempty"`
}

type AccountResponse struct {
	ID            string  `json:"id"`
	AccountNumber string  `json:"account_number"`
	AccountType   string  `json:"account_type"`
	BankName      string  `json:"bank_name"`
	BankIFSC      string  `json:"bank_ifsc"`
	HolderName    string  `json:"holder_name"`
	HolderType    string  `json:"holder_type"`
	Currency      string  `json:"currency"`
	Country       string  `json:"country"`
	SubjectID     *string `json:"subject_id,omitempty"`
}

// ── Transaction types ─────────────────────────────────────────────────────────

type CreateTransactionRequest struct {
	TxnRef             string  `json:"txn_ref"`
	TxnDate            string  `json:"txn_date"` // YYYY-MM-DD
	TxnTime            string  `json:"txn_time,omitempty"`
	Amount             float64 `json:"amount"`
	Currency           string  `json:"currency"`
	TxnType            string  `json:"txn_type"`
	Channel            string  `json:"channel"`
	Status             string  `json:"status"`
	Description        string  `json:"description"`
	Narration          string  `json:"narration"`
	FromAccountID      *string `json:"from_account_id,omitempty"`
	ToAccountID        *string `json:"to_account_id,omitempty"`
	FromAccountRaw     string  `json:"from_account_raw"`
	ToAccountRaw       string  `json:"to_account_raw"`
	FromNameRaw        string  `json:"from_name_raw"`
	ToNameRaw          string  `json:"to_name_raw"`
	IsFlagged          bool    `json:"is_flagged"`
	FlagReason         string  `json:"flag_reason"`
	RiskScore          int     `json:"risk_score"`
	CounterpartyBank   string  `json:"counterparty_bank"`
	CounterpartyCountry string `json:"counterparty_country"`
	SourceSystem       string  `json:"source_system"`
	ExternalID         string  `json:"external_id"`
}

type UpdateTransactionRequest struct {
	IsFlagged  *bool   `json:"is_flagged,omitempty"`
	FlagReason *string `json:"flag_reason,omitempty"`
	RiskScore  *int    `json:"risk_score,omitempty"`
	RelevanceNote *string `json:"relevance_note,omitempty"`
}

type LinkTransactionRequest struct {
	TransactionID string `json:"transaction_id"`
	RelevanceNote string `json:"relevance_note"`
}

type TransactionResponse struct {
	ID                  string   `json:"id"`
	TxnRef              string   `json:"txn_ref"`
	TxnDate             string   `json:"txn_date"`
	TxnTime             *string  `json:"txn_time,omitempty"`
	Amount              float64  `json:"amount"`
	Currency            string   `json:"currency"`
	TxnType             string   `json:"txn_type"`
	Channel             string   `json:"channel"`
	Status              string   `json:"status"`
	Description         string   `json:"description"`
	Narration           string   `json:"narration"`
	FromAccount         *AccountResponse `json:"from_account,omitempty"`
	ToAccount           *AccountResponse `json:"to_account,omitempty"`
	FromAccountRaw      string   `json:"from_account_raw"`
	ToAccountRaw        string   `json:"to_account_raw"`
	FromNameRaw         string   `json:"from_name_raw"`
	ToNameRaw           string   `json:"to_name_raw"`
	IsFlagged           bool     `json:"is_flagged"`
	FlagReason          string   `json:"flag_reason"`
	RiskScore           int      `json:"risk_score"`
	CounterpartyBank    string   `json:"counterparty_bank"`
	CounterpartyCountry string   `json:"counterparty_country"`
	SourceSystem        string   `json:"source_system"`
	ExternalID          string   `json:"external_id"`
	RelevanceNote       string   `json:"relevance_note,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ── Flow-of-funds graph ───────────────────────────────────────────────────────

type FlowNode struct {
	ID          string  `json:"id"`   // account ID or raw label
	Label       string  `json:"label"`
	HolderName  string  `json:"holder_name"`
	AccountType string  `json:"account_type"`
	IsSubject   bool    `json:"is_subject"`
	TotalIn     float64 `json:"total_in"`
	TotalOut    float64 `json:"total_out"`
	TxnCount    int     `json:"txn_count"`
}

type FlowEdge struct {
	ID          string  `json:"id"`
	Source      string  `json:"source"` // node ID
	Target      string  `json:"target"` // node ID
	Amount      float64 `json:"amount"`
	TxnCount    int     `json:"txn_count"`
	Channel     string  `json:"channel"` // dominant channel
	IsSuspicious bool   `json:"is_suspicious"` // any flagged txn on this edge
}

type FlowGraph struct {
	Nodes     []FlowNode `json:"nodes"`
	Edges     []FlowEdge `json:"edges"`
	HasCycles bool       `json:"has_cycles"`
	TotalFlow float64    `json:"total_flow"`
}

// ── CSV import ────────────────────────────────────────────────────────────────

type CSVImportRequest struct {
	CaseID    uuid.UUID         `json:"case_id"`
	Mapping   map[string]string `json:"mapping"` // csv_col → txn field
	Rows      []map[string]string `json:"rows"`
}

type CSVImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// ── Summary ───────────────────────────────────────────────────────────────────

type TransactionSummary struct {
	TotalCount    int     `json:"total_count"`
	TotalAmount   float64 `json:"total_amount"`
	FlaggedCount  int     `json:"flagged_count"`
	DebitTotal    float64 `json:"debit_total"`
	CreditTotal   float64 `json:"credit_total"`
	UniqueAccounts int    `json:"unique_accounts"`
	DateRange     struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"date_range"`
}
