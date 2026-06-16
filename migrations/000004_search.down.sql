DROP INDEX IF EXISTS idx_mail_recipients_email_trgm;
DROP INDEX IF EXISTS idx_mails_subject_trgm;
-- L'extension pg_trgm est laissée en place (potentiellement partagée).
