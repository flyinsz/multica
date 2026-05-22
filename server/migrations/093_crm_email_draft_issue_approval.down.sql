DROP INDEX IF EXISTS idx_crm_email_draft_issue;

ALTER TABLE crm_email_draft
    DROP COLUMN IF EXISTS approval_reason,
    DROP COLUMN IF EXISTS issue_id;
