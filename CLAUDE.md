# CLAUDE.md

Backstage-lite Internal Developer Portal (Go + Next.js + Postgres, later Azure).
Read `HANDOFF.md` before starting work — current state, verified quirks, next
steps. `docs/app-plan.md` is the roadmap and spec; consult it when planning
features.

## Hard rules

- **Never commit or push on your own initiative.** When work is verified,
  propose a conventional commit message and stop — the user makes the commit.
  Only run `git commit`/`git push` if the user's current message explicitly
  asks for it.
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
