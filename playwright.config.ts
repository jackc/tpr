import { defineConfig, devices } from '@playwright/test';

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
  testDir: './e2e/tests',

  fullyParallel: false,

  forbidOnly: !!process.env.CI,

  retries: process.env.CI ? 2 : 0,

  workers: 1,

  reporter: 'html',

  use: {
    baseURL: 'http://127.0.0.1:5000',

    trace: 'on-first-retry',

    screenshot: 'only-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  webServer: [
    {
      command: 'npm run dev',
      port: 5173,
      reuseExistingServer: !process.env.CI,
      timeout: 120000,
    },
    {
      command: 'TEST_ENDPOINTS=true go run main.go server -c tpr.test.conf --static-url http://127.0.0.1:5173',
      port: 5000,
      reuseExistingServer: !process.env.CI,
      timeout: 120000,
    },
  ],
});
