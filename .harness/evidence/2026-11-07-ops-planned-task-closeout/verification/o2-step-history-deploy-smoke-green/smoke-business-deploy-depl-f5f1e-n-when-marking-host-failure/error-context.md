# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy detail requires a reason when marking host failure
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:404:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByRole('button', { name: '标记失败' }).first()
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" getByRole('button', { name: '标记失败' }).first() with timeout 10000ms
  - waiting for getByRole('button', { name: '标记失败' }).first()

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
> 468 |       await expect(page.getByRole('button', { name: '标记失败' }).first()).toBeVisible();
      |                                                                        ^ Error: expect(locator).toBeVisible() failed
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
  489 |     const login = await loginByApi(request, adminCredentials);
  490 |     await installClientSession(page, login);
  491 |     const headers = apiRequestHeaders(login);
  492 |     const token = `deploy-script-template-${Date.now()}`;
  493 |
  494 |     const scope = await expectSuccess<BizScopeRow>(
  495 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  496 |         headers,
  497 |         data: {
  498 |           code: token,
  499 |           name: `${token}-scope`,
  500 |           environment: 'dev',
  501 |           status: 'active',
  502 |         },
  503 |       }),
  504 |     );
  505 |
  506 |     const host = await expectSuccess<HostRow>(
  507 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  508 |         headers,
  509 |         data: {
  510 |           hostname: `${token}-host`,
  511 |           ip: `10.247.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
  512 |           os: 'linux',
  513 |           businessScopeId: scope.id,
  514 |           labels: [{ key: 'biz', val: token }],
  515 |         },
  516 |       }),
  517 |     );
  518 |
  519 |     const pkg = await expectSuccess<PackageRow>(
  520 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  521 |         headers,
  522 |         data: {
  523 |           name: token,
  524 |           version: '1.30.2',
  525 |           installCommand: 'echo install',
  526 |           status: 'enabled',
  527 |         },
  528 |       }),
  529 |     );
  530 |
  531 |     const template = await expectSuccess<TemplateRow>(
  532 |       await request.post(`${apiBaseUrl}/business/deploy/templates`, {
  533 |         headers,
  534 |         data: {
  535 |           name: token,
  536 |           version: 'v1',
  537 |           packageId: pkg.id,
  538 |           executionMode: 'orchestrated',
  539 |           defaultAction: 'install',
  540 |           status: 'enabled',
  541 |           parameterSchema: {
  542 |             installRoot: '/data/nginx',
  543 |             serviceName: 'nginx',
  544 |             configRoot: '/data/nginx/conf',
  545 |           },
  546 |           steps: [
  547 |             {
  548 |               stepCode: 'prepare',
  549 |               stepName: '准备目录',
  550 |               stepType: 'script',
  551 |               action: 'install',
  552 |               stepConfig: {
  553 |                 precheckCommand: 'echo precheck {{hostIp}}',
  554 |                 script: 'echo render {{installRoot}} {{configRoot}}',
  555 |                 postcheckCommand: 'echo postcheck {{serviceName}}',
  556 |               },
  557 |               sort: 1,
  558 |             },
  559 |             {
  560 |               stepCode: 'install',
  561 |               stepName: '安装组件',
  562 |               stepType: 'package',
  563 |               action: 'install',
  564 |               packageId: pkg.id,
  565 |               sort: 2,
  566 |             },
  567 |           ],
  568 |         },
```
