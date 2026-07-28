# How this project uses AI

Most of the code here was written by [Claude Code](https://claude.com/claude-code). I set the direction, made the design calls, reviewed the diffs, and ran (nearly) every commit myself. This file explains how that actually works day to day, because "built with AI" covers everything from autocomplete to unsupervised generation, and those are very different claims.

The short version: the AI does the typing and a lot of the investigation. I decide what gets built, approve the approach before implementation starts, and own the commit history.

## The working agreement

`CLAUDE.md` sits at the repo root and loads at the start of every session. It holds the rules that matter enough to survive a context reset:

- **The AI never commits or pushes.** It proposes a conventional commit message(s) and stops. I run `git commit`. This keeps the history mine, and it forces a review checkpoint that is easy to skip when the agent is on a roll.
- **OpenAPI-first.** `backend/api/openapi.yaml` is the contract. Any API change edits the spec first, then regenerates both sides. CI fails on drift, so this is enforced rather than merely encouraged.
- **No CORS.** The browser only talks to the frontend origin.
- **Postgres listens on host port 5433**, not 5432, because a native install already owns 5432. Without this line written down, every fresh session tries to "fix" it back.
- **Verify before claiming done.** Tests, plus curl, plus a browser check for anything with a UI.

That last rule does more work than the others combined. An agent will tell you something is finished the moment the code compiles. Requiring evidence changes what gets reported.

## Managing context

Agent sessions have a memory horizon, and this project has outlived many of them. Four files carry state across that boundary.

`docs/app-plan.md` is the spec and the four-week roadmap. It was written before any code and has barely changed, which is a decent sign the planning was worth it.

`HANDOFF.md` is the session boundary document, written by a dedicated skill when work pauses. It records what is done, what is in progress, and, most usefully, a **What Didn't Work** section that never gets deleted. That section is the highest-value part of the repo for a fresh agent. It is where "Turbopack ignores `WATCHPACK_POLLING` on Windows bind mounts" lives, so nobody burns another hour rediscovering it.

Plan files live outside the repo, in `~/.claude/plans/`. Before a multi-step chunk of work, the agent researches the codebase, asks me the questions it genuinely cannot answer from the code, and writes a plan I approve or reject before anything is implemented. The week 2 plan runs to fourteen steps with a verification command for each. When reality diverges from the plan, the plan gets a "deviations" section rather than a quiet edit, so the reasoning stays auditable.

There is also a persistent memory directory for facts that span projects, like environment quirks and my preferences about how commits get made.

## Skills

Skills are instruction sets the agent loads when a task matches. `CLAUDE.md` maps them to trigger moments so they get used without me asking:

| Skill | When |
|---|---|
| `handoff` | Pausing or ending a session |
| `humanizer` | Any prose a human will read, including this file |
| `frontend-design`, `ui-ux-pro-max` | Building or restyling UI |
| `web-design-guidelines` | Reviewing UI and accessibility |
| `dataviz` | Before writing any chart code, including the week 4 dependency graph |
| `brainstorming`, `writing-plans` | Before implementation on anything non-trivial |
| `systematic-debugging` | Any bug, before proposing a fix |

The pattern worth stealing: skills that fire on a *moment* rather than on request. "Use the handoff skill when a session ends" is a rule the agent can follow on its own. "Use the handoff skill when I ask" means I have to remember, and I won't.

## Subagents

Three kinds get used here.

**Explore** agents search the codebase in parallel and report back. Planning week 2 started with two of them, one reading the Go backend and one the Next.js frontend, which is faster than doing it serially and keeps the raw file dumps out of the main conversation.

**Plan** agents take the exploration results and design an implementation. Their output gets reviewed against the actual source before I see it, because a plan agent working from a summary will confidently assert things that are not true. That happened: the plan predicted a sqlc type that turned out wrong, and checking caught it.

**test-runner** is a project-specific agent defined in `.claude/agents/test-runner.md`. It runs both suites and reports concisely. It knows that `go test -race` fails on this machine (no C toolchain) and that the integration tests need Docker Desktop started by hand.

## What "verified" means

Every step in the week 2 plan ends with commands whose output I can check:

```sh
cd backend && go test ./...            # includes a testcontainers Postgres
cd backend && go tool golangci-lint run
cd frontend && pnpm lint && pnpm typecheck && pnpm test
```

Plus curl against the running API and a Playwright screenshot for UI work. CI runs the same checks on every push, including a codegen drift check that fails if generated code does not match the spec.

Proving codegen is drift-free locally took a trick worth writing down. `git diff --exit-code` compares against the last commit, so with uncommitted work in the tree it reports your own changes as drift. Hashing the generated files, re-running the generators, and comparing hashes tests the thing you actually care about.

## Where the AI got it wrong

This is the honest part, and the reason I am not claiming the AI built this on its own.

**It reported success before checking.** The reason "verify before claiming done" is a hard rule is that the default behaviour is optimistic. Compiling is not working.

**It confidently predicted a wrong type.** The week 2 plan stated that a nullable `uuid` column would generate as `*uuid.UUID` under `emit_pointers_for_null_types`. It generated as `pgtype.UUID`, because the type override in `sqlc.yaml` only matches non-nullable columns. Plausible, specific, and wrong. Caught by reading the generated file instead of trusting the claim.

**It needed a human to make the trade-offs.** Four decisions shaped week 2: how the access token reaches the backend, which auth library, what an authenticated user with no team membership can see, and who can delete a service. The agent surfaced the options and had a recommendation for each, which is genuinely useful. But these are product decisions with security consequences, and picking them is not something I want to delegate.

**It found a real bug I would have missed.** Balance requires saying this too. While planning the OpenAPI changes, it dug into oapi-codegen's source and found that the generator tracks used response refs per operation, so sharing one error component across status codes makes the generated Go types depend on which statuses an operation happens to declare. Adding a 401 to one endpoint would have silently reshaped its 404 and broken the handler at build time. That is a subtle trap, and it got fixed structurally before it ever bit.

## Reproducing this setup

Everything is in the repo. `CLAUDE.md` is the entry point, `HANDOFF.md` is the current state, `docs/app-plan.md` is the destination, and `.claude/agents/` holds the project-specific agents. Clone it, open Claude Code, and the working agreement loads itself.
