package middleware

import (
	"net/http"
	"os"
)

// RequireTLS redirects plain HTTP requests to HTTPS when the REQUIRE_TLS
// environment variable is set to "true". The check uses the X-Forwarded-Proto
// header set by the reverse proxy (nginx / AWS ALB / GCP GLB); direct TLS
// termination at the app layer is also detected via r.TLS != nil.
//
// This satisfies RBI T2 — TLS enforcement without requiring a separate
// certificate at the application tier (typically handled by the proxy).
func RequireTLS(next http.Handler) http.Handler {
	enabled := os.Getenv("REQUIRE_TLS") == "true"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !enabled {
			next.ServeHTTP(w, r)
			return
		}
		// Treat direct TLS or a trusted proxy header as secure.
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			next.ServeHTTP(w, r)
			return
		}
		// Health and readiness probes must pass even over HTTP.
		if r.URL.Path == "/health" || r.URL.Path == "/ready" {
			next.ServeHTTP(w, r)
			return
		}
		target := "https://" + r.Host + r.RequestURI
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}
