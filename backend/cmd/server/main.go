package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Suchethan021/conveyor/backend/internal/api"
	"github.com/Suchethan021/conveyor/backend/internal/auth"
	"github.com/Suchethan021/conveyor/backend/internal/config"
	"github.com/Suchethan021/conveyor/backend/internal/db"
	"github.com/Suchethan021/conveyor/backend/internal/db/sqlc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()
	log.Println("connected to database")

	queries := sqlc.New(pool)

	authsvc, err := auth.NewService(cfg, queries)
	if err != nil {
		log.Fatalf("auth: %v", err)
	}
	if !authsvc.OAuthEnabled() {
		log.Println("warning: GitHub OAuth not configured (set GITHUB_CLIENT_ID/SECRET); login routes will return 503")
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.NewRouter(pool, authsvc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Serve in the background so we can wait for a shutdown signal.
	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
