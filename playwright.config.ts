import { defineConfig, devices } from '@playwright/test';

// Reserved test ports keep browser servers separate from the development servers.
const backendPort = process.env.TEST_BACKEND_PORT || '5000';
const vitePort = process.env.TEST_VITE_PORT || '5174';
const backendURL = process.env.TEST_BACKEND_URL || `http://127.0.0.1:${backendPort}`;
const baseURL = process.env.TEST_VITE_URL || `http://127.0.0.1:${vitePort}`;
const databaseURL = process.env.TEST_DATABASE_URL || 'postgres://tpr:secret@127.0.0.1:5432/tpr_test';

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
  testDir: './e2e/tests',

  fullyParallel: false,

  forbidOnly: !!process.env.CI,

  retries: process.env.CI ? 2 : 0,

  workers: 1,

  reporter: [['html', { open: 'never', host: '127.0.0.1', port: Number(process.env.PLAYWRIGHT_REPORT_PORT || '9323') }]],

  use: {
    baseURL,

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
      env: { VITE_PORT: vitePort, BACKEND_URL: backendURL },
      url: baseURL,
      reuseExistingServer: false,
      timeout: 120000,
      stdout: 'pipe',
      stderr: 'pipe',
    },
    {
      command: 'go run . server',
      env: { TEST_ENDPOINTS: 'true', DATABASE_URL: databaseURL, LOG_LEVEL: 'none', BIND_ADDRESS: '127.0.0.1', PORT: backendPort },
      url: `${backendURL}/health`,
      reuseExistingServer: false,
      timeout: 120000,
      stdout: 'pipe',
      stderr: 'pipe',
    },
  ],
});
