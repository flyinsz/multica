"use client";

import type { CRMContact } from "@multica/core/crm/types";
import { useWorkspacePaths } from "@multica/core/paths";
import { Button } from "@multica/ui/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@multica/ui/components/ui/dialog";
import type crmResources from "../../locales/en/crm.json";

type CRMResources = typeof crmResources;
type Translation = (selector: (resources: CRMResources) => string, options?: Record<string, unknown>) => string;

function messageTime(value?: string | null) {
  if (!value) return "";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function DetailRow({ label, value }: { label: string; value?: string | number | null }) {
  return (
    <div className="rounded-md border bg-muted/20 p-3">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className="mt-1 break-words text-sm">{value || "—"}</div>
    </div>
  );
}

export function CRMContactDetailDialog({ contact, open, onOpenChange, t }: { contact: CRMContact | null; open: boolean; onOpenChange: (open: boolean) => void; t: Translation }) {
  const paths = useWorkspacePaths();
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        {contact ? (
          <>
            <DialogHeader>
              <DialogTitle>{contact.name}</DialogTitle>
              <DialogDescription>{t(($) => $.emails.contact_detail)}</DialogDescription>
            </DialogHeader>
            <div className="grid gap-3 sm:grid-cols-2">
              <DetailRow label={t(($) => $.contacts.email)} value={contact.email} />
              <DetailRow label={t(($) => $.contacts.phone)} value={contact.phone} />
              <DetailRow label={t(($) => $.contacts.mobile)} value={contact.mobile} />
              <DetailRow label={t(($) => $.contacts.whatsapp)} value={contact.whatsapp || contact.whatsapp_id} />
              <DetailRow label={t(($) => $.contacts.wechat)} value={contact.wechat} />
              <DetailRow label={t(($) => $.contacts.job_title)} value={contact.job_title || contact.role_title || contact.role} />
              <DetailRow label={t(($) => $.contacts.department)} value={contact.department} />
              <DetailRow label={t(($) => $.contacts.decision_role)} value={contact.decision_role} />
              <DetailRow label={t(($) => $.contacts.preferred_language)} value={contact.preferred_language || contact.language} />
              <DetailRow label={t(($) => $.contacts.timezone)} value={contact.timezone} />
              <DetailRow label={t(($) => $.contacts.last_contacted_at)} value={messageTime(contact.last_contacted_at)} />
              <DetailRow label={t(($) => $.contacts.created_at)} value={messageTime(contact.created_at)} />
            </div>
            {contact.notes ? <div className="rounded-md border bg-muted/20 p-3 text-sm whitespace-pre-wrap">{contact.notes}</div> : null}
            <DialogFooter>
              <Button variant="outline" onClick={() => onOpenChange(false)}>{t(($) => $.actions.cancel)}</Button>
              <Button onClick={() => window.open(paths.crmContact(contact.id), "_blank", "noopener,noreferrer")}>{t(($) => $.contacts.open_detail)}</Button>
            </DialogFooter>
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
