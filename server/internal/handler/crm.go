package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// CRMAccountResponse is the JSON shape returned by the CRM account API.
type CRMAccountResponse struct {
	ID              string   `json:"id"`
	WorkspaceID     string   `json:"workspace_id"`
	Name            string   `json:"name"`
	AccountCode     *string  `json:"account_code"`
	AccountType     string   `json:"account_type"`
	Website         *string  `json:"website"`
	Country         *string  `json:"country"`
	CountryCode     *string  `json:"country_code"`
	CountryName     *string  `json:"country_name"`
	Region          *string  `json:"region"`
	City            *string  `json:"city"`
	Industry        *string  `json:"industry"`
	SubIndustry     *string  `json:"sub_industry"`
	Status          string   `json:"status"`
	OwnerID         *string  `json:"owner_id"`
	OwnerMemberID   *string  `json:"owner_member_id"`
	OwnerType       string   `json:"owner_type"`
	OwnerAgentID    *string  `json:"owner_agent_id"`
	Source          *string  `json:"source"`
	Rating          string   `json:"rating"`
	Priority        string   `json:"priority"`
	AnnualRevenue   *string  `json:"annual_revenue"`
	EmployeeCount   *string  `json:"employee_count"`
	Tags            []string `json:"tags"`
	Notes           *string  `json:"notes"`
	LastContactedAt *string  `json:"last_contacted_at"`
	NextFollowUpAt  *string  `json:"next_follow_up_at"`
	ContactCount    int64    `json:"contact_count"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type crmAccountRow struct {
	ID              pgtype.UUID
	WorkspaceID     pgtype.UUID
	Name            string
	NormalizedName  string
	AccountCode     pgtype.Text
	AccountType     string
	Website         pgtype.Text
	Country         pgtype.Text
	CountryCode     pgtype.Text
	CountryName     pgtype.Text
	Region          pgtype.Text
	City            pgtype.Text
	Industry        pgtype.Text
	SubIndustry     pgtype.Text
	Status          string
	OwnerID         pgtype.UUID
	OwnerMemberID   pgtype.UUID
	OwnerType       string
	OwnerAgentID    pgtype.UUID
	Source          pgtype.Text
	Rating          string
	Priority        string
	AnnualRevenue   pgtype.Text
	EmployeeCount   pgtype.Text
	Tags            []string
	Notes           pgtype.Text
	LastContactedAt pgtype.Timestamptz
	NextFollowUpAt  pgtype.Timestamptz
	CreatedAt       pgtype.Timestamptz
	UpdatedAt       pgtype.Timestamptz
	ContactCount    int64
}

func crmAccountToResponse(row crmAccountRow) CRMAccountResponse {
	tags := row.Tags
	if tags == nil {
		tags = []string{}
	}
	return CRMAccountResponse{
		ID:              uuidToString(row.ID),
		WorkspaceID:     uuidToString(row.WorkspaceID),
		Name:            row.Name,
		AccountCode:     textToPtr(row.AccountCode),
		AccountType:     row.AccountType,
		Website:         textToPtr(row.Website),
		Country:         textToPtr(row.Country),
		CountryCode:     textToPtr(row.CountryCode),
		CountryName:     textToPtr(row.CountryName),
		Region:          textToPtr(row.Region),
		City:            textToPtr(row.City),
		Industry:        textToPtr(row.Industry),
		SubIndustry:     textToPtr(row.SubIndustry),
		Status:          row.Status,
		OwnerID:         uuidToPtr(row.OwnerID),
		OwnerMemberID:   uuidToPtr(row.OwnerMemberID),
		OwnerType:       row.OwnerType,
		OwnerAgentID:    uuidToPtr(row.OwnerAgentID),
		Source:          textToPtr(row.Source),
		Rating:          row.Rating,
		Priority:        row.Priority,
		AnnualRevenue:   textToPtr(row.AnnualRevenue),
		EmployeeCount:   textToPtr(row.EmployeeCount),
		Tags:            tags,
		Notes:           textToPtr(row.Notes),
		LastContactedAt: timestampToPtr(row.LastContactedAt),
		NextFollowUpAt:  timestampToPtr(row.NextFollowUpAt),
		ContactCount:    row.ContactCount,
		CreatedAt:       timestampToString(row.CreatedAt),
		UpdatedAt:       timestampToString(row.UpdatedAt),
	}
}

type CreateCRMAccountRequest struct {
	Name            string   `json:"name"`
	AccountCode     *string  `json:"account_code"`
	AccountType     *string  `json:"account_type"`
	Website         *string  `json:"website"`
	Country         *string  `json:"country"`
	CountryCode     *string  `json:"country_code"`
	CountryName     *string  `json:"country_name"`
	Region          *string  `json:"region"`
	City            *string  `json:"city"`
	Industry        *string  `json:"industry"`
	SubIndustry     *string  `json:"sub_industry"`
	Status          *string  `json:"status"`
	OwnerID         *string  `json:"owner_id"`
	OwnerMemberID   *string  `json:"owner_member_id"`
	OwnerType       *string  `json:"owner_type"`
	OwnerAgentID    *string  `json:"owner_agent_id"`
	Source          *string  `json:"source"`
	Rating          *string  `json:"rating"`
	Priority        *string  `json:"priority"`
	AnnualRevenue   *string  `json:"annual_revenue"`
	EmployeeCount   *string  `json:"employee_count"`
	Tags            []string `json:"tags"`
	Notes           *string  `json:"notes"`
	LastContactedAt *string  `json:"last_contacted_at"`
	NextFollowUpAt  *string  `json:"next_follow_up_at"`
}

type UpdateCRMAccountRequest = CreateCRMAccountRequest

// CRMContactResponse is the JSON shape returned by the CRM contact API.
type CRMContactResponse struct {
	ID                string  `json:"id"`
	WorkspaceID       string  `json:"workspace_id"`
	AccountID         *string `json:"account_id"`
	Name              string  `json:"name"`
	Salutation        *string `json:"salutation"`
	Email             *string `json:"email"`
	Phone             *string `json:"phone"`
	Mobile            *string `json:"mobile"`
	WhatsappID        *string `json:"whatsapp_id"`
	Whatsapp          *string `json:"whatsapp"`
	Wechat            *string `json:"wechat"`
	LinkedinURL       *string `json:"linkedin_url"`
	RoleTitle         *string `json:"role_title"`
	JobTitle          *string `json:"job_title"`
	Department        *string `json:"department"`
	Role              *string `json:"role"`
	Language          *string `json:"language"`
	PreferredLanguage *string `json:"preferred_language"`
	Timezone          *string `json:"timezone"`
	IsPrimary         bool    `json:"is_primary"`
	DecisionRole      *string `json:"decision_role"`
	Notes             *string `json:"notes"`
	LastContactedAt   *string `json:"last_contacted_at"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type crmContactRow struct {
	ID                pgtype.UUID
	WorkspaceID       pgtype.UUID
	AccountID         pgtype.UUID
	Name              string
	Salutation        pgtype.Text
	Email             pgtype.Text
	Phone             pgtype.Text
	Mobile            pgtype.Text
	WhatsappID        pgtype.Text
	Whatsapp          pgtype.Text
	Wechat            pgtype.Text
	LinkedinURL       pgtype.Text
	RoleTitle         pgtype.Text
	JobTitle          pgtype.Text
	Department        pgtype.Text
	Role              pgtype.Text
	Language          pgtype.Text
	PreferredLanguage pgtype.Text
	Timezone          pgtype.Text
	IsPrimary         bool
	DecisionRole      pgtype.Text
	Notes             pgtype.Text
	LastContactedAt   pgtype.Timestamptz
	CreatedAt         pgtype.Timestamptz
	UpdatedAt         pgtype.Timestamptz
}

func crmContactToResponse(row crmContactRow) CRMContactResponse {
	return CRMContactResponse{
		ID:                uuidToString(row.ID),
		WorkspaceID:       uuidToString(row.WorkspaceID),
		AccountID:         uuidToPtr(row.AccountID),
		Name:              row.Name,
		Salutation:        textToPtr(row.Salutation),
		Email:             textToPtr(row.Email),
		Phone:             textToPtr(row.Phone),
		Mobile:            textToPtr(row.Mobile),
		WhatsappID:        textToPtr(row.WhatsappID),
		Whatsapp:          textToPtr(row.Whatsapp),
		Wechat:            textToPtr(row.Wechat),
		LinkedinURL:       textToPtr(row.LinkedinURL),
		RoleTitle:         textToPtr(row.RoleTitle),
		JobTitle:          textToPtr(row.JobTitle),
		Department:        textToPtr(row.Department),
		Role:              textToPtr(row.Role),
		Language:          textToPtr(row.Language),
		PreferredLanguage: textToPtr(row.PreferredLanguage),
		Timezone:          textToPtr(row.Timezone),
		IsPrimary:         row.IsPrimary,
		DecisionRole:      textToPtr(row.DecisionRole),
		Notes:             textToPtr(row.Notes),
		LastContactedAt:   timestampToPtr(row.LastContactedAt),
		CreatedAt:         timestampToString(row.CreatedAt),
		UpdatedAt:         timestampToString(row.UpdatedAt),
	}
}

type CreateCRMContactRequest struct {
	AccountID         *string `json:"account_id"`
	Name              string  `json:"name"`
	Salutation        *string `json:"salutation"`
	Email             *string `json:"email"`
	Phone             *string `json:"phone"`
	Mobile            *string `json:"mobile"`
	WhatsappID        *string `json:"whatsapp_id"`
	Whatsapp          *string `json:"whatsapp"`
	Wechat            *string `json:"wechat"`
	LinkedinURL       *string `json:"linkedin_url"`
	RoleTitle         *string `json:"role_title"`
	JobTitle          *string `json:"job_title"`
	Department        *string `json:"department"`
	Role              *string `json:"role"`
	Language          *string `json:"language"`
	PreferredLanguage *string `json:"preferred_language"`
	Timezone          *string `json:"timezone"`
	IsPrimary         *bool   `json:"is_primary"`
	DecisionRole      *string `json:"decision_role"`
	Notes             *string `json:"notes"`
	LastContactedAt   *string `json:"last_contacted_at"`
}

type UpdateCRMContactRequest = CreateCRMContactRequest

type CRMEmailThreadResponse struct {
	ID               string   `json:"id"`
	WorkspaceID      string   `json:"workspace_id"`
	AccountID        *string  `json:"account_id"`
	ContactID        *string  `json:"contact_id"`
	ProjectID        *string  `json:"project_id"`
	IssueID          *string  `json:"issue_id"`
	IssueIDs         []string `json:"issue_ids,omitempty"`
	Subject          string   `json:"subject"`
	ExternalThreadID *string  `json:"external_thread_id"`
	Mailbox          *string  `json:"mailbox"`
	Direction        string   `json:"direction"`
	Status           string   `json:"status"`
	LastMessageAt    *string  `json:"last_message_at"`
	LastSnippet      *string  `json:"last_snippet"`
	MessageCount     int64    `json:"message_count"`
	IsRead           bool     `json:"is_read"`
	IsStarred        bool     `json:"is_starred"`
	IsTrashed        bool     `json:"is_trashed"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type CRMEmailListItemResponse struct {
	ID                 string               `json:"id"`
	WorkspaceID        string               `json:"workspace_id"`
	ThreadID           string               `json:"thread_id"`
	AccountID          *string              `json:"account_id"`
	ContactID          *string              `json:"contact_id"`
	Subject            string               `json:"subject"`
	Snippet            *string              `json:"snippet"`
	Mailbox            *string              `json:"mailbox"`
	Folder             string               `json:"folder"`
	Direction          string               `json:"direction"`
	Status             string               `json:"status"`
	IsRead             bool                 `json:"is_read"`
	IsStarred          bool                 `json:"is_starred"`
	IsTrashed          bool                 `json:"is_trashed"`
	FromEmail          *string              `json:"from_email"`
	FromName           *string              `json:"from_name"`
	ToEmails           []string             `json:"to_emails"`
	SentAt             *string              `json:"sent_at"`
	ReceivedAt         *string              `json:"received_at"`
	Attachments        []crmEmailAttachment `json:"attachments"`
	AttachmentCount    int                  `json:"attachment_count"`
	ThreadMessageCount int64                `json:"thread_message_count"`
	CreatedAt          string               `json:"created_at"`
	UpdatedAt          string               `json:"updated_at"`
}

type CRMEmailListCountsResponse struct {
	Inbox       int64 `json:"inbox"`
	InboxUnread int64 `json:"inbox_unread"`
	Sent        int64 `json:"sent"`
	Spam        int64 `json:"spam"`
	Archived    int64 `json:"archived"`
	Starred     int64 `json:"starred"`
	Unlinked    int64 `json:"unlinked"`
	Trash       int64 `json:"trash"`
}

type crmEmailListItemRow struct {
	ID                 pgtype.UUID
	WorkspaceID        pgtype.UUID
	ThreadID           pgtype.UUID
	AccountID          pgtype.UUID
	ContactID          pgtype.UUID
	Subject            pgtype.Text
	Snippet            pgtype.Text
	Mailbox            pgtype.Text
	Folder             string
	Direction          string
	Status             string
	IsRead             bool
	IsStarred          bool
	IsTrashed          bool
	FromEmail          pgtype.Text
	FromName           pgtype.Text
	ToEmails           []string
	SentAt             pgtype.Timestamptz
	ReceivedAt         pgtype.Timestamptz
	Attachments        []byte
	AttachmentCount    int64
	ThreadMessageCount int64
	CreatedAt          pgtype.Timestamptz
	UpdatedAt          pgtype.Timestamptz
}

type crmEmailListThreadRow struct {
	ID               pgtype.UUID
	WorkspaceID      pgtype.UUID
	AccountID        pgtype.UUID
	ContactID        pgtype.UUID
	ProjectID        pgtype.UUID
	IssueID          pgtype.UUID
	Subject          pgtype.Text
	ExternalThreadID pgtype.Text
	Mailbox          pgtype.Text
	Direction        string
	Status           string
	LastMessageAt    pgtype.Timestamptz
	LastSnippet      pgtype.Text
	CreatedAt        pgtype.Timestamptz
	UpdatedAt        pgtype.Timestamptz
	MessageCount     int64
	IsRead           bool
	IsStarred        bool
	IsTrashed        bool
	IssueIDs         []string
}

type crmEmailThreadRow struct {
	ID               pgtype.UUID
	WorkspaceID      pgtype.UUID
	AccountID        pgtype.UUID
	ContactID        pgtype.UUID
	ProjectID        pgtype.UUID
	IssueID          pgtype.UUID
	IssueIDs         []pgtype.UUID
	Subject          string
	ExternalThreadID pgtype.Text
	Mailbox          pgtype.Text
	Direction        string
	Status           string
	LastMessageAt    pgtype.Timestamptz
	LastSnippet      pgtype.Text
	CreatedAt        pgtype.Timestamptz
	UpdatedAt        pgtype.Timestamptz
	MessageCount     int64
	IsRead           bool
	IsStarred        bool
	IsTrashed        bool
}

type CRMEmailMessageResponse struct {
	ID                string               `json:"id"`
	WorkspaceID       string               `json:"workspace_id"`
	ThreadID          string               `json:"thread_id"`
	AccountID         *string              `json:"account_id"`
	ContactID         *string              `json:"contact_id"`
	ExternalMessageID *string              `json:"external_message_id"`
	InReplyTo         *string              `json:"in_reply_to"`
	ReferenceIDs      []string             `json:"reference_ids"`
	Attachments       []crmEmailAttachment `json:"attachments"`
	SentAppendWarning *string              `json:"sent_append_warning"`
	RawSizeBytes      *int64               `json:"raw_size_bytes"`
	RawHeaders        map[string][]string  `json:"raw_headers"`
	FromEmail         *string              `json:"from_email"`
	FromName          *string              `json:"from_name"`
	ToEmails          []string             `json:"to_emails"`
	CcEmails          []string             `json:"cc_emails"`
	BccEmails         []string             `json:"bcc_emails"`
	Subject           *string              `json:"subject"`
	SentAt            *string              `json:"sent_at"`
	ReceivedAt        *string              `json:"received_at"`
	BodyText          *string              `json:"body_text"`
	BodyHTML          *string              `json:"body_html"`
	Snippet           *string              `json:"snippet"`
	Direction         string               `json:"direction"`
	CreatedAt         string               `json:"created_at"`
	UpdatedAt         string               `json:"updated_at"`
}

type crmEmailMessageRow struct {
	ID                pgtype.UUID
	WorkspaceID       pgtype.UUID
	ThreadID          pgtype.UUID
	AccountID         pgtype.UUID
	ContactID         pgtype.UUID
	ExternalMessageID pgtype.Text
	InReplyTo         pgtype.Text
	ReferenceIDs      []string
	Attachments       []byte
	SentAppendWarning pgtype.Text
	RawSizeBytes      pgtype.Int8
	RawHeaders        []byte
	FromEmail         pgtype.Text
	FromName          pgtype.Text
	ToEmails          []string
	CcEmails          []string
	BccEmails         []string
	Subject           pgtype.Text
	SentAt            pgtype.Timestamptz
	ReceivedAt        pgtype.Timestamptz
	BodyText          pgtype.Text
	BodyHTML          pgtype.Text
	Snippet           pgtype.Text
	Direction         string
	CreatedAt         pgtype.Timestamptz
	UpdatedAt         pgtype.Timestamptz
}

type CreateCRMEmailThreadRequest struct {
	AccountID        *string `json:"account_id"`
	ContactID        *string `json:"contact_id"`
	Subject          string  `json:"subject"`
	ExternalThreadID *string `json:"external_thread_id"`
	Mailbox          *string `json:"mailbox"`
	Direction        *string `json:"direction"`
	Status           *string `json:"status"`
	LastMessageAt    *string `json:"last_message_at"`
}

type UpdateCRMEmailThreadAssociationRequest struct {
	AccountID *string  `json:"account_id"`
	ContactID *string  `json:"contact_id"`
	ProjectID *string  `json:"project_id"`
	IssueID   *string  `json:"issue_id"`
	IssueIDs  []string `json:"issue_ids"`
}

type UpdateCRMEmailThreadStateRequest struct {
	Status    *string `json:"status"`
	IsRead    *bool   `json:"is_read"`
	IsStarred *bool   `json:"is_starred"`
	MessageID *string `json:"message_id"`
}

type CRMEmailThreadAssociationSuggestion struct {
	AccountID    string   `json:"account_id"`
	AccountName  string   `json:"account_name"`
	ContactID    *string  `json:"contact_id"`
	ContactName  *string  `json:"contact_name"`
	ContactEmail *string  `json:"contact_email"`
	Score        int      `json:"score"`
	Reasons      []string `json:"reasons"`
}

type CreateCRMEmailMessageRequest struct {
	AccountID         *string              `json:"account_id"`
	ContactID         *string              `json:"contact_id"`
	ExternalMessageID *string              `json:"external_message_id"`
	InReplyTo         *string              `json:"in_reply_to"`
	ReferenceIDs      []string             `json:"reference_ids"`
	Attachments       []crmEmailAttachment `json:"attachments"`
	FromEmail         *string              `json:"from_email"`
	FromName          *string              `json:"from_name"`
	ToEmails          []string             `json:"to_emails"`
	CcEmails          []string             `json:"cc_emails"`
	BccEmails         []string             `json:"bcc_emails"`
	Subject           *string              `json:"subject"`
	SentAt            *string              `json:"sent_at"`
	ReceivedAt        *string              `json:"received_at"`
	BodyText          *string              `json:"body_text"`
	BodyHTML          *string              `json:"body_html"`
	Snippet           *string              `json:"snippet"`
	Direction         string               `json:"direction"`
}

func uuidSliceToStrings(values []pgtype.UUID) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value.Valid {
			out = append(out, uuidToString(value))
		}
	}
	return out
}

func (h *Handler) loadCRMEmailThreadIssueIDs(ctx context.Context, threadID pgtype.UUID) []pgtype.UUID {
	rows, err := h.DB.Query(ctx, `SELECT issue_id FROM crm_email_thread_issue_link WHERE thread_id = $1 ORDER BY created_at ASC`, threadID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []pgtype.UUID
	for rows.Next() {
		var id pgtype.UUID
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func crmEmailThreadToResponse(row crmEmailThreadRow) CRMEmailThreadResponse {
	return CRMEmailThreadResponse{
		ID:               uuidToString(row.ID),
		WorkspaceID:      uuidToString(row.WorkspaceID),
		AccountID:        uuidToPtr(row.AccountID),
		ContactID:        uuidToPtr(row.ContactID),
		ProjectID:        uuidToPtr(row.ProjectID),
		IssueID:          uuidToPtr(row.IssueID),
		IssueIDs:         uuidSliceToStrings(row.IssueIDs),
		Subject:          row.Subject,
		ExternalThreadID: textToPtr(row.ExternalThreadID),
		Mailbox:          textToPtr(row.Mailbox),
		Direction:        row.Direction,
		Status:           row.Status,
		LastMessageAt:    timestampToPtr(row.LastMessageAt),
		LastSnippet:      textToPtr(row.LastSnippet),
		MessageCount:     row.MessageCount,
		IsRead:           row.IsRead,
		IsStarred:        row.IsStarred,
		IsTrashed:        row.IsTrashed,
		CreatedAt:        timestampToString(row.CreatedAt),
		UpdatedAt:        timestampToString(row.UpdatedAt),
	}
}

func crmEmailListItemToResponse(row crmEmailListItemRow) CRMEmailListItemResponse {
	toEmails := row.ToEmails
	if toEmails == nil {
		toEmails = []string{}
	}
	attachments := normalizeCRMEmailAttachments(row.Attachments)
	subject := strings.TrimSpace(crmTextValue(row.Subject))
	if subject == "" {
		subject = "(no subject)"
	}
	return CRMEmailListItemResponse{
		ID:                 uuidToString(row.ID),
		WorkspaceID:        uuidToString(row.WorkspaceID),
		ThreadID:           uuidToString(row.ThreadID),
		AccountID:          uuidToPtr(row.AccountID),
		ContactID:          uuidToPtr(row.ContactID),
		Subject:            subject,
		Snippet:            textToPtr(row.Snippet),
		Mailbox:            textToPtr(row.Mailbox),
		Folder:             row.Folder,
		Direction:          row.Direction,
		Status:             row.Status,
		IsRead:             row.IsRead,
		IsStarred:          row.IsStarred,
		IsTrashed:          row.IsTrashed,
		FromEmail:          textToPtr(row.FromEmail),
		FromName:           textToPtr(row.FromName),
		ToEmails:           toEmails,
		SentAt:             timestampToPtr(row.SentAt),
		ReceivedAt:         timestampToPtr(row.ReceivedAt),
		Attachments:        attachments,
		AttachmentCount:    int(row.AttachmentCount),
		ThreadMessageCount: row.ThreadMessageCount,
		CreatedAt:          timestampToString(row.CreatedAt),
		UpdatedAt:          timestampToString(row.UpdatedAt),
	}
}

func crmEmailListThreadToResponse(row crmEmailListThreadRow) CRMEmailThreadResponse {
	return CRMEmailThreadResponse{
		ID:               uuidToString(row.ID),
		WorkspaceID:      uuidToString(row.WorkspaceID),
		AccountID:        uuidToPtr(row.AccountID),
		ContactID:        uuidToPtr(row.ContactID),
		ProjectID:        uuidToPtr(row.ProjectID),
		IssueID:          uuidToPtr(row.IssueID),
		IssueIDs:         row.IssueIDs,
		Subject:          crmTextValue(row.Subject),
		ExternalThreadID: textToPtr(row.ExternalThreadID),
		Mailbox:          textToPtr(row.Mailbox),
		Direction:        row.Direction,
		Status:           row.Status,
		LastMessageAt:    timestampToPtr(row.LastMessageAt),
		LastSnippet:      textToPtr(row.LastSnippet),
		MessageCount:     row.MessageCount,
		IsRead:           row.IsRead,
		IsStarred:        row.IsStarred,
		IsTrashed:        row.IsTrashed,
		CreatedAt:        timestampToString(row.CreatedAt),
		UpdatedAt:        timestampToString(row.UpdatedAt),
	}
}

func normalizeCRMEmailAttachments(raw []byte) []crmEmailAttachment {
	attachments := []crmEmailAttachment{}
	if len(raw) == 0 {
		return attachments
	}
	if err := json.Unmarshal(raw, &attachments); err != nil {
		return attachments
	}
	for i := range attachments {
		attachments[i] = normalizeCRMEmailAttachment(attachments[i], i)
	}
	return attachments
}

func normalizeCRMEmailAttachment(att crmEmailAttachment, index int) crmEmailAttachment {
	att.FileName = att.DisplayName(index)
	att.LegacyName = att.FileName
	att.ContentType = cleanCRMEmailAttachmentContentType(att.ContentType)
	att.Size = att.DisplaySize()
	att.LegacySize = att.Size
	if strings.TrimSpace(att.Content) != "" {
		att.Content = normalizeCRMEmailAttachmentContent(att.Content)
	}
	return att
}

func normalizeCRMEmailAttachmentContent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "data:") {
		if comma := strings.Index(value, ","); comma >= 0 {
			return strings.TrimSpace(value[comma+1:])
		}
	}
	return value
}

func crmEmailMessageToResponse(row crmEmailMessageRow) CRMEmailMessageResponse {
	toEmails := row.ToEmails
	if toEmails == nil {
		toEmails = []string{}
	}
	ccEmails := row.CcEmails
	if ccEmails == nil {
		ccEmails = []string{}
	}
	bccEmails := row.BccEmails
	if bccEmails == nil {
		bccEmails = []string{}
	}
	referenceIDs := row.ReferenceIDs
	if referenceIDs == nil {
		referenceIDs = []string{}
	}
	attachments := normalizeCRMEmailAttachments(row.Attachments)
	rawHeaders := map[string][]string{}
	if len(row.RawHeaders) > 0 {
		_ = json.Unmarshal(row.RawHeaders, &rawHeaders)
	}
	var rawSizeBytes *int64
	if row.RawSizeBytes.Valid {
		value := row.RawSizeBytes.Int64
		rawSizeBytes = &value
	}
	return CRMEmailMessageResponse{
		ID:                uuidToString(row.ID),
		WorkspaceID:       uuidToString(row.WorkspaceID),
		ThreadID:          uuidToString(row.ThreadID),
		AccountID:         uuidToPtr(row.AccountID),
		ContactID:         uuidToPtr(row.ContactID),
		ExternalMessageID: textToPtr(row.ExternalMessageID),
		InReplyTo:         textToPtr(row.InReplyTo),
		ReferenceIDs:      referenceIDs,
		Attachments:       attachments,
		SentAppendWarning: textToPtr(row.SentAppendWarning),
		RawSizeBytes:      rawSizeBytes,
		RawHeaders:        rawHeaders,
		FromEmail:         textToPtr(row.FromEmail),
		FromName:          textToPtr(row.FromName),
		ToEmails:          toEmails,
		CcEmails:          ccEmails,
		BccEmails:         bccEmails,
		Subject:           textToPtr(row.Subject),
		SentAt:            timestampToPtr(row.SentAt),
		ReceivedAt:        timestampToPtr(row.ReceivedAt),
		BodyText:          textToPtr(row.BodyText),
		BodyHTML:          textToPtr(row.BodyHTML),
		Snippet:           textToPtr(row.Snippet),
		Direction:         row.Direction,
		CreatedAt:         timestampToString(row.CreatedAt),
		UpdatedAt:         timestampToString(row.UpdatedAt),
	}
}

type CRMAccountProfileResponse struct {
	ID            string          `json:"id"`
	WorkspaceID   string          `json:"workspace_id"`
	AccountID     string          `json:"account_id"`
	Summary       *string         `json:"summary"`
	ProfileJSON   json.RawMessage `json:"profile_json"`
	SourceSummary *string         `json:"source_summary"`
	UpdatedBy     *string         `json:"updated_by"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
}

func buildCRMProfileSourceSummary(noteSnippets, projectTitles, issueTitles []string) *string {
	parts := make([]string, 0, 3)
	if len(noteSnippets) > 0 {
		parts = append(parts, "notes: "+strings.Join(noteSnippets, " | "))
	}
	if len(projectTitles) > 0 {
		parts = append(parts, "projects: "+strings.Join(projectTitles, ", "))
	}
	if len(issueTitles) > 0 {
		parts = append(parts, "issues: "+strings.Join(issueTitles, ", "))
	}
	if len(parts) == 0 {
		return nil
	}
	summary := strings.Join(parts, " ; ")
	return &summary
}

func trimCRMProfileList(items []string, limit, snippetLimit int) []string {
	out := make([]string, 0, min(len(items), limit))
	for i := 0; i < len(items) && i < limit; i++ {
		item := trimCRMProfileSnippet(items[i], snippetLimit)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (h *Handler) resolveCRMProfileAgentLLMConfig(ctx context.Context) (string, string, string, string) {
	var model pgtype.Text
	var runtimeConfig, customEnv []byte
	if err := h.DB.QueryRow(ctx, `
		SELECT a.model, a.runtime_config, a.custom_env
		FROM agent a
		JOIN agent_runtime r ON r.id = a.runtime_id
		WHERE lower(a.name) = 'jarvis'
		  AND r.provider = 'hermes'
		ORDER BY CASE WHEN r.status = 'online' THEN 0 ELSE 1 END, a.updated_at DESC
		LIMIT 1
	`).Scan(&model, &runtimeConfig, &customEnv); err == nil {
		config := map[string]any{}
		_ = json.Unmarshal(runtimeConfig, &config)
		env := map[string]any{}
		_ = json.Unmarshal(customEnv, &env)
		baseURL := firstNonEmpty(stringValue(config["base_url"]), stringValue(config["baseURL"]), stringValue(env["HERMES_MODEL_BASE_URL"]), os.Getenv("HERMES_MODEL_BASE_URL"))
		apiKey := firstNonEmpty(stringValue(config["api_key"]), stringValue(config["apiKey"]), stringValue(env["HERMES_MODEL_API_KEY"]), os.Getenv("HERMES_MODEL_API_KEY"))
		modelName := firstNonEmpty(textValue(model), stringValue(config["model"]), stringValue(env["HERMES_MODEL"]), os.Getenv("HERMES_MODEL"))
		if baseURL != "" && apiKey != "" && modelName != "" {
			return baseURL, apiKey, modelName, "agent:Jarvis"
		}
	}
	return firstNonEmpty(os.Getenv("CRM_PROFILE_LLM_BASE_URL"), os.Getenv("HERMES_MODEL_BASE_URL")), firstNonEmpty(os.Getenv("CRM_PROFILE_LLM_API_KEY"), os.Getenv("HERMES_MODEL_API_KEY")), firstNonEmpty(os.Getenv("CRM_PROFILE_LLM_MODEL"), os.Getenv("HERMES_MODEL")), "env:fallback"
}

func stringValue(value any) string {
	s, _ := value.(string)
	return strings.TrimSpace(s)
}

func textValue(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}

func trimCRMProfileSnippet(s string, limit int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) <= limit {
		return s
	}
	return strings.TrimSpace(s[:limit])
}

func profileValueFromEvidence(emailSnippets, noteSnippets []string, label, fallback string) string {
	items := append([]string{}, emailSnippets...)
	items = append(items, noteSnippets...)
	matches := make([]string, 0, 3)
	for _, item := range items {
		clean := trimCRMProfileSnippet(strings.Join(strings.Fields(item), " "), 180)
		if clean == "" {
			continue
		}
		matches = append(matches, clean)
		if len(matches) >= 3 {
			break
		}
	}
	if len(matches) == 0 {
		return "——"
	}
	return label + "：" + strings.Join(matches, "；")
}

func summarizeCRMProfileEvidence(emailSnippets, noteSnippets []string) string {
	items := append([]string{}, emailSnippets...)
	items = append(items, noteSnippets...)
	trimmed := trimCRMProfileList(items, 3, 160)
	if len(trimmed) == 0 {
		return ""
	}
	return "往来记录摘要：" + strings.Join(trimmed, "；")
}

func cleanCRMProfileValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "——"
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '；' || r == ';' || r == '\n' || r == '\r'
	})
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "，,。.:：-— ")
		if part == "" {
			continue
		}
		if strings.Contains(part, "——") || isVagueCRMProfileFragment(part) {
			continue
		}
		kept = append(kept, part)
	}
	if len(kept) == 0 {
		return "——"
	}
	return strings.Join(kept, "；")
}

func isVagueCRMProfileFragment(part string) bool {
	part = strings.TrimSpace(part)
	return strings.HasPrefix(part, "具体") && len([]rune(part)) <= 12
}

func cleanCRMProfile(profile map[string]any) map[string]any {
	for _, key := range []string{"customer_summary", "business_model", "main_products", "procurement_needs", "pain_points", "decision_process", "communication_preference", "recent_progress", "risk_notes", "cooperation_history", "next_step_suggestions"} {
		if value, ok := profile[key].(string); ok {
			profile[key] = cleanCRMProfileValue(value)
		}
	}
	return profile
}

func buildCRMProfileNextSteps(mainProducts, procurementNeeds, decisionProcess string) string {
	steps := []string{}
	if mainProducts == "——" {
		steps = append(steps, "确认客户关注/采购产品")
	}
	if procurementNeeds == "——" {
		steps = append(steps, "补齐数量、目标价、交期、认证和物流要求")
	}
	if decisionProcess == "——" {
		steps = append(steps, "确认决策人和采购流程")
	}
	if len(steps) == 0 {
		steps = append(steps, "基于已确认需求推进报价、样品或下一次跟进")
	}
	return strings.Join(steps, "；")
}

func (h *Handler) generateCRMAccountProfileWithLLM(ctx context.Context, accountName, fallbackSummary, projectSource, issueSource, emailEvidence, noteEvidence, notes string) (map[string]any, error) {
	baseURL, apiKey, model, source := h.resolveCRMProfileAgentLLMConfig(ctx)
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)
	if baseURL == "" || apiKey == "" || model == "" {
		return nil, nil
	}
	prompt := "你是外贸CRM客户画像分析助手。请根据客户资料、邮件往来、项目、issue、备注，总结并填写JSON。只输出JSON，不要Markdown。字段必须包含：customer_summary,business_model,main_products,procurement_needs,pain_points,decision_process,communication_preference,recent_progress,risk_notes,cooperation_history,next_step_suggestions。要求：1) 用中文；2) 基于证据总结，不要直接堆原文；3) 未明确的字段只写“——”，不要写解释；4) 不要输出“具体业务模式——”“具体产品——”这类半句；有证据的部分单独成句，未知部分直接省略；5) 每个已明确字段写可执行、具体内容。\n\n" +
		"客户名：" + accountName + "\n基础摘要：" + fallbackSummary + "\n项目：" + projectSource + "\nIssue：" + issueSource + "\n邮件往来：" + emailEvidence + "\n沟通备注：" + noteEvidence + "\n客户备注：" + notes
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You generate strict JSON CRM customer profiles."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("CRM profile LLM HTTP status: " + resp.Status)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Choices) == 0 {
		return nil, errors.New("CRM profile LLM returned no choices")
	}
	content := strings.TrimSpace(out.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	var profile map[string]any
	if err := json.Unmarshal([]byte(content), &profile); err != nil {
		return nil, err
	}
	profile["generated_by"] = source
	return cleanCRMProfile(profile), nil
}

type UpsertCRMAccountProfileRequest struct {
	Summary     *string         `json:"summary"`
	ProfileJSON json.RawMessage `json:"profile_json"`
}

type CRMIMAPSettingResponse struct {
	ID              string  `json:"id"`
	WorkspaceID     string  `json:"workspace_id"`
	Label           string  `json:"label"`
	Email           string  `json:"email"`
	Host            string  `json:"host"`
	Port            int32   `json:"port"`
	TLSMode         string  `json:"tls_mode"`
	Username        string  `json:"username"`
	SecretRef       *string `json:"secret_ref"`
	SyncEnabled     bool    `json:"sync_enabled"`
	LastTestStatus  *string `json:"last_test_status"`
	LastTestMessage *string `json:"last_test_message"`
	LastTestedAt    *string `json:"last_tested_at"`
	OwnerType       *string `json:"owner_type"`
	OwnerID         *string `json:"owner_id"`
	SMTPHost        *string `json:"smtp_host"`
	SMTPPort        *int32  `json:"smtp_port"`
	SMTPTLSMode     *string `json:"smtp_tls_mode"`
	SMTPUsername    *string `json:"smtp_username"`
	SMTPSecretRef   *string `json:"smtp_secret_ref"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type UpsertCRMIMAPSettingRequest struct {
	ID            *string `json:"id"`
	Label         string  `json:"label"`
	Email         string  `json:"email"`
	Host          string  `json:"host"`
	Port          int32   `json:"port"`
	TLSMode       string  `json:"tls_mode"`
	Username      string  `json:"username"`
	SecretRef     *string `json:"secret_ref"`
	Secret        *string `json:"secret"`
	SyncEnabled   bool    `json:"sync_enabled"`
	OwnerType     *string `json:"owner_type"`
	OwnerID       *string `json:"owner_id"`
	SMTPHost      *string `json:"smtp_host"`
	SMTPPort      *int32  `json:"smtp_port"`
	SMTPTLSMode   *string `json:"smtp_tls_mode"`
	SMTPUsername  *string `json:"smtp_username"`
	SMTPSecretRef *string `json:"smtp_secret_ref"`
	SMTPSecret    *string `json:"smtp_secret"`
}

type CRMIMAPSyncRunResponse struct {
	ID             string  `json:"id"`
	MailboxID      *string `json:"mailbox_id"`
	MailboxEmail   *string `json:"mailbox_email"`
	Folder         string  `json:"folder"`
	Status         string  `json:"status"`
	RequestedLimit int32   `json:"requested_limit"`
	FetchedCount   int32   `json:"fetched_count"`
	ImportedCount  int32   `json:"imported_count"`
	SkippedCount   int32   `json:"skipped_count"`
	ErrorMessage   *string `json:"error_message"`
	StartedAt      string  `json:"started_at"`
	FinishedAt     *string `json:"finished_at"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type CRMIMAPPreviewRequest struct {
	MailboxID *string `json:"mailbox_id"`
	Folder    *string `json:"folder"`
	Limit     int     `json:"limit"`
	RangeDays int     `json:"range_days"`
}

type CRMIMAPImportRequest struct {
	MailboxID *string  `json:"mailbox_id"`
	Folder    *string  `json:"folder"`
	UIDs      []string `json:"uids"`
	Limit     int      `json:"limit"`
	RangeDays int      `json:"range_days"`
}

type CRMIMAPPreviewMessageResponse struct {
	UID               string   `json:"uid"`
	ExternalMessageID string   `json:"external_message_id"`
	Subject           string   `json:"subject"`
	FromEmail         string   `json:"from_email"`
	FromName          string   `json:"from_name"`
	ToEmails          []string `json:"to_emails"`
	CcEmails          []string `json:"cc_emails"`
	ReceivedAt        *string  `json:"received_at"`
	Snippet           string   `json:"snippet"`
	RawSize           int      `json:"raw_size"`
}

type CRMEmailDraftResponse struct {
	ID          string   `json:"id"`
	MailboxID   *string  `json:"mailbox_id"`
	ThreadID    *string  `json:"thread_id"`
	AccountID   *string  `json:"account_id"`
	ContactID   *string  `json:"contact_id"`
	ToEmails    []string `json:"to_emails"`
	CcEmails    []string `json:"cc_emails"`
	BccEmails   []string `json:"bcc_emails"`
	Subject     string   `json:"subject"`
	BodyText    string   `json:"body_text"`
	Status      string   `json:"status"`
	AIGenerated bool     `json:"ai_generated"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type CreateCRMEmailDraftRequest struct {
	MailboxID         *string              `json:"mailbox_id"`
	ThreadID          *string              `json:"thread_id"`
	AccountID         *string              `json:"account_id"`
	ContactID         *string              `json:"contact_id"`
	IssueID           *string              `json:"issue_id"`
	ApprovalReason    string               `json:"approval_reason"`
	ToEmails          []string             `json:"to_emails"`
	CcEmails          []string             `json:"cc_emails"`
	BccEmails         []string             `json:"bcc_emails"`
	Subject           string               `json:"subject"`
	BodyText          string               `json:"body_text"`
	BodyHTML          string               `json:"body_html"`
	InReplyTo         string               `json:"in_reply_to"`
	ReferenceIDs      []string             `json:"reference_ids"`
	Attachments       []crmEmailAttachment `json:"attachments"`
	SentAppendEnabled *bool                `json:"sent_append_enabled"`
	AIGenerated       bool                 `json:"ai_generated"`
}
type CRMProfileSuggestionResponse struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	AccountID   string          `json:"account_id"`
	Summary     *string         `json:"summary"`
	ProfileJSON json.RawMessage `json:"profile_json"`
	SourceCount int32           `json:"source_count"`
	Status      string          `json:"status"`
	CreatedAt   string          `json:"created_at"`
	AppliedAt   *string         `json:"applied_at"`
}

type CRMCommunicationNoteResponse struct {
	ID          string  `json:"id"`
	WorkspaceID string  `json:"workspace_id"`
	AccountID   *string `json:"account_id"`
	ContactID   *string `json:"contact_id"`
	Channel     string  `json:"channel"`
	Direction   string  `json:"direction"`
	OccurredAt  string  `json:"occurred_at"`
	Subject     *string `json:"subject"`
	Body        string  `json:"body"`
	CreatedBy   *string `json:"created_by"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type CreateCRMCommunicationNoteRequest struct {
	ContactID  *string `json:"contact_id"`
	Channel    *string `json:"channel"`
	Direction  *string `json:"direction"`
	OccurredAt *string `json:"occurred_at"`
	Subject    *string `json:"subject"`
	Body       string  `json:"body"`
}

type LinkCRMAccountProjectRequest struct {
	ProjectID  *string  `json:"project_id"`
	ProjectIDs []string `json:"project_ids"`
	Label      *string  `json:"label"`
}

type CreateCRMFollowUpIssueRequest struct {
	ProjectID    *string `json:"project_id"`
	Title        string  `json:"title"`
	Description  *string `json:"description"`
	Priority     *string `json:"priority"`
	AssigneeType *string `json:"assignee_type"`
	AssigneeID   *string `json:"assignee_id"`
	DueDate      *string `json:"due_date"`
}

type CRMFollowUpIssueResponse struct {
	Issue IssueResponse `json:"issue"`
}

func normalizeCRMName(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func normalizedCRMKey(s string) string {
	return strings.ToLowerSpecial(unicode.TurkishCase, normalizeCRMName(s))
}

func cleanStringForDB(value string) string {
	return strings.ToValidUTF8(value, "�")
}

func cleanOptionalText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	v := strings.TrimSpace(cleanStringForDB(*s))
	if v == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}

func cleanStatus(s *string) string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return "active"
	}
	return strings.TrimSpace(*s)
}

func cleanOptionalStringList(values []string) []string {
	cleaned := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		v := strings.TrimSpace(cleanStringForDB(value))
		if v == "" {
			continue
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, v)
	}
	return cleaned
}

func cleanDefault(s *string, fallback string) string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return fallback
	}
	return strings.TrimSpace(*s)
}

func cleanCountryCodeOrName(code, name *string) (pgtype.Text, pgtype.Text) {
	cleanCode := cleanOptionalText(code)
	cleanName := cleanOptionalText(name)
	if !cleanName.Valid && cleanCode.Valid {
		cleanName = cleanCode
	}
	if !cleanCode.Valid && cleanName.Valid {
		cleanCode = cleanName
	}
	return cleanCode, cleanName
}

func firstString(values ...*string) *string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			return value
		}
	}
	return nil
}

func validCRMAccountStatus(value string) bool {
	switch value {
	case "active", "inactive", "prospect", "archived":
		return true
	default:
		return false
	}
}

func validCRMAccountRating(value string) bool {
	switch value {
	case "hot", "warm", "cold", "unknown":
		return true
	default:
		return false
	}
}

func validCRMAccountPriority(value string) bool {
	switch value {
	case "high", "medium", "low":
		return true
	default:
		return false
	}
}

func validCRMAccountSource(value string) bool {
	switch value {
	case "manual", "email", "whatsapp", "website", "referral", "trade_show", "linkedin", "other":
		return true
	default:
		return false
	}
}

func validCRMFollowUpBucket(value string) bool {
	switch value {
	case "today", "next_7_days", "overdue", "none":
		return true
	default:
		return false
	}
}

func validCRMAccountSort(value string) bool {
	switch value {
	case "updated", "name", "next_follow_up", "priority_rating":
		return true
	default:
		return false
	}
}

func optionalUUID(w http.ResponseWriter, value *string, fieldName string) (pgtype.UUID, bool) {
	var zero pgtype.UUID
	if value == nil || strings.TrimSpace(*value) == "" {
		return zero, true
	}
	parsed, ok := parseUUIDOrBadRequest(w, strings.TrimSpace(*value), fieldName)
	if !ok {
		return zero, false
	}
	return parsed, true
}

func cleanNoteChannel(value *string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "manual"
	}
	return strings.TrimSpace(*value)
}

func validCRMCommunicationChannel(value string) bool {
	switch value {
	case "manual", "email", "whatsapp", "phone", "meeting", "other":
		return true
	default:
		return false
	}
}

func cleanNoteDirection(value *string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "outbound"
	}
	return strings.TrimSpace(*value)
}

func validCRMCommunicationDirection(value string) bool {
	switch value {
	case "inbound", "outbound", "internal":
		return true
	default:
		return false
	}
}

func cleanOptionalTimestamp(w http.ResponseWriter, s *string, fieldName string) (pgtype.Timestamptz, bool) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return pgtype.Timestamptz{}, true
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*s))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+fieldName+" format, expected RFC3339")
		return pgtype.Timestamptz{}, false
	}
	return pgtype.Timestamptz{Time: parsed, Valid: true}, true
}

func cleanOptionalBool(v *bool) bool {
	return v != nil && *v
}

func (h *Handler) crmWorkspaceUUID(w http.ResponseWriter, r *http.Request) (pgtype.UUID, bool) {
	return parseUUIDOrBadRequest(w, h.resolveWorkspaceID(r), "workspace id")
}

func (h *Handler) scanCRMAccount(row pgx.Row) (crmAccountRow, error) {
	var account crmAccountRow
	err := row.Scan(
		&account.ID, &account.WorkspaceID, &account.Name, &account.NormalizedName,
		&account.AccountCode, &account.AccountType, &account.Website, &account.Country,
		&account.CountryCode, &account.CountryName, &account.Region, &account.City,
		&account.Industry, &account.SubIndustry, &account.Status, &account.OwnerID,
		&account.OwnerMemberID, &account.OwnerType, &account.OwnerAgentID, &account.Source, &account.Rating, &account.Priority,
		&account.AnnualRevenue, &account.EmployeeCount, &account.Tags, &account.Notes,
		&account.LastContactedAt, &account.NextFollowUpAt, &account.CreatedAt, &account.UpdatedAt,
		&account.ContactCount,
	)
	return account, err
}

func (h *Handler) getCRMAccount(w http.ResponseWriter, r *http.Request, accountID pgtype.UUID, workspaceID pgtype.UUID) (crmAccountRow, bool) {
	row, err := h.scanCRMAccount(h.DB.QueryRow(r.Context(), `
		SELECT a.id, a.workspace_id, a.name, a.normalized_name, a.account_code, a.account_type,
		       a.website, a.country, a.country_code, a.country_name, a.region, a.city,
		       a.industry, a.sub_industry, a.status, a.owner_id, a.owner_member_id,
		       COALESCE(a.owner_type, 'member'), a.owner_agent_id, a.source, a.rating, a.priority, a.annual_revenue, a.employee_count,
		       a.tags, a.notes, a.last_contacted_at, a.next_follow_up_at,
		       a.created_at, a.updated_at, COUNT(c.id)::bigint AS contact_count
		FROM crm_account a
		LEFT JOIN crm_contact c ON c.account_id = a.id AND c.workspace_id = a.workspace_id
		WHERE a.id = $1 AND a.workspace_id = $2
		GROUP BY a.id
	`, accountID, workspaceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "CRM account not found")
			return crmAccountRow{}, false
		}
		writeError(w, http.StatusInternalServerError, "failed to load CRM account")
		return crmAccountRow{}, false
	}
	return row, true
}

func (h *Handler) CreateCRMAccount(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	var req CreateCRMAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := normalizeCRMName(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	status := cleanStatus(req.Status)
	if !validCRMAccountStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid account status")
		return
	}
	ownerID, ok := optionalUUID(w, req.OwnerID, "owner_id")
	if !ok {
		return
	}
	ownerMemberID, ok := optionalUUID(w, req.OwnerMemberID, "owner_member_id")
	if !ok {
		return
	}
	ownerAgentID, ok := optionalUUID(w, req.OwnerAgentID, "owner_agent_id")
	if !ok {
		return
	}
	ownerType := ""
	if req.OwnerType != nil {
		ownerType = strings.TrimSpace(*req.OwnerType)
	}
	if ownerType == "" {
		if ownerAgentID.Valid {
			ownerType = "agent"
		} else {
			ownerType = "member"
		}
	}
	if ownerType != "member" && ownerType != "agent" {
		writeError(w, http.StatusBadRequest, "invalid owner_type")
		return
	}
	if ownerType == "member" {
		ownerAgentID = pgtype.UUID{}
	} else {
		ownerMemberID = pgtype.UUID{}
	}
	lastContactedAt, ok := cleanOptionalTimestamp(w, req.LastContactedAt, "last_contacted_at")
	if !ok {
		return
	}
	nextFollowUpAt, ok := cleanOptionalTimestamp(w, req.NextFollowUpAt, "next_follow_up_at")
	if !ok {
		return
	}
	countryCode, countryName := cleanCountryCodeOrName(req.CountryCode, firstString(req.CountryName, req.Country))
	row, err := h.scanCRMAccount(h.DB.QueryRow(r.Context(), `
		INSERT INTO crm_account (
			workspace_id, name, normalized_name, account_code, account_type, website, country,
			country_code, country_name, region, city, industry, sub_industry, status, owner_id,
			owner_member_id, owner_type, owner_agent_id, source, rating, priority, annual_revenue, employee_count, tags,
			notes, last_contacted_at, next_follow_up_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		        $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
		        $21, $22, $23, $24, $25, $26, $27)
		RETURNING id, workspace_id, name, normalized_name, account_code, account_type,
		          website, country, country_code, country_name, region, city,
		          industry, sub_industry, status, owner_id, owner_member_id,
		          owner_type, owner_agent_id, source, rating, priority, annual_revenue, employee_count,
		          tags, notes, last_contacted_at, next_follow_up_at,
		          created_at, updated_at, 0::bigint
	`, workspaceID, name, normalizedCRMKey(name), cleanOptionalText(req.AccountCode), cleanDefault(req.AccountType, "prospect"),
		cleanOptionalText(req.Website), countryName, countryCode, countryName,
		cleanOptionalText(req.Region), cleanOptionalText(req.City), cleanOptionalText(req.Industry), cleanOptionalText(req.SubIndustry), status,
		ownerID, ownerMemberID, ownerType, ownerAgentID, cleanOptionalText(req.Source), cleanDefault(req.Rating, "unknown"), cleanDefault(req.Priority, "medium"),
		cleanOptionalText(req.AnnualRevenue), cleanOptionalText(req.EmployeeCount), cleanOptionalStringList(req.Tags), cleanOptionalText(req.Notes),
		lastContactedAt, nextFollowUpAt))
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "CRM account name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create CRM account")
		return
	}
	writeJSON(w, http.StatusCreated, crmAccountToResponse(row))
}

func (h *Handler) ListCRMAccounts(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	search := strings.TrimSpace(query.Get("search"))
	status := strings.TrimSpace(query.Get("status"))
	rating := strings.TrimSpace(query.Get("rating"))
	priority := strings.TrimSpace(query.Get("priority"))
	countryCode := strings.TrimSpace(query.Get("country_code"))
	industry := strings.TrimSpace(query.Get("industry"))
	source := strings.TrimSpace(query.Get("source"))
	followUpBucket := strings.TrimSpace(query.Get("follow_up_bucket"))
	sort := strings.TrimSpace(query.Get("sort"))
	if sort == "" {
		sort = "updated"
	}
	if status != "" && !validCRMAccountStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid account status")
		return
	}
	if rating != "" && !validCRMAccountRating(rating) {
		writeError(w, http.StatusBadRequest, "invalid account rating")
		return
	}
	if priority != "" && !validCRMAccountPriority(priority) {
		writeError(w, http.StatusBadRequest, "invalid account priority")
		return
	}
	if source != "" && !validCRMAccountSource(source) {
		writeError(w, http.StatusBadRequest, "invalid account source")
		return
	}
	if followUpBucket != "" && !validCRMFollowUpBucket(followUpBucket) {
		writeError(w, http.StatusBadRequest, "invalid follow up bucket")
		return
	}
	if !validCRMAccountSort(sort) {
		writeError(w, http.StatusBadRequest, "invalid account sort")
		return
	}
	var searchArg pgtype.Text
	if search != "" {
		searchArg = pgtype.Text{String: normalizedCRMKey(search), Valid: true}
	}
	textArg := func(value string) pgtype.Text {
		if value == "" {
			return pgtype.Text{}
		}
		return pgtype.Text{String: value, Valid: true}
	}
	orderBy := "a.updated_at DESC, a.created_at DESC"
	switch sort {
	case "name":
		orderBy = "a.normalized_name ASC, a.name ASC"
	case "next_follow_up":
		orderBy = "a.next_follow_up_at ASC NULLS LAST, a.updated_at DESC"
	case "priority_rating":
		orderBy = "CASE a.priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END ASC, CASE a.rating WHEN 'hot' THEN 1 WHEN 'warm' THEN 2 WHEN 'cold' THEN 3 ELSE 4 END ASC, a.updated_at DESC"
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT a.id, a.workspace_id, a.name, a.normalized_name, a.account_code, a.account_type,
		       a.website, a.country, a.country_code, a.country_name, a.region, a.city,
		       a.industry, a.sub_industry, a.status, a.owner_id, a.owner_member_id,
		       COALESCE(a.owner_type, 'member'), a.owner_agent_id, a.source, a.rating, a.priority, a.annual_revenue, a.employee_count,
		       a.tags, a.notes, a.last_contacted_at, a.next_follow_up_at,
		       a.created_at, a.updated_at, COUNT(c.id)::bigint AS contact_count
		FROM crm_account a
		LEFT JOIN crm_contact c ON c.account_id = a.id AND c.workspace_id = a.workspace_id
		WHERE a.workspace_id = $1
		  AND ($2::text IS NULL OR a.status = $2::text)
		  AND ($3::text IS NULL OR a.normalized_name LIKE '%' || $3::text || '%')
		  AND ($4::text IS NULL OR a.rating = $4::text)
		  AND ($5::text IS NULL OR a.priority = $5::text)
		  AND ($6::text IS NULL OR a.country_code = $6::text OR a.country = $6::text)
		  AND ($7::text IS NULL OR a.industry = $7::text)
		  AND ($8::text IS NULL OR a.source = $8::text)
		  AND (
		    $9::text IS NULL
		    OR ($9::text = 'today' AND a.next_follow_up_at >= date_trunc('day', now()) AND a.next_follow_up_at < date_trunc('day', now()) + interval '1 day')
		    OR ($9::text = 'next_7_days' AND a.next_follow_up_at >= now() AND a.next_follow_up_at < now() + interval '7 days')
		    OR ($9::text = 'overdue' AND a.next_follow_up_at < now())
		    OR ($9::text = 'none' AND a.next_follow_up_at IS NULL)
		  )
		GROUP BY a.id
		ORDER BY `+orderBy+`
		LIMIT 100
	`, workspaceID, textArg(status), searchArg, textArg(rating), textArg(priority), textArg(countryCode), textArg(industry), textArg(source), textArg(followUpBucket))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list CRM accounts")
		return
	}
	defer rows.Close()
	accounts := []CRMAccountResponse{}
	for rows.Next() {
		account, err := h.scanCRMAccount(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan CRM account")
			return
		}
		accounts = append(accounts, crmAccountToResponse(account))
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": accounts, "total": len(accounts)})
}

func (h *Handler) GetCRMAccount(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	account, ok := h.getCRMAccount(w, r, accountID, workspaceID)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, crmAccountToResponse(account))
}

func (h *Handler) UpdateCRMAccount(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	if _, ok := h.getCRMAccount(w, r, accountID, workspaceID); !ok {
		return
	}
	var req UpdateCRMAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := normalizeCRMName(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	status := cleanStatus(req.Status)
	if !validCRMAccountStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid account status")
		return
	}
	ownerID, ok := optionalUUID(w, req.OwnerID, "owner_id")
	if !ok {
		return
	}
	ownerMemberID, ok := optionalUUID(w, req.OwnerMemberID, "owner_member_id")
	if !ok {
		return
	}
	ownerAgentID, ok := optionalUUID(w, req.OwnerAgentID, "owner_agent_id")
	if !ok {
		return
	}
	ownerType := ""
	if req.OwnerType != nil {
		ownerType = strings.TrimSpace(*req.OwnerType)
	}
	if ownerType == "" {
		if ownerAgentID.Valid {
			ownerType = "agent"
		} else {
			ownerType = "member"
		}
	}
	if ownerType != "member" && ownerType != "agent" {
		writeError(w, http.StatusBadRequest, "invalid owner_type")
		return
	}
	if ownerType == "member" {
		ownerAgentID = pgtype.UUID{}
	} else {
		ownerMemberID = pgtype.UUID{}
	}
	lastContactedAt, ok := cleanOptionalTimestamp(w, req.LastContactedAt, "last_contacted_at")
	if !ok {
		return
	}
	nextFollowUpAt, ok := cleanOptionalTimestamp(w, req.NextFollowUpAt, "next_follow_up_at")
	if !ok {
		return
	}
	countryCode, countryName := cleanCountryCodeOrName(req.CountryCode, firstString(req.CountryName, req.Country))
	row, err := h.scanCRMAccount(h.DB.QueryRow(r.Context(), `
		UPDATE crm_account SET
			name = $3,
			normalized_name = $4,
			account_code = $5,
			account_type = $6,
			website = $7,
			country = $8,
			country_code = $9,
			country_name = $10,
			region = $11,
			city = $12,
			industry = $13,
			sub_industry = $14,
			status = $15,
			owner_id = $16,
			owner_member_id = $17,
			owner_type = $18,
			owner_agent_id = $19,
			source = $20,
			rating = $21,
			priority = $22,
			annual_revenue = $23,
			employee_count = $24,
			tags = $25,
			notes = $26,
			last_contacted_at = $27,
			next_follow_up_at = $28,
			updated_at = now()
		WHERE id = $1 AND workspace_id = $2
		RETURNING id, workspace_id, name, normalized_name, account_code, account_type,
		          website, country, country_code, country_name, region, city,
		          industry, sub_industry, status, owner_id, owner_member_id,
		          owner_type, owner_agent_id, source, rating, priority, annual_revenue, employee_count,
		          tags, notes, last_contacted_at, next_follow_up_at,
		          created_at, updated_at,
		          (SELECT COUNT(*)::bigint FROM crm_contact c WHERE c.account_id = crm_account.id AND c.workspace_id = crm_account.workspace_id)
	`, accountID, workspaceID, name, normalizedCRMKey(name), cleanOptionalText(req.AccountCode), cleanDefault(req.AccountType, "prospect"),
		cleanOptionalText(req.Website), countryName, countryCode, countryName, cleanOptionalText(req.Region), cleanOptionalText(req.City),
		cleanOptionalText(req.Industry), cleanOptionalText(req.SubIndustry), status, ownerID, ownerMemberID, ownerType, ownerAgentID, cleanOptionalText(req.Source),
		cleanDefault(req.Rating, "unknown"), cleanDefault(req.Priority, "medium"), cleanOptionalText(req.AnnualRevenue), cleanOptionalText(req.EmployeeCount),
		cleanOptionalStringList(req.Tags), cleanOptionalText(req.Notes), lastContactedAt, nextFollowUpAt))
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "CRM account name or code already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update CRM account")
		return
	}
	_, _ = h.regenerateCRMAccountProfile(r.Context(), workspaceID, accountID)
	writeJSON(w, http.StatusOK, crmAccountToResponse(row))
}

func (h *Handler) DeleteCRMAccount(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	if _, ok := h.getCRMAccount(w, r, accountID, workspaceID); !ok {
		return
	}
	if _, err := h.DB.Exec(r.Context(), `DELETE FROM crm_account WHERE id = $1 AND workspace_id = $2`, accountID, workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete CRM account")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateCRMContact(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	if _, ok := h.getCRMAccount(w, r, accountID, workspaceID); !ok {
		return
	}
	var req CreateCRMContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := normalizeCRMName(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	lastContactedAt, ok := cleanOptionalTimestamp(w, req.LastContactedAt, "last_contacted_at")
	if !ok {
		return
	}
	contact, err := h.scanCRMContact(h.DB.QueryRow(r.Context(), `
		INSERT INTO crm_contact (
			workspace_id, account_id, name, salutation, email, phone, mobile, whatsapp_id,
			whatsapp, wechat, linkedin_url, role_title, job_title, department, role,
			language, preferred_language, timezone, is_primary, decision_role, notes, last_contacted_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		        $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING id, workspace_id, account_id, name, salutation, email, phone, mobile,
		          whatsapp_id, whatsapp, wechat, linkedin_url, role_title, job_title,
		          department, role, language, preferred_language, timezone, is_primary,
		          decision_role, notes, last_contacted_at, created_at, updated_at
	`, workspaceID, accountID, name, cleanOptionalText(req.Salutation), cleanOptionalText(req.Email), cleanOptionalText(req.Phone),
		cleanOptionalText(req.Mobile), cleanOptionalText(req.WhatsappID), cleanOptionalText(req.Whatsapp), cleanOptionalText(req.Wechat),
		cleanOptionalText(req.LinkedinURL), cleanOptionalText(req.RoleTitle), cleanOptionalText(req.JobTitle), cleanOptionalText(req.Department),
		cleanOptionalText(req.Role), cleanOptionalText(req.Language), cleanOptionalText(req.PreferredLanguage), cleanOptionalText(req.Timezone),
		cleanOptionalBool(req.IsPrimary), cleanOptionalText(req.DecisionRole), cleanOptionalText(req.Notes), lastContactedAt))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create CRM contact")
		return
	}
	writeJSON(w, http.StatusCreated, crmContactToResponse(contact))
}

func (h *Handler) ListCRMContacts(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT id, workspace_id, account_id, name, salutation, email, phone, mobile,
		       whatsapp_id, whatsapp, wechat, linkedin_url, role_title, job_title,
		       department, role, language, preferred_language, timezone, is_primary,
		       decision_role, notes, last_contacted_at, created_at, updated_at
		FROM crm_contact WHERE workspace_id = $1 AND account_id = $2 ORDER BY is_primary DESC, created_at ASC
	`, workspaceID, accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list CRM contacts")
		return
	}
	defer rows.Close()
	contacts := []CRMContactResponse{}
	for rows.Next() {
		contact, err := h.scanCRMContact(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan CRM contact")
			return
		}
		contacts = append(contacts, crmContactToResponse(contact))
	}
	writeJSON(w, http.StatusOK, map[string]any{"contacts": contacts, "total": len(contacts)})
}

func (h *Handler) UpdateCRMContact(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	if _, ok := h.getCRMAccount(w, r, accountID, workspaceID); !ok {
		return
	}
	contactID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "contactId"), "contact id")
	if !ok {
		return
	}
	var req UpdateCRMContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := normalizeCRMName(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	lastContactedAt, ok := cleanOptionalTimestamp(w, req.LastContactedAt, "last_contacted_at")
	if !ok {
		return
	}
	contact, err := h.scanCRMContact(h.DB.QueryRow(r.Context(), `
		UPDATE crm_contact SET
			account_id = $3,
			name = $4,
			salutation = $5,
			email = $6,
			phone = $7,
			mobile = $8,
			whatsapp_id = $9,
			whatsapp = $10,
			wechat = $11,
			linkedin_url = $12,
			role_title = $13,
			job_title = $14,
			department = $15,
			role = $16,
			language = $17,
			preferred_language = $18,
			timezone = $19,
			is_primary = $20,
			decision_role = $21,
			notes = $22,
			last_contacted_at = $23,
			updated_at = now()
		WHERE id = $1 AND workspace_id = $2 AND account_id = $3
		RETURNING id, workspace_id, account_id, name, salutation, email, phone, mobile,
		          whatsapp_id, whatsapp, wechat, linkedin_url, role_title, job_title,
		          department, role, language, preferred_language, timezone, is_primary,
		          decision_role, notes, last_contacted_at, created_at, updated_at
	`, contactID, workspaceID, accountID, name, cleanOptionalText(req.Salutation), cleanOptionalText(req.Email), cleanOptionalText(req.Phone),
		cleanOptionalText(req.Mobile), cleanOptionalText(req.WhatsappID), cleanOptionalText(req.Whatsapp), cleanOptionalText(req.Wechat),
		cleanOptionalText(req.LinkedinURL), cleanOptionalText(req.RoleTitle), cleanOptionalText(req.JobTitle), cleanOptionalText(req.Department),
		cleanOptionalText(req.Role), cleanOptionalText(req.Language), cleanOptionalText(req.PreferredLanguage), cleanOptionalText(req.Timezone),
		cleanOptionalBool(req.IsPrimary), cleanOptionalText(req.DecisionRole), cleanOptionalText(req.Notes), lastContactedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "CRM contact not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update CRM contact")
		return
	}
	writeJSON(w, http.StatusOK, crmContactToResponse(contact))
}

func (h *Handler) DeleteCRMContact(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	contactID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "contactId"), "contact id")
	if !ok {
		return
	}
	commandTag, err := h.DB.Exec(r.Context(), `DELETE FROM crm_contact WHERE id = $1 AND workspace_id = $2 AND account_id = $3`, contactID, workspaceID, accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete CRM contact")
		return
	}
	if commandTag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "CRM contact not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) scanCRMContact(row pgx.Row) (crmContactRow, error) {
	var contact crmContactRow
	err := row.Scan(
		&contact.ID, &contact.WorkspaceID, &contact.AccountID, &contact.Name,
		&contact.Salutation, &contact.Email, &contact.Phone, &contact.Mobile,
		&contact.WhatsappID, &contact.Whatsapp, &contact.Wechat, &contact.LinkedinURL,
		&contact.RoleTitle, &contact.JobTitle, &contact.Department, &contact.Role,
		&contact.Language, &contact.PreferredLanguage, &contact.Timezone, &contact.IsPrimary,
		&contact.DecisionRole, &contact.Notes, &contact.LastContactedAt,
		&contact.CreatedAt, &contact.UpdatedAt,
	)
	return contact, err
}

func (h *Handler) scanCRMEmailThread(row pgx.Row) (crmEmailThreadRow, error) {
	var thread crmEmailThreadRow
	var lastSnippet pgtype.Text
	err := row.Scan(
		&thread.ID, &thread.WorkspaceID, &thread.AccountID, &thread.ContactID, &thread.ProjectID, &thread.IssueID,
		&thread.Subject, &thread.ExternalThreadID, &thread.Mailbox, &thread.Direction,
		&thread.Status, &thread.LastMessageAt, &lastSnippet, &thread.CreatedAt, &thread.UpdatedAt,
		&thread.MessageCount, &thread.IsRead, &thread.IsStarred, &thread.IsTrashed,
	)
	thread.LastSnippet = cleanSnippetText(lastSnippet)
	return thread, err
}

func cleanSnippetText(value pgtype.Text) pgtype.Text {
	if !value.Valid {
		return value
	}
	text := strings.Join(strings.Fields(value.String), " ")
	if text == "" {
		return pgtype.Text{}
	}
	if len([]rune(text)) > 180 {
		runes := []rune(text)
		text = string(runes[:180]) + "…"
	}
	return pgtype.Text{String: text, Valid: true}
}

func (h *Handler) scanCRMEmailMessage(row pgx.Row) (crmEmailMessageRow, error) {
	var message crmEmailMessageRow
	err := row.Scan(
		&message.ID, &message.WorkspaceID, &message.ThreadID, &message.AccountID,
		&message.ContactID, &message.ExternalMessageID, &message.InReplyTo, &message.ReferenceIDs,
		&message.Attachments, &message.SentAppendWarning, &message.RawSizeBytes, &message.RawHeaders,
		&message.FromEmail, &message.FromName, &message.ToEmails, &message.CcEmails, &message.BccEmails,
		&message.Subject, &message.SentAt, &message.ReceivedAt, &message.BodyText, &message.BodyHTML,
		&message.Snippet, &message.Direction, &message.CreatedAt, &message.UpdatedAt,
	)
	return message, err
}

func (h *Handler) getCRMEmailThread(w http.ResponseWriter, r *http.Request, threadID pgtype.UUID, workspaceID pgtype.UUID) (crmEmailThreadRow, bool) {
	thread, err := h.scanCRMEmailThread(h.DB.QueryRow(r.Context(), `
		SELECT t.id, t.workspace_id, COALESCE(t.account_id, c.account_id) AS account_id, t.contact_id, t.project_id, t.issue_id, t.subject,
		       t.external_thread_id, t.mailbox, t.direction, t.status, t.last_message_at,
		       (SELECT COALESCE(NULLIF(m2.snippet, ''), LEFT(COALESCE(NULLIF(m2.body_text, ''), regexp_replace(COALESCE(m2.body_html, ''), '<[^>]+>', ' ', 'g')), 220))
		        FROM crm_email_message m2
		        WHERE m2.thread_id = t.id AND m2.workspace_id = t.workspace_id
		        ORDER BY COALESCE(m2.sent_at, m2.received_at, m2.created_at) DESC
		        LIMIT 1) AS last_snippet,
		       t.created_at, t.updated_at, COUNT(m.id)::bigint AS message_count, t.is_read, t.is_starred, t.is_trashed
		FROM crm_email_thread t
		LEFT JOIN crm_contact c ON c.id = t.contact_id AND c.workspace_id = t.workspace_id
		LEFT JOIN crm_email_message m ON m.thread_id = t.id AND m.workspace_id = t.workspace_id
		WHERE t.id = $1 AND t.workspace_id = $2
		GROUP BY t.id, c.account_id
	`, threadID, workspaceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "CRM email thread not found")
			return crmEmailThreadRow{}, false
		}
		writeError(w, http.StatusInternalServerError, "failed to load CRM email thread")
		return crmEmailThreadRow{}, false
	}
	return thread, true
}

func (h *Handler) ListCRMEmailThreads(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	if err := h.autoLinkCRMEmailWorkspaceByContact(r.Context(), workspaceID); err != nil {
		slog.Warn("auto-link CRM email workspace failed", "error", err, "workspace_id", uuidToString(workspaceID))
	}
	accountIDRaw := strings.TrimSpace(r.URL.Query().Get("account_id"))
	accountID, ok := optionalUUID(w, optionalStringFromQuery(r, "account_id"), "account_id")
	if !ok {
		return
	}
	accountIDFilter := accountIDRaw
	folder := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("folder")))
	if folder == "" {
		folder = "inbox"
	}
	filter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("filter")))
	if filter == "" {
		filter = "all"
	}
	mailbox := strings.TrimSpace(r.URL.Query().Get("mailbox"))
	folderCondition := "$3 = 'all'"
	switch folder {
	case "inbox":
		folderCondition = "m.direction = 'inbound' AND t.status = 'open' AND COALESCE(m.is_trashed, t.is_trashed) = false AND lower(COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', ''), 'INBOX')) NOT LIKE ALL(ARRAY['%spam%', '%junk%', '%trash%', '%deleted%', '%archive%'])"
	case "sent":
		folderCondition = "(m.direction = 'outbound' OR lower(COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', ''))) IN ('sent', 'sent messages', 'sent items'))"
	case "spam":
		folderCondition = "(lower(COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', ''))) LIKE ANY(ARRAY['%spam%', '%junk%']) OR COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', '')) LIKE '%垃圾%')"
	case "archived":
		folderCondition = "(t.status = 'archived' OR lower(COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', ''))) IN ('archive', 'archived'))"
	case "starred":
		folderCondition = "COALESCE(m.is_starred, t.is_starred) = true"
	case "unlinked":
		folderCondition = "t.account_id IS NULL AND t.contact_id IS NULL"
	case "trash":
		folderCondition = "(t.status = 'trashed' OR COALESCE(m.is_trashed, t.is_trashed) = true OR lower(COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', ''))) IN ('trash', 'deleted messages', 'deleted items'))"
	}
	filterCondition := "TRUE"
	switch filter {
	case "unlinked":
		filterCondition = "t.account_id IS NULL AND t.contact_id IS NULL"
	case "linked":
		filterCondition = "(t.account_id IS NOT NULL OR t.contact_id IS NOT NULL)"
	case "unread":
		filterCondition = "COALESCE(m.is_read, t.is_read) = false"
	case "read":
		filterCondition = "COALESCE(m.is_read, t.is_read) = true"
	}
	query := `
		WITH message_rows AS (
			SELECT m.id, m.workspace_id, m.thread_id, COALESCE(m.account_id, t.account_id, c.account_id) AS account_id, COALESCE(m.contact_id, t.contact_id) AS contact_id,
			       m.subject, COALESCE(NULLIF(m.snippet, ''), LEFT(COALESCE(NULLIF(m.body_text, ''), regexp_replace(COALESCE(m.body_html, ''), '<[^>]+>', ' ', 'g')), 220)) AS snippet,
			       t.mailbox, COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', ''), 'INBOX') AS folder,
			       m.direction, t.status, COALESCE(m.is_read, t.is_read) AS is_read,
			       COALESCE(m.is_starred, t.is_starred) AS is_starred, COALESCE(m.is_trashed, t.is_trashed) AS is_trashed,
			       m.from_email, m.from_name, m.to_emails, m.sent_at, m.received_at,
			       COALESCE((SELECT jsonb_agg(elem - 'content' - 'data' - 'body') FROM jsonb_array_elements(CASE WHEN jsonb_typeof(m.attachments) = 'array' THEN m.attachments ELSE '[]'::jsonb END) AS elem), '[]'::jsonb) AS attachments,
			       jsonb_array_length(CASE WHEN jsonb_typeof(m.attachments) = 'array' THEN m.attachments ELSE '[]'::jsonb END)::bigint AS attachment_count,
			       COUNT(*) OVER (PARTITION BY m.thread_id)::bigint AS thread_message_count,
			       m.created_at, m.updated_at
			FROM crm_email_message m
			JOIN crm_email_thread t ON t.id = m.thread_id AND t.workspace_id = m.workspace_id
			LEFT JOIN crm_contact c ON c.id = COALESCE(m.contact_id, t.contact_id) AND c.workspace_id = m.workspace_id
			WHERE m.workspace_id::text = $1
			  AND ($2 = '' OR t.account_id::text = $2 OR m.account_id::text = $2 OR c.account_id::text = $2 OR EXISTS (SELECT 1 FROM crm_contact c WHERE c.workspace_id = m.workspace_id AND c.account_id::text = $2 AND lower(COALESCE(c.email, '')) <> '' AND (lower(m.from_email) = lower(c.email) OR EXISTS (SELECT 1 FROM unnest(m.to_emails) AS x(email) WHERE lower(x.email) = lower(c.email)) OR EXISTS (SELECT 1 FROM unnest(m.cc_emails) AS x(email) WHERE lower(x.email) = lower(c.email)))))
			  AND ($3 = '' OR t.mailbox = $3)
			  AND (` + folderCondition + `)
			  AND (` + filterCondition + `)
		)
		SELECT id, workspace_id, thread_id, account_id, contact_id, subject, snippet, mailbox, folder,
		       direction, status, is_read, is_starred, is_trashed, from_email, from_name, to_emails,
		       sent_at, received_at, attachments, attachment_count, thread_message_count, created_at, updated_at
		FROM message_rows
		ORDER BY COALESCE(sent_at, received_at, created_at) DESC
		LIMIT 100
	`
	workspaceIDFilter := uuidToString(workspaceID)
	rows, err := h.DB.Query(r.Context(), query, workspaceIDFilter, accountIDFilter, mailbox)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list CRM email messages")
		return
	}
	defer rows.Close()
	messages := []CRMEmailListItemResponse{}
	threadIDs := map[string]bool{}
	for rows.Next() {
		var item crmEmailListItemRow
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.ThreadID, &item.AccountID, &item.ContactID, &item.Subject,
			&item.Snippet, &item.Mailbox, &item.Folder, &item.Direction, &item.Status, &item.IsRead, &item.IsStarred,
			&item.IsTrashed, &item.FromEmail, &item.FromName, &item.ToEmails, &item.SentAt, &item.ReceivedAt,
			&item.Attachments, &item.AttachmentCount, &item.ThreadMessageCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan CRM email message")
			return
		}
		resp := crmEmailListItemToResponse(item)
		threadIDs[resp.ThreadID] = true
		messages = append(messages, resp)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("list CRM email messages iteration failed", "error", err, "workspace_id", uuidToString(workspaceID), "folder", folder, "filter", filter, "mailbox", mailbox)
		writeError(w, http.StatusInternalServerError, "failed to list CRM email messages: "+err.Error())
		return
	}
	threadIDKeys := make([]string, 0, len(threadIDs))
	for rawThreadID := range threadIDs {
		threadIDKeys = append(threadIDKeys, rawThreadID)
	}
	threadIDValues, _ := parseUUIDSliceOrBadRequest(w, threadIDKeys, "thread id")
	threads := []CRMEmailThreadResponse{}
	if len(threadIDValues) > 0 {
		threadRows, err := h.DB.Query(r.Context(), `
			SELECT t.id, t.workspace_id, COALESCE(t.account_id, c.account_id) AS account_id, t.contact_id, t.project_id, t.issue_id, t.subject,
			       t.external_thread_id, t.mailbox, t.direction, t.status, t.last_message_at,
			       (SELECT COALESCE(NULLIF(m2.snippet, ''), LEFT(COALESCE(NULLIF(m2.body_text, ''), regexp_replace(COALESCE(m2.body_html, ''), '<[^>]+>', ' ', 'g')), 220))
			        FROM crm_email_message m2
			        WHERE m2.thread_id = t.id AND m2.workspace_id = t.workspace_id
			        ORDER BY COALESCE(m2.sent_at, m2.received_at, m2.created_at) DESC
			        LIMIT 1) AS last_snippet,
			       t.created_at, t.updated_at, COUNT(DISTINCT m.id)::bigint AS message_count, t.is_read, t.is_starred, t.is_trashed,
			       COALESCE(array_agg(DISTINCT l.issue_id::text) FILTER (WHERE l.issue_id IS NOT NULL), ARRAY[]::text[]) AS issue_ids
			FROM crm_email_thread t
			LEFT JOIN crm_contact c ON c.id = t.contact_id AND c.workspace_id = t.workspace_id
			LEFT JOIN crm_email_message m ON m.thread_id = t.id AND m.workspace_id = t.workspace_id
			LEFT JOIN crm_email_thread_issue_link l ON l.thread_id = t.id
			WHERE t.workspace_id = $1 AND t.id = ANY($2::uuid[])
			GROUP BY t.id, c.account_id
		`, workspaceID, threadIDValues)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list CRM email threads")
			return
		}
		defer threadRows.Close()
		for threadRows.Next() {
			var thread crmEmailListThreadRow
			if err := threadRows.Scan(&thread.ID, &thread.WorkspaceID, &thread.AccountID, &thread.ContactID, &thread.ProjectID,
				&thread.IssueID, &thread.Subject, &thread.ExternalThreadID, &thread.Mailbox, &thread.Direction, &thread.Status,
				&thread.LastMessageAt, &thread.LastSnippet, &thread.CreatedAt, &thread.UpdatedAt, &thread.MessageCount,
				&thread.IsRead, &thread.IsStarred, &thread.IsTrashed, &thread.IssueIDs); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to scan CRM email thread")
				return
			}
			threads = append(threads, crmEmailListThreadToResponse(thread))
		}
		if err := threadRows.Err(); err != nil {
			slog.Warn("list CRM email threads iteration failed", "error", err, "workspace_id", uuidToString(workspaceID), "folder", folder)
			writeError(w, http.StatusInternalServerError, "failed to list CRM email threads: "+err.Error())
			return
		}
	}
	counts := h.crmEmailFolderCounts(r.Context(), workspaceID, accountID, mailbox)
	slog.Info("crm email threads listed", "workspace_id", uuidToString(workspaceID), "account_id", accountIDRaw, "folder", folder, "filter", filter, "mailbox", mailbox, "folder_condition", folderCondition, "filter_condition", filterCondition, "messages", len(messages), "threads", len(threads), "total", len(messages))
	writeJSON(w, http.StatusOK, map[string]any{"messages": messages, "threads": threads, "total": len(messages), "counts": counts})
}

func (h *Handler) crmEmailFolderCounts(ctx context.Context, workspaceID pgtype.UUID, accountID pgtype.UUID, mailbox string) CRMEmailListCountsResponse {
	var counts CRMEmailListCountsResponse
	_ = h.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE m.direction = 'inbound' AND t.status = 'open' AND COALESCE(m.is_trashed, t.is_trashed) = false AND lower(COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', ''), 'INBOX')) NOT LIKE ALL(ARRAY['%spam%', '%junk%', '%trash%', '%deleted%', '%archive%'])),
			COUNT(*) FILTER (WHERE m.direction = 'inbound' AND t.status = 'open' AND COALESCE(m.is_read, t.is_read) = false AND COALESCE(m.is_trashed, t.is_trashed) = false AND lower(COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', ''), 'INBOX')) NOT LIKE ALL(ARRAY['%spam%', '%junk%', '%trash%', '%deleted%', '%archive%'])),
			COUNT(*) FILTER (WHERE m.direction = 'outbound' OR lower(COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', ''))) IN ('sent', 'sent messages', 'sent items')),
			COUNT(*) FILTER (WHERE lower(COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', ''))) LIKE ANY(ARRAY['%spam%', '%junk%']) OR COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', '')) LIKE '%垃圾%'),
			COUNT(*) FILTER (WHERE t.status = 'archived' OR lower(COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', ''))) IN ('archive', 'archived')),
			COUNT(*) FILTER (WHERE COALESCE(m.is_starred, t.is_starred) = true),
			COUNT(*) FILTER (WHERE t.account_id IS NULL AND t.contact_id IS NULL),
			COUNT(*) FILTER (WHERE t.status = 'trashed' OR COALESCE(m.is_trashed, t.is_trashed) = true OR lower(COALESCE(NULLIF(m.folder, ''), NULLIF(m.source_metadata->>'folder', ''))) IN ('trash', 'deleted messages', 'deleted items'))
		FROM crm_email_message m
		JOIN crm_email_thread t ON t.id = m.thread_id AND t.workspace_id = m.workspace_id
		WHERE m.workspace_id = $1 AND ($2::uuid IS NULL OR t.account_id = $2) AND ($3 = '' OR t.mailbox = $3)
	`, workspaceID, accountID, mailbox).Scan(&counts.Inbox, &counts.InboxUnread, &counts.Sent, &counts.Spam, &counts.Archived, &counts.Starred, &counts.Unlinked, &counts.Trash)
	return counts
}

func (h *Handler) GetCRMEmailThread(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	threadID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "threadId"), "thread id")
	if !ok {
		return
	}
	if err := h.autoLinkCRMEmailThreadByContact(r.Context(), workspaceID, threadID); err != nil {
		slog.Warn("auto-link CRM email thread failed", "error", err, "workspace_id", uuidToString(workspaceID), "thread_id", uuidToString(threadID))
	}
	thread, ok := h.getCRMEmailThread(w, r, threadID, workspaceID)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, crmEmailThreadToResponse(thread))
}

func (h *Handler) autoLinkCRMEmailThreadByContact(ctx context.Context, workspaceID pgtype.UUID, threadID pgtype.UUID) error {
	const bestMatch = `
		SELECT c.account_id, c.id
		FROM crm_email_message m
		JOIN crm_contact c ON c.workspace_id = m.workspace_id
		WHERE m.workspace_id = $1
		  AND m.thread_id = $2
		  AND c.account_id IS NOT NULL
		  AND lower(trim(COALESCE(c.email, ''))) <> ''
		  AND (
			lower(trim(COALESCE(m.from_email, ''))) = lower(trim(c.email))
			OR EXISTS (SELECT 1 FROM unnest(COALESCE(m.to_emails, ARRAY[]::text[])) AS x(email) WHERE lower(trim(x.email)) = lower(trim(c.email)))
			OR EXISTS (SELECT 1 FROM unnest(COALESCE(m.cc_emails, ARRAY[]::text[])) AS x(email) WHERE lower(trim(x.email)) = lower(trim(c.email)))
		  )
		GROUP BY c.account_id, c.id, c.is_primary, c.updated_at
		ORDER BY count(*) DESC, c.is_primary DESC NULLS LAST, c.updated_at DESC NULLS LAST
		LIMIT 1
	`
	var accountID pgtype.UUID
	var contactID pgtype.UUID
	if err := h.DB.QueryRow(ctx, bestMatch, workspaceID, threadID).Scan(&accountID, &contactID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if _, err := h.DB.Exec(ctx, `
		UPDATE crm_email_thread
		SET account_id = COALESCE(account_id, $3), contact_id = COALESCE(contact_id, $4), updated_at = now()
		WHERE workspace_id = $1 AND id = $2 AND (account_id IS NULL OR contact_id IS NULL)
	`, workspaceID, threadID, accountID, contactID); err != nil {
		return err
	}
	_, err := h.DB.Exec(ctx, `
		UPDATE crm_email_message
		SET account_id = COALESCE(account_id, $3), contact_id = COALESCE(contact_id, $4), updated_at = now()
		WHERE workspace_id = $1 AND thread_id = $2 AND (account_id IS NULL OR contact_id IS NULL)
	`, workspaceID, threadID, accountID, contactID)
	return err
}

func (h *Handler) autoLinkCRMEmailWorkspaceByContact(ctx context.Context, workspaceID pgtype.UUID) error {
	const matchCTE = `
		WITH best AS (
			SELECT DISTINCT ON (m.thread_id)
				m.thread_id, c.account_id, c.id AS contact_id
			FROM crm_email_message m
			JOIN crm_contact c ON c.workspace_id = m.workspace_id
			WHERE m.workspace_id = $1
			  AND c.account_id IS NOT NULL
			  AND lower(trim(COALESCE(c.email, ''))) <> ''
			  AND (
				lower(trim(COALESCE(m.from_email, ''))) = lower(trim(c.email))
				OR EXISTS (SELECT 1 FROM unnest(COALESCE(m.to_emails, ARRAY[]::text[])) AS x(email) WHERE lower(trim(x.email)) = lower(trim(c.email)))
				OR EXISTS (SELECT 1 FROM unnest(COALESCE(m.cc_emails, ARRAY[]::text[])) AS x(email) WHERE lower(trim(x.email)) = lower(trim(c.email)))
			  )
			ORDER BY m.thread_id, c.is_primary DESC NULLS LAST, c.updated_at DESC NULLS LAST
		)
	`
	if _, err := h.DB.Exec(ctx, matchCTE+`
		UPDATE crm_email_thread t
		SET account_id = COALESCE(t.account_id, b.account_id),
			contact_id = COALESCE(t.contact_id, b.contact_id),
			updated_at = now()
		FROM best b
		WHERE t.workspace_id = $1 AND t.id = b.thread_id AND (t.account_id IS NULL OR t.contact_id IS NULL)
	`, workspaceID); err != nil {
		return err
	}
	_, err := h.DB.Exec(ctx, matchCTE+`
		UPDATE crm_email_message m
		SET account_id = COALESCE(m.account_id, b.account_id),
			contact_id = COALESCE(m.contact_id, b.contact_id),
			updated_at = now()
		FROM best b
		WHERE m.workspace_id = $1 AND m.thread_id = b.thread_id AND (m.account_id IS NULL OR m.contact_id IS NULL)
	`, workspaceID)
	return err
}

func emailDomain(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if at := strings.LastIndex(value, "@"); at >= 0 {
		value = value[at+1:]
	}
	value = strings.TrimPrefix(strings.TrimPrefix(value, "https://"), "http://")
	value = strings.TrimPrefix(value, "www.")
	if slash := strings.Index(value, "/"); slash >= 0 {
		value = value[:slash]
	}
	if colon := strings.Index(value, ":"); colon >= 0 {
		value = value[:colon]
	}
	return value
}

func isGenericCRMEmailDomain(domain string) bool {
	switch strings.ToLower(strings.TrimSpace(domain)) {
	case "gmail.com", "googlemail.com", "outlook.com", "hotmail.com", "live.com", "icloud.com", "me.com", "qq.com", "163.com", "126.com", "yahoo.com", "proton.me", "protonmail.com":
		return true
	default:
		return false
	}
}

func addSuggestionReason(reasons []string, reason string) []string {
	for _, existing := range reasons {
		if existing == reason {
			return reasons
		}
	}
	return append(reasons, reason)
}

func (h *Handler) SuggestCRMEmailThreadAssociations(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	threadID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "threadId"), "thread id")
	if !ok {
		return
	}
	if _, ok := h.getCRMEmailThread(w, r, threadID, workspaceID); !ok {
		return
	}

	messageRows, err := h.DB.Query(r.Context(), `
		SELECT from_email, to_emails, cc_emails
		FROM crm_email_message
		WHERE workspace_id = $1 AND thread_id = $2
		ORDER BY COALESCE(received_at, sent_at, created_at) ASC
	`, workspaceID, threadID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load CRM email messages")
		return
	}
	defer messageRows.Close()

	emails := map[string]bool{}
	domains := map[string]bool{}
	for messageRows.Next() {
		var from pgtype.Text
		var toEmails, ccEmails []string
		if err := messageRows.Scan(&from, &toEmails, &ccEmails); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan CRM email messages")
			return
		}
		candidates := append([]string{}, toEmails...)
		candidates = append(candidates, ccEmails...)
		if from.Valid {
			candidates = append(candidates, from.String)
		}
		for _, candidate := range candidates {
			email := strings.ToLower(strings.TrimSpace(candidate))
			domain := emailDomain(email)
			if email != "" && strings.Contains(email, "@") {
				emails[email] = true
			}
			if domain != "" {
				domains[domain] = true
			}
		}
	}

	type suggestionAccumulator struct {
		suggestion CRMEmailThreadAssociationSuggestion
	}
	suggestions := map[string]*suggestionAccumulator{}
	rows, err := h.DB.Query(r.Context(), `
		SELECT a.id, a.name, a.website, c.id, c.name, c.email
		FROM crm_account a
		LEFT JOIN crm_contact c ON c.account_id = a.id AND c.workspace_id = a.workspace_id
		WHERE a.workspace_id = $1
		ORDER BY a.updated_at DESC, c.is_primary DESC NULLS LAST
		LIMIT 500
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load CRM association candidates")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var accountID pgtype.UUID
		var accountName string
		var contactID pgtype.UUID
		var website, contactName, contactEmail pgtype.Text
		if err := rows.Scan(&accountID, &accountName, &website, &contactID, &contactName, &contactEmail); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan CRM association candidates")
			return
		}
		key := uuidToString(accountID)
		if contactID.Valid {
			key += ":" + uuidToString(contactID)
		}
		acc := suggestions[key]
		if acc == nil {
			acc = &suggestionAccumulator{suggestion: CRMEmailThreadAssociationSuggestion{AccountID: uuidToString(accountID), AccountName: accountName}}
			if contactID.Valid {
				v := uuidToString(contactID)
				acc.suggestion.ContactID = &v
			}
			if contactName.Valid {
				v := contactName.String
				acc.suggestion.ContactName = &v
			}
			if contactEmail.Valid {
				v := contactEmail.String
				acc.suggestion.ContactEmail = &v
			}
			suggestions[key] = acc
		}
		if contactEmail.Valid && emails[strings.ToLower(strings.TrimSpace(contactEmail.String))] {
			acc.suggestion.Score += 100
			acc.suggestion.Reasons = addSuggestionReason(acc.suggestion.Reasons, "contact email matches a thread sender or recipient")
		}
		if contactEmail.Valid && domains[emailDomain(contactEmail.String)] {
			acc.suggestion.Score += 40
			acc.suggestion.Reasons = addSuggestionReason(acc.suggestion.Reasons, "contact email domain matches the thread")
		}
		if website.Valid && domains[emailDomain(website.String)] {
			acc.suggestion.Score += 70
			acc.suggestion.Reasons = addSuggestionReason(acc.suggestion.Reasons, "customer website domain matches the sender domain")
		}
	}

	out := []CRMEmailThreadAssociationSuggestion{}
	for _, acc := range suggestions {
		if acc.suggestion.Score > 0 {
			out = append(out, acc.suggestion)
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Score > out[i].Score {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > 5 {
		out = out[:5]
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": out, "total": len(out)})
}

func (h *Handler) ListCRMIMAPSyncRuns(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	limit := 50
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 200 {
		limit = 200
	}
	query := `SELECT r.id, r.mailbox_id, s.email, r.folder, r.status, r.requested_limit, r.fetched_count, r.imported_count, r.skipped_count, r.error_message, r.started_at, r.finished_at, r.created_at, r.updated_at
		FROM crm_imap_sync_run r
		LEFT JOIN crm_imap_setting s ON s.id = r.mailbox_id AND s.workspace_id = r.workspace_id
		WHERE r.workspace_id=$1`
	args := []any{workspaceID}
	if status != "" && status != "all" {
		query += ` AND r.status=$2`
		args = append(args, status)
	}
	query += ` ORDER BY r.created_at DESC LIMIT $` + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := h.DB.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list CRM IMAP sync runs")
		return
	}
	defer rows.Close()

	runs := []CRMIMAPSyncRunResponse{}
	for rows.Next() {
		var id, mailboxID pgtype.UUID
		var mailboxEmail, errorMessage pgtype.Text
		var folder, runStatus string
		var requestedLimit, fetchedCount, importedCount, skippedCount int32
		var startedAt, finishedAt, createdAt, updatedAt pgtype.Timestamptz
		if err := rows.Scan(&id, &mailboxID, &mailboxEmail, &folder, &runStatus, &requestedLimit, &fetchedCount, &importedCount, &skippedCount, &errorMessage, &startedAt, &finishedAt, &createdAt, &updatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan CRM IMAP sync run")
			return
		}
		runs = append(runs, CRMIMAPSyncRunResponse{
			ID: uuidToString(id), MailboxID: uuidToPtr(mailboxID), MailboxEmail: textToPtr(mailboxEmail), Folder: folder, Status: runStatus,
			RequestedLimit: requestedLimit, FetchedCount: fetchedCount, ImportedCount: importedCount, SkippedCount: skippedCount,
			ErrorMessage: textToPtr(errorMessage), StartedAt: timestampToString(startedAt), FinishedAt: timestampToPtr(finishedAt), CreatedAt: timestampToString(createdAt), UpdatedAt: timestampToString(updatedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs, "total": len(runs)})
}

func (h *Handler) UpdateCRMEmailThreadState(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	threadID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "threadId"), "thread id")
	if !ok {
		return
	}
	var req UpdateCRMEmailThreadStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status != nil && *req.Status != "open" && *req.Status != "archived" {
		writeError(w, http.StatusBadRequest, "invalid email thread status")
		return
	}
	var messageID pgtype.UUID
	if req.MessageID != nil && strings.TrimSpace(*req.MessageID) != "" {
		parsedMessageID, ok := parseUUIDOrBadRequest(w, strings.TrimSpace(*req.MessageID), "message id")
		if !ok {
			return
		}
		messageID = parsedMessageID
		cmd, err := h.DB.Exec(r.Context(), `
			UPDATE crm_email_message
			SET is_read = COALESCE($4, is_read),
			    is_starred = COALESCE($5, is_starred),
			    folder = CASE WHEN $6='archived' THEN 'Archive' WHEN $6='open' AND folder IN ('Archive','Archived') THEN 'INBOX' ELSE folder END,
			    is_trashed = CASE WHEN $6='open' THEN false ELSE is_trashed END,
			    updated_at = now()
			WHERE id = $1 AND thread_id = $2 AND workspace_id = $3
		`, messageID, threadID, workspaceID, req.IsRead, req.IsStarred, req.Status)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update CRM email message")
			return
		}
		if cmd.RowsAffected() == 0 {
			writeError(w, http.StatusNotFound, "CRM email message not found")
			return
		}
	} else {
		_, _ = h.DB.Exec(r.Context(), `
			UPDATE crm_email_message
			SET is_read = COALESCE($3, is_read),
			    is_starred = COALESCE($4, is_starred),
			    folder = CASE WHEN $5='archived' THEN 'Archive' WHEN $5='open' AND folder IN ('Archive','Archived') THEN 'INBOX' ELSE folder END,
			    is_trashed = CASE WHEN $5='open' THEN false ELSE is_trashed END,
			    updated_at = now()
			WHERE thread_id = $1 AND workspace_id = $2
		`, threadID, workspaceID, req.IsRead, req.IsStarred, req.Status)
	}

	thread, err := h.scanCRMEmailThread(h.DB.QueryRow(r.Context(), `
		UPDATE crm_email_thread
		SET status = COALESCE($3, status),
		    is_read = COALESCE((SELECT bool_and(m.is_read) FROM crm_email_message m WHERE m.thread_id = crm_email_thread.id AND m.workspace_id = crm_email_thread.workspace_id), is_read),
		    is_starred = COALESCE($4, is_starred),
		    updated_at = now()
		WHERE id = $1 AND workspace_id = $2
		RETURNING id, workspace_id, account_id, contact_id, project_id, issue_id, subject,
		          external_thread_id, mailbox, direction, status, last_message_at,
		          (SELECT COALESCE(NULLIF(m.snippet, ''), LEFT(COALESCE(NULLIF(m.body_text, ''), regexp_replace(COALESCE(m.body_html, ''), '<[^>]+>', ' ', 'g')), 220)) FROM crm_email_message m WHERE m.thread_id = crm_email_thread.id AND m.workspace_id = crm_email_thread.workspace_id ORDER BY COALESCE(m.sent_at, m.received_at, m.created_at) DESC LIMIT 1) AS last_snippet,
		          created_at, updated_at,
		          (SELECT COUNT(*)::bigint FROM crm_email_message m WHERE m.thread_id = crm_email_thread.id AND m.workspace_id = crm_email_thread.workspace_id), is_read, is_starred, is_trashed
	`, threadID, workspaceID, req.Status, req.IsStarred))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "CRM email thread not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update CRM email thread")
		return
	}
	thread.IssueIDs = h.loadCRMEmailThreadIssueIDs(r.Context(), thread.ID)
	h.trySyncCRMEmailThreadFlags(r.Context(), workspaceID, threadID, req.IsRead, req.IsStarred)
	writeJSON(w, http.StatusOK, crmEmailThreadToResponse(thread))
}

func (h *Handler) UpdateCRMEmailThreadAssociation(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	threadID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "threadId"), "thread id")
	if !ok {
		return
	}
	if _, ok := h.getCRMEmailThread(w, r, threadID, workspaceID); !ok {
		return
	}
	var req UpdateCRMEmailThreadAssociationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	accountID, ok := optionalUUID(w, req.AccountID, "account_id")
	if !ok {
		return
	}
	contactID, ok := optionalUUID(w, req.ContactID, "contact_id")
	if !ok {
		return
	}
	projectID, ok := optionalUUID(w, req.ProjectID, "project_id")
	if !ok {
		return
	}
	if projectID.Valid {
		if _, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{ID: projectID, WorkspaceID: workspaceID}); err != nil {
			writeError(w, http.StatusBadRequest, "project not found in this workspace")
			return
		}
	}
	issueID, ok := optionalUUID(w, req.IssueID, "issue_id")
	if !ok {
		return
	}
	issueIDs := make([]pgtype.UUID, 0, len(req.IssueIDs))
	if len(req.IssueIDs) == 0 && issueID.Valid {
		issueIDs = append(issueIDs, issueID)
	}
	for _, rawIssueID := range req.IssueIDs {
		parsed, ok := parseUUIDOrBadRequest(w, rawIssueID, "issue_id")
		if !ok {
			return
		}
		issueIDs = append(issueIDs, parsed)
	}
	for _, linkedIssueID := range issueIDs {
		issue, err := h.Queries.GetIssueInWorkspace(r.Context(), db.GetIssueInWorkspaceParams{ID: linkedIssueID, WorkspaceID: workspaceID})
		if err != nil {
			writeError(w, http.StatusBadRequest, "issue not found in this workspace")
			return
		}
		if projectID.Valid && issue.ProjectID.Valid && issue.ProjectID.Bytes != projectID.Bytes {
			writeError(w, http.StatusBadRequest, "issue does not belong to selected project")
			return
		}
	}
	if contactID.Valid {
		var inferredAccountID pgtype.UUID
		if err := h.DB.QueryRow(r.Context(), `SELECT account_id FROM crm_contact WHERE id=$1 AND workspace_id=$2`, contactID, workspaceID).Scan(&inferredAccountID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusBadRequest, "contact not found in this workspace")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to load CRM contact account")
			return
		}
		if inferredAccountID.Valid {
			accountID = inferredAccountID
		}
	}
	primaryIssueID := issueID
	if len(issueIDs) > 0 {
		primaryIssueID = issueIDs[0]
	}
	thread, err := h.scanCRMEmailThread(h.DB.QueryRow(r.Context(), `
		UPDATE crm_email_thread
		SET account_id = $3, contact_id = $4, project_id = $5, issue_id = $6, updated_at = now()
		WHERE id = $1 AND workspace_id = $2
		RETURNING id, workspace_id, account_id, contact_id, project_id, issue_id, subject, external_thread_id, mailbox, direction, status, last_message_at,
		          (SELECT COALESCE(NULLIF(m.snippet, ''), LEFT(COALESCE(NULLIF(m.body_text, ''), regexp_replace(COALESCE(m.body_html, ''), '<[^>]+>', ' ', 'g')), 220)) FROM crm_email_message m WHERE m.thread_id = crm_email_thread.id AND m.workspace_id = crm_email_thread.workspace_id ORDER BY COALESCE(m.sent_at, m.received_at, m.created_at) DESC LIMIT 1) AS last_snippet,
		          created_at, updated_at,
		          (SELECT COUNT(*)::bigint FROM crm_email_message m WHERE m.thread_id = crm_email_thread.id AND m.workspace_id = crm_email_thread.workspace_id), is_read, is_starred, is_trashed
	`, threadID, workspaceID, accountID, contactID, projectID, primaryIssueID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update CRM email thread association")
		return
	}
	thread.IssueIDs = issueIDs
	if _, err := h.DB.Exec(r.Context(), `
		UPDATE crm_email_message
		SET account_id = $3, contact_id = $4, updated_at = now()
		WHERE thread_id = $1 AND workspace_id = $2
	`, threadID, workspaceID, accountID, contactID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update CRM email message association")
		return
	}
	if _, err := h.DB.Exec(r.Context(), `DELETE FROM crm_email_thread_issue_link WHERE thread_id = $1`, threadID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update CRM email issue links")
		return
	}
	for _, linkedIssueID := range issueIDs {
		if _, err := h.DB.Exec(r.Context(), `INSERT INTO crm_email_thread_issue_link (thread_id, issue_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, threadID, linkedIssueID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update CRM email issue links")
			return
		}
	}
	writeJSON(w, http.StatusOK, crmEmailThreadToResponse(thread))
}

func scanCRMIMAPSetting(row pgx.Row) (CRMIMAPSettingResponse, error) {
	var r CRMIMAPSettingResponse
	var id, ws pgtype.UUID
	var secretRef, status, msg, ownerType, smtpHost, smtpTLSMode, smtpUsername, smtpSecretRef pgtype.Text
	var ownerID pgtype.UUID
	var smtpPort pgtype.Int4
	var tested, created, updated pgtype.Timestamptz
	err := row.Scan(&id, &ws, &r.Label, &r.Email, &r.Host, &r.Port, &r.TLSMode, &r.Username, &secretRef, &r.SyncEnabled, &status, &msg, &tested, &ownerType, &ownerID, &smtpHost, &smtpPort, &smtpTLSMode, &smtpUsername, &smtpSecretRef, &created, &updated)
	r.ID = uuidToString(id)
	r.WorkspaceID = uuidToString(ws)
	r.SecretRef = textToPtr(secretRef)
	r.LastTestStatus = textToPtr(status)
	r.LastTestMessage = textToPtr(msg)
	r.LastTestedAt = timestampToPtr(tested)
	r.OwnerType = textToPtr(ownerType)
	r.OwnerID = uuidToPtr(ownerID)
	r.SMTPHost = textToPtr(smtpHost)
	if smtpPort.Valid {
		v := int32(smtpPort.Int32)
		r.SMTPPort = &v
	}
	r.SMTPTLSMode = textToPtr(smtpTLSMode)
	r.SMTPUsername = textToPtr(smtpUsername)
	r.SMTPSecretRef = textToPtr(smtpSecretRef)
	r.CreatedAt = timestampToString(created)
	r.UpdatedAt = timestampToString(updated)
	return r, err
}

func (h *Handler) ListCRMIMAPSettings(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT id, workspace_id, label, email, host, port, tls_mode, username, secret_ref, sync_enabled, last_test_status, last_test_message, last_tested_at, owner_type, owner_id, smtp_host, smtp_port, smtp_tls_mode, smtp_username, smtp_secret_ref, created_at, updated_at FROM crm_imap_setting WHERE workspace_id=$1 ORDER BY updated_at DESC`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list CRM IMAP settings")
		return
	}
	defer rows.Close()
	settings := []CRMIMAPSettingResponse{}
	for rows.Next() {
		item, err := scanCRMIMAPSetting(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan CRM IMAP setting")
			return
		}
		settings = append(settings, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings, "total": len(settings)})
}

func (h *Handler) UpsertCRMIMAPSetting(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	var req UpsertCRMIMAPSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	label := normalizeCRMName(req.Label)
	email := strings.TrimSpace(req.Email)
	host := strings.TrimSpace(req.Host)
	user := strings.TrimSpace(req.Username)
	tlsMode := cleanDefault(&req.TLSMode, "ssl")
	if label == "" || email == "" || host == "" || user == "" {
		writeError(w, http.StatusBadRequest, "label, email, host, and username are required")
		return
	}
	if req.Port <= 0 {
		req.Port = 993
	}
	secretRef := cleanOptionalText(req.SecretRef)
	if req.Secret != nil && strings.TrimSpace(*req.Secret) != "" {
		secretRef = pgtype.Text{String: encodeCRMIMAPInlineSecret(strings.TrimSpace(*req.Secret)), Valid: true}
	}
	if tlsMode != "ssl" && tlsMode != "starttls" && tlsMode != "none" {
		writeError(w, http.StatusBadRequest, "invalid tls_mode")
		return
	}
	ownerType := cleanOptionalText(req.OwnerType)
	ownerID, ok := optionalUUID(w, req.OwnerID, "owner_id")
	if !ok {
		return
	}
	smtpHost := cleanOptionalText(req.SMTPHost)
	smtpPort := pgtype.Int4{}
	if req.SMTPPort != nil && *req.SMTPPort > 0 {
		smtpPort = pgtype.Int4{Int32: *req.SMTPPort, Valid: true}
	}
	smtpTLSMode := cleanOptionalText(req.SMTPTLSMode)
	smtpUsername := cleanOptionalText(req.SMTPUsername)
	smtpSecretRef := cleanOptionalText(req.SMTPSecretRef)
	if req.SMTPSecret != nil && strings.TrimSpace(*req.SMTPSecret) != "" {
		smtpSecretRef = pgtype.Text{String: encodeCRMIMAPInlineSecret(strings.TrimSpace(*req.SMTPSecret)), Valid: true}
	}
	id, ok := optionalUUID(w, req.ID, "id")
	if !ok {
		return
	}
	var row pgx.Row
	if id.Valid {
		row = h.DB.QueryRow(r.Context(), `UPDATE crm_imap_setting SET label=$3,email=$4,host=$5,port=$6,tls_mode=$7,username=$8,secret_ref=$9,sync_enabled=$10,owner_type=$11,owner_id=$12,smtp_host=$13,smtp_port=$14,smtp_tls_mode=$15,smtp_username=$16,smtp_secret_ref=$17,updated_at=now() WHERE id=$1 AND workspace_id=$2 RETURNING id, workspace_id, label, email, host, port, tls_mode, username, secret_ref, sync_enabled, last_test_status, last_test_message, last_tested_at, owner_type, owner_id, smtp_host, smtp_port, smtp_tls_mode, smtp_username, smtp_secret_ref, created_at, updated_at`, id, workspaceID, label, email, host, req.Port, tlsMode, user, secretRef, req.SyncEnabled, ownerType, ownerID, smtpHost, smtpPort, smtpTLSMode, smtpUsername, smtpSecretRef)
	} else {
		row = h.DB.QueryRow(r.Context(), `INSERT INTO crm_imap_setting (workspace_id,label,email,host,port,tls_mode,username,secret_ref,sync_enabled,owner_type,owner_id,smtp_host,smtp_port,smtp_tls_mode,smtp_username,smtp_secret_ref) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id, workspace_id, label, email, host, port, tls_mode, username, secret_ref, sync_enabled, last_test_status, last_test_message, last_tested_at, owner_type, owner_id, smtp_host, smtp_port, smtp_tls_mode, smtp_username, smtp_secret_ref, created_at, updated_at`, workspaceID, label, email, host, req.Port, tlsMode, user, secretRef, req.SyncEnabled, ownerType, ownerID, smtpHost, smtpPort, smtpTLSMode, smtpUsername, smtpSecretRef)
	}
	item, err := scanCRMIMAPSetting(row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save CRM IMAP setting")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteCRMIMAPSetting(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	mailboxID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "mailboxId"), "mailbox id")
	if !ok {
		return
	}
	cmd, err := h.DB.Exec(r.Context(), `DELETE FROM crm_imap_setting WHERE id=$1 AND workspace_id=$2`, mailboxID, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete CRM IMAP setting")
		return
	}
	if cmd.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "CRM IMAP setting not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) TestCRMIMAPSetting(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	mailboxID := chi.URLParam(r, "mailboxId")
	cfg, ok := h.loadCRMIMAPConfig(w, r, workspaceID, &mailboxID)
	if !ok {
		return
	}

	status := "ok"
	msg := "IMAP connection successful"
	if _, err := fetchCRMEmailProviderMessages(cfg, "INBOX", 1, 0, nil); err != nil {
		status = "failed"
		msg = "IMAP connection failed: " + err.Error()
	}
	_, _ = h.DB.Exec(r.Context(), `UPDATE crm_imap_setting SET last_test_status=$3,last_test_message=$4,last_tested_at=now(),updated_at=now() WHERE id=$1 AND workspace_id=$2`, cfg.UUID, workspaceID, status, msg)
	writeJSON(w, http.StatusOK, map[string]any{"ok": status == "ok", "status": status, "message": msg})
}

func (h *Handler) GetCRMEmailEngineStatus(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	mailboxID := r.URL.Query().Get("mailbox_id")
	var mailboxIDPtr *string
	if mailboxID != "" {
		mailboxIDPtr = &mailboxID
	}
	cfg, ok := h.loadCRMIMAPConfig(w, r, workspaceID, mailboxIDPtr)
	if !ok {
		return
	}
	status, err := fetchCRMEmailEngineStatus(cfg)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch EmailEngine status: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) PreviewCRMIMAP(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	var req CRMIMAPPreviewRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	limit := req.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	cfg, ok := h.loadCRMIMAPConfig(w, r, workspaceID, req.MailboxID)
	if !ok {
		return
	}
	messages, err := fetchCRMEmailProviderMessages(cfg, cleanCRMIMAPFolder(req.Folder), limit, req.RangeDays, nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch IMAP messages: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": crmIMAPPreviewMessagesToResponse(messages), "total": len(messages), "limit": limit, "sync_enabled": false, "note": "Fetched live IMAP messages for manual preview; no messages imported yet."})
}

func (h *Handler) ImportCRMIMAP(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	var req CRMIMAPImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.UIDs) == 0 {
		writeError(w, http.StatusBadRequest, "uids are required")
		return
	}
	cfg, ok := h.loadCRMIMAPConfig(w, r, workspaceID, req.MailboxID)
	if !ok {
		return
	}
	messages, err := fetchCRMEmailProviderMessages(cfg, cleanCRMIMAPFolder(req.Folder), len(req.UIDs), 0, req.UIDs)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch IMAP messages: "+err.Error())
		return
	}
	imported, skipped, err := h.importCRMIMAPMessages(r.Context(), workspaceID, cfg, cleanCRMIMAPFolder(req.Folder), messages)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to import IMAP messages")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "fetched": len(messages), "imported": imported, "skipped": skipped})
}

func (h *Handler) SyncCRMIMAP(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	var req CRMIMAPImportRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	limit := req.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	cfg, ok := h.loadCRMIMAPConfig(w, r, workspaceID, req.MailboxID)
	if !ok {
		return
	}
	folder := cleanCRMIMAPFolder(req.Folder)
	staleCtx, staleCancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, _ = h.DB.Exec(staleCtx, `UPDATE crm_imap_sync_run SET status='failed', error_message='stale running sync reset before new run', finished_at=now(), updated_at=now() WHERE workspace_id=$1 AND mailbox_id=$2 AND status='running' AND started_at < now() - interval '2 minutes'`, workspaceID, cfg.UUID)
	staleCancel()
	var runID pgtype.UUID
	if err := h.DB.QueryRow(r.Context(), `INSERT INTO crm_imap_sync_run (workspace_id, mailbox_id, folder, requested_limit) VALUES ($1,$2,$3,$4) RETURNING id`, workspaceID, cfg.UUID, folder, limit).Scan(&runID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create IMAP sync run")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "run_id": uuidToString(runID), "status": "running"})

	go func(runID pgtype.UUID, workspaceID pgtype.UUID, cfg crmIMAPMailboxConfig, folder string, limit int, rangeDays int) {
		finishRun := func(query string, args ...any) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = h.DB.Exec(ctx, query, args...)
		}
		defer func() {
			if rec := recover(); rec != nil {
				finishRun(`UPDATE crm_imap_sync_run SET status='failed', error_message=$2, finished_at=now(), updated_at=now() WHERE id=$1`, runID, "panic during IMAP sync")
			}
		}()
		messages, err := fetchCRMEmailProviderMessages(cfg, folder, limit, rangeDays, nil)
		if err != nil {
			finishRun(`UPDATE crm_imap_sync_run SET status='failed', error_message=$2, finished_at=now(), updated_at=now() WHERE id=$1`, runID, err.Error())
			return
		}
		importCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		imported, skipped, err := h.importCRMIMAPMessages(importCtx, workspaceID, cfg, folder, messages)
		if err != nil {
			finishRun(`UPDATE crm_imap_sync_run SET status='failed', fetched_count=$2, error_message=$3, finished_at=now(), updated_at=now() WHERE id=$1`, runID, len(messages), err.Error())
			return
		}
		finishRun(`UPDATE crm_imap_sync_run SET status='ok', fetched_count=$2, imported_count=$3, skipped_count=$4, finished_at=now(), updated_at=now() WHERE id=$1`, runID, len(messages), imported, skipped)
	}(runID, workspaceID, cfg, folder, limit, req.RangeDays)
}
func cleanCRMIMAPFolder(folder *string) string {
	if folder == nil {
		return "INBOX"
	}
	value := strings.TrimSpace(*folder)
	if value == "" {
		return "INBOX"
	}
	return value
}

func isCRMEmailSpamFolder(folder string) bool {
	value := strings.ToLower(strings.TrimSpace(folder))
	value = strings.Trim(value, `"`)
	value = strings.ReplaceAll(value, "\\", "/")
	return value == "spam" || value == "junk" || strings.Contains(value, "/spam") || strings.Contains(value, "/junk") || strings.Contains(value, "垃圾") || strings.Contains(value, "垃圾邮件")
}

func canonicalCRMEmailFolder(folder string) string {
	value := strings.TrimSpace(folder)
	if value == "" {
		value = "INBOX"
	}
	if isCRMEmailSpamFolder(value) {
		return "Spam"
	}
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "sent" || strings.Contains(lower, "/sent") || strings.Contains(lower, "已发送") || strings.Contains(lower, "sent messages") {
		return "Sent"
	}
	if lower == "trash" || strings.Contains(lower, "/trash") || strings.Contains(lower, "deleted") || strings.Contains(lower, "废纸") || strings.Contains(lower, "已删除") {
		return "Trash"
	}
	if lower == "archive" || lower == "archived" || strings.Contains(lower, "/archive") || strings.Contains(lower, "归档") {
		return "Archive"
	}
	if strings.EqualFold(value, "INBOX") || strings.Contains(lower, "收件") {
		return "INBOX"
	}
	return value
}

func isCRMIMAPSentFolder(folder string) bool {
	folder = strings.ToLower(strings.TrimSpace(folder))
	return folder == "sent" || folder == "sent messages" || folder == "sent items" || folder == "已发送" || strings.Contains(folder, "sent")
}

func (h *Handler) loadCRMIMAPConfig(w http.ResponseWriter, r *http.Request, workspaceID pgtype.UUID, mailboxIDValue *string) (crmIMAPMailboxConfig, bool) {
	mailboxID, ok := optionalUUID(w, mailboxIDValue, "mailbox_id")
	if !ok {
		return crmIMAPMailboxConfig{}, false
	}
	query := `SELECT id, label, email, host, port, tls_mode, username, secret_ref, owner_type, owner_id, smtp_host, smtp_port, smtp_tls_mode, smtp_username, smtp_secret_ref FROM crm_imap_setting WHERE workspace_id=$1`
	args := []any{workspaceID}
	if mailboxID.Valid {
		query += ` AND id=$2`
		args = append(args, mailboxID)
	}
	query += ` ORDER BY updated_at DESC LIMIT 1`
	var cfg crmIMAPMailboxConfig
	var id pgtype.UUID
	var secretRef, ownerType, smtpHost, smtpTLSMode, smtpUsername, smtpSecretRef pgtype.Text
	var ownerID pgtype.UUID
	var smtpPort pgtype.Int4
	if err := h.DB.QueryRow(r.Context(), query, args...).Scan(&id, &cfg.Label, &cfg.Email, &cfg.Host, &cfg.Port, &cfg.TLSMode, &cfg.Username, &secretRef, &ownerType, &ownerID, &smtpHost, &smtpPort, &smtpTLSMode, &smtpUsername, &smtpSecretRef); err != nil {
		writeError(w, http.StatusNotFound, "CRM IMAP setting not found")
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

func crmIMAPPreviewMessagesToResponse(messages []crmIMAPFetchedMessage) []CRMIMAPPreviewMessageResponse {
	out := make([]CRMIMAPPreviewMessageResponse, 0, len(messages))
	for _, message := range messages {
		var receivedAt *string
		if !message.Date.IsZero() {
			value := message.Date.UTC().Format(time.RFC3339)
			receivedAt = &value
		}
		out = append(out, CRMIMAPPreviewMessageResponse{
			UID: message.UID, ExternalMessageID: message.MessageID, Subject: message.Subject,
			FromEmail: message.FromEmail, FromName: message.FromName, ToEmails: message.ToEmails,
			CcEmails: message.CcEmails, ReceivedAt: receivedAt, Snippet: message.Snippet, RawSize: message.RawSize,
		})
	}
	return out
}

func looksLikeCRMHTML(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "<html") || strings.Contains(value, "<body") || strings.Contains(value, "<div") || strings.Contains(value, "<p") || strings.Contains(value, "<br") || strings.Contains(value, "<table")
}

func normalizeCRMEmailThreadSubject(subject string) string {
	subject = strings.ToLower(strings.TrimSpace(subject))
	for {
		trimmed := strings.TrimSpace(subject)
		lower := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(lower, "re:"):
			subject = strings.TrimSpace(trimmed[3:])
		case strings.HasPrefix(lower, "fw:"):
			subject = strings.TrimSpace(trimmed[3:])
		case strings.HasPrefix(lower, "fwd:"):
			subject = strings.TrimSpace(trimmed[4:])
		default:
			if trimmed == "" {
				return "(no subject)"
			}
			return trimmed
		}
	}
}

func (h *Handler) resolveCRMEmailThreadForImport(ctx context.Context, workspaceID pgtype.UUID, cfg crmIMAPMailboxConfig, message crmIMAPFetchedMessage, subject string) (pgtype.UUID, pgtype.UUID, pgtype.UUID, error) {
	subject = cleanStringForDB(subject)
	message.FromEmail = cleanStringForDB(message.FromEmail)
	matchedAccountID, matchedContactID, err := h.matchCRMEmailParties(ctx, workspaceID, message)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	candidateIDs := normalizeCRMMessageIDSlice(cleanOptionalStringList(append(append([]string{}, message.References...), message.InReplyTo)))
	var threadID, accountID, contactID pgtype.UUID
	for i := len(candidateIDs) - 1; i >= 0; i-- {
		if err := h.DB.QueryRow(ctx, `SELECT m.thread_id, COALESCE(m.account_id,t.account_id,c.account_id), COALESCE(m.contact_id,t.contact_id) FROM crm_email_message m JOIN crm_email_thread t ON t.id=m.thread_id AND t.workspace_id=m.workspace_id LEFT JOIN crm_contact c ON c.id=COALESCE(m.contact_id,t.contact_id) AND c.workspace_id=m.workspace_id WHERE m.workspace_id=$1 AND m.external_message_id=$2 ORDER BY m.created_at DESC LIMIT 1`, workspaceID, candidateIDs[i]).Scan(&threadID, &accountID, &contactID); err == nil {
			if !accountID.Valid && matchedAccountID.Valid {
				if _, err := h.DB.Exec(ctx, `UPDATE crm_email_thread SET account_id=$3, contact_id=$4, updated_at=now() WHERE id=$1 AND workspace_id=$2 AND account_id IS NULL`, threadID, workspaceID, matchedAccountID, matchedContactID); err != nil {
					return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
				}
				return threadID, matchedAccountID, matchedContactID, nil
			}
			return threadID, accountID, contactID, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
		}
	}
	threadKey := cfg.ID + ":subject-from:" + normalizeCRMEmailThreadSubject(subject) + ":" + strings.ToLower(strings.TrimSpace(message.FromEmail))
	if err := h.DB.QueryRow(ctx, `SELECT t.id, COALESCE(t.account_id,c.account_id), t.contact_id FROM crm_email_thread t LEFT JOIN crm_contact c ON c.id=t.contact_id AND c.workspace_id=t.workspace_id WHERE t.workspace_id=$1 AND t.external_thread_id=$2 LIMIT 1`, workspaceID, threadKey).Scan(&threadID, &accountID, &contactID); err == nil {
		if !accountID.Valid && matchedAccountID.Valid {
			if _, err := h.DB.Exec(ctx, `UPDATE crm_email_thread SET account_id=$3, contact_id=$4, updated_at=now() WHERE id=$1 AND workspace_id=$2 AND account_id IS NULL`, threadID, workspaceID, matchedAccountID, matchedContactID); err != nil {
				return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
			}
			return threadID, matchedAccountID, matchedContactID, nil
		}
		return threadID, accountID, contactID, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	accountID, contactID = matchedAccountID, matchedContactID
	lastAt := pgtype.Timestamptz{}
	if !message.Date.IsZero() {
		lastAt = pgtype.Timestamptz{Time: message.Date, Valid: true}
	}
	if err := h.DB.QueryRow(ctx, `INSERT INTO crm_email_thread (workspace_id, account_id, contact_id, subject, external_thread_id, mailbox, direction, status, last_message_at) VALUES ($1,$2,$3,$4,$5,$6,'inbound','open',$7) RETURNING id`, workspaceID, accountID, contactID, subject, threadKey, cfg.Email, lastAt).Scan(&threadID); err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	return threadID, accountID, contactID, nil
}

func (h *Handler) matchCRMEmailParties(ctx context.Context, workspaceID pgtype.UUID, message crmIMAPFetchedMessage) (pgtype.UUID, pgtype.UUID, error) {
	emails := []string{message.FromEmail}
	emails = append(emails, message.ToEmails...)
	emails = append(emails, message.CcEmails...)
	seen := map[string]bool{}
	for _, email := range emails {
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" || seen[email] {
			continue
		}
		seen[email] = true
		accountID, contactID, err := h.matchCRMEmailContact(ctx, workspaceID, email)
		if err != nil {
			return pgtype.UUID{}, pgtype.UUID{}, err
		}
		if accountID.Valid || contactID.Valid {
			return accountID, contactID, nil
		}
	}
	return pgtype.UUID{}, pgtype.UUID{}, nil
}

func (h *Handler) matchCRMEmailContact(ctx context.Context, workspaceID pgtype.UUID, email string) (pgtype.UUID, pgtype.UUID, error) {
	var accountID, contactID pgtype.UUID
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return accountID, contactID, nil
	}
	if err := h.DB.QueryRow(ctx, `SELECT account_id, id FROM crm_contact WHERE workspace_id=$1 AND lower(email)=$2 ORDER BY is_primary DESC, updated_at DESC LIMIT 1`, workspaceID, email).Scan(&accountID, &contactID); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return pgtype.UUID{}, pgtype.UUID{}, err
		}
	} else {
		return accountID, contactID, nil
	}
	domain := emailDomain(email)
	if domain == "" || isGenericCRMEmailDomain(domain) {
		return pgtype.UUID{}, pgtype.UUID{}, nil
	}
	if err := h.DB.QueryRow(ctx, `
		SELECT a.id
		FROM crm_account a
		WHERE a.workspace_id=$1
		  AND (
			lower(regexp_replace(COALESCE(a.website,''), '^https?://(www\.)?', '')) LIKE $2
			OR EXISTS (SELECT 1 FROM crm_contact c WHERE c.workspace_id=a.workspace_id AND c.account_id=a.id AND lower(split_part(c.email,'@',2))=$3)
		  )
		ORDER BY a.updated_at DESC
		LIMIT 1`, workspaceID, "%"+domain+"%", domain).Scan(&accountID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgtype.UUID{}, pgtype.UUID{}, nil
		}
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	return accountID, pgtype.UUID{}, nil
}

func (h *Handler) importCRMIMAPMessages(ctx context.Context, workspaceID pgtype.UUID, cfg crmIMAPMailboxConfig, folder string, messages []crmIMAPFetchedMessage) (int, int, error) {
	imported := 0
	skipped := 0
	for _, message := range messages {
		externalID := cleanStringForDB(message.MessageID)
		if externalID == "" {
			externalID = cfg.ID + ":" + message.UID
		}
		var exists bool
		if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM crm_email_message WHERE workspace_id=$1 AND external_message_id=$2)`, workspaceID, externalID).Scan(&exists); err != nil {
			return imported, skipped, err
		}
		if exists {
			skipped++
			continue
		}
		subject := strings.TrimSpace(message.Subject)
		if subject == "" {
			subject = "(no subject)"
		}
		direction := "inbound"
		if isCRMIMAPSentFolder(folder) {
			direction = "outbound"
		}
		threadID, accountID, contactID, err := h.resolveCRMEmailThreadForImport(ctx, workspaceID, cfg, message, subject)
		if err != nil {
			return imported, skipped, err
		}
		receivedAt := pgtype.Timestamptz{}
		if !message.Date.IsZero() {
			receivedAt = pgtype.Timestamptz{Time: message.Date, Valid: true}
		}
		rawHeadersJSON, _ := json.Marshal(message.RawHeaders)
		attachmentsJSON, _ := json.Marshal(message.Attachments)
		bodyHTML := message.BodyHTML
		if strings.TrimSpace(bodyHTML) == "" && looksLikeCRMHTML(message.BodyText) {
			bodyHTML = message.BodyText
		}
		canonicalFolder := canonicalCRMEmailFolder(folder)
		isRead := strings.EqualFold(canonicalFolder, "Sent")
		sourceMetadataJSON, _ := json.Marshal(map[string]any{"provider": "imap", "mailbox_id": cfg.ID, "folder": canonicalFolder, "source_folder": folder, "uid": message.UID})
		_, execErr := h.DB.Exec(ctx, `INSERT INTO crm_email_message (workspace_id, thread_id, account_id, contact_id, external_message_id, in_reply_to, reference_ids, from_email, from_name, to_emails, cc_emails, subject, received_at, body_text, body_html, snippet, raw_size_bytes, raw_headers, attachments, direction, folder, is_read, source_metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)`, workspaceID, threadID, accountID, contactID, externalID, cleanOptionalText(&message.InReplyTo), cleanOptionalStringList(message.References), cleanOptionalText(&message.FromEmail), cleanOptionalText(&message.FromName), cleanOptionalStringList(message.ToEmails), cleanOptionalStringList(message.CcEmails), cleanOptionalText(&subject), receivedAt, cleanOptionalText(&message.BodyText), cleanOptionalText(&bodyHTML), cleanOptionalText(&message.Snippet), message.RawSize, rawHeadersJSON, attachmentsJSON, direction, canonicalFolder, isRead, sourceMetadataJSON)
		if execErr != nil {
			return imported, skipped, execErr
		}
		_, _ = h.DB.Exec(ctx, `UPDATE crm_email_thread SET account_id=COALESCE(account_id,$5), contact_id=COALESCE(contact_id,$6), last_message_at=COALESCE($3,last_message_at,now()), direction=CASE WHEN $4='outbound' THEN 'outbound' WHEN direction='outbound' THEN 'mixed' ELSE direction END, updated_at=now() WHERE id=$1 AND workspace_id=$2`, threadID, workspaceID, receivedAt, direction, accountID, contactID)
		if accountID.Valid {
			if shouldAutoRefreshCRMAccountProfile(ctx, h.DB, workspaceID) {
				if _, err := h.regenerateCRMAccountProfile(ctx, workspaceID, accountID); err != nil {
					slog.Warn("CRM account profile auto refresh failed after new email", "account_id", uuidToString(accountID), "error", err)
				}
			}
		}
		imported++
	}
	return imported, skipped, nil
}

func (h *Handler) ListCRMEmailDrafts(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	draftIDFilter := strings.TrimSpace(r.URL.Query().Get("draft_id"))
	rows, err := h.DB.Query(r.Context(), `SELECT id, mailbox_id, thread_id, account_id, contact_id, to_emails, cc_emails, bcc_emails, subject, body_text, status, ai_generated, created_at, updated_at FROM crm_email_draft WHERE workspace_id=$1 ORDER BY CASE WHEN $2 <> '' AND id::text = $2 THEN 0 ELSE 1 END, updated_at DESC LIMIT 100`, workspaceID, draftIDFilter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list CRM email drafts")
		return
	}
	defer rows.Close()
	items := []CRMEmailDraftResponse{}
	for rows.Next() {
		var id, mailboxID, threadID, accountID, contactID pgtype.UUID
		var subject, body, status string
		var toEmails, ccEmails, bccEmails []string
		var ai bool
		var created, updated pgtype.Timestamptz
		if err := rows.Scan(&id, &mailboxID, &threadID, &accountID, &contactID, &toEmails, &ccEmails, &bccEmails, &subject, &body, &status, &ai, &created, &updated); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan CRM email draft")
			return
		}
		items = append(items, CRMEmailDraftResponse{ID: uuidToString(id), MailboxID: uuidToPtr(mailboxID), ThreadID: uuidToPtr(threadID), AccountID: uuidToPtr(accountID), ContactID: uuidToPtr(contactID), ToEmails: toEmails, CcEmails: ccEmails, BccEmails: bccEmails, Subject: subject, BodyText: body, Status: status, AIGenerated: ai, CreatedAt: timestampToString(created), UpdatedAt: timestampToString(updated)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"drafts": items, "total": len(items)})
}

func (h *Handler) CreateCRMEmailDraft(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	var req CreateCRMEmailDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	mailboxID, ok := optionalUUID(w, req.MailboxID, "mailbox_id")
	if !ok {
		return
	}
	threadID, ok := optionalUUID(w, req.ThreadID, "thread_id")
	if !ok {
		return
	}
	accountID, ok := optionalUUID(w, req.AccountID, "account_id")
	if !ok {
		return
	}
	contactID, ok := optionalUUID(w, req.ContactID, "contact_id")
	if !ok {
		return
	}
	issueID, ok := optionalUUID(w, req.IssueID, "issue_id")
	if !ok {
		return
	}
	if req.ToEmails == nil {
		req.ToEmails = []string{}
	}
	if req.CcEmails == nil {
		req.CcEmails = []string{}
	}
	if req.BccEmails == nil {
		req.BccEmails = []string{}
	}
	if req.ReferenceIDs == nil {
		req.ReferenceIDs = []string{}
	}
	if req.Attachments == nil {
		req.Attachments = []crmEmailAttachment{}
	}
	attachmentsJSON, err := json.Marshal(req.Attachments)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid attachments")
		return
	}
	sentAppendEnabled := true
	if req.SentAppendEnabled != nil {
		sentAppendEnabled = *req.SentAppendEnabled
	}
	var id pgtype.UUID
	if err := h.DB.QueryRow(r.Context(), `INSERT INTO crm_email_draft (workspace_id, mailbox_id, thread_id, account_id, contact_id, issue_id, to_emails, cc_emails, bcc_emails, subject, body_text, body_html, in_reply_to, reference_ids, attachments, sent_append_enabled, ai_generated, approval_reason) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18) RETURNING id`, workspaceID, mailboxID, threadID, accountID, contactID, issueID, req.ToEmails, req.CcEmails, req.BccEmails, req.Subject, req.BodyText, cleanOptionalText(&req.BodyHTML), cleanOptionalText(&req.InReplyTo), req.ReferenceIDs, attachmentsJSON, sentAppendEnabled, req.AIGenerated, cleanOptionalText(&req.ApprovalReason)).Scan(&id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create CRM email draft")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": uuidToString(id)})
}

func (h *Handler) UpdateCRMEmailDraft(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	draftID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "draftId"), "draft id")
	if !ok {
		return
	}
	var req CreateCRMEmailDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	mailboxID, ok := optionalUUID(w, req.MailboxID, "mailbox_id")
	if !ok {
		return
	}
	threadID, ok := optionalUUID(w, req.ThreadID, "thread_id")
	if !ok {
		return
	}
	accountID, ok := optionalUUID(w, req.AccountID, "account_id")
	if !ok {
		return
	}
	contactID, ok := optionalUUID(w, req.ContactID, "contact_id")
	if !ok {
		return
	}
	issueID, ok := optionalUUID(w, req.IssueID, "issue_id")
	if !ok {
		return
	}
	if req.ToEmails == nil {
		req.ToEmails = []string{}
	}
	if req.CcEmails == nil {
		req.CcEmails = []string{}
	}
	if req.BccEmails == nil {
		req.BccEmails = []string{}
	}
	if req.ReferenceIDs == nil {
		req.ReferenceIDs = []string{}
	}
	if req.Attachments == nil {
		req.Attachments = []crmEmailAttachment{}
	}
	attachmentsJSON, err := json.Marshal(req.Attachments)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid attachments")
		return
	}
	if _, err := h.DB.Exec(r.Context(), `UPDATE crm_email_draft SET mailbox_id=$3, thread_id=$4, account_id=$5, contact_id=$6, issue_id=$7, to_emails=$8, cc_emails=$9, bcc_emails=$10, subject=$11, body_text=$12, body_html=$13, in_reply_to=$14, reference_ids=$15, attachments=$16, approval_reason=$17, updated_at=now() WHERE id=$1 AND workspace_id=$2 AND status <> 'sent'`, draftID, workspaceID, mailboxID, threadID, accountID, contactID, issueID, req.ToEmails, req.CcEmails, req.BccEmails, req.Subject, req.BodyText, cleanOptionalText(&req.BodyHTML), cleanOptionalText(&req.InReplyTo), req.ReferenceIDs, attachmentsJSON, cleanOptionalText(&req.ApprovalReason)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update CRM email draft")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": uuidToString(draftID)})
}

func (h *Handler) sendFirstPendingCRMEmailDraftForIssue(ctx context.Context, workspaceID, issueID pgtype.UUID) (pgtype.UUID, error) {
	var draftID, mailboxID, threadID, accountID, contactID pgtype.UUID
	var payload crmEmailSendPayload
	var attachmentsJSON []byte
	if err := h.DB.QueryRow(ctx, `UPDATE crm_email_draft SET status='sending', updated_at=now() WHERE id=(SELECT id FROM crm_email_draft WHERE workspace_id=$1 AND issue_id=$2 AND status IN ('pending_approval','draft','failed') AND sent_at IS NULL ORDER BY updated_at DESC LIMIT 1) RETURNING id, mailbox_id, thread_id, account_id, contact_id, to_emails, cc_emails, bcc_emails, subject, body_text, COALESCE(body_html,''), COALESCE(in_reply_to,''), reference_ids, attachments, sent_append_enabled`, workspaceID, issueID).Scan(&draftID, &mailboxID, &threadID, &accountID, &contactID, &payload.ToEmails, &payload.CcEmails, &payload.BccEmails, &payload.Subject, &payload.BodyText, &payload.BodyHTML, &payload.InReplyTo, &payload.ReferenceIDs, &attachmentsJSON, &payload.AppendToSent); err != nil {
		return draftID, err
	}
	if len(attachmentsJSON) > 0 {
		_ = json.Unmarshal(attachmentsJSON, &payload.Attachments)
	}
	var cfg crmIMAPMailboxConfig
	var id pgtype.UUID
	var secretRef, ownerType, smtpHost, smtpTLSMode, smtpUsername, smtpSecretRef pgtype.Text
	var ownerID pgtype.UUID
	var smtpPort pgtype.Int4
	query := `SELECT id, label, email, host, port, tls_mode, username, secret_ref, owner_type, owner_id, smtp_host, smtp_port, smtp_tls_mode, smtp_username, smtp_secret_ref FROM crm_imap_setting WHERE workspace_id=$1`
	args := []any{workspaceID}
	if mailboxID.Valid {
		query += ` AND id=$2`
		args = append(args, mailboxID)
	}
	query += ` ORDER BY updated_at DESC LIMIT 1`
	if err := h.DB.QueryRow(ctx, query, args...).Scan(&id, &cfg.Label, &cfg.Email, &cfg.Host, &cfg.Port, &cfg.TLSMode, &cfg.Username, &secretRef, &ownerType, &ownerID, &smtpHost, &smtpPort, &smtpTLSMode, &smtpUsername, &smtpSecretRef); err != nil {
		_, _ = h.DB.Exec(ctx, `UPDATE crm_email_draft SET status='failed', error_message=$3, updated_at=now() WHERE id=$1 AND workspace_id=$2`, draftID, workspaceID, err.Error())
		return draftID, err
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
	messageID, rawMessage, sentAt, err := sendCRMEmailProvider(cfg, payload)
	if err != nil {
		_, _ = h.DB.Exec(ctx, `UPDATE crm_email_draft SET status='failed', error_message=$3, updated_at=now() WHERE id=$1 AND workspace_id=$2`, draftID, workspaceID, err.Error())
		return draftID, err
	}
	appendWarning := ""
	if payload.AppendToSent && len(rawMessage) > 0 {
		if err := appendCRMIMAPSentMessage(cfg, rawMessage, sentAt); err != nil {
			appendWarning = "Sent folder append failed: " + sanitizeCRMSendError(err).Error()
		}
	}
	if !threadID.Valid {
		if err := h.DB.QueryRow(ctx, `INSERT INTO crm_email_thread (workspace_id, account_id, contact_id, subject, mailbox, direction, status, last_message_at, message_count) VALUES ($1,$2,$3,$4,$5,'outbound','open',now(),0) RETURNING id`, workspaceID, accountID, contactID, payload.Subject, cleanStringForDB(cfg.Email)).Scan(&threadID); err != nil {
			_, _ = h.DB.Exec(ctx, `UPDATE crm_email_draft SET status='failed', error_message=$3, updated_at=now() WHERE id=$1 AND workspace_id=$2`, draftID, workspaceID, err.Error())
			return draftID, err
		}
	}
	_, _ = h.DB.Exec(ctx, `INSERT INTO crm_email_message (workspace_id, thread_id, account_id, contact_id, direction, external_message_id, from_email, to_emails, cc_emails, bcc_emails, subject, body_text, body_html, in_reply_to, reference_ids, attachments, sent_append_warning, sent_at) VALUES ($1,$2,$3,$4,'outbound',$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17); UPDATE crm_email_thread SET direction='outbound', status='open', mailbox=$18, last_message_at=$17, message_count=message_count+1, updated_at=now() WHERE id=$2 AND workspace_id=$1`, workspaceID, threadID, accountID, contactID, messageID, cfg.Email, payload.ToEmails, payload.CcEmails, payload.BccEmails, payload.Subject, payload.BodyText, cleanOptionalText(&payload.BodyHTML), cleanOptionalText(&payload.InReplyTo), payload.ReferenceIDs, attachmentsJSON, cleanOptionalText(&appendWarning), sentAt, cleanStringForDB(cfg.Email))
	_, _ = h.DB.Exec(ctx, `UPDATE crm_email_draft SET status='sent', thread_id=$3, sent_at=$4, sent_append_warning=$5, error_message=NULL, updated_at=now() WHERE id=$1 AND workspace_id=$2`, draftID, workspaceID, threadID, sentAt, cleanOptionalText(&appendWarning))
	return draftID, h.markCRMEmailDraftIssueSent(ctx, workspaceID, issueID, draftID, "Issue 评论确认通过，系统已自动发送绑定草稿邮件，并将 Issue 标记为 done。")
}

func (h *Handler) SendCRMEmailDraft(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	draftID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "draftId"), "draft_id")
	if !ok {
		return
	}
	var mailboxID, threadID, accountID, contactID, issueID pgtype.UUID
	var payload crmEmailSendPayload
	var attachmentsJSON []byte
	if err := h.DB.QueryRow(r.Context(), `SELECT mailbox_id, thread_id, account_id, contact_id, issue_id, to_emails, cc_emails, bcc_emails, subject, body_text, COALESCE(body_html,''), COALESCE(in_reply_to,''), reference_ids, attachments, sent_append_enabled FROM crm_email_draft WHERE id=$1 AND workspace_id=$2 AND status <> 'sent'`, draftID, workspaceID).Scan(&mailboxID, &threadID, &accountID, &contactID, &issueID, &payload.ToEmails, &payload.CcEmails, &payload.BccEmails, &payload.Subject, &payload.BodyText, &payload.BodyHTML, &payload.InReplyTo, &payload.ReferenceIDs, &attachmentsJSON, &payload.AppendToSent); err != nil {
		writeError(w, http.StatusNotFound, "CRM email draft not found")
		return
	}
	if len(attachmentsJSON) > 0 {
		_ = json.Unmarshal(attachmentsJSON, &payload.Attachments)
	}
	mailboxIDString := uuidToString(mailboxID)
	cfg, ok := h.loadCRMIMAPConfig(w, r, workspaceID, &mailboxIDString)
	if !ok {
		return
	}
	messageID, rawMessage, sentAt, err := sendCRMEmailProvider(cfg, payload)
	if err != nil {
		_, _ = h.DB.Exec(r.Context(), `UPDATE crm_email_draft SET status='failed', error_message=$3, updated_at=now() WHERE id=$1 AND workspace_id=$2`, draftID, workspaceID, err.Error())
		writeError(w, http.StatusBadGateway, "failed to send CRM email draft: "+err.Error())
		return
	}
	appendWarning := ""
	if payload.AppendToSent {
		if len(rawMessage) == 0 {
			appendWarning = "Sent folder append skipped because SMTP provider did not return raw RFC822 message"
		} else if err := appendCRMIMAPSentMessage(cfg, rawMessage, sentAt); err != nil {
			appendWarning = "Sent folder append failed: " + sanitizeCRMSendError(err).Error()
		}
		if appendWarning != "" {
			slog.Warn("CRM sent append failed", "draft_id", uuidToString(draftID), "workspace_id", uuidToString(workspaceID), "warning", appendWarning)
		}
	}
	if !threadID.Valid {
		mailboxEmail := cleanStringForDB(cfg.Email)
		if err := h.DB.QueryRow(r.Context(), `INSERT INTO crm_email_thread (workspace_id, account_id, contact_id, subject, mailbox, direction, status, last_message_at, message_count) VALUES ($1,$2,$3,$4,$5,'outbound','open',now(),0) RETURNING id`, workspaceID, accountID, contactID, payload.Subject, mailboxEmail).Scan(&threadID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create CRM sent email thread")
			return
		}
	}
	_, _ = h.DB.Exec(r.Context(), `INSERT INTO crm_email_message (workspace_id, thread_id, account_id, contact_id, direction, external_message_id, from_email, to_emails, cc_emails, bcc_emails, subject, body_text, body_html, in_reply_to, reference_ids, attachments, sent_append_warning, sent_at) VALUES ($1,$2,$3,$4,'outbound',$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17); UPDATE crm_email_thread SET direction='outbound', status='open', mailbox=$18, last_message_at=$17, message_count=message_count+1, updated_at=now() WHERE id=$2 AND workspace_id=$1`, workspaceID, threadID, accountID, contactID, messageID, cfg.Email, payload.ToEmails, payload.CcEmails, payload.BccEmails, payload.Subject, payload.BodyText, cleanOptionalText(&payload.BodyHTML), cleanOptionalText(&payload.InReplyTo), payload.ReferenceIDs, attachmentsJSON, cleanOptionalText(&appendWarning), sentAt, cleanStringForDB(cfg.Email))
	_, _ = h.DB.Exec(r.Context(), `UPDATE crm_email_draft SET status='sent', thread_id=$3, sent_at=$5, sent_append_warning=$4, updated_at=now() WHERE id=$1 AND workspace_id=$2`, draftID, workspaceID, threadID, cleanOptionalText(&appendWarning), sentAt)
	if issueID.Valid {
		_ = h.addCRMInternalIssueComment(r.Context(), workspaceID, issueID, "绑定邮件草稿已手动修改并发送。\n\n草稿 ID："+uuidToString(draftID)+"\n\n请负责人确认客户跟进已完成后，将此 Issue 状态改为 done。")
	}
	if accountID.Valid && shouldAutoRefreshCRMAccountProfile(r.Context(), h.DB, workspaceID) {
		if _, err := h.regenerateCRMAccountProfile(r.Context(), workspaceID, accountID); err != nil {
			slog.Warn("CRM account profile auto refresh failed after sent email", "account_id", uuidToString(accountID), "error", err)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "sent", "message_id": messageID, "sent_append_warning": appendWarning})
}

func crmTextValue(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return strings.TrimSpace(t.String)
}

func (h *Handler) regenerateCRMAccountProfile(ctx context.Context, workspaceID, accountID pgtype.UUID) (CRMAccountProfileResponse, error) {
	var name, status, rating, priority string
	var website, countryName, industry, notes pgtype.Text
	if err := h.DB.QueryRow(ctx, `SELECT name, status, rating, priority, website, country_name, industry, notes FROM crm_account WHERE id=$1 AND workspace_id=$2`, accountID, workspaceID).Scan(&name, &status, &rating, &priority, &website, &countryName, &industry, &notes); err != nil {
		return CRMAccountProfileResponse{}, err
	}
	rows, err := h.DB.Query(ctx, `SELECT channel, direction, COALESCE(subject,''), body FROM crm_communication_note WHERE account_id=$1 AND workspace_id=$2 ORDER BY occurred_at DESC, created_at DESC LIMIT 5`, accountID, workspaceID)
	if err != nil {
		return CRMAccountProfileResponse{}, err
	}
	defer rows.Close()
	noteSnippets := make([]string, 0, 5)
	for rows.Next() {
		var channel, direction, subject, body string
		if err := rows.Scan(&channel, &direction, &subject, &body); err == nil {
			noteSnippets = append(noteSnippets, trimCRMProfileSnippet(strings.TrimSpace(strings.Join([]string{channel, direction, subject, body}, " ")), 220))
		}
	}
	rows, err = h.DB.Query(ctx, `SELECT direction, COALESCE(subject,''), COALESCE(body_text,'') FROM crm_email_message WHERE workspace_id=$1 AND account_id=$2 ORDER BY COALESCE(sent_at, received_at, created_at) DESC LIMIT 5`, workspaceID, accountID)
	if err != nil {
		return CRMAccountProfileResponse{}, err
	}
	defer rows.Close()
	emailSnippets := make([]string, 0, 5)
	for rows.Next() {
		var direction, subject, body string
		if err := rows.Scan(&direction, &subject, &body); err == nil {
			emailSnippets = append(emailSnippets, trimCRMProfileSnippet(strings.TrimSpace(strings.Join([]string{"email", direction, subject, body}, " ")), 260))
		}
	}
	rows, err = h.DB.Query(ctx, `SELECT DISTINCT p.title FROM project p JOIN project_resource pr ON pr.project_id=p.id AND pr.workspace_id=p.workspace_id WHERE p.workspace_id=$1 AND pr.resource_type='crm_account' AND pr.resource_ref->>'account_id'=$2 ORDER BY p.title ASC LIMIT 5`, workspaceID, uuidToString(accountID))
	if err != nil {
		return CRMAccountProfileResponse{}, err
	}
	defer rows.Close()
	projectTitles := make([]string, 0, 5)
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err == nil {
			projectTitles = append(projectTitles, title)
		}
	}
	rows, err = h.DB.Query(ctx, `SELECT DISTINCT COALESCE(NULLIF(i.number, 0)::text, ''), i.title, i.status, i.priority FROM issue i LEFT JOIN crm_email_thread t ON t.issue_id=i.id AND t.workspace_id=i.workspace_id LEFT JOIN crm_email_thread_issue_link til ON til.issue_id=i.id LEFT JOIN crm_email_thread lt ON lt.id=til.thread_id AND lt.workspace_id=i.workspace_id LEFT JOIN project_resource pr ON pr.project_id=i.project_id AND pr.workspace_id=i.workspace_id AND pr.resource_type='crm_account' AND pr.resource_ref->>'account_id'=$2 WHERE i.workspace_id=$1 AND (t.account_id=$3 OR lt.account_id=$3 OR pr.id IS NOT NULL) ORDER BY i.title ASC LIMIT 5`, workspaceID, uuidToString(accountID), accountID)
	if err != nil {
		return CRMAccountProfileResponse{}, err
	}
	defer rows.Close()
	issueTitles := make([]string, 0, 5)
	for rows.Next() {
		var identifier, title, issueStatus, issuePriority string
		if err := rows.Scan(&identifier, &title, &issueStatus, &issuePriority); err == nil {
			issueTitles = append(issueTitles, strings.TrimSpace(strings.Join([]string{identifier, title, issueStatus, issuePriority}, " ")))
		}
	}
	communicationSnippets := append([]string{}, noteSnippets...)
	communicationSnippets = append(communicationSnippets, emailSnippets...)
	country := crmTextValue(countryName)
	industryValue := crmTextValue(industry)
	baseParts := []string{name}
	if industryValue != "" {
		baseParts = append(baseParts, industryValue)
	}
	if country != "" {
		baseParts = append(baseParts, country)
	}
	summary := strings.TrimSpace(strings.Join(baseParts, " · "))
	if summary == "" {
		summary = "CRM customer profile"
	}
	projectSource := strings.Join(trimCRMProfileList(projectTitles, 5, 120), "\n")
	issueSource := strings.Join(trimCRMProfileList(issueTitles, 5, 160), "\n")
	emailEvidence := strings.Join(trimCRMProfileList(emailSnippets, 5, 160), "\n")
	noteEvidence := strings.Join(trimCRMProfileList(noteSnippets, 5, 160), "\n")
	businessModel := strings.TrimSpace(strings.Join([]string{industryValue, crmTextValue(website)}, " "))
	if businessModel == "" {
		businessModel = profileValueFromEvidence(emailSnippets, noteSnippets, "业务模式", "当前客户基础资料中没有明确业务模式。")
	}
	profileEvidenceSummary := summarizeCRMProfileEvidence(emailSnippets, noteSnippets)
	mainProducts := profileValueFromEvidence(emailSnippets, noteSnippets, "主营/关注产品", "请从后续邮件、报价、样品和订单中提炼客户主营/采购产品。")
	procurementNeeds := profileValueFromEvidence(emailSnippets, noteSnippets, "采购需求", "重点补齐需求产品、数量、目标价格、交期、认证要求、物流方式和采购频率。")
	painPoints := profileValueFromEvidence(emailSnippets, noteSnippets, "痛点/关注点", "当前往来未明确质量、价格、交期、付款、认证、沟通或售后痛点。")
	decisionProcess := profileValueFromEvidence(emailSnippets, noteSnippets, "决策链", "需识别询价人、技术确认人、采购负责人、财务/老板审批人和采购周期。")
	communicationPreference := profileValueFromEvidence(emailSnippets, noteSnippets, "沟通偏好", "根据回复速度、常用邮箱/WhatsApp/电话、语言和时区继续观察。")
	profile := map[string]any{
		"customer_summary":         strings.TrimSpace(strings.Join([]string{summary, profileEvidenceSummary}, "\n")),
		"business_model":           businessModel,
		"main_products":            mainProducts,
		"procurement_needs":        procurementNeeds,
		"pain_points":              painPoints,
		"decision_process":         decisionProcess,
		"communication_preference": communicationPreference,
		"recent_progress":          strings.TrimSpace(strings.Join([]string{projectSource, issueSource, profileEvidenceSummary}, "\n")),
		"risk_notes":               strings.TrimSpace(strings.Join([]string{crmTextValue(notes), "自动画像优先沉淀可从往来记录佐证的结论；原始片段仍保留在 evidence。"}, "\n")),
		"cooperation_history":      strings.TrimSpace(strings.Join([]string{projectSource, issueSource, profileEvidenceSummary}, "\n")),
		"next_step_suggestions":    buildCRMProfileNextSteps(mainProducts, procurementNeeds, decisionProcess),
		"evidence": map[string]any{
			"recent_email_snippets": emailEvidence,
			"recent_note_snippets":  noteEvidence,
			"projects":              projectSource,
			"issues":                issueSource,
		},
		"tags":           []string{status, rating, priority},
		"rating_hint":    rating,
		"priority_hint":  priority,
		"status_hint":    status,
		"auto_generated": true,
	}
	if llmProfile, err := h.generateCRMAccountProfileWithLLM(ctx, name, summary, projectSource, issueSource, emailEvidence, noteEvidence, crmTextValue(notes)); err == nil && len(llmProfile) > 0 {
		for key, value := range llmProfile {
			profile[key] = value
		}
		profile["auto_generated"] = true
		profile["generated_by"] = "llm"
	} else if err != nil {
		slog.Warn("CRM account profile LLM generation failed; using deterministic fallback", "account_id", uuidToString(accountID), "error", err)
	}
	profile = cleanCRMProfile(profile)
	profileJSON, _ := json.Marshal(profile)
	var id pgtype.UUID
	var rawProfile []byte
	var createdAt, updatedAt pgtype.Timestamptz
	if err := h.DB.QueryRow(ctx, `INSERT INTO crm_account_profile (workspace_id, account_id, summary, profile_json, updated_at) VALUES ($1,$2,$3,$4,now()) ON CONFLICT (account_id) DO UPDATE SET summary=EXCLUDED.summary, profile_json=EXCLUDED.profile_json, updated_at=now() RETURNING id, profile_json, created_at, updated_at`, workspaceID, accountID, summary, profileJSON).Scan(&id, &rawProfile, &createdAt, &updatedAt); err != nil {
		return CRMAccountProfileResponse{}, err
	}
	return CRMAccountProfileResponse{ID: uuidToString(id), WorkspaceID: uuidToString(workspaceID), AccountID: uuidToString(accountID), Summary: &summary, ProfileJSON: rawProfile, SourceSummary: buildCRMProfileSourceSummary(trimCRMProfileList(communicationSnippets, 5, 160), trimCRMProfileList(projectTitles, 5, 120), trimCRMProfileList(issueTitles, 5, 120)), CreatedAt: timestampToString(createdAt), UpdatedAt: timestampToString(updatedAt)}, nil
}

func (h *Handler) SuggestCRMAccountProfile(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	profile, err := h.regenerateCRMAccountProfile(r.Context(), workspaceID, accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "CRM account not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to generate CRM profile")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *Handler) ApplyCRMAccountProfileSuggestion(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	suggestionID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "suggestionId"), "suggestion id")
	if !ok {
		return
	}
	var summary pgtype.Text
	var profile json.RawMessage
	if err := h.DB.QueryRow(r.Context(), `SELECT summary, profile_json FROM crm_profile_suggestion WHERE id=$1 AND workspace_id=$2 AND account_id=$3 AND status='draft'`, suggestionID, workspaceID, accountID).Scan(&summary, &profile); err != nil {
		writeError(w, http.StatusNotFound, "CRM profile suggestion not found")
		return
	}
	_, err := h.DB.Exec(r.Context(), `INSERT INTO crm_account_profile (workspace_id, account_id, summary, profile_json, updated_at) VALUES ($1,$2,$3,$4,now()) ON CONFLICT (account_id) DO UPDATE SET summary=EXCLUDED.summary, profile_json=crm_account_profile.profile_json || EXCLUDED.profile_json, updated_at=now(); UPDATE crm_profile_suggestion SET status='applied', applied_at=now() WHERE id=$5 AND workspace_id=$1`, workspaceID, accountID, summary, profile, suggestionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to apply CRM profile suggestion")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func optionalStringFromQuery(r *http.Request, key string) *string {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil
	}
	return &value
}

func (h *Handler) CreateCRMEmailThread(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	var req CreateCRMEmailThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	subject := normalizeCRMName(req.Subject)
	if subject == "" {
		writeError(w, http.StatusBadRequest, "subject is required")
		return
	}
	accountID, ok := optionalUUID(w, req.AccountID, "account_id")
	if !ok {
		return
	}
	contactID, ok := optionalUUID(w, req.ContactID, "contact_id")
	if !ok {
		return
	}
	lastMessageAt, ok := cleanOptionalTimestamp(w, req.LastMessageAt, "last_message_at")
	if !ok {
		return
	}
	thread, err := h.scanCRMEmailThread(h.DB.QueryRow(r.Context(), `
		INSERT INTO crm_email_thread (workspace_id, account_id, contact_id, subject, external_thread_id, mailbox, direction, status, last_message_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, workspace_id, account_id, contact_id, project_id, issue_id, subject, external_thread_id, mailbox, direction, status, last_message_at, NULL::text AS last_snippet, created_at, updated_at, 0::bigint, is_read, is_starred, is_trashed
	`, workspaceID, accountID, contactID, subject, cleanOptionalText(req.ExternalThreadID), cleanOptionalText(req.Mailbox), cleanDefault(req.Direction, "inbound"), cleanDefault(req.Status, "open"), lastMessageAt))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create CRM email thread")
		return
	}
	writeJSON(w, http.StatusCreated, crmEmailThreadToResponse(thread))
}

func (h *Handler) ListCRMEmailMessages(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	threadID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "threadId"), "thread id")
	if !ok {
		return
	}
	if _, ok := h.getCRMEmailThread(w, r, threadID, workspaceID); !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT id, workspace_id, thread_id, account_id, contact_id, external_message_id,
		       in_reply_to, reference_ids, COALESCE((SELECT jsonb_agg(elem - 'content' - 'data' - 'body') FROM jsonb_array_elements(COALESCE(attachments, '[]'::jsonb)) AS elem), '[]'::jsonb), sent_append_warning,
		       raw_size_bytes, '{}'::jsonb, from_email, from_name,
		       to_emails, cc_emails, bcc_emails, subject,
		       sent_at, received_at, body_text, body_html, snippet, direction,
		       created_at, updated_at
		FROM crm_email_message
		WHERE workspace_id = $1 AND thread_id = $2
		ORDER BY COALESCE(sent_at, received_at, created_at) ASC
	`, workspaceID, threadID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list CRM email messages")
		return
	}
	defer rows.Close()
	messages := []CRMEmailMessageResponse{}
	for rows.Next() {
		message, err := h.scanCRMEmailMessage(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan CRM email message")
			return
		}
		messages = append(messages, crmEmailMessageToResponse(message))
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": messages, "total": len(messages)})
}

func (h *Handler) CreateCRMEmailMessage(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	threadID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "threadId"), "thread id")
	if !ok {
		return
	}
	thread, ok := h.getCRMEmailThread(w, r, threadID, workspaceID)
	if !ok {
		return
	}
	var req CreateCRMEmailMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	direction := strings.TrimSpace(req.Direction)
	if direction == "" {
		writeError(w, http.StatusBadRequest, "direction is required")
		return
	}
	accountID, ok := optionalUUID(w, req.AccountID, "account_id")
	if !ok {
		return
	}
	if !accountID.Valid {
		accountID = thread.AccountID
	}
	contactID, ok := optionalUUID(w, req.ContactID, "contact_id")
	if !ok {
		return
	}
	if !contactID.Valid {
		contactID = thread.ContactID
	}
	sentAt, ok := cleanOptionalTimestamp(w, req.SentAt, "sent_at")
	if !ok {
		return
	}
	receivedAt, ok := cleanOptionalTimestamp(w, req.ReceivedAt, "received_at")
	if !ok {
		return
	}
	attachmentsJSON, _ := json.Marshal(req.Attachments)
	message, err := h.scanCRMEmailMessage(h.DB.QueryRow(r.Context(), `
		INSERT INTO crm_email_message (
			workspace_id, thread_id, account_id, contact_id, external_message_id,
			in_reply_to, reference_ids, attachments, from_email, from_name,
			to_emails, cc_emails, bcc_emails, subject, sent_at, received_at,
			body_text, body_html, snippet, direction
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		        $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING id, workspace_id, thread_id, account_id, contact_id, external_message_id,
		          in_reply_to, reference_ids, COALESCE(attachments, '[]'::jsonb), sent_append_warning,
		          raw_size_bytes, COALESCE(raw_headers, '{}'::jsonb), from_email, from_name,
		          to_emails, cc_emails, bcc_emails, subject,
		          sent_at, received_at, body_text, body_html, snippet, direction,
		          created_at, updated_at
	`, workspaceID, threadID, accountID, contactID, cleanOptionalText(req.ExternalMessageID),
		cleanOptionalText(req.InReplyTo), cleanOptionalStringList(req.ReferenceIDs), attachmentsJSON,
		cleanOptionalText(req.FromEmail), cleanOptionalText(req.FromName), cleanOptionalStringList(req.ToEmails),
		cleanOptionalStringList(req.CcEmails), cleanOptionalStringList(req.BccEmails), cleanOptionalText(req.Subject),
		sentAt, receivedAt, cleanOptionalText(req.BodyText), cleanOptionalText(req.BodyHTML), cleanOptionalText(req.Snippet), direction))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create CRM email message")
		return
	}
	_, _ = h.DB.Exec(r.Context(), `
		UPDATE crm_email_thread
		SET last_message_at = COALESCE($3, $4, last_message_at, now()), updated_at = now()
		WHERE id = $1 AND workspace_id = $2
	`, threadID, workspaceID, sentAt, receivedAt)
	if accountID.Valid {
		_, _ = h.regenerateCRMAccountProfile(r.Context(), workspaceID, accountID)
	}
	writeJSON(w, http.StatusCreated, crmEmailMessageToResponse(message))
}

func (h *Handler) GetCRMAccountProfile(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	if _, ok := h.getCRMAccount(w, r, accountID, workspaceID); !ok {
		return
	}
	var id pgtype.UUID
	var summary pgtype.Text
	var updatedBy pgtype.UUID
	var createdAt, updatedAt pgtype.Timestamptz
	var rawProfile []byte
	err := h.DB.QueryRow(r.Context(), `
		SELECT id, summary, profile_json, updated_by, created_at, updated_at
		FROM crm_account_profile
		WHERE workspace_id = $1 AND account_id = $2
	`, workspaceID, accountID).Scan(&id, &summary, &rawProfile, &updatedBy, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusOK, nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get CRM account profile")
		return
	}
	writeJSON(w, http.StatusOK, CRMAccountProfileResponse{
		ID: uuidToString(id), WorkspaceID: uuidToString(workspaceID), AccountID: uuidToString(accountID),
		Summary: textToPtr(summary), ProfileJSON: json.RawMessage(rawProfile), UpdatedBy: uuidToPtr(updatedBy),
		CreatedAt: timestampToString(createdAt), UpdatedAt: timestampToString(updatedAt),
	})
}

func (h *Handler) UpsertCRMAccountProfile(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	if _, ok := h.getCRMAccount(w, r, accountID, workspaceID); !ok {
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	updatedBy, _ := parseUUIDLoose(userID)
	var req UpsertCRMAccountProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	profileJSON := req.ProfileJSON
	if len(profileJSON) == 0 {
		profileJSON = json.RawMessage("{}")
	}
	if !json.Valid(profileJSON) {
		writeError(w, http.StatusBadRequest, "profile_json must be valid JSON")
		return
	}
	var id pgtype.UUID
	var summary pgtype.Text
	var updatedByOut pgtype.UUID
	var createdAt, updatedAt pgtype.Timestamptz
	var rawProfile []byte
	err := h.DB.QueryRow(r.Context(), `
		INSERT INTO crm_account_profile (workspace_id, account_id, summary, profile_json, updated_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (account_id) DO UPDATE SET summary = EXCLUDED.summary, profile_json = EXCLUDED.profile_json, updated_by = EXCLUDED.updated_by, updated_at = now()
		RETURNING id, summary, profile_json, updated_by, created_at, updated_at
	`, workspaceID, accountID, cleanOptionalText(req.Summary), profileJSON, updatedBy).Scan(&id, &summary, &rawProfile, &updatedByOut, &createdAt, &updatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save CRM account profile")
		return
	}
	writeJSON(w, http.StatusOK, CRMAccountProfileResponse{ID: uuidToString(id), WorkspaceID: uuidToString(workspaceID), AccountID: uuidToString(accountID), Summary: textToPtr(summary), ProfileJSON: json.RawMessage(rawProfile), UpdatedBy: uuidToPtr(updatedByOut), CreatedAt: timestampToString(createdAt), UpdatedAt: timestampToString(updatedAt)})
}

func (h *Handler) CreateCRMCommunicationNote(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	if _, ok := h.getCRMAccount(w, r, accountID, workspaceID); !ok {
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	createdBy, _ := parseUUIDLoose(userID)
	var req CreateCRMCommunicationNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	channel := cleanNoteChannel(req.Channel)
	if !validCRMCommunicationChannel(channel) {
		writeError(w, http.StatusBadRequest, "invalid communication channel")
		return
	}
	direction := cleanNoteDirection(req.Direction)
	if !validCRMCommunicationDirection(direction) {
		writeError(w, http.StatusBadRequest, "invalid communication direction")
		return
	}
	contactID, ok := optionalUUID(w, req.ContactID, "contact_id")
	if !ok {
		return
	}
	if contactID.Valid {
		var exists bool
		if err := h.DB.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM crm_contact WHERE id = $1 AND workspace_id = $2 AND account_id = $3)`, contactID, workspaceID, accountID).Scan(&exists); err != nil || !exists {
			writeError(w, http.StatusBadRequest, "contact not found in this account")
			return
		}
	}
	var occurredAt pgtype.Timestamptz
	if req.OccurredAt != nil && strings.TrimSpace(*req.OccurredAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.OccurredAt))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid occurred_at format, expected RFC3339")
			return
		}
		occurredAt = pgtype.Timestamptz{Time: parsed, Valid: true}
	}
	var id, outWorkspaceID, outAccountID, outContactID, outCreatedBy pgtype.UUID
	var outChannel, outDirection, outBody string
	var outOccurredAt, outCreatedAt, outUpdatedAt pgtype.Timestamptz
	var outSubject pgtype.Text
	err := h.DB.QueryRow(r.Context(), `
		INSERT INTO crm_communication_note (workspace_id, account_id, contact_id, channel, direction, occurred_at, subject, body, created_by)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6, now()), $7, $8, $9)
		RETURNING id, workspace_id, account_id, contact_id, channel, direction, occurred_at, subject, body, created_by, created_at, updated_at
	`, workspaceID, accountID, contactID, channel, direction, occurredAt, cleanOptionalText(req.Subject), body, createdBy).Scan(
		&id, &outWorkspaceID, &outAccountID, &outContactID, &outChannel, &outDirection, &outOccurredAt, &outSubject, &outBody, &outCreatedBy, &outCreatedAt, &outUpdatedAt,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create CRM communication note")
		return
	}
	_, _ = h.regenerateCRMAccountProfile(r.Context(), workspaceID, accountID)
	writeJSON(w, http.StatusCreated, CRMCommunicationNoteResponse{
		ID: uuidToString(id), WorkspaceID: uuidToString(outWorkspaceID), AccountID: uuidToPtr(outAccountID), ContactID: uuidToPtr(outContactID),
		Channel: outChannel, Direction: outDirection, OccurredAt: timestampToString(outOccurredAt), Subject: textToPtr(outSubject), Body: outBody,
		CreatedBy: uuidToPtr(outCreatedBy), CreatedAt: timestampToString(outCreatedAt), UpdatedAt: timestampToString(outUpdatedAt),
	})
}

func (h *Handler) ListCRMCommunicationNotes(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	if _, ok := h.getCRMAccount(w, r, accountID, workspaceID); !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT id, workspace_id, account_id, contact_id, channel, direction, occurred_at, subject, body, created_by, created_at, updated_at
		FROM crm_communication_note
		WHERE workspace_id = $1 AND account_id = $2
		ORDER BY occurred_at DESC, created_at DESC
		LIMIT 100
	`, workspaceID, accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list CRM communication notes")
		return
	}
	defer rows.Close()
	notes := []CRMCommunicationNoteResponse{}
	for rows.Next() {
		var id, outWorkspaceID, outAccountID, outContactID, outCreatedBy pgtype.UUID
		var outChannel, outDirection, outBody string
		var outOccurredAt, outCreatedAt, outUpdatedAt pgtype.Timestamptz
		var outSubject pgtype.Text
		if err := rows.Scan(&id, &outWorkspaceID, &outAccountID, &outContactID, &outChannel, &outDirection, &outOccurredAt, &outSubject, &outBody, &outCreatedBy, &outCreatedAt, &outUpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan CRM communication note")
			return
		}
		notes = append(notes, CRMCommunicationNoteResponse{
			ID: uuidToString(id), WorkspaceID: uuidToString(outWorkspaceID), AccountID: uuidToPtr(outAccountID), ContactID: uuidToPtr(outContactID),
			Channel: outChannel, Direction: outDirection, OccurredAt: timestampToString(outOccurredAt), Subject: textToPtr(outSubject), Body: outBody,
			CreatedBy: uuidToPtr(outCreatedBy), CreatedAt: timestampToString(outCreatedAt), UpdatedAt: timestampToString(outUpdatedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"notes": notes, "total": len(notes)})
}

func (h *Handler) LinkCRMAccountProject(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	account, ok := h.getCRMAccount(w, r, accountID, workspaceID)
	if !ok {
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req LinkCRMAccountProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	projectIDs := append([]string{}, req.ProjectIDs...)
	if req.ProjectID != nil {
		projectIDs = append(projectIDs, *req.ProjectID)
	}
	projectIDs = cleanOptionalStringList(projectIDs)
	if len(projectIDs) == 0 {
		writeError(w, http.StatusBadRequest, "project_id or project_ids is required")
		return
	}
	parsedProjectIDs := make([]pgtype.UUID, 0, len(projectIDs))
	for i, rawProjectID := range projectIDs {
		projectID, ok := parseUUIDOrBadRequest(w, rawProjectID, "project_ids["+strconv.Itoa(i)+"]")
		if !ok {
			return
		}
		parsedProjectIDs = append(parsedProjectIDs, projectID)
	}
	labelText := account.Name
	if req.Label != nil && strings.TrimSpace(*req.Label) != "" {
		labelText = strings.TrimSpace(*req.Label)
	}
	ref, err := json.Marshal(map[string]any{"account_id": uuidToString(account.ID), "name": account.Name})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to link CRM account to project")
		return
	}
	creator, _ := h.parseUserUUIDOrZero(userID)
	created := make([]ProjectResourceResponse, 0, len(parsedProjectIDs))
	skipped := []string{}
	for i, projectID := range parsedProjectIDs {
		project, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{ID: projectID, WorkspaceID: workspaceID})
		if err != nil {
			writeError(w, http.StatusNotFound, "project_ids["+strconv.Itoa(i)+"] not found")
			return
		}
		count, _ := h.Queries.CountProjectResources(r.Context(), project.ID)
		resource, err := h.Queries.CreateProjectResource(r.Context(), db.CreateProjectResourceParams{
			ProjectID:    project.ID,
			WorkspaceID:  project.WorkspaceID,
			ResourceType: "crm_account",
			ResourceRef:  ref,
			Label:        pgtype.Text{String: labelText, Valid: true},
			Position:     int32(count),
			CreatedBy:    creator,
		})
		if err != nil {
			if isUniqueViolation(err) {
				skipped = append(skipped, uuidToString(project.ID))
				continue
			}
			writeError(w, http.StatusInternalServerError, "failed to link CRM account to project")
			return
		}
		if _, err := h.DB.Exec(r.Context(), `
			INSERT INTO crm_entity_link (workspace_id, crm_entity_type, crm_entity_id, target_type, target_id, relation_type, created_by)
			VALUES ($1, 'account', $2, 'project', $3, 'customer_for', $4)
			ON CONFLICT DO NOTHING
		`, workspaceID, accountID, project.ID, creator); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to link CRM account to project")
			return
		}
		created = append(created, projectResourceToResponse(resource))
	}
	if len(created) == 0 && len(skipped) > 0 {
		writeError(w, http.StatusConflict, "CRM account is already attached to selected projects")
		return
	}
	if req.ProjectID != nil && len(req.ProjectIDs) == 0 && len(created) == 1 {
		writeJSON(w, http.StatusCreated, created[0])
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"resources": created, "total": len(created), "skipped_project_ids": skipped})
}

func (h *Handler) CreateCRMFollowUpIssue(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	accountID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "accountId"), "account id")
	if !ok {
		return
	}
	account, ok := h.getCRMAccount(w, r, accountID, workspaceID)
	if !ok {
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req CreateCRMFollowUpIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Follow up: " + account.Name
	}
	priority := "none"
	if req.Priority != nil && strings.TrimSpace(*req.Priority) != "" {
		priority = strings.TrimSpace(*req.Priority)
	}
	if priority != "none" && priority != "low" && priority != "medium" && priority != "high" && priority != "urgent" {
		writeError(w, http.StatusBadRequest, "invalid priority")
		return
	}
	var assigneeType pgtype.Text
	var assigneeID pgtype.UUID
	if req.AssigneeType != nil && strings.TrimSpace(*req.AssigneeType) != "" {
		assigneeType = pgtype.Text{String: strings.TrimSpace(*req.AssigneeType), Valid: true}
	}
	if req.AssigneeID != nil && strings.TrimSpace(*req.AssigneeID) != "" {
		assigneeID, ok = parseUUIDOrBadRequest(w, strings.TrimSpace(*req.AssigneeID), "assignee_id")
		if !ok {
			return
		}
	}
	if status, msg := h.validateAssigneePair(r.Context(), r, h.resolveWorkspaceID(r), assigneeType, assigneeID); status != 0 {
		writeError(w, status, msg)
		return
	}
	var projectID pgtype.UUID
	if req.ProjectID != nil && strings.TrimSpace(*req.ProjectID) != "" {
		projectID, ok = parseUUIDOrBadRequest(w, strings.TrimSpace(*req.ProjectID), "project_id")
		if !ok {
			return
		}
		if _, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{ID: projectID, WorkspaceID: workspaceID}); err != nil {
			writeError(w, http.StatusBadRequest, "project not found in this workspace")
			return
		}
	}
	var dueDate pgtype.Timestamptz
	if req.DueDate != nil && strings.TrimSpace(*req.DueDate) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.DueDate))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid due_date format, expected RFC3339")
			return
		}
		dueDate = pgtype.Timestamptz{Time: parsed, Valid: true}
	}
	description := strings.TrimSpace("CRM follow-up for " + account.Name)
	if req.Description != nil && strings.TrimSpace(*req.Description) != "" {
		description = strings.TrimSpace(*req.Description)
	}
	creatorType, actualCreatorID := h.resolveActor(r, userID, uuidToString(workspaceID))
	creatorUUID := parseUUID(actualCreatorID)
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create follow-up issue")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)
	issueNumber, err := qtx.IncrementIssueCounter(r.Context(), workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create follow-up issue")
		return
	}
	issue, err := qtx.CreateIssueWithOrigin(r.Context(), db.CreateIssueWithOriginParams{
		WorkspaceID:  workspaceID,
		Title:        title,
		Description:  pgtype.Text{String: description, Valid: description != ""},
		Status:       "todo",
		Priority:     priority,
		AssigneeType: assigneeType,
		AssigneeID:   assigneeID,
		CreatorType:  creatorType,
		CreatorID:    creatorUUID,
		Position:     0,
		DueDate:      dueDate,
		Number:       issueNumber,
		ProjectID:    projectID,
		OriginType:   pgtype.Text{String: "crm_account", Valid: true},
		OriginID:     account.ID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create follow-up issue")
		return
	}
	if _, err := tx.Exec(r.Context(), `
		INSERT INTO crm_entity_link (workspace_id, crm_entity_type, crm_entity_id, target_type, target_id, relation_type, created_by)
		VALUES ($1, 'account', $2, 'issue', $3, 'follow_up_for', $4)
		ON CONFLICT DO NOTHING
	`, workspaceID, accountID, issue.ID, creatorUUID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to link follow-up issue")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create follow-up issue")
		return
	}
	prefix := h.getIssuePrefix(r.Context(), workspaceID)
	writeJSON(w, http.StatusCreated, CRMFollowUpIssueResponse{Issue: issueToResponse(issue, prefix)})
}

type CRMAISettingResponse struct {
	WorkspaceID     string          `json:"workspace_id"`
	AutomationKey   string          `json:"automation_key"`
	Enabled         bool            `json:"enabled"`
	IntervalMinutes int32           `json:"interval_minutes"`
	AssigneeAgentID *string         `json:"assignee_agent_id"`
	MaxItemsPerRun  int32           `json:"max_items_per_run"`
	Config          json.RawMessage `json:"config"`
	LastResult      json.RawMessage `json:"last_result"`
	LastCheckedAt   *time.Time      `json:"last_checked_at"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type UpdateCRMAISettingRequest struct {
	Enabled         *bool           `json:"enabled"`
	IntervalMinutes *int32          `json:"interval_minutes"`
	AssigneeAgentID *string         `json:"assignee_agent_id"`
	MaxItemsPerRun  *int32          `json:"max_items_per_run"`
	Config          json.RawMessage `json:"config"`
}

func defaultCRMAISettings(workspaceID pgtype.UUID) []CRMAISettingResponse {
	now := time.Now().UTC()
	return []CRMAISettingResponse{
		{WorkspaceID: uuidToString(workspaceID), AutomationKey: "email_pending_reply", Enabled: true, IntervalMinutes: 5, MaxItemsPerRun: 5, Config: json.RawMessage(`{}`), LastResult: json.RawMessage(`{}`), CreatedAt: now, UpdatedAt: now},
		{WorkspaceID: uuidToString(workspaceID), AutomationKey: "due_followup", Enabled: true, IntervalMinutes: 15, MaxItemsPerRun: 10, Config: json.RawMessage(`{}`), LastResult: json.RawMessage(`{}`), CreatedAt: now, UpdatedAt: now},
		{WorkspaceID: uuidToString(workspaceID), AutomationKey: "profile_new_activity_refresh", Enabled: true, IntervalMinutes: 5, MaxItemsPerRun: 20, Config: json.RawMessage(`{"trigger":"new_activity"}`), LastResult: json.RawMessage(`{}`), CreatedAt: now, UpdatedAt: now},
		{WorkspaceID: uuidToString(workspaceID), AutomationKey: "profile_daily_refresh", Enabled: false, IntervalMinutes: 1440, MaxItemsPerRun: 100, Config: json.RawMessage(`{"time":"03:00"}`), LastResult: json.RawMessage(`{}`), CreatedAt: now, UpdatedAt: now},
	}
}

func (h *Handler) ListCRMAISettings(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}

	rows, err := h.DB.Query(r.Context(), `
		WITH defaults AS (
			SELECT $1::uuid AS workspace_id, 'email_pending_reply'::text AS automation_key, true AS enabled, 5::int AS interval_minutes, NULL::uuid AS assignee_agent_id, 5::int AS max_items_per_run, '{}'::jsonb AS config
			UNION ALL
			SELECT $1::uuid, 'due_followup'::text, true, 15::int, NULL::uuid, 10::int, '{}'::jsonb
			UNION ALL
			SELECT $1::uuid, 'profile_new_activity_refresh'::text, true, 5::int, NULL::uuid, 20::int, '{"trigger":"new_activity"}'::jsonb
			UNION ALL
			SELECT $1::uuid, 'profile_daily_refresh'::text, false, 1440::int, NULL::uuid, 100::int, '{"time":"03:00"}'::jsonb
		)
		SELECT d.workspace_id, d.automation_key,
		       COALESCE(s.enabled, d.enabled) AS enabled,
		       COALESCE(s.interval_minutes, d.interval_minutes) AS interval_minutes,
		       COALESCE(s.assignee_agent_id, d.assignee_agent_id) AS assignee_agent_id,
		       COALESCE(s.max_items_per_run, d.max_items_per_run) AS max_items_per_run,
		       COALESCE(s.config, d.config) AS config,
		       COALESCE(s.last_result, '{}'::jsonb) AS last_result,
		       s.last_checked_at,
		       COALESCE(s.created_at, now()) AS created_at,
		       COALESCE(s.updated_at, now()) AS updated_at
		FROM defaults d
		LEFT JOIN crm_ai_setting s ON s.workspace_id = d.workspace_id AND s.automation_key = d.automation_key
		ORDER BY CASE d.automation_key WHEN 'email_pending_reply' THEN 1 WHEN 'due_followup' THEN 2 WHEN 'profile_new_activity_refresh' THEN 3 ELSE 4 END`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load CRM AI settings")
		return
	}
	defer rows.Close()

	settings := make([]CRMAISettingResponse, 0, 2)
	for rows.Next() {
		var item CRMAISettingResponse
		var assignee pgtype.UUID
		var lastChecked pgtype.Timestamptz
		var config []byte
		var lastResult []byte
		if err := rows.Scan(&item.WorkspaceID, &item.AutomationKey, &item.Enabled, &item.IntervalMinutes, &assignee, &item.MaxItemsPerRun, &config, &lastResult, &lastChecked, &item.CreatedAt, &item.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan CRM AI settings")
			return
		}
		if assignee.Valid {
			v := uuidToString(assignee)
			item.AssigneeAgentID = &v
		}
		if lastChecked.Valid {
			t := lastChecked.Time
			item.LastCheckedAt = &t
		}
		item.Config = json.RawMessage(config)
		item.LastResult = json.RawMessage(lastResult)
		settings = append(settings, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

func (h *Handler) UpdateCRMAISetting(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	key := chi.URLParam(r, "automationKey")
	if key != "email_pending_reply" && key != "due_followup" && key != "profile_new_activity_refresh" && key != "profile_daily_refresh" {
		writeError(w, http.StatusBadRequest, "invalid CRM AI setting key")
		return
	}
	var req UpdateCRMAISettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	enabled := true
	intervalMinutes := int32(5)
	maxItems := int32(5)
	config := json.RawMessage(`{}`)
	if key == "due_followup" {
		intervalMinutes = 15
		maxItems = 10
	} else if key == "profile_new_activity_refresh" {
		intervalMinutes = 5
		maxItems = 20
		config = json.RawMessage(`{"trigger":"new_activity"}`)
	} else if key == "profile_daily_refresh" {
		intervalMinutes = 1440
		maxItems = 100
		config = json.RawMessage(`{"time":"03:00"}`)
	}
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if req.IntervalMinutes != nil {
		intervalMinutes = *req.IntervalMinutes
	}
	if intervalMinutes < 1 || intervalMinutes > 1440 {
		writeError(w, http.StatusBadRequest, "interval_minutes must be between 1 and 1440")
		return
	}
	if req.MaxItemsPerRun != nil {
		maxItems = *req.MaxItemsPerRun
	}
	if maxItems < 1 || maxItems > 100 {
		writeError(w, http.StatusBadRequest, "max_items_per_run must be between 1 and 100")
		return
	}
	if len(req.Config) > 0 {
		config = req.Config
	}
	var assignee pgtype.UUID
	if req.AssigneeAgentID != nil && strings.TrimSpace(*req.AssigneeAgentID) != "" {
		parsed, parseErr := parseUUIDLoose(strings.TrimSpace(*req.AssigneeAgentID))
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid assignee_agent_id")
			return
		}
		assignee = parsed
	}

	var item CRMAISettingResponse
	var assigneeOut pgtype.UUID
	var lastChecked pgtype.Timestamptz
	var configOut []byte
	var lastResultOut []byte
	err := h.DB.QueryRow(r.Context(), `
		INSERT INTO crm_ai_setting (workspace_id, automation_key, enabled, interval_minutes, assignee_agent_id, max_items_per_run, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
		ON CONFLICT (workspace_id, automation_key) DO UPDATE SET
		  enabled = EXCLUDED.enabled,
		  interval_minutes = EXCLUDED.interval_minutes,
		  assignee_agent_id = EXCLUDED.assignee_agent_id,
		  max_items_per_run = EXCLUDED.max_items_per_run,
		  config = EXCLUDED.config,
		  updated_at = now()
		RETURNING workspace_id, automation_key, enabled, interval_minutes, assignee_agent_id, max_items_per_run, config, COALESCE(last_result, '{}'::jsonb), last_checked_at, created_at, updated_at`,
		workspaceID, key, enabled, intervalMinutes, assignee, maxItems, string(config)).Scan(&item.WorkspaceID, &item.AutomationKey, &item.Enabled, &item.IntervalMinutes, &assigneeOut, &item.MaxItemsPerRun, &configOut, &lastResultOut, &lastChecked, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save CRM AI setting")
		return
	}
	if assigneeOut.Valid {
		v := uuidToString(assigneeOut)
		item.AssigneeAgentID = &v
	}
	if lastChecked.Valid {
		t := lastChecked.Time
		item.LastCheckedAt = &t
	}
	item.Config = json.RawMessage(configOut)
	item.LastResult = json.RawMessage(lastResultOut)
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) ServeCRMEmailAttachment(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	messageID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "messageId"), "message id")
	if !ok {
		return
	}
	indexStr := chi.URLParam(r, "attachmentIndex")
	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 0 {
		writeError(w, http.StatusBadRequest, "invalid attachment index")
		return
	}
	var attachmentJSON []byte
	if err := h.DB.QueryRow(r.Context(),
		`SELECT attachments -> ($3::int) FROM crm_email_message WHERE id=$1 AND workspace_id=$2`,
		messageID, workspaceID, index).Scan(&attachmentJSON); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "CRM email message not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load CRM email attachment")
		return
	}
	if len(attachmentJSON) == 0 || string(attachmentJSON) == "null" {
		writeError(w, http.StatusNotFound, "attachment index out of range")
		return
	}
	var att crmEmailAttachment
	if err := json.Unmarshal(attachmentJSON, &att); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse attachment")
		return
	}
	att = normalizeCRMEmailAttachment(att, index)
	if strings.TrimSpace(att.Content) == "" {
		writeError(w, http.StatusNotFound, "attachment content is not available for this message")
		return
	}
	raw, err := base64.StdEncoding.DecodeString(att.Content)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(att.Content)
	}
	if err != nil {
		raw, err = base64.URLEncoding.DecodeString(att.Content)
	}
	if err != nil {
		raw, err = base64.RawURLEncoding.DecodeString(att.Content)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to decode attachment content")
		return
	}
	contentType := strings.TrimSpace(att.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	fileName := sanitizeCRMEmailAttachmentFileName(att.DisplayName(index), index)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"; filename*=UTF-8''`+url.PathEscape(fileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func sanitizeCRMEmailAttachmentFileName(value string, index int) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\"", "")
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, "/", "_")
	if value == "" || value == "." || value == ".." {
		return "attachment-" + strconv.Itoa(index+1)
	}
	return value
}

func (h *Handler) TrashCRMEmailThread(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	threadID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "threadId"), "thread id")
	if !ok {
		return
	}
	cmd, err := h.DB.Exec(r.Context(),
		`UPDATE crm_email_thread SET status='open', is_trashed=true, updated_at=now() WHERE id=$1 AND workspace_id=$2`,
		threadID, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to trash CRM email thread")
		return
	}
	if cmd.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "CRM email thread not found")
		return
	}
	_, _ = h.DB.Exec(r.Context(), `
		UPDATE crm_email_message
		SET is_trashed=true,
		    folder='Trash',
		    updated_at=now()
		WHERE thread_id=$1 AND workspace_id=$2
	`, threadID, workspaceID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) RestoreCRMEmailThread(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	threadID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "threadId"), "thread id")
	if !ok {
		return
	}
	cmd, err := h.DB.Exec(r.Context(),
		`UPDATE crm_email_thread SET status='open', is_trashed=false, updated_at=now() WHERE id=$1 AND workspace_id=$2`,
		threadID, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to restore CRM email thread")
		return
	}
	if cmd.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "CRM email thread not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) MoveCRMEmailThread(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	threadID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "threadId"), "thread id")
	if !ok {
		return
	}
	var req struct {
		Folder string `json:"folder"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Folder == "" {
		writeError(w, http.StatusBadRequest, "folder is required")
		return
	}
	allowed := map[string]bool{"inbox": true, "sent": true, "spam": true, "archived": true, "starred": true, "trash": true}
	if !allowed[req.Folder] {
		writeError(w, http.StatusBadRequest, "unsupported folder")
		return
	}
	thread, err := h.scanCRMEmailThread(h.DB.QueryRow(r.Context(),
		`UPDATE crm_email_thread
		 SET status = CASE $3
		       WHEN 'archived' THEN 'archived'
		       ELSE 'open'
		     END,
		     direction = CASE $3 WHEN 'sent' THEN 'outbound' ELSE direction END,
		     is_starred = CASE $3 WHEN 'starred' THEN true ELSE is_starred END,
		     is_trashed = CASE $3 WHEN 'trash' THEN true ELSE false END,
		     updated_at=now()
		 WHERE id=$1 AND workspace_id=$2
		 RETURNING id, workspace_id, account_id, contact_id, project_id, issue_id, subject,
		          external_thread_id, mailbox, direction, status, last_message_at,
		          (SELECT COALESCE(NULLIF(m.snippet, ''), LEFT(COALESCE(NULLIF(m.body_text, ''), regexp_replace(COALESCE(m.body_html, ''), '<[^>]+>', ' ', 'g')), 220)) FROM crm_email_message m WHERE m.thread_id = crm_email_thread.id AND m.workspace_id = crm_email_thread.workspace_id ORDER BY COALESCE(m.sent_at, m.received_at, m.created_at) DESC LIMIT 1) AS last_snippet,
		          created_at, updated_at,
		          (SELECT COUNT(*)::bigint FROM crm_email_message m WHERE m.thread_id = crm_email_thread.id AND m.workspace_id = crm_email_thread.workspace_id), is_read, is_starred, is_trashed`,
		threadID, workspaceID, req.Folder))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "CRM email thread not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to move CRM email thread")
		return
	}
	_, _ = h.DB.Exec(r.Context(), `
		UPDATE crm_email_message
		SET folder = CASE $3
		      WHEN 'archived' THEN 'Archive'
		      WHEN 'trash' THEN folder
		      WHEN 'spam' THEN 'Spam'
		      WHEN 'sent' THEN 'Sent'
		      ELSE 'INBOX'
		    END,
		    is_starred = CASE $3 WHEN 'starred' THEN true ELSE is_starred END,
		    is_trashed = CASE $3 WHEN 'trash' THEN true ELSE false END,
		    updated_at = now()
		WHERE thread_id = $1 AND workspace_id = $2
	`, threadID, workspaceID, req.Folder)
	thread.IssueIDs = h.loadCRMEmailThreadIssueIDs(r.Context(), thread.ID)
	writeJSON(w, http.StatusOK, crmEmailThreadToResponse(thread))
}

func (h *Handler) DeleteCRMEmailThread(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	threadID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "threadId"), "thread id")
	if !ok {
		return
	}
	cmd, err := h.DB.Exec(r.Context(),
		`DELETE FROM crm_email_thread WHERE id=$1 AND workspace_id=$2`,
		threadID, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete CRM email thread")
		return
	}
	if cmd.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "CRM email thread not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type CRMIMAPMailboxDiag struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Email      string  `json:"email"`
	Connected  bool    `json:"connected"`
	LastError  *string `json:"last_error,omitempty"`
	LatencyMs  *int    `json:"latency_ms,omitempty"`
	LastSyncAt *string `json:"last_sync_at,omitempty"`
}

type CRMIMAPDiagnosticsResponse struct {
	Mailboxes []CRMIMAPMailboxDiag `json:"mailboxes"`
}

func (h *Handler) GetCRMIMAPDiagnostics(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(),
		`SELECT id, label, email, host, port, tls_mode, username, secret_ref, sync_enabled, last_test_status, last_test_message, last_tested_at, owner_type, owner_id, smtp_host, smtp_port, smtp_tls_mode, smtp_username, smtp_secret_ref, created_at, updated_at FROM crm_imap_setting WHERE workspace_id=$1 ORDER BY updated_at DESC`,
		workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list IMAP mailboxes")
		return
	}
	defer rows.Close()

	diags := make([]CRMIMAPMailboxDiag, 0)
	for rows.Next() {
		var id pgtype.UUID
		var label, email, host, tlsMode, username, secretRef, lastTestStatus, lastTestMessage, ownerType, ownerID, smtpHost, smtpTLSMode, smtpUsername, smtpSecretRef string
		var port, smtpPort int32
		var syncEnabled bool
		var lastTestedAt pgtype.Timestamptz
		var createdAt, updatedAt pgtype.Timestamptz

		if err := rows.Scan(&id, &label, &email, &host, &port, &tlsMode, &username, &secretRef, &syncEnabled, &lastTestStatus, &lastTestMessage, &lastTestedAt, &ownerType, &ownerID, &smtpHost, &smtpPort, &smtpTLSMode, &smtpUsername, &smtpSecretRef, &createdAt, &updatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan IMAP mailbox")
			return
		}

		diag := CRMIMAPMailboxDiag{
			ID:    uuidToString(id),
			Label: label,
			Email: email,
		}

		if lastTestStatus == "ok" {
			diag.Connected = true
		}
		if lastTestMessage != "" && lastTestStatus != "ok" {
			msg := lastTestMessage
			diag.LastError = &msg
		}
		if lastTestedAt.Valid {
			ts := timestampToString(lastTestedAt)
			diag.LastSyncAt = &ts
		}

		diags = append(diags, diag)
	}
	writeJSON(w, http.StatusOK, CRMIMAPDiagnosticsResponse{Mailboxes: diags})
}

type testCRMIMAPConnRequest struct {
	Host     string `json:"host"`
	Port     int32  `json:"port"`
	TLSMode  string `json:"tls_mode"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) TestCRMIMAPConnection(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	_ = workspaceID

	var req testCRMIMAPConnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Host == "" || req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "host, username, and password are required")
		return
	}
	port := req.Port
	if port <= 0 {
		port = 993
	}
	tlsMode := req.TLSMode
	if tlsMode == "" {
		tlsMode = "tls"
	}

	start := time.Now()
	cfg := crmIMAPMailboxConfig{
		Host:      req.Host,
		Port:      port,
		TLSMode:   tlsMode,
		Username:  req.Username,
		SecretRef: "inline:" + base64.StdEncoding.EncodeToString([]byte(req.Password)),
	}
	_, err := fetchCRMEmailProviderMessages(cfg, "INBOX", 1, 0, nil)
	latencyMs := int(time.Since(start).Milliseconds())

	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         false,
			"status":     "failed",
			"message":    "IMAP connection failed: " + err.Error(),
			"latency_ms": latencyMs,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"status":     "ok",
		"message":    "IMAP connection successful",
		"latency_ms": latencyMs,
	})
}

func (h *Handler) ListCRMIMAPSyncErrors(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	limit := 50
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := h.DB.Query(r.Context(),
		`SELECT r.id, r.mailbox_id, s.email, r.folder, r.status, r.requested_limit, r.fetched_count, r.imported_count, r.skipped_count, r.error_message, r.started_at, r.finished_at, r.created_at, r.updated_at
		FROM crm_imap_sync_run r
		LEFT JOIN crm_imap_setting s ON s.id = r.mailbox_id AND s.workspace_id = r.workspace_id
		WHERE r.workspace_id=$1 AND r.status='error'
		ORDER BY r.created_at DESC LIMIT $2`, workspaceID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list IMAP sync errors")
		return
	}
	defer rows.Close()

	runs := make([]CRMIMAPSyncRunResponse, 0)
	for rows.Next() {
		var id, mailboxID pgtype.UUID
		var mailboxEmail, errorMessage pgtype.Text
		var folder, runStatus string
		var requestedLimit, fetchedCount, importedCount, skippedCount int32
		var startedAt, finishedAt, createdAt, updatedAt pgtype.Timestamptz
		if err := rows.Scan(&id, &mailboxID, &mailboxEmail, &folder, &runStatus, &requestedLimit, &fetchedCount, &importedCount, &skippedCount, &errorMessage, &startedAt, &finishedAt, &createdAt, &updatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan CRM IMAP sync run")
			return
		}
		runs = append(runs, CRMIMAPSyncRunResponse{
			ID: uuidToString(id), MailboxID: uuidToPtr(mailboxID), MailboxEmail: textToPtr(mailboxEmail), Folder: folder, Status: runStatus,
			RequestedLimit: requestedLimit, FetchedCount: fetchedCount, ImportedCount: importedCount, SkippedCount: skippedCount,
			ErrorMessage: textToPtr(errorMessage), StartedAt: timestampToString(startedAt), FinishedAt: timestampToPtr(finishedAt), CreatedAt: timestampToString(createdAt), UpdatedAt: timestampToString(updatedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs, "total": len(runs)})
}

func (h *Handler) SetCRMIMAPSyncCron(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.crmWorkspaceUUID(w, r)
	if !ok {
		return
	}
	mailboxIDStr := chi.URLParam(r, "mailboxId")
	mailboxID, ok := parseUUIDOrBadRequest(w, mailboxIDStr, "mailbox id")
	if !ok {
		return
	}
	var req struct {
		SyncEnabled bool `json:"sync_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cmd, err := h.DB.Exec(r.Context(),
		`UPDATE crm_imap_setting SET sync_enabled=$3, updated_at=now() WHERE id=$1 AND workspace_id=$2`,
		mailboxID, workspaceID, req.SyncEnabled)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update IMAP sync cron setting")
		return
	}
	if cmd.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "IMAP mailbox not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "sync_enabled": req.SyncEnabled})
}
