import { expect, test, type APIResponse } from '@playwright/test';
import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
import { startReviewedTask } from './reviewed-start';

// Includes real fixture lifecycle, multiple rendered states and remote asset loading.
// Keep assertion deadlines unchanged while giving this scenario a bounded budget.
test.describe.configure({ timeout: 90_000 });

async function success<T>(response: APIResponse): Promise<T> {
 const payload = await response.json(); expect(response.ok(), payload.message).toBeTruthy(); expect(payload.code,payload.message).toBe(200); return payload.data as T;
}
type CommandTask = {id:number;status:string;stateVersion:string;hosts:{id:number}[]};
for(const mode of ['light','dark'] as const){
 test(`viewed task command conflict refreshes without replay (${mode})`,async({page},testInfo)=>{
  const login=await loginByApi(page.request,adminCredentials);await installClientSession(page,login);await primeChineseLocale(page);
  await page.addInitScript(color=>localStorage.setItem('pantheon_color_mode',color),mode);
  const headers=apiRequestHeaders(login);const token=`command-${mode}-${Date.now()}`;
  let scopeId=0,hostId=0,packageId=0,taskId=0;let failed=false;
  try{
   scopeId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`,{headers,data:{code:token,name:token,environment:'dev',status:'active'}}))).id;
   hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:token,ip:`10.245.${Math.floor(Math.random()*200)+1}.${Math.floor(Math.random()*200)+1}`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;
   packageId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`,{headers,data:{name:token,version:'1.0.0',installCommand:'echo fixture',status:'enabled'}}))).id;
   taskId=(await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`,{headers,data:{name:token,packageId,businessScopeId:scopeId,targetType:'host',targetIds:[hostId],executorType:'manual'}}))).id;
   let running=await success<CommandTask>(await startReviewedTask(page.request,taskId,{headers,data:{}}));
   expect(running.stateVersion).toMatch(/^[a-f0-9]{64}$/);
   const paused=await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/pause`,{headers,data:{stateVersion:running.stateVersion}}));expect(paused.status).toBe('paused');
   running=await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/resume`,{headers,data:{stateVersion:paused.stateVersion}}));expect(running.status).toBe('running');
   await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${running.hosts[0].id}/result`,{headers,data:{status:'failed',errorMessage:'command fixture failure'}}));
   const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}));expect(current.status).toBe('failed');
   // Real public API rejects a missing precondition before mutation.
   const missing=await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/retry`,{headers,data:{}});expect((await missing.json()).code).not.toBe(200);
   let submitted=false;let commands=0;let refreshes=0;
   await page.route(`**/business/deploy/tasks/${taskId}`,async route=>{
    const response=await route.fetch();const body=await response.json();if(!submitted)body.data.stateVersion=running.stateVersion;else refreshes++;
    await route.fulfill({response,json:body});
   });
   await page.route(`**/business/deploy/tasks/${taskId}/retry`,async route=>{
    submitted=true;commands++;expect(route.request().postDataJSON().stateVersion).toBe(commands===1?running.stateVersion:current.stateVersion);await route.continue();
   });
   await page.setViewportSize({width:1440,height:900});await page.goto(`/business/deploy/task/${taskId}`);
   const retry=page.getByRole('button',{name:'重试失败步骤',exact:true});await expect(retry).toBeVisible();
   const conflict=page.waitForResponse(response=>response.url().endsWith(`/tasks/${taskId}/retry`));await retry.click();expect((await (await conflict).json()).code).toBe(409);
   await expect(page.locator('.arco-alert-content').getByText('任务状态已变化，请刷新并确认当前状态后重新操作。',{exact:true})).toBeVisible();await expect.poll(()=>refreshes).toBeGreaterThan(0);
   expect(commands).toBe(1);
   expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}))).status).toBe('failed');
   await page.screenshot({path:testInfo.outputPath(`command-conflict-${mode}-desktop.png`),fullPage:true});
   await page.setViewportSize({width:390,height:844});if(await page.locator('.app-shell--mobile-open').count())await page.getByRole('button',{name:'切换导航',exact:true}).click();await expect(page.locator('.app-shell__sider')).not.toBeInViewport();await page.evaluate(()=>window.scrollTo(0,0));await page.screenshot({path:testInfo.outputPath(`command-conflict-${mode}-mobile.png`),fullPage:true});
   expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
   await retry.click();await expect(page.locator('.arco-alert-content').getByText('任务状态已变化，请刷新并确认当前状态后重新操作。',{exact:true})).toHaveCount(0);
   await expect.poll(async()=> (await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}))).status).toBe('running');expect(commands).toBe(2);
  }catch(error){failed=true;throw error}finally{
   const cleanupErrors:string[]=[];
   try{
    const cleanupHeaders=await verifiedApiHeaders(page.request,login);
    if(taskId){const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers:cleanupHeaders}));if(current.status==='running')await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/cancel`,{headers:cleanupHeaders,data:{stateVersion:current.stateVersion}}));}
    for(const [id,endpoint] of [[taskId,'deploy/tasks'],[hostId,'cmdb/hosts'],[scopeId,'bizscope'],[packageId,'deploy/packages']] as const){if(!id)continue;try{await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`,{headers:cleanupHeaders}))}catch(error){cleanupErrors.push(`${endpoint}/${id}: ${String(error)}`)}}
   }catch(error){cleanupErrors.push(String(error))}
   await testInfo.attach('command-fixture-cleanup',{contentType:'application/json',body:JSON.stringify({token,taskId,hostId,scopeId,packageId,cleanupErrors})});if(!failed)expect(cleanupErrors).toEqual([]);
  }
 });
}

test('batch cancel retains failed selection and reports partial failure',async({page},testInfo)=>{
 const login=await loginByApi(page.request,adminCredentials);await installClientSession(page,login);await primeChineseLocale(page);
 const headers=apiRequestHeaders(login);const token=`command-batch-${Date.now()}`;
 let scopeId=0,hostId=0,packageId=0;const taskIds:number[]=[];const hostIds:number[]=[];let failed=false;
 try{
  scopeId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`,{headers,data:{code:token,name:token,environment:'dev',status:'active'}}))).id;
  hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:token,ip:`10.244.${Math.floor(Math.random()*200)+1}.${Math.floor(Math.random()*200)+1}`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;
  hostIds.push(hostId);
  packageId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`,{headers,data:{name:token,version:'1.0.0',installCommand:'echo batch fixture',status:'enabled'}}))).id;
  let staleVersion='';
  for(const suffix of ['a','b']){
   if(suffix==='b'){
    hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:`${token}-b-host`,ip:`10.243.${Math.floor(Math.random()*200)+1}.${Math.floor(Math.random()*200)+1}`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;hostIds.push(hostId);
   }
   const created=await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`,{headers,data:{name:`${token}-${suffix}`,packageId,businessScopeId:scopeId,targetType:'host',targetIds:[hostId],executorType:'manual'}}));taskIds.push(created.id);staleVersion=created.stateVersion;
   await success(await startReviewedTask(page.request,created.id,{headers,data:{}}));
  }
  let submissions=0;
  await page.route('**/business/deploy/tasks?*',async route=>{const response=await route.fetch();const body=await response.json();if(submissions===0){const item=body.data.items.find((item:CommandTask)=>item.id===taskIds[1]);if(item)item.stateVersion=staleVersion;}await route.fulfill({response,json:body});});
  await page.route('**/business/deploy/tasks/*/cancel',async route=>{submissions++;await route.continue();});
  await page.goto('/business/deploy/task');
  const filtered=page.waitForResponse(response=>response.request().method()==='GET'&&response.url().includes('/business/deploy/tasks?')&&new URL(response.url()).searchParams.get('keyword')===token);
  await page.locator('.search-toolbar input').first().fill(token);await filtered;
  const rowA=page.getByRole('row').filter({hasText:`${token}-a`}).first();const rowB=page.getByRole('row').filter({hasText:`${token}-b`}).first();
  await expect(rowA).toBeVisible();await expect(rowB).toBeVisible();await rowA.locator('label.arco-checkbox').click();await expect(rowA.getByRole('checkbox')).toBeChecked();await rowB.locator('label.arco-checkbox').click();await expect(rowB.getByRole('checkbox')).toBeChecked();
  await page.getByRole('button',{name:'取消',exact:true}).first().click();await page.getByRole('button',{name:'确定',exact:true}).click();
  await expect(page.locator('.arco-alert-content').getByText('部分任务未能取消。列表已刷新，请确认剩余选择后重新操作。',{exact:true})).toBeVisible();
  await expect(rowA.getByRole('checkbox')).not.toBeChecked();await expect(rowB.getByRole('checkbox')).toBeChecked();expect(submissions).toBe(2);
  expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskIds[0]}`,{headers}))).status).toBe('canceled');
  expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskIds[1]}`,{headers}))).status).toBe('running');
  await page.screenshot({path:testInfo.outputPath('command-batch-partial.png'),fullPage:true});
 }catch(error){failed=true;throw error}finally{
  const errors:string[]=[];
  try{
   const cleanupHeaders=await verifiedApiHeaders(page.request,login);
   for(const id of taskIds){const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${id}`,{headers:cleanupHeaders}));if(current.status==='running')await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${id}/cancel`,{headers:cleanupHeaders,data:{stateVersion:current.stateVersion}}));await success(await page.request.delete(`${apiBaseUrl}/business/deploy/tasks/${id}`,{headers:cleanupHeaders}));}
   for(const id of hostIds)await success(await page.request.delete(`${apiBaseUrl}/business/cmdb/hosts/${id}`,{headers:cleanupHeaders}));
   for(const [id,endpoint] of [[scopeId,'bizscope'],[packageId,'deploy/packages']] as const){if(id)await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`,{headers:cleanupHeaders}));}
  }catch(error){errors.push(String(error))}
  await testInfo.attach('batch-fixture-cleanup',{contentType:'application/json',body:JSON.stringify({token,taskIds,hostId,scopeId,packageId,errors})});if(!failed)expect(errors).toEqual([]);
 }
});
