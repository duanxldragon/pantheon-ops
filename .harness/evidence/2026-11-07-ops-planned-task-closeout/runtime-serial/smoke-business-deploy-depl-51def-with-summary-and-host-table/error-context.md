# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy detail page loads with summary and host table
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:327:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.page-container')
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('.page-container') with timeout 10000ms
  - waiting for locator('.page-container')

```

```yaml
- complementary:
    - text: P Pantheon Base
    - menu
- button "切换导航"
- list:
    - listitem: 首页
    - listitem: 任务详情
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
  306 |       await expect(templateKeywordInput).toBeVisible();
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
> 386 |       await expect(page.locator('.page-container')).toBeVisible();
      |                                                     ^ Error: expect(locator).toBeVisible() failed
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
  407 |
  408 |     const scope = await expectSuccess<BizScopeRow>(
  409 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  410 |         headers,
  411 |         data: {
  412 |           code: token,
  413 |           name: token,
  414 |           environment: 'dev',
  415 |           status: 'active',
  416 |         },
  417 |       }),
  418 |     );
  419 |
  420 |     const host = await expectSuccess<HostRow>(
  421 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  422 |         headers,
  423 |         data: {
  424 |           hostname: `${token}-host`,
  425 |           ip: `10.246.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
  426 |           os: 'linux',
  427 |           businessScopeId: scope.id,
  428 |           labels: [{ key: 'biz', val: token }],
  429 |         },
  430 |       }),
  431 |     );
  432 |
  433 |     const pkg = await expectSuccess<PackageRow>(
  434 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  435 |         headers,
  436 |         data: {
  437 |           name: token,
  438 |           version: '1.0.0',
  439 |           installCommand: 'echo install',
  440 |           status: 'enabled',
  441 |         },
  442 |       }),
  443 |     );
  444 |
  445 |     const task = await expectSuccess<TaskRow>(
  446 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  447 |         headers,
  448 |         data: {
  449 |           name: `${token}-task`,
  450 |           packageId: pkg.id,
  451 |           businessScopeId: scope.id,
  452 |           targetType: 'host',
  453 |           targetIds: [host.id],
  454 |           executorType: 'manual',
  455 |         },
  456 |       }),
  457 |     );
  458 |
  459 |     try {
  460 |       const started = await expectSuccess<TaskRow>(
  461 |         await request.post(`${apiBaseUrl}/business/deploy/tasks/${task.id}/start`, { headers }),
  462 |       );
  463 |       expect(started.hosts).toHaveLength(1);
  464 |
  465 |       await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'domcontentloaded' });
  466 |       await expect(page.getByRole('button', { name: '标记失败' }).first()).toBeVisible();
  467 |       await page.getByRole('button', { name: '标记失败' }).first().click();
  468 |
  469 |       const modal = page.locator('.arco-modal:visible').last();
  470 |       await expect(modal).toBeVisible();
  471 |       await modal.getByRole('button', { name: '保存' }).click();
  472 |       await expect(modal).toContainText('标记失败时必须填写错误原因');
  473 |
  474 |       await modal.getByRole('textbox', { name: '错误信息' }).fill('manual smoke failure reason');
  475 |       await modal.getByRole('button', { name: '保存' }).click();
  476 |       await expect(modal).toBeHidden();
  477 |       await expect(page.getByText('manual smoke failure reason').first()).toBeVisible();
  478 |       await page.screenshot({ path: testInfo.outputPath('deploy-mark-failed-reason-required.png'), fullPage: true });
  479 |     } finally {
  480 |       await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
  481 |       await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  482 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  483 |     }
  484 |   });
  485 |
  486 |   test('deploy template supports script steps and task params render in detail', async ({ page, request }, testInfo) => {
```
