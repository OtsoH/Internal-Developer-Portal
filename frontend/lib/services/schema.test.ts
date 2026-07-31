import { describe, expect, it } from "vitest";

import {
  formatTags,
  lifecycleLabels,
  lifecycles,
  maxTags,
  parseTags,
  serviceEditSchema,
  serviceFormSchema,
  slugify,
  toCreateBody,
  toUpdateBody,
  type ServiceEditValues,
  type ServiceFormValues,
} from "./schema";

const valid: ServiceFormValues = {
  name: "Payments API",
  slug: "payments-api",
  description: "Charges cards.",
  lifecycle: "production",
  repoUrl: "https://github.com/acme/payments-api",
  runbookUrl: "https://runbooks.acme.dev/payments-api",
  teamId: "3f6b2c1e-8a4d-4f2b-9c7e-1d5a0b3e7f92",
  tags: "go, PCI",
};

// Spelled out rather than destructured off `valid`, so that adding a field to
// the create form without deciding whether it is editable breaks this file.
const validEdit: ServiceEditValues = {
  name: valid.name,
  description: valid.description,
  lifecycle: valid.lifecycle,
  repoUrl: valid.repoUrl,
  runbookUrl: valid.runbookUrl,
  teamId: valid.teamId,
  tags: valid.tags,
};

function fieldErrors(values: Partial<ServiceFormValues>) {
  const result = serviceFormSchema.safeParse({ ...valid, ...values });
  if (result.success) return {};
  return result.error.flatten().fieldErrors;
}

describe("serviceFormSchema", () => {
  it("accepts a fully populated form", () => {
    expect(serviceFormSchema.safeParse(valid).success).toBe(true);
  });

  it("accepts blank optional fields", () => {
    const sparse = { ...valid, description: "", repoUrl: "", runbookUrl: "", tags: "" };
    expect(serviceFormSchema.safeParse(sparse).success).toBe(true);
  });

  it("requires a name and caps it at the spec's 120 characters", () => {
    expect(fieldErrors({ name: "   " }).name).toContain("Name is required");
    expect(fieldErrors({ name: "x".repeat(121) }).name).toBeDefined();
    expect(fieldErrors({ name: "x".repeat(120) }).name).toBeUndefined();
  });

  it.each([
    ["uppercase", "Payments-API"],
    ["spaces", "payments api"],
    ["underscores", "payments_api"],
    ["a leading hyphen", "-payments"],
    ["a trailing hyphen", "payments-"],
    ["a doubled hyphen", "payments--api"],
    ["nothing at all", ""],
  ])("rejects a slug with %s", (_, slug) => {
    expect(fieldErrors({ slug }).slug).toBeDefined();
  });

  it("caps the description at 2000 characters", () => {
    expect(fieldErrors({ description: "x".repeat(2000) }).description).toBeUndefined();
    expect(fieldErrors({ description: "x".repeat(2001) }).description).toBeDefined();
  });

  it("rejects a URL that is not a URL, but allows an empty one", () => {
    expect(fieldErrors({ repoUrl: "github.com/acme" }).repoUrl).toBeDefined();
    expect(fieldErrors({ runbookUrl: "" }).runbookUrl).toBeUndefined();
  });

  it("rejects an unchosen team", () => {
    expect(fieldErrors({ teamId: "" }).teamId).toContain("Choose a team");
  });

  it("rejects an unknown lifecycle", () => {
    const result = serviceFormSchema.safeParse({ ...valid, lifecycle: "retired" });
    expect(result.success).toBe(false);
  });

  it("counts tags after parsing, so duplicates do not push it over the cap", () => {
    const distinct = Array.from({ length: maxTags + 1 }, (_, i) => `t${i}`).join(",");
    expect(fieldErrors({ tags: distinct }).tags).toBeDefined();

    const duplicated = Array.from({ length: maxTags + 5 }, () => "go").join(",");
    expect(fieldErrors({ tags: duplicated }).tags).toBeUndefined();
  });
});

describe("serviceEditSchema", () => {
  it("has no slug field, because slugs are permanent", () => {
    expect(serviceEditSchema.safeParse(validEdit).success).toBe(true);

    const result = serviceEditSchema.safeParse({ ...validEdit, slug: valid.slug });
    expect(result.success && "slug" in result.data).toBe(false);
  });

  it("still enforces the shared constraints", () => {
    expect(serviceEditSchema.safeParse({ ...validEdit, name: "" }).success).toBe(
      false,
    );
  });
});

describe("lifecycles", () => {
  it("labels every option it offers", () => {
    for (const lifecycle of lifecycles) {
      expect(lifecycleLabels[lifecycle]).toBeTruthy();
    }
    expect(Object.keys(lifecycleLabels)).toHaveLength(lifecycles.length);
  });
});

describe("parseTags", () => {
  it("folds case and whitespace the way the backend does", () => {
    expect(parseTags(" Go , PCI ")).toEqual(["go", "pci"]);
  });

  it("drops blanks and collapses duplicates, keeping first-seen order", () => {
    expect(parseTags("go,,  ,GO, pci,go")).toEqual(["go", "pci"]);
  });

  it("returns nothing for an empty input", () => {
    expect(parseTags("")).toEqual([]);
    expect(parseTags("   ,  ")).toEqual([]);
  });

  it("round-trips through formatTags", () => {
    expect(parseTags(formatTags(["go", "pci"]))).toEqual(["go", "pci"]);
  });
});

describe("slugify", () => {
  it.each([
    ["Payments API", "payments-api"],
    ["  Payments   API  ", "payments-api"],
    ["Payments/API v2", "payments-api-v2"],
    ["Café Service", "cafe-service"],
    ["--Payments--", "payments"],
    ["!!!", ""],
    ["", ""],
  ])("turns %j into %j", (input, expected) => {
    expect(slugify(input)).toBe(expected);
  });

  it("produces a slug the schema accepts", () => {
    expect(fieldErrors({ slug: slugify("Payments API v2!") }).slug).toBeUndefined();
  });

  it("truncates to 120 characters without leaving a trailing hyphen", () => {
    const slug = slugify(`${"x".repeat(119)} y`);
    expect(slug.length).toBeLessThanOrEqual(120);
    expect(slug.endsWith("-")).toBe(false);
  });
});

describe("toCreateBody", () => {
  it("sends trimmed values and the parsed tag list", () => {
    expect(toCreateBody({ ...valid, name: "  Payments API  " })).toMatchObject({
      name: "Payments API",
      slug: "payments-api",
      teamId: valid.teamId,
      lifecycle: "production",
      tags: ["go", "pci"],
    });
  });

  it("omits blank optional fields rather than sending them empty", () => {
    const body = toCreateBody({
      ...valid,
      description: "",
      repoUrl: "",
      runbookUrl: "   ",
    });

    expect(body.description).toBeUndefined();
    expect(body.repoUrl).toBeUndefined();
    expect(body.runbookUrl).toBeUndefined();
  });
});

describe("toUpdateBody", () => {
  it("never sends a slug", () => {
    expect(toUpdateBody(validEdit)).not.toHaveProperty("slug");
  });

  it("carries the editable fields through", () => {
    expect(toUpdateBody({ ...validEdit, lifecycle: "deprecated" })).toMatchObject({
      name: "Payments API",
      teamId: valid.teamId,
      lifecycle: "deprecated",
      tags: ["go", "pci"],
    });
  });
});
