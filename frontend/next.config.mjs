import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Pins Turbopack's project root explicitly. Without this it walks up the
  // filesystem looking for a lockfile to infer the root, and on this
  // machine finds an unrelated one outside the repo — harmless, but this
  // avoids relying on directory-tree guessing.
  turbopack: {
    root: __dirname,
  },
};

export default nextConfig;