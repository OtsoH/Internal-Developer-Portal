package db

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed seed.sql
var seedSQL string

// Seed inserts idempotent dev fixtures. Only called when APP_SEED=true.
func Seed(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	if _, err := pool.Exec(ctx, seedSQL); err != nil {
		return fmt.Errorf("apply seed data: %w", err)
	}
	logger.Info("seed data applied")
	return nil
}
