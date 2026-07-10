package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OtsoH/internal-developer-portal/backend/internal/api"
	"github.com/OtsoH/internal-developer-portal/backend/internal/db"
	dbgen "github.com/OtsoH/internal-developer-portal/backend/internal/db/gen"
	"github.com/OtsoH/internal-developer-portal/backend/internal/httpx"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel(),
	}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	var queries *dbgen.Queries
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Warn("DATABASE_URL not set, skipping migrations; API serves stub data only")
	} else {
		if err := db.Migrate(databaseURL, logger); err != nil {
			return err
		}
		pool, err := pgxpool.New(context.Background(), databaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		if os.Getenv("APP_SEED") == "true" {
			if err := db.Seed(context.Background(), pool, logger); err != nil {
				return err
			}
		}
		queries = dbgen.New(pool)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// middleware.RealIP is deliberately absent: it is deprecated as spoofable
	// (trusts X-Forwarded-For & co. unconditionally, GHSA-3fxj-6jh8-hvhx).
	// Logs record the direct peer address instead.
	r.Use(httpx.RequestLogger(logger))
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	apiServer := api.NewServer(queries)
	r.Mount("/api/v1", api.HandlerFromMux(api.NewStrictHandler(apiServer, nil), chi.NewRouter()))

	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-stop:
		logger.Info("shutting down", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func logLevel() slog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
