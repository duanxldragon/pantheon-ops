import { expect, test } from '@playwright/test';
import { primeChineseLocale } from '../smoke/helpers/auth';
for (const mode of ['light', 'dark'] as const) {
 test(`shared condition editing and themed step buttons ${mode}`, async ({ page }, testInfo) => {
  await primeChineseLocale(page);
  await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  await page.goto('/__visual/operational-workbench?editor=1');
  const editor = page.locator('.condition-builder');
  await expect(editor).toBeVisible();
  const inputs = editor.locator('input[aria-label="值"]');
  await expect(inputs.first()).toBeEnabled();
  await inputs.first().fill('edited-condition');
  await expect(inputs.first()).toHaveValue('edited-condition');
  const initialRules = await editor.locator('input[aria-label="值"]').count();
  await editor.getByRole('button', { name: '添加条件' }).first().click();
  await expect(editor.locator('input[aria-label="值"]')).toHaveCount(initialRules + 1);
  await editor.getByRole('button', { name: '移除条件', exact: true }).last().click();
  await expect(editor.locator('input[aria-label="值"]')).toHaveCount(initialRules);
  const steps = page.locator('.execution-step-rail');
  await steps.getByRole('button').first().focus();
  await expect(steps.getByRole('button').first()).toBeFocused();
  if (mode === 'dark') expect(await steps.getByRole('button').first().evaluate((button) => getComputedStyle(button).backgroundColor)).not.toBe('rgb(255, 255, 255)');
  for (const [viewport, width, height] of [['desktop', 1440, 900], ['mobile', 390, 844]] as const) {
   await page.setViewportSize({ width, height });
   await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
   await editor.screenshot({ path: testInfo.outputPath(`conditions-${mode}-${viewport}.png`) });
   await steps.screenshot({ path: testInfo.outputPath(`steps-${mode}-${viewport}.png`) });
  }
 });
}
