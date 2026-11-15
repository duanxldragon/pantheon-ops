# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy template detail action opens a modal without leaving the list page
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:245:3

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
  206 |     const login = await loginByApi(request, adminCredentials);
  207 |     await installClientSession(page, login);
  208 |     const headers = apiRequestHeaders(login);
  209 |     const token = `deploy-package-modal-${Date.now()}`;
  210 |
  211 |     const pkg = await expectSuccess<PackageRow>(
  212 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  213 |         headers,
  214 |         data: {
  215 |           name: token,
  216 |           version: '1.30.2',
  217 |           executionMode: 'fixed',
  218 |           templateCode: 'nginx_systemd',
  219 |           sourceFileName: 'nginx-1.30.2.tar.gz',
  220 |           sourceUrl: `https://example.invalid/${token}.tar.gz`,
  221 |           status: 'enabled',
  222 |         },
  223 |       }),
  224 |     );
  225 |
  226 |     try {
  227 |       await page.goto('/business/deploy/package', { waitUntil: 'networkidle' });
  228 |       await expect(page).toHaveURL(/\/business\/deploy\/package$/);
  229 |       const row = page.getByRole('row').filter({ hasText: token }).first();
  230 |       await expect(row).toBeVisible();
  231 |       await row.getByRole('button', { name: '详情' }).click();
  232 |
  233 |       const modal = page.locator('.arco-modal:visible').last();
  234 |       await expect(modal).toBeVisible();
  235 |       await expect(modal).toContainText(token);
  236 |       await expect(modal).toContainText('nginx-1.30.2.tar.gz');
  237 |       await expect(modal).toContainText('模板说明');
  238 |       await expect(page).toHaveURL(/\/business\/deploy\/package$/);
  239 |       await page.screenshot({ path: testInfo.outputPath('deploy-package-detail-modal.png'), fullPage: true });
  240 |     } finally {
  241 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  242 |     }
  243 |   });
  244 |
  245 |   test('deploy template detail action opens a modal without leaving the list page', async ({ page, request }, testInfo) => {
  246 |     const login = await loginByApi(request, adminCredentials);
  247 |     await installClientSession(page, login);
  248 |     const headers = apiRequestHeaders(login);
  249 |     const token = `deploy-template-modal-${Date.now()}`;
  250 |
  251 |     const pkg = await expectSuccess<PackageRow>(
  252 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  253 |         headers,
  254 |         data: {
  255 |           name: `${token}-pkg`,
  256 |           version: '1.30.2',
  257 |           executionMode: 'fixed',
  258 |           templateCode: 'nginx_systemd',
  259 |           sourceFileName: 'nginx-1.30.2.tar.gz',
  260 |           status: 'enabled',
  261 |         },
  262 |       }),
  263 |     );
  264 |
  265 |     const template = await expectSuccess<TemplateRow>(
  266 |       await request.post(`${apiBaseUrl}/business/deploy/templates`, {
  267 |         headers,
  268 |         data: {
  269 |           name: token,
  270 |           version: 'v1',
  271 |           packageId: pkg.id,
  272 |           executionMode: 'orchestrated',
  273 |           defaultAction: 'install',
  274 |           status: 'enabled',
  275 |           parameterSchema: {
  276 |             installRoot: '/data/nginx',
  277 |           },
  278 |           steps: [
  279 |             {
  280 |               stepCode: 'prepare',
  281 |               stepName: '准备目录',
  282 |               stepType: 'script',
  283 |               action: 'install',
  284 |               stepConfig: {
  285 |                 script: 'mkdir -p /data/nginx',
  286 |               },
  287 |               sort: 1,
  288 |             },
  289 |             {
  290 |               stepCode: 'install',
  291 |               stepName: '安装组件',
  292 |               stepType: 'package',
  293 |               action: 'install',
  294 |               packageId: pkg.id,
  295 |               sort: 2,
  296 |             },
  297 |           ],
  298 |         },
  299 |       }),
  300 |     );
  301 |
  302 |     try {
  303 |       await page.goto('/business/deploy/template', { waitUntil: 'networkidle' });
  304 |       await expect(page).toHaveURL(/\/business\/deploy\/template$/);
  305 |       const templateKeywordInput = page.locator('.search-toolbar input').first();
> 306 |       await expect(templateKeywordInput).toBeVisible();
      |                                          ^ Error: expect(locator).toBeVisible() failed
  307 |       await templateKeywordInput.fill(token);
  308 |
  309 |       const row = page.getByRole('row').filter({ hasText: token }).first();
  310 |       await expect(row).toBeVisible();
  311 |       await row.getByRole('button', { name: '详情' }).click();
  312 |
  313 |       const modal = page.locator('.arco-modal:visible').last();
  314 |       await expect(modal).toBeVisible();
  315 |       await expect(modal).toContainText(token);
  316 |       await expect(modal).toContainText('准备目录');
  317 |       await expect(modal).toContainText('安装组件');
  318 |       await expect(modal).toContainText('/data/nginx');
  319 |       await expect(page).toHaveURL(/\/business\/deploy\/template$/);
  320 |       await page.screenshot({ path: testInfo.outputPath('deploy-template-detail-modal.png'), fullPage: true });
  321 |     } finally {
  322 |       await request.delete(`${apiBaseUrl}/business/deploy/templates/${template.id}`, { headers }).catch(() => undefined);
  323 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  324 |     }
  325 |   });
  326 |
  327 |   test('deploy detail page loads with summary and host table', async ({ page, request }, testInfo) => {
  328 |     const login = await loginByApi(request, adminCredentials);
  329 |     await installClientSession(page, login);
  330 |     const headers = apiRequestHeaders(login);
  331 |     const token = `deploy-detail-${Date.now()}`;
  332 |     const scope = await expectSuccess<BizScopeRow>(
  333 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  334 |         headers,
  335 |         data: {
  336 |           code: token,
  337 |           name: token,
  338 |           environment: 'prod',
  339 |           status: 'active',
  340 |         },
  341 |       }),
  342 |     );
  343 |
  344 |     const host = await expectSuccess<HostRow>(
  345 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  346 |         headers,
  347 |         data: {
  348 |           hostname: token,
  349 |           ip: `10.244.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
  350 |           os: 'linux',
  351 |           businessScopeId: scope.id,
  352 |           labels: [{ key: 'biz', val: token }],
  353 |         },
  354 |       }),
  355 |     );
  356 |     expect(host.status).toBe('assigned');
  357 |
  358 |     const pkg = await expectSuccess<PackageRow>(
  359 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  360 |         headers,
  361 |         data: {
  362 |           name: token,
  363 |           version: '1.0.0',
  364 |           installCommand: 'echo install',
  365 |           status: 'enabled',
  366 |         },
  367 |       }),
  368 |     );
  369 |
  370 |     const task = await expectSuccess<TaskRow>(
  371 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  372 |         headers,
  373 |         data: {
  374 |           name: token,
  375 |           packageId: pkg.id,
  376 |           businessScopeId: scope.id,
  377 |           targetType: 'host',
  378 |           targetIds: [host.id],
  379 |           executorType: 'manual',
  380 |         },
  381 |       }),
  382 |     );
  383 |
  384 |     try {
  385 |       await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'networkidle' });
  386 |       await expect(page.locator('.page-container')).toBeVisible();
  387 |       await expect(page.locator('.system-page-hero')).toBeVisible();
  388 |       await expect(page.locator('.system-list__table-card')).toBeVisible();
  389 |       await expect(page.locator('.page-header')).toBeVisible();
  390 |       await expect(page.getByText('任务状态流转', { exact: true })).toBeVisible();
  391 |       await expect(page.locator('.deploy-page__status-flow')).toBeVisible();
  392 |       await expect(page.getByText('任务级操作', { exact: true })).toBeVisible();
  393 |       await expect(page.getByText('主机执行明细', { exact: true })).toBeVisible();
  394 |       await page.screenshot({ path: testInfo.outputPath('deploy-task-detail.png'), fullPage: true });
  395 |     } finally {
  396 |       await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers });
  397 |       await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  398 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  399 |     }
  400 |   });
  401 |
  402 |   test('deploy detail requires a reason when marking host failure', async ({ page, request }, testInfo) => {
  403 |     const login = await loginByApi(request, adminCredentials);
  404 |     await installClientSession(page, login);
  405 |     const headers = apiRequestHeaders(login);
  406 |     const token = `deploy-mark-failed-${Date.now()}`;
```
