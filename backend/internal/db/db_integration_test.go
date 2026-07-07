package db_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/OtsoH/internal-developer-portal/backend/internal/db"
	dbgen "github.com/OtsoH/internal-developer-portal/backend/internal/db/gen"
)

// TestMigrateSeedAndListServices exercises the real database path end to end:
// migrations, seed fixtures and the sqlc list query against a throwaway
// Postgres container.
func TestMigrateSeedAndListServices(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}

	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("idp_test"),
		tcpostgres.WithUsername("idp"),
		tcpostgres.WithPassword("idp"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, testcontainers.TerminateContainer(pg))
	})

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	logger := slog.New(slog.DiscardHandler)
	require.NoError(t, db.Migrate(dsn, logger))
	// Running migrations twice must be a no-op, not an error.
	require.NoError(t, db.Migrate(dsn, logger))

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, db.Seed(ctx, pool, logger))
	// Seeding is idempotent; a second run must not duplicate rows.
	require.NoError(t, db.Seed(ctx, pool, logger))

	queries := dbgen.New(pool)

	services, err := queries.ListServices(ctx)
	require.NoError(t, err)
	require.Len(t, services, 5)

	bySlug := make(map[string]dbgen.ListServicesRow, len(services))
	for _, s := range services {
		bySlug[s.Slug] = s
	}

	gateway := bySlug["api-gateway"]
	require.Equal(t, "API Gateway", gateway.Name)
	require.Equal(t, "production", gateway.Lifecycle)
	require.Equal(t, "Platform", gateway.TeamName)
	require.Equal(t, []string{"edge", "go"}, gateway.Tags)

	legacy := bySlug["legacy-reports"]
	require.Equal(t, "deprecated", legacy.Lifecycle)
	require.Empty(t, legacy.Tags)

	teams, err := queries.ListTeams(ctx)
	require.NoError(t, err)
	require.Len(t, teams, 2)
}
