# Backend

Go API for the Internal Developer Portal. The [root README](../README.md) has the full picture.

The server is a chi router with structured JSON logging (`log/slog`). It talks to Postgres through pgx and sqlc-generated queries, and its HTTP surface is generated from the OpenAPI contract at `api/openapi.yaml`. The router itself is assembled in `internal/app`, so tests can exercise the real middleware chain rather than a lookalike.

Everything under `/api/v1` requires authentication; `/healthz` is public. The read endpoints (GET /services, GET /teams, GET /me) are wired to the database. Mutations still return 501; they land with RBAC in step 9.

## Development

The whole stack runs in Docker with hot reload (air):

```sh
docker compose up -d --build
```

To run the backend on its own against the containerized Postgres:

```sh
docker compose up -d postgres
DATABASE_URL='postgres://idp:idp@localhost:5433/idp?sslmode=disable' APP_SEED=true go run ./cmd/api
```

It listens on port 8080. `/healthz` returns `{"status":"ok"}`, and the API is mounted under `/api/v1`. Without `DATABASE_URL` the server still starts, but it skips migrations and serves stub data only.

Note the host port 5433, not 5432. A natively installed Postgres tends to own 5432 on Windows, so compose maps the container to 5433 to stay out of its way.

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `DATABASE_URL` | (none) | Postgres connection string. Omit it to run without a database. |
| `APP_SEED` | `false` | Set to `true` to load the idempotent dev seed on startup. |
| `PORT` | `8080` | HTTP listen port. |
| `LOG_LEVEL` | `info` | One of `debug`, `info`, `warn`, `error`. |
| `AUTH_MODE` | `dev` | `dev` or `entra`. Any other value is a startup error, never a silent fallback. |
| `OIDC_ISSUER` | (none) | Required when `AUTH_MODE=entra`. E.g. `https://<tenant>.ciamlogin.com/<tenantId>/v2.0`. |
| `OIDC_AUDIENCE` | (none) | Required when `AUTH_MODE=entra`. Must match the token's `aud`, which is the backend app registration and not Microsoft Graph. |

## Authentication

`AUTH_MODE=dev` (the default) trusts an `X-Dev-User` header naming a seeded user:

```sh
curl -H 'X-Dev-User: dev.editor@example.com' localhost:8080/api/v1/me
```

The header is read only in dev mode. In `entra` mode it is not consulted at all,
so it cannot be used to impersonate anyone. Dev mode never creates users either:
an address that isn't already in the `users` table is a 401, so a typo fails
loudly instead of yielding a role-less account.

That said, dev mode is a trust-the-network model. Anything that can reach the
port can assert any identity, and compose publishes 8080 on the host.
**Never expose a backend running with `AUTH_MODE=dev`.** The server logs a
warning at startup to make the mode obvious in any log you're reading.

`AUTH_MODE=entra` verifies Entra-issued JWTs. Discovery happens once at boot, so
an unreachable tenant fails the process immediately rather than every request
after it. Roles are never read from the token. They come from `team_members` on
every request, so a membership change takes effect without re-authenticating.

Without `DATABASE_URL` there is nothing to resolve a user against, so the
authenticator is not mounted: reads return empty results and writes answer 503.

## Database

Migrations live in `migrations/` and are embedded into the binary. golang-migrate runs them at startup as a library, so there is no separate migrate CLI to install. The seed in `internal/db/seed.sql` uses fixed UUIDs with `ON CONFLICT DO NOTHING`, so it is safe to re-run.

## Regenerating from the contract

`api/openapi.yaml` is the source of truth for the HTTP layer, and `internal/db/queries/*.sql` for the database layer. Both codegens run from this directory:

```sh
go generate ./...        # oapi-codegen: server interfaces from the OpenAPI spec
go tool sqlc generate    # sqlc: typed query methods from the SQL files
```

The tools are pinned as `tool` directives in `go.mod` (oapi-codegen v2.7.2, sqlc v1.31.1), so the versions travel with the repo and there are no global installs.

## Tests and lint

```sh
go test ./...            # full suite; the integration test needs Docker running
go test -short ./...     # unit tests only, no Docker required
go tool golangci-lint run
```

The integration test in `internal/db` spins up a real Postgres with testcontainers, so Docker needs to be running for it to pass; `-short` skips it. golangci-lint is pinned as a `tool` directive in `go.mod` like the codegens (config in `.golangci.yml`), so local runs and CI use the same version. The first run compiles it from source and is slow; after that it's cached.
