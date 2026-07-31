# Handoff

## Where we are

**Week 2, step 13 of 14.** Goal and 4-week roadmap: `docs/app-plan.md`.
Plan with per-step detail, verification commands, risks and the Entra setup
appendix: `C:\Users\otsoh\.claude\plans\plan-how-to-implement-cached-simon.md`.
Its "Progress and deviations" section is the authoritative record of where
reality differed from the plan — read that, not a summary of it.

Steps 1–11 have landed on `main`: `git log --oneline ab9bc22..HEAD`. **Step 12
is written and verified but not yet committed** — `ServiceForm`,
`/services/new`, `lib/api/errors.ts` and the role-gated list button.
`pnpm lint`, `pnpm typecheck`, `pnpm test` (73 tests, 8 files) and `pnpm build`
are clean, and both dev personas were walked through the running stack.

Traps live in skills that load when you enter the relevant tree:
`backend-gotchas` (backend/) and `frontend-gotchas` (frontend/). Machine-level
facts are in the memory directory. Read the matching one before you start.

## Next step

**Step 13 — detail page, edit, admin-only delete.** New:
`app/services/[serviceId]/page.tsx`, `.../edit/page.tsx`,
`components/services/delete-service-button.tsx` + tests. Modified:
`app/services/page.tsx` (link service names to the detail page).

Detail page (server): `notFound()` on 404, role from `getCurrentUser()`, metadata
in the existing panel language, and an action row — Edit at ≥EDITOR, Delete at
ADMIN. This is also where week 3's spec upload and week 4's dependency panel land.

Edit page: renders the read-only panel if the user cannot edit the owning team.
The team `<Select>` shows editable teams **plus the current owning team**, so a
viewer-of-current-team scenario cannot silently reassign ownership.

`delete-service-button.tsx` (`"use client"`): flips to an inline "Delete
permanently? · confirm / cancel" in place rather than opening a dialog — no
`alert-dialog` primitive needed, and the destructive action stays anchored.

Most of the work is already done. `ServiceForm` supports `mode="edit"` and is
tested in both modes; the edit page only has to fetch the service and render it.
`unwrap()` in `lib/api/errors.ts` already handles the 204 that DELETE returns.
**One thing to change:** `ServiceForm`'s `onSuccess` currently sends a *created*
service back to `/services` because the detail page did not exist — point it at
`/services/${saved.id}` once it does, and update its test.

> **`router.refresh()` in `onSuccess` is essential and easy to forget** —
> TanStack Query's cache has nothing to do with the RSC payload that renders the
> services table.

Verify: `pnpm lint && pnpm typecheck && pnpm test && pnpm build`, then Dev Editor
edits a Payments service and sees no Delete button; Dev Admin deletes a Platform
service; Dev Viewer sees no action buttons and gets the read-only panel at
`/edit`. Nobody is ADMIN on Payments in the seed — use Platform for delete.

## Remaining steps

- [ ] 14 — docs: ADR-0002, `docs/entra-setup.md`, README updates. **`CLAUDE.md`'s
      "No CORS" bullet still describes a Next rewrite; the rule survives but the
      mechanism does not.**
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
