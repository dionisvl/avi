// @ts-check
const { test, expect } = require('@playwright/test');
const path = require('path');
const { AUTH_TOKEN, BASE_URL, DEMO_ITEMS, ELECTRONICS, MOSCOW, mockCommonApi } = require('./helpers');

const AUTH_STATE = path.join(__dirname, '.auth.json');

async function goToItems(page) {
    await mockCommonApi(page);
    await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
    await expect(page.locator('[data-testid="nav-items"]')).toBeVisible({ timeout: 5_000 });
    await page.locator('[data-testid="nav-items"]').click();
    await expect(page.locator('[data-testid="item-row"]').first()).toBeVisible({ timeout: 5_000 });
}

test.describe('items', () => {
    test.use({ storageState: AUTH_STATE });

    test('catalog renders item rows with category, condition, city, and price', async ({ page }) => {
        await goToItems(page);

        const firstRow = page.locator('[data-testid="item-row"]').first();
        await expect(firstRow).toContainText('iPhone 15 Pro');
        await expect(firstRow).toContainText('Electronics');
        await expect(firstRow).toContainText('Moscow');
        await expect(firstRow).toContainText('used');
        await expect(firstRow).toContainText('RUB');
    });

    test('catalog switches category and city labels between en and ru locales', async ({ page }) => {
        await goToItems(page);

        const firstRow = page.locator('[data-testid="item-row"]').first();
        await expect(firstRow).toContainText('Electronics');
        await expect(firstRow).toContainText('Moscow');

        await page.locator('[data-testid="locale-ru"]').click();
        await expect(firstRow).toContainText('Электроника');
        await expect(firstRow).toContainText('Москва');
        await expect(page.locator('[data-testid="items-category-filter"]')).toContainText('Электроника');

        await page.locator('[data-testid="locale-en"]').click();
        await expect(firstRow).toContainText('Electronics');
        await expect(firstRow).toContainText('Moscow');
    });

    test('filters build classifieds query params', async ({ page }) => {
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

        await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
        await page.locator('[data-testid="nav-items"]').click();
        await page.locator('[data-testid="items-search"]').fill('phone');
        await page.locator('[data-testid="items-category-filter"]').selectOption(ELECTRONICS.id);
        await page.locator('[data-testid="items-condition-filter"]').selectOption('used');
        await page.locator('[data-testid="items-city-filter"]').selectOption(MOSCOW.id);
        await page.locator('[data-testid="items-price-min"]').fill('1000');
        await page.locator('[data-testid="items-price-max"]').fill('9900000');
        await page.locator('[data-testid="items-mine-only"]').check();
        await page.locator('[data-testid="items-filter"]').click();

        await expect.poll(() => seenUrls.length).toBeGreaterThan(0);
        const url = new URL(seenUrls.at(-1));
        expect(url.searchParams.get('search')).toBe('phone');
        expect(url.searchParams.get('category_ids')).toBe(ELECTRONICS.id);
        expect(url.searchParams.get('condition')).toBe('used');
        expect(url.searchParams.get('city_uuid')).toBe(MOSCOW.id);
        expect(url.searchParams.get('price_min')).toBe('1000');
        expect(url.searchParams.get('price_max')).toBe('9900000');
        expect(url.searchParams.get('seller_id')).toBe('11111111-1111-4111-8111-111111111111');
    });

    test('detail opens and promote listing posts payment payload', async ({ page }) => {
        let paymentPayload = null;
        await mockCommonApi(page);
        await page.route('**/api/v1/payments', async (route) => {
            paymentPayload = route.request().postDataJSON();
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    id: 'payment-1',
                    status: 'pending',
                    amount: { value: '100.00', currency: 'RUB' },
                    confirmation_url: 'https://payments.test/confirm/payment-1',
                }),
            });
        });

        await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
        await page.evaluate(() => {
            window.__paymentRedirectURL = '';
            const root = document.querySelector('[x-data]');
            root._x_dataStack[0].redirectToPayment = (url) => {
                window.__paymentRedirectURL = url;
            };
        });
        await page.locator('[data-testid="nav-items"]').click();
        await page.locator('[data-testid="item-row"]').first().getByRole('button', { name: 'View' }).click();

        const detail = page.locator('[data-testid="item-detail-section"]');
        await expect(detail.locator('h2')).toContainText('iPhone 15 Pro');
        await expect(detail).toContainText('Seller');
        await expect(detail.locator('[data-testid="promotion-payment-panel"]')).toBeVisible();
        await detail.locator('[data-testid="promote-listing"]').click();

        await expect.poll(() => page.evaluate(() => window.__paymentRedirectURL)).toBe('https://payments.test/confirm/payment-1');
        expect(paymentPayload).toMatchObject({
            purpose: 'promote_listing',
            subject_id: DEMO_ITEMS[0].id,
        });
        expect(paymentPayload.return_url).toContain('payment_return=promote_listing');
        expect(paymentPayload.return_url).toContain(`item_id=${DEMO_ITEMS[0].id}`);
    });

    test('create form validates and submits item payload', async ({ page }) => {
        let createPayload = null;
        await mockCommonApi(page);
        await page.route('**/api/v1/items', async (route) => {
            if (route.request().method() === 'POST') {
                createPayload = route.request().postDataJSON();
                await route.fulfill({
                    status: 201,
                    contentType: 'application/json',
                    body: JSON.stringify({ data: { ...DEMO_ITEMS[0], ...createPayload, id: 'new-item' } }),
                });
                return;
            }

            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    data: DEMO_ITEMS,
                    pagination: { page: 1, per_page: 20, total: DEMO_ITEMS.length, total_pages: 1 },
                }),
            });
        });

        await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
        await page.locator('[data-testid="nav-items"]').click();
        await page.locator('[data-testid="items-add"]').click();

        const form = page.locator('[data-testid="item-form-section"]');
        await expect(form.locator('h2')).toContainText('Add Item');
        await form.locator('[data-testid="item-title"]').fill('');
        await form.locator('[data-testid="item-category"]').selectOption('');
        await form.locator('[data-testid="item-create-submit"]').click();
        await expect(form).toContainText('Title, category, and city are required');

        await form.locator('[data-testid="item-title"]').fill('Demo laptop');
        await form.locator('[data-testid="item-category"]').selectOption(ELECTRONICS.id);
        await form.locator('[data-testid="item-city"]').selectOption(MOSCOW.id);
        await form.locator('[data-testid="item-condition"]').selectOption('used');
        await form.locator('[data-testid="item-price-amount"]').fill('1234500');
        await form.locator('[data-testid="item-tags"]').fill('warranty, delivery');
        await form.locator('[data-testid="item-create-submit"]').click();

        await expect.poll(() => createPayload).not.toBeNull();
        expect(createPayload).toMatchObject({
            title: 'Demo laptop',
            category_id: ELECTRONICS.id,
            city_uuid: MOSCOW.id,
            condition: 'used',
            price: { amount: 1234500, currency: 'RUB' },
            tags: ['warranty', 'delivery'],
        });
    });
});
