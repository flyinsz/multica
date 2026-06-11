"use client";

import type { ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { Building2, Mail, Phone, Smartphone, UserRound } from "lucide-react";
import { crmApi } from "@multica/core/crm/api";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import { Button } from "@multica/ui/components/ui/button";
import { Badge } from "@multica/ui/components/ui/badge";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { useNavigation } from "../../navigation";
import { useT } from "../../i18n";
import type crmResources from "../../locales/en/crm.json";

type CRMResources = typeof crmResources;
type Translation = (selector: (resources: CRMResources) => string, options?: Record<string, unknown>) => string;

function display(value?: string | null) {
  return value && value.trim() ? value : "—";
}

function formatDate(value?: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export function CRMContactDetailPage({ contactId }: { contactId: string }) {
  const wsId = useWorkspaceId();
  const p = useWorkspacePaths();
  const navigation = useNavigation();
  const { t: rawT } = useT("crm" as any);
  const t = rawT as Translation;
  const { data: contact, isLoading, error } = useQuery({
    queryKey: ["crm", wsId, "contacts", "detail", contactId],
    queryFn: () => crmApi.getCRMContact(contactId),
    enabled: Boolean(contactId),
  });

  if (isLoading) {
    return (
      <div className="space-y-4 p-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (error || !contact) {
    return (
      <div className="p-6">
        <h1 className="text-xl font-semibold">{t(($) => $.contacts.contact_not_found)}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t(($) => $.contacts.contact_not_found_description)}</p>
      </div>
    );
  }

  const accountPath = contact.account_id ? p.crmCustomer(contact.account_id) : null;
  const emailPath = `${p.crmEmails()}?contact=${encodeURIComponent(contact.id)}`;

  return (
    <div className="space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-2">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
              <UserRound className="h-5 w-5" />
            </div>
            <div>
              <h1 className="text-2xl font-semibold tracking-tight">{contact.name}</h1>
              <p className="text-sm text-muted-foreground">{display(contact.role_title || contact.job_title || contact.role)}</p>
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            {contact.is_primary ? <Badge>{t(($) => $.contacts.primary)}</Badge> : null}
            {contact.decision_role ? <Badge variant="secondary">{contact.decision_role}</Badge> : null}
            {contact.preferred_language || contact.language ? <Badge variant="outline">{contact.preferred_language || contact.language}</Badge> : null}
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          {accountPath ? (
            <Button variant="outline" onClick={() => navigation.push(accountPath)}>
              <Building2 className="mr-2 h-4 w-4" />
              {t(($) => $.contacts.open_customer)}
            </Button>
          ) : null}
          <Button variant="outline" onClick={() => navigation.push(emailPath)}>
            <Mail className="mr-2 h-4 w-4" />
            {t(($) => $.contacts.open_emails)}
          </Button>
        </div>
      </div>

      <section className="rounded-lg border bg-card p-4">
        <h2 className="mb-4 text-base font-semibold">{t(($) => $.contacts.contact_info)}</h2>
        <div className="grid gap-4 md:grid-cols-2">
          <Info icon={<Mail className="h-4 w-4" />} label={t(($) => $.contacts.email)} value={display(contact.email)} />
          <Info icon={<Phone className="h-4 w-4" />} label={t(($) => $.contacts.phone)} value={display(contact.phone)} />
          <Info icon={<Smartphone className="h-4 w-4" />} label={t(($) => $.contacts.mobile)} value={display(contact.mobile)} />
          <Info label={t(($) => $.contacts.whatsapp)} value={display(contact.whatsapp || contact.whatsapp_id)} />
          <Info label={t(($) => $.contacts.wechat)} value={display(contact.wechat)} />
          <Info label={t(($) => $.contacts.linkedin)} value={display(contact.linkedin_url)} />
        </div>
      </section>

      <section className="rounded-lg border bg-card p-4">
        <h2 className="mb-4 text-base font-semibold">{t(($) => $.contacts.profile)}</h2>
        <div className="grid gap-4 md:grid-cols-2">
          <Info label={t(($) => $.contacts.department)} value={display(contact.department)} />
          <Info label={t(($) => $.contacts.timezone)} value={display(contact.timezone)} />
          <Info label={t(($) => $.contacts.last_contacted_at)} value={formatDate(contact.last_contacted_at)} />
          <Info label={t(($) => $.contacts.created_at)} value={formatDate(contact.created_at)} />
        </div>
      </section>

      <section className="rounded-lg border bg-card p-4">
        <h2 className="mb-2 text-base font-semibold">{t(($) => $.contacts.notes)}</h2>
        <p className="whitespace-pre-wrap text-sm text-muted-foreground">{display(contact.notes)}</p>
      </section>
    </div>
  );
}

function Info({ icon, label, value }: { icon?: ReactNode; label: string; value: string }) {
  return (
    <div className="space-y-1">
      <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {icon}
        {label}
      </div>
      <div className="text-sm">{value}</div>
    </div>
  );
}
