"use client";

import { useMemo, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { Bot, Flame, Mail, PenLine, Plus, Sparkles, TrendingUp, Users } from "lucide-react";
import { crmAccountListOptions, crmEmailThreadListOptions } from "@multica/core/crm/queries";
import { useWorkspaceId } from "@multica/core/hooks";
import { useCRMWorkspacePaths } from "@multica/core/crm/paths";
import type { CRMAccount, CRMAccountFollowUpBucket, CRMAccountPriority, CRMAccountRating, CRMAccountStatus } from "@multica/core/types";
import { api } from "@multica/core/api";
import { Badge } from "@multica/ui/components/ui/badge";
import { Button } from "@multica/ui/components/ui/button";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { PageHeader } from "../../layout/page-header";
import { useT } from "../../i18n";
import { useNavigation } from "../../navigation";

function formatDate(value?: string | null) {
  return value ? new Date(value).toLocaleDateString() : "—";
}

type ReportFilter = Partial<Record<"status" | "rating" | "priority" | "country" | "industry" | "follow_up", string>>;
type ReportBucket = { key: string; label: string; value: number; filter: ReportFilter };

const statusOrder: CRMAccountStatus[] = ["prospect", "active", "inactive", "archived"];
const ratingOrder: CRMAccountRating[] = ["hot", "warm", "cold", "unknown"];
const priorityOrder: CRMAccountPriority[] = ["high", "medium", "low"];
const followUpOrder: CRMAccountFollowUpBucket[] = ["overdue", "today", "next_7_days", "none"];

function countBy<T extends string>(accounts: CRMAccount[], pick: (account: CRMAccount) => T | null | undefined) {
  return accounts.reduce((map, account) => {
    const key = pick(account);
    if (key) map.set(key, (map.get(key) ?? 0) + 1);
    return map;
  }, new Map<T, number>());
}

function followUpBucket(account: CRMAccount, now = new Date()): CRMAccountFollowUpBucket {
  if (!account.next_follow_up_at) return "none";
  const due = new Date(account.next_follow_up_at);
  if (Number.isNaN(due.getTime())) return "none";
  const todayEnd = new Date(now);
  todayEnd.setHours(23, 59, 59, 999);
  if (due < now) return "overdue";
  if (due <= todayEnd) return "today";
  const next7 = new Date(now);
  next7.setDate(next7.getDate() + 7);
  return due <= next7 ? "next_7_days" : "none";
}

function maxBucketValue(buckets: ReportBucket[]) {
  return Math.max(1, ...buckets.map((bucket) => bucket.value));
}

function ReportPanel({
  title,
  buckets,
  loading,
  onSelect,
  empty,
  icon,
}: {
  title: string;
  buckets: ReportBucket[];
  loading: boolean;
  onSelect: (filter: ReportFilter) => void;
  empty?: string;
  icon?: ReactNode;
}) {
  const max = maxBucketValue(buckets);
  return (
    <section className="rounded-lg border bg-card p-4">
      <div className="flex items-center justify-between text-sm font-medium">
        <h2>{title}</h2>
        {icon ? <span className="text-muted-foreground">{icon}</span> : null}
      </div>
      <div className="mt-3 space-y-2">
        {loading ? <Skeleton className="h-28" /> : buckets.length === 0 ? <p className="text-sm text-muted-foreground">{empty}</p> : buckets.map((bucket) => (
          <button key={bucket.key} type="button" className="grid w-full grid-cols-[minmax(90px,1fr)_3fr_3ch] items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs hover:bg-muted/50" onClick={() => onSelect(bucket.filter)}>
            <span className="truncate text-muted-foreground">{bucket.label}</span>
            <span className="h-2 rounded-full bg-muted"><span className="block h-2 rounded-full bg-primary/70" style={{ width: `${Math.max(5, (bucket.value / max) * 100)}%` }} /></span>
            <span className="text-right font-medium tabular-nums">{bucket.value}</span>
          </button>
        ))}
      </div>
    </section>
  );
}

export function CRMDashboardPage() {
  const wsId = useWorkspaceId();
  const paths = useCRMWorkspacePaths();
  const navigation = useNavigation();
  const { t } = useT("crm");
  const { data: todayFollowUps = [], isLoading: todayLoading } = useQuery(crmAccountListOptions(wsId, { follow_up_bucket: "today", sort: "next_follow_up" }));
  const { data: weekFollowUps = [], isLoading: weekLoading } = useQuery(crmAccountListOptions(wsId, { follow_up_bucket: "next_7_days", sort: "next_follow_up" }));
  const { data: overdueFollowUps = [], isLoading: overdueLoading } = useQuery(crmAccountListOptions(wsId, { follow_up_bucket: "overdue", sort: "next_follow_up" }));
  const { data: highPriorityAccounts = [], isLoading: highPriorityLoading } = useQuery(crmAccountListOptions(wsId, { priority: "high", sort: "priority_rating" }));
  const { data: hotAccounts = [] } = useQuery(crmAccountListOptions(wsId, { rating: "hot", sort: "priority_rating" }));
  const { data: allAccounts = [], isLoading: reportsLoading } = useQuery(crmAccountListOptions(wsId, { sort: "name" }));
  const { data: emailThreadData, isLoading: emailLoading } = useQuery(crmEmailThreadListOptions(wsId));
  const { data: aiSettingsData, isLoading: aiSettingsLoading } = useQuery({ queryKey: ["crm", "ai-settings"], queryFn: () => api.listCRMAISettings() });
  const { data: aiHistoryData, isLoading: aiHistoryLoading } = useQuery({ queryKey: ["crm", "ai-history", "dashboard"], queryFn: () => api.listCRMAIHistory({ limit: 8, days: 14 }) });
  const emailThreads = emailThreadData?.threads ?? [];
  const unreadEmailCount = (emailThreadData?.counts as any)?.inbox_unread ?? emailThreads.filter((thread: any) => thread.is_read !== true && thread.direction !== "outbound" && !thread.is_trashed).length;
  const topTodayFollowUps = todayFollowUps.slice(0, 6);
  const topWeekFollowUps = weekFollowUps.slice(0, 6);
  const topOverdueFollowUps = overdueFollowUps.slice(0, 6);
  const topHighPriorityAccounts = highPriorityAccounts.slice(0, 6);
  const emailMessages = emailThreadData?.messages ?? [];
  const topEmailMessages = emailMessages.slice(0, 6);
  const topEmailThreads = topEmailMessages.length ? topEmailMessages : emailThreads.slice(0, 6);
  const pendingReplyThreads = emailThreads.filter((thread: any) => thread.direction !== "outbound" && thread.is_read !== true && !thread.is_trashed);
  const aiSettings = (aiSettingsData?.settings ?? []) as Array<any>;
  const aiHistory = (aiHistoryData?.items ?? []) as Array<any>;
  const enabledAISettings = aiSettings.filter((setting) => setting.enabled !== false);
  const stats = useMemo(() => [
    { label: t(($) => $.dashboard.total_customers), value: allAccounts.length, icon: Users, filter: {} },
    { label: t(($) => $.dashboard.overdue_followups), value: overdueFollowUps.length, icon: Flame, filter: { follow_up: "overdue" } },
    { label: t(($) => $.dashboard.hot_customers), value: hotAccounts.length, icon: Flame, filter: { rating: "hot" } },
    { label: t(($) => $.dashboard.unread_emails), value: unreadEmailCount, icon: Mail, filter: null as ReportFilter | null | "ai" },
    { label: t(($) => $.dashboard.ai_automations), value: enabledAISettings.length, icon: Bot, filter: "ai" as const },
  ], [allAccounts.length, unreadEmailCount, overdueFollowUps.length, hotAccounts.length, enabledAISettings.length, t]);

  const navigateToCustomers = (filter: ReportFilter = {}) => {
    const params = new URLSearchParams();
    Object.entries(filter).forEach(([key, value]) => {
      if (value) params.set(key, value);
    });
    navigation.push(`${paths.customers()}${params.size ? `?${params.toString()}` : ""}`);
  };

  const openAIEmailComposer = () => navigation.push(`${paths.emails()}?compose=ai`);

  const reportGroups = useMemo(() => {
    const statuses = countBy(allAccounts, (account) => account.status);
    const ratings = countBy(allAccounts, (account) => account.rating);
    const priorities = countBy(allAccounts, (account) => account.priority);
    const followUpsByBucket = countBy(allAccounts, (account) => followUpBucket(account));
    return {
      funnel: ratingOrder.map((key) => ({ key, label: t(($) => $.ratings[key]), value: ratings.get(key) ?? 0, filter: { rating: key } })),
      status: statusOrder.map((key) => ({ key, label: t(($) => $.statuses[key]), value: statuses.get(key) ?? 0, filter: { status: key } })),
      priority: priorityOrder.map((key) => ({ key, label: t(($) => $.priorities[key]), value: priorities.get(key) ?? 0, filter: { priority: key } })),
      followUps: followUpOrder.map((key) => ({ key, label: t(($) => $.filters[`follow_up_${key}`]), value: followUpsByBucket.get(key) ?? 0, filter: { follow_up: key } })),
    };
  }, [allAccounts, t]);

  const accountList = (items: CRMAccount[], empty: string) => {
    if (items.length === 0) return <p className="text-sm text-muted-foreground">{empty}</p>;
    return <div className="space-y-2">{items.map((account) => (
      <button key={account.id} type="button" className="flex w-full items-center justify-between rounded-md border p-2 text-left text-sm hover:bg-muted/50" onClick={() => navigation.push(paths.customerDetail(account.id))}>
        <span className="truncate font-medium">{account.name}</span>
        <span className="ml-2 shrink-0 text-xs text-muted-foreground">{formatDate(account.next_follow_up_at || account.updated_at)}</span>
      </button>
    ))}</div>;
  };

  return (
    <div className="flex h-full min-h-0 flex-col">
      <PageHeader className="shrink-0 justify-between px-5">
        <div className="flex items-center gap-2">
          <Users className="size-4 text-muted-foreground" />
          <h1 className="text-sm font-medium">{t(($) => $.dashboard.title)}</h1>
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" onClick={openAIEmailComposer}><PenLine className="mr-1 size-4" />{t(($) => $.dashboard.ai_write_email)}</Button>
          <Button size="sm" variant="outline" onClick={() => navigation.push(paths.emails())}>{t(($) => $.tabs.emails)}{unreadEmailCount > 0 ? <Badge variant="default" className="ml-2 tabular-nums">{unreadEmailCount}</Badge> : null}</Button>
          <Button size="sm" onClick={() => navigation.push(paths.customers())}><Plus className="mr-1 size-4" />{t(($) => $.customers.title)}</Button>
        </div>
      </PageHeader>
      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-5 pb-11">
        <div className="grid gap-3 md:grid-cols-5">
          {stats.map(({ label, value, icon: Icon, filter }) => (
            <button key={label} type="button" className="rounded-lg border bg-card p-4 text-left transition hover:border-primary/40 hover:bg-muted/30" onClick={() => filter === "ai" ? navigation.push(paths.aiSettings()) : filter ? navigateToCustomers(filter) : navigation.push(paths.emails())}>
              <div className="flex items-center justify-between text-xs text-muted-foreground"><span>{label}</span><Icon className="size-4" /></div>
              <div className="mt-2 text-2xl font-semibold tabular-nums">{value}</div>
            </button>
          ))}
        </div>

        <section className="rounded-lg border bg-card p-4">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h2 className="text-sm font-medium">{t(($) => $.dashboard.pipeline_title)}</h2>
              <p className="mt-1 text-xs text-muted-foreground">{t(($) => $.dashboard.pipeline_help)}</p>
            </div>
            <TrendingUp className="size-4 text-muted-foreground" />
          </div>
          {reportsLoading ? <Skeleton className="mt-4 h-36" /> : (
            <div className="mt-4 grid gap-3 lg:grid-cols-4">
              {reportGroups.funnel.map((bucket, index) => {
                const width = `${Math.max(8, (bucket.value / maxBucketValue(reportGroups.funnel)) * 100)}%`;
                return (
                  <button key={bucket.key} type="button" className="rounded-lg border p-3 text-left hover:bg-muted/40" onClick={() => navigateToCustomers(bucket.filter)}>
                    <div className="flex items-center justify-between text-sm"><span>{bucket.label}</span><span className="font-semibold tabular-nums">{bucket.value}</span></div>
                    <div className="mt-3 h-2 rounded-full bg-muted"><div className="h-2 rounded-full bg-primary" style={{ width, opacity: 1 - index * 0.14 }} /></div>
                  </button>
                );
              })}
            </div>
          )}
        </section>

        <div className="grid gap-4 xl:grid-cols-3">
          <ReportPanel title={t(($) => $.dashboard.status_distribution)} icon={<TrendingUp className="size-4" />} buckets={reportGroups.status} loading={reportsLoading} onSelect={navigateToCustomers} />
          <ReportPanel title={t(($) => $.dashboard.priority_distribution)} buckets={reportGroups.priority} loading={reportsLoading} onSelect={navigateToCustomers} />
          <ReportPanel title={t(($) => $.dashboard.overdue_trend)} buckets={reportGroups.followUps} loading={reportsLoading} onSelect={navigateToCustomers} />
        </div>

        <section className="rounded-lg border bg-card p-4">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h2 className="flex items-center gap-2 text-sm font-medium"><Sparkles className="size-4 text-primary" />{t(($) => $.dashboard.ai_board_title)}</h2>
              <p className="mt-1 text-xs text-muted-foreground">{t(($) => $.dashboard.ai_board_help)}</p>
            </div>
            <Button size="sm" variant="outline" onClick={() => navigation.push(paths.aiSettings())}>{t(($) => $.dashboard.ai_settings)}</Button>
          </div>
          <div className="mt-4 grid gap-3 md:grid-cols-4">
            <button type="button" className="rounded-lg border p-3 text-left hover:bg-muted/40" onClick={() => navigation.push(paths.emails())}>
              <div className="text-xs text-muted-foreground">{t(($) => $.dashboard.ai_pending_replies)}</div>
              <div className="mt-2 text-2xl font-semibold tabular-nums">{pendingReplyThreads.length}</div>
            </button>
            <button type="button" className="rounded-lg border p-3 text-left hover:bg-muted/40" onClick={() => navigation.push(paths.aiSettings())}>
              <div className="text-xs text-muted-foreground">{t(($) => $.dashboard.ai_enabled_automations)}</div>
              <div className="mt-2 text-2xl font-semibold tabular-nums">{aiSettingsLoading ? "—" : enabledAISettings.length}</div>
            </button>
            <div className="rounded-lg border p-3">
              <div className="text-xs text-muted-foreground">{t(($) => $.dashboard.ai_recent_runs)}</div>
              <div className="mt-2 text-2xl font-semibold tabular-nums">{aiHistoryLoading ? "—" : aiHistory.length}</div>
            </div>
            <button type="button" className="rounded-lg border p-3 text-left hover:bg-muted/40" onClick={openAIEmailComposer}>
              <div className="text-xs text-muted-foreground">{t(($) => $.dashboard.ai_email_writer)}</div>
              <div className="mt-2 flex items-center gap-2 text-sm font-medium"><PenLine className="size-4" />{t(($) => $.dashboard.ai_start_writing)}</div>
            </button>
          </div>
          <div className="mt-4 grid gap-3 lg:grid-cols-2">
            <div className="rounded-lg border p-3">
              <h3 className="text-xs font-medium text-muted-foreground">{t(($) => $.dashboard.ai_work_queue)}</h3>
              <div className="mt-3 space-y-2">
                {pendingReplyThreads.slice(0, 4).length === 0 ? <p className="text-sm text-muted-foreground">{t(($) => $.dashboard.ai_no_pending)}</p> : pendingReplyThreads.slice(0, 4).map((thread: any) => {
                  const emailPath = `${paths.emails()}?thread=${encodeURIComponent(thread.thread_id ?? thread.id)}&folder=${encodeURIComponent(thread.folder ?? "inbox")}`;
                  return <button key={thread.id} type="button" className="flex w-full items-center justify-between rounded-md border p-2 text-left text-sm hover:bg-muted/50" onClick={() => navigation.push(emailPath)}><span className="truncate font-medium">{thread.subject || t(($) => $.notes.untitled)}</span><Mail className="ml-2 size-4 text-muted-foreground" /></button>;
                })}
              </div>
            </div>
            <div className="rounded-lg border p-3">
              <h3 className="text-xs font-medium text-muted-foreground">{t(($) => $.dashboard.ai_recent_activity)}</h3>
              <div className="mt-3 space-y-2">
                {aiHistoryLoading ? <Skeleton className="h-20" /> : aiHistory.length === 0 ? <p className="text-sm text-muted-foreground">{t(($) => $.dashboard.ai_no_activity)}</p> : aiHistory.slice(0, 4).map((item: any, index) => <div key={item.id ?? index} className="rounded-md border p-2 text-sm"><div className="truncate font-medium">{item.automation_key ?? item.kind ?? t(($) => $.dashboard.ai_activity)}</div><div className="mt-1 text-xs text-muted-foreground">{formatDate(item.created_at ?? item.updated_at)}</div></div>)}
              </div>
            </div>
          </div>
        </section>
        <div className="grid gap-4 lg:grid-cols-2">
          <section className="rounded-lg border bg-card p-4">
            <h2 className="text-sm font-medium">{t(($) => $.dashboard.today_title)}</h2>
            <div className="mt-3">{todayLoading ? <Skeleton className="h-24" /> : accountList(topTodayFollowUps, t(($) => $.dashboard.no_today))}</div>
          </section>
          <section className="rounded-lg border bg-card p-4">
            <h2 className="text-sm font-medium">{t(($) => $.dashboard.week_title)}</h2>
            <div className="mt-3">{weekLoading ? <Skeleton className="h-24" /> : accountList(topWeekFollowUps, t(($) => $.dashboard.no_week))}</div>
          </section>
          <section className="rounded-lg border bg-card p-4">
            <h2 className="text-sm font-medium">{t(($) => $.dashboard.overdue_title)}</h2>
            <div className="mt-3">{overdueLoading ? <Skeleton className="h-24" /> : accountList(topOverdueFollowUps, t(($) => $.dashboard.no_followups))}</div>
          </section>
          <section className="rounded-lg border bg-card p-4">
            <h2 className="text-sm font-medium">{t(($) => $.dashboard.high_priority_title)}</h2>
            <div className="mt-3">{highPriorityLoading ? <Skeleton className="h-24" /> : accountList(topHighPriorityAccounts, t(($) => $.dashboard.no_high_priority))}</div>
          </section>
          <section className="rounded-lg border bg-card p-4 lg:col-span-2">
            <h2 className="text-sm font-medium">{t(($) => $.dashboard.recent_emails_title)}</h2>
            <div className="mt-3">
              {emailLoading ? <Skeleton className="h-24" /> : topEmailThreads.length === 0 ? <p className="text-sm text-muted-foreground">{t(($) => $.dashboard.no_emails)}</p> : (
                <div className="space-y-2">{topEmailThreads.map((item) => {
                  const subject = item.subject || t(($) => $.notes.untitled);
                  const emailPath = `${paths.emails()}?thread=${encodeURIComponent((item as any).thread_id ?? item.id)}&folder=${encodeURIComponent((item as any).folder ?? "inbox")}`;
                  return (
                    <button key={item.id} type="button" className="flex w-full items-center justify-between rounded-md border p-2 text-left text-sm hover:bg-muted/50" onClick={() => navigation.push(emailPath)}>
                      <span className="truncate font-medium">{subject}</span>
                      <span className="ml-2 shrink-0 text-xs text-muted-foreground">{formatDate((item as any).received_at || (item as any).sent_at || (item as any).last_message_at || item.updated_at)}</span>
                    </button>
                  );
                })}</div>
              )}
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
