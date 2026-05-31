CREATE TABLE IF NOT EXISTS crm_customer_alias (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    account_id uuid REFERENCES crm_account(id) ON DELETE CASCADE,
    contact_id uuid REFERENCES crm_contact(id) ON DELETE SET NULL,
    alias text NOT NULL,
    alias_normalized text NOT NULL,
    alias_type text NOT NULL DEFAULT 'manual',
    weight integer NOT NULL DEFAULT 50,
    source_type text NOT NULL DEFAULT 'manual',
    source_id uuid,
    confidence text NOT NULL DEFAULT 'medium',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT crm_customer_alias_account_or_contact CHECK (account_id IS NOT NULL OR contact_id IS NOT NULL)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_crm_customer_alias_unique
    ON crm_customer_alias (workspace_id, alias_normalized, COALESCE(account_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(contact_id, '00000000-0000-0000-0000-000000000000'::uuid), alias_type);

CREATE INDEX IF NOT EXISTS idx_crm_customer_alias_lookup
    ON crm_customer_alias (workspace_id, alias_normalized, weight DESC, confidence);

CREATE INDEX IF NOT EXISTS idx_crm_customer_alias_account
    ON crm_customer_alias (workspace_id, account_id);

CREATE OR REPLACE VIEW crm_interaction AS
SELECT
    m.id,
    m.workspace_id,
    m.account_id,
    m.contact_id,
    'email'::text AS channel,
    m.id AS source_id,
    m.thread_id,
    m.direction,
    COALESCE(m.received_at, m.sent_at, m.created_at) AS occurred_at,
    COALESCE(m.subject, '') AS subject,
    COALESCE(m.body_text, regexp_replace(COALESCE(m.body_html, ''), '<[^>]+>', ' ', 'g'), '') AS body_text,
    LEFT(regexp_replace(COALESCE(m.snippet, m.body_text, ''), '\s+', ' ', 'g'), 500) AS body_summary,
    ''::text AS language,
    ''::text AS sentiment,
    ''::text AS intent,
    jsonb_build_array(jsonb_build_object('channel','email','message_id',m.id,'thread_id',m.thread_id)) AS source_refs
FROM crm_email_message m;
