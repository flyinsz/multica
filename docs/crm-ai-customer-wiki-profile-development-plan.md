# CRM AI + Customer Wiki/Profile Development Plan

> Scope guard: keep changes inside CRM customization layer as much as possible. Avoid Multica native/core non-CRM files unless explicitly approved.
> Build guard: do not run local `go test`, `go build`, frontend build, Docker image build, or other heavy local builds on the shared server. Use GitHub Actions/GHCR for verification when needed.

## 1. Goal

Convert CRM AI from scattered feature-specific context loading into a shared customer profile/wiki and AI context layer.

Current pattern:

```text
Each AI feature scans emails independently -> builds its own prompt -> calls LLM
```

Target pattern:

```text
Email / future WhatsApp / CRM data -> Customer Wiki/Profile -> AI Context Builder -> CRM AI features
```

Expected benefits:

- Lower token usage.
- Faster AI workflows.
- Better customer background accuracy.
- Less cross-email context leakage.
- Better recipient matching.
- Shared foundation for WhatsApp and future channels.

## 2. Non-goals / constraints

- Do not create standalone markdown files per customer as source of truth.
- Do not replace existing CRM tables when extension is enough.
- Do not auto-send customer emails.
- Do not use LLM for fast recipient lookup when deterministic/fuzzy matching is enough.
- Do not modify Multica non-CRM core unless approved.
- Do not run local Go tests/builds or frontend builds on shared server.

## 3. Data architecture

### 3.1 Reuse existing source tables

Core existing tables stay authoritative:

- `crm_account`
- `crm_contact`
- `crm_account_profile`
- `crm_email_thread`
- `crm_email_message`
- `crm_ai_setting`
- `issue`
- `project`
- `crm_entity_link`

### 3.2 Upgrade customer profile into wiki-style structured profile

Prefer extending `crm_account_profile.profile_json` before adding new tables.

Target JSON contract:

```json
{
  "summary": "",
  "business_background": "",
  "communication_summary": "",
  "open_issues": [],
  "risks": [],
  "preferences": [],
  "buying_signals": [],
  "follow_up_recommendation": {
    "next_follow_up_at": "",
    "reason": "",
    "confidence": "medium",
    "source_refs": []
  },
  "last_interactions": [],
  "aliases": [],
  "keywords": [],
  "contacts": [],
  "source_refs": [],
  "confidence": "medium",
  "last_refreshed_at": ""
}
```

### 3.3 Add customer alias/search index if current data is insufficient

Planned table: `crm_customer_alias`.

Fields:

- `id`
- `workspace_id`
- `account_id`
- `contact_id`
- `alias`
- `alias_normalized`
- `alias_type`: `email`, `email_prefix`, `contact_name`, `nickname`, `company_name`, `company_short_name`, `domain`, `manual`, `ai_extracted`, `whatsapp_name`, `whatsapp_number`
- `weight`
- `source_type`: `account`, `contact`, `email`, `whatsapp`, `profile`, `manual`
- `source_id`
- `confidence`
- `created_at`
- `updated_at`

Purpose:

- Fast recipient lookup.
- Customer/contact fuzzy search.
- Future WhatsApp identity matching.

### 3.4 Add or emulate unified interaction layer

Future target: `crm_interaction` abstraction.

Minimum first step can be an adapter/view over existing email tables.

Normalized fields:

- `id`
- `workspace_id`
- `account_id`
- `contact_id`
- `channel`: `email`, `whatsapp`, `manual_note`, `call`
- `source_id`
- `direction`: `inbound`, `outbound`
- `occurred_at`
- `subject`
- `body_text`
- `body_summary`
- `language`
- `sentiment`
- `intent`
- `source_refs`

## 4. AI Context Builder

Create shared CRM-only context builder.

Input:

```json
{
  "workspace_id": "",
  "account_id": "",
  "contact_id": "",
  "thread_id": "",
  "interaction_id": "",
  "function_name": "",
  "context_budget": 0
}
```

Output:

```json
{
  "customer_profile": {},
  "risk_notes": [],
  "open_issues": [],
  "preferences": [],
  "recent_interactions": [],
  "current_thread": [],
  "missing_info": [],
  "source_refs": []
}
```

Priority:

1. Structured customer profile.
2. Current thread/current interaction.
3. Recent key interaction summaries.
4. Raw message snippets only when necessary.
5. Attachments/OCR only on explicit need.

## 5. Recipient lookup redesign

### 5.1 Remove LLM keyword extraction from recipient lookup

Problem example:

```text
给yesdonny发邮件问一下近况
```

Bad result observed:

```text
yesdonny 近况 收件人关键词: yesdonny；需求: 问近况 yesdonny
```

Target: deterministic extraction returns only recipient term `yesdonny` and leaves intent separate.

### 5.2 Local parser rules

Extract recipient terms using local rules:

- `给(.+?)发邮件`
- `发给(.+?)`
- `to\s+(.+?)`
- email regex
- domain extraction
- English/number token extraction
- short Chinese name candidates

Return:

```json
{
  "recipient_terms": ["yesdonny"],
  "intent": "问近况"
}
```

### 5.3 Search strategy

Search in order:

1. Exact email.
2. Email prefix.
3. Contact name.
4. Account name.
5. Alias index.
6. Domain.
7. Fuzzy fallback.

Return candidates with:

- account
- contact
- email
- score
- match_reason
- source

### 5.4 UI behavior

- Show candidate popup for user confirmation.
- Multiple candidates require selection.
- High-confidence single candidate may be preselected but not auto-sent.

## 6. CRM AI feature changes

### 6.1 AI compose email

Flow:

```text
User input -> local recipient parser -> candidate popup -> AI Context Builder -> draft generation
```

Rules:

- No signature or signature placeholder in generated body.
- Default signature stays from compose box.
- New mail may generate subject.
- Reply preserves `Re:`.

### 6.2 AI reply assistance

Flow:

```text
Current draft/thread -> AI Context Builder -> background/risk/advice cards -> user refinements -> suggestion generation
```

Rules:

- Do not use unrelated selected thread as fallback.
- Background/risk info has no accept/adopt button.
- Only generated reply suggestion has accept/adopt.
- Multi-turn generation includes prior suggestion and user feedback.

### 6.3 Pending-reply patrol

Flow:

```text
Cheap SQL gate -> latest inbound unresolved -> profile + current thread context -> create review issue
```

Rules:

- Do not paste large history into issue.
- Include concise source refs.
- Request issue workflow to generate draft.
- Do not auto-send.

### 6.4 Due-followup patrol

Flow:

```text
next_follow_up_at due -> check recent email/future WhatsApp interactions -> profile context -> create follow-up issue if still needed
```

Rules:

- Avoid duplicates for same due window.
- Existing done issues may mean stale due date already handled.
- Do not rely only on stored `next_follow_up_at`; when profile/emails show customer already replied, paused, or needs different cadence, request LLM recommendation and update follow-up date.
- LLM recommendation must return `next_follow_up_at`, `reason`, `confidence`, and `source_refs`; low confidence creates human-review issue instead of silently changing date.
- Recommend channel in future: email/WhatsApp/both.

### 6.4.1 LLM-assisted follow-up date recommendation

Flow:

```text
profile refresh / due-followup review -> recent interactions + current follow-up state -> LLM suggests next_follow_up_at -> persist with reason/source refs
```

Rules:

- Use after profile refresh and due-followup patrol review.
- Consider latest inbound/outbound status, open issues, customer intent, buying signals, risk, and no-reply duration.
- If customer replied recently, push `next_follow_up_at` forward or clear/adjust due state according to profile judgment.
- If outbound follow-up was sent and no reply, choose next cadence based on context instead of fixed interval.
- Never create customer-facing email automatically; only update CRM follow-up metadata or create internal review issue.

### 6.5 New-activity profile refresh

Triggers:

- inbound email imported
- outbound email sent
- future WhatsApp inbound/outbound
- future manual note

Rules:

- Incremental update only.
- Update aliases/keywords/risks/open issues/preferences.
- Preserve source refs.

### 6.6 Daily profile refresh

Flow:

```text
cheap filter -> limited batch -> structured profile refresh
```

Refresh candidates:

- stale profile
- recent new interactions
- high priority account
- open issue
- overdue follow-up
- low profile confidence

## 7. AI settings improvements

Group settings:

1. Model configuration.
2. Context strategy.
3. Customer Wiki/Profile.
4. Mail AI.
5. Patrol tasks.
6. WhatsApp reserved settings.
7. Run history.

Feature config shape:

```json
{
  "enabled": true,
  "model_tier": "normal",
  "temperature": 0.2,
  "max_tokens": 1200,
  "use_customer_profile": true,
  "use_recent_interactions": true,
  "max_interactions": 8,
  "include_whatsapp": true,
  "require_human_confirm": true,
  "timeout_seconds": 60
}
```

Global guardrails:

- Do not fabricate price, lead time, stock, certifications, attachments, or prior promises.
- Do not auto-send customer emails.
- Do not auto-resolve ambiguous recipients.
- Do not generate signatures/sign-offs when default signature exists.
- Low confidence must request human confirmation.
- Keep Email/WhatsApp source labels.

## 8. Phased implementation plan

### Phase 1: recipient lookup without LLM

Acceptance:

- `给yesdonny发邮件问一下近况` extracts recipient term `yesdonny` only.
- Search is fast and deterministic.
- Candidate popup works.
- No signature placeholder is generated.

### Phase 2: structured customer wiki/profile schema

Acceptance:

- Profile JSON follows fixed schema.
- Customer detail shows summary/background/preferences/risks/open issues/aliases/source refs.
- Profile refresh outputs structured JSON.

### Phase 3: CRM AI Context Builder

Acceptance:

- Compose/reply AI use shared builder.
- Profile is preferred over long raw email history.
- Current thread context does not leak across selected emails.

### Phase 4: patrol integration

Acceptance:

- Pending-reply uses profile + current thread context.
- Due-followup uses profile + recent interactions.
- Profile refresh/due-followup can ask LLM for `next_follow_up_at` recommendation with reason/source refs.
- Follow-up date update respects reply/no-reply state and avoids stale due loops.
- New-activity refresh is incremental.
- Daily refresh is limited and budget-aware.

### Phase 5: WhatsApp-ready interaction layer

Acceptance:

- Email still works unchanged.
- Context builder can include future `channel=whatsapp` interactions.
- Alias matching supports phone/WhatsApp names.

## 9. Progress checklist

- [x] Save this plan in repo.
- [x] Inspect current CRM compose/reply AI implementation.
- [x] Inspect current profile refresh implementation.
- [x] Inspect current CRM AI settings implementation.
- [x] Implement local recipient parser.
- [x] Implement/extend CRM recipient candidate search endpoint.
- [x] Wire compose UI candidate popup.
- [x] Remove recipient lookup LLM path.
- [x] Add structured profile schema helpers.
- [x] Add/extend profile UI display.
- [x] Add context builder service.
- [x] Wire compose/reply AI to context builder.
- [x] Update pending-reply patrol context.
- [x] Update due-followup patrol context.
- [x] Add LLM follow-up date recommendation to profile refresh.
- [x] Add backend path to update `next_follow_up_at` with LLM reason/source refs.
- [x] Ensure due-followup patrol checks latest reply/no-reply state before creating issue.
- [x] Add low-confidence follow-up recommendation human-review issue path.
- [x] Update new-activity profile refresh.
- [x] Update daily profile refresh.
- [x] Add WhatsApp-ready interaction abstraction.
- [x] Run lightweight checks only (`git diff --check`, static inspection).
- [ ] Push branch and use GitHub Actions/GHCR for verification.
