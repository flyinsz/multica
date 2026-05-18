ALTER TABLE crm_email_thread ADD COLUMN IF NOT EXISTS is_read BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE crm_email_thread ADD COLUMN IF NOT EXISTS is_starred BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE crm_email_thread SET is_read = TRUE WHERE direction = 'outbound';
CREATE INDEX IF NOT EXISTS idx_crm_email_thread_flags ON crm_email_thread(workspace_id, is_read, is_starred);
