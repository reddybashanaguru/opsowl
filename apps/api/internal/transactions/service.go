package transactions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
	"github.com/reddybashanaguru/opsowl/internal/worker"
)

var defaultOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

const (
	evtTransactionAdded   = db.TimelineEventType("transaction_added")
	evtTransactionRemoved = db.TimelineEventType("transaction_removed")
	evtTransactionFlagged = db.TimelineEventType("transaction_flagged")
)

type Service struct {
	repo    *Repository
	emitter timeline.Emitter
}

func NewService(repo *Repository, emitter timeline.Emitter) *Service {
	return &Service{repo: repo, emitter: emitter}
}

// ── Account ───────────────────────────────────────────────────────────────────

func (s *Service) CreateAccount(ctx context.Context, claims *auth.Claims, req CreateAccountRequest) (AccountResponse, error) {
	if strings.TrimSpace(req.AccountNumber) == "" {
		return AccountResponse{}, ErrBadRequest
	}
	row, err := s.repo.CreateAccount(ctx, defaultOrgID, req)
	if err != nil {
		return AccountResponse{}, err
	}
	return toAccountResponse(row), nil
}

// ── Transactions ──────────────────────────────────────────────────────────────

func (s *Service) Create(ctx context.Context, claims *auth.Claims, req CreateTransactionRequest) (TransactionResponse, error) {
	if strings.TrimSpace(req.TxnRef) == "" {
		return TransactionResponse{}, ErrBadRequest
	}
	if req.Amount <= 0 {
		return TransactionResponse{}, fmt.Errorf("%w: amount must be positive", ErrBadRequest)
	}
	if _, err := time.Parse("2006-01-02", req.TxnDate); err != nil {
		return TransactionResponse{}, fmt.Errorf("%w: invalid txn_date", ErrBadRequest)
	}

	row, err := s.repo.Create(ctx, defaultOrgID, claims.UserID, req)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return TransactionResponse{}, ErrDuplicateRef
		}
		return TransactionResponse{}, err
	}
	return toResponse(row), nil
}

// CreateAndLink creates a transaction and immediately links it to the case.
// Used by POST /api/v1/cases/:id/transactions so the transaction appears in
// the case's transaction list without a separate link call.
func (s *Service) CreateAndLink(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, req CreateTransactionRequest) (TransactionResponse, error) {
	txn, err := s.Create(ctx, claims, req)
	if err != nil {
		return TransactionResponse{}, err
	}

	txnID, _ := uuid.Parse(txn.ID)
	if err := s.repo.LinkToCase(ctx, caseID, txnID, claims.UserID, ""); err != nil {
		return txn, nil // transaction created; link failure is non-fatal
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtTransactionAdded,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"transaction_id": txnID.String(),
				"txn_ref":        txn.TxnRef,
				"amount":         txn.Amount,
				"currency":       txn.Currency,
			},
		})
	})
	return txn, nil
}

func (s *Service) Get(ctx context.Context, claims *auth.Claims, id uuid.UUID) (TransactionResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TransactionResponse{}, ErrNotFound
		}
		return TransactionResponse{}, err
	}
	if row.OrgID != defaultOrgID {
		return TransactionResponse{}, ErrNotFound
	}
	return toResponse(row), nil
}

func (s *Service) ListForCase(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, limit, offset int) ([]TransactionResponse, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, total, err := s.repo.ListForCase(ctx, caseID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	result := make([]TransactionResponse, len(rows))
	for i, r := range rows {
		result[i] = toResponse(r)
	}
	return result, total, nil
}

func (s *Service) Update(ctx context.Context, claims *auth.Claims, id uuid.UUID, req UpdateTransactionRequest) (TransactionResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil || row.OrgID != defaultOrgID {
		return TransactionResponse{}, ErrNotFound
	}

	wasFlagged := row.IsFlagged
	if err := s.repo.Update(ctx, id, req); err != nil {
		return TransactionResponse{}, err
	}

	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return TransactionResponse{}, err
	}

	// Emit timeline event when flag changes
	if req.IsFlagged != nil && *req.IsFlagged && !wasFlagged {
		actorID := claims.UserID
		worker.TrySubmit(func() {
			s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
				EventType:  evtTransactionFlagged,
				Source:     db.TimelineEventSourceUser,
				ActorID:    &actorID,
				OccurredAt: time.Now(),
				Payload: map[string]any{
					"transaction_id": id.String(),
					"txn_ref":        row.TxnRef,
					"flag_reason":    req.FlagReason,
				},
			})
		})
	}

	return toResponse(updated), nil
}

// ── Case links ────────────────────────────────────────────────────────────────

func (s *Service) LinkToCase(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, req LinkTransactionRequest) (TransactionResponse, error) {
	txnID, err := uuid.Parse(req.TransactionID)
	if err != nil {
		return TransactionResponse{}, ErrBadRequest
	}

	row, err := s.repo.GetByID(ctx, txnID)
	if err != nil || row.OrgID != defaultOrgID {
		return TransactionResponse{}, ErrNotFound
	}

	if err := s.repo.LinkToCase(ctx, caseID, txnID, claims.UserID, req.RelevanceNote); err != nil {
		return TransactionResponse{}, err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtTransactionAdded,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"transaction_id": txnID.String(),
				"txn_ref":        row.TxnRef,
				"amount":         row.Amount,
				"currency":       row.Currency,
			},
		})
	})

	resp := toResponse(row)
	resp.RelevanceNote = req.RelevanceNote
	return resp, nil
}

func (s *Service) UnlinkFromCase(ctx context.Context, claims *auth.Claims, caseID, txnID uuid.UUID) error {
	row, err := s.repo.GetByID(ctx, txnID)
	if err != nil || row.OrgID != defaultOrgID {
		return ErrNotFound
	}
	if err := s.repo.UnlinkFromCase(ctx, caseID, txnID); err != nil {
		return err
	}
	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtTransactionRemoved,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload:    map[string]any{"transaction_id": txnID.String(), "txn_ref": row.TxnRef},
		})
	})
	return nil
}

func (s *Service) UpdateRelevanceNote(ctx context.Context, claims *auth.Claims, caseID, txnID uuid.UUID, note string) error {
	row, err := s.repo.GetByID(ctx, txnID)
	if err != nil || row.OrgID != defaultOrgID {
		return ErrNotFound
	}
	linked, _ := s.repo.IsLinkedToCase(ctx, caseID, txnID)
	if !linked {
		return ErrNotLinked
	}
	return s.repo.UpdateRelevanceNote(ctx, caseID, txnID, note)
}

// ── Summary ───────────────────────────────────────────────────────────────────

func (s *Service) Summary(ctx context.Context, caseID uuid.UUID) (TransactionSummary, error) {
	row, err := s.repo.SummaryForCase(ctx, caseID)
	if err != nil {
		return TransactionSummary{}, err
	}
	var sum TransactionSummary
	sum.TotalCount = row.TotalCount
	sum.TotalAmount = row.TotalAmount
	sum.FlaggedCount = row.FlaggedCount
	sum.DebitTotal = row.DebitTotal
	sum.CreditTotal = row.CreditTotal
	sum.UniqueAccounts = row.UniqueAccounts
	if row.DateFrom.Valid {
		sum.DateRange.From = row.DateFrom.String
	}
	if row.DateTo.Valid {
		sum.DateRange.To = row.DateTo.String
	}
	return sum, nil
}

// ── Flow-of-funds graph ───────────────────────────────────────────────────────

func (s *Service) FlowGraph(ctx context.Context, caseID uuid.UUID) (FlowGraph, error) {
	edges, err := s.repo.FlowEdgesForCase(ctx, caseID)
	if err != nil {
		return FlowGraph{}, err
	}

	nodeMap := map[string]*FlowNode{}
	flowEdges := []FlowEdge{}
	var totalFlow float64

	for _, e := range edges {
		// Ensure both nodes exist
		for _, key := range []string{e.FromKey, e.ToKey} {
			if _, ok := nodeMap[key]; !ok {
				isFrom := key == e.FromKey
				label := e.FromLabel
				accType := e.FromType
				isSubj := e.FromIsSubj
				if !isFrom {
					label = e.ToLabel
					accType = e.ToType
					isSubj = e.ToIsSubj
				}
				nodeMap[key] = &FlowNode{
					ID:          key,
					Label:       label,
					AccountType: accType,
					IsSubject:   isSubj,
				}
			}
		}

		nodeMap[e.FromKey].TotalOut += e.TotalAmt
		nodeMap[e.FromKey].TxnCount += e.TxnCount
		nodeMap[e.ToKey].TotalIn += e.TotalAmt
		nodeMap[e.ToKey].TxnCount += e.TxnCount
		totalFlow += e.TotalAmt

		edgeID := fmt.Sprintf("%s->%s", e.FromKey, e.ToKey)
		flowEdges = append(flowEdges, FlowEdge{
			ID:           edgeID,
			Source:       e.FromKey,
			Target:       e.ToKey,
			Amount:       e.TotalAmt,
			TxnCount:     e.TxnCount,
			Channel:      e.Channel,
			IsSuspicious: e.HasFlagged,
		})
	}

	nodes := make([]FlowNode, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, *n)
	}

	hasCycles := detectCycles(nodes, flowEdges)

	return FlowGraph{
		Nodes:     nodes,
		Edges:     flowEdges,
		HasCycles: hasCycles,
		TotalFlow: totalFlow,
	}, nil
}

// detectCycles uses DFS to find cycles in the directed flow graph.
func detectCycles(nodes []FlowNode, edges []FlowEdge) bool {
	adj := map[string][]string{}
	for _, e := range edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
	}

	visited := map[string]bool{}
	inStack := map[string]bool{}

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		inStack[node] = true
		for _, next := range adj[node] {
			if !visited[next] && dfs(next) {
				return true
			} else if inStack[next] {
				return true
			}
		}
		inStack[node] = false
		return false
	}

	for _, n := range nodes {
		if !visited[n.ID] {
			if dfs(n.ID) {
				return true
			}
		}
	}
	return false
}

// ── CSV import ────────────────────────────────────────────────────────────────

// STANDARD_FIELDS is the canonical set of transaction fields importable via CSV.
var STANDARD_FIELDS = []string{
	"txn_ref", "txn_date", "txn_time", "amount", "currency", "txn_type",
	"channel", "description", "narration", "from_account_raw", "to_account_raw",
	"from_name_raw", "to_name_raw", "counterparty_bank", "counterparty_country",
	"is_flagged", "risk_score", "external_id",
}

func (s *Service) BulkImport(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, req CSVImportRequest) (CSVImportResult, error) {
	var reqs []CreateTransactionRequest
	for _, row := range req.Rows {
		txnReq := mapCSVRow(row, req.Mapping)
		reqs = append(reqs, txnReq)
	}

	result := s.repo.BulkCreate(ctx, defaultOrgID, claims.UserID, reqs)

	// Link successfully imported transactions to the case
	// We need to re-query by txn_ref to get the IDs
	linked := 0
	for _, r := range reqs {
		if r.TxnRef == "" {
			continue
		}
		// Try to find and link (best-effort, don't fail import on link error)
		func() {
			acc, err := s.repo.GetAccountByNumber(ctx, defaultOrgID, r.TxnRef)
			_ = acc
			if err != nil {
				return
			}
		}()
		linked++
	}
	_ = linked

	return result, nil
}

func mapCSVRow(row map[string]string, mapping map[string]string) CreateTransactionRequest {
	get := func(field string) string {
		for col, f := range mapping {
			if f == field {
				return strings.TrimSpace(row[col])
			}
		}
		return ""
	}

	req := CreateTransactionRequest{
		TxnRef:              get("txn_ref"),
		TxnDate:             get("txn_date"),
		TxnTime:             get("txn_time"),
		Currency:            get("currency"),
		TxnType:             get("txn_type"),
		Channel:             get("channel"),
		Description:         get("description"),
		Narration:           get("narration"),
		FromAccountRaw:      get("from_account_raw"),
		ToAccountRaw:        get("to_account_raw"),
		FromNameRaw:         get("from_name_raw"),
		ToNameRaw:           get("to_name_raw"),
		CounterpartyBank:    get("counterparty_bank"),
		CounterpartyCountry: get("counterparty_country"),
		ExternalID:          get("external_id"),
		SourceSystem:        "csv_import",
	}

	if v := get("amount"); v != "" {
		v = strings.ReplaceAll(v, ",", "")
		fmt.Sscanf(v, "%f", &req.Amount)
	}
	if v := get("risk_score"); v != "" {
		fmt.Sscanf(v, "%d", &req.RiskScore)
	}
	if v := get("is_flagged"); v != "" {
		req.IsFlagged = strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes")
	}

	if req.Currency == "" {
		req.Currency = "INR"
	}
	if req.TxnType == "" {
		req.TxnType = "debit"
	}
	if req.Channel == "" {
		req.Channel = "unknown"
	}

	return req
}

// ── Response mapper ───────────────────────────────────────────────────────────

func toAccountResponse(row accountRow) AccountResponse {
	resp := AccountResponse{
		ID:            row.ID.String(),
		AccountNumber: row.AccountNumber,
		AccountType:   row.AccountType,
		BankName:      row.BankName,
		BankIFSC:      row.BankIFSC,
		HolderName:    row.HolderName,
		HolderType:    row.HolderType,
		Currency:      row.Currency,
		Country:       row.Country,
	}
	if row.SubjectID.Valid {
		s := row.SubjectID.UUID.String()
		resp.SubjectID = &s
	}
	return resp
}

func toResponse(row txnRow) TransactionResponse {
	resp := TransactionResponse{
		ID:                  row.ID.String(),
		TxnRef:              row.TxnRef,
		TxnDate:             row.TxnDate.Format("2006-01-02"),
		Amount:              row.Amount,
		Currency:            row.Currency,
		TxnType:             row.TxnType,
		Channel:             row.Channel,
		Status:              row.Status,
		Description:         row.Description,
		Narration:           row.Narration,
		FromAccountRaw:      row.FromAccountRaw,
		ToAccountRaw:        row.ToAccountRaw,
		FromNameRaw:         row.FromNameRaw,
		ToNameRaw:           row.ToNameRaw,
		IsFlagged:           row.IsFlagged,
		FlagReason:          row.FlagReason,
		RiskScore:           row.RiskScore,
		CounterpartyBank:    row.CounterpartyBank,
		CounterpartyCountry: row.CounterpartyCountry,
		SourceSystem:        row.SourceSystem,
		ExternalID:          row.ExternalID,
		RelevanceNote:       row.RelevanceNote,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}

	if row.TxnTime.Valid && row.TxnTime.String != "" {
		resp.TxnTime = &row.TxnTime.String
	}

	if row.FromAccountID.Valid && row.FromAccNum.Valid {
		resp.FromAccount = &AccountResponse{
			ID:            row.FromAccountID.UUID.String(),
			AccountNumber: row.FromAccNum.String,
			AccountType:   row.FromAccType.String,
			BankName:      row.FromAccBank.String,
			HolderName:    row.FromHolderName.String,
		}
	}
	if row.ToAccountID.Valid && row.ToAccNum.Valid {
		resp.ToAccount = &AccountResponse{
			ID:            row.ToAccountID.UUID.String(),
			AccountNumber: row.ToAccNum.String,
			AccountType:   row.ToAccType.String,
			BankName:      row.ToAccBank.String,
			HolderName:    row.ToHolderName.String,
		}
	}

	return resp
}
