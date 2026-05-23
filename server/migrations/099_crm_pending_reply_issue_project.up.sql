-- Link CRM pending-reply issues to customer-specific projects named CRM:<customer name>.
WITH reply_issues AS (
  SELECT i.id AS issue_id, i.workspace_id, t.account_id, a.name AS account_name
  FROM issue i
  JOIN crm_email_thread_issue_link l ON l.issue_id = i.id
  JOIN crm_email_thread t ON t.id = l.thread_id AND t.workspace_id = i.workspace_id
  JOIN crm_account a ON a.id = t.account_id AND a.workspace_id = i.workspace_id
  WHERE i.origin_type = 'crm_ai'
    AND COALESCE(a.name,'') <> ''
), inserted_projects AS (
  INSERT INTO project (workspace_id, title, description, icon, status, priority)
  SELECT DISTINCT r.workspace_id, 'CRM:' || r.account_name, 'CRM 客户专属项目：' || r.account_name, 'building-2', 'in_progress', 'medium'
  FROM reply_issues r
  WHERE NOT EXISTS (
    SELECT 1 FROM project p WHERE p.workspace_id = r.workspace_id AND lower(p.title) = lower('CRM:' || r.account_name)
  )
  ON CONFLICT DO NOTHING
  RETURNING id, workspace_id, title
), projects AS (
  SELECT p.id, p.workspace_id, p.title
  FROM project p
  WHERE EXISTS (
    SELECT 1 FROM reply_issues r WHERE r.workspace_id = p.workspace_id AND lower(p.title) = lower('CRM:' || r.account_name)
  )
), linked_issues AS (
  UPDATE issue i
  SET project_id = p.id,
      updated_at = now()
  FROM reply_issues r
  JOIN projects p ON p.workspace_id = r.workspace_id AND lower(p.title) = lower('CRM:' || r.account_name)
  WHERE i.id = r.issue_id
    AND i.workspace_id = r.workspace_id
  RETURNING i.id, i.workspace_id, p.id AS project_id, r.account_id, r.account_name
), linked_threads AS (
  UPDATE crm_email_thread t
  SET project_id = li.project_id,
      updated_at = now()
  FROM crm_email_thread_issue_link l
  JOIN linked_issues li ON li.id = l.issue_id
  WHERE t.id = l.thread_id AND t.workspace_id = li.workspace_id
  RETURNING t.id
)
INSERT INTO project_resource (project_id, workspace_id, resource_type, resource_ref, label, position)
SELECT DISTINCT li.project_id, li.workspace_id, 'crm_account', jsonb_build_object('account_id', li.account_id), li.account_name, 0
FROM linked_issues li
ON CONFLICT DO NOTHING;
