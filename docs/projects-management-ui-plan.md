# Projects Management UI Plan + Task Project Field Support

Date: 2026-04-23

## Assumptions

1. Keep current backend API as-is and only use existing endpoints:
   - `GET /api/v1/projects`, `GET /api/v1/projects/:slug`
   - `POST /api/v1/projects`, `PATCH /api/v1/projects/:slug`, `DELETE /api/v1/projects/:slug`
   - `GET /api/v1/tasks` with `project` query filter
   - `POST /api/v1/tasks` with required `project_slug`
2. Project management remains admin-only (enforced by backend role middleware).
3. MVP UI should prefer dropdown selection of project IDs/slugs instead of free-text where possible.
4. No new routes/pages are required; extend current embedded single-page UI.

## Current gap summary

- Task list UI does not expose project filter input.
- Task table does not show project identifier.
- Task create/edit uses manual project slug entry (error-prone).
- Admin area manages users only; there is no projects CRUD screen.

## Minimal solution design

### A) Project field support in tasks

1. Load project list once after login (`GET /api/v1/projects?limit=100&offset=0`).
2. Replace free-text project input in task create form with required `<select>` bound to project ID/slug.
3. Keep project immutable in edit form (backend task update does not support changing project slug).
4. Add `project` filter control in task dashboard filters and send `project=<slug>` in list query.
5. Show `project_slug` column in task list table.

### B) Projects management UI (admin)

1. Add `Projects` tab under admin view.
2. Projects list table:
   - columns: `id`, `name`, `repo_url`, `updated_at`
   - actions: `Edit`, `Delete`
3. Create project modal:
   - fields: `name`, `repo_url`
   - action: `POST /api/v1/projects`
4. Edit project modal:
   - fields: `name`, `repo_url`
   - action: `PATCH /api/v1/projects/:slug`
5. Delete action:
   - simple confirm dialog
   - action: `DELETE /api/v1/projects/:slug`

## Suggested implementation steps (small commits)

1. **Project data plumbing**
   - Add frontend state: `projects`, `projectsTotal`, `projectsError`.
   - Add `fetchProjects()` and call on successful login.
2. **Task project field support**
   - Add project filter select to task filters.
   - Add project column in task table.
   - Replace create-form project text input with project select.
3. **Admin projects list**
   - Add `adminSection` switch: `users` / `projects`.
   - Render projects table and paging.
4. **Projects CRUD actions**
   - Add create/edit modal handlers and delete handler.
   - Refresh projects list after each mutation.

## Acceptance criteria

1. In task dashboard, user can filter tasks by project.
2. Task table displays project ID/slug per row.
3. Task create requires project selection from known projects.
4. Admin can list, create, edit, and delete projects via UI.
5. All flows use existing backend endpoints with no API changes.

## Known tradeoffs

- Loading first 100 projects is simple for MVP; pagination can be added later.
- Keeping task project immutable in edit avoids unsupported backend operations.
- Single-page extension is faster now than creating a dedicated `/admin/projects` template.
