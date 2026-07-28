-- name: InsertAuditLog :exec
INSERT INTO audit_log (actor_id, action, entity_type, entity_id, payload)
VALUES ($1, $2, $3, $4, $5);
