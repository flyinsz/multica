package handler

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultCRMIMAPAutoSyncInterval = 5 * time.Minute

// CRMIMAPAutoSyncScheduler runs enabled CRM IMAP mailboxes on a fixed interval.
// It uses the same import path as the manual SyncCRMIMAP endpoint so migration
// and deployment stay inside the Multica backend, not an external cron script.
type CRMIMAPAutoSyncScheduler struct {
	db       *pgxpool.Pool
	interval time.Duration
	limit    int
}

func NewCRMIMAPAutoSyncScheduler(db *pgxpool.Pool, interval time.Duration, limit int) *CRMIMAPAutoSyncScheduler {
	if interval <= 0 {
		interval = defaultCRMIMAPAutoSyncInterval
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return &CRMIMAPAutoSyncScheduler{db: db, interval: interval, limit: limit}
}

func (s *CRMIMAPAutoSyncScheduler) Run(ctx context.Context) {
	if s == nil || s.db == nil {
		return
	}
	slog.Info("CRM IMAP auto sync scheduler started", "interval", s.interval.String(), "limit", s.limit)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			slog.Info("CRM IMAP auto sync scheduler stopped")
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *CRMIMAPAutoSyncScheduler) runOnce(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, s.interval)
	defer cancel()

	_, _ = s.db.Exec(ctx, `UPDATE crm_imap_sync_run SET status='failed', error_message='stale running sync reset by auto scheduler', finished_at=now(), updated_at=now() WHERE status='running' AND started_at < now() - interval '10 minutes'`)

	rows, err := s.db.Query(ctx, `
		SELECT id, workspace_id, label, email, host, port, tls_mode, username, secret_ref, owner_type, owner_id, smtp_host, smtp_port, smtp_tls_mode, smtp_username, smtp_secret_ref
		FROM crm_imap_setting s
		WHERE s.sync_enabled = true
		  AND NOT EXISTS (
			SELECT 1 FROM crm_imap_sync_run r
			WHERE r.workspace_id=s.workspace_id AND r.mailbox_id=s.id AND r.status='running'
		  )
		  AND COALESCE((
			SELECT max(COALESCE(r.finished_at, r.started_at, r.created_at))
			FROM crm_imap_sync_run r
			WHERE r.workspace_id=s.workspace_id AND r.mailbox_id=s.id AND r.folder='INBOX' AND r.status IN ('ok','failed')
		  ), 'epoch'::timestamptz) <= now() - $1::interval
		ORDER BY s.updated_at DESC
		LIMIT 10`, s.interval.String())
	if err != nil {
		slog.Warn("CRM IMAP auto sync mailbox query failed", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		cfg, workspaceID, ok := scanCRMIMAPAutoSyncMailbox(rows)
		if !ok {
			continue
		}
		s.syncMailbox(parent, workspaceID, cfg)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("CRM IMAP auto sync mailbox rows failed", "error", err)
	}
}

func scanCRMIMAPAutoSyncMailbox(row pgx.Row) (crmIMAPMailboxConfig, pgtype.UUID, bool) {
	var cfg crmIMAPMailboxConfig
	var workspaceID, id, ownerID pgtype.UUID
	var label, email, host, tlsMode, username, secretRef, ownerType, smtpHost, smtpTLSMode, smtpUsername, smtpSecretRef pgtype.Text
	var port, smtpPort pgtype.Int4
	if err := row.Scan(&id, &workspaceID, &label, &email, &host, &port, &tlsMode, &username, &secretRef, &ownerType, &ownerID, &smtpHost, &smtpPort, &smtpTLSMode, &smtpUsername, &smtpSecretRef); err != nil {
		slog.Warn("CRM IMAP auto sync mailbox scan failed", "error", err)
		return cfg, workspaceID, false
	}
	cfg.UUID = id
	cfg.ID = uuidToString(id)
	cfg.Label = crmTextValue(label)
	cfg.Email = crmTextValue(email)
	cfg.Host = crmTextValue(host)
	cfg.Port = port.Int32
	cfg.TLSMode = crmTextValue(tlsMode)
	cfg.Username = crmTextValue(username)
	cfg.SecretRef = crmTextValue(secretRef)
	cfg.OwnerType = crmTextValue(ownerType)
	cfg.OwnerID = uuidToString(ownerID)
	cfg.SMTPHost = crmTextValue(smtpHost)
	if smtpPort.Valid {
		cfg.SMTPPort = smtpPort.Int32
	}
	cfg.SMTPTLSMode = crmTextValue(smtpTLSMode)
	cfg.SMTPUsername = crmTextValue(smtpUsername)
	cfg.SMTPSecretRef = crmTextValue(smtpSecretRef)
	return cfg, workspaceID, true
}

func (s *CRMIMAPAutoSyncScheduler) syncMailbox(parent context.Context, workspaceID pgtype.UUID, cfg crmIMAPMailboxConfig) {
	runCtx, runCancel := context.WithTimeout(parent, 5*time.Minute)
	defer runCancel()

	var runID pgtype.UUID
	if err := s.db.QueryRow(runCtx, `INSERT INTO crm_imap_sync_run (workspace_id, mailbox_id, folder, requested_limit) VALUES ($1,$2,'INBOX',$3) RETURNING id`, workspaceID, cfg.UUID, s.limit).Scan(&runID); err != nil {
		slog.Warn("CRM IMAP auto sync run create failed", "mailbox", cfg.Email, "error", err)
		return
	}
	finishRun := func(query string, args ...any) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = s.db.Exec(ctx, query, args...)
	}
	messages, err := fetchCRMEmailProviderMessages(cfg, "INBOX", s.limit, 0, nil)
	if err != nil {
		finishRun(`UPDATE crm_imap_sync_run SET status='failed', error_message=$2, finished_at=now(), updated_at=now() WHERE id=$1`, runID, err.Error())
		slog.Warn("CRM IMAP auto sync fetch failed", "mailbox", cfg.Email, "error", err)
		return
	}
	importCtx, importCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer importCancel()
	h := &Handler{DB: s.db}
	imported, skipped, err := h.importCRMIMAPMessages(importCtx, workspaceID, cfg, "INBOX", messages)
	if err != nil {
		finishRun(`UPDATE crm_imap_sync_run SET status='failed', fetched_count=$2, error_message=$3, finished_at=now(), updated_at=now() WHERE id=$1`, runID, len(messages), err.Error())
		slog.Warn("CRM IMAP auto sync import failed", "mailbox", cfg.Email, "error", err)
		return
	}
	finishRun(`UPDATE crm_imap_sync_run SET status='ok', fetched_count=$2, imported_count=$3, skipped_count=$4, finished_at=now(), updated_at=now() WHERE id=$1`, runID, len(messages), imported, skipped)
	slog.Info("CRM IMAP auto sync completed", "mailbox", cfg.Email, "fetched", len(messages), "imported", imported, "skipped", skipped)
}
