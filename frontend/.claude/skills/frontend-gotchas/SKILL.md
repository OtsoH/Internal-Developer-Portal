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

**`shadcn add` generates imports from split `@radix-ui/react-*` packages, but this
project uses the unified `radix-ui` package.** Rewrite the generated imports after
every `add`. Its `sonner.tsx` also pulls `next-themes`; dark mode is unreachable
today, so hand-edit that file rather than taking the dependency.

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

## CI

`pnpm/action-setup` reads `packageManager` from the repo root by default, which
fails in this monorepo. `ci.yml` passes `package_json_file: frontend/package.json`.
