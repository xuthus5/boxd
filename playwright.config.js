import { defineConfig, devices } from '@playwright/test';

const port = Number(process.env.BOXD_E2E_PORT || 17662);
const baseURL = `http://127.0.0.1:${port}`;
const authState = 'test-results/.auth/admin.json';

export default defineConfig({
  testDir: './e2e/tests',
  outputDir: './test-results/artifacts',
  globalSetup: './e2e/global-setup.js',
  fullyParallel: false,
  workers: 1,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  timeout: 45_000,
  expect: {
    timeout: 10_000,
  },
  reporter: process.env.CI
    ? [['line'], ['html', { open: 'never', outputFolder: 'playwright-report' }]]
    : [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
  use: {
    baseURL,
    locale: 'zh-CN',
    timezoneId: 'Asia/Shanghai',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'desktop-chromium',
      use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 }, storageState: authState },
    },
    {
      name: 'desktop-firefox',
      use: { ...devices['Desktop Firefox'], viewport: { width: 1440, height: 900 }, storageState: authState },
    },
    {
      name: 'desktop-webkit',
      use: { ...devices['Desktop Safari'], viewport: { width: 1440, height: 900 }, storageState: authState },
    },
    {
      name: 'mobile-chromium',
      use: { ...devices['Pixel 7'], storageState: authState },
    },
  ],
  webServer: {
    command: `BOXD_E2E_PORT=${port} bash e2e/start-server.sh`,
    url: `${baseURL}/api/v1/health`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
