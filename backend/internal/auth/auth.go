// Package auth authenticates callers and carries their per-team roles.
//
// Authentication (is this a known user?) happens in the middleware here and
// answers 401. Authorization (may this user do this?) happens in the API
// handlers and answers 403, because the required role usually depends on
// persisted state — the team that owns the service being changed.
package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidToken is returned by a Verifier when a credential is missing,
// malformed or fails verification. The reason is wrapped for logging but never
// shown to the caller.
var ErrInvalidToken = errors.New("invalid token")

// Verifier turns a raw credential into an asserted Identity. Implementations do
// not touch the database — resolving an identity to a user with roles is
// UserResolver's job.
type Verifier interface {
	Verify(ctx context.Context, rawCredential string) (Identity, error)
}

// Identity is what a credential asserts, before any database lookup.
type Identity struct {
	// EntraOID is the object id from the token. Empty in dev mode, where the
	// caller only asserts an email.
	EntraOID string
	// Email is always normalized to lowercase.
	Email string
	Name  string
}

// Role is a role held within a single team. It mirrors the team_members.role
// CHECK constraint and the Role enum in the OpenAPI spec.
//
// This is declared here rather than imported from package api because api
// imports auth; the reverse would be an import cycle. Handlers convert at the
// boundary.
type Role string

const (
	RoleViewer Role = "VIEWER"
	RoleEditor Role = "EDITOR"
	RoleAdmin  Role = "ADMIN"
)

// rank orders roles so they can be compared. The zero Role ranks 0, below
// every real role, so a missing membership never satisfies a requirement.
func (r Role) rank() int {
	switch r {
	case RoleViewer:
		return 1
	case RoleEditor:
		return 2
	case RoleAdmin:
		return 3
	default:
		return 0
	}
}

// AtLeast reports whether r meets or exceeds min. An unrecognized role never
// does, whatever it is compared against.
func (r Role) AtLeast(min Role) bool {
	return r.rank() > 0 && r.rank() >= min.rank()
}

// TeamRole is a team membership: the role plus enough team detail to render a
// picker without a second lookup.
type TeamRole struct {
	TeamID   uuid.UUID
	TeamSlug string
	TeamName string
	Role     Role
}

// Principal is an authenticated user resolved against the database, with every
// membership loaded. Roles are read per request rather than carried in the
// token so that membership changes take effect without re-authenticating.
type Principal struct {
	UserID uuid.UUID
	Email  string
	Name   string
	// Roles is keyed by team id. A user with no entries is still an implicit
	// viewer of the whole catalog; the map only governs writes.
	Roles map[uuid.UUID]TeamRole
}

// RoleIn returns the user's role in a team, or the zero Role if they are not a
// member.
func (p Principal) RoleIn(teamID uuid.UUID) Role {
	return p.Roles[teamID].Role
}

// HasRoleIn reports whether the user holds at least min in the given team.
func (p Principal) HasRoleIn(teamID uuid.UUID, min Role) bool {
	return p.RoleIn(teamID).AtLeast(min)
}

type principalKey struct{}

// WithPrincipal stores an authenticated principal on the context.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom retrieves the principal stored by the authentication
// middleware. The second return is false on unauthenticated requests.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}
