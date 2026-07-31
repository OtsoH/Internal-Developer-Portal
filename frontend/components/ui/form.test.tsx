import { zodResolver } from "@hookform/resolvers/zod";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm } from "react-hook-form";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "./form";
import { Input } from "./input";

// form.tsx is this project's own bridge between react-hook-form and the Field
// primitives, so the accessibility wiring it generates is worth pinning: the
// label points at the control, the description and message are announced, and
// an invalid control says so.

const schema = z.object({ name: z.string().min(1, "Name is required") });

function TestForm() {
  const form = useForm({
    resolver: zodResolver(schema),
    defaultValues: { name: "" },
  });

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(() => {})}>
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Service name</FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              <FormDescription>Shown in the catalog.</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <button type="submit">Save</button>
      </form>
    </Form>
  );
}

describe("the react-hook-form bridge", () => {
  it("associates the label with the control", () => {
    render(<TestForm />);

    expect(screen.getByLabelText("Service name")).toBeInTheDocument();
  });

  it("points a valid control at its description alone", () => {
    render(<TestForm />);
    const input = screen.getByLabelText("Service name");

    expect(input).not.toHaveAttribute("aria-invalid");
    expect(input).toHaveAccessibleDescription("Shown in the catalog.");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("announces the validation message and marks the control invalid", async () => {
    const user = userEvent.setup();
    render(<TestForm />);

    await user.click(screen.getByRole("button", { name: "Save" }));

    const input = screen.getByLabelText("Service name");
    expect(await screen.findByRole("alert")).toHaveTextContent("Name is required");
    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(input).toHaveAccessibleDescription(
      "Shown in the catalog. Name is required",
    );
  });
});
