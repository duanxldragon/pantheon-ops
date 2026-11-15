# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\k8s\k8s-pages.spec.ts >> K8s business module smoke >> cluster list remains usable in a phone viewport
- Location: tests\smoke\business\k8s\k8s-pages.spec.ts:47:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.governance-summary-bar')
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('.governance-summary-bar') with timeout 10000ms
  - waiting for locator('.governance-summary-bar')

```

```yaml
- complementary:
    - text: P
    - menu
- button "切换导航"
- list:
    - listitem: 首页
    - listitem: K8s 集群
- button "平台功能台": 搜索菜单、页面、功能…
- button "平台偏好"
- 'button "用户: 用户"': 用
- tablist "已打开页面"
- main:
    - text: 加载失败 当前区块暂时无法返回数据，请重试或稍后再查看。
    - button "重试"
- text: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  1  | import { expect, test } from '@playwright/test';
  2  | import {
  3  |   adminCredentials,
  4  |   installClientSession,
  5  |   loginByApi,
  6  |   primeChineseLocale,
  7  | } from '../../helpers/auth';
  8  |
  9  | async function expectNoVisibleK8sI18nKeys(page: import('@playwright/test').Page) {
  10 |   const bodyText = await page.locator('body').innerText();
  11 |   expect(bodyText).not.toMatch(/\bbusiness\.k8s\.[A-Za-z0-9_.-]+\b/);
  12 |   expect(bodyText).not.toMatch(/\boperations\.k8s\.[A-Za-z0-9_.-]+\b/);
  13 | }
  14 |
  15 | test.describe('K8s business module smoke', () => {
  16 |   test.beforeEach(async ({ page }) => {
  17 |     const login = await loginByApi(page.request, adminCredentials);
  18 |     await installClientSession(page, login);
  19 |     await primeChineseLocale(page);
  20 |   });
  21 |
  22 |   test('cluster, release, and Kubernetes Service pages load on business routes', async ({ page }, testInfo) => {
  23 |     await page.setViewportSize({ width: 1440, height: 900 });
  24 |     await page.goto('/business/k8s/cluster', { waitUntil: 'networkidle' });
  25 |     await expect(page).toHaveURL(/\/business\/k8s\/cluster$/);
  26 |     await expect(page.locator('.governance-summary-bar')).toContainText('K8s 集群');
  27 |     await expect(page.locator('.system-list__table-card')).toBeVisible();
  28 |     await expectNoVisibleK8sI18nKeys(page);
  29 |     await page.screenshot({ path: testInfo.outputPath('k8s-cluster-desktop.png') });
  30 |
  31 |     await page.goto('/business/k8s/release', { waitUntil: 'networkidle' });
  32 |     await expect(page).toHaveURL(/\/business\/k8s\/release$/);
  33 |     await expect(page.locator('.governance-summary-bar')).toContainText('应用发布');
  34 |     await expect(page.locator('.system-list__table-card')).toBeVisible();
  35 |     await expectNoVisibleK8sI18nKeys(page);
  36 |     await page.screenshot({ path: testInfo.outputPath('k8s-release-desktop.png') });
  37 |
  38 |     await page.goto('/business/k8s/service', { waitUntil: 'domcontentloaded' });
  39 |     await expect(page).toHaveURL(/\/business\/k8s\/service$/);
  40 |     await expect(page.locator('.governance-summary-bar')).toContainText('服务');
  41 |     await expect(page.locator('.search-toolbar')).toBeVisible();
  42 |     await expect(page.locator('.system-list__table-card')).toBeVisible();
  43 |     await expectNoVisibleK8sI18nKeys(page);
  44 |     await page.screenshot({ path: testInfo.outputPath('k8s-service-desktop.png') });
  45 |   });
  46 |
  47 |   test('cluster list remains usable in a phone viewport', async ({ page }, testInfo) => {
  48 |     await page.setViewportSize({ width: 390, height: 844 });
  49 |     await page.goto('/business/k8s/cluster', { waitUntil: 'networkidle' });
> 50 |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
     |                                                           ^ Error: expect(locator).toBeVisible() failed
  51 |     await expect
  52 |       .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
  53 |       .toBe(true);
  54 |     await page.screenshot({ path: testInfo.outputPath('k8s-cluster-mobile.png') });
  55 |
  56 |     await page.goto('/business/k8s/service', { waitUntil: 'domcontentloaded' });
  57 |     await expect(page.locator('.governance-summary-bar')).toContainText('服务');
  58 |     await expect
  59 |       .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
  60 |       .toBe(true);
  61 |     await page.screenshot({ path: testInfo.outputPath('k8s-service-mobile.png') });
  62 |   });
  63 | });
  64 |
```
