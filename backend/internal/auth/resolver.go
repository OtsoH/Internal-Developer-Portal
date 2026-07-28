package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	dbgen "github.com/OtsoH/internal-developer-portal/backend/internal/db/gen"
)

// ErrUnknownUser means the credential was well-formed but names nobody this
// portal knows about.
var ErrUnknownUser = errors.New("unknown user")

// Queries is the slice of the generated query set the resolver needs. Taking an
// interface rather than *dbgen.Queries is what lets the middleware be unit
// tested without a database.
type Queries interface {
	GetUserByEntraOid(ctx context.Context, entraOid *string) (dbgen.User, error)
	GetUserByEmail(ctx context.Context, email string) (dbgen.User, error)
	UpsertUserByEmail(ctx context.Context, arg dbgen.UpsertUserByEmailParams) (dbgen.User, error)
	ListTeamRolesForUser(ctx context.Context, userID uuid.UUID) ([]dbgen.ListTeamRolesForUserRow, error)
}

// UserResolver turns an asserted Identity into a Principal with roles loaded.
type UserResolver struct {
	q Queries
}

func NewUserResolver(q Queries) *UserResolver {
	return &UserResolver{q: q}
}

// Resolve looks up (and in Entra mode provisions) the user, then loads every
// team membership.
//
// Roles are read on every request rather than cached. Two queries per request
// is cheap at this scale, and a TTL cache would make role changes take effect
// late — exactly the wrong property while demonstrating RBAC.
func (r *UserResolver) Resolve(ctx context.Context, id Identity) (Principal, error) {
	var (
		user dbgen.User
		err  error
	)

	if id.EntraOID != "" {
		user, err = r.resolveEntra(ctx, id)
	} else {
		// Dev mode never creates users: an unrecognized email is a 401 rather
		// than a silent role-less account, so a typo fails loudly.
		user, err = r.q.GetUserByEmail(ctx, id.Email)
		if errors.Is(err, pgx.ErrNoRows) {
			return Principal{}, ErrUnknownUser
		}
	}
	if err != nil {
		return Principal{}, err
	}

	rows, err := r.q.ListTeamRolesForUser(ctx, user.ID)
	if err != nil {
		return Principal{}, fmt.Errorf("load team roles: %w", err)
	}

	roles := make(map[uuid.UUID]TeamRole, len(rows))
	for _, row := range rows {
		roles[row.TeamID] = TeamRole{
			TeamID:   row.TeamID,
			TeamSlug: row.TeamSlug,
			TeamName: row.TeamName,
			Role:     Role(row.Role),
		}
	}

	return Principal{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		Roles:  roles,
	}, nil
}

// resolveEntra finds the user by object id, falling back to an email-keyed
// upsert on first sign-in.
//
// The oid lookup comes first so that a user who changed their email address is
// still recognized; going straight to the email upsert would try to attach
// their oid to a different row and trip users_entra_oid_key.
func (r *UserResolver) resolveEntra(ctx context.Context, id Identity) (dbgen.User, error) {
	user, err := r.q.GetUserByEntraOid(ctx, &id.EntraOID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return dbgen.User{}, fmt.Errorf("lookup user by oid: %w", err)
	}

	// First sign-in. The upsert adopts a pre-provisioned row with the same
	// email if one exists, so team roles can be granted before the person has
	// ever logged in.
	user, err = r.q.UpsertUserByEmail(ctx, dbgen.UpsertUserByEmailParams{
		EntraOid: &id.EntraOID,
		Email:    id.Email,
		Name:     id.Name,
	})
	if err != nil {
		return dbgen.User{}, fmt.Errorf("provision user %s: %w", id.Email, err)
	}
	return user, nil
}
