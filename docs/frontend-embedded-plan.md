# Embedded Frontend Implementation Plan (MVP)

## 1) Assumptions and boundaries

1. Backend remains the source of truth and keeps the existing REST API (`/api/v1/...`) unchanged.
2. Frontend is embedded into the Go service binary via `go:embed`; no separate frontend deploy target.
3. MVP scope is exactly:
   - personal/shared dashboards with filter/sort,
   - task CRUD + comments + status updates,
   - file upload via presigned S3 flow,
   - user admin page,
   - JWT auth.
4. We optimize for delivery speed and maintainability, not for complex SPA features (kanban, offline mode, realtime).

## 2) Chosen approach

- UI layer: Alpine.js (CDN import in templates).
- Styling: Tailwind CSS via CDN for MVP (no build pipeline initially).
- Data access: native `fetch` with small shared helper for auth header + JSON handling.
- Rendering: Go `html/template` for shell/layout; Alpine for interactivity.
- Navigation: single-page behavior with tab/panel switches (`x-show`), no client router.

## 3) Project structure

```text
web/
├── static/
│   └── js/
│       └── app.js
├── templates/
│   ├── layout.html
│   ├── index.html
│   ├── task_detail.html
│   └── admin.html
└── embed.go
```

## 4) Delivery phases (3–5 days)

### Phase 0 — Backend wiring for embedded assets (0.5 day)

- Add static/template folders and embed declarations.
- Register handlers for:
  - `/` (main HTML),
  - `/static/*` (JS/CSS),
  - existing `/api/v1/*` stays unchanged.
- Add cache headers for static assets.

**Exit criteria:** opening `/` returns rendered template; app.js loads.

### Phase 1 — Auth and shell (0.5 day)

- Implement layout with top nav and app container.
- Add Alpine global auth store:
  - login/logout,
  - token persistence in `localStorage`,
  - bootstrap current user (if token exists).
- Add generic `apiFetch` helper with `Authorization: Bearer <token>`.

**Exit criteria:** login works and protected API calls pass JWT.

### Phase 2 — Dashboards + task list (1 day)

- Add tabs: `My tasks` / `All tasks`.
- Implement filters:
  - status (multi-select),
  - priority,
  - due date range,
  - assignee/project where available.
- Implement sorting by key columns.
- Keep pagination server-driven if API supports it; otherwise temporary client-side paging with hard limit.

**Exit criteria:** user can filter/sort lists and open task detail.

### Phase 3 — Task CRUD + comments + status (1 day)

- Modal form for create/edit task.
- Status transition controls on detail panel/page.
- Comments list + add comment action.
- Basic optimistic refresh after successful mutations.

**Exit criteria:** full task lifecycle is operable from UI.

### Phase 4 — Attachments (S3 two-step) (0.5 day)

- Implement flow:
  1. POST metadata to create attachment record,
  2. PUT file to presigned URL,
  3. confirm upload.
- Show per-file upload state (pending/uploading/success/error).

**Exit criteria:** uploaded file is visible and downloadable from task.

### Phase 5 — User administration (0.5 day)

- Admin-only tab/page for users list and create/update/deactivate actions supported by backend.
- Role checks in UI + backend-enforced authorization remains primary control.

**Exit criteria:** admin can manage users through UI.

## 5) API contract checklist before implementation

Before coding screens, validate these endpoints and payload shapes:

- `POST /api/v1/auth/login`
- `GET /api/v1/tasks` (+ filters/sort params)
- `POST /api/v1/tasks`, `PUT/PATCH /api/v1/tasks/{id}`, `DELETE /api/v1/tasks/{id}`
- `GET/POST /api/v1/tasks/{id}/comments`
- `POST /api/v1/tasks/{id}/attachments` and `PUT /api/v1/attachments/{id}/confirm`
- Admin endpoints for users (`GET/POST/PATCH`).

If response envelopes differ (e.g., `{items, total}` vs raw arrays), finalize one shape first to avoid frontend rework.

## 6) Risks and tradeoffs

1. **CDN dependency (Alpine/Tailwind):** fastest start, but production may require vendoring for restricted corporate networks.
2. **No frontend build step:** simpler operations, but less optimization and type safety.
3. **Single-page without router:** minimal complexity, but deep-linking to specific views is limited.
4. **Alpine for MVP:** ideal now; if workflows become very stateful (kanban/realtime), migration path to Vue 3 should be planned.

## 7) Migration trigger to Vue (not now)

Re-evaluate stack only when at least two are true:

- >10 interactive screens,
- repeated complex shared state bugs,
- need for advanced client routing/deep links,
- rich drag/drop interactions (kanban/Gantt).

Until then, keep Alpine implementation intentionally simple.

## 8) Done definition for MVP frontend

- All PRD flows executable in browser with JWT-protected APIs.
- No separate frontend service in deployment.
- App embedded into Go binary and served by existing backend.
- Basic smoke checks pass:
  - login/logout,
  - list/filter/sort tasks,
  - create/update/comment/status change,
  - upload/confirm attachment,
  - admin user management.
