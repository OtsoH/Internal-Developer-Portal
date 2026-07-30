package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/OtsoH/internal-developer-portal/backend/internal/auth"
	dbgen "github.com/OtsoH/internal-developer-portal/backend/internal/db/gen"
)

// Sentinels for outcomes that are expected rather than broken. A transaction
// callback can only return a plain error, so telling "this caller lacks the
// role" apart from "the database is unreachable" has to happen after the
// rollback, through errors.Is.
var (
	errForbidden    = errors.New("forbidden")
	errNotFound     = errors.New("not found")
	errSlugConflict = errors.New("slug already taken")
)

// A service with two hundred tags is a mistake, not a use case. The spec does
// not cap the array, so the cap lives here rather than rejecting the request:
// silently keeping the first twenty is friendlier than a 400 for something the
// caller almost certainly did by accident.
const maxTags = 20

// inTx runs fn inside one transaction. Every mutation is all-or-nothing: the
// service row, its tags and the audit entry commit together or not at all.
func (s *Server) inTx(ctx context.Context, fn func(q *dbgen.Queries) error) error {
	tx, err := s.tx.Begin(ctx)
	if err != nil {
		return err
	}
	// A rollback after a successful commit is a no-op, so this needs no
	// committed flag: on every early return it is what actually undoes the work.
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// requireRole is the whole of authorization. It answers only "may they?" and
// leaves each handler to build its own typed 403 — see the package comment on
// internal/auth for why this is not middleware.
func requireRole(ctx context.Context, teamID uuid.UUID, min auth.Role) error {
	principal, ok := auth.PrincipalFrom(ctx)
	if !ok || !principal.HasRoleIn(teamID, min) {
		return errForbidden
	}
	return nil
}

func (s *Server) CreateService(ctx context.Context, request CreateServiceRequestObject) (CreateServiceResponseObject, error) {
	if !s.dbReady() {
		return CreateServicedefaultJSONResponse{Body: databaseUnavailable(), StatusCode: http.StatusServiceUnavailable}, nil
	}
	body := request.Body
	if body == nil {
		return CreateService400JSONResponse{badRequest("request body is required")}, nil
	}

	// Checked before any query, deliberately. A team that does not exist cannot
	// grant anyone a role, so an invented teamId is answered 403 and never
	// reveals whether that team is real.
	if err := requireRole(ctx, body.TeamId, auth.RoleEditor); err != nil {
		return CreateService403JSONResponse{forbidden("EDITOR role required on the owning team")}, nil
	}

	// Trimmed once, so the row and the audit entry describing it cannot disagree.
	name := strings.TrimSpace(body.Name)

	var created Service
	err := s.inTx(ctx, func(q *dbgen.Queries) error {
		id, err := q.InsertService(ctx, dbgen.InsertServiceParams{
			TeamID:      body.TeamId,
			Name:        name,
			Slug:        body.Slug,
			Description: textOrEmpty(body.Description),
			RepoUrl:     nilIfBlank(body.RepoUrl),
			RunbookUrl:  nilIfBlank(body.RunbookUrl),
			Lifecycle:   string(body.Lifecycle),
		})
		if err != nil {
			if isUniqueViolation(err, "services_slug_key") {
				return errSlugConflict
			}
			return err
		}
		if err := replaceTags(ctx, q, id, body.Tags); err != nil {
			return err
		}
		if err := writeAudit(ctx, q, actionServiceCreate, id, map[string]any{
			"slug": body.Slug, "name": name, "teamId": body.TeamId,
		}); err != nil {
			return err
		}
		created, err = readService(ctx, q, id)
		return err
	})

	switch {
	case err == nil:
		return CreateService201JSONResponse(created), nil
	case errors.Is(err, errSlugConflict):
		return CreateService409JSONResponse{conflict("slug_taken", "a service with that slug already exists")}, nil
	default:
		return nil, err
	}
}

func (s *Server) UpdateService(ctx context.Context, request UpdateServiceRequestObject) (UpdateServiceResponseObject, error) {
	if !s.dbReady() {
		return UpdateServicedefaultJSONResponse{Body: databaseUnavailable(), StatusCode: http.StatusServiceUnavailable}, nil
	}
	body := request.Body
	if body == nil {
		return UpdateService400JSONResponse{badRequest("request body is required")}, nil
	}

	name := strings.TrimSpace(body.Name)

	var updated Service
	err := s.inTx(ctx, func(q *dbgen.Queries) error {
		// One query serving three purposes: 404 detection, the owning team that
		// authorization needs, and a row lock so a concurrent mutation cannot
		// slip between the check and the write.
		existing, err := q.GetServiceForUpdate(ctx, request.ServiceId)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errNotFound
			}
			return err
		}
		// The role that matters is the one on the team owning the row today.
		if err := requireRole(ctx, existing.TeamID, auth.RoleEditor); err != nil {
			return err
		}
		// Moving a service between teams writes to both, so it takes EDITOR on
		// the destination as well. Without this, an editor on a team they are
		// leaving could hand their services to a team they have no say in.
		if body.TeamId != existing.TeamID {
			if err := requireRole(ctx, body.TeamId, auth.RoleEditor); err != nil {
				return err
			}
		}

		// Slug is absent from ServiceUpdate on purpose: it is the stable handle
		// other things will reference, so there is no rename path and no 409.
		if _, err := q.UpdateServiceRow(ctx, dbgen.UpdateServiceRowParams{
			ID:          request.ServiceId,
			TeamID:      body.TeamId,
			Name:        name,
			Description: textOrEmpty(body.Description),
			RepoUrl:     nilIfBlank(body.RepoUrl),
			RunbookUrl:  nilIfBlank(body.RunbookUrl),
			Lifecycle:   string(body.Lifecycle),
		}); err != nil {
			return err
		}
		if err := replaceTags(ctx, q, request.ServiceId, body.Tags); err != nil {
			return err
		}
		payload := map[string]any{
			"slug": existing.Slug, "name": name, "teamId": body.TeamId,
		}
		// Only when it actually moved. Recording an unchanged previousTeamId on
		// every edit would bury the handful of transfers anyone would search for.
		if body.TeamId != existing.TeamID {
			payload["previousTeamId"] = existing.TeamID
		}
		if err := writeAudit(ctx, q, actionServiceUpdate, request.ServiceId, payload); err != nil {
			return err
		}
		updated, err = readService(ctx, q, request.ServiceId)
		return err
	})

	switch {
	case err == nil:
		return UpdateService200JSONResponse(updated), nil
	case errors.Is(err, errNotFound):
		return UpdateService404JSONResponse{notFound("service not found")}, nil
	case errors.Is(err, errForbidden):
		return UpdateService403JSONResponse{forbidden("EDITOR role required on the owning team")}, nil
	default:
		return nil, err
	}
}

func (s *Server) DeleteService(ctx context.Context, request DeleteServiceRequestObject) (DeleteServiceResponseObject, error) {
	if !s.dbReady() {
		return DeleteServicedefaultJSONResponse{Body: databaseUnavailable(), StatusCode: http.StatusServiceUnavailable}, nil
	}

	err := s.inTx(ctx, func(q *dbgen.Queries) error {
		existing, err := q.GetServiceForUpdate(ctx, request.ServiceId)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errNotFound
			}
			return err
		}
		// Deleting is the one operation a team editor cannot do. It destroys
		// history that nothing else can restore.
		if err := requireRole(ctx, existing.TeamID, auth.RoleAdmin); err != nil {
			return err
		}

		// Captured before the row is gone. Once the DELETE lands, this is the
		// only place the slug, name and owning team still exist.
		payload := map[string]any{
			"slug": existing.Slug, "name": existing.Name, "teamId": existing.TeamID,
		}
		if _, err := q.DeleteServiceRow(ctx, request.ServiceId); err != nil {
			return err
		}
		return writeAudit(ctx, q, actionServiceDelete, request.ServiceId, payload)
	})

	switch {
	case err == nil:
		return DeleteService204Response{}, nil
	case errors.Is(err, errNotFound):
		return DeleteService404JSONResponse{notFound("service not found")}, nil
	case errors.Is(err, errForbidden):
		return DeleteService403JSONResponse{forbidden("ADMIN role required on the owning team")}, nil
	default:
		return nil, err
	}
}

// replaceTags makes the service's tag set exactly what was submitted. Clearing
// first means create and update share one path — on create the DELETE matches
// nothing, which costs a statement and saves a branch.
func replaceTags(ctx context.Context, q *dbgen.Queries, serviceID uuid.UUID, tags *[]string) error {
	if err := q.ClearServiceTags(ctx, serviceID); err != nil {
		return err
	}
	for _, name := range normalizeTags(tags) {
		tagID, err := q.UpsertTag(ctx, name)
		if err != nil {
			return err
		}
		if err := q.LinkServiceTag(ctx, dbgen.LinkServiceTagParams{ServiceID: serviceID, TagID: tagID}); err != nil {
			return err
		}
	}
	return nil
}

// readService re-reads the row through the same query GET uses, so a mutation
// response and a subsequent fetch can never disagree — tags in particular are
// assembled by that query and not by anything above it.
func readService(ctx context.Context, q *dbgen.Queries, id uuid.UUID) (Service, error) {
	row, err := q.GetService(ctx, id)
	if err != nil {
		return Service{}, err
	}
	return serviceFromRow(dbgen.ListServicesRow(row)), nil
}

// textOrEmpty flattens an absent description to the empty string. The column is
// NOT NULL with an empty-string default, and serviceFromRow turns that back into
// an absent field, so the round trip preserves what the caller meant.
func textOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

// nilIfBlank separates "not provided" from "provided empty". repo_url and
// runbook_url are nullable, not optional: an HTML form submits "" for a field
// nobody touched, and storing that renders an empty "repo ↗" link in the
// services table rather than the em dash that means "none".
func nilIfBlank(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// normalizeTags folds case and whitespace so "Go", " go" and "go" are one tag.
// tags is a shared vocabulary keyed by a unique name — without this, the same
// tag typed three ways becomes three rows that no filter can join back up.
func normalizeTags(tags *[]string) []string {
	if tags == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(*tags))
	out := make([]string, 0, len(*tags))
	for _, tag := range *tags {
		name := strings.ToLower(strings.TrimSpace(tag))
		if name == "" {
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
		if len(out) == maxTags {
			break
		}
	}
	return out
}
