# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy detail edit action routes back to list and opens edit modal
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:610:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByText('任务级操作', { exact: true })
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" getByText('任务级操作', { exact: true }) with timeout 10000ms
  - waiting for getByText('任务级操作', { exact: true })

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
  571 |
  572 |     const task = await expectSuccess<TaskRow>(
  573 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  574 |         headers,
  575 |         data: {
  576 |           name: `${token}-task`,
  577 |           templateId: template.id,
  578 |           businessScopeId: scope.id,
  579 |           targetType: 'host',
  580 |           targetIds: [host.id],
  581 |           executorType: 'manual',
  582 |           templateParams: {
  583 |             installRoot: '/data/nginx-custom',
  584 |             serviceName: 'nginx-custom',
  585 |             configRoot: '/data/nginx-custom/conf',
  586 |           },
  587 |         },
  588 |       }),
  589 |     );
  590 |
  591 |     try {
  592 |       await page.goto('/business/deploy/template', { waitUntil: 'networkidle' });
  593 |       await page.locator('.search-toolbar input').first().fill(token);
  594 |       await expect(page.getByText(token, { exact: true }).first()).toBeVisible();
  595 |       await page.screenshot({ path: testInfo.outputPath('deploy-script-template-list.png'), fullPage: true });
  596 |
  597 |       await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'networkidle' });
  598 |       await expect(page.getByText('configRoot', { exact: true })).toBeVisible();
  599 |       await expect(page.getByText('/data/nginx-custom/conf', { exact: true })).toBeVisible();
  600 |       await expect(page.getByText('/data/nginx-custom', { exact: true })).toBeVisible();
  601 |       await page.screenshot({ path: testInfo.outputPath('deploy-script-template-task-detail.png'), fullPage: true });
  602 |     } finally {
  603 |       await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
  604 |       await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  605 |       await request.delete(`${apiBaseUrl}/business/deploy/templates/${template.id}`, { headers }).catch(() => undefined);
  606 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  607 |     }
  608 |   });
  609 |
  610 |   test('deploy detail edit action routes back to list and opens edit modal', async ({ page, request }, testInfo) => {
  611 |     const login = await loginByApi(request, adminCredentials);
  612 |     await installClientSession(page, login);
  613 |     const headers = apiRequestHeaders(login);
  614 |     const token = `deploy-detail-edit-${Date.now()}`;
  615 |
  616 |     const scope = await expectSuccess<BizScopeRow>(
  617 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  618 |         headers,
  619 |         data: {
  620 |           code: token,
  621 |           name: `${token}-scope`,
  622 |           environment: 'dev',
  623 |           status: 'active',
  624 |         },
  625 |       }),
  626 |     );
  627 |
  628 |     const host = await expectSuccess<HostRow>(
  629 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  630 |         headers,
  631 |         data: {
  632 |           hostname: `${token}-host`,
  633 |           ip: `10.251.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
  634 |           os: 'linux',
  635 |           businessScopeId: scope.id,
  636 |           labels: [{ key: 'biz', val: token }],
  637 |         },
  638 |       }),
  639 |     );
  640 |
  641 |     const pkg = await expectSuccess<PackageRow>(
  642 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  643 |         headers,
  644 |         data: {
  645 |           name: token,
  646 |           version: '1.0.0',
  647 |           installCommand: 'echo install',
  648 |           status: 'enabled',
  649 |         },
  650 |       }),
  651 |     );
  652 |
  653 |     const task = await expectSuccess<TaskRow>(
  654 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  655 |         headers,
  656 |         data: {
  657 |           name: `${token}-task`,
  658 |           packageId: pkg.id,
  659 |           businessScopeId: scope.id,
  660 |           targetType: 'host',
  661 |           targetIds: [host.id],
  662 |           executorType: 'manual',
  663 |           remark: 'detail edit smoke',
  664 |         },
  665 |       }),
  666 |     );
  667 |     expect(task.status).toBe('draft');
  668 |
  669 |     try {
  670 |       await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'networkidle' });
> 671 |       await expect(page.getByText('任务级操作', { exact: true })).toBeVisible();
      |                                                              ^ Error: expect(locator).toBeVisible() failed
  672 |       await page.getByRole('button', { name: '编辑' }).click();
  673 |
  674 |       await expect(page).toHaveURL(new RegExp(`/business/deploy/task\\?editId=${task.id}$`));
  675 |       const modal = page.locator('.arco-modal').last();
  676 |       await expect(modal).toBeVisible();
  677 |       await expect(modal).toContainText('编辑任务');
  678 |       await expect(modal.locator('input').first()).toHaveValue(`${token}-task`);
  679 |       await modal.locator('.arco-form-item').filter({ hasText: '备注' }).locator('textarea').fill('detail edit smoke updated');
  680 |       await modal.getByRole('button', { name: '解析并核对目标' }).click();
  681 |       await expect(modal.locator('.deploy-target-selector').getByRole('status')).toContainText('已解析 1 个目标');
  682 |       await modal.getByRole('button', { name: '保存' }).click();
  683 |       await expect(modal).toBeHidden();
  684 |       await expect(page).toHaveURL(/\/business\/deploy\/task$/);
  685 |
  686 |       const detail = await expectSuccess<TaskRow>(
  687 |         await request.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`, { headers }),
  688 |       );
  689 |       expect(detail.remark).toBe('detail edit smoke updated');
  690 |       // A consumed edit intent must stay closed, while a later visit to the
  691 |       // same task must still open a fresh editor with the persisted values.
  692 |       await page.goto(`/business/deploy/task?editId=${task.id}`, { waitUntil: 'networkidle' });
  693 |       await expect(modal).toBeVisible();
  694 |       await expect(modal.locator('.arco-form-item').filter({ hasText: '备注' }).locator('textarea')).toHaveValue('detail edit smoke updated');
  695 |       await modal.getByRole('button', { name: '取消', exact: true }).click();
  696 |       await expect(modal).toBeHidden();
  697 |       await expect(page).toHaveURL(/\/business\/deploy\/task$/);
  698 |       await page.screenshot({ path: testInfo.outputPath('deploy-task-detail-edit-modal.png'), fullPage: true });
  699 |     } finally {
  700 |       const cleanupHeaders = await verifiedApiHeaders(request, login);
  701 |       await expectSuccess(await request.delete(`${apiBaseUrl}/business/deploy/tasks/${task.id}`, { headers: cleanupHeaders }));
  702 |       await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
  703 |       await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  704 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  705 |     }
  706 |   });
  707 |
  708 |   test('targeted smoke closes cmdb, bizscope, deploy with uploaded source package and visible task trace', async ({ page, request }, testInfo) => {
  709 |     const login = await loginByApi(request, adminCredentials);
  710 |     await installClientSession(page, login);
  711 |     const headers = apiRequestHeaders(login);
  712 |     const token = `deploy-closed-loop-${Date.now()}`;
  713 |     // Keep the synthetic SSH target unique across retries; CMDB enforces IP uniqueness.
  714 |     const smokeIp = `127.0.0.${Math.floor(Math.random() * 200) + 20}`;
  715 |     const sourceBuffer = Buffer.from('fake nginx source archive for smoke', 'utf8');
  716 |
  717 |     const scope = await expectSuccess<BizScopeRow>(
  718 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  719 |         headers,
  720 |         data: {
  721 |           code: token,
  722 |           name: `${token}-his-dev`,
  723 |           environment: 'dev',
  724 |           status: 'active',
  725 |         },
  726 |       }),
  727 |     );
  728 |
  729 |     const host = await expectSuccess<HostRow>(
  730 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  731 |         headers,
  732 |         data: {
  733 |           hostname: `${token}-host`,
  734 |           ip: smokeIp,
  735 |           sshPort: 1,
  736 |           os: 'linux',
  737 |           businessScopeId: scope.id,
  738 |           labels: [
  739 |             { key: 'biz', val: token },
  740 |             { key: 'env', val: 'dev' },
  741 |           ],
  742 |           remark: token,
  743 |         },
  744 |       }),
  745 |     );
  746 |     expect(host.status).toBe('assigned');
  747 |
  748 |     const uploadResponse = await request.post(`${apiBaseUrl}/system/upload?scope=deploy/package`, {
  749 |       headers,
  750 |       multipart: {
  751 |         file: {
  752 |           name: 'nginx-1.30.2.tar.gz',
  753 |           mimeType: 'application/gzip',
  754 |           buffer: sourceBuffer,
  755 |         },
  756 |       },
  757 |     });
  758 |     const uploadPayload = await expectSuccess<{ objectKey: string; originalName: string; url: string }>(uploadResponse);
  759 |     expect(uploadPayload.originalName).toBe('nginx-1.30.2.tar.gz');
  760 |
  761 |     const pkg = await expectSuccess<PackageRow>(
  762 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  763 |         headers,
  764 |         data: {
  765 |           name: `${token}-nginx`,
  766 |           version: '1.30.2',
  767 |           executionMode: 'fixed',
  768 |           templateCode: 'nginx_systemd',
  769 |           templateConfig: { scenario: 'systemd' },
  770 |           sourceObjectKey: uploadPayload.objectKey,
  771 |           sourceFileName: uploadPayload.originalName,
```
