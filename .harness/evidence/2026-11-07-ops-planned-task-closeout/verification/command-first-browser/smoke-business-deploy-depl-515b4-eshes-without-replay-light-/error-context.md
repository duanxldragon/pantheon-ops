# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-command.spec.ts >> viewed task command conflict refreshes without replay (light)
- Location: tests\smoke\business\deploy\deploy-command.spec.ts:10:2

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByRole('button', { name: '重试', exact: true })
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" getByRole('button', { name: '重试', exact: true }) with timeout 10000ms
  - waiting for getByRole('button', { name: '重试', exact: true })

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
    - tab "任务详情" [selected]
- main:
    - heading "command-light-1794062871424" [level=5]
    - text: 任务详情
    - button "返回"
    - text: 运维平台 / 部署任务
    - heading "command-light-1794062871424" [level=5]
    - text: 查看任务摘要、状态流转和每台目标主机的执行结果，便于运维排障与人工收口。 失败 目标主机 1 成功 0 失败 1 任务状态流转
    - list "任务状态流转":
        - listitem: 草稿
        - listitem: 待执行
        - listitem: 执行中
        - listitem: 成功
        - listitem: 失败
        - listitem: 已取消
    - text: 任务状态会根据启动、取消和主机结果汇总自动推进。 任务级操作 仅允许在可操作状态执行启动或取消，任务结果会同步汇总到详情页。 任务状态会根据启动、取消和主机结果汇总自动推进。
    - button "重试失败步骤"
    - button "删除"
    - table:
        - rowgroup:
            - row "任务来源 软件组件 软件组件 command-light-1794062871424 1.0.0":
                - cell "任务来源"
                - cell "软件组件"
                - cell "软件组件"
                - cell "command-light-1794062871424 1.0.0"
            - row "任务动作 安装 执行模式 固定任务":
                - cell "任务动作"
                - cell "安装"
                - cell "执行模式"
                - cell "固定任务"
            - row "模板参数 - 状态 失败":
                - cell "模板参数"
                - cell "-"
                - cell "状态"
                - cell "失败"
            - row "目标来源 主机 业务域 command-light-1794062871424":
                - cell "目标来源"
                - cell "主机"
                - cell "业务域"
                - cell "command-light-1794062871424"
            - row "执行方式 手动执行 外部任务 ID -":
                - cell "执行方式"
                - cell "手动执行"
                - cell "外部任务 ID"
                - cell "-"
            - row "开始时间 2026-11-07 22:47 结束时间 2026-11-07 22:47":
                - cell "开始时间"
                - cell "2026-11-07 22:47"
                - cell "结束时间"
                - cell "2026-11-07 22:47"
            - row "耗时 - 主机数 1":
                - cell "耗时"
                - cell "-"
                - cell "主机数"
                - cell "1"
            - row "成功数 0 失败数 1":
                - cell "成功数"
                - cell "0"
                - cell "失败数"
                - cell "1"
            - row "执行中 0 跳过数 0":
                - cell "执行中"
                - cell "0"
                - cell "跳过数"
                - cell "0"
            - row "备注 -":
                - cell "备注"
                - cell "-"
    - text: 主机执行明细 先查看各状态分布，再按主机展开过程日志和标准输出。 待执行 0 执行中 0 成功 0 失败 1 已跳过 0
    - table:
        - rowgroup:
            - row "主机 主机 IP 状态 开始时间 结束时间 耗时 错误信息 操作":
                - columnheader "主机"
                - columnheader "主机 IP"
                - columnheader "状态"
                - columnheader "开始时间"
                - columnheader "结束时间"
                - columnheader "耗时"
                - columnheader "错误信息"
                - columnheader "操作"
        - rowgroup:
            - row "command-light-1794062871424 10.245.62.75 失败 2026-11-07 22:47 2026-11-07 22:47 - command fixture failure":
                - cell "command-light-1794062871424"
                - cell "10.245.62.75"
                - cell "失败"
                - cell "2026-11-07 22:47"
                - cell "2026-11-07 22:47"
                - cell "-"
                - cell "command fixture failure"
                - cell
    - list:
        - listitem "上一页": .
        - listitem "第 1 页": "1"
        - listitem "下一页": .
    - region "日志来源":
        - combobox "执行目标": command-light-1794062871424 / 10.245.62.75
        - button "定位首个失败目标"
        - button "重新连接"
        - text: command-light-1794062871424 / 10.245.62.75
        - region "执行步骤":
            - text: 执行步骤
            - list:
                - listitem:
                    - button "定位步骤 1": package_default pending
        - button "展开日志"
        - combobox "日志流":
            - textbox "日志流"
            - text: 日志流
        - region "command-light-1794062871424 / command-light-1794062871424":
            - text: command-light-1794062871424 / command-light-1794062871424
            - textbox "搜索已保留日志"
            - button "暂停追尾"
            - switch "自动换行" [checked]
            - text: 暂无日志
        - text: command-light-1794062871424 10.245.62.75 / 2026-11-07 22:47 失败
        - table:
            - rowgroup:
                - row "步骤 步骤类型 阶段 开始时间 过程信息":
                    - columnheader "步骤"
                    - columnheader "步骤类型"
                    - columnheader "阶段"
                    - columnheader "开始时间"
                    - columnheader "过程信息"
            - rowgroup:
                - row "- - - 进入执行队列 2026-11-07 22:47 Task host queued":
                    - cell "- -"
                    - cell "-"
                    - cell "进入执行队列"
                    - cell "2026-11-07 22:47"
                    - cell "Task host queued"
                - row "- - - 记录执行结果 2026-11-07 22:47 status=failed":
                    - cell "- -"
                    - cell "-"
                    - cell "记录执行结果"
                    - cell "2026-11-07 22:47"
                    - cell "status=failed"
                - row "- - - 错误信息 2026-11-07 22:47 command fixture failure":
                    - cell "- -"
                    - cell "-"
                    - cell "错误信息"
                    - cell "2026-11-07 22:47"
                    - cell "command fixture failure"
        - separator
        - table:
            - rowgroup:
                - row "标准输出 -":
                    - cell "标准输出"
                    - cell "-"
                - row "错误输出 -":
                    - cell "错误输出"
                    - cell "-"
- text: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  1  | import { expect, test, type APIResponse } from '@playwright/test';
  2  | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
  3  | import { startReviewedTask } from './reviewed-start';
  4  |
  5  | async function success<T>(response: APIResponse): Promise<T> {
  6  |  const payload = await response.json(); expect(response.ok(), payload.message).toBeTruthy(); expect(payload.code,payload.message).toBe(200); return payload.data as T;
  7  | }
  8  | type CommandTask = {id:number;status:string;stateVersion:string;hosts:{id:number}[]};
  9  | for(const mode of ['light','dark'] as const){
  10 |  test(`viewed task command conflict refreshes without replay (${mode})`,async({page},testInfo)=>{
  11 |   const login=await loginByApi(page.request,adminCredentials);await installClientSession(page,login);await primeChineseLocale(page);
  12 |   await page.addInitScript(color=>localStorage.setItem('pantheon_color_mode',color),mode);
  13 |   const headers=apiRequestHeaders(login);const token=`command-${mode}-${Date.now()}`;
  14 |   let scopeId=0,hostId=0,packageId=0,taskId=0;let failed=false;
  15 |   try{
  16 |    scopeId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`,{headers,data:{code:token,name:token,environment:'dev',status:'active'}}))).id;
  17 |    hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:token,ip:`10.245.${Math.floor(Math.random()*200)+1}.75`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;
  18 |    packageId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`,{headers,data:{name:token,version:'1.0.0',installCommand:'echo fixture',status:'enabled'}}))).id;
  19 |    taskId=(await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`,{headers,data:{name:token,packageId,businessScopeId:scopeId,targetType:'host',targetIds:[hostId],executorType:'manual'}}))).id;
  20 |    const running=await success<CommandTask>(await startReviewedTask(page.request,taskId,{headers,data:{}}));
  21 |    expect(running.stateVersion).toMatch(/^[a-f0-9]{64}$/);
  22 |    await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${running.hosts[0].id}/result`,{headers,data:{status:'failed',errorMessage:'command fixture failure'}}));
  23 |    const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}));expect(current.status).toBe('failed');
  24 |    // Real public API rejects a missing precondition before mutation.
  25 |    const missing=await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/retry`,{headers,data:{}});expect((await missing.json()).code).not.toBe(200);
  26 |    let submitted=false;let commands=0;let refreshes=0;
  27 |    await page.route(`**/business/deploy/tasks/${taskId}`,async route=>{
  28 |     const response=await route.fetch();const body=await response.json();if(!submitted)body.data.stateVersion=running.stateVersion;else refreshes++;
  29 |     await route.fulfill({response,json:body});
  30 |    });
  31 |    await page.route(`**/business/deploy/tasks/${taskId}/retry`,async route=>{
  32 |     submitted=true;commands++;expect(route.request().postDataJSON().stateVersion).toBe(commands===1?running.stateVersion:current.stateVersion);await route.continue();
  33 |    });
  34 |    await page.setViewportSize({width:1440,height:900});await page.goto(`/business/deploy/task/${taskId}`);
> 35 |    const retry=page.getByRole('button',{name:'重试',exact:true});await expect(retry).toBeVisible();
     |                                                                                    ^ Error: expect(locator).toBeVisible() failed
  36 |    const conflict=page.waitForResponse(response=>response.url().endsWith(`/tasks/${taskId}/retry`));await retry.click();expect((await (await conflict).json()).code).toBe(409);
  37 |    await expect(page.getByText('任务状态已变化，请刷新并确认当前状态后重新操作。',{exact:true})).toBeVisible();await expect.poll(()=>refreshes).toBeGreaterThan(0);
  38 |    expect(commands).toBe(1);
  39 |    expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}))).status).toBe('failed');
  40 |    await page.screenshot({path:testInfo.outputPath(`command-conflict-${mode}-desktop.png`),fullPage:true});
  41 |    await page.setViewportSize({width:390,height:844});await page.screenshot({path:testInfo.outputPath(`command-conflict-${mode}-mobile.png`),fullPage:true});
  42 |    expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
  43 |    await retry.click();await expect(page.getByText('任务状态已变化，请刷新并确认当前状态后重新操作。',{exact:true})).toHaveCount(0);
  44 |    await expect.poll(async()=> (await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}))).status).toBe('running');expect(commands).toBe(2);
  45 |   }catch(error){failed=true;throw error}finally{
  46 |    const cleanupErrors:string[]=[];
  47 |    try{
  48 |     const cleanupHeaders=await verifiedApiHeaders(page.request,login);
  49 |     if(taskId){const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers:cleanupHeaders}));if(current.status==='running')await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/cancel`,{headers:cleanupHeaders,data:{stateVersion:current.stateVersion}}));}
  50 |     for(const [id,endpoint] of [[taskId,'deploy/tasks'],[hostId,'cmdb/hosts'],[scopeId,'bizscope'],[packageId,'deploy/packages']] as const){if(!id)continue;try{await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`,{headers:cleanupHeaders}))}catch(error){cleanupErrors.push(`${endpoint}/${id}: ${String(error)}`)}}
  51 |    }catch(error){cleanupErrors.push(String(error))}
  52 |    await testInfo.attach('command-fixture-cleanup',{contentType:'application/json',body:JSON.stringify({token,taskId,hostId,scopeId,packageId,cleanupErrors})});if(!failed)expect(cleanupErrors).toEqual([]);
  53 |   }
  54 |  });
  55 | }
  56 |
```
