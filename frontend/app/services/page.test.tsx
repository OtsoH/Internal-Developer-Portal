import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, type Mock } from "vitest";

import ServicesPage from "./page";
import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";

vi.mock("@/lib/api/client", () => ({
  api: { GET: vi.fn() },
}));

const getMock = api.GET as unknown as Mock;

type Service = components["schemas"]["Service"];

function service(overrides: Partial<Service>): Service {
  return {
    id: "00000000-0000-0000-0000-000000000001",
    name: "API Gateway",
    slug: "api-gateway",
    lifecycle: "production",
    team: {
      id: "00000000-0000-0000-0000-0000000000aa",
      name: "Platform",
      slug: "platform",
    },
    tags: [],
    createdAt: "2026-07-01T00:00:00Z",
    updatedAt: "2026-07-01T00:00:00Z",
    ...overrides,
  };
}

// ServicesPage is an async server component: call it as a function and
// render the resolved element.
async function renderPage() {
  render(await ServicesPage());
}

describe("ServicesPage", () => {
  beforeEach(() => {
    getMock.mockReset();
  });

  it("renders the services table with status line, tags and repo link", async () => {
    getMock.mockResolvedValue({
      data: {
        items: [
          service({
            tags: ["edge", "go"],
            repoUrl: "https://github.com/acme/gateway",
          }),
          service({
            id: "00000000-0000-0000-0000-000000000002",
            name: "Billing",
            slug: "billing",
            lifecycle: "beta",
            team: {
              id: "00000000-0000-0000-0000-0000000000bb",
              name: "Payments",
              slug: "payments",
            },
          }),
        ],
      },
      error: undefined,
    });

    await renderPage();

    expect(screen.getByText("API Gateway")).toBeInTheDocument();
    expect(screen.getByText("api-gateway")).toBeInTheDocument();
    expect(screen.getByText("Billing")).toBeInTheDocument();
    expect(
      screen.getByText("2 services · 2 teams · 1 in production"),
    ).toBeInTheDocument();
    expect(screen.getByText("edge")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /repo/ })).toHaveAttribute(
      "href",
      "https://github.com/acme/gateway",
    );
  });

  it("renders the empty state when no services exist", async () => {
    getMock.mockResolvedValue({ data: { items: [] }, error: undefined });

    await renderPage();

    expect(
      screen.getByText(/No services registered yet/),
    ).toBeInTheDocument();
  });

  it("renders an error panel when the API returns an error", async () => {
    getMock.mockResolvedValue({
      data: undefined,
      error: { code: "internal", message: "boom" },
    });

    await renderPage();

    expect(
      screen.getByText("The API returned an error: boom"),
    ).toBeInTheDocument();
  });

  it("renders an unreachable-backend panel when the request throws", async () => {
    getMock.mockRejectedValue(new Error("ECONNREFUSED"));

    await renderPage();

    expect(
      screen.getByText(/The backend API is not reachable/),
    ).toBeInTheDocument();
  });
});
