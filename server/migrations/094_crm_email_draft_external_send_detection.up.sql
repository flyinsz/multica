ALTER TABLE crm_email_draft
    ADD COLUMN IF NOT EXISTS external_draft_uid TEXT,
    ADD COLUMN IF NOT EXISTS external_draft_mailbox TEXT,
    ADD COLUMN IF NOT EXISTS external_sent_uid TEXT,
    ADD COLUMN IF NOT EXISTS external_sent_mailbox TEXT,
    ADD COLUMN IF NOT EXISTS sent_detection_status TEXT,
    ADD COLUMN IF NOT EXISTS sent_detection_confidence INTEGER,
    ADD COLUMN IF NOT EXISTS sent_detection_reason TEXT,
    ADD COLUMN IF NOT EXISTS sent_detected_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_crm_email_draft_detection
    ON crm_email_draft(workspace_id, sent_detection_status, sent_detected_at DESC)
    WHERE issue_id IS NOT NULL;
