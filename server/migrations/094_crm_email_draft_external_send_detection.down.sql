DROP INDEX IF EXISTS idx_crm_email_draft_detection;

ALTER TABLE crm_email_draft
    DROP COLUMN IF EXISTS sent_detected_at,
    DROP COLUMN IF EXISTS sent_detection_reason,
    DROP COLUMN IF EXISTS sent_detection_confidence,
    DROP COLUMN IF EXISTS sent_detection_status,
    DROP COLUMN IF EXISTS external_sent_mailbox,
    DROP COLUMN IF EXISTS external_sent_uid,
    DROP COLUMN IF EXISTS external_draft_mailbox,
    DROP COLUMN IF EXISTS external_draft_uid;
