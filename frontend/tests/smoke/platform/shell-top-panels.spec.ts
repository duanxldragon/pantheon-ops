import { test } from '../../fixtures/coverage';
import { expect } from '@playwright/test';
import { signInAsAdmin } from '../helpers/auth';

async function readWidth(locator: import('@playwright/test').Locator) {
  return locator.evaluate((element) => Math.round(element.getBoundingClientRect().width));
}

type Box = {
  x: number;
  y: number;
  width: number;
  height: number;
};

async function readBox(locator: import('@playwright/test').Locator): Promise<Box> {
  const box = await locator.boundingBox();
  expect(box).not.toBeNull();
  return box as Box;
}

async function expectNoViewportOverflow(page: import('@playwright/test').Page) {
  await expect
    .poll(async () =>
      page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      ),
    )
    .toBeLessThanOrEqual(1);
}

function expectBoxInsideViewport(page: import('@playwright/test').Page, box: Box) {
  const viewport = page.viewportSize();
  expect(viewport).not.toBeNull();
  expect(box.x).toBeGreaterThanOrEqual(-1);
  expect(box.y).toBeGreaterThanOrEqual(-1);
  expect(box.x + box.width).toBeLessThanOrEqual(viewport!.width + 1);
  expect(box.y + box.height).toBeLessThanOrEqual(viewport!.height + 1);
}

test('shell top panels keep readable desktop widths', async ({ page }, testInfo) => {
  await signInAsAdmin(page);
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/dashboard', { waitUntil: 'networkidle' });

  const noticeTrigger = page.getByRole('button', { name: /通知中心|Notification Center/ });
  await expect(noticeTrigger).toBeVisible();
  await noticeTrigger.click();

  const noticePanel = page.locator('.app-shell__notice-panel');
  await expect(noticePanel).toBeVisible();
  await expect(await readWidth(noticePanel)).toBeGreaterThanOrEqual(380);
  await page.screenshot({ path: testInfo.outputPath('shell-top-panels-desktop-notice.png') });

  await page.mouse.click(24, 24);
  await expect(noticePanel).toHaveCount(0);

  const preferenceTrigger = page.getByRole('button', { name: /平台偏好|Platform Preferences/ });
  await expect(preferenceTrigger).toBeVisible();
  await preferenceTrigger.click();

  const preferencePanel = page.locator('.app-shell__preference-panel');
  await expect(preferencePanel).toBeVisible();
  await expect(await readWidth(preferencePanel)).toBeGreaterThanOrEqual(380);
});

test('shell top panels and profile page stay contained on narrow viewports', async ({
  page,
}, testInfo) => {
  await signInAsAdmin(page);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/dashboard', { waitUntil: 'networkidle' });

  const searchTrigger = page.locator('.app-shell__search-trigger');
  await expect(searchTrigger).toBeVisible();
  const searchBox = await readBox(searchTrigger);

  const noticeTrigger = page.getByRole('button', { name: /通知中心|Notification Center/ });
  await expect(noticeTrigger).toBeVisible();
  await noticeTrigger.click();

  const noticePanel = page.locator('.app-shell__notice-panel');
  await expect(noticePanel).toBeVisible();
  const noticeBox = await readBox(noticePanel);
  expect(noticeBox.y).toBeGreaterThanOrEqual(searchBox.y + searchBox.height - 1);
  expectBoxInsideViewport(page, noticeBox);
  await expect(await readWidth(noticePanel)).toBeLessThanOrEqual(358);
  await page.screenshot({ path: testInfo.outputPath('shell-top-panels-narrow-notice.png') });
  await expectNoViewportOverflow(page);

  await noticeTrigger.click();
  await expect(noticePanel).toHaveCount(0);
  const preferenceTrigger = page.getByRole('button', { name: /平台偏好|Platform Preferences/ });
  await preferenceTrigger.click();
  const preferencePanel = page.locator('.app-shell__preference-panel');
  await expect(preferencePanel).toBeVisible();
  expectBoxInsideViewport(page, await readBox(preferencePanel));
  await expect(await readWidth(preferencePanel)).toBeLessThanOrEqual(358);
  await expectNoViewportOverflow(page);

  await page.goto('/system/profile', { waitUntil: 'networkidle' });
  await expect(page.locator('.submit-bar')).toBeVisible();
  await expect(page.locator('.arco-form')).toBeVisible();
  await expectNoViewportOverflow(page);
});
