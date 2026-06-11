package transactions

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// POST /api/v1/accounts
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest
	if err := response.DecodeBody(r, &req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	acc, err := h.svc.CreateAccount(r.Context(), claims, req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, acc)
}

// GET /api/v1/cases/:id/transactions
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 50
	}

	claims := auth.ClaimsFromCtx(r.Context())
	rows, total, err := h.svc.ListForCase(r.Context(), claims, caseID, limit, offset)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "LIST_ERROR", err.Error())
		return
	}
	response.JSONWithMeta(w, http.StatusOK, rows, map[string]any{"total": total})
}

// POST /api/v1/cases/:id/transactions
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	_, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req CreateTransactionRequest
	if err := response.DecodeBody(r, &req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	txn, err := h.svc.Create(r.Context(), claims, req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, txn)
}

// GET /api/v1/cases/:id/transactions/:txnId
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	txnID, ok := h.parseTxnID(w, r)
	if !ok {
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	txn, err := h.svc.Get(r.Context(), claims, txnID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, txn)
}

// PATCH /api/v1/cases/:id/transactions/:txnId
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	txnID, ok := h.parseTxnID(w, r)
	if !ok {
		return
	}

	var req UpdateTransactionRequest
	if err := response.DecodeBody(r, &req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	txn, err := h.svc.Update(r.Context(), claims, txnID, req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, txn)
}

// POST /api/v1/cases/:id/transactions/link
func (h *Handler) Link(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req LinkTransactionRequest
	if err := response.DecodeBody(r, &req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	txn, err := h.svc.LinkToCase(r.Context(), claims, caseID, req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, txn)
}

// DELETE /api/v1/cases/:id/transactions/:txnId
func (h *Handler) Unlink(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	txnID, ok := h.parseTxnID(w, r)
	if !ok {
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.UnlinkFromCase(r.Context(), claims, caseID, txnID); err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"unlinked": true})
}

// PATCH /api/v1/cases/:id/transactions/:txnId/note
func (h *Handler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	txnID, ok := h.parseTxnID(w, r)
	if !ok {
		return
	}

	var body struct {
		RelevanceNote string `json:"relevance_note"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck

	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.UpdateRelevanceNote(r.Context(), claims, caseID, txnID, body.RelevanceNote); err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// GET /api/v1/cases/:id/transactions/summary
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	sum, err := h.svc.Summary(r.Context(), caseID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "SUMMARY_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, sum)
}

// GET /api/v1/cases/:id/transactions/flow
func (h *Handler) FlowGraph(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	graph, err := h.svc.FlowGraph(r.Context(), caseID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "GRAPH_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, graph)
}

// POST /api/v1/cases/:id/transactions/import
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req CSVImportRequest
	if err := response.DecodeBody(r, &req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	req.CaseID = caseID

	claims := auth.ClaimsFromCtx(r.Context())
	result, err := h.svc.BulkImport(r.Context(), claims, caseID, req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// GET /api/v1/transactions/fields
func (h *Handler) Fields(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{"fields": STANDARD_FIELDS})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *Handler) parseTxnID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "txnId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid transaction id")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Err(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, ErrForbidden):
		response.Err(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, ErrBadRequest):
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, ErrAlreadyLinked):
		response.Err(w, http.StatusConflict, "ALREADY_LINKED", err.Error())
	case errors.Is(err, ErrNotLinked):
		response.Err(w, http.StatusNotFound, "NOT_LINKED", err.Error())
	case errors.Is(err, ErrDuplicateRef):
		response.Err(w, http.StatusConflict, "DUPLICATE_REF", err.Error())
	default:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
