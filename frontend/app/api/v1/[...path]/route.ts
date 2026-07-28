import { NextResponse, type NextRequest } from "next/server";

import { auth } from "@/auth";
import { authHeaders } from "@/lib/auth/forward";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const backendUrl = process.env.BACKEND_URL ?? "http://localhost:8080";

const HOP_BY_HOP_HEADERS = [
  "connection",
  "keep-alive",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailer",
  "transfer-encoding",
  "upgrade",
];

// A browser that could set these on the inbound request could self-assert an
// admin identity (X-Dev-User) or smuggle credentials meant for this origin
// only (Authorization, Cookie). This is the whole reason /api/v1 is a route
// handler and not a next.config.ts rewrite, which has no way to strip them.
const REQUEST_STRIP_HEADERS = new Set([
  ...HOP_BY_HOP_HEADERS,
  "host",
  "x-dev-user",
  "authorization",
  "cookie",
]);

// content-encoding/content-length describe the upstream fetch's own transfer,
// which fetch already decoded — forwarding them mismatches the body we send.
const RESPONSE_STRIP_HEADERS = new Set([
  ...HOP_BY_HOP_HEADERS,
  "content-encoding",
  "content-length",
]);

function filteredHeaders(source: Headers, strip: Set<string>): Headers {
  const headers = new Headers();
  for (const [key, value] of source) {
    if (!strip.has(key.toLowerCase())) {
      headers.set(key, value);
    }
  }
  return headers;
}

function unauthorized(): NextResponse {
  return NextResponse.json(
    { code: "unauthenticated", message: "Sign in required." },
    { status: 401 },
  );
}

async function proxy(
  req: NextRequest,
  ctx: { params: Promise<{ path: string[] }> },
): Promise<NextResponse | Response> {
  const session = await auth();
  if (!session?.user) {
    return unauthorized();
  }

  const { path } = await ctx.params;
  const upstream = new URL(`${backendUrl}/api/v1/${path.join("/")}`);
  upstream.search = req.nextUrl.search;

  const headers = filteredHeaders(req.headers, REQUEST_STRIP_HEADERS);
  for (const [key, value] of Object.entries(authHeaders(session))) {
    headers.set(key, value);
  }

  const body = req.body;
  const init: RequestInit & { duplex?: "half" } = {
    method: req.method,
    headers,
    body,
    redirect: "manual",
    cache: "no-store",
  };
  // duplex is required by undici for a streamed request body and is missing
  // from TypeScript's RequestInit.
  if (body) {
    init.duplex = "half";
  }

  const response = await fetch(upstream, init);

  return new NextResponse(response.body, {
    status: response.status,
    headers: filteredHeaders(response.headers, RESPONSE_STRIP_HEADERS),
  });
}

export {
  proxy as GET,
  proxy as POST,
  proxy as PUT,
  proxy as PATCH,
  proxy as DELETE,
  proxy as HEAD,
};
