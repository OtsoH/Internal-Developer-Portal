---
name: frontend-gotchas
description: Traps this Next.js frontend has already hit — read before editing frontend/ code, adding an npm dependency, running shadcn add, touching auth.ts or the BFF proxy route, writing a component test, or debugging a container that will not hot-reload. Each entry cost real time to find; none are guessable from the source.
---

# Frontend gotchas

Every entry here was paid for once. Re-read the relevant section before touching
that area rather than rediscovering it.

## Containers and dependencies

**Adding an npm dependency needs `--renew-anon-volumes`.** `/app/node_modules` is
an anonymous volume, so `docker compose up -d --build frontend` alone leaves it
stale and every route 500s with `Can't resolve '<new-package>'`:

```sh
docker compose up -d --build --renew-anon-volumes frontend
```

**The dev container runs webpack, not Turbopack.** Only webpack's watcher honors
`WATCHPACK_POLLING`, which Windows bind mounts require; Turbopack never sees the
change. Compose overrides the command to `pnpm exec next dev`. Native `pnpm dev`
still uses Turbopack and is fine.

## shadcn/ui

Init non-interactively (the `--base-color` flag no longer exists):

```sh
pnpm dlx shadcn@latest init --yes --base radix --preset nova --css-variables --no-monorepo
```

**`shadcn add` can generate imports from split `@radix-ui/react-*` packages, but
this project uses the unified `radix-ui` package.** Check the generated imports
after every `add`. The `radix-nova` style got this right for `label` and
`select` (both emit `from "radix-ui"`), so this is a check, not an automatic
rewrite — but confirm before assuming, and drop any split package that sneaks
into `package.json`, or you end up with two copies of a primitive.

**`shadcn add sonner` pulls `next-themes`.** Dark mode is unreachable today, so
`components/ui/sonner.tsx` is hand-edited to a fixed `theme="light"` with a TODO
and the dependency removed. Re-adding the component re-adds the dependency.

**`shadcn add form` is a silent no-op under `radix-nova`.** It prints "Checking
registry", exits 0 and writes nothing; `shadcn view form` shows the item with no
`files` at all. Nova replaced it with `field`, which has no react-hook-form
awareness. **`components/ui/form.tsx` is therefore this project's own code, not
registry output** — a bridge exposing the familiar `Form*` API over the `Field*`
primitives. Do not try to `shadcn add form` to "restore" it, and do not
regenerate it; edit it like any other source file. `components/ui/form.test.tsx`
pins the aria wiring it produces.

**Geist font variables must stay on `<html>`, not `<body>`.** The Nova preset
applies `font-sans` on `<html>`, so variables defined on body leave everything
rendering serif. They live on `<html>` in `app/layout.tsx`.

## Auth.js v5

**`next-auth/jwt` cannot be augmented from this repo.** It is a bare
`export * from "@auth/core/jwt"`, so `declare module "next-auth/jwt"` creates a
fresh interface instead of merging, and `@auth/core` is not resolvable from
`frontend/` under pnpm (it is a transitive dep). `types/next-auth.d.ts` augments
only `next-auth`'s `Session`; `auth.ts`'s `session()` callback narrows
`token.accessToken` with a runtime `typeof` check instead. **Do not add
`@auth/core` as a direct dependency to "fix" this** — the runtime narrow is the
intended shape.

**`AUTH_MODE`'s default must stay environment-aware** (`isProd ? "entra" : "dev"`),
not a flat `"dev"`. CI runs `next build` with `AUTH_MODE` deliberately unset; a
flat dev default fires the production guard on every CI build
(`Failed to collect page data for /_not-found`). The environment-aware default
also means forgetting the variable in a real deployment can never silently
downgrade it to the seeded dev personas. Unknown values throw, mirroring the
backend's `ConfigFromEnv`.

## The BFF proxy

`app/api/v1/[...path]/route.ts` is the only path from browser to backend. It
strips `x-dev-user`, `authorization`, `cookie`, `host` and hop-by-hop headers
from the inbound request before attaching the real credential. **A
`next.config.ts` rewrite cannot do this**, which is why the rewrite was retired.
Never reintroduce one. Never add CORS headers.

`lib/auth/forward.ts` is the single place that knows dev mode sends `X-Dev-User`
and Entra mode sends `Authorization: Bearer`. `getServerApi(session)` talks
straight to `BACKEND_URL` for server components; `lib/api/client.ts` is
browser-only and points at `/api/v1`.

## Testing

**Async server components are tested by calling them as functions:**

```ts
render(await ServicesPage())   // with vi.mock("@/lib/api/server")
```

**Route-handler tests need `// @vitest-environment node`** at the top of the
file; `next/server` misbehaves under the globally configured jsdom.

**Testing Library's auto-cleanup does not run here.** It only registers itself
when Vitest sets `globals: true`, and `vitest.config.ts` does not. Without an
explicit `afterEach(cleanup)` a second `render` in one file leaves the first
mounted and every query fails with "found multiple elements". It is wired in
`vitest.setup.ts` — do not remove it.

## CI

`pnpm/action-setup` reads `packageManager` from the repo root by default, which
fails in this monorepo. `ci.yml` passes `package_json_file: frontend/package.json`.
