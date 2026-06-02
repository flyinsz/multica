import { api } from "../api";
import { parseWithFallback } from "../api/schema";
import {
  CRMEmailEngineStatusSchema,
  CRMIMAPImportResponseSchema,
  CRMIMAPPreviewResponseSchema,
  EMPTY_CRM_EMAILENGINE_STATUS,
  EMPTY_CRM_IMAP_IMPORT_RESPONSE,
  EMPTY_CRM_IMAP_PREVIEW_RESPONSE,
} from "./schemas";
import type { CreateCRMAccountRequest, CreateCRMCommunicationNoteRequest, CreateCRMContactRequest, CreateCRMEmailMessageRequest, CreateCRMEmailThreadRequest, CreateCRMFollowUpIssueRequest, CRMAccount, CRMAccountProfile, CRMCommunicationNote, CRMContact, CRMEmailEngineStatus, CRMEmailMessage, CRMEmailThread, CRMIMAPImportResponse, CRMIMAPPreviewResponse, CRMIMAPSetting, CRMIMAPTestResponse, LinkCRMAccountProjectRequest, LinkCRMAccountProjectsResponse, ListCRMAccountsResponse, ListCRMCommunicationNotesResponse, ListCRMContactsResponse, ListCRMEmailMessagesResponse, ListCRMEmailThreadAssociationSuggestionsResponse, ListCRMEmailThreadsResponse, ListCRMIMAPSettingsResponse, UpsertCRMAccountProfileRequest, UpsertCRMIMAPSettingRequest, UpdateCRMEmailThreadAssociationRequest } from "./types";
import type { Issue, ProjectResource } from "../types";

export const crmApi = {
  async listCRMAccounts(params?: {
    status?: string;
    search?: string;
    rating?: string;
    priority?: string;
    country_code?: string;
    industry?: string;
    source?: string;
    follow_up_bucket?: string;
    sort?: string;
  }): Promise<ListCRMAccountsResponse> {
    const search = new URLSearchParams();
    if (params?.status) search.set("status", params.status);
    if (params?.search) search.set("search", params.search);
    if (params?.rating) search.set("rating", params.rating);
    if (params?.priority) search.set("priority", params.priority);
    if (params?.country_code) search.set("country_code", params.country_code);
    if (params?.industry) search.set("industry", params.industry);
    if (params?.source) search.set("source", params.source);
    if (params?.follow_up_bucket) search.set("follow_up_bucket", params.follow_up_bucket);
    if (params?.sort) search.set("sort", params.sort);
    const qs = search.toString();
    return api.request(`/api/crm/accounts${qs ? `?${qs}` : ""}`);
  },

  async getCRMAccount(id: string): Promise<CRMAccount> {
    return api.request(`/api/crm/accounts/${id}`);
  },

  async createCRMAccount(data: CreateCRMAccountRequest): Promise<CRMAccount> {
    return api.request("/api/crm/accounts", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  async updateCRMAccount(id: string, data: CreateCRMAccountRequest): Promise<CRMAccount> {
    return api.request(`/api/crm/accounts/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },

  async deleteCRMAccount(id: string): Promise<void> {
    return api.request(`/api/crm/accounts/${id}`, {
      method: "DELETE",
    });
  },

  async listCRMContacts(accountId: string): Promise<ListCRMContactsResponse> {
    return api.request(`/api/crm/accounts/${accountId}/contacts`);
  },

  async createCRMContact(accountId: string, data: CreateCRMContactRequest): Promise<CRMContact> {
    return api.request(`/api/crm/accounts/${accountId}/contacts`, {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  async updateCRMContact(accountId: string, contactId: string, data: CreateCRMContactRequest): Promise<CRMContact> {
    return api.request(`/api/crm/accounts/${accountId}/contacts/${contactId}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },

  async deleteCRMContact(accountId: string, contactId: string): Promise<void> {
    return api.request(`/api/crm/accounts/${accountId}/contacts/${contactId}`, {
      method: "DELETE",
    });
  },

  async getCRMAccountProfile(accountId: string): Promise<CRMAccountProfile | null> {
    return api.request(`/api/crm/accounts/${accountId}/profile`);
  },

  async upsertCRMAccountProfile(
    accountId: string,
    data: UpsertCRMAccountProfileRequest,
  ): Promise<CRMAccountProfile> {
    return api.request(`/api/crm/accounts/${accountId}/profile`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },

  async listCRMAISettings(): Promise<{ settings: unknown[] }> {
    return api.request("/api/crm/ai-settings");
  },

  async listCRMAIHistory(params: { limit?: number; offset?: number; days?: number; automation_key?: string } = {}): Promise<{ items: unknown[]; limit: number; offset: number; days: number; has_more: boolean }> {
    const query = new URLSearchParams();
    if (params.limit != null) query.set("limit", String(params.limit));
    if (params.offset != null) query.set("offset", String(params.offset));
    if (params.days != null) query.set("days", String(params.days));
    if (params.automation_key) query.set("automation_key", params.automation_key);
    const suffix = query.toString();
    return api.request(`/api/crm/ai-history${suffix ? `?${suffix}` : ""}`);
  },

  async updateCRMAISetting(key: string, data: unknown): Promise<unknown> {
    return api.request(`/api/crm/ai-settings/${key}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },

  async listCRMIMAPSettings(): Promise<ListCRMIMAPSettingsResponse> {
    return api.request("/api/crm/imap-settings");
  },

  async upsertCRMIMAPSetting(data: UpsertCRMIMAPSettingRequest): Promise<CRMIMAPSetting> {
    return api.request("/api/crm/imap-settings", {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },

  async testCRMIMAPSetting(mailboxId: string): Promise<CRMIMAPTestResponse> {
    return api.request(`/api/crm/imap-settings/${mailboxId}/test`, { method: "POST" });
  },

  async deleteCRMIMAPSetting(mailboxId: string): Promise<void> {
    return api.request(`/api/crm/imap-settings/${mailboxId}`, { method: "DELETE" });
  },

  async previewCRMIMAP(data: { mailbox_id?: string | null; folder?: string | null; limit?: number; range_days?: number }): Promise<CRMIMAPPreviewResponse> {
    const raw = await api.request<unknown>("/api/crm/imap/preview", {
      method: "POST",
      body: JSON.stringify(data),
    });
    return parseWithFallback(raw, CRMIMAPPreviewResponseSchema, EMPTY_CRM_IMAP_PREVIEW_RESPONSE, {
      endpoint: "POST /api/crm/imap/preview",
    });
  },

  async importCRMIMAP(data: { mailbox_id?: string | null; folder?: string | null; uids: string[]; limit?: number }): Promise<CRMIMAPImportResponse> {
    const raw = await api.request<unknown>("/api/crm/imap/import", {
      method: "POST",
      body: JSON.stringify(data),
    });
    return parseWithFallback(raw, CRMIMAPImportResponseSchema, EMPTY_CRM_IMAP_IMPORT_RESPONSE, {
      endpoint: "POST /api/crm/imap/import",
    });
  },

  async syncCRMIMAP(data: { mailbox_id?: string | null; folder?: string | null; limit?: number; range_days?: number }): Promise<CRMIMAPImportResponse> {
    const raw = await api.request<unknown>("/api/crm/imap/sync", {
      method: "POST",
      body: JSON.stringify(data),
    });
    return parseWithFallback(raw, CRMIMAPImportResponseSchema, EMPTY_CRM_IMAP_IMPORT_RESPONSE, {
      endpoint: "POST /api/crm/imap/sync",
    });
  },

  async listCRMIMAPSyncRuns(): Promise<{ runs: any[]; total: number }> {
    return api.request("/api/crm/imap/sync-runs");
  },

  async getCRMEmailEngineStatus(mailboxId?: string | null): Promise<CRMEmailEngineStatus> {
    const search = new URLSearchParams();
    if (mailboxId) search.set("mailbox_id", mailboxId);
    const qs = search.toString();
    const raw = await api.request<unknown>(`/api/crm/emailengine/status${qs ? `?${qs}` : ""}`);
    return parseWithFallback(raw, CRMEmailEngineStatusSchema, EMPTY_CRM_EMAILENGINE_STATUS, {
      endpoint: "GET /api/crm/emailengine/status",
    });
  },

  async refreshCRMAccountProfile(accountId: string): Promise<CRMAccountProfile> {
    return api.request(`/api/crm/accounts/${accountId}/profile/suggestions`, { method: "POST" });
  },

  async suggestCRMAccountProfile(accountId: string): Promise<CRMAccountProfile> {
    return this.refreshCRMAccountProfile(accountId);
  },

  async applyCRMAccountProfileSuggestion(accountId: string, suggestionId: string): Promise<{ ok: boolean }> {
    return api.request(`/api/crm/accounts/${accountId}/profile/suggestions/${suggestionId}/apply`, { method: "POST" });
  },

  async listCRMEmailDrafts(draftId?: string | null): Promise<{ drafts: any[]; total: number }> {
    const query = draftId ? `?draft_id=${encodeURIComponent(draftId)}` : "";
    return api.request(`/api/crm/email-drafts${query}`);
  },

  async createCRMEmailDraft(data: Record<string, unknown>): Promise<{ ok: boolean; id: string }> {
    return api.request("/api/crm/email-drafts", { method: "POST", body: JSON.stringify(data) });
  },

  async suggestCRMEmailDraftReply(data: Record<string, unknown>): Promise<{ chinese: string; customer_language: string; customer_reply: string; source: string }> {
    return api.request("/api/crm/email-drafts/ai-suggest", { method: "POST", body: JSON.stringify(data) });
  },

  async updateCRMEmailDraft(draftId: string, data: Record<string, unknown>): Promise<{ ok: boolean; id: string }> {
    return api.request(`/api/crm/email-drafts/${draftId}`, { method: "PATCH", body: JSON.stringify(data) });
  },

  async sendCRMEmailDraft(draftId: string): Promise<{ ok: boolean; status: string }> {
    return api.request(`/api/crm/email-drafts/${draftId}/send`, { method: "POST" });
  },

  async listCRMEmailThreads(params?: { account_id?: string; folder?: string; filter?: string; mailbox?: string }): Promise<ListCRMEmailThreadsResponse> {
    const search = new URLSearchParams();
    if (params?.account_id) search.set("account_id", params.account_id);
    if (params?.folder) search.set("folder", params.folder);
    if (params?.filter) search.set("filter", params.filter);
    if (params?.mailbox) search.set("mailbox", params.mailbox);
    const qs = search.toString();
    return api.request(`/api/crm/email-threads${qs ? `?${qs}` : ""}`);
  },

  async createCRMEmailThread(data: CreateCRMEmailThreadRequest): Promise<CRMEmailThread> {
    return api.request("/api/crm/email-threads", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  async getCRMEmailThread(threadId: string): Promise<CRMEmailThread> {
    return api.request(`/api/crm/email-threads/${threadId}`);
  },

  async listCRMEmailThreadAssociationSuggestions(threadId: string): Promise<ListCRMEmailThreadAssociationSuggestionsResponse> {
    return api.request(`/api/crm/email-threads/${threadId}/association-suggestions`);
  },

  async updateCRMEmailThreadAssociation(
    threadId: string,
    data: UpdateCRMEmailThreadAssociationRequest,
  ): Promise<CRMEmailThread> {
    return api.request(`/api/crm/email-threads/${threadId}/association`, {
      method: "PATCH",
      body: JSON.stringify(data),
    });
  },

  async updateCRMEmailThreadState(threadId: string, data: { status?: "open" | "archived"; is_read?: boolean; is_starred?: boolean; message_id?: string | null }): Promise<CRMEmailThread> {
    return api.request(`/api/crm/email-threads/${threadId}/state`, {
      method: "PATCH",
      body: JSON.stringify(data),
    });
  },

  getCRMEmailAttachmentUrl(wsId: string, messageId: string, attachmentIndex: number): string {
    void wsId;
    return `/api/crm/email-messages/${messageId}/attachment/${attachmentIndex}`;
  },

  async downloadCRMEmailAttachment(wsId: string, messageId: string, attachmentIndex: number): Promise<Blob> {
    void wsId;
    const path = this.getCRMEmailAttachmentUrl(wsId, messageId, attachmentIndex);
    return api.requestBlob(path);
  },

  async trashCRMEmailThread(threadId: string): Promise<void> {
    await api.request(`/api/crm/email-threads/${threadId}/trash`, { method: "POST" });
  },

  async restoreCRMEmailThread(threadId: string): Promise<void> {
    await api.request(`/api/crm/email-threads/${threadId}/restore`, { method: "POST" });
  },

  async deleteCRMEmailThread(threadId: string): Promise<void> {
    await api.request(`/api/crm/email-threads/${threadId}/delete`, { method: "DELETE" });
  },

  async moveCRMEmailThread(threadId: string, folder: string): Promise<CRMEmailThread> {
    return api.request<CRMEmailThread>(`/api/crm/email-threads/${threadId}/move-folder`, {
      method: "POST",
      body: JSON.stringify({ folder }),
    });
  },

  async getCRMIMAPDiagnostics(_wsId: string): Promise<any> {
    return api.request("/api/crm/imap/diagnostics");
  },

  async testCRMIMAPConnection(_wsId: string, config: Record<string, unknown>): Promise<any> {
    return api.request("/api/crm/imap/test-connection", {
      method: "POST",
      body: JSON.stringify(config),
    });
  },

  async listCRMIMAPSyncErrors(_wsId: string): Promise<any> {
    return api.request("/api/crm/sync-runs/errors");
  },

  async toggleCRMIMAPSyncCron(_wsId: string, mailboxId: string, sync_enabled: boolean): Promise<any> {
    return api.request(`/api/crm/imap/${mailboxId}/sync-cron`, {
      method: "POST",
      body: JSON.stringify({ sync_enabled }),
    });
  },

  async listCRMEmailMessages(threadId: string): Promise<ListCRMEmailMessagesResponse> {
    return api.request(`/api/crm/email-threads/${threadId}/messages`);
  },

  async createCRMEmailMessage(threadId: string, data: CreateCRMEmailMessageRequest): Promise<CRMEmailMessage> {
    return api.request(`/api/crm/email-threads/${threadId}/messages`, {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  async listCRMCommunicationNotes(accountId: string): Promise<ListCRMCommunicationNotesResponse> {
    return api.request(`/api/crm/accounts/${accountId}/notes`);
  },

  async createCRMCommunicationNote(
    accountId: string,
    data: CreateCRMCommunicationNoteRequest,
  ): Promise<CRMCommunicationNote> {
    return api.request(`/api/crm/accounts/${accountId}/notes`, {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  async linkCRMAccountProject(
    accountId: string,
    data: LinkCRMAccountProjectRequest,
  ): Promise<ProjectResource | LinkCRMAccountProjectsResponse> {
    return api.request(`/api/crm/accounts/${accountId}/projects`, {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  async createCRMFollowUpIssue(
    accountId: string,
    data: CreateCRMFollowUpIssueRequest,
  ): Promise<{ issue: Issue }> {
    return api.request(`/api/crm/accounts/${accountId}/follow-up-issues`, {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  // Labels
};
