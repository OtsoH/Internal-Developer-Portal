CREATE TABLE teams (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    slug       text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entra_oid  text UNIQUE,
    email      text NOT NULL UNIQUE,
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE team_members (
    team_id uuid NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role    text NOT NULL CHECK (role IN ('ADMIN', 'EDITOR', 'VIEWER')),
    PRIMARY KEY (team_id, user_id)
);

CREATE TABLE services (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     uuid NOT NULL REFERENCES teams (id),
    name        text NOT NULL,
    slug        text NOT NULL UNIQUE,
    description text NOT NULL DEFAULT '',
    repo_url    text,
    runbook_url text,
    lifecycle   text NOT NULL CHECK (lifecycle IN ('production', 'beta', 'deprecated')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_services_team_id ON services (team_id);

CREATE TABLE tags (
    id   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL UNIQUE
);

CREATE TABLE service_tags (
    service_id uuid NOT NULL REFERENCES services (id) ON DELETE CASCADE,
    tag_id     uuid NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (service_id, tag_id)
);

CREATE INDEX idx_service_tags_tag_id ON service_tags (tag_id);
