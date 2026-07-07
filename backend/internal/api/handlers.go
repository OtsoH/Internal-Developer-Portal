package api

import (
	"context"
	"net/http"
)

// Server implements StrictServerInterface. Handlers are stubs until the
// database layer lands; mutations return 501 until week 2.
type Server struct{}

var _ StrictServerInterface = (*Server)(nil)

func NewServer() *Server {
	return &Server{}
}

func (s *Server) ListServices(ctx context.Context, request ListServicesRequestObject) (ListServicesResponseObject, error) {
	return ListServices200JSONResponse(ServiceList{Items: []Service{}}), nil
}

func (s *Server) GetService(ctx context.Context, request GetServiceRequestObject) (GetServiceResponseObject, error) {
	return GetService404JSONResponse{notFound("service not found")}, nil
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
	return ListTeams200JSONResponse(TeamList{Items: []Team{}}), nil
}

func notFound(msg string) ErrorResponseJSONResponse {
	return ErrorResponseJSONResponse{Code: "not_found", Message: msg}
}

func notImplemented() Error {
	return Error{Code: "not_implemented", Message: "endpoint not implemented yet"}
}
