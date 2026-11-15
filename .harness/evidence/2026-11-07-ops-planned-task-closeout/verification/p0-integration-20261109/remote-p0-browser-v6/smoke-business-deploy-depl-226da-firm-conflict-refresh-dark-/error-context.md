# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-selection-preview.spec.ts >> selection and preview confirm/conflict/refresh (dark)
- Location: tests\smoke\business\deploy\deploy-selection-preview.spec.ts:12:2

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.deploy-target-selector').getByText('selection-preview-dark-1794194219211 · 10.249.39.91', { exact: true }).first()
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('.deploy-target-selector').getByText('selection-preview-dark-1794194219211 · 10.249.39.91', { exact: true }).first() with timeout 10000ms
  - waiting for locator('.deploy-target-selector').getByText('selection-preview-dark-1794194219211 · 10.249.39.91', { exact: true }).first()

```

```yaml
- complementary:
    - text: P Pantheon Base
    - menu:
        - menuitem "工作台":
            - button "工作台"
        - button "CMDB"
        - menuitem "业务域":
            - button "业务域"
        - menuitem "服务目录":
            - button "服务目录"
        - button "安装部署"
        - menuitem "软件组件":
            - button "软件组件"
        - menuitem "任务模板":
            - button "任务模板"
        - menuitem "部署任务":
            - button "部署任务"
        - button "Kubernetes"
        - button "可观测中心"
        - button "访问控制"
        - button "组织架构"
        - button "安全审计"
        - button "低代码平台"
        - button "平台配置"
        - menuitem "文件分发":
            - button "文件分发"
        - menuitem "ArgoCD":
            - button "ArgoCD"
- button "切换导航"
- list:
    - listitem: 首页
    - listitem: 安装部署
    - listitem: 部署任务
- button "平台功能台": 搜索菜单、页面、功能… Ctrl K
- button "通知中心"
- button "平台偏好"
- 'button "用户: Administrator"': A Administrator admin
- tablist "已打开页面":
    - tab "工作台"
    - tab "部署任务" [selected]
- main:
    - text: 运维平台 / 部署任务 部署任务 任务总数
    - strong: "4"
    - text: 执行中
    - strong: "1"
    - text: 成功
    - strong: "0"
    - button "任务治理视角"
    - search "筛选":
        - textbox "关键字"
        - combobox:
            - textbox "状态"
            - text: 状态
    - button "导出"
    - button "新增"
    - text: 已选 0 条
    - button "清空选择" [disabled]
    - button "取消" [disabled]
    - table:
        - rowgroup:
            - row "任务名称 任务来源 任务动作 目标来源 业务域 执行方式 状态 主机数 操作":
                - columnheader:
                    - checkbox
                - columnheader "任务名称"
                - columnheader "任务来源"
                - columnheader "任务动作"
                - columnheader "目标来源"
                - columnheader "业务域"
                - columnheader "执行方式"
                - columnheader "状态"
                - columnheader "主机数"
                - columnheader "操作"
        - rowgroup:
            - row "selection-preview-dark-1794194219211 selection-preview-dark-1794194219211-template 1.0.0 selection-preview-dark-1794194219211 2.0.0 安装 主机 selection-preview-dark-1794194219211 手动执行 草稿 0 详情 编辑 启动 删除":
                - cell:
                    - checkbox
                - cell "selection-preview-dark-1794194219211"
                - cell "selection-preview-dark-1794194219211-template 1.0.0 selection-preview-dark-1794194219211 2.0.0"
                - cell "安装"
                - cell "主机"
                - cell "selection-preview-dark-1794194219211"
                - cell "手动执行"
                - cell "草稿"
                - cell "0"
                - cell "详情 编辑 启动 删除":
                    - button "详情"
                    - button "编辑"
                    - button "启动"
                    - button "删除"
            - row "selection-preview-light-1794194187090 selection-preview-light-1794194187090-template 1.0.0 selection-preview-light-1794194187090 2.0.0 安装 主机 selection-preview-light-1794194187090 手动执行 草稿 0 详情 编辑 启动 删除":
                - cell:
                    - checkbox
                - cell "selection-preview-light-1794194187090"
                - cell "selection-preview-light-1794194187090-template 1.0.0 selection-preview-light-1794194187090 2.0.0"
                - cell "安装"
                - cell "主机"
                - cell "selection-preview-light-1794194187090"
                - cell "手动执行"
                - cell "草稿"
                - cell "0"
                - cell "详情 编辑 启动 删除":
                    - button "详情"
                    - button "编辑"
                    - button "启动"
                    - button "删除"
            - row "command-batch-1794194154196-b 软件组件 command-batch-1794194154196 1.0.0 安装 主机 command-batch-1794194154196 手动执行 执行中 1 详情 取消":
                - cell:
                    - checkbox
                - cell "command-batch-1794194154196-b"
                - cell "软件组件 command-batch-1794194154196 1.0.0"
                - cell "安装"
                - cell "主机"
                - cell "command-batch-1794194154196"
                - cell "手动执行"
                - cell "执行中"
                - cell "1"
                - cell "详情 取消":
                    - button "详情"
                    - button "取消"
            - row "command-batch-1794194154196-a 软件组件 command-batch-1794194154196 1.0.0 安装 主机 command-batch-1794194154196 手动执行 已取消 1 详情 删除":
                - cell:
                    - checkbox
                - cell "command-batch-1794194154196-a"
                - cell "软件组件 command-batch-1794194154196 1.0.0"
                - cell "安装"
                - cell "主机"
                - cell "command-batch-1794194154196"
                - cell "手动执行"
                - cell "已取消"
                - cell "1"
                - cell "详情 删除":
                    - button "详情"
                    - button "删除"
    - text: 总数
    - strong: "4"
    - list:
        - listitem "上一页": .
        - listitem "第 1 页": "1"
        - listitem "下一页": .
    - combobox: 10 条/页
- text: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  1   | import { expect, test, type APIResponse } from '@playwright/test';
  2   | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
  3   | import { startReviewedTask } from './reviewed-start';
  4   | import type { DeployTargetResolution } from '../../../../src/modules/business/deploy/api';
  5   | async function success<T>(response: APIResponse): Promise<T> {
  6   |  const body = await response.json();
  7   |  expect(response.ok(), `HTTP ${response.status()} ${body.message}`).toBeTruthy();
  8   |  expect(body.code, body.message).toBe(200);
  9   |  return body.data as T;
  10  | }
  11  | for (const mode of ['light', 'dark'] as const) {
  12  |  test(`selection and preview confirm/conflict/refresh (${mode})`, async ({ page }, testInfo) => {
  13  |   const login = await loginByApi(page.request, adminCredentials);
  14  |   await installClientSession(page, login); await primeChineseLocale(page);
  15  |   await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  16  |   const headers = apiRequestHeaders(login);
  17  |   const token = `selection-preview-${mode}-${Date.now()}`;
  18  |   let scopeId = 0, hostId = 0, packageId = 0, templateId = 0, taskId = 0;
  19  |   let failed = false;
  20  |   try {
  21  |    scopeId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers, data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
  22  |    const ip = `10.249.${Math.floor(Math.random() * 200) + 1}.91`;
  23  |    hostId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers, data: { hostname: token, ip, os: 'linux', businessScopeId: scopeId } }))).id;
  24  |    packageId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers, data: { name: token, version: '2.0.0', installCommand: 'echo local-review-fixture', status: 'enabled' } }))).id;
  25  |    templateId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/templates`, { headers, data: { name: `${token}-template`, version: '1.0.0', packageId, status: 'enabled' } }))).id;
  26  |    const selection = { businessScopeId: scopeId, hostIds: [hostId], groupIds: [], manual: [], excludedIds: [], includeScope: false };
  27  |    const resolved = await success<DeployTargetResolution>(await page.request.post(`${apiBaseUrl}/business/deploy/targets/resolve`, { headers, data: selection }));
  28  |    taskId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers, data: { name: token, packageId, templateId, templateParams: { serviceName: 'reviewed-service', port: '8080', rootPassword: '{{secret.password}}', custom: 'unrestricted-config-fixture' }, secretParams: { password: 'local-template-secret-fixture' }, executorType: 'manual', targetType: 'host', targetSelection: selection, scopeVersion: resolved.scopeVersion } }))).id;
  29  |    // Public execution cannot bypass review.
  30  |    const missing = await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/start`, { headers, data: {} });
  31  |    expect((await missing.json()).code).not.toBe(200);
  32  |    await page.setViewportSize({ width: 1440, height: 900 });
  33  |    await page.goto(`/business/deploy/task?editId=${taskId}`);
  34  |    const selector = page.locator('.deploy-target-selector');
> 35  |    await expect(selector.getByText(`${token} · ${ip}`, { exact: true }).first()).toBeVisible();
      |                                                                                  ^ Error: expect(locator).toBeVisible() failed
  36  |    await selector.getByLabel('每行输入一个主机 IP 或 ID').fill('not-a-host');
  37  |    await selector.getByRole('button', { name: '解析并核对目标' }).click();
  38  |    await expect(selector.getByText('目标无效或状态不允许，请调整选择后重新解析')).toBeVisible();
  39  |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-desktop-invalid.png`), fullPage: true });
  40  |    await page.setViewportSize({ width: 390, height: 844 });
  41  |    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  42  |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-mobile-invalid.png`), fullPage: true });
  43  |    await selector.getByLabel('每行输入一个主机 IP 或 ID').fill(ip);
  44  |    await selector.getByRole('button', { name: '解析并核对目标' }).click();
  45  |    await expect(selector.getByRole('status')).toContainText('已解析 1 个目标');
  46  |    await expect(selector.getByRole('status')).toContainText('去重 1');
  47  |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-mobile-ready.png`), fullPage: true });
  48  |    await page.locator('.arco-modal:visible').last().getByRole('button', { name: '取消', exact: true }).click();
  49  |    await page.goto(`/business/deploy/task/${taskId}`);
  50  |    let starts = 0;
  51  |    page.on('request', (request) => { if (request.method() === 'POST' && request.url().endsWith(`/tasks/${taskId}/start`)) starts++; });
  52  |    await page.getByRole('button', { name: '启动', exact: true }).click();
  53  |    const modal = page.locator('.arco-modal:visible').last();
  54  |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  55  |    expect(starts).toBe(0);
  56  |    const plan = modal.getByRole('region', { name: '已冻结的执行配置' });
  57  |    await expect(plan).toContainText(`${token}-template`);
  58  |    await expect(plan).toContainText('reviewed-service');
  59  |    await expect(plan).toContainText('8080');
  60  |    await expect(plan).toContainText('当前参数值尚未测量');
  61  |    await expect(plan).toContainText('失败处理：停止执行');
  62  |    const stepToggle = plan.locator('.arco-collapse-item-header').first();
  63  |    await stepToggle.focus();
  64  |    await page.keyboard.press('Enter');
  65  |    await expect(plan.getByText('reviewed-service', { exact: true })).toBeHidden();
  66  |    await page.keyboard.press('Enter');
  67  |    await expect(plan.getByText('reviewed-service', { exact: true })).toBeVisible();
  68  |    await expect(plan).not.toContainText('local-template-secret-fixture');
  69  |    await expect(plan).not.toContainText('unrestricted-config-fixture');
  70  |    await plan.scrollIntoViewIfNeeded();
  71  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-mobile.png`) });
  72  |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-mobile-ready.png`), fullPage: true });
  73  |    await page.setViewportSize({ width: 1440, height: 900 });
  74  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop.png`) });
  75  |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-ready.png`), fullPage: true });
  76  |    await page.emulateMedia({ reducedMotion: 'reduce' });
  77  |    await expect.poll(() => page.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches)).toBe(true);
  78  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop-reduced-motion.png`) });
  79  |    await page.evaluate(() => { document.documentElement.style.zoom = '2'; });
  80  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop-css-zoom200.png`) });
  81  |    await page.evaluate(() => { document.documentElement.style.zoom = ''; });
  82  |    // These interceptions prove UI recovery states, not a real role revocation.
  83  |    const previewUrl = `**/business/deploy/tasks/${taskId}/preview`;
  84  |    for (const scenario of [{ status: 403, message: 'permission.denied', text: '无权查看或执行此任务', label: 'forbidden' }, { status: 503, message: 'unavailable', text: '预览加载失败，请重试', label: 'error' }]) {
  85  |     await modal.getByRole('button', { name: '取消', exact: true }).click();
  86  |     await page.route(previewUrl, (route) => route.fulfill({ status: scenario.status, contentType: 'application/json', body: JSON.stringify({ code: scenario.status, message: scenario.message, data: null }) }));
  87  |     await page.getByRole('button', { name: '启动', exact: true }).click();
  88  |     await expect(modal.getByText(scenario.text, { exact: true })).toBeVisible();
  89  |     await expect(modal.getByRole('button', { name: '确认并执行' })).toBeDisabled();
  90  |     await expect(modal.getByRole('region', { name: '已冻结的执行配置' })).toHaveCount(0);
  91  |     await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-${scenario.label}.png`), fullPage: true });
  92  |     await page.unroute(previewUrl);
  93  |    }
  94  |    await modal.getByRole('button', { name: '刷新预览' }).click();
  95  |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  96  |    // Change a persisted fact after review; the running state must remain untouched.
  97  |    await success(await page.request.put(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers, data: { remark: 'changed after review' } }));
  98  |    await modal.getByRole('button', { name: '确认并执行' }).click();
  99  |    await expect(modal.getByText('预览已过期或数据已变化，请刷新预览', { exact: true })).toBeVisible();
  100 |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeDisabled();
  101 |    const current = await success<{status: string; stateVersion: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  102 |    expect(current.status).toBe('draft');
  103 |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-conflict.png`), fullPage: true });
  104 |    await modal.getByRole('button', { name: '刷新预览' }).click();
  105 |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  106 |    await modal.getByRole('button', { name: '确认并执行' }).focus();
  107 |    await page.keyboard.press('Enter');
  108 |    await expect(modal).toBeHidden();
  109 |    expect(starts).toBe(2);
  110 |    const started = await success<{status: string; stateVersion: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  111 |    expect(started.status).toBe('running');
  112 |   } catch (error) { failed = true; throw error; }
  113 |   finally {
  114 |    const errors: string[] = [];
  115 |    try {
  116 |     const cleanupHeaders = await verifiedApiHeaders(page.request, login);
  117 |     if (taskId) {
  118 |      const response = await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers: cleanupHeaders });
  119 |      const current = await response.json();
  120 |      if (current.data?.status === 'running') await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/cancel`, { headers: cleanupHeaders, data: { stateVersion: current.data?.stateVersion } }));
  121 |     }
  122 |     for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [templateId, 'deploy/templates'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  123 |      if (!id) continue;
  124 |      try { await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders })); }
  125 |      catch (error) { errors.push(`${endpoint}/${id}: ${String(error)}`); }
  126 |     }
  127 |    } catch (error) { errors.push(String(error)); }
  128 |    await testInfo.attach('owned-fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, taskId, hostId, scopeId, packageId, templateId, errors }) });
  129 |    if (!failed) expect(errors).toEqual([]);
  130 |   }
  131 |  });
  132 | }
  133 |
  134 | test('rollback draft uses frozen definition and its own reviewed start', async ({ page }, testInfo) => {
  135 |  const login = await loginByApi(page.request, adminCredentials);
```
