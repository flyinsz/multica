package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	var err error
	switch item.AutomationKey {
	case "email_pending_reply":
		result, err = h.runCRMPendingReplyAutomation(ctx, item.WorkspaceID, item.MaxItemsPerRun)
	case "due_followup":
		result, err = h.runCRMDueFollowupAutomation(ctx, item.WorkspaceID, item.MaxItemsPerRun)
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

func (h *Handler) runCRMPendingReplyAutomation(ctx context.Context, workspaceID pgtype.UUID, limit int) (map[string]any, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	rows, err := h.DB.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (m.thread_id)
				m.thread_id, m.id AS message_id, m.direction, m.subject, m.from_email, m.received_at, m.sent_at, m.created_at, m.is_read, m.is_trashed
			FROM crm_email_message m
			WHERE m.workspace_id=$1 AND COALESCE(m.is_trashed,false)=false
			ORDER BY m.thread_id, COALESCE(m.received_at, m.sent_at, m.created_at) DESC
		)
		SELECT t.id, COALESCE(latest.subject, t.subject, ''), COALESCE(a.name, ''), COALESCE(latest.received_at, latest.sent_at, latest.created_at, t.last_message_at, t.updated_at)
		FROM crm_email_thread t
		JOIN latest ON latest.thread_id=t.id
		LEFT JOIN crm_account a ON a.id=t.account_id AND a.workspace_id=t.workspace_id
		WHERE t.workspace_id=$1
		  AND COALESCE(t.is_trashed,false)=false
		  AND COALESCE(latest.is_trashed,false)=false
		  AND COALESCE(t.status,'open') <> 'archived'
		  AND latest.direction='inbound'
		  AND COALESCE(latest.is_read, t.is_read, false)=false
		  AND NOT EXISTS (
			SELECT 1 FROM issue i
			WHERE i.workspace_id=$1 AND i.origin_type='crm_ai' AND i.origin_id=t.id AND i.status NOT IN ('done','cancelled')
		  )
		ORDER BY COALESCE(latest.received_at, latest.sent_at, latest.created_at, t.last_message_at, t.updated_at) DESC
		LIMIT $2`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	created := 0
	candidates := 0
	for rows.Next() {
		var threadID pgtype.UUID
		var subject, accountName string
		var lastAt pgtype.Timestamptz
		if err := rows.Scan(&threadID, &subject, &accountName, &lastAt); err != nil {
			continue
		}
		candidates++
		title := stringsTrimForCRM(fmt.Sprintf("回复邮件：%s", subject), 120)
		body := fmt.Sprintf("CRM 邮件待回复巡检自动创建。\n客户：%s\n邮件主题：%s\n邮件线程：%s\n最新入站时间：%s", accountName, subject, uuidToString(threadID), timestampToString(lastAt))
		issueID, err := h.createCRMInternalIssue(ctx, workspaceID, title, body, uuidToString(threadID))
		if err == nil && issueID.Valid {
			created++
			_ = h.createCRMAIPendingReplyDraft(ctx, workspaceID, issueID, threadID, subject, accountName)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"automation_key": "email_pending_reply", "candidates": candidates, "created": created}, nil
}

func (h *Handler) runCRMDueFollowupAutomation(ctx context.Context, workspaceID pgtype.UUID, limit int) (map[string]any, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
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
	for rows.Next() {
		var accountID pgtype.UUID
		var name string
		var due pgtype.Timestamptz
		if err := rows.Scan(&accountID, &name, &due); err != nil {
			continue
		}
		title := stringsTrimForCRM(fmt.Sprintf("跟进客户：%s", name), 120)
		body := fmt.Sprintf("CRM 到期客户跟进自动创建。\n客户：%s\n客户ID：%s\n到期时间：%s", name, uuidToString(accountID), timestampToString(due))
		issueID, err := h.createCRMInternalIssue(ctx, workspaceID, title, body, uuidToString(accountID))
		if err == nil && issueID.Valid {
			created++
			_ = h.createCRMAIFollowupDraft(ctx, workspaceID, issueID, accountID, name, timestampToString(due))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"automation_key": "due_followup", "created": created}, nil
}

func stringsTrimForCRM(value string, limit int) string {
	if len([]rune(value)) <= limit {
		return value
	}
	r := []rune(value)
	return string(r[:limit])
}

func (h *Handler) createCRMAIPendingReplyDraft(ctx context.Context, workspaceID, issueID, threadID pgtype.UUID, subject, accountName string) error {
	var mailboxID, accountID, contactID pgtype.UUID
	var fromEmail, bodyText, bodyHTML, inReplyTo string
	var refs []string
	if err := h.DB.QueryRow(ctx, `
		SELECT s.id, m.account_id, m.contact_id, COALESCE(m.from_email,''), COALESCE(m.body_text,''), COALESCE(m.body_html,''), COALESCE(m.external_message_id,''), m.reference_ids
		FROM crm_email_message m
		JOIN crm_email_thread t ON t.id=m.thread_id AND t.workspace_id=m.workspace_id
		LEFT JOIN crm_imap_setting s ON s.workspace_id=m.workspace_id AND lower(s.email)=lower(t.mailbox)
		WHERE m.workspace_id=$1 AND m.thread_id=$2 AND m.direction='inbound'
		ORDER BY COALESCE(m.received_at, m.created_at) DESC LIMIT 1`, workspaceID, threadID).Scan(&mailboxID, &accountID, &contactID, &fromEmail, &bodyText, &bodyHTML, &inReplyTo, &refs); err != nil {
		return err
	}
	reason := fmt.Sprintf("草稿思路：这是待回复邮件巡检生成的回复草稿。依据最新入站邮件、线程主题和客户上下文生成，目标是先确认已收到并推进下一步。客户：%s。风险：AI 草稿需人工审核事实、价格、交期和附件后再发送。", accountName)
	draftBody := fmt.Sprintf("您好，\n\n感谢您的来信。我们已收到关于“%s”的信息，会尽快确认细节并回复您。\n\nBest regards", subject)
	var draftID pgtype.UUID
	if err := h.DB.QueryRow(ctx, `INSERT INTO crm_email_draft (workspace_id, mailbox_id, thread_id, account_id, contact_id, issue_id, to_emails, subject, body_text, body_html, in_reply_to, reference_ids, status, ai_generated, approval_reason) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'pending_approval',true,$13) RETURNING id`, workspaceID, mailboxID, threadID, accountID, contactID, issueID, []string{fromEmail}, "Re: "+subject, draftBody, cleanOptionalText(&bodyHTML), cleanOptionalText(&inReplyTo), refs, cleanOptionalText(&reason)).Scan(&draftID); err != nil {
		return err
	}
	return h.addCRMInternalIssueComment(ctx, workspaceID, issueID, fmt.Sprintf("已生成待审核邮件草稿。\n\n草稿链接：/crm/emails?draft=%s\n\n%s\n\n原邮件摘要：%s", uuidToString(draftID), reason, stringsTrimForCRM(bodyText, 600)))
}

func (h *Handler) createCRMAIFollowupDraft(ctx context.Context, workspaceID, issueID, accountID pgtype.UUID, accountName, dueText string) error {
	var mailboxID, contactID pgtype.UUID
	var email, contactName string
	if err := h.DB.QueryRow(ctx, `SELECT id FROM crm_imap_setting WHERE workspace_id=$1 AND enabled=true ORDER BY updated_at DESC LIMIT 1`, workspaceID).Scan(&mailboxID); err != nil {
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
	return h.addCRMInternalIssueComment(ctx, workspaceID, issueID, fmt.Sprintf("已生成待审核跟进邮件草稿。\n\n草稿链接：/crm/emails?draft=%s\n\n%s", uuidToString(draftID), reason))
}

func (h *Handler) createCRMInternalIssue(ctx context.Context, workspaceID pgtype.UUID, title, body, originID string) (pgtype.UUID, error) {
	var issueID pgtype.UUID
	originUUID, err := parseUUIDStringToPgtype(originID)
	if err != nil {
		return issueID, err
	}
	err = h.DB.QueryRow(ctx, `
		INSERT INTO issue (workspace_id, title, description, status, priority, origin_type, origin_id)
		SELECT $1, $2, $3, 'in_review', 'medium', 'crm_ai', $4
		WHERE NOT EXISTS (
			SELECT 1 FROM issue WHERE workspace_id=$1 AND origin_type='crm_ai' AND origin_id=$4 AND status NOT IN ('done','cancelled')
		)
		RETURNING id`, workspaceID, title, body, originUUID).Scan(&issueID)
	if err != nil && err == pgx.ErrNoRows {
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
