"use client";

import { useMemo, useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Archive, ArrowRight, Building2, Inbox, Link2, Mail, MailOpen, Search, Send, Settings, Star, UserRound } from "lucide-react";
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

type ComposeDraft = { mailboxId: string; accountId: string; contactId: string; to: string; cc: string; bcc: string; subject: string; body: string };

type MailboxDraft = { id?: string | null; label: string; email: string; host: string; port: string; tls_mode: "ssl" | "starttls" | "none"; username: string; secret_ref: string; secret: string; sync_enabled: boolean; owner_type: string; owner_id: string; smtp_host: string; smtp_port: string; smtp_tls_mode: string; smtp_username: string; smtp_secret_ref: string; smtp_secret: string };

const emptyMailboxDraft: MailboxDraft = { label: "", email: "", host: "", port: "993", tls_mode: "ssl", username: "", secret_ref: "", secret: "", sync_enabled: false, owner_type: "", owner_id: "", smtp_host: "", smtp_port: "465", smtp_tls_mode: "ssl", smtp_username: "", smtp_secret_ref: "", smtp_secret: "" };

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

function sanitizeEmailHtml(html?: string | null) {
  if (!html) return "";
  return html
    .replace(/<script[\s\S]*?<\/script>/gi, "")
    .replace(/<style[\s\S]*?<\/style>/gi, "")
    .replace(/\son\w+="[^"]*"/gi, "")
    .replace(/\son\w+='[^']*'/gi, "")
    .replace(/javascript:/gi, "");
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
    htmlBody: "HTML 正文",
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
    htmlBody: "HTML body",
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
  const [activeFolder, setActiveFolder] = useState<"inbox" | "sent" | "drafts" | "archived" | "starred" | "unlinked">("inbox");
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
  const openModal = useModalStore((state) => state.open);
  const setIssueDraft = useIssueDraftStore((state) => state.setDraft);
  const clearIssueDraft = useIssueDraftStore((state) => state.clearDraft);
  const { data: threads = [], isLoading } = useQuery(crmEmailThreadListOptions(wsId));
  const { data: accounts = [] } = useQuery(crmAccountListOptions(wsId, { sort: "name" }));
  const { data: members = [] } = useQuery({ queryKey: ["workspace", wsId, "members", "crm-mailbox"], queryFn: () => api.listMembers(wsId), enabled: Boolean(wsId) });
  const { data: agents = [] } = useQuery({ queryKey: ["agents", wsId, "crm-mailbox"], queryFn: () => api.listAgents({ workspace_id: wsId }), enabled: Boolean(wsId) });
  const { data: draftsData } = useQuery({ queryKey: ["crm", wsId, "email-drafts"], queryFn: () => api.listCRMEmailDrafts(), enabled: Boolean(wsId) });
  const { data: mailboxData } = useQuery({
    queryKey: ["crm", wsId, "imap-settings"],
    queryFn: () => api.listCRMIMAPSettings(),
    enabled: Boolean(wsId),
  });
  const { data: syncRunsData } = useQuery({
    queryKey: ["crm", wsId, "imap-sync-runs"],
    queryFn: () => api.listCRMIMAPSyncRuns(),
    enabled: Boolean(wsId),
  });
  const mailboxes = mailboxData?.settings ?? [];
  const selectedMailbox = mailboxes.find((mailbox) => mailbox.id === selectedMailboxId) ?? mailboxes[0] ?? null;
  const emailDrafts = draftsData?.drafts ?? [];
  const syncRuns = syncRunsData?.runs ?? [];

  const folderThreads = useMemo(() => {
    return threads.filter((thread) => {
      if (activeFolder === "sent") return thread.direction === "outbound";
      if (activeFolder === "archived") return thread.status === "archived";
      if (activeFolder === "starred") return false;
      if (activeFolder === "unlinked") return !thread.account_id;
      return thread.status !== "archived" && thread.direction !== "outbound";
    });
  }, [activeFolder, threads]);

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
    inbox: threads.filter((thread) => thread.status !== "archived" && thread.direction !== "outbound").length,
    sent: threads.filter((thread) => thread.direction === "outbound").length,
    drafts: emailDrafts.filter((draft: any) => draft.status !== "sent" && draft.status !== "discarded").length,
    archived: threads.filter((thread) => thread.status === "archived").length,
    starred: 0,
    unlinked: threads.filter((thread) => !thread.account_id).length,
  }), [threads, emailDrafts]);

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
      sync_enabled: false,
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
      queryClient.invalidateQueries({ queryKey: ["crm", wsId, "email-drafts"] });
      queryClient.invalidateQueries({ queryKey: crmKeys.emailThreads(wsId) });
    },
    onError: (error) => {
      setMailboxStatus(`SMTP send failed: ${mutationErrorMessage(error, "unknown error")}`);
    },
  });

  const saveEmailDraft = useMutation({
    mutationFn: async () => {
      if (!composeDraft) throw new Error(emailCopy.noDraft);
      const mailboxId = composeDraft.mailboxId || selectedMailbox?.id || mailboxes[0]?.id;
      if (!mailboxId) throw new Error(emailCopy.createMailboxFirst);
      return api.createCRMEmailDraft({
        mailbox_id: mailboxId,
        thread_id: selectedThread?.id ?? null,
        account_id: composeDraft.accountId || null,
        contact_id: composeDraft.contactId || null,
        to_emails: composeDraft.to.split(/[;,\n]/).map((value) => value.trim()).filter(Boolean),
        cc_emails: composeDraft.cc.split(/[;,\n]/).map((value) => value.trim()).filter(Boolean),
        bcc_emails: composeDraft.bcc.split(/[;,\n]/).map((value) => value.trim()).filter(Boolean),
        subject: composeDraft.subject.trim(),
        body_text: composeDraft.body,
      });
    },
    onSuccess: () => {
      setComposeDraft(null);
      setMailboxStatus(emailCopy.draftSaved);
      queryClient.invalidateQueries({ queryKey: ["crm", wsId, "email-drafts"] });
    },
  });

  const selectedThread = useMemo<CRMEmailThread | null>(() => {
    const found = threads.find((thread) => thread.id === selectedThreadId) ?? filteredThreads[0] ?? null;
    return found;
  }, [filteredThreads, selectedThreadId, threads]);

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
  const selectedIssueIds = selectedThread?.issue_ids?.length ? selectedThread.issue_ids : selectedThread?.issue_id ? [selectedThread.issue_id] : [];
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
    const subjectBase = selectedThread?.subject ?? "";
    const subject = mode === "forward"
      ? (subjectBase.toLowerCase().startsWith("fwd:") ? subjectBase : `Fwd: ${subjectBase}`)
      : subjectBase
        ? (subjectBase.toLowerCase().startsWith("re:") ? subjectBase : `Re: ${subjectBase}`)
        : "";
    const replyAll = mode === "reply-all";
    setComposeDraft({
      mailboxId: selectedMailbox?.id ?? mailboxes[0]?.id ?? "",
      accountId: selectedThread?.account_id ?? "",
      contactId: selectedThread?.contact_id ?? "",
      to: mode === "new" || mode === "forward" ? "" : inbound?.from_email ?? "",
      cc: replyAll ? (inbound?.cc_emails ?? []).join(", ") : "",
      bcc: "",
      subject,
      body: mode === "forward" ? "\n\n---- Forwarded message ----\n" : "",
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
    setEmailLinkDraft({ projectId, issueIds: selectedThread.issue_ids?.length ? selectedThread.issue_ids : selectedThread.issue_id ? [selectedThread.issue_id] : [] });
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
          <Button variant="outline" size="sm" onClick={() => setSettingsOpen(true)}>
            <Settings className="mr-1 size-3" />
            {t(($) => $.emails.mailbox_settings)}
          </Button>
        </div>
      </PageHeader>

      <div className="grid min-h-0 flex-1 grid-cols-1 gap-0 lg:grid-cols-[220px_360px_minmax(0,1fr)]">
        <aside className="flex min-h-0 flex-col border-r bg-card/80 p-3">
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
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <Badge variant={selectedMailbox?.last_test_status === "ok" ? "default" : "outline"}>{emailCopy.nativeProvider}</Badge>
              {selectedMailbox?.last_test_status ? <Badge variant="secondary">{selectedMailbox.last_test_status}</Badge> : null}
              {syncRuns.some((run: any) => run.status === "running" && (!selectedMailbox || run.mailbox_id === selectedMailbox.id)) ? <Badge variant="secondary">{emailCopy.syncing}</Badge> : null}
            </div>
            <div className="mt-2 text-xs text-muted-foreground">{selectedMailbox?.last_test_message || t(($) => $.emails.imap_not_connected)}</div>
            <details className="mt-3 rounded-md border bg-muted/20 p-2 text-xs">
              <summary className="cursor-pointer font-medium text-muted-foreground">Sync progress / history</summary>
              <div className="mt-2 space-y-1 text-muted-foreground">
                {syncRuns.length === 0 ? <p>No import runs yet.</p> : syncRuns.slice(0, 5).map((run: any) => (
                  <div key={run.id} className="rounded bg-background px-2 py-1">
                    <div className="truncate">{run.mailbox_email || run.folder || "INBOX"} · {run.status}</div>
                    <div className="tabular-nums">fetched {run.fetched_count ?? 0} / imported {run.imported_count ?? 0} / skipped {run.skipped_count ?? 0}</div>
                  </div>
                ))}
              </div>
            </details>
          </div>
          <nav className="space-y-1" aria-label={t(($) => $.emails.folder_nav)}>
            {([
              ["inbox", Inbox, t(($) => $.emails.folder_inbox)],
              ["sent", MailOpen, t(($) => $.emails.folder_sent)],
              ["drafts", Send, emailCopy.drafts],
              ["archived", Archive, t(($) => $.emails.folder_archived)],
              ["starred", Star, t(($) => $.emails.folder_starred)],
              ["unlinked", Link2, t(($) => $.emails.folder_unlinked)],
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
          <Button className="mt-auto" variant="outline" onClick={() => setSettingsOpen(true)}>{t(($) => $.emails.add_mailbox)}</Button>
        </aside>

        <aside className="flex min-h-0 flex-col border-r bg-background">
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
              {emailDrafts.length === 0 ? <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">{emailCopy.noDrafts}</div> : emailDrafts.map((draft: any) => (
                <div key={draft.id} className="mb-2 rounded-lg border bg-card p-3 text-sm">
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <div className="truncate font-medium">{draft.subject || emailCopy.noSubject}</div>
                      <div className="truncate text-xs text-muted-foreground">{emailCopy.to}: {(draft.to_emails ?? []).join(", ") || "—"}</div>
                    </div>
                    <Badge variant="outline">{draft.status}</Badge>
                  </div>
                  <p className="mt-2 line-clamp-3 text-xs text-muted-foreground">{draft.body_text}</p>
                  <Button className="mt-3" size="sm" variant="outline" disabled={draft.status === "sent" || sendDraft.isPending} onClick={() => sendDraft.mutate(draft.id)}>{emailCopy.send}</Button>
                </div>
              ))}
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
                return (
                  <button key={thread.id} type="button" className={`block w-full border-b px-4 py-3 text-left text-sm hover:bg-muted/60 ${active ? "bg-muted" : ""}`} onClick={() => setSelectedThreadId(thread.id)}>
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0 flex-1 truncate font-medium">{thread.subject}</div>
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
          {!selectedThread ? (
            <div className="p-10 text-center text-sm text-muted-foreground">{t(($) => $.emails.select_thread)}</div>
          ) : (
            <div className="flex h-full min-h-0 flex-col">
              <div className="border-b bg-background p-5">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <h2 className="truncate text-base font-semibold">{selectedThread.subject}</h2>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {[selectedThread.mailbox, selectedThread.direction, selectedThread.status, messageTime(selectedThread.last_message_at)].filter(Boolean).join(" · ")}
                    </p>
                  </div>
                  <Button variant={selectedAccount ? "outline" : "default"} size="sm" onClick={() => openAssociationDialog()}>
                    <Link2 className="mr-1 size-3" />
                    {selectedAccount ? t(($) => $.emails.change_association) : t(($) => $.emails.link_customer_contact)}
                  </Button>
                </div>
                <div className="mt-4 flex flex-wrap items-center gap-2 border-t pt-3">
                  <Button variant="outline" size="sm"><MailOpen className="mr-1 size-3" />{t(($) => $.emails.mark_read)}</Button>
                  <Button variant="outline" size="sm"><Archive className="mr-1 size-3" />{t(($) => $.emails.archive)}</Button>
                  <Button variant="outline" size="sm"><Star className="mr-1 size-3" />{t(($) => $.emails.star)}</Button>
                  <Button variant="outline" size="sm" disabled={!mailboxes.length} onClick={() => openComposeDraft("reply")}><Send className="mr-1 size-3" />{emailCopy.reply}</Button>
                  <Button variant="outline" size="sm" disabled={!mailboxes.length} onClick={() => openComposeDraft("reply-all")}>{emailCopy.replyAll}</Button>
                  <Button variant="outline" size="sm" disabled={!mailboxes.length} onClick={() => openComposeDraft("forward")}>{emailCopy.forward}</Button>
                  <Button variant="outline" size="sm" disabled={!selectedAccount} onClick={openEmailLinkDialog}><Link2 className="mr-1 size-3" />{t(($) => $.emails.link_project_issue)}</Button>
                </div>
                <div className="mt-3 flex flex-wrap items-center gap-2">
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
              <div className="min-h-0 flex-1 overflow-y-auto bg-muted/20 p-5">
                {messagesLoading ? (
                  <div className="space-y-3">
                    <Skeleton className="h-24 w-full" />
                    <Skeleton className="h-24 w-full" />
                  </div>
                ) : messages.length === 0 ? (
                  <div className="rounded-lg border border-dashed bg-background p-8 text-center text-sm text-muted-foreground">{t(($) => $.emails.no_messages)}</div>
                ) : (
                  <div className="space-y-3">
                    {messages.map((message) => (
                      <article key={message.id} className="rounded-lg border bg-background p-4 text-sm shadow-xs">
                        <div className="flex flex-wrap justify-between gap-2">
                          <div className="font-medium">{message.from_name || message.from_email || t(($) => $.common.not_available)}</div>
                          <div className="text-xs text-muted-foreground">{messageTime(message.sent_at || message.received_at)}</div>
                        </div>
                        <div className="mt-3 grid gap-2 text-xs text-muted-foreground sm:grid-cols-2">
                          <DetailRow label={emailCopy.from} value={[message.from_name, message.from_email].filter(Boolean).join(" <") + (message.from_name && message.from_email ? ">" : "")} />
                          <DetailRow label={emailCopy.to} value={message.to_emails.join(", ")} />
                          <DetailRow label={emailCopy.cc} value={message.cc_emails.join(", ")} />
                          <DetailRow label={emailCopy.date} value={messageTime(message.sent_at || message.received_at)} />
                        </div>
                        <div className="mt-4 rounded-md border bg-muted/20 p-3">
                          <div className="mb-2 text-xs font-medium text-muted-foreground">{message.body_html ? emailCopy.htmlBody : emailCopy.textBody}</div>
                          {message.body_html ? (
                            <div className="leading-6 text-foreground/80" dangerouslySetInnerHTML={{ __html: safeEmailHTML(message.body_html) }} />
                          ) : (
                            <div className="whitespace-pre-wrap leading-6 text-foreground/80">{message.body_text || message.snippet || t(($) => $.emails.no_body)}</div>
                          )}
                        </div>
                        <div className="mt-3 rounded-md border bg-muted/20 p-3">
                          <div className="text-xs font-medium text-muted-foreground">{emailCopy.attachments}</div>
                          {message.attachments?.length ? (
                            <div className="mt-2 space-y-2">
                              {message.attachments.map((attachment, index) => (
                                <div key={`${message.id}-attachment-${index}`} className="rounded border bg-background px-3 py-2 text-xs">
                                  <div className="font-medium">{attachment.filename || attachment.content_id || emailCopy.attachments}</div>
                                  <div className="mt-1 text-muted-foreground">{[attachment.content_type, attachment.disposition, `${attachment.size_bytes} ${emailCopy.bytes}`].filter(Boolean).join(" · ")}</div>
                                </div>
                              ))}
                            </div>
                          ) : <div className="mt-2 text-xs text-muted-foreground">{emailCopy.noAttachments}</div>}
                        </div>
                        <details className="mt-3 rounded-md border bg-muted/20 p-3 text-xs">
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


      <Dialog open={composeDraft !== null} onOpenChange={(open) => !open && setComposeDraft(null)}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{emailCopy.composeTitle}</DialogTitle>
            <DialogDescription>{emailCopy.composeHelp}</DialogDescription>
          </DialogHeader>
          {composeDraft && (
            <div className="space-y-3">
              <label className="space-y-1 text-sm">
                <span className="text-xs font-medium text-muted-foreground">{emailCopy.mailbox}</span>
                <select className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={composeDraft.mailboxId} onChange={(event) => setComposeDraft({ ...composeDraft, mailboxId: event.target.value })}>
                  {mailboxes.map((mailbox) => <option key={mailbox.id} value={mailbox.id}>{mailbox.label} · {mailbox.email}</option>)}
                </select>
              </label>
              <div className="grid gap-3 sm:grid-cols-2">
                <label className="space-y-1 text-sm">
                  <span className="text-xs font-medium text-muted-foreground">Customer</span>
                  <select className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={composeDraft.accountId} onChange={(event) => setComposeDraft({ ...composeDraft, accountId: event.target.value, contactId: "", to: "" })}>
                    <option value="">No customer</option>
                    {accounts.map((account: any) => <option key={account.id} value={account.id}>{account.name}</option>)}
                  </select>
                </label>
                <label className="space-y-1 text-sm">
                  <span className="text-xs font-medium text-muted-foreground">Contact</span>
                  <select className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={composeDraft.contactId} onChange={(event) => { const contact = composeAccountContacts.find((item: any) => item.id === event.target.value) as any; setComposeDraft({ ...composeDraft, contactId: event.target.value, to: contact?.email ?? composeDraft.to }); }} disabled={!composeDraft.accountId}>
                    <option value="">Manual recipient</option>
                    {composeAccountContacts.map((contact: any) => <option key={contact.id} value={contact.id}>{contact.name}{contact.email ? ` · ${contact.email}` : ""}</option>)}
                  </select>
                </label>
              </div>
              <Input aria-label={emailCopy.to} placeholder={emailCopy.to} value={composeDraft.to} onChange={(event) => setComposeDraft({ ...composeDraft, to: event.target.value })} />
              <div className="grid gap-3 sm:grid-cols-2">
                <Input aria-label={emailCopy.cc} placeholder={emailCopy.cc} value={composeDraft.cc} onChange={(event) => setComposeDraft({ ...composeDraft, cc: event.target.value })} />
                <Input aria-label={emailCopy.bcc} placeholder={emailCopy.bcc} value={composeDraft.bcc} onChange={(event) => setComposeDraft({ ...composeDraft, bcc: event.target.value })} />
              </div>
              <Input aria-label={emailCopy.subject} placeholder={emailCopy.subject} value={composeDraft.subject} onChange={(event) => setComposeDraft({ ...composeDraft, subject: event.target.value })} />
              <textarea
                aria-label={emailCopy.bodyLabel}
                className="min-h-48 w-full rounded-md border bg-background px-3 py-2 text-sm outline-none ring-offset-background placeholder:text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                placeholder={emailCopy.bodyPlaceholder}
                value={composeDraft.body}
                onChange={(event) => setComposeDraft({ ...composeDraft, body: event.target.value })}
              />
            </div>
          )}
          {saveEmailDraft.isError && <p className="text-xs text-destructive">{emailCopy.saveDraftError}</p>}
          <DialogFooter>
            <Button variant="outline" onClick={() => setComposeDraft(null)}>{emailCopy.cancel}</Button>
            <Button variant="secondary" disabled={!composeDraft?.to.trim() || !composeDraft?.subject.trim() || !composeDraft?.body.trim() || saveEmailDraft.isPending || sendDraft.isPending} onClick={async () => { const draft = await saveEmailDraft.mutateAsync(); sendDraft.mutate(draft.id); }}>{emailCopy.send}</Button>
            <Button disabled={!composeDraft?.to.trim() || !composeDraft?.subject.trim() || !composeDraft?.body.trim() || saveEmailDraft.isPending} onClick={() => saveEmailDraft.mutate()}>{emailCopy.saveDraft}</Button>
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
              <Badge variant="outline">{emailCopy.nativeProvider}</Badge>
            </div>
            <div className="mt-3 grid gap-2 text-xs text-muted-foreground sm:grid-cols-3">
              <div className="rounded-md border bg-background px-3 py-2">{emailCopy.step1}</div>
              <div className="rounded-md border bg-background px-3 py-2">{emailCopy.step2}</div>
              <div className="rounded-md border bg-background px-3 py-2">{emailCopy.step3}</div>
            </div>
          </div>
          <div className="grid gap-3 rounded-lg border bg-background p-3 text-xs sm:grid-cols-4">
            <DetailRow label="Account" value={mailboxes[0]?.email} />
            <DetailRow label="State" value={mailboxes[0]?.last_test_status ?? (mailboxes.length ? "configured" : "not configured")} />
            <DetailRow label="Sync runs" value={String(syncRuns.length)} />
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
            <Input aria-label={t(($) => $.emails.email_address)} placeholder="sales@example.com" value={mailboxDraft.email} onChange={(event) => setMailboxDraft((draft) => ({ ...draft, email: event.target.value }))} />
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
            <Button variant="outline" onClick={() => { setSettingsOpen(false); setMailboxStatus(null); }}>{t(($) => $.actions.cancel)}</Button>
            <Button variant="outline" disabled={testMailbox.isPending || saveMailbox.isPending || !mailboxDraft.label || !mailboxDraft.email || !mailboxDraft.host} onClick={() => testMailbox.mutate()}>{emailCopy.checkProvider}</Button>
            <Button disabled={saveMailbox.isPending || previewMailbox.isPending || importPreviewMessages.isPending || !mailboxDraft.label || !mailboxDraft.email} onClick={() => void saveAndImportMailbox()}>{emailCopy.saveAndImport}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
