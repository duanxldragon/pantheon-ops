import { expect, test, type Page } from '@playwright/test';
import { primeChineseLocale } from '../smoke/helpers/auth';

async function openWorkbench(page: Page, colorMode: 'light' | 'dark') {
  await page.addInitScript((mode) => {
    globalThis.localStorage.setItem('pantheon_color_mode', mode);
  }, colorMode);
  await primeChineseLocale(page);
  await page.goto('/__visual/operational-workbench', { waitUntil: 'domcontentloaded' });
  await expect(page.getByTestId('operational-workbench-fixture')).toBeVisible();
  await page.evaluate(async () => document.fonts.ready);
}

test('B1-B4 operational workbench visual and interaction contract', async ({ page }) => {
  await openWorkbench(page, 'light');
  const fixture = page.getByTestId('operational-workbench-fixture');

  await expect(fixture.locator('.submit-bar[data-sticky="true"]')).toBeVisible();
  await fixture.getByRole('button', { name: '保存' }).click();
  await expect(fixture.getByRole('alert')).toBeVisible();
  await expect(fixture.getByRole('button', { name: '保存' })).toBeEnabled();

  await fixture.getByRole('button', { name: '表格视图' }).click();
  await expect(page.getByRole('menu')).toBeVisible();
  await page.getByRole('menuitem', { name: '紧凑', exact: true }).click();
  await expect(fixture.locator('.app-table-shell--density-compact')).toBeVisible();

  await expect(fixture.locator('.task-log-viewer__row')).toHaveCount(120);
  await expect(fixture.getByText('已遮蔽')).toHaveCount(2);
  await expect(fixture.locator('.context-selector__option')).toHaveCount(80);
  await expect(fixture.locator('.execution-step-rail__step')).toHaveCount(20);
  await expect(fixture.locator('[data-dashboard-slot]')).toHaveCount(6);
  await expect(page).toHaveScreenshot('operational-workbench.png', { fullPage: true });
});

test('@mobile B1-B4 operational workbench remains single-column', async ({ page }) => {
  await openWorkbench(page, 'light');
  await expect(page.locator('.operational-visual-fixture__grid')).toHaveCSS(
    'grid-template-columns',
    '358px',
  );
  await expect(page.locator('.submit-bar[data-sticky="true"]')).toBeVisible();
  await expect(page).toHaveScreenshot('operational-workbench-mobile.png', { fullPage: true });
});

test('@dark B1-B4 operational workbench respects dark theme', async ({ page }) => {
  await openWorkbench(page, 'dark');
  await expect(page.locator('html')).toHaveAttribute('data-color-mode', 'dark');
  await expect(page.locator('body')).toHaveAttribute('arco-theme', 'dark');
  await expect(page.locator('.operational-primitive').first()).toBeVisible();
  await expect(page).toHaveScreenshot('operational-workbench-dark.png', { fullPage: true });
});
