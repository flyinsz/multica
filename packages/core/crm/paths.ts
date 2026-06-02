"use client";

import { useRequiredWorkspaceSlug } from "../paths";

const encode = (id: string) => encodeURIComponent(id);

function workspaceScoped(slug: string) {
  const ws = `/${encode(slug)}`;
  return {
    dashboard: () => `${ws}/crm/dashboard`,
    customers: () => `${ws}/crm/customers`,
    customerDetail: (id: string) => `${ws}/crm/customers/${encode(id)}`,
    emails: () => `${ws}/crm/emails`,
    aiSettings: () => `${ws}/crm/ai-settings`,
  };
}

export const crmPaths = {
  workspace: workspaceScoped,
};

export type CRMWorkspacePaths = ReturnType<typeof workspaceScoped>;

export function useCRMWorkspacePaths(): CRMWorkspacePaths {
  const slug = useRequiredWorkspaceSlug();
  return crmPaths.workspace(slug);
}
