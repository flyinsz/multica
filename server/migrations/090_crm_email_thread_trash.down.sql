DROP INDEX IF EXISTS idx_crm_email_thread_trash;
ALTER TABLE crm_email_thread DROP COLUMN IF EXISTS is_trashed;
