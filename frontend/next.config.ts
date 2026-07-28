import type { NextConfig } from "next";

// /api/v1/* used to be a rewrite here. It's now owned by
// app/api/v1/[...path]/route.ts, which attaches credentials server-side and
// strips anything a browser could use to forge them — a rewrite can't do
// that. The frontend origin is still the only one the browser ever talks to
// (no CORS).
const nextConfig: NextConfig = {};

export default nextConfig;
