# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy detail edit action routes back to list and opens edit modal
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:609:3

# Error details

```
Error: expect(locator).toBeHidden() failed

Locator:  locator('.arco-modal').last()
Expected: hidden
Received: visible
Timeout:  10000ms

Call log:
  - Expect "toBeHidden" locator('.arco-modal').last() with timeout 10000ms
  - waiting for locator('.arco-modal').last()
    - locator resolved to <div role="dialog" aria-modal="true" aria-labelledby="arco-dialog-11" class="arco-modal app-dialog zoomModal-appear-done zoomModal-enter-done">…</div>
    3 × unexpected value "visible"
      - locator resolved to <div role="dialog" aria-modal="true" aria-labelledby="arco-dialog-11" class="arco-modal app-dialog zoomModal-exit zoomModal-exit-active">…</div>
    - unexpected value "visible"
    - locator resolved to <div role="dialog" aria-modal="true" aria-labelledby="arco-dialog-11" class="arco-modal app-dialog zoomModal-enter zoomModal-enter-active">…</div>
    18 × unexpected value "visible"
       - locator resolved to <div role="dialog" aria-modal="true" aria-labelledby="arco-dialog-11" class="arco-modal app-dialog zoomModal-enter-done">…</div>
    - unexpected value "visible"

```

```yaml
- dialog "编辑任务":
    - text: 编辑任务
    - strong
    - text: 任务名称
    - textbox "任务名称": deploy-detail-edit-1794053355126-task
    - text: 任务来源
    - combobox: 软件组件
    - strong
    - text: 软件组件
    - combobox: deploy-detail-edit-1794053355126 1.0.0
    - text: deploy-detail-edit-1794053355126 1.0.0 直接按组件配置执行，适合单步安装或补充性操作。 任务动作
    - combobox: 安装
    - text: 目标来源
    - combobox: 主机
    - strong
    - text: 业务域
    - combobox: deploy-detail-edit-1794053355126-scope
    - textbox "搜索当前业务范围内的主机"
    - region "选择主机目标"
    - list
    - checkbox "包含当前业务范围的全部主机"
    - text: 包含当前业务范围的全部主机
    - combobox "按条件分组添加目标"
    - textbox "每行输入一个主机 IP 或 ID"
    - combobox "从候选主机中排除"
    - button "解析并核对目标"
    - text: 执行方式
    - combobox: 手动执行
    - text: 备注
    - textbox "备注": detail edit smoke updated
    - button "取消"
    - button "保存"
    - button "Close"
```

# Test source

```ts
  582 |             installRoot: '/data/nginx-custom',
  583 |             serviceName: 'nginx-custom',
  584 |             configRoot: '/data/nginx-custom/conf',
  585 |           },
  586 |         },
  587 |       }),
  588 |     );
  589 |
  590 |     try {
  591 |       await page.goto('/business/deploy/template', { waitUntil: 'networkidle' });
  592 |       await page.locator('.search-toolbar input').first().fill(token);
  593 |       await expect(page.getByText(token, { exact: true }).first()).toBeVisible();
  594 |       await page.screenshot({ path: testInfo.outputPath('deploy-script-template-list.png'), fullPage: true });
  595 |
  596 |       await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'networkidle' });
  597 |       await expect(page.getByText('configRoot', { exact: true })).toBeVisible();
  598 |       await expect(page.getByText('/data/nginx-custom/conf', { exact: true })).toBeVisible();
  599 |       await expect(page.getByText('/data/nginx-custom', { exact: true })).toBeVisible();
  600 |       await page.screenshot({ path: testInfo.outputPath('deploy-script-template-task-detail.png'), fullPage: true });
  601 |     } finally {
  602 |       await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
  603 |       await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  604 |       await request.delete(`${apiBaseUrl}/business/deploy/templates/${template.id}`, { headers }).catch(() => undefined);
  605 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  606 |     }
  607 |   });
  608 |
  609 |   test('deploy detail edit action routes back to list and opens edit modal', async ({ page, request }, testInfo) => {
  610 |     const login = await loginByApi(request, adminCredentials);
  611 |     await installClientSession(page, login);
  612 |     const headers = apiRequestHeaders(login);
  613 |     const token = `deploy-detail-edit-${Date.now()}`;
  614 |
  615 |     const scope = await expectSuccess<BizScopeRow>(
  616 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  617 |         headers,
  618 |         data: {
  619 |           code: token,
  620 |           name: `${token}-scope`,
  621 |           environment: 'dev',
  622 |           status: 'active',
  623 |         },
  624 |       }),
  625 |     );
  626 |
  627 |     const host = await expectSuccess<HostRow>(
  628 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  629 |         headers,
  630 |         data: {
  631 |           hostname: `${token}-host`,
  632 |           ip: `10.251.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
  633 |           os: 'linux',
  634 |           businessScopeId: scope.id,
  635 |           labels: [{ key: 'biz', val: token }],
  636 |         },
  637 |       }),
  638 |     );
  639 |
  640 |     const pkg = await expectSuccess<PackageRow>(
  641 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  642 |         headers,
  643 |         data: {
  644 |           name: token,
  645 |           version: '1.0.0',
  646 |           installCommand: 'echo install',
  647 |           status: 'enabled',
  648 |         },
  649 |       }),
  650 |     );
  651 |
  652 |     const task = await expectSuccess<TaskRow>(
  653 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  654 |         headers,
  655 |         data: {
  656 |           name: `${token}-task`,
  657 |           packageId: pkg.id,
  658 |           businessScopeId: scope.id,
  659 |           targetType: 'host',
  660 |           targetIds: [host.id],
  661 |           executorType: 'manual',
  662 |           remark: 'detail edit smoke',
  663 |         },
  664 |       }),
  665 |     );
  666 |     expect(task.status).toBe('draft');
  667 |
  668 |     try {
  669 |       await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'networkidle' });
  670 |       await expect(page.getByText('任务级操作', { exact: true })).toBeVisible();
  671 |       await page.getByRole('button', { name: '编辑' }).click();
  672 |
  673 |       await expect(page).toHaveURL(new RegExp(`/business/deploy/task\\?editId=${task.id}$`));
  674 |       const modal = page.locator('.arco-modal').last();
  675 |       await expect(modal).toBeVisible();
  676 |       await expect(modal).toContainText('编辑任务');
  677 |       await expect(modal.locator('input').first()).toHaveValue(`${token}-task`);
  678 |       await modal.locator('.arco-form-item').filter({ hasText: '备注' }).locator('textarea').fill('detail edit smoke updated');
  679 |       await modal.getByRole('button', { name: '解析并核对目标' }).click();
  680 |       await expect(modal.locator('.deploy-target-selector').getByRole('status')).toContainText('已解析 1 个目标');
  681 |       await modal.getByRole('button', { name: '保存' }).click();
> 682 |       await expect(modal).toBeHidden();
      |                           ^ Error: expect(locator).toBeHidden() failed
  683 |       await expect(page).toHaveURL(/\/business\/deploy\/task$/);
  684 |
  685 |       const detail = await expectSuccess<TaskRow>(
  686 |         await request.get(`${apiBaseUrl}/business/deploy/tasks/${task.id}`, { headers }),
  687 |       );
  688 |       expect(detail.remark).toBe('detail edit smoke updated');
  689 |       await page.screenshot({ path: testInfo.outputPath('deploy-task-detail-edit-modal.png'), fullPage: true });
  690 |     } finally {
  691 |       await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
  692 |       await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  693 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  694 |     }
  695 |   });
  696 |
  697 |   test('targeted smoke closes cmdb, bizscope, deploy with uploaded source package and visible task trace', async ({ page, request }, testInfo) => {
  698 |     const login = await loginByApi(request, adminCredentials);
  699 |     await installClientSession(page, login);
  700 |     const headers = apiRequestHeaders(login);
  701 |     const token = `deploy-closed-loop-${Date.now()}`;
  702 |     // Keep the synthetic SSH target unique across retries; CMDB enforces IP uniqueness.
  703 |     const smokeIp = `127.0.0.${Math.floor(Math.random() * 200) + 20}`;
  704 |     const sourceBuffer = Buffer.from('fake nginx source archive for smoke', 'utf8');
  705 |
  706 |     const scope = await expectSuccess<BizScopeRow>(
  707 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  708 |         headers,
  709 |         data: {
  710 |           code: token,
  711 |           name: `${token}-his-dev`,
  712 |           environment: 'dev',
  713 |           status: 'active',
  714 |         },
  715 |       }),
  716 |     );
  717 |
  718 |     const host = await expectSuccess<HostRow>(
  719 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  720 |         headers,
  721 |         data: {
  722 |           hostname: `${token}-host`,
  723 |           ip: smokeIp,
  724 |           sshPort: 1,
  725 |           os: 'linux',
  726 |           businessScopeId: scope.id,
  727 |           labels: [
  728 |             { key: 'biz', val: token },
  729 |             { key: 'env', val: 'dev' },
  730 |           ],
  731 |           remark: token,
  732 |         },
  733 |       }),
  734 |     );
  735 |     expect(host.status).toBe('assigned');
  736 |
  737 |     const uploadResponse = await request.post(`${apiBaseUrl}/system/upload?scope=deploy/package`, {
  738 |       headers,
  739 |       multipart: {
  740 |         file: {
  741 |           name: 'nginx-1.30.2.tar.gz',
  742 |           mimeType: 'application/gzip',
  743 |           buffer: sourceBuffer,
  744 |         },
  745 |       },
  746 |     });
  747 |     const uploadPayload = await expectSuccess<{ objectKey: string; originalName: string; url: string }>(uploadResponse);
  748 |     expect(uploadPayload.originalName).toBe('nginx-1.30.2.tar.gz');
  749 |
  750 |     const pkg = await expectSuccess<PackageRow>(
  751 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  752 |         headers,
  753 |         data: {
  754 |           name: `${token}-nginx`,
  755 |           version: '1.30.2',
  756 |           executionMode: 'fixed',
  757 |           templateCode: 'nginx_systemd',
  758 |           templateConfig: { scenario: 'systemd' },
  759 |           sourceObjectKey: uploadPayload.objectKey,
  760 |           sourceFileName: uploadPayload.originalName,
  761 |           sourceUrl: uploadPayload.url,
  762 |           uninstallCommand: 'systemctl stop nginx || true',
  763 |           status: 'enabled',
  764 |         },
  765 |       }),
  766 |     );
  767 |     expect(pkg.sourceFileName).toBe('nginx-1.30.2.tar.gz');
  768 |
  769 |     const task = await expectSuccess<TaskRow>(
  770 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  771 |         headers,
  772 |         data: {
  773 |           name: `${token}-task`,
  774 |           packageId: pkg.id,
  775 |           businessScopeId: scope.id,
  776 |           targetType: 'host',
  777 |           targetIds: [host.id],
  778 |           executorType: 'ssh',
  779 |           templateParams: {
  780 |             installRoot: '/data/nginx',
  781 |             serviceName: 'nginx',
  782 |           },
```
