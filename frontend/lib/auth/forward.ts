import type { Session } from "next-auth";

import { isDevAuth } from "@/auth";

/**
 * The single place that knows how a credential travels to the backend:
 * `X-Dev-User` in dev mode (see `backend/internal/auth/dev.go`),
 * `Authorization: Bearer` otherwise. Callers must already know `session.user`
 * exists — this never runs for an unauthenticated request.
 */
export function authHeaders(session: Session): Record<string, string> {
  if (isDevAuth) {
    return { "X-Dev-User": session.user?.email ?? "" };
  }
  return session.accessToken
    ? { Authorization: `Bearer ${session.accessToken}` }
    : {};
}
