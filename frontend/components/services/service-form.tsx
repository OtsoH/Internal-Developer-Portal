"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

import { api } from "@/lib/api/client";
import { ApiError, unwrap } from "@/lib/api/errors";
import type { components } from "@/lib/api/schema";
import {
  emptyFormValues,
  formSchemaFor,
  lifecycleLabels,
  lifecycles,
  slugify,
  toCreateBody,
  toFormValues,
  toUpdateBody,
  type ServiceFormMode,
  type ServiceFormValues,
} from "@/lib/services/schema";
import { Button } from "@/components/ui/button";
import { FieldGroup } from "@/components/ui/field";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";

type Service = components["schemas"]["Service"];
type TeamRole = components["schemas"]["TeamRole"];

type ServiceFormProps = {
  mode: ServiceFormMode;
  /**
   * Only the teams the signed-in user may write to. Populating the picker from
   * this list is what keeps the UI from letting anyone compose a request the
   * backend would answer with 403.
   */
  teams: TeamRole[];
  /** The service being edited. Required in edit mode, ignored in create mode. */
  service?: Service;
};

const lifecycleDot: Record<(typeof lifecycles)[number], string> = {
  production: "bg-status-production",
  beta: "bg-status-beta",
  deprecated: "bg-status-deprecated",
};

export function ServiceForm({ mode, teams, service }: ServiceFormProps) {
  const router = useRouter();
  // Auto-filling the slug is a convenience, not a rule: the moment someone
  // types in the slug field it belongs to them and the name stops overwriting it.
  const [slugIsMine, setSlugIsMine] = useState(mode === "edit");

  const form = useForm<ServiceFormValues>({
    resolver: zodResolver(formSchemaFor(mode)),
    defaultValues: service
      ? toFormValues(service)
      : emptyFormValues(teams.length === 1 ? teams[0].teamId : ""),
  });

  const mutation = useMutation({
    mutationFn: (values: ServiceFormValues) =>
      mode === "create"
        ? unwrap(api.POST("/services", { body: toCreateBody(values) }))
        : unwrap(
            api.PUT("/services/{serviceId}", {
              params: { path: { serviceId: service!.id } },
              body: toUpdateBody(values),
            }),
          ),
    onSuccess: (saved) => {
      toast.success(
        mode === "create" ? `Registered ${saved.name}` : `Saved ${saved.name}`,
      );
      // The services table is a server component, so TanStack Query's cache has
      // nothing to do with it. Without this refresh the list still shows the
      // RSC payload from before the mutation.
      router.refresh();
      // The detail page arrives in step 13; until then a new service lands back
      // on the list, where the caller can see the row they just created.
      router.push(mode === "create" ? "/services" : `/services/${saved.id}`);
    },
    onError: (error) => {
      if (!(error instanceof ApiError)) {
        toast.error("Something went wrong. Try again.");
        return;
      }

      switch (error.code) {
        case "slug_taken":
          form.setError("slug", {
            message: "That slug is taken. Pick another.",
          });
          form.setFocus("slug");
          return;
        case "unauthenticated":
          router.push("/signin");
          return;
        default:
          toast.error(error.message);
      }
    },
  });

  const submitLabel = mode === "create" ? "Register service" : "Save changes";

  return (
    <Form {...form}>
      <form
        noValidate
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        className="rounded-lg border bg-card p-6"
      >
        <FieldGroup>
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Name</FormLabel>
                <FormControl>
                  <Input
                    autoFocus={mode === "create"}
                    {...field}
                    onChange={(event) => {
                      field.onChange(event);
                      if (mode === "create" && !slugIsMine) {
                        form.setValue("slug", slugify(event.target.value), {
                          shouldValidate: form.formState.isSubmitted,
                        });
                      }
                    }}
                  />
                </FormControl>
                <FormDescription>
                  How the service appears in the catalog.
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="slug"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Slug</FormLabel>
                <FormControl>
                  <Input
                    className="font-mono"
                    disabled={mode === "edit"}
                    {...field}
                    onChange={(event) => {
                      setSlugIsMine(true);
                      field.onChange(event);
                    }}
                  />
                </FormControl>
                <FormDescription>
                  {mode === "edit"
                    ? "Slugs are permanent. Other systems link to this one."
                    : "Lowercase letters and numbers, separated by hyphens. Fills in from the name until you edit it."}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="teamId"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Owning team</FormLabel>
                <Select
                  value={field.value || undefined}
                  onValueChange={field.onChange}
                  disabled={field.disabled}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder="Choose a team" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {teams.map((team) => (
                      <SelectItem key={team.teamId} value={team.teamId}>
                        {team.teamName}
                        <span className="font-mono text-xs text-muted-foreground">
                          {team.role.toLowerCase()}
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FormDescription>
                  Only teams you can write to are listed.
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="lifecycle"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Lifecycle</FormLabel>
                <Select value={field.value} onValueChange={field.onChange}>
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {lifecycles.map((lifecycle) => (
                      <SelectItem key={lifecycle} value={lifecycle}>
                        <span
                          aria-hidden
                          className={`size-2 rounded-full ${lifecycleDot[lifecycle]}`}
                        />
                        {lifecycleLabels[lifecycle]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="description"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Description</FormLabel>
                <FormControl>
                  <Textarea rows={4} {...field} />
                </FormControl>
                <FormDescription>
                  What it does, in a sentence or two. Optional.
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="repoUrl"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Repository</FormLabel>
                <FormControl>
                  <Input
                    type="url"
                    inputMode="url"
                    placeholder="https://github.com/acme/service"
                    className="font-mono"
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="runbookUrl"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Runbook</FormLabel>
                <FormControl>
                  <Input
                    type="url"
                    inputMode="url"
                    placeholder="https://runbooks.acme.dev/service"
                    className="font-mono"
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="tags"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Tags</FormLabel>
                <FormControl>
                  <Input className="font-mono" placeholder="go, pci" {...field} />
                </FormControl>
                <FormDescription>
                  Separate with commas. Case and spacing are normalized.
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </FieldGroup>

        <div className="mt-6 flex items-center gap-2 border-t pt-6">
          <Button type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? "Saving…" : submitLabel}
          </Button>
          <Button variant="ghost" asChild>
            <Link href={service ? `/services/${service.id}` : "/services"}>
              Cancel
            </Link>
          </Button>
        </div>
      </form>
    </Form>
  );
}
