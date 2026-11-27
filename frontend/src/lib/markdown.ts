import { parse } from "marked";
import DOMPurify from "dompurify";
import { processAutoLinks } from "$lib/autolink";
import { resolveAttachmentRefs } from "$lib/resolve-attachment-refs";
import type { AutoLinkContext } from "$lib/autolink";

export interface RenderMarkdownOptions {
  attachmentsMap?: Map<number, string>;
  autoLinkCtx?: AutoLinkContext;
}

/**
 * Renders Markdown to sanitized HTML. Single entry point for Markdown
 * rendering: the marked options below are the only place parsing behaviour
 * is defined.
 */
export function renderMarkdown(
  text: string,
  options: RenderMarkdownOptions = {},
): string {
  if (!text) return "";
  const { attachmentsMap = new Map<number, string>(), autoLinkCtx = {} } = options;
  return DOMPurify.sanitize(
    parse(
      processAutoLinks(
        resolveAttachmentRefs(text, attachmentsMap),
        autoLinkCtx,
      ),
      { async: false, breaks: true },
    ) as string,
  );
}
