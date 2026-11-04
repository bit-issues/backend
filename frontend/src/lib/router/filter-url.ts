import { PRIORITIES, STATUSES } from "$lib/types/api";

export type FilterState = {
  project: string;
  statuses: string[];
  priorities: string[];
  search: string;
  sort: string;
  offset: number;
};

const SORT_PATTERN = /^-?(priority|status|due_date|created_at)$/;

function sanitizeList<T extends string>(
  value: string | null,
  allowed: readonly T[],
  fallback: readonly T[],
): T[] {
  if (value === null) return [...fallback];
  if (value === "") return [];
  const kept = value
    .split(",")
    .filter((v): v is T => (allowed as readonly string[]).includes(v));
  return kept.length > 0 ? [...kept] : [...fallback];
}

function sanitizeSort(value: string | null, fallback: string): string {
  if (value !== null && SORT_PATTERN.test(value)) return value;
  return fallback;
}

function sanitizeOffset(value: string | null, fallback: number): number {
  if (value === null) return fallback;
  const n = Number.parseInt(value, 10);
  return Number.isFinite(n) && n >= 0 ? n : fallback;
}

export function parseFilters(
  hash: string,
  defaults: FilterState,
): FilterState {
  const params = new URLSearchParams(hash.split("?")[1] ?? "");
  return {
    project: params.get("project") ?? defaults.project,
    statuses: sanitizeList(params.get("statuses"), STATUSES, defaults.statuses),
    priorities: sanitizeList(
      params.get("priorities"),
      PRIORITIES,
      defaults.priorities,
    ),
    search: params.get("search") ?? defaults.search,
    sort: sanitizeSort(params.get("sort"), defaults.sort),
    offset: sanitizeOffset(params.get("offset"), defaults.offset),
  };
}

export function buildFilterQuery(
  state: FilterState,
  defaults: FilterState,
): string {
  const parts: string[] = [];
  if (state.project && state.project !== defaults.project) {
    parts.push(`project=${encodeURIComponent(state.project)}`);
  }
  if (state.statuses.join(",") !== defaults.statuses.join(",")) {
    parts.push(`statuses=${state.statuses.join(",")}`);
  }
  if (state.priorities.join(",") !== defaults.priorities.join(",")) {
    parts.push(`priorities=${state.priorities.join(",")}`);
  }
  if (state.search && state.search !== defaults.search) {
    parts.push(`search=${encodeURIComponent(state.search)}`);
  }
  if (state.sort !== defaults.sort) {
    parts.push(`sort=${encodeURIComponent(state.sort)}`);
  }
  if (state.offset !== defaults.offset) {
    parts.push(`offset=${state.offset}`);
  }
  return parts.length > 0 ? `?${parts.join("&")}` : "";
}

export function replaceFilters(
  state: FilterState,
  defaults: FilterState,
): void {
  const path = (window.location.hash.slice(1) || "/").split("?")[0];
  history.replaceState(null, "", `#${path}${buildFilterQuery(state, defaults)}`);
}
