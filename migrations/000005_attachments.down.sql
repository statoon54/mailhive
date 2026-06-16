-- Dev-only rollback. N'utilisez pas en production.
DROP TABLE IF EXISTS mail_attachments;
DROP TABLE IF EXISTS attachment_blobs;
DROP TABLE IF EXISTS attachments;
