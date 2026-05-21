package report

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

// RegulatoryReportType represents the type of regulatory filing required.
type RegulatoryReportType string

const (
	ReportTypeSTR RegulatoryReportType = "STR" // Suspicious Transaction Report (FIU-IND)
	ReportTypeSAR RegulatoryReportType = "SAR" // Suspicious Activity Report (US FinCEN)
	ReportTypeCTR RegulatoryReportType = "CTR" // Cash Transaction Report
)

// RegulatoryReport is the structured report generated for regulatory submissions.
// Format follows FIU-IND guidelines (India PMLA 2002 / DPDP 2023).
type RegulatoryReport struct {
	ReportID      string               `json:"report_id"`
	ReportType    RegulatoryReportType `json:"report_type"`
	GeneratedAt   time.Time            `json:"generated_at"`
	GeneratedBy   string               `json:"generated_by"`
	CaseID        string               `json:"case_id"`
	CaseNumber    string               `json:"case_number"`
	FilingEntity  FilingEntity         `json:"filing_entity"`
	Subject       SubjectDetails       `json:"subject"`
	Transactions  []TransactionRecord  `json:"transactions"`
	Narrative     string               `json:"narrative"`
	SuspicionBasis string              `json:"suspicion_basis"`
	EvidenceSummary string             `json:"evidence_summary"`
	Attachments   []AttachmentRef      `json:"attachments"`
	Status        string               `json:"status"` // draft | final
}

type FilingEntity struct {
	Name           string `json:"name"`
	BranchCode     string `json:"branch_code,omitempty"`
	LEICode        string `json:"lei_code,omitempty"`
	ReportingOfficer string `json:"reporting_officer"`
	ContactEmail   string `json:"contact_email,omitempty"`
}

type SubjectDetails struct {
	Name        string `json:"name"`
	Relationship string `json:"relationship,omitempty"` // customer | counterparty | third_party
	AccountRef  string `json:"account_ref,omitempty"`
	PAN         string `json:"pan,omitempty"`    // India-specific
	AadhaarRef  string `json:"aadhaar_ref,omitempty"`
	RiskRating  string `json:"risk_rating,omitempty"`
}

type TransactionRecord struct {
	Date        string  `json:"date"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Reference   string  `json:"reference,omitempty"`
}

type AttachmentRef struct {
	DocumentID string `json:"document_id"`
	FileName   string `json:"file_name"`
	Category   string `json:"category"`
}

type generateRegulatoryReportRequest struct {
	ReportType      RegulatoryReportType `json:"report_type"`
	FilingEntity    FilingEntity         `json:"filing_entity"`
	Subject         SubjectDetails       `json:"subject"`
	Transactions    []TransactionRecord  `json:"transactions"`
	SuspicionBasis  string               `json:"suspicion_basis"`
	EvidenceSummary string               `json:"evidence_summary"`
}

// GenerateRegulatoryReport creates a structured STR/SAR/CTR report for a case.
func (h *Handler) GenerateRegulatoryReport(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req generateRegulatoryReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.ReportType == "" {
		req.ReportType = ReportTypeSTR
	}
	if req.SuspicionBasis == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "suspicion_basis is required")
		return
	}

	report, err := h.buildRegulatoryReport(r.Context(), caseID, req)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not generate regulatory report")
		return
	}
	response.JSON(w, http.StatusOK, report)
}

func (h *Handler) buildRegulatoryReport(ctx context.Context, caseID uuid.UUID, req generateRegulatoryReportRequest) (*RegulatoryReport, error) {
	row, err := h.queries.GetCaseByID(ctx, caseID)
	if err != nil {
		return nil, err
	}

	// Fetch documents attached to the case for evidence references
	docs, _ := h.queries.ListDocumentsByCase(ctx, db.ListDocumentsByCaseParams{
		CaseID:   caseID,
		RowLimit: 50,
	})

	attachments := make([]AttachmentRef, 0, len(docs))
	for _, d := range docs {
		attachments = append(attachments, AttachmentRef{
			DocumentID: d.ID.String(),
			FileName:   d.OriginalFilename,
			Category:   string(d.ProcessingStatus),
		})
	}

	// Build narrative from case description + evidence summary
	narrative := fmt.Sprintf("Case #%s: %s\n\n%s", row.CaseNumber, row.Description, req.EvidenceSummary)

	report := &RegulatoryReport{
		ReportID:        uuid.New().String(),
		ReportType:      req.ReportType,
		GeneratedAt:     time.Now().UTC(),
		CaseID:          caseID.String(),
		CaseNumber:      row.CaseNumber,
		FilingEntity:    req.FilingEntity,
		Subject:         req.Subject,
		Transactions:    req.Transactions,
		Narrative:       narrative,
		SuspicionBasis:  req.SuspicionBasis,
		EvidenceSummary: req.EvidenceSummary,
		Attachments:     attachments,
		Status:          "draft",
	}
	return report, nil
}

// ListRegulatoryTemplates returns available report type templates with required fields.
func (h *Handler) ListRegulatoryTemplates(w http.ResponseWriter, r *http.Request) {
	templates := []map[string]any{
		{
			"type":        ReportTypeSTR,
			"name":        "Suspicious Transaction Report",
			"authority":   "FIU-IND (Financial Intelligence Unit - India)",
			"regulation":  "Prevention of Money Laundering Act 2002 (PMLA)",
			"deadline":    "Within 7 days of transaction date",
			"required_fields": []string{"suspicion_basis", "transactions", "subject.name", "subject.account_ref"},
		},
		{
			"type":        ReportTypeSAR,
			"name":        "Suspicious Activity Report",
			"authority":   "FinCEN (US Financial Crimes Enforcement Network)",
			"regulation":  "Bank Secrecy Act (BSA)",
			"deadline":    "Within 30 days of detection",
			"required_fields": []string{"suspicion_basis", "subject.name", "evidence_summary"},
		},
		{
			"type":        ReportTypeCTR,
			"name":        "Cash Transaction Report",
			"authority":   "FIU-IND",
			"regulation":  "PMLA 2002",
			"deadline":    "Within 15 days of the month-end in which transaction occurred",
			"required_fields": []string{"transactions", "subject.name", "subject.pan"},
		},
	}
	response.JSON(w, http.StatusOK, map[string]any{"templates": templates})
}
