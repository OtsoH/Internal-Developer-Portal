# Handoff

## Where we are

**Week 2, step 12 of 14.** Goal and 4-week roadmap: `docs/app-plan.md`.
Plan with per-step detail, verification commands, risks and the Entra setup
appendix: `C:\Users\otsoh\.claude\plans\plan-how-to-implement-cached-simon.md`.
Its "Progress and deviations" section is the authoritative record of where
reality differed from the plan — read that, not a summary of it.

Steps 1–10 have landed on `main`: `git log --oneline ab9bc22..HEAD`. Step 11
(form deps, shadcn primitives, `lib/services/schema.ts`) is written and verified
— `pnpm lint`, `pnpm typecheck`, `pnpm test` (47 tests, 5 files) and `pnpm build`
all clean, container rebuilt — but not yet committed.

Traps live in skills that load when you enter the relevant tree:
`backend-gotchas` (backend/) and `frontend-gotchas` (frontend/). Machine-level
facts are in the memory directory. Read the matching one before you start.

## Next step

**Step 12 — create form and role gating.** New: `app/services/new/page.tsx`,
`components/services/service-form.tsx`, `lib/api/errors.ts` + tests. Modified:
`app/services/page.tsx` + its test.

Dedicated page, not a dialog: linkable URL, server-side role gating before any
client JS ships, and step 13's edit route falls out for free.

`app/services/new/page.tsx` (server) calls `requireSession()` + `getCurrentUser()`
and filters `teamRoles` to EDITOR/ADMIN, rendering a read-only notice if that
list is empty. **The team `<Select>` is populated only from editable teams**, so
the user cannot express a request that would 403. `/teams` is never needed.

`ServiceForm` (`"use client"`) takes `mode: "create" | "edit"`, RHF +
`zodResolver` over `serviceFormSchema`, TanStack Query `useMutation` over
`api.POST`. Slug auto-fills from the name via `slugify` until the user touches
the slug field. Errors route by code: `slug_taken` → `form.setError("slug", …)`
inline, not a toast; 403 → toast; 401 → `router.push("/signin")`.

> **`router.refresh()` in `onSuccess` is essential and easy to forget** —
> TanStack Query's cache has nothing to do with the RSC payload that renders the
> services table.

The list page fills its empty `<div>` sibling with a "New service" button,
disabled with a `title` when the user has no editor role, matching the header's
existing "Coming soon" idiom. Empty-state copy loses its "arrives in week 2"
line. Design language: `rounded-lg border bg-card p-6` panel, mono for
slug/tags, `font-mono text-xs text-muted-foreground` descriptions, and the
lifecycle `<Select>` rendering the same `bg-status-*` dots as the table.

Verify: `pnpm lint && pnpm typecheck && pnpm test && pnpm build`, then dev Editor
registers a service under Payments (redirect, toast, list updated) and dev Viewer
sees the disabled button. Playwright screenshot — step 11 shipped no visible UI,
so this is the first browser check since the form work began.

## Remaining steps

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
