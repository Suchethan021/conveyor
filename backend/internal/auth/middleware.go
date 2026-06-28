package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/Suchethan021/conveyor/backend/internal/httpx"
)

type ctxKey int

const userIDKey ctxKey = iota

// Authenticator populates the request context with the user id when a valid
// session cookie is present. It never rejects — pair it with RequireAuth to
// guard specific routes.
func (s *Service) Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id, ok := s.sessions.UserID(r); ok {
			r = r.WithContext(context.WithValue(r.Context(), userIDKey, id))
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAuth rejects requests that lack an authenticated user.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserIDFromContext(r.Context()); !ok {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// UserIDFromContext returns the authenticated user id, if any.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
