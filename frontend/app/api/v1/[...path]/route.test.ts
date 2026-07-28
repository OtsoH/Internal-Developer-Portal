// @vitest-environment node
import { NextRequest } from "next/server";
import { describe, expect, it, vi, beforeEach } from "vitest";

import { GET, POST } from "./route";

const { authMock } = vi.hoisted(() => ({ authMock: vi.fn() }));

vi.mock("@/auth", () => ({
  auth: authMock,
  isDevAuth: true,
}));

function ctx(path: string[]) {
  return { params: Promise.resolve({ path }) };
}

describe("proxy route", () => {
  beforeEach(() => {
    authMock.mockReset();
    vi.restoreAllMocks();
  });

  it("returns 401 with an Error body when there is no session", async () => {
    authMock.mockResolvedValue(null);

    const response = await GET(
      new NextRequest("http://localhost:3000/api/v1/services"),
      ctx(["services"]),
    );

    expect(response.status).toBe(401);
    await expect(response.json()).resolves.toEqual({
      code: "unauthenticated",
      message: "Sign in required.",
    });
  });

  it("forwards to the backend with the dev credential and strips the inbound auth headers", async () => {
    authMock.mockResolvedValue({ user: { email: "dev.editor@example.com" } });
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(
        new Response(JSON.stringify({ items: [] }), { status: 200 }),
      );

    const response = await GET(
      new NextRequest("http://localhost:3000/api/v1/services?lifecycle=beta", {
        headers: {
          "x-dev-user": "dev.admin@example.com",
          authorization: "Bearer forged",
          cookie: "authjs.session-token=forged",
        },
      }),
      ctx(["services"]),
    );

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe(
      "http://localhost:8080/api/v1/services?lifecycle=beta",
    );
    const sentHeaders = init?.headers as Headers;
    expect(sentHeaders.get("x-dev-user")).toBe("dev.editor@example.com");
    expect(sentHeaders.get("authorization")).toBeNull();
    expect(sentHeaders.get("cookie")).toBeNull();
  });

  it("streams a request body through with duplex: half", async () => {
    authMock.mockResolvedValue({ user: { email: "dev.editor@example.com" } });
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(new Response(null, { status: 201 }));

    await POST(
      new NextRequest("http://localhost:3000/api/v1/services", {
        method: "POST",
        body: JSON.stringify({ name: "Gateway" }),
        headers: { "content-type": "application/json" },
      }),
      ctx(["services"]),
    );

    const [, init] = fetchMock.mock.calls[0];
    expect(init?.body).not.toBeNull();
    expect((init as RequestInit & { duplex?: string }).duplex).toBe("half");
  });
});
