CREATE TABLE IF NOT EXISTS crm_ai_run (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
  automation_key text NOT NULL,
  status text NOT NULL DEFAULT 'success',
  started_at timestamptz NOT NULL DEFAULT now(),
  finished_at timestamptz NOT NULL DEFAULT now(),
  result jsonb NOT NULL DEFAULT '{}'::jsonb,
  error text
);

CREATE INDEX IF NOT EXISTS idx_crm_ai_run_workspace_key_started
  ON crm_ai_run (workspace_id, automation_key, started_at DESC);
