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
  - sqlc queries (`internal/db/queries/`, generated into `internal/db/gen`; run `go tool sqlc generate` from `backend/`). GET /services and GET /teams are real; **POST/PUT/DELETE return 501** — that's week 2 work, not a bug.
  - Idempotent seed (`internal/db/seed.sql`, gated by `APP_SEED=true`).
  - Integration test `internal/db/db_integration_test.go` (testcontainers, needs Docker running; `go test ./...`).
  - Tools pinned as go.mod `tool` directives: oapi-codegen v2.7.2, sqlc v1.31.1. go.mod says `go 1.26.0` (sqlc requires it; toolchain auto-downloads even though host Go is 1.25.6).
- **Frontend** (`frontend/`, Next.js 15.5.20, App Router, Tailwind v4, shadcn/ui radix-nova preset, pnpm 10.34.4):
  - Services list = async server component using generated openapi-fetch client (`lib/api/client.ts`, types regenerated with `pnpm generate:api`).
  - `/api/v1` proxied via rewrites in `next.config.ts` (`BACKEND_URL` env) — **no CORS anywhere, keep it that way**.
  - TanStack Query provider scaffolded (`app/providers.tsx`) but unused until week-2 mutations.
  - Design language: pine/ink oklch palette (hue 170), Geist Sans/Mono, mono slugs + lifecycle status dots (CSS vars `--status-production/beta/deprecated` in `app/globals.css`), `idp://` wordmark. Font variables must stay on `<html>` (see What Didn't Work).
- **Dev stack**: `docker compose up -d --build` → Postgres 17 + backend (air hot reload) + frontend (webpack dev + polling). All verified working including hot reload in containers.

## What Worked

- `go get -tool <pkg>@<version>` (Go 1.25+ tool directives) for reproducible codegen tooling — no global installs.
- golang-migrate as a library with embedded FS; URL scheme must be rewritten `postgres://` → `pgx5://` (done in `internal/db/migrate.go`).
- Seed idempotency via fixed UUIDs + `ON CONFLICT DO NOTHING`.
- Next rewrites instead of CORS; server components call `BACKEND_URL` directly.
- shadcn init non-interactively: `pnpm dlx shadcn@latest init --yes --base radix --preset nova --css-variables --no-monorepo` (`--base-color` flag no longer exists).
- Verifying UI with Playwright MCP (navigate + screenshot); artifacts go to `.playwright-mcp/` which is gitignored.

## What Didn't Work

- **Port 5432 on the host**: a native PostgreSQL 18 Windows service owns it — backend got "password authentication failed" because connections hit the native PG, not the container. Fix in place: compose maps Postgres to **host port 5433**. Native backend runs use `postgres://idp:idp@localhost:5433/idp?sslmode=disable`. Do not "fix" this back to 5432.
- **Turbopack in the frontend container**: never detects file changes on Windows bind mounts (ignores `WATCHPACK_POLLING`). Fix in place: compose overrides the command to `pnpm exec next dev` (webpack) with `WATCHPACK_POLLING=true`. Native `pnpm dev` keeps Turbopack.
- **Node 20's bundled corepack**: crashes with ERR_VM_DYNAMIC_IMPORT_CALLBACK_MISSING when activating pnpm. Fix: `npm i -g corepack@latest` first; pnpm pinned to 10.34.4.
- **Geist font variables on `<body>`**: the shadcn Nova preset applies `font-sans` on `<html>`, so vars defined on body left everything rendering serif. Vars must stay on the `<html>` element in `app/layout.tsx`.
- **oapi-codegen output path**: config `output:` resolves relative to the `go:generate` working dir, not the config file — a `../internal/api/gen.go` path created a stray `internal/internal/` tree. Output is now just `gen.go`.
- **PowerShell 5.1 quirks**: no `&&`; `Set-Location backend` fails if already in `backend/` (working dir persists between tool calls — check first).

## Next Steps (Week 2 — see docs/app-plan.md roadmap)

- [ ] Register Entra External ID tenant + two app registrations (backend API, frontend SPA) — needs the user's Azure account, ask them to do/authorize this.
- [ ] Backend auth middleware: OIDC JWT validation via `github.com/coreos/go-oidc`, claims → `User` upsert + team roles.
- [ ] Dev-mode auth toggle: `AUTH_MODE=dev` accepts a header-based identity for local compose (design decision already in app-plan.md).
- [ ] RBAC enforcement per handler (ADMIN/EDITOR/VIEWER per team, `team_members` table already exists).
- [ ] Implement POST/PUT/DELETE /services handlers (currently 501 stubs in `backend/internal/api/handlers.go`) + sqlc insert/update/delete queries + audit_log table migration.
- [ ] Frontend: NextAuth.js with Entra provider, login flow, protected routes.
- [ ] Service create/edit forms (React Hook Form + Zod) using the TanStack Query provider already scaffolded in `app/providers.tsx`.
- [ ] Milestone: logged-in EDITOR can create/edit a service; VIEWER is read-only.

Conventions to keep: one conventional commit per verified step on `main`; OpenAPI spec changes first, then regenerate both sides (`go generate ./...` + `pnpm generate:api`); verify before committing (curl + browser + tests); use design skills/browser tools for UI work (user explicitly asked).
