# Handoff

## Goal

Build the **Internal Developer Portal** described in `docs/app-plan.md` — a Backstage-lite service catalog (Go + Next.js + Postgres, later deployed to Azure) as a portfolio piece. Standout feature: an interactive React Flow dependency graph (week 4). Full 4-week roadmap, data model, tech decisions, and acceptance criteria are in `docs/app-plan.md` — read it first.

## Current Progress

**Week 1 is complete and verified** (2026-07-07). All 12 planned steps landed as individual conventional commits on `main` (`eecd117..b70af52`). Milestone holds: `curl localhost:8080/api/v1/services` returns seeded data; `http://localhost:3000/services` renders it (screenshot: `docs/screenshots/week1-services-list.png`).

What exists:

- **Contract**: `backend/api/openapi.yaml` (OpenAPI 3.0.3) — Service CRUD + Team listing, bearer-auth scheme declared but not enforced. Single source of truth for both codegens.
- **Backend** (`backend/`, Go, module `github.com/OtsoH/internal-developer-portal/backend`):
  - chi server, slog JSON logging, `/healthz`, API mounted at `/api/v1` (`cmd/api/main.go`).
  - oapi-codegen strict-server generated into `internal/api/gen.go` (regenerate: `go generate ./...` from `backend/`; config `backend/api/oapi-codegen.yaml`, output path resolves relative to `internal/api/`).
  - Migrations in `backend/migrations/`, embedded and run at startup via golang-migrate **as a library** (no CLI). Tables: teams, users, team_members, services, tags, service_tags.
  - sqlc queries (`internal/db/queries/`, generated into `internal/db/gen`; run `go tool sqlc generate` from `backend/`).
  - Idempotent seed (`internal/db/seed.sql`, gated by `APP_SEED=true`).
  - Integration test `internal/db/db_integration_test.go` (testcontainers, needs Docker running; `go test ./...`).
  - Tools pinned as go.mod `tool` directives: oapi-codegen v2.7.2, sqlc v1.31.1. go.mod says `go 1.26.0` (sqlc requires it; toolchain auto-downloads even though host Go is 1.25.6).
- **Frontend** (`frontend/`, Next.js 15.5.20, App Router, Tailwind v4, shadcn/ui radix-nova preset, pnpm 10.34.4):
  - Services list = async server component using generated openapi-fetch client (`lib/api/client.ts`, types regenerated with `pnpm generate:api`).
  - `/api/v1` proxied via rewrites in `next.config.ts` (`BACKEND_URL` env) — **no CORS anywhere, keep it that way**. (Week 2 step 6 replaces the rewrite with a BFF route handler; the no-CORS rule survives, the mechanism changes.)
  - TanStack Query provider scaffolded (`app/providers.tsx`) but unused until week-2 mutations.
  - Design language: pine/ink oklch palette (hue 170), Geist Sans/Mono, mono slugs + lifecycle status dots (CSS vars `--status-production/beta/deprecated` in `app/globals.css`), `idp://` wordmark. Font variables must stay on `<html>` (see What Didn't Work).
- **Dev stack**: `docker compose up -d --build` → Postgres 17 + backend (air hot reload) + frontend (webpack dev + polling). All verified working including hot reload in containers.

**CI + test automation landed (2026-07-11), pipeline green on `main`:**

- **CI**: `.github/workflows/ci.yml` — push-to-main + PR + manual triggers, path-filtered jobs via dorny/paths-filter (backend on `backend/**`, frontend on `frontend/**` + the OpenAPI spec, workflow file triggers both), `ci-ok` gate job for future branch protection. Backend job: golangci-lint, codegen drift check (`go generate` + sqlc + `git diff --exit-code`), build, `go test -race` incl. the testcontainers integration test. Frontend job: frozen-lockfile install, API-client drift check, ESLint, `tsc --noEmit`, Vitest, `next build`. CI badge in root README.
- **Backend tests**: `internal/api/handlers_test.go` unit tests (nil-DB mode, 501 stubs, `serviceFromRow` mapping) run under `go test -short` without Docker.
- **Lint**: golangci-lint v2.12.2 pinned as a go.mod `tool` directive (same version local + CI, run `go tool golangci-lint run`; config `backend/.golangci.yml`).
- **Frontend tests**: Vitest 4 + Testing Library (jsdom, native tsconfig path aliases in `vitest.config.ts`, jest-dom via `vitest.setup.ts`). Starter tests: `app/services/page.test.tsx` (async server component pattern: `render(await ServicesPage())` with mocked `@/lib/api/client`) and `lib/utils.test.ts`.
- **test-runner subagent**: `.claude/agents/test-runner.md` runs both suites and reports concisely; knows Docker-Desktop/-short fallback.
- **CLAUDE.md** at repo root: session bootstrap pointers, hard rules, command index, skill-usage map.

---

## Week 2 in progress (2026-07-29) — steps 1–6 of 14 done and committed

**The approved plan lives at `C:\Users\otsoh\.claude\plans\plan-how-to-implement-cached-simon.md`.** Read it before continuing — it has the full 14-step breakdown, the confirmed design decisions, per-step verification commands, a risks table, and the manual Entra setup appendix. It also has a "Progress and deviations" section recording where reality differed from the plan, now through step 6; it should gain a step 7 entry before step 7 starts.

### Design decisions already confirmed with the user (do not re-litigate)

| Decision | Choice |
|---|---|
| Token transport | **BFF proxy route handler.** `app/api/v1/[...path]/route.ts` replaced the `next.config.ts` rewrite in step 6; it reads the Auth.js session server-side and attaches credentials. The access token never reaches the browser. Still zero CORS. |
| Frontend auth | **Auth.js v5** (`next-auth@5` beta, pinned to an exact version), `MicrosoftEntraID` provider plus a `Credentials` provider gated to dev mode. |
| Read access | Any valid credential = **implicit VIEWER over the whole catalog**. |
| Write access | EDITOR or ADMIN in `team_members` for the service's owning team. |
| Delete | **ADMIN on the owning team, hard delete.** `audit_log` is the paper trail. |
| Roles in the session? | **No.** Session carries identity + access token only; roles always come from `GET /me`, so `team_members` changes take effect without re-authenticating. |
| Where RBAC is enforced | Authentication is chi middleware (401); **authorization is in the handlers** (403). Reasons are in the plan's step 9 — briefly: PUT/DELETE authorization depends on the existing row's team, PUT needs two checks when a service moves team, and a strict middleware would need an unchecked `switch operationID` to build the right 403 response type. |

### Sequencing constraint

Entra tenant + app registrations need the user's own Azure account, so **every step is verifiable with `AUTH_MODE=dev` alone**. The real-Entra path is written but stays unverified until the manual checklist at the end of the plan is done. Nothing blocks on it.

### What is done (steps 1–6), all verified and committed

- **Step 1 — OpenAPI contract.** Every error status now has its own named response component (`BadRequest`/`Unauthorized`/`Forbidden`/`NotFound`/`Conflict`); `ErrorResponse` remains only for `default`. Added 401 across the board, 403 on mutations, a `Role` enum with `x-enum-varnames`, `TeamRole`, `CurrentUser`, and `GET /me`. `handlers.go`'s `notFound` now returns `NotFoundJSONResponse`; `GetCurrentUser` is a 501 stub until step 7.
- **Step 2 — `audit_log` migration + queries.** `migrations/000002_audit_log.{up,down}.sql`; new query files `users.sql`, `tags.sql`, `audit.sql`; service insert/update/delete plus `GetServiceForUpdate` appended to `services.sql`.
- **Step 3 — RBAC seed.** Added `dev.viewer@example.com` and cross-team memberships so all four outcomes are demonstrable. Integration test extended with user/membership/audit assertions.
- **Step 4 — `internal/auth` package, unwired.** `auth.go` (Identity, Role, TeamRole, Principal, context helpers, `Verifier` interface, `ErrInvalidToken`), `config.go`, `dev.go`, `oidc.go`, `resolver.go`, `middleware.go`, plus `auth_test.go`, `config_test.go`, `dev_test.go`, `middleware_test.go`. 11 tests pass without Docker.
- **Step 5 — Auth.js v5 on the frontend.** `frontend/auth.ts` (mode-gated `Credentials`/`MicrosoftEntraID` providers, session carries identity + access token only, never roles), `types/next-auth.d.ts` (Session augmentation only — see What Didn't Work), `app/api/auth/[...nextauth]/route.ts`, `lib/auth/session.ts` (`requireSession()`), `app/signin/page.tsx` (one server-action button per seeded persona with its role hint), `SiteHeader` is now an async server component showing email + sign-out. `app/services/page.tsx` calls `requireSession()` — the only route gated so far. `.env.example` + `.gitignore` negation, `docker-compose.yml` (`AUTH_MODE`/`AUTH_SECRET`/`AUTH_TRUST_HOST`), `ci.yml` builds with `AUTH_MODE` unset on purpose (see below).

Backend verified: `go build ./...`, `go tool golangci-lint run` (0 issues), `go test ./...` including testcontainers, codegen proven drift-free by hashing generated files across a re-run.
Frontend verified: `pnpm lint`, `pnpm typecheck`, `pnpm test` (6/6), `pnpm build`, plus a Playwright pass — signed out `/services` 307s to `/signin`; sign in as Dev Editor lands on `/services` with email + sign-out in the header; sign-out returns to `/signin`. Live stack (`docker compose up -d --build`) serving 200 on both `:8080/api/v1/services` and `:3000/services`.

- **Step 6 — BFF proxy + split API client.** `app/api/v1/[...path]/route.ts` proxies `/api/v1/*`, stripping inbound `x-dev-user`/`authorization`/`cookie`/hop-by-hop/`host` before attaching the real credential and forwarding; strips `content-encoding`/`content-length` on the response; streams the body; 401s with an `Error`-shaped body when there's no session. `lib/auth/forward.ts` (`authHeaders(session)`) is the one place that knows dev mode sends `X-Dev-User` and Entra mode sends `Authorization: Bearer`. `lib/api/server.ts` adds `getServerApi(session)` (credentialed client straight to `BACKEND_URL`, bypassing the proxy — server components don't need it) and `getCurrentUser()` wrapped in React `cache()`, calling `auth()` itself so it stays zero-argument and dedupes across whatever in a request calls it. `lib/api/client.ts` is now browser-only (`baseUrl: "/api/v1"`, no branch); `next.config.ts`'s rewrite is gone, replaced by the route handler. `app/services/page.tsx` switched from the old client to `getServerApi(session)`.
  Verified: `pnpm lint`, `pnpm typecheck`, `pnpm test` (9/9), `pnpm build` (`/api/v1/[...path]` lists as a dynamic route). The two-commit split was checked out independently before landing — BFF-infra-only state (new files, nothing wired up yet) typechecks/lints/tests clean on its own — not just reasoned through. Live stack + Playwright: signed out, both `/services` and `/api/v1/services` reject correctly; a forged `X-Dev-User: dev.admin@example.com` from outside the app is stripped and still 401s; signed in as Dev Editor, `/services` renders all 5 seeded services through `getServerApi`, `/api/v1/services` returns real JSON through the proxy; sign-out reverts both. Backend still accepts unauthenticated requests (200) — expected, that's step 7.

### The dev-mode RBAC matrix now seeded

| actor | Platform | Payments |
|---|---|---|
| `dev.admin@example.com` | ADMIN — create, edit, delete | *no membership* → 403 on every mutation |
| `dev.editor@example.com` | VIEWER — 403 on mutations (membership without power) | EDITOR — create/edit, 403 on delete |
| `dev.viewer@example.com` | VIEWER — 403 on mutations | *no membership* → 403 |

Everyone reads the whole catalog regardless.

### Commit history so far (steps 1–6, all landed on `main`)

Steps 1–4 landed as one commit per file-group (`8802cb0..7972cc1`; see `git log --oneline 8802cb0..7972cc1`). Step 5 landed as 5 commits (`091c967..8b52833`): env plumbing → `auth.ts` + type augmentation → route handler + session guard → sign-in page + header → the services-page gate itself. Step 6 landed as 2 commits: `feat(frontend): add the BFF proxy route handler and server-side API client` (`frontend/lib/auth/forward.ts`, `frontend/lib/api/server.ts`, `frontend/app/api/v1/[...path]/route.ts`, `frontend/app/api/v1/[...path]/route.test.ts`) then `feat(frontend): route the services page through the BFF proxy, retiring the rewrite` (`frontend/lib/api/client.ts`, `frontend/next.config.ts`, `frontend/app/services/page.tsx`, `frontend/app/services/page.test.tsx`) — `git log --oneline` for the exact hashes.

Steps 1–5's split was reasoned through, not individually checked out and re-verified. Step 6's split *was* checked out independently before landing (see above) — the stronger guarantee the CLAUDE.md rule below asks for.

### CLAUDE.md was updated this session (2026-07-29)

The Hard Rules now require **proposing a commit split**, not one giant diff: one concern per commit, exact `git add` paths, dependency-ordered (module before callers, test with its code, config before what needs it at build time), and an explicit statement of whether the ordering was *verified* (checked out and tested) or only *reasoned through*. Re-read CLAUDE.md's Hard Rules section before proposing commits for steps 6–14.

### Branching/CI decision (not yet acted on)

Discussed and deferred: `ci.yml`'s existing triggers (`push: main` + bare `pull_request`) already support a feature-branch workflow with zero changes — pushing a branch runs nothing, opening a PR runs the full suite, `ci-ok` is a real required-check candidate. Decision: **finish week 2 on `main`** as before; revisit branching at the week 2 → week 3 boundary. Blocker to fix first: `gh` CLI is not installed, which is why CI has to be checked via the badge SVG (see "Polling the GitHub API unauthenticated" in What Didn't Work) — install it (`winget install --id GitHub.cli`) before adopting PRs, or PR review becomes browser-only round trips.

## What Worked

- `go get -tool <pkg>@<version>` (Go 1.25+ tool directives) for reproducible codegen tooling — no global installs.
- golang-migrate as a library with embedded FS; URL scheme must be rewritten `postgres://` → `pgx5://` (done in `internal/db/migrate.go`).
- Seed idempotency via fixed UUIDs + `ON CONFLICT DO NOTHING`.
- Next rewrites instead of CORS; server components call `BACKEND_URL` directly. (Replaced by the BFF route handler in step 6 — the no-CORS property is preserved.)
- shadcn init non-interactively: `pnpm dlx shadcn@latest init --yes --base radix --preset nova --css-variables --no-monorepo` (`--base-color` flag no longer exists).
- Verifying UI with Playwright MCP (navigate + screenshot); artifacts go to `.playwright-mcp/` which is gitignored.
- Testing async server components by calling them as functions: `render(await ServicesPage())` with `vi.mock("@/lib/api/server")` (was `@/lib/api/client` before step 6's split).
- Checking CI without `gh` (not installed): the badge SVG at `github.com/OtsoH/Internal-Developer-Portal/actions/workflows/ci.yml/badge.svg` shows passing/failing without API rate limits.
- **Proving codegen drift-freedom locally** without a clean tree: hash the generated files, re-run `go generate ./...` + `go tool sqlc generate`, hash again, compare. `git diff --exit-code` only works against a committed baseline, so it reports uncommitted work as "drift".
- **Adding a migration needs no Go change** — `migrations/embed.go` globs `*.sql` and `sqlc.yaml`'s `schema:` points at the directory, so both pick new files up automatically.
- **An environment-aware `AUTH_MODE` default** (`isProd ? "entra" : "dev"`) instead of a flat default, mirroring the backend's `ConfigFromEnv` by making unknown values throw rather than silently falling back. Forgetting the variable in a real deployment can never downgrade it to the seeded dev personas.

## What Didn't Work

- **Port 5432 on the host**: a native PostgreSQL 18 Windows service owns it — backend got "password authentication failed" because connections hit the native PG, not the container. Fix in place: compose maps Postgres to **host port 5433**. Native backend runs use `postgres://idp:idp@localhost:5433/idp?sslmode=disable`. Do not "fix" this back to 5432.
- **Turbopack in the frontend container**: never detects file changes on Windows bind mounts (ignores `WATCHPACK_POLLING`). Fix in place: compose overrides the command to `pnpm exec next dev` (webpack) with `WATCHPACK_POLLING=true`. Native `pnpm dev` keeps Turbopack.
- **Node 20's bundled corepack**: crashes with ERR_VM_DYNAMIC_IMPORT_CALLBACK_MISSING when activating pnpm. Fix: `npm i -g corepack@latest` first; pnpm pinned to 10.34.4.
- **Geist font variables on `<body>`**: the shadcn Nova preset applies `font-sans` on `<html>`, so vars defined on body left everything rendering serif. Vars must stay on the `<html>` element in `app/layout.tsx`.
- **oapi-codegen output path**: config `output:` resolves relative to the `go:generate` working dir, not the config file — a `../internal/api/gen.go` path created a stray `internal/internal/` tree. Output is now just `gen.go`.
- **PowerShell 5.1 quirks**: no `&&`; `Set-Location backend` fails if already in `backend/` (working dir persists between tool calls — check first).
- **pnpm/action-setup with a root-less package.json**: the action reads `packageManager` from the repo root by default and failed in this monorepo. Fix in place: `package_json_file: frontend/package.json` in ci.yml.
- **`go test -race` locally**: needs cgo, and this Windows host has no C toolchain — run plain `go test` locally; CI (ubuntu) covers `-race`.
- **Polling the GitHub API unauthenticated**: 60 req/h rate limit gets exhausted fast by a 10 s poll loop (`gh` CLI is not installed on this machine). Use the badge SVG instead, or install/auth `gh`.
- **Sharing one `ErrorResponse` component across statuses in the OpenAPI spec** (fixed in week 2 step 1): oapi-codegen tracks used response refs *per operation*, so only the first response referencing it got the embedded `struct{ ErrorResponseJSONResponse }` form and later ones degraded to a bare `type X Error`. Generated Go shapes therefore depended on which statuses an operation happened to declare — adding a 401 to `GetService` would have silently reshaped its 404 and broken `handlers.go`. Fixed structurally with one named component per status; **do not collapse them back**.
- **`emit_pointers_for_null_types` does not cover overridden types** (fixed in week 2 step 2): `audit_log.actor_id` generated as `pgtype.UUID`, not `*uuid.UUID`, because the `db_type: "uuid"` override in `sqlc.yaml` only matches the non-nullable case. Fix in place: a second override with `nullable: true` and `pointer: true`. Any future nullable UUID column now behaves; relevant for week 3's `service_dependencies` and `api_specs`.
- **`go get github.com/coreos/go-oidc/v3` alone leaves go.sum incomplete** — the build then fails on missing entries for `go-jose/v4` and `golang.org/x/oauth2`. Fetching the package path instead (`go get github.com/coreos/go-oidc/v3/oidc@v3.20.0`) resolves the transitive entries.
- **Adding unexported helpers before their first caller trips golangci-lint's `unused`.** The plan sketched adding `badRequest`/`forbidden`/`conflict`/`unauthorized` alongside `notFound` in step 1; they had to be deferred to the steps that call them (7 and 9). Same reason `normalizeEmail` sits in `dev.go` rather than `oidc.go`.
- **The plan's flat `AUTH_MODE=dev` default breaks the production build** (found in step 5): with CI leaving `AUTH_MODE` unset on purpose (see plan risk #7) and the plan's `auth.ts` defaulting unset to `"dev"`, the production guard fires on every CI build (`Failed to collect page data for /_not-found`). Fixed by making the default environment-aware — see What Worked. If a future step touches `auth.ts`'s mode resolution, keep that behavior.
- **`next-auth/jwt` cannot be augmented from this repo** (found in step 5): it's a bare `export * from "@auth/core/jwt"`, so `declare module "next-auth/jwt"` creates a fresh interface instead of merging with the real one, and `@auth/core` isn't resolvable from `frontend/` root under pnpm (it's a transitive dep, not a direct one). `types/next-auth.d.ts` only augments `next-auth`'s `Session`; `auth.ts`'s `session()` callback narrows the JWT's `accessToken` with `typeof token.accessToken === "string"` at runtime instead of typing it. Don't add `@auth/core` as a direct dep just to fix this — the runtime narrow is the intended shape.
- **`docker compose up -d --build` does not refresh an anonymous volume** (found in step 5, will recur at steps 11 and 12): after `pnpm add next-auth`, `--build` alone left the frontend container's `/app/node_modules` volume stale (`Can't resolve 'next-auth'`, then 500s on every route). Fix: `docker compose up -d --build --renew-anon-volumes frontend` whenever a step adds an npm dependency. Plan risk #6 mentions rebuilding but not this flag.

## Next Steps

Steps 7–14 of the plan, in order. Each is independently verifiable with dev-mode auth. Before starting, re-read the plan's step text (this file only summarizes) and CLAUDE.md's Hard Rules (updated this session — commit proposals must now be a dependency-ordered split with exact `git add` paths, not one commit per step).

- [x] **Step 6 — BFF proxy + split the API client.** Done — see the Week 2 section above.
- [ ] **Step 7 — Wire auth into the backend.** Widen `Server` with **variadic options** (`WithTxBeginner`, `WithLogger`) so all five `NewServer(nil)` call sites in `handlers_test.go` compile untouched; mutations gain a `s.q == nil || s.tx == nil` → 503 guard so nil-DB mode survives. Extract `internal/app/router.go` so step 10's integration test exercises the production router. Hoist the pool out of `run()` in `main.go`. `/healthz` stays on the outer router (public).
- [ ] **Step 8 — Spec-driven request validation** via `nethttp-middleware`. Three traps: `swagger.Servers = nil`, an `AuthenticationFunc` returning nil (kin-openapi enforces `security` and would 401 every dev-mode request), and replacing the plain-text default error handler. Droppable step with a hand-written fallback — see the plan.
- [ ] **Step 9 — Mutation handlers** with transactions, tag normalization, audit rows and pg-error mapping. Match the **constraint name** not just SQLSTATE 23505 (`tags_name_key` must not be reported as a slug conflict). Check ordering per operation is spelled out in the plan.
- [ ] **Step 10 — API integration test**: the {admin, editor, viewer} × {POST, PUT, DELETE} × {Platform, Payments} matrix, built through `app.NewRouter`.
- [ ] **Step 11 — Frontend form deps + Zod schema.** `shadcn add` pulls split `@radix-ui/react-*` packages, but this project uses the **unified `radix-ui`** package — rewrite the generated imports. Its `sonner.tsx` pulls `next-themes`; dark mode is unreachable today, so hand-edit instead of adding the dep.
- [ ] **Step 12 — Create form + role-gated list button.**
- [ ] **Step 13 — Detail page, edit, admin-only delete.**
- [ ] **Step 14 — Docs**: ADR-0002, `docs/entra-setup.md`, README/HANDOFF updates, and **`CLAUDE.md`'s "No CORS" bullet still describes the rewrite** — the rule survives but the mechanism changes in step 6.
- [ ] **Manual, needs the user's Azure account**: register the Entra External ID tenant + two app registrations. Full checklist is the appendix of the plan file. The failure to expect: a missing/wrong API scope makes Entra issue a Microsoft Graph token that looks valid but fails the audience check.

## Conventions to keep

- **Propose a dependency-ordered commit split, never one big diff — Claude proposes, the user commits.** Claude never commits or pushes unless the current message explicitly asks. One concern per commit, exact `git add` paths, and an explicit note on whether the ordering was verified (checked out + tested each step) or only reasoned through. Full rule in CLAUDE.md's Hard Rules (updated 2026-07-29).
- **OpenAPI-first**: spec changes first, then regenerate both sides (`go generate ./...` + `pnpm generate:api`). CI fails on drift.
- **Verify before claiming done**: tests + curl + browser check for UI work.
- Use the design skills and Playwright browser tools for UI work (the user asked for this explicitly).
- Run the `humanizer` skill over user-facing prose before finalizing.
- **Still on `main`, no feature branches yet.** `ci.yml` already supports a branch+PR workflow (push-to-main and pull_request triggers both exist), but the switch is deferred to the week 2 → week 3 boundary and blocked on installing `gh` first (see What Didn't Work / the branching note above).
