import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";

export const metadata = { title: "Services" };

type Service = components["schemas"]["Service"];
type Lifecycle = components["schemas"]["Lifecycle"];

const lifecycleDot: Record<Lifecycle, string> = {
  production: "bg-status-production",
  beta: "bg-status-beta",
  deprecated: "bg-status-deprecated",
};

function LifecycleIndicator({ lifecycle }: { lifecycle: Lifecycle }) {
  return (
    <span className="inline-flex items-center gap-2 font-mono text-xs text-muted-foreground">
      <span
        aria-hidden
        className={`size-2 rounded-full ${lifecycleDot[lifecycle]}`}
      />
      {lifecycle}
    </span>
  );
}

function StatusLine({ services }: { services: Service[] }) {
  const teams = new Set(services.map((s) => s.team.slug)).size;
  const production = services.filter(
    (s) => s.lifecycle === "production",
  ).length;
  return (
    <p className="mt-2 font-mono text-xs text-muted-foreground">
      {services.length} services · {teams} teams · {production} in production
    </p>
  );
}

async function fetchServices(): Promise<
  { services: Service[] } | { error: string }
> {
  try {
    const { data, error } = await api.GET("/services", {
      cache: "no-store",
    });
    if (error) {
      return { error: `The API returned an error: ${error.message}` };
    }
    return { services: data.items };
  } catch {
    return {
      error:
        "The backend API is not reachable. Start it with `docker compose up` and reload.",
    };
  }
}

export default async function ServicesPage() {
  const result = await fetchServices();

  if ("error" in result) {
    return (
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Services</h1>
        <div className="mt-6 rounded-lg border border-dashed p-8 text-center">
          <p className="font-mono text-sm text-muted-foreground">
            {result.error}
          </p>
        </div>
      </div>
    );
  }

  const { services } = result;

  return (
    <div>
      <div className="flex items-end justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Services</h1>
          <StatusLine services={services} />
        </div>
      </div>

      {services.length === 0 ? (
        <div className="mt-6 rounded-lg border border-dashed p-8 text-center">
          <p className="text-sm text-muted-foreground">
            No services registered yet. Service creation arrives with the
            editor role in week 2.
          </p>
        </div>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-lg border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Service</TableHead>
                <TableHead>Team</TableHead>
                <TableHead>Lifecycle</TableHead>
                <TableHead>Tags</TableHead>
                <TableHead className="text-right">Repository</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {services.map((service) => (
                <TableRow key={service.id}>
                  <TableCell>
                    <div className="font-medium">{service.name}</div>
                    <div className="font-mono text-xs text-muted-foreground">
                      {service.slug}
                    </div>
                  </TableCell>
                  <TableCell>{service.team.name}</TableCell>
                  <TableCell>
                    <LifecycleIndicator lifecycle={service.lifecycle} />
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {service.tags.map((tag) => (
                        <Badge
                          key={tag}
                          variant="secondary"
                          className="font-mono text-xs"
                        >
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell className="text-right">
                    {service.repoUrl ? (
                      <a
                        href={service.repoUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="font-mono text-xs text-primary underline-offset-4 hover:underline"
                      >
                        repo ↗
                      </a>
                    ) : (
                      <span className="font-mono text-xs text-muted-foreground/50">
                        —
                      </span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
