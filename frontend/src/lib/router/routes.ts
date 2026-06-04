import type { ComponentType } from 'svelte'

export interface RouteDef {
  pattern: string
  component: ComponentType
  auth?: boolean
  role?: 'admin'
}

export interface MatchResult {
  route: RouteDef
  params: Record<string, string>
}

export function matchRoute(
  path: string,
  routes: RouteDef[],
): MatchResult | null {
  for (const route of routes) {
    const params = matchPattern(route.pattern, path)
    if (params) {
      return { route, params }
    }
  }
  return null
}

function matchPattern(
  pattern: string,
  path: string,
): Record<string, string> | null {
  if (pattern === path) return {}

  const patParts = pattern.split('/')
  const pathParts = path.split('/')

  if (patParts.length !== pathParts.length) return null

  const params: Record<string, string> = {}
  for (let i = 0; i < patParts.length; i++) {
    if (patParts[i].startsWith(':')) {
      params[patParts[i].slice(1)] = pathParts[i]
    } else if (patParts[i] !== pathParts[i]) {
      return null
    }
  }

  return params
}

export function navigate(hash: string): void {
  window.location.hash = hash
}
