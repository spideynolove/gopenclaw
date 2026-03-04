package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type TenantContextKey struct{}

func ExtractTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		apiKey := parts[1]
		ctx := context.WithValue(r.Context(), TenantContextKey{}, apiKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetTenantFromContext(ctx context.Context) (string, error) {
	tenant, ok := ctx.Value(TenantContextKey{}).(string)
	if !ok || tenant == "" {
		return "", errors.New("tenant not found in context")
	}
	return tenant, nil
}
