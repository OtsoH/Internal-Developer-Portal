package api

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"slices"

	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/OtsoH/internal-developer-portal/backend/internal/auth"
	dbgen "github.com/OtsoH/internal-developer-portal/backend/internal/db/gen"
)

// TxBeginner starts a transaction. Mutations need one; reads do not. Declaring
// the requirement as an interface rather than taking *pgxpool.Pool states what
// is actually used and keeps the server fakeable without a database.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Server implements StrictServerInterface backed by sqlc queries.
// Mutations return 501 until step 9. A nil queries value (no DATABASE_URL)
// keeps read endpoints serving empty lists so /healthz-only mode still works.
type Server struct {
	q      *dbgen.Queries
	tx     TxBeginner
	logger *slog.Logger
}

var _ StrictServerInterface = (*Server)(nil)

// ServerOption configures a Server. Options rather than a widened signature so
// that call sites needing only the defaults — the unit tests — stay untouched.
type ServerOption func(*Server)

// WithTxBeginner supplies the transaction source mutations require. Without it
// they answer 503 rather than panicking.
func WithTxBeginner(tx TxBeginner) ServerOption {
	return func(s *Server) { s.tx = tx }
}

// WithLogger replaces the default slog.Default() logger. A nil logger is
// ignored, so the field is never nil.
func WithLogger(l *slog.Logger) ServerOption {
	return func(s *Server) {
		if l != nil {
			s.logger = l
		}
	}
}

func NewServer(q *dbgen.Queries, opts ...ServerOption) *Server {
	s := &Server{q: q, logger: slog.Default()}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// dbReady reports whether this server can perform a mutation. Reads degrade to
// empty results without a database, but a write that cannot be persisted has to
// say so rather than silently succeeding.
func (s *Server) dbReady() bool {
	if s.q != nil && s.tx != nil {
		return true
	}
	s.logger.Warn("mutation rejected: server has no database connection")
	return false
}

func (s *Server) ListServices(ctx context.Context, request ListServicesRequestObject) (ListServicesResponseObject, error) {
	if s.q == nil {
		return ListServices200JSONResponse(ServiceList{Items: []Service{}}), nil
	}
	rows, err := s.q.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]Service, 0, len(rows))
	for _, row := range rows {
		items = append(items, serviceFromRow(row))
	}
	return ListServices200JSONResponse(ServiceList{Items: items}), nil
}

func (s *Server) GetService(ctx context.Context, request GetServiceRequestObject) (GetServiceResponseObject, error) {
	if s.q == nil {
		return GetService404JSONResponse{notFound("service not found")}, nil
	}
	row, err := s.q.GetService(ctx, request.ServiceId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GetService404JSONResponse{notFound("service not found")}, nil
		}
		return nil, err
	}
	return GetService200JSONResponse(serviceFromRow(dbgen.ListServicesRow(row))), nil
}

// The three mutations live in mutations.go: they share a transaction helper, an
// authorization helper and the normalization rules, and none of that belongs in
// the file holding the read paths.

// GetCurrentUser answers from the principal the authentication middleware
// attached — no query of its own. Principal.Roles already carries each team's
// slug and name, which is why it is a map of TeamRole rather than bare roles.
func (s *Server) GetCurrentUser(ctx context.Context, request GetCurrentUserRequestObject) (GetCurrentUserResponseObject, error) {
	principal, ok := auth.PrincipalFrom(ctx)
	if !ok {
		// Only reachable when the authenticator is not mounted, i.e. the
		// no-database mode where the resolver cannot work.
		return GetCurrentUser401JSONResponse{unauthorized("not authenticated")}, nil
	}

	roles := make([]TeamRole, 0, len(principal.Roles))
	for _, role := range principal.Roles {
		roles = append(roles, TeamRole{
			TeamId:   role.TeamID,
			TeamSlug: role.TeamSlug,
			TeamName: role.TeamName,
			Role:     Role(role.Role),
		})
	}
	// Map iteration order is randomized in Go, so sort for a stable response.
	slices.SortFunc(roles, func(a, b TeamRole) int {
		return cmp.Compare(a.TeamName, b.TeamName)
	})

	return GetCurrentUser200JSONResponse(CurrentUser{
		Id:        principal.UserID,
		Email:     openapi_types.Email(principal.Email),
		Name:      principal.Name,
		TeamRoles: roles,
	}), nil
}

func (s *Server) ListTeams(ctx context.Context, request ListTeamsRequestObject) (ListTeamsResponseObject, error) {
	if s.q == nil {
		return ListTeams200JSONResponse(TeamList{Items: []Team{}}), nil
	}
	teams, err := s.q.ListTeams(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]Team, 0, len(teams))
	for _, t := range teams {
		items = append(items, Team{
			Id:        t.ID,
			Name:      t.Name,
			Slug:      t.Slug,
			CreatedAt: t.CreatedAt,
		})
	}
	return ListTeams200JSONResponse(TeamList{Items: items}), nil
}

func serviceFromRow(row dbgen.ListServicesRow) Service {
	svc := Service{
		Id:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		Lifecycle: Lifecycle(row.Lifecycle),
		Team: TeamRef{
			Id:   row.TeamID,
			Name: row.TeamName,
			Slug: row.TeamSlug,
		},
		Tags:       row.Tags,
		RepoUrl:    row.RepoUrl,
		RunbookUrl: row.RunbookUrl,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
	if row.Description != "" {
		svc.Description = &row.Description
	}
	return svc
}

func notFound(msg string) NotFoundJSONResponse {
	return NotFoundJSONResponse{Code: "not_found", Message: msg}
}

func unauthorized(msg string) UnauthorizedJSONResponse {
	return UnauthorizedJSONResponse{Code: "unauthenticated", Message: msg}
}

func badRequest(msg string) BadRequestJSONResponse {
	return BadRequestJSONResponse{Code: "bad_request", Message: msg}
}

func forbidden(msg string) ForbiddenJSONResponse {
	return ForbiddenJSONResponse{Code: "forbidden", Message: msg}
}

// Unlike the helpers above, the code is a parameter. 409 will mean more than one
// thing to a client — a taken slug now, a dependency cycle in week 3 — and a
// flat "conflict" would not let them tell those apart.
func conflict(code, msg string) ConflictJSONResponse {
	return ConflictJSONResponse{Code: code, Message: msg}
}

// databaseUnavailable is the honest answer when the process runs without
// DATABASE_URL: reads degrade to empty, writes cannot.
func databaseUnavailable() Error {
	return Error{Code: "database_unavailable", Message: "no database is configured"}
}
