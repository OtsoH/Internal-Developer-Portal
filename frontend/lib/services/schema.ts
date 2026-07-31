import { z } from "zod";

import type { components } from "@/lib/api/schema";

type Lifecycle = components["schemas"]["Lifecycle"];
type Service = components["schemas"]["Service"];
type ServiceCreate = components["schemas"]["ServiceCreate"];
type ServiceUpdate = components["schemas"]["ServiceUpdate"];

/**
 * The lifecycle options a form offers, in the order they should be shown.
 *
 * These constants duplicate the OpenAPI enum, which is unavoidable: the browser
 * needs them to render a picker, and step 8 validates the request against the
 * spec itself, so a client that drifts cannot corrupt anything. Drift is still
 * caught at build time — `satisfies` rejects a value the spec has dropped, and
 * the `Record<Lifecycle, string>` on the labels rejects one it has gained.
 */
export const lifecycles = [
  "production",
  "beta",
  "deprecated",
] as const satisfies readonly Lifecycle[];

export const lifecycleLabels: Record<Lifecycle, string> = {
  production: "Production",
  beta: "Beta",
  deprecated: "Deprecated",
};

/** Mirrors `ServiceCreate.slug`'s pattern in `backend/api/openapi.yaml`. */
export const slugPattern = /^[a-z0-9]+(-[a-z0-9]+)*$/;

/** Mirrors the backend's `maxTags`, which silently truncates past this many. */
export const maxTags = 20;

// repo_url and runbook_url are nullable, not required. A form submits "" for a
// field nobody touched, so "" has to pass validation here and be dropped on the
// way out rather than stored as a blank link.
const optionalUrl = z.union([
  z.literal(""),
  z.url("Enter a full URL, including https://"),
]);

export const serviceFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Name is required")
    .max(120, "Name must be 120 characters or fewer"),
  slug: z
    .string()
    .trim()
    .min(1, "Slug is required")
    .max(120, "Slug must be 120 characters or fewer")
    .regex(slugPattern, "Lowercase letters and numbers, separated by hyphens"),
  description: z
    .string()
    .trim()
    .max(2000, "Description must be 2000 characters or fewer"),
  lifecycle: z.enum(lifecycles),
  repoUrl: optionalUrl,
  runbookUrl: optionalUrl,
  // `z.guid()`, not `z.uuid()`. Zod 4's `uuid()` enforces the RFC 9562 version
  // and variant nibbles, which the seeded team ids (11111111-1111-…) do not
  // carry — it would make every dev-mode team unselectable. OpenAPI's
  // `format: uuid` asserts a shape, not a version, and so does `guid()`.
  teamId: z.guid("Choose a team"),
  // One text input, comma-separated. The cap is checked against the parsed
  // list, because the backend would drop the overflow without saying so.
  tags: z
    .string()
    .refine(
      (value) => parseTags(value).length <= maxTags,
      `Up to ${maxTags} tags`,
    ),
});

/** Slugs are permanent, so the edit form has no slug field to validate. */
export const serviceEditSchema = serviceFormSchema.omit({ slug: true });

export type ServiceFormValues = z.infer<typeof serviceFormSchema>;
export type ServiceEditValues = z.infer<typeof serviceEditSchema>;

export type ServiceFormMode = "create" | "edit";

/**
 * The schema a form validates against, given its mode. Both produce the same
 * shape so one `useForm` covers both, but edit mode relaxes `slug` to a plain
 * string: it renders the field disabled, and refusing to submit over a value
 * the user is not allowed to change would deadlock the form.
 */
export function formSchemaFor(mode: ServiceFormMode) {
  return mode === "create"
    ? serviceFormSchema
    : serviceFormSchema.extend({ slug: z.string() });
}

/**
 * Splits the tags input into the normalized list the backend stores: lowercase,
 * trimmed, blanks dropped, duplicates collapsed, first-wins order preserved.
 * Keeping the same rules here means what the user sees after saving is what the
 * form showed them.
 */
export function parseTags(input: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];

  for (const part of input.split(",")) {
    const tag = part.trim().toLowerCase();
    if (tag === "" || seen.has(tag)) continue;
    seen.add(tag);
    out.push(tag);
  }

  return out;
}

/** Renders a stored tag list back into the comma-separated input's value. */
export function formatTags(tags: readonly string[]): string {
  return tags.join(", ");
}

/**
 * Derives a slug candidate from a service name. Only a suggestion — the user
 * can overwrite it, and the backend is the authority on whether it is free.
 */
export function slugify(name: string): string {
  return name
    .normalize("NFKD")
    .replace(/[̀-ͯ]/g, "") // the accents NFKD just split off
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 120)
    .replace(/-+$/, "");
}

/**
 * The form state a blank create form starts from. `teamId` is pre-chosen when
 * the caller has exactly one team to offer, because a picker with one option is
 * a question with one answer.
 */
export function emptyFormValues(teamId = ""): ServiceFormValues {
  return {
    name: "",
    slug: "",
    description: "",
    lifecycle: "production",
    repoUrl: "",
    runbookUrl: "",
    teamId,
    tags: "",
  };
}

/**
 * The form state that shows an existing service. Absent optional fields become
 * "" rather than `undefined`, so every input stays controlled from first render
 * — React warns loudly the first time one flips from uncontrolled to controlled.
 */
export function toFormValues(service: Service): ServiceFormValues {
  return {
    name: service.name,
    slug: service.slug,
    description: service.description ?? "",
    lifecycle: service.lifecycle,
    repoUrl: service.repoUrl ?? "",
    runbookUrl: service.runbookUrl ?? "",
    teamId: service.team.id,
    tags: formatTags(service.tags),
  };
}

// An untouched optional field is "", which must be omitted from the request
// body rather than sent blank — see `nilIfBlank` in the backend's mutations.go.
function blankToUndefined(value: string): string | undefined {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

export function toCreateBody(values: ServiceFormValues): ServiceCreate {
  return {
    name: values.name.trim(),
    slug: values.slug.trim(),
    teamId: values.teamId,
    lifecycle: values.lifecycle,
    description: blankToUndefined(values.description),
    repoUrl: blankToUndefined(values.repoUrl),
    runbookUrl: blankToUndefined(values.runbookUrl),
    tags: parseTags(values.tags),
  };
}

export function toUpdateBody(values: ServiceEditValues): ServiceUpdate {
  return {
    name: values.name.trim(),
    teamId: values.teamId,
    lifecycle: values.lifecycle,
    description: blankToUndefined(values.description),
    repoUrl: blankToUndefined(values.repoUrl),
    runbookUrl: blankToUndefined(values.runbookUrl),
    tags: parseTags(values.tags),
  };
}
