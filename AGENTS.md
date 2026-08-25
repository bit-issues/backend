# AGENTS.md

Repo-local agent instructions for the BitIssues backend. Repo-local rules override `~/.config/opencode/AGENTS.md` defaults where they conflict.

## Stack (brief)

- Go 1.25 backend: Fiber HTTP, Bun ORM (MySQL), Uber Fx DI, goose migrations
- Svelte 5 / Vite frontend in `frontend/`
- Lint: golangci-lint (strict preset); frontend: svelte-check
- Env config via koanf, `SECTION__FIELD` double-underscore convention

## Testing gotchas

- bun MySQL dialect probes `SELECT version()` during `bun.NewDB`; register that sqlmock expectation before any other expectations.
- `golang.org/x/sync/singleflight` caches only in-flight calls, never completed ones; late callers re-run the fn. Concurrency tests need an in-flight barrier.
- goose migration files piped to the mysql client execute BOTH Up and Down blocks; validate Up-only by extracting the Up block.
- go-sqlmock v1.5.2: `MatchExpectationsInOrder(false)` is a method on Sqlmock, not a `New()` option.
- MySQL `DELETE ... WHERE pk = ?` with `RowsAffected()==1` is a valid atomic single-use consume gate (no `DELETE RETURNING` on MySQL 8.0).
- errorlint rejects `'%w: %v'` with an error arg; Go 1.20+ multiple `%w` verbs (`'%w: %w'`) satisfy both errors.Is and errorlint.
- fiberfx JSON error handler masks all 5xx messages to 'Internal Server Error'; handler tests must assert status codes, not 5xx bodies.
- Fiber `c.SendStatus(200)` returns text/plain 'OK'; HTTP clients must tolerate non-JSON 2xx bodies.
- fx v1.24 validates graphs via package-level `fx.ValidateApp(options...)`, not `app.Validate()`.
- fx wiring of a setter via `fx.Invoke` keeps constructor signatures stable, preserving existing test call sites.

## Lint conventions

- Strict set: exhaustruct/mnd/shadow/wrapcheck/err113.
- exhaustruct: satisfy by explicitly listing zero-value fields (e.g. `TokenResolver: nil`).
- Prefer named consts over magic numbers; never shadow `err`; wrap external errors with `%w`.
- golangci-lint serializes via a shared cache lock; concurrent lint runs in one workspace block each other - use an isolated TMPDIR per run.

## OAuth & security patterns

- Token redaction requirements include test failure messages, not only production logger calls and API serialization.
- OAuth CSRF state: validate bindings (user, redirect URI) and expiry BEFORE conditional-delete consumption; atomic conditional delete (1 affected row) is the single-use gate; invalid attempts must not burn a valid state.
- Delete failures during state consumption must abort the flow; never continue to code exchange.
- OAuth callback (browser redirect target) must be registered before the global auth middleware; the state binds the unauthenticated callback to the initiating admin.
- Map 403 handling before the generic client-error branch (restkit.AsAPIError ordering); never expose credentials in error mappings.
- Bitbucket Cloud OAuth2: access tokens expire in 7200s (2h); refresh tokens are long-lived but single-use per refresh and user-revocable; proactively refresh ~15m before expiry under singleflight.

## Known repo state

- Frontend svelte-check is red at HEAD baseline: 77 pre-existing errors across 32 files (bits-ui v2.18.1 type incompatibilities in ui/select, ui/pagination, ui/table, badge, passkey.ts, App.svelte routes). Feature work must gate on zero NEW errors, not a green check.
- Frontend build emits a pre-existing a11y warning for AttachmentUpload.svelte:93.
