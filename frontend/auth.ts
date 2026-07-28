import NextAuth, { type NextAuthConfig } from "next-auth";
import Credentials from "next-auth/providers/credentials";
import MicrosoftEntraID from "next-auth/providers/microsoft-entra-id";

const isProd = process.env.NODE_ENV === "production";

// Unset means dev locally and entra in production, so forgetting the variable
// never downgrades a deployment to the seeded personas. Matches the backend's
// ConfigFromEnv, which likewise errors rather than falling back silently.
const authMode = process.env.AUTH_MODE ?? (isProd ? "entra" : "dev");
if (authMode !== "dev" && authMode !== "entra") {
  throw new Error(`AUTH_MODE must be "dev" or "entra", got "${authMode}".`);
}

// The throw is the gate that matters: a production build that explicitly asks
// for dev mode fails at module load rather than shipping a sign-in page that
// hands out any identity for the asking.
if (authMode === "dev" && isProd) {
  throw new Error(
    "AUTH_MODE=dev is not permitted in a production build. Set AUTH_MODE=entra.",
  );
}

export const isDevAuth = authMode === "dev" && !isProd;

/**
 * The personas seeded by `backend/internal/db/seed.sql`. The dev provider
 * authorizes these three addresses and nothing else, so a typo cannot become a
 * silent role-less viewer. `roles` is display copy for the sign-in page — the
 * real roles always come from `GET /me`.
 */
export const devUsers = [
  {
    email: "dev.admin@example.com",
    name: "Dev Admin",
    roles: "ADMIN on Platform · no membership in Payments",
  },
  {
    email: "dev.editor@example.com",
    name: "Dev Editor",
    roles: "VIEWER on Platform · EDITOR on Payments",
  },
  {
    email: "dev.viewer@example.com",
    name: "Dev Viewer",
    roles: "VIEWER on Platform · no membership in Payments",
  },
] as const;

const devProvider = Credentials({
  id: "dev",
  name: "Dev persona",
  credentials: { email: { label: "Email", type: "email" } },
  authorize(credentials) {
    const email = String(credentials?.email ?? "").toLowerCase();
    const user = devUsers.find((u) => u.email === email);
    return user ? { id: user.email, email: user.email, name: user.name } : null;
  },
});

// The API scope is what makes Entra address the access token to our backend
// rather than to Microsoft Graph. Getting it wrong yields a token that looks
// valid and fails the backend's audience check — see docs/entra-setup.md.
const entraProvider = MicrosoftEntraID({
  authorization: {
    params: {
      scope: ["openid", "profile", "email", "offline_access", process.env.BACKEND_API_SCOPE]
        .filter(Boolean)
        .join(" "),
    },
  },
});

const providers: NextAuthConfig["providers"] = isDevAuth
  ? [devProvider]
  : [entraProvider];

export const { handlers, auth, signIn, signOut } = NextAuth({
  providers,
  pages: { signIn: "/signin" },
  callbacks: {
    // The session carries identity and the access token, never roles: roles are
    // read from the database on every request so a team_members change takes
    // effect without anyone signing out.
    jwt({ token, account }) {
      if (account?.access_token) {
        token.accessToken = account.access_token;
      }
      return token;
    },
    session({ session, token }) {
      if (typeof token.accessToken === "string") {
        session.accessToken = token.accessToken;
      }
      return session;
    },
  },
});
