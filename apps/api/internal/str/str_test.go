package str_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/str"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

// ── test infra ────────────────────────────────────────────────────────────────

// decodeData unwraps the standard {"data": ...} API envelope.
func decodeData(t *testing.T, body []byte, dst any) {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if err := json.Unmarshal(env.Data, dst); err != nil {
		t.Fatalf("decode data: %v", err)
	}
}

func testDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	ctx := context.Background()

	pgc, err := postgres.Run(ctx,
		"pgvector/pgvector:pg17",
		postgres.WithDatabase("opsowl_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	connStr, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	sqlDB := stdlib.OpenDBFromPool(pool)

	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}
	if err := goose.Up(sqlDB, "../../db/migrations/incremental"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return sqlDB, func() {
		pool.Close()
		_ = pgc.Terminate(ctx)
	}
}

func newJWT(t *testing.T) *auth.JWTManager {
	t.Helper()
	mgr, err := auth.NewJWTManager("../../keys/private.pem", "../../keys/public.pem", time.Hour)
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	return mgr
}

func seedUser(t *testing.T, db *sql.DB, email string, role dbgen.UserRole) uuid.UUID {
	t.Helper()
	hash, _ := auth.HashPassword("TestP@ss1")
	u, err := dbgen.New(db).CreateUser(context.Background(), dbgen.CreateUserParams{
		Email: email, PasswordHash: hash,
		FullName: string(role), Role: role,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u.ID
}

func seedCase(t *testing.T, db *sql.DB, creatorID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO cases (case_number, title, description, status, priority, created_by, organization_id, tags)
		VALUES ('OWL-TEST-' || nextval('case_number_seq')::TEXT,
		        'STR test case', '', 'open', 'high', $1,
		        '00000000-0000-0000-0000-000000000001', '{}')
		RETURNING id`, creatorID).Scan(&id)
	if err != nil {
		t.Fatalf("seed case: %v", err)
	}
	return id
}

func bearer(t *testing.T, mgr *auth.JWTManager, id uuid.UUID, email string, role dbgen.UserRole) string {
	t.Helper()
	tok, err := mgr.Sign(id, email, role)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return "Bearer " + tok
}

func newRouter(db *sql.DB, jwtMgr *auth.JWTManager) http.Handler {
	r := chi.NewRouter()
	str.RegisterRoutes(r, db, jwtMgr, timeline.NoopEmitter{})
	return r
}

func draftBody(basis string) []byte {
	b, _ := json.Marshal(map[string]any{
		"suspicion_basis":  basis,
		"narrative":        "Large cash deposits inconsistent with customer profile",
		"evidence_summary": "3 transactions >10L in 5 days",
		"filing_entity":    map[string]any{"name": "Test Bank", "reporting_officer": "Jane"},
		"subject":          map[string]any{"name": "John Doe", "relationship": "customer"},
		"transactions":     []any{},
	})
	return b
}

// ── DB-layer maker-checker constraint ────────────────────────────────────────

func TestSTR_DBMakerCheckerConstraint_BlocksSameUser(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	mgrID := seedUser(t, db, "mgr@mc.test", dbgen.UserRoleManager)
	caseID := seedCase(t, db, mgrID)
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Insert a filing with submitted_by = approved_by — must violate CHECK
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO str_filings
		  (case_id, org_id, suspicion_basis, submitted_by, approved_by, status)
		VALUES ($1, $2, 'test', $3, $3, 'approved')`,
		caseID, orgID, mgrID,
	)
	if err == nil {
		t.Fatal("expected DB constraint error when submitted_by = approved_by, got nil")
	}
}

func TestSTR_DBMakerCheckerConstraint_AllowsDifferentUsers(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	mgr1 := seedUser(t, db, "mgr1@mc.test", dbgen.UserRoleManager)
	mgr2 := seedUser(t, db, "mgr2@mc.test", dbgen.UserRoleManager)
	caseID := seedCase(t, db, mgr1)
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO str_filings
		  (case_id, org_id, suspicion_basis, submitted_by, approved_by, status)
		VALUES ($1, $2, 'test', $3, $4, 'approved')`,
		caseID, orgID, mgr1, mgr2,
	)
	if err != nil {
		t.Fatalf("different users should satisfy maker-checker: %v", err)
	}
}

// ── Application-layer workflow tests ─────────────────────────────────────────

func TestSTR_FullWorkflow_DraftToApproved(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()
	jwtMgr := newJWT(t)

	mgr1ID := seedUser(t, db, "maker@wf.test", dbgen.UserRoleManager)
	mgr2ID := seedUser(t, db, "checker@wf.test", dbgen.UserRoleManager)
	caseID := seedCase(t, db, mgr1ID)
	router := newRouter(db, jwtMgr)

	// 1. Create draft
	req := httptest.NewRequest(http.MethodPost, "/cases/"+caseID.String()+"/str",
		bytes.NewReader(draftBody("Unusual cash transactions")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgr1ID, "maker@wf.test", dbgen.UserRoleManager))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create draft: expected 201, got %d — %s", rr.Code, rr.Body.String())
	}

	var created str.FilingResponse
	decodeData(t, rr.Body.Bytes(), &created)
	if created.Status != "draft" {
		t.Errorf("expected status=draft, got %s", created.Status)
	}

	// 2. Submit (maker)
	req = httptest.NewRequest(http.MethodPost,
		"/cases/"+caseID.String()+"/str/"+created.ID.String()+"/submit", nil)
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgr1ID, "maker@wf.test", dbgen.UserRoleManager))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("submit: expected 200, got %d — %s", rr.Code, rr.Body.String())
	}

	var submitted str.FilingResponse
	decodeData(t, rr.Body.Bytes(), &submitted)
	if submitted.Status != "submitted" {
		t.Errorf("expected status=submitted, got %s", submitted.Status)
	}

	// 3. Approve — must be DIFFERENT person (maker-checker)
	req = httptest.NewRequest(http.MethodPost,
		"/cases/"+caseID.String()+"/str/"+created.ID.String()+"/approve", nil)
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgr2ID, "checker@wf.test", dbgen.UserRoleManager))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d — %s", rr.Code, rr.Body.String())
	}

	var approved str.FilingResponse
	decodeData(t, rr.Body.Bytes(), &approved)
	if approved.Status != "approved" {
		t.Errorf("expected status=approved, got %s", approved.Status)
	}
}

func TestSTR_SelfApprovalBlocked(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()
	jwtMgr := newJWT(t)

	mgrID := seedUser(t, db, "solo@wf.test", dbgen.UserRoleManager)
	caseID := seedCase(t, db, mgrID)
	router := newRouter(db, jwtMgr)

	// Create draft
	req := httptest.NewRequest(http.MethodPost, "/cases/"+caseID.String()+"/str",
		bytes.NewReader(draftBody("Fraud suspicion")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgrID, "solo@wf.test", dbgen.UserRoleManager))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var created str.FilingResponse
	decodeData(t, rr.Body.Bytes(), &created)

	// Submit
	req = httptest.NewRequest(http.MethodPost,
		"/cases/"+caseID.String()+"/str/"+created.ID.String()+"/submit", nil)
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgrID, "solo@wf.test", dbgen.UserRoleManager))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Self-approval: SAME person tries to approve — must be 403
	req = httptest.NewRequest(http.MethodPost,
		"/cases/"+caseID.String()+"/str/"+created.ID.String()+"/approve", nil)
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgrID, "solo@wf.test", dbgen.UserRoleManager))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("self-approval must return 403, got %d — %s", rr.Code, rr.Body.String())
	}
}

func TestSTR_AnalystCannotCreate(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()
	jwtMgr := newJWT(t)

	mgrID := seedUser(t, db, "mgr@ac.test", dbgen.UserRoleManager)
	analystID := seedUser(t, db, "analyst@ac.test", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, mgrID)
	router := newRouter(db, jwtMgr)

	req := httptest.NewRequest(http.MethodPost, "/cases/"+caseID.String()+"/str",
		bytes.NewReader(draftBody("Unusual activity")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(t, jwtMgr, analystID, "analyst@ac.test", dbgen.UserRoleAnalyst))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("analyst must get 403, got %d", rr.Code)
	}
}

// ── FINnet 2.0 XML export ─────────────────────────────────────────────────────

func TestSTR_FINnetXML_Structure(t *testing.T) {
	filing := &str.FilingResponse{
		ID:              uuid.New(),
		CaseID:          uuid.New(),
		ReportType:      "STR",
		Status:          "approved",
		SuspicionBasis:  "Unusual cash transactions",
		Narrative:       "Three transactions exceeding 10L in one week",
		EvidenceSummary: "Transaction records attached",
		SubmittedByName: "Alice Manager",
		ApprovedByName:  "Bob Admin",
		FilingEntity:    json.RawMessage(`{"name":"Test Bank","reporting_officer":"Alice"}`),
		Subject:         json.RawMessage(`{"name":"John Doe","relationship":"customer","pan":"ABCDE1234F"}`),
		Transactions:    json.RawMessage(`[{"date":"2024-05-01","amount":1500000,"currency":"INR","type":"CREDIT","description":"Cash deposit"}]`),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	now := time.Now()
	filing.SubmittedAt = &now
	filing.ApprovedAt = &now

	xmlData, err := str.BuildFINnetXML(filing)
	if err != nil {
		t.Fatalf("BuildFINnetXML: %v", err)
	}

	// Must be valid XML
	var doc str.FINnetReport
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("invalid XML output: %v\n---\n%s", err, xmlData)
	}

	// Required fields
	if doc.ReportType != "STR" {
		t.Errorf("ReportType: want STR, got %s", doc.ReportType)
	}
	if doc.ReportRefNo == "" {
		t.Error("ReportRefNo must not be empty")
	}
	if doc.FilingDate == "" {
		t.Error("FilingDate must not be empty")
	}
	if doc.FilingEntity.EntityName != "Test Bank" {
		t.Errorf("EntityName: want 'Test Bank', got %s", doc.FilingEntity.EntityName)
	}
	if doc.Subject.Name != "John Doe" {
		t.Errorf("Subject.Name: want 'John Doe', got %s", doc.Subject.Name)
	}
	if doc.Subject.Identifiers.PAN != "ABCDE1234F" {
		t.Errorf("PAN: want 'ABCDE1234F', got %s", doc.Subject.Identifiers.PAN)
	}
	if len(doc.Transactions.Items) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(doc.Transactions.Items))
	}
	if doc.Activity.SuspicionType == "" {
		t.Error("SuspicionType must not be empty")
	}
	if doc.Certification.SubmittedBy != "Alice Manager" {
		t.Errorf("SubmittedBy: want 'Alice Manager', got %s", doc.Certification.SubmittedBy)
	}
	if doc.Certification.ApprovedBy != "Bob Admin" {
		t.Errorf("ApprovedBy: want 'Bob Admin', got %s", doc.Certification.ApprovedBy)
	}
}

// ── goAML 4.0 XML export ──────────────────────────────────────────────────────

func fullGoAMLFiling() *str.FilingResponse {
	now := time.Now()
	f := &str.FilingResponse{
		ID:              uuid.New(),
		CaseID:          uuid.New(),
		ReportType:      "STR",
		Status:          "approved",
		SuspicionBasis:  "Unusual cash transactions",
		Narrative:       "Subject made three cash deposits exceeding ₹10L within five business days",
		EvidenceSummary: "Transaction records: refs TXN-A, TXN-B, TXN-C",
		SubmittedByName: "Alice Manager",
		ApprovedByName:  "Bob Admin",
		FilingEntity: json.RawMessage(`{
			"name": "Test Bank Ltd",
			"rentity_id": "FIU-IN-TEST-0001",
			"rentity_branch": "BRANCH-MUM-001",
			"swift_code": "TESTINBB",
			"officer_first_name": "Alice",
			"officer_last_name": "Manager",
			"officer_gender": "F",
			"officer_title": "Ms",
			"officer_id_number": "ABCDE1234F",
			"officer_id_type": "PAN",
			"officer_phone": "+91-9876543210",
			"officer_email": "alice@testbank.com",
			"officer_occupation": "Chief Compliance Officer",
			"entity_address": "123 Fort Road",
			"entity_city": "Mumbai",
			"entity_state": "MH",
			"entity_zip": "400001"
		}`),
		Subject: json.RawMessage(`{
			"first_name": "John",
			"last_name": "Doe",
			"gender": "M",
			"title": "Mr",
			"birthdate": "1985-06-15",
			"id_type": "PAN",
			"id_number": "QWERT5678Y",
			"nationality": "IN",
			"account_number": "1234567890",
			"account_name": "John Doe",
			"client_number": "CUST-001",
			"account_opened": "2020-01-15",
			"account_balance": 250000,
			"account_status": "A"
		}`),
		Transactions: json.RawMessage(`[
			{
				"date": "2026-04-01",
				"amount": 1500000,
				"currency": "INR",
				"description": "Cash deposit — teller window",
				"transmode_code": "C",
				"transaction_number": "TXN-A",
				"transaction_location": "Mumbai Fort Branch",
				"from_institution": "Test Bank Ltd",
				"from_swift": "TESTINBB",
				"from_account_number": "1234567890"
			},
			{
				"date": "2026-04-03",
				"amount": 1200000,
				"currency": "INR",
				"description": "Cash deposit — ATM lobby",
				"transmode_code": "C",
				"transaction_number": "TXN-B",
				"transaction_location": "Mumbai Fort Branch"
			}
		]`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	f.SubmittedAt = &now
	f.ApprovedAt = &now
	return f
}

func TestGoAML_Validation_PassesOnCompleteData(t *testing.T) {
	errs := str.ValidateGoAMLReadiness(fullGoAMLFiling())
	if len(errs) != 0 {
		t.Errorf("expected no validation errors on complete filing, got: %+v", errs)
	}
}

func TestGoAML_Validation_FailsOnMissingRentityID(t *testing.T) {
	f := fullGoAMLFiling()
	f.FilingEntity = json.RawMessage(`{
		"officer_first_name": "Alice",
		"officer_last_name": "Manager",
		"officer_id_number": "ABCDE1234F",
		"officer_id_type": "PAN"
	}`)
	errs := str.ValidateGoAMLReadiness(f)
	found := false
	for _, e := range errs {
		if e.Field == "filing_entity.rentity_id" {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for missing rentity_id")
	}
}

func TestGoAML_Validation_FailsOnNoTransactions(t *testing.T) {
	f := fullGoAMLFiling()
	f.Transactions = json.RawMessage(`[]`)
	errs := str.ValidateGoAMLReadiness(f)
	found := false
	for _, e := range errs {
		if e.Field == "transactions" {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for zero transactions")
	}
}

func TestGoAML_Validation_FailsOnMissingTransactionMode(t *testing.T) {
	f := fullGoAMLFiling()
	f.Transactions = json.RawMessage(`[{"date":"2026-04-01","amount":1500000}]`)
	errs := str.ValidateGoAMLReadiness(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Field, "transmode_code") {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for missing transmode_code")
	}
}

func TestGoAML_XML_Structure(t *testing.T) {
	xmlData, errs := str.BuildGoAMLXML(fullGoAMLFiling())
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got: %+v", errs)
	}
	if xmlData == nil {
		t.Fatal("expected XML data, got nil")
	}

	var doc str.GoAMLReport
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("invalid XML output: %v\n---\n%s", err, xmlData)
	}

	if doc.RentityID != "FIU-IN-TEST-0001" {
		t.Errorf("RentityID: want FIU-IN-TEST-0001, got %s", doc.RentityID)
	}
	if doc.ReportCode != "STR" {
		t.Errorf("ReportCode: want STR, got %s", doc.ReportCode)
	}
	if doc.SubmissionCode != "E" {
		t.Errorf("SubmissionCode: want E, got %s", doc.SubmissionCode)
	}
	if doc.ReportingPerson.FirstName != "Alice" {
		t.Errorf("OfficerFirstName: want Alice, got %s", doc.ReportingPerson.FirstName)
	}
	if doc.ReportingPerson.IDNumber != "ABCDE1234F" {
		t.Errorf("OfficerIDNumber: want ABCDE1234F, got %s", doc.ReportingPerson.IDNumber)
	}
	if doc.Reason == "" {
		t.Error("Reason must not be empty")
	}
	if len(doc.Transactions) != 2 {
		t.Errorf("expected 2 transactions, got %d", len(doc.Transactions))
	}
	if doc.Transactions[0].TransmodeCode != "C" {
		t.Errorf("TransmodeCode: want C, got %s", doc.Transactions[0].TransmodeCode)
	}
	if doc.Transactions[0].AmountLocal != "1500000.00" {
		t.Errorf("AmountLocal: want 1500000.00, got %s", doc.Transactions[0].AmountLocal)
	}
	if doc.Transactions[0].FromMyClient == nil {
		t.Error("FromMyClient must not be nil")
	}
	if doc.Transactions[0].FromMyClient.FromPerson.FirstName != "John" {
		t.Errorf("FromPerson.FirstName: want John, got %s", doc.Transactions[0].FromMyClient.FromPerson.FirstName)
	}
	if len(doc.Indicators.Indicators) == 0 {
		t.Error("Indicators must not be empty")
	}
}

func TestGoAML_XML_SubjectNameFallback(t *testing.T) {
	f := fullGoAMLFiling()
	// subject has only "name" (not split first/last)
	f.Subject = json.RawMessage(`{
		"name": "Jane Smith",
		"id_number": "PQRST9012Z",
		"id_type": "PAN"
	}`)
	xmlData, errs := str.BuildGoAMLXML(f)
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got: %+v", errs)
	}
	var doc str.GoAMLReport
	_ = xml.Unmarshal(xmlData, &doc)
	if doc.Transactions[0].FromMyClient.FromPerson.FirstName != "Jane" {
		t.Errorf("name split: want first=Jane, got %s", doc.Transactions[0].FromMyClient.FromPerson.FirstName)
	}
	if doc.Transactions[0].FromMyClient.FromPerson.LastName != "Smith" {
		t.Errorf("name split: want last=Smith, got %s", doc.Transactions[0].FromMyClient.FromPerson.LastName)
	}
}

func TestGoAML_HTTP_ValidateEndpoint(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()
	jwtMgr := newJWT(t)

	mgrID := seedUser(t, db, "mgr@goaml.test", dbgen.UserRoleManager)
	caseID := seedCase(t, db, mgrID)
	router := newRouter(db, jwtMgr)

	// Create a minimal draft (missing goAML fields)
	req := httptest.NewRequest(http.MethodPost, "/cases/"+caseID.String()+"/str",
		bytes.NewReader(draftBody("Unusual cash")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgrID, "mgr@goaml.test", dbgen.UserRoleManager))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var created str.FilingResponse
	decodeData(t, rr.Body.Bytes(), &created)

	// Validate — expect errors since rentity_id and transactions are missing
	req = httptest.NewRequest(http.MethodGet,
		"/cases/"+caseID.String()+"/str/"+created.ID.String()+"/validate/goaml", nil)
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgrID, "mgr@goaml.test", dbgen.UserRoleManager))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("validate: expected 200, got %d — %s", rr.Code, rr.Body.String())
	}

	var body struct {
		Data struct {
			Ready  bool                      `json:"ready"`
			Errors []str.GoAMLValidationError `json:"errors"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Ready {
		t.Error("expected ready=false for incomplete filing")
	}
	if len(body.Data.Errors) == 0 {
		t.Error("expected validation errors, got none")
	}
}

func TestGoAML_HTTP_ExportBlockedForAnalyst(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()
	jwtMgr := newJWT(t)

	mgrID := seedUser(t, db, "mgr@exp.test", dbgen.UserRoleManager)
	analystID := seedUser(t, db, "ana@exp.test", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, mgrID)
	router := newRouter(db, jwtMgr)

	req := httptest.NewRequest(http.MethodPost, "/cases/"+caseID.String()+"/str",
		bytes.NewReader(draftBody("Test")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(t, jwtMgr, mgrID, "mgr@exp.test", dbgen.UserRoleManager))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var created str.FilingResponse
	decodeData(t, rr.Body.Bytes(), &created)

	// Analyst tries to export goAML — must be 403
	req = httptest.NewRequest(http.MethodGet,
		"/cases/"+caseID.String()+"/str/"+created.ID.String()+"/export/goaml", nil)
	req.Header.Set("Authorization", bearer(t, jwtMgr, analystID, "ana@exp.test", dbgen.UserRoleAnalyst))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("analyst must get 403 on goAML export, got %d", rr.Code)
	}
}

func TestSTR_FINnetXML_AadhaarMasked(t *testing.T) {
	filing := &str.FilingResponse{
		ID:             uuid.New(),
		CaseID:         uuid.New(),
		ReportType:     "STR",
		SuspicionBasis: "test",
		FilingEntity:   json.RawMessage(`{}`),
		Subject:        json.RawMessage(`{"name":"Jane","aadhaar_ref":"123456789012"}`),
		Transactions:   json.RawMessage(`[]`),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	xmlData, err := str.BuildFINnetXML(filing)
	if err != nil {
		t.Fatalf("BuildFINnetXML: %v", err)
	}

	// Raw Aadhaar must NOT appear in the XML
	if bytes.Contains(xmlData, []byte("123456789012")) {
		t.Error("raw Aadhaar number must NOT appear in FINnet XML — DPDP Act 2023 s.4 violation")
	}
}
