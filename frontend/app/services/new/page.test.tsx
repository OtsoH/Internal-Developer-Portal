import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import NewServicePage from "./page";
import type { components } from "@/lib/api/schema";

const { currentUserMock, serviceFormMock } = vi.hoisted(() => ({
  currentUserMock: vi.fn(),
  serviceFormMock: vi.fn(),
}));

vi.mock("@/lib/api/server", () => ({ getCurrentUser: currentUserMock }));

vi.mock("@/lib/auth/session", () => ({
  requireSession: vi.fn().mockResolvedValue({
    user: { email: "dev.editor@example.com", name: "Dev Editor" },
  }),
}));

// The form has its own test; here it only needs to report the props the page
// gave it, so the gating decision is what is under test.
vi.mock("@/components/services/service-form", () => ({
  ServiceForm: (props: unknown) => {
    serviceFormMock(props);
    return <div data-testid="service-form" />;
  },
}));

type Role = components["schemas"]["Role"];
type TeamRole = components["schemas"]["TeamRole"];

function team(name: string, role: Role): TeamRole {
  return {
    teamId: `${name.toLowerCase()}-id`,
    teamSlug: name.toLowerCase(),
    teamName: name,
    role,
  };
}

function user(teamRoles: TeamRole[]) {
  return {
    id: "aaaaaaaa-0000-0000-0000-000000000002",
    email: "dev.editor@example.com",
    name: "Dev Editor",
    teamRoles,
  };
}

async function renderPage() {
  render(await NewServicePage());
}

describe("NewServicePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("offers the form to someone who can write to a team", async () => {
    currentUserMock.mockResolvedValue(user([team("Payments", "EDITOR")]));

    await renderPage();

    expect(screen.getByTestId("service-form")).toBeInTheDocument();
  });

  it("offers only the teams the user may write to", async () => {
    currentUserMock.mockResolvedValue(
      user([
        team("Platform", "VIEWER"),
        team("Payments", "EDITOR"),
        team("Core", "ADMIN"),
      ]),
    );

    await renderPage();

    expect(serviceFormMock).toHaveBeenCalledWith(
      expect.objectContaining({
        mode: "create",
        teams: [team("Payments", "EDITOR"), team("Core", "ADMIN")],
      }),
    );
  });

  it("explains the missing role instead of rendering a form nobody can submit", async () => {
    currentUserMock.mockResolvedValue(user([team("Platform", "VIEWER")]));

    await renderPage();

    expect(screen.queryByTestId("service-form")).toBeNull();
    expect(screen.getByText(/needs the editor role on a team/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /back to services/ })).toHaveAttribute(
      "href",
      "/services",
    );
  });

  it("says the roles are unavailable rather than blaming the user", async () => {
    currentUserMock.mockResolvedValue(null);

    await renderPage();

    expect(screen.queryByTestId("service-form")).toBeNull();
    expect(screen.getByText(/Could not load your team roles/)).toBeInTheDocument();
    expect(screen.queryByText(/needs the editor role/)).toBeNull();
  });
});
