-- Dev-only rollback. N'utilisez pas en production.
DROP TABLE IF EXISTS mail_recipients_archive;
DROP TABLE IF EXISTS mails_archive_default;
DROP TABLE IF EXISTS mails_archive;
DROP TABLE IF EXISTS mail_recipients;
DROP TABLE IF EXISTS mails;
DROP TYPE IF EXISTS recipient_type;
DROP TYPE IF EXISTS mail_priority;
DROP TYPE IF EXISTS mail_status;
