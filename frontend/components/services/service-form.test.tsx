import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ServiceForm } from "./service-form";
import type { components } from "@/lib/api/schema";

const { postMock, putMock, pushMock, refreshMock, toastMock } = vi.hoisted(() => ({
  postMock: vi.fn(),
  putMock: vi.fn(),
  pushMock: vi.fn(),
  refreshMock: vi.fn(),
  toastMock: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/lib/api/client", () => ({
  api: { POST: postMock, PUT: putMock },
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: pushMock, refresh: refreshMock }),
}));

vi.mock("sonner", () => ({ toast: toastMock }));

type Service = components["schemas"]["Service"];
type TeamRole = components["schemas"]["TeamRole"];

const platform: TeamRole = {
  teamId: "00000000-0000-0000-0000-0000000000aa",
  teamSlug: "platform",
  teamName: "Platform",
  role: "EDITOR",
};

const payments: TeamRole = {
  teamId: "00000000-0000-0000-0000-0000000000bb",
  teamSlug: "payments",
  teamName: "Payments",
  role: "ADMIN",
};

const existing: Service = {
  id: "00000000-0000-0000-0000-000000000001",
  name: "Billing",
  slug: "billing",
  description: "Charges cards.",
  lifecycle: "beta",
  repoUrl: "https://github.com/acme/billing",
  team: { id: platform.teamId, name: "Platform", slug: "platform" },
  tags: ["go", "pci"],
  createdAt: "2026-07-01T00:00:00Z",
  updatedAt: "2026-07-01T00:00:00Z",
};

/** What openapi-fetch resolves with on success. */
function ok<T>(status: number, data: T) {
  return { data, error: undefined, response: new Response(null, { status }) };
}

/** What openapi-fetch resolves with for an HTTP error. */
function failed(status: number, code: string, message: string) {
  return {
    data: undefined,
    error: { code, message },
    response: new Response(null, { status }),
  };
}

function renderForm(props: Partial<React.ComponentProps<typeof ServiceForm>> = {}) {
  const client = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={client}>
      <ServiceForm mode="create" teams={[platform]} {...props} />
    </QueryClientProvider>,
  );
}

describe("ServiceForm in create mode", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fills the slug from the name until the slug is edited", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.type(screen.getByLabelText("Name"), "Payments API");
    expect(screen.getByLabelText("Slug")).toHaveValue("payments-api");

    await user.type(screen.getByLabelText("Slug"), "-v2");
    await user.type(screen.getByLabelText("Name"), " Gateway");

    expect(screen.getByLabelText("Slug")).toHaveValue("payments-api-v2");
  });

  it("preselects the only team the user can write to", () => {
    renderForm();

    expect(screen.getByRole("combobox", { name: "Owning team" })).toHaveTextContent(
      "Platform",
    );
  });

  it("asks which team when there is more than one", () => {
    renderForm({ teams: [platform, payments] });

    expect(screen.getByRole("combobox", { name: "Owning team" })).toHaveTextContent(
      "Choose a team",
    );
  });

  it("posts the mapped body, then refreshes and returns to the list", async () => {
    const user = userEvent.setup();
    postMock.mockResolvedValue(ok(201, { ...existing, name: "Payments API" }));
    renderForm();

    await user.type(screen.getByLabelText("Name"), "Payments API");
    await user.type(screen.getByLabelText("Tags"), "Go, , go, PCI");
    await user.click(screen.getByRole("button", { name: "Register service" }));

    await waitFor(() => expect(postMock).toHaveBeenCalledTimes(1));
    expect(postMock).toHaveBeenCalledWith("/services", {
      body: {
        name: "Payments API",
        slug: "payments-api",
        teamId: platform.teamId,
        lifecycle: "production",
        description: undefined,
        repoUrl: undefined,
        runbookUrl: undefined,
        tags: ["go", "pci"],
      },
    });

    await waitFor(() => expect(refreshMock).toHaveBeenCalled());
    expect(pushMock).toHaveBeenCalledWith("/services");
    expect(toastMock.success).toHaveBeenCalledWith("Registered Payments API");
  });

  it("refuses to submit an invalid form", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.click(screen.getByRole("button", { name: "Register service" }));

    expect(await screen.findByText("Name is required")).toBeInTheDocument();
    expect(postMock).not.toHaveBeenCalled();
  });

  it("shows a taken slug on the field itself, not as a toast", async () => {
    const user = userEvent.setup();
    postMock.mockResolvedValue(
      failed(409, "slug_taken", "a service with that slug already exists"),
    );
    renderForm();

    await user.type(screen.getByLabelText("Name"), "Payments API");
    await user.click(screen.getByRole("button", { name: "Register service" }));

    expect(
      await screen.findByText("That slug is taken. Pick another."),
    ).toBeInTheDocument();
    expect(toastMock.error).not.toHaveBeenCalled();
  });

  it("toasts a refusal from the backend", async () => {
    const user = userEvent.setup();
    postMock.mockResolvedValue(
      failed(403, "forbidden", "EDITOR role required on the owning team"),
    );
    renderForm();

    await user.type(screen.getByLabelText("Name"), "Payments API");
    await user.click(screen.getByRole("button", { name: "Register service" }));

    await waitFor(() =>
      expect(toastMock.error).toHaveBeenCalledWith(
        "EDITOR role required on the owning team",
      ),
    );
    expect(pushMock).not.toHaveBeenCalled();
  });

  it("sends an expired session back to sign in", async () => {
    const user = userEvent.setup();
    postMock.mockResolvedValue(
      failed(401, "unauthenticated", "not authenticated"),
    );
    renderForm();

    await user.type(screen.getByLabelText("Name"), "Payments API");
    await user.click(screen.getByRole("button", { name: "Register service" }));

    await waitFor(() => expect(pushMock).toHaveBeenCalledWith("/signin"));
    expect(toastMock.error).not.toHaveBeenCalled();
  });
});

describe("ServiceForm in edit mode", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows the service and locks the slug", () => {
    renderForm({ mode: "edit", service: existing });

    expect(screen.getByLabelText("Name")).toHaveValue("Billing");
    expect(screen.getByLabelText("Slug")).toBeDisabled();
    expect(screen.getByLabelText("Slug")).toHaveValue("billing");
    expect(screen.getByLabelText("Tags")).toHaveValue("go, pci");
    expect(screen.getByLabelText("Description")).toHaveValue("Charges cards.");
  });

  it("puts the editable fields and never the slug", async () => {
    const user = userEvent.setup();
    putMock.mockResolvedValue(ok(200, { ...existing, name: "Billing Core" }));
    renderForm({ mode: "edit", service: existing });

    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "Billing Core");
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(putMock).toHaveBeenCalledTimes(1));

    const [path, options] = putMock.mock.calls[0];
    expect(path).toBe("/services/{serviceId}");
    expect(options.params).toEqual({ path: { serviceId: existing.id } });
    expect(options.body).not.toHaveProperty("slug");
    expect(options.body).toMatchObject({
      name: "Billing Core",
      teamId: platform.teamId,
      lifecycle: "beta",
      tags: ["go", "pci"],
    });
  });
});
