ALTER TABLE crm_email_message
    ADD COLUMN IF NOT EXISTS folder TEXT NOT NULL DEFAULT 'INBOX',
    ADD COLUMN IF NOT EXISTS is_read BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_starred BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_trashed BOOLEAN NOT NULL DEFAULT false;

UPDATE crm_email_message m
SET folder = COALESCE(NULLIF(m.source_metadata->>'folder', ''), 'INBOX')
WHERE folder IS NULL OR folder = '';

UPDATE crm_email_message m
SET is_read = COALESCE(t.is_read, m.is_read),
    is_starred = COALESCE(t.is_starred, m.is_starred),
    is_trashed = COALESCE(t.is_trashed, m.is_trashed),
    folder = CASE
        WHEN t.is_trashed = true OR t.status = 'trashed' THEN 'Trash'
        WHEN t.status = 'archived' THEN 'Archive'
        WHEN t.direction = 'outbound' THEN 'Sent'
        ELSE COALESCE(NULLIF(m.source_metadata->>'folder', ''), m.folder, 'INBOX')
    END
FROM crm_email_thread t
WHERE t.id = m.thread_id AND t.workspace_id = m.workspace_id;

CREATE INDEX IF NOT EXISTS idx_crm_email_message_workspace_folder_time
    ON crm_email_message(workspace_id, folder, COALESCE(sent_at, received_at, created_at) DESC);

CREATE INDEX IF NOT EXISTS idx_crm_email_message_workspace_unread
    ON crm_email_message(workspace_id, folder, is_read)
    WHERE is_read = false;
