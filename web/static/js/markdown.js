(function initMarkdownRenderer() {
  if (typeof marked !== "undefined" && typeof marked.use === "function") {
    marked.use({ breaks: true, gfm: true });
  }

  const parseMarkdown =
    typeof marked !== "undefined" && marked && typeof marked.parse === "function"
      ? (src) => marked.parse(src)
      : null;

  const sanitizeHtml =
    typeof DOMPurify !== "undefined" && DOMPurify && typeof DOMPurify.sanitize === "function"
      ? (html) =>
        DOMPurify.sanitize(html, {
          USE_PROFILES: { html: true },
          ADD_ATTR: ["target", "rel", "class"],
        })
      : null;

  const escapeHtml = (value) =>
    String(value ?? "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");

  const emptyPlaceholder = '<p class="text-slate-500">—</p>';

  const CSET_RE = /<<cset\s+([a-f0-9]{7,40})\s*>>/gi;

  const linkifyCset = (text, repoUrl) => {
    if (!repoUrl) return text;
    const baseUrl = String(repoUrl).replace(/\/+$/, "");
    return text.replace(CSET_RE, (_, hash) =>
      `<a href="${baseUrl}/commits/${hash}" target="_blank" rel="noopener noreferrer" class="font-mono text-xs underline decoration-dotted">${hash}</a>`,
    );
  };

  window.renderMarkdown = function renderMarkdown(raw, { emptyHtml = emptyPlaceholder, repoUrl = "" } = {}) {
    const text = String(raw ?? "").trim();
    if (!text) return emptyHtml;

    if (!parseMarkdown || !sanitizeHtml) {
      return `<p class="whitespace-pre-wrap">${escapeHtml(text)}</p>`;
    }

    try {
      return sanitizeHtml(parseMarkdown(linkifyCset(text, repoUrl)));
    } catch {
      return `<p class="whitespace-pre-wrap">${escapeHtml(text)}</p>`;
    }
  };
})();
