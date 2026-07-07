package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	dbgen "github.com/OtsoH/internal-developer-portal/backend/internal/db/gen"
)

// Server implements StrictServerInterface backed by sqlc queries.
// Mutations return 501 until week 2. A nil queries value (no DATABASE_URL)
// keeps read endpoints serving empty lists so /healthz-only mode still works.
type Server struct {
	q *dbgen.Queries
}

var _ StrictServerInterface = (*Server)(nil)

func NewServer(q *dbgen.Queries) *Server {
	return &Server{q: q}
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

func (s *Server) CreateService(ctx context.Context, request CreateServiceRequestObject) (CreateServiceResponseObject, error) {
	return CreateServicedefaultJSONResponse{Body: notImplemented(), StatusCode: http.StatusNotImplemented}, nil
}

func (s *Server) UpdateService(ctx context.Context, request UpdateServiceRequestObject) (UpdateServiceResponseObject, error) {
	return UpdateServicedefaultJSONResponse{Body: notImplemented(), StatusCode: http.StatusNotImplemented}, nil
}

func (s *Server) DeleteService(ctx context.Context, request DeleteServiceRequestObject) (DeleteServiceResponseObject, error) {
	return DeleteServicedefaultJSONResponse{Body: notImplemented(), StatusCode: http.StatusNotImplemented}, nil
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

func notFound(msg string) ErrorResponseJSONResponse {
	return ErrorResponseJSONResponse{Code: "not_found", Message: msg}
}

func notImplemented() Error {
	return Error{Code: "not_implemented", Message: "endpoint not implemented yet"}
}
