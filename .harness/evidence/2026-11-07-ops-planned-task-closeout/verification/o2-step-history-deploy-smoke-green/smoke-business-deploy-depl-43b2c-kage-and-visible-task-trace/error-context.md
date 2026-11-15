# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> targeted smoke closes cmdb, bizscope, deploy with uploaded source package and visible task trace
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:708:3

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
  772 |           sourceUrl: uploadPayload.url,
  773 |           uninstallCommand: 'systemctl stop nginx || true',
  774 |           status: 'enabled',
  775 |         },
  776 |       }),
  777 |     );
  778 |     expect(pkg.sourceFileName).toBe('nginx-1.30.2.tar.gz');
  779 |
  780 |     const task = await expectSuccess<TaskRow>(
  781 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  782 |         headers,
  783 |         data: {
  784 |           name: `${token}-task`,
  785 |           packageId: pkg.id,
  786 |           businessScopeId: scope.id,
  787 |           targetType: 'host',
  788 |           targetIds: [host.id],
  789 |           executorType: 'ssh',
  790 |           templateParams: {
  791 |             installRoot: '/data/nginx',
  792 |             serviceName: 'nginx',
  793 |           },
  794 |           remark: token,
  795 |         },
  796 |       }),
  797 |     );
  798 |     expect(task.status).toBe('draft');
  799 |
  800 |     const credential = await expectSuccess<{ id: number }>(
  801 |       await request.post(`${apiBaseUrl}/business/deploy/credentials`, {
  802 |         headers,
  803 |         data: {
  804 |           name: `${token}-credential`,
  805 |           username: 'root',
  806 |           authMode: 'password',
  807 |           secret: 'smoke-secret',
  808 |         },
  809 |       }),
  810 |     );
  811 |
  812 |     await expectSuccess<TaskRow>(
  813 |       await startReviewedTask(request, task.id, {
  814 |         headers,
  815 |         data: {
  816 |           credentialRefId: credential.id,
  817 |           hostFingerprint: 'SHA256:test',
  818 |         },
  819 |       }),
  820 |     );
  821 |     await expect.poll(async () => {
  822 |       const response = await request.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`, { headers });
  823 |       const payload = await response.json() as ApiEnvelope<TaskRow>;
  824 |       return payload.data.status;
  825 |     }, { timeout: 10_000 }).toBe('failed');
  826 |
  827 |     const detail = await expectSuccess<TaskRow>(
  828 |       await request.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`, { headers }),
  829 |     );
  830 |     expect(detail.businessScopeName).toContain(token);
  831 |     expect(detail.hosts).toHaveLength(1);
  832 |     expect(detail.hosts[0].traceSteps?.length || 0).toBeGreaterThan(0);
  833 |     expect(detail.hosts[0].traceSteps?.map((step) => step.phase)).toEqual(
  834 |       expect.arrayContaining(['start', 'result', 'error']),
  835 |     );
  836 |
  837 |     await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'domcontentloaded' });
> 838 |     await expect(page.locator('.page-container').filter({ has: page.locator('.system-page-hero') })).toBeVisible();
      |                                                                                                      ^ Error: expect(locator).toBeVisible() failed
  839 |     await expect(page.locator('.system-page-hero')).toContainText('目标主机');
  840 |     await expect(page.getByRole('cell', { name: '业务域' })).toBeVisible();
  841 |     await expect(page.getByText('installRoot', { exact: true })).toBeVisible();
  842 |     await expect(page.getByText('/data/nginx', { exact: true })).toBeVisible();
  843 |     await expect(page.getByRole('cell', { name: 'SSH 执行' })).toBeVisible();
  844 |     await expect(page.locator('.system-list__table-card')).toContainText('失败');
  845 |     await expect(page.locator('.system-list__table-card')).toContainText('SSH 连接失败');
  846 |     await expect(page.locator('.system-list__table-card')).not.toContainText('business.deploy.task.sshConnectFailed');
  847 |     await page.screenshot({ path: testInfo.outputPath('deploy-closed-loop-task-detail.png'), fullPage: true });
  848 |
  849 |     const hostDetail = await expectSuccess<HostRow>(
  850 |       await request.get(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }),
  851 |     );
  852 |     expect(hostDetail.status).toBe('assigned');
  853 |
  854 |     await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
  855 |     await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  856 |     await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  857 |     await request.delete(`${apiBaseUrl}/business/deploy/credentials/${credential.id}`, { headers }).catch(() => undefined);
  858 |   });
  859 |
  860 |   test('deploy pages stay within a phone viewport', async ({ page }, testInfo) => {
  861 |     await page.setViewportSize({ width: 390, height: 844 });
  862 |
  863 |     await page.goto('/business/deploy/package', { waitUntil: 'networkidle' });
  864 |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
  865 |     await expect
  866 |       .poll(async () =>
  867 |         page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1),
  868 |       )
  869 |       .toBe(true);
  870 |     await page.screenshot({ path: testInfo.outputPath('deploy-package-list-mobile.png'), fullPage: true });
  871 |
  872 |     await page.goto('/business/deploy/template', { waitUntil: 'networkidle' });
  873 |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
  874 |     await expect
  875 |       .poll(async () =>
  876 |         page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1),
  877 |       )
  878 |       .toBe(true);
  879 |     await page.screenshot({ path: testInfo.outputPath('deploy-template-list-mobile.png'), fullPage: true });
  880 |
  881 |     await page.goto('/business/deploy/task', { waitUntil: 'networkidle' });
  882 |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
  883 |     await expect
  884 |       .poll(async () =>
  885 |         page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1),
  886 |       )
  887 |       .toBe(true);
  888 |     await page.screenshot({ path: testInfo.outputPath('deploy-task-list-mobile.png'), fullPage: true });
  889 |   });
  890 | });
  891 |
```
