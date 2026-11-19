# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\service\service-pages.spec.ts >> Service business module smoke >> instance rebind is explicit, resolves its owner, and submits a versioned complete target
- Location: tests\smoke\business\service\service-pages.spec.ts:267:3

# Error details

```
Test timeout of 30000ms exceeded.
```

```
Error: locator.click: Test timeout of 30000ms exceeded.
Call log:
  - waiting for getByRole('tabpanel', { name: '服务实例', exact: true }).locator('tbody tr').filter({ hasText: 'catalog-service-1' }).getByRole('button', { name: '编辑', exact: true })

```

# Test source

```ts
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
> 279 |     await panel.locator('tbody tr').filter({ hasText: 'catalog-service-1' }).getByRole('button', { name: '编辑', exact: true }).click();
      |                                                                                                                               ^ Error: locator.click: Test timeout of 30000ms exceeded.
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
  301 |       conflicts.push(route.request().postDataJSON() as Record<string, unknown>);
  302 |       await route.fulfill({ status: 409, json: { code: 409, messageKey: 'business.service.state_stale', message: 'Lifecycle version conflict' } });
  303 |     });
  304 |     await page.goto('/business/service-catalog/instances', { waitUntil: 'domcontentloaded' });
  305 |     const panel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
  306 |     await panel.locator('tbody tr').filter({ hasText: 'catalog-service-1' }).getByRole('button', { name: '编辑', exact: true }).click();
  307 |     const dialog = page.getByRole('dialog');
  308 |     await formField(page, dialog, '版本').locator('input').fill('3.0.0');
  309 |     await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
  310 |     await expect.poll(() => conflicts.length).toBe(1);
  311 |     await expect(page.getByText('实例状态已变化，请刷新列表并重新核对后操作。', { exact: true })).toBeVisible();
  312 |     await expect(dialog).toBeVisible();
  313 |     await expect(formField(page, dialog, '版本').locator('input')).toHaveValue('3.0.0');
  314 |     expect(conflicts[0]).toEqual({ desiredVersion: '3.0.0', expectedLifecycleVersion: 7 });
  315 |     await dialog.getByRole('button', { name: '取消', exact: true }).click();
  316 |   });
  317 |
  318 |   test('application, service, and VM instance complete the catalog lifecycle', async ({ page, request }, testInfo) => {
  319 |     await page.setViewportSize({ width: 1440, height: 900 });
  320 |     const consoleErrors: string[] = [];
  321 |     page.on('console', (message) => {
  322 |       if (message.type() === 'error') consoleErrors.push(message.text());
  323 |     });
  324 |     const login = await loginByApi(request, adminCredentials);
  325 |     const headers = apiRequestHeaders(login);
  326 |     const token = `service-smoke-${Date.now()}`;
  327 |     let scopeId = 0;
  328 |     let hostId = 0;
  329 |     let applicationId = 0;
  330 |     let serviceId = 0;
  331 |     let instanceId = 0;
  332 |
  333 |     try {
  334 |       const scope = await expectSuccess<{ id: number }>(
  335 |         await request.post(`${apiBaseUrl}/business/bizscope`, {
  336 |           headers,
  337 |           data: { code: token, name: token, environment: 'prod', status: 'active' },
  338 |         }),
  339 |       );
  340 |       scopeId = scope.id;
  341 |       const host = await expectSuccess<{ id: number }>(
  342 |         await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  343 |           headers,
  344 |           data: {
  345 |             hostname: `${token}-host`,
  346 |             ip: `10.251.${Math.floor(Math.random() * 200) + 20}.${Math.floor(Math.random() * 200) + 20}`,
  347 |             sshPort: 22,
  348 |             os: 'linux',
  349 |             osVersion: 'Ubuntu 24.04',
  350 |             cpuCores: 2,
  351 |             memoryGb: 4,
  352 |             diskGb: 40,
  353 |             labels: [{ key: 'service', val: token }],
  354 |             owner: 'service-smoke',
  355 |             remark: token,
  356 |           },
  357 |         }),
  358 |       );
  359 |       hostId = host.id;
  360 |       await expectSuccess(
  361 |         await request.post(`${apiBaseUrl}/business/bizscope/${scopeId}/hosts/bind`, {
  362 |           headers,
  363 |           data: { hostIds: [hostId] },
  364 |         }),
  365 |       );
  366 |       const application = await expectSuccess<{ id: number }>(
  367 |         await request.post(`${apiBaseUrl}/business/service/applications`, {
  368 |           headers,
  369 |           data: { code: `${token}-app`, name: `${token}-application`, businessScopeId: scopeId, owner: 'service-smoke' },
  370 |         }),
  371 |       );
  372 |       applicationId = application.id;
  373 |       const service = await expectSuccess<{ id: number }>(
  374 |         await request.post(`${apiBaseUrl}/business/service/services`, {
  375 |           headers,
  376 |           data: { applicationId, code: `${token}-svc`, name: `${token}-service`, runtimeType: 'vm' },
  377 |         }),
  378 |       );
  379 |       serviceId = service.id;
```
