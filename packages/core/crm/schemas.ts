import { z } from "zod";
import type { CRMEmailEngineStatus, CRMIMAPImportResponse, CRMIMAPPreviewResponse } from "./types";

export const CRMIMAPPreviewMessageSchema = z.object({
  uid: z.string(),
  external_message_id: z.string().default(""),
  subject: z.string().default(""),
  from_email: z.string().default(""),
  from_name: z.string().default(""),
  to_emails: z.array(z.string()).default([]),
  cc_emails: z.array(z.string()).default([]),
  received_at: z.string().nullable().optional(),
  snippet: z.string().default(""),
  raw_size: z.number().default(0),
}).loose();

export const CRMIMAPPreviewResponseSchema = z.object({
  messages: z.array(CRMIMAPPreviewMessageSchema).default([]),
  total: z.number().default(0),
  limit: z.number().default(0),
  sync_enabled: z.boolean().default(false),
  note: z.string().default(""),
}).loose();

export const EMPTY_CRM_IMAP_PREVIEW_RESPONSE: CRMIMAPPreviewResponse = {
  messages: [],
  total: 0,
  limit: 0,
  sync_enabled: false,
  note: "",
};

export const CRMIMAPImportResponseSchema = z.object({
  ok: z.boolean().default(false),
  run_id: z.string().optional(),
  status: z.string().optional(),
  fetched: z.number().default(0),
  imported: z.number().default(0),
  skipped: z.number().default(0),
}).loose();

export const EMPTY_CRM_IMAP_IMPORT_RESPONSE: CRMIMAPImportResponse = {
  ok: false,
  fetched: 0,
  imported: 0,
  skipped: 0,
};

const CRMEmailEngineFolderSchema = z.object({
  path: z.string().default(""),
  name: z.string().default(""),
  special_use: z.string().nullable().optional(),
  total: z.number().default(0),
  unread: z.number().default(0),
}).loose();

export const CRMEmailEngineStatusSchema = z.object({
  enabled: z.boolean().default(false),
  configured: z.boolean().default(false),
  base_url: z.string().nullable().optional(),
  account: z.string().nullable().optional(),
  state: z.string().nullable().optional(),
  syncing: z.boolean().default(false),
  last_error: z.string().nullable().optional(),
  folders: z.array(CRMEmailEngineFolderSchema).default([]),
  fallback_provider: z.string().default("imap_smtp"),
}).loose();

export const EMPTY_CRM_EMAILENGINE_STATUS: CRMEmailEngineStatus = {
  enabled: false,
  configured: false,
  syncing: false,
  folders: [],
  fallback_provider: "imap_smtp",
};
