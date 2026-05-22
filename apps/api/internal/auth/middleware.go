package auth

import (
	"net/http"
	"strings"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

// JWTValidate extracts and verifies the JWT. Browser clients send it as an
// httpOnly access_token cookie; API/script clients send Authorization: Bearer.
// Cookie takes precedence so the Bearer header is not required for browsers.
func JWTValidate(jwtMgr *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenStr string

			// httpOnly cookie (browser clients — XSS cannot read this)
			if c, err := r.Cookie("access_token"); err == nil && c.Value != "" {
				tokenStr = c.Value
			}

			// Bearer header fallback (API clients, scripts, non-browser integrations)
			if tokenStr == "" {
				if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
					tokenStr = strings.TrimPrefix(h, "Bearer ")
				}
			}

			if tokenStr == "" {
				response.Err(w, http.StatusUnauthorized, "MISSING_TOKEN", "authorization required")
				return
			}

			claims, err := jwtMgr.Verify(tokenStr)
			if err != nil {
				response.Err(w, http.StatusUnauthorized, "INVALID_TOKEN", "token is invalid or expired")
				return
			}
			// Reject MFA partial tokens — they are only valid for the challenge endpoint.
			if claims.MFAPending {
				response.Err(w, http.StatusUnauthorized, "MFA_TOKEN_NOT_ACCEPTED", "complete MFA challenge first")
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
