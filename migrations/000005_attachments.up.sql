-- Pièces jointes dédupliquées et adressées par contenu (content-addressed).
-- Objectif : une même pièce jointe envoyée à N destinataires d'une campagne
-- n'est stockée qu'une seule fois par tenant (clé = SHA-256 du contenu), au lieu
-- d'être recopiée en base64 dans chaque ligne mails.attachments.

-- Métadonnées + référence. Le contenu vit soit dans attachment_blobs (backend
-- postgres), soit dans un object store externe (backend s3), selon la colonne storage.
CREATE TABLE attachments (
    id           UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sha256       BYTEA NOT NULL,          -- hash du contenu brut (32 octets)
    size_bytes   BIGINT NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    storage      TEXT NOT NULL,           -- 'postgres' | 's3' : backend détenant le blob
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Déduplication par tenant : un contenu donné n'a qu'une ligne par tenant.
    UNIQUE (tenant_id, sha256)
);

-- Index pour le balayage GC : on parcourt par date de création, et l'orphelinage
-- est calculé à la volée (NOT EXISTS sur mail_attachments) plutôt que via un
-- compteur dénormalisé — celui-ci dériverait lors des suppressions en cascade
-- (archivage des mails), laissant des blobs jamais collectés.
CREATE INDEX idx_attachments_created ON attachments (created_at);

-- Contenu pour le backend postgres uniquement. Adressé par (tenant_id, sha256),
-- sans FK vers attachments : le BlobStore reste découplé de la table de
-- métadonnées (symétrique avec l'adaptateur S3 dont la clé est tenant_id/sha256).
-- Table séparée pour que les SELECT de métadonnées (attachments) ne tirent jamais
-- le BYTEA lourd via TOAST.
CREATE TABLE attachment_blobs (
    tenant_id UUID  NOT NULL,
    sha256    BYTEA NOT NULL,
    data      BYTEA NOT NULL,
    PRIMARY KEY (tenant_id, sha256)
);

-- Lien mail -> pièce jointe. Le filename est propre au mail (deux mails peuvent
-- référencer le même contenu sous des noms différents) ; la dédup porte sur le
-- contenu (sha256), pas sur le nom.
CREATE TABLE mail_attachments (
    mail_id       UUID NOT NULL REFERENCES mails(id) ON DELETE CASCADE,
    attachment_id UUID NOT NULL REFERENCES attachments(id),
    filename      TEXT NOT NULL,
    position      INT  NOT NULL DEFAULT 0,
    PRIMARY KEY (mail_id, attachment_id, filename)
);

CREATE INDEX idx_mail_attachments_attachment ON mail_attachments (attachment_id);
