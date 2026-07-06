export interface RecentProject {
  id: string
  name: string
}

const RECENT_PROJECTS_KEY = 'bit_recent_projects'
const RECENT_PROJECTS_MAX = 5

let _recentProjects = $state<RecentProject[]>(loadRecent())

function loadRecent(): RecentProject[] {
  try {
    const raw = localStorage.getItem(RECENT_PROJECTS_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed
      .map((p) => ({
        id: String(p?.id || '').trim(),
        name: String(p?.name || '').trim(),
      }))
      .filter((p) => p.id)
      .slice(0, RECENT_PROJECTS_MAX)
  } catch {
    return []
  }
}

function saveRecent(items: RecentProject[]) {
  try {
    localStorage.setItem(RECENT_PROJECTS_KEY, JSON.stringify(items))
  } catch {
    /* ignore */
  }
}

export function getRecentProjects(): RecentProject[] {
  return _recentProjects
}

export function touchProject(project: { id: string; name?: string }) {
  const id = String(project.id || '').trim()
  if (!id) return

  let name = String(project.name || id).trim() || id
  const existing = _recentProjects.find((p) => p.id === id)
  if (existing?.name && name.toLowerCase() === id.toLowerCase()) {
    if (existing.name.toLowerCase() !== id.toLowerCase()) {
      name = existing.name
    }
  }

  const items = _recentProjects.filter((p) => p.id !== id)
  items.unshift({ id, name })
  _recentProjects = items.slice(0, RECENT_PROJECTS_MAX)
  saveRecent(_recentProjects)
}

export function enrichRecentFromProjects(
  projects: { id: string; name: string }[],
) {
  if (!projects.length || !_recentProjects.length) return

  const byId = new Map(projects.map((p) => [String(p.id), String(p.name || p.id)]))
  let changed = false
  const updated = _recentProjects.map((item) => {
    const name = byId.get(item.id)
    if (name && name !== item.name) {
      changed = true
      return { ...item, name }
    }
    return item
  })
  if (changed) {
    _recentProjects = updated
    saveRecent(_recentProjects)
  }
}

export function parseProjectSlugFromPath(path: string): string {
  try {
    let full = String(path || '')
    if (full.startsWith('#')) full = full.slice(1)
    if (full && !full.startsWith('/')) full = `/${full}`

    const qIdx = full.indexOf('?')
    const qs = new URLSearchParams(qIdx >= 0 ? full.slice(qIdx + 1) : '')
    const fromQuery = qs.get('project')
    if (fromQuery) return fromQuery.trim()

    const base = (qIdx >= 0 ? full.slice(0, qIdx) : full).split('?')[0]
    const match = base.match(/^\/projects\/([^/]+)$/)
    if (match) return decodeURIComponent(match[1] || '').trim()

    const legacyMatch = base.match(/^\/projects\/([^/]+)\/tasks$/)
    if (legacyMatch) return decodeURIComponent(legacyMatch[1] || '').trim()

    const taskMatch = base.match(/^\/task\/([^/]+)\/\d+$/)
    if (taskMatch) return decodeURIComponent(taskMatch[1] || '').trim()

    const taskEditMatch = base.match(/^\/task\/([^/]+)\/\d+\/edit$/)
    if (taskEditMatch) return decodeURIComponent(taskEditMatch[1] || '').trim()

    if (base.startsWith('/tasks/new/')) {
      return decodeURIComponent(base.slice('/tasks/new/'.length) || '').trim()
    }
  } catch {
    /* ignore */
  }
  return ''
}

export function isRecentProjectActive(id: string, _currentPath: string): boolean {
  const slug = String(id || '').trim()
  if (!slug) return false
  const full =
    typeof window !== 'undefined'
      ? window.location.hash.slice(1) || '/'
      : _currentPath
  return parseProjectSlugFromPath(full) === slug
}

export function trackRecentFromPath(path: string) {
  const slug = parseProjectSlugFromPath(path)
  if (!slug) return
  const recent = _recentProjects.find((p) => p.id === slug)
  touchProject({ id: slug, name: recent?.name || slug })
}
