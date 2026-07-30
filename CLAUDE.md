# CLAUDE.md

Backstage-lite Internal Developer Portal (Go + Next.js + Postgres, later Azure).
Read `HANDOFF.md` before starting work: it is a cursor, not a ledger, and holds
only where we are, the next step, and standing constraints. Hard-won traps live
in the `backend-gotchas` and `frontend-gotchas` skills, which load when you work
in those trees — read the matching one before editing there. `docs/app-plan.md`
is the roadmap and spec; consult it when planning features. `docs/AI.md` is
written for human readers, not for you; skip it unless asked.

## Hard rules

- **Never commit or push on your own initiative.** When work is verified,
  propose the commits and stop — the user makes them. Only run `git commit`/
  `git push` if the user's current message explicitly asks for it.
- **Propose a split, not one big commit.** Break verified work along feature
  seams — one concern per commit, usually a few related files. For each
  commit give the conventional message and the exact `git add` paths, in the
  order they must be applied. Every intermediate commit has to build, lint
  and test clean on its own, so order by dependency: a module before its
  callers, a test with the code it covers, config before whatever needs it at
  build time. Say whether you verified that ordering or only reasoned it
  through. If a split would need code regeneration between commits, say so
  and offer the collapsed alternative.
- **OpenAPI-first.** Change `backend/api/openapi.yaml` first, then regenerate
  both sides: `go generate ./...` (backend) and `pnpm generate:api`
  (frontend). CI fails on codegen drift.
- **No CORS.** The browser only talks to the frontend origin. A BFF route
  handler (`frontend/app/api/v1/[...path]/route.ts`) proxies `/api/v1` to the
  backend, reading the session server-side and attaching the credential — it
  replaced the old `next.config.ts` rewrite in week 2 step 6, because a rewrite
  cannot strip forged inbound headers. Never add CORS headers.
- **Postgres is on host port 5433 by design** (a native install owns 5432).
  Do not "fix" it back to 5432.
- **Verify before claiming done:** tests + curl + browser check for UI work.

## Commands

Full reference: `backend/README.md` and `frontend/README.md`. Most used:

- Backend (from `backend/`): `go test ./...` (integration test needs Docker
  Desktop, started manually), `go test -short ./...`, `go tool golangci-lint run`
- Frontend (from `frontend/`): `pnpm test`, `pnpm lint`, `pnpm typecheck`
- Whole test/lint suite: delegate to the `test-runner` agent
- Dev stack: `docker compose up -d --build`

## Skills — use at the matching moment without being asked

- **Editing anything under `backend/` → `backend-gotchas` first**; anything
  under `frontend/` → `frontend-gotchas`. These carry the traps that used to
  live in HANDOFF.md and cost real time to find.
- Building or restyling UI → `frontend-design`; `ui-ux-pro-max` when
  designing new views (palettes, UX patterns)
- Reviewing UI/accessibility → `web-design-guidelines`
- Any chart or graph visualization (incl. the React Flow dependency graph) →
  `dataviz` before writing chart code
- User-facing prose (READMEs, docs, UI copy) → `humanizer` before finalizing
- Verifying UI changes → Playwright browser tools (pass a `filename` under
  `.playwright-mcp/`; a bare name lands in the repo root, which is not ignored)
- Pausing or ending a session → `handoff` skill. It routes finished work to its
  permanent home and keeps `HANDOFF.md` under 120 lines; do not append to it.
