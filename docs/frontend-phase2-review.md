# Phase 2 Review — Embedded Frontend Task Dashboard

Date: 2026-04-23

## Scope reviewed

- `web/templates/index.html`
- `web/static/js/app.js`
- Task list endpoints and query DTOs used by the frontend:
  - `GET /api/v1/tasks`
  - `GET /api/v1/tasks/me`

## Summary

Phase 2 is **complete for MVP scope**.

Implemented:
- Personal/shared dashboard switch (`/tasks/me` vs `/tasks`).
- Status and priority filters.
- Sorting control and server query integration.
- Task table rendering with loading/empty/error states.

Remaining caveat:
- Task detail opens as API JSON (`/api/v1/tasks/:id`) until a dedicated HTML detail page is added in a later phase.

## What matches backend contract

Current UI query params match backend `TaskListQuery`:
- `limit`, `offset`, `sort`, `statuses`, `priorities`.

This is correct for current server DTO shape and repository sorting support.

## Risks / issues found

1. `apiFetch` always tries `JSON.parse(text)` when body is non-empty.
   - If a non-JSON error body is returned by middleware/proxy, this throws before the HTTP status path is handled.
2. Review-only note: frontend currently combines Phase 1 auth shell and Phase 2 listing in a single page component.
   - This is acceptable for MVP, but makes future task-detail/admin split slightly harder.

## Recommended follow-up (next phases)

1. Replace API JSON task link with dedicated HTML task detail page.
2. Optionally add page-size selector (current pagination keeps fixed limit of 20).

## Verdict

- **Acceptable as MVP and complete for Phase 2 target** with server-backed filters/sort and basic pagination in place.
