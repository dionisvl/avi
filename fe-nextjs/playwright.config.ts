import { defineConfig, devices } from "@playwright/test";

/**
 * Smoke-level e2e config. Targets the app through the Traefik host (never
 * localhost). Start the stack first with `make dev` at the repo root so both
 * the frontend (http://avi.test) and the Go API are up.
 */
const BASE_URL = process.env.E2E_BASE_URL ?? "http://avi.test";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  reporter: "list",
  use: {
    baseURL: BASE_URL,
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
