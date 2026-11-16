import { fireEvent, render, screen, cleanup } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, useLocation } from 'react-router-dom';
import DeployExecutionWorkbench from '../../../../src/modules/business/deploy/task/DeployExecutionWorkbench';
import type { DeployTaskRow, DeployTaskLogRow } from '../../../../src/modules/business/deploy/api';

const fixture = vi.hoisted(() => ({ rows: [] as DeployTaskLogRow[], state: 'ready', truncated: false, reload: vi.fn() }));
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('../../../../src/modules/business/deploy/task/useDeployTaskLogs', () => ({ useDeployTaskLogs: () => fixture }));
vi.mock('@arco-design/web-react', () => ({
  Button: ({ children, ...props }: React.ComponentProps<'button'>) => <button {...props}>{children}</button>,
  Space: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Tag: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
  Grid: Object.assign(() => null, { Row: ({ children }: { children: React.ReactNode }) => <div>{children}</div>, Col: ({ children }: { children: React.ReactNode }) => <div>{children}</div> }),
  Form: Object.assign(({ children }: { children: React.ReactNode }) => <div>{children}</div>, { Item: ({ children }: { children: React.ReactNode }) => <div>{children}</div>, useForm: () => [{ setFieldsValue: vi.fn(), validate: async () => ({}) }, {}] }),
  Select: ({ options, value, onChange, 'aria-label': label }: { options: {value: string|number; label: string}[]; value?: string|number; onChange: (value: string|number|undefined) => void; 'aria-label': string }) => <select aria-label={label} value={value ?? ''} onChange={(event) => onChange(event.target.value || undefined)}><option value="" />{options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select>,
}));
vi.mock('../../../../src/components', () => ({ AppModal: ({ children }: { children: React.ReactNode }) => <div>{children}</div> }));
vi.mock('../../../../src/components/operational', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../../../src/components/operational')>()),
  ExecutionStepRail: ({ steps, onStepClick }: { steps: {id: string; title: string; status: string}[]; onStepClick: (step: {id: string}) => void }) => <div>{steps.map((step) => <button key={step.id} data-status={step.status} onClick={() => onStepClick(step)}>{step.title}</button>)}</div>,
  TaskLogViewer: ({ chunks, onPausedChange, paused }: { chunks: {sequence: number; content: string}[]; onPausedChange: (paused: boolean) => void; paused: boolean }) => <div><button onClick={() => onPausedChange(!paused)}>{paused ? 'resume-tail' : 'pause-tail'}</button>{chunks.map((chunk) => <p key={chunk.sequence}>{chunk.content}</p>)}</div>,
}));
function Location() { return <output data-testid="location">{useLocation().search}</output>; }
const query = () => new URLSearchParams(screen.getByTestId('location').textContent ?? '');
function buildTask(): DeployTaskRow {
  return { id: 7, name: 'deployment', status: 'failed', hosts: [
    {id: 11, hostname: 'alpha', hostIp: '10.0.0.1', status: 'failed'},
    {id: 12, hostname: 'beta', hostIp: '10.0.0.2', status: 'success'},
  ], steps: [
    {id: 101, taskHostId: 11, stepKey: 'old-install', stepOrder: 1, status: 'success', attemptNo: 1, stateVersion: 1},
    {id: 201, taskHostId: 11, stepKey: 'latest-check', stepOrder: 1, status: 'success', attemptNo: 2, stateVersion: 1},
    {id: 202, taskHostId: 11, stepKey: 'latest-install', stepOrder: 2, status: 'failed', attemptNo: 2, stateVersion: 1},
    {id: 301, taskHostId: 12, stepKey: 'beta-install', stepOrder: 1, status: 'success', attemptNo: 1, stateVersion: 1},
  ] } as DeployTaskRow;
}
function log(sequence: number, taskStepId: number, content: string, stream='stdout', taskHostId=11): DeployTaskLogRow {
  return {sequence, taskHostId, taskStepId, attemptNo: taskStepId === 0 ? 0 : [101,301].includes(taskStepId) ? 1 : 2, content, stream, createdAt: '2026-11-08T00:00:00Z'};
}
function setup(path = '/task?host=11', task = buildTask()) {
  const view = (current: DeployTaskRow) => <MemoryRouter initialEntries={[path]}><DeployExecutionWorkbench task={current} stale={false} renderHost={() => null} /><Location /></MemoryRouter>;
  return { ...render(view(task)), view };
}
afterEach(() => { cleanup(); fixture.rows=[]; fixture.state='ready'; });

describe('persisted host/attempt/step/stream context', () => {
  it('shows the latest persisted attempt by default without old or unattributed logs', () => {
    fixture.rows=[log(1,101,'old error'),log(2,202,'latest error'),log(3,0,'unattributed')];
    setup();
    expect(screen.queryByText('old error')).toBeNull();
    expect(screen.queryByText('unattributed')).toBeNull();
    expect(screen.getByText('latest error')).toBeTruthy();
    expect(screen.queryByRole('button',{name:'old-install'})).toBeNull();
  });
  it('keeps an explicit old attempt selected when a newer attempt arrives', () => {
    fixture.rows=[log(1,101,'old error'),log(2,202,'new error')];
    const task=buildTask(); const {rerender,view}=setup('/task?host=11&attempt=1',task);
    expect(screen.getByText('old error')).toBeTruthy();
    task.steps?.push({id:401,taskHostId:11,stepKey:'future-install',stepOrder:1,status:'running',attemptNo:3,stateVersion:1});
    rerender(view({...task}));
    expect(screen.getByText('old error')).toBeTruthy();
    expect(screen.queryByText('new error')).toBeNull();
    expect(query().get('attempt')).toBe('1');
  });
  it('infers the attempt from a valid step deep link when attempt is absent', () => {
    fixture.rows=[log(1,101,'old error'),log(2,202,'new error')];
    setup('/task?host=11&step=101');
    expect(screen.getByText('old error')).toBeTruthy();
    expect(screen.queryByText('new error')).toBeNull();
    expect(screen.queryByRole('button',{name:'latest-install'})).toBeNull();
  });
  it('cannot use a step from another attempt to override an explicit attempt', () => {
    fixture.rows=[log(1,101,'old error'),log(2,202,'new error')];
    setup('/task?host=11&attempt=2&step=101');
    expect(screen.queryByText('old error')).toBeNull();
    expect(screen.getByText('new error')).toBeTruthy();
  });
  it('supports explicit all-attempt review, including unattributed host logs', () => {
    fixture.rows=[log(1,101,'old error'),log(2,202,'new error'),log(3,0,'unattributed')];
    setup('/task?host=11&attempt=all');
    for(const text of ['old error','new error','unattributed']) expect(screen.getByText(text)).toBeTruthy();
  });
  it('resets dependent selections on host change while preserving unrelated URL state', () => {
    fixture.rows=[log(1,101,'old error'),log(2,301,'beta result','stdout',12)];
    setup('/task?host=11&attempt=1&step=101&stream=stderr&tab=audit');
    fireEvent.change(screen.getByLabelText('business.deploy.logs.host'),{target:{value:'12'}});
    expect(query().get('host')).toBe('12');
    for(const name of ['attempt','step','stream']) expect(query().has(name)).toBe(false);
    expect(query().get('tab')).toBe('audit');
    expect(screen.queryByText('old error')).toBeNull();
    expect(screen.getByText('beta result')).toBeTruthy();
  });
  it('persists stream selection in the URL and preserves the selected attempt', () => {
    fixture.rows=[log(1,202,'normal output'),log(2,202,'error output','stderr')];
    setup('/task?host=11&attempt=2&stream=stderr');
    expect(screen.queryByText('normal output')).toBeNull();
    expect(screen.getByText('error output')).toBeTruthy();
    fireEvent.change(screen.getByLabelText('business.deploy.logs.stream'),{target:{value:'stdout'}});
    expect(query().get('stream')).toBe('stdout');
    expect(query().get('attempt')).toBe('2');
    expect(screen.getByText('normal output')).toBeTruthy();
    expect(screen.queryByText('error output')).toBeNull();
  });
  it('jumps to the first failed step in the latest attempt and its available error stream', () => {
    fixture.rows=[log(1,101,'old error','stderr'),log(2,202,'latest error','stderr')];
    setup('/task?host=12');
    fireEvent.click(screen.getByRole('button',{name:'business.deploy.logs.jumpFailure'}));
    expect(query().get('host')).toBe('11');
    expect(query().get('attempt')).toBe('2');
    expect(query().get('step')).toBe('202');
    expect(query().get('stream')).toBe('stderr');
    expect(query().get('firstError')).toBe('1');
    expect(screen.getByText('latest error')).toBeTruthy();
    expect(screen.queryByText('old error')).toBeNull();
  });
  it('requests the first error even when the failed host is outside the current log window', () => {
    fixture.rows=[]; setup('/task?host=12');
    fireEvent.click(screen.getByRole('button',{name:'business.deploy.logs.jumpFailure'}));
    expect(query().get('host')).toBe('11'); expect(query().get('step')).toBe('202');
    expect(query().get('stream')).toBe('stderr'); expect(query().get('firstError')).toBe('1');
    fireEvent.click(screen.getByRole('button',{name:'business.deploy.logs.showStepLogs'}));
    expect(query().has('firstError')).toBe(false); expect(query().get('step')).toBe('202');
  });
  it('uses recorded log attempts after a shared step checkpoint moves to a later attempt', () => {
    const task=buildTask(); task.steps=task.steps?.filter((step) => step.id!==101);
    task.logAttempts=[{taskHostId:11,attemptNo:1},{taskHostId:11,attemptNo:2}];
    fixture.rows=[{...log(1,202,'historical same-step error'),attemptNo:1},log(2,202,'current same-step error')];
    setup('/task?host=11&attempt=1',task);
    expect(screen.getByText('historical same-step error')).toBeTruthy();
    expect(screen.queryByText('current same-step error')).toBeNull();
    expect(screen.getByText('business.deploy.logs.historicalAttempt')).toBeTruthy();
  });
  it('does not resurrect a paused snapshot after denial and reauthorization', () => {
    fixture.rows=[log(1,202,'before denial')];
    const task=buildTask(); const {rerender,view}=setup('/task?host=11',task);
    fireEvent.click(screen.getByRole('button',{name:'pause-tail'}));
    fixture.state='forbidden'; rerender(view({...task}));
    expect(screen.queryByText('before denial')).toBeNull();
    fixture.state='ready';fixture.rows=[log(2,202,'after reauthorization')];rerender(view({...task}));
    expect(screen.queryByText('before denial')).toBeNull();
    expect(screen.getByText('after reauthorization')).toBeTruthy();
  });
});

describe('recorded checkpoint history', () => {
 it('keeps a shared step historical checkpoint separate from its current retry', () => {
  const task=buildTask(); task.steps=task.steps?.filter((step)=>step.id!==101);
  Object.assign(task,{stepHistory:[{id:202,taskHostId:11,stepKey:'recorded-install',stepOrder:2,status:'failed',attemptNo:1,stateVersion:3,capturedAt:'2026-11-08T00:01:00Z'}]});
  task.logAttempts=[{taskHostId:11,attemptNo:1}];
  fixture.rows=[{...log(1,202,'historical output'),attemptNo:1},log(2,202,'current output')];
  setup('/task?host=11&attempt=1&step=202',task);
  expect(screen.getByRole('button',{name:'recorded-install'}).getAttribute('data-status')).toBe('failed');
  expect(screen.getByText('business.deploy.logs.historySnapshot')).toBeTruthy();
  expect(screen.getByText('historical output')).toBeTruthy();
  expect(screen.queryByText('current output')).toBeNull();
 });
 it('keeps recorded attempts selectable even when they produced no logs', () => {
  const task=buildTask();task.steps=task.steps?.filter((step)=>step.id!==101);
  Object.assign(task,{stepHistory:[{id:202,taskHostId:11,stepKey:'no-output-attempt',stepOrder:2,status:'pending',attemptNo:4,stateVersion:9,capturedAt:'2026-11-08T00:01:00Z'}]});
  setup('/task?host=11&attempt=4',task);
  expect(screen.getByRole('button',{name:'no-output-attempt'})).toBeTruthy();
  expect(screen.getByLabelText('business.deploy.logs.attempts').getAttribute('value')??(screen.getByLabelText('business.deploy.logs.attempts') as HTMLSelectElement).value).toBe('4');
 });
 it('explicitly marks a bounded history projection as incomplete', () => {
  const task=buildTask();Object.assign(task,{stepHistoryLimited:true});setup('/task?host=11',task);
  expect(screen.getByText('business.deploy.logs.historyLimited')).toBeTruthy();
 });
 it('uses current checkpoint state when a recorded snapshot of that same attempt is older', () => {
  const task=buildTask();Object.assign(task,{stepHistory:[{id:201,taskHostId:11,stepKey:'stale-recorded-state',stepOrder:1,status:'failed',attemptNo:2,stateVersion:0,capturedAt:'2026-11-08T00:01:00Z'}]});
  setup('/task?host=11&attempt=2',task);
  expect(screen.getByRole('button',{name:'latest-check'}).getAttribute('data-status')).toBe('success');
  expect(screen.queryByRole('button',{name:'stale-recorded-state'})).toBeNull();
 });
});

it('does not silently replace a valid missing historical attempt with current results', () => {
 const task=buildTask();task.steps=task.steps?.filter((step)=>step.id!==101);Object.assign(task,{stepHistoryLimited:true});
 fixture.rows=[log(2,202,'current result must not become historical')];setup('/task?host=11&attempt=1',task);
 expect(screen.getByText('business.deploy.logs.historicalAttempt')).toBeTruthy();
 expect(screen.queryByText('current result must not become historical')).toBeNull();
});


describe('unequal checkpoint counters', () => {
 it('locates the actual failed step using its own attempt, including the default selection', () => {
  const task=buildTask();
  task.steps=[
   {id:201,taskHostId:11,stepKey:'prepare',stepOrder:1,status:'success',attemptNo:2,stateVersion:2},
   {id:202,taskHostId:11,stepKey:'install',stepOrder:2,status:'failed',attemptNo:1,stateVersion:1},
  ];
  fixture.rows=[{...log(1,202,'actual failure','stderr'),attemptNo:1}];
  setup('/task?host=11',task);
  expect(screen.getByText('actual failure')).toBeTruthy();
  fireEvent.click(screen.getByRole('button',{name:'business.deploy.logs.jumpFailure'}));
  expect(query().get('step')).toBe('202'); expect(query().get('attempt')).toBe('1');
  expect(query().get('firstError')).toBe('1'); expect(screen.getByText('actual failure')).toBeTruthy();
 });
});


it('defaults unequal successful checkpoint counters to all attempts without hiding either current checkpoint', () => {
 const task=buildTask(); task.status='success'; task.hosts![0].status='success';
 task.steps=[
  {id:201,taskHostId:11,stepKey:'prepare',stepOrder:1,status:'success',attemptNo:2,stateVersion:2},
  {id:202,taskHostId:11,stepKey:'install',stepOrder:2,status:'success',attemptNo:1,stateVersion:1},
 ];
 fixture.rows=[{...log(1,202,'retained install'),attemptNo:1},log(2,201,'retained prepare')];
 setup('/task?host=11',task);
 expect(screen.getByRole('button',{name:'prepare'})).toBeTruthy();
 expect(screen.getByRole('button',{name:'install'})).toBeTruthy();
 expect(screen.getByText('retained install')).toBeTruthy(); expect(screen.getByText('retained prepare')).toBeTruthy();
});
