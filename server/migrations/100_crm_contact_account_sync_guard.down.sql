DROP TRIGGER IF EXISTS trg_crm_communication_note_contact_account ON crm_communication_note;
DROP TRIGGER IF EXISTS trg_crm_email_draft_contact_account ON crm_email_draft;
DROP TRIGGER IF EXISTS trg_crm_email_message_contact_account ON crm_email_message;
DROP TRIGGER IF EXISTS trg_crm_email_thread_contact_account ON crm_email_thread;
DROP FUNCTION IF EXISTS crm_infer_account_id_from_contact();
