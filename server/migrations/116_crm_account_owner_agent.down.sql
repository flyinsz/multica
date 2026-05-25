DROP INDEX IF EXISTS idx_crm_account_owner_agent;
ALTER TABLE crm_account DROP CONSTRAINT IF EXISTS crm_account_owner_type_check;
ALTER TABLE crm_account DROP COLUMN IF EXISTS owner_agent_id;
ALTER TABLE crm_account DROP COLUMN IF EXISTS owner_type;
