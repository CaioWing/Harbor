CREATE TABLE IF NOT EXISTS deployments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(255) NOT NULL,
    artifact_id         UUID NOT NULL REFERENCES artifacts(id),
    status              VARCHAR(20) NOT NULL DEFAULT 'scheduled',
    target_device_ids   UUID[],
    target_device_tags  TEXT[],
    target_device_types TEXT[],
    max_parallel        INT DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at          TIMESTAMPTZ,
    finished_at         TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS deployment_devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id   UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    device_id       UUID NOT NULL REFERENCES devices(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempts        INT DEFAULT 0,
    log             TEXT DEFAULT '',
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,

    UNIQUE(deployment_id, device_id)
);

CREATE INDEX idx_dd_deployment ON deployment_devices(deployment_id);
CREATE INDEX idx_dd_device ON deployment_devices(device_id);
CREATE INDEX idx_dd_status ON deployment_devices(status);
