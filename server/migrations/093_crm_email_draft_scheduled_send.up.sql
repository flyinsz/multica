ALTER TABLE crm_email_draft
    ADD COLUMN IF NOT EXISTS scheduled_send_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS scheduled_send_last_attempt_at TIMESTAMPTZ;

ALTER TABLE crm_email_draft
    DROP CONSTRAINT IF EXISTS crm_email_draft_status_check;

ALTER TABLE crm_email_draft
    ADD CONSTRAINT crm_email_draft_status_check CHECK (status IN ('draft','pending_approval','scheduled','sending','sent','discarded','failed'));

CREATE INDEX IF NOT EXISTS idx_crm_email_draft_scheduled_due ON crm_email_draft(workspace_id, scheduled_send_at) WHERE status='scheduled' AND scheduled_send_at IS NOT NULL;
