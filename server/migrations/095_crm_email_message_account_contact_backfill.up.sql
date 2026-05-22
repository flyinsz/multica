-- Backfill CRM email message associations from existing thread/contact links.
UPDATE crm_email_thread t
SET account_id = COALESCE(t.account_id, c.account_id),
    updated_at = now()
FROM crm_contact c
WHERE c.workspace_id = t.workspace_id
  AND c.id = t.contact_id
  AND t.account_id IS NULL;

UPDATE crm_email_message m
SET account_id = COALESCE(
        m.account_id,
        (SELECT t.account_id FROM crm_email_thread t WHERE t.workspace_id=m.workspace_id AND t.id=m.thread_id),
        (SELECT c.account_id FROM crm_contact c WHERE c.workspace_id=m.workspace_id AND c.id=m.contact_id)
    ),
    contact_id = COALESCE(
        m.contact_id,
        (SELECT t.contact_id FROM crm_email_thread t WHERE t.workspace_id=m.workspace_id AND t.id=m.thread_id)
    ),
    updated_at = now()
WHERE m.account_id IS NULL OR m.contact_id IS NULL;
