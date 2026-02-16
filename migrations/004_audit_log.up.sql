CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor       VARCHAR(255) NOT NULL,           -- user_id or device_id
    actor_type  VARCHAR(20) NOT NULL,            -- management, device, system
    action      VARCHAR(100) NOT NULL,           -- e.g. device.accept, artifact.upload
    resource    VARCHAR(100) NOT NULL DEFAULT '', -- e.g. device, artifact, deployment
    resource_id VARCHAR(100) DEFAULT '',         -- UUID of the resource
    details     JSONB DEFAULT '{}',              -- additional context
    ip_address  VARCHAR(45) DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_actor ON audit_log(actor);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_resource ON audit_log(resource, resource_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);
