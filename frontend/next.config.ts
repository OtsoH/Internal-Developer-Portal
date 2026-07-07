import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    // Browser requests to /api/v1/* are proxied to the Go backend so the
    // frontend origin is the only one the browser ever talks to (no CORS).
    const backend = process.env.BACKEND_URL ?? "http://localhost:8080";
    return [
      {
        source: "/api/v1/:path*",
        destination: `${backend}/api/v1/:path*`,
      },
    ];
  },
};

export default nextConfig;
