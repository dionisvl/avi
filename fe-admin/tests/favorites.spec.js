// @ts-check
const { test, expect } = require('@playwright/test');
const path = require('path');
const { BASE_URL, DEMO_ITEMS, mockCommonApi } = require('./helpers');

const AUTH_STATE = path.join(__dirname, '.auth.fixture-user-only.json');

test.describe('favorites', () => {
    test.use({ storageState: AUTH_STATE });

    test('favorites tab opens and shows empty state', async ({ page }) => {
        await mockCommonApi(page, { favorites: [] });
        await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
        await page.locator('[data-testid="nav-favorites"]').click();
        await expect(page.locator('[data-testid="favorites-section"]')).toBeVisible({ timeout: 5_000 });
        await expect(page.locator('text=No favorites yet')).toBeVisible({ timeout: 5_000 });
    });

    test('add and remove item favorite from detail', async ({ page }) => {
        const favorites = [];
        await mockCommonApi(page, { favorites });

        await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
        await page.locator('[data-testid="nav-items"]').click();
        await page.locator('[data-testid="item-row"]').first().getByRole('button', { name: 'View' }).click();

        const detail = page.locator('[data-testid="item-detail-section"]');
        const addBtn = detail.locator('[data-testid="add-favorite"]');
        const removeBtn = detail.locator('[data-testid="remove-favorite"]');

        await expect(addBtn).toBeVisible({ timeout: 5_000 });
        await addBtn.click();
        await expect(removeBtn).toBeVisible({ timeout: 5_000 });

        await page.locator('[data-testid="nav-favorites"]').click();
        await expect(page.locator('[data-testid="favorite-item"]')).toHaveCount(1, { timeout: 5_000 });
        await expect(page.locator('[data-testid="favorite-item"]')).toContainText(DEMO_ITEMS[0].title);

        await page.locator('[data-testid="remove-favorite-btn"]').click();
        await expect(page.locator('[data-testid="favorite-item"]')).toHaveCount(0, { timeout: 5_000 });
        await expect(page.locator('text=No favorites yet')).toBeVisible({ timeout: 5_000 });
    });
});
