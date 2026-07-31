import Link from "next/link";

import { ServiceForm } from "@/components/services/service-form";
import { getCurrentUser } from "@/lib/api/server";
import { requireSession } from "@/lib/auth/session";

export const metadata = { title: "Register a service" };

function Panel({ children }: { children: React.ReactNode }) {
  return (
    <div className="mt-6 rounded-lg border border-dashed p-8 text-center">
      {children}
    </div>
  );
}

export default async function NewServicePage() {
  await requireSession();
  const user = await getCurrentUser();

  // A null user means /me failed, not that the caller lacks a role. Saying
  // "you cannot register services" when the backend is down sends people to
  // ask an admin for access they already have.
  const roles = user?.teamRoles;
  const editableTeams =
    roles?.filter((team) => team.role === "ADMIN" || team.role === "EDITOR") ??
    [];

  return (
    <div className="mx-auto w-full max-w-2xl">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">
          Register a service
        </h1>
        <p className="mt-2 font-mono text-xs text-muted-foreground">
          new service · owned by a team you can write to
        </p>
      </div>

      {!roles ? (
        <Panel>
          <p className="font-mono text-sm text-muted-foreground">
            Could not load your team roles. Reload once the backend is back.
          </p>
        </Panel>
      ) : editableTeams.length === 0 ? (
        <Panel>
          <p className="text-sm text-muted-foreground">
            Registering a service needs the editor role on a team. Ask an
            administrator of the team that will own it to add you.
          </p>
          <Link
            href="/services"
            className="mt-4 inline-block font-mono text-xs text-primary underline-offset-4 hover:underline"
          >
            back to services
          </Link>
        </Panel>
      ) : (
        <div className="mt-6">
          <ServiceForm mode="create" teams={editableTeams} />
        </div>
      )}
    </div>
  );
}
