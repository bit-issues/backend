export interface AutoLinkContext {
  repoUrl?: string
}

interface AutoLinkRule {
  name: string
  regex: RegExp
  replace: (fullMatch: string, captures: string[], ctx: AutoLinkContext) => string
}

const rules: AutoLinkRule[] = []

export function registerRule(rule: AutoLinkRule): void {
  rules.push(rule)
}

export function processAutoLinks(text: string, ctx: AutoLinkContext): string {
  let result = text
  for (const rule of rules) {
    result = result.replace(rule.regex, (...args) => {
      const captures = args.slice(1, args.length - 2) as string[]
      return rule.replace(args[0], captures, ctx)
    })
  }
  return result
}

registerRule({
  name: "cset",
  regex: /<<cset\s+([a-f0-9]{7,40})>>/g,
  replace: (_full, captures, ctx) => {
    const hash = captures[0]
    const short = hash.slice(0, 7)
    if (!ctx.repoUrl) return short
    const base = ctx.repoUrl.replace(/\/+$/, "")
    return `<a class="autolink" href="${base}/commits/${hash}" target="_blank" rel="noreferrer">${short}</a>`
  },
})
