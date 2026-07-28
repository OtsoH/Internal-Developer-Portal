import { cache } from "react";
import createClient from "openapi-fetch";
import type { Session } from "next-auth";

import { auth } from "@/auth";
import { authHeaders } from "@/lib/auth/forward";
import type { components, paths } from "./schema";

const backendUrl = process.env.BACKEND_URL ?? "http://localhost:8080";

/**
 * A client that talks to the Go API directly (not the `/api/v1` BFF route,
 * which only the browser needs) with the session's credential attached to
 * every request.
 */
export function getServerApi(session: Session) {
  const client = createClient<paths>({ baseUrl: `${backendUrl}/api/v1` });
  const headers = authHeaders(session);
  client.use({
    onRequest({ request }) {
      for (const [key, value] of Object.entries(headers)) {
        request.headers.set(key, value);
      }
      return request;
    },
  });
  return client;
}

/**
 * Wrapped in React's `cache()` so the header, the services page and any
 * nested server component share one `/me` round trip per request instead of
 * one each.
 */
export const getCurrentUser = cache(async (): Promise<
  components["schemas"]["CurrentUser"] | null
> => {
  const session = await auth();
  if (!session?.user) {
    return null;
  }
  const { data, error } = await getServerApi(session).GET("/me", {
    cache: "no-store",
  });
  return error ? null : data;
});
