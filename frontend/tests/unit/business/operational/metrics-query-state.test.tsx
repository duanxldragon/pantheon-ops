import { Children, useState, type ReactElement, type ReactNode } from 'react';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import MonitoringCenter from '../../../../src/modules/business/observability/MonitoringCenter';
const remote=vi.hoisted(()=>({query:vi.fn(),sources:vi.fn(),targets:vi.fn(),scopes:vi.fn(),hasPerm:vi.fn<(permission:string)=>boolean>(()=>true),listeners:new Set<()=>void>(),form:{validate:vi.fn(),resetFields:vi.fn()}}));
vi.mock('react-i18next',()=>({useTranslation:()=>({t:(key:string,options?:{count?:number})=>key+(options?.count===undefined?'':':'+options.count)})}));
vi.mock('@duanxldragon/pantheon-base-ui',()=>({usePermission:()=>({hasPerm:remote.hasPerm}),isForbiddenRequestError:(error:{status?:number})=>error?.status===403,useAuthStore:{subscribe:(listener:()=>void)=>{remote.listeners.add(listener);return ()=>remote.listeners.delete(listener);}}}));
vi.mock('../../../../src/modules/business/observability/api',()=>({queryMetrics:remote.query,getDatasourceList:remote.sources,getMonitoringTargetList:remote.targets,createDatasource:vi.fn(),createMonitoringTarget:vi.fn(),deleteDatasource:vi.fn(),deleteMonitoringTarget:vi.fn(),testDatasourceConnection:vi.fn()}));
vi.mock('../../../../src/modules/business/bizscope/api',()=>({getBizScopeOptions:remote.scopes}));
vi.mock('../../../../src/modules/business/observability/QueryWorkspaceHistory',()=>({default:()=>null}));
vi.mock('../../../../src/components/patterns/forms/PasswordInput',()=>({default:()=>null}));
vi.mock('@arco-design/web-react',()=>{
  const Box=({children}:{children?:ReactNode})=><div>{children}</div>;
  const Select=Object.assign(({children,value,onChange,...props}:{children:ReactNode;value:number;onChange:(value:number)=>void;'aria-label'?:string})=><select aria-label={props['aria-label']} value={value} onChange={(event)=>onChange(Number(event.target.value))}>{children}</select>,{Option:({children,value}:{children:ReactNode;value:number})=><option value={value}>{children}</option>});
  const Tabs=Object.assign(({children,defaultActiveTab}:{children:ReactNode;defaultActiveTab:string})=>{const [active,setActive]=useState(defaultActiveTab);const panes=Children.toArray(children) as ReactElement<{title:string;children:ReactNode}>[];return <div>{panes.map((pane)=><button role="tab" key={pane.key} onClick={()=>setActive(String(pane.key).replace(/^\.\$/,''))}>{pane.props.title}</button>)}{panes.find((pane)=>String(pane.key).replace(/^\.\$/,'')===active)?.props.children}</div>;},{TabPane:Box});
  return {Space:Box,Card:Box,Tag:Box,Tabs,Form:Object.assign(Box,{Item:Box,useForm:()=>[remote.form]}),Select,Input:Object.assign(Box,{TextArea:Box}),InputNumber:()=>null,Switch:()=>null,Alert:({content}:{content:ReactNode})=><p>{content}</p>,Button:({children,disabled,onClick}:{children:ReactNode;disabled?:boolean;onClick?:()=>void})=><button disabled={disabled} onClick={onClick}>{children}</button>,Popconfirm:Box,Message:{success:vi.fn(),error:vi.fn()}};
});
vi.mock('../../../../src/components',()=>{
  const Box=({children}:{children?:ReactNode})=><div>{children}</div>;
  return {PageContainer:Box,GovernanceSummaryBar:()=>null,AppModal:({visible,children,onCancel}:{visible:boolean;children:ReactNode;onCancel:()=>void})=>visible?<aside><button onClick={onCancel}>close-form</button>{children}</aside>:null,PageLoading:()=> <p>loading</p>,PageEmpty:({description}:{description:string})=><p>{description}</p>,PageError:()=> <p>request-error</p>,AppTable:({data}:{data:{id:number;series?:string;value?:string;targetName?:string}[]})=><section>{data.map((row)=><p key={row.id}>{row.series}{row.value}{row.targetName}</p>)}</section>,SearchToolbar:({inlineFilters,extra}:{inlineFilters:ReactNode;extra:ReactNode})=><div>{inlineFilters}{extra}</div>};
});
const result={historyUnavailable:false,result:{status:'success',data:{resultType:'vector',result:[{metric:{job:'metrics-fixture'},value:[1794009600,'1']}]}}};
const run=async()=>{const button=screen.getByRole('button',{name:'business.observability.workspace.run'}) as HTMLButtonElement;await waitFor(()=>expect(button.disabled).toBe(false));fireEvent.click(button);};
const openQueries=async()=>fireEvent.click(await screen.findByRole('tab',{name:'business.observability.workspace.metricsTab'}));
beforeEach(()=>{remote.sources.mockResolvedValue({items:[{id:1,name:'Prometheus',type:'prometheus',isDefault:true},{id:2,name:'VictoriaMetrics',type:'victoriametrics'}]});remote.targets.mockResolvedValue({items:[]});remote.scopes.mockResolvedValue([]);remote.hasPerm.mockReturnValue(true);});
afterEach(()=>{cleanup();vi.resetAllMocks();});
describe('metrics query workspace lifecycle',()=>{
  it('never executes on source discovery and sends a reviewed recipe only after Run',async()=>{
    remote.query.mockResolvedValue({...result,historyUnavailable:true});render(<MonitoringCenter/>);await openQueries();await screen.findByText('Prometheus');expect(remote.query).not.toHaveBeenCalled();await run();await screen.findByText('job=metrics-fixture1');expect(remote.query.mock.calls[0][1]).toMatchObject({recipe:{sourceKind:'prometheus',rules:[{field:'availability',op:'instant'}]}});expect(screen.getByText('business.observability.workspace.historyUnavailable')).toBeTruthy();
  });
  it('aborts a pending query on source changes and refuses an ignored-abort reply',async()=>{
    let resolve!:(value:typeof result)=>void;remote.query.mockReturnValueOnce(new Promise((done)=>{resolve=done;}));render(<MonitoringCenter/>);await openQueries();await screen.findByText('Prometheus');await run();const signal=remote.query.mock.calls[0][2] as AbortSignal;
    fireEvent.change(screen.getByRole('combobox',{name:'business.observability.common.datasource'}),{target:{value:'2'}});expect(signal.aborted).toBe(true);await act(async()=>resolve(result));expect(screen.queryByText('job=metrics-fixture1')).toBeNull();
    remote.query.mockResolvedValue(result);await run();await screen.findByText('job=metrics-fixture1');expect(remote.query.mock.calls[1][1].recipe.sourceKind).toBe('victoriametrics');
  });
  it('aborts on identity change, rejects late data and displays safe live-denial state',async()=>{
    let resolve!:(value:typeof result)=>void;remote.query.mockReturnValueOnce(new Promise((done)=>{resolve=done;}));render(<MonitoringCenter/>);await openQueries();await screen.findByText('Prometheus');await run();const signal=remote.query.mock.calls[0][2] as AbortSignal;
    act(()=>remote.listeners.forEach((listener)=>listener()));expect(signal.aborted).toBe(true);await act(async()=>resolve(result));expect(screen.queryByText('job=metrics-fixture1')).toBeNull();await screen.findByText('Prometheus');await waitFor(()=>expect((screen.getByRole('button',{name:'business.observability.workspace.run'}) as HTMLButtonElement).disabled).toBe(false));
    remote.query.mockRejectedValue({status:403,message:'synthetic-private-response'});await run();await screen.findByText('business.observability.workspace.forbidden');expect(screen.queryByText(/synthetic-private-response/)).toBeNull();expect((screen.getByRole('button',{name:'business.observability.workspace.run'}) as HTMLButtonElement).disabled).toBe(true);
  });
  it('rejects malformed provider data without treating it as an empty successful query',async()=>{
    remote.query.mockResolvedValue({...result,result:{status:'error',data:{resultType:'vector',result:[]}}});render(<MonitoringCenter/>);await openQueries();await screen.findByText('Prometheus');await run();await screen.findByText('request-error');expect(screen.queryByText('business.observability.workspace.metricsEmpty')).toBeNull();
  });
  it.each([403,502])('keeps queries usable when a deferred targets request fails with %s',async(status)=>{
    let reject!:(error:unknown)=>void;remote.targets.mockReturnValue(new Promise((_resolve,fail)=>{reject=fail;}));remote.query.mockResolvedValue(result);
    remote.hasPerm.mockImplementation((permission)=>permission==='business:observability:metric_source:view');
    render(<MonitoringCenter/>);await openQueries();await screen.findByText('Prometheus');
    await act(async()=>reject({status,message:'sensitive-target-failure'}));await run();await screen.findByText('job=metrics-fixture1');
    expect(remote.scopes).not.toHaveBeenCalled();fireEvent.click(screen.getByRole('tab',{name:'business.observability.monitoring.targetsTab'}));
    await screen.findByText(status===403?'business.observability.workspace.forbidden':'request-error');expect(screen.queryByText(/sensitive-target-failure/)).toBeNull();
  });
  it.each([403,502])('isolates a deferred BizScope %s to the create-target form',async(status)=>{
    let reject!:(error:unknown)=>void;remote.scopes.mockReturnValue(new Promise((_resolve,fail)=>{reject=fail;}));remote.query.mockResolvedValue(result);
    render(<MonitoringCenter/>);fireEvent.click(await screen.findByRole('button',{name:'business.observability.monitoring.addTarget'}));
    await waitFor(()=>expect(remote.scopes).toHaveBeenCalledOnce());await act(async()=>reject({status,message:'sensitive-scope-failure'}));
    await screen.findByText(status===403?'business.observability.workspace.forbidden':'request-error');
    fireEvent.click(screen.getByRole('button',{name:'close-form'}));await openQueries();await screen.findByText('Prometheus');await run();await screen.findByText('job=metrics-fixture1');
    expect(screen.queryByText(/sensitive-scope-failure/)).toBeNull();
  });
  it('shows only a warning count for successful partial metrics and clears it on the next Run',async()=>{
    const marker='private-provider-warning';const logging=vi.spyOn(console,'log');const errors=vi.spyOn(console,'error');
    remote.query.mockResolvedValueOnce({...result,result:{...result.result,warnings:[marker,marker+'-2']}}).mockResolvedValueOnce(result);
    render(<MonitoringCenter/>);await openQueries();await screen.findByText('Prometheus');await run();await screen.findByText('job=metrics-fixture1');
    await screen.findByText('business.observability.workspace.metricsPartial:2');expect(document.body.textContent).not.toContain(marker);expect(JSON.stringify({...localStorage,...sessionStorage})).not.toContain(marker);expect(JSON.stringify(logging.mock.calls.concat(errors.mock.calls))).not.toContain(marker);
    await run();await screen.findByText('job=metrics-fixture1');expect(screen.queryByText('business.observability.workspace.metricsPartial:2')).toBeNull();logging.mockRestore();errors.mockRestore();
  });
  it('aborts source/target discovery on auth changes and rejects ignored-abort data',async()=>{
    let resolveSources!:(value:unknown)=>void,resolveTargets!:(value:unknown)=>void;
    remote.sources.mockReturnValueOnce(new Promise((resolve)=>{resolveSources=resolve;}));remote.targets.mockReturnValueOnce(new Promise((resolve)=>{resolveTargets=resolve;}));
    render(<MonitoringCenter/>);await waitFor(()=>expect(remote.sources).toHaveBeenCalledOnce());
    const sourceSignal=remote.sources.mock.calls[0][0] as AbortSignal;const targetSignal=remote.targets.mock.calls[0][0] as AbortSignal;
    act(()=>remote.listeners.forEach((listener)=>listener()));expect(sourceSignal.aborted).toBe(true);expect(targetSignal.aborted).toBe(true);
    await act(async()=>{resolveSources({items:[{id:99,name:'private-late-source',type:'prometheus'}]});resolveTargets({items:[{id:99,targetName:'private-late-target'}]});});
    expect(document.body.textContent).not.toContain('private-late');await openQueries();await screen.findByText('Prometheus');expect(screen.queryByText('private-late-source')).toBeNull();
  });
  it('aborts a pending BizScope form request on close and discards the ignored-abort reply',async()=>{
    let resolve!:(value:unknown)=>void;remote.scopes.mockReturnValueOnce(new Promise((done)=>{resolve=done;}));
    render(<MonitoringCenter/>);fireEvent.click(await screen.findByRole('button',{name:'business.observability.monitoring.addTarget'}));await waitFor(()=>expect(remote.scopes).toHaveBeenCalledOnce());
    const signal=remote.scopes.mock.calls[0][0] as AbortSignal;fireEvent.click(screen.getByRole('button',{name:'close-form'}));expect(signal.aborted).toBe(true);
    await act(async()=>resolve([{id:99,name:'private-late-scope'}]));fireEvent.click(screen.getByRole('button',{name:'business.observability.monitoring.addTarget'}));
    await waitFor(()=>expect(remote.scopes).toHaveBeenCalledTimes(2));expect(document.body.textContent).not.toContain('private-late-scope');
  });
  it.each([null,['warning',17]])('rejects malformed warning metadata %s',async(warnings)=>{
    remote.query.mockResolvedValue({...result,result:{...result.result,warnings}});render(<MonitoringCenter/>);await openQueries();await screen.findByText('Prometheus');await run();await screen.findByText('request-error');expect(screen.queryByText('job=metrics-fixture1')).toBeNull();
  });
});
