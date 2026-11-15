# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy detail edit action routes back to list and opens edit modal
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:608:3

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
  569 |
  570 |     const task = await expectSuccess<TaskRow>(
  571 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  572 |         headers,
  573 |         data: {
  574 |           name: `${token}-task`,
  575 |           templateId: template.id,
  576 |           businessScopeId: scope.id,
  577 |           targetType: 'host',
  578 |           targetIds: [host.id],
  579 |           executorType: 'manual',
  580 |           templateParams: {
  581 |             installRoot: '/data/nginx-custom',
  582 |             serviceName: 'nginx-custom',
  583 |             configRoot: '/data/nginx-custom/conf',
  584 |           },
  585 |         },
  586 |       }),
  587 |     );
  588 |
  589 |     try {
  590 |       await page.goto('/business/deploy/template', { waitUntil: 'networkidle' });
  591 |       await page.locator('.search-toolbar input').first().fill(token);
  592 |       await expect(page.getByText(token, { exact: true }).first()).toBeVisible();
  593 |       await page.screenshot({ path: testInfo.outputPath('deploy-script-template-list.png'), fullPage: true });
  594 |
  595 |       await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'networkidle' });
  596 |       await expect(page.getByText('configRoot', { exact: true })).toBeVisible();
  597 |       await expect(page.getByText('/data/nginx-custom/conf', { exact: true })).toBeVisible();
  598 |       await expect(page.getByText('/data/nginx-custom', { exact: true })).toBeVisible();
  599 |       await page.screenshot({ path: testInfo.outputPath('deploy-script-template-task-detail.png'), fullPage: true });
  600 |     } finally {
  601 |       await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
  602 |       await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  603 |       await request.delete(`${apiBaseUrl}/business/deploy/templates/${template.id}`, { headers }).catch(() => undefined);
  604 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  605 |     }
  606 |   });
  607 |
  608 |   test('deploy detail edit action routes back to list and opens edit modal', async ({ page, request }, testInfo) => {
  609 |     const login = await loginByApi(request, adminCredentials);
  610 |     await installClientSession(page, login);
  611 |     const headers = apiRequestHeaders(login);
  612 |     const token = `deploy-detail-edit-${Date.now()}`;
  613 |
  614 |     const scope = await expectSuccess<BizScopeRow>(
  615 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  616 |         headers,
  617 |         data: {
  618 |           code: token,
  619 |           name: `${token}-scope`,
  620 |           environment: 'dev',
  621 |           status: 'active',
  622 |         },
  623 |       }),
  624 |     );
  625 |
  626 |     const host = await expectSuccess<HostRow>(
  627 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  628 |         headers,
  629 |         data: {
  630 |           hostname: `${token}-host`,
  631 |           ip: `10.251.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
  632 |           os: 'linux',
  633 |           businessScopeId: scope.id,
  634 |           labels: [{ key: 'biz', val: token }],
  635 |         },
  636 |       }),
  637 |     );
  638 |
  639 |     const pkg = await expectSuccess<PackageRow>(
  640 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  641 |         headers,
  642 |         data: {
  643 |           name: token,
  644 |           version: '1.0.0',
  645 |           installCommand: 'echo install',
  646 |           status: 'enabled',
  647 |         },
  648 |       }),
  649 |     );
  650 |
  651 |     const task = await expectSuccess<TaskRow>(
  652 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  653 |         headers,
  654 |         data: {
  655 |           name: `${token}-task`,
  656 |           packageId: pkg.id,
  657 |           businessScopeId: scope.id,
  658 |           targetType: 'host',
  659 |           targetIds: [host.id],
  660 |           executorType: 'manual',
  661 |           remark: 'detail edit smoke',
  662 |         },
  663 |       }),
  664 |     );
  665 |     expect(task.status).toBe('draft');
  666 |
  667 |     try {
  668 |       await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'networkidle' });
> 669 |       await expect(page.getByText('任务级操作', { exact: true })).toBeVisible();
      |                                                              ^ Error: expect(locator).toBeVisible() failed
  670 |       await page.getByRole('button', { name: '编辑' }).click();
  671 |
  672 |       await expect(page).toHaveURL(new RegExp(`/business/deploy/task\\?editId=${task.id}$`));
  673 |       const modal = page.locator('.arco-modal').last();
  674 |       await expect(modal).toBeVisible();
  675 |       await expect(modal).toContainText('编辑任务');
  676 |       await expect(modal.locator('input').first()).toHaveValue(`${token}-task`);
  677 |       await modal.locator('textarea').fill('detail edit smoke updated');
  678 |       await modal.getByRole('button', { name: '保存' }).click();
  679 |       await expect(modal).toBeHidden();
  680 |       await expect(page).toHaveURL(/\/business\/deploy\/task$/);
  681 |
  682 |       const detail = await expectSuccess<TaskRow>(
  683 |         await request.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`, { headers }),
  684 |       );
  685 |       expect(detail.remark).toBe('detail edit smoke updated');
  686 |       await page.screenshot({ path: testInfo.outputPath('deploy-task-detail-edit-modal.png'), fullPage: true });
  687 |     } finally {
  688 |       await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
  689 |       await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  690 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  691 |     }
  692 |   });
  693 |
  694 |   test('targeted smoke closes cmdb, bizscope, deploy with uploaded source package and visible task trace', async ({ page, request }, testInfo) => {
  695 |     const login = await loginByApi(request, adminCredentials);
  696 |     await installClientSession(page, login);
  697 |     const headers = apiRequestHeaders(login);
  698 |     const token = `deploy-closed-loop-${Date.now()}`;
  699 |     // Keep the synthetic SSH target unique across retries; CMDB enforces IP uniqueness.
  700 |     const smokeIp = `127.0.0.${Math.floor(Math.random() * 200) + 20}`;
  701 |     const sourceBuffer = Buffer.from('fake nginx source archive for smoke', 'utf8');
  702 |
  703 |     const scope = await expectSuccess<BizScopeRow>(
  704 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  705 |         headers,
  706 |         data: {
  707 |           code: token,
  708 |           name: `${token}-his-dev`,
  709 |           environment: 'dev',
  710 |           status: 'active',
  711 |         },
  712 |       }),
  713 |     );
  714 |
  715 |     const host = await expectSuccess<HostRow>(
  716 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  717 |         headers,
  718 |         data: {
  719 |           hostname: `${token}-host`,
  720 |           ip: smokeIp,
  721 |           sshPort: 1,
  722 |           os: 'linux',
  723 |           businessScopeId: scope.id,
  724 |           labels: [
  725 |             { key: 'biz', val: token },
  726 |             { key: 'env', val: 'dev' },
  727 |           ],
  728 |           remark: token,
  729 |         },
  730 |       }),
  731 |     );
  732 |     expect(host.status).toBe('assigned');
  733 |
  734 |     const uploadResponse = await request.post(`${apiBaseUrl}/system/upload?scope=deploy/package`, {
  735 |       headers,
  736 |       multipart: {
  737 |         file: {
  738 |           name: 'nginx-1.30.2.tar.gz',
  739 |           mimeType: 'application/gzip',
  740 |           buffer: sourceBuffer,
  741 |         },
  742 |       },
  743 |     });
  744 |     const uploadPayload = await expectSuccess<{ objectKey: string; originalName: string; url: string }>(uploadResponse);
  745 |     expect(uploadPayload.originalName).toBe('nginx-1.30.2.tar.gz');
  746 |
  747 |     const pkg = await expectSuccess<PackageRow>(
  748 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  749 |         headers,
  750 |         data: {
  751 |           name: `${token}-nginx`,
  752 |           version: '1.30.2',
  753 |           executionMode: 'fixed',
  754 |           templateCode: 'nginx_systemd',
  755 |           templateConfig: { scenario: 'systemd' },
  756 |           sourceObjectKey: uploadPayload.objectKey,
  757 |           sourceFileName: uploadPayload.originalName,
  758 |           sourceUrl: uploadPayload.url,
  759 |           uninstallCommand: 'systemctl stop nginx || true',
  760 |           status: 'enabled',
  761 |         },
  762 |       }),
  763 |     );
  764 |     expect(pkg.sourceFileName).toBe('nginx-1.30.2.tar.gz');
  765 |
  766 |     const task = await expectSuccess<TaskRow>(
  767 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  768 |         headers,
  769 |         data: {
```
