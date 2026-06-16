-- Baseline v1 : journaux d'audit partitionnés.

CREATE TABLE audit_logs (
    id UUID NOT NULL DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    action VARCHAR(20) NOT NULL,
    resource_type VARCHAR(30) NOT NULL,
    resource_id VARCHAR(255),
    status VARCHAR(10) NOT NULL,
    status_code INTEGER NOT NULL,
    error_message TEXT,
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    details TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Partition par défaut (attrape-tout de sécurité)
CREATE TABLE audit_logs_default PARTITION OF audit_logs DEFAULT;

CREATE INDEX idx_audit_logs_tenant_created ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_status ON audit_logs (status);
