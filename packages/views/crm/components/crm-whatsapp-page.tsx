"use client";
/* eslint-disable i18next/no-literal-string */

import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Building2, MessageCircle, Plus, PlusCircle, RefreshCw, Send, UserRound } from "lucide-react";
import { crmApi } from "@multica/core/crm/api";
import { crmAccountListOptions, crmContactListOptions, crmKeys, crmWhatsAppMessageListOptions, crmWhatsAppThreadListOptions } from "@multica/core/crm/queries";
import { useWorkspaceId } from "@multica/core/hooks";
import { useIssueDraftStore } from "@multica/core/issues";
import { useModalStore } from "@multica/core/modals";
import { Badge } from "@multica/ui/components/ui/badge";
import { Button } from "@multica/ui/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@multica/ui/components/ui/dialog";
import { Input } from "@multica/ui/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@multica/ui/components/ui/select";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { PageHeader } from "../../layout/page-header";

const NONE = "__none__";

export function CRMWhatsAppPage() {
  const wsId = useWorkspaceId();
  const queryClient = useQueryClient();
  const openModal = useModalStore((state) => state.open);
  const setIssueDraft = useIssueDraftStore((state) => state.setDraft);
  const clearIssueDraft = useIssueDraftStore((state) => state.clearDraft);

  const { data: threads = [], isLoading: threadsLoading } = useQuery(crmWhatsAppThreadListOptions(wsId));
  const { data: accounts = [] } = useQuery(crmAccountListOptions(wsId, { sort: "name" }));
  const [selectedThreadId, setSelectedThreadId] = useState<string>("");
  const selectedThread = useMemo(() => threads.find((thread) => thread.id === (selectedThreadId || threads[0]?.id)), [threads, selectedThreadId]);
  const { data: messages = [], isLoading: messagesLoading } = useQuery(crmWhatsAppMessageListOptions(wsId, selectedThread?.id ?? ""));
  const [replyText, setReplyText] = useState("");
  const [accountId, setAccountId] = useState("");
  const [contactId, setContactId] = useState("");
  const [createAccountOpen, setCreateAccountOpen] = useState(false);
  const [createContactOpen, setCreateContactOpen] = useState(false);
  const [accountForm, setAccountForm] = useState({ name: "", website: "", notes: "" });
  const [contactForm, setContactForm] = useState({ name: "", email: "", phone: "", whatsapp: "", jobTitle: "", notes: "" });
  const selectedAccount = useMemo(() => accounts.find((account) => account.id === accountId), [accounts, accountId]);
  const { data: contacts = [], isLoading: contactsLoading } = useQuery(crmContactListOptions(wsId, accountId));
  const selectedContact = useMemo(() => contacts.find((contact) => contact.id === contactId), [contacts, contactId]);

  const sendMutation = useMutation({
    mutationFn: () => crmApi.sendCRMWhatsAppMessage(selectedThread?.id ?? "", { body_text: replyText }),
    onSuccess: async () => {
      setReplyText("");
      await queryClient.invalidateQueries({ queryKey: crmKeys.whatsappMessages(wsId, selectedThread?.id ?? "") });
      await queryClient.invalidateQueries({ queryKey: crmKeys.whatsappThreads(wsId) });
    },
  });

  const associationMutation = useMutation({
    mutationFn: () => crmApi.updateCRMWhatsAppThreadAssociation(selectedThread?.id ?? "", { account_id: accountId || null, contact_id: contactId || null }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: crmKeys.whatsappThreads(wsId) });
    },
  });

  const createAccountMutation = useMutation({
    mutationFn: () => crmApi.createCRMAccount({
      name: accountForm.name.trim(),
      website: accountForm.website.trim() || null,
      notes: accountForm.notes.trim() || null,
      source: "whatsapp",
      status: "active",
      account_type: "prospect",
      rating: "unknown",
      priority: "medium",
    }),
    onSuccess: async (account) => {
      setAccountId(account.id);
      setContactId("");
      setAccountForm({ name: "", website: "", notes: "" });
      setCreateAccountOpen(false);
      await queryClient.invalidateQueries({ queryKey: crmKeys.accounts(wsId) });
    },
  });

  const createContactMutation = useMutation({
    mutationFn: () => crmApi.createCRMContact(accountId, {
      name: contactForm.name.trim(),
      email: contactForm.email.trim() || null,
      phone: contactForm.phone.trim() || null,
      mobile: contactForm.phone.trim() || null,
      whatsapp_id: contactForm.whatsapp.trim() || selectedThread?.phone_number || null,
      whatsapp: contactForm.whatsapp.trim() || selectedThread?.phone_number || null,
      job_title: contactForm.jobTitle.trim() || null,
      role_title: contactForm.jobTitle.trim() || null,
      notes: contactForm.notes.trim() || null,
      is_primary: contacts.length === 0,
    }),
    onSuccess: async (contact) => {
      setContactId(contact.id);
      setContactForm({ name: "", email: "", phone: "", whatsapp: "", jobTitle: "", notes: "" });
      setCreateContactOpen(false);
      await queryClient.invalidateQueries({ queryKey: crmKeys.contacts(wsId, accountId) });
    },
  });

  useEffect(() => {
    setAccountId(selectedThread?.account_id ?? "");
    setContactId(selectedThread?.contact_id ?? "");
    setReplyText("");
  }, [selectedThread?.id, selectedThread?.account_id, selectedThread?.contact_id]);

  useEffect(() => {
    if (contactId && !contacts.some((contact) => contact.id === contactId)) setContactId("");
  }, [accountId, contactId, contacts]);

  const syncMutation = useMutation({
    mutationFn: () => crmApi.syncCRMWhatsAppFromHermes(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: crmKeys.whatsappThreads(wsId) });
    },
  });

  const openCreateIssue = () => {
    if (!selectedThread) return;
    const latest = messages[0];
    clearIssueDraft();
    setIssueDraft({
      title: `WhatsApp follow-up: ${selectedThread.title || selectedThread.phone_number || selectedThread.external_chat_id}`,
      description: [
        `WhatsApp chat: ${selectedThread.title || selectedThread.external_chat_id}`,
        selectedThread.phone_number ? `Phone: ${selectedThread.phone_number}` : "",
        selectedAccount ? `Customer: ${selectedAccount.name}` : "",
        selectedContact ? `Contact: ${selectedContact.name}` : "",
        latest?.body_text ? `Latest message: ${latest.body_text}` : "",
      ].filter(Boolean).join("\n"),
      priority: "medium",
      status: "in_review",
    });
    openModal("create-issue", {
      onCreated: async () => {
        await queryClient.invalidateQueries({ queryKey: crmKeys.whatsappThreads(wsId) });
      },
    });
  };

  const formatThreadTime = (value?: string | null) => value ? new Date(value).toLocaleString() : "No messages";

  return (
    <div className="flex h-full min-h-0 flex-col bg-muted/10">
      <PageHeader className="justify-between px-5">
        <div className="flex items-center gap-2">
          <MessageCircle className="size-4 text-muted-foreground" />
          <h1 className="text-sm font-medium">CRM WhatsApp</h1>
          {!threadsLoading && <span className="text-xs text-muted-foreground tabular-nums">{threads.length}</span>}
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => syncMutation.mutate()} disabled={syncMutation.isPending}>
            <RefreshCw className="mr-2 size-4" />Sync Hermes
          </Button>
          <Button size="sm" onClick={openCreateIssue} disabled={!selectedThread}>
            <PlusCircle className="mr-2 size-4" />Create issue
          </Button>
        </div>
      </PageHeader>

      <div className="grid min-h-0 flex-1 grid-cols-[360px_minmax(0,1fr)] border-t bg-background">
        <aside className="min-h-0 overflow-y-auto border-r bg-muted/20 p-3">
          {threadsLoading ? <div className="space-y-2"><Skeleton className="h-20" /><Skeleton className="h-20" /></div> : threads.length === 0 ? (
            <div className="rounded-lg border bg-card p-6 text-sm text-muted-foreground">No WhatsApp threads yet. Click Sync Hermes.</div>
          ) : (
            <div className="space-y-2">
              {threads.map((thread) => {
                const active = thread.id === selectedThread?.id;
                return (
                  <button key={thread.id} type="button" onClick={() => setSelectedThreadId(thread.id)} className={`w-full rounded-lg border bg-card p-3 text-left shadow-sm transition hover:bg-accent/50 ${active ? "border-primary/40 bg-accent" : ""}`}>
                    <div className="flex items-center justify-between gap-2">
                      <div className="truncate text-sm font-medium">{thread.title || thread.phone_number || thread.external_chat_id}</div>
                      {thread.unread_count > 0 ? <Badge className="tabular-nums">{thread.unread_count}</Badge> : null}
                    </div>
                    <div className="mt-1 truncate text-xs text-muted-foreground">{thread.phone_number || thread.external_chat_id}</div>
                    <div className="mt-2 line-clamp-2 text-xs text-muted-foreground">{thread.last_message_text || "No preview"}</div>
                    <div className="mt-2 flex items-center justify-between text-[11px] text-muted-foreground">
                      <span>{formatThreadTime(thread.last_message_at)}</span>
                      {thread.account_id ? <Badge variant="outline">Linked</Badge> : null}
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </aside>

        <main className="flex min-h-0 flex-col overflow-hidden">
          {!selectedThread ? (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground"><MessageCircle className="mr-2 size-4" />Select WhatsApp thread</div>
          ) : (
            <>
              <section className="border-b bg-card px-5 py-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="truncate text-sm font-semibold">{selectedThread.title || selectedThread.external_chat_id}</div>
                    <div className="mt-1 text-xs text-muted-foreground">{selectedThread.phone_number || selectedThread.external_chat_id}</div>
                  </div>
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" onClick={() => { setAccountForm({ name: selectedThread.title || selectedThread.phone_number || "", website: "", notes: selectedThread.phone_number ? `WhatsApp: ${selectedThread.phone_number}` : "" }); setCreateAccountOpen(true); }}>
                      <Plus className="mr-1 size-4" />New customer
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => { setContactForm({ name: selectedThread.title || "", email: "", phone: selectedThread.phone_number || "", whatsapp: selectedThread.phone_number || "", jobTitle: "", notes: "" }); setCreateContactOpen(true); }} disabled={!accountId}>
                      <Plus className="mr-1 size-4" />New contact
                    </Button>
                  </div>
                </div>

                <div className="mt-4 grid gap-3 lg:grid-cols-[minmax(220px,1fr)_minmax(220px,1fr)_auto]">
                  <div className="space-y-1.5">
                    <div className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground"><Building2 className="size-3.5" />Customer</div>
                    <Select value={accountId || NONE} onValueChange={(value) => { const nextValue = value ?? NONE; setAccountId(nextValue === NONE ? "" : nextValue); setContactId(""); }}>
                      <SelectTrigger className="w-full bg-background"><SelectValue placeholder="Select customer" /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value={NONE}>Unlinked customer</SelectItem>
                        {accounts.map((account) => <SelectItem key={account.id} value={account.id}>{account.name}</SelectItem>)}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-1.5">
                    <div className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground"><UserRound className="size-3.5" />Contact</div>
                    <Select value={contactId || NONE} onValueChange={(value) => { const nextValue = value ?? NONE; setContactId(nextValue === NONE ? "" : nextValue); }} disabled={!accountId || contactsLoading}>
                      <SelectTrigger className="w-full bg-background"><SelectValue placeholder={accountId ? "Select contact" : "Select customer first"} /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value={NONE}>{contactsLoading ? "Loading contacts..." : "Unlinked contact"}</SelectItem>
                        {contacts.map((contact) => <SelectItem key={contact.id} value={contact.id}>{contact.name}{contact.email ? ` · ${contact.email}` : contact.phone ? ` · ${contact.phone}` : ""}</SelectItem>)}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="flex items-end">
                    <Button className="w-full lg:w-auto" onClick={() => associationMutation.mutate()} disabled={associationMutation.isPending || (!accountId && !!contactId)}>
                      Save link
                    </Button>
                  </div>
                </div>
                {associationMutation.isError ? <p className="mt-2 text-xs text-destructive">Failed to save association.</p> : null}
              </section>

              <div className="min-h-0 flex-1 overflow-y-auto p-5">
                {messagesLoading ? (
                  <div className="space-y-3"><Skeleton className="h-20" /><Skeleton className="h-20" /></div>
                ) : messages.length === 0 ? (
                  <div className="rounded-lg border bg-card p-6 text-sm text-muted-foreground">No messages stored for this thread.</div>
                ) : (
                  <div className="flex flex-col-reverse gap-3">
                    {messages.map((message) => (
                      <div key={message.id} className={`max-w-[75%] rounded-2xl border px-4 py-3 text-sm shadow-sm ${message.direction === "outbound" ? "ml-auto border-primary/20 bg-primary/10" : "bg-card"}`}>
                        <div className="mb-1 text-xs text-muted-foreground">{message.direction} · {message.sent_at || message.received_at || ""}</div>
                        <div className="whitespace-pre-wrap break-words">{message.body_text || "[no text]"}</div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="border-t bg-card p-4">
                <div className="flex gap-2">
                  <textarea className="min-h-20 flex-1 rounded-md border bg-background px-3 py-2 text-sm" placeholder="Reply on WhatsApp" value={replyText} onChange={(e) => setReplyText(e.target.value)} />
                  <Button className="self-end" onClick={() => sendMutation.mutate()} disabled={!replyText.trim() || sendMutation.isPending}>
                    <Send className="mr-2 size-4" />Send
                  </Button>
                </div>
              </div>
            </>
          )}
        </main>
      </div>

      <Dialog open={createAccountOpen} onOpenChange={setCreateAccountOpen}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>New customer</DialogTitle>
            <DialogDescription>Create customer from this WhatsApp conversation.</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 rounded-lg border bg-muted/20 p-4 sm:grid-cols-2">
            <Input value={accountForm.name} onChange={(e) => setAccountForm({ ...accountForm, name: e.target.value })} placeholder="Customer name" />
            <Input value={accountForm.website} onChange={(e) => setAccountForm({ ...accountForm, website: e.target.value })} placeholder="Website" />
            <textarea className="min-h-24 rounded-md border bg-background px-3 py-2 text-sm sm:col-span-2" value={accountForm.notes} onChange={(e) => setAccountForm({ ...accountForm, notes: e.target.value })} placeholder="Notes" />
          </div>
          {createAccountMutation.isError ? <p className="text-xs text-destructive">Failed to create customer.</p> : null}
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateAccountOpen(false)}>Cancel</Button>
            <Button disabled={!accountForm.name.trim() || createAccountMutation.isPending} onClick={() => createAccountMutation.mutate()}>Create customer</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={createContactOpen} onOpenChange={setCreateContactOpen}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>New contact</DialogTitle>
            <DialogDescription>{selectedAccount ? `Create contact under ${selectedAccount.name}.` : "Select customer first."}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 rounded-lg border bg-muted/20 p-4 sm:grid-cols-2">
            <Input value={contactForm.name} onChange={(e) => setContactForm({ ...contactForm, name: e.target.value })} placeholder="Contact name" />
            <Input value={contactForm.email} onChange={(e) => setContactForm({ ...contactForm, email: e.target.value })} placeholder="Email" />
            <Input value={contactForm.phone} onChange={(e) => setContactForm({ ...contactForm, phone: e.target.value })} placeholder="Phone" />
            <Input value={contactForm.whatsapp} onChange={(e) => setContactForm({ ...contactForm, whatsapp: e.target.value })} placeholder="WhatsApp" />
            <Input value={contactForm.jobTitle} onChange={(e) => setContactForm({ ...contactForm, jobTitle: e.target.value })} placeholder="Job title" />
            <textarea className="min-h-24 rounded-md border bg-background px-3 py-2 text-sm sm:col-span-2" value={contactForm.notes} onChange={(e) => setContactForm({ ...contactForm, notes: e.target.value })} placeholder="Notes" />
          </div>
          {createContactMutation.isError ? <p className="text-xs text-destructive">Failed to create contact.</p> : null}
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateContactOpen(false)}>Cancel</Button>
            <Button disabled={!accountId || !contactForm.name.trim() || createContactMutation.isPending} onClick={() => createContactMutation.mutate()}>Create contact</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
