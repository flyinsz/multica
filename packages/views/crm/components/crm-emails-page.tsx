"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Archive, ArrowRight, Building2, Inbox, Link2, Mail, MailOpen, Paperclip, Search, Send, Settings, Star, Trash2, Undo2, UserRound, Wrench, Activity, RefreshCw } from "lucide-react";
import { api } from "@multica/core/api";
import { useWorkspaceId } from "@multica/core/hooks";
import { issueKeys, useIssueDraftStore } from "@multica/core/issues";
import { useModalStore } from "@multica/core/modals";
import { crmAccountListOptions, crmContactListOptions, crmEmailMessageListOptions, crmEmailThreadListOptions, crmKeys } from "@multica/core/crm/queries";
import { useWorkspacePaths } from "@multica/core/paths";
import type { CRMAccount, CRMContact, CRMEmailThread, CRMEmailThreadAssociationSuggestion, CRMIMAPPreviewMessage, CRMIMAPSetting, CreateCRMContactRequest, Issue, Project } from "@multica/core/types";
import { Badge } from "@multica/ui/components/ui/badge";
import { Button } from "@multica/ui/components/ui/button";
import { normalizeLocale } from "../geo";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@multica/ui/components/ui/dialog";
import { Input } from "@multica/ui/components/ui/input";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { PageHeader } from "../../layout/page-header";
import { useNavigation } from "../../navigation";
import { useT } from "../../i18n";

type AssociationDraft = {
  accountId: string;
  contactId: string;
  contactName: string;
  contactEmail: string;
};

type EmailLinkDraft = { projectId: string; issueIds: string[] };

type ComposeAttachment = { file_name: string; content_type: string; content: string; size: number };
type ComposeDraft = { draftId?: string; mailboxId: string; accountId: string; contactId: string; to: string; cc: string; bcc: string; subject: string; body: string; attachments: ComposeAttachment[] };

type MailboxDraft = { id?: string | null; label: string; email: string; host: string; port: string; tls_mode: "ssl" | "starttls" | "none"; username: string; secret_ref: string; secret: string; sync_enabled: boolean; owner_type: string; owner_id: string; smtp_host: string; smtp_port: string; smtp_tls_mode: string; smtp_username: string; smtp_secret_ref: string; smtp_secret: string };

const emptyMailboxDraft: MailboxDraft = { label: "", email: "", host: "", port: "993", tls_mode: "ssl", username: "", secret_ref: "", secret: "", sync_enabled: false, owner_type: "", owner_id: "", smtp_host: "", smtp_port: "465", smtp_tls_mode: "ssl", smtp_username: "", smtp_secret_ref: "", smtp_secret: "" };

function inferMailboxPreset(email: string) {
  const domain = email.trim().toLowerCase().split("@")[1] ?? "";
  if (domain === "qq.com") return { host: "imap.qq.com", port: "993", smtp_host: "smtp.qq.com", smtp_port: "465", tls_mode: "ssl" as const, smtp_tls_mode: "ssl" };
  if (domain === "gmail.com") return { host: "imap.gmail.com", port: "993", smtp_host: "smtp.gmail.com", smtp_port: "465", tls_mode: "ssl" as const, smtp_tls_mode: "ssl" };
  if (domain === "outlook.com" || domain === "hotmail.com" || domain === "live.com") return { host: "outlook.office365.com", port: "993", smtp_host: "smtp.office365.com", smtp_port: "587", tls_mode: "ssl" as const, smtp_tls_mode: "starttls" };
  return null;
}

function looksLikeHTML(value?: string | null) {
  return Boolean(value && /<\s*(html|body|div|p|br|table|span|a|img|strong|em|ul|ol|li)\b/i.test(value));
}

function emailHTMLBody(message: { body_html?: string | null; body_text?: string | null }) {
  return message.body_html || (looksLikeHTML(message.body_text) ? message.body_text || "" : "");
}

function emailHTMLBodyWithCID(message: { body_html?: string | null; body_text?: string | null; attachments?: Array<{ content_id?: string; content_type?: string; content?: string }> | null }) {
  let html = emailHTMLBody(message);
  if (!html || !message.attachments?.length) return html;
  for (const att of message.attachments) {
    if (att.content_id && att.content) {
      const cid = att.content_id.replace(/^</, "").replace(/>$/, "");
      const re = new RegExp(`src=["']cid:${cid}["']`, "gi");
      html = html.replace(re, `src="data:${att.content_type || "application/octet-stream"};base64,${att.content}"`);
    }
  }
  return html;
}

function safeEmailHTML(html: string) {
  return html
    .replace(/<script[\s\S]*?>[\s\S]*?<\/script>/gi, "")
    .replace(/\son\w+\s*=\s*(["']).*?\1/gi, "")
    .replace(/\s(href|src)\s*=\s*(["'])\s*javascript:[\s\S]*?\2/gi, ' $1="#"');
}

function mailboxToDraft(setting?: CRMIMAPSetting | null): MailboxDraft {
  if (!setting) return emptyMailboxDraft;
  return { id: setting.id, label: setting.label, email: setting.email, host: setting.host, port: String(setting.port), tls_mode: setting.tls_mode, username: setting.username, secret_ref: setting.secret_ref ?? "", secret: "", sync_enabled: setting.sync_enabled, owner_type: setting.owner_type ?? "", owner_id: setting.owner_id ?? "", smtp_host: setting.smtp_host ?? "", smtp_port: String(setting.smtp_port ?? 465), smtp_tls_mode: setting.smtp_tls_mode ?? "ssl", smtp_username: setting.smtp_username ?? "", smtp_secret_ref: setting.smtp_secret_ref ?? "", smtp_secret: "" };
}

function messageTime(value?: string | null) {
  return value ? new Date(value).toLocaleString() : "—";
}

function inferContactDraft(messages: Array<{ from_name?: string | null; from_email?: string | null; direction: string }>): Pick<AssociationDraft, "contactName" | "contactEmail"> {
  const inbound = messages.find((message) => message.direction === "inbound" && (message.from_name || message.from_email));
  const email = inbound?.from_email ?? "";
  const name = inbound?.from_name || email.split("@")[0] || "";
  return { contactName: name, contactEmail: email };
}

function DetailRow({ label, value }: { label: string; value?: string | null }) {
  return (
    <div className="rounded-md border bg-muted/20 px-3 py-2">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className="mt-1 truncate text-sm">{value || "—"}</div>
    </div>
  );
}

function mutationErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback;
}

function AssociationChip({ icon, label, value, onClick }: { icon: ReactNode; label: string; value?: string | null; onClick?: () => void }) {
  return (
    <button
      type="button"
      className="group inline-flex min-w-0 items-center gap-2 rounded-full border bg-background px-3 py-1.5 text-left text-sm hover:bg-muted/60 disabled:pointer-events-none disabled:opacity-70"
      onClick={onClick}
      disabled={!onClick}
    >
      <span className="shrink-0 text-muted-foreground">{icon}</span>
      <span className="min-w-0">
        <span className="block text-[11px] leading-none text-muted-foreground">{label}</span>
        <span className="block truncate font-medium">{value || "—"}</span>
      </span>
    </button>
  );
}

function DiagnosticsDialog({ wsId, open, onOpenChange }: { wsId: string; open: boolean; onOpenChange: (open: boolean) => void }) {
  const diagnostics = useQuery({
    queryKey: ["crm", wsId, "imap-diagnostics"],
    queryFn: () => api.getCRMIMAPDiagnostics(wsId),
    enabled: open && Boolean(wsId),
  });
  const syncErrors = useQuery({
    queryKey: ["crm", wsId, "sync-errors"],
    queryFn: () => api.listCRMIMAPSyncErrors(wsId),
    enabled: open && Boolean(wsId),
  });
  const testConnection = useMutation({
    mutationFn: (config: Record<string, unknown>) => api.testCRMIMAPConnection(wsId, config),
  });
  const diagnosticMailboxes = Array.isArray((diagnostics.data as any)?.mailboxes)
    ? (diagnostics.data as any).mailboxes
    : Array.isArray(diagnostics.data)
      ? diagnostics.data
      : [];
  const syncErrorItems = Array.isArray((syncErrors.data as any)?.errors)
    ? (syncErrors.data as any).errors
    : Array.isArray(syncErrors.data)
      ? syncErrors.data
      : [];
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle><Wrench className="mr-2 inline size-4" />IMAP Diagnostics</DialogTitle>
          <DialogDescription>Connection status, sync errors, and mailbox diagnostics</DialogDescription>
        </DialogHeader>
        {diagnostics.isLoading ? (
          <Skeleton className="h-32 w-full" />
        ) : diagnostics.error ? (
          <p className="text-sm text-destructive">Failed to load diagnostics</p>
        ) : (
          <div className="space-y-3">
            {diagnosticMailboxes.length ? diagnosticMailboxes.map((mailbox: any, i: number) => (
              <div key={mailbox.id || i} className="rounded-lg border bg-card p-4 text-sm">
                <div className="flex items-center justify-between">
                  <span className="font-medium">{mailbox.email || mailbox.label || `Mailbox ${i + 1}`}</span>
                  <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
                    mailbox.connected || mailbox.connection_status === "ok" ? "bg-green-100 text-green-700" :
                    mailbox.last_error || mailbox.connection_status === "error" ? "bg-red-100 text-red-700" :
                    "bg-yellow-100 text-yellow-700"
                  }`}>
                    <Activity className="size-3" />
                    {mailbox.connected ? "ok" : mailbox.connection_status || "unknown"}
                  </span>
                </div>
                {mailbox.last_error && <p className="mt-1 text-xs text-red-600">Error: {mailbox.last_error}</p>}
                {mailbox.latency_ms != null && <p className="mt-1 text-xs text-muted-foreground">Latency: {mailbox.latency_ms}ms</p>}
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-2"
                  disabled={testConnection.isPending}
                  onClick={() => testConnection.mutate(mailbox)}
                >
                  Test Connection
                </Button>
              </div>
            )) : (
              <p className="text-sm text-muted-foreground">No mailbox diagnostics found.</p>
            )}
          </div>
        )}
        <div className="mt-4">
          <h4 className="mb-2 text-sm font-semibold">Recent Sync Errors</h4>
          {syncErrors.isLoading ? (
            <Skeleton className="h-20 w-full" />
          ) : syncErrorItems.length > 0 ? (
            <div className="max-h-40 space-y-2 overflow-y-auto">
              {syncErrorItems.map((err: any, i: number) => (
                <div key={i} className="rounded border bg-muted/20 px-3 py-2 text-xs">
                  <span className="font-medium text-red-600">{err.error || err.message}</span>
                  {err.mailbox && <span className="ml-2 text-muted-foreground">· {err.mailbox}</span>}
                  {err.created_at && <span className="ml-2 text-muted-foreground">· {messageTime(err.created_at)}</span>}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">No sync errors found.</p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

export function CRMEmailsPage() {
  const wsId = useWorkspaceId();
  const queryClient = useQueryClient();
  const navigation = useNavigation();
  const paths = useWorkspacePaths();
  const { t, i18n } = useT("crm");
  const locale = normalizeLocale(i18n.language);
  const emailCopy = locale === "zh-Hans" ? {
    compose: "写邮件",
    drafts: "草稿",
    reply: "回复",
    replyAll: "全部回复",
    forward: "转发",
    send: "发送",
    to: "收件人",
    cc: "抄送",
    bcc: "密送",
    subject: "主题",
    bodyPlaceholder: "写邮件正文…",
    bodyLabel: "邮件正文",
    saveDraftError: "保存草稿失败。请检查邮箱和收件人字段。",
    from: "发件人",
    date: "日期",
    htmlBody: "邮件正文",
    textBody: "文本正文",
    attachments: "附件",
    noAttachments: "无附件",
    mimeMetadata: "MIME 元数据",
    messageId: "Message-ID",
    inReplyTo: "In-Reply-To",
    references: "References",
    rawSize: "原始大小",
    contentType: "Content-Type",
    contentDisposition: "Disposition",
    bytes: "字节",
    noSubject: "（无主题）",
    mailbox: "邮箱",
    nativeProvider: "原生邮箱",
    notConfigured: "未配置",
    syncing: "同步中",
    noDrafts: "暂无草稿。先写邮件或回复，再保存为草稿发送。",
    draftSent: "草稿已发送。",
    draftSaved: "草稿已保存。",
    noDraft: "当前没有正在编辑的草稿",
    createMailboxFirst: "请先创建邮箱再保存草稿",
    composeTitle: "写 CRM 邮件草稿",
    composeHelp: "先保存为草稿，确认内容后从草稿箱发送。",
    cancel: "取消",
    saveDraft: "保存草稿",
    mailboxSettingsTitle: "CRM 原生邮箱设置",
    mailboxSettingsHelp: "使用 ImapFlow 收信、Nodemailer 发信；Multica 自己显示邮箱工作台。",
    providerLabel: "服务：原生 IMAP/SMTP",
    providerHelp: "配置 IMAP 收件与 SMTP 发件后，收件箱刷新、草稿发送和 CRM 关联会在 Multica 内完成。",
    step1: "1. 保存 IMAP/SMTP 账号",
    step2: "2. 测试连接",
    step3: "3. 导入邮件",
    secretRef: "IMAP 密钥引用",
    secretRefHelp: "IMAP 密钥引用；密码存入后端密钥存储",
    importRange: "导入范围",
    recent7: "最近 7 天",
    recent30: "最近 30 天",
    recent90: "最近 90 天",
    smtpUsername: "SMTP 用户名",
    smtpPassword: "SMTP 应用密码",
    transportNote: "主要传输使用原生 IMAP/SMTP。密钥保存在后端；此处只显示 CRM 邮箱记录和导入范围。",
    checkProvider: "检查连接",
    saveAndImport: "保存并导入",
    providerFolders: "服务端文件夹",
    fallbackMailbox: "sales@example.com",
    providerStatusFallback: "原生 IMAP/SMTP",
    savingImporting: "正在保存邮箱并导入所选范围…",
    mailboxSavedNoMessages: "邮箱已保存。所选范围内没有邮件。",
    imported: (imported: number, skipped: number) => `已导入 ${imported} 封；跳过 ${skipped} 封。`,
    fetched: (note: string, total: number) => `${note} 已获取 ${total} 封邮件。`,
    savedImported: (imported: number, skipped: number) => `邮箱已保存。已导入 ${imported} 封；跳过 ${skipped} 封。`,
  } : {
    compose: "Compose",
    drafts: "Drafts",
    reply: "Reply",
    replyAll: "Reply all",
    forward: "Forward",
    send: "Send",
    to: "To",
    cc: "Cc",
    bcc: "Bcc",
    subject: "Subject",
    bodyPlaceholder: "Write email body…",
    bodyLabel: "Email body",
    saveDraftError: "Failed to save draft. Check mailbox and recipient fields.",
    from: "From",
    date: "Date",
    htmlBody: "Email body",
    textBody: "Text body",
    attachments: "Attachments",
    noAttachments: "No attachments",
    mimeMetadata: "MIME metadata",
    messageId: "Message-ID",
    inReplyTo: "In-Reply-To",
    references: "References",
    rawSize: "Raw size",
    contentType: "Content-Type",
    contentDisposition: "Disposition",
    bytes: "bytes",
    noSubject: "(no subject)",
    mailbox: "Mailbox",
    nativeProvider: "Native mailbox",
    notConfigured: "Not configured",
    syncing: "Syncing",
    noDrafts: "No drafts yet. Compose or reply first, then save draft before sending.",
    draftSent: "Draft sent.",
    draftSaved: "Draft saved.",
    noDraft: "No draft is being composed",
    createMailboxFirst: "Create a mailbox before saving a draft",
    composeTitle: "Compose CRM email draft",
    composeHelp: "Save draft first, then send from Drafts after review.",
    cancel: "Cancel",
    saveDraft: "Save draft",
    mailboxSettingsTitle: "CRM native mailbox settings",
    mailboxSettingsHelp: "Use ImapFlow for inbox sync and Nodemailer for sending while Multica owns the mailbox workspace UI.",
    providerLabel: "Provider: native IMAP/SMTP",
    providerHelp: "Configure IMAP receive and SMTP send so inbox refresh, draft sending, and CRM linking stay inside Multica.",
    step1: "1. Save IMAP/SMTP account",
    step2: "2. Test connection",
    step3: "3. Import messages",
    secretRef: "IMAP secret reference",
    secretRefHelp: "IMAP secret reference; password is stored in backend secret storage",
    importRange: "Import range",
    recent7: "Recent 7 days",
    recent30: "Recent 30 days",
    recent90: "Recent 90 days",
    smtpUsername: "SMTP username",
    smtpPassword: "SMTP app password",
    transportNote: "Primary transport uses native IMAP/SMTP. Secrets stay in backend storage; this form shows CRM mailbox record and import range.",
    checkProvider: "Check connection",
    saveAndImport: "Save and import",
    providerFolders: "Server folders",
    fallbackMailbox: "sales@example.com",
    providerStatusFallback: "Native IMAP/SMTP",
    savingImporting: "Saving mailbox and importing selected range…",
    mailboxSavedNoMessages: "Mailbox saved. No messages found in selected range.",
    imported: (imported: number, skipped: number) => `Imported ${imported}; skipped ${skipped}.`,
    fetched: (note: string, total: number) => `${note} ${total} messages fetched.`,
    savedImported: (imported: number, skipped: number) => `Mailbox saved. Imported ${imported}; skipped ${skipped}.`,
  };
  const [search, setSearch] = useState("");
  const [activeFolder, setActiveFolder] = useState<"inbox" | "sent" | "drafts" | "archived" | "starred" | "unlinked" | "trash">("inbox");
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [selectedMailboxId, setSelectedMailboxId] = useState<string | null>(null);
  const [mailboxDraft, setMailboxDraft] = useState<MailboxDraft>(emptyMailboxDraft);
  const [mailboxStatus, setMailboxStatus] = useState<string | null>(null);
  const [previewMessages, setPreviewMessages] = useState<CRMIMAPPreviewMessage[]>([]);
  const [selectedPreviewUIDs, setSelectedPreviewUIDs] = useState<string[]>([]);
  const [importRangeDays, setImportRangeDays] = useState(30);
  const [selectedThreadId, setSelectedThreadId] = useState<string | null>(null);
  const [detailDialog, setDetailDialog] = useState<{ type: "account"; account: CRMAccount } | { type: "contact"; contact: CRMContact } | null>(null);
  const [associationDraft, setAssociationDraft] = useState<AssociationDraft | null>(null);
  const [emailLinkDraft, setEmailLinkDraft] = useState<EmailLinkDraft | null>(null);
  const [composeDraft, setComposeDraft] = useState<ComposeDraft | null>(null);
  const [composeAccountSearch, setComposeAccountSearch] = useState("");
  const [composeRecipientPickerOpen, setComposeRecipientPickerOpen] = useState(false);
  const [diagnosticsOpen, setDiagnosticsOpen] = useState(false);
  const composeFullWidth = Boolean(composeDraft);
  const openModal = useModalStore((state) => state.open);
  const setIssueDraft = useIssueDraftStore((state) => state.setDraft);
  const clearIssueDraft = useIssueDraftStore((state) => state.clearDraft);
  const { data: threads = [], isLoading } = useQuery(crmEmailThreadListOptions(wsId));
  const { data: accounts = [] } = useQuery(crmAccountListOptions(wsId, { sort: "name" }));
  const { data: members = [] } = useQuery({ queryKey: ["workspace", wsId, "members", "crm-mailbox"], queryFn: () => api.listMembers(wsId), enabled: Boolean(wsId) });
  const { data: agents = [] } = useQuery({ queryKey: ["agents", wsId, "crm-mailbox"], queryFn: () => api.listAgents({ workspace_id: wsId }), enabled: Boolean(wsId) });
  const { data: draftsData } = useQuery({ queryKey: ["crm", wsId, "email-drafts"], queryFn: () => api.listCRMEmailDrafts(), enabled: Boolean(wsId), refetchOnMount: "always" });
  const { data: mailboxData } = useQuery({
    queryKey: ["crm", wsId, "imap-settings"],
    queryFn: () => api.listCRMIMAPSettings(),
    enabled: Boolean(wsId),
  });
  const { data: syncRunsData, dataUpdatedAt: syncRunsUpdatedAt } = useQuery({
    queryKey: ["crm", wsId, "imap-sync-runs"],
    queryFn: () => api.listCRMIMAPSyncRuns(),
    enabled: Boolean(wsId),
    refetchInterval: (query) => query.state.data?.runs?.some((run: any) => run.status === "running") ? 3000 : false,
  });
  const mailboxes = mailboxData?.settings ?? [];
  const selectedMailbox = mailboxes.find((mailbox) => mailbox.id === selectedMailboxId) ?? mailboxes[0] ?? null;
  const emailDrafts = draftsData?.drafts ?? [];
  const syncRuns = syncRunsData?.runs ?? [];

  const draftToCompose = (draft: any): ComposeDraft => ({
    draftId: draft.id,
    mailboxId: draft.mailbox_id ?? selectedMailbox?.id ?? "",
    accountId: draft.account_id ?? "",
    contactId: draft.contact_id ?? "",
    to: (draft.to_emails ?? []).join(", "),
    cc: (draft.cc_emails ?? []).join(", "),
    bcc: (draft.bcc_emails ?? []).join(", "),
    subject: draft.subject ?? "",
    body: draft.body_text ?? "",
    attachments: Array.isArray(draft.attachments) ? draft.attachments.map((attachment: any) => ({
      file_name: attachment.file_name || attachment.filename || "attachment",
      content_type: attachment.content_type || "application/octet-stream",
      content: attachment.content || "",
      size: attachment.size || attachment.size_bytes || 0,
    })) : [],
  });

  const openDraftInComposer = (draft: any) => {
    setSelectedThreadId(null);
    setComposeDraft(draftToCompose(draft));
  };

  const activeSyncRuns = useMemo(() => {
    const cutoff = Date.now() - 2 * 60 * 1000;
    return syncRuns.filter((run: any) => run.status === "running" && (!run.started_at || new Date(run.started_at).getTime() > cutoff));
  }, [syncRuns]);

  useEffect(() => {
    if (!wsId || !syncRunsUpdatedAt) return;
    const latest = syncRuns[0];
    if (!latest || latest.status === "running") return;
    queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
    queryClient.invalidateQueries({ queryKey: ["crm", wsId, "imap-settings"] });
    queryClient.invalidateQueries({ queryKey: ["crm", wsId, "email-drafts"] });
  }, [queryClient, syncRuns, syncRunsUpdatedAt, wsId]);

  const mailboxThreads = useMemo(() => {
    if (!selectedMailbox?.email) return threads;
    return threads.filter((thread) => (thread.mailbox ?? "") === selectedMailbox.email);
  }, [selectedMailbox?.email, threads]);

  const mailboxDrafts = useMemo(() => {
    if (!selectedMailbox?.id) return emailDrafts;
    return emailDrafts.filter((draft: any) => (draft.mailbox_id ?? "") === selectedMailbox.id);
  }, [emailDrafts, selectedMailbox?.id]);

  const visibleMailboxDrafts = useMemo(
    () => mailboxDrafts.filter((draft: any) => draft.status !== "discarded"),
    [mailboxDrafts],
  );

  useEffect(() => {
    if (activeFolder !== "drafts") return;
    void queryClient.invalidateQueries({ queryKey: ["crm", wsId, "email-drafts"] });
    if (!visibleMailboxDrafts.length) {
      setComposeDraft(null);
      return;
    }
    if (composeDraft?.draftId && visibleMailboxDrafts.some((draft: any) => draft.id === composeDraft.draftId)) return;
    openDraftInComposer(visibleMailboxDrafts[0]);
  }, [activeFolder, visibleMailboxDrafts, composeDraft?.draftId]);

  const folderThreads = useMemo(() => {
    return mailboxThreads.filter((thread) => {
      if (activeFolder === "trash") return thread.status === "trashed" || thread.is_trashed === true;
      if (activeFolder === "sent") return thread.direction === "outbound" && !thread.is_trashed;
      if (activeFolder === "archived") return thread.status === "archived" && !thread.is_trashed;
      if (activeFolder === "starred") return Boolean(thread.is_starred) && !thread.is_trashed;
      if (activeFolder === "unlinked") return !thread.account_id && !thread.is_trashed;
      return thread.status !== "archived" && thread.direction !== "outbound" && !thread.is_trashed;
    });
  }, [activeFolder, mailboxThreads]);

  const filteredThreads = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return folderThreads;
    return folderThreads.filter((thread) => [thread.subject, thread.mailbox, thread.direction, thread.status]
      .filter(Boolean)
      .join(" ")
      .toLowerCase()
      .includes(q));
  }, [folderThreads, search]);

  const folderCounts = useMemo(() => ({
    inbox: mailboxThreads.filter((thread) => thread.status !== "archived" && thread.direction !== "outbound" && !thread.is_trashed).length,
    sent: mailboxThreads.filter((thread) => thread.direction === "outbound" && !thread.is_trashed).length,
    drafts: visibleMailboxDrafts.length,
    archived: mailboxThreads.filter((thread) => thread.status === "archived" && !thread.is_trashed).length,
    starred: mailboxThreads.filter((thread) => thread.is_starred && !thread.is_trashed).length,
    unlinked: mailboxThreads.filter((thread) => !thread.account_id && !thread.is_trashed).length,
    trash: mailboxThreads.filter((thread) => thread.status === "trashed" || thread.is_trashed).length,
  } as const), [mailboxThreads, visibleMailboxDrafts.length]);

  const saveMailbox = useMutation({
    mutationFn: () => api.upsertCRMIMAPSetting({
      id: mailboxDraft.id,
      label: mailboxDraft.label,
      email: mailboxDraft.email,
      host: mailboxDraft.host,
      port: Number(mailboxDraft.port) || 993,
      tls_mode: mailboxDraft.tls_mode,
      username: mailboxDraft.username || mailboxDraft.email,
      secret_ref: mailboxDraft.secret_ref || null,
      secret: mailboxDraft.secret || null,
      sync_enabled: mailboxDraft.sync_enabled,
      owner_type: mailboxDraft.owner_type || null,
      owner_id: mailboxDraft.owner_id || null,
      smtp_host: mailboxDraft.smtp_host || null,
      smtp_port: Number(mailboxDraft.smtp_port) || null,
      smtp_tls_mode: mailboxDraft.smtp_tls_mode || null,
      smtp_username: mailboxDraft.smtp_username || null,
      smtp_secret_ref: mailboxDraft.smtp_secret_ref || null,
      smtp_secret: mailboxDraft.smtp_secret || null,
    }),
    onSuccess: (setting) => {
      setMailboxDraft(mailboxToDraft(setting));
      setMailboxStatus(t(($) => $.emails.mailbox_saved));
      queryClient.invalidateQueries({ queryKey: ["crm", wsId, "imap-settings"] });
    },
  });

  const testMailbox = useMutation({
    mutationFn: async () => {
      const setting = mailboxDraft.id ? null : await saveMailbox.mutateAsync();
      return api.testCRMIMAPSetting(setting?.id ?? mailboxDraft.id ?? "");
    },
    onSuccess: (result) => {
      setMailboxStatus(result.message);
      queryClient.invalidateQueries({ queryKey: ["crm", wsId, "imap-settings"] });
    },
  });

  const deleteMailbox = useMutation({
    mutationFn: (mailboxId: string) => api.deleteCRMIMAPSetting(mailboxId),
    onSuccess: async (_result, mailboxId) => {
      setMailboxStatus("Mailbox deleted.");
      setSelectedMailboxId((current) => current === mailboxId ? null : current);
      setMailboxDraft(emptyMailboxDraft);
      setPreviewMessages([]);
      setSelectedPreviewUIDs([]);
      await queryClient.invalidateQueries({ queryKey: ["crm", wsId, "imap-settings"] });
      await queryClient.invalidateQueries({ queryKey: ["crm", wsId, "imap-sync-runs"] });
    },
  });

  const previewMailbox = useMutation({
    mutationFn: () => api.previewCRMIMAP({ mailbox_id: mailboxDraft.id, folder: "INBOX", limit: 500, range_days: importRangeDays }),
    onSuccess: (result) => {
      setPreviewMessages(result.messages);
      setSelectedPreviewUIDs(result.messages.map((message) => message.uid));
      setMailboxStatus(emailCopy.fetched(result.note, result.total));
    },
  });

  const importPreviewMessages = useMutation({
    mutationFn: () => api.importCRMIMAP({ mailbox_id: mailboxDraft.id, folder: "INBOX", uids: selectedPreviewUIDs }),
    onSuccess: (result) => {
      setMailboxStatus(emailCopy.imported(result.imported, result.skipped));
      setPreviewMessages([]);
      setSelectedPreviewUIDs([]);
      queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
    },
  });

  const saveAndImportMailbox = async () => {
    setMailboxStatus(emailCopy.savingImporting);
    try {
      const setting = await saveMailbox.mutateAsync();
      setSelectedMailboxId(setting.id);
      const preview = await api.previewCRMIMAP({ mailbox_id: setting.id, folder: "INBOX", limit: 500, range_days: importRangeDays });
      const uids = preview.messages.map((message) => message.uid);
      setPreviewMessages(preview.messages);
      setSelectedPreviewUIDs(uids);
      if (!uids.length) {
        setMailboxStatus(emailCopy.mailboxSavedNoMessages);
        setSettingsOpen(false);
        return;
      }
      const result = await api.importCRMIMAP({ mailbox_id: setting.id, folder: "INBOX", uids });
      setMailboxStatus(emailCopy.savedImported(result.imported, result.skipped));
      queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
      queryClient.invalidateQueries({ queryKey: ["crm", wsId, "imap-settings"] });
      queryClient.invalidateQueries({ queryKey: ["crm", wsId, "imap-sync-runs"] });
      setSettingsOpen(false);
    } catch (error) {
      setMailboxStatus(`Import failed: ${mutationErrorMessage(error, "unknown error")}`);
    }
  };

  const sendDraft = useMutation({
    mutationFn: (draftId: string) => api.sendCRMEmailDraft(draftId),
    onSuccess: () => {
      setMailboxStatus(emailCopy.draftSent);
      setComposeDraft(null);
      setActiveFolder("sent");
      queryClient.invalidateQueries({ queryKey: ["crm", wsId, "email-drafts"] });
      queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
    },
    onError: (error) => {
      setMailboxStatus(`SMTP send failed: ${mutationErrorMessage(error, "unknown error")}`);
    },
  });

  const updateCachedThread = (thread: CRMEmailThread) => {
    queryClient.setQueriesData<any>({ queryKey: crmKeys.emailThreads(wsId) }, (current: any) => {
      if (Array.isArray(current)) return current.map((item) => item.id === thread.id ? { ...item, ...thread } : item);
      if (Array.isArray(current?.threads)) {
        return { ...current, threads: current.threads.map((item: CRMEmailThread) => item.id === thread.id ? { ...item, ...thread } : item) };
      }
      return current;
    });
    queryClient.setQueryData(crmKeys.emailThread(wsId, thread.id), thread);
  };

  const patchCachedThread = (threadId: string, patch: Partial<CRMEmailThread>) => {
    queryClient.setQueriesData<any>({ queryKey: crmKeys.emailThreads(wsId) }, (current: any) => {
      if (Array.isArray(current)) return current.map((item) => item.id === threadId ? { ...item, ...patch } : item);
      if (Array.isArray(current?.threads)) {
        return { ...current, threads: current.threads.map((item: CRMEmailThread) => item.id === threadId ? { ...item, ...patch } : item) };
      }
      return current;
    });
    queryClient.setQueryData<CRMEmailThread | undefined>(crmKeys.emailThread(wsId, threadId), (current) => (
      current ? { ...current, ...patch } : current
    ));
  };

  const updateThreadState = useMutation({
    mutationFn: ({ threadId, data }: { threadId: string; data: { status?: "open" | "archived"; is_read?: boolean; is_starred?: boolean } }) => api.updateCRMEmailThreadState(threadId, data),
    onMutate: async ({ threadId, data }) => {
      await queryClient.cancelQueries({ queryKey: crmKeys.emailThreads(wsId) });
      patchCachedThread(threadId, data);
    },
    onSuccess: async (thread) => {
      updateCachedThread(thread);
      await queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId), refetchType: "active" });
    },
    onError: async (error) => {
      setMailboxStatus(`Update failed: ${mutationErrorMessage(error, "unknown error")}`);
      await queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId), refetchType: "active" });
    },
  });

  const trashThread = useMutation({
    mutationFn: ({ threadId }: { threadId: string }) => api.trashCRMEmailThread(threadId),
    onSuccess: async () => {
      setMailboxStatus("Moved to trash.");
      setSelectedThreadId(null);
      await queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
    },
    onError: (error) => setMailboxStatus(`Trash failed: ${mutationErrorMessage(error, "unknown error")}`),
  });

  const restoreThread = useMutation({
    mutationFn: ({ threadId }: { threadId: string }) => api.restoreCRMEmailThread(threadId),
    onSuccess: async () => {
      setMailboxStatus("Restored.");
      setActiveFolder("inbox");
      setSelectedThreadId(null);
      await queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
    },
    onError: (error) => setMailboxStatus(`Restore failed: ${mutationErrorMessage(error, "unknown error")}`),
  });

  const deleteThread = useMutation({
    mutationFn: ({ threadId }: { threadId: string }) => api.deleteCRMEmailThread(threadId),
    onSuccess: async () => {
      setMailboxStatus("Deleted forever.");
      setSelectedThreadId(null);
      await queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
    },
    onError: (error) => setMailboxStatus(`Delete failed: ${mutationErrorMessage(error, "unknown error")}`),
  });

  const moveThread = useMutation({
    mutationFn: ({ threadId, folder }: { threadId: string; folder: string }) => api.moveCRMEmailThread(threadId, folder),
    onSuccess: async (_result, variables) => {
      setMailboxStatus(`Moved to ${variables.folder}.`);
      setActiveFolder(variables.folder as typeof activeFolder);
      setSelectedThreadId(null);
      await queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
    },
    onError: (error) => setMailboxStatus(`Move failed: ${mutationErrorMessage(error, "unknown error")}`),
  });

  const refreshMailbox = useMutation({
    mutationFn: () => api.syncCRMIMAP({ mailbox_id: selectedMailbox?.id ?? null, folder: activeFolder === "sent" ? "Sent" : "INBOX", limit: 500, range_days: importRangeDays }),
    onSuccess: async (result) => {
      setMailboxStatus(result.status === "running" ? "同步已开始，正在后台导入…" : emailCopy.imported(result.imported ?? 0, result.skipped ?? 0));
      await queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
      await queryClient.invalidateQueries({ queryKey: ["crm", wsId, "imap-sync-runs"] });
    },
    onError: (error) => setMailboxStatus(error instanceof Error ? error.message : "Sync failed"),
  });

  const saveEmailDraft = useMutation({
    mutationFn: async () => {
      if (!composeDraft) throw new Error(emailCopy.noDraft);
      const mailboxId = composeDraft.mailboxId || selectedMailbox?.id || mailboxes[0]?.id;
      if (!mailboxId) throw new Error(emailCopy.createMailboxFirst);
      const payload = {
        mailbox_id: mailboxId,
        thread_id: selectedThread?.id ?? null,
        account_id: composeDraft.accountId || null,
        contact_id: composeDraft.contactId || null,
        to_emails: composeDraft.to.split(/[;,\n]/).map((value) => value.trim()).filter(Boolean),
        cc_emails: composeDraft.cc.split(/[;,\n]/).map((value) => value.trim()).filter(Boolean),
        bcc_emails: composeDraft.bcc.split(/[;,\n]/).map((value) => value.trim()).filter(Boolean),
        subject: composeDraft.subject.trim(),
        body_text: composeDraft.body,
        attachments: composeDraft.attachments.map(({ file_name, content_type, content }) => ({ file_name, content_type, content })),
      };
      return composeDraft.draftId ? api.updateCRMEmailDraft(composeDraft.draftId, payload) : api.createCRMEmailDraft(payload);
    },
    onSuccess: () => {
      setComposeDraft(null);
      setMailboxStatus(emailCopy.draftSaved);
      setActiveFolder("drafts");
      queryClient.invalidateQueries({ queryKey: ["crm", wsId, "email-drafts"] });
    },
  });

  const selectedThread = useMemo<CRMEmailThread | null>(() => {
    const found = filteredThreads.find((thread) => thread.id === selectedThreadId) ?? filteredThreads[0] ?? null;
    return found;
  }, [filteredThreads, selectedThreadId]);

  const linkedAccountId = selectedThread?.account_id ?? "";
  const { data: contacts = [] } = useQuery({
    ...crmContactListOptions(wsId, linkedAccountId),
    enabled: Boolean(linkedAccountId),
  });
  const draftAccountId = associationDraft?.accountId ?? "";
  const { data: draftAccountContacts = [] } = useQuery({
    ...crmContactListOptions(wsId, draftAccountId),
    enabled: Boolean(draftAccountId),
  });
  const composeAccountId = composeDraft?.accountId ?? "";
  const { data: composeAccountContacts = [] } = useQuery({
    ...crmContactListOptions(wsId, composeAccountId),
    enabled: Boolean(composeAccountId),
  });
  const filteredComposeAccounts = useMemo(() => {
    const q = composeAccountSearch.trim().toLowerCase();
    if (!q) return accounts;
    return accounts.filter((account) => [account.name, account.website, account.industry, account.country]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(q)));
  }, [accounts, composeAccountSearch]);
  const { data: messages = [], isLoading: messagesLoading } = useQuery({
    ...crmEmailMessageListOptions(wsId, selectedThread?.id ?? ""),
    enabled: Boolean(selectedThread?.id),
  });
  const { data: associationSuggestions = [] } = useQuery({
    queryKey: [...crmKeys.emailThread(wsId, selectedThread?.id ?? ""), "association-suggestions"],
    queryFn: async () => (await api.listCRMEmailThreadAssociationSuggestions(selectedThread?.id ?? "")).suggestions,
    enabled: Boolean(selectedThread?.id && !selectedThread?.account_id),
  });
  const { data: projects = [] } = useQuery({
    queryKey: ["projects", wsId, "crm-email-link-picker"],
    queryFn: async () => (await api.listProjects()).projects,
  });
  const { data: issues = [] } = useQuery({
    queryKey: ["issues", wsId, "crm-email-link-picker", emailLinkDraft?.projectId ?? selectedThread?.project_id ?? ""],
    queryFn: async () => (await api.listIssues({ project_id: emailLinkDraft?.projectId || selectedThread?.project_id || undefined, limit: 100 })).issues,
  });

  const selectedAccount = accounts.find((account) => account.id === linkedAccountId) ?? null;
  const selectedContact = contacts.find((contact) => contact.id === (selectedThread?.contact_id ?? "")) ?? null;
  const selectedProject = projects.find((project) => project.id === (selectedThread?.project_id ?? "")) ?? null;
  const selectedIssueIds = Array.from(new Set(selectedThread?.issue_ids?.length ? selectedThread.issue_ids : selectedThread?.issue_id ? [selectedThread.issue_id] : []));
  const selectedIssues = issues.filter((issue) => selectedIssueIds.includes(issue.id));
  const defaultProjectTitle = selectedAccount ? `CRM:${selectedAccount.name}` : "";
  const crmNamedProject = selectedAccount ? projects.find((project) => project.title === defaultProjectTitle) : null;


  const openAssociationDialog = (suggestion?: CRMEmailThreadAssociationSuggestion) => {
    const inferred = inferContactDraft(messages);
    setAssociationDraft({
      accountId: suggestion?.account_id ?? selectedThread?.account_id ?? "",
      contactId: suggestion?.contact_id ?? selectedThread?.contact_id ?? "",
      contactName: suggestion?.contact_name ?? selectedContact?.name ?? inferred.contactName,
      contactEmail: suggestion?.contact_email ?? selectedContact?.email ?? inferred.contactEmail,
    });
  };

  const openComposeDraft = (mode: "new" | "reply" | "reply-all" | "forward" = "reply") => {
    const inbound = messages.find((message) => message.direction === "inbound" && message.from_email);
    const lastMsg = messages[messages.length - 1] || inbound;
    const subjectBase = selectedThread?.subject ?? "";
    const subject = mode === "forward"
      ? (subjectBase.toLowerCase().startsWith("fwd:") ? subjectBase : `Fwd: ${subjectBase}`)
      : subjectBase
        ? (subjectBase.toLowerCase().startsWith("re:") ? subjectBase : `Re: ${subjectBase}`)
        : "";
    const replyAll = mode === "reply-all";
    const date = lastMsg ? messageTime(lastMsg.sent_at || lastMsg.received_at) : "";
    const from = lastMsg?.from_name || lastMsg?.from_email || "";
    const originalBody = lastMsg?.body_text || "";
    let body = "";
    if (mode === "reply" || mode === "reply-all") {
      body = `\n\n\n> On ${date} ${from} wrote:\n> ${originalBody.split('\n').join('\n> ')}`;
    } else if (mode === "forward") {
      body = `\n\n---- Forwarded message ----\nSubject: ${subjectBase}\nFrom: ${from}\nDate: ${date}\n\n${originalBody}`;
    }
    setComposeDraft({
      mailboxId: selectedMailbox?.id ?? mailboxes[0]?.id ?? "",
      accountId: selectedThread?.account_id ?? "",
      contactId: selectedThread?.contact_id ?? "",
      to: mode === "new" || mode === "forward" ? "" : inbound?.from_email ?? "",
      cc: replyAll ? (inbound?.cc_emails ?? []).join(", ") : "",
      bcc: "",
      subject,
      body,
      attachments: mode === "forward" ? (lastMsg?.attachments?.map((a: any) => ({file_name: a.filename || 'attachment', content_type: a.content_type || 'application/octet-stream', content: a.content || '', size: a.size_bytes})) || []) : [],
    });
  };

  const updateAssociation = useMutation({
    mutationFn: async () => {
      if (!selectedThread || !associationDraft) throw new Error("No email association draft selected");
      let contactId = associationDraft.contactId || null;
      if (!contactId && associationDraft.accountId && associationDraft.contactName.trim()) {
        const payload: CreateCRMContactRequest = {
          account_id: associationDraft.accountId,
          name: associationDraft.contactName.trim(),
          email: associationDraft.contactEmail.trim() || null,
          is_primary: false,
        };
        const contact = await api.createCRMContact(associationDraft.accountId, payload);
        contactId = contact.id;
      }
      return api.updateCRMEmailThreadAssociation(selectedThread.id, {
        account_id: associationDraft.accountId || null,
        contact_id: contactId,
      });
    },
    onSuccess: async (thread) => {
      setAssociationDraft(null);
      await queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
      await queryClient.invalidateQueries({ queryKey: crmKeys.emailThread(wsId, thread.id) });
      if (thread.account_id) await queryClient.invalidateQueries({ queryKey: crmKeys.contacts(wsId, thread.account_id) });
    },
  });

  const openEmailLinkDialog = async () => {
    if (!selectedThread || !selectedAccount) return;
    let projectId = selectedThread.project_id ?? crmNamedProject?.id ?? "";
    if (!projectId) {
      const project = await api.createProject({
        title: defaultProjectTitle,
        status: "in_progress",
        priority: "medium",
        resources: [{ resource_type: "crm_account", resource_ref: { account_id: selectedAccount.id }, label: selectedAccount.name }],
      });
      projectId = project.id;
      await queryClient.invalidateQueries({ queryKey: ["projects", wsId, "crm-email-link-picker"] });
    }
    setEmailLinkDraft({ projectId, issueIds: Array.from(new Set(selectedThread.issue_ids?.length ? selectedThread.issue_ids : selectedThread.issue_id ? [selectedThread.issue_id] : [])) });
  };

  const updateEmailLinks = useMutation({
    mutationFn: async () => {
      if (!selectedThread || !emailLinkDraft) throw new Error("No email link draft selected");
      return api.updateCRMEmailThreadAssociation(selectedThread.id, {
        account_id: selectedThread.account_id ?? null,
        contact_id: selectedThread.contact_id ?? null,
        project_id: emailLinkDraft.projectId || null,
        issue_id: emailLinkDraft.issueIds[0] ?? null,
        issue_ids: emailLinkDraft.issueIds,
      });
    },
    onSuccess: async (thread) => {
      setEmailLinkDraft(null);
      await queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
      await queryClient.invalidateQueries({ queryKey: crmKeys.emailThread(wsId, thread.id) });
    },
  });

  const openCreateFollowUpIssue = () => {
    if (!selectedThread || !emailLinkDraft) return;
    clearIssueDraft();
    setIssueDraft({
      title: `${t(($) => $.emails.follow_up_issue_prefix)} ${selectedThread.subject}`.trim(),
      description: selectedThread.subject,
      priority: "medium",
      status: "in_review",
    });
    openModal("create-issue", {
      project_id: emailLinkDraft.projectId,
      onCreated: async (issue: Issue) => {
        const nextIssueIds = Array.from(new Set([...emailLinkDraft.issueIds, issue.id]));
        setEmailLinkDraft({ ...emailLinkDraft, issueIds: nextIssueIds });
        await api.updateCRMEmailThreadAssociation(selectedThread.id, {
          account_id: selectedThread.account_id ?? null,
          contact_id: selectedThread.contact_id ?? null,
          project_id: emailLinkDraft.projectId || null,
          issue_id: nextIssueIds[0] ?? null,
          issue_ids: nextIssueIds,
        });
        await queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
        await queryClient.invalidateQueries({ queryKey: issueKeys.all(wsId) });
      },
    });
  };

  const draftContacts = associationDraft?.accountId ? draftAccountContacts : [];

  return (
    <div className="flex h-full flex-col bg-muted/20">
      <PageHeader className="justify-between border-b bg-background px-5">
        <div className="flex items-center gap-2">
          <Mail className="size-4 text-muted-foreground" />
          <h1 className="text-sm font-medium">{t(($) => $.emails.workspace_title)}</h1>
          {!isLoading && <Badge variant="secondary" className="tabular-nums">{threads.length}</Badge>}
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" disabled={!mailboxes.length} onClick={() => openComposeDraft("new")}>
            <Send className="mr-1 size-3" />
            {emailCopy.compose}
          </Button>
          <Button variant="outline" size="sm" title="Diagnostics" onClick={() => setDiagnosticsOpen(true)}>
            <Wrench className="size-3" />
          </Button>
          <Button variant="outline" size="sm" onClick={() => { setMailboxDraft(emptyMailboxDraft); setMailboxStatus(null); setSettingsOpen(true); }}>
            <Settings className="mr-1 size-3" />
            {t(($) => $.emails.mailbox_settings)}
          </Button>
        </div>
      </PageHeader>

      <div className={`grid min-h-0 flex-1 grid-cols-1 gap-0 ${composeFullWidth ? "lg:grid-cols-1" : "lg:grid-cols-[220px_360px_minmax(0,1fr)]"}`}>
        <aside className={`min-h-0 flex-col border-r bg-card/80 p-3 ${composeFullWidth ? "hidden" : "flex"}`}>
          <div className="mb-3 rounded-lg border bg-background p-3">
            <div className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{t(($) => $.emails.mailboxes)}</div>
            <select
              aria-label="Active mailbox"
              className="mt-2 h-9 w-full rounded-md border bg-background px-2 text-sm"
              value={selectedMailbox?.id ?? ""}
              onChange={(event) => setSelectedMailboxId(event.target.value || null)}
            >
              {mailboxes.length === 0 ? <option value="">{emailCopy.fallbackMailbox}</option> : null}
              {mailboxes.map((mailbox) => <option key={mailbox.id} value={mailbox.id}>{mailbox.label || mailbox.email}</option>)}
            </select>
            {activeSyncRuns.some((run: any) => !selectedMailbox || run.mailbox_id === selectedMailbox.id) ? (
              <div className="mt-2 text-xs text-muted-foreground">{emailCopy.syncing}…</div>
            ) : null}
            {selectedMailbox ? (
              <Button className="mt-3 w-full" size="sm" variant="outline" disabled={refreshMailbox.isPending} onClick={() => refreshMailbox.mutate()}>
                {refreshMailbox.isPending ? emailCopy.syncing : "Refresh new mail"}
              </Button>
            ) : null}
          </div>
          <nav className="space-y-1" aria-label={t(($) => $.emails.folder_nav)}>
            {([
              ["inbox", Inbox, t(($) => $.emails.folder_inbox)],
              ["sent", MailOpen, t(($) => $.emails.folder_sent)],
              ["drafts", Send, emailCopy.drafts],
              ["archived", Archive, t(($) => $.emails.folder_archived)],
              ["starred", Star, t(($) => $.emails.folder_starred)],
              ["unlinked", Link2, t(($) => $.emails.folder_unlinked)],
              ["trash", Trash2, "Trash"],
            ] as const).map(([folder, Icon, label]) => (
              <button
                key={folder}
                type="button"
                className={`flex w-full items-center justify-between rounded-md px-3 py-2 text-sm hover:bg-muted ${activeFolder === folder ? "bg-muted font-medium" : ""}`}
                onClick={() => setActiveFolder(folder)}
              >
                <span className="flex items-center gap-2"><Icon className="size-4 text-muted-foreground" />{label}</span>
                <Badge variant="secondary" className="tabular-nums">{folderCounts[folder]}</Badge>
              </button>
            ))}
          </nav>
          <Button className="mt-auto" variant="outline" onClick={() => { setMailboxDraft(emptyMailboxDraft); setMailboxStatus(null); setSettingsOpen(true); }}>{t(($) => $.emails.add_mailbox)}</Button>
        </aside>

        <aside className={`min-h-0 flex-col border-r bg-background ${composeFullWidth ? "hidden" : "flex"}`}>
          <div className="border-b p-3">
            <div className="relative">
              <Search className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
              <Input className="pl-8" placeholder={t(($) => $.emails.search_placeholder)} value={search} onChange={(event) => setSearch(event.target.value)} />
            </div>
          </div>
          {isLoading ? (
            <section className="space-y-2 p-3">
              <Skeleton className="h-16 w-full" />
              <Skeleton className="h-16 w-full" />
            </section>
          ) : activeFolder === "drafts" ? (
            <section className="min-h-0 flex-1 overflow-y-auto p-3">
              {visibleMailboxDrafts.length === 0 ? <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">{emailCopy.noDrafts}</div> : visibleMailboxDrafts.map((draft: any) => {
                const active = composeDraft?.draftId === draft.id;
                return (
                  <button key={draft.id} type="button" className={`mb-2 block w-full rounded-lg border bg-card p-3 text-left text-sm hover:bg-muted/60 ${active ? "ring-2 ring-primary/40" : ""}`} onClick={() => openDraftInComposer(draft)}>
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <div className="truncate font-medium">{draft.subject || emailCopy.noSubject}</div>
                        <div className="truncate text-xs text-muted-foreground">{emailCopy.to}: {(draft.to_emails ?? []).join(", ") || "—"}</div>
                      </div>
                      <Badge variant="outline">{draft.status}</Badge>
                    </div>
                    <p className="mt-2 line-clamp-3 text-xs text-muted-foreground">{draft.body_text}</p>
                  </button>
                );
              })}
            </section>
          ) : filteredThreads.length === 0 ? (
            <section className="m-3 rounded-lg border border-dashed bg-card p-10 text-center">
              <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
                <Mail className="size-5" />
              </div>
              <h2 className="mt-4 text-base font-semibold">{t(($) => $.emails.empty_title)}</h2>
              <p className="mx-auto mt-2 max-w-xl text-sm text-muted-foreground">
                {t(($) => $.emails.empty_description)}
              </p>
            </section>
          ) : (
            <section className="min-h-0 flex-1 overflow-y-auto">
              {filteredThreads.map((thread) => {
                const active = selectedThread?.id === thread.id;
                const isUnread = thread.is_read !== true;
                return (
                  <button key={thread.id} type="button" className={`block w-full border-b px-4 py-3 text-left text-sm hover:bg-muted/60 ${active ? "bg-muted" : ""}`} onClick={() => { setSelectedThreadId(thread.id); if (isUnread) updateThreadState.mutate({ threadId: thread.id, data: { is_read: true } }); }}>
                    <div className="flex items-start justify-between gap-2">
                      <div className={`min-w-0 flex-1 truncate ${isUnread ? "font-bold text-foreground" : "font-normal text-foreground/70"}`}>{thread.subject}</div>
                      {!thread.account_id && <Badge variant="outline">{t(($) => $.emails.unlinked_badge)}</Badge>}
                    </div>
                    <div className="mt-1 truncate text-xs text-muted-foreground">
                      {[thread.mailbox, thread.direction, thread.status, t(($) => $.common.count_messages, { count: thread.message_count })].filter(Boolean).join(" · ")}
                    </div>
                  </button>
                );
              })}
            </section>
          )}
        </aside>

        <section className="min-h-0 overflow-hidden bg-background">
          {composeDraft ? (
            <div className="flex h-full min-h-0 flex-col bg-background">
              <div className="border-b p-5">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h2 className="text-base font-semibold">{emailCopy.composeTitle}</h2>
                    <p className="mt-1 text-xs text-muted-foreground">{emailCopy.composeHelp}</p>
                  </div>
                  <Button variant="outline" size="sm" onClick={() => setComposeDraft(null)}>{emailCopy.cancel}</Button>
                </div>
              </div>
              <div className="min-h-0 flex-1 overflow-y-auto p-5">
                <div className="h-full w-full space-y-3 rounded-lg border bg-card p-4">
                  <label className="space-y-1 text-sm">
                    <span className="text-xs font-medium text-muted-foreground">{emailCopy.mailbox}</span>
                    <select className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={composeDraft.mailboxId} onChange={(event) => setComposeDraft({ ...composeDraft, mailboxId: event.target.value })}>
                      {mailboxes.map((mailbox) => <option key={mailbox.id} value={mailbox.id}>{mailbox.label} · {mailbox.email}</option>)}
                    </select>
                  </label>
                  <div className="flex gap-2">
                    <Input aria-label={emailCopy.to} placeholder={emailCopy.to} value={composeDraft.to} onChange={(event) => setComposeDraft({ ...composeDraft, to: event.target.value })} />
                    <Button type="button" variant="outline" onClick={() => setComposeRecipientPickerOpen(true)}>{t(($) => $.emails.link_customer_contact)}</Button>
                  </div>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <Input aria-label={emailCopy.cc} placeholder={emailCopy.cc} value={composeDraft.cc} onChange={(event) => setComposeDraft({ ...composeDraft, cc: event.target.value })} />
                    <Input aria-label={emailCopy.bcc} placeholder={emailCopy.bcc} value={composeDraft.bcc} onChange={(event) => setComposeDraft({ ...composeDraft, bcc: event.target.value })} />
                  </div>
                  <Input aria-label={emailCopy.subject} placeholder={emailCopy.subject} value={composeDraft.subject} onChange={(event) => setComposeDraft({ ...composeDraft, subject: event.target.value })} />
                  <textarea aria-label={emailCopy.bodyLabel} className="min-h-64 w-full rounded-md border bg-background px-3 py-2 text-sm outline-none ring-offset-background placeholder:text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" placeholder={emailCopy.bodyPlaceholder} value={composeDraft.body} onChange={(event) => setComposeDraft({ ...composeDraft, body: event.target.value })} />
                  <div className="rounded-md border bg-muted/20 p-3 text-sm">
                    <div className="mb-2 flex items-center justify-between text-xs font-medium text-muted-foreground"><span>{emailCopy.attachments}</span><label className="inline-flex cursor-pointer items-center gap-1 rounded border bg-background px-2 py-1 hover:bg-muted"><Paperclip className="size-3" />添加附件<input type="file" multiple className="hidden" onChange={async (event) => { const files = Array.from(event.target.files ?? []); const added = await Promise.all(files.map((file) => new Promise<ComposeAttachment>((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve({ file_name: file.name, content_type: file.type || "application/octet-stream", content: String(reader.result || "").split(",")[1] || "", size: file.size }); reader.onerror = () => reject(reader.error); reader.readAsDataURL(file); }))); setComposeDraft({ ...composeDraft, attachments: [...composeDraft.attachments, ...added] }); event.currentTarget.value = ""; }} /></label></div>
                    {composeDraft.attachments.length ? <div className="space-y-2">{composeDraft.attachments.map((attachment, index) => <div key={`${attachment.file_name}-${index}`} className="flex items-center justify-between rounded border bg-background px-3 py-2 text-xs"><span className="truncate">{attachment.file_name} · {Math.ceil(attachment.size / 1024)} KB</span><Button variant="ghost" size="sm" onClick={() => setComposeDraft({ ...composeDraft, attachments: composeDraft.attachments.filter((_, itemIndex) => itemIndex !== index) })}><Trash2 className="size-3" /></Button></div>)}</div> : <div className="text-xs text-muted-foreground">{emailCopy.noAttachments}</div>}
                  </div>
                  {saveEmailDraft.isError && <p className="text-xs text-destructive">{emailCopy.saveDraftError}</p>}
                  <div className="flex justify-end gap-2 border-t pt-3">
                    <Button variant="outline" onClick={() => setComposeDraft(null)}>{emailCopy.cancel}</Button>
                    <Button disabled={!composeDraft.to.trim() || !composeDraft.subject.trim() || !composeDraft.body.trim() || saveEmailDraft.isPending || sendDraft.isPending} onClick={async () => { const draft = await saveEmailDraft.mutateAsync(); sendDraft.mutate(draft.id); }}>{emailCopy.send}</Button>
                    <Button disabled={!composeDraft.to.trim() || !composeDraft.subject.trim() || !composeDraft.body.trim() || saveEmailDraft.isPending} onClick={() => saveEmailDraft.mutate()}>{emailCopy.saveDraft}</Button>
                  </div>
                </div>
              </div>
            </div>
          ) : !selectedThread ? (
            <div className="p-10 text-center text-sm text-muted-foreground">{t(($) => $.emails.select_thread)}</div>
          ) : (
            <div className="flex h-full min-h-0 flex-col">
              <div className="border-b bg-background p-3">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <h2 className={`truncate text-base ${selectedThread.is_read === true ? "font-semibold text-foreground/80" : "font-bold text-foreground"}`}>{selectedThread.subject}</h2>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {[selectedThread.mailbox, selectedThread.direction, selectedThread.status, messageTime(selectedThread.last_message_at)].filter(Boolean).join(" · ")}
                    </p>
                  </div>
                  <Button variant={selectedAccount ? "outline" : "default"} size="sm" onClick={() => openAssociationDialog()}>
                    <Link2 className="mr-1 size-3" />
                    {selectedAccount ? t(($) => $.emails.change_association) : t(($) => $.emails.link_customer_contact)}
                  </Button>
                </div>
                <div className="mt-2 flex flex-wrap items-center gap-1.5 border-t pt-2">
                  <Button variant="outline" size="sm" disabled={updateThreadState.isPending} onClick={() => updateThreadState.mutate({ threadId: selectedThread.id, data: { is_read: selectedThread.is_read !== true } })}>{selectedThread.is_read === true ? <Mail className="mr-1 size-3" /> : <MailOpen className="mr-1 size-3" />}{selectedThread.is_read === true ? "标记未读" : t(($) => $.emails.mark_read)}</Button>
                  <Button variant="outline" size="sm" disabled={updateThreadState.isPending} onClick={() => updateThreadState.mutate({ threadId: selectedThread.id, data: { status: selectedThread.status === "archived" ? "open" : "archived" } })}><Archive className="mr-1 size-3" />{selectedThread.status === "archived" ? "Unarchive" : t(($) => $.emails.archive)}</Button>
                  <Button variant="outline" size="sm" disabled={updateThreadState.isPending} onClick={() => updateThreadState.mutate({ threadId: selectedThread.id, data: { is_starred: !selectedThread.is_starred } })}><Star className="mr-1 size-3" />{selectedThread.is_starred ? "Unstar" : t(($) => $.emails.star)}</Button>
                  <Button variant="outline" size="sm" disabled={!mailboxes.length} onClick={() => openComposeDraft("reply")}><Send className="mr-1 size-3" />{emailCopy.reply}</Button>
                  <Button variant="outline" size="sm" disabled={!mailboxes.length} onClick={() => openComposeDraft("reply-all")}>{emailCopy.replyAll}</Button>
                  <Button variant="outline" size="sm" disabled={!mailboxes.length} onClick={() => openComposeDraft("forward")}>{emailCopy.forward}</Button>
                  <Button variant="outline" size="sm" disabled={!selectedAccount} onClick={openEmailLinkDialog}><Link2 className="mr-1 size-3" />{t(($) => $.emails.link_project_issue)}</Button>
                  {activeFolder !== "trash" ? (
                    <Button variant="outline" size="sm" disabled={trashThread.isPending} onClick={() => trashThread.mutate({ threadId: selectedThread.id })}><Trash2 className="mr-1 size-3" />Trash</Button>
                  ) : (
                    <>
                      <Button variant="outline" size="sm" disabled={restoreThread.isPending} onClick={() => restoreThread.mutate({ threadId: selectedThread.id })}><Undo2 className="mr-1 size-3" />Restore</Button>
                      <Button variant="destructive" size="sm" disabled={deleteThread.isPending} onClick={() => { if (window.confirm("Delete forever? This cannot be undone.")) deleteThread.mutate({ threadId: selectedThread.id }); }}><Trash2 className="mr-1 size-3" />Delete forever</Button>
                    </>
                  )}
                  <div className="relative inline-flex items-center">
                    <select
                      aria-label="Move to folder"
                      className="h-8 rounded-md border bg-background px-2 text-xs"
                      value=""
                      onChange={(e) => { if (e.target.value) { moveThread.mutate({ threadId: selectedThread.id, folder: e.target.value }); } }}
                    >
                      <option value="">Move to…</option>
                      <option value="inbox">Inbox</option>
                      <option value="sent">Sent</option>
                      <option value="archived">Archived</option>
                      <option value="starred">Starred</option>
                      <option value="trash">Trash</option>
                    </select>
                  </div>
                </div>
                <div className="mt-2 flex flex-wrap items-center gap-1.5">
                  <AssociationChip icon={<Building2 className="size-4" />} label={t(($) => $.emails.linked_customer)} value={selectedAccount?.name ?? t(($) => $.emails.no_customer)} onClick={selectedAccount ? () => setDetailDialog({ type: "account", account: selectedAccount }) : undefined} />
                  <AssociationChip icon={<UserRound className="size-4" />} label={t(($) => $.emails.linked_contact)} value={selectedContact?.name ?? t(($) => $.emails.no_contact)} onClick={selectedContact ? () => setDetailDialog({ type: "contact", contact: selectedContact }) : undefined} />
                  <AssociationChip icon={<Building2 className="size-4" />} label={t(($) => $.emails.related_project)} value={selectedProject?.title ?? t(($) => $.emails.no_project_link)} />
                  <AssociationChip icon={<Link2 className="size-4" />} label={t(($) => $.emails.related_issue)} value={selectedIssues.length ? selectedIssues.map((issue) => issue.identifier).join(", ") : t(($) => $.emails.no_issue_link)} />
                  {selectedAccount && (
                    <Button variant="ghost" size="sm" onClick={() => navigation.push(paths.crmCustomerDetail(selectedAccount.id))}>
                      {t(($) => $.emails.open_customer)} <ArrowRight className="ml-1 size-3" />
                    </Button>
                  )}
                </div>
                {!selectedAccount && associationSuggestions.length > 0 && (
                  <div className="mt-3 rounded-lg border bg-muted/20 p-3">
                    <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">{t(($) => $.emails.association_suggestions)}</div>
                    <div className="space-y-2">
                      {associationSuggestions.map((suggestion) => (
                        <div key={`${suggestion.account_id}:${suggestion.contact_id ?? "account"}`} className="flex flex-wrap items-start justify-between gap-3 rounded-md border bg-background p-3 text-sm">
                          <div className="min-w-0 flex-1">
                            <div className="font-medium">{suggestion.account_name}{suggestion.contact_name ? ` · ${suggestion.contact_name}` : ""}</div>
                            <div className="mt-1 text-xs text-muted-foreground">{suggestion.reasons.join("; ")}</div>
                          </div>
                          <Button size="sm" variant="outline" onClick={() => openAssociationDialog(suggestion)}>{t(($) => $.emails.use_suggestion)}</Button>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                {updateAssociation.isError && <p className="mt-2 text-xs text-destructive">{t(($) => $.emails.association_error)}</p>}
              </div>
              <div className="min-h-0 flex-1 overflow-y-auto bg-muted/20 p-3">
                {messagesLoading ? (
                  <div className="space-y-2">
                    <Skeleton className="h-24 w-full" />
                    <Skeleton className="h-24 w-full" />
                  </div>
                ) : messages.length === 0 ? (
                  <div className="rounded-lg border border-dashed bg-background p-8 text-center text-sm text-muted-foreground">{t(($) => $.emails.no_messages)}</div>
                ) : (
                  <div className="space-y-2">
                    {messages.map((message) => (
                      <article key={message.id} className="rounded-lg border bg-background p-3 text-sm shadow-xs">
                        <div className="flex flex-wrap justify-between gap-2">
                          <div className="font-medium">{message.from_name || message.from_email || t(($) => $.common.not_available)}</div>
                          <div className="text-xs text-muted-foreground">{messageTime(message.sent_at || message.received_at)}</div>
                        </div>
                        <div className="mt-2 grid gap-1.5 text-xs text-muted-foreground sm:grid-cols-2">
                          <DetailRow label={emailCopy.from} value={[message.from_name, message.from_email].filter(Boolean).join(" <") + (message.from_name && message.from_email ? ">" : "")} />
                          <DetailRow label={emailCopy.to} value={message.to_emails.join(", ")} />
                          <DetailRow label={emailCopy.cc} value={message.cc_emails.join(", ")} />
                          <DetailRow label={emailCopy.date} value={messageTime(message.sent_at || message.received_at)} />
                        </div>
                        <div className="mt-2 rounded-md border bg-muted/20 p-2.5">
                          <div className="mb-1.5 text-xs font-medium text-muted-foreground">{emailCopy.htmlBody}</div>
                          {emailHTMLBodyWithCID(message) ? (
                            <div className="leading-5 text-foreground/80" dangerouslySetInnerHTML={{ __html: safeEmailHTML(emailHTMLBodyWithCID(message)) }} />
                          ) : (
                            <div className="whitespace-pre-wrap leading-5 text-foreground/80">{message.body_text || message.snippet || t(($) => $.emails.no_body)}</div>
                          )}
                        </div>
                        <div className="mt-2 rounded-md border bg-muted/20 p-2.5">
                          <div className="text-xs font-medium text-muted-foreground">{emailCopy.attachments}</div>
                          {message.attachments?.length ? (
                            <div className="mt-2 space-y-2">
                              {message.attachments.map((attachment, index) => (
                                <div key={`${message.id}-attachment-${index}`} className="rounded border bg-background px-3 py-2 text-xs">
                                  <a
                                    href={api.getCRMEmailAttachmentUrl(wsId, message.id, index)}
                                    download={attachment.filename || `attachment-${index}`}
                                    className="font-medium text-primary hover:underline"
                                  >{attachment.filename || attachment.content_id || emailCopy.attachments}</a>
                                  <div className="mt-1 text-muted-foreground">{[attachment.content_type, attachment.disposition, `${attachment.size_bytes} ${emailCopy.bytes}`].filter(Boolean).join(" · ")}</div>
                                </div>
                              ))}
                            </div>
                          ) : <div className="mt-2 text-xs text-muted-foreground">{emailCopy.noAttachments}</div>}
                        </div>
                        <details className="mt-2 rounded-md border bg-muted/20 p-2.5 text-xs">
                          <summary className="cursor-pointer font-medium text-muted-foreground">{emailCopy.mimeMetadata}</summary>
                          <div className="mt-3 grid gap-2 sm:grid-cols-2">
                            <DetailRow label={emailCopy.messageId} value={message.external_message_id} />
                            <DetailRow label={emailCopy.inReplyTo} value={message.in_reply_to} />
                            <DetailRow label={emailCopy.references} value={message.reference_ids?.join(", ")} />
                            <DetailRow label={emailCopy.rawSize} value={message.raw_size_bytes ? `${message.raw_size_bytes} ${emailCopy.bytes}` : ""} />
                            <DetailRow label={emailCopy.contentType} value={message.raw_headers?.["Content-Type"]?.join(", ") ?? message.raw_headers?.["content-type"]?.join(", ")} />
                            <DetailRow label={emailCopy.contentDisposition} value={message.raw_headers?.["Content-Disposition"]?.join(", ") ?? message.raw_headers?.["content-disposition"]?.join(", ")} />
                          </div>
                        </details>
                      </article>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </section>
      </div>

      <Dialog open={detailDialog !== null} onOpenChange={(open) => !open && setDetailDialog(null)}>
        <DialogContent className="sm:max-w-lg">
          {detailDialog?.type === "account" && (
            <>
              <DialogHeader>
                <DialogTitle>{detailDialog.account.name}</DialogTitle>
                <DialogDescription>{t(($) => $.emails.customer_detail)}</DialogDescription>
              </DialogHeader>
              <div className="grid gap-3 sm:grid-cols-2">
                <DetailRow label={t(($) => $.customers.status)} value={t(($) => $.statuses[detailDialog.account.status])} />
                <DetailRow label={t(($) => $.customers.rating)} value={t(($) => $.ratings[detailDialog.account.rating])} />
                <DetailRow label={t(($) => $.customers.priority)} value={t(($) => $.priorities[detailDialog.account.priority])} />
                <DetailRow label={t(($) => $.customers.country)} value={detailDialog.account.country_name || detailDialog.account.country} />
                <DetailRow label={t(($) => $.customers.website)} value={detailDialog.account.website} />
                <DetailRow label={t(($) => $.customers.next_follow_up_at)} value={messageTime(detailDialog.account.next_follow_up_at)} />
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setDetailDialog(null)}>{t(($) => $.actions.cancel)}</Button>
                <Button onClick={() => navigation.push(paths.crmCustomerDetail(detailDialog.account.id))}>{t(($) => $.emails.open_customer)}</Button>
              </DialogFooter>
            </>
          )}
          {detailDialog?.type === "contact" && (
            <>
              <DialogHeader>
                <DialogTitle>{detailDialog.contact.name}</DialogTitle>
                <DialogDescription>{t(($) => $.emails.contact_detail)}</DialogDescription>
              </DialogHeader>
              <div className="grid gap-3 sm:grid-cols-2">
                <DetailRow label={t(($) => $.contacts.email)} value={detailDialog.contact.email} />
                <DetailRow label={t(($) => $.contacts.phone)} value={detailDialog.contact.phone || detailDialog.contact.mobile} />
                <DetailRow label={t(($) => $.contacts.whatsapp)} value={detailDialog.contact.whatsapp || detailDialog.contact.whatsapp_id} />
                <DetailRow label={t(($) => $.contacts.job_title)} value={detailDialog.contact.job_title || detailDialog.contact.role_title} />
                <DetailRow label={t(($) => $.contacts.department)} value={detailDialog.contact.department} />
                <DetailRow label={t(($) => $.contacts.preferred_language)} value={detailDialog.contact.preferred_language || detailDialog.contact.language} />
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={associationDraft !== null} onOpenChange={(open) => !open && setAssociationDraft(null)}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>{t(($) => $.emails.link_customer_contact)}</DialogTitle>
            <DialogDescription>{t(($) => $.emails.link_help)}</DialogDescription>
          </DialogHeader>
          {associationDraft && (
            <div className="space-y-4">
              <label className="space-y-1 text-sm">
                <span className="text-xs font-medium text-muted-foreground">{t(($) => $.emails.linked_customer)}</span>
                <select aria-label={t(($) => $.emails.linked_customer)} className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={associationDraft.accountId} onChange={(event) => setAssociationDraft({ ...associationDraft, accountId: event.target.value, contactId: "" })}>
                  <option value="">{t(($) => $.emails.no_customer)}</option>
                  {accounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}
                </select>
              </label>
              {associationDraft.accountId && draftContacts.length > 0 && (
                <label className="space-y-1 text-sm">
                  <span className="text-xs font-medium text-muted-foreground">{t(($) => $.emails.existing_contact)}</span>
                  <select aria-label={t(($) => $.emails.linked_contact)} className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={associationDraft.contactId} onChange={(event) => setAssociationDraft({ ...associationDraft, contactId: event.target.value })}>
                    <option value="">{t(($) => $.emails.create_new_contact)}</option>
                    {draftContacts.map((contact) => <option key={contact.id} value={contact.id}>{contact.name}</option>)}
                  </select>
                </label>
              )}
              {associationDraft.accountId && !associationDraft.contactId && (
                <div className="grid gap-3 rounded-lg border bg-muted/20 p-3 sm:grid-cols-2">
                  <div className="sm:col-span-2 text-xs font-medium text-muted-foreground">{t(($) => $.emails.new_contact_title)}</div>
                  <Input aria-label={t(($) => $.contacts.name)} value={associationDraft.contactName} onChange={(event) => setAssociationDraft({ ...associationDraft, contactName: event.target.value })} placeholder={t(($) => $.contacts.name)} />
                  <Input aria-label={t(($) => $.contacts.email)} value={associationDraft.contactEmail} onChange={(event) => setAssociationDraft({ ...associationDraft, contactEmail: event.target.value })} placeholder={t(($) => $.contacts.email)} />
                </div>
              )}
            </div>
          )}
          {updateAssociation.isError && <p className="text-xs text-destructive">{t(($) => $.emails.association_error)}</p>}
          <DialogFooter>
            <Button variant="outline" onClick={() => setAssociationDraft(null)}>{t(($) => $.actions.cancel)}</Button>
            <Button disabled={!associationDraft?.accountId || updateAssociation.isPending} onClick={() => updateAssociation.mutate()}>{t(($) => $.emails.save_association)}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={emailLinkDraft !== null} onOpenChange={(open) => !open && setEmailLinkDraft(null)}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>{t(($) => $.emails.link_project_issue)}</DialogTitle>
            <DialogDescription>{t(($) => $.emails.email_link_help)}</DialogDescription>
          </DialogHeader>
          {emailLinkDraft && (
            <div className="space-y-4">
              <label className="space-y-1 text-sm">
                <span className="text-xs font-medium text-muted-foreground">{t(($) => $.emails.related_project)}</span>
                <select aria-label={t(($) => $.emails.related_project)} className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={emailLinkDraft.projectId} onChange={(event) => setEmailLinkDraft({ projectId: event.target.value, issueIds: [] })}>
                  {projects.map((project: Project) => <option key={project.id} value={project.id}>{project.title}</option>)}
                </select>
                <p className="text-xs text-muted-foreground">{t(($) => $.emails.default_project_hint, { title: defaultProjectTitle })}</p>
              </label>
              <label className="space-y-1 text-sm">
                <span className="text-xs font-medium text-muted-foreground">{t(($) => $.emails.related_issue)}</span>
                <div className="max-h-48 space-y-2 overflow-auto rounded-md border bg-background p-2">
                  {issues.map((issue: Issue) => {
                    const checked = emailLinkDraft.issueIds.includes(issue.id);
                    return (
                      <label key={issue.id} className="flex items-center gap-2 rounded-md px-2 py-1 text-sm hover:bg-muted">
                        <input aria-label={`${t(($) => $.emails.related_issue)} ${issue.identifier}`} type="checkbox" checked={checked} onChange={() => setEmailLinkDraft({ ...emailLinkDraft, issueIds: checked ? emailLinkDraft.issueIds.filter((id) => id !== issue.id) : [...emailLinkDraft.issueIds, issue.id] })} />
                        <span>{issue.identifier} · {issue.title} · {issue.status.replace(/_/g, " ")}</span>
                      </label>
                    );
                  })}
                  {!issues.length && <div className="px-2 py-1 text-xs text-muted-foreground">{t(($) => $.emails.no_issue_link)}</div>}
                </div>
              </label>
              <Button variant="outline" type="button" onClick={openCreateFollowUpIssue}>{t(($) => $.emails.create_follow_up_issue)}</Button>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setEmailLinkDraft(null)}>{t(($) => $.actions.cancel)}</Button>
            <Button disabled={!emailLinkDraft?.projectId || updateEmailLinks.isPending} onClick={() => updateEmailLinks.mutate()}>{t(($) => $.emails.save_email_link)}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={composeRecipientPickerOpen} onOpenChange={setComposeRecipientPickerOpen}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>选择收件人</DialogTitle>
            <DialogDescription>检索客户和联系人，选中后填入收件人。</DialogDescription>
          </DialogHeader>
          {composeDraft && (
            <div className="space-y-3">
              <Input placeholder="Search customer" value={composeAccountSearch} onChange={(event) => setComposeAccountSearch(event.target.value)} />
              <div className="max-h-80 overflow-y-auto rounded-md border bg-background">
                {filteredComposeAccounts.map((account) => (
                  <button key={account.id} type="button" className={`block w-full border-b px-3 py-2 text-left text-sm hover:bg-muted ${composeDraft.accountId === account.id ? "bg-muted" : ""}`} onClick={() => setComposeDraft({ ...composeDraft, accountId: account.id, contactId: "", to: "" })}>
                    <div className="font-medium">{account.name}</div>
                    <div className="text-xs text-muted-foreground">{[account.website, account.country_name || account.country, account.industry].filter(Boolean).join(" · ")}</div>
                  </button>
                ))}
                {!filteredComposeAccounts.length && <div className="p-4 text-sm text-muted-foreground">No customer found.</div>}
              </div>
              {composeDraft.accountId && (
                <div className="rounded-md border bg-muted/20 p-3">
                  <div className="mb-2 text-xs font-medium text-muted-foreground">Contact</div>
                  <div className="max-h-48 space-y-2 overflow-y-auto">
                    {composeAccountContacts.map((contact: any) => (
                      <button key={contact.id} type="button" className="block w-full rounded border bg-background px-3 py-2 text-left text-sm hover:bg-muted" onClick={() => { setComposeDraft({ ...composeDraft, contactId: contact.id, to: contact.email ?? composeDraft.to }); setComposeRecipientPickerOpen(false); }}>
                        <div className="font-medium">{contact.name}</div>
                        <div className="text-xs text-muted-foreground">{contact.email || "No email"}</div>
                      </button>
                    ))}
                    {!composeAccountContacts.length && <div className="text-xs text-muted-foreground">No contacts. Type recipient manually.</div>}
                  </div>
                </div>
              )}
              <Input aria-label={emailCopy.to} placeholder="Manual recipient email" value={composeDraft.to} onChange={(event) => setComposeDraft({ ...composeDraft, to: event.target.value })} />
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setComposeRecipientPickerOpen(false)}>取消</Button>
            <Button disabled={!composeDraft?.to.trim()} onClick={() => setComposeRecipientPickerOpen(false)}>确定</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={settingsOpen} onOpenChange={setSettingsOpen}>
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{emailCopy.mailboxSettingsTitle}</DialogTitle>
            <DialogDescription>{emailCopy.mailboxSettingsHelp}</DialogDescription>
          </DialogHeader>
          <div className="rounded-lg border bg-muted/30 p-4 text-sm">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="font-medium">{emailCopy.providerLabel}</div>
                <p className="mt-1 text-xs text-muted-foreground">{emailCopy.providerHelp}</p>
              </div>
            </div>
            <div className="mt-3 grid gap-2 text-xs text-muted-foreground sm:grid-cols-3">
              <div className="rounded-md border bg-background px-3 py-2">{emailCopy.step1}</div>
              <div className="rounded-md border bg-background px-3 py-2">{emailCopy.step2}</div>
              <div className="rounded-md border bg-background px-3 py-2">{emailCopy.step3}</div>
            </div>
          </div>
          <div className="grid gap-3 rounded-lg border bg-background p-3 text-xs sm:grid-cols-2">
            <DetailRow label="Account" value={mailboxes[0]?.email} />
            <DetailRow label="Transport" value="imap_smtp" />
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <select
              aria-label="CRM mailbox record"
              className="h-9 rounded-md border bg-background px-3 text-sm sm:col-span-2"
              value={mailboxDraft.id ?? "new"}
              onChange={(event) => setMailboxDraft(event.target.value === "new" ? emptyMailboxDraft : mailboxToDraft(mailboxes.find((mailbox) => mailbox.id === event.target.value)))}
            >
              <option value="new">New CRM mailbox</option>
              {mailboxes.map((mailbox) => <option key={mailbox.id} value={mailbox.id}>{mailbox.label} · {mailbox.email}</option>)}
            </select>
            <Input aria-label="Mailbox display name" placeholder="Mailbox display name" value={mailboxDraft.label} onChange={(event) => setMailboxDraft((draft) => ({ ...draft, label: event.target.value }))} />
            <Input aria-label={t(($) => $.emails.email_address)} placeholder="sales@example.com" value={mailboxDraft.email} onChange={(event) => setMailboxDraft((draft) => { const email = event.target.value; const preset = inferMailboxPreset(email); return { ...draft, ...(preset && !draft.id ? preset : {}), email, username: draft.username || email, smtp_username: draft.smtp_username || email }; })} />
            <Input aria-label="IMAP host" placeholder="IMAP host" value={mailboxDraft.host} onChange={(event) => setMailboxDraft((draft) => ({ ...draft, host: event.target.value }))} />
            <Input aria-label="IMAP port" placeholder="993" value={mailboxDraft.port} onChange={(event) => setMailboxDraft((draft) => ({ ...draft, port: event.target.value }))} />
            <select aria-label={t(($) => $.emails.tls_mode)} className="h-9 rounded-md border bg-background px-3 text-sm" value={mailboxDraft.tls_mode} onChange={(event) => setMailboxDraft((draft) => ({ ...draft, tls_mode: event.target.value as MailboxDraft["tls_mode"] }))}>
              <option value="ssl">{t(($) => $.emails.tls_ssl)}</option>
              <option value="starttls">{t(($) => $.emails.tls_starttls)}</option>
              <option value="none">{t(($) => $.emails.tls_none)}</option>
            </select>
            <Input aria-label={t(($) => $.emails.username)} placeholder={t(($) => $.emails.username)} value={mailboxDraft.username} onChange={(event) => setMailboxDraft((draft) => ({ ...draft, username: event.target.value }))} />
            <Input className="sm:col-span-2" aria-label={emailCopy.secretRef} placeholder={emailCopy.secretRefHelp} value={mailboxDraft.secret_ref} onChange={(event) => setMailboxDraft((draft) => ({ ...draft, secret_ref: event.target.value }))} />
            <label className="space-y-1 text-sm sm:col-span-2">
              <span className="text-xs font-medium text-muted-foreground">Bind mailbox to member or AI agent</span>
              <select className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={`${mailboxDraft.owner_type}:${mailboxDraft.owner_id}`} onChange={(event) => { const [owner_type, owner_id] = event.target.value.split(":"); setMailboxDraft((draft) => ({ ...draft, owner_type: owner_type || "", owner_id: owner_id || "" })); }}>
                <option value=":">Unassigned</option>
                {members.map((member: any) => <option key={`user-${member.id}`} value={`user:${member.user_id ?? member.id}`}>Member · {member.user?.name ?? member.user?.email ?? member.email ?? member.id}</option>)}
                {agents.map((agent: any) => <option key={`agent-${agent.id}`} value={`agent:${agent.id}`}>AI agent · {agent.name}</option>)}
              </select>
            </label>
            <label className="space-y-1 text-sm">
              <span className="text-xs font-medium text-muted-foreground">Import range</span>
              <select className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={importRangeDays} onChange={(event) => setImportRangeDays(Number(event.target.value))}>
                <option value={7}>Recent 7 days</option>
                <option value={30}>Recent 30 days</option>
                <option value={90}>Recent 90 days</option>
                <option value={365}>Recent 1 year</option>
              </select>
            </label>
            <label className="col-span-2 flex items-center gap-3 rounded-md border bg-muted/20 px-4 py-3 text-sm">
              <input
                type="checkbox"
                className="size-4"
                checked={mailboxDraft.sync_enabled}
                onChange={(event) => setMailboxDraft((draft) => ({ ...draft, sync_enabled: event.target.checked }))}
              />
              <div>
                <div className="font-medium">Auto sync</div>
                <div className="text-xs text-muted-foreground">Automatically sync new emails via cron</div>
              </div>
              {mailboxDraft.id && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="ml-auto"
                  disabled={saveMailbox.isPending}
                  onClick={() => api.toggleCRMIMAPSyncCron(wsId, mailboxDraft.id!, mailboxDraft.sync_enabled).then(() => queryClient.invalidateQueries({ queryKey: ["crm", wsId, "imap-settings"] }))}
                >
                  <RefreshCw className="mr-1 size-3" />Apply
                </Button>
              )}
            </label>
            <Input aria-label="SMTP host" placeholder="SMTP host" value={mailboxDraft.smtp_host} onChange={(event) => setMailboxDraft((draft) => ({ ...draft, smtp_host: event.target.value }))} />
            <Input aria-label="SMTP port" placeholder="465" value={mailboxDraft.smtp_port} onChange={(event) => setMailboxDraft((draft) => ({ ...draft, smtp_port: event.target.value }))} />
            <select aria-label="SMTP TLS mode" className="h-9 rounded-md border bg-background px-3 text-sm" value={mailboxDraft.smtp_tls_mode} onChange={(event) => setMailboxDraft((draft) => ({ ...draft, smtp_tls_mode: event.target.value }))}>
              <option value="ssl">SMTP SSL</option>
              <option value="starttls">SMTP STARTTLS</option>
            </select>
            <Input aria-label={emailCopy.smtpUsername} placeholder={emailCopy.smtpUsername} value={mailboxDraft.smtp_username} onChange={(event) => setMailboxDraft((draft) => ({ ...draft, smtp_username: event.target.value }))} />
            <Input className="sm:col-span-2" aria-label={emailCopy.smtpPassword} placeholder={emailCopy.smtpPassword} value={mailboxDraft.smtp_secret} onChange={(event) => setMailboxDraft((draft) => ({ ...draft, smtp_secret: event.target.value }))} />
          </div>
          <p className="rounded-md border bg-muted/30 p-3 text-xs text-muted-foreground">{emailCopy.transportNote}</p>
          {mailboxStatus ? <p className="rounded-md border bg-muted/20 p-3 text-xs text-muted-foreground">{mailboxStatus}</p> : null}
          {previewMessages.length > 0 ? (
            <div className="max-h-80 space-y-2 overflow-y-auto rounded-md border bg-muted/20 p-3">
              <div className="flex items-center justify-between text-xs text-muted-foreground">
                <span>{previewMessages.length} live IMAP messages · selected by default</span>
              </div>
              {previewMessages.map((message) => {
                const checked = selectedPreviewUIDs.includes(message.uid);
                return (
                  <label key={message.uid} className="flex gap-2 rounded border bg-background p-2 text-xs">
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={(event) => setSelectedPreviewUIDs((uids) => event.target.checked ? [...uids, message.uid] : uids.filter((uid) => uid !== message.uid))}
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium">{message.subject || "(no subject)"}</span>
                      <span className="block truncate text-muted-foreground">{message.from_name || message.from_email || "unknown"} · {messageTime(message.received_at)}</span>
                      <span className="mt-1 block line-clamp-2 text-muted-foreground">{message.snippet}</span>
                    </span>
                  </label>
                );
              })}
            </div>
          ) : null}
          <DialogFooter>
            {mailboxDraft.id ? <Button variant="destructive" disabled={deleteMailbox.isPending} onClick={() => { if (mailboxDraft.id) deleteMailbox.mutate(mailboxDraft.id, { onSuccess: () => setSettingsOpen(false) }); }}>Delete mailbox</Button> : null}
            <Button variant="outline" onClick={() => { setSettingsOpen(false); setMailboxStatus(null); }}>{t(($) => $.actions.cancel)}</Button>
            <Button variant="outline" disabled={testMailbox.isPending || saveMailbox.isPending || !mailboxDraft.label || !mailboxDraft.email || !mailboxDraft.host} onClick={() => testMailbox.mutate()}>{emailCopy.checkProvider}</Button>
            <Button disabled={saveMailbox.isPending || previewMailbox.isPending || importPreviewMessages.isPending || !mailboxDraft.label || !mailboxDraft.email} onClick={() => void saveAndImportMailbox()}>{emailCopy.saveAndImport}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <DiagnosticsDialog wsId={wsId} open={diagnosticsOpen} onOpenChange={setDiagnosticsOpen} />
    </div>
  );
}
