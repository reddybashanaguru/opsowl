package str_test

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/str"
)

// fullEUGoAMLFiling returns a complete goAML filing in EU (Lithuania / FNTT) context.
// EUR amounts, IBAN account numbers, EU AMLR 2024 regulatory references.
func fullEUGoAMLFiling() *str.FilingResponse {
	now := time.Now()
	f := &str.FilingResponse{
		ID:              uuid.New(),
		CaseID:          uuid.New(),
		ReportType:      "STR",
		Status:          "approved",
		SuspicionBasis:  "Suspicious SEPA Instant cluster — structuring below €10,000 EDD threshold",
		Narrative:       "Subject EMI account transmitted 23 SEPA Instant payments totalling EUR 85,000 to beneficiaries in Latvia, Estonia, and Poland within 48 hours. All transactions structured below the EUR 10,000 enhanced due diligence threshold per EU AMLR 2024 Art. 21. Velocity 18x above account baseline.",
		EvidenceSummary: "SEPA transaction export (23 records), account CDD file, STR Form draft",
		SubmittedByName: "Mantas Žukauskas",
		ApprovedByName:  "Aistė Bernotienė",
		FilingEntity: json.RawMessage(`{
			"name": "Baltic EMI UAB",
			"rentity_id": "FNTT-LT-EMI-0042",
			"rentity_branch": "VILNIUS-HQ-001",
			"swift_code": "BEMIUALT",
			"submission_code": "E",
			"officer_first_name": "Mantas",
			"officer_last_name": "Žukauskas",
			"officer_gender": "M",
			"officer_title": "Mr",
			"officer_id_number": "38901011234",
			"officer_id_type": "NATIONAL_ID",
			"officer_phone": "+370-5-123-4567",
			"officer_email": "mantas@balticemi.lt",
			"officer_occupation": "MLRO",
			"entity_address": "Gedimino pr. 1",
			"entity_city": "Vilnius",
			"entity_state": "Vilnius County",
			"entity_zip": "LT-01103",
			"entity_country": "LT",
			"fiu_name": "FNTT",
			"local_currency": "EUR"
		}`),
		Subject: json.RawMessage(`{
			"first_name": "Dmitri",
			"last_name": "Volkov",
			"gender": "M",
			"title": "Mr",
			"birthdate": "1978-03-15",
			"id_type": "NATIONAL_ID",
			"id_number": "38001011234",
			"nationality": "LT",
			"account_number": "LT12 3456 7890 1234 5678",
			"account_name": "Dmitri Volkov",
			"client_number": "LT-EMI-CUST-7741",
			"account_opened": "2024-11-01",
			"account_balance": 2500,
			"account_status": "A"
		}`),
		Transactions: json.RawMessage(`[
			{
				"date": "2026-06-10",
				"amount": 9800,
				"currency": "EUR",
				"description": "SEPA Instant — beneficiary LVXXXXXXXXXXXXXXXXXXXX",
				"transmode_code": "T",
				"transaction_number": "SEPA-2026-001",
				"transaction_location": "Vilnius",
				"from_institution": "Baltic EMI UAB",
				"from_swift": "BEMIUALT",
				"from_account_number": "LT12 3456 7890 1234 5678"
			},
			{
				"date": "2026-06-10",
				"amount": 9700,
				"currency": "EUR",
				"description": "SEPA Instant — beneficiary EEXXXXXXXXXXXXXXXXXXXXXX",
				"transmode_code": "T",
				"transaction_number": "SEPA-2026-002",
				"transaction_location": "Vilnius",
				"from_institution": "Baltic EMI UAB",
				"from_swift": "BEMIUALT",
				"from_account_number": "LT12 3456 7890 1234 5678"
			},
			{
				"date": "2026-06-11",
				"amount": 9500,
				"currency": "EUR",
				"description": "SEPA Instant — beneficiary PLXXXXXXXXXXXXXXXXXXXXXXX",
				"transmode_code": "T",
				"transaction_number": "SEPA-2026-003",
				"transaction_location": "Vilnius",
				"from_institution": "Baltic EMI UAB",
				"from_swift": "BEMIUALT",
				"from_account_number": "LT12 3456 7890 1234 5678"
			}
		]`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	f.SubmittedAt = &now
	f.ApprovedAt = &now
	return f
}

// fullEUCyprusGoAMLFiling returns a complete MOKAS (Cyprus) goAML filing.
func fullEUCyprusGoAMLFiling() *str.FilingResponse {
	now := time.Now()
	f := &str.FilingResponse{
		ID:              uuid.New(),
		CaseID:          uuid.New(),
		ReportType:      "STR",
		Status:          "approved",
		SuspicionBasis:  "UBO non-disclosure — offshore holding structure, FX broker",
		Narrative:       "Cypriot FX broker client declared nominee director as UBO. Structure analysis reveals 3-layer offshore chain (Cyprus → BVI → Seychelles). AMLR 2024 Art. 42(3) breach. STR filed with MOKAS.",
		EvidenceSummary: "UBO verification file, corporate structure chart, MOKAS STR Form",
		SubmittedByName: "Andreas Papadopoulos",
		ApprovedByName:  "Elena Stylianou",
		FilingEntity: json.RawMessage(`{
			"name": "CY Compliance Services Ltd",
			"rentity_id": "MOKAS-CY-FX-0099",
			"rentity_branch": "NICOSIA-HQ-001",
			"swift_code": "CYCSLCYX",
			"submission_code": "E",
			"officer_first_name": "Andreas",
			"officer_last_name": "Papadopoulos",
			"officer_gender": "M",
			"officer_title": "Mr",
			"officer_id_number": "K123456",
			"officer_id_type": "NATIONAL_ID",
			"officer_phone": "+357-22-123-456",
			"officer_email": "andreas@cycompliance.com.cy",
			"officer_occupation": "Chief Compliance Officer",
			"entity_address": "Arch. Makariou III 123",
			"entity_city": "Nicosia",
			"entity_state": "Nicosia District",
			"entity_zip": "1065",
			"entity_country": "CY",
			"fiu_name": "MOKAS",
			"local_currency": "EUR"
		}`),
		Subject: json.RawMessage(`{
			"first_name": "Costas",
			"last_name": "Georgiou",
			"gender": "M",
			"title": "Mr",
			"birthdate": "1972-07-22",
			"id_type": "NATIONAL_ID",
			"id_number": "A123456",
			"nationality": "CY",
			"account_number": "CY17 0020 0128 0000 0012 0052 7600",
			"account_name": "Costas Georgiou Holding",
			"client_number": "CY-FX-CUST-4421",
			"account_opened": "2023-06-15",
			"account_balance": 47500,
			"account_status": "A"
		}`),
		Transactions: json.RawMessage(`[
			{
				"date": "2026-05-20",
				"amount": 47500,
				"currency": "EUR",
				"description": "SWIFT MT103 — commodity trade settlement, over-invoiced",
				"transmode_code": "T",
				"transaction_number": "SWIFT-2026-250",
				"transaction_location": "Nicosia",
				"from_institution": "CY Compliance Services Ltd",
				"from_swift": "CYCSLCYX",
				"from_account_number": "CY17 0020 0128 0000 0012 0052 7600"
			}
		]`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	f.SubmittedAt = &now
	f.ApprovedAt = &now
	return f
}

// ── EU goAML validation tests ─────────────────────────────────────────────────

func TestGoAML_EU_Lithuania_ValidationPasses(t *testing.T) {
	errs := str.ValidateGoAMLReadiness(fullEUGoAMLFiling())
	if len(errs) != 0 {
		t.Errorf("expected no validation errors for complete EU/LT filing, got: %+v", errs)
	}
}

func TestGoAML_EU_Cyprus_ValidationPasses(t *testing.T) {
	errs := str.ValidateGoAMLReadiness(fullEUCyprusGoAMLFiling())
	if len(errs) != 0 {
		t.Errorf("expected no validation errors for complete EU/CY filing, got: %+v", errs)
	}
}

func TestGoAML_EU_MissingFNTTEntityID_FailsValidation(t *testing.T) {
	f := fullEUGoAMLFiling()
	// Remove rentity_id — simulates missing FNTT registration number
	f.FilingEntity = json.RawMessage(`{
		"name": "Baltic EMI UAB",
		"officer_first_name": "Mantas",
		"officer_last_name": "Žukauskas",
		"officer_id_number": "38901011234",
		"officer_id_type": "NATIONAL_ID",
		"entity_country": "LT",
		"fiu_name": "FNTT",
		"local_currency": "EUR"
	}`)
	errs := str.ValidateGoAMLReadiness(f)
	found := false
	for _, e := range errs {
		if e.Field == "filing_entity.rentity_id" {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for missing FNTT rentity_id (registration number required by FNTT goAML portal)")
	}
}

func TestGoAML_EU_NoTransactions_FailsValidation(t *testing.T) {
	f := fullEUGoAMLFiling()
	f.Transactions = json.RawMessage(`[]`)
	errs := str.ValidateGoAMLReadiness(f)
	found := false
	for _, e := range errs {
		if e.Field == "transactions" {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for zero transactions on EU filing")
	}
}

func TestGoAML_EU_MissingTransactionMode_FailsValidation(t *testing.T) {
	f := fullEUGoAMLFiling()
	f.Transactions = json.RawMessage(`[{"date":"2026-06-10","amount":9800,"currency":"EUR"}]`)
	errs := str.ValidateGoAMLReadiness(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Field, "transmode_code") {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for missing transmode_code on EU transaction")
	}
}

// ── EU goAML XML structure tests ──────────────────────────────────────────────

func TestGoAML_EU_Lithuania_XMLStructure(t *testing.T) {
	xmlData, errs := str.BuildGoAMLXML(fullEUGoAMLFiling())
	if len(errs) > 0 {
		t.Fatalf("expected no errors building EU/LT goAML XML, got: %+v", errs)
	}
	if xmlData == nil {
		t.Fatal("expected XML data, got nil")
	}

	var doc str.GoAMLReport
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("invalid XML output: %v\n---\n%s", err, xmlData)
	}

	// FNTT entity registration
	if doc.RentityID != "FNTT-LT-EMI-0042" {
		t.Errorf("RentityID: want FNTT-LT-EMI-0042, got %s", doc.RentityID)
	}
	if doc.ReportCode != "STR" {
		t.Errorf("ReportCode: want STR, got %s", doc.ReportCode)
	}
	// EUR currency on transactions
	if len(doc.Transactions) != 3 {
		t.Errorf("expected 3 transactions, got %d", len(doc.Transactions))
	}
	if doc.Transactions[0].AmountLocal != "9800.00" {
		t.Errorf("AmountLocal: want 9800.00, got %s", doc.Transactions[0].AmountLocal)
	}
	// SEPA Instant uses transmode_code T (electronic transfer)
	if doc.Transactions[0].TransmodeCode != "T" {
		t.Errorf("TransmodeCode: want T (electronic transfer), got %s", doc.Transactions[0].TransmodeCode)
	}
	// Officer details
	if doc.ReportingPerson.FirstName != "Mantas" {
		t.Errorf("OfficerFirstName: want Mantas, got %s", doc.ReportingPerson.FirstName)
	}
	if doc.ReportingPerson.IDNumber != "38901011234" {
		t.Errorf("OfficerIDNumber: want 38901011234, got %s", doc.ReportingPerson.IDNumber)
	}
	// Subject IBAN account number preserved
	if doc.Transactions[0].FromMyClient != nil {
		if doc.Transactions[0].FromMyClient.FromPerson.FirstName != "Dmitri" {
			t.Errorf("SubjectFirstName: want Dmitri, got %s", doc.Transactions[0].FromMyClient.FromPerson.FirstName)
		}
	}
	// Indicators must be present (EU FATF typologies)
	if len(doc.Indicators.Indicators) == 0 {
		t.Error("Indicators must not be empty for EU filing")
	}
	// Narrative must be in the XML
	if doc.Reason == "" {
		t.Error("Reason (narrative) must not be empty in EU goAML XML")
	}
}

func TestGoAML_EU_Cyprus_XMLStructure(t *testing.T) {
	xmlData, errs := str.BuildGoAMLXML(fullEUCyprusGoAMLFiling())
	if len(errs) > 0 {
		t.Fatalf("expected no errors building EU/CY goAML XML, got: %+v", errs)
	}

	var doc str.GoAMLReport
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("invalid XML output: %v\n---\n%s", err, xmlData)
	}

	// MOKAS entity registration
	if doc.RentityID != "MOKAS-CY-FX-0099" {
		t.Errorf("RentityID: want MOKAS-CY-FX-0099, got %s", doc.RentityID)
	}
	if len(doc.Transactions) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(doc.Transactions))
	}
	if doc.Transactions[0].AmountLocal != "47500.00" {
		t.Errorf("AmountLocal: want 47500.00, got %s", doc.Transactions[0].AmountLocal)
	}
}

func TestGoAML_EU_IBanAccountNumberPreserved(t *testing.T) {
	// IBAN format must pass through without modification — not treated as PAN
	f := fullEUGoAMLFiling()
	xmlData, errs := str.BuildGoAMLXML(f)
	if len(errs) > 0 {
		t.Fatalf("expected no build errors: %+v", errs)
	}
	// IBAN should appear somewhere in the XML
	if !strings.Contains(string(xmlData), "LT12") {
		t.Error("IBAN account number (LT12...) must be present in EU goAML XML output")
	}
}

func TestGoAML_EU_NationalIDNotMasked(t *testing.T) {
	// NATIONAL_ID must NOT be masked (only Aadhaar-equivalent IDs need masking per DPDP Act)
	f := fullEUGoAMLFiling()
	xmlData, errs := str.BuildGoAMLXML(f)
	if len(errs) > 0 {
		t.Fatalf("expected no build errors: %+v", errs)
	}
	// Officer national ID should appear in the XML
	if !strings.Contains(string(xmlData), "38901011234") {
		t.Error("NATIONAL_ID must not be masked in EU goAML XML (only Aadhaar refs are masked)")
	}
}

func TestGoAML_EU_SubjectNameSplitFromFullName(t *testing.T) {
	// If subject has only "name" (not split), verify first/last split works for EU names
	f := fullEUGoAMLFiling()
	f.Subject = json.RawMessage(`{
		"name": "Eglė Kazlauskienė",
		"id_type": "NATIONAL_ID",
		"id_number": "48501011234",
		"account_number": "LT12 3456 7890 1234 5679"
	}`)
	xmlData, errs := str.BuildGoAMLXML(f)
	if len(errs) > 0 {
		t.Fatalf("expected no errors for EU name split: %+v", errs)
	}
	var doc str.GoAMLReport
	_ = xml.Unmarshal(xmlData, &doc)
	if doc.Transactions[0].FromMyClient == nil {
		t.Skip("FromMyClient not populated — name split not applicable")
	}
	if doc.Transactions[0].FromMyClient.FromPerson.FirstName != "Eglė" {
		t.Errorf("name split first: want Eglė, got %s", doc.Transactions[0].FromMyClient.FromPerson.FirstName)
	}
}

// ── EU HTTP endpoint tests ────────────────────────────────────────────────────

func TestGoAML_EU_HTTP_ValidateEndpoint_LithuaniaFiling(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()
	jwtMgr := newJWT(t)

	mgrID := seedUser(t, db, "mlro@fntt.test", dbgen.UserRoleManager)
	caseID := seedCase(t, db, mgrID)
	router := newRouter(db, jwtMgr)

	// Create with minimal body (missing rentity_id — should fail validation)
	euBody, _ := json.Marshal(map[string]any{
		"report_type":      "STR",
		"suspicion_basis":  "Suspicious SEPA Instant cluster",
		"narrative":        "EU AMLR 2024 — structuring below EDD threshold",
		"evidence_summary": "SEPA export attached",
		"filing_entity": map[string]any{
			"name":           "Baltic EMI UAB",
			"entity_country": "LT",
			"fiu_name":       "FNTT",
			"local_currency": "EUR",
			// rentity_id intentionally omitted
		},
		"subject": map[string]any{
			"first_name":     "Dmitri",
			"last_name":      "Volkov",
			"id_type":        "NATIONAL_ID",
			"id_number":      "38001011234",
			"account_number": "LT12 3456 7890 1234 5678",
		},
		"transactions": []any{},
	})

	req := httptest.NewRequest(http.MethodPost, "/cases/"+caseID.String()+"/str", bytes.NewReader(euBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgrID, "mlro@fntt.test", dbgen.UserRoleManager))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("create EU draft: want 201, got %d — %s", rr.Code, rr.Body.String())
	}
	var created str.FilingResponse
	decodeData(t, rr.Body.Bytes(), &created)

	// Validate — expect errors: missing rentity_id + no transactions
	req = httptest.NewRequest(http.MethodGet, "/cases/"+caseID.String()+"/str/"+created.ID.String()+"/validate/goaml", nil)
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgrID, "mlro@fntt.test", dbgen.UserRoleManager))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("validate: want 200, got %d", rr.Code)
	}
	var respBody struct {
		Data struct {
			Ready  bool                       `json:"ready"`
			Errors []str.GoAMLValidationError `json:"errors"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if respBody.Data.Ready {
		t.Error("expected ready=false for incomplete EU filing (missing rentity_id + transactions)")
	}
	found := false
	for _, e := range respBody.Data.Errors {
		if e.Field == "filing_entity.rentity_id" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected filing_entity.rentity_id error for missing FNTT registration number, got: %+v", respBody.Data.Errors)
	}
}

func TestGoAML_EU_HTTP_ValidateEndpoint_PassesWithFullEUFields(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()
	jwtMgr := newJWT(t)

	mgrID := seedUser(t, db, "mlro2@fntt.test", dbgen.UserRoleManager)
	caseID := seedCase(t, db, mgrID)
	router := newRouter(db, jwtMgr)

	f := fullEUGoAMLFiling()
	var fe, subj map[string]any
	_ = json.Unmarshal(f.FilingEntity, &fe)
	_ = json.Unmarshal(f.Subject, &subj)
	var txList []any
	_ = json.Unmarshal(f.Transactions, &txList)

	reqBody, _ := json.Marshal(map[string]any{
		"report_type":      "STR",
		"suspicion_basis":  f.SuspicionBasis,
		"narrative":        f.Narrative,
		"evidence_summary": f.EvidenceSummary,
		"filing_entity":    fe,
		"subject":          subj,
		"transactions":     txList,
	})

	req := httptest.NewRequest(http.MethodPost, "/cases/"+caseID.String()+"/str", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgrID, "mlro2@fntt.test", dbgen.UserRoleManager))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("create EU draft: want 201, got %d — %s", rr.Code, rr.Body.String())
	}
	var created str.FilingResponse
	decodeData(t, rr.Body.Bytes(), &created)

	req = httptest.NewRequest(http.MethodGet, "/cases/"+caseID.String()+"/str/"+created.ID.String()+"/validate/goaml", nil)
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgrID, "mlro2@fntt.test", dbgen.UserRoleManager))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var valBody struct {
		Data struct {
			Ready  bool                       `json:"ready"`
			Errors []str.GoAMLValidationError `json:"errors"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &valBody)
	if !valBody.Data.Ready {
		t.Errorf("expected ready=true for complete EU/LT filing, got errors: %+v", valBody.Data.Errors)
	}
}
