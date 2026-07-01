CREATE TABLE crm_whatsapp_account (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        UUID NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    provider            TEXT NOT NULL DEFAULT 'hermes',
    provider_account_id TEXT NOT NULL DEFAULT 'default',
    display_name        TEXT NOT NULL DEFAULT '',
    phone_number        TEXT NOT NULL DEFAULT '',
    status              TEXT NOT NULL DEFAULT 'unknown',
    config              JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_sync_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, provider, provider_account_id)
);

CREATE INDEX idx_crm_whatsapp_account_workspace
    ON crm_whatsapp_account(workspace_id);

CREATE TABLE crm_whatsapp_thread (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        UUID NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    whatsapp_account_id UUID NOT NULL REFERENCES crm_whatsapp_account(id) ON DELETE CASCADE,
    external_chat_id    TEXT NOT NULL,
    title               TEXT NOT NULL DEFAULT '',
    phone_number        TEXT NOT NULL DEFAULT '',
    account_id          UUID REFERENCES crm_account(id) ON DELETE SET NULL,
    contact_id          UUID REFERENCES crm_contact(id) ON DELETE SET NULL,
    last_message_at     TIMESTAMPTZ,
    unread_count        INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, whatsapp_account_id, external_chat_id)
);

CREATE INDEX idx_crm_whatsapp_thread_workspace_last
    ON crm_whatsapp_thread(workspace_id, last_message_at DESC NULLS LAST);
CREATE INDEX idx_crm_whatsapp_thread_account
    ON crm_whatsapp_thread(account_id) WHERE account_id IS NOT NULL;
CREATE INDEX idx_crm_whatsapp_thread_contact
    ON crm_whatsapp_thread(contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX idx_crm_whatsapp_thread_phone
    ON crm_whatsapp_thread(workspace_id, phone_number);

CREATE TABLE crm_whatsapp_message (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        UUID NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    thread_id           UUID NOT NULL REFERENCES crm_whatsapp_thread(id) ON DELETE CASCADE,
    external_message_id TEXT NOT NULL,
    direction           TEXT NOT NULL CHECK (direction IN ('inbound','outbound')),
    from_number         TEXT NOT NULL DEFAULT '',
    to_number           TEXT NOT NULL DEFAULT '',
    body_text           TEXT NOT NULL DEFAULT '',
    media               JSONB NOT NULL DEFAULT '[]'::jsonb,
    sent_at             TIMESTAMPTZ,
    received_at         TIMESTAMPTZ,
    raw                 JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, thread_id, external_message_id)
);

CREATE INDEX idx_crm_whatsapp_message_thread_time
    ON crm_whatsapp_message(thread_id, COALESCE(sent_at, received_at, created_at) DESC);
CREATE INDEX idx_crm_whatsapp_message_workspace_time
    ON crm_whatsapp_message(workspace_id, COALESCE(sent_at, received_at, created_at) DESC);
CREATE INDEX idx_crm_whatsapp_message_body_fts
    ON crm_whatsapp_message USING gin(to_tsvector('simple', body_text));

CREATE TABLE crm_whatsapp_issue_link (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    message_id   UUID NOT NULL REFERENCES crm_whatsapp_message(id) ON DELETE CASCADE,
    issue_id     UUID NOT NULL REFERENCES issue(id) ON DELETE CASCADE,
    reason       TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, message_id, issue_id)
);

CREATE INDEX idx_crm_whatsapp_issue_link_message
    ON crm_whatsapp_issue_link(message_id);
CREATE INDEX idx_crm_whatsapp_issue_link_issue
    ON crm_whatsapp_issue_link(issue_id);
