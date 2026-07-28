// The Entra access token rides on the session so the BFF proxy can attach it
// server-side. It is never exposed to the browser.
declare module "next-auth" {
  interface Session {
    accessToken?: string;
  }
}

// The JWT side is deliberately not augmented: `next-auth/jwt` only re-exports
// `@auth/core/jwt`, so augmenting it declares a fresh interface instead of
// merging, and `@auth/core` is not resolvable from here under pnpm. auth.ts
// narrows the token bag at runtime instead.

export {};
