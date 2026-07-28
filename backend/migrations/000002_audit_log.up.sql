-- Append-only trail of who changed what. Mutations write one row per successful
-- create/update/delete inside the same transaction as the change itself.
CREATE TABLE audit_log (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Nullable and SET NULL: removing a user must not erase their history.
    actor_id uuid REFERENCES users (id) ON DELETE SET NULL,
    -- No CHECK constraint. Unlike lifecycle and role, the action vocabulary
    -- grows with every feature (spec.upload, dependency.create); Go constants
    -- carry it instead of a migration per verb.
    action text NOT NULL,
    entity_type text NOT NULL,
    -- Deliberately no foreign key: deletes are hard, and the audit row
    -- outliving the entity it describes is the entire point.
    entity_id uuid NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_entity ON audit_log (entity_type, entity_id, at DESC);
CREATE INDEX idx_audit_log_at ON audit_log (at DESC);
CREATE INDEX idx_audit_log_actor ON audit_log (actor_id, at DESC);
