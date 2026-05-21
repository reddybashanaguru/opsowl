package apikeys

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Route("/api-keys", func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Delete("/{id}", h.Revoke)
	})
}

// APIKeyMiddleware authenticates requests using the X-API-Key header.
// It sets the user's claims in context when a valid key is found.
func APIKeyMiddleware(sqlDB *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawKey := r.Header.Get("X-API-Key")
			if rawKey == "" {
				next.ServeHTTP(w, r)
				return
			}
			userID, scopes, err := ValidateKey(sqlDB, r.Context(), rawKey)
			if err != nil {
				http.Error(w, `{"error":"invalid_api_key"}`, http.StatusUnauthorized)
				return
			}
			ctx := auth.ContextWithAPIClaims(r.Context(), userID, scopes)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
