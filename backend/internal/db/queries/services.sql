-- name: ListServices :many
SELECT s.id,
       s.name,
       s.slug,
       s.description,
       s.repo_url,
       s.runbook_url,
       s.lifecycle,
       s.created_at,
       s.updated_at,
       t.id   AS team_id,
       t.name AS team_name,
       t.slug AS team_slug,
       coalesce(
           array_agg(tg.name ORDER BY tg.name) FILTER (WHERE tg.id IS NOT NULL),
           '{}'
       )::text[] AS tags
FROM services s
JOIN teams t ON t.id = s.team_id
LEFT JOIN service_tags st ON st.service_id = s.id
LEFT JOIN tags tg ON tg.id = st.tag_id
GROUP BY s.id, t.id
ORDER BY s.name;

-- name: GetService :one
SELECT s.id,
       s.name,
       s.slug,
       s.description,
       s.repo_url,
       s.runbook_url,
       s.lifecycle,
       s.created_at,
       s.updated_at,
       t.id   AS team_id,
       t.name AS team_name,
       t.slug AS team_slug,
       coalesce(
           array_agg(tg.name ORDER BY tg.name) FILTER (WHERE tg.id IS NOT NULL),
           '{}'
       )::text[] AS tags
FROM services s
JOIN teams t ON t.id = s.team_id
LEFT JOIN service_tags st ON st.service_id = s.id
LEFT JOIN tags tg ON tg.id = st.tag_id
WHERE s.id = $1
GROUP BY s.id, t.id;

-- Does triple duty for PUT and DELETE: 404 detection, the owning-team lookup
-- that authorization needs, and a row lock so a concurrent mutation on the same
-- service cannot interleave between the check and the write.
-- name: GetServiceForUpdate :one
SELECT id, team_id, slug, name
FROM services
WHERE id = $1
FOR UPDATE;

-- name: InsertService :one
INSERT INTO services (team_id, name, slug, description, repo_url, runbook_url, lifecycle)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- updated_at is set explicitly because the table has no trigger. A trigger was
-- considered and rejected: an invisible mutation sqlc cannot see is worse than
-- one visible line here.
-- name: UpdateServiceRow :one
UPDATE services
SET team_id = $2,
    name = $3,
    description = $4,
    repo_url = $5,
    runbook_url = $6,
    lifecycle = $7,
    updated_at = now()
WHERE id = $1
RETURNING id;

-- name: DeleteServiceRow :execrows
DELETE FROM services WHERE id = $1;
