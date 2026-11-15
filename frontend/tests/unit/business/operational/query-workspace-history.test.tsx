import type { ReactElement, ReactNode } from 'react';
import { Children } from 'react';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import QueryWorkspaceHistory from '../../../../src/modules/business/observability/QueryWorkspaceHistory';
import { initialQueryRecipe } from '../../../../src/modules/business/observability/query-workspace-state';
import type { DatasourceRow, QueryHistoryRow, SavedViewRow } from '../../../../src/modules/business/observability/api';
const remote=vi.hoisted(()=>({catalog:vi.fn(),history:vi.fn(),views:vi.fn(),restoreHistory:vi.fn(),restoreView:vi.fn(),save:vi.fn(),deleteHistory:vi.fn(),deleteView:vi.fn(),hasPerm:vi.fn<(permission:string)=>boolean>(()=>true),identities:new Set<()=>void>()}));
vi.mock('react-i18next',()=>({useTranslation:()=>({t:(key:string)=>key})}));
vi.mock('@duanxldragon/pantheon-base-ui',()=>({usePermission:()=>({hasPerm:remote.hasPerm}),isForbiddenRequestError:(error:{status?:number})=>error?.status===403,useAuthStore:{subscribe:(listener:()=>void)=>{remote.identities.add(listener);return ()=>remote.identities.delete(listener);}}}));
vi.mock('../../../../src/modules/business/observability/api',()=>({getQueryCatalog:remote.catalog,getQueryHistory:remote.history,getSavedViews:remote.views,restoreQueryHistory:remote.restoreHistory,restoreSavedView:remote.restoreView,saveQueryView:remote.save,deleteQueryHistory:remote.deleteHistory,deleteSavedView:remote.deleteView}));
vi.mock('@arco-design/web-react',()=>{
  const Box=({children}:{children?:ReactNode})=><div>{children}</div>;
  const Select=Object.assign(({children,value,onChange,...props}:{children?:ReactNode;value:string;onChange:(value:string)=>void;'aria-label'?:string;disabled?:boolean})=><select aria-label={props['aria-label']} disabled={props.disabled} value={value} onChange={(event)=>onChange(event.target.value)}>{children}</select>,{Option:({children,value,disabled}:{children?:ReactNode;value:string;disabled?:boolean})=><option value={value} disabled={disabled}>{children}</option>});
  const Tabs=Object.assign(({children,onChange}:{children:ReactNode;onChange:(tab:string)=>void})=><div>{Children.map(children,(child)=>child&&<button onClick={()=>onChange(String((child as ReactElement).key))}>{(child as ReactElement<{title:string}>).props.title}</button>)}</div>,{TabPane:()=>null});
  return {Space:Box,Tag:Box,Select,Tabs,Alert:({content}:{content:ReactNode})=><div role="status">{content}</div>,Button:({children,disabled,onClick}:{children:ReactNode;disabled?:boolean;onClick?:()=>void})=><button disabled={disabled} onClick={onClick}>{children}</button>,Popconfirm:({children,onOk}:{children:ReactNode;onOk:()=>void})=><span onClick={onOk}>{children}</span>,Pagination:({onChange}:{onChange:(page:number)=>void})=><button onClick={()=>onChange(2)}>page2</button>};
});
vi.mock('../../../../src/components',()=>({AppDrawer:({visible,children,onCancel}:{visible:boolean;children:ReactNode;onCancel:()=>void})=>visible?<aside><button onClick={onCancel}>close</button>{children}</aside>:null,PageEmpty:({description}:{description:string})=><p>{description}</p>,PageError:()=> <p>request-error</p>,PageLoading:()=> <p>loading</p>}));
const key=(suffix:string)=>'business.observability.workspace.'+suffix;
const recipe=initialQueryRecipe('loki');
const source:DatasourceRow={id:1,type:'loki',name:'Loki',url:'',status:'healthy',isDefault:true,createdAt:'2026-11-11T00:00:00Z'};
const history:QueryHistoryRow={id:7,datasourceId:1,sourceKind:'loki',time:recipe.time,limit:500,resultMode:'table',outcome:'success',durationMs:1,createdAt:'2026-11-11T00:00:00Z',recipe,expressionUnavailable:false};
const view:SavedViewRow={id:8,datasourceId:1,revision:3,recipe,updatedAt:'2026-11-11T00:00:00Z'};
const page=(items:unknown[])=>({items,total:items.length,catalogVersion:1});
function deferred<T>(){let resolve!:(value:T)=>void;const promise=new Promise<T>((yes)=>{resolve=yes;});return {promise,resolve};}
function mount(){const before=vi.fn(),restore=vi.fn(),denied=vi.fn();const props={sourceId:1,kind:'loki' as const,sources:[source],recipe,onRecipeChange:vi.fn(),onBeforeRestore:before,onRestore:restore,onDenied:denied,epoch:0};const rendered=render(<QueryWorkspaceHistory {...props}/>);return {...rendered,before,restore,denied,props};}
async function open(){fireEvent.click(screen.getByRole('button',{name:key('historyViews')}));await screen.findByText(key('queryLabel'));}
beforeEach(()=>{
  remote.catalog.mockResolvedValue({version:1,sourceKind:'loki',fields:[{field:'stream',ops:['exists']},{field:'severity',ops:['equals'],values:['info','warning','error']}],durations:['5m','15m','1h','6h','24h'],maxLimit:5000,resultModes:['table']});
  remote.history.mockResolvedValue(page([history]));remote.views.mockResolvedValue(page([view]));remote.restoreHistory.mockResolvedValue(page([history]));remote.restoreView.mockResolvedValue(page([view]));remote.hasPerm.mockReturnValue(true);
});
afterEach(()=>{cleanup();vi.resetAllMocks();});
describe('private query workspace restore and authority',()=>{
  it('clears parent work before restore and returns only whitelisted conditions',async()=>{
    const fixture=mount();await open();fireEvent.click(screen.getByRole('button',{name:key('restore')}));
    expect(fixture.before).toHaveBeenCalledOnce();expect(remote.restoreHistory).toHaveBeenCalledWith(7,expect.any(AbortSignal));
    await waitFor(()=>expect(fixture.restore).toHaveBeenCalledWith({datasourceId:1,recipe,time:recipe.time,limit:500,expressionUnavailable:false}));
    expect(fixture.before.mock.invocationCallOrder[0]).toBeLessThan(remote.restoreHistory.mock.invocationCallOrder[0]);expect(screen.queryByRole('complementary')).toBeNull();
  });
  it('restores raw history as metadata with no recipe or editor text',async()=>{
    remote.restoreHistory.mockResolvedValue(page([{...history,recipe:undefined,expressionUnavailable:true,rawQuery:'synthetic-private'}]));const fixture=mount();await open();fireEvent.click(screen.getByRole('button',{name:key('restore')}));
    await waitFor(()=>expect(fixture.restore).toHaveBeenCalledWith({datasourceId:1,recipe:null,time:recipe.time,limit:500,expressionUnavailable:true}));expect(screen.queryByText(/synthetic-private/)).toBeNull();
  });
  it.each(['wrong-owner-source','wrong-id','unknown-recipe','invalid-time','cross-kind-metadata'])('refuses hostile restore DTO: %s',async(kind)=>{
    const hostile={...history,...(kind==='wrong-owner-source'?{datasourceId:999}:kind==='wrong-id'?{id:12}:kind==='unknown-recipe'?{recipe:{...recipe,rules:[{field:'sql',op:'raw'}]}}:kind==='cross-kind-metadata'?{recipe:undefined,expressionUnavailable:true,sourceKind:'tempo'}:{recipe:undefined,expressionUnavailable:true,time:{mode:'relative',duration:'365d'}})};
    remote.restoreHistory.mockResolvedValue(page([hostile]));const fixture=mount();await open();fireEvent.click(screen.getByRole('button',{name:key('restore')}));await screen.findByText('request-error');expect(fixture.restore).not.toHaveBeenCalled();
  });
  it('aborts history, restore and catalog on identity changes and discards late replies',async()=>{
    const pending=deferred<ReturnType<typeof page>>();remote.restoreHistory.mockReturnValue(pending.promise);const fixture=mount();await open();fireEvent.click(screen.getByRole('button',{name:key('restore')}));const signal=remote.restoreHistory.mock.calls[0][1] as AbortSignal;
    act(()=>remote.identities.forEach((listener)=>listener()));expect(signal.aborted).toBe(true);
    await act(async()=>pending.resolve(page([history])));expect(fixture.restore).not.toHaveBeenCalled();expect(screen.queryByRole('complementary')).toBeNull();
  });
  it.each(['source','kind','removed-source','recipe'])('rejects an ignored-abort restore after %s changes',async(change)=>{
    const pending=deferred<ReturnType<typeof page>>();remote.restoreHistory.mockReturnValue(pending.promise);
    const fixture=mount();await open();fireEvent.click(screen.getByRole('button',{name:key('restore')}));
    const signal=remote.restoreHistory.mock.calls[0][1] as AbortSignal;
    const next={...fixture.props,...(change==='source'?{sourceId:2,sources:[source,{...source,id:2}]}:change==='kind'?{kind:'tempo' as const}:change==='removed-source'?{sources:[]}:{recipe:{...recipe,limit:250}})};
    fixture.rerender(<QueryWorkspaceHistory {...next}/>);
    expect(signal.aborted).toBe(true);
    await act(async()=>pending.resolve(page([history])));
    expect(fixture.restore).not.toHaveBeenCalled();expect(fixture.before).toHaveBeenCalledOnce();
  });
  it('cancels an ignored-abort restore when raw/time/limit selection changes',async()=>{
    const pending=deferred<ReturnType<typeof page>>();remote.restoreHistory.mockReturnValue(pending.promise);const fixture=mount();
    fixture.rerender(<QueryWorkspaceHistory {...fixture.props} selectionKey="raw-before"/>);await open();fireEvent.click(screen.getByRole('button',{name:key('restore')}));
    const signal=remote.restoreHistory.mock.calls[0][1] as AbortSignal;fixture.rerender(<QueryWorkspaceHistory {...fixture.props} selectionKey="raw-after"/>);
    expect(signal.aborted).toBe(true);await act(async()=>pending.resolve(page([history])));expect(fixture.restore).not.toHaveBeenCalled();
  });
  it('aborts catalogue discovery when the selected source is removed and rejects its late reply',async()=>{
    const catalog={version:1,sourceKind:'loki',fields:[{field:'stream',ops:['exists']},{field:'severity',ops:['equals'],values:['info','warning','error']}],durations:['5m','15m','1h','6h','24h'],maxLimit:5000,resultModes:['table']};
    const pending=deferred<typeof catalog>();remote.catalog.mockReturnValueOnce(pending.promise);const fixture=mount();await waitFor(()=>expect(remote.catalog).toHaveBeenCalledOnce());
    const signal=remote.catalog.mock.calls[0][1] as AbortSignal;fixture.rerender(<QueryWorkspaceHistory {...fixture.props} sources={[]}/>);expect(signal.aborted).toBe(true);
    await act(async()=>pending.resolve(catalog));expect((screen.getByRole('combobox',{name:key('editorMode')}) as HTMLSelectElement).disabled).toBe(true);expect(fixture.restore).not.toHaveBeenCalled();
  });
  it('does not cache lists after closing and refuses malformed list metadata',async()=>{
    const fixture=mount();await open();fireEvent.click(screen.getByRole('button',{name:'close'}));remote.history.mockResolvedValue(page([{...history,id:{private:'marker'}}]));fireEvent.click(screen.getByRole('button',{name:key('historyViews')}));await screen.findByText('request-error');expect(remote.history).toHaveBeenCalledTimes(2);expect(screen.queryByText(/marker/)).toBeNull();expect(fixture.restore).not.toHaveBeenCalled();
  });
  it('clears workspace on live 403 without displaying the response message',async()=>{
    remote.history.mockRejectedValue({status:403,message:'private-token'});const fixture=mount();fireEvent.click(screen.getByRole('button',{name:key('historyViews')}));await screen.findByText(key('forbidden'));expect(fixture.denied).toHaveBeenCalledOnce();expect(screen.queryByText(/private-token/)).toBeNull();
  });
  it('passes expectedRevision when replacing a view and exposes a safe 409 conflict',async()=>{
    remote.save.mockRejectedValue({status:409,message:'private-query'});mount();await open();fireEvent.click(screen.getByRole('button',{name:key('views')}));await screen.findByText(key('viewLabel'));fireEvent.click(screen.getByRole('button',{name:key('replace')}));await screen.findByText(key('conflict'));expect(remote.save).toHaveBeenCalledWith(1,recipe,expect.any(AbortSignal),expect.objectContaining({id:8,revision:3}));expect(screen.queryByText(/private-query/)).toBeNull();
  });
  it('applies exact permission keys and never sends a denied restore',async()=>{
    remote.hasPerm.mockImplementation((permission:string)=>!permission.endsWith(':restore'));mount();await open();expect(screen.queryByRole('button',{name:key('restore')})).toBeNull();expect(remote.hasPerm).toHaveBeenCalledWith('business:observability:query-history:restore');expect(remote.restoreHistory).not.toHaveBeenCalled();
  });
});
