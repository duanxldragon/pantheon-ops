# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> targeted smoke closes cmdb, bizscope, deploy with uploaded source package and visible task trace
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:694:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.page-container')
Expected: visible
Error: strict mode violation: locator('.page-container') resolved to 2 elements:
    1) <div data-testid="route-content-fallback" class="page-container route-content-fallback">…</div> aka getByTestId('route-content-fallback').first()
    2) <div data-testid="route-content-fallback" class="page-container route-content-fallback">…</div> aka getByTestId('route-content-fallback').nth(1)

Call log:
  - Expect "toBeVisible" locator('.page-container') with timeout 10000ms
  - waiting for locator('.page-container')

```

# Page snapshot

```yaml
- generic [ref=e3]:
    - complementary [ref=e4]:
        - generic [ref=e5]:
            - generic [ref=e6]:
                - generic [ref=e7]: P
                - generic [ref=e8]: Pantheon Base
            - menu [ref=e12]:
                - generic [ref=e13]:
                    - menuitem [ref=e14] [cursor=pointer]:
                        - button "工作台" [ref=e15]
                    - button "CMDB" [ref=e25] [cursor=pointer]
                    - menuitem [ref=e37] [cursor=pointer]:
                        - button "业务域" [ref=e38]
                    - menuitem [ref=e47] [cursor=pointer]:
                        - button "服务目录" [ref=e48]
                    - button "安装部署" [ref=e60] [cursor=pointer]
                    - button "Kubernetes" [ref=e72] [cursor=pointer]
                    - button "可观测中心" [ref=e84] [cursor=pointer]
                    - button "访问控制" [ref=e96] [cursor=pointer]
                    - button "组织架构" [ref=e110] [cursor=pointer]
                    - button "安全审计" [ref=e125] [cursor=pointer]
                    - button "低代码平台" [ref=e137] [cursor=pointer]
                    - button "平台配置" [ref=e149] [cursor=pointer]
                    - menuitem [ref=e158] [cursor=pointer]:
                        - button "文件分发" [ref=e159]
                    - menuitem [ref=e168] [cursor=pointer]:
                        - button "ArgoCD" [ref=e169]
    - generic [ref=e175]:
        - generic [ref=e176]:
            - generic [ref=e177]:
                - button "切换导航" [ref=e178] [cursor=pointer]
                - list [ref=e182]:
                    - listitem [ref=e183]: 首页
                    - listitem [ref=e187]: 任务详情
            - generic [ref=e188]:
                - button "平台功能台" [ref=e190]:
                    - generic [ref=e193]: 搜索菜单、页面、功能…
                    - generic [ref=e194]: Ctrl K
                - button "通知中心" [ref=e196] [cursor=pointer]
                - button "平台偏好" [ref=e200] [cursor=pointer]
                - 'button "用户: Administrator" [ref=e205] [cursor=pointer]':
                    - generic [ref=e206]: A
                    - generic [ref=e208]:
                        - generic [ref=e209]: Administrator
                        - generic [ref=e210]: admin
        - tablist "已打开页面" [ref=e211]:
            - tab "工作台" [ref=e212] [cursor=pointer]
            - tab "任务详情" [selected] [ref=e216] [cursor=pointer]
        - main [ref=e218]
        - generic [ref=e226]: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
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
  770 |           name: `${token}-task`,
  771 |           packageId: pkg.id,
  772 |           businessScopeId: scope.id,
  773 |           targetType: 'host',
  774 |           targetIds: [host.id],
  775 |           executorType: 'ssh',
  776 |           templateParams: {
  777 |             installRoot: '/data/nginx',
  778 |             serviceName: 'nginx',
  779 |           },
  780 |           remark: token,
  781 |         },
  782 |       }),
  783 |     );
  784 |     expect(task.status).toBe('draft');
  785 |
  786 |     const credential = await expectSuccess<{ id: number }>(
  787 |       await request.post(`${apiBaseUrl}/business/deploy/credentials`, {
  788 |         headers,
  789 |         data: {
  790 |           name: `${token}-credential`,
  791 |           username: 'root',
  792 |           authMode: 'password',
  793 |           secret: 'smoke-secret',
  794 |         },
  795 |       }),
  796 |     );
  797 |
  798 |     await expectSuccess<TaskRow>(
  799 |       await request.post(`${apiBaseUrl}/business/deploy/tasks/${task.id}/start`, {
  800 |         headers,
  801 |         data: {
  802 |           credentialRefId: credential.id,
  803 |           hostFingerprint: 'SHA256:test',
  804 |         },
  805 |       }),
  806 |     );
  807 |     await expect.poll(async () => {
  808 |       const response = await request.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`, { headers });
  809 |       const payload = await response.json() as ApiEnvelope<TaskRow>;
  810 |       return payload.data.status;
  811 |     }, { timeout: 10_000 }).toBe('failed');
  812 |
  813 |     const detail = await expectSuccess<TaskRow>(
  814 |       await request.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`, { headers }),
  815 |     );
  816 |     expect(detail.businessScopeName).toContain(token);
  817 |     expect(detail.hosts).toHaveLength(1);
  818 |     expect(detail.hosts[0].traceSteps?.length || 0).toBeGreaterThan(0);
  819 |     expect(detail.hosts[0].traceSteps?.map((step) => step.phase)).toEqual(
  820 |       expect.arrayContaining(['start', 'result', 'error']),
  821 |     );
  822 |
  823 |     await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'domcontentloaded' });
> 824 |     await expect(page.locator('.page-container')).toBeVisible();
      |                                                   ^ Error: expect(locator).toBeVisible() failed
  825 |     await expect(page.locator('.system-page-hero')).toContainText('目标主机');
  826 |     await expect(page.getByRole('cell', { name: '业务域' })).toBeVisible();
  827 |     await expect(page.getByText('installRoot', { exact: true })).toBeVisible();
  828 |     await expect(page.getByText('/data/nginx', { exact: true })).toBeVisible();
  829 |     await expect(page.getByRole('cell', { name: 'SSH 执行' })).toBeVisible();
  830 |     await expect(page.locator('.system-list__table-card')).toContainText('失败');
  831 |     await expect(page.locator('.system-list__table-card')).toContainText('SSH 连接失败');
  832 |     await expect(page.locator('.system-list__table-card')).not.toContainText('business.deploy.task.sshConnectFailed');
  833 |     await page.screenshot({ path: testInfo.outputPath('deploy-closed-loop-task-detail.png'), fullPage: true });
  834 |
  835 |     const hostDetail = await expectSuccess<HostRow>(
  836 |       await request.get(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }),
  837 |     );
  838 |     expect(hostDetail.status).toBe('assigned');
  839 |
  840 |     await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
  841 |     await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  842 |     await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  843 |     await request.delete(`${apiBaseUrl}/business/deploy/credentials/${credential.id}`, { headers }).catch(() => undefined);
  844 |   });
  845 |
  846 |   test('deploy pages stay within a phone viewport', async ({ page }, testInfo) => {
  847 |     await page.setViewportSize({ width: 390, height: 844 });
  848 |
  849 |     await page.goto('/business/deploy/package', { waitUntil: 'networkidle' });
  850 |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
  851 |     await expect
  852 |       .poll(async () =>
  853 |         page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1),
  854 |       )
  855 |       .toBe(true);
  856 |     await page.screenshot({ path: testInfo.outputPath('deploy-package-list-mobile.png'), fullPage: true });
  857 |
  858 |     await page.goto('/business/deploy/template', { waitUntil: 'networkidle' });
  859 |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
  860 |     await expect
  861 |       .poll(async () =>
  862 |         page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1),
  863 |       )
  864 |       .toBe(true);
  865 |     await page.screenshot({ path: testInfo.outputPath('deploy-template-list-mobile.png'), fullPage: true });
  866 |
  867 |     await page.goto('/business/deploy/task', { waitUntil: 'networkidle' });
  868 |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
  869 |     await expect
  870 |       .poll(async () =>
  871 |         page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1),
  872 |       )
  873 |       .toBe(true);
  874 |     await page.screenshot({ path: testInfo.outputPath('deploy-task-list-mobile.png'), fullPage: true });
  875 |   });
  876 | });
  877 |
```
