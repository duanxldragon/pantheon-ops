import { act, cleanup, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { copyQueryRecipe, copyWorkspaceRow, initialQueryRecipe, supportedQueryCatalog, useQueryWorkspaceIdentity, validQueryTime } from '../../../../src/modules/business/observability/queryWorkspaceState';
import type { QueryCatalog, QueryHistoryRow, QueryRecipe } from '../../../../src/modules/business/observability/api';
const auth = vi.hoisted(() => ({ listeners: new Set<() => void>() }));
vi.mock('@duanxldragon/pantheon-base-ui', () => ({ useAuthStore: { subscribe: (listener: () => void) => { auth.listeners.add(listener); return () => auth.listeners.delete(listener); } } }));
afterEach(cleanup);
describe('query workspace DTO and lifetime boundary', () => {
  it.each(['5m', '15m', '1h', '6h', '24h'])('allows only reviewed relative duration %s', (duration) => {
    expect(validQueryTime({mode:'relative',duration} as QueryRecipe['time'])).toBe(true);
  });
  it.each([
    {mode:'relative',duration:'365d'}, {mode:'relative',duration:'15m',actorId:1},
    {mode:'absolute',start:'2026-11-10T00:00:00',end:'2026-11-10T00:15:00Z'},
    {mode:'absolute',start:'2026-11-10T00:00:00Z',end:'2026-11-12T00:00:00Z'},
  ])('rejects unknown time conditions %#', (time) => expect(validQueryTime(time as QueryRecipe['time'])).toBe(false));
  it('requires the Loki stream and rejects duplicate, unknown and cross-kind rules', () => {
    const base = initialQueryRecipe('loki');
    expect(copyQueryRecipe({...base,rules:[{field:'severity',op:'equals',value:'error'}]},'loki')).toBeNull();
    expect(copyQueryRecipe({...base,rules:[...base.rules,...base.rules]},'loki')).toBeNull();
    expect(copyQueryRecipe({...base,rules:[{field:'stream',op:'exists'},{field:'url',op:'equals',value:'secret'}]},'loki')).toBeNull();
    expect(copyQueryRecipe(base,'tempo')).toBeNull();
    expect(copyQueryRecipe({...base,actorId:42} as QueryRecipe,'loki')).toBeNull();
  });
  it('does not accept compiler expansions, alternate operations or wider limits', () => {
    const catalog: QueryCatalog = {version:1,sourceKind:'tempo',fields:[{field:'service',ops:['exists']}],durations:['5m','15m','1h','6h','24h'],maxLimit:1000,resultModes:['table']};
    expect(supportedQueryCatalog(catalog,'tempo')).toBe(true);
    expect(supportedQueryCatalog({...catalog,maxLimit:10000},'tempo')).toBe(false);
    expect(supportedQueryCatalog({...catalog,fields:[{field:'service',ops:['exists','raw']}]},'tempo')).toBe(false);
    expect(supportedQueryCatalog({...catalog,sourceKind:'loki'},'tempo')).toBe(false);
  });
  it('copies nested recipes and never retains an extra provider payload in history metadata', () => {
    const recipe=initialQueryRecipe('loki');
    const original={id:1,datasourceId:1,sourceKind:'loki',time:recipe.time,limit:recipe.limit,resultMode:'table',outcome:'success',durationMs:3,createdAt:'2026-11-11T00:00:00Z',recipe,expressionUnavailable:false,providerResult:'private-result'} as QueryHistoryRow;
    const copied=copyWorkspaceRow(original,'history')!;
    expect(copied).not.toHaveProperty('providerResult'); expect(copied.recipe).not.toBe(recipe);
    expect(copied.recipe?.rules).not.toBe(recipe.rules);
    expect(copyWorkspaceRow({...original,id:0},'history')).toBeNull();
    expect(copyWorkspaceRow({...original,expressionUnavailable:true},'history')).toBeNull();
  });
  it('invalidates synchronously on repeated auth updates and pagehide before a late response can apply', () => {
    const invalidate=vi.fn(); const {result,unmount}=renderHook(()=>useQueryWorkspaceIdentity(invalidate));
    const first=result.current.snapshot();
    act(()=>auth.listeners.forEach((listener)=>listener()));
    expect(result.current.isCurrent(first)).toBe(false); expect(invalidate).toHaveBeenCalledOnce();
    const second=result.current.snapshot(); act(()=>auth.listeners.forEach((listener)=>listener()));
    expect(result.current.isCurrent(second)).toBe(false);
    const third=result.current.snapshot(); act(()=>window.dispatchEvent(new Event('pagehide')));
    expect(result.current.isCurrent(third)).toBe(false);
    const last=result.current.snapshot(); unmount(); expect(result.current.isCurrent(last)).toBe(false); expect(auth.listeners.size).toBe(0);
  });
});
