// @ts-check
const { defineConfig, devices } = require('@playwright/test');
const path = require('path');

const AUTH_STATE = path.join(__dirname, 'tests/.auth.json');

module.exports = defineConfig({
    testDir: './tests',
    timeout: 20_000,
    retries: 0,
    workers: 1,
    globalSetup: './tests/global-setup.js',
    reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
    webServer: {
        command: 'npm run build && npx vite preview --host 127.0.0.1 --port 4173',
        url: 'http://127.0.0.1:4173',
        reuseExistingServer: !process.env.CI,
        timeout: 60_000,
    },

    use: {
        baseURL: 'http://127.0.0.1:4173',
        headless: true,
        trace: 'retain-on-failure',
        screenshot: 'only-on-failure',
    },

    projects: [
        {
            name: 'chromium',
            use: {
                ...devices['Desktop Chrome'],
                storageState: AUTH_STATE,
            },
        },
    ],
});
