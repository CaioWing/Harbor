CREATE TABLE IF NOT EXISTS artifacts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             VARCHAR(255) NOT NULL,
    version          VARCHAR(100) NOT NULL,
    description      TEXT DEFAULT '',
    file_name        VARCHAR(255) NOT NULL,
    file_size        BIGINT NOT NULL,
    checksum_sha256  VARCHAR(64) NOT NULL,
    target_path      VARCHAR(500) NOT NULL,
    file_mode        VARCHAR(10) DEFAULT '0644',
    file_owner       VARCHAR(100) DEFAULT '',
    device_types     TEXT[] NOT NULL,
    storage_path     VARCHAR(500) NOT NULL,
    pre_install_cmd  TEXT DEFAULT '',
    post_install_cmd TEXT DEFAULT '',
    rollback_cmd     TEXT DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(name, version)
);

CREATE INDEX idx_artifacts_device_types ON artifacts USING GIN(device_types);
