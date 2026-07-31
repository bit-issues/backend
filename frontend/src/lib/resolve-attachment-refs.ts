/**
 * Replaces Markdown image references using attachment IDs with actual presigned S3 URLs.
 *
 * Transforms: ![alt text](attachment:123)
 * Into:       ![alt text](<presigned-s3-url>)
 *
 * URLs are resolved from the attachmentsMap (id -> presigned download URL).
 * This avoids sending JWTs in <img> src attributes — the presigned S3 URL
 * is a time-limited, resource-specific URL that requires no auth header.
 *
 * If the attachment ID is not found in the map, the reference is left
 * unresolved (renders as broken image / alt text) rather than silently failing.
 */
export function resolveAttachmentRefs(
  text: string,
  attachmentsMap: Map<number, string>,
): string {
  if (!attachmentsMap.size) return text;
  return text.replace(
    /!\[([^\]]*)\]\(attachment:(\d+)\)/g,
    (_match, alt, id) => {
      const url = attachmentsMap.get(Number(id));
      // angle-bracket destination so presigned URLs containing `)` or
      // whitespace are not parsed as malformed Markdown image destinations
      if (url) return `![${alt}](<${url}>)`;
      return _match; // leave unresolved if attachment not found
    },
  );
}
