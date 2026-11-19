# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\service\service-pages.spec.ts >> Service business module smoke >> denied optional target query stays in the create form and preserves the catalog
- Location: tests\smoke\business\service\service-pages.spec.ts:191:3

# Error details

```
Error: expect(locator).toHaveCount(expected) failed

Locator:  getByRole('tabpanel', { name: '服务实例', exact: true }).locator('tbody tr')
Expected: 2
Received: 0
Timeout:  10000ms

Call log:
  - Expect "toHaveCount" getByRole('tabpanel', { name: '服务实例', exact: true }).locator('tbody tr') with timeout 10000ms
  - waiting for getByRole('tabpanel', { name: '服务实例', exact: true }).locator('tbody tr')
    23 × locator resolved to 0 elements
       - unexpected value "0"

```

# Test source

```ts
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
  131 |       await page.getByRole('tab', { name: tab, exact: true }).click();
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
> 200 |     await expect(panel.locator('tbody tr')).toHaveCount(2);
      |                                             ^ Error: expect(locator).toHaveCount(expected) failed
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
  232 |       workloadQueries.push(url);
  233 |       const namespace = url.searchParams.get('namespace');
  234 |       await route.fulfill({ json: { code: 200, data: { items: [
  235 |         { namespace, kind: 'Deployment', name: `${namespace}-workload` },
  236 |         { namespace: 'other-namespace', kind: 'Deployment', name: 'other-namespace-workload' },
  237 |         { namespace, kind: 'StatefulSet', name: 'other-kind-workload' },
  238 |       ] } } });
  239 |     });
  240 |     await page.goto('/business/service-catalog/instances', { waitUntil: 'domcontentloaded' });
  241 |     await page.getByRole('button', { name: '新增服务实例', exact: true }).click();
  242 |     const dialog = page.getByRole('dialog');
  243 |     await chooseOption(page, dialog, '服务', 'catalog-application-1 / catalog-service-1');
  244 |     await chooseOption(page, dialog, '目标', 'Kubernetes');
  245 |     await formField(page, dialog, 'Kubernetes 集群').locator('.arco-select').click();
  246 |     await expect(page.locator('.arco-select-option:visible').filter({ hasText: 'other-scope-cluster' })).toHaveCount(0);
  247 |     await page.locator('.arco-select-option:visible').filter({ hasText: /^allowed-cluster$/ }).click();
  248 |     await chooseOption(page, dialog, '命名空间', 'ns-one');
  249 |     await chooseOption(page, dialog, '工作负载类型', 'Deployment');
  250 |     await formField(page, dialog, '工作负载名称').locator('.arco-select').click();
  251 |     await expect(page.locator('.arco-select-option:visible').filter({ hasText: /other-namespace-workload|other-kind-workload/ })).toHaveCount(0);
  252 |     await page.locator('.arco-select-option:visible').filter({ hasText: /^ns-one-workload$/ }).click();
  253 |     await chooseOption(page, dialog, '命名空间', 'ns-two');
  254 |     await expect(formField(page, dialog, '工作负载名称').locator('.arco-select')).not.toContainText('ns-one-workload');
  255 |     await expect(formField(page, dialog, '工作负载类型').locator('.arco-select')).not.toContainText('Deployment');
  256 |     await expect(formField(page, dialog, '工作负载名称').locator('.arco-select')).toHaveClass(/arco-select-disabled/);
  257 |     expect(clusterQueries).toHaveLength(1);
  258 |     expect(clusterQueries[0].searchParams.get('businessScopeId')).toBe('41');
  259 |     expect(clusterQueries[0].searchParams.get('pageSize')).toBe('25');
  260 |     expect(workloadQueries).toHaveLength(1);
  261 |     expect(workloadQueries[0].searchParams.get('clusterId')).toBe('11');
  262 |     expect(workloadQueries[0].searchParams.get('namespace')).toBe('ns-one');
  263 |     expect(workloadQueries[0].searchParams.get('kind')).toBe('Deployment');
  264 |     await dialog.getByRole('button', { name: '取消', exact: true }).click();
  265 |   });
  266 |
  267 |   test('instance rebind is explicit, resolves its owner, and submits a versioned complete target', async ({ page }) => {
  268 |     const fixture = await installCatalogFixture(page, 2);
  269 |     const targets: URL[] = [];
  270 |     await page.route(/\/api\/v1\/business\/cmdb\/hosts(?:\?.*)?$/, async (route) => {
  271 |       targets.push(new URL(route.request().url()));
  272 |       await route.fulfill({ json: { code: 200, data: { items: [
  273 |         { id: 201, businessScopeId: 41, hostname: 'allowed-host', ip: '10.0.0.1' },
  274 |         { id: 202, businessScopeId: 42, hostname: 'other-scope-host', ip: '10.0.0.2' },
  275 |       ], total: 2 } } });
  276 |     });
  277 |     await page.goto('/business/service-catalog/instances', { waitUntil: 'domcontentloaded' });
  278 |     const panel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
  279 |     await panel.locator('tbody tr').filter({ hasText: 'catalog-service-1' }).getByRole('button', { name: '编辑', exact: true }).click();
  280 |     const dialog = page.getByRole('dialog');
  281 |     await expect(formField(page, dialog, '当前版本').locator('input')).toHaveAttribute('readonly', '');
  282 |     expect(targets).toEqual([]);
  283 |     expect(fixture.reads.some((url) => url.pathname.endsWith('/services/1'))).toBe(false);
  284 |     await formField(page, dialog, '更换绑定目标').getByRole('switch').click();
  285 |     await formField(page, dialog, 'CMDB 主机').locator('.arco-select').click();
  286 |     await expect(page.locator('.arco-select-option:visible').filter({ hasText: 'other-scope-host' })).toHaveCount(0);
  287 |     await page.locator('.arco-select-option:visible').filter({ hasText: /^allowed-host \(10\.0\.0\.1\)$/ }).click();
  288 |     await formField(page, dialog, '版本').locator('input').fill('2.1.0');
  289 |     await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
  290 |     await expect(dialog).toBeHidden();
  291 |     expect(fixture.writes.at(-1)?.data).toEqual({ desiredVersion: '2.1.0', expectedLifecycleVersion: 7, target: { targetType: 'vm', hostId: 201 } });
  292 |     expect(fixture.reads.some((url) => url.pathname.endsWith('/services/1'))).toBe(true);
  293 |     expect(targets[0].searchParams.get('businessScopeId')).toBe('41');
  294 |     await expectNoVisibleServiceI18nKeys(page);
  295 |   });
  296 |
  297 |   test('a stale instance update retains the form and the user version input', async ({ page }) => {
  298 |     await installCatalogFixture(page, 2);
  299 |     const conflicts: Record<string, unknown>[] = [];
  300 |     await page.route('**/api/v1/business/service/instances/1', async (route) => {
```
