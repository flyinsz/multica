-- Backfill CRM email associations where a contact was linked but the account was left empty.
UPDATE crm_email_thread t
SET account_id = c.account_id,
    updated_at = now()
FROM crm_contact c
WHERE t.contact_id = c.id
  AND t.workspace_id = c.workspace_id
  AND t.contact_id IS NOT NULL
  AND t.account_id IS NULL
  AND c.account_id IS NOT NULL;

UPDATE crm_email_message m
SET account_id = c.account_id,
    updated_at = now()
FROM crm_contact c
WHERE m.contact_id = c.id
  AND m.workspace_id = c.workspace_id
  AND m.contact_id IS NOT NULL
  AND m.account_id IS NULL
  AND c.account_id IS NOT NULL;
