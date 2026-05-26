import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type Page } from '@playwright/test';
import { signInAsAdmin } from '../../helpers/auth';
import { expectPagePathname } from '../../helpers/url-pattern';

const mainContentSelectors = [
  'main',
  'main .app-shell__content-inner',
  'main .system-page-template',
  'main .system-list__table-card',
  'main .filter-panel',
  'main .permission-workbench__tabs',
  'main .page-panel',
  'main .i18n-list-page',
];

const pageIdentitySelectors = [
  '.governance-summary-bar',
  '.system-list__table-card',
  '.permission-workbench__tabs',
  '.dict-workbench',
  '.setting-group-page',
  '.module-manager-page',
  '.generator-wizard-card',
  '.dashboard-hero-card',
  '.auth-security-page',
];

async function navigateInShell(page: Page, path: string) {
  if (page.url() === 'about:blank') {
    await page.goto('/dashboard', { waitUntil: 'networkidle' });
  }
  await page.evaluate((nextPath) => {
    window.history.pushState({}, '', nextPath);
    window.dispatchEvent(new PopStateEvent('popstate'));
  }, path);
  expectPagePathname(page, path);
}

async function expectPageIdentityReady(page: Page, title: string | RegExp) {
  await expect(page.getByText(title, { exact: false }).filter({ visible: true }).first()).toBeVisible();
  await expect(page.locator(pageIdentitySelectors.join(', ')).first()).toBeVisible();
}

async function measureMainContentWidth(page: Page) {
  return page.evaluate((selectors) => {
    const isVisible = (element: Element) => {
      const style = window.getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0;
    };
    const widths = selectors.flatMap((selector) =>
      Array.from(document.querySelectorAll<HTMLElement>(selector))
        .filter(isVisible)
        .map((element) => Math.round(element.getBoundingClientRect().width)),
    );
    return widths.length ? Math.max(...widths) : 0;
  }, mainContentSelectors);
}

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const admissionConfig = JSON.parse(
  fs.readFileSync(path.resolve(__dirname, '../../../../config/system-page-admission.json'), 'utf8'),
) as Array<{
  path: string;
  title: string;
  governanceDrawer: 'allowed' | 'forbidden';
  governanceButtonText?: string;
  governanceDrawerTitle?: string;
}>;

const governancePages = admissionConfig
  .filter((item) => item.governanceDrawer === 'allowed')
  .map((item) => ({
    path: item.path,
    title: item.title,
    button: item.governanceButtonText || '',
    drawerTitle: item.governanceDrawerTitle || '',
  }));

test('governance insight opens as drawer without compressing main pages', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);

  for (const item of governancePages) {
    await navigateInShell(page, item.path);
    await expectPageIdentityReady(page, item.title);
    await expect(page.locator('.page-split-layout')).toHaveCount(0);
    const mainWidthBefore = await measureMainContentWidth(page);
    expect(mainWidthBefore, item.path).toBeGreaterThan(700);

    await page.getByRole('button', { name: item.button }).first().click();

    const drawer = page.locator('.governance-insight-drawer');
    await expect(drawer).toBeVisible();
    await expect(drawer.getByText(item.drawerTitle, { exact: true }).first()).toBeVisible();
    await expect(page.locator('.page-split-layout')).toHaveCount(0);
    const mainWidthAfter = await measureMainContentWidth(page);
    expect(mainWidthAfter, item.path).toBeGreaterThanOrEqual(mainWidthBefore - 4);
  }
});
