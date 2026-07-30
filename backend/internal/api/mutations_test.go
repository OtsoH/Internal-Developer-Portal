package api

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/OtsoH/internal-developer-portal/backend/internal/auth"
	dbgen "github.com/OtsoH/internal-developer-portal/backend/internal/db/gen"
)

// Begin fails loudly instead of returning a usable transaction. Create must
// answer 403 before it opens one, so a test that reaches this has found the bug
// it was looking for: a request that got to the database on someone else's
// authority.
type refusingTxBeginner struct{}

func (refusingTxBeginner) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("Begin must not be reached: authorization runs first")
}

func serverWithoutRealDB() *Server {
	return NewServer(&dbgen.Queries{}, WithTxBeginner(refusingTxBeginner{}))
}

func createRequest(teamID uuid.UUID) CreateServiceRequestObject {
	return CreateServiceRequestObject{Body: &CreateServiceJSONRequestBody{
		Name:      "Billing API",
		Slug:      "billing-api",
		TeamId:    teamID,
		Lifecycle: "production",
	}}
}

func principalWith(teamID uuid.UUID, role auth.Role) auth.Principal {
	return auth.Principal{
		UserID: uuid.New(),
		Email:  "dev.editor@example.com",
		Roles: map[uuid.UUID]auth.TeamRole{
			teamID: {TeamID: teamID, TeamSlug: "payments", TeamName: "Payments", Role: role},
		},
	}
}

// The ordering the plan calls for: an invented teamId is 403, not 404 and not a
// foreign-key error, so the response cannot be used to probe which teams exist.
func TestCreateChecksRoleBeforeTouchingTheDatabase(t *testing.T) {
	member := uuid.New()
	unknown := uuid.New()

	cases := map[string]struct {
		ctx    context.Context
		teamID uuid.UUID
	}{
		"no principal at all": {
			ctx:    context.Background(),
			teamID: member,
		},
		"viewer in the target team": {
			ctx:    auth.WithPrincipal(context.Background(), principalWith(member, auth.RoleViewer)),
			teamID: member,
		},
		"editor, but in a different team": {
			ctx:    auth.WithPrincipal(context.Background(), principalWith(member, auth.RoleEditor)),
			teamID: unknown,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resp, err := serverWithoutRealDB().CreateService(tc.ctx, createRequest(tc.teamID))
			require.NoError(t, err)

			denied, ok := resp.(CreateService403JSONResponse)
			require.True(t, ok, "expected 403 response, got %T", resp)
			require.Equal(t, "forbidden", denied.Code)
		})
	}
}

// An editor does clear the check — which is what makes the cases above mean
// something, rather than passing because every create is refused.
func TestCreateWithEditorRoleProceedsToTheDatabase(t *testing.T) {
	team := uuid.New()
	ctx := auth.WithPrincipal(context.Background(), principalWith(team, auth.RoleEditor))

	_, err := serverWithoutRealDB().CreateService(ctx, createRequest(team))

	require.ErrorContains(t, err, "Begin must not be reached",
		"authorization passed, so the handler should have gone on to open a transaction")
}

func TestRequireRole(t *testing.T) {
	team := uuid.New()
	ctx := auth.WithPrincipal(context.Background(), principalWith(team, auth.RoleEditor))

	require.NoError(t, requireRole(ctx, team, auth.RoleViewer), "editor outranks viewer")
	require.NoError(t, requireRole(ctx, team, auth.RoleEditor))
	require.ErrorIs(t, requireRole(ctx, team, auth.RoleAdmin), errForbidden, "editor is not admin")
	require.ErrorIs(t, requireRole(ctx, uuid.New(), auth.RoleViewer), errForbidden, "no membership")
}

func TestNormalizeTags(t *testing.T) {
	require.Nil(t, normalizeTags(nil))
	require.Empty(t, normalizeTags(&[]string{"", "   "}))

	tags := []string{"Go", " go ", "PCI", "", "go"}
	require.Equal(t, []string{"go", "pci"}, normalizeTags(&tags),
		"case and whitespace fold to one tag, order preserved")

	many := make([]string, maxTags+5)
	for i := range many {
		many[i] = string(rune('a' + i))
	}
	require.Len(t, normalizeTags(&many), maxTags, "the cap keeps the first maxTags")
}

// The distinction that matters: a field the user never touched arrives as "",
// and storing that would render an empty "repo ↗" link instead of an em dash.
func TestNilIfBlank(t *testing.T) {
	require.Nil(t, nilIfBlank(nil))

	for _, blank := range []string{"", "   ", "\t\n"} {
		require.Nil(t, nilIfBlank(&blank), "%q should store as NULL", blank)
	}

	padded := "  https://github.com/acme/billing  "
	got := nilIfBlank(&padded)
	require.NotNil(t, got)
	require.Equal(t, "https://github.com/acme/billing", *got)
}

func TestTextOrEmpty(t *testing.T) {
	require.Empty(t, textOrEmpty(nil))

	padded := "  edge routing  "
	require.Equal(t, "edge routing", textOrEmpty(&padded))
}

// Matching SQLSTATE alone would report a concurrent tag insert as a slug
// conflict, sending the caller off to rename a field that was never at fault.
func TestIsUniqueViolationMatchesTheConstraint(t *testing.T) {
	slugTaken := &pgconn.PgError{Code: "23505", ConstraintName: "services_slug_key"}
	tagRace := &pgconn.PgError{Code: "23505", ConstraintName: "tags_name_key"}
	fkViolation := &pgconn.PgError{Code: "23503", ConstraintName: "services_team_id_fkey"}

	require.True(t, isUniqueViolation(slugTaken, "services_slug_key"))
	require.False(t, isUniqueViolation(tagRace, "services_slug_key"))
	require.False(t, isUniqueViolation(fkViolation, "services_slug_key"))
	require.False(t, isUniqueViolation(errors.New("connection refused"), "services_slug_key"))

	// pgx wraps driver errors, so the check has to survive being buried.
	require.True(t, isUniqueViolation(errors.Join(errors.New("insert failed"), slugTaken), "services_slug_key"))
}
