ALTER TABLE crm_email_draft
    DROP COLUMN IF EXISTS sent_append_warning,
    DROP COLUMN IF EXISTS sent_append_enabled,
    DROP COLUMN IF EXISTS attachments,
    DROP COLUMN IF EXISTS reference_ids,
    DROP COLUMN IF EXISTS in_reply_to;
