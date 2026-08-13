import { test } from '../../fixtures/coverage';
import { expect, type Page } from '@playwright/test';
import { signInAsAdmin } from '../helpers/auth';

const pageErrorTexts = [
  '加载失败',
  '网络异常',
  '请求超时',
  'Load failed',
  'Network error',
  'Request timed out',
];

const desktopAuditPages = [
  '/system/user',
  '/system/role',
  '/system/menu',
  '/system/dept',
  '/system/post',
  '/system/permission',
  '/system/dict',
  '/system/i18n',
  '/system/login-log',
  '/system/session',
  '/system/security-event',
  '/system/operation-log',
] as const;

const layoutViewportCases = [
  { width: 1440, height: 900, minFilterControlWidth: 160 },
  { width: 1280, height: 900, minFilterControlWidth: 140 },
] as const;

async function expectNoPageError(page: Page) {
  for (const text of pageErrorTexts) {
    await expect(page.getByText(text, { exact: false })).toHaveCount(0);
  }
}

test.describe('system layout contract', () => {
  for (const viewport of layoutViewportCases) {
    test(`list pages keep controls readable at ${viewport.width}w`, async ({ page }) => {
      await signInAsAdmin(page);
      await page.setViewportSize({ width: viewport.width, height: viewport.height });

      for (const target of desktopAuditPages) {
        await page.goto(target, { waitUntil: 'domcontentloaded' });
        await expect(page.locator('.app-shell__content')).toBeVisible();
        await expectNoPageError(page);

        const metrics = await page.evaluate(() => {
          const visible = (node: Element) => {
            const rect = node.getBoundingClientRect();
            const style = globalThis.getComputedStyle(node);
            return (
              rect.width > 0 &&
              rect.height > 0 &&
              style.visibility !== 'hidden' &&
              style.display !== 'none'
            );
          };

          const controlSelector =
            '.filter-panel .arco-input-inner-wrapper, .filter-panel input.arco-input, .filter-panel .arco-input-password, .filter-panel .arco-input-number, .filter-panel .arco-select-view, .filter-panel .arco-tree-select-view, .filter-panel .arco-picker, .filter-panel .arco-textarea-wrapper';
          const controlWrappers = Array.from(document.querySelectorAll(controlSelector))
            .map((node) =>
              node.closest(
                '.arco-input-inner-wrapper, .arco-input-password, .arco-input-number, .arco-select-view, .arco-tree-select-view, .arco-picker, .arco-textarea-wrapper',
              ) || node,
            )
            .filter((node, index, nodes) => nodes.indexOf(node) === index)
            .filter(visible);
          const controlWidths = controlWrappers.map((node) =>
            Math.round(node.getBoundingClientRect().width),
          );

          const actionBlocks = Array.from(document.querySelectorAll('.system-list__actions'))
            .filter(visible)
            .map((node) => ({
              text: node.textContent?.replace(/\s+/g, ' ').trim() || '',
              height: Math.round(node.getBoundingClientRect().height),
            }));

          const wrappedActionButtons = Array.from(
            document.querySelectorAll('.system-list__actions .arco-btn-text'),
          )
            .filter(visible)
            .map((node) => ({
              text: node.textContent?.replace(/\s+/g, ' ').trim() || '',
              height: Math.round(node.getBoundingClientRect().height),
            }))
            .filter((node) => node.height > 32);

          return {
            controlWidths,
            actionBlocks,
            wrappedActionButtons,
          };
        });

        if (metrics.controlWidths.length > 0) {
          const minControlWidth = Math.min(...metrics.controlWidths);
          expect(
            minControlWidth,
            `${target} has an undersized filter control at ${viewport.width}px: ${minControlWidth}px`,
          ).toBeGreaterThanOrEqual(viewport.minFilterControlWidth);
        }

        const tallActionBlock = metrics.actionBlocks.find((item) => item.height > 36);
        expect(
          tallActionBlock,
          tallActionBlock
            ? `${target} has wrapped row actions at ${viewport.width}px: "${tallActionBlock.text}" (${tallActionBlock.height}px)`
            : '',
        ).toBeUndefined();

        expect(
          metrics.wrappedActionButtons,
          `${target} has wrapped row-action buttons at ${viewport.width}px`,
        ).toEqual([]);
      }
    });
  }
});
