package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type crmWhatsAppMessagePayload struct {
	MessageID string          `json:"message_id"`
	ChatID    string          `json:"chat_id"`
	ChatName  string          `json:"chat_name"`
	IsGroup   bool            `json:"is_group"`
	Direction string          `json:"direction"`
	From      string          `json:"from"`
	To        string          `json:"to"`
	BodyText  string          `json:"body_text"`
	Timestamp string          `json:"timestamp"`
	Media     json.RawMessage `json:"media"`
	Raw       json.RawMessage `json:"raw"`
}

type crmWhatsAppWebhookPayload struct {
	Event     string                    `json:"event"`
	Provider  string                    `json:"provider"`
	AccountID string                    `json:"account_id"`
	Message   crmWhatsAppMessagePayload `json:"message"`
}

type crmWhatsAppThreadResponse struct {
	ID              string     `json:"id"`
	WorkspaceID     string     `json:"workspace_id"`
	ExternalChatID  string     `json:"external_chat_id"`
	Title           string     `json:"title"`
	PhoneNumber     string     `json:"phone_number"`
	AccountID       *string    `json:"account_id,omitempty"`
	ContactID       *string    `json:"contact_id,omitempty"`
	LastMessageAt   *time.Time `json:"last_message_at,omitempty"`
	UnreadCount     int        `json:"unread_count"`
	LastMessageText string     `json:"last_message_text,omitempty"`
}

type crmWhatsAppMessageResponse struct {
	ID                string          `json:"id"`
	ThreadID          string          `json:"thread_id"`
	ExternalMessageID string          `json:"external_message_id"`
	Direction         string          `json:"direction"`
	FromNumber        string          `json:"from_number"`
	ToNumber          string          `json:"to_number"`
	BodyText          string          `json:"body_text"`
	Media             json.RawMessage `json:"media"`
	SentAt            *time.Time      `json:"sent_at,omitempty"`
	ReceivedAt        *time.Time      `json:"received_at,omitempty"`
}

func (h *Handler) ListCRMWhatsAppThreads(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT t.id, t.workspace_id, t.external_chat_id, t.title, t.phone_number,
		       t.account_id, t.contact_id, t.last_message_at, t.unread_count,
		       COALESCE((SELECT m.body_text FROM crm_whatsapp_message m WHERE m.thread_id = t.id ORDER BY COALESCE(m.sent_at, m.received_at, m.created_at) DESC LIMIT 1), '') AS last_message_text
		FROM crm_whatsapp_thread t
		WHERE t.workspace_id = $1
		ORDER BY t.last_message_at DESC NULLS LAST, t.updated_at DESC
		LIMIT 200`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list whatsapp threads")
		return
	}
	defer rows.Close()
	out := []crmWhatsAppThreadResponse{}
	for rows.Next() {
		var id, ws pgtype.UUID
		var accountID, contactID pgtype.UUID
		var resp crmWhatsAppThreadResponse
		var last pgtype.Timestamptz
		if err := rows.Scan(&id, &ws, &resp.ExternalChatID, &resp.Title, &resp.PhoneNumber, &accountID, &contactID, &last, &resp.UnreadCount, &resp.LastMessageText); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan whatsapp thread")
			return
		}
		resp.ID = uuidToString(id)
		resp.WorkspaceID = uuidToString(ws)
		if accountID.Valid {
			v := uuidToString(accountID)
			resp.AccountID = &v
		}
		if contactID.Valid {
			v := uuidToString(contactID)
			resp.ContactID = &v
		}
		if last.Valid {
			resp.LastMessageAt = &last.Time
		}
		out = append(out, resp)
	}
	writeJSON(w, http.StatusOK, map[string]any{"threads": out})
}

func (h *Handler) ListCRMWhatsAppMessages(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	threadID := chi.URLParam(r, "threadId")
	var threadUUID pgtype.UUID
	if err := threadUUID.Scan(threadID); err != nil || !threadUUID.Valid {
		writeError(w, http.StatusBadRequest, "invalid thread id")
		return
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT id, thread_id, external_message_id, direction, from_number, to_number, body_text, media, sent_at, received_at
		FROM crm_whatsapp_message
		WHERE workspace_id = $1 AND thread_id = $2
		ORDER BY COALESCE(sent_at, received_at, created_at) DESC
		LIMIT 200`, workspaceID, threadUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list whatsapp messages")
		return
	}
	defer rows.Close()
	out := []crmWhatsAppMessageResponse{}
	for rows.Next() {
		var id, tid pgtype.UUID
		var media []byte
		var sent, received pgtype.Timestamptz
		var resp crmWhatsAppMessageResponse
		if err := rows.Scan(&id, &tid, &resp.ExternalMessageID, &resp.Direction, &resp.FromNumber, &resp.ToNumber, &resp.BodyText, &media, &sent, &received); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan whatsapp message")
			return
		}
		resp.ID = uuidToString(id)
		resp.ThreadID = uuidToString(tid)
		resp.Media = json.RawMessage(media)
		if sent.Valid {
			resp.SentAt = &sent.Time
		}
		if received.Valid {
			resp.ReceivedAt = &received.Time
		}
		out = append(out, resp)
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": out})
}

func (h *Handler) SyncCRMWhatsAppFromHermes(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	base := strings.TrimRight(os.Getenv("CRM_WHATSAPP_HERMES_BASE_URL"), "/")
	if base == "" {
		base = "http://127.0.0.1:3000"
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(base + "/crm/chats?limit=200")
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to reach hermes whatsapp bridge")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		writeError(w, http.StatusBadGateway, "hermes whatsapp bridge returned error")
		return
	}
	var chats struct {
		Chats []struct {
			ChatID          string `json:"chat_id"`
			Title           string `json:"title"`
			PhoneNumber     string `json:"phone_number"`
			LastMessageAt   string `json:"last_message_at"`
			LastMessageText string `json:"last_message_text"`
			IsGroup         bool   `json:"is_group"`
			UnreadCount     int    `json:"unread_count"`
		} `json:"chats"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chats); err != nil {
		writeError(w, http.StatusBadGateway, "invalid hermes chats response")
		return
	}
	imported := 0
	for _, chat := range chats.Chats {
		msgResp, err := client.Get(base + "/crm/chats/" + url.PathEscape(chat.ChatID) + "/messages?limit=100")
		if err != nil {
			continue
		}
		var payload struct {
			Messages []crmWhatsAppMessagePayload `json:"messages"`
		}
		_ = json.NewDecoder(msgResp.Body).Decode(&payload)
		_ = msgResp.Body.Close()
		for _, msg := range payload.Messages {
			if msg.ChatID == "" {
				msg.ChatID = chat.ChatID
			}
			if msg.ChatName == "" {
				msg.ChatName = chat.Title
			}
			if _, err := h.upsertCRMWhatsAppMessage(r.Context(), workspaceID, "hermes", "default", msg); err == nil {
				imported++
			} else {
				slog.Warn("failed to import whatsapp message", "error", err)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"imported": imported})
}

func (h *Handler) ReceiveCRMWhatsAppHermesWebhook(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	if expected := strings.TrimSpace(os.Getenv("CRM_WHATSAPP_HERMES_WEBHOOK_TOKEN")); expected != "" {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got != expected {
			writeError(w, http.StatusUnauthorized, "invalid webhook token")
			return
		}
	}
	var payload crmWhatsAppWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	provider := strings.TrimSpace(payload.Provider)
	if provider == "" {
		provider = "hermes_baileys"
	}
	accountID := strings.TrimSpace(payload.AccountID)
	if accountID == "" {
		accountID = "default"
	}
	messageID, err := h.upsertCRMWhatsAppMessage(r.Context(), workspaceID, provider, accountID, payload.Message)
	if err != nil {
		slog.Error("failed to upsert whatsapp webhook", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to upsert whatsapp message")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message_id": messageID})
}

func (h *Handler) upsertCRMWhatsAppMessage(ctx context.Context, workspaceID pgtype.UUID, provider, providerAccountID string, msg crmWhatsAppMessagePayload) (string, error) {
	if strings.TrimSpace(msg.ChatID) == "" || strings.TrimSpace(msg.MessageID) == "" {
		return "", fmt.Errorf("chat_id and message_id required")
	}
	dir := strings.ToLower(strings.TrimSpace(msg.Direction))
	if dir != "outbound" {
		dir = "inbound"
	}
	media := []byte("[]")
	if len(bytes.TrimSpace(msg.Media)) > 0 {
		media = msg.Media
	}
	raw := []byte("{}")
	if len(bytes.TrimSpace(msg.Raw)) > 0 {
		raw = msg.Raw
	}
	ts, _ := time.Parse(time.RFC3339, msg.Timestamp)
	var tsArg any
	if !ts.IsZero() {
		tsArg = ts
	}
	phone := msg.From
	if dir == "outbound" {
		phone = msg.To
	}
	chatTitle := msg.ChatName
	if chatTitle == "" {
		chatTitle = phone
	}
	var id pgtype.UUID
	err := h.DB.QueryRow(ctx, `
		WITH acct AS (
			INSERT INTO crm_whatsapp_account (workspace_id, provider, provider_account_id, status)
			VALUES ($1, $2, $3, 'connected')
			ON CONFLICT (workspace_id, provider, provider_account_id)
			DO UPDATE SET status = 'connected', updated_at = now()
			RETURNING id
		), th AS (
			INSERT INTO crm_whatsapp_thread (workspace_id, whatsapp_account_id, external_chat_id, title, phone_number, last_message_at, unread_count)
			SELECT $1, acct.id, $4, $5, $6, $7, CASE WHEN $8 = 'inbound' THEN 1 ELSE 0 END FROM acct
			ON CONFLICT (workspace_id, whatsapp_account_id, external_chat_id)
			DO UPDATE SET title = EXCLUDED.title, phone_number = COALESCE(NULLIF(EXCLUDED.phone_number, ''), crm_whatsapp_thread.phone_number), last_message_at = GREATEST(crm_whatsapp_thread.last_message_at, EXCLUDED.last_message_at), updated_at = now()
			RETURNING id
		), ins AS (
			INSERT INTO crm_whatsapp_message (workspace_id, thread_id, external_message_id, direction, from_number, to_number, body_text, media, sent_at, received_at, raw)
			SELECT $1, th.id, $9, $8, $10, $11, $12, $13::jsonb, CASE WHEN $8 = 'outbound' THEN $7::timestamptz ELSE NULL END, CASE WHEN $8 = 'inbound' THEN $7::timestamptz ELSE NULL END, $14::jsonb FROM th
			ON CONFLICT (workspace_id, thread_id, external_message_id)
			DO UPDATE SET body_text = EXCLUDED.body_text, media = EXCLUDED.media, raw = EXCLUDED.raw, updated_at = now()
			RETURNING id
		)
		SELECT id FROM ins`, workspaceID, provider, providerAccountID, msg.ChatID, chatTitle, phone, tsArg, dir, msg.MessageID, msg.From, msg.To, msg.BodyText, string(media), string(raw)).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return uuidToString(id), nil
}
