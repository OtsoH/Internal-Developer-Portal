import Link from "next/link";

import { auth, signOut } from "@/auth";

const authControl =
  "rounded-md px-2 py-1 font-mono text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none";

const nav = [
  { href: "/services", label: "Services" },
  { href: "/teams", label: "Teams", disabled: true },
  { href: "/graph", label: "Graph", disabled: true },
];

export async function SiteHeader() {
  const session = await auth();

  return (
    <header className="border-b bg-card">
      <div className="mx-auto flex h-14 w-full max-w-6xl items-center gap-6 px-4 sm:px-6">
        <Link href="/" className="flex items-baseline gap-2">
          <span className="font-mono text-sm font-semibold text-primary">
            idp://
          </span>
          <span className="text-sm font-semibold tracking-tight">
            Internal Developer Portal
          </span>
        </Link>
        <nav className="flex items-center gap-1 text-sm">
          {nav.map((item) =>
            item.disabled ? (
              <span
                key={item.href}
                className="cursor-not-allowed rounded-md px-3 py-1.5 text-muted-foreground/50"
                title="Coming soon"
              >
                {item.label}
              </span>
            ) : (
              <Link
                key={item.href}
                href={item.href}
                className="rounded-md px-3 py-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              >
                {item.label}
              </Link>
            ),
          )}
        </nav>
        {session?.user ? (
          <div className="ml-auto flex items-center gap-3">
            <span className="hidden font-mono text-xs text-muted-foreground sm:block">
              {session.user.email}
            </span>
            <form
              action={async () => {
                "use server";
                await signOut({ redirectTo: "/signin" });
              }}
            >
              <button type="submit" className={authControl}>
                sign out
              </button>
            </form>
          </div>
        ) : (
          <Link href="/signin" className={`ml-auto ${authControl}`}>
            sign in
          </Link>
        )}
      </div>
    </header>
  );
}
