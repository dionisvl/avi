import { expect, test } from "@playwright/test";

test.describe("item detail gallery", () => {
  test("opens every product photo in the viewer", async ({ page }) => {
    const res = await page.goto("/items/iphone-14-pro-max");
    expect(res?.status()).toBe(200);

    await page.getByRole("button", { name: "Open photo 3" }).click();
    await expect(page.getByRole("dialog", { name: "Photo 3 of 3" })).toBeVisible();

    const nextPhoto = page.getByRole("button", { name: "Next photo" });
    const previousPhoto = page.getByRole("button", { name: "Previous photo" });

    await nextPhoto.click();
    await expect(page.getByRole("dialog", { name: "Photo 1 of 3" })).toBeVisible();

    await nextPhoto.click();
    await expect(page.getByRole("dialog", { name: "Photo 2 of 3" })).toBeVisible();

    await previousPhoto.click();
    await expect(page.getByRole("dialog", { name: "Photo 1 of 3" })).toBeVisible();

    await nextPhoto.click();
    await nextPhoto.click();
    await expect(page.getByRole("dialog", { name: "Photo 3 of 3" })).toBeVisible();

    await page.getByRole("button", { name: "Close photo" }).last().click();
    await expect(page.getByRole("dialog")).toBeHidden();
  });

  test("shows the seller message action on the item page", async ({ page }) => {
    const res = await page.goto("/items/iphone-14-pro-max");
    expect(res?.status()).toBe(200);

    await expect(page.getByRole("link", { name: "Message seller" })).toBeVisible();
  });
});
