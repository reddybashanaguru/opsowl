package license

import (
	"net/http"
	"time"

	"github.com/reddybashanaguru/opsowl/internal/response"
)

const (
	// warnDaysBeforeExpiry triggers a warning header when the license is
	// within this many days of expiry.
	warnDaysBeforeExpiry = 30
)

// Middleware returns an HTTP middleware that rejects all non-health requests
// when the license has expired.  If no license is loaded the middleware is a
// transparent pass-through (development / trial mode).
//
// A response header "X-License-Expiry" is added to every response so the
// frontend can show a renewal banner when expiry is approaching.
func Middleware(c Checker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !c.Loaded() {
				// No license configured — pass through (dev/trial mode).
				next.ServeHTTP(w, r)
				return
			}

			exp := c.ExpiresAt()

			// Always add the expiry header so the frontend can surface a banner.
			w.Header().Set("X-License-Expiry", exp.Format(time.RFC3339))

			// Warn header when within 30 days of expiry.
			if daysUntil := time.Until(exp).Hours() / 24; daysUntil <= warnDaysBeforeExpiry {
				w.Header().Set("X-License-Warning",
					"License expires soon — contact support@opsowl.dev to renew")
			}

			// Block expired license on all routes except /health.
			if time.Now().After(exp) && r.URL.Path != "/health" {
				response.Err(w, http.StatusPaymentRequired,
					"LICENSE_EXPIRED",
					"Your OpsOwl license has expired. Contact support@opsowl.dev to renew.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// FeatureGate returns a middleware that rejects requests with 402 when the
// named feature is not included in the license.  Use it to gate individual
// route groups:
//
//	r.With(license.FeatureGate(lic, "sanctions")).Get("/sanctions/...", ...)
func FeatureGate(c Checker, feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !c.Loaded() {
				// No license — allow all features in dev/trial mode.
				next.ServeHTTP(w, r)
				return
			}
			if !c.Has(feature) {
				response.Err(w, http.StatusPaymentRequired,
					"FEATURE_NOT_LICENSED",
					"The '"+feature+"' module is not included in your license. Contact support@opsowl.dev to upgrade.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
