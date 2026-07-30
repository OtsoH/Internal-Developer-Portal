# Handoff

## Where we are

**Week 2, step 10 of 14.** Goal and 4-week roadmap: `docs/app-plan.md`.
Plan with per-step detail, verification commands, risks and the Entra setup
appendix: `C:\Users\otsoh\.claude\plans\plan-how-to-implement-cached-simon.md`.
Its "Progress and deviations" section is the authoritative record of where
reality differed from the plan — read that, not a summary of it.

Steps 1–9 have landed on `main`: `git log --oneline ab9bc22..HEAD`.

Traps live in skills that load when you enter the relevant tree:
`backend-gotchas` (backend/) and `frontend-gotchas` (frontend/). Machine-level
facts are in the memory directory. Read the matching one before you start.

## Next step

**Step 10 — API integration test.** New: `internal/api/api_integration_test.go`,
package `api_test`. Same shape as `db_integration_test.go`: `testing.Short()`
skip, testcontainers `postgres:17-alpine`, `db.Migrate` + `db.Seed`, then
`app.NewRouter(...)` behind `httptest.NewServer`.

This is what buys back the cost of authorization living in the handlers rather
than in middleware: enforcement is not structural, so the matrix is what catches
a handler that forgets. Table-driven, a deliberate departure from
`handlers_test.go`'s style at 18 cells.

Cover: 401 with no header and with an unknown email; {admin, editor, viewer} ×
{POST, PUT, DELETE} × {Platform, Payments}; duplicate slug → 409 `slug_taken`;
bad slug → 400; PUT unknown uuid → 404; PUT moving a service to a team you cannot
edit → 403 *with the row unchanged*; tag round-trip and normalization; a removed
tag losing its `service_tags` link while the `tags` row survives; `updatedAt`
strictly greater after PUT; DELETE then GET → 204 then 404; one `audit_log` row
per successful mutation, the delete row outliving the service; `GET /me` shape.

Verify: Docker up, `go test ./internal/api/... -run Integration -v`, and
`go test -short ./...` still skips it.

## Remaining steps

- [ ] 11 — frontend form deps, shadcn primitives, Zod schema
- [ ] 12 — create form + role-gated list button
- [ ] 13 — detail page, edit, admin-only delete
- [ ] 14 — docs: ADR-0002, `docs/entra-setup.md`, README updates
- [ ] Manual, needs the user's Azure account: Entra tenant + two app registrations
      (checklist is the plan's appendix)

## Standing constraints

- **Claude never commits or pushes.** Propose a dependency-ordered split with
  exact `git add` paths and say whether the ordering was verified or reasoned.
- **OpenAPI-first**: spec first, then regenerate both sides. CI fails on drift.
- **No CORS.** The BFF route handler is the only browser-to-backend path; never
  reintroduce a `next.config.ts` rewrite, which cannot strip forged headers.
- **Never expose a backend running `AUTH_MODE=dev`** — anything that reaches the
  port can assert any identity. Compose publishes 8080 on the host.
- **Roles come from `GET /me`, never from the session or token.**
- Every step must stay verifiable with `AUTH_MODE=dev` alone; the real Entra path
  stays unverified until the user runs the manual appendix.
- Use the design skills and Playwright for UI work; run `humanizer` over
  user-facing prose.
- **Still on `main`.** `ci.yml` already supports branch+PR (both triggers exist);
  the switch is deferred to the week 2 → 3 boundary and blocked on installing
  `gh` (`winget install --id GitHub.cli`). Until then CI is checked via the badge
  SVG, because unauthenticated API polling burns the 60 req/h limit.

## Open questions for the user

- Adopt feature branches + PRs at the week 2 → 3 boundary, or stay on `main`?
  Blocked on installing `gh` first.
