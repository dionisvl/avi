// @ts-check
const { test, expect } = require('@playwright/test');
const { AUTH_TOKEN, BASE_URL, mockCommonApi } = require('./helpers');

test.use({ storageState: { cookies: [], origins: [] } });

const loginForm = (page) => page.locator('[data-testid="login-form"]');

test('shows login form on open', async ({ page }) => {
    await mockCommonApi(page);
    await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
    await expect(loginForm(page).locator('input[type="email"]')).toBeVisible({ timeout: 5_000 });
    await expect(loginForm(page).locator('button:has-text("Login")')).toBeVisible();
});

test('full auth flow: login -> items visible -> logout', async ({ page }) => {
    await mockCommonApi(page);
    await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
    await loginForm(page).locator('input[type="email"]').fill('test@example.com');
    await loginForm(page).locator('input[type="password"]').fill('password123');
    await loginForm(page).locator('button:has-text("Login")').click();

    await expect.poll(() => page.evaluate(() => Boolean(localStorage.getItem('auth_token')))).toBe(true);
    await expect(page.locator('[data-testid="nav-logout"]')).toBeVisible({ timeout: 5_000 });
    await expect(page.locator('[data-testid="items-section"]')).toBeVisible({ timeout: 5_000 });

    await page.locator('[data-testid="nav-logout"]').click();
    expect(await page.evaluate(() => localStorage.getItem('auth_token'))).toBeNull();
    await expect(loginForm(page).locator('button:has-text("Login")')).toBeVisible();
});

test('redirects to login when stored session is rejected on startup', async ({ page }) => {
    await mockCommonApi(page);
    await page.route('**/api/v1/user/me', async (route) => {
        await route.fulfill({
            status: 401,
            contentType: 'application/json',
            body: JSON.stringify({ detail: 'Unauthorized' }),
        });
    });
    await page.addInitScript((token) => {
        localStorage.setItem('auth_token', token);
    }, AUTH_TOKEN);

    await page.goto(`${BASE_URL}/profile`, { waitUntil: 'domcontentloaded' });

    await expect(page).toHaveURL(/\/login$/);
    await expect(loginForm(page)).toBeVisible({ timeout: 5_000 });
    await expect.poll(() => page.evaluate(() => localStorage.getItem('auth_token'))).toBeNull();
});

test('redirects to login when an authenticated request loses the session', async ({ page }) => {
    await mockCommonApi(page);
    await page.route('**/api/v1/items**', async (route) => {
        if (new URL(route.request().url()).pathname !== '/api/v1/items') {
            await route.fallback();
            return;
        }
        await route.fulfill({
            status: 401,
            contentType: 'application/json',
            body: JSON.stringify({ detail: 'Session expired' }),
        });
    });
    await page.addInitScript((token) => {
        localStorage.setItem('auth_token', token);
    }, AUTH_TOKEN);

    await page.goto(`${BASE_URL}/items`, { waitUntil: 'domcontentloaded' });

    await expect(page).toHaveURL(/\/login$/);
    await expect(loginForm(page)).toBeVisible({ timeout: 5_000 });
    await expect.poll(() => page.evaluate(() => localStorage.getItem('auth_token'))).toBeNull();
});

test('registration sends neutral user payload without role selection', async ({ page }) => {
    let registerPayload = null;
    await mockCommonApi(page);
    await page.route('**/api/v1/auth/register', async (route) => {
        registerPayload = route.request().postDataJSON();
        await route.fulfill({
            status: 201,
            contentType: 'application/json',
            body: JSON.stringify({ message: 'registered' }),
        });
    });

    await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
    await page.getByRole('button', { name: 'Create Account' }).click();
    await page.locator('[data-testid="register-form"] input[type="email"]').fill('new-user@example.com');
    await page.locator('[data-testid="register-form"] input[type="password"]').first().fill('password123');
    await page.locator('[data-testid="register-form"] input[type="password"]').last().fill('password123');
    await page.locator('[data-testid="register-form"] button:has-text("Register")').click();

    await expect.poll(() => registerPayload).not.toBeNull();
    expect(registerPayload).toMatchObject({
        email: 'new-user@example.com',
        password: 'password123',
        locale: 'ru',
    });
    expect(registerPayload).not.toHaveProperty('roles');
    await expect(page.locator('[data-testid="verify-form"]')).toBeVisible();
});
