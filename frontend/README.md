# Frontend

Next.js (App Router) frontend for the Internal Developer Portal. The [root README](../README.md) has the full picture.

## Development

```sh
pnpm install
pnpm dev
```

Runs at http://localhost:3000. Requests to `/api/v1/*` are proxied to the Go backend through Next rewrites (`BACKEND_URL`, default `http://localhost:8080`), so the browser only ever talks to the frontend origin and there is no CORS setup.

The fastest loop on Windows is to run only the infrastructure in Docker and the frontend natively:

```sh
docker compose up -d postgres backend
pnpm dev
```

## Regenerating the API client

Types are generated from the shared OpenAPI contract at `../backend/api/openapi.yaml`:

```sh
pnpm generate:api
```
