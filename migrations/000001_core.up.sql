-- Baseline v1 : tables de configuration (tenants, SMTP, templates, branding).

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    api_key VARCHAR(64) NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    settings JSONB NOT NULL DEFAULT '{"rate_limit": 10, "rate_burst": 20, "max_destinataires": 1000}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_api_key ON tenants(api_key);
CREATE INDEX idx_tenants_is_active ON tenants(is_active);

-- Configurations SMTP par tenant
CREATE TABLE smtp_configs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL DEFAULT 587,
    username VARCHAR(255),
    password TEXT,
    auth_method VARCHAR(20) NOT NULL DEFAULT 'PLAIN',
    tls_policy VARCHAR(20) NOT NULL DEFAULT 'opportunistic',
    from_email VARCHAR(255) NOT NULL,
    from_name VARCHAR(255) NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    charset VARCHAR(20) NOT NULL DEFAULT 'UTF-8',
    encoding VARCHAR(20) NOT NULL DEFAULT 'quoted-printable',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_smtp_configs_tenant_id ON smtp_configs(tenant_id);

-- Contrainte : un seul is_default=true par tenant
CREATE UNIQUE INDEX idx_smtp_configs_default_per_tenant
    ON smtp_configs(tenant_id) WHERE is_default = true;

-- Templates de mails
CREATE TABLE mail_templates (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    subject_tmpl TEXT NOT NULL DEFAULT '',
    text_body TEXT NOT NULL DEFAULT '',
    html_body TEXT NOT NULL DEFAULT '',
    variables JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, slug)
);

CREATE INDEX idx_mail_templates_tenant_id ON mail_templates(tenant_id);

-- Branding applicatif (ligne unique : id = 1)
CREATE TABLE IF NOT EXISTS app_branding (
    id          INTEGER PRIMARY KEY CHECK (id = 1),
    app_title   VARCHAR(255) NOT NULL DEFAULT 'MailHive',
    app_subtitle VARCHAR(255) NOT NULL DEFAULT 'Gestion des mails',
    logo_data   TEXT NOT NULL DEFAULT '',
    logo_content_type VARCHAR(100) NOT NULL DEFAULT '',
    timezone    VARCHAR(50) NOT NULL DEFAULT 'Europe/Paris',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO app_branding (id) VALUES (1) ON CONFLICT DO NOTHING;
