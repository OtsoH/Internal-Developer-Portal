import { redirect } from "next/navigation";
import type { Session } from "next-auth";

import { auth } from "@/auth";

/**
 * Guards a server-rendered page. There is deliberately no `middleware.ts`:
 * Auth.js v5 middleware runs on the edge runtime, where the Credentials
 * provider does not work, and the BFF proxy already rejects unauthenticated
 * API calls. Protected pages call this instead.
 */
export async function requireSession(): Promise<Session> {
  const session = await auth();
  if (!session?.user) {
    redirect("/signin");
  }
  return session;
}
