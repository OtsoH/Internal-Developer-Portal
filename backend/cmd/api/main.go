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

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OtsoH/internal-developer-portal/backend/internal/api"
	"github.com/OtsoH/internal-developer-portal/backend/internal/app"
	"github.com/OtsoH/internal-developer-portal/backend/internal/auth"
	"github.com/OtsoH/internal-developer-portal/backend/internal/db"
	dbgen "github.com/OtsoH/internal-developer-portal/backend/internal/db/gen"
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
	ctx := context.Background()

	authCfg, err := auth.ConfigFromEnv()
	if err != nil {
		return err
	}

	// The pool is declared out here so it can reach both the seed and the
	// router; a nil pool is the no-database mode.
	var (
		pool    *pgxpool.Pool
		queries *dbgen.Queries
	)
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL == "" {
		logger.Warn("DATABASE_URL not set, skipping migrations; API serves stub data only")
	} else {
		if err := db.Migrate(databaseURL, logger); err != nil {
			return err
		}
		pool, err = pgxpool.New(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		if os.Getenv("APP_SEED") == "true" {
			if err := db.Seed(ctx, pool, logger); err != nil {
				return err
			}
		}
		queries = dbgen.New(pool)
	}

	r, err := app.NewRouter(ctx, app.Deps{
		Queries: queries,
		Tx:      txBeginner(pool),
		Auth:    authCfg,
		Logger:  logger,
	})
	if err != nil {
		return err
	}

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

// txBeginner converts a possibly-nil pool into a possibly-nil interface.
//
// Assigning a nil *pgxpool.Pool straight into an interface field would produce a
// non-nil interface holding a nil pointer, so the server's own "do I have a
// database?" check would pass and mutations would panic instead of answering
// 503. The nil has to be reintroduced at the conversion.
func txBeginner(pool *pgxpool.Pool) api.TxBeginner {
	if pool == nil {
		return nil
	}
	return pool
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
