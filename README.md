# Internal Developer Portal

A Backstage-lite service catalog: it tracks what services exist, who owns them, what APIs they expose, and how they depend on each other.

> After building flag governance at Elisa, I built service governance.

## Features (MVP)

- Service catalog with ownership, lifecycle, repo/runbook links and tags
- Auth via Microsoft Entra External ID, with ADMIN/EDITOR/VIEWER roles per team
- OpenAPI spec upload, validation, versioning and Redoc rendering
- Interactive dependency graph (React Flow) with cycle detection
- Postgres full-text search across names, descriptions and tags
- Runs on Azure: Container Apps, Postgres Flexible Server, Key Vault, App Insights, Bicep IaC, GitHub Actions with OIDC

## Tech Stack

| Layer | Tech |
|---|---|
| Backend | Go, chi, oapi-codegen, pgx + sqlc, golang-migrate, log/slog |
| Frontend | Next.js (App Router), TypeScript, Tailwind, shadcn/ui, TanStack Query, React Flow |
| Contract | OpenAPI 3 (`backend/api/openapi.yaml`), the single source of truth for both codegens |
| Database | PostgreSQL 17 |
| Infra | Azure Container Apps, ACR, Key Vault, App Insights, defined in Bicep |
| CI/CD | GitHub Actions with OIDC federation (no static cloud secrets) |

## Repository Layout

```
├── backend/          # Go API (chi + sqlc), OpenAPI spec, migrations
├── frontend/         # Next.js app
├── deploy/
│   ├── bicep/        # Azure IaC
│   └── github/       # reusable workflow snippets
├── docs/
│   └── adr/          # architecture decision records
└── docker-compose.yml
```

## Quickstart

Requires Docker Desktop.

```sh
docker compose up -d --build
```

| Service | URL |
|---|---|
| Frontend | http://localhost:3000 |
| API | http://localhost:8080/api/v1/services |
| Health | http://localhost:8080/healthz |
| Postgres | localhost:5433 (`idp`/`idp`, db `idp`); host port 5433 avoids clashing with a native install |

The database is migrated and seeded automatically on backend startup (`APP_SEED=true` in compose). Both apps hot-reload on edit, via polling because Windows bind mounts emit no file events.

**Fastest dev loop on Windows:** run only the infra in Docker and the frontend natively:

```sh
docker compose up -d postgres backend
cd frontend && pnpm install && pnpm dev
```

### Regenerating from the OpenAPI contract

`backend/api/openapi.yaml` is the single source of truth:

```sh
cd backend && go generate ./...     # Go server interfaces (oapi-codegen) + sqlc queries
cd frontend && pnpm generate:api    # TypeScript client types (openapi-typescript)
```

## Documentation

- [Implementation plan](docs/app-plan.md)
- [ADRs](docs/adr/)
