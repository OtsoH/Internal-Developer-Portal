import { redirect } from "next/navigation";

import { auth, devUsers, isDevAuth, signIn } from "@/auth";
import { Button } from "@/components/ui/button";

export const metadata = { title: "Sign in" };

export default async function SignInPage() {
  const session = await auth();
  if (session?.user) {
    redirect("/services");
  }

  return (
    <div className="mx-auto max-w-md py-12">
      <h1 className="text-2xl font-semibold tracking-tight">Sign in</h1>
      <p className="mt-2 font-mono text-xs text-muted-foreground">
        {isDevAuth
          ? "auth mode: dev · pick a seeded persona"
          : "auth mode: entra"}
      </p>

      <div className="mt-6 rounded-lg border bg-card p-6">
        {isDevAuth ? (
          <>
            <ul className="flex flex-col gap-2">
              {devUsers.map((user) => (
                <li key={user.email}>
                  <form
                    action={async () => {
                      "use server";
                      await signIn("dev", {
                        email: user.email,
                        redirectTo: "/services",
                      });
                    }}
                  >
                    <button
                      type="submit"
                      className="w-full rounded-lg border px-4 py-3 text-left transition-colors hover:border-primary/40 hover:bg-accent focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
                    >
                      <span className="block text-sm font-medium">
                        {user.name}
                      </span>
                      <span className="mt-0.5 block font-mono text-xs text-muted-foreground">
                        {user.email}
                      </span>
                      <span className="mt-1 block font-mono text-xs text-muted-foreground/70">
                        {user.roles}
                      </span>
                    </button>
                  </form>
                </li>
              ))}
            </ul>
            <p className="mt-4 font-mono text-xs text-muted-foreground/70">
              Every persona reads the whole catalog. Roles decide who can write.
            </p>
          </>
        ) : (
          <form
            action={async () => {
              "use server";
              await signIn("microsoft-entra-id", { redirectTo: "/services" });
            }}
          >
            <Button type="submit" size="lg" className="w-full">
              Sign in with Microsoft
            </Button>
          </form>
        )}
      </div>
    </div>
  );
}
