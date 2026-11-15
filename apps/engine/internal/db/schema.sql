-- PockDev engine schema. Idempotent: safe to run on every startup.

CREATE TABLE IF NOT EXISTS projects (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    slug           TEXT NOT NULL UNIQUE,
    endpoint_key   TEXT NOT NULL UNIQUE,
    sdk_key        TEXT NOT NULL UNIQUE,
    target_url     TEXT NOT NULL DEFAULT '',
    github_repo    TEXT NOT NULL DEFAULT '',
    whatsapp_phone TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'active',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Add the multi-tenant Stage 2 tables and migrate projects in place. Safe to
-- run on every startup.

CREATE TABLE IF NOT EXISTS workspaces (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    slug          TEXT NOT NULL UNIQUE,
    owner_user_id TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO workspaces (id, name, slug) VALUES ('wsp_default', 'Default Workspace', 'default')
    ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS users (
    id                  TEXT PRIMARY KEY,
    github_id           BIGINT NOT NULL UNIQUE,
    github_login        TEXT NOT NULL,
    name                TEXT NOT NULL DEFAULT '',
    email               TEXT NOT NULL DEFAULT '',
    avatar_url          TEXT NOT NULL DEFAULT '',
    default_workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS workspace_members (
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    user_id      TEXT NOT NULL REFERENCES users(id),
    role         TEXT NOT NULL DEFAULT 'owner',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, user_id)
);

CREATE TABLE IF NOT EXISTS sessions (
    token_hash   TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id),
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE projects ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT 'wsp_default';
ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_slug_key;
CREATE UNIQUE INDEX IF NOT EXISTS projects_workspace_slug_idx ON projects (workspace_id, slug);
CREATE INDEX IF NOT EXISTS projects_workspace_idx ON projects (workspace_id, created_at ASC);

-- Stage 3 billing: plan columns on workspaces plus idempotent payment records.
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS plan TEXT NOT NULL DEFAULT 'free';
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS plan_status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS plan_expires_at TIMESTAMPTZ;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS paystack_customer_code TEXT NOT NULL DEFAULT '';
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS paystack_subscription_code TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS billing_events (
    reference    TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    amount_kobo  BIGINT NOT NULL DEFAULT 0,
    currency     TEXT NOT NULL DEFAULT 'NGN',
    status       TEXT NOT NULL DEFAULT 'success',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS deliveries (
    id                  TEXT PRIMARY KEY,
    project_id          TEXT NOT NULL REFERENCES projects(id),
    endpoint_key        TEXT NOT NULL DEFAULT '',
    status              TEXT NOT NULL DEFAULT 'retrying',
    http_status         INTEGER NOT NULL DEFAULT 0,
    attempt_count       INTEGER NOT NULL DEFAULT 0,
    schema_drift_note   TEXT NOT NULL DEFAULT '',
    payload_preview     TEXT NOT NULL DEFAULT '',
    payload_keys        JSONB NOT NULL DEFAULT '[]',
    headers             JSONB NOT NULL DEFAULT '{}',
    storage_key         TEXT NOT NULL DEFAULT '',
    linked_incident_id  TEXT NOT NULL DEFAULT '',
    last_error          TEXT NOT NULL DEFAULT '',
    next_attempt_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS deliveries_due_idx
    ON deliveries (status, next_attempt_at);
CREATE INDEX IF NOT EXISTS deliveries_project_idx
    ON deliveries (project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS deliveries_incident_idx
    ON deliveries (linked_incident_id);

CREATE TABLE IF NOT EXISTS incidents (
    id               TEXT PRIMARY KEY,
    project_id       TEXT NOT NULL REFERENCES projects(id),
    title            TEXT NOT NULL,
    plain_summary    TEXT NOT NULL DEFAULT '',
    file_path        TEXT NOT NULL DEFAULT '',
    commit_sha       TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'open',
    paused_count     INTEGER NOT NULL DEFAULT 0,
    pr_url           TEXT NOT NULL DEFAULT '',
    origin           TEXT NOT NULL DEFAULT '',
    origin_ref       TEXT NOT NULL DEFAULT '',
    mute_until       TIMESTAMPTZ,
    last_alert_phone TEXT NOT NULL DEFAULT '',
    last_alert_at    TIMESTAMPTZ,
    last_seen_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS incidents_project_idx
    ON incidents (project_id, created_at DESC);

CREATE TABLE IF NOT EXISTS error_groups (
    id                 TEXT PRIMARY KEY,
    project_id         TEXT NOT NULL REFERENCES projects(id),
    fingerprint        TEXT NOT NULL,
    title              TEXT NOT NULL,
    file_path          TEXT NOT NULL DEFAULT '',
    line               INTEGER NOT NULL DEFAULT 0,
    event_count        INTEGER NOT NULL DEFAULT 0,
    window_count       INTEGER NOT NULL DEFAULT 0,
    window_started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    incident_id        TEXT NOT NULL DEFAULT '',
    first_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, fingerprint)
);

CREATE TABLE IF NOT EXISTS error_events (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id),
    group_id    TEXT NOT NULL REFERENCES error_groups(id),
    message     TEXT NOT NULL,
    stack       TEXT NOT NULL DEFAULT '',
    file_path   TEXT NOT NULL DEFAULT '',
    line        INTEGER NOT NULL DEFAULT 0,
    delivery_id TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS error_events_delivery_idx
    ON error_events (delivery_id);
