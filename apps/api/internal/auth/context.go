package auth

import (
	"context"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type contextKey string

const claimsKey contextKey = "auth_claims"

func ContextWithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func ClaimsFromCtx(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}

// ContextWithAPIClaims builds a minimal Claims from an API key identity
// so downstream handlers can read ClaimsFromCtx without JWT-specific logic.
func ContextWithAPIClaims(ctx context.Context, userID uuid.UUID, scopes []string) context.Context {
	_ = scopes // scopes are validated at the handler layer if fine-grained control is needed
	c := &Claims{
		UserID: userID,
		Role:   db.UserRoleAnalyst, // default least-privilege role for API key callers
	}
	return context.WithValue(ctx, claimsKey, c)
}
