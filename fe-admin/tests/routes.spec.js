// @ts-check
const { test, expect } = require('@playwright/test');
const path = require('path');
const { BASE_URL, DEMO_ITEMS, ELECTRONICS, MOSCOW, mockCommonApi } = require('./helpers');

const AUTH_STATE = path.join(__dirname, '.auth.json');

test.describe('browser routes', () => {
    test.use({ storageState: AUTH_STATE });

    test('opens catalog from /items deep link', async ({ page }) => {
        await mockCommonApi(page);

        await page.goto(`${BASE_URL}/items`, { waitUntil: 'domcontentloaded' });

        await expect(page).toHaveURL(/\/items$/);
        await expect(page.locator('[data-testid="items-section"]')).toBeVisible({ timeout: 5_000 });
        await expect(page.locator('[data-testid="item-row"]').first()).toContainText(DEMO_ITEMS[0].title);
    });

    test('opens item detail from /items/:id deep link', async ({ page }) => {
        await mockCommonApi(page);

        await page.goto(`${BASE_URL}/items/${DEMO_ITEMS[0].id}`, { waitUntil: 'domcontentloaded' });

        await expect(page).toHaveURL(new RegExp(`/items/${DEMO_ITEMS[0].id}$`));
        await expect(page.locator('[data-testid="item-detail-section"] h2')).toContainText(DEMO_ITEMS[0].title, { timeout: 5_000 });
    });

    test('updates browser history when opening and leaving item detail', async ({ page }) => {
        await mockCommonApi(page);

        await page.goto(`${BASE_URL}/items`, { waitUntil: 'domcontentloaded' });
        await page.locator('[data-testid="item-row"]').first().getByRole('button', { name: 'View' }).click();
        await expect(page).toHaveURL(new RegExp(`/items/${DEMO_ITEMS[0].id}$`));
        await expect(page.locator('[data-testid="item-detail-section"]')).toBeVisible();

        await page.goBack();

        await expect(page).toHaveURL(/\/items$/);
        await expect(page.locator('[data-testid="items-section"]')).toBeVisible({ timeout: 5_000 });
    });

    test('hydrates and syncs catalog filters with the query string', async ({ page }) => {
        const seenUrls = [];
        await mockCommonApi(page);
        await page.route('**/api/v1/items?*', async (route) => {
            seenUrls.push(route.request().url());
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    data: DEMO_ITEMS,
                    pagination: { page: 1, per_page: 20, total: DEMO_ITEMS.length, total_pages: 1 },
                }),
            });
        });

        await page.goto(`${BASE_URL}/items?search=phone&category_ids=${ELECTRONICS.id}&condition=used&city_uuid=${MOSCOW.id}&price_min=1000&price_max=9900000`, {
            waitUntil: 'domcontentloaded',
        });

        await expect(page.locator('[data-testid="items-section"]')).toBeVisible({ timeout: 5_000 });
        await expect(page.locator('[data-testid="items-search"]')).toHaveValue('phone');
        await expect(page.locator('[data-testid="items-category-filter"]')).toHaveValue(ELECTRONICS.id);
        await expect(page.locator('[data-testid="items-condition-filter"]')).toHaveValue('used');
        await expect(page.locator('[data-testid="items-city-filter"]')).toHaveValue(MOSCOW.id);
        await expect(page.locator('[data-testid="items-price-min"]')).toHaveValue('1000');
        await expect(page.locator('[data-testid="items-price-max"]')).toHaveValue('9900000');

        await expect.poll(() => seenUrls.length).toBeGreaterThan(0);
        const apiRequestUrl = new URL(seenUrls.at(-1));
        expect(apiRequestUrl.searchParams.get('search')).toBe('phone');
        expect(apiRequestUrl.searchParams.get('category_ids')).toBe(ELECTRONICS.id);

        await page.locator('[data-testid="items-search"]').fill('bike');
        await page.locator('[data-testid="items-filter"]').click();

        await expect(page).toHaveURL(/\/items\?/);
        const browserUrl = new URL(page.url());
        expect(browserUrl.searchParams.get('search')).toBe('bike');
        expect(browserUrl.searchParams.get('category_ids')).toBe(ELECTRONICS.id);
    });
});

test.describe('auth route guard', () => {
    test.use({ storageState: { cookies: [], origins: [] } });

    test('returns to protected route after login', async ({ page }) => {
        await mockCommonApi(page);

        await page.goto(`${BASE_URL}/profile`, { waitUntil: 'domcontentloaded' });

        await expect(page.locator('[data-testid="login-form"]')).toBeVisible({ timeout: 5_000 });
        await expect.poll(() => new URL(page.url()).searchParams.get('return_to')).toBe('/profile');

        await page.locator('[data-testid="login-form"] input[type="email"]').fill('test@example.com');
        await page.locator('[data-testid="login-form"] input[type="password"]').fill('password123');
        await page.locator('[data-testid="login-form"] button:has-text("Login")').click();

        await expect(page).toHaveURL(/\/profile$/);
        await expect(page.locator('[data-testid="profile-section"]')).toBeVisible({ timeout: 5_000 });
    });
});
