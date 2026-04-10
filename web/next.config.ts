import type { NextConfig } from "next";

/**
 * `/api/*` проксируется в runtime через `app/api/[[...path]]/route.ts`
 * на `API_PROXY_URL` (Docker Compose, Render и т.д.).
 */
const nextConfig: NextConfig = {};

export default nextConfig;
