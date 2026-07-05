import type { NextConfig } from "next";

/**
 * Build the Next.js image `remotePatterns` from environment variables so that
 * no image hosts are hard-coded (this is an open-source project — hosts differ
 * per deployment). Sources:
 *   - NEXT_PUBLIC_API_URL              → API host that serves locally-stored uploads
 *   - NEXT_PUBLIC_S3_PUBLIC_BASE_URL   → S3 (or S3-compatible) public bucket base
 *   - NEXT_PUBLIC_IMAGE_HOSTS          → extra hosts (comma-separated), e.g.
 *                                        placeholder/seed image hosts, wildcards
 *
 * Each entry in NEXT_PUBLIC_IMAGE_HOSTS is either a bare hostname (defaults to
 * https, wildcards like `*.s3.example.com` allowed) or `scheme://hostname` to
 * pin the protocol (e.g. `http://localhost`).
 */
type RemotePattern = NonNullable<NonNullable<NextConfig["images"]>["remotePatterns"]>[number];

function patternFromUrl(raw: string | undefined): RemotePattern[] {
  if (!raw) return [];
  try {
    const { protocol, hostname } = new URL(raw);
    return [{ protocol: protocol.replace(":", "") as "http" | "https", hostname }];
  } catch {
    return [];
  }
}

function patternFromHost(entry: string): RemotePattern[] {
  const host = entry.trim();
  if (!host) return [];
  // Allow `scheme://host` to override the default protocol.
  const withScheme = host.includes("://") ? host : `https://${host}`;
  try {
    const { protocol, hostname } = new URL(withScheme);
    return [{ protocol: protocol.replace(":", "") as "http" | "https", hostname }];
  } catch {
    return [];
  }
}

const remotePatterns: RemotePattern[] = [
  ...patternFromUrl(process.env.NEXT_PUBLIC_API_URL),
  ...patternFromUrl(process.env.NEXT_PUBLIC_S3_PUBLIC_BASE_URL),
  ...(process.env.NEXT_PUBLIC_IMAGE_HOSTS ?? "").split(",").flatMap(patternFromHost),
];

const nextConfig: NextConfig = {
  // Standalone output: self-contained server bundle for a small runtime image.
  output: "standalone",
  // React Compiler: stable in Next 16, enables automatic memoization.
  reactCompiler: true,
  images: {
    // Item photos come from S3 (NEXT_PUBLIC_S3_PUBLIC_BASE_URL) or the API host
    // (local uploads); seed/demo data may use extra hosts. All are supplied via
    // env so nothing is hard-coded here — see the helpers above.
    remotePatterns,
  },
};

export default nextConfig;
