package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"

	"github.com/codenxtlab/bhejna/internal/db"
)

type contextKey string

const tenantKey contextKey = "tenant"

// APIKeyMiddleware extracts the Authorization: Bearer <key> header and looks up the tenant.
func APIKeyMiddleware(database *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Unauthorized: Missing or invalid API key", http.StatusUnauthorized)
				return
			}

			key := strings.TrimPrefix(authHeader, "Bearer ")
			tenant, err := database.GetTenantByAccessToken(key)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if tenant == nil {
				http.Error(w, "Unauthorized: Invalid API key", http.StatusUnauthorized)
				return
			}

			if tenant.IsPaused {
				http.Error(w, "Forbidden: Tenant account is paused", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), tenantKey, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// MetaSignatureMiddleware validates the X-Hub-Signature-256 header.
func MetaSignatureMiddleware(appSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Cap Meta webhook bodies at 1MB to prevent OOM from oversized payloads
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

			signature := r.Header.Get("X-Hub-Signature-256")
			if signature == "" {
				http.Error(w, "Unauthorized: Missing signature", http.StatusUnauthorized)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			// Restore the body for subsequent handlers
			r.Body = io.NopCloser(bytes.NewBuffer(body))

			if !validateSignature(body, signature, appSecret) {
				http.Error(w, "Unauthorized: Invalid signature", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// InternalJWTMiddleware protects internal routes with a shared secret.
func InternalJWTMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != jwtSecret {
				http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func validateSignature(payload []byte, signature string, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	actualSig := signature[7:]
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	expectedSig := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(actualSig), []byte(expectedSig))
}

func GetTenant(ctx context.Context) *db.Tenant {
	if tenant, ok := ctx.Value(tenantKey).(*db.Tenant); ok {
		return tenant
	}
	return nil
}
