import createClient from "openapi-fetch";
import type { paths } from "./schema";

// Browser-only. The BFF route handler at app/api/v1/[...path]/route.ts
// attaches credentials server-side, so the browser never needs to (no CORS
// needed either — same origin throughout). Server components use
// lib/api/server.ts instead, which talks to the Go API directly.
export const api = createClient<paths>({ baseUrl: "/api/v1" });
