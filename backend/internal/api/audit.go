package api

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/OtsoH/internal-developer-portal/backend/internal/auth"
	dbgen "github.com/OtsoH/internal-developer-portal/backend/internal/db/gen"
)

// The action vocabulary. audit_log deliberately carries no CHECK constraint on
// this column — unlike lifecycle and role, it grows with every feature — so
// these constants are the whole definition, and adding a verb costs no
// migration.
const (
	actionServiceCreate = "service.create"
	actionServiceUpdate = "service.update"
	actionServiceDelete = "service.delete"

	entityService = "service"
)

// writeAudit records one mutation. It takes the transaction's queries rather
// than the server's, so the trail commits with the change it describes: a
// service that exists always has the row saying who created it, and a rolled-back
// attempt leaves nothing behind.
//
// A nil actor is not an error. audit_log.actor_id is nullable so that deleting a
// user cannot erase their history, and the same nullability covers the
// no-authenticator mode where there is no principal to name.
func writeAudit(ctx context.Context, q *dbgen.Queries, action string, serviceID uuid.UUID, payload map[string]any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var actor *uuid.UUID
	if principal, ok := auth.PrincipalFrom(ctx); ok {
		actor = &principal.UserID
	}

	return q.InsertAuditLog(ctx, dbgen.InsertAuditLogParams{
		ActorID:    actor,
		Action:     action,
		EntityType: entityService,
		EntityID:   serviceID,
		Payload:    encoded,
	})
}
