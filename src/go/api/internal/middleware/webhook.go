package middleware

import (
	"crypto/subtle"
	"net/http"
)

// RequireInternalAPIKey returns middleware that validates the X-Webhook-Secret
// header against the configured internal API key. Used for service-to-service
// auth on webhook ingestion endpoints.
func RequireInternalAPIKey(apiKey string) func(http.Handler) http.Handler {
	keyBytes := []byte(apiKey)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := r.Header.Get("X-Webhook-Secret")
			if provided == "" {
				http.Error(w, `{"message":"missing X-Webhook-Secret header"}`, http.StatusUnauthorized)
				return
			}

			if subtle.ConstantTimeCompare(keyBytes, []byte(provided)) != 1 {
				http.Error(w, `{"message":"invalid webhook secret"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
