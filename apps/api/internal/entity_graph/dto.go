package entity_graph

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound         = errors.New("relationship not found")
	ErrSelfLink         = errors.New("cannot link a subject to itself")
	ErrDuplicate        = errors.New("relationship already exists")
	ErrInvalidType      = errors.New("unknown relationship type")
	ErrPeerNotFound     = errors.New("target subject not found")
)

// RelationshipTypes mirrors the canonical set defined in migration 00038.
var RelationshipTypes = []string{
	"OWNS_ACCOUNT",
	"TRANSACTS_WITH",
	"FAMILY_OF",
	"EMPLOYED_BY",
	"DIRECTOR_OF",
	"BENEFICIAL_OWNER_OF",
	"SAME_ADDRESS",
	"SAME_PHONE",
	"SAME_IP",
	"SAME_DEVICE",
	"GUARANTOR_OF",
	"INTRODUCED_BY",
}

var relTypeSet map[string]struct{}

func init() {
	relTypeSet = make(map[string]struct{}, len(RelationshipTypes))
	for _, t := range RelationshipTypes {
		relTypeSet[t] = struct{}{}
	}
}

func ValidRelationshipType(t string) bool {
	_, ok := relTypeSet[t]
	return ok
}

// ── Requests ─────────────────────────────────────────────────────────────────

type CreateRelationshipRequest struct {
	ToSubjectID      uuid.UUID `json:"to_subject_id"`
	RelationshipType string    `json:"relationship_type"`
	// 0.0–1.0. 1.0 = confirmed/manual; lower for inferred/probabilistic.
	Strength *float64 `json:"strength,omitempty"`
	Notes    string   `json:"notes,omitempty"`
}

// ── Responses ─────────────────────────────────────────────────────────────────

// RelationshipResponse is a normalised view of one edge from a subject's POV.
// direction = "outbound" (subject is origin) | "inbound" (subject is target).
type RelationshipResponse struct {
	ID               uuid.UUID `json:"id"`
	Direction        string    `json:"direction"`
	PeerSubjectID    uuid.UUID `json:"peer_subject_id"`
	PeerName         string    `json:"peer_name"`
	PeerEntityType   string    `json:"peer_entity_type"`
	PeerRiskLevel    string    `json:"peer_risk_level"`
	PeerRiskScore    int       `json:"peer_risk_score"`
	PeerIsPEP        bool      `json:"peer_is_pep"`
	PeerIsSanctioned bool      `json:"peer_is_sanctioned"`
	RelationshipType string    `json:"relationship_type"`
	Strength         float64   `json:"strength"`
	Source           string    `json:"source"`
	Notes            string    `json:"notes,omitempty"`
	FirstSeenAt      time.Time `json:"first_seen_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// NetworkNode is a vertex in the traversal result.
type NetworkNode struct {
	ID            uuid.UUID `json:"id"`
	Label         string    `json:"label"` // canonical_name
	EntityType    string    `json:"entity_type"`
	RiskLevel     string    `json:"risk_level"`
	RiskScore     int       `json:"risk_score"`
	IsPEP         bool      `json:"is_pep"`
	IsSanctioned  bool      `json:"is_sanctioned"`
	IsWatchlisted bool      `json:"is_watchlisted"`
	Depth         int       `json:"depth"` // hops from root (0 = root itself)
	IsRoot        bool      `json:"is_root"`
}

// NetworkEdge is a directed edge in the traversal result.
type NetworkEdge struct {
	ID               uuid.UUID `json:"id"`
	Source           uuid.UUID `json:"source"`
	Target           uuid.UUID `json:"target"`
	RelationshipType string    `json:"relationship_type"`
	Strength         float64   `json:"strength"`
}

// NetworkResponse is the full graph returned by the Traverse API.
type NetworkResponse struct {
	RootID        uuid.UUID     `json:"root_id"`
	Depth         int           `json:"depth"`
	Nodes         []NetworkNode `json:"nodes"`
	Edges         []NetworkEdge `json:"edges"`
	HighRiskCount int           `json:"high_risk_count"` // PEP or sanctioned nodes (excl root)
}
