import createClient from "openapi-fetch";
import type { paths } from "./schema";

// Server components call the backend directly via BACKEND_URL; the browser
// goes through the Next.js rewrite proxy at /api/v1 (no CORS needed).
const baseUrl =
  typeof window === "undefined"
    ? `${process.env.BACKEND_URL ?? "http://localhost:8080"}/api/v1`
    : "/api/v1";

export const api = createClient<paths>({ baseUrl });
