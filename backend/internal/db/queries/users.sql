-- name: GetUserByEntraOid :one
SELECT id, entra_oid, email, name, created_at
FROM users
WHERE entra_oid = $1;

-- Email comparison is exact rather than lower(email) = lower($1): a functional
-- comparison cannot use users_email_key. Callers normalize in Go before both
-- the lookup and the insert.
-- name: GetUserByEmail :one
SELECT id, entra_oid, email, name, created_at
FROM users
WHERE email = $1;

-- COALESCE keeps an already-linked oid rather than overwriting it, which lets a
-- seeded or pre-provisioned user (entra_oid IS NULL) be adopted by a real Entra
-- login with the same address. Team roles can therefore be granted before the
-- person has ever signed in.
-- name: UpsertUserByEmail :one
INSERT INTO users (entra_oid, email, name)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE
    SET entra_oid = COALESCE(users.entra_oid, EXCLUDED.entra_oid),
        name = EXCLUDED.name
RETURNING id, entra_oid, email, name, created_at;

-- Loaded once per request into the principal, so there is no single-team
-- variant: every authorization check reads the map already in memory.
-- name: ListTeamRolesForUser :many
SELECT tm.team_id,
       t.slug AS team_slug,
       t.name AS team_name,
       tm.role
FROM team_members tm
JOIN teams t ON t.id = tm.team_id
WHERE tm.user_id = $1
ORDER BY t.name;
