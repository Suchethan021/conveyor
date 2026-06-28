package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Suchethan021/conveyor/backend/internal/auth"
	"github.com/Suchethan021/conveyor/backend/internal/db/sqlc"
)

// NewRouter builds the HTTP handler tree with the standard middleware stack
// and mounts the auth and project routes. corsOrigins are the browser origins
// allowed to call the API with credentials (for split-domain deploys).
func NewRouter(pool *pgxpool.Pool, queries *sqlc.Queries, authsvc *auth.Service, corsOrigins []string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Allow the configured frontend origin(s) to call the API with cookies.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Populate user context from the session cookie on every request.
	r.Use(authsvc.Authenticator)

	r.Get("/healthz", healthz(pool))
	authsvc.Mount(r)

	// Authenticated routes, scoped to the logged-in user.
	ph := &projectHandlers{q: queries}
	bh := &buildHandlers{q: queries}
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth)

		r.Post("/api/projects", ph.create)
		r.Get("/api/projects", ph.list)
		r.Get("/api/projects/{id}", ph.get)

		r.Post("/api/projects/{id}/builds", bh.trigger)
		r.Get("/api/projects/{id}/builds", bh.listForProject)

		r.Get("/api/builds/{id}", bh.get)
		r.Get("/api/builds/{id}/logs", bh.logs)
		r.Get("/api/builds/{id}/logs/stream", bh.stream)
		r.Post("/api/builds/{id}/cancel", bh.cancel)
	})

	return r
}

// healthz reports liveness and database reachability.
func healthz(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		status, code := "ok", http.StatusOK
		if err := pool.Ping(ctx); err != nil {
			status, code = "db unavailable", http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
	}
}
