import { StrictMode } from 'react';
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { DashboardOperationalWidget, DashboardOperationalSnapshot } from '../../../src/core/router/types';
vi.mock('react-i18next',()=>({useTranslation:()=>({t:(key:string)=>key})}));
const navigate=vi.fn();
vi.mock('react-router-dom',()=>({useNavigate:()=>navigate}));
vi.mock('../../../src/components',()=>({DateTimeMeta:({value}:{value:string})=><span>{value}</span>,PageLoading:()=> <span>loading</span>,PageEmpty:()=> <span>empty</span>,PageRequestError:({onRetry}:{onRetry:()=>void})=><button onClick={onRetry}>retry</button>}));
vi.mock('../../../src/api/request',()=>({isForbiddenRequestError:(e:unknown)=>(e as {kind?:string})?.kind==='forbidden'}));
import OperationalDashboardWidgets from '../../../src/modules/platform/OperationalDashboardWidgets';
function widget(load: NonNullable<DashboardOperationalWidget['dataSource']>['load'],key='test'):DashboardOperationalWidget {
 return {key,slot:'attention-queue',sourceDomain:'business/test',titleKey:key,descriptionKey:'description',path:'/business/test',permission:'test:view',registrationOwner:'test',cleanupPolicy:'remove_with_source_module',dataSource:{load,timeoutMs:1000},metadata:{owner:'test',cleanupPolicy:'remove_with_source_module',freshness:{policy:'snapshot',maxAgeMs:60000,refreshIntervalMs:30000},queryBudget:{maxRequestsPerMinute:1,maxItems:3,maxRenderItems:2},errorIsolation:'widget',emptyStateKey:'empty',errorStateKey:'error'}};
}
const snapshot=():DashboardOperationalSnapshot=>({updatedAt:new Date().toISOString(),items:[{id:'a',label:'host-a',value:2,query:{businessScopeId:'7',status:'failed'}}]});
afterEach(()=>{cleanup();vi.useRealTimers();navigate.mockClear();});
describe('operational dashboard isolation and budgets',()=>{
 it('renders bounded provider data and preserves context on drilldown',async()=>{
  const load=vi.fn(async()=>snapshot());render(<OperationalDashboardWidgets widgets={[widget(load)]}/>);
  fireEvent.click(await screen.findByText('host-a'));expect(navigate).toHaveBeenCalledWith('/business/test?businessScopeId=7&status=failed');
  expect(load.mock.calls[0][0].maxItems).toBe(3);
 });
 it('ignores late responses after unmount and aborts the provider',async()=>{
  let resolve!:(value:DashboardOperationalSnapshot)=>void;let signal!:AbortSignal;
  const load=vi.fn((req:{signal:AbortSignal;maxItems:number})=>{signal=req.signal;return new Promise<DashboardOperationalSnapshot>(r=>{resolve=r;});});
  const {unmount}=render(<OperationalDashboardWidgets widgets={[widget(load)]}/>);await act(async()=>{await Promise.resolve();});unmount();expect(signal.aborted).toBe(true);
  await act(async()=>resolve(snapshot()));expect(screen.queryByText('host-a')).toBeNull();
 });
 it('denial clears the widget and stops polling',async()=>{
  vi.useFakeTimers();const load=vi.fn(async()=>{throw {kind:'forbidden'};});render(<OperationalDashboardWidgets widgets={[widget(load)]}/>);
  await act(async()=>{await Promise.resolve();});expect(screen.queryByText('test')).toBeNull();
  await act(async()=>{vi.advanceTimersByTime(120000);});expect(load).toHaveBeenCalledTimes(1);
 });
 it('enforces a deadline even if the provider ignores abort and bounds retry rate',async()=>{
  vi.useFakeTimers();const load=vi.fn(()=>new Promise<DashboardOperationalSnapshot>(()=>{}));render(<OperationalDashboardWidgets widgets={[widget(load)]}/>);
  await act(async()=>{await Promise.resolve();});await act(async()=>{vi.advanceTimersByTime(1001);});expect(screen.getByText('retry')).toBeTruthy();
  fireEvent.click(screen.getByText('retry'));expect(screen.getByText('loading')).toBeTruthy();expect(screen.queryByText('retry')).toBeNull();await act(async()=>{vi.advanceTimersByTime(30000);});expect(load).toHaveBeenCalledTimes(1);
  await act(async()=>{vi.advanceTimersByTime(30000);});expect(load).toHaveBeenCalledTimes(2);
 });
 it('shows queued retry feedback with retained data without bypassing the budget',async()=>{
  vi.useFakeTimers();const load=vi.fn().mockResolvedValueOnce(snapshot()).mockRejectedValueOnce(new Error('provider unavailable')).mockImplementation(async()=>snapshot());
  render(<OperationalDashboardWidgets widgets={[widget(load)]}/>);
  await act(async()=>{await Promise.resolve();});expect(screen.getByText('host-a')).toBeTruthy();
  await act(async()=>{vi.advanceTimersByTime(60000);});expect(screen.getByText('retry')).toBeTruthy();expect(load).toHaveBeenCalledTimes(2);
  fireEvent.click(screen.getByText('retry'));expect(screen.getByText('host-a')).toBeTruthy();expect(screen.getByText('dashboard.operational.refreshing')).toBeTruthy();expect(screen.queryByText('retry')).toBeNull();
  await act(async()=>{vi.advanceTimersByTime(30000);});expect(load).toHaveBeenCalledTimes(2);
  await act(async()=>{vi.advanceTimersByTime(30000);});expect(load).toHaveBeenCalledTimes(3);expect(screen.queryByText('dashboard.operational.refreshing')).toBeNull();
 });
 it('invalid/over-budget data fails only its widget',async()=>{
  const bad=widget(async()=>({...snapshot(),items:Array.from({length:4},(_,i)=>({id:String(i),label:'excess'}))}),'bad');
  render(<OperationalDashboardWidgets widgets={[bad,widget(async()=>snapshot(),'good')]}/>);
  expect(await screen.findByText('host-a')).toBeTruthy();expect(await screen.findByText('retry')).toBeTruthy();expect(screen.queryByText('excess')).toBeNull();
 });
 it('StrictMode setup replay does not spend the initial request budget',async()=>{
  const load=vi.fn(async()=>snapshot());render(<StrictMode><OperationalDashboardWidgets widgets={[widget(load)]}/></StrictMode>);
  expect(await screen.findByText('host-a')).toBeTruthy();expect(load).toHaveBeenCalledTimes(1);
 });
 it('does not request semantic-only registrations' ,()=>{
  const semantic=widget(async()=>snapshot());delete semantic.dataSource;const {container}=render(<OperationalDashboardWidgets widgets={[semantic]}/>);expect(container.childElementCount).toBe(0);
 });
});
