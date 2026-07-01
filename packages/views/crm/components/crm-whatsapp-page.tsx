"use client";
/* eslint-disable i18next/no-literal-string */

import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { MessageCircle, RefreshCw, PlusCircle } from "lucide-react";
import { crmApi } from "@multica/core/crm/api";
import { crmKeys, crmWhatsAppMessageListOptions, crmWhatsAppThreadListOptions } from "@multica/core/crm/queries";
import { useWorkspaceId } from "@multica/core/hooks";
import { useIssueDraftStore } from "@multica/core/issues";
import { useModalStore } from "@multica/core/modals";
import { Badge } from "@multica/ui/components/ui/badge";
import { Button } from "@multica/ui/components/ui/button";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { PageHeader } from "../../layout/page-header";

export function CRMWhatsAppPage() {
  const wsId = useWorkspaceId();
  const queryClient = useQueryClient();
  const openModal = useModalStore((state) => state.open);
  const setIssueDraft = useIssueDraftStore((state) => state.setDraft);
  const clearIssueDraft = useIssueDraftStore((state) => state.clearDraft);
  const { data: threads = [], isLoading: threadsLoading } = useQuery(crmWhatsAppThreadListOptions(wsId));
  const [selectedThreadId, setSelectedThreadId] = useState<string>("");
  const selectedThread = useMemo(() => threads.find((thread) => thread.id === (selectedThreadId || threads[0]?.id)), [threads, selectedThreadId]);
  const { data: messages = [], isLoading: messagesLoading } = useQuery(crmWhatsAppMessageListOptions(wsId, selectedThread?.id ?? ""));
  const [replyText, setReplyText] = useState("");
  const [accountId, setAccountId] = useState("");
  const [contactId, setContactId] = useState("");

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

  useEffect(() => {
    setAccountId(selectedThread?.account_id ?? "");
    setContactId(selectedThread?.contact_id ?? "");
    setReplyText("");
  }, [selectedThread?.id, selectedThread?.account_id, selectedThread?.contact_id]);

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

  return (
    <div className="flex h-full min-h-0 flex-col">
      <PageHeader className="justify-between">
        <div>
          <div className="font-medium">CRM WhatsApp</div>
          <div className="text-xs text-muted-foreground">Hermes bridge customer communication history</div>
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
      <div className="grid min-h-0 flex-1 grid-cols-[340px_minmax(0,1fr)] border-t">
        <aside className="min-h-0 overflow-y-auto border-r bg-muted/20">
          {threadsLoading ? <div className="space-y-2 p-3"><Skeleton className="h-16" /><Skeleton className="h-16" /></div> : threads.length === 0 ? (
            <div className="p-6 text-sm text-muted-foreground">No WhatsApp threads yet. Click Sync Hermes.</div>
          ) : threads.map((thread) => {
            const active = thread.id === selectedThread?.id;
            return (
              <button key={thread.id} type="button" onClick={() => setSelectedThreadId(thread.id)} className={`w-full border-b p-3 text-left hover:bg-background ${active ? "bg-background" : ""}`}>
                <div className="flex items-center justify-between gap-2">
                  <div className="truncate font-medium">{thread.title || thread.phone_number || thread.external_chat_id}</div>
                  {thread.unread_count > 0 ? <Badge>{thread.unread_count}</Badge> : null}
                </div>
                <div className="mt-1 truncate text-xs text-muted-foreground">{thread.phone_number || thread.external_chat_id}</div>
                <div className="mt-2 line-clamp-2 text-xs text-muted-foreground">{thread.last_message_text}</div>
              </button>
            );
          })}
        </aside>
        <main className="flex min-h-0 flex-col overflow-hidden p-4">
          {selectedThread ? (
            <div className="mb-3 space-y-2 rounded-lg border p-3 text-xs">
              <div className="font-medium">{selectedThread.title || selectedThread.external_chat_id}</div>
              <div className="grid grid-cols-2 gap-2">
                <input className="rounded border bg-background px-2 py-1" placeholder="Account ID" value={accountId} onChange={(e) => setAccountId(e.target.value)} />
                <input className="rounded border bg-background px-2 py-1" placeholder="Contact ID" value={contactId} onChange={(e) => setContactId(e.target.value)} />
              </div>
              <Button size="sm" variant="outline" onClick={() => associationMutation.mutate()} disabled={associationMutation.isPending}>Save association</Button>
            </div>
          ) : null}
          <div className="min-h-0 flex-1 overflow-y-auto">
          {!selectedThread ? (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground"><MessageCircle className="mr-2 size-4" />Select WhatsApp thread</div>
          ) : messagesLoading ? (
            <div className="space-y-3"><Skeleton className="h-20" /><Skeleton className="h-20" /></div>
          ) : messages.length === 0 ? (
            <div className="text-sm text-muted-foreground">No messages stored for this thread.</div>
          ) : (
            <div className="flex flex-col-reverse gap-3">
              {messages.map((message) => (
                <div key={message.id} className={`max-w-[75%] rounded-lg border p-3 text-sm ${message.direction === "outbound" ? "ml-auto bg-primary/10" : "bg-card"}`}>
                  <div className="mb-1 text-xs text-muted-foreground">{message.direction} · {message.sent_at || message.received_at || ""}</div>
                  <div className="whitespace-pre-wrap break-words">{message.body_text || "[no text]"}</div>
                </div>
              ))}
            </div>
          )}
          </div>
          {selectedThread ? (
            <div className="mt-3 flex gap-2 border-t pt-3">
              <textarea className="min-h-20 flex-1 rounded border bg-background p-2 text-sm" placeholder="Reply on WhatsApp" value={replyText} onChange={(e) => setReplyText(e.target.value)} />
              <Button onClick={() => sendMutation.mutate()} disabled={!replyText.trim() || sendMutation.isPending}>Send</Button>
            </div>
          ) : null}
        </main>
      </div>
    </div>
  );
}
