import os from 'node:os';
import path from 'node:path';
import { defineConfig } from '@playwright/test';

const outputDir = process.env.PANTHEON_PLAYWRIGHT_OUTPUT_DIR ?? path.join(os.tmpdir(), 'pantheon-playwright', 'api');

export default defineConfig({
  testDir: './tests/smoke',
  outputDir,
  timeout: 30_000,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: 'list',
  workers: 1,
});
