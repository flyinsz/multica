"use client";

import { useMemo, useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Building2, Plus, Search } from "lucide-react";
import { api } from "@multica/core/api";
import { useWorkspaceId } from "@multica/core/hooks";
import { crmAccountListOptions, crmKeys } from "@multica/core/crm/queries";
import { useCRMWorkspacePaths } from "@multica/core/crm/paths";
import { useNavigation } from "../../navigation";
import type { CRMAccountFollowUpBucket, CRMAccountPriority, CRMAccountRating, CRMAccountSort, CRMAccountSource, CRMAccountStatus, CRMAccountType, ListCRMAccountsParams } from "@multica/core/types";
import { Button } from "@multica/ui/components/ui/button";
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@multica/ui/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@multica/ui/components/ui/table";
import { useT } from "../../i18n";
import type crmResources from "../../locales/en/crm.json";
import { PageHeader } from "../../layout/page-header";
import { COUNTRY_OPTIONS, countryByCode, loadCityOptions, loadRegionOptions, localizedName, localizedSort, normalizeLocale, useLocationSelection } from "../geo";
import { appendTag, CRM_INDUSTRY_OPTIONS, formatDateTimeLocal, industryLabel, optionLabel, splitTags, subIndustryOptions } from "../options";
import { crmApi } from "@multica/core/crm/api";

type CRMResources = typeof crmResources;
type Translation = (
  selector: (resources: CRMResources) => string,
  options?: Record<string, unknown>,
) => string;

type AccountOwnerType = "" | "member" | "agent";

type AccountFormState = {
  name: string;
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

const blankAccountForm = (): AccountFormState => ({
  name: "",
  accountType: "prospect",
  status: "active",
  source: "manual",
  rating: "unknown",
  priority: "medium",
  ownerType: "",
  ownerMemberID: "",
  ownerAgentID: "",
  website: "",
  countryCode: "",
  regionCode: "",
  cityCode: "",
  industry: "",
  subIndustry: "",
  annualRevenue: "",
  employeeCount: "",
  tags: "",
  nextFollowUpAt: formatDateTimeLocal(),
  notes: "",
});

const tagSuggestions = (accounts: Array<{ tags?: string[] | null }>) => {
  const counts = new Map<string, number>();
  accounts.forEach((account) => {
    account.tags?.forEach((tag) => {
      const normalized = tag.trim();
      if (!normalized) return;
      counts.set(normalized, (counts.get(normalized) ?? 0) + 1);
    });
  });
  return [...counts.entries()]
    .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
    .map(([tag]) => tag)
    .slice(0, 12);
};


function AccountStatusLabel({ status, t }: { status: CRMAccountStatus; t: Translation }) {
  const labels: Record<CRMAccountStatus, string> = {
    active: t(($) => $.statuses.active),
    inactive: t(($) => $.statuses.inactive),
    prospect: t(($) => $.statuses.prospect),
    archived: t(($) => $.statuses.archived),
  };
  return labels[status] ?? status;
}

function AccountTypeLabel({ type, t }: { type: CRMAccountType; t: Translation }) {
  const labels: Record<CRMAccountType, string> = {
    prospect: t(($) => $.account_types.prospect),
    customer: t(($) => $.account_types.customer),
    partner: t(($) => $.account_types.partner),
    supplier: t(($) => $.account_types.supplier),
    competitor: t(($) => $.account_types.competitor),
    other: t(($) => $.account_types.other),
  };
  return labels[type] ?? type;
}


const accountTypeLabel = (t: Translation, value: CRMAccountType) => t(($) => $.account_types[value] ?? value);
const statusLabel = (t: Translation, value: CRMAccountStatus) => t(($) => $.statuses[value] ?? value);
const sourceLabel = (t: Translation, value: CRMAccountSource) => t(($) => $.sources[value] ?? value);
const ratingLabel = (t: Translation, value: CRMAccountRating) => t(($) => $.ratings[value] ?? value);
const priorityLabel = (t: Translation, value: CRMAccountPriority) => t(($) => $.priorities[value] ?? value);
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

function AccountForm({
  form,
  setForm,
  t,
  locale,
  suggestedTags,
  members,
  agents,
}: {
  form: AccountFormState;
  setForm: (next: AccountFormState) => void;
  t: Translation;
  locale: "en" | "zh-Hans";
  suggestedTags: string[];
  members: Array<{ id: string; user_id: string; name: string; email: string; user?: { name?: string; email?: string } }>;
  agents: Array<{ id: string; name: string }>;
}) {
  const { regions, cities, regionsLoading, citiesLoading } = useLocationSelection(form.countryCode, form.regionCode, locale);
  const countries = useMemo(() => localizedSort(COUNTRY_OPTIONS, locale), [locale]);
  const subIndustries = subIndustryOptions(form.industry);

  return (
    <div className="grid max-h-[70vh] gap-4 overflow-y-auto rounded-lg border bg-muted/20 p-4 sm:grid-cols-2">
      <LabeledField label={t(($) => $.customers.new_customer_name)}><Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder={t(($) => $.customers.new_customer_name)} /></LabeledField>
      <LabeledField label={t(($) => $.customers.website)}><Input value={form.website} onChange={(e) => setForm({ ...form, website: e.target.value })} placeholder={t(($) => $.customers.website)} /></LabeledField>
      <LabeledField label={t(($) => $.customers.account_type)}><Select value={form.accountType} onValueChange={(value) => setForm({ ...form, accountType: value as CRMAccountType })}><SelectTrigger className="w-full"><SelectValue>{() => accountTypeLabel(t, form.accountType)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="prospect">{t(($) => $.account_types.prospect)}</SelectItem><SelectItem value="customer">{t(($) => $.account_types.customer)}</SelectItem><SelectItem value="partner">{t(($) => $.account_types.partner)}</SelectItem><SelectItem value="supplier">{t(($) => $.account_types.supplier)}</SelectItem><SelectItem value="competitor">{t(($) => $.account_types.competitor)}</SelectItem><SelectItem value="other">{t(($) => $.account_types.other)}</SelectItem></SelectContent></Select></LabeledField>
      <LabeledField label={t(($) => $.customers.status)}><Select value={form.status} onValueChange={(value) => setForm({ ...form, status: value as CRMAccountStatus })}><SelectTrigger className="w-full"><SelectValue>{() => statusLabel(t, form.status)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="active">{t(($) => $.statuses.active)}</SelectItem><SelectItem value="prospect">{t(($) => $.statuses.prospect)}</SelectItem><SelectItem value="inactive">{t(($) => $.statuses.inactive)}</SelectItem><SelectItem value="archived">{t(($) => $.statuses.archived)}</SelectItem></SelectContent></Select></LabeledField>
      <LabeledField label={t(($) => $.customers.source)}><Select value={form.source} onValueChange={(value) => setForm({ ...form, source: value as CRMAccountSource })}><SelectTrigger className="w-full"><SelectValue>{() => sourceLabel(t, form.source)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="manual">{t(($) => $.sources.manual)}</SelectItem><SelectItem value="email">{t(($) => $.sources.email)}</SelectItem><SelectItem value="whatsapp">{t(($) => $.sources.whatsapp)}</SelectItem><SelectItem value="website">{t(($) => $.sources.website)}</SelectItem><SelectItem value="referral">{t(($) => $.sources.referral)}</SelectItem><SelectItem value="trade_show">{t(($) => $.sources.trade_show)}</SelectItem><SelectItem value="linkedin">{t(($) => $.sources.linkedin)}</SelectItem><SelectItem value="other">{t(($) => $.sources.other)}</SelectItem></SelectContent></Select></LabeledField>
      <LabeledField label={t(($) => $.customers.rating)}><Select value={form.rating} onValueChange={(value) => setForm({ ...form, rating: value as CRMAccountRating })}><SelectTrigger className="w-full"><SelectValue>{() => ratingLabel(t, form.rating)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="unknown">{t(($) => $.ratings.unknown)}</SelectItem><SelectItem value="hot">{t(($) => $.ratings.hot)}</SelectItem><SelectItem value="warm">{t(($) => $.ratings.warm)}</SelectItem><SelectItem value="cold">{t(($) => $.ratings.cold)}</SelectItem></SelectContent></Select></LabeledField>
      <LabeledField label={t(($) => $.customers.priority)}><Select value={form.priority} onValueChange={(value) => setForm({ ...form, priority: value as CRMAccountPriority })}><SelectTrigger className="w-full"><SelectValue>{() => priorityLabel(t, form.priority)}</SelectValue></SelectTrigger><SelectContent><SelectItem value="medium">{t(($) => $.priorities.medium)}</SelectItem><SelectItem value="high">{t(($) => $.priorities.high)}</SelectItem><SelectItem value="low">{t(($) => $.priorities.low)}</SelectItem></SelectContent></Select></LabeledField>
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

export function CRMPage() {
  const wsId = useWorkspaceId();
  const queryClient = useQueryClient();
  const navigation = useNavigation();
  const paths = useCRMWorkspacePaths();
  const { t: rawT, i18n } = useT("crm");
  const t = rawT as Translation;
  const locale = normalizeLocale(i18n.language);
  const searchParams = navigation.searchParams;
  const [search, setSearch] = useState(() => searchParams.get("search") ?? "");
  const [statusFilter, setStatusFilter] = useState<CRMAccountStatus | "">(() => (searchParams.get("status") as CRMAccountStatus | null) ?? "");
  const [ratingFilter, setRatingFilter] = useState<CRMAccountRating | "">(() => (searchParams.get("rating") as CRMAccountRating | null) ?? "");
  const [priorityFilter, setPriorityFilter] = useState<CRMAccountPriority | "">(() => (searchParams.get("priority") as CRMAccountPriority | null) ?? "");
  const [countryFilter, setCountryFilter] = useState(() => searchParams.get("country") ?? "");
  const [industryFilter, setIndustryFilter] = useState(() => searchParams.get("industry") ?? "");
  const [sourceFilter, setSourceFilter] = useState<CRMAccountSource | "">(() => (searchParams.get("source") as CRMAccountSource | null) ?? "");
  const [followUpBucket, setFollowUpBucket] = useState<CRMAccountFollowUpBucket | "">(() => (searchParams.get("follow_up") as CRMAccountFollowUpBucket | null) ?? (searchParams.get("follow_up_bucket") as CRMAccountFollowUpBucket | null) ?? "");
  const [sort, setSort] = useState<CRMAccountSort>(() => (searchParams.get("sort") as CRMAccountSort | null) ?? "name");
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState<AccountFormState>(() => blankAccountForm());

  const accountListParams = useMemo<ListCRMAccountsParams>(() => ({
    search,
    status: statusFilter,
    rating: ratingFilter,
    priority: priorityFilter,
    country_code: countryFilter,
    industry: industryFilter,
    source: sourceFilter,
    follow_up_bucket: followUpBucket,
    sort,
  }), [countryFilter, followUpBucket, industryFilter, priorityFilter, ratingFilter, search, sort, sourceFilter, statusFilter]);
  const { data: accounts = [], isLoading } = useQuery(crmAccountListOptions(wsId, accountListParams));
  const { data: members = [] } = useQuery({ queryKey: ["workspace", wsId, "members", "crm-accounts"], queryFn: () => api.listMembers(wsId), enabled: Boolean(wsId) });
  const { data: agents = [] } = useQuery({ queryKey: ["agents", wsId, "crm-accounts"], queryFn: () => api.listAgents({ workspace_id: wsId }), enabled: Boolean(wsId) });
  const sortedAccounts = useMemo(
    () => sort === "name" ? [...accounts].sort((a, b) => a.name.localeCompare(b.name, locale === "zh-Hans" ? "zh-Hans-CN-u-co-pinyin" : "en")) : accounts,
    [accounts, locale, sort],
  );
  const suggestedTags = useMemo(() => tagSuggestions(accounts), [accounts]);

  const createAccount = useMutation({
    mutationFn: async () => {
      const country = countryByCode(form.countryCode);
      const regions = await loadRegionOptions(form.countryCode, locale);
      const region = regions.find((option) => option.code === form.regionCode);
      const cities = await loadCityOptions(form.countryCode, form.regionCode, locale);
      const city = cities.find((option) => option.code === form.cityCode);
      return crmApi.createCRMAccount({
        name: form.name,
        account_type: form.accountType,
        website: form.website || null,
        country: form.countryCode || null,
        country_code: form.countryCode || null,
        country_name: country ? localizedName(country.name, locale) : null,
        region: region ? localizedName(region.name, locale) : null,
        city: city ? localizedName(city.name, locale) : null,
        industry: form.industry || null,
        sub_industry: form.subIndustry || null,
        status: form.status,
        source: form.source,
        rating: form.rating,
        priority: form.priority,
        owner_type: form.ownerType || null,
        owner_member_id: form.ownerType === "member" ? form.ownerMemberID || null : null,
        owner_agent_id: form.ownerType === "agent" ? form.ownerAgentID || null : null,
        annual_revenue: form.annualRevenue || null,
        employee_count: form.employeeCount || null,
        tags: splitTags(form.tags),
        next_follow_up_at: form.nextFollowUpAt ? new Date(form.nextFollowUpAt).toISOString() : null,
        notes: form.notes || null,
      });
    },
    onSuccess: async (account) => {
      setForm(blankAccountForm());
      setCreateOpen(false);
      await queryClient.invalidateQueries({ queryKey: crmKeys.accounts(wsId) });
      navigation.push(paths.customerDetail(account.id));
    },
  });

  return (
    <div className="flex h-full flex-col">
      <PageHeader className="justify-between px-5">
        <div className="flex items-center gap-2">
          <Building2 className="size-4 text-muted-foreground" />
          <h1 className="text-sm font-medium">{t(($) => $.customers.title)}</h1>
          {!isLoading && <span className="text-xs text-muted-foreground tabular-nums">{accounts.length}</span>}
        </div>
        <Button size="sm" onClick={() => setCreateOpen(true)}>
          <Plus className="mr-1 size-4" /> {t(($) => $.customers.add_customer)}
        </Button>
      </PageHeader>

      <div className="space-y-4 p-5">
        <div className="grid gap-2 lg:grid-cols-[minmax(220px,1fr)_repeat(8,minmax(130px,160px))]">
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
            <Input className="pl-8" value={search} onChange={(e) => setSearch(e.target.value)} placeholder={t(($) => $.customers.search_placeholder)} />
          </div>
          <select aria-label={t(($) => $.filters.status)} className="h-9 rounded-md border bg-background px-3 text-sm" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value as CRMAccountStatus | "")}>
            <option value="">{t(($) => $.filters.all_statuses)}</option>
            <option value="active">{t(($) => $.statuses.active)}</option>
            <option value="prospect">{t(($) => $.statuses.prospect)}</option>
            <option value="inactive">{t(($) => $.statuses.inactive)}</option>
            <option value="archived">{t(($) => $.statuses.archived)}</option>
          </select>
          <select aria-label={t(($) => $.filters.rating)} className="h-9 rounded-md border bg-background px-3 text-sm" value={ratingFilter} onChange={(e) => setRatingFilter(e.target.value as CRMAccountRating | "")}>
            <option value="">{t(($) => $.filters.all_ratings)}</option>
            <option value="hot">{t(($) => $.ratings.hot)}</option>
            <option value="warm">{t(($) => $.ratings.warm)}</option>
            <option value="cold">{t(($) => $.ratings.cold)}</option>
            <option value="unknown">{t(($) => $.ratings.unknown)}</option>
          </select>
          <select aria-label={t(($) => $.filters.priority)} className="h-9 rounded-md border bg-background px-3 text-sm" value={priorityFilter} onChange={(e) => setPriorityFilter(e.target.value as CRMAccountPriority | "")}>
            <option value="">{t(($) => $.filters.all_priorities)}</option>
            <option value="high">{t(($) => $.priorities.high)}</option>
            <option value="medium">{t(($) => $.priorities.medium)}</option>
            <option value="low">{t(($) => $.priorities.low)}</option>
          </select>
          <select aria-label={t(($) => $.filters.country)} className="h-9 rounded-md border bg-background px-3 text-sm" value={countryFilter} onChange={(e) => setCountryFilter(e.target.value)}>
            <option value="">{t(($) => $.filters.all_countries)}</option>
            {COUNTRY_OPTIONS.map((country) => <option key={country.code} value={country.code}>{localizedName(country.name, locale)}</option>)}
          </select>
          <select aria-label={t(($) => $.filters.industry)} className="h-9 rounded-md border bg-background px-3 text-sm" value={industryFilter} onChange={(e) => setIndustryFilter(e.target.value)}>
            <option value="">{t(($) => $.filters.all_industries)}</option>
            {CRM_INDUSTRY_OPTIONS.map((industry) => <option key={industry.value} value={industry.value}>{industryLabel(industry.value, locale)}</option>)}
          </select>
          <select aria-label={t(($) => $.filters.source)} className="h-9 rounded-md border bg-background px-3 text-sm" value={sourceFilter} onChange={(e) => setSourceFilter(e.target.value as CRMAccountSource | "")}>
            <option value="">{t(($) => $.filters.all_sources)}</option>
            <option value="manual">{t(($) => $.sources.manual)}</option>
            <option value="email">{t(($) => $.sources.email)}</option>
            <option value="whatsapp">{t(($) => $.sources.whatsapp)}</option>
            <option value="website">{t(($) => $.sources.website)}</option>
            <option value="referral">{t(($) => $.sources.referral)}</option>
            <option value="trade_show">{t(($) => $.sources.trade_show)}</option>
            <option value="linkedin">{t(($) => $.sources.linkedin)}</option>
            <option value="other">{t(($) => $.sources.other)}</option>
          </select>
          <select aria-label={t(($) => $.filters.follow_up)} className="h-9 rounded-md border bg-background px-3 text-sm" value={followUpBucket} onChange={(e) => setFollowUpBucket(e.target.value as CRMAccountFollowUpBucket | "")}>
            <option value="">{t(($) => $.filters.any_follow_up)}</option>
            <option value="today">{t(($) => $.filters.follow_up_today)}</option>
            <option value="next_7_days">{t(($) => $.filters.follow_up_next_7_days)}</option>
            <option value="overdue">{t(($) => $.filters.follow_up_overdue)}</option>
            <option value="none">{t(($) => $.filters.follow_up_none)}</option>
          </select>
          <select aria-label={t(($) => $.filters.sort)} className="h-9 rounded-md border bg-background px-3 text-sm" value={sort} onChange={(e) => setSort(e.target.value as CRMAccountSort)}>
            <option value="name">{t(($) => $.filters.sort_name)}</option>
            <option value="updated">{t(($) => $.filters.sort_updated)}</option>
            <option value="next_follow_up">{t(($) => $.filters.sort_next_follow_up)}</option>
            <option value="priority_rating">{t(($) => $.filters.sort_priority_rating)}</option>
          </select>
        </div>

        <section className="rounded-lg border bg-card">
          {isLoading ? (
            <div className="space-y-2 p-4">
              {Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-12 w-full" />)}
            </div>
          ) : sortedAccounts.length === 0 ? (
            <div className="p-10 text-center text-sm text-muted-foreground">{t(($) => $.customers.empty)}</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t(($) => $.customers.title)}</TableHead>
                  <TableHead>{t(($) => $.customers.account_type)}</TableHead>
                  <TableHead>{t(($) => $.customers.status)}</TableHead>
                  <TableHead>{t(($) => $.customers.country)}</TableHead>
                  <TableHead>{t(($) => $.customers.industry)}</TableHead>
                  <TableHead>{t(($) => $.tabs.contacts)}</TableHead>
                  <TableHead>{t(($) => $.customers.next_follow_up_at)}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sortedAccounts.map((account) => (
                  <TableRow
                    key={account.id}
                    className="cursor-pointer"
                    onClick={() => navigation.push(paths.customerDetail(account.id))}
                  >
                    <TableCell className="font-medium">{account.name}</TableCell>
                    <TableCell><AccountTypeLabel type={account.account_type} t={t} /></TableCell>
                    <TableCell><AccountStatusLabel status={account.status} t={t} /></TableCell>
                    <TableCell>{account.country_code ? localizedName(countryByCode(account.country_code)?.name ?? { en: account.country_name || account.country_code, zh: account.country_name || account.country_code }, locale) : account.country_name || account.country || "—"}</TableCell>
                    <TableCell>{[account.industry, account.sub_industry].filter(Boolean).join(" · ") || "—"}</TableCell>
                    <TableCell>{account.contact_count}</TableCell>
                    <TableCell>{account.next_follow_up_at ? new Date(account.next_follow_up_at).toLocaleString() : "—"}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </section>
      </div>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>{t(($) => $.customers.add_customer)}</DialogTitle>
            <DialogDescription>{t(($) => $.customers.basic_profile)}</DialogDescription>
          </DialogHeader>
          <AccountForm form={form} setForm={setForm} t={t} locale={locale} suggestedTags={suggestedTags} members={members} agents={agents} />
          {createAccount.isError && <p className="text-xs text-destructive">{t(($) => $.customers.create_error)}</p>}
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>{t(($) => $.actions.cancel)}</Button>
            <Button disabled={!form.name.trim() || createAccount.isPending} onClick={() => createAccount.mutate()}>{t(($) => $.customers.add_customer)}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

/* crm image build trigger: account form select polish */
