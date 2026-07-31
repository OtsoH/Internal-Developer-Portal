import type { components } from "./schema";

type ErrorBody = components["schemas"]["Error"];

/**
 * A failed API call, normalized. `code` is what callers branch on — the backend
 * sends a machine-readable one with every error (see `handlers.go`), and the
 * gaps below cover the responses that never reach a handler: a 204 body, a
 * proxy error page, a request that never got a response at all.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export function isApiError(value: unknown): value is ApiError {
  return value instanceof ApiError;
}

/** The request never reached the backend, so there is no status to report. */
export const NETWORK_ERROR = "network_error";

// Used only when a response carries no JSON body of its own — a gateway
// timeout from the BFF proxy, say. A handler's own code always wins.
const codeByStatus: Record<number, string> = {
  400: "bad_request",
  401: "unauthenticated",
  403: "forbidden",
  404: "not_found",
  409: "conflict",
  500: "internal",
  503: "database_unavailable",
};

const messageByCode: Record<string, string> = {
  unauthenticated: "Your session has expired. Sign in again to continue.",
  forbidden: "You do not have permission to do that.",
  not_found: "That service no longer exists.",
  database_unavailable: "The catalog database is unavailable. Try again shortly.",
  [NETWORK_ERROR]: "Could not reach the server. Check your connection and retry.",
};

function isErrorBody(value: unknown): value is ErrorBody {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as ErrorBody).code === "string" &&
    typeof (value as ErrorBody).message === "string"
  );
}

function fallbackMessage(code: string, status: number): string {
  return messageByCode[code] ?? `The request failed (HTTP ${status}).`;
}

/**
 * Runs an openapi-fetch call and returns its data, throwing `ApiError` on any
 * failure so TanStack Query's `onError` sees one shape. openapi-fetch resolves
 * with `{ error }` for an HTTP error but *rejects* when the fetch itself fails,
 * so both paths have to be folded together here rather than at every call site.
 */
export async function unwrap<T>(
  call: Promise<{ data?: T; error?: unknown; response: Response }>,
): Promise<T> {
  let result: { data?: T; error?: unknown; response: Response };

  try {
    result = await call;
  } catch {
    throw new ApiError(0, NETWORK_ERROR, fallbackMessage(NETWORK_ERROR, 0));
  }

  const { data, error, response } = result;

  if (error === undefined && response.ok) {
    return data as T;
  }

  const status = response.status;

  if (isErrorBody(error)) {
    throw new ApiError(status, error.code, error.message);
  }

  const code = codeByStatus[status] ?? "unknown";
  throw new ApiError(status, code, fallbackMessage(code, status));
}
