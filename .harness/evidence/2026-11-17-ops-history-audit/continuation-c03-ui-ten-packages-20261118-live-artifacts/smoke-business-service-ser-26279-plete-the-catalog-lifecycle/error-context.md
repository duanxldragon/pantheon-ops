# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\service\service-pages.spec.ts >> Service business module smoke >> application, service, and VM instance complete the catalog lifecycle
- Location: tests\smoke\business\service\service-pages.spec.ts:417:3

# Error details

```
Error: expect(locator).toContainText(expected) failed

Locator: getByRole('tabpanel', { name: '服务实例', exact: true })
Expected substring: "Host #532"
Received string:    "新增服务实例服务环境目标运行目标观测状态健康状态操作service-smoke-1794990500892-serviceprodvm主机 #532unknownunknown编辑协调卡住状态总数1120 条/页"
Timeout: 10000ms

Call log:
  - Expect "toContainText" getByRole('tabpanel', { name: '服务实例', exact: true }) with timeout 10000ms
  - waiting for getByRole('tabpanel', { name: '服务实例', exact: true })
    23 × locator resolved to <div tabindex="0" role="tabpanel" id="arco-tabs-1-panel-2" aria-labelledby="arco-tabs-1-tab-2" class="arco-tabs-content-item arco-tabs-content-item-active">…</div>
       - unexpected value "新增服务实例服务环境目标运行目标观测状态健康状态操作service-smoke-1794990500892-serviceprodvm主机 #532unknownunknown编辑协调卡住状态总数1120 条/页"

```

```yaml
- tabpanel "服务实例":
    - search "筛选":
        - button "新增服务实例"
    - table:
        - rowgroup:
            - row "服务 环境 目标 运行目标 观测状态 健康状态 操作":
                - columnheader "服务"
                - columnheader "环境"
                - columnheader "目标"
                - columnheader "运行目标"
                - columnheader "观测状态"
                - columnheader "健康状态"
                - columnheader "操作"
        - rowgroup:
            - 'row "service-smoke-1794990500892-service prod vm 主机 #532 unknown unknown 编辑 协调卡住状态 删除"':
                - cell "service-smoke-1794990500892-service"
                - cell "prod"
                - cell "vm"
                - 'cell "主机 #532"'
                - cell "unknown"
                - cell "unknown"
                - cell "编辑 协调卡住状态 删除":
                    - button "编辑"
                    - button "协调卡住状态"
                    - button "删除"
    - text: 总数
    - strong: "1"
    - list:
        - listitem "上一页": .
        - listitem "第 1 页": "1"
        - listitem "下一页": .
    - combobox: 20 条/页
```

# Test source

```ts
  399 |       await expect(readOnlyPanel.locator('tbody tr')).toHaveCount(2);
  400 |       await expect(readOnlyPanel.getByRole('button', { name: /新增|编辑|删除/ })).toHaveCount(0);
  401 |       for (const control of await readOnlyPanel.getByRole('switch').all()) await expect(control).toBeDisabled();
  402 |       await capture('read-only');
  403 |
  404 |       fixture.applications.length = 0;
  405 |       fixture.services.length = 0;
  406 |       fixture.instances.length = 0;
  407 |       await page.reload({ waitUntil: 'domcontentloaded' });
  408 |       const emptyPanel = page.getByRole('tabpanel', { name: '应用', exact: true });
  409 |       await expect(emptyPanel).toContainText('暂无应用。');
  410 |       await expect(emptyPanel.getByRole('button', { name: /新增|编辑|删除/ })).toHaveCount(0);
  411 |       await capture('read-only-empty');
  412 |       expect(fixture.writes).toEqual([]);
  413 |       expect(optionalRequests).toEqual([]);
  414 |     });
  415 |   }
  416 |
  417 |   test('application, service, and VM instance complete the catalog lifecycle', async ({ page, request }, testInfo) => {
  418 |     await page.setViewportSize({ width: 1440, height: 900 });
  419 |     const consoleErrors: string[] = [];
  420 |     page.on('console', (message) => {
  421 |       if (message.type() === 'error') consoleErrors.push(message.text());
  422 |     });
  423 |     const login = await loginByApi(request, adminCredentials);
  424 |     const headers = apiRequestHeaders(login);
  425 |     const token = `service-smoke-${Date.now()}`;
  426 |     let scopeId = 0;
  427 |     let hostId = 0;
  428 |     let applicationId = 0;
  429 |     let serviceId = 0;
  430 |     let instanceId = 0;
  431 |
  432 |     try {
  433 |       const scope = await expectSuccess<{ id: number }>(
  434 |         await request.post(`${apiBaseUrl}/business/bizscope`, {
  435 |           headers,
  436 |           data: { code: token, name: token, environment: 'prod', status: 'active' },
  437 |         }),
  438 |       );
  439 |       scopeId = scope.id;
  440 |       const host = await expectSuccess<{ id: number }>(
  441 |         await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  442 |           headers,
  443 |           data: {
  444 |             hostname: `${token}-host`,
  445 |             ip: `10.251.${Math.floor(Math.random() * 200) + 20}.${Math.floor(Math.random() * 200) + 20}`,
  446 |             sshPort: 22,
  447 |             os: 'linux',
  448 |             osVersion: 'Ubuntu 24.04',
  449 |             cpuCores: 2,
  450 |             memoryGb: 4,
  451 |             diskGb: 40,
  452 |             labels: [{ key: 'service', val: token }],
  453 |             owner: 'service-smoke',
  454 |             remark: token,
  455 |           },
  456 |         }),
  457 |       );
  458 |       hostId = host.id;
  459 |       await expectSuccess(
  460 |         await request.post(`${apiBaseUrl}/business/bizscope/${scopeId}/hosts/bind`, {
  461 |           headers,
  462 |           data: { hostIds: [hostId] },
  463 |         }),
  464 |       );
  465 |       const application = await expectSuccess<{ id: number }>(
  466 |         await request.post(`${apiBaseUrl}/business/service/applications`, {
  467 |           headers,
  468 |           data: { code: `${token}-app`, name: `${token}-application`, businessScopeId: scopeId, owner: 'service-smoke' },
  469 |         }),
  470 |       );
  471 |       applicationId = application.id;
  472 |       const service = await expectSuccess<{ id: number }>(
  473 |         await request.post(`${apiBaseUrl}/business/service/services`, {
  474 |           headers,
  475 |           data: { applicationId, code: `${token}-svc`, name: `${token}-service`, runtimeType: 'vm' },
  476 |         }),
  477 |       );
  478 |       serviceId = service.id;
  479 |       const instance = await expectSuccess<{ id: number }>(
  480 |         await request.post(`${apiBaseUrl}/business/service/instances`, {
  481 |           headers,
  482 |           data: { serviceId, environment: 'prod', targetType: 'vm', hostId, desiredVersion: '1.2.0' },
  483 |         }),
  484 |       );
  485 |       instanceId = Number(instance.id);
  486 |       expect(instanceId).toBeGreaterThan(0);
  487 |
  488 |       await page.goto('/business/service-catalog', { waitUntil: 'domcontentloaded' });
  489 |       await expect(page).toHaveURL(/\/business\/service-catalog$/);
  490 |       await expect(page.locator('.governance-summary-bar')).toContainText('服务目录');
  491 |       await expect(page.locator('.system-list__table-card:visible')).toContainText(`${token}-application`);
  492 |
  493 |       await page.getByRole('tab', { name: '服务', exact: true }).click();
  494 |       const servicePanel = page.getByRole('tabpanel', { name: '服务', exact: true });
  495 |       await expect(servicePanel).toContainText(`${token}-service`);
  496 |       await page.getByRole('tab', { name: '服务实例', exact: true }).click();
  497 |       const instancePanel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
  498 |       await expect(instancePanel).toContainText(`${token}-service`);
> 499 |       await expect(instancePanel).toContainText(`Host #${hostId}`);
      |                                   ^ Error: expect(locator).toContainText(expected) failed
  500 |       await expectNoVisibleServiceI18nKeys(page);
  501 |       await page.screenshot({ path: testInfo.outputPath('service-catalog-desktop.png') });
  502 |       expect(consoleErrors).toEqual([]);
  503 |     } finally {
  504 |       const cleanupFailures: string[] = [];
  505 |       const cleanup = async (label: string, responsePromise: Promise<APIResponse>) => {
  506 |         try {
  507 |           const response = await responsePromise;
  508 |           const payload = await response.json().catch(() => null) as ApiEnvelope<unknown> | null;
  509 |           if (!response.ok() || payload?.code !== 200) {
  510 |             cleanupFailures.push(
  511 |               `${label}: HTTP ${response.status()} / code ${payload?.code ?? 'unknown'} / ${payload?.message ?? 'no message'}`,
  512 |             );
  513 |           }
  514 |         } catch (error) {
  515 |           cleanupFailures.push(`${label}: ${error instanceof Error ? error.message : String(error)}`);
  516 |         }
  517 |       };
  518 |       if (!instanceId && serviceId) {
  519 |         try {
  520 |           const response = await request.get(
  521 |             `${apiBaseUrl}/business/service/instances?serviceId=${serviceId}&page=1&pageSize=20`,
  522 |             { headers },
  523 |           );
  524 |           const payload = await response.json() as ApiEnvelope<{ items: Array<{ id: number }> }>;
  525 |           instanceId = Number(payload.data?.items?.[0]?.id ?? 0);
  526 |           if (!response.ok() || payload.code !== 200 || !instanceId) {
  527 |             cleanupFailures.push('resolve instance: unable to find the created service instance');
  528 |           }
  529 |         } catch (error) {
  530 |           cleanupFailures.push(`resolve instance: ${error instanceof Error ? error.message : String(error)}`);
  531 |         }
  532 |       }
  533 |       if (instanceId) await cleanup('instance', request.delete(`${apiBaseUrl}/business/service/instances/${instanceId}`, { headers }));
  534 |       if (serviceId) await cleanup('service', request.delete(`${apiBaseUrl}/business/service/services/${serviceId}`, { headers }));
  535 |       if (applicationId) await cleanup('application', request.delete(`${apiBaseUrl}/business/service/applications/${applicationId}`, { headers }));
  536 |       if (scopeId && hostId) await cleanup('unbind host', request.delete(`${apiBaseUrl}/business/bizscope/${scopeId}/hosts/${hostId}`, { headers }));
  537 |       if (hostId) await cleanup('host', request.delete(`${apiBaseUrl}/business/cmdb/hosts/${hostId}`, { headers }));
  538 |       if (scopeId) await cleanup('scope', request.delete(`${apiBaseUrl}/business/bizscope/${scopeId}`, { headers }));
  539 |       expect(cleanupFailures).toEqual([]);
  540 |     }
  541 |   });
  542 |
  543 |   test('service page remains usable in a phone viewport', async ({ page }, testInfo) => {
  544 |     await page.setViewportSize({ width: 390, height: 844 });
  545 |     await page.goto('/business/service-catalog', { waitUntil: 'domcontentloaded' });
  546 |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
  547 |     await expect(page.getByRole('tab', { name: '服务实例', exact: true })).toBeVisible();
  548 |     await expect
  549 |       .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
  550 |       .toBe(true);
  551 |     await page.screenshot({ path: testInfo.outputPath('service-list-mobile.png') });
  552 |   });
  553 | });
  554 |
```
