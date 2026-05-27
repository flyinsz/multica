package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CRMAIAutoScheduler struct {
	db       *pgxpool.Pool
	interval time.Duration
}

func NewCRMAIAutoScheduler(db *pgxpool.Pool, interval time.Duration) *CRMAIAutoScheduler {
	if interval <= 0 {
		interval = time.Minute
	}
	return &CRMAIAutoScheduler{db: db, interval: interval}
}

func (s *CRMAIAutoScheduler) Run(ctx context.Context) {
	if s == nil || s.db == nil {
		return
	}
	slog.Info("CRM AI automation scheduler started", "interval", s.interval.String())
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			slog.Info("CRM AI automation scheduler stopped")
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

type crmAISettingRow struct {
	WorkspaceID     pgtype.UUID
	AutomationKey   string
	IntervalMinutes int
	MaxItemsPerRun  int
	Config          json.RawMessage
}

type crmAIIssueActorConfig struct {
	CreatorType      string `json:"issue_creator_type"`
	CreatorID        string `json:"issue_creator_id"`
	TodoAssigneeType string `json:"issue_todo_assignee_type"`
	TodoAssigneeID   string `json:"issue_todo_assignee_id"`
}

func (s *CRMAIAutoScheduler) runOnce(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()

	if err := ensureDefaultCRMAISettings(ctx, s.db); err != nil {
		slog.Warn("CRM AI defaults ensure failed", "error", err)
		return
	}

	rows, err := s.db.Query(ctx, `
		SELECT workspace_id, automation_key, interval_minutes, max_items_per_run, config
		FROM crm_ai_setting
		WHERE enabled=true
		  AND COALESCE(last_checked_at, 'epoch'::timestamptz) <= now() - (interval_minutes || ' minutes')::interval
		ORDER BY COALESCE(last_checked_at, 'epoch'::timestamptz) ASC
		LIMIT 20`)
	if err != nil {
		slog.Warn("CRM AI due settings query failed", "error", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var item crmAISettingRow
		if err := rows.Scan(&item.WorkspaceID, &item.AutomationKey, &item.IntervalMinutes, &item.MaxItemsPerRun, &item.Config); err != nil {
			slog.Warn("CRM AI setting scan failed", "error", err)
			continue
		}
		s.runSetting(parent, item)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("CRM AI settings rows failed", "error", err)
	}
}

func ensureDefaultCRMAISettings(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		INSERT INTO crm_ai_setting (workspace_id, automation_key, enabled, interval_minutes, max_items_per_run, config)
		SELECT w.id, v.key, v.enabled, v.interval_minutes, v.max_items, v.config::jsonb
		FROM workspace w
		CROSS JOIN (VALUES
			('email_pending_reply', true, 5, 10, '{}'::text),
			('due_followup', true, 60, 20, '{}'::text),
			('profile_new_activity_refresh', true, 5, 20, '{"trigger":"new_activity"}'::text),
			('profile_daily_refresh', false, 1440, 100, '{"time":"03:00"}'::text)
		) AS v(key, enabled, interval_minutes, max_items, config)
		ON CONFLICT (workspace_id, automation_key) DO NOTHING`)
	return err
}

func (s *CRMAIAutoScheduler) runSetting(parent context.Context, item crmAISettingRow) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	defer cancel()

	h := &Handler{DB: s.db}
	result := map[string]any{"automation_key": item.AutomationKey, "checked_at": time.Now().UTC().Format(time.RFC3339)}
	approvedSent, notifiedSent := h.runCRMApprovedDraftStateAutomation(ctx, item.WorkspaceID, item.MaxItemsPerRun)
	if approvedSent > 0 {
		result["approved_done_drafts_sent"] = approvedSent
	}
	if notifiedSent > 0 {
		result["sent_draft_done_notifications"] = notifiedSent
	}
	var err error
	switch item.AutomationKey {
	case "email_pending_reply":
		result, err = h.runCRMPendingReplyAutomation(ctx, item.WorkspaceID, item.MaxItemsPerRun, item.Config)
	case "due_followup":
		result, err = h.runCRMDueFollowupAutomation(ctx, item.WorkspaceID, item.MaxItemsPerRun, item.Config)
	case "profile_new_activity_refresh":
		result, err = h.runCRMRecentActivityProfileRefresh(ctx, item.WorkspaceID, item.MaxItemsPerRun)
	case "profile_daily_refresh":
		if !crmAIDailyWindowDue(item.Config) {
			result["skipped"] = "outside_configured_time"
			break
		}
		result, err = h.runCRMFullProfileRefresh(ctx, item.WorkspaceID, item.MaxItemsPerRun)
	default:
		err = fmt.Errorf("unsupported CRM AI automation %s", item.AutomationKey)
	}
	if err != nil {
		result["error"] = err.Error()
		slog.Warn("CRM AI automation run failed", "key", item.AutomationKey, "workspace_id", uuidToString(item.WorkspaceID), "error", err)
	}
	payload, _ := json.Marshal(result)
	_, _ = s.db.Exec(context.Background(), `UPDATE crm_ai_setting SET last_checked_at=now(), last_result=$3, updated_at=now() WHERE workspace_id=$1 AND automation_key=$2`, item.WorkspaceID, item.AutomationKey, payload)
}

func crmAIDailyWindowDue(config json.RawMessage) bool {
	var cfg struct {
		Time string `json:"time"`
	}
	_ = json.Unmarshal(config, &cfg)
	if cfg.Time == "" {
		cfg.Time = "03:00"
	}
	now := time.Now()
	want, err := time.Parse("15:04", cfg.Time)
	if err != nil {
		return true
	}
	return now.Hour() == want.Hour() && now.Minute() < 10
}

func shouldAutoRefreshCRMAccountProfile(ctx context.Context, db dbExecutor, workspaceID pgtype.UUID) bool {
	var enabled bool
	if err := db.QueryRow(ctx, `SELECT enabled FROM crm_ai_setting WHERE workspace_id=$1 AND automation_key='profile_new_activity_refresh'`, workspaceID).Scan(&enabled); err != nil {
		return true
	}
	return enabled
}

func (h *Handler) runCRMRecentActivityProfileRefresh(ctx context.Context, workspaceID pgtype.UUID, limit int) (map[string]any, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := h.DB.Query(ctx, `
		SELECT DISTINCT t.account_id
		FROM crm_email_thread t
		WHERE t.workspace_id=$1 AND t.account_id IS NOT NULL AND t.updated_at > now() - interval '24 hours'
		ORDER BY t.account_id
		LIMIT $2`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return h.refreshProfilesFromRows(ctx, workspaceID, rows, "profile_new_activity_refresh")
}

func (h *Handler) runCRMFullProfileRefresh(ctx context.Context, workspaceID pgtype.UUID, limit int) (map[string]any, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := h.DB.Query(ctx, `
		SELECT id FROM crm_account WHERE workspace_id=$1 ORDER BY updated_at DESC LIMIT $2`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return h.refreshProfilesFromRows(ctx, workspaceID, rows, "profile_daily_refresh")
}

func (h *Handler) refreshProfilesFromRows(ctx context.Context, workspaceID pgtype.UUID, rows pgx.Rows, key string) (map[string]any, error) {
	refreshed := 0
	failed := 0
	for rows.Next() {
		var accountID pgtype.UUID
		if err := rows.Scan(&accountID); err != nil {
			failed++
			continue
		}
		if _, err := h.regenerateCRMAccountProfile(ctx, workspaceID, accountID); err != nil {
			failed++
			slog.Warn("CRM account profile scheduled refresh failed", "key", key, "account_id", uuidToString(accountID), "error", err)
			continue
		}
		refreshed++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"automation_key": key, "refreshed": refreshed, "failed": failed}, nil
}

func (h *Handler) runCRMPendingReplyAutomation(ctx context.Context, workspaceID pgtype.UUID, limit int, config json.RawMessage) (map[string]any, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	issueActors, err := h.crmAIIssueActors(ctx, workspaceID, config)
	if err != nil {
		return nil, err
	}
	rows, err := h.DB.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (m.thread_id)
				m.thread_id, m.id AS message_id, m.direction, m.subject, m.from_email, m.received_at, m.sent_at, m.created_at, m.is_read, m.is_trashed, m.folder
			FROM crm_email_message m
			WHERE m.workspace_id=$1 AND COALESCE(m.is_trashed,false)=false
			ORDER BY m.thread_id, COALESCE(m.received_at, m.sent_at, m.created_at) DESC
		)
		SELECT t.id,
		       COALESCE(latest.message_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       t.account_id,
		       t.contact_id,
		       COALESCE(a.name, ''),
		       (SELECT m.id FROM member m WHERE m.workspace_id=$1 AND m.user_id=a.owner_member_id LIMIT 1),
		       COALESCE(latest.subject, t.subject, ''),
		       COALESCE(latest.received_at, latest.sent_at, latest.created_at, t.last_message_at, t.updated_at)
		FROM crm_email_thread t
		JOIN latest ON latest.thread_id=t.id
		LEFT JOIN crm_account a ON a.id=t.account_id AND a.workspace_id=t.workspace_id
		WHERE t.workspace_id=$1
		  AND COALESCE(t.is_trashed,false)=false
		  AND COALESCE(latest.is_trashed,false)=false
		  AND COALESCE(t.status,'open') <> 'archived'
		  AND latest.direction='inbound'
		  AND lower(COALESCE(NULLIF(latest.folder,''), 'INBOX')) NOT LIKE ALL(ARRAY['%spam%', '%junk%', '%trash%', '%deleted%', '%archive%'])
		  AND lower(COALESCE(latest.from_email,'')) NOT LIKE ALL(ARRAY['no-reply@%', 'noreply@%', 'postmaster@%', 'mailer-daemon@%'])
		  AND latest.message_id IS NOT NULL
		  AND NOT EXISTS (
			SELECT 1 FROM issue i
			WHERE i.workspace_id=$1 AND i.origin_type='crm_ai' AND i.origin_id=latest.message_id
		  )
		ORDER BY COALESCE(latest.received_at, latest.sent_at, latest.created_at, t.last_message_at, t.updated_at) DESC
		LIMIT $2`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	created := 0
	createdIssues := []map[string]string{}
	candidates := 0
	for rows.Next() {
		var threadID, messageID, accountID, contactID, ownerMemberID pgtype.UUID
		var subject, accountName string
		var lastAt pgtype.Timestamptz
		if err := rows.Scan(&threadID, &messageID, &accountID, &contactID, &accountName, &ownerMemberID, &subject, &lastAt); err != nil {
			continue
		}
		candidates++
		title := stringsTrimForCRM(fmt.Sprintf("回复邮件：%s", subject), 120)
		messageLink := h.crmWorkspaceAppURL(ctx, workspaceID, "/crm/emails?message="+uuidToString(messageID)+"&thread="+uuidToString(threadID))
		reviewerLine := h.crmDraftReviewerLine(ctx, workspaceID, ownerMemberID)
		body := h.buildCRMPendingReplyIssueBody(threadID, messageID, accountID, contactID, accountName, subject, messageLink, timestampToString(lastAt), reviewerLine)
		parentIssueID, err := h.findCRMEmailThreadParentIssue(ctx, workspaceID, threadID)
		if err != nil {
			slog.Warn("CRM pending reply parent issue lookup failed", "workspace_id", uuidToString(workspaceID), "thread_id", uuidToString(threadID), "error", err)
		}
		projectID, err := h.ensureCRMAccountProject(ctx, workspaceID, accountID, accountName)
		if err != nil {
			slog.Warn("CRM pending reply project lookup failed", "workspace_id", uuidToString(workspaceID), "account_id", uuidToString(accountID), "error", err)
		}
		if parentIssueID.Valid {
			created++
			createdIssues = append(createdIssues, map[string]string{"id": uuidToString(parentIssueID), "title": title, "thread_id": uuidToString(threadID), "message_id": uuidToString(messageID), "kind": "merged_comment"})
			comment := h.buildCRMPendingReplyMergeComment(threadID, messageID, accountID, contactID, subject, messageLink, timestampToString(lastAt), reviewerLine)
			if err := h.addCRMInternalIssueComment(ctx, workspaceID, parentIssueID, comment); err != nil {
				slog.Warn("CRM pending reply merge comment failed", "workspace_id", uuidToString(workspaceID), "issue_id", uuidToString(parentIssueID), "thread_id", uuidToString(threadID), "error", err)
			}
			continue
		}
		issueID, err := h.createCRMEmailPendingReplyIssue(ctx, workspaceID, title, body, threadID, messageID, parentIssueID, projectID, issueActors)
		if err != nil {
			slog.Warn("CRM pending reply issue creation failed", "workspace_id", uuidToString(workspaceID), "thread_id", uuidToString(threadID), "error", err)
			continue
		}
		if issueID.Valid {
			created++
			createdIssues = append(createdIssues, map[string]string{"id": uuidToString(issueID), "title": title, "thread_id": uuidToString(threadID), "message_id": uuidToString(messageID), "kind": "created"})
			if accountID.Valid || contactID.Valid {
				_, _ = h.DB.Exec(ctx, `UPDATE crm_email_thread SET account_id=COALESCE(account_id,$3), contact_id=COALESCE(contact_id,$4), updated_at=now() WHERE workspace_id=$1 AND id=$2`, workspaceID, threadID, accountID, contactID)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"automation_key": "email_pending_reply", "candidates": candidates, "created": created, "created_issues": createdIssues}, nil
}

func (h *Handler) runCRMDueFollowupAutomation(ctx context.Context, workspaceID pgtype.UUID, limit int, config json.RawMessage) (map[string]any, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	issueActors, err := h.crmAIIssueActors(ctx, workspaceID, config)
	if err != nil {
		return nil, err
	}
	rows, err := h.DB.Query(ctx, `
		SELECT id, name, next_follow_up_at
		FROM crm_account
		WHERE workspace_id=$1 AND next_follow_up_at IS NOT NULL AND next_follow_up_at <= now() AND status <> 'inactive'
		ORDER BY next_follow_up_at ASC
		LIMIT $2`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	created := 0
	createdIssues := []map[string]string{}
	for rows.Next() {
		var accountID pgtype.UUID
		var name string
		var due pgtype.Timestamptz
		if err := rows.Scan(&accountID, &name, &due); err != nil {
			continue
		}
		title := stringsTrimForCRM(fmt.Sprintf("跟进客户：%s", name), 120)
		var ownerMemberID pgtype.UUID
		_ = h.DB.QueryRow(ctx, `SELECT (SELECT m.id FROM member m WHERE m.workspace_id=$1 AND m.user_id=a.owner_member_id LIMIT 1) FROM crm_account a WHERE a.workspace_id=$1 AND a.id=$2`, workspaceID, accountID).Scan(&ownerMemberID)
		reviewerLine := h.crmDraftReviewerLine(ctx, workspaceID, ownerMemberID)
		body := fmt.Sprintf("CRM 到期客户跟进自动创建。\n客户：%s\n客户ID：%s\n到期时间：%s\n\n处理要求：\n1. Issue 初始负责人使用 CRM AI 配置中的 todo 阶段负责人，由其生成客户跟进邮件草稿。\n2. 草稿生成完成后，必须把 Issue 从 todo 转入审核阶段，并把负责人改为邮件草稿审核人。\n3. %s\n4. 审核通过后才能发送邮件。", name, uuidToString(accountID), timestampToString(due), reviewerLine)
		issueID, err := h.createCRMInternalIssue(ctx, workspaceID, title, body, uuidToString(accountID), issueActors)
		if err == nil && issueID.Valid {
			created++
			createdIssues = append(createdIssues, map[string]string{"id": uuidToString(issueID), "title": title, "account_id": uuidToString(accountID), "kind": "created"})
			_ = h.createCRMAIFollowupDraft(ctx, workspaceID, issueID, accountID, name, timestampToString(due))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"automation_key": "due_followup", "created": created, "created_issues": createdIssues}, nil
}

func stringsTrimForCRM(value string, limit int) string {
	if len([]rune(value)) <= limit {
		return value
	}
	r := []rune(value)
	return string(r[:limit])
}

func (h *Handler) crmDraftReviewerLine(ctx context.Context, workspaceID, reviewerMemberID pgtype.UUID) string {
	if !reviewerMemberID.Valid {
		ownerMemberID, ownerLabel, err := h.crmWorkspaceOwnerMember(ctx, workspaceID)
		if err != nil || !ownerMemberID.Valid {
			return "邮件草稿审核人：客户没有负责人，请交由工作区 owner/admin 审核。"
		}
		return fmt.Sprintf("邮件草稿审核人：客户没有负责人，请交由工作区 owner %s（member:%s）审核。", ownerLabel, uuidToString(ownerMemberID))
	}
	var name, email string
	if err := h.DB.QueryRow(ctx, `SELECT COALESCE(u.name,''), COALESCE(u.email,'') FROM member m JOIN "user" u ON u.id=m.user_id WHERE m.workspace_id=$1 AND m.id=$2 LIMIT 1`, workspaceID, reviewerMemberID).Scan(&name, &email); err != nil {
		return fmt.Sprintf("邮件草稿审核人：客户所有人 member:%s。", uuidToString(reviewerMemberID))
	}
	label := strings.TrimSpace(name)
	if label == "" {
		label = strings.TrimSpace(email)
	}
	if label == "" {
		label = uuidToString(reviewerMemberID)
	}
	return fmt.Sprintf("邮件草稿审核人：客户所有人 %s（member:%s）。", label, uuidToString(reviewerMemberID))
}

func (h *Handler) crmWorkspaceOwnerMember(ctx context.Context, workspaceID pgtype.UUID) (pgtype.UUID, string, error) {
	var memberID pgtype.UUID
	var name, email string
	err := h.DB.QueryRow(ctx, `SELECT m.id, COALESCE(u.name,''), COALESCE(u.email,'') FROM member m JOIN "user" u ON u.id=m.user_id WHERE m.workspace_id=$1 ORDER BY CASE m.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, m.created_at ASC LIMIT 1`, workspaceID).Scan(&memberID, &name, &email)
	if err != nil {
		return memberID, "", err
	}
	label := strings.TrimSpace(name)
	if label == "" {
		label = strings.TrimSpace(email)
	}
	if label == "" {
		label = uuidToString(memberID)
	}
	return memberID, label, nil
}

func (h *Handler) buildCRMPendingReplyIssueBody(threadID, messageID, accountID, contactID pgtype.UUID, accountName, subject, messageLink, latestAt, reviewerLine string) string {
	if strings.TrimSpace(accountName) == "" {
		accountName = "未绑定客户"
	}
	return fmt.Sprintf("CRM 邮件待回复巡检自动创建。\n\n客户：%s\n邮件主题：%s\n邮件线程：%s\n最新邮件ID：%s\n客户ID：%s\n联系人ID：%s\n原邮件：%s\n最新入站时间：%s\n\n处理要求：\n1. Issue 初始负责人使用 CRM AI 配置中的 todo 阶段负责人，由其生成待审核邮件草稿。\n2. 生成草稿前，必须通过 CRM MCP 查询客户 profile、当前邮件线程、最新原邮件和历史往来；调用 MCP 时 UUID 参数必须使用纯 UUID 字符串，不要包含花括号；不要要求巡检程序把这些内容展开写进 Issue。\n3. 草稿必须使用中文撰写，邮件正文先写正式回复，再按照系统中回复邮件的逻辑在正文下方引用原邮件内容；不要在开头引用或概括原邮件关键问题。\n4. 草稿应结合客户历史往来、客户资料、当前邮件线程和最新入站邮件内容。\n5. 事实、报价、交期、质量承诺、售后承诺、附件内容必须人工审核后才能发送。\n6. 草稿生成完成后，必须把 Issue 从 todo 转入审核阶段，并把负责人改为邮件草稿审核人。\n7. %s\n\n流转说明：todo 阶段由配置负责人接手；进入审核阶段时显性转交给上述客户所有人。", accountName, subject, uuidToString(threadID), uuidToString(messageID), uuidToString(accountID), uuidToString(contactID), messageLink, latestAt, reviewerLine)
}

func (h *Handler) buildCRMPendingReplyMergeComment(threadID, messageID, accountID, contactID pgtype.UUID, subject, messageLink, latestAt, reviewerLine string) string {
	return fmt.Sprintf("同一邮件线程收到新的入站邮件，请由 Multica Issue 流程把新内容合并进当前未处理的回复草稿。\n\n邮件主题：%s\n邮件线程：%s\n最新邮件ID：%s\n客户ID：%s\n联系人ID：%s\n原邮件：%s\n最新入站时间：%s\n\n处理要求：通过 CRM MCP 查询客户 profile、当前邮件线程、最新原邮件和历史往来；调用 MCP 时 UUID 参数必须使用纯 UUID 字符串，不要包含花括号；不要把历史往来/profile 摘要展开写进 Issue。无客户/联系人绑定、客户资料为空、历史往来为空，不是“不回复”的理由；必须先按潜在客户/真实业务咨询判断。若像潜在客户，应基于当前邮件谨慎回复，不虚构上下文；只有明显 spam/no-reply/系统通知/营销群发才不建议回复。草稿必须中文撰写，邮件正文先写正式回复，再按照系统中回复邮件的逻辑在正文下方引用原邮件内容；不要在开头引用或概括原邮件关键问题。%s", subject, uuidToString(threadID), uuidToString(messageID), uuidToString(accountID), uuidToString(contactID), messageLink, latestAt, reviewerLine)
}

func (h *Handler) createCRMAIFollowupDraft(ctx context.Context, workspaceID, issueID, accountID pgtype.UUID, accountName, dueText string) error {
	var mailboxID, contactID pgtype.UUID
	var email, contactName string
	if err := h.DB.QueryRow(ctx, `SELECT id FROM crm_imap_setting WHERE workspace_id=$1 AND sync_enabled=true ORDER BY updated_at DESC LIMIT 1`, workspaceID).Scan(&mailboxID); err != nil {
		return err
	}
	if err := h.DB.QueryRow(ctx, `SELECT id, COALESCE(name,''), COALESCE(email,'') FROM crm_contact WHERE workspace_id=$1 AND account_id=$2 AND COALESCE(email,'')<>'' ORDER BY is_primary DESC, updated_at DESC LIMIT 1`, workspaceID, accountID).Scan(&contactID, &contactName, &email); err != nil {
		return h.addCRMInternalIssueComment(ctx, workspaceID, issueID, "已创建到期跟进 Issue，但未生成邮件草稿：客户没有可用联系人邮箱。请补充联系人后手动创建草稿。")
	}
	reason := fmt.Sprintf("草稿思路：这是到期跟进自动化生成的跟进草稿。依据客户 next_follow_up_at 到期时间 %s、客户名称和主联系人生成，目标是礼貌重启沟通并确认下一步需求。风险：发送前请人工确认报价、交期、项目上下文和联系人是否正确。", dueText)
	body := fmt.Sprintf("Hi %s,\n\nI hope you are doing well. I wanted to follow up with you regarding %s and check whether there are any updates or new requirements we can support.\n\nBest regards", contactName, accountName)
	var draftID pgtype.UUID
	if err := h.DB.QueryRow(ctx, `INSERT INTO crm_email_draft (workspace_id, mailbox_id, account_id, contact_id, issue_id, to_emails, subject, body_text, status, ai_generated, approval_reason) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending_approval',true,$9) RETURNING id`, workspaceID, mailboxID, accountID, contactID, issueID, []string{email}, "Follow up: "+accountName, body, cleanOptionalText(&reason)).Scan(&draftID); err != nil {
		return err
	}
	draftURL := h.crmWorkspaceAppURL(ctx, workspaceID, "/crm/emails?draft="+uuidToString(draftID))
	return h.addCRMInternalIssueComment(ctx, workspaceID, issueID, fmt.Sprintf("已生成待审核跟进邮件草稿。\n\n草稿链接：[%s](%s)\n\n%s", draftURL, draftURL, reason))
}

func (h *Handler) findCRMEmailThreadParentIssue(ctx context.Context, workspaceID, threadID pgtype.UUID) (pgtype.UUID, error) {
	var issueID pgtype.UUID
	err := h.DB.QueryRow(ctx, `
		SELECT i.id
		FROM crm_email_thread_issue_link l
		JOIN issue i ON i.id=l.issue_id AND i.workspace_id=$1
		WHERE l.thread_id=$2 AND i.origin_type='crm_ai' AND i.parent_issue_id IS NULL AND i.status <> 'cancelled'
		ORDER BY i.created_at ASC
		LIMIT 1`, workspaceID, threadID).Scan(&issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, nil
	}
	return issueID, err
}

func (h *Handler) crmAgentByName(ctx context.Context, workspaceID pgtype.UUID, name string) (pgtype.UUID, error) {
	var agentID pgtype.UUID
	err := h.DB.QueryRow(ctx, `SELECT id FROM agent WHERE workspace_id=$1 AND lower(name)=lower($2) AND archived_at IS NULL LIMIT 1`, workspaceID, name).Scan(&agentID)
	return agentID, err
}

func (h *Handler) ensureCRMAccountProject(ctx context.Context, workspaceID, accountID pgtype.UUID, accountName string) (pgtype.UUID, error) {
	var projectID pgtype.UUID
	if !accountID.Valid || strings.TrimSpace(accountName) == "" {
		return projectID, nil
	}
	projectTitle := "CRM:" + strings.TrimSpace(accountName)
	err := h.DB.QueryRow(ctx, `
		WITH existing AS (
			SELECT id FROM project WHERE workspace_id=$1 AND lower(title)=lower($2) LIMIT 1
		), inserted AS (
			INSERT INTO project (workspace_id, title, description, icon, status, priority)
			SELECT $1, $2, $3, 'building-2', 'in_progress', 'medium'
			WHERE NOT EXISTS (SELECT 1 FROM existing)
			ON CONFLICT DO NOTHING
			RETURNING id
		), selected AS (
			SELECT id FROM inserted UNION ALL SELECT id FROM existing LIMIT 1
		), resource_link AS (
			INSERT INTO project_resource (project_id, workspace_id, resource_type, resource_ref, label, position)
			SELECT id, $1, 'crm_account', jsonb_build_object('account_id', $4::uuid), $5, COALESCE((SELECT MAX(position)+1 FROM project_resource WHERE project_id=id), 0)
			FROM selected
			ON CONFLICT DO NOTHING
		)
		SELECT id FROM selected`, workspaceID, projectTitle, "CRM 客户专属项目："+accountName, accountID, accountName).Scan(&projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, nil
	}
	return projectID, err
}

func (h *Handler) createCRMEmailPendingReplyIssue(ctx context.Context, workspaceID pgtype.UUID, title, body string, threadID, messageID, parentIssueID, projectID pgtype.UUID, actors crmAIIssueActorConfig) (pgtype.UUID, error) {
	var issueID pgtype.UUID
	creatorType, creatorID := actors.CreatorType, mustParsePgUUID(actors.CreatorID)
	assigneeType, assigneeID := actors.TodoAssigneeType, mustParsePgUUID(actors.TodoAssigneeID)
	err := h.DB.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO issue (workspace_id, title, description, status, priority, assignee_type, assignee_id, creator_type, creator_id, parent_issue_id, origin_type, origin_id, number, project_id)
			SELECT $1, $2, $3, 'todo', 'medium', $6, $7, $8, $9, $10, 'crm_ai', $4, COALESCE((SELECT MAX(number) FROM issue WHERE workspace_id=$1), 0) + 1, $12
			WHERE NOT EXISTS (
				SELECT 1 FROM issue WHERE workspace_id=$1 AND origin_type='crm_ai' AND origin_id=$4
			)
			RETURNING id
		), linked AS (
			INSERT INTO crm_email_thread_issue_link (thread_id, issue_id)
			SELECT $11, id FROM inserted
			ON CONFLICT DO NOTHING
		), thread_update AS (
			UPDATE crm_email_thread
			SET issue_id=COALESCE(NULLIF($10, '00000000-0000-0000-0000-000000000000'::uuid), (SELECT id FROM inserted)), updated_at=now()
			WHERE workspace_id=$1 AND id=$11 AND EXISTS (SELECT 1 FROM inserted)
		)
		SELECT id FROM inserted`, workspaceID, title, body, messageID, creatorType, assigneeType, assigneeID, creatorType, creatorID, parentIssueID, threadID, projectID).Scan(&issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, nil
	}
	return issueID, err
}

func (h *Handler) crmAIIssueActors(ctx context.Context, workspaceID pgtype.UUID, config json.RawMessage) (crmAIIssueActorConfig, error) {
	var cfg crmAIIssueActorConfig
	_ = json.Unmarshal(config, &cfg)
	if cfg.CreatorType == "" || cfg.CreatorID == "" {
		actorType, actorID, err := h.crmIssueActorByAgentNameOrFallback(ctx, workspaceID, "CRM-Assistant")
		if err != nil {
			return cfg, err
		}
		cfg.CreatorType = actorType
		cfg.CreatorID = uuidToString(actorID)
	}
	if cfg.TodoAssigneeType == "" || cfg.TodoAssigneeID == "" {
		actorType, actorID, err := h.crmIssueActorByAgentNameOrFallback(ctx, workspaceID, "Jarvis")
		if err != nil {
			return cfg, err
		}
		cfg.TodoAssigneeType = actorType
		cfg.TodoAssigneeID = uuidToString(actorID)
	}
	if err := h.validateCRMIssueActor(ctx, workspaceID, cfg.CreatorType, cfg.CreatorID); err != nil {
		return cfg, fmt.Errorf("invalid CRM AI issue creator: %w", err)
	}
	if err := h.validateCRMIssueActor(ctx, workspaceID, cfg.TodoAssigneeType, cfg.TodoAssigneeID); err != nil {
		return cfg, fmt.Errorf("invalid CRM AI issue todo assignee: %w", err)
	}
	return cfg, nil
}

func (h *Handler) validateCRMIssueActor(ctx context.Context, workspaceID pgtype.UUID, actorType, actorID string) error {
	parsed, err := parseUUIDStringToPgtype(actorID)
	if err != nil {
		return err
	}
	switch actorType {
	case "member":
		return h.DB.QueryRow(ctx, `SELECT id FROM member WHERE workspace_id=$1 AND id=$2`, workspaceID, parsed).Scan(&parsed)
	case "agent":
		return h.DB.QueryRow(ctx, `SELECT id FROM agent WHERE workspace_id=$1 AND id=$2 AND archived_at IS NULL`, workspaceID, parsed).Scan(&parsed)
	default:
		return fmt.Errorf("unsupported actor type %q", actorType)
	}
}

func mustParsePgUUID(value string) pgtype.UUID {
	parsed, _ := parseUUIDStringToPgtype(value)
	return parsed
}

func (h *Handler) crmIssueActorByAgentNameOrFallback(ctx context.Context, workspaceID pgtype.UUID, name string) (string, pgtype.UUID, error) {
	agentID, err := h.crmAgentByName(ctx, workspaceID, name)
	if err == nil && agentID.Valid {
		return "agent", agentID, nil
	}
	return h.crmIssueCommentAuthor(ctx, workspaceID)
}

func (h *Handler) createCRMInternalIssue(ctx context.Context, workspaceID pgtype.UUID, title, body, originID string, actors crmAIIssueActorConfig) (pgtype.UUID, error) {
	var issueID pgtype.UUID
	originUUID, err := parseUUIDStringToPgtype(originID)
	if err != nil {
		return issueID, err
	}
	creatorType, creatorID := actors.CreatorType, mustParsePgUUID(actors.CreatorID)
	assigneeType, assigneeID := actors.TodoAssigneeType, mustParsePgUUID(actors.TodoAssigneeID)
	err = h.DB.QueryRow(ctx, `
		INSERT INTO issue (workspace_id, title, description, status, priority, assignee_type, assignee_id, creator_type, creator_id, origin_type, origin_id, number)
		SELECT $1, $2, $3, 'todo', 'medium', NULLIF($7::text,''), $8, $5::text, $6, 'crm_ai', $4, COALESCE((SELECT MAX(number) FROM issue WHERE workspace_id=$1), 0) + 1
		WHERE NOT EXISTS (
			SELECT 1 FROM issue WHERE workspace_id=$1 AND origin_type='crm_ai' AND origin_id=$4 AND status NOT IN ('done','cancelled')
		)
		RETURNING id`, workspaceID, title, body, originUUID, creatorType, creatorID, assigneeType, assigneeID).Scan(&issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, nil
	}
	return issueID, err
}

func (h *Handler) crmIssueCommentAuthor(ctx context.Context, workspaceID pgtype.UUID) (string, pgtype.UUID, error) {
	var memberID pgtype.UUID
	if err := h.DB.QueryRow(ctx, `SELECT id FROM member WHERE workspace_id=$1 ORDER BY CASE role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, created_at ASC LIMIT 1`, workspaceID).Scan(&memberID); err == nil && memberID.Valid {
		return "member", memberID, nil
	}
	var agentID pgtype.UUID
	if err := h.DB.QueryRow(ctx, `SELECT id FROM agent WHERE workspace_id=$1 ORDER BY created_at ASC LIMIT 1`, workspaceID).Scan(&agentID); err != nil {
		return "", pgtype.UUID{}, err
	}
	return "agent", agentID, nil
}

func (h *Handler) addCRMInternalIssueComment(ctx context.Context, workspaceID, issueID pgtype.UUID, content string) error {
	authorType, authorID, err := h.crmIssueCommentAuthor(ctx, workspaceID)
	if err != nil {
		return err
	}
	_, err = h.DB.Exec(ctx, `INSERT INTO comment (issue_id, workspace_id, author_type, author_id, content, type) VALUES ($1,$2,$3,$4,$5,'system')`, issueID, workspaceID, authorType, authorID, content)
	return err
}

func (h *Handler) crmWorkspaceAppURL(ctx context.Context, workspaceID pgtype.UUID, path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	var slug string
	if err := h.DB.QueryRow(ctx, `SELECT slug FROM workspace WHERE id=$1`, workspaceID).Scan(&slug); err == nil && strings.TrimSpace(slug) != "" {
		return h.crmAppURL("/" + strings.Trim(strings.TrimSpace(slug), "/") + path)
	}
	return h.crmAppURL(path)
}

func (h *Handler) crmAppURL(path string) string {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("MULTICA_APP_URL")), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN")), "/")
	}
	if base == "" {
		base = "http://localhost:3000"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func (h *Handler) runCRMApprovedDraftStateAutomation(ctx context.Context, workspaceID pgtype.UUID, limit int) (int, int) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := h.DB.Query(ctx, `
		SELECT d.issue_id
		FROM crm_email_draft d
		JOIN issue i ON i.id=d.issue_id AND i.workspace_id=d.workspace_id
		WHERE d.workspace_id=$1
		  AND d.issue_id IS NOT NULL
		  AND d.status IN ('pending_approval','draft','failed')
		  AND d.sent_at IS NULL
		  AND i.status='done'
		ORDER BY d.updated_at ASC
		LIMIT $2`, workspaceID, limit)
	if err != nil {
		slog.Warn("CRM approved draft query failed", "workspace_id", uuidToString(workspaceID), "error", err)
		return 0, 0
	}
	defer rows.Close()
	approvedSent := 0
	for rows.Next() {
		var issueID pgtype.UUID
		if err := rows.Scan(&issueID); err != nil || !issueID.Valid {
			continue
		}
		if _, err := h.sendFirstPendingCRMEmailDraftForIssue(ctx, workspaceID, issueID); err != nil {
			_ = h.addCRMInternalIssueComment(ctx, workspaceID, issueID, "Issue 已标记为 done，但自动发送绑定草稿失败："+err.Error())
			continue
		}
		approvedSent++
	}

	externalConfirmed, externalSuspected, draftMissing := h.detectExternalSentCRMEmailDrafts(ctx, workspaceID, limit)
	notifiedSent := h.commentManuallySentDraftsNeedingDone(ctx, workspaceID, limit)
	if externalConfirmed > 0 || externalSuspected > 0 || draftMissing > 0 {
		slog.Info("CRM external draft send detection completed", "workspace_id", uuidToString(workspaceID), "confirmed", externalConfirmed, "suspected", externalSuspected, "missing", draftMissing)
	}
	return approvedSent + externalConfirmed, notifiedSent + externalSuspected + draftMissing
}

func (h *Handler) commentManuallySentDraftsNeedingDone(ctx context.Context, workspaceID pgtype.UUID, limit int) int {
	rows, err := h.DB.Query(ctx, `
		SELECT d.id, d.issue_id
		FROM crm_email_draft d
		JOIN issue i ON i.id=d.issue_id AND i.workspace_id=d.workspace_id
		WHERE d.workspace_id=$1
		  AND d.issue_id IS NOT NULL
		  AND d.status='sent'
		  AND d.sent_at IS NOT NULL
		  AND i.status <> 'done'
		  AND NOT EXISTS (
			SELECT 1 FROM comment c
			WHERE c.issue_id=d.issue_id
			  AND c.workspace_id=d.workspace_id
			  AND c.content LIKE '%' || d.id::text || '%'
			  AND c.content LIKE '%请负责人%done%'
		  )
		ORDER BY d.sent_at ASC
		LIMIT $2`, workspaceID, limit)
	if err != nil {
		slog.Warn("CRM manually sent draft notification query failed", "workspace_id", uuidToString(workspaceID), "error", err)
		return 0
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var draftID, issueID pgtype.UUID
		if err := rows.Scan(&draftID, &issueID); err != nil || !draftID.Valid || !issueID.Valid {
			continue
		}
		if err := h.addCRMInternalIssueComment(ctx, workspaceID, issueID, fmt.Sprintf("绑定邮件草稿已手动修改并发送。\n\n草稿 ID：%s\n\n请负责人确认客户跟进已完成后，将此 Issue 状态改为 done。", uuidToString(draftID))); err == nil {
			count++
		}
	}
	return count
}

func (h *Handler) detectExternalSentCRMEmailDrafts(ctx context.Context, workspaceID pgtype.UUID, limit int) (int, int, int) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := h.DB.Query(ctx, `
		SELECT d.id, d.issue_id, d.thread_id, d.mailbox_id, lower(d.subject), lower(array_to_string(d.to_emails, ',')), COALESCE(d.in_reply_to,''), d.reference_ids, d.updated_at, COALESCE(d.external_draft_uid,''), COALESCE(d.external_draft_mailbox,'')
		FROM crm_email_draft d
		JOIN issue i ON i.id=d.issue_id AND i.workspace_id=d.workspace_id
		WHERE d.workspace_id=$1
		  AND d.issue_id IS NOT NULL
		  AND d.thread_id IS NOT NULL
		  AND d.status IN ('pending_approval','draft','failed')
		  AND d.sent_at IS NULL
		  AND i.status <> 'done'
		ORDER BY d.updated_at ASC
		LIMIT $2`, workspaceID, limit)
	if err != nil {
		slog.Warn("CRM external draft detection query failed", "workspace_id", uuidToString(workspaceID), "error", err)
		return 0, 0, 0
	}
	defer rows.Close()
	confirmed, suspected, missing := 0, 0, 0
	for rows.Next() {
		var draftID, issueID, threadID, mailboxID pgtype.UUID
		var subjectLower, toLower, inReplyTo, externalDraftUID, externalDraftMailbox string
		var refs []string
		var updatedAt pgtype.Timestamptz
		if err := rows.Scan(&draftID, &issueID, &threadID, &mailboxID, &subjectLower, &toLower, &inReplyTo, &refs, &updatedAt, &externalDraftUID, &externalDraftMailbox); err != nil {
			continue
		}
		_ = mailboxID
		var sentID pgtype.UUID
		var sentAt pgtype.Timestamptz
		var sentSubject, sentTo, sentInReplyTo string
		var sentRefs []string
		matchErr := h.DB.QueryRow(ctx, `
			SELECT m.id, COALESCE(m.sent_at,m.received_at,m.created_at), lower(COALESCE(m.subject,'')), lower(array_to_string(m.to_emails,',')), COALESCE(m.in_reply_to,''), m.reference_ids
			FROM crm_email_message m
			WHERE m.workspace_id=$1
			  AND m.thread_id=$2
			  AND (m.direction='outbound' OR lower(COALESCE(NULLIF(m.folder,''), NULLIF(m.source_metadata->>'folder',''))) IN ('sent','sent messages','sent items'))
			  AND COALESCE(m.sent_at,m.received_at,m.created_at) >= COALESCE($3::timestamptz, now() - interval '30 days') - interval '10 minutes'
			ORDER BY COALESCE(m.sent_at,m.received_at,m.created_at) DESC
			LIMIT 1`, workspaceID, threadID, updatedAt).Scan(&sentID, &sentAt, &sentSubject, &sentTo, &sentInReplyTo, &sentRefs)
		if matchErr == nil && sentID.Valid {
			score := 0
			reasons := []string{}
			if inReplyTo != "" && sentInReplyTo == inReplyTo {
				score += 50
				reasons = append(reasons, "in_reply_to matched")
			}
			if referencesOverlap(refs, sentRefs) {
				score += 50
				reasons = append(reasons, "references matched")
			}
			if toLower != "" && sentTo != "" && emailListOverlaps(toLower, sentTo) {
				score += 20
				reasons = append(reasons, "recipient matched")
			}
			if normalizeCRMSubject(subjectLower) != "" && normalizeCRMSubject(subjectLower) == normalizeCRMSubject(sentSubject) {
				score += 15
				reasons = append(reasons, "subject matched")
			}
			if sentAt.Valid && updatedAt.Valid && sentAt.Time.After(updatedAt.Time.Add(-10*time.Minute)) {
				score += 10
				reasons = append(reasons, "sent time after draft update")
			}
			if score >= 70 {
				_, _ = h.DB.Exec(ctx, `UPDATE crm_email_draft SET status='sent', sent_at=$3, sent_detection_status='external_sent_confirmed', sent_detection_confidence=$4, sent_detection_reason=$5, external_sent_uid=$6, external_sent_mailbox='Sent', sent_detected_at=now(), updated_at=now() WHERE id=$1 AND workspace_id=$2 AND sent_at IS NULL`, draftID, workspaceID, sentAt, score, strings.Join(reasons, "; "), uuidToString(sentID))
				_ = h.addCRMInternalIssueComment(ctx, workspaceID, issueID, fmt.Sprintf("系统在 Sent 文件夹中高置信匹配到绑定草稿已从外部邮箱发送。\n\n草稿 ID：%s\nSent 邮件 ID：%s\n置信度：%d\n依据：%s\n\n请负责人确认客户跟进已完成后，将此 Issue 状态改为 done。", uuidToString(draftID), uuidToString(sentID), score, strings.Join(reasons, "；")))
				confirmed++
			} else if score >= 50 {
				_, _ = h.DB.Exec(ctx, `UPDATE crm_email_draft SET sent_detection_status='external_sent_suspected', sent_detection_confidence=$3, sent_detection_reason=$4, external_sent_uid=$5, external_sent_mailbox='Sent', sent_detected_at=now(), updated_at=now() WHERE id=$1 AND workspace_id=$2`, draftID, workspaceID, score, strings.Join(reasons, "; "), uuidToString(sentID))
				_ = h.addCRMInternalIssueComment(ctx, workspaceID, issueID, fmt.Sprintf("系统在 Sent 文件夹中发现疑似已外部发送的邮件，但置信度不足以自动确认。\n\n草稿 ID：%s\n疑似 Sent 邮件 ID：%s\n置信度：%d\n依据：%s\n\n请负责人检查邮箱 Sent 文件夹，并确认是否将此 Issue 改为 done。", uuidToString(draftID), uuidToString(sentID), score, strings.Join(reasons, "；")))
				suspected++
			}
			continue
		}
		if strings.TrimSpace(externalDraftUID) == "" || strings.TrimSpace(externalDraftMailbox) == "" {
			continue
		}
		var draftStillExists bool
		_ = h.DB.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM crm_email_message m
			WHERE m.workspace_id=$1 AND m.thread_id=$2
			  AND lower(COALESCE(NULLIF(m.folder,''), NULLIF(m.source_metadata->>'folder',''))) IN ('drafts','draft','草稿箱')
			  AND lower(COALESCE(m.subject,''))=$3
		)`, workspaceID, threadID, subjectLower).Scan(&draftStillExists)
		if !draftStillExists {
			cmd, _ := h.DB.Exec(ctx, `UPDATE crm_email_draft SET sent_detection_status='draft_missing_unconfirmed', sent_detection_confidence=0, sent_detection_reason='draft not found in Drafts and no confident Sent match', sent_detected_at=now(), updated_at=now() WHERE id=$1 AND workspace_id=$2 AND COALESCE(sent_detection_status,'') <> 'draft_missing_unconfirmed'`, draftID, workspaceID)
			if cmd.RowsAffected() > 0 {
				_ = h.addCRMInternalIssueComment(ctx, workspaceID, issueID, fmt.Sprintf("绑定草稿已不在邮箱 Drafts 文件夹中，但系统未能确认是否已发送。\n\n草稿 ID：%s\n\n请负责人检查邮箱 Sent 文件夹或客户回复，并确认是否将此 Issue 改为 done。", uuidToString(draftID)))
				missing++
			}
		}
	}
	return confirmed, suspected, missing
}

func normalizeCRMSubject(value string) string {
	v := strings.TrimSpace(strings.ToLower(value))
	for {
		old := v
		v = strings.TrimSpace(strings.TrimPrefix(v, "re:"))
		v = strings.TrimSpace(strings.TrimPrefix(v, "fw:"))
		v = strings.TrimSpace(strings.TrimPrefix(v, "fwd:"))
		if v == old {
			return v
		}
	}
}

func referencesOverlap(a, b []string) bool {
	seen := map[string]bool{}
	for _, value := range a {
		v := strings.TrimSpace(strings.ToLower(value))
		if v != "" {
			seen[v] = true
		}
	}
	for _, value := range b {
		if seen[strings.TrimSpace(strings.ToLower(value))] {
			return true
		}
	}
	return false
}

func emailListOverlaps(a, b string) bool {
	seen := map[string]bool{}
	for _, part := range strings.Split(a, ",") {
		v := strings.TrimSpace(part)
		if v != "" {
			seen[v] = true
		}
	}
	for _, part := range strings.Split(b, ",") {
		if seen[strings.TrimSpace(part)] {
			return true
		}
	}
	return false
}

func (h *Handler) markCRMEmailDraftIssueSent(ctx context.Context, workspaceID, issueID, draftID pgtype.UUID, note string) error {
	_, err := h.DB.Exec(ctx, `UPDATE issue SET status='done', updated_at=now() WHERE id=$1 AND workspace_id=$2 AND status NOT IN ('done','cancelled')`, issueID, workspaceID)
	if err != nil {
		return err
	}
	return h.addCRMInternalIssueComment(ctx, workspaceID, issueID, fmt.Sprintf("%s\n\n草稿 ID：%s", note, uuidToString(draftID)))
}

func parseUUIDStringToPgtype(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(value)
	return id, err
}
