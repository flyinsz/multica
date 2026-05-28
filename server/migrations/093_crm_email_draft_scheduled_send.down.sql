DROP INDEX IF EXISTS idx_crm_email_draft_scheduled_due;

ALTER TABLE crm_email_draft
    DROP CONSTRAINT IF EXISTS crm_email_draft_status_check;

ALTER TABLE crm_email_draft
    ADD CONSTRAINT crm_email_draft_status_check CHECK (status IN ('draft','pending_approval','sent','discarded','failed'));

ALTER TABLE crm_email_draft
    DROP COLUMN IF EXISTS scheduled_send_last_attempt_at,
    DROP COLUMN IF EXISTS scheduled_send_at;
