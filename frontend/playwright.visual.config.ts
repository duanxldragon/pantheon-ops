import { defineConfig } from '@playwright/test';

const externalWebServer = process.env.PANTHEON_EXTERNAL_WEB_SERVER === '1';
const webBaseUrl = process.env.PANTHEON_WEB_BASE_URL ?? 'http://127.0.0.1:5173';

export default defineConfig({
  testDir: './tests/visual',
  timeout: 30_000,
  workers: 1,
  expect: {
    timeout: 10_000,
    toHaveScreenshot: {
      animations: 'disabled',
      maxDiffPixelRatio: 0.01,
    },
  },
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  reporter: 'list',
  use: {
    baseURL: webBaseUrl,
    viewport: {
      width: 1440,
      height: 900,
    },
    trace: 'retain-on-failure',
  },
  ...(externalWebServer
    ? {}
    : {
        webServer: {
          command: 'node scripts/start-smoke-vite.mjs --host 127.0.0.1 --port 5173',
          url: webBaseUrl,
          reuseExistingServer: true,
          timeout: 30_000,
        },
      }),
});
