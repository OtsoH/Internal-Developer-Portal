# Backend

Go API for the Internal Developer Portal. The [root README](../README.md) has the full picture.

The server is a chi router with structured JSON logging (`log/slog`). It talks to Postgres through pgx and sqlc-generated queries, and its HTTP surface is generated from the OpenAPI contract at `api/openapi.yaml`. Right now the read endpoints (GET /services, GET /teams) are wired to the database; the mutation endpoints return 501 until week 2.

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

## Database

Migrations live in `migrations/` and are embedded into the binary. golang-migrate runs them at startup as a library, so there is no separate migrate CLI to install. The seed in `internal/db/seed.sql` uses fixed UUIDs with `ON CONFLICT DO NOTHING`, so it is safe to re-run.

## Regenerating from the contract

`api/openapi.yaml` is the source of truth for the HTTP layer, and `internal/db/queries/*.sql` for the database layer. Both codegens run from this directory:

```sh
go generate ./...        # oapi-codegen: server interfaces from the OpenAPI spec
go tool sqlc generate    # sqlc: typed query methods from the SQL files
```

The tools are pinned as `tool` directives in `go.mod` (oapi-codegen v2.7.2, sqlc v1.31.1), so the versions travel with the repo and there are no global installs.

## Tests

```sh
go test ./...
```

The integration test in `internal/db` spins up a real Postgres with testcontainers, so Docker needs to be running for it to pass.
