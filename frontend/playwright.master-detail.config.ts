import os from 'node:os';
import path from 'node:path';
import { defineConfig } from '@playwright/test';

const externalWebServer = process.env.PANTHEON_EXTERNAL_WEB_SERVER === '1';
const webBaseUrl = process.env.PANTHEON_WEB_BASE_URL ?? 'http://127.0.0.1:5174';
const outputDir = process.env.PANTHEON_PLAYWRIGHT_OUTPUT_DIR ?? path.join(os.tmpdir(), 'pantheon-playwright', 'master-detail');

export default defineConfig({
  testDir: './tests/smoke',
  outputDir,
  timeout: 120_000,
  workers: 1,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: false,
  reporter: 'list',
  use: {
    baseURL: webBaseUrl,
    trace: 'retain-on-failure',
  },
  ...(externalWebServer
    ? {}
    : {
        webServer: {
          command:
            'node scripts/start-smoke-vite.mjs --host 127.0.0.1 --port 5174 --setup scripts/master-detail-qa-setup.mjs',
          url: webBaseUrl,
          reuseExistingServer: false,
          timeout: 60_000,
        },
      }),
});
