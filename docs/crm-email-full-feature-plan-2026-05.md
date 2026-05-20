# Multica CRM 邮件全功能补齐开发计划（2026-05）

目标：把 Donny 最新提出的 CRM 邮件调整写成可执行计划，分两批交付，避免分步开发遗漏。

硬约束：
- 直接 assistant 执行，不走 Kanban，除非 Donny 明确要求。
- 源码目录：`/root/work/multica-crm`。
- 分支：`multica-with-crm`。
- 部署目录：`/www/docker/multica`。
- 不在生产服务器跑本地 build/test；只允许轻量 `git diff --check`、文件检查、HTTP/DB 检查。构建验证走 GitHub Actions/GHCR。
- 部署必须用 `/www/docker/multica/docker-compose.yml` + `/www/docker/multica/docker-compose.selfhost.yml`，保留端口/volume。
- 部署前 `docker rmi -f ghcr.io/flyinsz/multica-web:crm-latest ghcr.io/flyinsz/multica-backend:crm-latest`（按实际改动选择前端/后端）。
- 不使用 EmailEngine iframe/嵌入，保持 Multica CRM 原生 UI。
- UI 文案必须跟随系统语言，不硬编码单语言（本计划中的中文是需求说明，代码实现需按现有 i18n 结构处理）。

## 总需求清单

1. 垃圾邮件箱同步
- 同步服务商 spam/junk 文件夹邮件。
- CRM 邮件文件夹中能看到垃圾邮件箱。
- 兼容常见 IMAP 文件夹名：`Junk`、`Spam`、`[Gmail]/Spam`、`垃圾邮件` 等。

2. 客户关系工作区邮件菜单显示收件箱未读数
- 邮件菜单后显示 inbox 未读 badge。
- 统计收件箱未读，不含垃圾邮件/废纸篓/归档。
- 同步后自动刷新。

3. CRM 菜单/路由调整
- CRM 菜单改名为“仪表盘”。
- 路由改为 `/crm/dashboard`。
- `/crm` 建议保留兼容跳转到 `/crm/dashboard`。

4. 邮件详情页客户/联系人按钮与弹窗
- 去除邮件详情页原“打开客户”按钮。
- 关联客户弹窗增加客户基本信息。
- 关联客户弹窗按钮改为“客户详情”。
- 关联联系人弹窗增加“联系人详情”按钮。
- 邮件调整关联客户的菜单合并到关联客户弹窗中。

5. 邮件列表关联状态
- 当前“未关联”图标改为显示“已关联”。
- 已关联应展示正向状态/客户名；未关联弱提示或空状态。

6. 邮件列表快速筛选
- 邮件列表上方增加标签：全部、未关联、已关联、未读、已读。
- 筛选应服务端支持，避免大列表前端全量过滤。

7. 三点菜单与批量操作
- 搜索栏右侧增加三点菜单，参考 Multica 原生 `/inbox`。
- 菜单项：全部标为已读、全部标为未读、全选、多选。
- 全选/多选后列表出现 checkbox。
- 多选时出现批量操作下拉框。
- 批量操作：归档、星标、移入废纸篓等（建议同时支持标为已读/未读）。

8. 附件下载修复
- 邮件附件仍无法正常下载，是 P0 bug。
- 登录态下点击附件必须下载真实文件。
- 文件名、Content-Type、大小、内容正确。
- 至少测 PDF、图片、Office/Excel、中文文件名。

9. 邮件列表 message-first
- 拆掉“按线程归集”作为主列表展示。
- 邮件列表按单封 `crm_email_message` 独立显示。
- Thread 可作为后台关联/回复链保留，但 UI 主列表不要聚合成 thread。

## 批次划分

### 第一批（P0：数据完整 + 可处理邮件）

目标：先让邮件列表可用、完整、可筛、可下载。

范围：
1. 垃圾邮件箱同步与文件夹识别。
2. 邮件列表改 message-first 单封邮件显示。
3. 邮件菜单 inbox unread badge。
4. 快速筛选：全部、未关联、已关联、未读、已读。
5. 邮件列表关联状态改为显示“已关联”。
6. 附件下载修复。

验收：
- 可看到收件箱、垃圾邮件等文件夹。
- 垃圾邮件箱中能看到同步到的 spam/junk 邮件。
- 邮件列表每行是单封邮件，不再按线程聚合。
- 邮件菜单显示 inbox 未读数，且和未读筛选一致。
- 快筛标签可切换并刷新列表。
- 已关联邮件显示正向“已关联/客户名”状态。
- 附件登录态点击能下载真实文件。
- 无新增 console error。
- CI 成功、GHCR 镜像部署生产，健康检查通过。

可能改动：
- Backend：message list API、message count API、folder sync mapping、attachment handler/import storage。
- Frontend：CRM email page list model、folder rail、sidebar badge、filters、attachment click。
- DB：如现有字段不足，新增兼容迁移；优先复用现有 status/folder/mailbox/message 字段。

### 第二批（P1：效率操作 + 关联体验 + 导航整理）

目标：补齐批量处理、弹窗信息和导航结构。

范围：
1. 搜索栏右侧三点菜单。
2. 全部标为已读、全部标为未读。
3. 全选/多选模式、checkbox。
4. 批量操作下拉：归档、星标、移入废纸篓、标为已读/未读。
5. 关联客户弹窗增强：基本信息 + 客户详情按钮 + 调整关联菜单合并。
6. 关联联系人弹窗增强：联系人基本信息 + 联系人详情按钮。
7. 邮件详情页移除独立“打开客户”按钮。
8. CRM 菜单改“仪表盘”，路由改 `/crm/dashboard`，保留 `/crm` 兼容重定向。

验收：
- 三点菜单显示且操作生效。
- 多选后 checkbox 和批量操作栏出现。
- 批量归档/星标/移入废纸篓/已读未读更新 DB + IMAP 状态或至少 DB 状态，并记录不能同步 IMAP 的明确原因。
- 关联客户弹窗信息足够判断是否关联。
- 客户详情/联系人详情能打开正确详情页。
- CRM 仪表盘路由可访问，旧 `/crm` 不断链。
- CI 成功、GHCR 镜像部署生产，健康检查通过。

## 开发注意

- message-first 是第一批核心，不要只把 thread UI 改名。
- 批量操作对象应按 message id 设计，不能再绑死 thread id。
- folder/mailbox 不要混淆：mailbox 是邮箱地址/账户，folder/status 是文件夹/状态。
- 归档/垃圾箱/星标/已读未读要尽量对齐 IMAP flags/move；如果某服务商文件夹不可识别，要 fallback 到 DB 状态并在 UI/日志清楚显示。
- 附件下载不能只验证 401/路由命中；必须验证登录态真实下载。
- 大规模改动必须分批 commit，第一批部署稳定后再做第二批。
