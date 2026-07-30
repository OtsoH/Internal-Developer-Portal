# Handoff

## Where we are

**Week 2, step 11 of 14.** Goal and 4-week roadmap: `docs/app-plan.md`.
Plan with per-step detail, verification commands, risks and the Entra setup
appendix: `C:\Users\otsoh\.claude\plans\plan-how-to-implement-cached-simon.md`.
Its "Progress and deviations" section is the authoritative record of where
reality differed from the plan — read that, not a summary of it.

Steps 1–9 have landed on `main`: `git log --oneline ab9bc22..HEAD`. Step 10
(`internal/api/api_integration_test.go`, the RBAC integration matrix) is written
and verified — `go test ./internal/api/... -run Integration -v` passes all 27
subtests, `go test ./...` and lint are clean — but not yet committed.

Traps live in skills that load when you enter the relevant tree:
`backend-gotchas` (backend/) and `frontend-gotchas` (frontend/). Machine-level
facts are in the memory directory. Read the matching one before you start.

## Next step

**Step 11 — frontend form deps, shadcn primitives, Zod schema.** Modified:
`package.json`, `app/layout.tsx`. New: `components/ui/{input,label,textarea,
select,form,sonner}.tsx`, `lib/services/schema.ts` + test.

Deps: `react-hook-form`, `zod@^4`, `@hookform/resolvers@^5` (v5+ required for
Zod 4), `sonner`, `@testing-library/user-event` (dev). Two install-time traps:
`shadcn add` pulls split `@radix-ui/react-*` packages, but this project uses the
unified `radix-ui` package (`components/ui/button.tsx` does
`import { Slot } from "radix-ui"`) — rewrite the generated imports and drop the
split packages. shadcn's `sonner.tsx` imports `next-themes`, which this project
doesn't use (no dark-mode toggle yet) — hand-edit to a fixed theme with a TODO.

Zod 4 renamed `z.string().url()` → `z.url()` and `.uuid()` → `z.uuid()`.
`lib/services/schema.ts` mirrors the OpenAPI constraints; `serviceEditSchema =
serviceFormSchema.omit({ slug: true })` since slug is immutable. Export
`toCreateBody`/`toUpdateBody` (drop `""` → `undefined`), `parseTags`, `slugify`.

Verify: `pnpm lint && pnpm typecheck && pnpm test && pnpm build`. Adding deps
needs `docker compose up -d --build frontend` after — `/app/node_modules` is an
anonymous volume, a host `pnpm install` alone never reaches the container.

## Remaining steps

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
