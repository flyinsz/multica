DROP INDEX IF EXISTS idx_crm_email_thread_flags;
ALTER TABLE crm_email_thread DROP COLUMN IF EXISTS is_starred;
ALTER TABLE crm_email_thread DROP COLUMN IF EXISTS is_read;
