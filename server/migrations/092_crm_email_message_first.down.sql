DROP INDEX IF EXISTS idx_crm_email_message_workspace_unread;
DROP INDEX IF EXISTS idx_crm_email_message_workspace_folder_time;

ALTER TABLE crm_email_message
    DROP COLUMN IF EXISTS is_trashed,
    DROP COLUMN IF EXISTS is_starred,
    DROP COLUMN IF EXISTS is_read,
    DROP COLUMN IF EXISTS folder;
