import { expect, test } from "@playwright/test";

/**
 * Homepage smoke tests. These assume the Go API is running (via `make dev` at
 * the repo root) so listings load from real data. The listing assertions wait
 * for client-side TanStack Query fetches to resolve.
 */
test.describe("homepage", () => {
  test("loads with 200 and Avi branding", async ({ page }) => {
    const res = await page.goto("/");
    expect(res?.status()).toBe(200);
    await expect(page).toHaveTitle(/Avi/);
    // Brand appears in the header logo.
    await expect(page.getByRole("banner").getByText("Avi", { exact: true })).toBeVisible();
  });

  test("shows hero and search bar", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Find what you need" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Search" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Post a listing" }).first()).toBeVisible();
  });

  test("renders the promo blocks", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("Finds for your interior")).toBeVisible();
    await expect(page.getByText("Tech at great prices")).toBeVisible();
    await expect(page.getByText("Get ready to travel")).toBeVisible();
  });

  test("renders both listing sections", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Recommended for you" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "New listings" })).toBeVisible();
  });

  test("loads real listing cards from the API", async ({ page }) => {
    await page.goto("/");
    // Cards link to /items/<slug>. Wait for at least one to appear after the
    // client-side fetch resolves (not present in the SSR skeleton shell).
    const cardLinks = page.locator('a[href^="/items/"]');
    await expect(cardLinks.first()).toBeVisible({ timeout: 15_000 });
    expect(await cardLinks.count()).toBeGreaterThan(0);
  });
});
