-- Ensure CRM pending-reply issues are owned by CRM-Assistant, assigned to Jarvis,
-- and linked to their CRM email thread so future replies can create child issues.
CREATE TEMP TABLE tmp_crm_pending_reply_issue_backfill ON COMMIT DROP AS
SELECT
  i.id AS issue_id,
  i.workspace_id,
  i.origin_id AS thread_id,
  ca.id AS crm_assistant_id,
  j.id AS jarvis_id
FROM issue i
JOIN crm_email_thread t ON t.workspace_id = i.workspace_id AND t.id = i.origin_id
JOIN agent ca ON ca.workspace_id = i.workspace_id AND lower(ca.name) = lower('CRM-Assistant') AND ca.archived_at IS NULL
JOIN agent j ON j.workspace_id = i.workspace_id AND lower(j.name) = lower('Jarvis') AND j.archived_at IS NULL
WHERE i.origin_type = 'crm_ai'
  AND i.title LIKE '回复邮件：%';

INSERT INTO crm_email_thread_issue_link (thread_id, issue_id)
SELECT thread_id, issue_id
FROM tmp_crm_pending_reply_issue_backfill
ON CONFLICT DO NOTHING;

CREATE TEMP TABLE tmp_crm_pending_reply_issue_parent ON COMMIT DROP AS
SELECT DISTINCT ON (l.thread_id)
  l.thread_id,
  i.id AS parent_issue_id
FROM crm_email_thread_issue_link l
JOIN issue i ON i.id = l.issue_id
WHERE i.origin_type = 'crm_ai'
  AND i.title LIKE '回复邮件：%'
ORDER BY l.thread_id, i.created_at ASC;

UPDATE issue i
SET creator_type = 'agent',
    creator_id = r.crm_assistant_id,
    assignee_type = 'agent',
    assignee_id = r.jarvis_id,
    status = CASE WHEN i.status NOT IN ('done','cancelled') THEN 'todo' ELSE i.status END,
    parent_issue_id = CASE WHEN i.id = p.parent_issue_id THEN NULL ELSE p.parent_issue_id END,
    updated_at = now()
FROM tmp_crm_pending_reply_issue_backfill r
JOIN tmp_crm_pending_reply_issue_parent p ON p.thread_id = r.thread_id
WHERE i.id = r.issue_id;

UPDATE issue i
SET origin_id = latest.message_id,
    updated_at = now()
FROM tmp_crm_pending_reply_issue_backfill r
JOIN LATERAL (
  SELECT m.id AS message_id
  FROM crm_email_message m
  WHERE m.workspace_id = r.workspace_id
    AND m.thread_id = r.thread_id
    AND m.direction = 'inbound'
    AND COALESCE(m.is_trashed,false) = false
  ORDER BY COALESCE(m.received_at, m.sent_at, m.created_at) DESC
  LIMIT 1
) latest ON true
WHERE i.id = r.issue_id
  AND i.origin_id = r.thread_id;

UPDATE crm_email_thread t
SET issue_id = p.parent_issue_id,
    updated_at = now()
FROM tmp_crm_pending_reply_issue_parent p
WHERE t.id = p.thread_id;
