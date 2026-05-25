ALTER TABLE crm_account
  ADD COLUMN IF NOT EXISTS owner_type text NOT NULL DEFAULT 'member',
  ADD COLUMN IF NOT EXISTS owner_agent_id uuid NULL REFERENCES agent(id) ON DELETE SET NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'crm_account_owner_type_check'
  ) THEN
    ALTER TABLE crm_account
      ADD CONSTRAINT crm_account_owner_type_check CHECK (owner_type IN ('member', 'agent'));
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_crm_account_owner_agent ON crm_account(owner_agent_id) WHERE owner_agent_id IS NOT NULL;
