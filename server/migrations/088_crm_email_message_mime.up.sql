ALTER TABLE crm_email_message
    ADD COLUMN IF NOT EXISTS raw_size_bytes INTEGER,
    ADD COLUMN IF NOT EXISTS in_reply_to TEXT,
    ADD COLUMN IF NOT EXISTS reference_ids TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS raw_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS attachments JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_crm_email_message_in_reply_to ON crm_email_message(workspace_id, in_reply_to) WHERE in_reply_to IS NOT NULL;
