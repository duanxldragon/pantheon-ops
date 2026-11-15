# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy template detail action opens a modal without leaving the list page
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:247:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.search-toolbar input').first()
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('.search-toolbar input').first() with timeout 10000ms
  - waiting for locator('.search-toolbar input').first()

```

```yaml
- complementary:
    - text: P Pantheon Base
    - menu
- button "切换导航"
- list:
    - listitem: 首页
    - listitem: 任务模板
- button "平台功能台": 搜索菜单、页面、功能… Ctrl K
- button "平台偏好"
- 'button "用户: 用户"': 用 用户
- tablist "已打开页面"
- main:
    - text: 加载失败 当前区块暂时无法返回数据，请重试或稍后再查看。
    - button "重试"
- text: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  208 |     const login = await loginByApi(request, adminCredentials);
  209 |     await installClientSession(page, login);
  210 |     const headers = apiRequestHeaders(login);
  211 |     const token = `deploy-package-modal-${Date.now()}`;
  212 |
  213 |     const pkg = await expectSuccess<PackageRow>(
  214 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  215 |         headers,
  216 |         data: {
  217 |           name: token,
  218 |           version: '1.30.2',
  219 |           executionMode: 'fixed',
  220 |           templateCode: 'nginx_systemd',
  221 |           sourceFileName: 'nginx-1.30.2.tar.gz',
  222 |           sourceUrl: `https://example.invalid/${token}.tar.gz`,
  223 |           status: 'enabled',
  224 |         },
  225 |       }),
  226 |     );
  227 |
  228 |     try {
  229 |       await page.goto('/business/deploy/package', { waitUntil: 'networkidle' });
  230 |       await expect(page).toHaveURL(/\/business\/deploy\/package$/);
  231 |       const row = page.getByRole('row').filter({ hasText: token }).first();
  232 |       await expect(row).toBeVisible();
  233 |       await row.getByRole('button', { name: '详情' }).click();
  234 |
  235 |       const modal = page.locator('.arco-modal:visible').last();
  236 |       await expect(modal).toBeVisible();
  237 |       await expect(modal).toContainText(token);
  238 |       await expect(modal).toContainText('nginx-1.30.2.tar.gz');
  239 |       await expect(modal).toContainText('模板说明');
  240 |       await expect(page).toHaveURL(/\/business\/deploy\/package$/);
  241 |       await page.screenshot({ path: testInfo.outputPath('deploy-package-detail-modal.png'), fullPage: true });
  242 |     } finally {
  243 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  244 |     }
  245 |   });
  246 |
  247 |   test('deploy template detail action opens a modal without leaving the list page', async ({ page, request }, testInfo) => {
  248 |     const login = await loginByApi(request, adminCredentials);
  249 |     await installClientSession(page, login);
  250 |     const headers = apiRequestHeaders(login);
  251 |     const token = `deploy-template-modal-${Date.now()}`;
  252 |
  253 |     const pkg = await expectSuccess<PackageRow>(
  254 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  255 |         headers,
  256 |         data: {
  257 |           name: `${token}-pkg`,
  258 |           version: '1.30.2',
  259 |           executionMode: 'fixed',
  260 |           templateCode: 'nginx_systemd',
  261 |           sourceFileName: 'nginx-1.30.2.tar.gz',
  262 |           status: 'enabled',
  263 |         },
  264 |       }),
  265 |     );
  266 |
  267 |     const template = await expectSuccess<TemplateRow>(
  268 |       await request.post(`${apiBaseUrl}/business/deploy/templates`, {
  269 |         headers,
  270 |         data: {
  271 |           name: token,
  272 |           version: 'v1',
  273 |           packageId: pkg.id,
  274 |           executionMode: 'orchestrated',
  275 |           defaultAction: 'install',
  276 |           status: 'enabled',
  277 |           parameterSchema: {
  278 |             installRoot: '/data/nginx',
  279 |           },
  280 |           steps: [
  281 |             {
  282 |               stepCode: 'prepare',
  283 |               stepName: '准备目录',
  284 |               stepType: 'script',
  285 |               action: 'install',
  286 |               stepConfig: {
  287 |                 script: 'mkdir -p /data/nginx',
  288 |               },
  289 |               sort: 1,
  290 |             },
  291 |             {
  292 |               stepCode: 'install',
  293 |               stepName: '安装组件',
  294 |               stepType: 'package',
  295 |               action: 'install',
  296 |               packageId: pkg.id,
  297 |               sort: 2,
  298 |             },
  299 |           ],
  300 |         },
  301 |       }),
  302 |     );
  303 |
  304 |     try {
  305 |       await page.goto('/business/deploy/template', { waitUntil: 'networkidle' });
  306 |       await expect(page).toHaveURL(/\/business\/deploy\/template$/);
  307 |       const templateKeywordInput = page.locator('.search-toolbar input').first();
> 308 |       await expect(templateKeywordInput).toBeVisible();
      |                                          ^ Error: expect(locator).toBeVisible() failed
  309 |       await templateKeywordInput.fill(token);
  310 |
  311 |       const row = page.getByRole('row').filter({ hasText: token }).first();
  312 |       await expect(row).toBeVisible();
  313 |       await row.getByRole('button', { name: '详情' }).click();
  314 |
  315 |       const modal = page.locator('.arco-modal:visible').last();
  316 |       await expect(modal).toBeVisible();
  317 |       await expect(modal).toContainText(token);
  318 |       await expect(modal).toContainText('准备目录');
  319 |       await expect(modal).toContainText('安装组件');
  320 |       await expect(modal).toContainText('/data/nginx');
  321 |       await expect(page).toHaveURL(/\/business\/deploy\/template$/);
  322 |       await page.screenshot({ path: testInfo.outputPath('deploy-template-detail-modal.png'), fullPage: true });
  323 |     } finally {
  324 |       await request.delete(`${apiBaseUrl}/business/deploy/templates/${template.id}`, { headers }).catch(() => undefined);
  325 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  326 |     }
  327 |   });
  328 |
  329 |   test('deploy detail page loads with summary and host table', async ({ page, request }, testInfo) => {
  330 |     const login = await loginByApi(request, adminCredentials);
  331 |     await installClientSession(page, login);
  332 |     const headers = apiRequestHeaders(login);
  333 |     const token = `deploy-detail-${Date.now()}`;
  334 |     const scope = await expectSuccess<BizScopeRow>(
  335 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  336 |         headers,
  337 |         data: {
  338 |           code: token,
  339 |           name: token,
  340 |           environment: 'prod',
  341 |           status: 'active',
  342 |         },
  343 |       }),
  344 |     );
  345 |
  346 |     const host = await expectSuccess<HostRow>(
  347 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  348 |         headers,
  349 |         data: {
  350 |           hostname: token,
  351 |           ip: `10.244.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
  352 |           os: 'linux',
  353 |           businessScopeId: scope.id,
  354 |           labels: [{ key: 'biz', val: token }],
  355 |         },
  356 |       }),
  357 |     );
  358 |     expect(host.status).toBe('assigned');
  359 |
  360 |     const pkg = await expectSuccess<PackageRow>(
  361 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  362 |         headers,
  363 |         data: {
  364 |           name: token,
  365 |           version: '1.0.0',
  366 |           installCommand: 'echo install',
  367 |           status: 'enabled',
  368 |         },
  369 |       }),
  370 |     );
  371 |
  372 |     const task = await expectSuccess<TaskRow>(
  373 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  374 |         headers,
  375 |         data: {
  376 |           name: token,
  377 |           packageId: pkg.id,
  378 |           businessScopeId: scope.id,
  379 |           targetType: 'host',
  380 |           targetIds: [host.id],
  381 |           executorType: 'manual',
  382 |         },
  383 |       }),
  384 |     );
  385 |
  386 |     try {
  387 |       await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'networkidle' });
  388 |       await expect(page.locator('.page-container').filter({ has: page.locator('.system-page-hero') })).toBeVisible();
  389 |       await expect(page.locator('.system-page-hero')).toBeVisible();
  390 |       await expect(page.locator('.system-list__table-card')).toBeVisible();
  391 |       await expect(page.locator('.page-header')).toBeVisible();
  392 |       await expect(page.getByText('任务状态流转', { exact: true })).toBeVisible();
  393 |       await expect(page.locator('.deploy-page__status-flow')).toBeVisible();
  394 |       await expect(page.getByText('任务级操作', { exact: true })).toBeVisible();
  395 |       await expect(page.getByText('主机执行明细', { exact: true })).toBeVisible();
  396 |       await page.screenshot({ path: testInfo.outputPath('deploy-task-detail.png'), fullPage: true });
  397 |     } finally {
  398 |       await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers });
  399 |       await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  400 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  401 |     }
  402 |   });
  403 |
  404 |   test('deploy detail requires a reason when marking host failure', async ({ page, request }, testInfo) => {
  405 |     const login = await loginByApi(request, adminCredentials);
  406 |     await installClientSession(page, login);
  407 |     const headers = apiRequestHeaders(login);
  408 |     const token = `deploy-mark-failed-${Date.now()}`;
```
