import Link from "next/link";

const nav = [
  { href: "/services", label: "Services" },
  { href: "/teams", label: "Teams", disabled: true },
  { href: "/graph", label: "Graph", disabled: true },
];

export function SiteHeader() {
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
        <span className="ml-auto hidden font-mono text-xs text-muted-foreground sm:block">
          env: local
        </span>
      </div>
    </header>
  );
}
