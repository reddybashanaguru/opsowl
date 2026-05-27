package insights

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
	"github.com/reddybashanaguru/opsowl/internal/worker"
)

type Handler struct {
	queries      *db.Queries
	aiServiceURL string
	httpClient   *http.Client
}

func NewHandler(queries *db.Queries, aiServiceURL string) *Handler {
	return &Handler{
		queries:      queries,
		aiServiceURL: aiServiceURL,
		httpClient:   &http.Client{Timeout: 120 * time.Second},
	}
}

type InsightsResponse struct {
	ID                 *uuid.UUID  `json:"id,omitempty"`
	CaseID             uuid.UUID   `json:"case_id"`
	RiskScore          *float64    `json:"risk_score"`
	RiskLevel          string      `json:"risk_level"`
	RiskFactors        interface{} `json:"risk_factors"`
	Entities           interface{} `json:"entities"`
	Anomalies          interface{} `json:"anomalies"`
	RecommendedActions interface{} `json:"recommended_actions"`
	ModelUsed          *string     `json:"model_used"`
	Confidence         *float64    `json:"confidence"`
	AnalysedAt         *time.Time  `json:"analysed_at,omitempty"`
	AIAvailable        bool        `json:"ai_available"`
}

type aiAnalyseRequest struct {
	CaseID string `json:"case_id"`
}

type aiAnalyseResponse struct {
	RiskScore          float64     `json:"risk_score"`
	RiskFactors        interface{} `json:"risk_factors"`
	Entities           interface{} `json:"entities"`
	Anomalies          interface{} `json:"anomalies"`
	RecommendedActions interface{} `json:"recommended_actions"`
	ModelUsed          string      `json:"model_used"`
	Confidence         float64     `json:"confidence"`
}

var emptyInsights = func(caseID uuid.UUID) InsightsResponse {
	return InsightsResponse{
		CaseID:             caseID,
		AIAvailable:        false,
		RiskLevel:          "unknown",
		RiskFactors:        []interface{}{},
		Entities:           map[string]interface{}{},
		Anomalies:          []interface{}{},
		RecommendedActions: []interface{}{},
	}
}

// GET /api/v1/cases/:id/insights
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	row, err := h.queries.GetLatestCaseAIInsights(r.Context(), caseID)
	if err != nil {
		response.JSON(w, http.StatusOK, emptyInsights(caseID))
		return
	}
	response.JSON(w, http.StatusOK, rowToResponse(row))
}

// POST /api/v1/cases/:id/insights/analyse
// Returns 202 Accepted immediately. The analysis runs in the background worker
// pool. Clients should poll GET /cases/:id/insights until ai_available=true.
func (h *Handler) Analyse(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	worker.TrySubmit(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		aiResp, aiErr := h.callAIAnalyse(ctx, caseID.String())
		if aiErr != nil {
			return
		}
		_ = h.storeInsights(context.Background(), caseID, aiResp)
	})

	response.JSON(w, http.StatusAccepted, map[string]any{
		"queued":  true,
		"case_id": caseID.String(),
	})
}

// storeInsights persists AI analysis results to the database.
func (h *Handler) storeInsights(ctx context.Context, caseID uuid.UUID, aiResp *aiAnalyseResponse) error {
	rfJSON, _ := json.Marshal(aiResp.RiskFactors)
	entJSON, _ := json.Marshal(aiResp.Entities)
	anomJSON, _ := json.Marshal(aiResp.Anomalies)
	recJSON, _ := json.Marshal(aiResp.RecommendedActions)

	_, err := h.queries.UpsertCaseAIInsights(ctx, db.UpsertCaseAIInsightsParams{
		CaseID:             caseID,
		RiskScore:          sql.NullString{String: fmt.Sprintf("%.2f", aiResp.RiskScore), Valid: true},
		RiskFactors:        rfJSON,
		Entities:           entJSON,
		Anomalies:          anomJSON,
		RecommendedActions: recJSON,
		ModelUsed:          sql.NullString{String: aiResp.ModelUsed, Valid: aiResp.ModelUsed != ""},
		Confidence:         sql.NullString{String: fmt.Sprintf("%.2f", aiResp.Confidence), Valid: true},
	})
	return err
}

func (h *Handler) callAIAnalyse(ctx context.Context, caseID string) (*aiAnalyseResponse, error) {
	body, _ := json.Marshal(aiAnalyseRequest{CaseID: caseID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.aiServiceURL+"/analyse/case", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai service %d", resp.StatusCode)
	}
	var out aiAnalyseResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func rowToResponse(row db.CaseAiInsight) InsightsResponse {
	riskLevel := "unknown"
	if row.RiskLevel.Valid {
		riskLevel = row.RiskLevel.String
	}
	r := InsightsResponse{
		ID:          &row.ID,
		CaseID:      row.CaseID,
		RiskLevel:   riskLevel,
		AIAvailable: true,
		AnalysedAt:  &row.AnalysedAt,
	}
	if row.RiskScore.Valid {
		var f float64
		fmt.Sscanf(row.RiskScore.String, "%f", &f)
		r.RiskScore = &f
	}
	if row.Confidence.Valid {
		var f float64
		fmt.Sscanf(row.Confidence.String, "%f", &f)
		r.Confidence = &f
	}
	if row.ModelUsed.Valid {
		r.ModelUsed = &row.ModelUsed.String
	}
	var rf, ent, an, rec interface{}
	_ = json.Unmarshal(row.RiskFactors, &rf)
	_ = json.Unmarshal(row.Entities, &ent)
	_ = json.Unmarshal(row.Anomalies, &an)
	_ = json.Unmarshal(row.RecommendedActions, &rec)
	r.RiskFactors = rf
	r.Entities = ent
	r.Anomalies = an
	r.RecommendedActions = rec
	return r
}
