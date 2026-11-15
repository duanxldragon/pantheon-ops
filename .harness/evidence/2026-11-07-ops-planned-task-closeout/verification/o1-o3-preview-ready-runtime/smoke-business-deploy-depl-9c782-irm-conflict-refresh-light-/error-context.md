# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-selection-preview.spec.ts >> selection and preview confirm/conflict/refresh (light)
- Location: tests\smoke\business\deploy\deploy-selection-preview.spec.ts:11:2

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.deploy-target-selector').getByText('selection-preview-light-1794051049081 · 10.249.58.91', { exact: true }).first()
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('.deploy-target-selector').getByText('selection-preview-light-1794051049081 · 10.249.58.91', { exact: true }).first() with timeout 10000ms
  - waiting for locator('.deploy-target-selector').getByText('selection-preview-light-1794051049081 · 10.249.58.91', { exact: true }).first()

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
    - strong: "37"
    - text: 执行中
    - strong: "0"
    - text: 成功
    - strong: "1"
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
            - row "selection-preview-light-1794051049081 软件组件 selection-preview-light-1794051049081 2.0.0 安装 主机 selection-preview-light-1794051049081 手动执行 草稿 0 详情 编辑 启动 删除":
                - cell:
                    - checkbox
                - cell "selection-preview-light-1794051049081"
                - cell "软件组件 selection-preview-light-1794051049081 2.0.0"
                - cell "安装"
                - cell "主机"
                - cell "selection-preview-light-1794051049081"
                - cell "手动执行"
                - cell "草稿"
                - cell "0"
                - cell "详情 编辑 启动 删除":
                    - button "详情"
                    - button "编辑"
                    - button "启动"
                    - button "删除"
            - row "deploy-closed-loop-1794044952682-task 软件组件 deploy-closed-loop-1794044952682-nginx 1.30.2 安装 主机 deploy-closed-loop-1794044952682-his-dev SSH 执行 失败 1 详情 删除":
                - cell:
                    - checkbox
                - cell "deploy-closed-loop-1794044952682-task"
                - cell "软件组件 deploy-closed-loop-1794044952682-nginx 1.30.2"
                - cell "安装"
                - cell "主机"
                - cell "deploy-closed-loop-1794044952682-his-dev"
                - cell "SSH 执行"
                - cell "失败"
                - cell "1"
                - cell "详情 删除":
                    - button "详情"
                    - button "删除"
            - row "deploy-detail-edit-1794044947273-task 软件组件 deploy-detail-edit-1794044947273 1.0.0 安装 主机 deploy-detail-edit-1794044947273-scope 手动执行 草稿 0 详情 编辑 启动 删除":
                - cell:
                    - checkbox
                - cell "deploy-detail-edit-1794044947273-task"
                - cell "软件组件 deploy-detail-edit-1794044947273 1.0.0"
                - cell "安装"
                - cell "主机"
                - cell "deploy-detail-edit-1794044947273-scope"
                - cell "手动执行"
                - cell "草稿"
                - cell "0"
                - cell "详情 编辑 启动 删除":
                    - button "详情"
                    - button "编辑"
                    - button "启动"
                    - button "删除"
            - row "deploy-script-template-1794044942133-task deploy-script-template-1794044942133 v1 deploy-script-template-1794044942133 1.30.2 安装 主机 deploy-script-template-1794044942133-scope 手动执行 草稿 0 详情 编辑 启动 删除":
                - cell:
                    - checkbox
                - cell "deploy-script-template-1794044942133-task"
                - cell "deploy-script-template-1794044942133 v1 deploy-script-template-1794044942133 1.30.2"
                - cell "安装"
                - cell "主机"
                - cell "deploy-script-template-1794044942133-scope"
                - cell "手动执行"
                - cell "草稿"
                - cell "0"
                - cell "详情 编辑 启动 删除":
                    - button "详情"
                    - button "编辑"
                    - button "启动"
                    - button "删除"
            - row "deploy-mark-failed-1794044938473-task 软件组件 deploy-mark-failed-1794044938473 1.0.0 安装 主机 deploy-mark-failed-1794044938473 手动执行 失败 1 详情 删除":
                - cell:
                    - checkbox
                - cell "deploy-mark-failed-1794044938473-task"
                - cell "软件组件 deploy-mark-failed-1794044938473 1.0.0"
                - cell "安装"
                - cell "主机"
                - cell "deploy-mark-failed-1794044938473"
                - cell "手动执行"
                - cell "失败"
                - cell "1"
                - cell "详情 删除":
                    - button "详情"
                    - button "删除"
            - row "deploy-detail-1794044935197 软件组件 deploy-detail-1794044935197 1.0.0 安装 主机 deploy-detail-1794044935197 手动执行 草稿 0 详情 编辑 启动 删除":
                - cell:
                    - checkbox
                - cell "deploy-detail-1794044935197"
                - cell "软件组件 deploy-detail-1794044935197 1.0.0"
                - cell "安装"
                - cell "主机"
                - cell "deploy-detail-1794044935197"
                - cell "手动执行"
                - cell "草稿"
                - cell "0"
                - cell "详情 编辑 启动 删除":
                    - button "详情"
                    - button "编辑"
                    - button "启动"
                    - button "删除"
            - row "deploy-detail-modal-1794044924425-task 软件组件 deploy-detail-modal-1794044924425 1.30.2 安装 主机 deploy-detail-modal-1794044924425 手动执行 成功 1 详情 删除":
                - cell:
                    - checkbox
                - cell "deploy-detail-modal-1794044924425-task"
                - cell "软件组件 deploy-detail-modal-1794044924425 1.30.2"
                - cell "安装"
                - cell "主机"
                - cell "deploy-detail-modal-1794044924425"
                - cell "手动执行"
                - cell "成功"
                - cell "1"
                - cell "详情 删除":
                    - button "详情"
                    - button "删除"
            - row "deploy-closed-loop-1794044393097-task 软件组件 deploy-closed-loop-1794044393097-nginx 1.30.2 安装 主机 deploy-closed-loop-1794044393097-his-dev SSH 执行 失败 1 详情 删除":
                - cell:
                    - checkbox
                - cell "deploy-closed-loop-1794044393097-task"
                - cell "软件组件 deploy-closed-loop-1794044393097-nginx 1.30.2"
                - cell "安装"
                - cell "主机"
                - cell "deploy-closed-loop-1794044393097-his-dev"
                - cell "SSH 执行"
                - cell "失败"
                - cell "1"
                - cell "详情 删除":
                    - button "详情"
                    - button "删除"
            - row "deploy-detail-edit-1794044387311-task 软件组件 deploy-detail-edit-1794044387311 1.0.0 安装 主机 deploy-detail-edit-1794044387311-scope 手动执行 草稿 0 详情 编辑 启动 删除":
                - cell:
                    - checkbox
                - cell "deploy-detail-edit-1794044387311-task"
                - cell "软件组件 deploy-detail-edit-1794044387311 1.0.0"
                - cell "安装"
                - cell "主机"
                - cell "deploy-detail-edit-1794044387311-scope"
                - cell "手动执行"
                - cell "草稿"
                - cell "0"
                - cell "详情 编辑 启动 删除":
                    - button "详情"
                    - button "编辑"
                    - button "启动"
                    - button "删除"
            - row "deploy-script-template-1794044380878-task deploy-script-template-1794044380878 v1 deploy-script-template-1794044380878 1.30.2 安装 主机 deploy-script-template-1794044380878-scope 手动执行 草稿 0 详情 编辑 启动 删除":
                - cell:
                    - checkbox
                - cell "deploy-script-template-1794044380878-task"
                - cell "deploy-script-template-1794044380878 v1 deploy-script-template-1794044380878 1.30.2"
                - cell "安装"
                - cell "主机"
                - cell "deploy-script-template-1794044380878-scope"
                - cell "手动执行"
                - cell "草稿"
                - cell "0"
                - cell "详情 编辑 启动 删除":
                    - button "详情"
                    - button "编辑"
                    - button "启动"
                    - button "删除"
    - text: 总数
    - strong: "37"
    - list:
        - listitem "上一页":
            - button "首页" [disabled]
            - text: .
        - listitem "第 1 页": "1"
        - listitem "第 2 页": "2"
        - listitem "第 3 页": "3"
        - listitem "第 4 页": "4"
        - listitem "下一页":
            - button "末页"
            - text: .
    - combobox: 10 条/页
- text: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  1  | import { expect, test, type APIResponse } from '@playwright/test';
  2  | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
  3  | import type { DeployTargetResolution } from '../../../../src/modules/business/deploy/api';
  4  | async function success<T>(response: APIResponse): Promise<T> {
  5  |  const body = await response.json();
  6  |  expect(response.ok(), `HTTP ${response.status()} ${body.message}`).toBeTruthy();
  7  |  expect(body.code, body.message).toBe(200);
  8  |  return body.data as T;
  9  | }
  10 | for (const mode of ['light', 'dark'] as const) {
  11 |  test(`selection and preview confirm/conflict/refresh (${mode})`, async ({ page }, testInfo) => {
  12 |   const login = await loginByApi(page.request, adminCredentials);
  13 |   await installClientSession(page, login); await primeChineseLocale(page);
  14 |   await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  15 |   const headers = apiRequestHeaders(login);
  16 |   const token = `selection-preview-${mode}-${Date.now()}`;
  17 |   let scopeId = 0, hostId = 0, packageId = 0, taskId = 0;
  18 |   let failed = false;
  19 |   try {
  20 |    scopeId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers, data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
  21 |    const ip = `10.249.${Math.floor(Math.random() * 200) + 1}.91`;
  22 |    hostId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers, data: { hostname: token, ip, os: 'linux', businessScopeId: scopeId } }))).id;
  23 |    packageId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers, data: { name: token, version: '2.0.0', installCommand: 'echo local-review-fixture', status: 'enabled' } }))).id;
  24 |    const selection = { businessScopeId: scopeId, hostIds: [hostId], groupIds: [], manual: [], excludedIds: [], includeScope: false };
  25 |    const resolved = await success<DeployTargetResolution>(await page.request.post(`${apiBaseUrl}/business/deploy/targets/resolve`, { headers, data: selection }));
  26 |    taskId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers, data: { name: token, packageId, executorType: 'manual', targetType: 'host', targetSelection: selection, scopeVersion: resolved.scopeVersion } }))).id;
  27 |    // Public execution cannot bypass review.
  28 |    const missing = await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/start`, { headers, data: {} });
  29 |    expect((await missing.json()).code).not.toBe(200);
  30 |    await page.setViewportSize({ width: 1440, height: 900 });
  31 |    await page.goto(`/business/deploy/task?editId=${taskId}`);
  32 |    const selector = page.locator('.deploy-target-selector');
> 33 |    await expect(selector.getByText(`${token} · ${ip}`, { exact: true }).first()).toBeVisible();
     |                                                                                  ^ Error: expect(locator).toBeVisible() failed
  34 |    await selector.getByLabel('每行输入一个主机 IP 或 ID').fill('not-a-host');
  35 |    await selector.getByRole('button', { name: '解析并核对目标' }).click();
  36 |    await expect(selector.getByText('目标无效或状态不允许，请调整选择后重新解析')).toBeVisible();
  37 |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-desktop-invalid.png`), fullPage: true });
  38 |    await page.setViewportSize({ width: 390, height: 844 });
  39 |    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  40 |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-mobile-invalid.png`), fullPage: true });
  41 |    await selector.getByLabel('每行输入一个主机 IP 或 ID').fill(ip);
  42 |    await selector.getByRole('button', { name: '解析并核对目标' }).click();
  43 |    await expect(selector.getByRole('status')).toContainText('已解析 1 个目标');
  44 |    await expect(selector.getByRole('status')).toContainText('去重 1');
  45 |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-mobile-ready.png`), fullPage: true });
  46 |    await page.locator('.arco-modal:visible').last().getByRole('button', { name: '取消', exact: true }).click();
  47 |    await page.goto(`/business/deploy/task/${taskId}`);
  48 |    let starts = 0;
  49 |    page.on('request', (request) => { if (request.method() === 'POST' && request.url().endsWith(`/tasks/${taskId}/start`)) starts++; });
  50 |    await page.getByRole('button', { name: '启动', exact: true }).click();
  51 |    const modal = page.locator('.arco-modal:visible').last();
  52 |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  53 |    expect(starts).toBe(0);
  54 |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-mobile-ready.png`), fullPage: true });
  55 |    await page.setViewportSize({ width: 1440, height: 900 });
  56 |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-ready.png`), fullPage: true });
  57 |    // Change a persisted fact after review; the running state must remain untouched.
  58 |    await success(await page.request.put(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers, data: { remark: 'changed after review' } }));
  59 |    await modal.getByRole('button', { name: '确认并执行' }).click();
  60 |    await expect(modal.getByText('预览已过期或数据已变化，请刷新预览', { exact: true })).toBeVisible();
  61 |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeDisabled();
  62 |    const current = await success<{status: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  63 |    expect(current.status).toBe('draft');
  64 |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-conflict.png`), fullPage: true });
  65 |    await modal.getByRole('button', { name: '刷新预览' }).click();
  66 |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  67 |    await modal.getByRole('button', { name: '确认并执行' }).focus();
  68 |    await page.keyboard.press('Enter');
  69 |    await expect(modal).toBeHidden();
  70 |    expect(starts).toBe(2);
  71 |    const started = await success<{status: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  72 |    expect(started.status).toBe('running');
  73 |   } catch (error) { failed = true; throw error; }
  74 |   finally {
  75 |    const errors: string[] = [];
  76 |    try {
  77 |     const cleanupHeaders = await verifiedApiHeaders(page.request, login);
  78 |     if (taskId) {
  79 |      const response = await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers: cleanupHeaders });
  80 |      const current = await response.json();
  81 |      if (current.data?.status === 'running') await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/cancel`, { headers: cleanupHeaders }));
  82 |     }
  83 |     for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  84 |      if (!id) continue;
  85 |      try { await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders })); }
  86 |      catch (error) { errors.push(`${endpoint}/${id}: ${String(error)}`); }
  87 |     }
  88 |    } catch (error) { errors.push(String(error)); }
  89 |    await testInfo.attach('owned-fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, taskId, hostId, scopeId, packageId, errors }) });
  90 |    if (!failed) expect(errors).toEqual([]);
  91 |   }
  92 |  });
  93 | }
  94 |
```
