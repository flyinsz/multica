ALTER TABLE crm_email_draft
    ADD COLUMN IF NOT EXISTS issue_id UUID REFERENCES issue(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS approval_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_crm_email_draft_issue ON crm_email_draft(issue_id) WHERE issue_id IS NOT NULL;
