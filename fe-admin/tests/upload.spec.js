// @ts-check
const { test, expect } = require('@playwright/test');
const path = require('path');
const { BASE_URL, mockCommonApi } = require('./helpers');

const AUTH_STATE = path.join(__dirname, '.auth.json');

test.describe('upload ui', () => {
    test.use({ storageState: AUTH_STATE });

    test('item create form initializes photo upload control', async ({ page }) => {
        await mockCommonApi(page);
        await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
        await page.locator('[data-testid="nav-items"]').click();
        await page.locator('[data-testid="items-add"]').click();

        await expect(page.locator('#item-photo-input')).toBeAttached();
        await expect(page.locator('.filepond--root').first()).toBeVisible({ timeout: 5_000 });
    });
});
