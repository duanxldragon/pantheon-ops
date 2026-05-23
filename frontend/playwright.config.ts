import os from 'node:os';
import path from 'node:path';
import { defineConfig } from '@playwright/test';

const externalWebServer = process.env.PANTHEON_EXTERNAL_WEB_SERVER === '1';
const webBaseUrl = process.env.PANTHEON_WEB_BASE_URL ?? 'http://127.0.0.1:5173';
const outputDir = process.env.PANTHEON_PLAYWRIGHT_OUTPUT_DIR ?? path.join(os.tmpdir(), 'pantheon-playwright', 'default');

export default defineConfig({
  testDir: './tests/smoke',
  outputDir,
  timeout: 30_000,
  workers: process.env.CI ? 2 : 1,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: 'list',
  use: {
    baseURL: webBaseUrl,
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
