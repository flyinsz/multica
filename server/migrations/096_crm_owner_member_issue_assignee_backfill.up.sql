-- Ensure CRM accounts have a member owner and CRM AI issues can be assigned to that owner.
WITH default_member AS (
  SELECT DISTINCT ON (workspace_id) workspace_id, id AS member_id
  FROM member
  ORDER BY workspace_id, CASE role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, created_at ASC
)
UPDATE crm_account a
SET owner_member_id = dm.member_id,
    updated_at = now()
FROM default_member dm
WHERE dm.workspace_id = a.workspace_id
  AND a.owner_member_id IS NULL;

UPDATE issue i
SET assignee_type = 'member',
    assignee_id = a.owner_member_id,
    updated_at = now()
FROM crm_email_thread t
JOIN crm_account a ON a.workspace_id=t.workspace_id AND a.id=t.account_id
WHERE i.workspace_id=t.workspace_id
  AND i.origin_type='crm_ai'
  AND i.origin_id=t.id
  AND i.status NOT IN ('done','cancelled')
  AND i.assignee_id IS NULL
  AND a.owner_member_id IS NOT NULL;
