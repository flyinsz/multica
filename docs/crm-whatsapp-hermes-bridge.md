# CRM WhatsApp Provider: Hermes Bridge

Status: planned implementation
Owner goal: customer WhatsApp communication management in Multica CRM, reusing Hermes WhatsApp bridge instead of rebuilding WhatsApp protocol handling.

## Goals

- Sync WhatsApp customer communication into Multica CRM.
- Preserve traceable records for inbound and outbound messages, including messages sent from mobile phone.
- Link WhatsApp chats/messages to CRM accounts and contacts.
- Let high-risk or actionable WhatsApp messages create Multica issues.
- Support AI summary/reply drafting using CRM profile, email history, WhatsApp history, and workspace documents.
- Keep WhatsApp as customer communication channel, not email approval channel.

## Non-goals

- Do not use WhatsApp as email approval channel.
- Do not put CRM business logic inside Hermes.
- Do not make Hermes business data source of truth.
- Do not directly let AI auto-send WhatsApp messages to customers.
- Do not rely on browser draft boxes.

## Architecture

```text
WhatsApp mobile / WhatsApp Web
  -> Hermes WhatsApp Bridge (Baileys, protocol/session/send/receive)
  -> Multica CRM Hermes Provider (sync/webhook/send)
  -> Multica CRM DB (threads/messages/customer links/issues)
  -> CRM UI / AI / issue workflow
```

Hermes owns:

- WhatsApp login/session.
- Baileys connection.
- Message receive/send/media handling.
- Thin CRM HTTP API.
- Message store for bridge replay and history sync.

Multica owns:

- CRM accounts/contacts.
- WhatsApp threads/messages as business records.
- Customer/contact matching.
- Issue creation and review workflow.
- AI summary/draft/risk classification.
- Workspace document injection.
- UI and audit logs.

## Current Hermes audit

Runtime process:

```text
node /root/.hermes/custom/whatsapp-bridge-proxy.js --port 3000 --session /root/.hermes/whatsapp/session --mode bot
```

Health:

```http
GET http://127.0.0.1:3000/health
```

Existing endpoints:

- `GET /messages` queue-drains new incoming events only.
- `POST /send`
- `POST /edit`
- `POST /send-media`
- `POST /typing`
- `GET /chat/:id`
- `GET /health`

Gaps:

- No durable message store.
- No chat list API.
- No per-chat history API.
- No CRM webhook push.
- `/messages` drains queue, unsuitable as CRM sync source.
- Current Baileys socket has `syncFullHistory: false`.
- Existing bridge filters some `fromMe` messages; CRM mode must persist mobile outbound messages.
- Send endpoint lacks idempotency key.

## Hermes CRM API contract

Add endpoints without breaking existing Hermes gateway endpoints.

### Status

`GET /crm/status`

```json
{
  "ok": true,
  "connected": true,
  "provider": "hermes_baileys",
  "mode": "bot",
  "account_id": "default",
  "phone_number": "...",
  "display_name": "...",
  "queue_length": 0,
  "uptime": 123
}
```

### Chats

`GET /crm/chats?limit=100&cursor=...`

```json
{
  "chats": [
    {
      "chat_id": "123@s.whatsapp.net",
      "title": "Customer Name",
      "phone_number": "+123",
      "is_group": false,
      "last_message_at": "2026-06-27T10:00:00Z",
      "last_message_text": "hello",
      "unread_count": 0
    }
  ],
  "next_cursor": ""
}
```

### Messages

`GET /crm/chats/:chatId/messages?limit=100&before=...`

```json
{
  "messages": [
    {
      "message_id": "ABCDEF",
      "chat_id": "123@s.whatsapp.net",
      "direction": "inbound",
      "from": "+123",
      "to": "+8613...",
      "body_text": "hello",
      "timestamp": "2026-06-27T10:00:00Z",
      "media": [],
      "raw": {}
    }
  ]
}
```

### Send

`POST /crm/send`

```json
{
  "chat_id": "123@s.whatsapp.net",
  "to": "+123",
  "body_text": "hello",
  "idempotency_key": "uuid"
}
```

Response:

```json
{
  "ok": true,
  "message_id": "ABCDEF",
  "status": "sent"
}
```

### Webhook to Multica

Hermes posts:

```http
POST http://127.0.0.1:18080/api/crm/whatsapp/hermes/webhook
```

Payload:

```json
{
  "event": "message.upsert",
  "provider": "hermes_baileys",
  "account_id": "default",
  "message": {
    "message_id": "ABCDEF",
    "chat_id": "123@s.whatsapp.net",
    "direction": "inbound",
    "from": "+123",
    "to": "+8613...",
    "body_text": "hello",
    "timestamp": "2026-06-27T10:00:00Z",
    "media": [],
    "raw": {}
  }
}
```

## Hermes store requirements

Use lightweight durable store in Hermes bridge, e.g. SQLite under `/root/.hermes/whatsapp/crm-bridge.sqlite`.

Tables concept:

- `crm_bridge_messages`
  - `message_id`
  - `chat_id`
  - `direction`
  - `from_id`
  - `to_id`
  - `body_text`
  - `media_json`
  - `timestamp_ms`
  - `raw_json`
  - unique `(chat_id, message_id)`
- `crm_bridge_sends`
  - `idempotency_key`
  - `chat_id`
  - `message_id`
  - `status`
  - `created_at`

Store both inbound and outbound, including `fromMe=true` mobile-origin messages. Continue existing `/messages` behavior for Hermes gateway compatibility.

## Multica DB design

Tables:

- `crm_whatsapp_account`
  - `id`, `workspace_id`, `provider`, `provider_account_id`, `display_name`, `phone_number`, `status`, `config`, `last_sync_at`, timestamps
- `crm_whatsapp_thread`
  - `id`, `workspace_id`, `whatsapp_account_id`, `external_chat_id`, `title`, `phone_number`, `account_id`, `contact_id`, `last_message_at`, `unread_count`, timestamps
- `crm_whatsapp_message`
  - `id`, `workspace_id`, `thread_id`, `external_message_id`, `direction`, `from_number`, `to_number`, `body_text`, `media`, `sent_at`, `received_at`, `raw`, timestamps
  - unique `(workspace_id, thread_id, external_message_id)`
- `crm_whatsapp_issue_link`
  - `id`, `workspace_id`, `message_id`, `issue_id`, `reason`, timestamps

## Contact matching

Priority:

1. `crm_contact.whatsapp_id` exact.
2. `crm_contact.phone` normalized.
3. Same account contact phone normalized.
4. Unmatched candidate.

Rules:

- Do not auto-bind uncertain contacts.
- Show suggestions and allow manual confirm.
- Store confidence/reason when possible.

## Issue creation rules

Create or suggest issue when WhatsApp message indicates:

- quotation request
- delivery/order change
- complaint
- payment/logistics/certification risk
- urgent follow-up
- unknown customer business inquiry
- AI confidence high that human handling needed
- user clicks “转 issue”

Issue content template:

```text
来源：WhatsApp
客户：{{account/contact or unmatched phone}}
线程：{{thread link}}
消息时间：{{message time}}

客户原文：
{{message body}}

AI 摘要：
{{summary}}

建议处理：
{{suggested action}}

风险点：
{{risks}}
```

Issue number rule:

- Use/sync `workspace.issue_counter`.
- Never raw `MAX(number)+1`.

## AI context

WhatsApp AI draft/classification should use:

- current WhatsApp thread
- CRM customer profile/wiki
- email history summary
- open issues and recent follow-ups
- pinned and matched workspace documents
- user instruction

AI output:

- Chinese internal summary
- customer-language reply draft
- risk points
- suggested issue creation flag
- referenced document paths for internal display

## Rollout plan

1. Save this plan.
2. Hermes bridge: add `/crm/*`, durable store, webhook, send idempotency.
3. Multica backend: add DB migrations, provider sync, webhook receiver, message import, issue creation helper.
4. Multica UI: WhatsApp CRM inbox, customer detail tab, link contact, one-click issue.
5. Later: WhatsApp reply draft/send from CRM.

## Safety rules

- Do not run local Go test/build on production server.
- Use GitHub Actions/GHCR for Multica image builds.
- Deploy only from `/www/docker/multica`.
- Preserve CRM customizations during upstream merges.
- Do not `docker compose down -v`.
- Keep Hermes old WhatsApp endpoints compatible.
- Do not delete Camofox cache/assets/volumes.
- Do not auto-send AI WhatsApp replies.
