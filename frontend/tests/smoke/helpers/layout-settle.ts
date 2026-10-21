import { expect, type Locator } from '@playwright/test';

/**
 * 读取元素渲染宽度（取整像素）。
 */
export function readElementWidth(locator: Locator): Promise<number> {
  return locator.evaluate((element) => Math.round(element.getBoundingClientRect().width));
}

/**
 * 等待元素布局与过渡动画结束，避免在动画中间采样。
 *
 * Arco 的 Sider 与 Dropdown 面板宽度都是 CSS 过渡动画（默认 200ms），
 * 点击后立即测量会读到动画中间值：
 * - platform-shell-critical 侧边栏折叠曾读到 248 vs 248（折叠尚未开始）
 *   与 215 vs 108（展开尚未结束），造成主干 Core Smoke 偶发失败；
 * - shell-top-panels 曾读到弹出面板仍在动画中的盒模型，使窄视口包含性断言偶发失败。
 *
 * 这里先等盒模型签名稳定，再等元素自身及子树的动画真正结束，然后调用方才测量。
 * 它只消除测量竞态，不放松任何断言阈值。
 */
export async function settleElementLayout(locator: Locator): Promise<void> {
  let previous = '';
  await expect
    .poll(
      async () => {
        const box = await locator.boundingBox();
        if (!box) return false;
        const signature = `${Math.round(box.x)}:${Math.round(box.y)}:${Math.round(box.width)}:${Math.round(box.height)}`;
        const stable = signature === previous;
        previous = signature;
        return stable;
      },
      { timeout: 5000, intervals: [100, 150, 200, 250] },
    )
    .toBe(true);
  await locator.evaluate(async (element) => {
    const animations = element.getAnimations({ subtree: true });
    await Promise.all(animations.map((animation) => animation.finished.catch(() => undefined)));
  });
}
