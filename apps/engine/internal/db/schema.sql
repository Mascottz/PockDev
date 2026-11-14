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
