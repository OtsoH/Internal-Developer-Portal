# Handoff

## Where we are

**Week 2, step 8 of 14.** Goal and 4-week roadmap: `docs/app-plan.md`.
Plan with per-step detail, verification commands, risks and the Entra setup
appendix: `C:\Users\otsoh\.claude\plans\plan-how-to-implement-cached-simon.md`.
Its "Progress and deviations" section is the authoritative record of where
reality differed from the plan — read that, not a summary of it.

Steps 1–7 have landed on `main`: `git log --oneline ab9bc22..HEAD`.

Traps live in skills that load when you enter the relevant tree:
`backend-gotchas` (backend/) and `frontend-gotchas` (frontend/). Machine-level
facts are in the memory directory. Read the matching one before you start.

## Next step

**Step 8 — spec-driven request validation** via `oapi-codegen/nethttp-middleware`.
oapi-codegen generates no validation, so a 5000-character name currently reaches
Postgres. `embedded-spec: true` already emits `GetSwagger()` and `kin-openapi` is
already a direct dependency, so the spec can do the work.

Mount it in `internal/app/router.go` on `apiRouter`, **after** the authenticator.
Three traps, all in the one middleware:

1. Set `swagger.Servers = nil`, or the validator expects a doubly-prefixed path.
2. Supply an `AuthenticationFunc` returning `nil`. kin-openapi enforces the
   spec's `security` block, so without it **every dev-mode request 401s**.
3. Replace the default error handler; it emits `text/plain`. Reuse the `Error`
   shape from `api.StrictOptions` so validation failures match every other 400.

Droppable step: if it fights back, hand-write `validateServiceCreate` instead
(the plan has the fallback) and fold it into step 9.

Verify: `?lifecycle=bogus` → 400 JSON, `?lifecycle=beta` → 200,
`/services/not-a-uuid` → 400, `/healthz` still 200. Note the behaviour change —
`?lifecycle=bogus` is currently ignored silently.

## Remaining steps

- [ ] 9 — mutation handlers: transactions, tags, audit rows, pg-error mapping, 403
- [ ] 10 — API integration test, the role × operation × team matrix
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
