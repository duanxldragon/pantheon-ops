# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-command.spec.ts >> batch cancel retains failed selection and reports partial failure
- Location: tests\smoke\business\deploy\deploy-command.spec.ts:57:1

# Error details

```
Error: business.deploy.task.leaseConflict

expect(received).toBeTruthy()

Received: false
```

# Test source

```ts
  1  | import { expect, test, type APIResponse } from '@playwright/test';
  2  | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
  3  | import { startReviewedTask } from './reviewed-start';
  4  |
  5  | async function success<T>(response: APIResponse): Promise<T> {
> 6  |  const payload = await response.json(); expect(response.ok(), payload.message).toBeTruthy(); expect(payload.code,payload.message).toBe(200); return payload.data as T;
     |                                                                                ^ Error: business.deploy.task.leaseConflict
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
  41 |    await page.setViewportSize({width:390,height:844});await page.screenshot({path:testInfo.outputPath(`command-conflict-${mode}-mobile.png`),fullPage:true});
  42 |    expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
  43 |    await retry.click();await expect(page.locator('.arco-alert-content').getByText('任务状态已变化，请刷新并确认当前状态后重新操作。',{exact:true})).toHaveCount(0);
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
  60 |  let scopeId=0,hostId=0,packageId=0;const taskIds:number[]=[];let failed=false;
  61 |  try{
  62 |   scopeId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`,{headers,data:{code:token,name:token,environment:'dev',status:'active'}}))).id;
  63 |   hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:token,ip:`10.244.${Math.floor(Math.random()*200)+1}.77`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;
  64 |   packageId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`,{headers,data:{name:token,version:'1.0.0',installCommand:'echo batch fixture',status:'enabled'}}))).id;
  65 |   let staleVersion='';
  66 |   for(const suffix of ['a','b']){
  67 |    const created=await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`,{headers,data:{name:`${token}-${suffix}`,packageId,businessScopeId:scopeId,targetType:'host',targetIds:[hostId],executorType:'manual'}}));taskIds.push(created.id);staleVersion=created.stateVersion;
  68 |    await success(await startReviewedTask(page.request,created.id,{headers,data:{}}));
  69 |   }
  70 |   let submissions=0;
  71 |   await page.route('**/business/deploy/tasks?*',async route=>{const response=await route.fetch();const body=await response.json();if(submissions===0){const item=body.data.items.find((item:CommandTask)=>item.id===taskIds[1]);if(item)item.stateVersion=staleVersion;}await route.fulfill({response,json:body});});
  72 |   await page.route('**/business/deploy/tasks/*/cancel',async route=>{submissions++;await route.continue();});
  73 |   await page.goto('/business/deploy/task');await page.locator('.search-toolbar input').first().fill(token);
  74 |   const rowA=page.getByRole('row').filter({hasText:`${token}-a`}).first();const rowB=page.getByRole('row').filter({hasText:`${token}-b`}).first();
  75 |   await expect(rowA).toBeVisible();await expect(rowB).toBeVisible();await rowA.getByRole('checkbox').check();await rowB.getByRole('checkbox').check();
  76 |   await page.getByRole('button',{name:'取消',exact:true}).first().click();await page.getByRole('button',{name:'确定',exact:true}).click();
  77 |   await expect(page.locator('.arco-alert-content').getByText('部分任务未能取消。列表已刷新，请确认剩余选择后重新操作。',{exact:true})).toBeVisible();
  78 |   await expect(rowA.getByRole('checkbox')).not.toBeChecked();await expect(rowB.getByRole('checkbox')).toBeChecked();expect(submissions).toBe(2);
  79 |   expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskIds[0]}`,{headers}))).status).toBe('canceled');
  80 |   expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskIds[1]}`,{headers}))).status).toBe('running');
  81 |   await page.screenshot({path:testInfo.outputPath('command-batch-partial.png'),fullPage:true});
  82 |  }catch(error){failed=true;throw error}finally{
  83 |   const errors:string[]=[];
  84 |   try{
  85 |    const cleanupHeaders=await verifiedApiHeaders(page.request,login);
  86 |    for(const id of taskIds){const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${id}`,{headers:cleanupHeaders}));if(current.status==='running')await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${id}/cancel`,{headers:cleanupHeaders,data:{stateVersion:current.stateVersion}}));await success(await page.request.delete(`${apiBaseUrl}/business/deploy/tasks/${id}`,{headers:cleanupHeaders}));}
  87 |    for(const [id,endpoint] of [[hostId,'cmdb/hosts'],[scopeId,'bizscope'],[packageId,'deploy/packages']] as const){if(id)await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`,{headers:cleanupHeaders}));}
  88 |   }catch(error){errors.push(String(error))}
  89 |   await testInfo.attach('batch-fixture-cleanup',{contentType:'application/json',body:JSON.stringify({token,taskIds,hostId,scopeId,packageId,errors})});if(!failed)expect(errors).toEqual([]);
  90 |  }
  91 | });
  92 |
```
