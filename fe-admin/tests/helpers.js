// @ts-check
const path = require('path');
const fs = require('fs');

const BASE_URL = 'http://127.0.0.1:4173';
const API_URL = 'http://api.avi.test';
const API_V1_URL = `${API_URL}/api/v1`;

const AUTH_TOKEN = 'mock-access-token';

const TEST_USER = {
    id: '11111111-1111-4111-8111-111111111111',
    email: 'test@example.com',
    password: 'password123',
    name: 'Test User',
    roles: ['ROLE_USER'],
};

const FIXTURE_USER_ROLE_USER_ONLY = {
    ...TEST_USER,
    id: '22222222-2222-4222-8222-222222222222',
    email: 'fixture-user@example.com',
};

const MOSCOW = {
    id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
    slug: 'moscow',
    names: { en: 'Moscow', ru: 'Москва' },
    is_active: true,
};

const ELECTRONICS = {
    id: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb',
    slug: 'electronics',
    names: { en: 'Electronics', ru: 'Электроника' },
    sort_order: 10,
};

const TRANSPORT = {
    id: 'cccccccc-cccc-4ccc-8ccc-cccccccccccc',
    slug: 'transport',
    names: { en: 'Transport', ru: 'Транспорт' },
    sort_order: 20,
};

const DEMO_ITEMS = [
    {
        id: 'dddddddd-dddd-4ddd-8ddd-dddddddddddd',
        seller_id: TEST_USER.id,
        seller: { id: TEST_USER.id, email: TEST_USER.email, name: TEST_USER.name },
        category_id: ELECTRONICS.id,
        category: ELECTRONICS,
        city_id: MOSCOW.id,
        city: MOSCOW,
        slug: 'iphone-15-pro',
        title: 'iPhone 15 Pro',
        description: 'Clean demo listing with realistic metadata.',
        price: { amount: 9500000, currency: 'RUB' },
        condition: 'used',
        status: 'published',
        tags: ['phone', 'warranty'],
        photos: [],
        is_favorited: false,
        created_at: '2026-06-01T10:00:00Z',
        updated_at: '2026-06-01T10:00:00Z',
    },
    {
        id: 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee',
        seller_id: '33333333-3333-4333-8333-333333333333',
        seller: { id: '33333333-3333-4333-8333-333333333333', email: 'seller@example.com', name: 'Seller User' },
        category_id: TRANSPORT.id,
        category: TRANSPORT,
        city_id: MOSCOW.id,
        city: MOSCOW,
        slug: 'city-bike',
        title: 'City Bike',
        description: 'Almost new bike.',
        price: { amount: 2500000, currency: 'RUB' },
        condition: 'new',
        status: 'published',
        tags: ['delivery'],
        photos: [],
        is_favorited: false,
        created_at: '2026-06-02T10:00:00Z',
        updated_at: '2026-06-02T10:00:00Z',
    },
];

function buildStorageState(token = AUTH_TOKEN) {
    return {
        cookies: [],
        origins: [
            {
                origin: BASE_URL,
                localStorage: [{ name: 'auth_token', value: token }],
            },
        ],
    };
}

async function loginAndSaveState() {
    const statePath = path.join(__dirname, '.auth.json');
    fs.writeFileSync(statePath, JSON.stringify(buildStorageState(), null, 2));
    return statePath;
}

async function loginAndSaveStateAs(user, filename) {
    const statePath = path.join(__dirname, filename);
    fs.writeFileSync(statePath, JSON.stringify(buildStorageState(`${AUTH_TOKEN}-${user.email}`), null, 2));
    return statePath;
}

async function mockCommonApi(page, { user = TEST_USER, items = DEMO_ITEMS, favorites = [] } = {}) {
    await page.route('**/api/v1/cities', async (route) => {
        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ data: [MOSCOW] }),
        });
    });

    await page.route('**/api/v1/categories?*', async (route) => {
        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ data: [ELECTRONICS, TRANSPORT] }),
        });
    });

    await page.route('**/api/v1/user/me', async (route) => {
        if (route.request().method() === 'PATCH') {
            const payload = route.request().postDataJSON();
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ ...user, ...payload }),
            });
            return;
        }

        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(user),
        });
    });

    await page.route('**/api/v1/auth/login', async (route) => {
        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ access_token: AUTH_TOKEN }),
        });
    });

    await page.route('**/api/v1/auth/register', async (route) => {
        await route.fulfill({
            status: 201,
            contentType: 'application/json',
            body: JSON.stringify({ message: 'registered' }),
        });
    });

    await page.route('**/api/v1/auth/verify-email', async (route) => {
        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ message: 'verified' }),
        });
    });

    await page.route('**/api/v1/auth/change-password', async (route) => {
        const payload = route.request().postDataJSON();
        if (payload.current_password === 'wrongpassword') {
            await route.fulfill({
                status: 401,
                contentType: 'application/json',
                body: JSON.stringify({ detail: 'Invalid current password' }),
            });
            return;
        }

        await route.fulfill({ status: 204 });
    });

    await page.route('**/api/v1/items', async (route) => {
        if (route.request().method() === 'POST') {
            const payload = route.request().postDataJSON();
            await route.fulfill({
                status: 201,
                contentType: 'application/json',
                body: JSON.stringify({ data: { ...DEMO_ITEMS[0], ...payload, id: 'ffffffff-ffff-4fff-8fff-ffffffffffff' } }),
            });
            return;
        }

        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
                data: items,
                pagination: { page: 1, per_page: 20, total: items.length, total_pages: 1 },
            }),
        });
    });

    await page.route('**/api/v1/items?*', async (route) => {
        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
                data: items,
                pagination: { page: 1, per_page: 20, total: items.length, total_pages: 1 },
            }),
        });
    });

    await page.route('**/api/v1/items/*', async (route) => {
        const url = new URL(route.request().url());
        const id = url.pathname.split('/').pop();
        const item = items.find((entry) => entry.id === id) || items[0];

        if (route.request().method() === 'PATCH') {
            const payload = route.request().postDataJSON();
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ data: { ...item, ...payload } }),
            });
            return;
        }

        if (route.request().method() === 'DELETE') {
            await route.fulfill({ status: 204 });
            return;
        }

        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ data: item }),
        });
    });

    await page.route('**/api/v1/items/favorites', async (route) => {
        if (route.request().method() === 'POST') {
            const payload = route.request().postDataJSON();
            const item = items.find((entry) => entry.id === payload.item_id) || items[0];
            favorites.splice(0, favorites.length, { id: 'fav-1', item_id: item.id, item: { ...item, is_favorited: true } });
            await route.fulfill({
                status: 201,
                contentType: 'application/json',
                body: JSON.stringify({ message: 'Added to favorites' }),
            });
            return;
        }

        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
                data: favorites,
                pagination: { page: 1, per_page: 20, total: favorites.length, total_pages: 1 },
            }),
        });
    });

    await page.route('**/api/v1/items/favorites/*', async (route) => {
        const id = route.request().url().split('/').pop();
        favorites.splice(0, favorites.length, ...favorites.filter((fav) => fav.item_id !== id));
        await route.fulfill({ status: 204 });
    });

    await page.route('**/api/v1/contact-messages', async (route) => {
        await route.fulfill({
            status: 201,
            contentType: 'application/json',
            body: JSON.stringify({ data: { id: 'contact-1' } }),
        });
    });
}

module.exports = {
    AUTH_TOKEN,
    TEST_USER,
    FIXTURE_USER_ROLE_USER_ONLY,
    API_URL,
    API_V1_URL,
    BASE_URL,
    MOSCOW,
    ELECTRONICS,
    TRANSPORT,
    DEMO_ITEMS,
    buildStorageState,
    loginAndSaveState,
    loginAndSaveStateAs,
    mockCommonApi,
};
