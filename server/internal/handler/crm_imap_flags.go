package handler

import (
	"context"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

func (h *Handler) trySyncCRMEmailThreadFlags(ctx context.Context, workspaceID pgtype.UUID, threadID pgtype.UUID, isRead *bool, isStarred *bool) {
	if isRead == nil && isStarred == nil {
		return
	}
	var mailboxID pgtype.UUID
	var externalID, mailbox string
	err := h.DB.QueryRow(ctx, `
		SELECT s.id, COALESCE(m.external_message_id,''), COALESCE(t.mailbox,'INBOX')
		FROM crm_email_thread t
		JOIN crm_email_message m ON m.thread_id = t.id AND m.workspace_id = t.workspace_id
		LEFT JOIN crm_imap_setting s ON s.workspace_id = t.workspace_id AND lower(s.email) = lower(t.mailbox)
		WHERE t.id=$1 AND t.workspace_id=$2 AND m.direction <> 'outbound'
		ORDER BY COALESCE(m.received_at, m.created_at) DESC
		LIMIT 1
	`, threadID, workspaceID).Scan(&mailboxID, &externalID, &mailbox)
	if err != nil || !mailboxID.Valid {
		return
	}
	cfg, ok := h.loadCRMIMAPConfigByID(ctx, workspaceID, mailboxID)
	if !ok {
		return
	}
	uid := uidFromCRMExternalMessageID(externalID, cfg.ID)
	if uid == "" {
		return
	}
	if err := syncCRMIMAPThreadFlags(cfg, mailbox, uid, isRead, isStarred); err != nil {
		slog.Warn("CRM IMAP flag sync failed", "workspace_id", uuidToString(workspaceID), "thread_id", uuidToString(threadID), "mailbox_id", cfg.ID, "uid", uid, "error", sanitizeCRMSendError(err).Error())
	}
}

func (h *Handler) loadCRMIMAPConfigByID(ctx context.Context, workspaceID pgtype.UUID, mailboxID pgtype.UUID) (crmIMAPMailboxConfig, bool) {
	query := `SELECT id, label, email, host, port, tls_mode, username, secret_ref, owner_type, owner_id, smtp_host, smtp_port, smtp_tls_mode, smtp_username, smtp_secret_ref FROM crm_imap_setting WHERE workspace_id=$1 AND id=$2 LIMIT 1`
	var cfg crmIMAPMailboxConfig
	var id pgtype.UUID
	var secretRef, ownerType, smtpHost, smtpTLSMode, smtpUsername, smtpSecretRef pgtype.Text
	var ownerID pgtype.UUID
	var smtpPort pgtype.Int4
	if err := h.DB.QueryRow(ctx, query, workspaceID, mailboxID).Scan(&id, &cfg.Label, &cfg.Email, &cfg.Host, &cfg.Port, &cfg.TLSMode, &cfg.Username, &secretRef, &ownerType, &ownerID, &smtpHost, &smtpPort, &smtpTLSMode, &smtpUsername, &smtpSecretRef); err != nil {
		return crmIMAPMailboxConfig{}, false
	}
	cfg.UUID = id
	cfg.ID = uuidToString(id)
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
	return cfg, true
}

func uidFromCRMExternalMessageID(externalID string, mailboxID string) string {
	externalID = strings.TrimSpace(externalID)
	mailboxID = strings.TrimSpace(mailboxID)
	prefix := mailboxID + ":"
	if mailboxID != "" && strings.HasPrefix(externalID, prefix) {
		uid := strings.TrimSpace(strings.TrimPrefix(externalID, prefix))
		if uid != "" && strings.IndexFunc(uid, func(r rune) bool { return r < '0' || r > '9' }) == -1 {
			return uid
		}
	}
	return ""
}
