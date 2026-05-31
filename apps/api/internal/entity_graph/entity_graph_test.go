package entity_graph

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

// ── Unit tests (no DB) ────────────────────────────────────────────────────

func TestValidRelationshipType(t *testing.T) {
	valid := []string{"OWNS_ACCOUNT", "FAMILY_OF", "TRANSACTS_WITH", "DIRECTOR_OF", "SAME_PHONE"}
	for _, rt := range valid {
		if !ValidRelationshipType(rt) {
			t.Errorf("expected %q to be valid", rt)
		}
	}
	invalid := []string{"", "RANDOM", "owns_account", "CANCELLED"}
	for _, rt := range invalid {
		if ValidRelationshipType(rt) {
			t.Errorf("expected %q to be invalid", rt)
		}
	}
}

func TestSelfLinkError(t *testing.T) {
	svc := &Service{repo: &Repository{}}
	id := uuid.New()
	claims := &auth.Claims{UserID: uuid.New()}
	f := 1.0
	_, err := svc.Create(context.Background(), claims, id, CreateRelationshipRequest{
		ToSubjectID:      id,
		RelationshipType: "FAMILY_OF",
		Strength:         &f,
	})
	if err != ErrSelfLink {
		t.Fatalf("expected ErrSelfLink, got %v", err)
	}
}

func TestInvalidTypeError(t *testing.T) {
	svc := &Service{repo: &Repository{}}
	claims := &auth.Claims{UserID: uuid.New()}
	_, err := svc.Create(context.Background(), claims, uuid.New(), CreateRelationshipRequest{
		ToSubjectID:      uuid.New(),
		RelationshipType: "NOT_A_TYPE",
	})
	if err != ErrInvalidType {
		t.Fatalf("expected ErrInvalidType, got %v", err)
	}
}

func TestUuidsToStrings(t *testing.T) {
	ids := []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000001"), uuid.MustParse("00000000-0000-0000-0000-000000000002")}
	strs := uuidsToStrings(ids)
	if len(strs) != 2 || strs[0] != ids[0].String() {
		t.Fatal("uuidsToStrings mismatch")
	}
}

// ── Integration tests (require TEST_DB_URL) ───────────────────────────────

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		t.Skip("TEST_DB_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCreateAndListRelationship(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	// Create two subjects directly in DB for testing
	orgID := defaultOrgID
	var subA, subB uuid.UUID
	_ = db.QueryRowContext(ctx, `INSERT INTO subjects (org_id, canonical_name, entity_type) VALUES ($1,'Test-A','individual') RETURNING id`, orgID).Scan(&subA)
	_ = db.QueryRowContext(ctx, `INSERT INTO subjects (org_id, canonical_name, entity_type) VALUES ($1,'Test-B','company') RETURNING id`, orgID).Scan(&subB)
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM subjects WHERE id IN ($1,$2)`, subA, subB)
	})

	f := 0.9
	creator := uuid.MustParse("9f89ac54-d49e-4559-82fd-0a6988716130")
	rel, err := repo.Create(ctx, subA, subB, creator, CreateRelationshipRequest{
		ToSubjectID:      subB,
		RelationshipType: "EMPLOYED_BY",
		Strength:         &f,
		Notes:            "confirmed via HR records",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if rel.RelationshipType != "EMPLOYED_BY" {
		t.Errorf("unexpected type %q", rel.RelationshipType)
	}
	if rel.Notes != "confirmed via HR records" {
		t.Errorf("notes not preserved: %q", rel.Notes)
	}

	// Duplicate must fail
	_, err = repo.Create(ctx, subA, subB, creator, CreateRelationshipRequest{
		ToSubjectID: subB, RelationshipType: "EMPLOYED_BY",
	})
	if err != ErrDuplicate {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}

	// List from subA's perspective
	rels, err := repo.List(ctx, subA)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rels) != 1 || rels[0].Direction != "outbound" {
		t.Fatalf("unexpected list result: %+v", rels)
	}

	// List from subB's perspective — inbound
	rels, err = repo.List(ctx, subB)
	if err != nil {
		t.Fatalf("list from B: %v", err)
	}
	if len(rels) != 1 || rels[0].Direction != "inbound" {
		t.Fatalf("expected inbound from B, got %+v", rels)
	}

	// Deactivate
	if err := repo.Deactivate(ctx, rel.ID, subA); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	rels, _ = repo.List(ctx, subA)
	if len(rels) != 0 {
		t.Fatal("expected 0 relationships after deactivation")
	}
}

func TestTraverse(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	orgID := defaultOrgID
	var a, b, c uuid.UUID
	_ = db.QueryRowContext(ctx, `INSERT INTO subjects (org_id, canonical_name, entity_type) VALUES ($1,'TrvA','individual') RETURNING id`, orgID).Scan(&a)
	_ = db.QueryRowContext(ctx, `INSERT INTO subjects (org_id, canonical_name, entity_type) VALUES ($1,'TrvB','individual') RETURNING id`, orgID).Scan(&b)
	_ = db.QueryRowContext(ctx, `INSERT INTO subjects (org_id, canonical_name, entity_type) VALUES ($1,'TrvC','individual') RETURNING id`, orgID).Scan(&c)
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM subjects WHERE id IN ($1,$2,$3)`, a, b, c)
	})

	creator := uuid.MustParse("9f89ac54-d49e-4559-82fd-0a6988716130")
	// A → B (FAMILY_OF), B → C (TRANSACTS_WITH)
	_, _ = repo.Create(ctx, a, b, creator, CreateRelationshipRequest{ToSubjectID: b, RelationshipType: "FAMILY_OF"})
	_, _ = repo.Create(ctx, b, c, creator, CreateRelationshipRequest{ToSubjectID: c, RelationshipType: "TRANSACTS_WITH"})

	net, err := repo.Traverse(ctx, a, 2)
	if err != nil {
		t.Fatalf("traverse: %v", err)
	}

	nodeIDs := map[uuid.UUID]int{}
	for _, n := range net.Nodes {
		nodeIDs[n.ID] = n.Depth
	}

	if nodeIDs[a] != 0 {
		t.Errorf("root depth should be 0, got %d", nodeIDs[a])
	}
	if nodeIDs[b] != 1 {
		t.Errorf("B depth should be 1, got %d", nodeIDs[b])
	}
	if nodeIDs[c] != 2 {
		t.Errorf("C depth should be 2, got %d", nodeIDs[c])
	}
	if len(net.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(net.Edges))
	}
}

