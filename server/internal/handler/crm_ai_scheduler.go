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
		SELECT t.id, t.subject, COALESCE(a.name, ''), COALESCE(t.last_message_at, t.updated_at)
		FROM crm_email_thread t
		LEFT JOIN crm_account a ON a.id=t.account_id AND a.workspace_id=t.workspace_id
		WHERE t.workspace_id=$1 AND t.status='open' AND t.direction IN ('inbound','mixed') AND COALESCE(t.is_read,false)=false AND COALESCE(t.is_trashed,false)=false
		ORDER BY COALESCE(t.last_message_at, t.updated_at) DESC
		LIMIT $2`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	created := 0
	for rows.Next() {
		var threadID pgtype.UUID
		var subject, accountName string
		var lastAt pgtype.Timestamptz
		if err := rows.Scan(&threadID, &subject, &accountName, &lastAt); err != nil {
			continue
		}
		title := stringsTrimForCRM(fmt.Sprintf("回复邮件：%s", subject), 120)
		body := fmt.Sprintf("CRM 邮件待回复巡检自动创建。\n客户：%s\n邮件主题：%s\n邮件线程：%s", accountName, subject, uuidToString(threadID))
		if _, err := h.createCRMInternalIssue(ctx, workspaceID, title, body, uuidToString(threadID)); err == nil {
			created++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"automation_key": "email_pending_reply", "created": created}, nil
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
		if _, err := h.createCRMInternalIssue(ctx, workspaceID, title, body, uuidToString(accountID)); err == nil {
			created++
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

func (h *Handler) createCRMInternalIssue(ctx context.Context, workspaceID pgtype.UUID, title, body, originID string) (pgtype.UUID, error) {
	var issueID pgtype.UUID
	originUUID, err := parseUUIDStringToPgtype(originID)
	if err != nil {
		return issueID, err
	}
	err = h.DB.QueryRow(ctx, `
		INSERT INTO issue (workspace_id, title, description, status, priority, origin_type, origin_id)
		SELECT $1, $2, $3, 'todo', 'medium', 'crm_ai', $4
		WHERE NOT EXISTS (
			SELECT 1 FROM issue WHERE workspace_id=$1 AND origin_type='crm_ai' AND origin_id=$4 AND status NOT IN ('done','cancelled')
		)
		RETURNING id`, workspaceID, title, body, originUUID).Scan(&issueID)
	if err != nil && err == pgx.ErrNoRows {
		return pgtype.UUID{}, nil
	}
	return issueID, err
}

func parseUUIDStringToPgtype(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(value)
	return id, err
}
