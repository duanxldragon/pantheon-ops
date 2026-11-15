# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy detail requires a reason when marking host failure
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:402:3

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
> 466 |       await expect(page.getByRole('button', { name: '标记失败' }).first()).toBeVisible();
      |                                                                        ^ Error: expect(locator).toBeVisible() failed
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
  487 |     const login = await loginByApi(request, adminCredentials);
  488 |     await installClientSession(page, login);
  489 |     const headers = apiRequestHeaders(login);
  490 |     const token = `deploy-script-template-${Date.now()}`;
  491 |
  492 |     const scope = await expectSuccess<BizScopeRow>(
  493 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  494 |         headers,
  495 |         data: {
  496 |           code: token,
  497 |           name: `${token}-scope`,
  498 |           environment: 'dev',
  499 |           status: 'active',
  500 |         },
  501 |       }),
  502 |     );
  503 |
  504 |     const host = await expectSuccess<HostRow>(
  505 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  506 |         headers,
  507 |         data: {
  508 |           hostname: `${token}-host`,
  509 |           ip: `10.247.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
  510 |           os: 'linux',
  511 |           businessScopeId: scope.id,
  512 |           labels: [{ key: 'biz', val: token }],
  513 |         },
  514 |       }),
  515 |     );
  516 |
  517 |     const pkg = await expectSuccess<PackageRow>(
  518 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  519 |         headers,
  520 |         data: {
  521 |           name: token,
  522 |           version: '1.30.2',
  523 |           installCommand: 'echo install',
  524 |           status: 'enabled',
  525 |         },
  526 |       }),
  527 |     );
  528 |
  529 |     const template = await expectSuccess<TemplateRow>(
  530 |       await request.post(`${apiBaseUrl}/business/deploy/templates`, {
  531 |         headers,
  532 |         data: {
  533 |           name: token,
  534 |           version: 'v1',
  535 |           packageId: pkg.id,
  536 |           executionMode: 'orchestrated',
  537 |           defaultAction: 'install',
  538 |           status: 'enabled',
  539 |           parameterSchema: {
  540 |             installRoot: '/data/nginx',
  541 |             serviceName: 'nginx',
  542 |             configRoot: '/data/nginx/conf',
  543 |           },
  544 |           steps: [
  545 |             {
  546 |               stepCode: 'prepare',
  547 |               stepName: '准备目录',
  548 |               stepType: 'script',
  549 |               action: 'install',
  550 |               stepConfig: {
  551 |                 precheckCommand: 'echo precheck {{hostIp}}',
  552 |                 script: 'echo render {{installRoot}} {{configRoot}}',
  553 |                 postcheckCommand: 'echo postcheck {{serviceName}}',
  554 |               },
  555 |               sort: 1,
  556 |             },
  557 |             {
  558 |               stepCode: 'install',
  559 |               stepName: '安装组件',
  560 |               stepType: 'package',
  561 |               action: 'install',
  562 |               packageId: pkg.id,
  563 |               sort: 2,
  564 |             },
  565 |           ],
  566 |         },
```
