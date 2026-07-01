// @ts-check
const { test, expect } = require('@playwright/test');
const path = require('path');
const { BASE_URL, TEST_USER, mockCommonApi } = require('./helpers');

const AUTH_STATE = path.join(__dirname, '.auth.json');

test.describe('chat ui', () => {
    test.use({ storageState: AUTH_STATE });

    test('opens conversations and sends a message', async ({ page }) => {
        const conversationID = '11111111-1111-4111-8111-111111111111';
        const peerID = '22222222-2222-4222-8222-222222222222';
        let sentPayload = null;

        await mockCommonApi(page);

        await page.route('**/api/v1/chat/conversations', async (route) => {
            if (route.request().method() !== 'GET') {
                await route.fulfill({
                    status: 201,
                    contentType: 'application/json',
                    body: JSON.stringify({ id: conversationID, peer_id: peerID, peer_name: 'Seller User' }),
                });
                return;
            }

            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify([
                    {
                        id: conversationID,
                        peer_id: peerID,
                        peer_name: 'Seller User',
                        last_message_preview: 'Hello',
                        last_message_at: '2026-06-21T10:00:00Z',
                        unread_count: 2,
                    },
                ]),
            });
        });

        await page.route(`**/api/v1/chat/conversations/${conversationID}/messages?limit=50`, async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify(Array.from({ length: 5 }, (_, index) => ({
                    id: `33333333-3333-4333-8333-${String(index).padStart(12, '0')}`,
                    sender_id: peerID,
                    body: index === 4 ? 'Hello' : `Earlier message ${index + 1}`,
                    created_at: `2026-06-21T10:${String(index).padStart(2, '0')}:00Z`,
                }))),
            });
        });

        await page.route(`**/api/v1/chat/conversations/${conversationID}/read`, async (route) => {
            await route.fulfill({ status: 204 });
        });

        await page.route(`**/api/v1/chat/conversations/${conversationID}/messages`, async (route) => {
            if (route.request().method() !== 'POST') {
                await route.fallback();
                return;
            }

            sentPayload = route.request().postDataJSON();
            await route.fulfill({
                status: 201,
                contentType: 'application/json',
                body: JSON.stringify({
                    id: '44444444-4444-4444-8444-444444444444',
                    sender_id: TEST_USER.id,
                    body: sentPayload.body,
                    created_at: '2026-06-21T10:05:00Z',
                }),
            });
        });

        await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
        await page.locator('[data-testid="nav-chat"]').click();

        await expect(page.locator('[data-testid="chat-section"]')).toBeVisible();
        await expect(page.locator('[data-testid="chat-conversation"]')).toContainText('Seller User');
        await expect(page.locator('[data-testid="chat-message"]').last()).toContainText('Hello');

        await page.locator('[data-testid="chat-message-input"]').fill('Hi there');
        await page.locator('[data-testid="chat-send"]').click();

        await expect(page.locator('[data-testid="chat-message"]').last()).toContainText('Hi there');
        expect(sentPayload).toEqual({ body: 'Hi there' });
    });
});
