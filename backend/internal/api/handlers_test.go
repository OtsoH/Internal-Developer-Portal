package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

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

func TestMutationsReturnNotImplemented(t *testing.T) {
	srv := NewServer(nil)
	ctx := context.Background()

	createResp, err := srv.CreateService(ctx, CreateServiceRequestObject{})
	require.NoError(t, err)
	create, ok := createResp.(CreateServicedefaultJSONResponse)
	require.True(t, ok, "expected default response, got %T", createResp)
	require.Equal(t, http.StatusNotImplemented, create.StatusCode)
	require.Equal(t, "not_implemented", create.Body.Code)

	updateResp, err := srv.UpdateService(ctx, UpdateServiceRequestObject{ServiceId: uuid.New()})
	require.NoError(t, err)
	update, ok := updateResp.(UpdateServicedefaultJSONResponse)
	require.True(t, ok, "expected default response, got %T", updateResp)
	require.Equal(t, http.StatusNotImplemented, update.StatusCode)

	deleteResp, err := srv.DeleteService(ctx, DeleteServiceRequestObject{ServiceId: uuid.New()})
	require.NoError(t, err)
	del, ok := deleteResp.(DeleteServicedefaultJSONResponse)
	require.True(t, ok, "expected default response, got %T", deleteResp)
	require.Equal(t, http.StatusNotImplemented, del.StatusCode)
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
