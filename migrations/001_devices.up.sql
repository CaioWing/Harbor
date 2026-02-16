CREATE TABLE IF NOT EXISTS devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity_hash   VARCHAR(64) UNIQUE NOT NULL,
    identity_data   JSONB NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    auth_token_hash VARCHAR(64),
    inventory       JSONB DEFAULT '{}',
    device_type     VARCHAR(100) NOT NULL,
    tags            TEXT[] DEFAULT '{}',
    last_check_in   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_devices_status ON devices(status);
CREATE INDEX idx_devices_device_type ON devices(device_type);
CREATE INDEX idx_devices_tags ON devices USING GIN(tags);
CREATE INDEX idx_devices_inventory ON devices USING GIN(inventory);
