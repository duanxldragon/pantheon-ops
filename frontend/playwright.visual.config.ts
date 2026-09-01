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
  snapshotPathTemplate: '{testDir}/{testFilePath}-snapshots/{arg}-{platform}{ext}',
  use: {
    baseURL: webBaseUrl,
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'desktop-light',
      grepInvert: /@(mobile|dark)/,
      use: {
        colorScheme: 'light',
        viewport: { width: 1440, height: 900 },
      },
    },
    {
      name: 'mobile-light',
      grep: /@mobile/,
      use: {
        colorScheme: 'light',
        isMobile: true,
        viewport: { width: 390, height: 844 },
      },
    },
    {
      name: 'desktop-dark',
      grep: /@dark/,
      use: {
        colorScheme: 'dark',
        viewport: { width: 1440, height: 900 },
      },
    },
  ],
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
