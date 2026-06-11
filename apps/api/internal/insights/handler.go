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
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
	"github.com/reddybashanaguru/opsowl/internal/worker"
)

// aiAnalyseResult carries both the parsed response and timing metadata
// so the caller can persist a full audit record.
type aiAnalyseResult struct {
	response  *aiAnalyseResponse
	latencyMs int32
	startedAt time.Time
}

type Handler struct {
	queries      *db.Queries
	aiServiceURL string
	httpClient   *http.Client
	emitter      timeline.Emitter
}

func NewHandler(queries *db.Queries, aiServiceURL string, emitter timeline.Emitter) *Handler {
	return &Handler{
		queries:      queries,
		aiServiceURL: aiServiceURL,
		httpClient:   &http.Client{Timeout: 120 * time.Second},
		emitter:      emitter,
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

	claims := auth.ClaimsFromCtx(r.Context())

	triggeredBy := claims.UserID
	worker.TrySubmit(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		result, aiErr := h.callAIAnalyseWithTiming(ctx, caseID.String())
		if aiErr != nil {
			// Log the failed attempt to ai_logs so the audit trail shows "analysis attempted but failed"
			promptJSON := fmt.Sprintf(`{"case_id":"%s","action":"analyse_case"}`, caseID.String())
			_ = h.logToAILogs(context.Background(), caseID, triggeredBy, "analyse_case",
				promptJSON, fmt.Sprintf("error: %v", aiErr), "unknown", result.latencyMs, 0)
			return
		}
		if err := h.storeInsights(context.Background(), caseID, result.response); err != nil {
			return
		}
		// Log the full AI interaction to ai_logs for regulatory auditability.
		// This creates a tamper-evident record of what was analysed, what the AI returned,
		// and which model version was used — answerable 24 months later by an FIU examiner.
		promptJSON := fmt.Sprintf(`{"case_id":"%s","action":"analyse_case","triggered_at":"%s"}`,
			caseID.String(), result.startedAt.UTC().Format(time.RFC3339))
		respJSON, _ := json.Marshal(result.response)
		aiLogID := h.logToAILogs(context.Background(), caseID, triggeredBy, "analyse_case",
			promptJSON, string(respJSON), result.response.ModelUsed, result.latencyMs,
			int32(result.response.Confidence*100))

		payload := map[string]any{
			"risk_level": riskLevelFromScore(result.response.RiskScore),
			"risk_score": result.response.RiskScore,
			"model_used": result.response.ModelUsed,
			"confidence": result.response.Confidence,
		}
		if aiLogID != uuid.Nil {
			payload["ai_log_id"] = aiLogID.String()
		}
		_ = h.emitter.Emit(context.Background(), timeline.Event{
			CaseID:     caseID,
			EventType:  db.TimelineEventTypeAiInsightsGenerated,
			Source:     db.TimelineEventSourceAi,
			ActorID:    &triggeredBy,
			OccurredAt: time.Now(),
			Payload:    payload,
		})
	})

	response.JSON(w, http.StatusAccepted, map[string]any{
		"queued":  true,
		"case_id": caseID.String(),
	})
}

// TriggerAnalysis queues an AI risk analysis for a case without an HTTP request.
// Used by alert automation to immediately analyse auto-created cases.
func (h *Handler) TriggerAnalysis(caseID, triggeredBy uuid.UUID) {
	worker.TrySubmit(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		result, aiErr := h.callAIAnalyseWithTiming(ctx, caseID.String())
		if aiErr != nil {
			promptJSON := fmt.Sprintf(`{"case_id":"%s","action":"analyse_case","trigger":"auto"}`, caseID.String())
			_ = h.logToAILogs(context.Background(), caseID, triggeredBy, "analyse_case",
				promptJSON, fmt.Sprintf("error: %v", aiErr), "unknown", result.latencyMs, 0)
			return
		}
		if err := h.storeInsights(context.Background(), caseID, result.response); err != nil {
			return
		}
		promptJSON := fmt.Sprintf(`{"case_id":"%s","action":"analyse_case","trigger":"auto","triggered_at":"%s"}`,
			caseID.String(), result.startedAt.UTC().Format(time.RFC3339))
		respJSON, _ := json.Marshal(result.response)
		aiLogID := h.logToAILogs(context.Background(), caseID, triggeredBy, "analyse_case",
			promptJSON, string(respJSON), result.response.ModelUsed, result.latencyMs,
			int32(result.response.Confidence*100))
		payload := map[string]any{
			"risk_level":  riskLevelFromScore(result.response.RiskScore),
			"risk_score":  result.response.RiskScore,
			"model_used":  result.response.ModelUsed,
			"auto_trigger": true,
		}
		if aiLogID != uuid.Nil {
			payload["ai_log_id"] = aiLogID.String()
		}
		_ = h.emitter.Emit(context.Background(), timeline.Event{
			CaseID:     caseID,
			EventType:  db.TimelineEventTypeAiInsightsGenerated,
			Source:     db.TimelineEventSourceAi,
			ActorID:    &triggeredBy,
			OccurredAt: time.Now(),
			Payload:    payload,
		})
	})
}

func riskLevelFromScore(score float64) string {
	switch {
	case score >= 75:
		return "high"
	case score >= 40:
		return "medium"
	default:
		return "low"
	}
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

func (h *Handler) callAIAnalyseWithTiming(ctx context.Context, caseID string) (aiAnalyseResult, error) {
	start := time.Now()
	result := aiAnalyseResult{startedAt: start}

	body, _ := json.Marshal(aiAnalyseRequest{CaseID: caseID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.aiServiceURL+"/analyse/case", bytes.NewReader(body))
	if err != nil {
		result.latencyMs = int32(time.Since(start).Milliseconds())
		return result, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.httpClient.Do(req)
	result.latencyMs = int32(time.Since(start).Milliseconds())
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("ai service %d", resp.StatusCode)
	}
	var out aiAnalyseResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return result, err
	}
	result.response = &out
	return result, nil
}

// logToAILogs persists an AI interaction record and returns the created log ID.
// Returns uuid.Nil on error — callers treat this as non-fatal.
func (h *Handler) logToAILogs(ctx context.Context, caseID, triggeredBy uuid.UUID,
	actionType, promptText, responseText, modelUsed string,
	latencyMs, confidencePercent int32) uuid.UUID {

	confStr := sql.NullString{}
	if confidencePercent > 0 {
		confStr = sql.NullString{String: fmt.Sprintf("%d", confidencePercent), Valid: true}
	}
	log, err := h.queries.CreateAILog(ctx, db.CreateAILogParams{
		CaseID:          caseID,
		TriggeredBy:     triggeredBy,
		ActionType:      actionType,
		PromptText:      promptText,
		ResponseText:    responseText,
		ModelUsed:       modelUsed,
		Provider:        "internal_ai_service",
		TokensInput:     0, // Python service doesn't return token counts yet
		TokensOutput:    0,
		LatencyMs:       latencyMs,
		ConfidenceScore: confStr,
		Citations:       json.RawMessage(`[]`),
	})
	if err != nil {
		return uuid.Nil
	}
	return log.ID
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
