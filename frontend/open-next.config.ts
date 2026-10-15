import { defineCloudflareConfig } from "@opennextjs/cloudflare";

// Default OpenNext Cloudflare configuration: full Next.js routing support
// served from static assets + the Worker bundle. No incremental cache
// binding yet — the demo app is client-side-fetched, so ISR caching is
// not needed; add defineCloudflareConfig({ incrementalCache: ... }) later
// if server-side caching becomes valuable.
export default defineCloudflareConfig();
