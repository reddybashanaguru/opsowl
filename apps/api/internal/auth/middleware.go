package auth

import (
	"net/http"
	"strings"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

// JWTValidate extracts and verifies the Bearer token, injecting Claims into ctx.
func JWTValidate(jwtMgr *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				response.Err(w, http.StatusUnauthorized, "MISSING_TOKEN", "authorization header required")
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := jwtMgr.Verify(tokenStr)
			if err != nil {
				response.Err(w, http.StatusUnauthorized, "INVALID_TOKEN", "token is invalid or expired")
				return
			}
			next.ServeHTTP(w, r.WithContext(ContextWithClaims(r.Context(), claims)))
		})
	}
}

// RequireRole returns 403 if the caller's role is not in the allowed set.
func RequireRole(roles ...db.UserRole) func(http.Handler) http.Handler {
	allowed := make(map[db.UserRole]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromCtx(r.Context())
			if claims == nil {
				response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
				return
			}
			if _, ok := allowed[claims.Role]; !ok {
				response.Err(w, http.StatusForbidden, "INSUFFICIENT_ROLE", "you do not have permission to perform this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
