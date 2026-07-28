-- The no-op DO UPDATE is load-bearing: DO NOTHING returns zero rows, so
-- RETURNING id would yield pgx.ErrNoRows for every tag that already exists.
-- name: UpsertTag :one
INSERT INTO tags (name)
VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING id;

-- name: LinkServiceTag :exec
INSERT INTO service_tags (service_id, tag_id)
VALUES ($1, $2)
ON CONFLICT (service_id, tag_id) DO NOTHING;

-- Clears the links only. Rows in tags are left behind on purpose: they are a
-- shared vocabulary, not per-service data.
-- name: ClearServiceTags :exec
DELETE FROM service_tags WHERE service_id = $1;
