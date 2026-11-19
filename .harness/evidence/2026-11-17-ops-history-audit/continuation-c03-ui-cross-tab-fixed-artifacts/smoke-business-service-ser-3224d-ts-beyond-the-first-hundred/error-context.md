# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\service\service-pages.spec.ts >> Service business module smoke >> server totals and page six allow editing objects beyond the first hundred
- Location: tests\smoke\business\service\service-pages.spec.ts:122:3

# Error details

```
Test timeout of 30000ms exceeded.
```

```
Error: locator.click: Test timeout of 30000ms exceeded.
Call log:
  - waiting for getByRole('tab', { name: '应用', exact: true })

```

# Test source

```ts
  31  |     id: index + 1, code: `catalog-app-${index + 1}`, name: `catalog-application-${index + 1}`,
  32  |     businessScopeId: 41, businessScopeName: 'catalog-scope', deptId: 1, status: 'active',
  33  |     owner: 'catalog-owner', remark: '', serviceCount: 1, createdAt: '', updatedAt: '',
  34  |   }));
  35  |   const services: ServiceRow[] = applications.map((application) => ({
  36  |     id: application.id, applicationId: application.id, applicationCode: application.code,
  37  |     applicationName: application.name, businessScopeId: 41, code: `catalog-svc-${application.id}`,
  38  |     name: `catalog-service-${application.id}`, runtimeType: 'hybrid', description: '',
  39  |     status: 'active', instanceCount: 1, createdAt: '', updatedAt: '',
  40  |   }));
  41  |   const instances: ServiceInstanceRow[] = services.map((service) => ({
  42  |     id: service.id, serviceId: service.id, serviceCode: service.code, serviceName: service.name,
  43  |     environment: 'prod', targetType: 'vm', hostId: service.id, k8sClusterId: 0,
  44  |     namespace: '', workloadKind: '', workloadName: '', desiredVersion: '1.2.0', currentVersion: '1.1.0',
  45  |     rollbackVersion: '', desiredState: 'running', observedState: 'running', healthState: 'healthy',
  46  |     healthMessage: '', healthRevision: '', lifecycleVersion: 7, status: 'active',
  47  |   }));
  48  |   const lists: Record<string, Array<ApplicationRow | ServiceRow | ServiceInstanceRow>> = { applications, services, instances };
  49  |   const reads: URL[] = [];
  50  |   const writes: Array<{ path: string; data: Record<string, unknown> }> = [];
  51  |   await page.route(/\/api\/v1\/business\/service\/(applications|services|instances)(\/\d+)?(?:\?.*)?$/, async (route) => {
  52  |     const request = route.request();
  53  |     const url = new URL(request.url());
  54  |     const parts = url.pathname.split('/');
  55  |     const updating = request.method() === 'PUT';
  56  |     const detail = /^\d+$/.test(parts.at(-1)!);
  57  |     const resource = parts.at(detail ? -2 : -1)!;
  58  |     const rows = lists[resource];
  59  |     if (updating) {
  60  |       const data = request.postDataJSON() as Record<string, unknown>;
  61  |       writes.push({ path: url.pathname, data });
  62  |       const row = rows.find((item) => item.id === Number(parts.at(-1)))!;
  63  |       Object.assign(row, data);
  64  |       await route.fulfill({ json: { code: 200, data: row } });
  65  |       return;
  66  |     }
  67  |     reads.push(url);
  68  |     if (detail) {
  69  |       await route.fulfill({ json: { code: 200, data: rows.find((item) => item.id === Number(parts.at(-1))) } });
  70  |       return;
  71  |     }
  72  |     const current = Number(url.searchParams.get('page') ?? 1);
  73  |     const pageSize = Number(url.searchParams.get('pageSize') ?? 20);
  74  |     const keyword = url.searchParams.get('keyword') ?? '';
  75  |     const filtered = rows.filter((row) => 'name' in row ? row.name.includes(keyword) || row.code.includes(keyword) : true);
  76  |     await route.fulfill({ json: { code: 200, data: { items: filtered.slice((current - 1) * pageSize, current * pageSize), total: filtered.length, page: current, pageSize } } });
  77  |   });
  78  |   return { applications, services, instances, reads, writes };
  79  | }
  80  |
  81  | function formField(page: Page, dialog: Locator, label: string) {
  82  |   return dialog.locator('.arco-form-item').filter({ has: page.getByText(label, { exact: true }) });
  83  | }
  84  |
  85  | async function chooseOption(page: Page, dialog: Locator, label: string, option: string) {
  86  |   await formField(page, dialog, label).locator('.arco-select').click();
  87  |   await page.locator('.arco-select-option:visible').filter({ hasText: new RegExp(`^${option.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`) }).click();
  88  | }
  89  |
  90  | test.describe('Service business module smoke', () => {
  91  |   test.beforeEach(async ({ page }) => {
  92  |     const login = await loginByApi(page.request, adminCredentials);
  93  |     await installClientSession(page, login);
  94  |     await primeChineseLocale(page);
  95  |   });
  96  |
  97  |   for (const scenario of [
  98  |     { name: 'creating a service ignores an empty application search in another tab', searchTab: '应用', parentResource: 'applications', placeholder: '搜索应用编码或名称', emptyText: '暂无应用。', createTab: '服务', button: '新增服务', field: '应用', option: 'catalog-application-1' },
  99  |     { name: 'creating an instance ignores an empty service search in another tab', searchTab: '服务', parentResource: 'services', placeholder: '搜索服务名称或编码', emptyText: '暂无服务。', createTab: '服务实例', button: '新增服务实例', field: '服务', option: 'catalog-application-1 / catalog-service-1' },
  100 |   ]) {
  101 |     test(scenario.name, async ({ page }) => {
  102 |       const fixture = await installCatalogFixture(page, 2);
  103 |       await page.goto('/business/service-catalog', { waitUntil: 'domcontentloaded' });
  104 |       await page.getByRole('tab', { name: scenario.searchTab, exact: true }).click();
  105 |       const searchedPanel = page.getByRole('tabpanel', { name: scenario.searchTab, exact: true });
  106 |       await expect(searchedPanel.locator('tbody tr')).toHaveCount(2);
  107 |       await searchedPanel.getByPlaceholder(scenario.placeholder, { exact: true }).fill('no-match-c03');
  108 |       await expect(searchedPanel).toContainText(scenario.emptyText);
  109 |       await page.getByRole('tab', { name: scenario.createTab, exact: true }).click();
  110 |       const createButton = page.getByRole('tabpanel', { name: scenario.createTab, exact: true }).getByRole('button', { name: scenario.button, exact: true });
  111 |       await expect(createButton).toBeEnabled();
  112 |       await createButton.click();
  113 |       const dialog = page.getByRole('dialog');
  114 |       await formField(page, dialog, scenario.field).locator('.arco-select').click();
  115 |       await expect(page.locator('.arco-select-option:visible').filter({ hasText: scenario.option })).toBeVisible();
  116 |       expect(fixture.reads.some((url) => url.pathname.endsWith(scenario.parentResource) && url.searchParams.get('pageSize') === '25' && !url.searchParams.get('keyword'))).toBe(true);
  117 |       expect(fixture.writes).toEqual([]);
  118 |       await dialog.getByRole('button', { name: '取消', exact: true }).click();
  119 |     });
  120 |   }
  121 |
  122 |   test('server totals and page six allow editing objects beyond the first hundred', async ({ page }, testInfo) => {
  123 |     const fixture = await installCatalogFixture(page);
  124 |     await page.setViewportSize({ width: 1440, height: 900 });
  125 |     await page.goto('/business/service-catalog', { waitUntil: 'domcontentloaded' });
  126 |     for (const [tab, resource, name, field] of [
  127 |       ['应用', 'applications', 'catalog-application-101', '应用'],
  128 |       ['服务', 'services', 'catalog-service-101', '服务'],
  129 |       ['服务实例', 'instances', 'catalog-service-101', '版本'],
  130 |     ]) {
> 131 |       await page.getByRole('tab', { name: tab, exact: true }).click();
      |                                                               ^ Error: locator.click: Test timeout of 30000ms exceeded.
  132 |       const panel = page.getByRole('tabpanel', { name: tab, exact: true });
  133 |       await expect(panel.locator('.app-table__pagination-total-text')).toContainText('125');
  134 |       await panel.locator('.arco-pagination-item').filter({ hasText: /^6$/ }).click();
  135 |       const row = panel.locator('tbody tr').filter({ hasText: name });
  136 |       await expect(row).toBeVisible();
  137 |       await row.getByRole('button', { name: '编辑', exact: true }).click();
  138 |       const dialog = page.getByRole('dialog');
  139 |       if (resource === 'applications') {
  140 |         await expect(formField(page, dialog, '编码').locator('input')).toBeDisabled();
  141 |       }
  142 |       if (resource === 'services') {
  143 |         await expect(formField(page, dialog, '编码').locator('input')).toBeDisabled();
  144 |       }
  145 |       const value = resource === 'instances' ? '2.0.0' : `${name}-edited`;
  146 |       await formField(page, dialog, field).locator('input').fill(value);
  147 |       await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
  148 |       await expect(dialog).toBeHidden();
  149 |       const write = fixture.writes.at(-1)!;
  150 |       expect(write.path).toBe(`/api/v1/business/service/${resource}/101`);
  151 |       expect(write.data).not.toHaveProperty('code');
  152 |       expect(write.data).not.toHaveProperty('businessScopeId');
  153 |       expect(write.data).not.toHaveProperty('applicationId');
  154 |       expect(write.data).not.toHaveProperty('serviceId');
  155 |       if (resource === 'instances') expect(write.data).toEqual({ desiredVersion: '2.0.0', expectedLifecycleVersion: 7 });
  156 |       else expect(write.data.name).toBe(value);
  157 |       expect(fixture.reads.some((url) => url.pathname.endsWith(resource) && url.searchParams.get('page') === '6')).toBe(true);
  158 |     }
  159 |     await expect(page.locator('.governance-summary-bar')).toContainText('125');
  160 |     await expectNoVisibleServiceI18nKeys(page);
  161 |     await page.screenshot({ path: testInfo.outputPath('service-catalog-pagination-edit.png') });
  162 |   });
  163 |
  164 |   test('read-only catalog does not request CMDB or K8s and hides mutation controls', async ({ page }) => {
  165 |     await installCatalogFixture(page, 2);
  166 |     await page.route('**/api/v1/auth/me', async (route) => {
  167 |       const response = await route.fetch();
  168 |       const payload = await response.json();
  169 |       payload.data.roles = [];
  170 |       payload.data.perms = ['business:service:list'];
  171 |       await route.fulfill({ response, json: payload });
  172 |     });
  173 |     const optionalRequests: string[] = [];
  174 |     await page.route(/\/api\/v1\/business\/(cmdb|k8s)\//, async (route) => {
  175 |       optionalRequests.push(route.request().url());
  176 |       await route.fulfill({ status: 403, json: { code: 403, message: 'Forbidden' } });
  177 |     });
  178 |     await page.goto('/business/service-catalog', { waitUntil: 'domcontentloaded' });
  179 |     for (const tab of ['应用', '服务', '服务实例']) {
  180 |       await page.getByRole('tab', { name: tab, exact: true }).click();
  181 |       const panel = page.getByRole('tabpanel', { name: tab, exact: true });
  182 |       await expect(panel.locator('tbody tr')).toHaveCount(2);
  183 |       await expect(panel.getByRole('button', { name: /新增/ })).toHaveCount(0);
  184 |       await expect(panel.locator('tbody').getByRole('button')).toHaveCount(0);
  185 |       for (const control of await panel.getByRole('switch').all()) await expect(control).toBeDisabled();
  186 |     }
  187 |     expect(optionalRequests).toEqual([]);
  188 |     await expectNoVisibleServiceI18nKeys(page);
  189 |   });
  190 |
  191 |   test('denied optional target query stays in the create form and preserves the catalog', async ({ page }) => {
  192 |     const fixture = await installCatalogFixture(page, 2);
  193 |     const targets: URL[] = [];
  194 |     await page.route(/\/api\/v1\/business\/cmdb\/hosts(?:\?.*)?$/, async (route) => {
  195 |       targets.push(new URL(route.request().url()));
  196 |       await route.fulfill({ status: 403, json: { code: 403, message: 'Forbidden' } });
  197 |     });
  198 |     await page.goto('/business/service-catalog/instances', { waitUntil: 'domcontentloaded' });
  199 |     const panel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
  200 |     await expect(panel.locator('tbody tr')).toHaveCount(2);
  201 |     expect(targets).toEqual([]);
  202 |     await panel.getByRole('button', { name: '新增服务实例', exact: true }).click();
  203 |     const dialog = page.getByRole('dialog');
  204 |     await chooseOption(page, dialog, '服务', 'catalog-application-1 / catalog-service-1');
  205 |     await expect(dialog).toContainText('无权查询目标选项');
  206 |     await expect(dialog.getByRole('button', { name: /确定|确认/, exact: true })).toBeDisabled();
  207 |     expect(targets).toHaveLength(1);
  208 |     expect(targets[0].searchParams.get('businessScopeId')).toBe('41');
  209 |     expect(targets[0].searchParams.get('pageSize')).toBe('25');
  210 |     expect(fixture.reads.some((url) => url.pathname.endsWith('/services') && url.searchParams.get('pageSize') === '25')).toBe(true);
  211 |     await dialog.getByRole('button', { name: '取消', exact: true }).click();
  212 |     await expect(panel.locator('tbody tr')).toHaveCount(2);
  213 |     await expectNoVisibleServiceI18nKeys(page);
  214 |   });
  215 |
  216 |   test('K8s options are scope-bound and namespace changes clear stale workloads', async ({ page }) => {
  217 |     await installCatalogFixture(page, 2);
  218 |     const clusterQueries: URL[] = [];
  219 |     const workloadQueries: URL[] = [];
  220 |     await page.route(/\/api\/v1\/business\/k8s\/clusters(?:\?.*)?$/, async (route) => {
  221 |       clusterQueries.push(new URL(route.request().url()));
  222 |       await route.fulfill({ json: { code: 200, data: { items: [
  223 |         { id: 11, businessScopeId: 41, name: 'allowed-cluster' },
  224 |         { id: 12, businessScopeId: 42, name: 'other-scope-cluster' },
  225 |       ], total: 2 } } });
  226 |     });
  227 |     await page.route('**/api/v1/business/k8s/clusters/11/namespaces', async (route) => {
  228 |       await route.fulfill({ json: { code: 200, data: { items: [{ name: 'ns-one' }, { name: 'ns-two' }] } } });
  229 |     });
  230 |     await page.route(/\/api\/v1\/business\/k8s\/workloads(?:\?.*)?$/, async (route) => {
  231 |       const url = new URL(route.request().url());
```
