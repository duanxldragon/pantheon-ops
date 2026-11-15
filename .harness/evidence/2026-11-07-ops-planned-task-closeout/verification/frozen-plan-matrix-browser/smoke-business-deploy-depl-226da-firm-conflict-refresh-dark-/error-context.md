# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-selection-preview.spec.ts >> selection and preview confirm/conflict/refresh (dark)
- Location: tests\smoke\business\deploy\deploy-selection-preview.spec.ts:11:2

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.arco-modal:visible').last().getByText('无权限预览或执行此任务', { exact: true })
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('.arco-modal:visible').last().getByText('无权限预览或执行此任务', { exact: true }) with timeout 10000ms
  - waiting for locator('.arco-modal:visible').last().getByText('无权限预览或执行此任务', { exact: true })

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
    - listitem: 任务详情
- button "平台功能台": 搜索菜单、页面、功能… Ctrl K
- button "通知中心"
- button "平台偏好"
- 'button "用户: Administrator"': A Administrator admin
- tablist "已打开页面":
    - tab "工作台"
    - tab "部署任务"
    - tab "任务详情" [selected]
- main:
    - heading "selection-preview-dark-1794057754495" [level=5]
    - text: 任务详情
    - button "返回"
    - text: 运维平台 / 部署任务
    - heading "selection-preview-dark-1794057754495" [level=5]
    - text: 查看任务摘要、状态流转和每台目标主机的执行结果，便于运维排障与人工收口。 草稿 目标主机 0 成功 0 失败 0 任务状态流转
    - list "任务状态流转":
        - listitem: 草稿
        - listitem: 待执行
        - listitem: 执行中
        - listitem: 成功
        - listitem: 失败
        - listitem: 已取消
    - text: 任务状态会根据启动、取消和主机结果汇总自动推进。 任务级操作 仅允许在可操作状态执行启动或取消，任务结果会同步汇总到详情页。 任务状态会根据启动、取消和主机结果汇总自动推进。
    - button "编辑"
    - button "启动"
    - button "删除"
    - table:
        - rowgroup:
            - row "任务来源 selection-preview-dark-1794057754495-template 1.0.0 软件组件 selection-preview-dark-1794057754495 2.0.0":
                - cell "任务来源"
                - cell "selection-preview-dark-1794057754495-template 1.0.0"
                - cell "软件组件"
                - cell "selection-preview-dark-1794057754495 2.0.0"
            - row "任务动作 安装 执行模式 固定任务":
                - cell "任务动作"
                - cell "安装"
                - cell "执行模式"
                - cell "固定任务"
            - row "模板参数 已配置，详见下方参数区 状态 草稿":
                - cell "模板参数"
                - cell "已配置，详见下方参数区"
                - cell "状态"
                - cell "草稿"
            - row "目标来源 主机 业务域 selection-preview-dark-1794057754495":
                - cell "目标来源"
                - cell "主机"
                - cell "业务域"
                - cell "selection-preview-dark-1794057754495"
            - row "执行方式 手动执行 外部任务 ID -":
                - cell "执行方式"
                - cell "手动执行"
                - cell "外部任务 ID"
                - cell "-"
            - row "开始时间 - 结束时间 -":
                - cell "开始时间"
                - cell "-"
                - cell "结束时间"
                - cell "-"
            - row "耗时 - 主机数 0":
                - cell "耗时"
                - cell "-"
                - cell "主机数"
                - cell "0"
            - row "成功数 0 失败数 0":
                - cell "成功数"
                - cell "0"
                - cell "失败数"
                - cell "0"
            - row "执行中 0 跳过数 0":
                - cell "执行中"
                - cell "0"
                - cell "跳过数"
                - cell "0"
            - row "备注 -":
                - cell "备注"
                - cell "-"
    - text: 模板参数
    - table:
        - rowgroup:
            - row "custom unrestricted-config-fixture port 8080":
                - cell "custom"
                - cell "unrestricted-config-fixture"
                - cell "port"
                - cell "8080"
            - 'row "rootPassword {{secret.password}} serviceName reviewed-service"':
                - cell "rootPassword"
                - 'cell "{{secret.password}}"'
                - cell "serviceName"
                - cell "reviewed-service"
    - text: 主机执行明细 先查看各状态分布，再按主机展开过程日志和标准输出。 待执行 0 执行中 0 成功 0 失败 0 已跳过 0 任务尚未生成主机执行明细
- text: Pantheon Platform · Enterprise Admin Foundation
- dialog "执行前预览":
    - text: 执行前预览
    - region "目标版本变化":
        - text: 目标版本变化
        - button "显示未改变项"
        - text: 无权查看或执行此任务
    - button "取消"
    - button "刷新预览"
    - button "确认并执行" [disabled]
    - button "Close"
```

# Test source

```ts
  1   | import { expect, test, type APIResponse } from '@playwright/test';
  2   | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
  3   | import type { DeployTargetResolution } from '../../../../src/modules/business/deploy/api';
  4   | async function success<T>(response: APIResponse): Promise<T> {
  5   |  const body = await response.json();
  6   |  expect(response.ok(), `HTTP ${response.status()} ${body.message}`).toBeTruthy();
  7   |  expect(body.code, body.message).toBe(200);
  8   |  return body.data as T;
  9   | }
  10  | for (const mode of ['light', 'dark'] as const) {
  11  |  test(`selection and preview confirm/conflict/refresh (${mode})`, async ({ page }, testInfo) => {
  12  |   const login = await loginByApi(page.request, adminCredentials);
  13  |   await installClientSession(page, login); await primeChineseLocale(page);
  14  |   await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  15  |   const headers = apiRequestHeaders(login);
  16  |   const token = `selection-preview-${mode}-${Date.now()}`;
  17  |   let scopeId = 0, hostId = 0, packageId = 0, templateId = 0, taskId = 0;
  18  |   let failed = false;
  19  |   try {
  20  |    scopeId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers, data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
  21  |    const ip = `10.249.${Math.floor(Math.random() * 200) + 1}.91`;
  22  |    hostId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers, data: { hostname: token, ip, os: 'linux', businessScopeId: scopeId } }))).id;
  23  |    packageId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers, data: { name: token, version: '2.0.0', installCommand: 'echo local-review-fixture', status: 'enabled' } }))).id;
  24  |    templateId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/templates`, { headers, data: { name: `${token}-template`, version: '1.0.0', packageId, status: 'enabled' } }))).id;
  25  |    const selection = { businessScopeId: scopeId, hostIds: [hostId], groupIds: [], manual: [], excludedIds: [], includeScope: false };
  26  |    const resolved = await success<DeployTargetResolution>(await page.request.post(`${apiBaseUrl}/business/deploy/targets/resolve`, { headers, data: selection }));
  27  |    taskId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers, data: { name: token, packageId, templateId, templateParams: { serviceName: 'reviewed-service', port: '8080', rootPassword: '{{secret.password}}', custom: 'unrestricted-config-fixture' }, secretParams: { password: 'local-template-secret-fixture' }, executorType: 'manual', targetType: 'host', targetSelection: selection, scopeVersion: resolved.scopeVersion } }))).id;
  28  |    // Public execution cannot bypass review.
  29  |    const missing = await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/start`, { headers, data: {} });
  30  |    expect((await missing.json()).code).not.toBe(200);
  31  |    await page.setViewportSize({ width: 1440, height: 900 });
  32  |    await page.goto(`/business/deploy/task?editId=${taskId}`);
  33  |    const selector = page.locator('.deploy-target-selector');
  34  |    await expect(selector.getByText(`${token} · ${ip}`, { exact: true }).first()).toBeVisible();
  35  |    await selector.getByLabel('每行输入一个主机 IP 或 ID').fill('not-a-host');
  36  |    await selector.getByRole('button', { name: '解析并核对目标' }).click();
  37  |    await expect(selector.getByText('目标无效或状态不允许，请调整选择后重新解析')).toBeVisible();
  38  |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-desktop-invalid.png`), fullPage: true });
  39  |    await page.setViewportSize({ width: 390, height: 844 });
  40  |    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  41  |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-mobile-invalid.png`), fullPage: true });
  42  |    await selector.getByLabel('每行输入一个主机 IP 或 ID').fill(ip);
  43  |    await selector.getByRole('button', { name: '解析并核对目标' }).click();
  44  |    await expect(selector.getByRole('status')).toContainText('已解析 1 个目标');
  45  |    await expect(selector.getByRole('status')).toContainText('去重 1');
  46  |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-mobile-ready.png`), fullPage: true });
  47  |    await page.locator('.arco-modal:visible').last().getByRole('button', { name: '取消', exact: true }).click();
  48  |    await page.goto(`/business/deploy/task/${taskId}`);
  49  |    let starts = 0;
  50  |    page.on('request', (request) => { if (request.method() === 'POST' && request.url().endsWith(`/tasks/${taskId}/start`)) starts++; });
  51  |    await page.getByRole('button', { name: '启动', exact: true }).click();
  52  |    const modal = page.locator('.arco-modal:visible').last();
  53  |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  54  |    expect(starts).toBe(0);
  55  |    const plan = modal.getByRole('region', { name: '已冻结的执行配置' });
  56  |    await expect(plan).toContainText(`${token}-template`);
  57  |    await expect(plan).toContainText('reviewed-service');
  58  |    await expect(plan).toContainText('8080');
  59  |    await expect(plan).toContainText('当前参数值尚未测量');
  60  |    await expect(plan).toContainText('失败处理：停止执行');
  61  |    const stepToggle = plan.locator('.arco-collapse-item-header').first();
  62  |    await stepToggle.focus();
  63  |    await page.keyboard.press('Enter');
  64  |    await expect(plan.getByText('reviewed-service', { exact: true })).toBeHidden();
  65  |    await page.keyboard.press('Enter');
  66  |    await expect(plan.getByText('reviewed-service', { exact: true })).toBeVisible();
  67  |    await expect(plan).not.toContainText('local-template-secret-fixture');
  68  |    await expect(plan).not.toContainText('unrestricted-config-fixture');
  69  |    await plan.scrollIntoViewIfNeeded();
  70  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-mobile.png`) });
  71  |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-mobile-ready.png`), fullPage: true });
  72  |    await page.setViewportSize({ width: 1440, height: 900 });
  73  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop.png`) });
  74  |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-ready.png`), fullPage: true });
  75  |    await page.emulateMedia({ reducedMotion: 'reduce' });
  76  |    await expect.poll(() => page.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches)).toBe(true);
  77  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop-reduced-motion.png`) });
  78  |    await page.evaluate(() => { document.documentElement.style.zoom = '2'; });
  79  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop-css-zoom200.png`) });
  80  |    await page.evaluate(() => { document.documentElement.style.zoom = ''; });
  81  |    // These interceptions prove UI recovery states, not a real role revocation.
  82  |    const previewUrl = `**/business/deploy/tasks/${taskId}/preview`;
  83  |    for (const scenario of [{ status: 403, message: 'permission.denied', text: '无权查看或执行此任务', label: 'forbidden' }, { status: 503, message: 'unavailable', text: '预览加载失败，请重试', label: 'error' }]) {
  84  |     await modal.getByRole('button', { name: '取消', exact: true }).click();
  85  |     await page.route(previewUrl, (route) => route.fulfill({ status: scenario.status, contentType: 'application/json', body: JSON.stringify({ code: scenario.status, message: scenario.message, data: null }) }));
  86  |     await page.getByRole('button', { name: '启动', exact: true }).click();
> 87  |     await expect(modal.getByText(scenario.text, { exact: true })).toBeVisible();
      |                                                                   ^ Error: expect(locator).toBeVisible() failed
  88  |     await expect(modal.getByRole('button', { name: '确认并执行' })).toBeDisabled();
  89  |     await expect(modal.getByRole('region', { name: '已冻结的执行配置' })).toHaveCount(0);
  90  |     await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-${scenario.label}.png`), fullPage: true });
  91  |     await page.unroute(previewUrl);
  92  |    }
  93  |    await modal.getByRole('button', { name: '刷新预览' }).click();
  94  |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  95  |    // Change a persisted fact after review; the running state must remain untouched.
  96  |    await success(await page.request.put(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers, data: { remark: 'changed after review' } }));
  97  |    await modal.getByRole('button', { name: '确认并执行' }).click();
  98  |    await expect(modal.getByText('预览已过期或数据已变化，请刷新预览', { exact: true })).toBeVisible();
  99  |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeDisabled();
  100 |    const current = await success<{status: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  101 |    expect(current.status).toBe('draft');
  102 |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-conflict.png`), fullPage: true });
  103 |    await modal.getByRole('button', { name: '刷新预览' }).click();
  104 |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  105 |    await modal.getByRole('button', { name: '确认并执行' }).focus();
  106 |    await page.keyboard.press('Enter');
  107 |    await expect(modal).toBeHidden();
  108 |    expect(starts).toBe(2);
  109 |    const started = await success<{status: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  110 |    expect(started.status).toBe('running');
  111 |   } catch (error) { failed = true; throw error; }
  112 |   finally {
  113 |    const errors: string[] = [];
  114 |    try {
  115 |     const cleanupHeaders = await verifiedApiHeaders(page.request, login);
  116 |     if (taskId) {
  117 |      const response = await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers: cleanupHeaders });
  118 |      const current = await response.json();
  119 |      if (current.data?.status === 'running') await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/cancel`, { headers: cleanupHeaders }));
  120 |     }
  121 |     for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [templateId, 'deploy/templates'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  122 |      if (!id) continue;
  123 |      try { await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders })); }
  124 |      catch (error) { errors.push(`${endpoint}/${id}: ${String(error)}`); }
  125 |     }
  126 |    } catch (error) { errors.push(String(error)); }
  127 |    await testInfo.attach('owned-fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, taskId, hostId, scopeId, packageId, templateId, errors }) });
  128 |    if (!failed) expect(errors).toEqual([]);
  129 |   }
  130 |  });
  131 | }
  132 |
```
