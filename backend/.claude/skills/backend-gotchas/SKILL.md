---
name: backend-gotchas
description: Traps this Go backend has already hit — read before editing backend/ Go code, changing api/openapi.yaml, running go generate or sqlc generate, adding a migration, wiring handlers or middleware, or debugging a golangci-lint failure. Each entry cost real time to find; none are guessable from the source.
---

# Backend gotchas

Every entry here was paid for once. Re-read the relevant section before touching
that area rather than rediscovering it.

## OpenAPI and oapi-codegen

**One named response component per status. Never collapse them.**
oapi-codegen tracks used response refs *per operation*. If several statuses share
one component, only the first gets the embedded `struct{ XJSONResponse }` form
and the rest degrade to a bare `type X Error`. The generated Go shape then
depends on which statuses an operation happens to declare, so adding a 401 to one
endpoint silently reshapes its 404 and breaks the handler at build time.
`components/responses` therefore has `BadRequest`/`Unauthorized`/`Forbidden`/
`NotFound`/`Conflict` as separate entries; `ErrorResponse` is only for `default`.

**Three error-handler hooks exist, at three depths.** Wiring only some leaves the
others emitting `text/plain` with `err.Error()` in the body:

| Hook | Fires on |
|---|---|
| `ChiServerOptions.ErrorHandlerFunc` | path/query parameter binding |
| `StrictHTTPServerOptions.RequestErrorHandlerFunc` | JSON body decode |
| `StrictHTTPServerOptions.ResponseErrorHandlerFunc` | handler returned a non-nil error |

Use `NewStrictHandlerWithOptions` + `HandlerWithOptions` with
`api.StrictOptions(logger)` and `api.ChiErrorHandler`, never the bare
`NewStrictHandler(x, nil)` / `HandlerFromMux` pair. Policy: 4xx messages are safe
to echo (they contain only spec-declared parameter names and the caller's own
input); 5xx stays opaque with the detail logged. `internal/api/errors_test.go`
fails if this regresses.

**The `output:` path in `api/oapi-codegen.yaml` resolves relative to the
`go:generate` directive's directory**, not the config file. A `../internal/api/gen.go`
value once created a stray `internal/internal/` tree. It is just `gen.go`.

## sqlc

**A type override does not cover the nullable case.** `db_type: "uuid"` matches
only non-nullable columns, so `audit_log.actor_id` generated as `pgtype.UUID`
instead of `*uuid.UUID` despite `emit_pointers_for_null_types`. `sqlc.yaml` now
carries a second override with `nullable: true` and `pointer: true`. Any new
nullable column of an overridden type needs the same treatment.

**Adding a migration needs no Go change.** `migrations/embed.go` globs `*.sql`
and `sqlc.yaml`'s `schema:` points at the directory, so both pick up new files
automatically.

## Go and golangci-lint

**Do not add an unexported helper before its first caller.** `unused` flags it
and the build fails lint. Add each helper in the change that calls it.

**A nil pointer assigned into an interface field is not nil.** Passing a nil
`*pgxpool.Pool` into an `api.TxBeginner` field yields a non-nil interface holding
a nil pointer, so a `!= nil` guard passes and the code panics later.
`txBeginner(pool)` in `cmd/api/main.go` reintroduces the nil at the conversion.
Any new `Deps` field taking a possibly-nil pointer as an interface needs this.

**`go get <module>` can leave `go.sum` incomplete.** `go get github.com/coreos/go-oidc/v3`
left missing entries for `go-jose/v4` and `golang.org/x/oauth2`. Fetch the
*package* path instead: `go get github.com/coreos/go-oidc/v3/oidc@v3.20.0`.

**`go test -race` does not run on this machine.** It needs cgo and there is no C
toolchain here. Run plain `go test` locally; CI on ubuntu covers `-race`.

## Migrations and seed

golang-migrate runs as a **library** at startup from an embedded FS, so there is
no CLI to install. The pgx/v5 driver registers the `pgx5` scheme, so the URL is
rewritten `postgres://` → `pgx5://` in `internal/db/migrate.go`. "No change" is
success. The seed is idempotent through fixed UUIDs plus `ON CONFLICT DO NOTHING`.

With N replicas, golang-migrate takes a `pg_advisory_lock` so concurrent boots
serialize. A migration that fails midway marks the schema **dirty** and every
replica then refuses to start until it is cleared by hand.

## Verifying codegen is drift-free locally

`git diff --exit-code` compares against the last commit, so with uncommitted work
in the tree it reports your own changes as drift. Hash the generated files,
re-run both generators, hash again, compare:

```sh
before=$(sha256sum internal/api/gen.go internal/db/gen/*.go | sha256sum)
go generate ./... && go tool sqlc generate
after=$(sha256sum internal/api/gen.go internal/db/gen/*.go | sha256sum)
[ "$before" = "$after" ] && echo "drift-free"
```

## Where authorization lives

Authentication is chi middleware answering 401 (`internal/auth`). **Authorization
is in the handlers**, answering 403, because the required role depends on the
owning team of the row being changed. A `StrictMiddlewareFunc` would need an
unchecked `switch operationID` to build the right typed 403 response and would
fail open on any newly added operation. The integration test's role × operation
matrix is what catches a handler that forgets.
