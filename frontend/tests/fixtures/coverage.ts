// Playwright page fixture: collects window.__coverage__ after each test.
// vite-plugin-istanbul instruments the source during Vite dev serving and
// populates window.__coverage__ as pages load. This fixture extracts that
// data so it can be merged into an lcov report after the smoke run.
/* eslint-disable react-hooks/rules-of-hooks */
import { test as base } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import * as fs from 'node:fs';
import * as path from 'node:path';

const COVERAGE_DIR = path.resolve('coverage/.tmp');
const COLLECT = process.env.PANTHEON_COLLECT_COVERAGE === '1';

export const test = base.extend({
  page: async ({ page }, use) => {
    await use(page);
    if (!COLLECT) return;
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const coverage = await page.evaluate(() => (window as any).__coverage__);
      if (coverage) {
        fs.mkdirSync(COVERAGE_DIR, { recursive: true });
        const key = `coverage-${Date.now()}-${randomUUID().slice(0, 8)}`;
        fs.writeFileSync(path.join(COVERAGE_DIR, `${key}.json`), JSON.stringify(coverage));
      }
    } catch {
      // Page may have been closed before evaluation — safe to ignore.
    }
  },
});
