# Internal Developer Portal

A Backstage-lite **Internal Developer Portal**: a service catalog that tracks what services exist, who owns them, what APIs they expose, and how they depend on each other.

> After building flag governance at Elisa, I built service governance.

## Features (MVP)

- 📇 **Service catalog** — CRUD with ownership, lifecycle, repo/runbook links, tags
- 🔐 **Auth & RBAC** — Microsoft Entra External ID, ADMIN/EDITOR/VIEWER per team
- 📜 **API specs** — upload, validate, version and render OpenAPI docs (Redoc)
- 🕸️ **Dependency graph** — interactive React Flow visualization with cycle detection
- 🔎 **Search** — Postgres full-text search across names, descriptions and tags
- ☁️ **Azure-native** — Container Apps, Postgres Flexible Server, Key Vault, App Insights, Bicep IaC, GitHub Actions with OIDC

## Tech Stack

| Layer | Tech |
|---|---|
| Backend | Go, chi, oapi-codegen, pgx + sqlc, golang-migrate, log/slog |
| Frontend | Next.js (App Router), TypeScript, Tailwind, shadcn/ui, TanStack Query, React Flow |
| Contract | OpenAPI 3 (`backend/api/openapi.yaml`) — single source of truth for both codegens |
| Database | PostgreSQL 17 |
| Infra | Azure Container Apps, ACR, Key Vault, App Insights — Bicep |
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

_Coming with the first vertical slice — `docker compose up` will start Postgres + backend + frontend._

## Documentation

- [Implementation plan](docs/app-plan.md)
- [ADRs](docs/adr/)
