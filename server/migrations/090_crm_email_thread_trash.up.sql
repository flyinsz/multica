ALTER TABLE crm_email_thread ADD COLUMN IF NOT EXISTS is_trashed BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_crm_email_thread_trash ON crm_email_thread(workspace_id, is_trashed);
