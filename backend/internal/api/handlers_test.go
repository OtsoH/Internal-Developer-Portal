package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/OtsoH/internal-developer-portal/backend/internal/auth"
	dbgen "github.com/OtsoH/internal-developer-portal/backend/internal/db/gen"
)

// Nil queries = no DATABASE_URL: read endpoints must keep serving empty
// results instead of panicking, so /healthz-only mode stays usable.
func TestListServicesWithoutDB(t *testing.T) {
	srv := NewServer(nil)

	resp, err := srv.ListServices(context.Background(), ListServicesRequestObject{})
	require.NoError(t, err)

	list, ok := resp.(ListServices200JSONResponse)
	require.True(t, ok, "expected 200 response, got %T", resp)
	require.Empty(t, list.Items)
}

func TestListTeamsWithoutDB(t *testing.T) {
	srv := NewServer(nil)

	resp, err := srv.ListTeams(context.Background(), ListTeamsRequestObject{})
	require.NoError(t, err)

	list, ok := resp.(ListTeams200JSONResponse)
	require.True(t, ok, "expected 200 response, got %T", resp)
	require.Empty(t, list.Items)
}

func TestGetServiceWithoutDB(t *testing.T) {
	srv := NewServer(nil)

	resp, err := srv.GetService(context.Background(), GetServiceRequestObject{ServiceId: uuid.New()})
	require.NoError(t, err)

	notFound, ok := resp.(GetService404JSONResponse)
	require.True(t, ok, "expected 404 response, got %T", resp)
	require.Equal(t, "not_found", notFound.Code)
}

// Without a database a write cannot be persisted, so it has to say so rather
// than reporting the 501 that means "not built yet".
func TestMutationsWithoutDBReturnUnavailable(t *testing.T) {
	srv := NewServer(nil)
	ctx := context.Background()

	createResp, err := srv.CreateService(ctx, CreateServiceRequestObject{})
	require.NoError(t, err)
	create, ok := createResp.(CreateServicedefaultJSONResponse)
	require.True(t, ok, "expected default response, got %T", createResp)
	require.Equal(t, http.StatusServiceUnavailable, create.StatusCode)
	require.Equal(t, "database_unavailable", create.Body.Code)

	updateResp, err := srv.UpdateService(ctx, UpdateServiceRequestObject{ServiceId: uuid.New()})
	require.NoError(t, err)
	update, ok := updateResp.(UpdateServicedefaultJSONResponse)
	require.True(t, ok, "expected default response, got %T", updateResp)
	require.Equal(t, http.StatusServiceUnavailable, update.StatusCode)

	deleteResp, err := srv.DeleteService(ctx, DeleteServiceRequestObject{ServiceId: uuid.New()})
	require.NoError(t, err)
	del, ok := deleteResp.(DeleteServicedefaultJSONResponse)
	require.True(t, ok, "expected default response, got %T", deleteResp)
	require.Equal(t, http.StatusServiceUnavailable, del.StatusCode)
}

// The 501 stubs are still reachable once a database is present — that is what
// step 9 replaces. A fake TxBeginner is enough to get past the readiness guard.
func TestMutationsWithDBReturnNotImplemented(t *testing.T) {
	srv := NewServer(&dbgen.Queries{}, WithTxBeginner(fakeTxBeginner{}))
	ctx := context.Background()

	createResp, err := srv.CreateService(ctx, CreateServiceRequestObject{})
	require.NoError(t, err)
	create, ok := createResp.(CreateServicedefaultJSONResponse)
	require.True(t, ok, "expected default response, got %T", createResp)
	require.Equal(t, http.StatusNotImplemented, create.StatusCode)
	require.Equal(t, "not_implemented", create.Body.Code)
}

type fakeTxBeginner struct{}

func (fakeTxBeginner) Begin(context.Context) (pgx.Tx, error) { return nil, nil }

// GetCurrentUser reads the principal the middleware attached. Roles come back
// sorted by team name so the response is stable across map iterations.
func TestGetCurrentUserFromPrincipal(t *testing.T) {
	platform := uuid.New()
	payments := uuid.New()
	principal := auth.Principal{
		UserID: uuid.New(),
		Email:  "dev.editor@example.com",
		Name:   "Dev Editor",
		Roles: map[uuid.UUID]auth.TeamRole{
			platform: {TeamID: platform, TeamSlug: "platform", TeamName: "Platform", Role: auth.RoleViewer},
			payments: {TeamID: payments, TeamSlug: "payments", TeamName: "Payments", Role: auth.RoleEditor},
		},
	}
	ctx := auth.WithPrincipal(context.Background(), principal)

	resp, err := NewServer(nil).GetCurrentUser(ctx, GetCurrentUserRequestObject{})
	require.NoError(t, err)

	me, ok := resp.(GetCurrentUser200JSONResponse)
	require.True(t, ok, "expected 200 response, got %T", resp)
	require.Equal(t, principal.UserID, me.Id)
	require.Equal(t, "Dev Editor", me.Name)
	require.Len(t, me.TeamRoles, 2)
	require.Equal(t, "Payments", me.TeamRoles[0].TeamName, "roles must sort by team name")
	require.Equal(t, RoleEditor, me.TeamRoles[0].Role)
	require.Equal(t, "Platform", me.TeamRoles[1].TeamName)
	require.Equal(t, RoleViewer, me.TeamRoles[1].Role)
}

// Reachable only in the no-database mode, where the authenticator is not
// mounted and therefore no principal exists.
func TestGetCurrentUserWithoutPrincipal(t *testing.T) {
	resp, err := NewServer(nil).GetCurrentUser(context.Background(), GetCurrentUserRequestObject{})
	require.NoError(t, err)

	unauth, ok := resp.(GetCurrentUser401JSONResponse)
	require.True(t, ok, "expected 401 response, got %T", resp)
	require.Equal(t, "unauthenticated", unauth.Code)
}

func TestServiceFromRow(t *testing.T) {
	repoURL := "https://github.com/acme/gateway"
	now := time.Now().UTC()
	row := dbgen.ListServicesRow{
		ID:          uuid.New(),
		Name:        "API Gateway",
		Slug:        "api-gateway",
		Description: "Edge routing",
		RepoUrl:     &repoURL,
		Lifecycle:   "production",
		CreatedAt:   now,
		UpdatedAt:   now,
		TeamID:      uuid.New(),
		TeamName:    "Platform",
		TeamSlug:    "platform",
		Tags:        []string{"edge", "go"},
	}

	svc := serviceFromRow(row)

	require.Equal(t, row.ID, svc.Id)
	require.Equal(t, "API Gateway", svc.Name)
	require.Equal(t, Lifecycle("production"), svc.Lifecycle)
	require.NotNil(t, svc.Description)
	require.Equal(t, "Edge routing", *svc.Description)
	require.Equal(t, &repoURL, svc.RepoUrl)
	require.Nil(t, svc.RunbookUrl)
	require.Equal(t, row.TeamID, svc.Team.Id)
	require.Equal(t, "platform", svc.Team.Slug)
	require.Equal(t, []string{"edge", "go"}, svc.Tags)
}

func TestServiceFromRowEmptyDescription(t *testing.T) {
	svc := serviceFromRow(dbgen.ListServicesRow{Description: ""})
	require.Nil(t, svc.Description, "empty description must be omitted, not rendered as \"\"")
}
