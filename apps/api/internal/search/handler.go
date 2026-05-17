package search

import (
	"net/http"
	"strconv"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/cases"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	q := r.URL.Query().Get("q")
	if q == "" {
		response.Err(w, http.StatusBadRequest, "MISSING_QUERY", "query parameter 'q' is required")
		return
	}

	limit := int32(10)
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.ParseInt(lStr, 10, 32); err == nil && l > 0 {
			limit = int32(l)
		}
	}

	filter := r.URL.Query().Get("filter")

	results, err := h.svc.Search(r.Context(), cases.DefaultOrgID, SearchRequest{
		Query:  q,
		Limit:  limit,
		Filter: filter,
	})
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "SEARCH_FAILED", "search failed")
		return
	}

	_ = claims
	response.JSON(w, http.StatusOK, results)
}
