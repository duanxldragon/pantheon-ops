import { test } from '../../fixtures/coverage';
import { expect } from '@playwright/test';
import {
  expectNoPageError,
  expectPageBodyReady,
  expectPagePathname,
  expectVisiblePageTitle,
  isIgnorableConsoleError,
  signInAsAdmin,
  systemPages,
  workspacePages,
} from './system-pages-helpers';

test.beforeEach(async ({ page }) => {
  await signInAsAdmin(page);
});

for (const pageMeta of systemPages) {
  test(`system smoke: ${pageMeta.path}`, async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', (message) => {
      if (message.type() === 'error' && !isIgnorableConsoleError(message.text())) {
        consoleErrors.push(message.text());
      }
    });

    await page.goto(pageMeta.path, { waitUntil: 'domcontentloaded' });
    expectPagePathname(page, pageMeta.path);
    await expectVisiblePageTitle(page, pageMeta.title);
    await expectNoPageError(page);
    await expectPageBodyReady(page);
    expect(consoleErrors).toEqual([]);
  });
}

for (const pageMeta of workspacePages) {
  test(`workspace smoke: ${pageMeta.path} is reachable`, async ({ page }) => {
    await page.goto(pageMeta.path, { waitUntil: 'domcontentloaded' });
    expectPagePathname(page, pageMeta.path);
    if ('title' in pageMeta && pageMeta.title) {
      await expectVisiblePageTitle(page, pageMeta.title);
    }
    await expectNoPageError(page);
    await pageMeta.assertReady(page);
  });
}
