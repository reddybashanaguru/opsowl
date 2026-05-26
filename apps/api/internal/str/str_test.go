package str_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
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
	str.RegisterRoutes(r, db, jwtMgr)
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
