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
