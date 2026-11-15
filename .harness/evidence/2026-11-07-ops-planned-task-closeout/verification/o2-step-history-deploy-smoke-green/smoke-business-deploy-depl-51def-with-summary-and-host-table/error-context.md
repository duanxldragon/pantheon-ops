# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy detail page loads with summary and host table
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:329:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.page-container').filter({ has: locator('.system-page-hero') })
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('.page-container').filter({ has: locator('.system-page-hero') }) with timeout 10000ms
  - waiting for locator('.page-container').filter({ has: locator('.system-page-hero') })

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
  308 |       await expect(templateKeywordInput).toBeVisible();
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
> 388 |       await expect(page.locator('.page-container').filter({ has: page.locator('.system-page-hero') })).toBeVisible();
      |                                                                                                        ^ Error: expect(locator).toBeVisible() failed
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
  409 |
  410 |     const scope = await expectSuccess<BizScopeRow>(
  411 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  412 |         headers,
  413 |         data: {
  414 |           code: token,
  415 |           name: token,
  416 |           environment: 'dev',
  417 |           status: 'active',
  418 |         },
  419 |       }),
  420 |     );
  421 |
  422 |     const host = await expectSuccess<HostRow>(
  423 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  424 |         headers,
  425 |         data: {
  426 |           hostname: `${token}-host`,
  427 |           ip: `10.246.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
  428 |           os: 'linux',
  429 |           businessScopeId: scope.id,
  430 |           labels: [{ key: 'biz', val: token }],
  431 |         },
  432 |       }),
  433 |     );
  434 |
  435 |     const pkg = await expectSuccess<PackageRow>(
  436 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  437 |         headers,
  438 |         data: {
  439 |           name: token,
  440 |           version: '1.0.0',
  441 |           installCommand: 'echo install',
  442 |           status: 'enabled',
  443 |         },
  444 |       }),
  445 |     );
  446 |
  447 |     const task = await expectSuccess<TaskRow>(
  448 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  449 |         headers,
  450 |         data: {
  451 |           name: `${token}-task`,
  452 |           packageId: pkg.id,
  453 |           businessScopeId: scope.id,
  454 |           targetType: 'host',
  455 |           targetIds: [host.id],
  456 |           executorType: 'manual',
  457 |         },
  458 |       }),
  459 |     );
  460 |
  461 |     try {
  462 |       const started = await expectSuccess<TaskRow>(
  463 |         await startReviewedTask(request, task.id, { headers }),
  464 |       );
  465 |       expect(started.hosts).toHaveLength(1);
  466 |
  467 |       await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'domcontentloaded' });
  468 |       await expect(page.getByRole('button', { name: '标记失败' }).first()).toBeVisible();
  469 |       await page.getByRole('button', { name: '标记失败' }).first().click();
  470 |
  471 |       const modal = page.locator('.arco-modal:visible').last();
  472 |       await expect(modal).toBeVisible();
  473 |       await modal.getByRole('button', { name: '保存' }).click();
  474 |       await expect(modal).toContainText('标记失败时必须填写错误原因');
  475 |
  476 |       await modal.getByRole('textbox', { name: '错误信息' }).fill('manual smoke failure reason');
  477 |       await modal.getByRole('button', { name: '保存' }).click();
  478 |       await expect(modal).toBeHidden();
  479 |       await expect(page.getByText('manual smoke failure reason').first()).toBeVisible();
  480 |       await page.screenshot({ path: testInfo.outputPath('deploy-mark-failed-reason-required.png'), fullPage: true });
  481 |     } finally {
  482 |       await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
  483 |       await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  484 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  485 |     }
  486 |   });
  487 |
  488 |   test('deploy template supports script steps and task params render in detail', async ({ page, request }, testInfo) => {
```
