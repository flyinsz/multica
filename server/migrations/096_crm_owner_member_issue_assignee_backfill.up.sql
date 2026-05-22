-- Ensure CRM accounts have a user owner and CRM AI issues are assigned to that user's workspace member.
WITH default_member AS (
  SELECT DISTINCT ON (workspace_id) workspace_id, id AS member_id, user_id
  FROM member
  ORDER BY workspace_id, CASE role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, created_at ASC
)
UPDATE crm_account a
SET owner_member_id = dm.user_id,
    updated_at = now()
FROM default_member dm
WHERE dm.workspace_id = a.workspace_id
  AND a.owner_member_id IS NULL;

UPDATE issue i
SET assignee_type = 'member',
    assignee_id = COALESCE(owner_member.id, default_member.member_id),
    updated_at = now()
FROM crm_email_thread t
JOIN crm_account a ON a.workspace_id=t.workspace_id AND a.id=t.account_id
LEFT JOIN member owner_member ON owner_member.workspace_id=a.workspace_id AND owner_member.user_id=a.owner_member_id
LEFT JOIN LATERAL (
  SELECT id AS member_id
  FROM member m
  WHERE m.workspace_id=a.workspace_id
  ORDER BY CASE role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, created_at ASC
  LIMIT 1
) default_member ON true
WHERE i.workspace_id=t.workspace_id
  AND i.origin_type='crm_ai'
  AND i.origin_id=t.id
  AND i.status NOT IN ('done','cancelled')
  AND i.assignee_id IS NULL
  AND COALESCE(owner_member.id, default_member.member_id) IS NOT NULL;
