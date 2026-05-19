ALTER TABLE issue DROP CONSTRAINT IF EXISTS issue_origin_type_check;
ALTER TABLE issue ADD CONSTRAINT issue_origin_type_check
    CHECK (origin_type IN ('autopilot', 'quick_create'));

ALTER TABLE crm_ai_setting DROP CONSTRAINT IF EXISTS crm_ai_setting_automation_key_check;
ALTER TABLE crm_ai_setting ADD CONSTRAINT crm_ai_setting_automation_key_check
    CHECK (automation_key IN ('email_pending_reply', 'due_followup'));
