package auth

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is an unexported type used as a key for values stored in context.
// Using a private type prevents collisions with keys from other packages.
type contextKey struct{}

// RequireAuth returns an HTTP middleware that validates the Bearer JWT in the
// Authorization header. On success, the verified *Claims are stored in the
// request context (retrieve with ClaimsFromContext). On failure, 401 is
// returned and the next handler is not called.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := claimsFromRequest(secret, r)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), contextKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin returns an HTTP middleware that performs the same JWT validation
// as RequireAuth and additionally requires the IsAdmin claim to be true.
// Returns 401 when the token is missing or invalid, 403 when the user is not
// an admin.
func RequireAdmin(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := claimsFromRequest(secret, r)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !claims.IsAdmin {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			ctx := context.WithValue(r.Context(), contextKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves the *Claims stored in ctx by RequireAuth or
// RequireAdmin. Returns (nil, false) when no claims are present.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(contextKey{}).(*Claims)
	return c, ok
}

// claimsFromRequest extracts and verifies the Bearer token from r.
func claimsFromRequest(secret string, r *http.Request) (*Claims, error) {
	authHeader := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return nil, ErrTokenInvalid
	}
	tokenStr := strings.TrimPrefix(authHeader, prefix)
	if tokenStr == "" {
		return nil, ErrTokenInvalid
	}
	return Verify(secret, tokenStr)
}
