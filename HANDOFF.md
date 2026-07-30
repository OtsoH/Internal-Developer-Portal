# Handoff

## Where we are

**Week 2, step 9 of 14.** Goal and 4-week roadmap: `docs/app-plan.md`.
Plan with per-step detail, verification commands, risks and the Entra setup
appendix: `C:\Users\otsoh\.claude\plans\plan-how-to-implement-cached-simon.md`.
Its "Progress and deviations" section is the authoritative record of where
reality differed from the plan — read that, not a summary of it.

Steps 1–8 have landed on `main`: `git log --oneline ab9bc22..HEAD`.

Traps live in skills that load when you enter the relevant tree:
`backend-gotchas` (backend/) and `frontend-gotchas` (frontend/). Machine-level
facts are in the memory directory. Read the matching one before you start.

## Next step

**Step 9 — mutation handlers.** `CreateService`/`UpdateService`/`DeleteService`
still answer 501. New: `internal/api/{mutations,pgerr,audit}.go` +
`mutations_test.go`. Shape and rationale are in the plan; the parts that are easy
to get wrong:

- **Authorization is in the handlers, 403, via one `requireRole(ctx, teamID, min)`.**
  Not a `StrictMiddlewareFunc` — see the plan for why, and `backend-gotchas`.
- **Check order.** Create: `requireRole(body.TeamId, EDITOR)` *before* any query,
  so a bogus `teamId` is 403 and never leaks whether the team exists. Update and
  delete: tx → `GetServiceForUpdate` → `ErrNoRows` = 404 → role check on the
  *existing* row's team, and on the new team too if the team is changing.
- **One transaction per mutation**, via `inTx(ctx, func(q *dbgen.Queries) error)`.
  The callback returns a plain error, so expected outcomes need sentinels
  (`errForbidden`, `errNotFound`, `errSlugConflict`) plus `errors.Is`.
- **Match the constraint name, not just SQLSTATE 23505.** `services_slug_key` is
  409 `slug_taken`; `tags_name_key` raises 23505 too and must not report as one.
- **`nilIfBlank` for repo/runbook URLs.** A form submits `""` for an untouched
  field, and storing that renders an empty `repo ↗` link.
- The delete audit payload must be captured *before* the row is gone.

Verify with curl as each persona: editor creates in Payments → 201 with
normalized tags; viewer → 403; editor in Platform (VIEWER there) → 403; duplicate
slug → 409; PUT → 200 with `updatedAt` moved; admin DELETE in Payments → 403.
Then inspect `audit_log`.

## Remaining steps

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
