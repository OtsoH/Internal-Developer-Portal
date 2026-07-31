import { describe, expect, it } from "vitest";

import { ApiError, NETWORK_ERROR, isApiError, unwrap } from "./errors";

function result<T>(
  status: number,
  body: { data?: T; error?: unknown } = {},
): Promise<{ data?: T; error?: unknown; response: Response }> {
  return Promise.resolve({
    ...body,
    response: new Response(null, { status }),
  });
}

describe("unwrap", () => {
  it("returns the data of a successful call", async () => {
    await expect(unwrap(result(201, { data: { id: "svc-1" } }))).resolves.toEqual({
      id: "svc-1",
    });
  });

  it("resolves a 204 that carries no body", async () => {
    await expect(unwrap(result(204))).resolves.toBeUndefined();
  });

  it("carries the backend's own code and message through", async () => {
    const failure = await unwrap(
      result(409, {
        error: { code: "slug_taken", message: "a service with that slug already exists" },
      }),
    ).catch((error: unknown) => error);

    expect(isApiError(failure)).toBe(true);
    expect(failure).toMatchObject({
      status: 409,
      code: "slug_taken",
      message: "a service with that slug already exists",
    });
  });

  it("synthesizes a code from the status when the body is not an Error", async () => {
    // A gateway or proxy failure returns HTML, not the API's error shape.
    const failure = await unwrap(result(503, { error: "<html>oops</html>" })).catch(
      (error: unknown) => error,
    );

    expect(failure).toMatchObject({ status: 503, code: "database_unavailable" });
    expect((failure as ApiError).message).toMatch(/unavailable/i);
  });

  it("falls back to an unknown code for a status it does not recognize", async () => {
    const failure = await unwrap(result(418, { error: {} })).catch(
      (error: unknown) => error,
    );

    expect(failure).toMatchObject({ status: 418, code: "unknown" });
    expect((failure as ApiError).message).toContain("418");
  });

  it("reports a rejected fetch as a network error with no status", async () => {
    const failure = await unwrap(Promise.reject(new TypeError("Failed to fetch"))).catch(
      (error: unknown) => error,
    );

    expect(failure).toMatchObject({ status: 0, code: NETWORK_ERROR });
  });

  it("treats a non-ok response with no error field as a failure", async () => {
    // openapi-fetch leaves `error` undefined when the body will not parse.
    const failure = await unwrap(result(500)).catch((error: unknown) => error);

    expect(failure).toMatchObject({ status: 500, code: "internal" });
  });
});

describe("isApiError", () => {
  it("rejects other error types", () => {
    expect(isApiError(new Error("boom"))).toBe(false);
    expect(isApiError({ status: 403, code: "forbidden" })).toBe(false);
    expect(isApiError(new ApiError(403, "forbidden", "no"))).toBe(true);
  });
});
