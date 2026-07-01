// @ts-check
const { test, expect } = require('@playwright/test');
const path = require('path');
const { BASE_URL, ELECTRONICS, MOSCOW, mockCommonApi } = require('./helpers');

const AUTH_STATE = path.join(__dirname, '.auth.json');

async function goToProfile(page) {
    await mockCommonApi(page);
    await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
    await page.locator('[data-testid="nav-profile"]').click();
    await expect(page.locator('[data-testid="profile-section"]')).toBeVisible({ timeout: 5_000 });
}

test.describe('profile', () => {
    test.use({ storageState: AUTH_STATE });

    test('profile tab opens with account controls', async ({ page }) => {
        await goToProfile(page);

        await expect(page.locator('[data-testid="profile-name"]')).toBeVisible();
        await expect(page.locator('[data-testid="profile-save"]')).toBeVisible();
        await expect(page.locator('[data-testid="current-password"]')).toBeVisible();
        await expect(page.locator('[data-testid="new-password"]')).toBeVisible();
        await expect(page.locator('[data-testid="delete-account-btn"]')).toBeVisible();
    });

    test('update name shows success message', async ({ page }) => {
        await goToProfile(page);
        const profile = page.locator('[data-testid="profile-section"]');

        await profile.locator('[data-testid="profile-name"]').fill('Test User Updated');
        await profile.locator('[data-testid="profile-save"]').click();

        await expect(profile.locator('[data-testid="profile-success"]')).toContainText('updated', { timeout: 5_000 });
    });

    test('update profile sends item-based preferences', async ({ page }) => {
        let patchPayload = null;
        await mockCommonApi(page);
        await page.route('**/api/v1/user/me', async (route) => {
            if (route.request().method() === 'PATCH') {
                patchPayload = route.request().postDataJSON();
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        id: '11111111-1111-4111-8111-111111111111',
                        email: 'test@example.com',
                        name: patchPayload.name,
                        roles: ['ROLE_USER'],
                        preferences: patchPayload.preferences,
                    }),
                });
                return;
            }

            await route.fallback();
        });

        await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
        await page.locator('[data-testid="nav-profile"]').click();
        const profile = page.locator('[data-testid="profile-section"]');

        await profile.locator('[data-testid="profile-name"]').fill('Test User Updated');
        await profile.locator('[data-testid="profile-pref-category"]').selectOption(ELECTRONICS.id);
        await profile.locator('[data-testid="profile-pref-city"]').selectOption(MOSCOW.id);
        await profile.locator('[data-testid="profile-pref-condition"]').selectOption('used');
        await profile.locator('[data-testid="profile-pref-search"]').fill('phone');
        await profile.locator('[data-testid="profile-pref-price-min"]').fill('1000');
        await profile.locator('[data-testid="profile-pref-price-max"]').fill('9900000');
        await profile.locator('[data-testid="profile-save"]').click();

        await expect.poll(() => patchPayload).not.toBeNull();
        expect(patchPayload).toMatchObject({
            name: 'Test User Updated',
            preferences: {
                category_id: ELECTRONICS.id,
                city_id: MOSCOW.id,
                condition: 'used',
                search: 'phone',
                price_min: 1000,
                price_max: 9900000,
            },
        });
        await expect(profile.locator('[data-testid="profile-success"]')).toContainText('updated', { timeout: 5_000 });
    });

    test('empty name shows validation error', async ({ page }) => {
        await goToProfile(page);
        const profile = page.locator('[data-testid="profile-section"]');

        await profile.locator('[data-testid="profile-name"]').fill('');
        await profile.locator('[data-testid="profile-save"]').click();

        await expect(profile.locator('[data-testid="profile-error"]')).toContainText('required', { timeout: 3_000 });
    });

    test('change password error and success states', async ({ page }) => {
        await goToProfile(page);
        const profile = page.locator('[data-testid="profile-section"]');

        await profile.locator('[data-testid="current-password"]').fill('wrongpassword');
        await profile.locator('[data-testid="new-password"]').fill('newpassword999');
        await profile.locator('[data-testid="change-password-btn"]').click();
        await expect(profile.locator('[data-testid="change-password-error"]')).toContainText('Invalid current password', { timeout: 5_000 });

        await profile.locator('[data-testid="current-password"]').fill('password123');
        await profile.locator('[data-testid="new-password"]').fill('password1234');
        await profile.locator('[data-testid="change-password-btn"]').click();
        await expect(profile.locator('[data-testid="change-password-success"]')).toContainText('changed', { timeout: 5_000 });
    });

    test('delete account requires password', async ({ page }) => {
        await goToProfile(page);
        const profile = page.locator('[data-testid="profile-section"]');

        await profile.locator('[data-testid="delete-password"]').fill('');
        await profile.locator('[data-testid="delete-account-btn"]').click();

        await expect(profile.locator('[data-testid="delete-account-error"]')).toContainText('required', { timeout: 3_000 });
    });
});
