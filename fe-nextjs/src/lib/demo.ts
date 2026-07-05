/**
 * Demo-mode helpers.
 *
 * This build is a public demo: registration and personal-data collection are
 * disabled on the frontend, and a few preset accounts are advertised openly so
 * anyone can sign in. All behaviour is driven by build-time `NEXT_PUBLIC_*` env
 * vars (see the root `.env` files and the migration `00002_seed_demo_users.sql`,
 * which is the source of truth for the preset accounts).
 */

/** A preset account whose credentials are shown publicly on the login screen. */
export interface DemoUser {
  email: string;
  password: string;
  label: string;
}

/** True when the app runs as a public demo (registration + PD collection disabled). */
export function isDemoMode(): boolean {
  return process.env.NEXT_PUBLIC_DEMO_MODE === "true";
}

/**
 * Preset accounts parsed from `NEXT_PUBLIC_DEMO_USERS` (a JSON array).
 * Returns an empty list if the var is missing or malformed. Must mirror the
 * users seeded by migration `00002_seed_demo_users.sql`.
 */
export function getDemoUsers(): DemoUser[] {
  try {
    const parsed = JSON.parse(process.env.NEXT_PUBLIC_DEMO_USERS ?? "[]");
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(
      (u): u is DemoUser =>
        typeof u?.email === "string" &&
        typeof u?.password === "string" &&
        typeof u?.label === "string",
    );
  } catch {
    return [];
  }
}

/**
 * Guard for personal-data-mutating flows (registration, contact/lead forms).
 * Throws in demo mode so callers can short-circuit and surface a message.
 */
export function assertNotDemo(): void {
  if (isDemoMode()) {
    throw new Error("Disabled in demo mode");
  }
}
