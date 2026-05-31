package entity_graph

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

var defaultOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ── Create ─────────────────────────────────────────────────────────────────

func (r *Repository) Create(
	ctx context.Context,
	fromID, toID, createdBy uuid.UUID,
	req CreateRelationshipRequest,
) (*RelationshipResponse, error) {
	strength := 1.0
	if req.Strength != nil {
		strength = *req.Strength
	}
	meta := map[string]interface{}{}
	if req.Notes != "" {
		meta["notes"] = req.Notes
	}
	metaJSON, _ := json.Marshal(meta)

	var relID uuid.UUID
	err := r.db.QueryRowContext(ctx, `
INSERT INTO entity_relationships
  (org_id, from_subject_id, to_subject_id, relationship_type, strength, source, metadata, created_by)
VALUES ($1,$2,$3,$4,$5,'manual',$6,$7)
RETURNING id`,
		defaultOrgID, fromID, toID, req.RelationshipType, strength, metaJSON, createdBy,
	).Scan(&relID)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrDuplicate
		}
		return nil, err
	}
	return r.getByID(ctx, relID)
}

// ── GetByID ───────────────────────────────────────────────────────────────

func (r *Repository) getByID(ctx context.Context, relID uuid.UUID) (*RelationshipResponse, error) {
	const q = `
SELECT er.id, er.from_subject_id, er.to_subject_id, er.relationship_type,
       er.strength, er.source, er.metadata, er.first_seen_at, er.created_at,
       fs.canonical_name, fs.entity_type, fs.risk_level, fs.risk_score, fs.is_pep, fs.is_sanctioned,
       ts.canonical_name, ts.entity_type, ts.risk_level, ts.risk_score, ts.is_pep, ts.is_sanctioned
FROM entity_relationships er
JOIN subjects fs ON fs.id = er.from_subject_id
JOIN subjects ts ON ts.id = er.to_subject_id
WHERE er.id = $1 AND er.org_id = $2 AND er.is_active = true`

	var rel RelationshipResponse
	var fromID, toID uuid.UUID
	var metaRaw []byte
	var fromName, fromEntityType, fromRiskLevel string
	var fromRiskScore int
	var fromIsPEP, fromIsSanctioned bool
	var toName, toEntityType, toRiskLevel string
	var toRiskScore int
	var toIsPEP, toIsSanctioned bool

	err := r.db.QueryRowContext(ctx, q, relID, defaultOrgID).Scan(
		&rel.ID, &fromID, &toID, &rel.RelationshipType,
		&rel.Strength, &rel.Source, &metaRaw, &rel.FirstSeenAt, &rel.CreatedAt,
		&fromName, &fromEntityType, &fromRiskLevel, &fromRiskScore, &fromIsPEP, &fromIsSanctioned,
		&toName, &toEntityType, &toRiskLevel, &toRiskScore, &toIsPEP, &toIsSanctioned,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Normalise as outbound from fromID perspective
	rel.Direction = "outbound"
	rel.PeerSubjectID = toID
	rel.PeerName = toName
	rel.PeerEntityType = toEntityType
	rel.PeerRiskLevel = toRiskLevel
	rel.PeerRiskScore = toRiskScore
	rel.PeerIsPEP = toIsPEP
	rel.PeerIsSanctioned = toIsSanctioned

	var m map[string]interface{}
	if json.Unmarshal(metaRaw, &m) == nil {
		if n, ok := m["notes"].(string); ok {
			rel.Notes = n
		}
	}
	return &rel, nil
}

// ── List ──────────────────────────────────────────────────────────────────

// List returns all active relationships where subjectID is from OR to.
// Direction is set relative to subjectID.
func (r *Repository) List(ctx context.Context, subjectID uuid.UUID) ([]RelationshipResponse, error) {
	const q = `
SELECT er.id,
  CASE WHEN er.from_subject_id = $2 THEN 'outbound' ELSE 'inbound' END AS direction,
  CASE WHEN er.from_subject_id = $2 THEN er.to_subject_id   ELSE er.from_subject_id END AS peer_id,
  s.canonical_name, s.entity_type, s.risk_level, s.risk_score, s.is_pep, s.is_sanctioned,
  er.relationship_type, er.strength, er.source, er.metadata,
  er.first_seen_at, er.created_at
FROM entity_relationships er
JOIN subjects s ON s.id = (CASE WHEN er.from_subject_id = $2 THEN er.to_subject_id ELSE er.from_subject_id END)
WHERE er.org_id = $1 AND er.is_active = true
  AND (er.from_subject_id = $2 OR er.to_subject_id = $2)
  AND s.deleted_at IS NULL
ORDER BY er.created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, defaultOrgID, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RelationshipResponse
	for rows.Next() {
		var rel RelationshipResponse
		var metaRaw []byte
		if err := rows.Scan(
			&rel.ID, &rel.Direction, &rel.PeerSubjectID,
			&rel.PeerName, &rel.PeerEntityType, &rel.PeerRiskLevel, &rel.PeerRiskScore,
			&rel.PeerIsPEP, &rel.PeerIsSanctioned,
			&rel.RelationshipType, &rel.Strength, &rel.Source, &metaRaw,
			&rel.FirstSeenAt, &rel.CreatedAt,
		); err != nil {
			return nil, err
		}
		var m map[string]interface{}
		if json.Unmarshal(metaRaw, &m) == nil {
			if n, ok := m["notes"].(string); ok {
				rel.Notes = n
			}
		}
		out = append(out, rel)
	}
	if out == nil {
		out = []RelationshipResponse{}
	}
	return out, rows.Err()
}

// ── Deactivate ────────────────────────────────────────────────────────────

// Deactivate soft-deletes a relationship; subjectID is verified to be one of
// the two endpoints to prevent cross-subject tampering.
func (r *Repository) Deactivate(ctx context.Context, relID, subjectID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `
UPDATE entity_relationships SET is_active = false, updated_at = NOW()
WHERE id = $1 AND org_id = $2 AND is_active = true
  AND (from_subject_id = $3 OR to_subject_id = $3)`,
		relID, defaultOrgID, subjectID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Traverse ──────────────────────────────────────────────────────────────

// Traverse performs a BFS from rootID up to maxDepth hops and returns the
// full subgraph as nodes + edges. Capped at 80 nodes to protect the UI.
func (r *Repository) Traverse(ctx context.Context, rootID uuid.UUID, maxDepth int) (*NetworkResponse, error) {
	if maxDepth < 1 {
		maxDepth = 1
	}
	if maxDepth > 3 {
		maxDepth = 3
	}

	type rawEdge struct {
		ID           uuid.UUID
		FromID       uuid.UUID
		ToID         uuid.UUID
		RelType      string
		Strength     float64
	}

	visited := map[uuid.UUID]int{rootID: 0}
	frontier := []uuid.UUID{rootID}
	var allEdges []rawEdge

	for depth := 1; depth <= maxDepth && len(frontier) > 0; depth++ {
		strs := uuidsToStrings(frontier)
		rows, err := r.db.QueryContext(ctx, `
SELECT id, from_subject_id, to_subject_id, relationship_type, strength
FROM entity_relationships
WHERE org_id = $1 AND is_active = true
  AND (from_subject_id = ANY($2::uuid[]) OR to_subject_id = ANY($2::uuid[]))`,
			defaultOrgID, pq.Array(strs))
		if err != nil {
			return nil, err
		}

		var next []uuid.UUID
		for rows.Next() {
			var e rawEdge
			if err := rows.Scan(&e.ID, &e.FromID, &e.ToID, &e.RelType, &e.Strength); err != nil {
				rows.Close()
				return nil, err
			}
			allEdges = append(allEdges, e)
			for _, nodeID := range []uuid.UUID{e.FromID, e.ToID} {
				if _, seen := visited[nodeID]; !seen {
					visited[nodeID] = depth
					next = append(next, nodeID)
				}
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}

		// Cap total nodes
		if len(visited) >= 80 {
			break
		}
		frontier = next
	}

	// Deduplicate edges (BFS may encounter the same edge from multiple frontiers)
	edgeSeen := map[uuid.UUID]struct{}{}
	edges := make([]NetworkEdge, 0, len(allEdges))
	for _, e := range allEdges {
		if _, dup := edgeSeen[e.ID]; dup {
			continue
		}
		edgeSeen[e.ID] = struct{}{}
		edges = append(edges, NetworkEdge{
			ID: e.ID, Source: e.FromID, Target: e.ToID,
			RelationshipType: e.RelType, Strength: e.Strength,
		})
	}

	// Fetch subject data for all nodes in the subgraph
	nodeIDs := make([]string, 0, len(visited))
	for id := range visited {
		nodeIDs = append(nodeIDs, id.String())
	}
	subjectRows, err := r.db.QueryContext(ctx, `
SELECT id, canonical_name, entity_type, risk_level, risk_score, is_pep, is_sanctioned, is_watchlisted
FROM subjects
WHERE id = ANY($1::uuid[]) AND org_id = $2 AND deleted_at IS NULL`,
		pq.Array(nodeIDs), defaultOrgID)
	if err != nil {
		return nil, err
	}
	defer subjectRows.Close()

	nodes := make([]NetworkNode, 0, len(visited))
	highRisk := 0
	for subjectRows.Next() {
		var n NetworkNode
		if err := subjectRows.Scan(
			&n.ID, &n.Label, &n.EntityType, &n.RiskLevel, &n.RiskScore,
			&n.IsPEP, &n.IsSanctioned, &n.IsWatchlisted,
		); err != nil {
			return nil, err
		}
		n.Depth = visited[n.ID]
		n.IsRoot = n.ID == rootID
		if !n.IsRoot && (n.IsPEP || n.IsSanctioned || n.RiskLevel == "critical" || n.RiskLevel == "high") {
			highRisk++
		}
		nodes = append(nodes, n)
	}

	return &NetworkResponse{
		RootID:        rootID,
		Depth:         maxDepth,
		Nodes:         nodes,
		Edges:         edges,
		HighRiskCount: highRisk,
	}, subjectRows.Err()
}

func uuidsToStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}
