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

  reporter: [['html', { open: 'never' }]],

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
      stdout: 'pipe',
      stderr: 'pipe',
    },
    {
      command: 'TEST_ENDPOINTS=true TPR_DB_HOST=127.0.0.1 TPR_DB_NAME=tpr_test TPR_DB_USER=tpr TPR_DB_PASSWORD=secret TPR_LOG_LEVEL=none go run main.go server --port 5000 --static-url http://127.0.0.1:5173',
      port: 5000,
      reuseExistingServer: !process.env.CI,
      timeout: 120000,
      stdout: 'pipe',
      stderr: 'pipe',
    },
  ],
});
