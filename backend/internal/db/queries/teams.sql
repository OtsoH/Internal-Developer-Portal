-- name: ListTeams :many
SELECT id, name, slug, created_at
FROM teams
ORDER BY name;
