"use client";

import { Mail, SlidersHorizontal, Users } from "lucide-react";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@multica/ui/components/ui/sidebar";
import { useCRMWorkspacePaths } from "@multica/core/crm/paths";
import { AppLink } from "../navigation";
import { useT } from "../i18n";

function isNavActive(pathname: string, href: string): boolean {
  return pathname === href || pathname.startsWith(href + "/");
}

export function CRMSidebarNavGroup({ pathname }: { pathname: string }) {
  const paths = useCRMWorkspacePaths();
  const { t } = useT("crm");
  const items = [
    { key: "dashboard", href: paths.dashboard(), label: t(($) => $.dashboard.title), icon: Users },
    { key: "customers", href: paths.customers(), label: t(($) => $.customers.title), icon: Users },
    { key: "emails", href: paths.emails(), label: t(($) => $.tabs.emails), icon: Mail },
    { key: "ai-settings", href: paths.aiSettings(), label: t(($) => $.dashboard.ai_settings), icon: SlidersHorizontal },
  ];

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{t(($) => $.common.module)}</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu className="gap-0.5">
          {items.map((item) => {
            const Icon = item.icon;
            return (
              <SidebarMenuItem key={item.key}>
                <SidebarMenuButton
                  isActive={isNavActive(pathname, item.href)}
                  render={<AppLink href={item.href} />}
                  className="text-muted-foreground hover:not-data-active:bg-sidebar-accent/70 data-active:bg-sidebar-accent data-active:text-sidebar-accent-foreground"
                >
                  <Icon />
                  <span>{item.label}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            );
          })}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
