import { expect, test, type Page, type TestInfo } from '@playwright/test';
import type { K8sServiceRow } from '../../../../src/modules/business/k8s/api';
import {
  adminCredentials,
  installClientSession,
  loginByApi,
  primeChineseLocale,
} from '../../helpers/auth';

async function expectNoVisibleK8sI18nKeys(page: import('@playwright/test').Page) {
  const bodyText = await page.locator('body').innerText();
  expect(bodyText).not.toMatch(/\bbusiness\.k8s\.[A-Za-z0-9_.-]+\b/);
  expect(bodyText).not.toMatch(/\boperations\.k8s\.[A-Za-z0-9_.-]+\b/);
}

test.describe('K8s business module smoke', () => {
  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
  });

  test('cluster, release, and Kubernetes Service pages load on business routes', async ({
    page,
  }, testInfo) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/business/k8s/cluster', { waitUntil: 'networkidle' });
    await expect(page).toHaveURL(/\/business\/k8s\/cluster$/);
    await expect(page.locator('.governance-summary-bar')).toContainText('K8s 集群');
    await expect(page.locator('.system-list__table-card')).toBeVisible();
    await expectNoVisibleK8sI18nKeys(page);
    await page.screenshot({ path: testInfo.outputPath('k8s-cluster-desktop.png') });

    await page.goto('/business/k8s/release', { waitUntil: 'networkidle' });
    await expect(page).toHaveURL(/\/business\/k8s\/release$/);
    await expect(page.locator('.governance-summary-bar')).toContainText('应用发布');
    await expect(page.locator('.system-list__table-card')).toBeVisible();
    await expectNoVisibleK8sI18nKeys(page);
    await page.screenshot({ path: testInfo.outputPath('k8s-release-desktop.png') });

    await page.goto('/business/k8s/service', { waitUntil: 'domcontentloaded' });
    await expect(page).toHaveURL(/\/business\/k8s\/service$/);
    await expect(page.locator('.governance-summary-bar')).toContainText('服务');
    await expect(page.locator('.search-toolbar')).toBeVisible();
    await expect(page.locator('.system-list__table-card')).toBeVisible();
    await expectNoVisibleK8sI18nKeys(page);
    await page.screenshot({ path: testInfo.outputPath('k8s-service-desktop.png') });
  });

  test('cluster list remains usable in a phone viewport', async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/business/k8s/cluster', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect
      .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
      .toBe(true);
    await page.screenshot({ path: testInfo.outputPath('k8s-cluster-mobile.png') });

    await page.goto('/business/k8s/service', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.governance-summary-bar')).toContainText('服务');
    await expect
      .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
      .toBe(true);
    await page.screenshot({ path: testInfo.outputPath('k8s-service-mobile.png') });
  });
});

// Business-data fixtures exercise the UI/HTTP contract. Real login remains in
// use; provider admission and live role acceptance belong to the runtime lane.
async function installServiceFixture(
  page: Page,
  options: {
    stale?: boolean;
    empty?: boolean;
    missingIdentity?: boolean;
    selectorless?: boolean;
    listGate?: Promise<void>;
    updateGate?: Promise<void>;
  } = {},
) {
  const service: K8sServiceRow = {
    name: 'c04-routing',
    namespace: 'prod',
    type: 'NodePort',
    clusterIP: '10.96.0.42',
    uid: options.missingIdentity ? '' : 'c04-uid-original',
    resourceVersion: '27',
    selector: options.selectorless ? null : { app: 'web', tier: '' },
    ports: [
      { name: 'http', port: 80, targetPort: 'http-web', nodePort: 30080, protocol: 'TCP' },
      { name: 'metrics', port: 9090, targetPort: 9091, nodePort: 30090, protocol: 'TCP' },
    ],
  };
  let rows = options.empty ? [] : [service];
  const writes: Array<{ method: string; url: URL; body?: Record<string, unknown> }> = [];
  const reads: URL[] = [];
  await page.route(/\/api\/v1\/business\/k8s\/clusters(?:\?.*)?$/, (route) =>
    route.fulfill({
      json: { code: 200, data: { items: [{ id: 41, name: 'c04-cluster' }], total: 1 } },
    }),
  );
  await page.route(
    /\/api\/v1\/business\/k8s\/clusters\/41\/services(?:\/[^?]+)?(?:\?.*)?$/,
    async (route) => {
      const request = route.request();
      const url = new URL(request.url());
      if (request.method() === 'GET') {
        reads.push(url);
        if (options.listGate) await options.listGate;
        await route.fulfill({ json: { code: 200, data: { items: rows, total: rows.length } } });
        return;
      }
      const body =
        request.method() === 'DELETE'
          ? undefined
          : (request.postDataJSON() as Record<string, unknown>);
      writes.push({ method: request.method(), url, body });
      if (request.method() === 'PUT' && options.updateGate) await options.updateGate;
      if (options.stale) {
        await route.fulfill({
          status: 409,
          json: { code: 409, message: 'k8s.service.resource_version_conflict' },
        });
        return;
      }
      if (request.method() === 'DELETE') rows = [];
      else if (request.method() === 'PUT') Object.assign(service, body, { resourceVersion: '28' });
      else if (request.method() === 'POST') rows.push({ ...service, ...body } as K8sServiceRow);
      await route.fulfill({
        json: { code: 200, data: request.method() === 'DELETE' ? null : service },
      });
    },
  );
  return { service, reads, writes };
}

async function openServiceEditor(page: Page) {
  await page.goto('/business/k8s/service', { waitUntil: 'domcontentloaded' });
  const row = page.locator('tbody tr').filter({ hasText: 'c04-routing' });
  await row.getByRole('button', { name: '编辑', exact: true }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  return dialog;
}

async function captureServiceState(page: Page, testInfo: TestInfo, name: string) {
  await expect
    .poll(() => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
    .toBe(true);
  await expectNoVisibleK8sI18nKeys(page);
  expect(await page.locator('body').innerText()).not.toMatch(
    /\bk8s\.(service|namespace)\.[A-Za-z0-9_.-]+\b/,
  );
  const path = testInfo.outputPath(`${name}.png`);
  await page.screenshot({ path, animations: 'disabled' });
  await testInfo.attach(name, { path, contentType: 'image/png' });
}

async function expectReadableServiceNotice(page: Page, testInfo: TestInfo, name: string) {
  const notice = page
    .locator('.arco-message')
    .filter({ hasText: '服务已变化，请刷新列表并重新核对后操作。' });
  await expect(notice).toHaveCount(1);
  await expect(page.locator('.arco-message-error')).toHaveCount(1);
  await expect(notice).toBeVisible();
  const measure = () =>
    notice.evaluate((node) => {
      const rect = node.getBoundingClientRect();
      const inset = Math.min(4, rect.width / 4, rect.height / 4);
      const points = [
        [rect.left + rect.width / 2, rect.top + rect.height / 2],
        [rect.left + inset, rect.top + inset],
        [rect.right - inset, rect.top + inset],
        [rect.left + inset, rect.bottom - inset],
        [rect.right - inset, rect.bottom - inset],
      ];
      const ancestors: Element[] = [];
      for (let current: Element | null = node; current; current = current.parentElement)
        ancestors.push(current);
      return {
        rect: {
          left: rect.left,
          top: rect.top,
          right: rect.right,
          bottom: rect.bottom,
          width: rect.width,
          height: rect.height,
        },
        insideViewport:
          rect.width > 0 &&
          rect.height > 0 &&
          rect.left >= 0 &&
          rect.top >= 0 &&
          rect.right <= globalThis.innerWidth &&
          rect.bottom <= globalThis.innerHeight,
        opaque: ancestors.every((element) => Number(getComputedStyle(element).opacity) === 1),
        animationSettled: [
          ...node.getAnimations({ subtree: true }),
          ...ancestors.flatMap((element) => element.getAnimations()),
        ].every((animation) => animation.playState !== 'running' && !animation.pending),
        contentFits:
          node.scrollWidth <= node.clientWidth + 1 && node.scrollHeight <= node.clientHeight + 1,
        points: points.map(([x, y]) => ({
          x,
          y,
          uncovered: document.elementFromPoint(x, y)?.closest('.arco-message') === node,
        })),
      };
    });
  await expect
    .poll(async () => {
      const result = await measure();
      return (
        result.insideViewport &&
        result.opaque &&
        result.animationSettled &&
        result.contentFits &&
        result.points.every((point) => point.uncovered)
      );
    })
    .toBe(true);
  await testInfo.attach(`${name}-notice-readability`, {
    body: JSON.stringify(await measure(), null, 2),
    contentType: 'application/json',
  });
}

test.describe('C04 Kubernetes Service UI contract', () => {
  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
  });

  test('editing preserves named target ports, allocated node ports and read identity', async ({
    page,
  }) => {
    const fixture = await installServiceFixture(page);
    const dialog = await openServiceEditor(page);
    await expect(dialog.getByLabel('名称', { exact: true })).toBeDisabled();
    await expect(dialog.getByLabel('命名空间', { exact: true })).toBeDisabled();
    await expect(dialog.getByLabel('目标端口', { exact: true }).first()).toHaveValue('http-web');
    await dialog.getByLabel('选择器', { exact: true }).fill('app=api, tier=');
    await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
    await expect(dialog).toBeHidden();
    expect(fixture.writes).toHaveLength(1);
    const write = fixture.writes[0];
    expect(write.method).toBe('PUT');
    expect(write.url.pathname).toBe('/api/v1/business/k8s/clusters/41/services/c04-routing');
    expect(write.url.searchParams.get('namespace')).toBe('prod');
    expect(write.body).toEqual({
      type: 'NodePort',
      selector: { app: 'api', tier: '' },
      uid: 'c04-uid-original',
      resourceVersion: '27',
      ports: [
        { name: 'http', port: 80, targetPort: 'http-web', nodePort: 30080, protocol: 'TCP' },
        { name: 'metrics', port: 9090, targetPort: 9091, nodePort: 30090, protocol: 'TCP' },
      ],
    });
    expect(fixture.reads.every((url) => url.pathname.endsWith('/services'))).toBe(true);
  });

  test('creating accepts a named target port and blocks an invalid target name', async ({
    page,
  }) => {
    const fixture = await installServiceFixture(page);
    await page.goto('/business/k8s/service', { waitUntil: 'domcontentloaded' });
    await page.getByRole('button', { name: '新增服务', exact: true }).click();
    const dialog = page.getByRole('dialog');
    await dialog.getByLabel('名称', { exact: true }).fill('c04-created');
    await dialog.getByLabel('目标端口', { exact: true }).fill('HTTP');
    await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
    await expect(dialog.getByRole('alert')).toContainText('请检查端口范围、名称和目标端口');
    expect(fixture.writes).toHaveLength(0);
    await dialog.getByLabel('目标端口', { exact: true }).fill('http-web');
    await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
    await expect(dialog).toBeHidden();
    expect(fixture.writes[0].method).toBe('POST');
    expect(fixture.writes[0].url.searchParams.get('namespace')).toBe('default');
    expect(fixture.writes[0].body).toEqual({
      name: 'c04-created',
      type: 'ClusterIP',
      selector: {},
      ports: [{ port: 80, targetPort: 'http-web', protocol: 'TCP' }],
    });
  });

  test('malformed or duplicate selectors block writes and empty selector remains valid', async ({
    page,
  }) => {
    const fixture = await installServiceFixture(page, { selectorless: true });
    const dialog = await openServiceEditor(page);
    for (const selector of ['app=api, invalid', 'app=api, app=web']) {
      await dialog.getByLabel('选择器', { exact: true }).fill(selector);
      await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
      await expect(dialog.getByRole('alert')).toContainText('选择器格式无效或键重复');
      expect(fixture.writes).toHaveLength(0);
    }
    await dialog.getByLabel('选择器', { exact: true }).fill('');
    await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
    await expect(dialog).toBeHidden();
    expect(fixture.writes[0].body!.selector).toEqual({});
  });

  test('multi-port names validate and switching to ClusterIP removes node ports', async ({
    page,
  }) => {
    const fixture = await installServiceFixture(page);
    const dialog = await openServiceEditor(page);
    await dialog.getByRole('button', { name: '添加端口', exact: true }).click();
    await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
    await expect(dialog.getByRole('alert')).toContainText('多个端口必须使用互不重复的名称');
    expect(fixture.writes).toHaveLength(0);
    await dialog
      .getByLabel('端口名称', { exact: true })
      .nth(2)
      .fill('012345678901234567890123456789');
    await dialog.getByLabel('目标端口', { exact: true }).nth(2).fill('http-web');
    await dialog.locator('#k8s-service-type').click();
    await page
      .locator('.arco-select-option:visible')
      .filter({ hasText: /^ClusterIP$/ })
      .click();
    await expect(dialog.getByLabel('节点端口', { exact: true })).toHaveCount(0);
    await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
    await expect(dialog).toBeHidden();
    expect(fixture.writes[0].body!.type).toBe('ClusterIP');
    const ports = fixture.writes[0].body!.ports as Array<Record<string, unknown>>;
    expect(ports).toHaveLength(3);
    expect(ports.every((port) => !Object.hasOwn(port, 'nodePort'))).toBe(true);
    expect(ports[2].name).toBe('012345678901234567890123456789');
    expect(ports[2].targetPort).toBe('http-web');
  });

  test('stale edit shows one translated error and keeps draft and original version', async ({
    page,
  }, testInfo) => {
    const fixture = await installServiceFixture(page, { stale: true });
    const dialog = await openServiceEditor(page);
    await dialog.getByLabel('选择器', { exact: true }).fill('app=user-draft');
    await dialog.getByLabel('目标端口', { exact: true }).first().fill('http-next');
    await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
    await expect(dialog.getByRole('button', { name: /确定|确认/, exact: true })).not.toHaveClass(
      /arco-btn-loading/,
    );
    await expectReadableServiceNotice(page, testInfo, 'c04-edit-conflict');
    await expect(dialog).toBeVisible();
    await expect(dialog.getByLabel('选择器', { exact: true })).toHaveValue('app=user-draft');
    await expect(dialog.getByLabel('目标端口', { exact: true }).first()).toHaveValue('http-next');
    await expect(dialog).toContainText('27');
    expect(fixture.writes[0].body).toMatchObject({
      uid: 'c04-uid-original',
      resourceVersion: '27',
    });
    expect(fixture.reads).toHaveLength(1);
    expect(fixture.service.selector).toEqual({ app: 'web', tier: '' });
    await captureServiceState(page, testInfo, 'c04-edit-conflict');
  });

  for (const stale of [false, true]) {
    test(`${stale ? 'stale' : 'successful'} delete sends UID and version preconditions`, async ({
      page,
    }, testInfo) => {
      const fixture = await installServiceFixture(page, { stale });
      await page.goto('/business/k8s/service', { waitUntil: 'domcontentloaded' });
      const row = page.locator('tbody tr').filter({ hasText: 'c04-routing' });
      await row.getByRole('button', { name: '删除', exact: true }).click();
      await page
        .locator('.arco-popconfirm')
        .getByRole('button', { name: /确定|确认/, exact: true })
        .click();
      await expect.poll(() => fixture.writes.length).toBe(1);
      const write = fixture.writes[0];
      expect(write.method).toBe('DELETE');
      expect(write.url.searchParams.get('namespace')).toBe('prod');
      expect(write.url.searchParams.get('uid')).toBe('c04-uid-original');
      expect(write.url.searchParams.get('resourceVersion')).toBe('27');
      if (stale) {
        await expectReadableServiceNotice(page, testInfo, 'c04-delete-conflict');
        await expect(row.getByRole('button', { name: '删除', exact: true })).toBeEnabled();
        expect(fixture.reads).toHaveLength(1);
        await captureServiceState(page, testInfo, 'c04-delete-conflict');
      } else {
        await expect(page.locator('.system-list__table-card')).toContainText(
          '暂无 Kubernetes Service。',
        );
        await captureServiceState(page, testInfo, 'c04-delete-empty');
      }
    });
  }

  test('incomplete read identity disables edit and delete', async ({ page }) => {
    const fixture = await installServiceFixture(page, { missingIdentity: true });
    await page.goto('/business/k8s/service', { waitUntil: 'domcontentloaded' });
    const row = page.locator('tbody tr').filter({ hasText: 'c04-routing' });
    await expect(row.getByRole('button', { name: '编辑', exact: true })).toBeDisabled();
    await expect(row.getByRole('button', { name: '删除', exact: true })).toBeDisabled();
    expect(fixture.writes).toHaveLength(0);
  });

  test('read-only permissions hide each Service mutation', async ({ page }, testInfo) => {
    const fixture = await installServiceFixture(page);
    await page.route('**/api/v1/auth/me', async (route) => {
      const response = await route.fetch();
      const payload = await response.json();
      payload.data.roles = [];
      payload.data.perms = ['business:k8s:service:list', 'business:k8s:cluster:list'];
      await route.fulfill({ response, json: payload });
    });
    await page.goto('/business/k8s/service', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('tbody tr').filter({ hasText: 'c04-routing' })).toBeVisible();
    for (const name of ['新增服务', '编辑', '删除'])
      await expect(page.getByRole('button', { name, exact: true })).toHaveCount(0);
    expect(fixture.writes).toHaveLength(0);
    await captureServiceState(page, testInfo, 'c04-read-only');
  });

  for (const viewport of [
    { width: 1440, height: 900 },
    { width: 390, height: 844 },
  ]) {
    test(`rendered list error and empty recovery at ${viewport.width}px`, async ({
      page,
    }, testInfo) => {
      await page.setViewportSize(viewport);
      await installServiceFixture(page, { empty: true });
      let failing = true;
      await page.route(
        /\/api\/v1\/business\/k8s\/clusters\/41\/services(?:\?.*)?$/,
        async (route) => {
          if (!failing) {
            await route.fallback();
            return;
          }
          await route.fulfill({ status: 503, json: { code: 500, message: 'request.failed' } });
        },
      );
      await page.goto('/business/k8s/service', { waitUntil: 'domcontentloaded' });
      const retry = page.locator('.system-list__table-card').getByRole('button');
      await expect(retry).toBeVisible();
      await captureServiceState(page, testInfo, `c04-${viewport.width}-list-error`);
      failing = false;
      await retry.click();
      await expect(page.locator('.system-list__table-card')).toContainText(
        '暂无 Kubernetes Service。',
      );
      await captureServiceState(page, testInfo, `c04-${viewport.width}-list-empty`);
    });

    test(`rendered primary states at ${viewport.width}px`, async ({ page }, testInfo) => {
      await page.setViewportSize(viewport);
      let releaseList!: () => void;
      let releaseUpdate!: () => void;
      const listGate = new Promise<void>((resolve) => {
        releaseList = resolve;
      });
      const updateGate = new Promise<void>((resolve) => {
        releaseUpdate = resolve;
      });
      const fixture = await installServiceFixture(page, { stale: true, listGate, updateGate });
      try {
        await page.goto('/business/k8s/service', { waitUntil: 'domcontentloaded' });
        await expect.poll(() => fixture.reads.length).toBe(1);
        await captureServiceState(page, testInfo, `c04-${viewport.width}-loading`);
        releaseList();
        await page
          .locator('tbody tr')
          .filter({ hasText: 'c04-routing' })
          .getByRole('button', { name: '编辑', exact: true })
          .click();
        const dialog = page.getByRole('dialog');
        await captureServiceState(page, testInfo, `c04-${viewport.width}-editor-top`);
        await dialog.getByLabel('目标端口', { exact: true }).first().fill('http-next');
        await dialog.getByLabel('节点端口', { exact: true }).last().scrollIntoViewIfNeeded();
        await captureServiceState(page, testInfo, `c04-${viewport.width}-editor-ports`);
        await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
        await expect.poll(() => fixture.writes.length).toBe(1);
        await expect(dialog.getByLabel('选择器', { exact: true })).toBeDisabled();
        await captureServiceState(page, testInfo, `c04-${viewport.width}-submitting`);
        releaseUpdate();
        const message = page
          .locator('.arco-message')
          .filter({ hasText: '服务已变化，请刷新列表并重新核对后操作。' });
        await expect(
          dialog.getByRole('button', { name: /确定|确认/, exact: true }),
        ).not.toHaveClass(/arco-btn-loading/);
        await expect(dialog.getByLabel('选择器', { exact: true })).toBeEnabled();
        expect(fixture.writes).toHaveLength(1);
        expect(fixture.writes[0].body).toMatchObject({
          uid: 'c04-uid-original',
          resourceVersion: '27',
        });
        await expectReadableServiceNotice(page, testInfo, `c04-${viewport.width}-conflict`);
        await expect(dialog.getByLabel('目标端口', { exact: true }).first()).toHaveValue(
          'http-next',
        );
        await captureServiceState(page, testInfo, `c04-${viewport.width}-conflict`);
        await dialog.getByRole('button', { name: '取消', exact: true }).click();
        await expect(dialog).toBeHidden();
        await expect(message).toHaveCount(0, { timeout: 1000 });
        await page.getByRole('button', { name: '新增服务', exact: true }).click();
        await expect(message).toHaveCount(0);
        await captureServiceState(page, testInfo, `c04-${viewport.width}-create`);
      } finally {
        releaseList();
        releaseUpdate();
      }
    });
  }
});
