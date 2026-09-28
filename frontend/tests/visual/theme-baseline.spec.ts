import { expect, test, type Page } from '@playwright/test';
import { primeChineseLocale } from '../smoke/helpers/auth';

/**
 * Four-theme screenshot baseline (fix-report §4 item 4).
 *
 * The 2026-07-15 CSS token fixes (list-page.css shadows, ModuleWizard.css
 * surfaces) were verified per-theme by reading only; this spec makes the
 * multi-theme correctness checkable: each theme is primed through the same
 * localStorage key `pantheon_theme` the app reads at boot (src/core/theme/
 * theme.ts), asserted via the `data-pantheon-theme` attribute, then captured.
 *
 * The login page is used because it renders without any backend session, so the
 * baseline run only needs the vite dev server (same constraint as the existing
 * login baselines). Snapshots live in tests/visual/theme-baseline.spec.ts-snapshots/
 * and include the {platform} segment (win32 on the authoring host).
 */

async function primeTheme(page: Page, theme: string) {
  await page.addInitScript((value) => {
    try {
      globalThis.localStorage?.setItem('pantheon_theme', value);
    } catch {
      // about:blank and other opaque origins do not expose storage.
    }
  }, theme);
}

async function waitForVisualStability(page: Page) {
  await page.waitForLoadState('domcontentloaded');
  await page.evaluate(async () => {
    await document.fonts.ready;
  });
}

const themeKeys = ['indigo', 'emerald', 'violet', 'slate'] as const;

for (const theme of themeKeys) {
  test(`login page visual baseline @theme:${theme}`, async ({ page }) => {
    await primeTheme(page, theme);
    await primeChineseLocale(page);
    await page.goto('/login', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.auth-login-page')).toBeVisible();
    await expect(page.locator('html')).toHaveAttribute('data-pantheon-theme', theme);
    await waitForVisualStability(page);

    await expect(page).toHaveScreenshot(`login-theme-${theme}.png`);
  });
}
