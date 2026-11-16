import { afterEach, describe, expect, it, vi } from 'vitest';
import { getDatasourceList, getMonitoringTargetList, getTrace, queryLogs, queryMetrics, restoreQueryHistory, saveQueryView, searchTraces } from '../../../../src/modules/business/observability/api';
import { getBizScopeOptions } from '../../../../src/modules/business/bizscope/api';
import { initialQueryRecipe } from '../../../../src/modules/business/observability/queryWorkspaceState';
const client=vi.hoisted(()=>({request:vi.fn()}));
vi.mock('@duanxldragon/pantheon-base-ui',()=>({apiRequest:client.request}));
afterEach(()=>client.request.mockReset());
describe('query workspace HTTP contract',()=>{
  it('passes cancellation to catalogue lists and keeps old BizScope callers compatible',()=>{
    const controller=new AbortController();getDatasourceList(controller.signal);getMonitoringTargetList(controller.signal);getBizScopeOptions(controller.signal);getBizScopeOptions();
    expect(client.request.mock.calls.slice(0,3).every(([config])=>config.signal===controller.signal)).toBe(true);
    expect(client.request.mock.calls[2][0]).toMatchObject({url:'/business/bizscope/options',method:'get'});
    expect(client.request.mock.calls[3][0].signal).toBeUndefined();
  });
  it('keeps raw editor text in POST JSON, never query params or URL',()=>{
    const controller=new AbortController(); const rawQuery='synthetic-expression-marker';
    queryLogs(42,{rawQuery,time:{mode:'relative',duration:'15m'},limit:500,resultMode:'table',historyMode:'coalesced',historyId:91},controller.signal);
    const config=client.request.mock.calls[0][0];
    expect(config).toMatchObject({method:'POST',url:'/business/observability/datasources/42/logs',timeout:30000,skipErrorMessage:true,signal:controller.signal,data:{rawQuery,historyMode:'coalesced',historyId:91}});
    expect(config.params).toBeUndefined(); expect(config.url).not.toContain(rawQuery);
  });
  it('uses recipe-only Tempo and exact public metric/restore routes',()=>{
    searchTraces(7,{recipe:initialQueryRecipe('tempo')}); queryMetrics(8,{recipe:initialQueryRecipe('victoriametrics')}); restoreQueryHistory(11);
    expect(client.request.mock.calls.map(([config])=>[config.method,config.url])).toEqual([['POST','/business/observability/datasources/7/traces'],['POST','/business/observability/datasources/8/query'],['POST','/business/observability/query-history/11/restore']]);
    expect(client.request.mock.calls[0][0].data.rawQuery).toBeUndefined();
  });
  it('uses expected revision for immutable-source view replacement and encodes trace IDs',()=>{
    const recipe=initialQueryRecipe('loki'); saveQueryView(8,recipe,undefined,{id:3,revision:4}); getTrace(8,'trace/id');
    expect(client.request.mock.calls[0][0]).toMatchObject({method:'PUT',url:'/business/observability/saved-views/3',data:{datasourceId:8,expectedRevision:4,recipe}});
    expect(client.request.mock.calls[1][0].url).toBe('/business/observability/datasources/8/traces/trace%2Fid');
  });
});
