package auth

import (
	"context"
	"net/http"

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

// ContextWithAPIClaims builds minimal Claims from an API key identity.
// Scopes are stored and enforced by RequireScope middleware.
func ContextWithAPIClaims(ctx context.Context, userID uuid.UUID, scopes []string) context.Context {
	c := &Claims{
		UserID: userID,
		Role:   db.UserRoleAnalyst, // least-privilege role for API key callers
		Scopes: scopes,
	}
	return context.WithValue(ctx, claimsKey, c)
}

// RequireScope returns middleware that allows the request only when:
//   - the caller is a JWT user (Scopes is nil — no scope restriction), OR
//   - the caller is an API key holder whose Scopes list contains the given scope.
func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := ClaimsFromCtx(r.Context())
			if c == nil {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"missing credentials"}}`, http.StatusUnauthorized)
				return
			}
			// JWT callers (nil Scopes) bypass scope checks — they rely on role-based access only.
			if c.Scopes == nil {
				next.ServeHTTP(w, r)
				return
			}
			for _, s := range c.Scopes {
				if s == scope {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":{"code":"FORBIDDEN","message":"API key missing required scope: `+scope+`"}}`, http.StatusForbidden)
		})
	}
}
