"use client";
/* eslint-disable i18next/no-literal-string */

import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Activity, ArrowLeft, Bot, Clock, Mail, MoreHorizontal, RefreshCw, Settings, Users } from "lucide-react";
import { api } from "@multica/core/api";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import { crmKeys } from "@multica/core/crm/queries";
import { agentListOptions, memberListOptions } from "@multica/core/workspace/queries";
import { Badge } from "@multica/ui/components/ui/badge";
import { Button } from "@multica/ui/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@multica/ui/components/ui/dropdown-menu";
import { Input } from "@multica/ui/components/ui/input";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { PageHeader } from "../../layout/page-header";

type SettingKey = "email_pending_reply" | "due_followup" | "profile_new_activity_refresh" | "profile_daily_refresh";

type CRMAIConfig = {
  follow_up_lead_days?: number;
  duplicate_protection_days?: number;
  handled_window_hours?: number;
  same_subject_dedupe_days?: number;
  stale_done_issue_days?: number;
  profile_refresh_min_interval_minutes?: number;
  time?: string;
  issue_creator_type?: "member" | "agent";
  issue_creator_id?: string;
  issue_todo_assignee_type?: "member" | "agent";
  issue_todo_assignee_id?: string;
};

type CRMAILastResult = {
  checked_at?: string;
  candidates?: number;
  created?: number;
  issues_created?: number;
  tasks_queued?: number;
  skipped_contacted?: number;
  skipped_existing_issue?: number;
  skipped_existing_task?: number;
  skipped_title_duplicate?: number;
  skipped_handled?: number;
  skipped_done_issue?: number;
  created_issues?: Array<{ id: string; title?: string; kind?: string; thread_id?: string; message_id?: string; account_id?: string }>;
  note?: string;
};

type CRMAISetting = {
  workspace_id: string;
  automation_key: SettingKey;
  enabled: boolean;
  interval_minutes: number;
  assignee_agent_id?: string | null;
  max_items_per_run: number;
  last_checked_at?: string | null;
  config?: CRMAIConfig | null;
  last_result?: CRMAILastResult | null;
};

type CRMAIHistoryItem = {
  id: string;
  title: string;
  status: string;
  origin_id?: string | null;
  automation_key: SettingKey | "other";
  created_at: string;
  updated_at: string;
};

type ActorType = "member" | "agent";
type FormState = Pick<CRMAISetting, "enabled" | "interval_minutes" | "assignee_agent_id" | "max_items_per_run"> & Required<Pick<CRMAIConfig, "follow_up_lead_days" | "duplicate_protection_days" | "handled_window_hours" | "same_subject_dedupe_days" | "stale_done_issue_days" | "profile_refresh_min_interval_minutes" | "time">> & {
  issue_creator_type: ActorType;
  issue_creator_id: string;
  issue_todo_assignee_type: ActorType;
  issue_todo_assignee_id: string;
};

const meta: Record<SettingKey, { title: string; description: string; icon: typeof Mail }> = {
  email_pending_reply: {
    title: "邮件待回复巡检",
    description: "低成本 SQL 检查邮件线程；先审视是否已处理或已有 issue，再决定是否启动 AI。",
    icon: Mail,
  },
  due_followup: {
    title: "到期客户跟进",
    description: "检查到期/即将到期客户；先审视是否已联系、已有 issue 或重复任务，再启动 AI。",
    icon: Users,
  },
  profile_new_activity_refresh: {
    title: "新互动客户画像刷新",
    description: "收到新邮件或发出新邮件后刷新关联客户画像；带最小刷新间隔，避免频繁重复生成。",
    icon: Activity,
  },
  profile_daily_refresh: {
    title: "每日客户画像全量刷新",
    description: "按每天指定时间批量刷新客户画像，用于补齐长期未更新资料。",
    icon: RefreshCw,
  },
};

const defaults: Record<SettingKey, Required<Pick<CRMAIConfig, "follow_up_lead_days" | "duplicate_protection_days" | "handled_window_hours" | "same_subject_dedupe_days" | "stale_done_issue_days" | "profile_refresh_min_interval_minutes" | "time">>> = {
  email_pending_reply: {
    follow_up_lead_days: 0,
    duplicate_protection_days: 7,
    handled_window_hours: 48,
    same_subject_dedupe_days: 7,
    stale_done_issue_days: 7,
    profile_refresh_min_interval_minutes: 60,
    time: "03:00",
  },
  due_followup: {
    follow_up_lead_days: 0,
    duplicate_protection_days: 7,
    handled_window_hours: 48,
    same_subject_dedupe_days: 7,
    stale_done_issue_days: 7,
    profile_refresh_min_interval_minutes: 60,
    time: "03:00",
  },
  profile_new_activity_refresh: {
    follow_up_lead_days: 0,
    duplicate_protection_days: 7,
    handled_window_hours: 48,
    same_subject_dedupe_days: 7,
    stale_done_issue_days: 7,
    profile_refresh_min_interval_minutes: 60,
    time: "03:00",
  },
  profile_daily_refresh: {
    follow_up_lead_days: 0,
    duplicate_protection_days: 7,
    handled_window_hours: 48,
    same_subject_dedupe_days: 7,
    stale_done_issue_days: 7,
    profile_refresh_min_interval_minutes: 60,
    time: "03:00",
  },
};

function numberValue(value: number, min: number, max: number) {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, Math.round(value)));
}

function fmt(value?: string | null) {
  return value ? new Date(value).toLocaleString() : "—";
}

function buildForm(setting: CRMAISetting): FormState {
  const d = defaults[setting.automation_key];
  const c = setting.config || {};
  return {
    enabled: setting.enabled,
    interval_minutes: setting.interval_minutes,
    assignee_agent_id: setting.assignee_agent_id || "",
    max_items_per_run: setting.max_items_per_run,
    follow_up_lead_days: c.follow_up_lead_days ?? d.follow_up_lead_days,
    duplicate_protection_days: c.duplicate_protection_days ?? d.duplicate_protection_days,
    handled_window_hours: c.handled_window_hours ?? d.handled_window_hours,
    same_subject_dedupe_days: c.same_subject_dedupe_days ?? d.same_subject_dedupe_days,
    stale_done_issue_days: c.stale_done_issue_days ?? d.stale_done_issue_days,
    profile_refresh_min_interval_minutes: c.profile_refresh_min_interval_minutes ?? d.profile_refresh_min_interval_minutes,
    time: c.time ?? d.time,
    issue_creator_type: c.issue_creator_type || "agent",
    issue_creator_id: c.issue_creator_id || "",
    issue_todo_assignee_type: c.issue_todo_assignee_type || "agent",
    issue_todo_assignee_id: c.issue_todo_assignee_id || setting.assignee_agent_id || "",
  };
}

function SettingHistory({ automationKey }: { automationKey: SettingKey }) {
  const wsId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const [limit, setLimit] = useState(20);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const { data, isLoading, isFetching } = useQuery({
    queryKey: [...crmKeys.aiSettings(wsId), "history", automationKey, limit],
    queryFn: () => api.listCRMAIHistory({ automation_key: automationKey, days: 30, limit, offset: 0 }),
    select: (res) => ({ ...res, items: res.items as CRMAIHistoryItem[] }),
    enabled: Boolean(wsId),
    refetchInterval: autoRefresh ? 5000 : false,
    refetchIntervalInBackground: false,
  });
  const items = data?.items ?? [];
  const latest = items[0];
  const info = meta[automationKey];
  return (
    <div className="min-h-0 flex-1 overflow-y-auto p-5 pb-10">
      <section className="rounded-lg border bg-card p-4">
        <div className="mb-3 flex items-center justify-between gap-3 text-sm font-medium">
          <span className="inline-flex items-center gap-2"><Clock className="size-4" />{info.title} · 最近一次</span>
          <label className="inline-flex items-center gap-2 text-xs font-normal text-muted-foreground">
            <input type="checkbox" checked={autoRefresh} onChange={(e) => setAutoRefresh(e.target.checked)} />
            自动刷新
          </label>
        </div>
        {isLoading ? <Skeleton className="h-16 w-full" /> : latest ? (
          <a href={paths.issueDetail(latest.id)} className="block rounded-md border bg-background p-3 hover:bg-muted/40">
            <div className="flex items-center justify-between gap-3">
              <div className="truncate text-sm font-medium">{latest.title}</div>
              <Badge variant="secondary">{latest.status}</Badge>
            </div>
            <div className="mt-1 text-xs text-muted-foreground">{fmt(latest.created_at)}</div>
          </a>
        ) : <div className="text-sm text-muted-foreground">暂无运行记录</div>}
      </section>
      <section className="mt-4 rounded-lg border bg-card">
        <div className="flex h-11 items-center justify-between border-b px-4 text-sm font-medium">
          <span>最近30天所有历史</span>
          <span className="text-xs text-muted-foreground">已加载 {items.length}</span>
        </div>
        {isLoading ? (
          <div className="space-y-2 p-4">{Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} className="h-12 w-full" />)}</div>
        ) : items.length ? (
          <div className="divide-y">
            {items.map((item) => (
              <a key={item.id} href={paths.issueDetail(item.id)} className="grid grid-cols-[minmax(220px,1fr)_120px_180px] items-center gap-3 px-4 py-3 text-sm hover:bg-muted/40">
                <div className="truncate font-medium">{item.title}</div>
                <Badge variant="secondary" className="w-fit">{item.status}</Badge>
                <div className="text-xs text-muted-foreground">{fmt(item.created_at)}</div>
              </a>
            ))}
          </div>
        ) : <div className="p-4 text-sm text-muted-foreground">暂无运行记录</div>}
        {data?.has_more ? (
          <div className="border-t p-4 text-center">
            <Button variant="outline" size="sm" disabled={isFetching} onClick={() => setLimit((value) => value + 20)}>{isFetching ? "加载中..." : "加载更多"}</Button>
          </div>
        ) : null}
      </section>
    </div>
  );
}

function actorLabel(actor: { type: ActorType; id: string }, agents: Array<{ id: string; name: string }>, members: Array<{ id: string; name?: string; email?: string }>) {
  if (!actor.id) return "默认";
  if (actor.type === "agent") return agents.find((agent) => agent.id === actor.id)?.name ?? "未知 Agent";
  const member = members.find((item) => item.id === actor.id);
  return member?.name || member?.email || "未知成员";
}

function ActorSelect({ label, type, value, agents, members, onChange }: { label: string; type: ActorType; value: string; agents: Array<{ id: string; name: string }>; members: Array<{ id: string; name?: string; email?: string }>; onChange: (next: { type: ActorType; id: string }) => void }) {
  return (
    <label className="space-y-1 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <div className="grid grid-cols-[92px_1fr] gap-2">
        <select className="h-9 rounded-md border bg-background px-2 text-sm" value={type} onChange={(e) => onChange({ type: e.target.value as ActorType, id: "" })}>
          <option value="agent">Agent</option>
          <option value="member">成员</option>
        </select>
        <select className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={value} onChange={(e) => onChange({ type, id: e.target.value })}>
          <option value="">默认</option>
          {type === "agent"
            ? agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)
            : members.map((member) => <option key={member.id} value={member.id}>{member.name || member.email || member.id}</option>)}
        </select>
      </div>
    </label>
  );
}

function SettingCard({ setting, agents, members }: { setting: CRMAISetting; agents: Array<{ id: string; name: string }>; members: Array<{ id: string; name?: string; email?: string }> }) {
  const wsId = useWorkspaceId();
  const qc = useQueryClient();
  const info = meta[setting.automation_key];
  const Icon = info.icon;
  const [form, setForm] = useState<FormState>(() => buildForm(setting));

  useEffect(() => setForm(buildForm(setting)), [setting]);

  const save = useMutation({
    mutationFn: () => {
      const config: CRMAIConfig = {};
      if (setting.automation_key === "email_pending_reply") {
        config.duplicate_protection_days = numberValue(form.duplicate_protection_days, 0, 365);
        config.handled_window_hours = numberValue(form.handled_window_hours, 0, 24 * 365);
        config.stale_done_issue_days = numberValue(form.stale_done_issue_days, 0, 365);
        config.same_subject_dedupe_days = numberValue(form.same_subject_dedupe_days, 0, 365);
      } else if (setting.automation_key === "due_followup") {
        config.duplicate_protection_days = numberValue(form.duplicate_protection_days, 0, 365);
        config.handled_window_hours = numberValue(form.handled_window_hours, 0, 24 * 365);
        config.stale_done_issue_days = numberValue(form.stale_done_issue_days, 0, 365);
        config.follow_up_lead_days = numberValue(form.follow_up_lead_days, 0, 365);
      } else if (setting.automation_key === "profile_new_activity_refresh") {
        config.profile_refresh_min_interval_minutes = numberValue(form.profile_refresh_min_interval_minutes, 0, 24 * 60);
      } else if (setting.automation_key === "profile_daily_refresh") {
        config.time = form.time || "03:00";
      }
      return api.updateCRMAISetting(setting.automation_key, {
        enabled: form.enabled,
        interval_minutes: numberValue(form.interval_minutes, 1, 1440),
        assignee_agent_id: form.assignee_agent_id || null,
        max_items_per_run: numberValue(form.max_items_per_run, 1, 100),
        config: {
          ...config,
          issue_creator_type: form.issue_creator_type,
          issue_creator_id: form.issue_creator_id || undefined,
          issue_todo_assignee_type: form.issue_todo_assignee_type,
          issue_todo_assignee_id: form.issue_todo_assignee_id || undefined,
        },
      });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: crmKeys.aiSettings(wsId) }),
  });

  return (
    <section className="rounded-lg border bg-card p-4">
      <div className="flex items-start justify-between gap-4">
        <div className="flex gap-3">
          <div className="mt-0.5 rounded-md bg-muted p-2"><Icon className="size-4 text-muted-foreground" /></div>
          <div>
            <h2 className="text-sm font-medium">{info.title}</h2>
            <p className="mt-1 max-w-2xl text-sm text-muted-foreground">{info.description}</p>
          </div>
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={form.enabled} onChange={(e) => setForm((s) => ({ ...s, enabled: e.target.checked }))} />
          启用
        </label>
      </div>
      <div className="mt-4 grid gap-4 md:grid-cols-4">
        <label className="space-y-1 text-sm">
          <span className="text-muted-foreground">检查间隔（分钟）</span>
          <Input type="number" min={1} max={1440} value={form.interval_minutes} onChange={(e) => setForm((s) => ({ ...s, interval_minutes: Number(e.target.value) }))} />
        </label>
        <label className="space-y-1 text-sm">
          <span className="text-muted-foreground">单次最多创建</span>
          <Input type="number" min={1} max={100} value={form.max_items_per_run} onChange={(e) => setForm((s) => ({ ...s, max_items_per_run: Number(e.target.value) }))} />
        </label>
        {setting.automation_key === "due_followup" ? (
          <label className="space-y-1 text-sm">
            <span className="text-muted-foreground">到期前几天开始</span>
            <Input type="number" min={0} max={365} value={form.follow_up_lead_days} onChange={(e) => setForm((s) => ({ ...s, follow_up_lead_days: Number(e.target.value) }))} />
          </label>
        ) : setting.automation_key === "profile_new_activity_refresh" ? (
          <label className="space-y-1 text-sm">
            <span className="text-muted-foreground">最小刷新间隔（分钟）</span>
            <Input type="number" min={0} max={1440} value={form.profile_refresh_min_interval_minutes} onChange={(e) => setForm((s) => ({ ...s, profile_refresh_min_interval_minutes: Number(e.target.value) }))} />
          </label>
        ) : setting.automation_key === "profile_daily_refresh" ? (
          <label className="space-y-1 text-sm">
            <span className="text-muted-foreground">每日刷新时间</span>
            <Input type="time" value={form.time} onChange={(e) => setForm((s) => ({ ...s, time: e.target.value }))} />
          </label>
        ) : (
          <label className="space-y-1 text-sm">
            <span className="text-muted-foreground">同主题去重窗口（天）</span>
            <Input type="number" min={0} max={365} value={form.same_subject_dedupe_days} onChange={(e) => setForm((s) => ({ ...s, same_subject_dedupe_days: Number(e.target.value) }))} />
          </label>
        )}
        {setting.automation_key === "email_pending_reply" || setting.automation_key === "due_followup" ? (
          <>
            <ActorSelect label="Issue 创建人" type={form.issue_creator_type} value={form.issue_creator_id} agents={agents} members={members} onChange={(next) => setForm((s) => ({ ...s, issue_creator_type: next.type, issue_creator_id: next.id }))} />
            <ActorSelect label="Issue todo 负责人" type={form.issue_todo_assignee_type} value={form.issue_todo_assignee_id} agents={agents} members={members} onChange={(next) => setForm((s) => ({ ...s, issue_todo_assignee_type: next.type, issue_todo_assignee_id: next.id, assignee_agent_id: next.type === "agent" ? next.id : "" }))} />
          </>
        ) : (
          <label className="space-y-1 text-sm">
            <span className="text-muted-foreground">执行 Agent</span>
            <select className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={form.assignee_agent_id || ""} onChange={(e) => setForm((s) => ({ ...s, assignee_agent_id: e.target.value }))}>
              <option value="">默认 Agent</option>
              {agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}
            </select>
          </label>
        )}
        {setting.automation_key === "email_pending_reply" || setting.automation_key === "due_followup" ? (
          <>
            <label className="space-y-1 text-sm">
              <span className="text-muted-foreground">重复保护窗口（天）</span>
              <Input type="number" min={0} max={365} value={form.duplicate_protection_days} onChange={(e) => setForm((s) => ({ ...s, duplicate_protection_days: Number(e.target.value) }))} />
            </label>
            <label className="space-y-1 text-sm">
              <span className="text-muted-foreground">已处理判断窗口（小时）</span>
              <Input type="number" min={0} max={8760} value={form.handled_window_hours} onChange={(e) => setForm((s) => ({ ...s, handled_window_hours: Number(e.target.value) }))} />
            </label>
            <label className="space-y-1 text-sm">
              <span className="text-muted-foreground">done issue 保护（天）</span>
              <Input type="number" min={0} max={365} value={form.stale_done_issue_days} onChange={(e) => setForm((s) => ({ ...s, stale_done_issue_days: Number(e.target.value) }))} />
            </label>
          </>
        ) : null}
      </div>
      <div className="mt-4 flex items-center justify-between text-xs text-muted-foreground">
        <span>上次检查：{fmt(setting.last_checked_at)}</span>
        <Button size="sm" onClick={() => save.mutate()} disabled={save.isPending}>{save.isPending ? "保存中..." : "保存"}</Button>
      </div>
    </section>
  );
}

function SettingListRow({
  setting,
  agentName,
  onConfigure,
  onHistory,
}: {
  setting: CRMAISetting;
  agentName: string;
  onConfigure: () => void;
  onHistory: () => void;
}) {
  const info = meta[setting.automation_key];
  const Icon = info.icon;
  return (
    <div
      className="grid min-h-16 grid-cols-[minmax(260px,1fr)_120px_160px_160px_60px] items-center border-b px-4 text-sm last:border-b-0 hover:bg-muted/30"
      onDoubleClick={onConfigure}
    >
      <div className="flex min-w-0 items-center gap-3">
        <div className="shrink-0 rounded-md bg-muted p-2">
          <Icon className="size-4 text-muted-foreground" />
        </div>
        <div className="min-w-0">
          <div className="truncate font-medium">{info.title}</div>
          <div className="mt-0.5 truncate text-xs text-muted-foreground">{info.description}</div>
        </div>
      </div>
      <div>
        <Badge variant={setting.enabled ? "default" : "secondary"}>{setting.enabled ? "已启用" : "已停用"}</Badge>
      </div>
      <div className="truncate text-xs text-muted-foreground">{agentName}</div>
      <div className="truncate text-xs text-muted-foreground">{fmt(setting.last_checked_at)}</div>
      <div className="flex justify-end" onClick={(e) => e.stopPropagation()}>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button variant="ghost" size="icon-sm" aria-label="打开操作菜单" />
            }
          >
            <MoreHorizontal className="h-4 w-4 text-muted-foreground" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-auto">
            <DropdownMenuItem onClick={onConfigure}>
              <Settings className="h-3.5 w-3.5" />
              配置
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onHistory}>
              <Clock className="h-3.5 w-3.5" />
              运行记录
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}

export function CRMAISettingsPage() {
  const wsId = useWorkspaceId();
  const { data = [], isLoading } = useQuery({ queryKey: crmKeys.aiSettings(wsId), queryFn: () => api.listCRMAISettings(), select: (res) => res.settings as CRMAISetting[] });
  const { data: agents = [] } = useQuery(agentListOptions(wsId));
  const { data: members = [] } = useQuery(memberListOptions(wsId));
  const [selectedKey, setSelectedKey] = useState<SettingKey | null>(null);
  const [mode, setMode] = useState<"config" | "history">("config");

  const selectedSetting = useMemo(
    () => data.find((setting) => setting.automation_key === selectedKey) ?? null,
    [data, selectedKey],
  );

  return (
    <div className="flex h-full min-h-0 flex-col">
      <PageHeader className="justify-between px-5">
        <div className="flex items-center gap-2">
          <Bot className="size-4 text-muted-foreground" />
          <h1 className="text-sm font-medium">CRM AI</h1>
          {!selectedSetting && data.length > 0 ? (
            <span className="font-mono text-xs tabular-nums text-muted-foreground/70">{data.length}</span>
          ) : null}
        </div>
        {selectedSetting ? (
          <Button type="button" variant="ghost" size="sm" onClick={() => { setSelectedKey(null); setMode("config"); }}>
            <ArrowLeft className="h-3.5 w-3.5" />
            返回列表
          </Button>
        ) : null}
      </PageHeader>

      {selectedSetting ? (
        mode === "history" ? (
          <SettingHistory automationKey={selectedSetting.automation_key} />
        ) : (
          <div className="min-h-0 flex-1 overflow-y-auto p-5 pb-10">
            <SettingCard setting={selectedSetting} agents={agents} members={members} />
          </div>
        )
      ) : (
        <div className="flex min-h-0 flex-1 flex-col gap-4 p-5">
          <div className="rounded-lg border bg-card p-4 text-sm text-muted-foreground">
            这些设置控制 Hermes 低成本 SQL watchdog。SQL 先做候选筛选与去重/已处理审视；只有发现真实待处理事项时，才创建 Multica issue 并启动 AI。
          </div>
          <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border bg-background">
            <div className="grid h-11 shrink-0 grid-cols-[minmax(260px,1fr)_120px_160px_160px_60px] items-center border-b px-4 text-xs font-medium text-muted-foreground">
              <div>自动化</div>
              <div>状态</div>
              <div>执行 Agent</div>
              <div>上次检查</div>
              <div />
            </div>
            {isLoading ? (
              <div className="space-y-2 p-4">
                {Array.from({ length: 4 }).map((_, i) => (
                  <Skeleton key={i} className="h-14 w-full rounded-md" />
                ))}
              </div>
            ) : (
              <div className="min-h-0 flex-1 overflow-y-auto">
                {data.map((setting) => (
                  <SettingListRow
                    key={setting.automation_key}
                    setting={setting}
                    agentName={actorLabel({ type: setting.config?.issue_todo_assignee_type || "agent", id: setting.config?.issue_todo_assignee_id || setting.assignee_agent_id || "" }, agents, members)}
                    onConfigure={() => { setSelectedKey(setting.automation_key); setMode("config"); }}
                    onHistory={() => { setSelectedKey(setting.automation_key); setMode("history"); }}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
