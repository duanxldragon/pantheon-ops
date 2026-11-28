import { expect, test, type APIResponse, type Locator, type Page, type TestInfo } from '@playwright/test';
import type { ApplicationRow, ServiceInstanceRow, ServiceRow } from '../../../../src/modules/business/service/api';
import {
  adminCredentials,
  apiBaseUrl,
  apiRequestHeaders,
  installClientSession,
  loginByApi,
  primeChineseLocale,
} from '../../helpers/auth';

type ApiEnvelope<T> = { code: number; data: T; message?: string };

async function expectSuccess<T>(response: APIResponse): Promise<T> {
  expect(response.ok()).toBeTruthy();
  const payload = (await response.json()) as ApiEnvelope<T>;
  expect(payload.code).toBe(200);
  return payload.data;
}

async function expectNoVisibleServiceI18nKeys(page: import('@playwright/test').Page) {
  const bodyText = await page.locator('body').innerText();
  expect(bodyText).not.toMatch(/\bbusiness\.service\.[A-Za-z0-9_.-]+\b/);
  expect(bodyText).not.toMatch(/\boperations\.service\.[A-Za-z0-9_.-]+\b/);
}

// These fixtures verify the HTTP/UI contract; real owner authorization is
// exercised separately by the API/runtime acceptance lane.
async function installCatalogFixture(page: Page, count = 125) {
  const applications: ApplicationRow[] = Array.from({ length: count }, (_, index) => ({
    id: index + 1, code: `catalog-app-${index + 1}`, name: `catalog-application-${index + 1}`,
    businessScopeId: 41, businessScopeName: 'catalog-scope', deptId: 1, status: 'active',
    owner: 'catalog-owner', remark: '', serviceCount: 1, createdAt: '', updatedAt: '',
  }));
  const services: ServiceRow[] = applications.map((application) => ({
    id: application.id, applicationId: application.id, applicationCode: application.code,
    applicationName: application.name, businessScopeId: 41, code: `catalog-svc-${application.id}`,
    name: `catalog-service-${application.id}`, runtimeType: 'hybrid', description: '',
    status: 'active', instanceCount: 1, createdAt: '', updatedAt: '',
  }));
  const instances: ServiceInstanceRow[] = services.map((service) => ({
    id: service.id, serviceId: service.id, serviceCode: service.code, serviceName: service.name,
    applicationId: service.applicationId, applicationCode: service.applicationCode, businessScopeId: 41,
    environment: 'prod', targetType: 'vm', hostId: service.id, k8sClusterId: 0,
    namespace: '', workloadKind: '', workloadName: '', desiredVersion: '1.2.0', currentVersion: '1.1.0',
    rollbackVersion: '', desiredState: 'running', observedState: 'running', healthState: 'healthy',
    healthMessage: '', healthRevision: '', lifecycleVersion: 7, status: 'active',
  }));
  const lists: Record<string, Array<ApplicationRow | ServiceRow | ServiceInstanceRow>> = { applications, services, instances };
  const reads: URL[] = [];
  const writes: Array<{ path: string; data: Record<string, unknown> }> = [];
  await page.route(/\/api\/v1\/business\/service\/(applications|services|instances)(\/\d+)?(?:\?.*)?$/, async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const parts = url.pathname.split('/');
    const updating = request.method() === 'PUT';
    const detail = /^\d+$/.test(parts.at(-1)!);
    const resource = parts.at(detail ? -2 : -1)!;
    const rows = lists[resource];
    if (updating) {
      const data = request.postDataJSON() as Record<string, unknown>;
      writes.push({ path: url.pathname, data });
      const row = rows.find((item) => item.id === Number(parts.at(-1)))!;
      Object.assign(row, data);
      await route.fulfill({ json: { code: 200, data: row } });
      return;
    }
    reads.push(url);
    if (detail) {
      await route.fulfill({ json: { code: 200, data: rows.find((item) => item.id === Number(parts.at(-1))) } });
      return;
    }
    const current = Number(url.searchParams.get('page') ?? 1);
    const pageSize = Number(url.searchParams.get('pageSize') ?? 20);
    const keyword = url.searchParams.get('keyword') ?? '';
    const filtered = rows.filter((row) => 'name' in row ? row.name.includes(keyword) || row.code.includes(keyword) : true);
    await route.fulfill({ json: { code: 200, data: { items: filtered.slice((current - 1) * pageSize, current * pageSize), total: filtered.length, page: current, pageSize } } });
  });
  return { applications, services, instances, reads, writes };
}

function formField(page: Page, dialog: Locator, label: string) {
  return dialog.locator('.arco-form-item').filter({ has: page.getByText(label, { exact: true }) });
}

async function chooseOption(page: Page, dialog: Locator, label: string, option: string) {
  await formField(page, dialog, label).locator('.arco-select').click();
  await page.locator('.arco-select-option:visible').filter({ hasText: new RegExp(`^${option.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`) }).click();
}

async function captureCatalogState(page: Page, testInfo: TestInfo, name: string) {
  await expect.poll(() => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1)).toBe(true);
  await expectNoVisibleServiceI18nKeys(page);
  const screenshotPath = testInfo.outputPath(`${name}.png`);
  await page.screenshot({ path: screenshotPath, animations: 'disabled' });
  await testInfo.attach(name, { path: screenshotPath, contentType: 'image/png' });
}

test.describe('Service business module smoke', () => {
  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
  });

  for (const scenario of [
    { name: 'creating a service ignores an empty application search in another tab', searchTab: '应用', parentResource: 'applications', placeholder: '搜索应用编码或名称', emptyText: '暂无应用。', createTab: '服务', button: '新增服务', field: '应用', option: 'catalog-application-1' },
    { name: 'creating an instance ignores an empty service search in another tab', searchTab: '服务', parentResource: 'services', placeholder: '搜索服务名称或编码', emptyText: '暂无服务。', createTab: '服务实例', button: '新增服务实例', field: '服务', option: 'catalog-application-1 / catalog-service-1' },
  ]) {
    test(scenario.name, async ({ page }) => {
      const fixture = await installCatalogFixture(page, 2);
      await page.goto('/business/service-catalog', { waitUntil: 'domcontentloaded' });
      await page.getByRole('tab', { name: scenario.searchTab, exact: true }).click();
      const searchedPanel = page.getByRole('tabpanel', { name: scenario.searchTab, exact: true });
      await expect(searchedPanel.locator('tbody tr')).toHaveCount(2);
      await searchedPanel.getByPlaceholder(scenario.placeholder, { exact: true }).fill('no-match-c03');
      await expect(searchedPanel).toContainText(scenario.emptyText);
      await page.getByRole('tab', { name: scenario.createTab, exact: true }).click();
      const createButton = page.getByRole('tabpanel', { name: scenario.createTab, exact: true }).getByRole('button', { name: scenario.button, exact: true });
      await expect(createButton).toBeEnabled();
      await createButton.click();
      const dialog = page.getByRole('dialog');
      await formField(page, dialog, scenario.field).locator('.arco-select').click();
      await expect(page.locator('.arco-select-option:visible').filter({ hasText: scenario.option })).toBeVisible();
      expect(fixture.reads.some((url) => url.pathname.endsWith(scenario.parentResource) && url.searchParams.get('pageSize') === '25' && !url.searchParams.get('keyword'))).toBe(true);
      expect(fixture.writes).toEqual([]);
      await dialog.getByRole('button', { name: '取消', exact: true }).click();
    });
  }

  test('server totals and page six allow editing objects beyond the first hundred', async ({ page }, testInfo) => {
    const fixture = await installCatalogFixture(page);
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/business/service-catalog', { waitUntil: 'domcontentloaded' });
    for (const [tab, resource, name, field] of [
      ['应用', 'applications', 'catalog-application-101', '应用'],
      ['服务', 'services', 'catalog-service-101', '服务'],
      ['服务实例', 'instances', 'catalog-service-101', '版本'],
    ]) {
      await page.getByRole('tab', { name: tab, exact: true }).click();
      const panel = page.getByRole('tabpanel', { name: tab, exact: true });
      await expect(panel.locator('.app-table__pagination-total-text')).toContainText('125');
      await panel.locator('.arco-pagination-item').filter({ hasText: /^6$/ }).click();
      const row = panel.locator('tbody tr').filter({ hasText: name });
      await expect(row).toBeVisible();
      await row.getByRole('button', { name: '编辑', exact: true }).click();
      const dialog = page.getByRole('dialog');
      if (resource === 'applications') {
        await expect(formField(page, dialog, '编码').locator('input')).toBeDisabled();
      }
      if (resource === 'services') {
        await expect(formField(page, dialog, '编码').locator('input')).toBeDisabled();
      }
      const value = resource === 'instances' ? '2.0.0' : `${name}-edited`;
      await formField(page, dialog, field).locator('input').fill(value);
      await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
      await expect(dialog).toBeHidden();
      const write = fixture.writes.at(-1)!;
      expect(write.path).toBe(`/api/v1/business/service/${resource}/101`);
      expect(write.data).not.toHaveProperty('code');
      expect(write.data).not.toHaveProperty('businessScopeId');
      expect(write.data).not.toHaveProperty('applicationId');
      expect(write.data).not.toHaveProperty('serviceId');
      if (resource === 'instances') expect(write.data).toEqual({ desiredVersion: '2.0.0', expectedLifecycleVersion: 7 });
      else expect(write.data.name).toBe(value);
      expect(fixture.reads.some((url) => url.pathname.endsWith(resource) && url.searchParams.get('page') === '6')).toBe(true);
    }
    await expect(page.locator('.governance-summary-bar')).toContainText('125');
    await expectNoVisibleServiceI18nKeys(page);
    await page.screenshot({ path: testInfo.outputPath('service-catalog-pagination-edit.png') });
  });

  test('read-only catalog does not request CMDB or K8s and hides mutation controls', async ({ page }) => {
    await installCatalogFixture(page, 2);
    await page.route('**/api/v1/auth/me', async (route) => {
      const response = await route.fetch();
      const payload = await response.json();
      payload.data.roles = [];
      payload.data.perms = ['business:service:list'];
      await route.fulfill({ response, json: payload });
    });
    const optionalRequests: string[] = [];
    await page.route(/\/api\/v1\/business\/(cmdb|k8s)\//, async (route) => {
      optionalRequests.push(route.request().url());
      await route.fulfill({ status: 403, json: { code: 403, message: 'Forbidden' } });
    });
    await page.goto('/business/service-catalog', { waitUntil: 'domcontentloaded' });
    for (const tab of ['应用', '服务', '服务实例']) {
      await page.getByRole('tab', { name: tab, exact: true }).click();
      const panel = page.getByRole('tabpanel', { name: tab, exact: true });
      await expect(panel.locator('tbody tr')).toHaveCount(2);
      await expect(panel.getByRole('button', { name: /新增/ })).toHaveCount(0);
      await expect(panel.locator('tbody').getByRole('button')).toHaveCount(0);
      for (const control of await panel.getByRole('switch').all()) await expect(control).toBeDisabled();
    }
    expect(optionalRequests).toEqual([]);
    await expectNoVisibleServiceI18nKeys(page);
  });

  test('denied optional target query stays in the create form and preserves the catalog', async ({ page }) => {
    const fixture = await installCatalogFixture(page, 2);
    const targets: URL[] = [];
    await page.route(/\/api\/v1\/business\/cmdb\/hosts(?:\?.*)?$/, async (route) => {
      targets.push(new URL(route.request().url()));
      await route.fulfill({ status: 403, json: { code: 403, message: 'Forbidden' } });
    });
    await page.goto('/business/service-catalog/instances', { waitUntil: 'domcontentloaded' });
    const panel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
    await expect(panel.locator('tbody tr')).toHaveCount(2);
    expect(targets).toEqual([]);
    await panel.getByRole('button', { name: '新增服务实例', exact: true }).click();
    const dialog = page.getByRole('dialog');
    await chooseOption(page, dialog, '服务', 'catalog-application-1 / catalog-service-1');
    await expect(dialog).toContainText('无权查询目标选项');
    await expect(dialog.getByRole('button', { name: /确定|确认/, exact: true })).toBeDisabled();
    expect(targets).toHaveLength(1);
    expect(targets[0].searchParams.get('businessScopeId')).toBe('41');
    expect(targets[0].searchParams.get('pageSize')).toBe('25');
    expect(fixture.reads.some((url) => url.pathname.endsWith('/services') && url.searchParams.get('pageSize') === '25')).toBe(true);
    await dialog.getByRole('button', { name: '取消', exact: true }).click();
    await expect(panel.locator('tbody tr')).toHaveCount(2);
    await expectNoVisibleServiceI18nKeys(page);
  });

  test('K8s options are scope-bound and namespace changes clear stale workloads', async ({ page }) => {
    await installCatalogFixture(page, 2);
    const clusterQueries: URL[] = [];
    const workloadQueries: URL[] = [];
    await page.route(/\/api\/v1\/business\/k8s\/clusters(?:\?.*)?$/, async (route) => {
      clusterQueries.push(new URL(route.request().url()));
      await route.fulfill({ json: { code: 200, data: { items: [
        { id: 11, businessScopeId: 41, name: 'allowed-cluster' },
        { id: 12, businessScopeId: 42, name: 'other-scope-cluster' },
      ], total: 2 } } });
    });
    await page.route('**/api/v1/business/k8s/clusters/11/namespaces', async (route) => {
      await route.fulfill({ json: { code: 200, data: { items: [{ name: 'ns-one' }, { name: 'ns-two' }] } } });
    });
    await page.route(/\/api\/v1\/business\/k8s\/workloads(?:\?.*)?$/, async (route) => {
      const url = new URL(route.request().url());
      workloadQueries.push(url);
      const namespace = url.searchParams.get('namespace');
      await route.fulfill({ json: { code: 200, data: { items: [
        { namespace, kind: 'Deployment', name: `${namespace}-workload` },
        { namespace: 'other-namespace', kind: 'Deployment', name: 'other-namespace-workload' },
        { namespace, kind: 'StatefulSet', name: 'other-kind-workload' },
      ] } } });
    });
    await page.goto('/business/service-catalog/instances', { waitUntil: 'domcontentloaded' });
    await page.getByRole('button', { name: '新增服务实例', exact: true }).click();
    const dialog = page.getByRole('dialog');
    await chooseOption(page, dialog, '服务', 'catalog-application-1 / catalog-service-1');
    await chooseOption(page, dialog, '目标', 'Kubernetes');
    await formField(page, dialog, 'Kubernetes 集群').locator('.arco-select').click();
    await expect(page.locator('.arco-select-option:visible').filter({ hasText: 'other-scope-cluster' })).toHaveCount(0);
    await page.locator('.arco-select-option:visible').filter({ hasText: /^allowed-cluster$/ }).click();
    await chooseOption(page, dialog, '命名空间', 'ns-one');
    await chooseOption(page, dialog, '工作负载类型', 'Deployment');
    await formField(page, dialog, '工作负载名称').locator('.arco-select').click();
    await expect(page.locator('.arco-select-option:visible').filter({ hasText: /other-namespace-workload|other-kind-workload/ })).toHaveCount(0);
    await page.locator('.arco-select-option:visible').filter({ hasText: /^ns-one-workload$/ }).click();
    await chooseOption(page, dialog, '命名空间', 'ns-two');
    await expect(formField(page, dialog, '工作负载名称').locator('.arco-select')).not.toContainText('ns-one-workload');
    await expect(formField(page, dialog, '工作负载类型').locator('.arco-select')).not.toContainText('Deployment');
    await expect(formField(page, dialog, '工作负载名称').locator('.arco-select')).toHaveClass(/arco-select-disabled/);
    expect(clusterQueries).toHaveLength(1);
    expect(clusterQueries[0].searchParams.get('businessScopeId')).toBe('41');
    expect(clusterQueries[0].searchParams.get('pageSize')).toBe('25');
    expect(workloadQueries).toHaveLength(1);
    expect(workloadQueries[0].searchParams.get('clusterId')).toBe('11');
    expect(workloadQueries[0].searchParams.get('namespace')).toBe('ns-one');
    expect(workloadQueries[0].searchParams.get('kind')).toBe('Deployment');
    await dialog.getByRole('button', { name: '取消', exact: true }).click();
  });

  test('instance rebind is explicit, resolves its owner, and submits a versioned complete target', async ({ page }) => {
    const fixture = await installCatalogFixture(page, 2);
    const targets: URL[] = [];
    await page.route(/\/api\/v1\/business\/cmdb\/hosts(?:\?.*)?$/, async (route) => {
      targets.push(new URL(route.request().url()));
      await route.fulfill({ json: { code: 200, data: { items: [
        { id: 201, businessScopeId: 41, hostname: 'allowed-host', ip: '10.0.0.1' },
        { id: 202, businessScopeId: 42, hostname: 'other-scope-host', ip: '10.0.0.2' },
      ], total: 2 } } });
    });
    await page.goto('/business/service-catalog/instances', { waitUntil: 'domcontentloaded' });
    const panel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
    await panel.locator('tbody tr').filter({ hasText: 'catalog-service-1' }).getByRole('button', { name: '编辑', exact: true }).click();
    const dialog = page.getByRole('dialog');
    await expect(formField(page, dialog, '当前版本').locator('input')).toHaveAttribute('readonly', '');
    expect(targets).toEqual([]);
    expect(fixture.reads.some((url) => url.pathname.endsWith('/services/1'))).toBe(false);
    await formField(page, dialog, '更换绑定目标').getByRole('switch').click();
    await formField(page, dialog, 'CMDB 主机').locator('.arco-select').click();
    await expect(page.locator('.arco-select-option:visible').filter({ hasText: 'other-scope-host' })).toHaveCount(0);
    await page.locator('.arco-select-option:visible').filter({ hasText: /^allowed-host \(10\.0\.0\.1\)$/ }).click();
    await formField(page, dialog, '版本').locator('input').fill('2.1.0');
    await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
    await expect(dialog).toBeHidden();
    expect(fixture.writes.at(-1)?.data).toEqual({ desiredVersion: '2.1.0', expectedLifecycleVersion: 7, target: { targetType: 'vm', hostId: 201 } });
    expect(fixture.reads.some((url) => url.pathname.endsWith('/services/1'))).toBe(true);
    expect(targets[0].searchParams.get('businessScopeId')).toBe('41');
    await expectNoVisibleServiceI18nKeys(page);
  });

  test('a stale instance update retains the form and the user version input', async ({ page }) => {
    await installCatalogFixture(page, 2);
    const conflicts: Record<string, unknown>[] = [];
    await page.route('**/api/v1/business/service/instances/1', async (route) => {
      conflicts.push(route.request().postDataJSON() as Record<string, unknown>);
      // Owner conflicts use HTTP 200 with the machine key in envelope.message.
      await route.fulfill({ status: 200, json: { code: 409, data: null, message: 'business.service.state_stale' } });
    });
    await page.goto('/business/service-catalog/instances', { waitUntil: 'domcontentloaded' });
    const panel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
    await panel.locator('tbody tr').filter({ hasText: 'catalog-service-1' }).getByRole('button', { name: '编辑', exact: true }).click();
    const dialog = page.getByRole('dialog');
    await formField(page, dialog, '版本').locator('input').fill('3.0.0');
    const submit = dialog.getByRole('button', { name: /确定|确认/, exact: true });
    await submit.click();
    await expect.poll(() => conflicts.length).toBe(1);
    await expect(submit).not.toHaveClass(/arco-btn-loading/);
    await expect(page.locator('.arco-message-error')).toHaveCount(1);
    const staleError = page.locator('.arco-message-error').getByText('实例状态已变化，请刷新列表并重新核对后操作。', { exact: true });
    await expect(staleError).toHaveCount(1);
    await expect(staleError).toBeVisible();
    await expect(dialog).toBeVisible();
    await expect(formField(page, dialog, '版本').locator('input')).toHaveValue('3.0.0');
    expect(conflicts[0]).toEqual({ desiredVersion: '3.0.0', expectedLifecycleVersion: 7 });
    await expectNoVisibleServiceI18nKeys(page);
    await dialog.getByRole('button', { name: '取消', exact: true }).click();
  });

  for (const viewport of [
    { name: 'desktop', width: 1440, height: 900 },
    { name: 'mobile', width: 390, height: 844 },
  ]) {
    test(`catalog desktop and mobile rendered states ${viewport.name}`, async ({ page }, testInfo) => {
      test.setTimeout(60_000);
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      const fixture = await installCatalogFixture(page, 2);
      const conflicts: Record<string, unknown>[] = [];
      await page.route('**/api/v1/business/service/instances/1', async (route) => {
        conflicts.push(route.request().postDataJSON() as Record<string, unknown>);
        await route.fulfill({ status: 200, json: { code: 409, data: null, message: 'business.service.state_stale' } });
      });
      const optionalRequests: string[] = [];
      await page.route(/\/api\/v1\/business\/(cmdb|k8s)\//, async (route) => {
        optionalRequests.push(route.request().url());
        await route.fulfill({ status: 403, json: { code: 403, data: null, message: 'common.forbidden' } });
      });
      const capture = (state: string) => captureCatalogState(page, testInfo, `${viewport.width}x${viewport.height}-${state}`);

      await page.goto('/business/service-catalog/applications', { waitUntil: 'domcontentloaded' });
      const applicationPanel = page.getByRole('tabpanel', { name: '应用', exact: true });
      await expect(applicationPanel.locator('tbody tr')).toHaveCount(2);
      const createApplication = applicationPanel.getByRole('button', { name: '新增应用', exact: true });
      await expect(createApplication).toBeInViewport({ ratio: 1 });
      await createApplication.focus();
      await expect(createApplication).toBeFocused();
      await capture('catalog');

      await page.getByRole('tab', { name: '服务实例', exact: true }).click();
      const instancePanel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
      await expect(instancePanel.getByRole('button', { name: '新增服务实例', exact: true })).toBeInViewport({ ratio: 1 });
      await instancePanel.locator('tbody tr').filter({ hasText: 'catalog-service-1' }).getByRole('button', { name: '编辑', exact: true }).click();
      const dialog = page.getByRole('dialog');
      const version = formField(page, dialog, '版本').locator('input');
      await expect(formField(page, dialog, '当前版本').locator('input')).toHaveAttribute('readonly', '');
      await version.fill('3.0.0');
      await expect(version).toBeFocused();
      const submit = dialog.getByRole('button', { name: /确定|确认/, exact: true });
      await expect(submit).toBeInViewport({ ratio: 1 });
      await expect(dialog.getByRole('button', { name: '取消', exact: true })).toBeInViewport({ ratio: 1 });
      await capture('instance-editor');

      await submit.click();
      await expect(submit).not.toHaveClass(/arco-btn-loading/);
      await expect(page.locator('.arco-message-error')).toHaveCount(1);
      const staleError = page.locator('.arco-message-error').getByText('实例状态已变化，请刷新列表并重新核对后操作。', { exact: true });
      await expect(staleError).toHaveCount(1);
      await expect(staleError).toBeVisible();
      await expect(dialog).toBeVisible();
      await expect(version).toHaveValue('3.0.0');
      expect(conflicts).toEqual([{ desiredVersion: '3.0.0', expectedLifecycleVersion: 7 }]);
      await capture('stale-error-input-retained');
      await dialog.getByRole('button', { name: '取消', exact: true }).click();

      await page.route('**/api/v1/auth/me', async (route) => {
        const response = await route.fetch();
        const payload = await response.json();
        payload.data.roles = [];
        payload.data.perms = ['business:service:list'];
        await route.fulfill({ response, json: payload });
      });
      // Clear only this isolated browser's cached user before role rehydration.
      await page.addInitScript(() => localStorage.removeItem('pantheon_auth_user'));
      await page.goto('/business/service-catalog/applications', { waitUntil: 'domcontentloaded' });
      const readOnlyPanel = page.getByRole('tabpanel', { name: '应用', exact: true });
      await expect(readOnlyPanel.locator('tbody tr')).toHaveCount(2);
      await expect(readOnlyPanel.getByRole('button', { name: /新增|编辑|删除/ })).toHaveCount(0);
      for (const control of await readOnlyPanel.getByRole('switch').all()) await expect(control).toBeDisabled();
      await capture('read-only');

      fixture.applications.length = 0;
      fixture.services.length = 0;
      fixture.instances.length = 0;
      await page.reload({ waitUntil: 'domcontentloaded' });
      const emptyPanel = page.getByRole('tabpanel', { name: '应用', exact: true });
      await expect(emptyPanel).toContainText('暂无应用。');
      await expect(emptyPanel.getByRole('button', { name: /新增|编辑|删除/ })).toHaveCount(0);
      await capture('read-only-empty');
      expect(fixture.writes).toEqual([]);
      expect(optionalRequests).toEqual([]);
    });
  }

  test('application, service, and VM instance complete the catalog lifecycle', async ({ page, request }, testInfo) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    const consoleErrors: string[] = [];
    page.on('console', (message) => {
      if (message.type() === 'error') consoleErrors.push(message.text());
    });
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `service-smoke-${Date.now()}`;
    let scopeId = 0;
    let hostId = 0;
    let applicationId = 0;
    let serviceId = 0;
    let instanceId = 0;

    try {
      const scope = await expectSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/bizscope`, {
          headers,
          data: { code: token, name: token, environment: 'prod', status: 'active' },
        }),
      );
      scopeId = scope.id;
      const host = await expectSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
          headers,
          data: {
            hostname: `${token}-host`,
            ip: `10.251.${Math.floor(Math.random() * 200) + 20}.${Math.floor(Math.random() * 200) + 20}`,
            sshPort: 22,
            os: 'linux',
            osVersion: 'Ubuntu 24.04',
            cpuCores: 2,
            memoryGb: 4,
            diskGb: 40,
            labels: [{ key: 'service', val: token }],
            owner: 'service-smoke',
            remark: token,
          },
        }),
      );
      hostId = host.id;
      await expectSuccess(
        await request.post(`${apiBaseUrl}/business/bizscope/${scopeId}/hosts/bind`, {
          headers,
          data: { hostIds: [hostId] },
        }),
      );
      const application = await expectSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/service/applications`, {
          headers,
          data: { code: `${token}-app`, name: `${token}-application`, businessScopeId: scopeId, owner: 'service-smoke' },
        }),
      );
      applicationId = application.id;
      const service = await expectSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/service/services`, {
          headers,
          data: { applicationId, code: `${token}-svc`, name: `${token}-service`, runtimeType: 'vm' },
        }),
      );
      serviceId = service.id;
      const instance = await expectSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/service/instances`, {
          headers,
          data: { serviceId, environment: 'prod', targetType: 'vm', hostId, desiredVersion: '1.2.0' },
        }),
      );
      instanceId = Number(instance.id);
      expect(instanceId).toBeGreaterThan(0);

      await page.goto('/business/service-catalog', { waitUntil: 'domcontentloaded' });
      await expect(page).toHaveURL(/\/business\/service-catalog$/);
      await expect(page.locator('.governance-summary-bar')).toContainText('服务目录');
      await expect(page.locator('.system-list__table-card:visible')).toContainText(`${token}-application`);

      await page.getByRole('tab', { name: '服务', exact: true }).click();
      const servicePanel = page.getByRole('tabpanel', { name: '服务', exact: true });
      await expect(servicePanel).toContainText(`${token}-service`);
      await page.getByRole('tab', { name: '服务实例', exact: true }).click();
      const instancePanel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
      await expect(instancePanel).toContainText(`${token}-service`);
      await expect(instancePanel).toContainText(`主机 #${hostId}`);
      await expectNoVisibleServiceI18nKeys(page);
      await page.screenshot({ path: testInfo.outputPath('service-catalog-desktop.png') });
      expect(consoleErrors).toEqual([]);
    } finally {
      const cleanupFailures: string[] = [];
      const cleanup = async (label: string, responsePromise: Promise<APIResponse>) => {
        try {
          const response = await responsePromise;
          const payload = await response.json().catch(() => null) as ApiEnvelope<unknown> | null;
          if (!response.ok() || payload?.code !== 200) {
            cleanupFailures.push(
              `${label}: HTTP ${response.status()} / code ${payload?.code ?? 'unknown'} / ${payload?.message ?? 'no message'}`,
            );
          }
        } catch (error) {
          cleanupFailures.push(`${label}: ${error instanceof Error ? error.message : String(error)}`);
        }
      };
      if (!instanceId && serviceId) {
        try {
          const response = await request.get(
            `${apiBaseUrl}/business/service/instances?serviceId=${serviceId}&page=1&pageSize=20`,
            { headers },
          );
          const payload = await response.json() as ApiEnvelope<{ items: Array<{ id: number }> }>;
          instanceId = Number(payload.data?.items?.[0]?.id ?? 0);
          if (!response.ok() || payload.code !== 200 || !instanceId) {
            cleanupFailures.push('resolve instance: unable to find the created service instance');
          }
        } catch (error) {
          cleanupFailures.push(`resolve instance: ${error instanceof Error ? error.message : String(error)}`);
        }
      }
      if (instanceId) await cleanup('instance', request.delete(`${apiBaseUrl}/business/service/instances/${instanceId}`, { headers }));
      if (serviceId) await cleanup('service', request.delete(`${apiBaseUrl}/business/service/services/${serviceId}`, { headers }));
      if (applicationId) await cleanup('application', request.delete(`${apiBaseUrl}/business/service/applications/${applicationId}`, { headers }));
      if (scopeId && hostId) await cleanup('unbind host', request.delete(`${apiBaseUrl}/business/bizscope/${scopeId}/hosts/${hostId}`, { headers }));
      if (hostId) await cleanup('host', request.delete(`${apiBaseUrl}/business/cmdb/hosts/${hostId}`, { headers }));
      if (scopeId) await cleanup('scope', request.delete(`${apiBaseUrl}/business/bizscope/${scopeId}`, { headers }));
      expect(cleanupFailures).toEqual([]);
    }
  });

  test('service page remains usable in a phone viewport', async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/business/service-catalog', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect(page.getByRole('tab', { name: '服务实例', exact: true })).toBeVisible();
    await expect
      .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
      .toBe(true);
    await page.screenshot({ path: testInfo.outputPath('service-list-mobile.png') });
  });
});

test.describe('Service instance deploy entry (T20)', () => {
  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
  });

  test('instance filters query the server with environment, target and status', async ({ page }) => {
    const fixture = await installCatalogFixture(page, 3);
    await page.goto('/business/service-catalog/instances', { waitUntil: 'domcontentloaded' });
    const panel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
    await expect(panel.locator('tbody tr')).toHaveCount(3);
    const instanceReads = () => fixture.reads.filter((url) => url.pathname.endsWith('/instances'));

    await panel.locator('.arco-select-view').filter({ hasText: '环境' }).click();
    await page.locator('.arco-select-option:visible').filter({ hasText: /^prod$/ }).click();
    await expect.poll(() => instanceReads().some((url) => url.searchParams.get('environment') === 'prod')).toBe(true);
    expect(instanceReads().at(-1)?.searchParams.get('page')).toBe('1');

    await panel.locator('.arco-select-view').filter({ hasText: '目标类型' }).click();
    await page.locator('.arco-select-option:visible').filter({ hasText: /^k8s$/ }).click();
    await expect.poll(() => instanceReads().some((url) => url.searchParams.get('targetType') === 'k8s')).toBe(true);

    await panel.locator('.arco-select-view').filter({ hasText: '状态' }).click();
    await page.locator('.arco-select-option:visible').filter({ hasText: /^启用$/ }).click();
    await expect.poll(() => instanceReads().some((url) => url.searchParams.get('status') === 'active')).toBe(true);
    const last = instanceReads().at(-1)!;
    expect(last.searchParams.get('environment')).toBe('prod');
    expect(last.searchParams.get('targetType')).toBe('k8s');
    expect(last.searchParams.get('status')).toBe('active');
    expect(last.searchParams.get('page')).toBe('1');
  });

  test('a role without deploy permission does not see the instance deploy entry', async ({ page }) => {
    await installCatalogFixture(page, 2);
    await page.route('**/api/v1/auth/me', async (route) => {
      const response = await route.fetch();
      const payload = await response.json();
      payload.data.roles = [];
      payload.data.perms = ['business:service:list'];
      await route.fulfill({ response, json: payload });
    });
    await page.goto('/business/service-catalog/instances', { waitUntil: 'domcontentloaded' });
    const panel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
    await expect(panel.locator('tbody tr')).toHaveCount(2);
    await expect(panel.getByRole('button', { name: '部署', exact: true })).toHaveCount(0);
    await expect(panel.getByRole('button', { name: /新增|编辑|删除|协调/ })).toHaveCount(0);
  });

  test('deploying from an instance opens the prefilled create task modal once', async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await installCatalogFixture(page, 1);
    await page.route(/\/api\/v1\/business\/bizscope\/options(?:\?.*)?$/, async (route) => {
      await route.fulfill({ json: { code: 200, data: [{ id: 41, name: 'catalog-scope', label: 'catalog-scope', value: 41 }] } });
    });
    await page.route(/\/api\/v1\/business\/cmdb\/hosts(?:\?.*)?$/, async (route) => {
      await route.fulfill({ json: { code: 200, data: { items: [{ id: 1, businessScopeId: 41, hostname: 'catalog-host', ip: '10.0.0.1' }], total: 1, page: 1, pageSize: 25 } } });
    });
    await page.route(/\/api\/v1\/business\/cmdb\/groups(?:\?.*)?$/, async (route) => {
      await route.fulfill({ json: { code: 200, data: [] } });
    });
    await page.route(/\/api\/v1\/business\/deploy\/(templates|packages|credentials|tasks)(?:\?.*)?$/, async (route) => {
      const path = new URL(route.request().url()).pathname;
      if (path.endsWith('/templates')) {
        await route.fulfill({ json: { code: 200, data: { items: [{ id: 1, name: 'catalog-template', version: '1.0.0', description: '', category: 'base', executionMode: 'fixed', defaultAction: 'install', packageId: 2, packageName: 'catalog-package', packageVersion: '1.0.0', status: 'enabled', stepCount: 1, steps: [], createdAt: '', updatedAt: '', createdBy: '', updatedBy: '' }], total: 1, page: 1, pageSize: 100 } } });
        return;
      }
      if (path.endsWith('/packages')) {
        await route.fulfill({ json: { code: 200, data: { items: [{ id: 2, name: 'catalog-package', version: '1.0.0', description: '', executionMode: 'fixed', status: 'enabled', createdAt: '', updatedAt: '' }], total: 1, page: 1, pageSize: 100 } } });
        return;
      }
      if (path.endsWith('/credentials')) {
        await route.fulfill({ json: { code: 200, data: [] } });
        return;
      }
      await route.fulfill({ json: { code: 200, data: { items: [], total: 0, page: 1, pageSize: 10 } } });
    });

    await page.goto('/business/service-catalog/instances', { waitUntil: 'domcontentloaded' });
    const panel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
    await panel.locator('tbody tr').filter({ hasText: 'catalog-service-1' }).getByRole('button', { name: '部署', exact: true }).click();
    await expect(page).toHaveURL(/\/business\/deploy\/task\?/);
    const intentUrl = new URL(page.url());
    expect(intentUrl.searchParams.get('createFor')).toBe('service-instance');
    expect(intentUrl.searchParams.get('serviceInstanceId')).toBe('1');
    expect(intentUrl.searchParams.get('serviceId')).toBe('1');
    expect(intentUrl.searchParams.get('businessScopeId')).toBe('41');

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog).toContainText('catalog-scope');
    await expect
      .poll(async () => dialog.locator('input').evaluateAll((els) => els.map((el) => (el as HTMLInputElement).value)))
      .toContain('catalog-service-1 部署');

    await page.screenshot({ path: testInfo.outputPath('t20-deploy-entry-1440.png') });
    // The intent is consumed once: reloading the deploy page must not reopen the editor.
    expect(new URL(page.url()).searchParams.has('createFor')).toBe(false);
    await page.reload({ waitUntil: 'domcontentloaded' });
    await expect(page.getByRole('dialog')).toHaveCount(0);
    await page.screenshot({ path: testInfo.outputPath('t20-deploy-list-after-intent.png') });

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(
      '/business/deploy/task?createFor=service-instance&serviceId=1&serviceInstanceId=1&businessScopeId=41&hostId=1&serviceName=catalog-service-1',
      { waitUntil: 'domcontentloaded' },
    );
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect
      .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
      .toBe(true);
    await page.screenshot({ path: testInfo.outputPath('t20-deploy-entry-390.png') });
  });
});
