-- Recherche plein-texte partielle : rendre les ILIKE '%x%' sargables.
-- Sans index trigramme, la recherche sur le sujet et l'email des destinataires
-- provoque des scans séquentiels (wildcard en tête, non-indexable par un B-tree).

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_mails_subject_trgm ON mails USING GIN (subject gin_trgm_ops);
CREATE INDEX idx_mail_recipients_email_trgm ON mail_recipients USING GIN (email gin_trgm_ops);
