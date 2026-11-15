# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-command.spec.ts >> viewed task command conflict refreshes without replay (light)
- Location: tests\smoke\business\deploy\deploy-command.spec.ts:10:2

# Error details

```
Test timeout of 60000ms exceeded.
```

```
Error: locator.click: Test timeout of 60000ms exceeded.
Call log:
  - waiting for getByRole('button', { name: '重试失败步骤', exact: true })
    - locator resolved to <button type="button" class="arco-btn arco-btn-secondary arco-btn-size-default arco-btn-shape-square">…</button>
  - attempting click action
    - waiting for element to be visible, enabled and stable
    - element is visible, enabled and stable
    - scrolling into view if needed
    - done scrolling
    - <span class="app-shell__menu-entry-label">可观测中心</span> from <aside class="arco-layout-sider arco-layout-sider-light app-shell__sider">…</aside> subtree intercepts pointer events
  - retrying click action
    - waiting for element to be visible, enabled and stable
    - element is visible, enabled and stable
    - scrolling into view if needed
    - done scrolling
    - <div role="menu" class="arco-menu arco-menu-light arco-menu-vertical app-shell__menu">…</div> from <aside class="arco-layout-sider arco-layout-sider-light app-shell__sider">…</aside> subtree intercepts pointer events
  - retrying click action
    - waiting 20ms
    - waiting for element to be visible, enabled and stable
    - element is visible, enabled and stable
    - scrolling into view if needed
    - done scrolling
    - <span class="app-shell__menu-entry-label">可观测中心</span> from <aside class="arco-layout-sider arco-layout-sider-light app-shell__sider">…</aside> subtree intercepts pointer events
  2 × retrying click action
      - waiting 100ms
      - waiting for element to be visible, enabled and stable
      - element is visible, enabled and stable
      - scrolling into view if needed
      - done scrolling
      - <div class="app-shell__brand">…</div> from <aside class="arco-layout-sider arco-layout-sider-light app-shell__sider">…</aside> subtree intercepts pointer events
  26 × retrying click action
       - waiting 500ms
       - waiting for element to be visible, enabled and stable
       - element is visible, enabled and stable
       - scrolling into view if needed
       - done scrolling
       - <div role="menu" class="arco-menu arco-menu-light arco-menu-vertical app-shell__menu">…</div> from <aside class="arco-layout-sider arco-layout-sider-light app-shell__sider">…</aside> subtree intercepts pointer events
     - retrying click action
       - waiting 500ms
       - waiting for element to be visible, enabled and stable
       - element is visible, enabled and stable
       - scrolling into view if needed
       - done scrolling
       - <span class="app-shell__menu-entry-label">可观测中心</span> from <aside class="arco-layout-sider arco-layout-sider-light app-shell__sider">…</aside> subtree intercepts pointer events
     - retrying click action
       - waiting 500ms
       - waiting for element to be visible, enabled and stable
       - element is visible, enabled and stable
       - scrolling into view if needed
       - done scrolling
       - <div class="app-shell__brand">…</div> from <aside class="arco-layout-sider arco-layout-sider-light app-shell__sider">…</aside> subtree intercepts pointer events
     - retrying click action
       - waiting 500ms
       - waiting for element to be visible, enabled and stable
       - element is visible, enabled and stable
       - scrolling into view if needed
       - done scrolling
       - <div class="app-shell__brand">…</div> from <aside class="arco-layout-sider arco-layout-sider-light app-shell__sider">…</aside> subtree intercepts pointer events
  - retrying click action
    - waiting 500ms
    - waiting for element to be visible, enabled and stable
    - element is visible, enabled and stable
    - scrolling into view if needed
    - done scrolling
    - <div role="menu" class="arco-menu arco-menu-light arco-menu-vertical app-shell__menu">…</div> from <aside class="arco-layout-sider arco-layout-sider-light app-shell__sider">…</aside> subtree intercepts pointer events
  - retrying click action
    - waiting 500ms
    - waiting for element to be visible, enabled and stable

```

# Page snapshot

```yaml
- generic [ref=e3]:
    - complementary [ref=e4]:
        - generic [ref=e5]:
            - generic [ref=e6]: P
            - menu [ref=e10]:
                - generic [ref=e11]:
                    - menuitem [ref=e12] [cursor=pointer]:
                        - button "工作台" [ref=e13]
                    - button "CMDB" [ref=e23] [cursor=pointer]
                    - menuitem [ref=e35] [cursor=pointer]:
                        - button "业务域" [ref=e36]
                    - menuitem [ref=e45] [cursor=pointer]:
                        - button "服务目录" [ref=e46]
                    - button "安装部署" [ref=e58] [cursor=pointer]
                    - button "Kubernetes" [ref=e70] [cursor=pointer]
                    - button "可观测中心" [ref=e82] [cursor=pointer]
                    - button "访问控制" [ref=e94] [cursor=pointer]
                    - button "组织架构" [ref=e108] [cursor=pointer]
                    - button "安全审计" [ref=e123] [cursor=pointer]
                    - button "低代码平台" [ref=e135] [cursor=pointer]
                    - button "平台配置" [ref=e147] [cursor=pointer]
                    - menuitem [ref=e156] [cursor=pointer]:
                        - button "文件分发" [ref=e157]
                    - menuitem [ref=e166] [cursor=pointer]:
                        - button "ArgoCD" [ref=e167]
    - button "关闭" [ref=e173] [cursor=pointer]
    - generic [ref=e174]:
        - generic [ref=e175]:
            - generic [ref=e176]:
                - button "切换导航" [active] [ref=e177] [cursor=pointer]
                - list [ref=e181]:
                    - listitem [ref=e182]: 首页
                    - listitem [ref=e186]: 任务详情
            - generic [ref=e187]:
                - button "平台功能台" [ref=e189]:
                    - generic [ref=e192]: 搜索菜单、页面、功能…
                - button "通知中心" [ref=e194] [cursor=pointer]
                - button "平台偏好" [ref=e198] [cursor=pointer]
                - 'button "用户: Administrator" [ref=e203] [cursor=pointer]':
                    - generic [ref=e204]: A
        - tablist "已打开页面" [ref=e206]:
            - tab "工作台" [ref=e207] [cursor=pointer]
            - tab "任务详情" [selected] [ref=e211] [cursor=pointer]
        - main [ref=e213]:
            - generic [ref=e215]:
                - generic [ref=e216]:
                    - generic [ref=e217]:
                        - heading "command-light-1794063381418" [level=5] [ref=e219]
                        - generic [ref=e220]: 任务详情
                    - button "返回" [ref=e224] [cursor=pointer]
                - generic [ref=e227]:
                    - status [ref=e229]:
                        - generic [ref=e233]: 任务状态已变化，请刷新并确认当前状态后重新操作。
                    - generic [ref=e237]:
                        - generic [ref=e238]:
                            - generic [ref=e239]:
                                - generic [ref=e240]: 运维平台 / 部署任务
                                - heading "command-light-1794063381418" [level=5] [ref=e241]
                                - generic [ref=e242]: 查看任务摘要、状态流转和每台目标主机的执行结果，便于运维排障与人工收口。
                            - generic [ref=e243]: 失败
                        - generic [ref=e245]:
                            - generic [ref=e246]:
                                - generic [ref=e247]: 目标主机
                                - generic [ref=e248]: "1"
                            - generic [ref=e249]:
                                - generic [ref=e250]: 成功
                                - generic [ref=e251]: "0"
                            - generic [ref=e252]:
                                - generic [ref=e253]: 失败
                                - generic [ref=e254]: "1"
                    - generic [ref=e258]:
                        - generic [ref=e259]: 任务状态流转
                        - list "任务状态流转" [ref=e261]:
                            - listitem [ref=e262]:
                                - generic [ref=e264]: 草稿
                            - listitem [ref=e265]:
                                - generic [ref=e267]: 待执行
                            - listitem [ref=e268]:
                                - generic [ref=e270]: 执行中
                            - listitem [ref=e271]:
                                - generic [ref=e273]: 成功
                            - listitem [ref=e274]:
                                - generic [ref=e276]: 失败
                            - listitem [ref=e277]:
                                - generic [ref=e279]: 已取消
                        - generic [ref=e280]: 任务状态会根据启动、取消和主机结果汇总自动推进。
                    - generic [ref=e284]:
                        - generic [ref=e285]: 任务级操作
                        - generic [ref=e287]:
                            - generic [ref=e288]:
                                - generic [ref=e289]: 仅允许在可操作状态执行启动或取消，任务结果会同步汇总到详情页。
                                - generic [ref=e290]: 任务状态会根据启动、取消和主机结果汇总自动推进。
                            - generic [ref=e291]:
                                - button "重试失败步骤" [ref=e293] [cursor=pointer]
                                - button "删除" [ref=e297] [cursor=pointer]
                    - table [ref=e305]:
                        - rowgroup [ref=e306]:
                            - row [ref=e307]:
                                - cell "任务来源" [ref=e308]
                                - cell "软件组件" [ref=e309]
                                - cell "软件组件" [ref=e310]
                                - cell "command-light-1794063381418 1.0.0" [ref=e311]
                            - row [ref=e312]:
                                - cell "任务动作" [ref=e313]
                                - cell "安装" [ref=e314]
                                - cell "执行模式" [ref=e315]
                                - cell "固定任务" [ref=e316]
                            - row [ref=e317]:
                                - cell "模板参数" [ref=e318]
                                - cell "-" [ref=e319]
                                - cell "状态" [ref=e320]
                                - cell "失败" [ref=e321]
                            - row [ref=e322]:
                                - cell "目标来源" [ref=e323]
                                - cell "主机" [ref=e324]
                                - cell "业务域" [ref=e325]
                                - cell "command-light-1794063381418" [ref=e326]
                            - row [ref=e327]:
                                - cell "执行方式" [ref=e328]
                                - cell "手动执行" [ref=e329]
                                - cell "外部任务 ID" [ref=e330]
                                - cell "-" [ref=e331]
                            - row [ref=e332]:
                                - cell "开始时间" [ref=e333]
                                - cell "2026-11-07 22:56" [ref=e334]
                                - cell "结束时间" [ref=e335]
                                - cell "2026-11-07 22:56" [ref=e336]
                            - row [ref=e337]:
                                - cell "耗时" [ref=e338]
                                - cell "-" [ref=e339]
                                - cell "主机数" [ref=e340]
                                - cell "1" [ref=e341]
                            - row [ref=e342]:
                                - cell "成功数" [ref=e343]
                                - cell "0" [ref=e344]
                                - cell "失败数" [ref=e345]
                                - cell "1" [ref=e346]
                            - row [ref=e347]:
                                - cell "执行中" [ref=e348]
                                - cell "0" [ref=e349]
                                - cell "跳过数" [ref=e350]
                                - cell "0" [ref=e351]
                            - row [ref=e352]:
                                - cell "备注" [ref=e353]
                                - cell "-" [ref=e354]
                    - generic [ref=e358]:
                        - generic [ref=e361]:
                            - generic [ref=e362]: 主机执行明细
                            - generic [ref=e363]: 先查看各状态分布，再按主机展开过程日志和标准输出。
                        - generic [ref=e365]:
                            - generic [ref=e366]:
                                - generic [ref=e367]: 待执行
                                - generic [ref=e368]: "0"
                            - generic [ref=e369]:
                                - generic [ref=e370]: 执行中
                                - generic [ref=e371]: "0"
                            - generic [ref=e372]:
                                - generic [ref=e373]: 成功
                                - generic [ref=e374]: "0"
                            - generic [ref=e375]:
                                - generic [ref=e376]: 失败
                                - generic [ref=e377]: "1"
                            - generic [ref=e378]:
                                - generic [ref=e379]: 已跳过
                                - generic [ref=e380]: "0"
                        - generic [ref=e382]:
                            - generic [ref=e383]:
                                - generic [ref=e384]: 共 1 条
                                - generic [ref=e385]: 左右滑动查看更多字段
                            - generic [ref=e388]:
                                - table [ref=e392]:
                                    - rowgroup [ref=e402]:
                                        - row [ref=e403]:
                                            - columnheader "主机" [ref=e404]
                                            - columnheader "主机 IP" [ref=e406]
                                            - columnheader "状态" [ref=e408]
                                            - columnheader "开始时间" [ref=e410]
                                            - columnheader "结束时间" [ref=e412]
                                            - columnheader "耗时" [ref=e414]
                                            - columnheader "错误信息" [ref=e416]
                                            - columnheader "操作" [ref=e418]
                                    - rowgroup [ref=e420]:
                                        - row [ref=e421]:
                                            - cell "command-light-1794063381418" [ref=e422]
                                            - cell "10.245.153.75" [ref=e424]
                                            - cell "失败" [ref=e426]
                                            - cell "2026-11-07 22:56" [ref=e431]
                                            - cell "2026-11-07 22:56" [ref=e433]
                                            - cell "-" [ref=e435]
                                            - cell "command fixture failure" [ref=e437]
                                            - cell [ref=e439]
                                - list [ref=e445]:
                                    - listitem "上一页" [ref=e446]: .
                                    - listitem "第 1 页" [ref=e449] [cursor=pointer]: "1"
                                    - listitem "下一页" [ref=e450]: .
                    - region "日志来源" [ref=e454]:
                        - generic [ref=e455]:
                            - combobox "执行目标" [ref=e457] [cursor=pointer]:
                                - generic "command-light-1794063381418 / 10.245.153.75" [ref=e458]:
                                    - generic [ref=e459]:
                                        - textbox [aria-hidden]
                                        - generic [ref=e460]: command-light-1794063381418 / 10.245.153.75
                            - button "定位首个失败目标" [ref=e466] [cursor=pointer]
                            - button "重新连接" [ref=e468] [cursor=pointer]
                            - generic [ref=e469]: command-light-1794063381418 / 10.245.153.75
                        - region "执行步骤" [ref=e472]:
                            - list [ref=e475]:
                                - listitem [ref=e476]:
                                    - button "定位步骤 1" [ref=e477]:
                                        - generic [ref=e479]: package_default
                                        - generic [ref=e480]: pending
                        - generic [ref=e483]:
                            - generic [ref=e484]:
                                - button "展开日志" [ref=e486] [cursor=pointer]
                                - combobox "日志流" [ref=e488] [cursor=pointer]:
                                    - generic [ref=e490]:
                                        - textbox "日志流"
                                        - generic [ref=e491]: 日志流
                            - region "command-light-1794063381418 / command-light-1794063381418" [ref=e496]:
                                - generic [ref=e497]:
                                    - generic [ref=e498]: command-light-1794063381418 / command-light-1794063381418
                                    - generic [ref=e499]:
                                        - textbox "搜索已保留日志" [ref=e503]
                                        - button "暂停追尾" [ref=e507] [cursor=pointer]
                                        - switch "自动换行" [checked] [ref=e508] [cursor=pointer]
                                - generic [ref=e510]: 暂无日志
                        - generic [ref=e515]:
                            - generic [ref=e517]:
                                - generic [ref=e518]:
                                    - generic [ref=e519]: command-light-1794063381418
                                    - generic [ref=e520]: 10.245.153.75 / 2026-11-07 22:56
                                - generic [ref=e522]: 失败
                            - generic [ref=e525]:
                                - generic [ref=e526]:
                                    - generic [ref=e527]: 共 3 条
                                    - generic [ref=e528]: 左右滑动查看更多字段
                                - table [ref=e535]:
                                    - rowgroup [ref=e542]:
                                        - row [ref=e543]:
                                            - columnheader "步骤" [ref=e544]
                                            - columnheader "步骤类型" [ref=e546]
                                            - columnheader "阶段" [ref=e548]
                                            - columnheader "开始时间" [ref=e550]
                                            - columnheader "过程信息" [ref=e552]
                                    - rowgroup [ref=e554]:
                                        - row [ref=e555]:
                                            - cell "- -" [ref=e556]:
                                                - generic [ref=e559]:
                                                    - generic [ref=e560]: "-"
                                                    - generic [ref=e561]: "-"
                                            - cell "-" [ref=e562]
                                            - cell "进入执行队列" [ref=e564]
                                            - cell "2026-11-07 22:56" [ref=e569]
                                            - cell "Task host queued" [ref=e571]
                                        - row [ref=e573]:
                                            - cell "- -" [ref=e574]:
                                                - generic [ref=e577]:
                                                    - generic [ref=e578]: "-"
                                                    - generic [ref=e579]: "-"
                                            - cell "-" [ref=e580]
                                            - cell "记录执行结果" [ref=e582]
                                            - cell "2026-11-07 22:56" [ref=e587]
                                            - cell "status=failed" [ref=e589]
                                        - row [ref=e591]:
                                            - cell "- -" [ref=e592]:
                                                - generic [ref=e595]:
                                                    - generic [ref=e596]: "-"
                                                    - generic [ref=e597]: "-"
                                            - cell "-" [ref=e598]
                                            - cell "错误信息" [ref=e600]
                                            - cell "2026-11-07 22:56" [ref=e605]
                                            - cell "command fixture failure" [ref=e607]
                            - generic [ref=e609]:
                                - separator
                            - table [ref=e613]:
                                - rowgroup [ref=e614]:
                                    - row [ref=e615]:
                                        - cell "标准输出" [ref=e616]
                                        - cell "-" [ref=e617]
                                    - row [ref=e619]:
                                        - cell "错误输出" [ref=e620]
                                        - cell "-" [ref=e621]
        - generic [ref=e623]: Pantheon Platform · Enterprise Admin Foundation
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
  35 |    const retry=page.getByRole('button',{name:'重试失败步骤',exact:true});await expect(retry).toBeVisible();
  36 |    const conflict=page.waitForResponse(response=>response.url().endsWith(`/tasks/${taskId}/retry`));await retry.click();expect((await (await conflict).json()).code).toBe(409);
  37 |    await expect(page.locator('.arco-alert-content').getByText('任务状态已变化，请刷新并确认当前状态后重新操作。',{exact:true})).toBeVisible();await expect.poll(()=>refreshes).toBeGreaterThan(0);
  38 |    expect(commands).toBe(1);
  39 |    expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}))).status).toBe('failed');
  40 |    await page.screenshot({path:testInfo.outputPath(`command-conflict-${mode}-desktop.png`),fullPage:true});
  41 |    await page.setViewportSize({width:390,height:844});await page.getByRole('button',{name:'切换导航',exact:true}).click();await page.evaluate(()=>window.scrollTo(0,0));await page.screenshot({path:testInfo.outputPath(`command-conflict-${mode}-mobile.png`),fullPage:true});
  42 |    expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
> 43 |    await retry.click();await expect(page.locator('.arco-alert-content').getByText('任务状态已变化，请刷新并确认当前状态后重新操作。',{exact:true})).toHaveCount(0);
     |                ^ Error: locator.click: Test timeout of 60000ms exceeded.
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
  57 | test('batch cancel retains failed selection and reports partial failure',async({page},testInfo)=>{
  58 |  const login=await loginByApi(page.request,adminCredentials);await installClientSession(page,login);await primeChineseLocale(page);
  59 |  const headers=apiRequestHeaders(login);const token=`command-batch-${Date.now()}`;
  60 |  let scopeId=0,hostId=0,packageId=0;const taskIds:number[]=[];const hostIds:number[]=[];let failed=false;
  61 |  try{
  62 |   scopeId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`,{headers,data:{code:token,name:token,environment:'dev',status:'active'}}))).id;
  63 |   hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:token,ip:`10.244.${Math.floor(Math.random()*200)+1}.77`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;
  64 |   hostIds.push(hostId);
  65 |   packageId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`,{headers,data:{name:token,version:'1.0.0',installCommand:'echo batch fixture',status:'enabled'}}))).id;
  66 |   let staleVersion='';
  67 |   for(const suffix of ['a','b']){
  68 |    if(suffix==='b'){
  69 |     hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:`${token}-b-host`,ip:`10.243.${Math.floor(Math.random()*200)+1}.77`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;hostIds.push(hostId);
  70 |    }
  71 |    const created=await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`,{headers,data:{name:`${token}-${suffix}`,packageId,businessScopeId:scopeId,targetType:'host',targetIds:[hostId],executorType:'manual'}}));taskIds.push(created.id);staleVersion=created.stateVersion;
  72 |    await success(await startReviewedTask(page.request,created.id,{headers,data:{}}));
  73 |   }
  74 |   let submissions=0;
  75 |   await page.route('**/business/deploy/tasks?*',async route=>{const response=await route.fetch();const body=await response.json();if(submissions===0){const item=body.data.items.find((item:CommandTask)=>item.id===taskIds[1]);if(item)item.stateVersion=staleVersion;}await route.fulfill({response,json:body});});
  76 |   await page.route('**/business/deploy/tasks/*/cancel',async route=>{submissions++;await route.continue();});
  77 |   await page.goto('/business/deploy/task');
  78 |   const filtered=page.waitForResponse(response=>response.request().method()==='GET'&&response.url().includes('/business/deploy/tasks?')&&new URL(response.url()).searchParams.get('keyword')===token);
  79 |   await page.locator('.search-toolbar input').first().fill(token);await filtered;
  80 |   const rowA=page.getByRole('row').filter({hasText:`${token}-a`}).first();const rowB=page.getByRole('row').filter({hasText:`${token}-b`}).first();
  81 |   await expect(rowA).toBeVisible();await expect(rowB).toBeVisible();await rowA.locator('label.arco-checkbox').click();await expect(rowA.getByRole('checkbox')).toBeChecked();await rowB.locator('label.arco-checkbox').click();await expect(rowB.getByRole('checkbox')).toBeChecked();
  82 |   await page.getByRole('button',{name:'取消',exact:true}).first().click();await page.getByRole('button',{name:'确定',exact:true}).click();
  83 |   await expect(page.locator('.arco-alert-content').getByText('部分任务未能取消。列表已刷新，请确认剩余选择后重新操作。',{exact:true})).toBeVisible();
  84 |   await expect(rowA.getByRole('checkbox')).not.toBeChecked();await expect(rowB.getByRole('checkbox')).toBeChecked();expect(submissions).toBe(2);
  85 |   expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskIds[0]}`,{headers}))).status).toBe('canceled');
  86 |   expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskIds[1]}`,{headers}))).status).toBe('running');
  87 |   await page.screenshot({path:testInfo.outputPath('command-batch-partial.png'),fullPage:true});
  88 |  }catch(error){failed=true;throw error}finally{
  89 |   const errors:string[]=[];
  90 |   try{
  91 |    const cleanupHeaders=await verifiedApiHeaders(page.request,login);
  92 |    for(const id of taskIds){const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${id}`,{headers:cleanupHeaders}));if(current.status==='running')await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${id}/cancel`,{headers:cleanupHeaders,data:{stateVersion:current.stateVersion}}));await success(await page.request.delete(`${apiBaseUrl}/business/deploy/tasks/${id}`,{headers:cleanupHeaders}));}
  93 |    for(const id of hostIds)await success(await page.request.delete(`${apiBaseUrl}/business/cmdb/hosts/${id}`,{headers:cleanupHeaders}));
  94 |    for(const [id,endpoint] of [[scopeId,'bizscope'],[packageId,'deploy/packages']] as const){if(id)await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`,{headers:cleanupHeaders}));}
  95 |   }catch(error){errors.push(String(error))}
  96 |   await testInfo.attach('batch-fixture-cleanup',{contentType:'application/json',body:JSON.stringify({token,taskIds,hostId,scopeId,packageId,errors})});if(!failed)expect(errors).toEqual([]);
  97 |  }
  98 | });
  99 |
```
