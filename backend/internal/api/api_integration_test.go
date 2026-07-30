package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/OtsoH/internal-developer-portal/backend/internal/app"
	"github.com/OtsoH/internal-developer-portal/backend/internal/auth"
	"github.com/OtsoH/internal-developer-portal/backend/internal/db"
	dbgen "github.com/OtsoH/internal-developer-portal/backend/internal/db/gen"
)

// The three dev personas and the two seeded teams, matched to seed.sql. Their
// asymmetric memberships are what make every RBAC outcome reachable from one
// fixed dataset instead of one this test would otherwise have to construct.
var (
	platformTeamID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	paymentsTeamID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
)

const (
	devAdmin  = "dev.admin@example.com"
	devEditor = "dev.editor@example.com"
	devViewer = "dev.viewer@example.com"
)

// testEnv wraps one Postgres container, the production router mounted behind a
// real HTTP server, and the sqlc queries used to set up fixtures and inspect
// state the API itself does not expose (the tags table, audit_log).
type testEnv struct {
	baseURL string
	http    *http.Client
	pool    *pgxpool.Pool
	queries *dbgen.Queries
}

// setupTestServer builds the whole stack once per test run: container,
// migrations, seed, the production router from app.NewRouter, and an
// httptest.Server in front of it. Every subtest below shares this instance —
// mutating subtests use their own fixture rows rather than the seeded services,
// so they can run in any order without disturbing one another.
func setupTestServer(t *testing.T) *testEnv {
	t.Helper()
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

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, db.Seed(ctx, pool, logger))

	queries := dbgen.New(pool)

	router, err := app.NewRouter(ctx, app.Deps{
		Queries: queries,
		Tx:      pool,
		Auth:    auth.Config{Mode: auth.ModeDev},
		Logger:  logger,
	})
	require.NoError(t, err)

	ts := httptest.NewServer(router)
	t.Cleanup(ts.Close)

	return &testEnv{baseURL: ts.URL, http: ts.Client(), pool: pool, queries: queries}
}

// request drives the API the way a real client does: over HTTP, through the
// production router, authenticating with the X-Dev-User header the same way
// the frontend's BFF proxy does in AUTH_MODE=dev.
func (env *testEnv) request(t *testing.T, method, path, devUser string, body any) (int, map[string]any) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, env.baseURL+path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if devUser != "" {
		req.Header.Set(auth.DevUserHeader, devUser)
	}

	resp, err := env.http.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	decoded := map[string]any{}
	if len(data) > 0 {
		require.NoError(t, json.Unmarshal(data, &decoded), "response body: %s", data)
	}
	return resp.StatusCode, decoded
}

// insertFixtureService writes a service directly through sqlc, bypassing the
// API. Mutating subtests need a row nothing else depends on, and a random slug
// keeps concurrent-looking fixtures from ever colliding on the unique index.
func insertFixtureService(t *testing.T, env *testEnv, teamID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id, err := env.queries.InsertService(context.Background(), dbgen.InsertServiceParams{
		TeamID:      teamID,
		Name:        name,
		Slug:        "fixture-" + uuid.NewString(),
		Description: "",
		Lifecycle:   "production",
	})
	require.NoError(t, err)
	return id
}

func parseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, s)
	require.NoError(t, err)
	return parsed
}

func toStrings(v any) []string {
	raw, _ := v.([]any)
	out := make([]string, len(raw))
	for i, item := range raw {
		out[i], _ = item.(string)
	}
	return out
}

// TestAPIIntegration exercises the production router — auth, validation and
// the mutation handlers — against a real Postgres container. Unit tests in
// mutations_test.go and handlers_test.go cover the pure pieces and the
// database-free RBAC ordering; this is where authorization is checked against
// the real per-team memberships the plan's seed data encodes.
func TestAPIIntegration(t *testing.T) {
	env := setupTestServer(t)

	t.Run("Unauthenticated", func(t *testing.T) {
		status, body := env.request(t, http.MethodGet, "/api/v1/services", "", nil)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", body["code"])

		status, body = env.request(t, http.MethodGet, "/api/v1/me", "nobody@example.com", nil)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", body["code"])
	})

	// The matrix from the plan's seed table: three personas, three mutating
	// operations, two teams. Table-driven rather than db_integration_test.go's
	// linear style, since 18 cells written out by hand would bury the one
	// property that matters — which cells are allowed — under repetition.
	t.Run("RBACMatrix", func(t *testing.T) {
		teams := []struct {
			name string
			id   uuid.UUID
		}{
			{"platform", platformTeamID},
			{"payments", paymentsTeamID},
		}
		personas := []struct {
			name  string
			email string
		}{
			{"admin", devAdmin},
			{"editor", devEditor},
			{"viewer", devViewer},
		}
		// Mirrors seed.sql: Dev Admin is ADMIN on Platform and absent from
		// Payments; Dev Editor is VIEWER on Platform and EDITOR on Payments;
		// Dev Viewer holds VIEWER on Platform only.
		canWrite := map[string]map[string]bool{
			"admin":  {"platform": true, "payments": false},
			"editor": {"platform": false, "payments": true},
			"viewer": {"platform": false, "payments": false},
		}
		// Delete needs ADMIN specifically, so Dev Editor's EDITOR-on-Payments
		// does not carry over.
		canDelete := map[string]map[string]bool{
			"admin":  {"platform": true, "payments": false},
			"editor": {"platform": false, "payments": false},
			"viewer": {"platform": false, "payments": false},
		}

		for _, persona := range personas {
			for _, team := range teams {
				write := canWrite[persona.name][team.name]
				del := canDelete[persona.name][team.name]

				t.Run(persona.name+"/"+team.name+"/POST", func(t *testing.T) {
					status, body := env.request(t, http.MethodPost, "/api/v1/services", persona.email, map[string]any{
						"name":      "RBAC POST " + persona.name + " " + team.name,
						"slug":      "rbac-post-" + persona.name + "-" + team.name,
						"teamId":    team.id.String(),
						"lifecycle": "production",
					})
					if write {
						require.Equal(t, http.StatusCreated, status, "%v", body)
					} else {
						require.Equal(t, http.StatusForbidden, status, "%v", body)
						require.Equal(t, "forbidden", body["code"])
					}
				})

				t.Run(persona.name+"/"+team.name+"/PUT", func(t *testing.T) {
					id := insertFixtureService(t, env, team.id, "RBAC PUT fixture")
					status, body := env.request(t, http.MethodPut, "/api/v1/services/"+id.String(), persona.email, map[string]any{
						"name":      "RBAC PUT attempt",
						"teamId":    team.id.String(),
						"lifecycle": "beta",
					})
					if write {
						require.Equal(t, http.StatusOK, status, "%v", body)
						return
					}
					require.Equal(t, http.StatusForbidden, status, "%v", body)
					row, err := env.queries.GetService(context.Background(), id)
					require.NoError(t, err)
					require.Equal(t, "RBAC PUT fixture", row.Name, "a denied update must not change the row")
				})

				t.Run(persona.name+"/"+team.name+"/DELETE", func(t *testing.T) {
					id := insertFixtureService(t, env, team.id, "RBAC DELETE fixture")
					status, body := env.request(t, http.MethodDelete, "/api/v1/services/"+id.String(), persona.email, nil)
					if del {
						require.Equal(t, http.StatusNoContent, status, "%v", body)
						return
					}
					require.Equal(t, http.StatusForbidden, status, "%v", body)
					_, err := env.queries.GetService(context.Background(), id)
					require.NoError(t, err, "a denied delete must leave the row in place")
				})
			}
		}
	})

	t.Run("DuplicateSlugConflict", func(t *testing.T) {
		status, body := env.request(t, http.MethodPost, "/api/v1/services", devAdmin, map[string]any{
			"name":      "Duplicate Gateway",
			"slug":      "api-gateway", // seeded slug, still owned by Platform
			"teamId":    platformTeamID.String(),
			"lifecycle": "production",
		})
		require.Equal(t, http.StatusConflict, status, "%v", body)
		require.Equal(t, "slug_taken", body["code"])
	})

	t.Run("BadSlug", func(t *testing.T) {
		status, body := env.request(t, http.MethodPost, "/api/v1/services", devEditor, map[string]any{
			"name":      "Bad Slug Service",
			"slug":      "Not A Slug",
			"teamId":    paymentsTeamID.String(),
			"lifecycle": "production",
		})
		require.Equal(t, http.StatusBadRequest, status, "%v", body)
		require.Equal(t, "bad_request", body["code"])
	})

	t.Run("UpdateUnknownID", func(t *testing.T) {
		status, body := env.request(t, http.MethodPut, "/api/v1/services/"+uuid.NewString(), devEditor, map[string]any{
			"name":      "Ghost",
			"teamId":    paymentsTeamID.String(),
			"lifecycle": "production",
		})
		require.Equal(t, http.StatusNotFound, status, "%v", body)
		require.Equal(t, "not_found", body["code"])
	})

	// Dev Editor is EDITOR on Payments (may edit the row) but only VIEWER on
	// Platform (may not receive it) — the two-sided check UpdateService runs
	// when teamId actually changes.
	t.Run("UpdateMoveToUneditableTeam", func(t *testing.T) {
		id := insertFixtureService(t, env, paymentsTeamID, "Movable Service")

		status, body := env.request(t, http.MethodPut, "/api/v1/services/"+id.String(), devEditor, map[string]any{
			"name":      "Attempted Move",
			"teamId":    platformTeamID.String(),
			"lifecycle": "production",
		})
		require.Equal(t, http.StatusForbidden, status, "%v", body)

		row, err := env.queries.GetService(context.Background(), id)
		require.NoError(t, err)
		require.Equal(t, paymentsTeamID, row.TeamID, "the row must not move on a denied update")
		require.Equal(t, "Movable Service", row.Name)
	})

	t.Run("TagRoundTrip", func(t *testing.T) {
		status, body := env.request(t, http.MethodPost, "/api/v1/services", devAdmin, map[string]any{
			"name":      "Tag Service",
			"slug":      "tag-service-" + uuid.NewString(),
			"teamId":    platformTeamID.String(),
			"lifecycle": "production",
			"tags":      []string{"Go", " go ", "PCI", ""},
		})
		require.Equal(t, http.StatusCreated, status, "%v", body)
		require.ElementsMatch(t, []string{"go", "pci"}, toStrings(body["tags"]),
			"case and whitespace fold, blanks drop")
		id := body["id"].(string)

		status, body = env.request(t, http.MethodPut, "/api/v1/services/"+id, devAdmin, map[string]any{
			"name":      "Tag Service",
			"teamId":    platformTeamID.String(),
			"lifecycle": "production",
			"tags":      []string{"pci"},
		})
		require.Equal(t, http.StatusOK, status, "%v", body)
		require.Equal(t, []string{"pci"}, toStrings(body["tags"]))

		serviceID := uuid.MustParse(id)

		var goTagID uuid.UUID
		require.NoError(t, env.pool.QueryRow(context.Background(),
			`SELECT id FROM tags WHERE name = 'go'`).Scan(&goTagID),
			"the tags row is shared vocabulary and must survive a removal")

		var linkCount int
		require.NoError(t, env.pool.QueryRow(context.Background(),
			`SELECT count(*) FROM service_tags WHERE service_id = $1 AND tag_id = $2`,
			serviceID, goTagID).Scan(&linkCount))
		require.Zero(t, linkCount, "the service_tags link must be gone even though the tag survives")
	})

	t.Run("UpdatedAtStrictlyGreater", func(t *testing.T) {
		status, body := env.request(t, http.MethodPost, "/api/v1/services", devAdmin, map[string]any{
			"name":      "Timestamp Service",
			"slug":      "timestamp-service-" + uuid.NewString(),
			"teamId":    platformTeamID.String(),
			"lifecycle": "production",
		})
		require.Equal(t, http.StatusCreated, status, "%v", body)
		id := body["id"].(string)
		created := parseTime(t, body["updatedAt"].(string))

		time.Sleep(10 * time.Millisecond)

		status, body = env.request(t, http.MethodPut, "/api/v1/services/"+id, devAdmin, map[string]any{
			"name":      "Timestamp Service",
			"teamId":    platformTeamID.String(),
			"lifecycle": "beta",
		})
		require.Equal(t, http.StatusOK, status, "%v", body)
		updated := parseTime(t, body["updatedAt"].(string))

		require.True(t, updated.After(created), "updatedAt must move forward: %s -> %s", created, updated)
	})

	t.Run("DeleteThenGet", func(t *testing.T) {
		id := insertFixtureService(t, env, platformTeamID, "Ephemeral Service")

		status, body := env.request(t, http.MethodDelete, "/api/v1/services/"+id.String(), devAdmin, nil)
		require.Equal(t, http.StatusNoContent, status, "%v", body)

		status, body = env.request(t, http.MethodGet, "/api/v1/services/"+id.String(), devAdmin, nil)
		require.Equal(t, http.StatusNotFound, status, "%v", body)
	})

	t.Run("AuditLog", func(t *testing.T) {
		ctx := context.Background()

		status, body := env.request(t, http.MethodPost, "/api/v1/services", devAdmin, map[string]any{
			"name":      "Audited Service",
			"slug":      "audited-service-" + uuid.NewString(),
			"teamId":    platformTeamID.String(),
			"lifecycle": "production",
		})
		require.Equal(t, http.StatusCreated, status, "%v", body)
		id := uuid.MustParse(body["id"].(string))

		status, body = env.request(t, http.MethodPut, "/api/v1/services/"+id.String(), devAdmin, map[string]any{
			"name":      "Audited Service Renamed",
			"teamId":    platformTeamID.String(),
			"lifecycle": "beta",
		})
		require.Equal(t, http.StatusOK, status, "%v", body)

		status, body = env.request(t, http.MethodDelete, "/api/v1/services/"+id.String(), devAdmin, nil)
		require.Equal(t, http.StatusNoContent, status, "%v", body)

		rows, err := env.pool.Query(ctx, `SELECT action FROM audit_log WHERE entity_id = $1 ORDER BY at`, id)
		require.NoError(t, err)
		defer rows.Close()

		var actions []string
		for rows.Next() {
			var action string
			require.NoError(t, rows.Scan(&action))
			actions = append(actions, action)
		}
		require.NoError(t, rows.Err())

		require.Equal(t, []string{"service.create", "service.update", "service.delete"}, actions,
			"one row per successful mutation, the delete row outliving the service it describes")

		var serviceExists bool
		require.NoError(t, env.pool.QueryRow(ctx,
			`SELECT exists(SELECT 1 FROM services WHERE id = $1)`, id).Scan(&serviceExists))
		require.False(t, serviceExists)
	})

	t.Run("GetCurrentUserShape", func(t *testing.T) {
		status, body := env.request(t, http.MethodGet, "/api/v1/me", devEditor, nil)
		require.Equal(t, http.StatusOK, status, "%v", body)

		require.Equal(t, devEditor, body["email"])
		require.Equal(t, "Dev Editor", body["name"])
		require.NotEmpty(t, body["id"])

		roles, ok := body["teamRoles"].([]any)
		require.True(t, ok)
		require.Len(t, roles, 2)

		byTeam := make(map[string]string, len(roles))
		for _, r := range roles {
			role := r.(map[string]any)
			byTeam[role["teamSlug"].(string)] = role["role"].(string)
		}
		require.Equal(t, "VIEWER", byTeam["platform"])
		require.Equal(t, "EDITOR", byTeam["payments"])
	})
}
