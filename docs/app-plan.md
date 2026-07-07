# Internal Developer Portal — Implementation Plan

## Context

Build a Backstage-lite **Internal Developer Portal** as a portfolio piece. It is the spiritual sequel to the Elisa feature-flag system: same enterprise/devtool audience, same OpenAPI-first Go + Next.js shape, but solves a different real problem — teams losing track of *what services exist, who owns them, what APIs they expose, and how they depend on each other*.

- **Goal:** portfolio showcase demonstrating full-stack capability on Azure at Elisa scale.
- **Timeline:** 2–4 weeks part-time.
- **Standout feature:** interactive service-dependency graph.
- **Narrative:** *"After building flag governance at Elisa, I built service governance."*

## Scope

### In scope (MVP)
- Auth via **Microsoft Entra External ID** + dev-mode fallback for local.
- Entities: `Team`, `User`, `Service`, `Dependency`, `ApiSpec`, `Tag`.
- RBAC roles **ADMIN / EDITOR / VIEWER**, scoped per team (carry over from Elisa).
- Service CRUD with metadata (name, description, owner team, repo URL, runbook URL, lifecycle: prod/beta/deprecated, tags).
- OpenAPI spec upload → validate → version → render with Redoc.
- Declare upstream/downstream dependencies between services; detect cycles on insert.
- Postgres full-text search across name/description/tags.
- **Dependency graph visualization** (React Flow) — the standout feature.
- CI/CD to Azure Container Apps via GitHub Actions with OIDC (no static cloud secrets).
- Observability via Application Insights + structured JSON logs.

### Explicitly out of scope
Plugin system, software templates/scaffolding, TechDocs hosting, scorecards, multi-cloud, on-call/incidents, billing — all Backstage features that don't strengthen the portfolio narrative.

## Tech Stack

**Backend (Go 1.22+)**
- HTTP: `chi` router
- OpenAPI codegen: `oapi-codegen` (generates types + chi server interfaces from `api/openapi.yaml`)
- DB: `pgx` + `sqlc` for type-safe queries
- Migrations: `golang-migrate`
- Auth: `github.com/coreos/go-oidc` for Entra JWT validation
- Logging: stdlib `log/slog`
- Testing: `testify` + `testcontainers-go` (real Postgres in integration tests)

**Frontend (Next.js 15 + TS)**
- App Router, server components where they help
- Tailwind + `shadcn/ui` for the design system
- TanStack Query for server state
- React Hook Form + Zod
- Auth: NextAuth.js with Entra provider (simpler than raw MSAL for App Router)
- `react-flow` for the dependency graph
- `redoc` for OpenAPI rendering
- Vitest + Testing Library; one Playwright happy-path

**Azure**
- **Container Apps** (backend + frontend, two apps)
- **Azure Container Registry**
- **Postgres Flexible Server** (Burstable B1ms is fine for portfolio)
- **Entra External ID** for auth
- **Key Vault** for secrets; **Managed Identity** for Container Apps → Postgres/Key Vault
- **Application Insights**
- IaC: **Bicep** (Microsoft-native, no extra tooling)

**CI/CD**
- GitHub Actions, OIDC federation to Azure (no long-lived service principal secret)

## Repository Layout

Single monorepo. Justified by: shared OpenAPI spec as source of truth, single CI pipeline, full-stack PRs in one diff, solo developer.

```
internal-dev-portal/
├── backend/
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── api/          # generated + handler wiring
│   │   ├── auth/         # OIDC middleware, RBAC
│   │   ├── db/           # sqlc-generated queries + migrations
│   │   ├── services/     # service registry domain
│   │   ├── deps/         # dependency graph + cycle detection
│   │   ├── specs/        # OpenAPI upload + validation
│   │   └── search/       # FTS wrapper
│   ├── api/openapi.yaml  # SINGLE SOURCE OF TRUTH
│   ├── migrations/
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── app/              # Next.js App Router
│   ├── components/
│   ├── lib/
│   │   └── api/          # generated TS client from openapi.yaml
│   ├── Dockerfile
│   └── package.json
├── deploy/
│   ├── bicep/            # main.bicep + modules
│   └── github/           # reusable workflow snippets
├── docs/
│   ├── architecture.md
│   ├── adr/              # 3–5 short ADRs (auth choice, monorepo, sqlc vs gorm, etc.)
│   └── screenshots/
├── .github/workflows/
│   ├── ci.yml            # lint + test + build
│   └── deploy.yml        # build images → push ACR → deploy Container Apps
├── docker-compose.yml    # local dev: postgres + backend + frontend
└── README.md
```

## Roadmap

### Week 1 — Foundations & first vertical slice
- Initialize monorepo, README skeleton, ADR-0001 (monorepo).
- Backend: `go mod init`, chi server, `/healthz`, structured logging.
- Define `openapi.yaml` for `Service` CRUD + `Team` listing.
- `oapi-codegen` config; verify generated server interface compiles.
- Postgres migrations: `teams`, `users`, `services`, `tags`, `service_tags`.
- `sqlc` queries for services.
- `docker-compose.yml` for Postgres + backend + frontend hot-reload.
- Frontend: Next.js + Tailwind + shadcn/ui scaffold, base layout, services list page (mock data).
- **Milestone:** `curl localhost:8080/api/v1/services` returns seeded data; frontend lists them.

### Week 2 — Auth + RBAC + Service management UI
- Register Entra External ID tenant + app registrations (backend API, frontend SPA).
- Backend middleware: OIDC token verification, claims → `User` + roles.
- Dev-mode auth toggle (`AUTH_MODE=dev` accepts a header for local).
- RBAC enforcement on each handler.
- Services CRUD endpoints (POST/PUT/DELETE) wired to UI.
- Frontend: NextAuth Entra provider, login flow, protected routes, service create/edit forms with React Hook Form + Zod.
- **Milestone:** logged-in EDITOR can create/edit a service in deployed-feeling UX; VIEWER is read-only.

### Week 3 — OpenAPI specs, dependencies, search
- `api_specs` table (versioned per service, content in DB as JSONB or Blob — start with JSONB).
- Upload endpoint with OpenAPI validation (`libopenapi`).
- Frontend: spec upload UI + Redoc render of latest version.
- `service_dependencies` table; POST/DELETE endpoints; cycle detection (recursive CTE on insert).
- Postgres FTS: `tsvector` column on services, GIN index, search endpoint with ranking.
- Frontend: global search bar, results page with filters (lifecycle, tag, team).
- **Milestone:** can upload `petstore.yaml`, render docs, declare A→B, see cycle attempt rejected, find by keyword.

### Week 4 — Dependency graph viz, deploy, polish
- Backend: `GET /api/v1/graph?scope=team:X` returns nodes + edges.
- Frontend: React Flow graph view; click a node → service detail drawer; filter by lifecycle/tag.
- Bicep templates: resource group, ACR, Container Apps env, two apps, Postgres, Key Vault, App Insights, managed identities, role assignments.
- GitHub Actions: CI (lint, vet, test, build images) and Deploy (push to ACR, `az containerapp update`).
- App Insights wired (OTel Go SDK + Next.js instrumentation).
- Playwright happy-path: log in → create service → upload spec → declare dependency → see in graph.
- README: hero screenshot, architecture diagram, demo GIF, deployment instructions, decisions/ADR links.
- **Milestone:** publicly reachable URL, GitHub README that sells the project in 30 seconds.

## Key Design Decisions

- **OpenAPI-first.** `backend/api/openapi.yaml` is the contract. Backend generates Go types/server interfaces with `oapi-codegen`; frontend generates a typed TS client (`openapi-typescript` + `openapi-fetch`). Same playbook as Elisa.
- **Dev-mode auth fallback.** Real Entra in deployed envs; an env-gated header-based identity in local docker-compose. Keeps tests fast and onboarding painless without weakening production auth.
- **sqlc over ORM.** Type-safe, SQL-first, no runtime magic — easier to discuss in interviews and matches Go community norms.
- **Single shared graph endpoint.** Avoids the frontend doing N+1 lookups to render the graph; a single response with nodes + edges feeds React Flow directly.
- **Bicep over Terraform.** Single Azure target → Bicep keeps tooling minimal and signals Azure fluency.

## Data Model (minimum)

```
teams(id, name, slug, created_at)
users(id, entra_oid, email, name, created_at)
team_members(team_id, user_id, role)         -- ADMIN/EDITOR/VIEWER
services(id, team_id, name, slug, description, repo_url, runbook_url,
         lifecycle, search_tsv, created_at, updated_at)
tags(id, name)
service_tags(service_id, tag_id)
api_specs(id, service_id, version, content_jsonb, uploaded_by, uploaded_at)
service_dependencies(upstream_id, downstream_id, created_at)   -- PK both cols
audit_log(id, actor_id, action, entity_type, entity_id, payload, at)
```

## Verification

**Local end-to-end:**
1. `docker-compose up` → Postgres + backend + frontend healthy.
2. `cd backend && go test ./...` — all unit + integration tests pass (testcontainers spins a real Postgres).
3. `cd frontend && pnpm test` — Vitest green.
4. `pnpm exec playwright test` — happy-path E2E green.
5. Manual: open `http://localhost:3000`, log in via dev-mode, create a team, register a service, upload `samples/petstore.yaml`, declare a dependency, view it in the graph.

**Deployed end-to-end:**
1. Push to `main` → CI green → Deploy workflow updates both Container Apps.
2. Smoke test script hits `/healthz` on both apps + one authenticated endpoint with a service principal token.
3. App Insights shows incoming requests with trace IDs; no exceptions in last 1h.
4. Log into deployed URL with a real Entra account, run the same manual flow as local.

**Portfolio-readiness check (the *real* acceptance criteria):**
- A recruiter can read the README in 60 seconds and understand the value.
- A senior engineer can read the architecture doc + skim the OpenAPI spec in 5 minutes and find it credible.
- The dependency graph view produces a screenshot worth pinning at the top of the README.
