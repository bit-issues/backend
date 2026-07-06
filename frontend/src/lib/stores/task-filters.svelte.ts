export type FilterSnapshot = {
  filterProject?: string
  filterStatuses: string[]
  filterPriorities: string[]
  sort: string
  offset: number
}

let store = $state<Record<string, FilterSnapshot>>({})

export function getSnapshot(path: string): FilterSnapshot | null {
  return store[path] ?? null
}

export function saveSnapshot(path: string, snapshot: FilterSnapshot): void {
  store[path] = snapshot
}

export function clearSnapshot(path: string): void {
  delete store[path]
}
