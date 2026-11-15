import { act, cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import DeployTaskDetail from '../../../../src/modules/business/deploy/task/DeployTaskDetail';
import { getDeployTaskDetail, type DeployTaskRow } from '../../../../src/modules/business/deploy/api';
import { RequestError } from '../../../../src/api/request';

const permissions = vi.hoisted(() => ({ allowed: true }));
vi.mock('../../../../src/hooks/usePermission', () => ({ usePermission: () => ({ hasPerm: () => permissions.allowed }) }));
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('../../../../src/modules/business/deploy/api', () => ({ getDeployTaskDetail: vi.fn(), getDeployTaskEventStreamUrl: (id: number) => `/stream/${id}` }));
vi.mock('../../../../src/modules/business/deploy/task/DeployExecutionWorkbench', () => ({ default: ({ task }: { task: DeployTaskRow }) => <div>{task.name}-logs</div> }));
vi.mock('../../../../src/components', () => ({
  AppModal: () => null,
  PageLoading: () => <div>loading</div>, PageForbidden: () => <div>forbidden</div>,
  PageNotFound: () => <div>not-found</div>, PageEmpty: () => <div>empty</div>,
  PageError: () => <div>error</div>, PageRequestError: () => <div>request-error</div>,
}));
vi.mock('../../../../src/components/data-display/AppTable', () => ({ default: () => null }));
vi.mock('../../../../src/components/patterns/layout/PageContainer', () => ({ default: ({ children }: { children: React.ReactNode }) => <div>{children}</div> }));
vi.mock('../../../../src/modules/business/shared/BusinessPageHeader', () => ({ default: ({ title, extra }: { title: string; extra: React.ReactNode }) => <div>{title}{extra}</div> }));

class Source {
  static instances: Source[] = [];
  listeners: Record<string, () => void> = {};
  closed = false;
  onopen?: () => void;
  onerror?: () => void;
  constructor() { Source.instances.push(this); }
  addEventListener(name: string, callback: () => void) { this.listeners[name] = callback; }
  close() { this.closed = true; }
}
const flush = () => act(async () => { await Promise.resolve(); });
async function setup(status: string) {
  vi.stubGlobal('EventSource', Source);
  vi.stubGlobal('matchMedia', () => ({ matches: false, addListener: () => {}, removeListener: () => {} }));
  vi.mocked(getDeployTaskDetail).mockResolvedValue({ id: 7, name: 'visible-task', status, executorType: 'manual', hosts: [], templateParams: {} } as unknown as DeployTaskRow);
  render(<MemoryRouter initialEntries={['/task/7']}><Routes><Route path="/task/:id" element={<DeployTaskDetail />} /></Routes></MemoryRouter>);
  await flush();
}
afterEach(() => { cleanup(); vi.resetAllMocks(); vi.unstubAllGlobals(); vi.useRealTimers(); Source.instances = []; permissions.allowed = true; });
describe('rendered deployment detail controls and read freshness', () => {
  for (const status of ['paused', 'pause_requested', 'cancel_requested']) {
    it(`exposes supported manual cancellation in ${status}`, async () => {
      await setup(status);
      expect(screen.getByRole('button', { name: 'business.deploy.task.cancel' })).toBeTruthy();
    });
  }
  it('does not expose controls without permission and offers rollback draft only on supported terminal states', async () => {
    permissions.allowed = false; await setup('paused');
    expect(screen.queryByRole('button', { name: 'business.deploy.task.cancel' })).toBeNull();
    cleanup(); permissions.allowed = true; await setup('failed');
    expect(screen.getByRole('button', { name: 'business.deploy.task.rollbackDraft' })).toBeTruthy();
  });
  it('shows forbidden and removes task/log content when a connected stream is denied', async () => {
    await setup('running'); const source = Source.instances[0];
    expect(screen.getByText('visible-task-logs')).toBeTruthy();
    act(() => { source.listeners.forbidden(); }); await flush();
    expect(screen.getByText('forbidden')).toBeTruthy();
    expect(screen.queryByText('visible-task-logs')).toBeNull(); expect(source.closed).toBe(true);
  });
  it('marks a connected silent refresh stale and resumes recoverable polling', async () => {
    vi.useFakeTimers(); await setup('running'); const source = Source.instances[0];
    act(() => { source.onopen?.(); });
    vi.mocked(getDeployTaskDetail).mockRejectedValueOnce(new Error('offline'));
    act(() => { source.listeners.task(); }); await flush();
    expect(screen.getByText('business.deploy.task.refreshStale')).toBeTruthy();
    await act(async () => { await vi.advanceTimersByTimeAsync(5000); });
    expect(screen.queryByText('business.deploy.task.refreshStale')).toBeNull();
    expect(getDeployTaskDetail).toHaveBeenCalledTimes(3);
  });
  it('reads the terminal result when the last connected SSE event arrives during an older running request', async () => {
    vi.useFakeTimers(); await setup('running'); const source = Source.instances[0];
    act(() => { source.onopen?.(); });
    const capturedRunning = await vi.mocked(getDeployTaskDetail).mock.results[0].value as DeployTaskRow;
    let resolveRunning: ((value: DeployTaskRow) => void) | undefined;
    vi.mocked(getDeployTaskDetail)
      .mockImplementationOnce(() => new Promise((resolve) => { resolveRunning = resolve; }))
      .mockResolvedValueOnce({ ...capturedRunning, status: 'success' });
    act(() => { source.listeners.task(); }); await flush();
    expect(getDeployTaskDetail).toHaveBeenCalledTimes(2);
    // The server completes after this request captured running; no later SSE or polling follows.
    act(() => { source.listeners.task(); }); await flush();
    expect(getDeployTaskDetail).toHaveBeenCalledTimes(2);
    await act(async () => { resolveRunning?.(capturedRunning); }); await flush();
    expect(getDeployTaskDetail).toHaveBeenCalledTimes(3);
    expect(screen.getAllByText('business.deploy.task.status.success').some((element) => element.closest('.arco-tag'))).toBe(true);
    expect(screen.queryByText('business.deploy.task.refreshStale')).toBeNull();
    expect(source.closed).toBe(true);
  });
  it('clears the task on silent detail denial', async () => {
    await setup('running');
    vi.mocked(getDeployTaskDetail).mockRejectedValueOnce(new RequestError({ kind: 'forbidden', status: 403 }));
    act(() => { Source.instances[0].listeners.task(); }); await flush();
    expect(screen.getByText('forbidden')).toBeTruthy();
    expect(screen.queryByText('visible-task-logs')).toBeNull();
  });
});
