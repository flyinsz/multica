"use client";

/**
 * MentionView — NodeView for rendering @mentions inline in the editor.
 *
 * Member/agent mentions: plain "@Name" text with .mention class styling.
 * Issue/project mentions: chips inside custom <a> tags that support cmd/shift-click
 * to open in a new tab (AppLink doesn't expose that intent hook).
 * CRM mentions: draft/account/contact chips route to CRM-owned pages.
 */

import { NodeViewWrapper } from "@tiptap/react";
import type { NodeViewProps } from "@tiptap/react";
import { useWorkspacePaths } from "@multica/core/paths";
import { useNavigation } from "../../navigation";
import { IssueChip } from "../../issues/components/issue-chip";
import { ProjectChip } from "../../projects/components/project-chip";

export function MentionView({ node }: NodeViewProps) {
  const { type, id, label } = node.attrs;

  if (type === "issue") {
    return (
      <NodeViewWrapper as="span" className="inline">
        <IssueMention issueId={id} fallbackLabel={label} />
      </NodeViewWrapper>
    );
  }

  if (type === "project") {
    return (
      <NodeViewWrapper as="span" className="inline">
        <ProjectMention projectId={id} fallbackLabel={label} />
      </NodeViewWrapper>
    );
  }

  if (type === "crm-draft") {
    return (
      <NodeViewWrapper as="span" className="inline">
        <CRMDraftMention draftId={id} fallbackLabel={label} />
      </NodeViewWrapper>
    );
  }

  if (type === "crm-account") {
    return (
      <NodeViewWrapper as="span" className="inline">
        <CRMCustomerMention customerId={id} fallbackLabel={label} />
      </NodeViewWrapper>
    );
  }

  if (type === "crm-contact") {
    return (
      <NodeViewWrapper as="span" className="inline">
        <CRMContactMention contactId={id} fallbackLabel={label} />
      </NodeViewWrapper>
    );
  }

  return (
    <NodeViewWrapper as="span" className="inline">
      <span className="mention">@{label ?? id}</span>
    </NodeViewWrapper>
  );
}

function CRMCustomerMention({ customerId, fallbackLabel }: { customerId: string; fallbackLabel?: string }) {
  const p = useWorkspacePaths();
  const { push, openInNewTab } = useNavigation();
  const customerPath = p.crmCustomer(customerId);
  const label = fallbackLabel || `Customer ${customerId.slice(0, 8)}`;
  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.metaKey || e.ctrlKey || e.shiftKey) {
      if (openInNewTab) openInNewTab(customerPath, label);
      return;
    }
    push(customerPath);
  };
  return (
    <a href={customerPath} onClick={handleClick} className="issue-mention inline-flex rounded border px-1.5 py-0.5 text-xs hover:bg-accent transition-colors">
      🏢 {label}
    </a>
  );
}

function CRMContactMention({ contactId, fallbackLabel }: { contactId: string; fallbackLabel?: string }) {
  const p = useWorkspacePaths();
  const { push, openInNewTab } = useNavigation();
  const contactPath = p.crmContact(contactId);
  const label = fallbackLabel || `Contact ${contactId.slice(0, 8)}`;
  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.metaKey || e.ctrlKey || e.shiftKey) {
      if (openInNewTab) openInNewTab(contactPath, label);
      return;
    }
    push(contactPath);
  };
  return (
    <a href={contactPath} onClick={handleClick} className="issue-mention inline-flex rounded border px-1.5 py-0.5 text-xs hover:bg-accent transition-colors">
      👤 {label}
    </a>
  );
}

function CRMDraftMention({ draftId, fallbackLabel }: { draftId: string; fallbackLabel?: string }) {
  const p = useWorkspacePaths();
  const { push, openInNewTab } = useNavigation();
  const draftPath = p.crmEmailDraft(draftId);
  const label = fallbackLabel || `Draft ${draftId.slice(0, 8)}`;

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.metaKey || e.ctrlKey || e.shiftKey) {
      if (openInNewTab) openInNewTab(draftPath, label);
      return;
    }
    push(draftPath);
  };

  return (
    <a href={draftPath} onClick={handleClick} className="issue-mention inline-flex rounded border px-1.5 py-0.5 text-xs hover:bg-accent transition-colors">
      ✉️ {label}
    </a>
  );
}

function ProjectMention({ projectId, fallbackLabel }: { projectId: string; fallbackLabel?: string }) {
  const p = useWorkspacePaths();
  const { push, openInNewTab } = useNavigation();
  const projectPath = p.projectDetail(projectId);

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.metaKey || e.ctrlKey || e.shiftKey) {
      if (openInNewTab) openInNewTab(projectPath, fallbackLabel);
      return;
    }
    push(projectPath);
  };

  return (
    <a href={projectPath} onClick={handleClick} className="project-mention inline-flex">
      <ProjectChip
        projectId={projectId}
        fallbackLabel={fallbackLabel}
        className="cursor-pointer hover:bg-accent transition-colors"
      />
    </a>
  );
}

function IssueMention({ issueId, fallbackLabel }: { issueId: string; fallbackLabel?: string }) {
  const p = useWorkspacePaths();
  const { push, openInNewTab } = useNavigation();
  const issuePath = p.issueDetail(issueId);

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.metaKey || e.ctrlKey || e.shiftKey) {
      if (openInNewTab) openInNewTab(issuePath, fallbackLabel);
      return;
    }
    push(issuePath);
  };

  return (
    <a href={issuePath} onClick={handleClick} className="issue-mention inline-flex">
      <IssueChip
        issueId={issueId}
        fallbackLabel={fallbackLabel}
        className="cursor-pointer hover:bg-accent transition-colors"
      />
    </a>
  );
}
