"use client";

import * as React from "react";
import {
  Markdown as MarkdownBase,
  type MarkdownProps as MarkdownBaseProps,
  type RenderMode,
} from "@multica/ui/markdown";
import { useConfigStore } from "@multica/core/config";
import type { Attachment as AttachmentRecord } from "@multica/core/types";
import { useWorkspacePaths } from "@multica/core/paths";
import { IssueMentionCard } from "../issues/components/issue-mention-card";
import { useNavigation } from "../navigation";
import {
  Attachment as AttachmentRenderer,
  AttachmentDownloadProvider,
} from "../editor";

export type { RenderMode };

export interface MarkdownProps extends MarkdownBaseProps {
  /**
   * Attachments associated with the surrounding entity (chat message, skill
   * file). When passed, the renderer resolves inline image / file-card URLs
   * to full attachment records via AttachmentDownloadProvider, unlocking the
   * unified hover toolbar / lightbox / preview-modal behavior used in
   * editor surfaces.
   */
  attachments?: AttachmentRecord[];
}

/**
 * Default renderMention that delegates to IssueMentionCard for issue mentions
 * and renders a styled span for other mention types.
 */
function defaultRenderMention({
  type,
  id,
  label,
}: {
  type: string;
  id: string;
  label?: string;
}): React.ReactNode {
  if (type === "issue") {
    return <IssueMentionCard issueId={id} />;
  }
  if (type === "crm-draft") {
    return <CRMDraftMention draftId={id} />;
  }
  if (type === "crm-account") {
    return <CRMCustomerMention customerId={id} label={label} />;
  }
  if (type === "crm-contact") {
    return <CRMContactMention contactId={id} label={label} />;
  }
  return null;
}

function CRMCustomerMention({ customerId, label }: { customerId: string; label?: string }): React.JSX.Element {
  const p = useWorkspacePaths();
  const { push, openInNewTab } = useNavigation();
  const customerPath = p.crmCustomer(customerId);
  const displayLabel = label || `Customer ${customerId.slice(0, 8)}`;
  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.metaKey || e.ctrlKey || e.shiftKey) {
      if (openInNewTab) openInNewTab(customerPath, displayLabel);
      return;
    }
    push(customerPath);
  };
  return (
    <a href={customerPath} onClick={handleClick} className="issue-mention inline-flex rounded border px-1.5 py-0.5 text-xs hover:bg-accent transition-colors">
      🏢 {displayLabel}
    </a>
  );
}

function CRMContactMention({ contactId, label }: { contactId: string; label?: string }): React.JSX.Element {
  const p = useWorkspacePaths();
  const { push, openInNewTab } = useNavigation();
  const contactPath = p.crmContact(contactId);
  const displayLabel = label || `Contact ${contactId.slice(0, 8)}`;
  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.metaKey || e.ctrlKey || e.shiftKey) {
      if (openInNewTab) openInNewTab(contactPath, displayLabel);
      return;
    }
    push(contactPath);
  };
  return (
    <a href={contactPath} onClick={handleClick} className="issue-mention inline-flex rounded border px-1.5 py-0.5 text-xs hover:bg-accent transition-colors">
      👤 {displayLabel}
    </a>
  );
}

function CRMDraftMention({ draftId }: { draftId: string }): React.JSX.Element {
  const p = useWorkspacePaths();
  const { push, openInNewTab } = useNavigation();
  const draftPath = p.crmEmailDraft(draftId);
  const label = `Draft ${draftId.slice(0, 8)}`;

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
    <a
      href={draftPath}
      onClick={handleClick}
      className="issue-mention inline-flex rounded border px-1.5 py-0.5 text-xs hover:bg-accent transition-colors"
    >
      ✉️ {label}
    </a>
  );
}

function renderImage({ src, alt }: { src: string; alt: string }): React.ReactNode {
  return (
    <AttachmentRenderer
      attachment={{
        kind: "url",
        url: src,
        filename: alt,
        // chat / skill markdown `![]()` is structurally an image. Without
        // forceKind, empty/descriptive alt strings would route to the
        // file-card chrome via getPreviewKind autodetect.
        forceKind: "image",
      }}
    />
  );
}

function renderFileCard({
  href,
  filename,
}: {
  href: string;
  filename: string;
}): React.ReactNode {
  return (
    <AttachmentRenderer
      attachment={{ kind: "url", url: href, filename }}
    />
  );
}

/**
 * App-level Markdown wrapper. Injects:
 *   - IssueMentionCard for issue mentions
 *   - cdnDomain from the config store (drives fileCard preprocessing)
 *   - unified <Attachment> as the image / file-card renderer
 *   - AttachmentDownloadProvider so url → record resolution works inside
 *     the injected <Attachment> components
 */
export function Markdown(props: MarkdownProps): React.JSX.Element {
  const cdnDomain = useConfigStore((s) => s.cdnDomain);
  const { attachments, ...rest } = props;
  return (
    <AttachmentDownloadProvider attachments={attachments}>
      <MarkdownBase
        renderMention={defaultRenderMention}
        renderImage={renderImage}
        renderFileCard={renderFileCard}
        cdnDomain={cdnDomain}
        {...rest}
      />
    </AttachmentDownloadProvider>
  );
}

export const MemoizedMarkdown = React.memo(Markdown);
MemoizedMarkdown.displayName = "MemoizedMarkdown";
