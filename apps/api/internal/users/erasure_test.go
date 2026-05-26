package users_test

import (
	"context"
	"database/sql"
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
	"github.com/reddybashanaguru/opsowl/internal/users"
)

// ── test infra ────────────────────────────────────────────────────────────────

func erasureTestDB(t *testing.T) (*sql.DB, func()) {
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
		t.Fatalf("start postgres: %v", err)
	}
	connStr, _ := pgc.ConnectionString(ctx, "sslmode=disable")
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	db := stdlib.OpenDBFromPool(pool)

	goose.SetBaseFS(nil)
	_ = goose.SetDialect("postgres")
	if err := goose.Up(db, "../../db/migrations/incremental"); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	return db, func() { pool.Close(); _ = pgc.Terminate(ctx) }
}

func erasureJWT(t *testing.T) *auth.JWTManager {
	t.Helper()
	mgr, err := auth.NewJWTManager("../../keys/private.pem", "../../keys/public.pem", time.Hour)
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	return mgr
}

func seedErasureUser(t *testing.T, db *sql.DB, email string, role dbgen.UserRole) uuid.UUID {
	t.Helper()
	hash, _ := auth.HashPassword("TestP@ss1")
	u, err := dbgen.New(db).CreateUser(context.Background(), dbgen.CreateUserParams{
		Email: email, PasswordHash: hash,
		FullName: "Test " + string(role), Role: role,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u.ID
}

func seedErasureCase(t *testing.T, db *sql.DB, creatorID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO cases (case_number, title, description, status, priority, created_by, organization_id, tags)
		VALUES ('OWL-ERASE-' || nextval('case_number_seq')::TEXT, 'Erasure test', '', 'open', 'high', $1,
		        '00000000-0000-0000-0000-000000000001', '{}')
		RETURNING id`, creatorID).Scan(&id)
	if err != nil {
		t.Fatalf("seed case: %v", err)
	}
	return id
}

func newErasureRouter(db *sql.DB, jwtMgr *auth.JWTManager) http.Handler {
	q := dbgen.New(db)
	repo := users.NewRepository(q).WithDB(db)
	authSvc := auth.NewService(auth.NewRepository(q), jwtMgr)
	svc := users.NewService(repo, authSvc)
	h := users.NewHandler(svc)
	r := chi.NewRouter()
	users.RegisterRoutes(r, h, jwtMgr)
	return r
}

func bearerToken(t *testing.T, mgr *auth.JWTManager, id uuid.UUID, email string, role dbgen.UserRole) string {
	t.Helper()
	tok, err := mgr.Sign(id, email, role)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return "Bearer " + tok
}

// ── C3 erasure tests ──────────────────────────────────────────────────────────

func TestErasure_AllowedWhenNoSTRHold(t *testing.T) {
	db, teardown := erasureTestDB(t)
	defer teardown()
	jwtMgr := erasureJWT(t)

	adminID := seedErasureUser(t, db, "admin@erase.test", dbgen.UserRoleAdmin)
	targetID := seedErasureUser(t, db, "target@erase.test", dbgen.UserRoleAnalyst)
	router := newErasureRouter(db, jwtMgr)

	req := httptest.NewRequest(http.MethodDelete, "/users/"+targetID.String()+"/data", nil)
	req.Header.Set("Authorization", bearerToken(t, jwtMgr, adminID, "admin@erase.test", dbgen.UserRoleAdmin))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — %s", rr.Code, rr.Body.String())
	}

	// Verify PII is erased in DB
	var email, fullName string
	var erasedAt sql.NullTime
	err := db.QueryRowContext(context.Background(),
		"SELECT email, full_name, erased_at FROM users WHERE id = $1", targetID).
		Scan(&email, &fullName, &erasedAt)
	if err != nil {
		t.Fatalf("verify erasure: %v", err)
	}
	if email == "target@erase.test" {
		t.Error("email should be anonymised after erasure")
	}
	if fullName != "[ERASED]" {
		t.Errorf("full_name should be [ERASED], got %s", fullName)
	}
	if !erasedAt.Valid {
		t.Error("erased_at should be set after erasure")
	}
}

func TestErasure_BlockedByPMLAHold(t *testing.T) {
	db, teardown := erasureTestDB(t)
	defer teardown()
	jwtMgr := erasureJWT(t)

	adminID := seedErasureUser(t, db, "admin2@erase.test", dbgen.UserRoleAdmin)
	subjectID := seedErasureUser(t, db, "subject@erase.test", dbgen.UserRoleAnalyst)
	caseID := seedErasureCase(t, db, adminID)
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Create a STR filing that references subjectID as submitted_by
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO str_filings (case_id, org_id, suspicion_basis, status, submitted_by)
		VALUES ($1, $2, 'test hold', 'submitted', $3)`,
		caseID, orgID, subjectID)
	if err != nil {
		t.Fatalf("seed STR filing: %v", err)
	}

	router := newErasureRouter(db, jwtMgr)

	req := httptest.NewRequest(http.MethodDelete, "/users/"+subjectID.String()+"/data", nil)
	req.Header.Set("Authorization", bearerToken(t, jwtMgr, adminID, "admin2@erase.test", dbgen.UserRoleAdmin))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409 PMLA_HOLD, got %d — %s", rr.Code, rr.Body.String())
	}
}

func TestErasure_AnalystForbidden(t *testing.T) {
	db, teardown := erasureTestDB(t)
	defer teardown()
	jwtMgr := erasureJWT(t)

	analystID := seedErasureUser(t, db, "analyst@erase.test", dbgen.UserRoleAnalyst)
	targetID := seedErasureUser(t, db, "target2@erase.test", dbgen.UserRoleViewer)
	router := newErasureRouter(db, jwtMgr)

	req := httptest.NewRequest(http.MethodDelete, "/users/"+targetID.String()+"/data", nil)
	req.Header.Set("Authorization", bearerToken(t, jwtMgr, analystID, "analyst@erase.test", dbgen.UserRoleAnalyst))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}
