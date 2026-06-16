-- Dev-only rollback. N'utilisez pas en production.
DROP TABLE IF EXISTS app_branding;
DROP TABLE IF EXISTS mail_templates;
DROP TABLE IF EXISTS smtp_configs;
DROP TABLE IF EXISTS tenants;
-- L'extension uuid-ossp est laissée en place (partagée).
