-- Keep CRM email/contact associations consistent at database level.
-- Any row with contact_id must inherit crm_contact.account_id when account_id is empty.

CREATE OR REPLACE FUNCTION crm_infer_account_id_from_contact()
RETURNS trigger AS $$
BEGIN
  IF NEW.contact_id IS NOT NULL AND NEW.account_id IS NULL THEN
    SELECT c.account_id INTO NEW.account_id
    FROM crm_contact c
    WHERE c.id = NEW.contact_id
      AND c.workspace_id = NEW.workspace_id
      AND c.account_id IS NOT NULL
    LIMIT 1;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_crm_email_thread_contact_account ON crm_email_thread;
CREATE TRIGGER trg_crm_email_thread_contact_account
BEFORE INSERT OR UPDATE OF contact_id, account_id, workspace_id ON crm_email_thread
FOR EACH ROW EXECUTE FUNCTION crm_infer_account_id_from_contact();

DROP TRIGGER IF EXISTS trg_crm_email_message_contact_account ON crm_email_message;
CREATE TRIGGER trg_crm_email_message_contact_account
BEFORE INSERT OR UPDATE OF contact_id, account_id, workspace_id ON crm_email_message
FOR EACH ROW EXECUTE FUNCTION crm_infer_account_id_from_contact();

DROP TRIGGER IF EXISTS trg_crm_email_draft_contact_account ON crm_email_draft;
CREATE TRIGGER trg_crm_email_draft_contact_account
BEFORE INSERT OR UPDATE OF contact_id, account_id, workspace_id ON crm_email_draft
FOR EACH ROW EXECUTE FUNCTION crm_infer_account_id_from_contact();

DROP TRIGGER IF EXISTS trg_crm_communication_note_contact_account ON crm_communication_note;
CREATE TRIGGER trg_crm_communication_note_contact_account
BEFORE INSERT OR UPDATE OF contact_id, account_id, workspace_id ON crm_communication_note
FOR EACH ROW EXECUTE FUNCTION crm_infer_account_id_from_contact();

UPDATE crm_email_thread t
SET account_id = c.account_id,
    updated_at = now()
FROM crm_contact c
WHERE t.contact_id = c.id
  AND t.workspace_id = c.workspace_id
  AND t.account_id IS NULL
  AND c.account_id IS NOT NULL;

UPDATE crm_email_message m
SET account_id = c.account_id,
    updated_at = now()
FROM crm_contact c
WHERE m.contact_id = c.id
  AND m.workspace_id = c.workspace_id
  AND m.account_id IS NULL
  AND c.account_id IS NOT NULL;

UPDATE crm_email_draft d
SET account_id = c.account_id,
    updated_at = now()
FROM crm_contact c
WHERE d.contact_id = c.id
  AND d.workspace_id = c.workspace_id
  AND d.account_id IS NULL
  AND c.account_id IS NOT NULL;

UPDATE crm_communication_note n
SET account_id = c.account_id,
    updated_at = now()
FROM crm_contact c
WHERE n.contact_id = c.id
  AND n.workspace_id = c.workspace_id
  AND n.account_id IS NULL
  AND c.account_id IS NOT NULL;
