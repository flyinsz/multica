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
