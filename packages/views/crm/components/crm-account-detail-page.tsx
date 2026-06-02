"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Building2, ChevronRight, Pencil, Plus, Trash2 } from "lucide-react";
import { api } from "@multica/core/api";
import { useProjectDraftStore } from "@multica/core/projects";
import {
  crmAccountDetailOptions,
  crmAccountProfileOptions,
  crmCommunicationNoteListOptions,
  crmContactListOptions,
  crmEmailThreadListOptions,
  crmKeys,
} from "@multica/core/crm/queries";
import { useWorkspaceId } from "@multica/core/hooks";
import { issueKeys, useIssueDraftStore } from "@multica/core/issues";
import { useModalStore } from "@multica/core/modals";
import { projectKeys, projectListOptions } from "@multica/core/projects";
import { useWorkspacePaths } from "@multica/core/paths";
import { useNavigation } from "../../navigation";
import type {
  CRMAccount,
  CRMAccountPriority,
  CRMAccountRating,
  CRMAccountSource,
  CRMAccountStatus,
  CRMAccountType,
  CRMCommunicationChannel,
  CRMCommunicationDirection,
  CRMContact,
  CRMContactDecisionRole,
  CreateCRMContactRequest,
  Issue,
  Project,
} from "@multica/core/types";
import { Button } from "@multica/ui/components/ui/button";
import { Checkbox } from "@multica/ui/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@multica/ui/components/ui/dialog";
import { Badge } from "@multica/ui/components/ui/badge";
import { Input } from "@multica/ui/components/ui/input";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@multica/ui/components/ui/select";
import { Textarea } from "@multica/ui/components/ui/textarea";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@multica/ui/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@multica/ui/components/ui/tabs";
import { useT } from "../../i18n";
import { CreateProjectModal } from "../../modals/create-project";
import type crmResources from "../../locales/en/crm.json";
import { COUNTRY_OPTIONS, countryByCode, findCityCode, findRegionCode, loadCityOptions, loadRegionOptions, localizedName, localizedSort, normalizeLocale, useLocationSelection } from "../geo";
import { appendTag, CRM_INDUSTRY_OPTIONS, industryLabel, optionLabel, splitTags, subIndustryOptions } from "../options";

type CRMResources = typeof crmResources;
type Translation = (
  selector: (resources: CRMResources) => string,
  options?: Record<string, unknown>,
) => string;

type Locale = "en" | "zh-Hans";

type AccountOwnerType = "" | "member" | "agent";

type AccountFormState = {
  name: string;
  accountCode: string;
  accountType: CRMAccountType;
  status: CRMAccountStatus;
  source: CRMAccountSource;
  rating: CRMAccountRating;
  priority: CRMAccountPriority;
  ownerType: AccountOwnerType;
  ownerMemberID: string;
  ownerAgentID: string;
  website: string;
  countryCode: string;
  regionCode: string;
  cityCode: string;
  industry: string;
  subIndustry: string;
  annualRevenue: string;
  employeeCount: string;
  tags: string;
  nextFollowUpAt: string;
  notes: string;
};

type ContactFormState = {
  id?: string;
  name: string;
  email: string;
  phone: string;
  mobile: string;
  whatsapp: string;
  wechat: string;
  jobTitle: string;
  department: string;
  decisionRole: CRMContactDecisionRole | "";
  preferredLanguage: string;
  timezone: string;
  isPrimary: boolean;
  notes: string;
};

type ProfileFormState = {
  summary: string;
  businessModel: string;
  mainProducts: string;
  procurementNeeds: string;
  painPoints: string;
  decisionProcess: string;
  communicationPreference: string;
  riskNotes: string;
  cooperationHistory: string;
  aliases: string;
  searchKeywords: string;
};

type NoteFormState = {
  channel: CRMCommunicationChannel;
  direction: CRMCommunicationDirection;
  subject: string;
  body: string;
};

const PROFILE_FIELD_KEYS = [
  "business_model",
  "main_products",
  "procurement_needs",
  "pain_points",
  "decision_process",
  "communication_preference",
  "risk_notes",
  "cooperation_history",
] as const;

function profileText(profileJson: Record<string, unknown> | undefined, key: (typeof PROFILE_FIELD_KEYS)[number]) {
  const value = profileJson?.[key];
  return typeof value === "string" ? value : "";
}

function profileDisplayText(profileJson: Record<string, unknown> | undefined, key: string) {
  const value = profileJson?.[key];
  if (typeof value === "string") return value;
  if (Array.isArray(value)) return value.map((item) => typeof item === "string" ? item : JSON.stringify(item)).join("、");
  if (value && typeof value === "object") return JSON.stringify(value);
  if (typeof value === "number") return String(value);
  return "";
}

function profileFollowUpValue(profileJson: Record<string, unknown> | undefined) {
  const recommendation = profileJson?.follow_up_recommendation;
  const rec = recommendation && typeof recommendation === "object" && !Array.isArray(recommendation) ? recommendation as Record<string, unknown> : null;
  const rawDate = rec?.next_follow_up_at ?? rec?.date ?? profileJson?.next_follow_up_at;
  const date = typeof rawDate === "string" && rawDate ? new Date(rawDate) : null;
  const parts = [
    date && !Number.isNaN(date.getTime()) ? date.toLocaleString() : typeof rawDate === "string" ? rawDate : "",
    typeof rec?.reason === "string" ? rec.reason : "",
    typeof rec?.confidence === "string" ? `置信度：${rec.confidence}` : "",
  ].filter(Boolean);
  return parts.join(" · ");
}

function emailSummaryText(item: { snippet?: string | null; subject?: string | null }) {
  return (item.snippet || item.subject || "").replace(/\s+/g, " ").trim().slice(0, 180);
}

function profileToForm(profile: { summary?: string | null; profile_json?: Record<string, unknown> } | null | undefined): ProfileFormState {
  return {
    summary: profile?.summary ?? "",
    businessModel: profileText(profile?.profile_json, "business_model"),
    mainProducts: profileText(profile?.profile_json, "main_products"),
    procurementNeeds: profileText(profile?.profile_json, "procurement_needs"),
    painPoints: profileText(profile?.profile_json, "pain_points"),
    decisionProcess: profileText(profile?.profile_json, "decision_process"),
    communicationPreference: profileText(profile?.profile_json, "communication_preference"),
    riskNotes: profileText(profile?.profile_json, "risk_notes"),
    cooperationHistory: profileText(profile?.profile_json, "cooperation_history"),
    aliases: profileDisplayText(profile?.profile_json, "aliases"),
    searchKeywords: profileDisplayText(profile?.profile_json, "search_keywords"),
  };
}

function profilePayload(form: ProfileFormState, current?: { profile_json?: Record<string, unknown> } | null) {
  return {
    summary: form.summary || null,
    profile_json: {
      ...(current?.profile_json ?? {}),
      business_model: form.businessModel,
      main_products: form.mainProducts,
      procurement_needs: form.procurementNeeds,
      pain_points: form.painPoints,
      decision_process: form.decisionProcess,
      communication_preference: form.communicationPreference,
      risk_notes: form.riskNotes,
      cooperation_history: form.cooperationHistory,
      aliases: form.aliases.split(/[\n,，、]+/).map((item) => item.trim()).filter(Boolean),
      search_keywords: form.searchKeywords.split(/[\n,，、]+/).map((item) => item.trim()).filter(Boolean),
    },
  };
}

function blankNoteForm(): NoteFormState {
  return { channel: "manual", direction: "note", subject: "", body: "" };
}

const toDateTimeLocal = (value?: string | null) => value ? value.slice(0, 16) : "";
const fromDateTimeLocal = (value: string) => value ? new Date(value).toISOString() : null;

const tagSuggestions = (account?: CRMAccount | null) => account?.tags?.filter(Boolean).slice(0, 12) ?? [];

function countryName(codeOrName: string | null | undefined, locale: Locale) {
  const country = countryByCode(codeOrName);
  return country ? localizedName(country.name, locale) : codeOrName || "";
}


function ProfileTextarea({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="space-y-1 text-sm">
      <span className="block text-xs font-medium text-muted-foreground">{label}</span>
      <Textarea className="min-h-20" value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

function accountToForm(account: CRMAccount): AccountFormState {
  const country = countryByCode(account.country_code || account.country);

  return {
    name: account.name,
    accountCode: account.account_code ?? "",
    accountType: account.account_type,
    status: account.status,
    source: account.source ?? "manual",
    rating: account.rating,
    priority: account.priority,
    ownerType: account.owner_agent_id ? "agent" : account.owner_member_id ? "member" : "",
    ownerMemberID: account.owner_member_id ?? "",
    ownerAgentID: account.owner_agent_id ?? "",
    website: account.website ?? "",
    countryCode: country?.code ?? account.country_code ?? account.country ?? "",
    regionCode: account.region ?? "",
    cityCode: account.city ?? "",
    industry: account.industry ?? "",
    subIndustry: account.sub_industry ?? "",
    annualRevenue: account.annual_revenue ?? "",
    employeeCount: account.employee_count ?? "",
    tags: account.tags?.join(", ") ?? "",
    nextFollowUpAt: toDateTimeLocal(account.next_follow_up_at),
    notes: account.notes ?? "",
  };
}

const blankContactForm = (): ContactFormState => ({
  name: "",
  email: "",
  phone: "",
  mobile: "",
  whatsapp: "",
  wechat: "",
  jobTitle: "",
  department: "",
  decisionRole: "",
  preferredLanguage: "",
  timezone: "",
  isPrimary: false,
  notes: "",
});

function contactToForm(contact: CRMContact): ContactFormState {
  return {
    id: contact.id,
    name: contact.name,
    email: contact.email ?? "",
    phone: contact.phone ?? "",
    mobile: contact.mobile ?? "",
    whatsapp: contact.whatsapp ?? contact.whatsapp_id ?? "",
    wechat: contact.wechat ?? "",
    jobTitle: contact.job_title ?? contact.role_title ?? "",
    department: contact.department ?? "",
    decisionRole: contact.decision_role ?? "",
    preferredLanguage: contact.preferred_language ?? contact.language ?? "",
    timezone: contact.timezone ?? "",
    isPrimary: contact.is_primary,
    notes: contact.notes ?? "",
  };
}

function AccountStatusLabel({ status, t }: { status: CRMAccountStatus; t: Translation }) {
  const labels: Record<CRMAccountStatus, string> = {
    active: t(($) => $.statuses.active),
    inactive: t(($) => $.statuses.inactive),
    prospect: t(($) => $.statuses.prospect),
    archived: t(($) => $.statuses.archived),
  };
  return labels[status] ?? status;
}

function ChannelLabel({ channel, t }: { channel: string; t: Translation }) {
  const labels: Record<string, string> = {
    manual: t(($) => $.channels.manual),
    email: t(($) => $.channels.email),
    whatsapp: t(($) => $.channels.whatsapp),
    phone: t(($) => $.channels.phone),
    meeting: t(($) => $.channels.meeting),
    other: t(($) => $.channels.other),
  };
  return labels[channel] ?? channel;
}

function FieldRow({ label, value }: { label: string; value?: string | null }) {
  return (
    <div className="rounded-md border bg-background/50 p-3">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm">{value || "—"}</div>
    </div>
  );
}


const accountTypeLabel = (t: Translation, value: CRMAccountType) => t(($) => $.account_types[value] ?? value);
const statusLabel = (t: Translation, value: CRMAccountStatus) => t(($) => $.statuses[value] ?? value);
const localizedNameOrFallback = (name: Parameters<typeof localizedName>[0] | undefined, locale: "en" | "zh-Hans", fallback: string) => name ? localizedName(name, locale) : fallback;
const optionLabelOrFallback = (option: Parameters<typeof optionLabel>[0] | undefined, locale: "en" | "zh-Hans", fallback: string) => option ? optionLabel(option, locale) : fallback;

const ownerLabel = (t: Translation, form: AccountFormState, members: Array<{ id: string; user_id: string; name: string; email: string; user?: { name?: string; email?: string } }>, agents: Array<{ id: string; name: string }>) => {
  if (form.ownerType === "member") {
    const member = members.find((item) => item.user_id === form.ownerMemberID || item.id === form.ownerMemberID);
    return member ? `成员 · ${member.name || member.email || member.user?.name || member.user?.email || member.id}` : t(($) => $.customers.owner);
  }
  if (form.ownerType === "agent") {
    const agent = agents.find((item) => item.id === form.ownerAgentID);
    return agent ? `Agent · ${agent.name}` : t(($) => $.customers.owner);
  }
  return t(($) => $.customers.owner);
};

function LabeledField({ label, className = "", children }: { label: string; className?: string; children: ReactNode }) {
  return (
    <label className={`space-y-1.5 ${className}`}>
      <span className="block text-xs font-medium text-muted-foreground">{label}</span>
      {children}
    </label>
  );
}

function AccountForm({ form, setForm, t, locale, suggestedTags = [], members = [], agents = [] }: { form: AccountFormState; setForm: (next: AccountFormState) => void; t: Translation; locale: Locale; suggestedTags?: string[]; members?: Array<{ id: string; user_id: string; name: string; email: string; user?: { name?: string; email?: string } }>; agents?: Array<{ id: string; name: string }> }) {
  const { regions, cities, regionsLoading, citiesLoading } = useLocationSelection(form.countryCode, form.regionCode, locale);
  const countries = useMemo(() => localizedSort(COUNTRY_OPTIONS, locale), [locale]);
  const subIndustries = subIndustryOptions(form.industry);
  return (
    <div className="grid max-h-[70vh] gap-4 overflow-y-auto rounded-lg border bg-muted/20 p-4 sm:grid-cols-2">
      <LabeledField label={t(($) => $.customers.new_customer_name)}><Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder={t(($) => $.customers.new_customer_name)} /></LabeledField>
      <LabeledField label={t(($) => $.customers.website)}><Input value={form.website} onChange={(e) => setForm({ ...form, website: e.target.value })} placeholder={t(($) => $.customers.website)} /></LabeledField>
      <LabeledField label={t(($) => $.customers.account_type)}><Select value={form.accountType} onValueChange={(value) => setForm({ ...form, accountType: value as CRMAccountType })}><SelectTrigger className="w-full"><SelectValue>{() => accountTypeLabel(t, form.accountType)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="prospect">{t(($) => $.account_types.prospect)}</SelectItem><SelectItem value="customer">{t(($) => $.account_types.customer)}</SelectItem><SelectItem value="partner">{t(($) => $.account_types.partner)}</SelectItem><SelectItem value="supplier">{t(($) => $.account_types.supplier)}</SelectItem><SelectItem value="competitor">{t(($) => $.account_types.competitor)}</SelectItem><SelectItem value="other">{t(($) => $.account_types.other)}</SelectItem></SelectContent></Select></LabeledField>
      <LabeledField label={t(($) => $.customers.status)}><Select value={form.status} onValueChange={(value) => setForm({ ...form, status: value as CRMAccountStatus })}><SelectTrigger className="w-full"><SelectValue>{() => statusLabel(t, form.status)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="active">{t(($) => $.statuses.active)}</SelectItem><SelectItem value="prospect">{t(($) => $.statuses.prospect)}</SelectItem><SelectItem value="inactive">{t(($) => $.statuses.inactive)}</SelectItem><SelectItem value="archived">{t(($) => $.statuses.archived)}</SelectItem></SelectContent></Select></LabeledField>
      <LabeledField label={t(($) => $.customers.owner)}><Select value={form.ownerType === "agent" ? `agent:${form.ownerAgentID}` : form.ownerType === "member" ? `member:${form.ownerMemberID}` : "none"} onValueChange={(value) => { const nextValue = value ?? "none"; const [type, id = ""] = nextValue === "none" ? ["", ""] : nextValue.split(":"); setForm({ ...form, ownerType: (type || "") as AccountOwnerType, ownerMemberID: type === "member" ? id : "", ownerAgentID: type === "agent" ? id : "" }); }}><SelectTrigger className="w-full"><SelectValue>{() => ownerLabel(t, form, members, agents)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="none">{t(($) => $.customers.owner)}</SelectItem>{members.map((member) => <SelectItem key={member.id} value={`member:${member.user_id}`}>Member · {member.name || member.email || member.user?.name || member.user?.email || member.id}</SelectItem>)}{agents.map((agent) => <SelectItem key={agent.id} value={`agent:${agent.id}`}>Agent · {agent.name}</SelectItem>)}</SelectContent></Select></LabeledField>
      <LabeledField label={t(($) => $.customers.country)}><Select value={form.countryCode || "none"} onValueChange={(value) => setForm({ ...form, countryCode: !value || value === "none" ? "" : value, regionCode: "", cityCode: "" })}><SelectTrigger className="w-full"><SelectValue>{() => form.countryCode ? localizedNameOrFallback(countries.find((option) => option.code === form.countryCode)?.name, locale, form.countryCode) : t(($) => $.customers.country)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="none">{t(($) => $.customers.country)}</SelectItem>{countries.map((option) => <SelectItem key={option.code} value={option.code}>{localizedName(option.name, locale)}</SelectItem>)}</SelectContent></Select></LabeledField>
      <LabeledField label={t(($) => $.customers.region)}><Select value={form.regionCode || "none"} onValueChange={(value) => setForm({ ...form, regionCode: !value || value === "none" ? "" : value, cityCode: "" })} disabled={!form.countryCode || regionsLoading}><SelectTrigger className="w-full"><SelectValue>{() => form.regionCode ? localizedNameOrFallback(regions.find((option) => option.code === form.regionCode)?.name, locale, form.regionCode) : t(($) => $.customers.region)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="none">{regionsLoading ? `${t(($) => $.customers.region)}...` : t(($) => $.customers.region)}</SelectItem>{regions.map((option) => <SelectItem key={option.code} value={option.code}>{localizedName(option.name, locale)}</SelectItem>)}</SelectContent></Select></LabeledField>
      <LabeledField label={t(($) => $.customers.city)}><Select value={form.cityCode || "none"} onValueChange={(value) => setForm({ ...form, cityCode: !value || value === "none" ? "" : value })} disabled={!form.regionCode || citiesLoading}><SelectTrigger className="w-full"><SelectValue>{() => form.cityCode ? localizedNameOrFallback(cities.find((option) => option.code === form.cityCode)?.name, locale, form.cityCode) : t(($) => $.customers.city)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="none">{citiesLoading ? `${t(($) => $.customers.city)}...` : t(($) => $.customers.city)}</SelectItem>{cities.map((option) => <SelectItem key={option.code} value={option.code}>{localizedName(option.name, locale)}</SelectItem>)}</SelectContent></Select></LabeledField>
      <LabeledField label={t(($) => $.customers.industry)}><Select value={form.industry || "none"} onValueChange={(value) => setForm({ ...form, industry: !value || value === "none" ? "" : value, subIndustry: "" })}><SelectTrigger className="w-full"><SelectValue>{() => form.industry ? industryLabel(form.industry, locale) : t(($) => $.customers.industry)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="none">{t(($) => $.customers.industry)}</SelectItem>{CRM_INDUSTRY_OPTIONS.map((option) => <SelectItem key={option.value} value={option.value}>{industryLabel(option.value, locale)}</SelectItem>)}</SelectContent></Select></LabeledField>
      <LabeledField label={t(($) => $.customers.sub_industry)}><Select value={form.subIndustry || "none"} onValueChange={(value) => setForm({ ...form, subIndustry: !value || value === "none" ? "" : value })} disabled={!form.industry}><SelectTrigger className="w-full"><SelectValue>{() => form.subIndustry ? optionLabelOrFallback(subIndustries.find((option) => option.value === form.subIndustry), locale, form.subIndustry) : t(($) => $.customers.sub_industry)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="none">{t(($) => $.customers.sub_industry)}</SelectItem>{subIndustries.map((option) => <SelectItem key={option.value} value={option.value}>{optionLabel(option, locale)}</SelectItem>)}</SelectContent></Select></LabeledField>
      <LabeledField label={t(($) => $.customers.annual_revenue)}><Input value={form.annualRevenue} onChange={(e) => setForm({ ...form, annualRevenue: e.target.value })} placeholder={t(($) => $.customers.annual_revenue)} /></LabeledField>
      <LabeledField label={t(($) => $.customers.employee_count)}><Input value={form.employeeCount} onChange={(e) => setForm({ ...form, employeeCount: e.target.value })} placeholder={t(($) => $.customers.employee_count)} /></LabeledField>
      <LabeledField label={t(($) => $.customers.tags)} className="sm:col-span-2"><div className="space-y-2">
        <Input aria-label={t(($) => $.customers.tags)} value={form.tags} onChange={(e) => setForm({ ...form, tags: e.target.value })} placeholder={t(($) => $.customers.tags_placeholder)} />
        {suggestedTags.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {suggestedTags.map((tag) => (
              <button key={tag} type="button" className="rounded-full border px-2 py-0.5 text-xs text-muted-foreground hover:bg-accent" onClick={() => setForm({ ...form, tags: appendTag(form.tags, tag) })}>{tag}</button>
            ))}
          </div>
        )}
      </div></LabeledField>
      <LabeledField label={t(($) => $.customers.next_follow_up_at)} className="sm:col-span-2"><Input aria-label={t(($) => $.customers.next_follow_up_at)} type="datetime-local" value={form.nextFollowUpAt} onChange={(e) => setForm({ ...form, nextFollowUpAt: e.target.value })} /></LabeledField>
      <LabeledField label={t(($) => $.customers.notes)} className="sm:col-span-2"><textarea className="min-h-24 w-full rounded-md border bg-background px-3 py-2 text-sm" value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} placeholder={t(($) => $.customers.notes)} /></LabeledField>
    </div>
  );
}

function ContactForm({ form, setForm, t }: { form: ContactFormState; setForm: (next: ContactFormState) => void; t: Translation }) {
  return (
    <div className="grid max-h-[70vh] gap-4 overflow-y-auto rounded-lg border bg-muted/20 p-4 sm:grid-cols-2">
      <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder={t(($) => $.contacts.name)} />
      <Input value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} placeholder={t(($) => $.contacts.email)} />
      <Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} placeholder={t(($) => $.contacts.phone)} />
      <Input value={form.mobile} onChange={(e) => setForm({ ...form, mobile: e.target.value })} placeholder={t(($) => $.contacts.mobile)} />
      <Input value={form.whatsapp} onChange={(e) => setForm({ ...form, whatsapp: e.target.value })} placeholder={t(($) => $.contacts.whatsapp)} />
      <Input value={form.wechat} onChange={(e) => setForm({ ...form, wechat: e.target.value })} placeholder={t(($) => $.contacts.wechat)} />
      <Input value={form.jobTitle} onChange={(e) => setForm({ ...form, jobTitle: e.target.value })} placeholder={t(($) => $.contacts.job_title)} />
      <Input value={form.department} onChange={(e) => setForm({ ...form, department: e.target.value })} placeholder={t(($) => $.contacts.department)} />
      <select className="h-9 rounded-md border bg-background px-3 text-sm" value={form.decisionRole} onChange={(e) => setForm({ ...form, decisionRole: e.target.value as CRMContactDecisionRole | "" })}>
        <option value="">{t(($) => $.contacts.decision_role)}</option>
        <option value="decision_maker">{t(($) => $.decision_roles.decision_maker)}</option>
        <option value="influencer">{t(($) => $.decision_roles.influencer)}</option>
        <option value="buyer">{t(($) => $.decision_roles.buyer)}</option>
        <option value="user">{t(($) => $.decision_roles.user)}</option>
        <option value="technical">{t(($) => $.decision_roles.technical)}</option>
        <option value="finance">{t(($) => $.decision_roles.finance)}</option>
        <option value="gatekeeper">{t(($) => $.decision_roles.gatekeeper)}</option>
        <option value="other">{t(($) => $.decision_roles.other)}</option>
      </select>
      <Input value={form.preferredLanguage} onChange={(e) => setForm({ ...form, preferredLanguage: e.target.value })} placeholder={t(($) => $.contacts.preferred_language)} />
      <Input value={form.timezone} onChange={(e) => setForm({ ...form, timezone: e.target.value })} placeholder={t(($) => $.contacts.timezone)} />
      <label className="flex items-center gap-2 text-xs text-muted-foreground">
        <input type="checkbox" checked={form.isPrimary} onChange={(e) => setForm({ ...form, isPrimary: e.target.checked })} />
        {t(($) => $.contacts.is_primary)}
      </label>
      <textarea className="min-h-24 rounded-md border bg-background px-3 py-2 text-sm sm:col-span-2" value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} placeholder={t(($) => $.contacts.notes)} />
    </div>
  );
}

function contactPayload(form: ContactFormState): CreateCRMContactRequest {
  return {
    name: form.name,
    email: form.email || null,
    phone: form.phone || null,
    mobile: form.mobile || null,
    whatsapp_id: form.whatsapp || null,
    whatsapp: form.whatsapp || null,
    wechat: form.wechat || null,
    job_title: form.jobTitle || null,
    role_title: form.jobTitle || null,
    department: form.department || null,
    decision_role: form.decisionRole || null,
    preferred_language: form.preferredLanguage || null,
    language: form.preferredLanguage || null,
    timezone: form.timezone || null,
    is_primary: form.isPrimary,
    notes: form.notes || null,
  };
}

function crmAccountResource(project: Project, accountId: string) {
  const resources = project.resources ?? [];
  return resources.find((resource) => {
    const ref = resource.resource_ref as { account_id?: string };
    return resource.resource_type === "crm_account" && ref.account_id === accountId;
  });
}

function projectLinkedToAccount(project: Project, accountId: string) {
  return Boolean(crmAccountResource(project, accountId));
}

function formatIssueOption(issue: Issue) {
  return [issue.identifier, issue.title].filter(Boolean).join(" · ") || issue.id;
}

function nextLinkedProjectTitle(accountName: string, projects: Project[]) {
  const baseTitle = `CRM:${accountName}`;
  const existingTitles = new Set(projects.map((project) => project.title.toLowerCase()));
  if (!existingTitles.has(baseTitle.toLowerCase())) return baseTitle;
  let suffix = 2;
  while (existingTitles.has(`${baseTitle} ${suffix}`.toLowerCase())) suffix += 1;
  return `${baseTitle} ${suffix}`;
}

function toggleProjectId(projectIds: string[], projectId: string) {
  return projectIds.includes(projectId)
    ? projectIds.filter((id) => id !== projectId)
    : [...projectIds, projectId];
}

export function CRMAccountDetailPage({ accountId }: { accountId: string }) {
  const wsId = useWorkspaceId();
  const queryClient = useQueryClient();
  const navigation = useNavigation();
  const paths = useWorkspacePaths();
  const { t: rawT, i18n } = useT("crm");
  const t = rawT as Translation;
  const locale = normalizeLocale(i18n.language);

  const { data: account, isLoading: accountLoading } = useQuery(crmAccountDetailOptions(wsId, accountId));
  const { data: profile, isLoading: profileLoading } = useQuery(crmAccountProfileOptions(wsId, accountId));
  const { data: contacts = [], isLoading: contactsLoading } = useQuery(crmContactListOptions(wsId, accountId));
  const { data: notes = [], isLoading: notesLoading } = useQuery(crmCommunicationNoteListOptions(wsId, accountId));
  const { data: emailThreadData, isLoading: emailThreadsLoading } = useQuery(crmEmailThreadListOptions(wsId, accountId, ""));
  const emailThreads = emailThreadData?.threads ?? [];
  const emailMessages = emailThreadData?.messages ?? [];
  const accountEmailItems = emailMessages.length ? emailMessages : emailThreads;
  const unreadEmailCount = (emailThreadData?.counts as any)?.inbox_unread ?? emailThreads.filter((thread: any) => thread.is_read !== true && thread.direction !== "outbound" && !thread.is_trashed).length;
  const { data: projects = [] } = useQuery(projectListOptions(wsId));
  const { data: members = [] } = useQuery({ queryKey: ["workspace", wsId, "members", "crm-account-detail"], queryFn: () => api.listMembers(wsId), enabled: Boolean(wsId) });
  const { data: agents = [] } = useQuery({ queryKey: ["agents", wsId, "crm-account-detail"], queryFn: () => api.listAgents({ workspace_id: wsId }), enabled: Boolean(wsId) });
  const ownerLabel = useMemo(() => {
    if (!account) return "";
    if (account.owner_agent_id) {
      const agent = agents.find((item) => item.id === account.owner_agent_id);
      return agent ? `Agent: ${agent.name}` : `Agent: ${account.owner_agent_id}`;
    }
    if (account.owner_member_id) {
      const member = members.find((item) => item.user_id === account.owner_member_id || item.id === account.owner_member_id);
      const memberName = member?.name || member?.email || account.owner_member_id;
      return `Member: ${memberName}`;
    }
    return "";
  }, [account, agents, members]);

  const [accountForm, setAccountForm] = useState<AccountFormState | null>(null);
  const [profileForm, setProfileForm] = useState<ProfileFormState | null>(null);
  const [contactForm, setContactForm] = useState<ContactFormState | null>(null);
  const [noteForm, setNoteForm] = useState<NoteFormState>(() => blankNoteForm());
  const [selectedProjectIds, setSelectedProjectIds] = useState<string[]>([]);
  const [selectedFollowUpProjectId, setSelectedFollowUpProjectId] = useState("");
  const [createLinkedProjectOpen, setCreateLinkedProjectOpen] = useState(false);
  const setProjectDraft = useProjectDraftStore((s) => s.setDraft);
  const clearProjectDraft = useProjectDraftStore((s) => s.clearDraft);
  const openModal = useModalStore((s) => s.open);
  const setIssueDraft = useIssueDraftStore((s) => s.setDraft);
  const clearIssueDraft = useIssueDraftStore((s) => s.clearDraft);

  const { data: followUpIssues = [] } = useQuery({
    queryKey: [...issueKeys.all(wsId), "crm-follow-up", selectedFollowUpProjectId],
    queryFn: async () => {
      if (!selectedFollowUpProjectId) return [];
      const response = await api.listIssues({ project_id: selectedFollowUpProjectId, limit: 50 });
      return response.issues;
    },
    enabled: Boolean(selectedFollowUpProjectId),
  });

  const linkedProjectIds = useMemo(
    () => projects.filter((project) => projectLinkedToAccount(project, accountId)).map((project) => project.id),
    [accountId, projects],
  );
  const linkedProjects = useMemo(
    () => projects.filter((project) => selectedProjectIds.includes(project.id)),
    [projects, selectedProjectIds],
  );

  useEffect(() => {
    setSelectedProjectIds((current) => {
      const merged = Array.from(new Set([...current, ...linkedProjectIds]));
      return merged.length === current.length && merged.every((id, index) => id === current[index]) ? current : merged;
    });
  }, [linkedProjectIds]);

  useEffect(() => {
    if (selectedFollowUpProjectId && !selectedProjectIds.includes(selectedFollowUpProjectId)) {
      setSelectedFollowUpProjectId("");
    }
  }, [selectedFollowUpProjectId, selectedProjectIds]);

  const updateAccount = useMutation({
    mutationFn: async () => {
      if (!accountForm) throw new Error("missing account form");
      const country = countryByCode(accountForm.countryCode);
      const regions = await loadRegionOptions(accountForm.countryCode, locale);
      const region = regions.find((option) => option.code === accountForm.regionCode);
      const cities = await loadCityOptions(accountForm.countryCode, accountForm.regionCode, locale);
      const city = cities.find((option) => option.code === accountForm.cityCode);
      return api.updateCRMAccount(accountId, {
        name: accountForm.name,
        account_code: accountForm.accountCode || null,
        account_type: accountForm.accountType,
        website: accountForm.website || null,
        country: accountForm.countryCode || null,
        country_code: accountForm.countryCode || null,
        country_name: country ? localizedName(country.name, locale) : null,
        region: region ? localizedName(region.name, locale) : accountForm.regionCode || null,
        city: city ? localizedName(city.name, locale) : accountForm.cityCode || null,
        industry: accountForm.industry || null,
        sub_industry: accountForm.subIndustry || null,
        status: accountForm.status,
        source: accountForm.source,
        rating: accountForm.rating,
        priority: accountForm.priority,
        owner_type: accountForm.ownerType || null,
        owner_member_id: accountForm.ownerType === "member" ? accountForm.ownerMemberID || null : null,
        owner_agent_id: accountForm.ownerType === "agent" ? accountForm.ownerAgentID || null : null,
        annual_revenue: accountForm.annualRevenue || null,
        employee_count: accountForm.employeeCount || null,
        tags: splitTags(accountForm.tags),
        next_follow_up_at: fromDateTimeLocal(accountForm.nextFollowUpAt),
        notes: accountForm.notes || null,
      });
    },
    onSuccess: async () => {
      setAccountForm(null);
      await queryClient.invalidateQueries({ queryKey: crmKeys.accounts(wsId) });
      await queryClient.invalidateQueries({ queryKey: crmKeys.accountDetail(wsId, accountId) });
    },
  });

  const deleteAccount = useMutation({
    mutationFn: () => api.deleteCRMAccount(accountId),
    onSuccess: () => navigation.push(paths.crmCustomers()),
  });

  const saveContact = useMutation({
    mutationFn: () => {
      if (!contactForm) throw new Error("missing contact form");
      return contactForm.id
        ? api.updateCRMContact(accountId, contactForm.id, contactPayload(contactForm))
        : api.createCRMContact(accountId, contactPayload(contactForm));
    },
    onSuccess: async () => {
      setContactForm(null);
      await queryClient.invalidateQueries({ queryKey: crmKeys.accounts(wsId) });
      await queryClient.invalidateQueries({ queryKey: crmKeys.contacts(wsId, accountId) });
    },
  });

  const deleteContact = useMutation({
    mutationFn: (contactId: string) => api.deleteCRMContact(accountId, contactId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: crmKeys.accounts(wsId) });
      await queryClient.invalidateQueries({ queryKey: crmKeys.contacts(wsId, accountId) });
    },
  });

  const saveProfile = useMutation({
    mutationFn: () => {
      if (!profileForm) throw new Error("missing profile form");
      return api.upsertCRMAccountProfile(accountId, profilePayload(profileForm, profile));
    },
    onSuccess: async () => {
      setProfileForm(null);
      await queryClient.invalidateQueries({ queryKey: crmKeys.profile(wsId, accountId) });
    },
  });

  const refreshProfile = useMutation({
    mutationFn: () => api.refreshCRMAccountProfile(accountId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: crmKeys.profile(wsId, accountId), refetchType: "active" });
      await queryClient.invalidateQueries({ queryKey: crmKeys.detail(wsId, accountId), refetchType: "active" });
      await queryClient.invalidateQueries({ queryKey: crmKeys.notes(wsId, accountId), refetchType: "active" });
    },
  });

  const createNote = useMutation({
    mutationFn: () => api.createCRMCommunicationNote(accountId, {
      body: noteForm.body,
      channel: noteForm.channel,
      direction: noteForm.direction,
      subject: noteForm.subject || null,
    }),
    onSuccess: async () => {
      setNoteForm(blankNoteForm());
      await queryClient.invalidateQueries({ queryKey: crmKeys.notes(wsId, accountId) });
    },
  });

  const linkProject = useMutation({
    mutationFn: async ({ project, shouldLink }: { project: Project; shouldLink: boolean }) => {
      if (shouldLink) {
        return api.createProjectResource(project.id, {
          resource_type: "crm_account",
          resource_ref: { account_id: accountId, name: account?.name ?? "" },
          label: account?.name ?? undefined,
        });
      }
      const resource = crmAccountResource(project, accountId);
      if (!resource) return undefined;
      await api.deleteProjectResource(project.id, resource.id);
      return undefined;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: projectKeys.list(wsId) });
      await queryClient.invalidateQueries({ queryKey: crmKeys.accountDetail(wsId, accountId) });
    },
    onError: async () => {
      setSelectedProjectIds(linkedProjectIds);
      await queryClient.invalidateQueries({ queryKey: projectKeys.list(wsId) });
    },
  });

  const handleProjectToggle = (project: Project) => {
    const shouldLink = !selectedProjectIds.includes(project.id);
    setSelectedProjectIds((current) => toggleProjectId(current, project.id));
    linkProject.mutate({ project, shouldLink });
  };

  const openCreateLinkedProject = () => {
    if (!account) return;
    clearProjectDraft();
    setProjectDraft({
      title: nextLinkedProjectTitle(account.name, projects),
      description: account.notes ?? "",
      status: "planned",
      priority: "medium",
    });
    setCreateLinkedProjectOpen(true);
  };

  const handleLinkedProjectCreated = async (project: Project) => {
    await queryClient.invalidateQueries({ queryKey: projectKeys.list(wsId) });
    await queryClient.invalidateQueries({ queryKey: crmKeys.accountDetail(wsId, accountId) });
    setSelectedProjectIds((current) => Array.from(new Set([...current, project.id])));
    setSelectedFollowUpProjectId(project.id);
  };

  const openCreateFollowUpIssue = () => {
    if (!account) return;
    clearIssueDraft();
    setIssueDraft({
      title: t(($) => $.projects.follow_up_placeholder, { name: account.name }),
      priority: "medium",
    });
    openModal("create-issue", { project_id: selectedFollowUpProjectId || undefined });
  };


  if (accountLoading || !account) {
    return (
      <div className="flex h-full items-center justify-center">
        <Skeleton className="h-16 w-80" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <div className="shrink-0 border-b px-5 py-4">
        <nav className="mb-3 flex items-center gap-1 text-xs" aria-label="Breadcrumb">
          <button
            type="button"
            className="rounded px-1.5 py-1 text-muted-foreground hover:bg-accent hover:text-foreground"
            onClick={() => navigation.push(paths.crmCustomers())}
          >
            {t(($) => $.customers.title)}
          </button>
          <ChevronRight className="size-3 text-muted-foreground/60" />
          <span className="max-w-[24rem] truncate px-1.5 py-1 font-medium text-foreground">{account.name}</span>
        </nav>
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <Building2 className="size-4 text-muted-foreground" />
              <h1 className="truncate text-sm font-medium">{account.name}</h1>
              <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-700 dark:text-emerald-300">
                <AccountStatusLabel status={account.status} t={t} />
              </span>
            </div>
            <p className="mt-1 text-xs text-muted-foreground">
              {[account.website, countryName(account.country_code || account.country_name || account.country, locale), account.region, account.industry].filter(Boolean).join(" · ") || t(($) => $.customers.basic_profile)}
            </p>
          </div>
          <div className="flex items-center gap-2">
              <Button size="sm" variant="outline" onClick={async () => {
                const next = accountToForm(account);
                next.regionCode = await findRegionCode(next.countryCode, account.region);
                next.cityCode = await findCityCode(next.countryCode, next.regionCode, account.city);
                setAccountForm(next);
              }}>
              <Pencil className="mr-1 size-4" /> {t(($) => $.actions.edit)}
            </Button>
            <Button size="sm" variant="outline" disabled={deleteAccount.isPending} onClick={() => window.confirm(t(($) => $.customers.delete_confirm)) && deleteAccount.mutate()}>
              <Trash2 className="mr-1 size-4" /> {t(($) => $.actions.delete)}
            </Button>
          </div>
        </div>
      </div>

      <Tabs defaultValue="overview" className="min-h-0 flex-1 gap-0">
        <div className="border-b px-6 py-3">
          <TabsList variant="line" className="w-full justify-start overflow-x-auto">
            <TabsTrigger value="overview">{t(($) => $.tabs.overview)}</TabsTrigger>
            <TabsTrigger value="contacts">{t(($) => $.tabs.contacts)}</TabsTrigger>
            <TabsTrigger value="profile">{t(($) => $.tabs.profile)}</TabsTrigger>
            <TabsTrigger value="projects">{t(($) => $.tabs.projects)}</TabsTrigger>
            <TabsTrigger value="emails">{t(($) => $.tabs.emails)}{unreadEmailCount > 0 ? <Badge variant="default" className="ml-2 tabular-nums">{unreadEmailCount}</Badge> : null}</TabsTrigger>
            <TabsTrigger value="notes">{t(($) => $.tabs.notes)}</TabsTrigger>
          </TabsList>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto p-6">
          <TabsContent value="overview" className="space-y-6">
            <section className="rounded-lg border bg-card p-4">
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                <FieldRow label={t(($) => $.customers.account_code)} value={account.account_code} />
                <FieldRow label={t(($) => $.customers.status)} value={t(($) => $.statuses[account.status])} />
                <FieldRow label={t(($) => $.customers.account_type)} value={t(($) => $.account_types[account.account_type])} />
                <FieldRow label={t(($) => $.customers.source)} value={account.source ? t(($) => $.sources[account.source as CRMAccountSource]) : null} />
                <FieldRow label={t(($) => $.customers.rating)} value={t(($) => $.ratings[account.rating])} />
                <FieldRow label={t(($) => $.customers.priority)} value={t(($) => $.priorities[account.priority])} />
                <FieldRow label={t(($) => $.customers.owner)} value={ownerLabel} />
                <FieldRow label={t(($) => $.customers.website)} value={account.website} />
                <FieldRow label={t(($) => $.customers.country)} value={countryName(account.country_code || account.country_name || account.country, locale)} />
                <FieldRow label={t(($) => $.customers.region)} value={account.region} />
                <FieldRow label={t(($) => $.customers.city)} value={account.city} />
                <FieldRow label={t(($) => $.customers.industry)} value={account.industry ? industryLabel(account.industry, locale) : null} />
                <FieldRow label={t(($) => $.customers.sub_industry)} value={account.sub_industry} />
                <FieldRow label={t(($) => $.customers.annual_revenue)} value={account.annual_revenue} />
                <FieldRow label={t(($) => $.customers.employee_count)} value={account.employee_count} />
                <FieldRow label={t(($) => $.customers.tags)} value={account.tags?.join(", ")} />
                <FieldRow label={t(($) => $.customers.last_contacted_at)} value={account.last_contacted_at ? new Date(account.last_contacted_at).toLocaleString() : null} />
                <FieldRow label={t(($) => $.customers.next_follow_up_at)} value={account.next_follow_up_at ? new Date(account.next_follow_up_at).toLocaleString() : null} />
                <FieldRow label="Created" value={account.created_at ? new Date(account.created_at).toLocaleString() : null} />
                <FieldRow label="Updated" value={account.updated_at ? new Date(account.updated_at).toLocaleString() : null} />
              </div>
              {account.notes && <p className="mt-4 whitespace-pre-wrap rounded-md border bg-background/50 p-3 text-sm">{account.notes}</p>}
            </section>
          </TabsContent>

          <TabsContent value="contacts" className="space-y-4">
            <div className="flex justify-end">
              <Button size="sm" onClick={() => setContactForm(blankContactForm())}>
                <Plus className="mr-1 size-4" /> {t(($) => $.contacts.add)}
              </Button>
            </div>
            <section className="rounded-lg border bg-card">
              {contactsLoading ? (
                <div className="space-y-2 p-4"><Skeleton className="h-12 w-full" /><Skeleton className="h-12 w-full" /></div>
              ) : contacts.length === 0 ? (
                <div className="p-10 text-center text-sm text-muted-foreground">{t(($) => $.contacts.empty)}</div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t(($) => $.contacts.name)}</TableHead>
                      <TableHead>{t(($) => $.contacts.job_title)}</TableHead>
                      <TableHead>{t(($) => $.contacts.email)}</TableHead>
                      <TableHead>{t(($) => $.contacts.whatsapp)}</TableHead>
                      <TableHead>{t(($) => $.contacts.is_primary)}</TableHead>
                      <TableHead></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {contacts.map((contact) => (
                      <TableRow key={contact.id}>
                        <TableCell className="font-medium">{contact.name}</TableCell>
                        <TableCell>{contact.job_title || contact.role_title || "—"}</TableCell>
                        <TableCell>{contact.email || "—"}</TableCell>
                        <TableCell>{contact.whatsapp || contact.whatsapp_id || "—"}</TableCell>
                        <TableCell>{contact.is_primary ? "✓" : "—"}</TableCell>
                        <TableCell className="text-right">
                          <Button size="sm" variant="ghost" aria-label={t(($) => $.actions.edit)} onClick={() => setContactForm(contactToForm(contact))}><Pencil className="size-4" /></Button>
                          <Button size="sm" variant="ghost" aria-label={t(($) => $.actions.delete)} disabled={deleteContact.isPending} onClick={() => window.confirm(t(($) => $.contacts.delete_confirm)) && deleteContact.mutate(contact.id)}><Trash2 className="size-4" /></Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </section>
          </TabsContent>

          <TabsContent value="profile" className="space-y-6">
            <section className="rounded-lg border bg-card p-4">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <h3 className="text-sm font-medium">{t(($) => $.profile.title)}</h3>
                  <p className="mt-1 text-xs text-muted-foreground">{t(($) => $.profile.help)}</p>
                </div>
                <div className="flex gap-2">
                  <Button size="sm" variant="outline" disabled={refreshProfile.isPending} onClick={() => refreshProfile.mutate()}>
                    {refreshProfile.isPending ? "画像刷新中…" : t(($) => $.profile.refresh)}
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => setProfileForm(profileToForm(profile))}>
                    <Pencil className="mr-1 size-4" /> {profile ? t(($) => $.profile.edit) : t(($) => $.profile.add)}
                  </Button>
                </div>
              </div>
              {profileLoading ? (
                <div className="mt-4 space-y-2"><Skeleton className="h-16 w-full" /><Skeleton className="h-16 w-full" /></div>
              ) : profile ? (
                <div className="mt-4 space-y-4">
                  <FieldRow label={t(($) => $.profile.summary_title)} value={profile.summary} />
                  <FieldRow label={t(($) => $.profile.source_summary)} value={profile.source_summary} />
                  <FieldRow label={t(($) => $.profile.updated_at)} value={profile.updated_at ? new Date(profile.updated_at).toLocaleString() : null} />
                  <FieldRow label="当前下次跟进时间" value={account.next_follow_up_at ? new Date(account.next_follow_up_at).toLocaleString() : null} />
                  <FieldRow label="AI 建议下次跟进" value={profileFollowUpValue(profile.profile_json)} />
                  <div className="grid gap-3 sm:grid-cols-2">
                    <FieldRow label={t(($) => $.profile.business_model)} value={profileText(profile.profile_json, "business_model")} />
                    <FieldRow label={t(($) => $.profile.main_products)} value={profileText(profile.profile_json, "main_products")} />
                    <FieldRow label={t(($) => $.profile.procurement_needs)} value={profileText(profile.profile_json, "procurement_needs")} />
                    <FieldRow label={t(($) => $.profile.pain_points)} value={profileText(profile.profile_json, "pain_points")} />
                    <FieldRow label={t(($) => $.profile.decision_process)} value={profileText(profile.profile_json, "decision_process")} />
                    <FieldRow label={t(($) => $.profile.communication_preference)} value={profileText(profile.profile_json, "communication_preference")} />
                    <FieldRow label={t(($) => $.profile.risk_notes)} value={profileText(profile.profile_json, "risk_notes")} />
                    <FieldRow label={t(($) => $.profile.cooperation_history)} value={profileText(profile.profile_json, "cooperation_history")} />
                    <FieldRow label="Aliases / 别名" value={profileDisplayText(profile.profile_json, "aliases")} />
                    <FieldRow label="Search keywords / 检索关键词" value={profileDisplayText(profile.profile_json, "search_keywords")} />
                    <FieldRow label="Source refs / 来源依据" value={profileDisplayText(profile.profile_json, "source_refs")} />
                    <FieldRow label="Confidence / 置信度" value={profileDisplayText(profile.profile_json, "confidence")} />
                  </div>
                </div>
              ) : (
                <div className="mt-4 rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">{t(($) => $.profile.empty)}</div>
              )}
            </section>
          </TabsContent>

          <TabsContent value="projects" className="grid gap-6 lg:grid-cols-2">
            <section className="rounded-lg border bg-card p-4">
              <h3 className="text-sm font-medium">{t(($) => $.projects.link_title)}</h3>
              <p className="mt-1 text-xs text-muted-foreground">{t(($) => $.projects.selector_help)}</p>
              <div className="mt-4 space-y-3">
                <div className="max-h-56 overflow-y-auto rounded-md border bg-background">
                  {projects.length === 0 ? (
                    <div className="p-4 text-sm text-muted-foreground">{t(($) => $.projects.empty)}</div>
                  ) : projects.map((project) => {
                    const checked = selectedProjectIds.includes(project.id);
                    return (
                      <button
                        key={project.id}
                        type="button"
                        className="flex w-full items-center gap-3 border-b px-3 py-2 text-left text-sm last:border-b-0 hover:bg-muted/60"
                        onClick={() => handleProjectToggle(project)}
                      >
                        <Checkbox checked={checked} className="pointer-events-none" aria-label={project.title} />
                        <span className="min-w-0 flex-1 truncate">{project.title}</span>
                        {projectLinkedToAccount(project, accountId) && <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">{t(($) => $.projects.linked)}</span>}
                      </button>
                    );
                  })}
                </div>
                <p className="text-xs text-muted-foreground">{t(($) => $.projects.selected_count, { count: selectedProjectIds.length })}</p>
                <Button className="w-full" disabled={linkProject.isPending} onClick={openCreateLinkedProject}><Plus className="mr-1 size-4" /> {t(($) => $.projects.create_linked_project)}</Button>
                {linkProject.isError && <p className="text-xs text-destructive">{t(($) => $.projects.link_error)}</p>}
              </div>
            </section>
            <section className="rounded-lg border bg-card p-4">
              <h3 className="text-sm font-medium">{t(($) => $.projects.follow_up_title)}</h3>
              <div className="mt-4 space-y-3">
                <select className="h-9 w-full rounded-md border bg-background px-3 text-sm" aria-label={t(($) => $.projects.select)} value={selectedFollowUpProjectId} onChange={(e) => setSelectedFollowUpProjectId(e.target.value)}>
                  <option value="">{t(($) => $.projects.select)}</option>
                  {linkedProjects.map((project) => <option key={project.id} value={project.id}>{project.title}</option>)}
                </select>
                {selectedFollowUpProjectId && (
                  <div className="rounded-md border bg-background/50 p-3">
                    <div className="text-xs font-medium text-muted-foreground">{t(($) => $.projects.existing_issues)}</div>
                    <div className="mt-2 space-y-1">
                      {followUpIssues.length === 0 ? (
                        <p className="text-sm text-muted-foreground">{t(($) => $.projects.no_issues)}</p>
                      ) : followUpIssues.map((issue) => (
                        <p key={issue.id} className="truncate text-sm">{formatIssueOption(issue)}</p>
                      ))}
                    </div>
                  </div>
                )}
                <Button className="w-full" onClick={openCreateFollowUpIssue}>{t(($) => $.projects.create_follow_up)}</Button>
              </div>
            </section>
          </TabsContent>

          <TabsContent value="emails" className="space-y-6">
            <section className="rounded-lg border bg-card">
              {emailThreadsLoading ? <div className="space-y-2 p-4"><Skeleton className="h-16 w-full" /><Skeleton className="h-16 w-full" /></div> : accountEmailItems.length === 0 ? <div className="p-10 text-center text-sm text-muted-foreground">{t(($) => $.emails.account_empty)}</div> : (
                <div className="divide-y">
                  {accountEmailItems.map((item: any) => {
                    const threadId = item.thread_id ?? item.id;
                    const folder = item.folder || (item.direction === "outbound" ? "sent" : "inbox");
                    const when = item.received_at || item.sent_at || item.last_message_at || item.updated_at;
                    const sender = item.direction === "outbound" ? (item.to_emails || []).join(", ") : [item.from_name, item.from_email].filter(Boolean).join(" <").replace(/ <$/, "");
                    return (
                      <button key={item.id} type="button" className="block w-full px-4 py-3 text-left text-sm hover:bg-muted/50" onClick={() => window.open(`${paths.crmEmails()}?thread=${encodeURIComponent(threadId)}&folder=${encodeURIComponent(folder)}`, "_blank", "noopener,noreferrer") }>
                        <div className="flex items-center justify-between gap-3">
                          <div className="min-w-0 truncate font-medium">{item.subject || t(($) => $.notes.untitled)}</div>
                          <Badge variant={item.direction === "outbound" ? "secondary" : "default"}>{item.direction === "outbound" ? "发件箱" : "收件箱"}</Badge>
                        </div>
                        <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">{emailSummaryText(item) || "—"}</div>
                        <div className="mt-2 text-xs text-muted-foreground">{[item.mailbox, sender, when ? new Date(when).toLocaleString() : "", item.thread_message_count ? t(($) => $.common.count_messages, { count: item.thread_message_count }) : null].filter(Boolean).join(" · ")}</div>
                      </button>
                    );
                  })}
                </div>
              )}
            </section>
          </TabsContent>

          <TabsContent value="notes" className="space-y-6">
            <section className="rounded-lg border bg-card">
              <div className="border-b p-4">
                <div className="grid gap-3 sm:grid-cols-3">
                  <label className="space-y-1 text-sm">
                    <span className="text-xs font-medium text-muted-foreground">{t(($) => $.notes.channel)}</span>
                    <select className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={noteForm.channel} onChange={(event) => setNoteForm((current) => ({ ...current, channel: event.target.value as CRMCommunicationChannel }))}>
                      {(["manual", "email", "whatsapp", "phone", "meeting", "other"] as CRMCommunicationChannel[]).map((channel) => <option key={channel} value={channel}>{t(($) => $.channels[channel])}</option>)}
                    </select>
                  </label>
                  <label className="space-y-1 text-sm">
                    <span className="text-xs font-medium text-muted-foreground">{t(($) => $.notes.direction)}</span>
                    <select className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={noteForm.direction} onChange={(event) => setNoteForm((current) => ({ ...current, direction: event.target.value as CRMCommunicationDirection }))}>
                      {(["note", "inbound", "outbound"] as CRMCommunicationDirection[]).map((direction) => <option key={direction} value={direction}>{t(($) => $.directions[direction])}</option>)}
                    </select>
                  </label>
                  <label className="space-y-1 text-sm">
                    <span className="text-xs font-medium text-muted-foreground">{t(($) => $.notes.subject)}</span>
                    <Input value={noteForm.subject} onChange={(event) => setNoteForm((current) => ({ ...current, subject: event.target.value }))} placeholder={t(($) => $.notes.subject_placeholder)} />
                  </label>
                </div>
                <Textarea className="mt-3 min-h-24" value={noteForm.body} onChange={(event) => setNoteForm((current) => ({ ...current, body: event.target.value }))} placeholder={t(($) => $.notes.placeholder)} />
                <div className="mt-2 flex justify-end"><Button size="sm" disabled={!noteForm.body.trim() || createNote.isPending} onClick={() => createNote.mutate()}><Plus className="mr-1 size-4" /> {t(($) => $.notes.add)}</Button></div>
              </div>
              {notesLoading ? <div className="space-y-2 p-4"><Skeleton className="h-16 w-full" /><Skeleton className="h-16 w-full" /></div> : notes.length === 0 ? <div className="p-8 text-center text-sm text-muted-foreground">{t(($) => $.notes.empty)}</div> : <div className="divide-y">{notes.map((note) => <div key={note.id} className="px-4 py-3 text-sm"><div className="font-medium">{note.subject || t(($) => $.notes.untitled)}</div><div className="mt-1 whitespace-pre-wrap">{note.body}</div><div className="mt-2 text-xs text-muted-foreground"><ChannelLabel channel={note.channel} t={t} /> · {t(($) => $.directions[note.direction])} · {new Date(note.occurred_at).toLocaleString()}</div></div>)}</div>}
            </section>
          </TabsContent>
        </div>
      </Tabs>

      <Dialog open={accountForm !== null} onOpenChange={(open) => !open && setAccountForm(null)}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader><DialogTitle>{t(($) => $.customers.edit_customer)}</DialogTitle><DialogDescription>{account.name}</DialogDescription></DialogHeader>
          {accountForm && <AccountForm form={accountForm} setForm={setAccountForm} t={t} locale={locale} suggestedTags={tagSuggestions(account)} members={members} agents={agents} />}
          {updateAccount.isError && <p className="text-xs text-destructive">{t(($) => $.customers.create_error)}</p>}
          <DialogFooter><Button variant="outline" onClick={() => setAccountForm(null)}>{t(($) => $.actions.cancel)}</Button><Button disabled={!accountForm?.name.trim() || updateAccount.isPending} onClick={() => updateAccount.mutate()}>{t(($) => $.actions.save)}</Button></DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={profileForm !== null} onOpenChange={(open) => !open && setProfileForm(null)}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader><DialogTitle>{t(($) => $.profile.edit_title)}</DialogTitle><DialogDescription>{account.name}</DialogDescription></DialogHeader>
          {profileForm && (
            <div className="max-h-[65vh] space-y-4 overflow-y-auto pr-1">
              <ProfileTextarea label={t(($) => $.profile.summary_title)} value={profileForm.summary} onChange={(value) => setProfileForm((current) => current && ({ ...current, summary: value }))} />
              <div className="grid gap-4 sm:grid-cols-2">
                <ProfileTextarea label={t(($) => $.profile.business_model)} value={profileForm.businessModel} onChange={(value) => setProfileForm((current) => current && ({ ...current, businessModel: value }))} />
                <ProfileTextarea label={t(($) => $.profile.main_products)} value={profileForm.mainProducts} onChange={(value) => setProfileForm((current) => current && ({ ...current, mainProducts: value }))} />
                <ProfileTextarea label={t(($) => $.profile.procurement_needs)} value={profileForm.procurementNeeds} onChange={(value) => setProfileForm((current) => current && ({ ...current, procurementNeeds: value }))} />
                <ProfileTextarea label={t(($) => $.profile.pain_points)} value={profileForm.painPoints} onChange={(value) => setProfileForm((current) => current && ({ ...current, painPoints: value }))} />
                <ProfileTextarea label={t(($) => $.profile.decision_process)} value={profileForm.decisionProcess} onChange={(value) => setProfileForm((current) => current && ({ ...current, decisionProcess: value }))} />
                <ProfileTextarea label={t(($) => $.profile.communication_preference)} value={profileForm.communicationPreference} onChange={(value) => setProfileForm((current) => current && ({ ...current, communicationPreference: value }))} />
                <ProfileTextarea label={t(($) => $.profile.risk_notes)} value={profileForm.riskNotes} onChange={(value) => setProfileForm((current) => current && ({ ...current, riskNotes: value }))} />
                <ProfileTextarea label={t(($) => $.profile.cooperation_history)} value={profileForm.cooperationHistory} onChange={(value) => setProfileForm((current) => current && ({ ...current, cooperationHistory: value }))} />
                <ProfileTextarea label="Aliases / 别名" value={profileForm.aliases} onChange={(value) => setProfileForm((current) => current && ({ ...current, aliases: value }))} />
                <ProfileTextarea label="Search keywords / 检索关键词" value={profileForm.searchKeywords} onChange={(value) => setProfileForm((current) => current && ({ ...current, searchKeywords: value }))} />
              </div>
            </div>
          )}
          {saveProfile.isError && <p className="text-xs text-destructive">{t(($) => $.profile.save_error)}</p>}
          <DialogFooter><Button variant="outline" onClick={() => setProfileForm(null)}>{t(($) => $.actions.cancel)}</Button><Button disabled={saveProfile.isPending} onClick={() => saveProfile.mutate()}>{t(($) => $.actions.save)}</Button></DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={contactForm !== null} onOpenChange={(open) => !open && setContactForm(null)}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader><DialogTitle>{contactForm?.id ? t(($) => $.contacts.edit_title) : t(($) => $.contacts.add_title)}</DialogTitle><DialogDescription>{account.name}</DialogDescription></DialogHeader>
          {contactForm && <ContactForm form={contactForm} setForm={setContactForm} t={t} />}
          {saveContact.isError && <p className="text-xs text-destructive">{t(($) => $.contacts.create_error)}</p>}
          <DialogFooter><Button variant="outline" onClick={() => setContactForm(null)}>{t(($) => $.actions.cancel)}</Button><Button disabled={!contactForm?.name.trim() || saveContact.isPending} onClick={() => saveContact.mutate()}>{t(($) => $.actions.save)}</Button></DialogFooter>
        </DialogContent>
      </Dialog>

      {createLinkedProjectOpen && account && (
        <CreateProjectModal
          onClose={() => setCreateLinkedProjectOpen(false)}
          navigateOnCreate={false}
          onCreated={handleLinkedProjectCreated}
          initialResources={[{
            resource_type: "crm_account",
            resource_ref: { account_id: account.id, name: account.name },
            label: account.name,
          }]}
        />
      )}
    </div>
  );
}
