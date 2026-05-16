package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return r.q.GetUserByEmail(ctx, email)
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *Repository) UpdateUserLastLogin(ctx context.Context, id uuid.UUID) error {
	return r.q.UpdateUserLastLogin(ctx, id)
}

func (r *Repository) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
	return r.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
}

func (r *Repository) GetRefreshTokenByHash(ctx context.Context, hash string) (db.RefreshToken, error) {
	return r.q.GetRefreshTokenByHash(ctx, hash)
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	return r.q.RevokeRefreshToken(ctx, id)
}

func (r *Repository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	return r.q.RevokeAllUserRefreshTokens(ctx, userID)
}

func (r *Repository) CreateAuditLog(ctx context.Context, eventCode string, actorID *uuid.UUID, ipAddress, resourceType string, resourceID *uuid.UUID, metadata map[string]any, severity string) error {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	var actor uuid.NullUUID
	if actorID != nil {
		actor = uuid.NullUUID{UUID: *actorID, Valid: true}
	}
	var resID uuid.NullUUID
	if resourceID != nil {
		resID = uuid.NullUUID{UUID: *resourceID, Valid: true}
	}
	return r.q.CreateAuditLog(ctx, db.CreateAuditLogParams{
		EventCode:    eventCode,
		ActorID:      actor,
		IpAddress:    sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		ResourceType: sql.NullString{String: resourceType, Valid: resourceType != ""},
		ResourceID:   resID,
		Metadata:     metaJSON,
		Severity:     severity,
	})
}
