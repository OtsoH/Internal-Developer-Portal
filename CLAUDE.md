# CLAUDE.md

Backstage-lite Internal Developer Portal (Go + Next.js + Postgres, later Azure).
Read `HANDOFF.md` before starting work — current state, verified quirks, next
steps. `docs/app-plan.md` is the roadmap and spec; consult it when planning
features.

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
- **No CORS.** The browser only talks to the frontend origin; Next rewrites
  proxy `/api/v1` to the backend. Never add CORS headers.
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

- Building or restyling UI → `frontend-design`; `ui-ux-pro-max` when
  designing new views (palettes, UX patterns)
- Reviewing UI/accessibility → `web-design-guidelines`
- Any chart or graph visualization (incl. the React Flow dependency graph) →
  `dataviz` before writing chart code
- User-facing prose (READMEs, docs, UI copy) → `humanizer` before finalizing
- Verifying UI changes → Playwright browser tools (screenshots land in
  `.playwright-mcp/`, gitignored)
- Pausing or ending a session → `handoff` skill to update `HANDOFF.md`
