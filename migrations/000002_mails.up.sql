-- Baseline v1 : données mail (envois + archivage) et types ENUM partagés.

-- ENUMs (doivent exister avant les tables qui les référencent)
CREATE TYPE mail_status AS ENUM ('pending', 'queued', 'sending', 'sent', 'failed', 'cancelled');
CREATE TYPE mail_priority AS ENUM ('critical', 'default', 'low');
CREATE TYPE recipient_type AS ENUM ('to', 'cc', 'bcc');

-- Table principale des mails
CREATE TABLE mails (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    smtp_config_id UUID REFERENCES smtp_configs(id) ON DELETE SET NULL,
    template_id UUID REFERENCES mail_templates(id) ON DELETE SET NULL,
    from_email VARCHAR(255) NOT NULL,
    from_name VARCHAR(255) NOT NULL DEFAULT '',
    subject TEXT NOT NULL,
    text_body TEXT NOT NULL DEFAULT '',
    html_body TEXT NOT NULL DEFAULT '',
    template_data JSONB NOT NULL DEFAULT '{}',
    status mail_status NOT NULL DEFAULT 'pending',
    status_message TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0,
    priority mail_priority NOT NULL DEFAULT 'default',
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    task_id VARCHAR(255) NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Colonnes ajoutées par ALTER TABLE dans l'historique (legacy 000009, 000011),
    -- conservées en fin de table pour préserver l'équivalence du schéma post-squash.
    spam_score REAL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    compressed_body BYTEA
);

CREATE INDEX idx_mails_tenant_status_created ON mails (tenant_id, status, created_at DESC);
CREATE INDEX idx_mails_created_at ON mails(created_at DESC);
CREATE INDEX idx_mails_tags ON mails USING GIN (tags);

-- Destinataires
CREATE TABLE mail_recipients (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    mail_id UUID NOT NULL REFERENCES mails(id) ON DELETE CASCADE,
    type recipient_type NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT ''
);

CREATE INDEX idx_mail_recipients_mail_id ON mail_recipients(mail_id);

-- Archive des mails terminés de plus de 90 jours.
-- Partitionnée, sans FK pour la performance.
CREATE TABLE mails_archive (
    id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    smtp_config_id UUID,
    template_id UUID,
    from_email VARCHAR(255) NOT NULL,
    from_name VARCHAR(255) NOT NULL DEFAULT '',
    subject TEXT NOT NULL,
    text_body TEXT NOT NULL DEFAULT '',
    html_body TEXT NOT NULL DEFAULT '',
    template_data JSONB NOT NULL DEFAULT '{}',
    status mail_status NOT NULL,
    status_message TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0,
    priority mail_priority NOT NULL DEFAULT 'default',
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    task_id VARCHAR(255) NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Colonnes ajoutées par ALTER TABLE dans l'historique (legacy 000009, 000011),
    -- conservées en fin de table pour préserver l'équivalence du schéma post-squash.
    spam_score REAL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    compressed_body BYTEA,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Partition par défaut (attrape-tout)
CREATE TABLE mails_archive_default PARTITION OF mails_archive DEFAULT;

-- Archive des destinataires
CREATE TABLE mail_recipients_archive (
    id UUID NOT NULL,
    mail_id UUID NOT NULL,
    type recipient_type NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT ''
);

CREATE INDEX idx_mails_archive_tenant_created ON mails_archive (tenant_id, created_at DESC);
CREATE INDEX idx_mail_recipients_archive_mail_id ON mail_recipients_archive (mail_id);
