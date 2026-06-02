# Multica CRM 弱耦合改造 TODO

目标：在不丢失 CRM 功能、尽量不影响性能的前提下，把 CRM 二开从 Multica 官方核心代码中剥离，降低未来升级冲突。

工作分支：`crm-decouple-audit-20260602`
生产分支：`multica-with-crm`
部署目录：`/www/docker/multica`

## 原则

- 能放 CRM 专属目录，就不改 Multica 原核心文件。
- 非 CRM 核心文件只允许保留极薄注册点/extension slot。
- 不改官方 API response shape，除非无替代方案。
- 不为了方便修改 `comment.go` 等非 CRM 核心逻辑。
- `crm_*`、`crm_ai_scheduler.go`、`packages/views/crm/*`、`packages/core/crm/*` 可作为 CRM 专属区域。
- 每完成一项必须有：代码变更、测试/构建验证、风险说明。
- 合并生产前必须走 GH Actions/GHCR、镜像 tag、健康检查、CRM UI 可见验证。

## 完成度总览

- [x] 审计 CRM 分支相对官方 upstream 的非 CRM 核心改动
- [x] 分类强耦合点：可回退 / 必须桥接 / CRM 专属
- [x] 解耦 server 核心 router 中的 CRM 路由
- [x] 回退 Project API 内嵌 resources 的核心改动
- [x] 解耦前端 CRM API client
- [x] 解耦 CRM schemas/types
- [x] 解耦 CRM paths
- [x] 解耦 sidebar CRM 导航
- [ ] 解耦 CRM scheduler 启动逻辑
- [ ] 审计 sqlc/generated 与 DB 查询边界
- [ ] 全量测试、GHCR 构建、部署验证

## 1. 已完成：server router 解耦

### 问题

`server/cmd/server/router.go` 原来直接塞大量 CRM 路由，升级官方 router 时容易冲突。

### 处理

- [x] 在核心 router 中只保留一行注册钩子：

```go
registerCRMRoutes(r, h)
```

- [x] 新建 CRM 专属路由文件：

```text
server/cmd/server/router_crm.go
```

### 验证

- [x] `go test ./internal/handler ./cmd/server` 通过

### 后续风险

- `router.go` 仍有 1 行 CRM 注册点，属于可接受薄桥接。

## 2. 已完成：Project API resources 解耦

### 问题

CRM 曾修改官方 `ProjectResponse`，让 Project 内嵌 `resources`。这改变官方 API shape，强耦合。

### 处理

- [x] 回退：

```text
server/internal/handler/project.go
packages/core/types/project.ts
```

- [x] CRM 客户详情页改用官方已有资源接口：

```ts
projectResourcesOptions(wsId, project.id)
api.listProjectResources(project.id)
```

- [x] CRM 自己维护 `projectResourcesById`，不再依赖 `project.resources`。

### 验证

- [x] `go test ./internal/handler ./cmd/server` 通过
- [ ] 前端 build 验证
- [ ] CRM 客户详情页项目关联 UI 验证

### 性能影响

- 理论上会从“项目列表一次返回 resources”变成“项目列表 + 项目资源查询”。
- React Query 会缓存资源查询。
- 若项目很多，可后续加 CRM 专属批量 endpoint：`/api/crm/project-resources?account_id=...`，不要改官方 Project API。

## 3. 已完成：前端 CRM API client 解耦

### 当前强耦合

`packages/core/api/client.ts` 内含大量 CRM 方法：

- `listCRMAccounts`
- `getCRMAccount`
- `createCRMAccount`
- `updateCRMAccount`
- `deleteCRMAccount`
- `listCRMContacts`
- `createCRMContact`
- `updateCRMContact`
- `deleteCRMContact`
- `getCRMAccountProfile`
- `upsertCRMAccountProfile`
- `refreshCRMAccountProfile`
- `listCRMEmailThreads`
- `listCRMEmailMessages`
- `createCRMEmailDraft`
- `sendCRMEmailDraft`
- IMAP / EmailEngine / AI settings 相关方法

### 目标结构

- [ ] 核心 `ApiClient` 只保留通用请求能力：

```ts
api.request<T>(path, init)
```

或导出安全 helper：

```ts
request<T>(path, init)
```

- [ ] 新建 CRM 专属 API 文件：

```text
packages/core/crm/api.ts
```

- [ ] CRM API 方法全部搬入：

```ts
export const crmApi = {
  listAccounts,
  getAccount,
  createAccount,
  ...
}
```

- [ ] `packages/core/crm/queries.ts` 改为调用 `crmApi`，不再调用 `api.*CRM*`。

- [ ] CRM 页面中的直接调用改为 `crmApi.*`：

```text
packages/views/crm/components/*.tsx
```

### 验证

- [ ] 搜索确认 `api.*CRM*` 不再存在：

```bash
rg 'api\.(listCRM|getCRM|createCRM|updateCRM|deleteCRM|upsertCRM|refreshCRM|suggestCRM|applyCRM|testCRM|previewCRM|importCRM|syncCRM|trashCRM|restoreCRM|moveCRM|downloadCRM|getCRMEmailAttachment|toggleCRM)' packages
```

- [x] 前端 typecheck 通过（未在本机跑 Go test）。
- [ ] CRM dashboard / customers / account detail / emails / AI settings 基本流程验证

### 性能影响

无。仍走同一个 fetch/request、同一个后端 endpoint。

## 4. 已完成：CRM schemas/types 解耦

### 当前强耦合

`packages/core/api/schemas.ts` 里存在 CRM schema 和 empty fallback。

### 目标结构

- [ ] 新建：

```text
packages/core/crm/schemas.ts
packages/core/crm/types.ts
```

CRM types no longer exported from `packages/core/types/index.ts`; CRM callers import `@multica/core/crm/types`.

- [x] CRM schema 迁移到 CRM 专属文件。
- [x] `packages/core/api/schemas.ts` 回退官方核心 schema。
- [x] CRM API 使用 CRM schema 做 parse/fallback。

### 验证

- [x] 搜索确认核心 schemas 中无 CRM：

```bash
rg 'CRM|crm' packages/core/api/schemas.ts
```

- [x] 前端 typecheck 通过（未在本机跑 Go test）。
- [x] CRM API fallback 行为不变：schema/fallback 搬迁，调用点不变。

## 5. 已完成：CRM paths 解耦

### 当前强耦合

`packages/core/paths/paths.ts` 里加入了：

- `crm()`
- `crmCustomers()`
- `crmCustomerDetail(id)`
- `crmEmails()`
- `crmAISettings()`

### 目标结构

- [x] 新建：

```text
packages/core/crm/paths.ts
```

- [x] 提供：

```ts
crmPaths.workspace(slug).dashboard()
crmPaths.workspace(slug).customers()
crmPaths.workspace(slug).customerDetail(id)
crmPaths.workspace(slug).emails()
crmPaths.workspace(slug).aiSettings()
```

- [x] CRM 页面改用 `useCRMWorkspacePaths`。
- [x] 核心 `paths.ts` 回退官方内容。

### 验证

- [x] 搜索确认核心 paths 中无 CRM：

```bash
rg 'crm' packages/core/paths/paths.ts
```

- [ ] CRM 页面跳转路径正常
- [ ] dashboard AI 写信入口仍正确跳邮件工作区

### 性能影响

无。纯字符串函数。

## 6. 已完成：sidebar CRM 导航解耦

### 当前强耦合

`packages/views/layout/app-sidebar.tsx` 里硬插 CRM 导航和 CRM 文案 key。

### 目标结构

优先方案：核心只保留 extension slot。

- [x] 新建 CRM nav provider：

```text
packages/views/crm/sidebar-nav.tsx
```

- [x] 核心 sidebar 只保留最小桥接：

```tsx
<SidebarExtensions workspaceSlug={workspaceSlug} />
```

或最小桥接：

```tsx
<CRMNavGroup />
```

- [x] CRM nav 内部自己处理：
  - icon
  - title
  - active path
  - CRM paths
  - i18n 文案

- [x] layout locale 中 CRM 文案已移出，改用 CRM locale 自有文案。

### 验证

- [ ] sidebar 显示 CRM 入口
- [ ] CRM dashboard/customers/emails/AI settings 入口可点击
- [ ] 非 CRM sidebar 项不变
- [ ] 核心 sidebar diff 尽量小

### 性能影响

无。只渲染少量 nav item。

## 7. 待做：CRM scheduler 启动逻辑解耦

### 当前强耦合

核心 handler/init 里启动 CRM AI scheduler。

### 目标结构

- [ ] 新建 CRM 启动模块：

```text
server/internal/handler/crm_bootstrap.go
```

或：

```text
server/internal/crm/bootstrap.go
```

- [ ] 核心只留可接受薄桥接：

```go
h.StartCRMServices(ctx)
```

或在 CRM route registration 内完成 scheduler 注册。

- [ ] `crm_ai_scheduler.go` 保持 CRM 专属。

### 验证

- [ ] 后端启动无 panic
- [ ] mailbox sync / AI scheduler 行为不变
- [ ] sync errors watchdog 数据不变

### 性能影响

无。启动位置改变，不改变任务执行逻辑。

## 8. 待做：sqlc/generated 与 DB 查询边界审计

### 当前风险

`server/pkg/db/generated/*` 可能因 CRM SQL 或核心 SQL 变动产生大面积 diff。

### 目标

- [ ] 确认 CRM SQL 都集中在 CRM 命名文件或 CRM query 段。
- [ ] 不修改官方非 CRM query，除非必要。
- [ ] 若必须新增 query，命名带 CRM 前缀。
- [ ] 生成文件无法完全避免，但要保证来源清楚。

### 验证

- [ ] `sqlc generate` 后 diff 可解释
- [ ] 非 CRM query 无功能变更
- [ ] 后端 `go test ./...` 通过

## 9. 全量验证计划

### 本地/CI 验证

- [x] `git diff --check`
- [ ] `go test ./...` in `server`（本机跳过，避免卡死；交给 GitHub Actions）
- [ ] 前端 build：

```bash
pnpm --filter @multica/web build
```

- [ ] CRM 相关测试：

```bash
pnpm --filter @multica/views test -- --run packages/views/crm
```

- [ ] GH Actions/GHCR 成功

### 部署验证

- [ ] 确认目标分支：`multica-with-crm`
- [ ] 确认 GHCR tag：`crm-sha-xxxxxxx`
- [ ] 备份部署 `.env`
- [ ] 更新 `/www/docker/multica/.env` 中 `MULTICA_IMAGE_TAG`
- [ ] `docker compose pull`
- [ ] `docker compose up -d --remove-orphans`
- [ ] 验证容器镜像 tag
- [ ] 前端：`http://127.0.0.1:13010` 返回 `200 OK`
- [ ] 后端：`http://127.0.0.1:18080/health` 返回 `{"status":"ok"}`
- [ ] revision label 与部署 tag 对齐

### CRM UI 可见验证

- [ ] CRM dashboard 可打开
- [ ] dashboard 内容超出时可滚动
- [ ] dashboard 邮件数量显示未读邮件数
- [ ] dashboard AI 写信入口复用邮件工作区 AI 写信
- [ ] 最近邮件区域布局无右侧空缺
- [ ] 客户列表可打开
- [ ] 客户详情可打开
- [ ] 客户画像摘要显示丰富 `customer_summary`
- [ ] 刷新客户画像可用
- [ ] 刷新画像后备注/动态 tab 出现记录
- [ ] 高置信度跟进建议可按护栏更新 `next_follow_up_at`
- [ ] 联系人增删改正常
- [ ] 项目关联/取消关联正常
- [ ] 邮件工作区可打开
- [ ] 邮件线程可打开
- [ ] 邮件附件 URL/下载正常
- [ ] AI 写信可提取收件人
- [ ] AI 写信可创建草稿
- [ ] 草稿编辑/发送流程正常
- [ ] IMAP 设置、测试、预览、同步正常
- [ ] AI settings 页面可打开并保存

## 10. 合并策略

- [ ] 审计分支通过后，合并到 `multica-with-crm`
- [ ] 不向 upstream `multica-ai/multica` 开 PR，除非 Donny 明确要求
- [ ] 保留清晰 commit message：

```text
refactor(crm): isolate CRM extensions from core Multica code
```

- [ ] 合并后触发 GHCR 构建
- [ ] 只部署 `crm-sha-*` tag

## 11. 回滚策略

只回滚镜像 tag，不动数据库卷。

- [ ] 记录当前旧 tag
- [ ] 备份 `.env`
- [ ] 改回旧 `MULTICA_IMAGE_TAG`
- [ ] `docker compose pull`
- [ ] `docker compose up -d --remove-orphans`
- [ ] 验证前后端健康检查

禁止：

```bash
docker compose down -v
```

## 12. 当前已提交记录

- [x] `121bf588108b refactor(crm): reduce coupling to core router and project API`
  - 分支：`crm-decouple-audit-20260602`
  - 内容：router 解耦 + Project API resources 回退 + CRM 页面适配资源查询

## 13. 后续执行顺序

1. [ ] CRM paths 解耦
2. [ ] sidebar 导航解耦
3. [ ] scheduler 启动解耦
4. [x] CRM API client 解耦
5. [x] CRM schemas/types 解耦
6. [ ] sqlc/generated 审计
7. [ ] 前后端测试
8. [ ] GHCR 构建
9. [ ] 部署验证
10. [ ] UI 可见验证
